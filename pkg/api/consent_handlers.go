package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/telemedicine/zkhealth/pkg/consent"
)

// Consent request and response types
type CreateConsentRequest struct {
	PatientID           string   `json:"patient_id"`
	ConsentType         string   `json:"consent_type"`
	Description         string   `json:"description"`
	PartyIDs            []string `json:"party_ids"`
	Roles               []string `json:"roles"`
	ExpiryDays          int      `json:"expiry_days"`
	AllPartiesRequired  bool     `json:"all_parties_required"`
	Resources           []string `json:"resources"`
}

type UpdatePartyConsentRequest struct {
	PartyID  string `json:"party_id"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
	ZKProof  string `json:"zk_proof,omitempty"`
}

type VerifyConsentRequest struct {
	PartyID    string `json:"party_id"`
	PatientID  string `json:"patient_id"`
	ResourceID string `json:"resource_id"`
	Type       string `json:"type"`
}

// createConsent creates a new multi-party consent
func (s *HealthServer) createConsent(w http.ResponseWriter, r *http.Request) {
	var req CreateConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.PatientID == "" || req.ConsentType == "" || len(req.PartyIDs) == 0 || len(req.Roles) == 0 {
		respondError(w, http.StatusBadRequest, "Patient ID, consent type, party IDs, and roles are required")
		return
	}

	// Set default expiry if not provided
	if req.ExpiryDays <= 0 {
		req.ExpiryDays = 30 // Default to 30 days
	}

	// Create consent
	consentID, err := s.ConsentManager.CreateConsent(
		r.Context(),
		req.PatientID,
		consent.ConsentType(req.ConsentType),
		req.Description,
		req.PartyIDs,
		req.Roles,
		req.ExpiryDays,
		req.AllPartiesRequired,
		req.Resources,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create consent: "+err.Error())
		return
	}

	// Log the event
	eventID, err := s.EventLogger.LogEvent(
		r.Context(),
		"consent_created",
		req.PatientID,
		map[string]interface{}{
			"consent_id":   consentID,
			"consent_type": req.ConsentType,
			"party_ids":    req.PartyIDs,
		},
	)
	if err != nil {
		// Just log the error, don't fail the request
		log.Printf("Failed to log consent creation event: %v", err)
	}

	// Return response
	respondJSON(w, http.StatusCreated, map[string]string{
		"consent_id": consentID,
		"event_id":   eventID,
	})
}

// getConsent retrieves a consent by ID
func (s *HealthServer) getConsent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	consentID := vars["id"]
	if consentID == "" {
		respondError(w, http.StatusBadRequest, "Consent ID is required")
		return
	}

	// Get consent
	consentObj, err := s.ConsentManager.GetConsent(r.Context(), consentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get consent: "+err.Error())
		return
	}

	if consentObj == nil {
		respondError(w, http.StatusNotFound, "Consent not found")
		return
	}

	// Return response
	respondJSON(w, http.StatusOK, consentObj)
}

// updatePartyConsent updates a party's consent status
func (s *HealthServer) updatePartyConsent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	consentID := vars["id"]
	if consentID == "" {
		respondError(w, http.StatusBadRequest, "Consent ID is required")
		return
	}

	var req UpdatePartyConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.PartyID == "" || req.Status == "" {
		respondError(w, http.StatusBadRequest, "Party ID and status are required")
		return
	}

	// Convert status string to ConsentStatus
	var status consent.ConsentStatus
	switch req.Status {
	case "approved":
		status = consent.StatusApproved
	case "revoked":
		status = consent.StatusRevoked
	case "pending":
		status = consent.StatusPending
	default:
		respondError(w, http.StatusBadRequest, "Invalid status, must be 'approved', 'revoked', or 'pending'")
		return
	}

	// Update party consent
	err := s.ConsentManager.UpdatePartyConsent(r.Context(), consentID, req.PartyID, status, req.Reason, req.ZKProof)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update party consent: "+err.Error())
		return
	}

	// Log the event
	eventID, err := s.EventLogger.LogEvent(
		r.Context(),
		"consent_updated",
		req.PartyID,
		map[string]interface{}{
			"consent_id": consentID,
			"status":     req.Status,
			"reason":     req.Reason,
		},
	)
	if err != nil {
		// Just log the error, don't fail the request
		log.Printf("Failed to log consent update event: %v", err)
	}

	// Return response
	respondJSON(w, http.StatusOK, map[string]string{
		"message":  "Consent updated successfully",
		"event_id": eventID,
	})
}

// getPatientConsents retrieves all consents for a patient
func (s *HealthServer) getPatientConsents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	patientID := vars["patientId"]
	if patientID == "" {
		respondError(w, http.StatusBadRequest, "Patient ID is required")
		return
	}

	// Get patient consents
	consents, err := s.ConsentManager.GetPatientConsents(r.Context(), patientID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get patient consents: "+err.Error())
		return
	}

	// Return response
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"patient_id": patientID,
		"consents":   consents,
	})
}

// verifyConsent verifies if a party has consent to access resources
func (s *HealthServer) verifyConsent(w http.ResponseWriter, r *http.Request) {
	var req VerifyConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.PartyID == "" || req.PatientID == "" || req.Type == "" {
		respondError(w, http.StatusBadRequest, "Party ID, patient ID, and consent type are required")
		return
	}

	// Verify consent
	hasConsent, consentID, err := s.ConsentManager.VerifyConsent(
		r.Context(),
		req.PartyID,
		req.PatientID,
		req.ResourceID,
		consent.ConsentType(req.Type),
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify consent: "+err.Error())
		return
	}

	// Log the verification attempt
	eventType := "consent_verification"
	if hasConsent {
		eventType = "consent_verified"
	} else {
		eventType = "consent_verification_failed"
	}

	s.EventLogger.LogEvent(
		r.Context(),
		eventType,
		req.PartyID,
		map[string]interface{}{
			"patient_id":  req.PatientID,
			"resource_id": req.ResourceID,
			"type":        req.Type,
			"has_consent": hasConsent,
			"consent_id":  consentID,
		},
	)

	// Return response
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"has_consent": hasConsent,
		"consent_id":  consentID,
	})
}

// getActiveConsentsByParty retrieves active consents for a party
func (s *HealthServer) getActiveConsentsByParty(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	partyID := vars["partyId"]
	if partyID == "" {
		respondError(w, http.StatusBadRequest, "Party ID is required")
		return
	}

	// Get active consents
	consents, err := s.ConsentManager.GetActiveConsentsByParty(r.Context(), partyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get active consents: "+err.Error())
		return
	}

	// Return response
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"party_id": partyID,
		"consents": consents,
	})
}
