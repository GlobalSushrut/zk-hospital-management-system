package policy

// PolicyStore stores and manages healthcare data access policies
type PolicyStore struct {
	RoleAccess             map[string][]string
	DataSensitivityLevels  map[string]string
	JurisdictionAgreements map[string][]string
	ValidPurposes          map[string][]string
	AccessRules            []AccessRule
}

// AccessRule defines a rule for data access
type AccessRule struct {
	Role        string
	RecordType  string
	Sensitivity string
	Purpose     string
	Action      string
	Condition   func(map[string]interface{}) bool
}

// NewPolicyStore creates a new policy store with default policies
func NewPolicyStore() *PolicyStore {
	store := &PolicyStore{
		RoleAccess: map[string][]string{
			"physician":       {"*"},                                        // Physicians can access all record types
			"nurse":           {"medical_history", "medication", "vitals"},  // Nurses have limited access
			"researcher":      {"anonymized_data"},                          // Researchers can only see anonymized data
			"insurance_agent": {"billing", "claims", "coverage"},            // Insurance agents see financial info
			"admin":           {"demographic", "billing", "appointment"},    // Admins see administrative data
			"patient":         {"*"},                                        // Patients can see their own records
			"family":          {"demographic", "appointment"},               // Family members see basic info
		},
		
		DataSensitivityLevels: map[string]string{
			"demographic":     "low",
			"appointment":     "low",
			"billing":         "medium",
			"claims":          "medium",
			"coverage":        "medium",
			"vitals":          "medium",
			"medication":      "high",
			"medical_history": "high",
			"lab_results":     "high",
			"mental_health":   "high",
			"genetic_data":    "high",
		},
		
		JurisdictionAgreements: map[string][]string{
			"california": {"new_york", "illinois", "texas"},
			"new_york":   {"california", "massachusetts", "illinois"},
			"texas":      {"california", "florida"},
			"florida":    {"texas", "georgia"},
			"illinois":   {"california", "new_york", "ohio"},
		},
		
		ValidPurposes: map[string][]string{
			"medical_history": {"treatment", "diagnosis", "emergency", "research"},
			"medication":      {"treatment", "emergency"},
			"lab_results":     {"treatment", "diagnosis", "research"},
			"vitals":          {"treatment", "diagnosis", "emergency", "research"},
			"mental_health":   {"treatment", "emergency"},
			"genetic_data":    {"treatment", "research"},
			"billing":         {"payment", "insurance", "audit"},
			"claims":          {"payment", "insurance", "audit"},
			"demographic":     {"treatment", "administration", "identification", "*"},
			"appointment":     {"scheduling", "administration", "*"},
		},
		
		AccessRules: []AccessRule{},
	}
	
	// Initialize default access rules
	store.initializeDefaultRules()
	
	return store
}

// initializeDefaultRules sets up default access control rules
func (s *PolicyStore) initializeDefaultRules() {
	// Rule: Emergency access overrides most restrictions
	s.AccessRules = append(s.AccessRules, AccessRule{
		Role:       "*",
		RecordType: "*",
		Action:     "read",
		Condition: func(ctx map[string]interface{}) bool {
			emergency, _ := ctx["emergency"].(bool)
			return emergency
		},
	})
	
	// Rule: Physicians can access all patient data with patient consent
	s.AccessRules = append(s.AccessRules, AccessRule{
		Role:       "physician",
		RecordType: "*",
		Action:     "read",
		Condition: func(ctx map[string]interface{}) bool {
			hasConsent, _ := ctx["has_consent"].(bool)
			return hasConsent
		},
	})
	
	// Rule: Require two-factor auth for high sensitivity data
	s.AccessRules = append(s.AccessRules, AccessRule{
		Role:        "*",
		RecordType:  "*",
		Sensitivity: "high",
		Action:      "read",
		Condition: func(ctx map[string]interface{}) bool {
			authMethod, _ := ctx["auth_method"].(string)
			return authMethod == "two_factor" || authMethod == "biometric"
		},
	})
}

// HasJurisdictionAgreement checks if two jurisdictions have a data sharing agreement
func (s *PolicyStore) HasJurisdictionAgreement(from, to string) bool {
	if from == to {
		return true // Same jurisdiction
	}
	
	// Check for agreement from -> to
	if agreements, exists := s.JurisdictionAgreements[from]; exists {
		for _, agreement := range agreements {
			if agreement == to {
				return true
			}
		}
	}
	
	return false
}

// GetSensitivityLevel returns the sensitivity level for a record type
func (s *PolicyStore) GetSensitivityLevel(recordType string) string {
	if level, exists := s.DataSensitivityLevels[recordType]; exists {
		return level
	}
	return "medium" // Default sensitivity level
}

// IsPurposeValid checks if a purpose is valid for a record type
func (s *PolicyStore) IsPurposeValid(recordType, purpose string) bool {
	validPurposes, exists := s.ValidPurposes[recordType]
	if !exists {
		return false
	}
	
	for _, validPurpose := range validPurposes {
		if validPurpose == purpose || validPurpose == "*" {
			return true
		}
	}
	
	return false
}

// CanRoleAccessRecord checks if a role can access a record type
func (s *PolicyStore) CanRoleAccessRecord(role, recordType string) bool {
	allowedTypes, exists := s.RoleAccess[role]
	if !exists {
		return false
	}
	
	for _, allowedType := range allowedTypes {
		if allowedType == recordType || allowedType == "*" {
			return true
		}
	}
	
	return false
}
