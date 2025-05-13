package zkgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/telemedicine/zkhealth/pkg/zkproof"
)

// ZKGatewayMiddleware implements HTTP middleware for token validation and rate limiting
type ZKGatewayMiddleware struct {
	TokenGenerator *TokenGenerator
	RateLimiter    *RateLimiter
	ZKIdentity     *zkproof.ZKIdentity
	PublicPaths    []string
}

// ClaimContext is the key type for storing claims in the request context
type ClaimContext string

const (
	// ClaimContextKey is the key used to store claim information in the request context
	ClaimContextKey ClaimContext = "claim_context"
	
	// PartyIDContextKey is the key used to store party ID in the request context
	PartyIDContextKey ClaimContext = "party_id_context"
	
	// APIKeyHeader is the header used for providing API tokens
	APIKeyHeader = "X-ZK-API-Key"
)

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TokenClaims represents the claims extracted from a validated token
type TokenClaims struct {
	PartyID  string `json:"party_id"`
	Claim    string `json:"claim"`
	TokenID  string `json:"token_id"`
	IssuedAt string `json:"issued_at"`
}

// NewZKGatewayMiddleware creates a new ZK gateway middleware
func NewZKGatewayMiddleware(
	ctx context.Context,
	mongoURI string,
	zkIdentity *zkproof.ZKIdentity,
	publicPaths []string,
) (*ZKGatewayMiddleware, error) {
	// Create token generator
	tokenGenerator, err := NewTokenGenerator(ctx, mongoURI, zkIdentity)
	if err != nil {
		return nil, fmt.Errorf("failed to create token generator: %v", err)
	}

	// Create rate limiter
	rateLimiter, err := NewRateLimiter(ctx, mongoURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate limiter: %v", err)
	}

	return &ZKGatewayMiddleware{
		TokenGenerator: tokenGenerator,
		RateLimiter:    rateLimiter,
		ZKIdentity:     zkIdentity,
		PublicPaths:    publicPaths,
	}, nil
}

// Close closes all connections
func (m *ZKGatewayMiddleware) Close(ctx context.Context) error {
	if err := m.TokenGenerator.Close(ctx); err != nil {
		return err
	}
	
	return m.RateLimiter.Close(ctx)
}

