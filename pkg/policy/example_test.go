package policy

import (
	"fmt"
	"os"
	"time"
)

// RunPolicyEngineDemo runs a demonstration of the policy engine
func RunPolicyEngineDemo() {
	// Create default configuration
	config := CreateDefaultConfig()

	// Initialize engine with configuration
	engine := InitializeEngine(config)

	// Define test scenarios
	scenarios := []struct {
		Country     string
		Role        string
		Action      string
		Description string
	}{
		{"IN", "general_doctor", "prescribe", "Indian doctor prescription"},
		{"IN", "general_doctor", "issue_certificate", "Indian general doctor certificate (should fail)"},
		{"CA", "specialist", "issue_certificate", "Canadian specialist certificate"},
		{"CA", "general_doctor", "issue_certificate", "Canadian general doctor certificate (should fail)"},
		{"US", "specialist", "prescribe", "US specialist prescription"},
		{"GB", "nurse", "prescribe", "UK nurse prescription (should fail)"},
		{"GB", "specialist", "diagnose", "UK specialist diagnosis"},
	}

	// Create table header for markdown output
	fmt.Println("| Country | Role | Action | Allowed | Validator |")
	fmt.Println("| ------- | ---- | ------ | ------- | --------- |")

	// Run validation for each scenario
	for _, scenario := range scenarios {
		// Create validation request
		req := ValidationRequest{
			Actor: ActorInfo{
				ID:   fmt.Sprintf("actor-%s-%s", scenario.Country, scenario.Role),
				Role: scenario.Role,
				Attributes: map[string]string{
					"country": scenario.Country,
				},
				ZKProofs: []string{"sample_proof_1", "sample_proof_2"},
			},
			Action:   scenario.Action,
			Location: scenario.Country,
			Resource: ResourceInfo{
				ID:   "resource-123",
				Type: "medical_record",
				Attributes: map[string]string{
					"patient_id":   "patient-456",
					"record_type":  "prescription",
					"created_date": time.Now().Format(time.RFC3339),
				},
				OwnerID: "patient-456",
			},
			Timestamp:     time.Now(),
			RequestID:     fmt.Sprintf("req-%d", time.Now().UnixNano()),
			ClientAddress: "192.168.1.1",
		}

		// Validate action
		result := engine.ValidateAction(req)

		// Get validator name
		validatorName := result.ValidatorName
		if !result.Allowed || validatorName == "" {
			if result.Allowed {
				validatorName = "None (not required)"
			} else {
				if scenario.Role == "general_doctor" && scenario.Action == "issue_certificate" {
					validatorName = fmt.Sprintf("%s (not enough role power)", GetValidatorNameForCountry(scenario.Country))
				} else {
					validatorName = "None"
				}
			}
		}

		// Print result in markdown table format
		allowedMark := "❌ No"
		if result.Allowed {
			allowedMark = "✅ Yes"
		}

		fmt.Printf("| %s | %s | %s | %s | %s |\n",
			scenario.Country,
			scenario.Role,
			scenario.Action,
			allowedMark,
			validatorName,
		)
	}
}

// GetValidatorNameForCountry returns the validator name for a country
func GetValidatorNameForCountry(country string) string {
	switch country {
	case "IN":
		return "Medical Council of India"
	case "CA":
		return "Health Canada"
	case "US":
		return "US Department of Health & Human Services"
	case "GB":
		return "National Health Service UK"
	default:
		return "Unknown"
	}
}

// ExportPolicyConfiguration exports the policy configuration to a file
func ExportPolicyConfiguration(format string, filePath string) error {
	config := CreateDefaultConfig()

	switch format {
	case "json":
		return SavePolicyConfigToFile(config, filePath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// If run as a standalone test
func main() {
	fmt.Println("🔍 Example Results from Policy Engine Test:")
	fmt.Println("")
	RunPolicyEngineDemo()
	fmt.Println("")

	// Export configuration if requested
	if len(os.Args) > 2 && os.Args[1] == "export" {
		format := "json"
		filePath := os.Args[2]
		if err := ExportPolicyConfiguration(format, filePath); err != nil {
			fmt.Printf("Error exporting configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration exported to %s in %s format\n", filePath, format)
	}
}
