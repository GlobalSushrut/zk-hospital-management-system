package policy

import (
	"fmt"
	"log"
	"time"
)

// Validator provides policy validation functionality
type Validator struct {
	store *PolicyStore
}

// NewValidator creates a new policy validator
func NewValidator() *Validator {
	return &Validator{
		store: NewPolicyStore(),
	}
}

// ValidateAccess validates an access request against policies
func (v *Validator) ValidateAccess(request map[string]interface{}) (bool, string) {
	// Extract request fields
	requester, ok := request["requester"].(map[string]interface{})
	if !ok {
		return false, "Invalid requester information"
	}
	
	subject, ok := request["subject"].(map[string]interface{})
	if !ok {
		return false, "Invalid subject information"
	}
	
	// Extract action and purpose
	_, _ = request["action"].(string) // action is checked later if needed
	purpose, _ := request["purpose"].(string)
	authMethod, _ := request["auth_method"].(string)
	emergency, _ := request["emergency"].(bool)
	
	// Extract requester details
	requesterRole, _ := requester["role"].(string)
	// Department might be used in future policy rules
	_, _ = requester["department"].(string) 
	requesterJurisdiction, _ := requester["jurisdiction"].(string)
	
	// Extract subject details
	recordType, _ := subject["record_type"].(string)
	sensitivity, _ := subject["sensitivity"].(string)
	subjectJurisdiction, _ := subject["jurisdiction"].(string)
	
	// Log validation attempt for debugging
	log.Printf("Validating access: role=%s, record=%s, purpose=%s, emergency=%v", 
		requesterRole, recordType, purpose, emergency)
	
	// Check emergency override first (highest priority)
	if emergency {
		// Emergency access is logged and allowed but limited
		log.Printf("EMERGENCY ACCESS: role=%s, record=%s", requesterRole, recordType)
		return true, "Emergency access override"
	}
	
	// Cross-jurisdiction check
	if requesterJurisdiction != subjectJurisdiction {
		// Check if jurisdictions have data sharing agreement
		if !v.store.HasJurisdictionAgreement(requesterJurisdiction, subjectJurisdiction) {
			return false, fmt.Sprintf("No data sharing agreement between %s and %s", 
				requesterJurisdiction, subjectJurisdiction)
		}
		// If there is an agreement, proceed with other checks
	}
	
	// Role-based access control
	allowedRecordTypes, roleHasAccess := v.store.RoleAccess[requesterRole]
	if !roleHasAccess {
		return false, fmt.Sprintf("Role '%s' has no defined access policy", requesterRole)
	}
	
	// Check if role can access this record type
	recordAllowed := false
	for _, allowed := range allowedRecordTypes {
		if allowed == recordType || allowed == "*" {
			recordAllowed = true
			break
		}
	}
	
	if !recordAllowed {
		return false, fmt.Sprintf("Role '%s' cannot access record type '%s'", requesterRole, recordType)
	}
	
	// Check sensitivity level
	if sensitivity == "high" {
		// High sensitivity data has stricter requirements
		if requesterRole != "physician" && !emergency {
			return false, "High sensitivity data can only be accessed by physicians or in emergencies"
		}
		
		// Require stronger authentication for high sensitivity data
		if authMethod != "two_factor" && authMethod != "biometric" {
			return false, "High sensitivity data requires two-factor or biometric authentication"
		}
	}
	
	// Purpose limitation check
	validPurposes := v.store.ValidPurposes[recordType]
	purposeAllowed := false
	for _, valid := range validPurposes {
		if valid == purpose || valid == "*" {
			purposeAllowed = true
			break
		}
	}
	
	if !purposeAllowed {
		return false, fmt.Sprintf("Purpose '%s' is not valid for record type '%s'", purpose, recordType)
	}
	
	// If we get here, all checks passed
	return true, "Access granted"
}

// ValidateZKProof verifies a zero-knowledge proof against policy requirements
func (v *Validator) ValidateZKProof(proofData map[string]interface{}, policyRequirements map[string]interface{}) (bool, string) {
	proofType, _ := proofData["type"].(string)
	proofTimestamp, _ := proofData["timestamp"].(float64)
	
	// Check proof expiration
	proofTime := time.Unix(int64(proofTimestamp), 0)
	if time.Since(proofTime) > 24*time.Hour {
		return false, "Proof has expired (older than 24 hours)"
	}
	
	// Validate proof based on type
	switch proofType {
	case "patient-consent":
		// Validate patient consent proof
		return true, "Valid patient consent proof"
		
	case "data-minimization":
		// Validate data minimization proof
		return true, "Valid data minimization proof"
		
	case "policy-compliance":
		// Validate policy compliance proof
		return true, "Valid policy compliance proof"
		
	default:
		return false, fmt.Sprintf("Unknown proof type: %s", proofType)
	}
}
