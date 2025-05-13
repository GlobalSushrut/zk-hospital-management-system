package policy

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

// PolicyConfig represents the complete policy configuration
type PolicyConfig struct {
	Countries  []CountryRules    `json:"countries"`
	Validators []ValidatorConfig `json:"validators"`
	Roles      []RoleConfig      `json:"roles"`
}

// LoadPolicyConfigFromFile loads a policy configuration from a JSON file
func LoadPolicyConfigFromFile(filePath string) (*PolicyConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config PolicyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// InitializeEngine creates and initializes a PolicyEngine with the provided configuration
func InitializeEngine(config *PolicyConfig) *PolicyEngine {
	engine := NewPolicyEngine()

	// Add country rules
	for _, countryRule := range config.Countries {
		engine.AddCountryRules(countryRule)
	}

	// Add validators
	for _, validator := range config.Validators {
		engine.AddValidator(validator)
	}

	// Add role configurations
	for _, role := range config.Roles {
		engine.AddRoleConfig(role)
	}

	return engine
}

// SavePolicyConfigToFile saves a policy configuration to a JSON file
func SavePolicyConfigToFile(config *PolicyConfig, filePath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, data, os.ModePerm)
}

// CreateDefaultConfig creates a default policy configuration with sample data
func CreateDefaultConfig() *PolicyConfig {
	// Define sample validators
	validators := []ValidatorConfig{
		{
			ID:           "mci_validator",
			Name:         "Medical Council of India",
			Country:      "IN",
			ValidatesFor: []string{"prescribe", "diagnose", "refer"},
			PublicKey:    "sample_public_key_mci",
			API:          "https://api.mci.gov.in/validate",
		},
		{
			ID:           "health_canada",
			Name:         "Health Canada",
			Country:      "CA",
			ValidatesFor: []string{"prescribe", "diagnose", "refer", "issue_certificate"},
			PublicKey:    "sample_public_key_hc",
			API:          "https://api.healthcanada.ca/validate",
		},
		{
			ID:           "nhs_validator",
			Name:         "National Health Service UK",
			Country:      "GB",
			ValidatesFor: []string{"prescribe", "diagnose", "refer", "issue_certificate"},
			PublicKey:    "sample_public_key_nhs",
			API:          "https://api.nhs.uk/validate",
		},
		{
			ID:           "us_hhs",
			Name:         "US Department of Health & Human Services",
			Country:      "US",
			ValidatesFor: []string{"prescribe", "diagnose", "refer", "issue_certificate"},
			PublicKey:    "sample_public_key_hhs",
			API:          "https://api.hhs.gov/validate",
		},
	}

	// Define roles
	roles := []RoleConfig{
		{
			Name:           "general_doctor",
			Strength:       5,
			AllowedActions: []string{"prescribe", "diagnose", "refer"},
			CanDelegate:    false,
			RequiresMFA:    true,
		},
		{
			Name:           "specialist",
			Strength:       8,
			AllowedActions: []string{"prescribe", "diagnose", "refer", "issue_certificate"},
			CanDelegate:    true,
			RequiresMFA:    true,
		},
		{
			Name:           "nurse",
			Strength:       3,
			AllowedActions: []string{"record_vitals", "administer_medication"},
			CanDelegate:    false,
			RequiresMFA:    true,
		},
		{
			Name:           "admin",
			Strength:       2,
			AllowedActions: []string{"view_records", "schedule_appointment"},
			CanDelegate:    false,
			RequiresMFA:    true,
		},
		{
			Name:           "researcher",
			Strength:       4,
			AllowedActions: []string{"access_anonymized_data", "run_analytics"},
			CanDelegate:    false,
			RequiresMFA:    true,
		},
	}

	// Define country rules
	countries := []CountryRules{
		{
			Country:        "IN",
			RegulatoryBody: "Medical Council of India",
			RequiredFields: []string{"doctor_id", "patient_id", "prescription_details"},
			ActionRuleMap: map[string]ActionRule{
				"prescribe": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "mci_validator",
					AuditRequired:       true,
					RetentionPeriod:     365 * 24 * time.Hour, // 1 year
				},
				"diagnose": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "mci_validator",
					AuditRequired:       true,
					RetentionPeriod:     730 * 24 * time.Hour, // 2 years
				},
				"refer": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "mci_validator",
					AuditRequired:       true,
					RetentionPeriod:     365 * 24 * time.Hour, // 1 year
				},
				"issue_certificate": {
					RequiredRoles:      []string{"specialist"},
					MinimumRoleStrength: 8,
					RequiresValidator:   true,
					ValidatorID:         "mci_validator",
					AuditRequired:       true,
					RetentionPeriod:     1825 * 24 * time.Hour, // 5 years
				},
				"record_vitals": {
					RequiredRoles:      []string{"nurse", "general_doctor", "specialist"},
					MinimumRoleStrength: 3,
					RequiresValidator:   false,
					AuditRequired:       true,
					RetentionPeriod:     365 * 24 * time.Hour, // 1 year
				},
			},
			ValidatorMapping: map[string]string{
				"prescribe":        "mci_validator",
				"diagnose":         "mci_validator",
				"refer":            "mci_validator",
				"issue_certificate": "mci_validator",
			},
		},
		{
			Country:        "CA",
			RegulatoryBody: "Health Canada",
			RequiredFields: []string{"doctor_id", "patient_id", "prescription_details"},
			ActionRuleMap: map[string]ActionRule{
				"prescribe": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "health_canada",
					AuditRequired:       true,
					RetentionPeriod:     365 * 24 * time.Hour, // 1 year
				},
				"diagnose": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "health_canada",
					AuditRequired:       true,
					RetentionPeriod:     730 * 24 * time.Hour, // 2 years
				},
				"refer": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "health_canada",
					AuditRequired:       true,
					RetentionPeriod:     365 * 24 * time.Hour, // 1 year
				},
				"issue_certificate": {
					RequiredRoles:      []string{"specialist"},
					MinimumRoleStrength: 8,
					RequiresValidator:   true,
					ValidatorID:         "health_canada",
					AuditRequired:       true,
					RetentionPeriod:     1825 * 24 * time.Hour, // 5 years
				},
			},
			ValidatorMapping: map[string]string{
				"prescribe":        "health_canada",
				"diagnose":         "health_canada",
				"refer":            "health_canada",
				"issue_certificate": "health_canada",
			},
		},
		{
			Country:        "US",
			RegulatoryBody: "Department of Health & Human Services",
			RequiredFields: []string{"doctor_id", "patient_id", "prescription_details"},
			ActionRuleMap: map[string]ActionRule{
				"prescribe": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "us_hhs",
					AuditRequired:       true,
					RetentionPeriod:     365 * 24 * time.Hour, // 1 year
				},
				"diagnose": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "us_hhs",
					AuditRequired:       true,
					RetentionPeriod:     730 * 24 * time.Hour, // 2 years
				},
				"refer": {
					RequiredRoles:      []string{"general_doctor", "specialist"},
					MinimumRoleStrength: 5,
					RequiresValidator:   true,
					ValidatorID:         "us_hhs",
					AuditRequired:       true,
					RetentionPeriod:     365 * 24 * time.Hour, // 1 year
				},
				"issue_certificate": {
					RequiredRoles:      []string{"specialist"},
					MinimumRoleStrength: 8,
					RequiresValidator:   true,
					ValidatorID:         "us_hhs",
					AuditRequired:       true,
					RetentionPeriod:     1825 * 24 * time.Hour, // 5 years
				},
			},
			ValidatorMapping: map[string]string{
				"prescribe":        "us_hhs",
				"diagnose":         "us_hhs",
				"refer":            "us_hhs",
				"issue_certificate": "us_hhs",
			},
		},
	}

	return &PolicyConfig{
		Countries:  countries,
		Validators: validators,
		Roles:      roles,
	}
}
