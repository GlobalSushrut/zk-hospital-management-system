package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/telemedicine/zkhealth/pkg/policy"
)

// Global policy engine instance
var policyEngine *policy.PolicyEngine

// InitializePolicyEngine initializes the policy engine with default configuration
func InitializePolicyEngine() {
	config := policy.CreateDefaultConfig()
	policyEngine = policy.InitializeEngine(config)
}

// PolicyValidationRequest represents the API request for policy validation
type PolicyValidationRequest struct {
	ActorID        string            `json:"actor_id"`
	ActorRole      string            `json:"actor_role"`
	ActorAttributes map[string]string `json:"actor_attributes,omitempty"`
	Action         string            `json:"action"`
	Location       string            `json:"location"`
	ResourceID     string            `json:"resource_id"`
	ResourceType   string            `json:"resource_type"`
	ResourceAttrs  map[string]string `json:"resource_attributes,omitempty"`
	OwnerID        string            `json:"owner_id"`
	ZKProofs       []string          `json:"zk_proofs,omitempty"`
}

// PolicyValidationResponse represents the API response for policy validation
type PolicyValidationResponse struct {
	RequestID      string    `json:"request_id"`
	Allowed        bool      `json:"allowed"`
	Reason         string    `json:"reason"`
	ValidatorID    string    `json:"validator_id,omitempty"`
	ValidatorName  string    `json:"validator_name,omitempty"`
	ValidationTime time.Time `json:"validation_time"`
	AuditRecord    bool      `json:"audit_record_created"`
}

// ValidatePolicyHandler handles policy validation requests
func ValidatePolicyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PolicyValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert API request to internal validation request
	validationReq := policy.ValidationRequest{
		Actor: policy.ActorInfo{
			ID:         req.ActorID,
			Role:       req.ActorRole,
			Attributes: req.ActorAttributes,
			ZKProofs:   req.ZKProofs,
		},
		Action:   req.Action,
		Location: req.Location,
		Resource: policy.ResourceInfo{
			ID:         req.ResourceID,
			Type:       req.ResourceType,
			Attributes: req.ResourceAttrs,
			OwnerID:    req.OwnerID,
		},
		Timestamp:     time.Now(),
		RequestID:     uuid.New().String(),
		ClientAddress: r.RemoteAddr,
	}

	// Validate using policy engine
	result := policyEngine.ValidateAction(validationReq)

	// Prepare response
	response := PolicyValidationResponse{
		RequestID:      validationReq.RequestID,
		Allowed:        result.Allowed,
		Reason:         result.Reason,
		ValidatorID:    result.ValidatorID,
		ValidatorName:  result.ValidatorName,
		ValidationTime: result.ValidationTime,
		AuditRecord:    result.AuditRecord != nil,
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllowedActionsHandler returns all actions allowed for a role in a location
func GetAllowedActionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters from query string
	role := r.URL.Query().Get("role")
	location := r.URL.Query().Get("location")

	if role == "" || location == "" {
		http.Error(w, "Missing required parameters: role, location", http.StatusBadRequest)
		return
	}

	// Get allowed actions
	allowedActions := policyEngine.GetAllowedActions(role, location)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allowedActions)
}

// GetValidatorForActionHandler returns validator for an action in a location
func GetValidatorForActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters from query string
	action := r.URL.Query().Get("action")
	location := r.URL.Query().Get("location")

	if action == "" || location == "" {
		http.Error(w, "Missing required parameters: action, location", http.StatusBadRequest)
		return
	}

	// Get validator
	validator, exists := policyEngine.GetValidatorForAction(action, location)
	if !exists {
		http.Error(w, "No validator found", http.StatusNotFound)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validator)
}
