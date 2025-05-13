package policy

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// PolicyEngine provides location-aware, role-based policy validation
type PolicyEngine struct {
	mu              sync.RWMutex
	geographicRules map[string]CountryRules    // country code -> rules
	validators      map[string]ValidatorConfig // validator ID -> config
	roleScopes      map[string]RoleConfig      // role -> config
	validationCache map[string]CacheEntry      // cache for validation results
}

// CountryRules defines the regulatory framework for a specific country
type CountryRules struct {
	Country          string
	RegulatoryBody   string
	RequiredFields   []string
	ActionRuleMap    map[string]ActionRule // action -> rules
	ValidatorMapping map[string]string     // action -> validator ID
}

// ActionRule defines the requirements for a specific action
type ActionRule struct {
	RequiredRoles      []string
	MinimumRoleStrength int
	RequiresValidator   bool
	ValidatorID         string
	AuditRequired       bool
	RetentionPeriod     time.Duration
}

// ValidatorConfig defines a validation authority
type ValidatorConfig struct {
	ID           string
	Name         string
	Country      string
	ValidatesFor []string // list of action types this validator can validate
	PublicKey    string   // verification key
	API          string   // API endpoint for validation
}

// RoleConfig defines permissions for a specific role
type RoleConfig struct {
	Name           string
	Strength       int // hierarchical strength (higher = more authority)
	AllowedActions []string
	CanDelegate    bool
	RequiresMFA    bool
}

// CacheEntry for validation results
type CacheEntry struct {
	Result      bool
	ValidatorID string
	Timestamp   time.Time
	TTL         time.Duration
}

// ValidationRequest contains all data needed for policy validation
type ValidationRequest struct {
	Actor         ActorInfo
	Action        string
	Location      string // country code
	Resource      ResourceInfo
	Timestamp     time.Time
	RequestID     string
	ClientAddress string
}

// ActorInfo contains information about the requester
type ActorInfo struct {
	ID         string
	Role       string
	Attributes map[string]string
	ZKProofs   []string
}

// ResourceInfo contains information about the accessed resource
type ResourceInfo struct {
	ID         string
	Type       string
	Attributes map[string]string
	OwnerID    string
}

// ValidationResult contains the outcome of a policy validation
type ValidationResult struct {
	Allowed        bool
	Reason         string
	ValidatorID    string
	ValidatorName  string
	ValidationTime time.Time
	AuditRecord    *AuditRecord
}

// AuditRecord contains audit information for a validation
type AuditRecord struct {
	RequestID      string
	ActorID        string
	ActorRole      string
	Action         string
	ResourceID     string
	ResourceType   string
	Location       string
	ValidationTime time.Time
	Allowed        bool
	ValidatorID    string
	ClientAddress  string
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		geographicRules: make(map[string]CountryRules),
		validators:      make(map[string]ValidatorConfig),
		roleScopes:      make(map[string]RoleConfig),
		validationCache: make(map[string]CacheEntry),
	}
}

// AddCountryRules adds or updates rules for a country
func (pe *PolicyEngine) AddCountryRules(rules CountryRules) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.geographicRules[strings.ToUpper(rules.Country)] = rules
}

// AddValidator adds or updates a validator
func (pe *PolicyEngine) AddValidator(validator ValidatorConfig) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.validators[validator.ID] = validator
}

// AddRoleConfig adds or updates a role configuration
func (pe *PolicyEngine) AddRoleConfig(role RoleConfig) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.roleScopes[role.Name] = role
}