// Middleware returns an HTTP middleware function for securing API endpoints
func (m *ZKGatewayMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is public
		if m.isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from header
		tokenID := r.Header.Get(APIKeyHeader)
		if tokenID == "" {
			m.sendError(w, http.StatusUnauthorized, "Missing API key", "No API key provided in the X-ZK-API-Key header")
			return
		}

		// Validate token
		ctx := r.Context()
		isValid, token, err := m.TokenGenerator.ValidateToken(ctx, tokenID)
		if err != nil {
			log.Printf("Error validating token: %v", err)
			m.sendError(w, http.StatusInternalServerError, "Token validation error", "An error occurred while validating your API key")
			return
		}

		if !isValid || token == nil {
			m.sendError(w, http.StatusUnauthorized, "Invalid API key", "The provided API key is invalid, expired, or revoked")
			return
		}

		// Check rate limits
		allowed, err := m.RateLimiter.AllowRequest(ctx, token.PartyID, r.URL.Path, token.Claim)
		if err != nil {
			log.Printf("Rate limiting error: %v", err)
			m.sendError(w, http.StatusInternalServerError, "Rate limit check error", "An error occurred while checking rate limits")
			return
		}

		if !allowed {
			m.sendError(w, http.StatusTooManyRequests, "Rate limit exceeded", "You have exceeded the rate limit for this endpoint")
			return
		}

		// Add claims to context
		tokenClaims := TokenClaims{
			PartyID:  token.PartyID,
			Claim:    token.Claim,
			TokenID:  token.TokenID,
			IssuedAt: token.CreatedAt.Format(time.RFC3339),
		}

		newCtx := context.WithValue(ctx, ClaimContextKey, tokenClaims)
		newCtx = context.WithValue(newCtx, PartyIDContextKey, token.PartyID)

		// Continue with the request
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

// AdminOnlyMiddleware ensures only admin users can access an endpoint
func (m *ZKGatewayMiddleware) AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get claims from context
		ctx := r.Context()
		claimsValue := ctx.Value(ClaimContextKey)
		if claimsValue == nil {
			m.sendError(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
			return
		}

		claims, ok := claimsValue.(TokenClaims)
		if !ok {
			m.sendError(w, http.StatusInternalServerError, "Context error", "Failed to parse authentication context")
			return
		}

		// Check if user has admin claim
		if claims.Claim != "admin" {
			m.sendError(w, http.StatusForbidden, "Access denied", "Admin access required")
			return
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// DoctorOnlyMiddleware ensures only doctors can access an endpoint
func (m *ZKGatewayMiddleware) DoctorOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get claims from context
		ctx := r.Context()
		claimsValue := ctx.Value(ClaimContextKey)
		if claimsValue == nil {
			m.sendError(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
			return
		}

		claims, ok := claimsValue.(TokenClaims)
		if !ok {
			m.sendError(w, http.StatusInternalServerError, "Context error", "Failed to parse authentication context")
			return
		}

		// Check if user has doctor claim
		if claims.Claim != "doctor" && claims.Claim != "admin" {
			m.sendError(w, http.StatusForbidden, "Access denied", "Doctor access required")
			return
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// PatientDataMiddleware ensures users can only access their own patient data
func (m *ZKGatewayMiddleware) PatientDataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get party ID from context
		ctx := r.Context()
		partyIDValue := ctx.Value(PartyIDContextKey)
		if partyIDValue == nil {
			m.sendError(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
			return
		}

		partyID, ok := partyIDValue.(string)
		if !ok {
			m.sendError(w, http.StatusInternalServerError, "Context error", "Failed to parse party ID from context")
			return
		}

		// Get claims from context to check if admin or doctor
		claimsValue := ctx.Value(ClaimContextKey)
		if claimsValue == nil {
			m.sendError(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
			return
		}

		claims, ok := claimsValue.(TokenClaims)
		if !ok {
			m.sendError(w, http.StatusInternalServerError, "Context error", "Failed to parse authentication context")
			return
		}

		// Extract patient ID from URL or request
		// This is a simplified example; in a real implementation,
		// you would extract the patient ID from the URL path
		patientID := r.URL.Query().Get("patient_id")
		if patientID == "" {
			// Try to get from path - assuming a RESTful pattern like /patients/{id}
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) >= 3 && parts[1] == "patients" {
				patientID = parts[2]
			}
		}

		if patientID == "" {
			m.sendError(w, http.StatusBadRequest, "Invalid request", "Patient ID not specified")
			return
		}

		// Check if user is requesting their own data or has elevated permissions
		if patientID != partyID && claims.Claim != "admin" && claims.Claim != "doctor" {
			m.sendError(w, http.StatusForbidden, "Access denied", "You can only access your own patient data")
			return
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// isPublicPath checks if a path is in the public paths list
func (m *ZKGatewayMiddleware) isPublicPath(path string) bool {
	// Always allow OPTIONS requests for CORS
	if strings.HasPrefix(path, "/auth") || strings.HasPrefix(path, "/public") {
		return true
	}
	
	for _, publicPath := range m.PublicPaths {
		if publicPath == path || 
		   (strings.HasSuffix(publicPath, "*") && 
		    strings.HasPrefix(path, publicPath[:len(publicPath)-1])) {
			return true
		}
	}
	
	return false
}

// sendError sends a JSON error response
func (m *ZKGatewayMiddleware) sendError(w http.ResponseWriter, statusCode int, errorType, message string) {
	response := ErrorResponse{
		Error:   errorType,
		Code:    statusCode,
		Message: message,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// GetTokenClaims extracts token claims from the request context
func GetTokenClaims(r *http.Request) (*TokenClaims, bool) {
	claimsValue := r.Context().Value(ClaimContextKey)
	if claimsValue == nil {
		return nil, false
	}
	
	claims, ok := claimsValue.(TokenClaims)
	if !ok {
		return nil, false
	}
	
	return &claims, true
}

// GetPartyID extracts party ID from the request context
func GetPartyID(r *http.Request) (string, bool) {
	partyIDValue := r.Context().Value(PartyIDContextKey)
	if partyIDValue == nil {
		return "", false
	}
	
	partyID, ok := partyIDValue.(string)
	if !ok {
		return "", false
	}
	
	return partyID, true
}
