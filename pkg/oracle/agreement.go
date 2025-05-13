package oracle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/telemedicine/zkhealth/pkg/merkletree"
)

// ClauseType defines the type of clause in an agreement
type ClauseType string

const (
	// ClauseConsent represents a consent requirement
	ClauseConsent ClauseType = "consent"
	// ClauseRetention represents a data retention policy
	ClauseRetention ClauseType = "retention"
	// ClauseJurisdiction represents a jurisdictional requirement
	ClauseJurisdiction ClauseType = "jurisdiction"
	// ClauseAccess represents data access rights
	ClauseAccess ClauseType = "access"
	// ClauseTransfer represents data transfer policies
	ClauseTransfer ClauseType = "transfer"
	// ClauseReporting represents reporting requirements
	ClauseReporting ClauseType = "reporting"
)

// Precondition represents a condition that must be met for a clause
type Precondition struct {
	VariableName string      `json:"variable_name"`
	Operator     string      `json:"operator"` // "=", ">", "<", "contains", "exists", etc.
	Value        interface{} `json:"value"`
}

// Clause represents a specific agreement clause with preconditions
type Clause struct {
	ID             string        `json:"id"`
	Type           ClauseType    `json:"type"`
	Description    string        `json:"description"`
	LegalReference string        `json:"legal_reference"`
	Preconditions  []Precondition `json:"preconditions"`
	RawText        string        `json:"raw_text"` // Original legal text
}

// Agreement represents a regulatory agreement (e.g., HIPAA, GDPR)
type Agreement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Jurisdiction string   `json:"jurisdiction"`
	EffectiveDate time.Time `json:"effective_date"`
	Clauses      []Clause  `json:"clauses"`
	Hash         string    `json:"hash"` // SHA256 hash of all clauses
	MerkleRoot   string    `json:"merkle_root"` // Merkle root of all clauses
}

// OracleAgreement manages regulatory agreements and their verification
type OracleAgreement struct {
	Agreements map[string]*Agreement // Map of agreement ID to Agreement
}

// NewOracleAgreement creates a new oracle agreement manager
func NewOracleAgreement() *OracleAgreement {
	return &OracleAgreement{
		Agreements: make(map[string]*Agreement),
	}
}

// LoadAgreementFromFile loads an agreement from a JSON or YAML file
func (oa *OracleAgreement) LoadAgreementFromFile(filePath string) (*Agreement, error) {
	// Read file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agreement file: %v", err)
	}

	// Parse JSON
	var agreement Agreement
	if err := json.Unmarshal(data, &agreement); err != nil {
		return nil, fmt.Errorf("failed to parse agreement: %v", err)
	}

	// Validate agreement
	if agreement.ID == "" || len(agreement.Clauses) == 0 {
		return nil, errors.New("invalid agreement: missing ID or clauses")
	}

	// Generate hash and merkle root
	agreementHash, merkleRoot, err := oa.generateAgreementHashes(agreement)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agreement hashes: %v", err)
	}

	agreement.Hash = agreementHash
	agreement.MerkleRoot = merkleRoot

	// Store agreement
	oa.Agreements[agreement.ID] = &agreement

	return &agreement, nil
}

// GenerateAgreementHashes generates hash and merkle root for an agreement
func (oa *OracleAgreement) generateAgreementHashes(agreement Agreement) (string, string, error) {
	// Create a slice to hold clause data for hashing
	clauseData := make([]string, len(agreement.Clauses))

	// Convert each clause to a JSON string for hashing
	for i, clause := range agreement.Clauses {
		clauseBytes, err := json.Marshal(clause)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal clause: %v", err)
		}
		clauseData[i] = string(clauseBytes)
	}

	// Generate a Merkle tree from clause data
	merkleTree, err := merkletree.NewMerkleTree(clauseData)
	if err != nil {
		return "", "", fmt.Errorf("failed to build merkle tree: %v", err)
	}

	// Get the merkle root
	merkleRoot := merkleTree.GetRootHash()

	// Generate overall agreement hash (concatenate and hash all clauses)
	allClausesData := ""
	for _, clause := range clauseData {
		allClausesData += clause
	}

	agreementHash := sha256Hash(allClausesData)

	return agreementHash, merkleRoot, nil
}

