package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/telemedicine/zkhealth/pkg/zkproof"
)

// ExecutionEvent represents an event that needs validation against agreement clauses
type ExecutionEvent struct {
	EventID           string                 `json:"event_id"`
	EventType         string                 `json:"event_type"`
	AgreementID       string                 `json:"agreement_id"`
	ClauseIDs         []string               `json:"clause_ids"`
	Timestamp         time.Time              `json:"timestamp"`
	Context           map[string]interface{} `json:"context"`
	SignerID          string                 `json:"signer_id"`
	ZKProof           string                 `json:"zk_proof"`
}

// ValidationResult represents the result of validating an execution event
type ValidationResult struct {
	Valid           bool                  `json:"valid"`
	EventID         string                `json:"event_id"`
	AgreementID     string                `json:"agreement_id"`
	ClauseValidations map[string]bool     `json:"clause_validations"`
	Timestamp       time.Time             `json:"timestamp"`
	ValidationNotes []string              `json:"validation_notes"`
}

// ExecutionValidator validates execution events against oracle agreements
type ExecutionValidator struct {
	Oracle    *OracleAgreement
	ZKIdentity *zkproof.ZKIdentity
}

// NewExecutionValidator creates a new execution validator
func NewExecutionValidator(oracle *OracleAgreement, zkIdentity *zkproof.ZKIdentity) *ExecutionValidator {
	return &ExecutionValidator{
		Oracle:    oracle,
		ZKIdentity: zkIdentity,
	}
}

// ValidateEvent validates an execution event against the relevant agreement clauses
func (ev *ExecutionValidator) ValidateEvent(ctx context.Context, event ExecutionEvent) (*ValidationResult, error) {
	// Initialize validation result
	result := &ValidationResult{
		Valid:            true, // Start with valid, will be set to false if any clause fails
		EventID:          event.EventID,
		AgreementID:      event.AgreementID,
		ClauseValidations: make(map[string]bool),
		Timestamp:        time.Now().UTC(),
		ValidationNotes:  []string{},
	}

	// Check if the agreement exists
	agreement, exists := ev.Oracle.Agreements[event.AgreementID]
	if !exists {
		result.Valid = false
		result.ValidationNotes = append(result.ValidationNotes, 
			fmt.Sprintf("Agreement not found: %s", event.AgreementID))
		return result, nil
	}

	// Verify ZK proof if provided
	if event.ZKProof != "" {
		// Get identity from ZK proof
		identity, err := ev.ZKIdentity.GetIdentityByProof(ctx, event.ZKProof)
		if err != nil {
			result.Valid = false
			result.ValidationNotes = append(result.ValidationNotes, 
				fmt.Sprintf("Failed to verify ZK proof: %v", err))
			return result, nil
		}

		// Check if identity matches the signer
		if identity == nil || identity.PartyID != event.SignerID {
			result.Valid = false
			result.ValidationNotes = append(result.ValidationNotes, 
				"ZK proof does not match the signer identity")
			return result, nil
		}
	} else {
		// No ZK proof provided
		result.Valid = false
		result.ValidationNotes = append(result.ValidationNotes, 
			"No ZK proof provided with the event")
		return result, nil
	}

	// Validate each clause
	for _, clauseID := range event.ClauseIDs {
		isValid, err := ev.Oracle.VerifyClausePreconditions(
			event.AgreementID, 
			clauseID, 
			event.Context,
		)
		
		if err != nil {
			result.ValidationNotes = append(result.ValidationNotes, 
				fmt.Sprintf("Error validating clause %s: %v", clauseID, err))
			result.ClauseValidations[clauseID] = false
			result.Valid = false
			continue
		}
		
		result.ClauseValidations[clauseID] = isValid
		
		if !isValid {
			result.Valid = false
			result.ValidationNotes = append(result.ValidationNotes, 
				fmt.Sprintf("Clause %s validation failed", clauseID))
		}
	}

	return result, nil
}

// CreateZKSignedEvent creates an execution event with a ZK proof
func (ev *ExecutionValidator) CreateZKSignedEvent(
	ctx context.Context,
	eventID string,
	eventType string,
	agreementID string,
	clauseIDs []string,
	context map[string]interface{},
	signerID string,
	claim string,
) (*ExecutionEvent, error) {
	// Register or validate the signer identity
	zkProof, err := ev.ZKIdentity.RegisterIdentity(ctx, signerID, claim)
	if err != nil {
		return nil, fmt.Errorf("failed to register identity: %v", err)
	}

	// Create event
	event := &ExecutionEvent{
		EventID:     eventID,
		EventType:   eventType,
		AgreementID: agreementID,
		ClauseIDs:   clauseIDs,
		Timestamp:   time.Now().UTC(),
		Context:     context,
		SignerID:    signerID,
		ZKProof:     zkProof,
	}

	return event, nil
}

// VerifyEventSignature verifies that an event's ZK signature is valid
func (ev *ExecutionValidator) VerifyEventSignature(ctx context.Context, event ExecutionEvent) (bool, error) {
	if event.ZKProof == "" || event.SignerID == "" {
		return false, errors.New("missing ZK proof or signer ID")
	}

	// Get identity from ZK proof
	identity, err := ev.ZKIdentity.GetIdentityByProof(ctx, event.ZKProof)
	if err != nil {
		return false, fmt.Errorf("failed to verify ZK proof: %v", err)
	}

	// Check if identity exists and matches the signer
	if identity == nil {
		return false, nil
	}

	return identity.PartyID == event.SignerID, nil
}

// SerializeEvent serializes an execution event to JSON
func (ev *ExecutionValidator) SerializeEvent(event ExecutionEvent) (string, error) {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("failed to serialize event: %v", err)
	}
	return string(eventBytes), nil
}

// DeserializeEvent deserializes an execution event from JSON
func (ev *ExecutionValidator) DeserializeEvent(eventJSON string) (*ExecutionEvent, error) {
	var event ExecutionEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return nil, fmt.Errorf("failed to deserialize event: %v", err)
	}
	return &event, nil
}