// ValidateAction checks if the requested action is allowed
func (pe *PolicyEngine) ValidateAction(req ValidationRequest) ValidationResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// Create a default negative result
	result := ValidationResult{
		Allowed:        false,
		Reason:         "Validation failed",
		ValidationTime: time.Now(),
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s:%s:%s", req.Actor.ID, req.Action, req.Location, req.Resource.ID)
	if entry, found := pe.validationCache[cacheKey]; found {
		if time.Since(entry.Timestamp) < entry.TTL {
			result.Allowed = entry.Result
			result.ValidatorID = entry.ValidatorID
			if v, exists := pe.validators[entry.ValidatorID]; exists {
				result.ValidatorName = v.Name
			}
			if result.Allowed {
				result.Reason = "Allowed (cached)"
			} else {
				result.Reason = "Denied (cached)"
			}
			return result
		}
		// Cache expired, remove it
		delete(pe.validationCache, cacheKey)
	}

	// Get country rules
	countryRules, countryExists := pe.geographicRules[strings.ToUpper(req.Location)]
	if !countryExists {
		result.Reason = fmt.Sprintf("No rules defined for country: %s", req.Location)
		return result
	}

	// Get action rules
	actionRule, actionExists := countryRules.ActionRuleMap[req.Action]
	if !actionExists {
		result.Reason = fmt.Sprintf("Action %s not defined for country %s", req.Action, req.Location)
		return result
	}

	// Check role permissions
	roleConfig, roleExists := pe.roleScopes[req.Actor.Role]
	if !roleExists {
		result.Reason = fmt.Sprintf("Role %s not defined", req.Actor.Role)
		return result
	}

	// Check if action is allowed for role
	actionAllowed := false
	for _, allowedAction := range roleConfig.AllowedActions {
		if allowedAction == req.Action || allowedAction == "*" {
			actionAllowed = true
			break
		}
	}
	if !actionAllowed {
		result.Reason = fmt.Sprintf("Action %s not allowed for role %s", req.Action, req.Actor.Role)
		return result
	}

	// Check role strength
	if roleConfig.Strength < actionRule.MinimumRoleStrength {
		result.Reason = fmt.Sprintf("Role %s has insufficient strength for action %s", req.Actor.Role, req.Action)
		return result
	}

	// Get validator if required
	var validatorConfig ValidatorConfig
	if actionRule.RequiresValidator {
		validatorID := actionRule.ValidatorID
		if validatorID == "" {
			validatorID = countryRules.ValidatorMapping[req.Action]
		}
		if validatorID == "" {
			result.Reason = fmt.Sprintf("No validator defined for action %s in country %s", req.Action, req.Location)
			return result
		}

		var validatorExists bool
		validatorConfig, validatorExists = pe.validators[validatorID]
		if !validatorExists {
			result.Reason = fmt.Sprintf("Validator %s not found", validatorID)
			return result
		}

		// Validator exists - in a real system, we would call the validator's API here
		// For now, we'll simulate validation success
		result.ValidatorID = validatorID
		result.ValidatorName = validatorConfig.Name
	}

	// All checks passed
	result.Allowed = true
	result.Reason = "Action allowed"

	// Create audit record if required
	if actionRule.AuditRequired {
		result.AuditRecord = &AuditRecord{
			RequestID:      req.RequestID,
			ActorID:        req.Actor.ID,
			ActorRole:      req.Actor.Role,
			Action:         req.Action,
			ResourceID:     req.Resource.ID,
			ResourceType:   req.Resource.Type,
			Location:       req.Location,
			ValidationTime: result.ValidationTime,
			Allowed:        result.Allowed,
			ValidatorID:    result.ValidatorID,
			ClientAddress:  req.ClientAddress,
		}
	}

	// Cache the result
	pe.validationCache[cacheKey] = CacheEntry{
		Result:      result.Allowed,
		ValidatorID: result.ValidatorID,
		Timestamp:   time.Now(),
		TTL:         5 * time.Minute, // Cache for 5 minutes
	}

	return result
}

// GetAllowedActions returns all actions allowed for a role in a specific country
func (pe *PolicyEngine) GetAllowedActions(role, country string) map[string]bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	results := make(map[string]bool)
	
	roleConfig, roleExists := pe.roleScopes[role]
	if !roleExists {
		return results
	}

	countryRules, countryExists := pe.geographicRules[strings.ToUpper(country)]
	if !countryExists {
		return results
	}

	for action, actionRule := range countryRules.ActionRuleMap {
		// Check if action is in role's allowed actions
		actionAllowed := false
		for _, allowedAction := range roleConfig.AllowedActions {
			if allowedAction == action || allowedAction == "*" {
				actionAllowed = true
				break
			}
		}

		// Check role strength
		if actionAllowed && roleConfig.Strength >= actionRule.MinimumRoleStrength {
			results[action] = true
		} else {
			results[action] = false
		}
	}

	return results
}

// GetValidatorForAction returns the validator for a specific action in a country
func (pe *PolicyEngine) GetValidatorForAction(action, country string) (ValidatorConfig, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var empty ValidatorConfig

	countryRules, countryExists := pe.geographicRules[strings.ToUpper(country)]
	if !countryExists {
		return empty, false
	}

	actionRule, actionExists := countryRules.ActionRuleMap[action]
	if !actionExists {
		return empty, false
	}

	if !actionRule.RequiresValidator {
		return empty, false
	}

	validatorID := actionRule.ValidatorID
	if validatorID == "" {
		validatorID = countryRules.ValidatorMapping[action]
	}
	if validatorID == "" {
		return empty, false
	}

	validator, exists := pe.validators[validatorID]
	return validator, exists
}

// ClearCache clears the validation cache
func (pe *PolicyEngine) ClearCache() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.validationCache = make(map[string]CacheEntry)
}