// VerifyClausePreconditions verifies if all preconditions of a clause are met
func (oa *OracleAgreement) VerifyClausePreconditions(agreementID, clauseID string, context map[string]interface{}) (bool, error) {
	// Get the agreement
	agreement, exists := oa.Agreements[agreementID]
	if !exists {
		return false, fmt.Errorf("agreement not found: %s", agreementID)
	}

	// Find the clause
	var targetClause *Clause
	for _, clause := range agreement.Clauses {
		if clause.ID == clauseID {
			targetClause = &clause
			break
		}
	}

	if targetClause == nil {
		return false, fmt.Errorf("clause not found: %s", clauseID)
	}

	// Check all preconditions
	for _, precond := range targetClause.Preconditions {
		contextValue, exists := context[precond.VariableName]
		if !exists {
			return false, fmt.Errorf("context variable not found: %s", precond.VariableName)
		}

		// Check the condition based on the operator
		switch precond.Operator {
		case "=", "==", "equals":
			if contextValue != precond.Value {
				return false, nil
			}
		case ">":
			// Type assertions would be necessary here based on expected types
			// This is a simplified version
			floatContextValue, ok1 := contextValue.(float64)
			floatCondValue, ok2 := precond.Value.(float64)
			if !ok1 || !ok2 {
				return false, fmt.Errorf("type mismatch for > operator: %s", precond.VariableName)
			}
			if floatContextValue <= floatCondValue {
				return false, nil
			}
		case "<":
			floatContextValue, ok1 := contextValue.(float64)
			floatCondValue, ok2 := precond.Value.(float64)
			if !ok1 || !ok2 {
				return false, fmt.Errorf("type mismatch for < operator: %s", precond.VariableName)
			}
			if floatContextValue >= floatCondValue {
				return false, nil
			}
		case "contains":
			strContextValue, ok1 := contextValue.(string)
			strCondValue, ok2 := precond.Value.(string)
			if !ok1 || !ok2 {
				return false, fmt.Errorf("type mismatch for contains operator: %s", precond.VariableName)
			}
			// Simple string contains check
			if !strings.Contains(strContextValue, strCondValue) {
				return false, nil
			}
		case "exists":
			boolValue, ok := precond.Value.(bool)
			if !ok || !boolValue {
				return false, nil
			}
			// The variable exists since we already checked at the beginning
		default:
			return false, fmt.Errorf("unsupported operator: %s", precond.Operator)
		}
	}

	// All preconditions passed
	return true, nil
}

// VerifyAgreementHash verifies the hash of an agreement
func (oa *OracleAgreement) VerifyAgreementHash(agreementID, providedHash string) (bool, error) {
	agreement, exists := oa.Agreements[agreementID]
	if !exists {
		return false, fmt.Errorf("agreement not found: %s", agreementID)
	}

	return agreement.Hash == providedHash, nil
}

// GetClauseProof gets the merkle proof for a specific clause
func (oa *OracleAgreement) GetClauseProof(agreementID, clauseID string) ([]string, error) {
	agreement, exists := oa.Agreements[agreementID]
	if !exists {
		return nil, fmt.Errorf("agreement not found: %s", agreementID)
	}

	// Find the index of the clause
	clauseIndex := -1
	clauseData := make([]string, len(agreement.Clauses))
	
	for i, clause := range agreement.Clauses {
		clauseBytes, err := json.Marshal(clause)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal clause: %v", err)
		}
		clauseData[i] = string(clauseBytes)
		
		if clause.ID == clauseID {
			clauseIndex = i
		}
	}

	if clauseIndex == -1 {
		return nil, fmt.Errorf("clause not found: %s", clauseID)
	}

	// Regenerate the merkle tree and get the proof
	merkleTree, err := merkletree.NewMerkleTree(clauseData)
	if err != nil {
		return nil, fmt.Errorf("failed to build merkle tree: %v", err)
	}

	proof, err := merkleTree.GenerateProof(clauseData, clauseIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %v", err)
	}

	return proof, nil
}

// sha256Hash computes the SHA-256 hash of a string
func sha256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
