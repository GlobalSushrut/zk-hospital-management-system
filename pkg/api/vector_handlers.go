package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Vector request and response types
type StartVectorRequest struct {
	PatientID string `json:"patient_id"`
	Symptom   string `json:"symptom"`
	DoctorID  string `json:"doctor_id"`
}

type UpdateVectorStepRequest struct {
	Step string `json:"step"`
}

type CompleteVectorRequest struct {
	Outcome string `json:"outcome"`
	Success bool   `json:"success"`
}

type FeedbackRequest struct {
	Feedback string `json:"feedback"`
}

type ResolveAlertRequest struct {
	ActionTaken string `json:"action_taken"`
}

// startTreatmentVector initiates a new treatment vector
func (s *HealthServer) startTreatmentVector(w http.ResponseWriter, r *http.Request) {
	var req StartVectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.PatientID == "" || req.Symptom == "" || req.DoctorID == "" {
		respondError(w, http.StatusBadRequest, "Patient ID, symptom, and doctor ID are required")
		return
	}

	// Validate doctor identity
	isDoctor, err := s.ZkIdentity.ValidateClaim(r.Context(), req.DoctorID, "doctor")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to validate doctor identity: "+err.Error())
		return
	}

	if !isDoctor {
		respondError(w, http.StatusUnauthorized, "Invalid doctor identity")
		return
	}

	// Start treatment vector
	vectorID, err := s.MisalignmentTracker.StartTreatmentVector(r.Context(), req.PatientID, req.Symptom, req.DoctorID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start treatment vector: "+err.Error())
		return
	}

	// Log the event
	eventID, err := s.EventLogger.LogEvent(
		r.Context(),
		"treatment_vector_started",
		req.DoctorID,
		map[string]interface{}{
			"vector_id":  vectorID,
			"patient_id": req.PatientID,
			"symptom":    req.Symptom,
		},
	)
	if err != nil {
		// Just log the error, don't fail the request
		log.Printf("Failed to log treatment vector start event: %v", err)
	}

	// Return response
	respondJSON(w, http.StatusCreated, map[string]string{
		"vector_id": vectorID,
		"event_id":  eventID,
	})
}

// updateVectorStep adds a step to the actual treatment path
func (s *HealthServer) updateVectorStep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vectorID := vars["id"]
	if vectorID == "" {
		respondError(w, http.StatusBadRequest, "Vector ID is required")
		return
	}

	var req UpdateVectorStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.Step == "" {
		respondError(w, http.StatusBadRequest, "Step is required")
		return
	}

	// Update vector step
	err := s.MisalignmentTracker.UpdateActualPath(r.Context(), vectorID, req.Step)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update vector step: "+err.Error())
		return
	}

	// Log the event
	s.EventLogger.LogEvent(
		r.Context(),
		"treatment_step_added",
		vectorID,
		map[string]interface{}{
			"step": req.Step,
		},
	)

	// Return response
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Treatment step added successfully",
	})
}

// completeVector marks a treatment vector as completed
func (s *HealthServer) completeVector(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vectorID := vars["id"]
	if vectorID == "" {
		respondError(w, http.StatusBadRequest, "Vector ID is required")
		return
	}

	var req CompleteVectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.Outcome == "" {
		respondError(w, http.StatusBadRequest, "Outcome is required")
		return
	}

	// Complete the vector
	err := s.MisalignmentTracker.CompleteTreatmentVector(r.Context(), vectorID, req.Outcome, req.Success)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to complete treatment vector: "+err.Error())
		return
	}

	// Log the event
	eventID, err := s.EventLogger.LogEvent(
		r.Context(),
		"treatment_vector_completed",
		vectorID,
		map[string]interface{}{
			"outcome": req.Outcome,
			"success": req.Success,
		},
	)
	if err != nil {
		// Just log the error, don't fail the request
		log.Printf("Failed to log treatment completion event: %v", err)
	}

	// Mark the event as completed
	s.EventLogger.ResolveEvent(r.Context(), eventID, "completed")

	// Return response
	respondJSON(w, http.StatusOK, map[string]string{
		"message":  "Treatment vector completed successfully",
		"event_id": eventID,
	})
}

// addVectorFeedback adds feedback to a treatment vector
func (s *HealthServer) addVectorFeedback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vectorID := vars["id"]
	if vectorID == "" {
		respondError(w, http.StatusBadRequest, "Vector ID is required")
		return
	}

	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.Feedback == "" {
		respondError(w, http.StatusBadRequest, "Feedback is required")
		return
	}

	// Add feedback
	err := s.MisalignmentTracker.AddFeedback(r.Context(), vectorID, req.Feedback)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to add feedback: "+err.Error())
		return
	}

	// Log the event
	s.EventLogger.LogEvent(
		r.Context(),
		"treatment_feedback_added",
		vectorID,
		map[string]interface{}{
			"feedback": req.Feedback,
		},
	)

	// Return response
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Feedback added successfully",
	})
}

// resolveVectorAlert resolves an alert on a treatment vector
func (s *HealthServer) resolveVectorAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vectorID := vars["id"]
	indexStr := vars["index"]
	if vectorID == "" || indexStr == "" {
		respondError(w, http.StatusBadRequest, "Vector ID and alert index are required")
		return
	}

	// Convert index to int
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid alert index")
		return
	}

	var req ResolveAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.ActionTaken == "" {
		respondError(w, http.StatusBadRequest, "Action taken is required")
		return
	}

	// Resolve the alert
	err = s.MisalignmentTracker.ResolveAlert(r.Context(), vectorID, index, req.ActionTaken)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to resolve alert: "+err.Error())
		return
	}

	// Log the event
	s.EventLogger.LogEvent(
		r.Context(),
		"treatment_alert_resolved",
		vectorID,
		map[string]interface{}{
			"alert_index":  index,
			"action_taken": req.ActionTaken,
		},
	)

	// Return response
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Alert resolved successfully",
	})
}

// getActiveVectorsByPatient gets active treatment vectors for a patient
func (s *HealthServer) getActiveVectorsByPatient(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	patientID := vars["patientId"]
	if patientID == "" {
		respondError(w, http.StatusBadRequest, "Patient ID is required")
		return
	}

	// Get active vectors
	vectors, err := s.MisalignmentTracker.GetActiveVectorsByPatient(r.Context(), patientID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get active vectors: "+err.Error())
		return
	}

	// Return response
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"patient_id": patientID,
		"vectors":    vectors,
	})
}

// getMisalignedVectors gets vectors with high misalignment scores
func (s *HealthServer) getMisalignedVectors(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	doctorID := vars["doctorId"]
	if doctorID == "" {
		respondError(w, http.StatusBadRequest, "Doctor ID is required")
		return
	}

	// Get threshold from query parameter, default to 0.5
	thresholdStr := r.URL.Query().Get("threshold")
	threshold := 0.5
	if thresholdStr != "" {
		var err error
		threshold, err = strconv.ParseFloat(thresholdStr, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid threshold value")
			return
		}
	}

	// Get misaligned vectors
	vectors, err := s.MisalignmentTracker.GetMisalignedVectors(r.Context(), doctorID, threshold)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get misaligned vectors: "+err.Error())
		return
	}

	// Return response
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"doctor_id":  doctorID,
		"threshold":  threshold,
		"vectors":    vectors,
	})
}
