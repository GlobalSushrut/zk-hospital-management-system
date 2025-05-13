package policy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// PolicyServer provides HTTP API access to policy validation
type PolicyServer struct {
	Router      *mux.Router
	Validator   *Validator
	Port        int
	PolicyStore *PolicyStore
}

// NewPolicyServer creates a new policy server
func NewPolicyServer(port int) *PolicyServer {
	server := &PolicyServer{
		Router:      mux.NewRouter(),
		Validator:   NewValidator(),
		Port:        port,
		PolicyStore: NewPolicyStore(),
	}

	// Initialize routes
	server.setupRoutes()
	return server
}

// Start starts the policy server
func (s *PolicyServer) Start() error {
	addr := fmt.Sprintf(":%d", s.Port)
	log.Printf("Starting policy validation server on %s", addr)
	return http.ListenAndServe(addr, s.Router)
}

// setupRoutes initializes the API routes
func (s *PolicyServer) setupRoutes() {
	// Basic policy validation endpoint
	s.Router.HandleFunc("/policy/validate", s.validatePolicy).Methods("POST")
	
	// Role-based validation
	s.Router.HandleFunc("/policy/role", s.validateRolePolicy).Methods("POST")
	
	// Cross-jurisdiction validation
	s.Router.HandleFunc("/policy/cross-jurisdiction", s.validateCrossJurisdiction).Methods("POST")
	
	// Document access endpoint
	s.Router.HandleFunc("/document/store", s.storeDocument).Methods("POST")
	s.Router.HandleFunc("/document/retrieve", s.retrieveDocument).Methods("GET")
}

// validatePolicy handles the /policy/validate endpoint
func (s *PolicyServer) validatePolicy(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	result, reason := s.Validator.ValidateAccess(request)

	response := map[string]interface{}{
		"allowed": result,
		"reason":  reason,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, response)
}

// validateRolePolicy handles the /policy/role endpoint
func (s *PolicyServer) validateRolePolicy(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Extract requester info
	requester, _ := request["requester"].(map[string]interface{})
	role, _ := requester["role"].(string)
	
	// Extract subject info
	subject, _ := request["subject"].(map[string]interface{})
	recordType, _ := subject["record_type"].(string)
	sensitivity, _ := subject["sensitivity"].(string)
	
	// Apply role-based rules
	allowed := false
	reason := "Access denied by default role policy"
	
	switch role {
	case "physician":
		allowed = true
		reason = "Physicians have access to all patient records"
	case "nurse":
		if sensitivity != "high" {
			allowed = true
			reason = "Nurses have access to low and medium sensitivity records"
		} else {
			reason = "Nurses do not have access to high sensitivity records"
		}
	case "researcher":
		if recordType == "anonymized_data" {
			allowed = true
			reason = "Researchers have access to anonymized data only"
		} else {
			reason = "Researchers do not have access to identifiable patient records"
		}
	case "insurance_agent":
		if recordType == "billing" || recordType == "claims" {
			allowed = true
			reason = "Insurance agents have access to billing and claims records"
		} else {
			reason = "Insurance agents do not have access to clinical records"
		}
	default:
		reason = fmt.Sprintf("Unknown role '%s' has no defined access policy", role)
	}

	response := map[string]interface{}{
		"allowed": allowed,
		"reason":  reason,
		"role":    role,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, response)
}

// validateCrossJurisdiction handles the /policy/cross-jurisdiction endpoint
func (s *PolicyServer) validateCrossJurisdiction(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Extract requester and subject jurisdiction
	requester, _ := request["requester"].(map[string]interface{})
	requesterJurisdiction, _ := requester["jurisdiction"].(string)
	
	subject, _ := request["subject"].(map[string]interface{})
	subjectJurisdiction, _ := subject["jurisdiction"].(string)
	
	emergency, _ := request["emergency"].(bool)
	
	// Apply cross-jurisdiction rules
	allowed := false
	reason := "Cross-jurisdiction access denied by default policy"
	
	// Check for same jurisdiction (always allowed)
	if requesterJurisdiction == subjectJurisdiction {
		allowed = true
		reason = "Same jurisdiction access is allowed"
	} else {
		// Check for agreements between jurisdictions
		hasAgreement := s.PolicyStore.HasJurisdictionAgreement(requesterJurisdiction, subjectJurisdiction)
		
		if hasAgreement {
			allowed = true
			reason = fmt.Sprintf("Access allowed due to agreement between %s and %s", 
				requesterJurisdiction, subjectJurisdiction)
		} else if emergency {
			allowed = true
			reason = "Emergency override for cross-jurisdiction access"
		} else {
			reason = fmt.Sprintf("No data sharing agreement exists between %s and %s", 
				requesterJurisdiction, subjectJurisdiction)
		}
	}

	response := map[string]interface{}{
		"allowed":   allowed,
		"reason":    reason,
		"requester_jurisdiction": requesterJurisdiction,
		"subject_jurisdiction":   subjectJurisdiction,
		"emergency": emergency,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, response)
}

// storeDocument handles the /document/store endpoint
func (s *PolicyServer) storeDocument(w http.ResponseWriter, r *http.Request) {
	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Generate document ID
	docID := fmt.Sprintf("doc-%d", time.Now().UnixNano())
	
	response := map[string]interface{}{
		"doc_id":    docID,
		"timestamp": time.Now().Format(time.RFC3339),
		"status":    "stored",
		"location":  fmt.Sprintf("/documents/%s", docID),
	}

	respondJSON(w, http.StatusCreated, response)
}

// retrieveDocument handles the /document/retrieve endpoint
func (s *PolicyServer) retrieveDocument(w http.ResponseWriter, r *http.Request) {
	docID := r.URL.Query().Get("id")
	if docID == "" {
		respondError(w, http.StatusBadRequest, "Document ID is required")
		return
	}

	// Simulate retrieving document
	document := map[string]interface{}{
		"id":          docID,
		"title":       "Patient Medical Record",
		"created_at":  time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
		"content":     "This is a simulated medical document content",
		"owner_id":    "P10001",
		"sensitivity": "medium",
	}

	respondJSON(w, http.StatusOK, document)
}

// Helper functions for HTTP responses
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{"error": message})
}
