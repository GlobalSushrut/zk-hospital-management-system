package policy

import (
	"fmt"
)

// OracleIntegration provides integration between the policy engine and oracle chain validator
type OracleIntegration struct {
	policyEngine *PolicyEngine
}

// OracleValidationRequest represents a request to validate a policy against an oracle agreement
type OracleValidationRequest struct {
	PolicyRequest ValidationRequest
	AgreementID   string
	ClauseIDs     []string
}

// OracleValidationResult represents the result of policy validation with oracle integration
type OracleValidationResult struct {
	PolicyResult      ValidationResult
	OracleValidated   bool
	AgreementID       string
	ValidClauses      []string
	InvalidClauses    []string
	ValidationDetails map[string]string
}

// NewOracleIntegration creates a new oracle integration
func NewOracleIntegration(engine *PolicyEngine) *OracleIntegration {
	return &OracleIntegration{
		policyEngine: engine,
	}
}

// ValidatePolicyWithOracle validates a policy request against both policy engine and oracle
func (oi *OracleIntegration) ValidatePolicyWithOracle(request OracleValidationRequest) OracleValidationResult {
	// First validate with policy engine
	policyResult := oi.policyEngine.ValidateAction(request.PolicyRequest)
	
	// Prepare the result
	result := OracleValidationResult{
		PolicyResult:      policyResult,
		OracleValidated:   false,
		AgreementID:       request.AgreementID,
		ValidClauses:      []string{},
		InvalidClauses:    []string{},
		ValidationDetails: make(map[string]string),
	}
	
	// If policy validation failed, we don't need to check the oracle
	if !policyResult.Allowed {
		result.ValidationDetails["policy_failure_reason"] = policyResult.Reason
		return result
	}
	
	// Proceed with oracle validation 
	// In a real implementation, this would call the Oracle Chain Validator
	// For demonstration, we'll simulate the validation
	
	// Assuming all clauses pass validation for simplicity
	// In a real implementation, each clause would be validated individually
	for _, clauseID := range request.ClauseIDs {
		// Simulate oracle validation
		clauseValid := simulateOracleClauseValidation(
			request.AgreementID, 
			clauseID, 
			request.PolicyRequest.Location,
			request.PolicyRequest.Actor.Role,
			request.PolicyRequest.Action,
		)
		
		if clauseValid {
			result.ValidClauses = append(result.ValidClauses, clauseID)
			result.ValidationDetails[clauseID] = "Clause validated successfully"
		} else {
			result.InvalidClauses = append(result.InvalidClauses, clauseID)
			result.ValidationDetails[clauseID] = "Clause validation failed"
		}
	}
	
	// Mark as oracle validated if all clauses passed
	result.OracleValidated = len(result.InvalidClauses) == 0
	
	return result
}

// GenerateOracleClausesForPolicy generates oracle clauses based on policy rules
func (oi *OracleIntegration) GenerateOracleClausesForPolicy(countryCode string, action string) []OracleClause {
	country, exists := oi.policyEngine.geographicRules[countryCode]
	if !exists {
		return []OracleClause{}
	}
	
	actionRule, exists := country.ActionRuleMap[action]
	if !exists {
		return []OracleClause{}
	}
	
	// Generate clauses based on policy rules
	clauses := []OracleClause{
		{
			ID:          fmt.Sprintf("%s-%s-role-check", countryCode, action),
			Type:        "role_validation",
			Description: fmt.Sprintf("Validates that actor has required role for %s in %s", action, countryCode),
			Condition:   fmt.Sprintf("actor.role IN (%s) AND actor.role.strength >= %d", 
				joinStrings(actionRule.RequiredRoles), actionRule.MinimumRoleStrength),
		},
	}
	
	// Add validator clause if required
	if actionRule.RequiresValidator {
		validatorID := actionRule.ValidatorID
		if validatorID == "" {
			validatorID = country.ValidatorMapping[action]
		}
		
		if validatorID != "" {
			validator, exists := oi.policyEngine.validators[validatorID]
			if exists {
				clauses = append(clauses, OracleClause{
					ID:          fmt.Sprintf("%s-%s-validator", countryCode, action),
					Type:        "validator_approval",
					Description: fmt.Sprintf("Requires approval from %s for %s in %s", validator.Name, action, countryCode),
					Condition:   fmt.Sprintf("validator.id == '%s' AND validator.approved == true", validatorID),
				})
			}
		}
	}
	
	// Add audit clause if required
	if actionRule.AuditRequired {
		clauses = append(clauses, OracleClause{
			ID:          fmt.Sprintf("%s-%s-audit", countryCode, action),
			Type:        "audit_requirement",
			Description: fmt.Sprintf("Requires audit trail for %s in %s", action, countryCode),
			Condition:   "audit.enabled == true",
		})
	}
	
	return clauses
}

// OracleClause represents a clause in an oracle agreement
type OracleClause struct {
	ID          string
	Type        string
	Description string
	Condition   string
}

// Helper functions

// joinStrings joins string array elements with commas, and quotes each element
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	
	result := fmt.Sprintf("'%s'", strs[0])
	for i := 1; i < len(strs); i++ {
		result += fmt.Sprintf(", '%s'", strs[i])
	}
	
	return result
}

// simulateOracleClauseValidation simulates validation of an oracle clause
// In a real implementation, this would call the actual Oracle Chain Validator
func simulateOracleClauseValidation(
	agreementID string, 
	clauseID string,
	country string,
	role string, 
	action string,
) bool {
	// Simulate failure for specific combinations
	if country == "IN" && role == "general_doctor" && action == "issue_certificate" {
		return false
	}
	
	if country == "CA" && role == "general_doctor" && action == "issue_certificate" {
		return false
	}
	
	// Default to success for demonstration
	return true
}
