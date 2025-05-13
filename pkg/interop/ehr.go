package interop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// EHRSystem represents the type of EHR system
type EHRSystem string

const (
	// EHRSystemEpic represents Epic EHR system
	EHRSystemEpic EHRSystem = "Epic"
	
	// EHRSystemCerner represents Cerner EHR system
	EHRSystemCerner EHRSystem = "Cerner"
	
	// EHRSystemAllscripts represents Allscripts EHR system
	EHRSystemAllscripts EHRSystem = "Allscripts"
)

// EHRClient provides an interface for interacting with EHR systems
type EHRClient struct {
	baseURL    string
	system     EHRSystem
	apiKey     string
	authToken  string
	client     *http.Client
	headers    map[string]string
	cache      *EHRCache
	fhirClient *FHIRClient // Some EHRs expose FHIR APIs
}

// EHRCache provides a caching layer for EHR API responses
type EHRCache struct {
	data       map[string]cacheEntry
	mutex      sync.RWMutex
	maxAge     time.Duration
	maxEntries int
}

type cacheEntry struct {
	data      interface{}
	timestamp time.Time
}

// NewEHRClient creates a new EHR client
func NewEHRClient(baseURL string, system EHRSystem) *EHRClient {
	// Ensure baseURL ends with a slash
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	
	client := &EHRClient{
		baseURL: baseURL,
		system:  system,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		cache: &EHRCache{
			data:       make(map[string]cacheEntry),
			maxAge:     5 * time.Minute,
			maxEntries: 1000,
		},
	}
	
	// For systems with FHIR APIs, initialize the FHIR client
	if system == EHRSystemEpic || system == EHRSystemCerner {
		client.fhirClient = NewFHIRClient(baseURL+"fhir/", FHIRR4)
	}
	
	return client
}

// SetAPIKey sets the API key for authentication
func (c *EHRClient) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
	c.headers["X-API-Key"] = apiKey
}

// SetAuthToken sets the authentication token
func (c *EHRClient) SetAuthToken(token string) {
	c.authToken = token
	c.headers["Authorization"] = "Bearer " + token
}

// Ping checks connectivity to the EHR system
func (c *EHRClient) Ping(ctx context.Context) error {
	// Make a simple request to check connectivity
	_, err := c.request(ctx, "GET", "ping", nil)
	return err
}

// AuthenticateWithCredentials authenticates with username and password
func (c *EHRClient) AuthenticateWithCredentials(ctx context.Context, username, password string) error {
	// Prepare authentication request
	authData := map[string]string{
		"username": username,
		"password": password,
	}
	
	// Different EHR systems have different authentication endpoints
	endpoint := ""
	switch c.system {
	case EHRSystemEpic:
		endpoint = "oauth2/token"
	case EHRSystemCerner:
		endpoint = "auth/token"
	case EHRSystemAllscripts:
		endpoint = "auth"
	default:
		return fmt.Errorf("unsupported EHR system: %s", c.system)
	}
	
	// Make the authentication request
	resp, err := c.request(ctx, "POST", endpoint, authData)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	// Parse the response
	var authResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	
	if err := json.Unmarshal(resp, &authResp); err != nil {
		return fmt.Errorf("failed to parse authentication response: %w", err)
	}
	
	// Set the authentication token
	c.SetAuthToken(authResp.AccessToken)
	
	// If the EHR system has a FHIR client, set the token there too
	if c.fhirClient != nil {
		c.fhirClient.SetAuthToken(authResp.AccessToken)
	}
	
	return nil
}

// GetPatient retrieves patient information by ID
func (c *EHRClient) GetPatient(ctx context.Context, patientID string) (map[string]interface{}, error) {
	// Check cache first
	cacheKey := "patient:" + patientID
	if data, found := c.getFromCache(cacheKey); found {
		return data.(map[string]interface{}), nil
	}
	
	// Use FHIR client if available for this EHR system
	if c.fhirClient != nil {
		patient, err := c.fhirClient.GetPatient(ctx, patientID)
		if err == nil {
			c.addToCache(cacheKey, patient)
			return patient, nil
		}
		// Fall back to native API if FHIR fails
	}
	
	// Determine the endpoint based on the EHR system
	endpoint := ""
	switch c.system {
	case EHRSystemEpic:
		endpoint = "api/patients/" + patientID
	case EHRSystemCerner:
		endpoint = "patients/" + patientID
	case EHRSystemAllscripts:
		endpoint = "patient/" + patientID
	default:
		return nil, fmt.Errorf("unsupported EHR system: %s", c.system)
	}
	
	// Make the request
	resp, err := c.request(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient: %w", err)
	}
	
	// Parse the response
	var patient map[string]interface{}
	if err := json.Unmarshal(resp, &patient); err != nil {
		return nil, fmt.Errorf("failed to parse patient data: %w", err)
	}
	
	// Cache the result
	c.addToCache(cacheKey, patient)
	
	return patient, nil
}

// SearchPatients searches for patients based on criteria
func (c *EHRClient) SearchPatients(ctx context.Context, criteria map[string]string) ([]map[string]interface{}, error) {
	// Build cache key from criteria
	var cacheKey strings.Builder
	cacheKey.WriteString("patientsearch:")
	for k, v := range criteria {
		cacheKey.WriteString(k + "=" + v + ";")
	}
	
	// Check cache
	if data, found := c.getFromCache(cacheKey.String()); found {
		return data.([]map[string]interface{}), nil
	}
	
	// Use FHIR client if available
	if c.fhirClient != nil {
		patients, err := c.fhirClient.SearchPatients(ctx, criteria)
		if err == nil {
			c.addToCache(cacheKey.String(), patients)
			return patients, nil
		}
		// Fall back to native API if FHIR fails
	}
	
	// Determine the endpoint and query parameters
	endpoint := ""
	queryParams := url.Values{}
	for k, v := range criteria {
		queryParams.Add(k, v)
	}
	
	switch c.system {
	case EHRSystemEpic:
		endpoint = "api/patients/search"
	case EHRSystemCerner:
		endpoint = "patients/search"
	case EHRSystemAllscripts:
		endpoint = "patient/search"
	default:
		return nil, fmt.Errorf("unsupported EHR system: %s", c.system)
	}
	
	// Add query parameters to the endpoint
	if len(queryParams) > 0 {
		endpoint += "?" + queryParams.Encode()
	}
	
	// Make the request
	resp, err := c.request(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search patients: %w", err)
	}
	
	// Parse the response
	var searchResp struct {
		Patients []map[string]interface{} `json:"patients"`
		Count    int                      `json:"count"`
	}
	
	if err := json.Unmarshal(resp, &searchResp); err != nil {
		// Try to parse as an array if struct parsing fails
		var patients []map[string]interface{}
		if err2 := json.Unmarshal(resp, &patients); err2 != nil {
			return nil, fmt.Errorf("failed to parse patient search results: %w", err)
		}
		
		c.addToCache(cacheKey.String(), patients)
		return patients, nil
	}
	
	c.addToCache(cacheKey.String(), searchResp.Patients)
	return searchResp.Patients, nil
}

// GetEncounter retrieves encounter information by ID
func (c *EHRClient) GetEncounter(ctx context.Context, encounterID string) (map[string]interface{}, error) {
	// Check cache
	cacheKey := "encounter:" + encounterID
	if data, found := c.getFromCache(cacheKey); found {
		return data.(map[string]interface{}), nil
	}
	
	// Determine the endpoint
	endpoint := ""
	switch c.system {
	case EHRSystemEpic:
		endpoint = "api/encounters/" + encounterID
	case EHRSystemCerner:
		endpoint = "encounters/" + encounterID
	case EHRSystemAllscripts:
		endpoint = "encounter/" + encounterID
	default:
		return nil, fmt.Errorf("unsupported EHR system: %s", c.system)
	}
	
	// Make the request
	resp, err := c.request(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get encounter: %w", err)
	}
	
	// Parse the response
	var encounter map[string]interface{}
	if err := json.Unmarshal(resp, &encounter); err != nil {
		return nil, fmt.Errorf("failed to parse encounter data: %w", err)
	}
	
	// Cache the result
	c.addToCache(cacheKey, encounter)
	
	return encounter, nil
}

// GetPatientEncounters retrieves encounters for a patient
func (c *EHRClient) GetPatientEncounters(ctx context.Context, patientID string) ([]map[string]interface{}, error) {
	// Check cache
	cacheKey := "patient:" + patientID + ":encounters"
	if data, found := c.getFromCache(cacheKey); found {
		return data.([]map[string]interface{}), nil
	}
	
	// Determine the endpoint
	endpoint := ""
	switch c.system {
	case EHRSystemEpic:
		endpoint = "api/patients/" + patientID + "/encounters"
	case EHRSystemCerner:
		endpoint = "patients/" + patientID + "/encounters"
	case EHRSystemAllscripts:
		endpoint = "patient/" + patientID + "/encounters"
	default:
		return nil, fmt.Errorf("unsupported EHR system: %s", c.system)
	}
	
	// Make the request
	resp, err := c.request(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient encounters: %w", err)
	}
	
	// Parse the response
	var encounterResp struct {
		Encounters []map[string]interface{} `json:"encounters"`
		Count      int                      `json:"count"`
	}
	
	if err := json.Unmarshal(resp, &encounterResp); err != nil {
		// Try to parse as an array if struct parsing fails
		var encounters []map[string]interface{}
		if err2 := json.Unmarshal(resp, &encounters); err2 != nil {
			return nil, fmt.Errorf("failed to parse patient encounters: %w", err)
		}
		
		c.addToCache(cacheKey, encounters)
		return encounters, nil
	}
	
	c.addToCache(cacheKey, encounterResp.Encounters)
	return encounterResp.Encounters, nil
}

// GetMedications retrieves medications for a patient
func (c *EHRClient) GetMedications(ctx context.Context, patientID string) ([]map[string]interface{}, error) {
	// Check cache
	cacheKey := "patient:" + patientID + ":medications"
	if data, found := c.getFromCache(cacheKey); found {
		return data.([]map[string]interface{}), nil
	}
	
	// Determine the endpoint
	endpoint := ""
	switch c.system {
	case EHRSystemEpic:
		endpoint = "api/patients/" + patientID + "/medications"
	case EHRSystemCerner:
		endpoint = "patients/" + patientID + "/medications"
	case EHRSystemAllscripts:
		endpoint = "patient/" + patientID + "/medications"
	default:
		return nil, fmt.Errorf("unsupported EHR system: %s", c.system)
	}
	
	// Make the request
	resp, err := c.request(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient medications: %w", err)
	}
	
	// Parse the response
	var medicationResp struct {
		Medications []map[string]interface{} `json:"medications"`
		Count       int                      `json:"count"`
	}
	
	if err := json.Unmarshal(resp, &medicationResp); err != nil {
		// Try to parse as an array if struct parsing fails
		var medications []map[string]interface{}
		if err2 := json.Unmarshal(resp, &medications); err2 != nil {
			return nil, fmt.Errorf("failed to parse patient medications: %w", err)
		}
		
		c.addToCache(cacheKey, medications)
		return medications, nil
	}
	
	c.addToCache(cacheKey, medicationResp.Medications)
	return medicationResp.Medications, nil
}

// GetDocuments retrieves documents for a patient
func (c *EHRClient) GetDocuments(ctx context.Context, patientID string) ([]map[string]interface{}, error) {
	// Check cache
	cacheKey := "patient:" + patientID + ":documents"
	if data, found := c.getFromCache(cacheKey); found {
		return data.([]map[string]interface{}), nil
	}
	
	// Determine the endpoint
	endpoint := ""
	switch c.system {
	case EHRSystemEpic:
		endpoint = "api/patients/" + patientID + "/documents"
	case EHRSystemCerner:
		endpoint = "patients/" + patientID + "/documents"
	case EHRSystemAllscripts:
		endpoint = "patient/" + patientID + "/documents"
	default:
		return nil, fmt.Errorf("unsupported EHR system: %s", c.system)
	}
	
	// Make the request
	resp, err := c.request(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient documents: %w", err)
	}
	
	// Parse the response
	var documentResp struct {
		Documents []map[string]interface{} `json:"documents"`
		Count     int                      `json:"count"`
	}
	
	if err := json.Unmarshal(resp, &documentResp); err != nil {
		// Try to parse as an array if struct parsing fails
		var documents []map[string]interface{}
		if err2 := json.Unmarshal(resp, &documents); err2 != nil {
			return nil, fmt.Errorf("failed to parse patient documents: %w", err)
		}
		
		c.addToCache(cacheKey, documents)
		return documents, nil
	}
	
	c.addToCache(cacheKey, documentResp.Documents)
	return documentResp.Documents, nil
}

// ScheduleAppointment schedules an appointment for a patient
func (c *EHRClient) ScheduleAppointment(ctx context.Context, appointment map[string]interface{}) (map[string]interface{}, error) {
	// Determine the endpoint
	endpoint := ""
	switch c.system {
	case EHRSystemEpic:
		endpoint = "api/appointments"
	case EHRSystemCerner:
		endpoint = "appointments"
	case EHRSystemAllscripts:
		endpoint = "appointment"
	default:
		return nil, fmt.Errorf("unsupported EHR system: %s", c.system)
	}
	
	// Make the request
	resp, err := c.request(ctx, "POST", endpoint, appointment)
	if err != nil {
		return nil, fmt.Errorf("failed to schedule appointment: %w", err)
	}
	
	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse appointment response: %w", err)
	}
	
	return result, nil
}

// request is a helper method to make HTTP requests to the EHR API
func (c *EHRClient) request(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	// Prepare URL
	requestURL := c.baseURL + endpoint
	
	// Prepare body if provided
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = strings.NewReader(string(bodyBytes))
	} else {
		// Use empty reader for nil body to avoid nil pointer dereference
		bodyReader = strings.NewReader("")
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	
	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	respBody := make([]byte, 0)
	buffer := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			respBody = append(respBody, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}
	
	// Check for error response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

// addToCache adds data to the cache
func (c *EHRClient) addToCache(key string, data interface{}) {
	c.cache.mutex.Lock()
	defer c.cache.mutex.Unlock()
	
	// Add to cache
	c.cache.data[key] = cacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
	
	// Check if we need to evict old entries
	if len(c.cache.data) > c.cache.maxEntries {
		c.evictOldestCache()
	}
}

// getFromCache gets data from the cache
func (c *EHRClient) getFromCache(key string) (interface{}, bool) {
	c.cache.mutex.RLock()
	defer c.cache.mutex.RUnlock()
	
	entry, found := c.cache.data[key]
	if !found {
		return nil, false
	}
	
	// Check if entry is expired
	if time.Since(entry.timestamp) > c.cache.maxAge {
		return nil, false
	}
	
	return entry.data, true
}

// evictOldestCache removes the oldest entries from the cache
func (c *EHRClient) evictOldestCache() {
	// Find the oldest entries
	type keyAge struct {
		key       string
		timestamp time.Time
	}
	
	entries := make([]keyAge, 0, len(c.cache.data))
	for k, v := range c.cache.data {
		entries = append(entries, keyAge{k, v.timestamp})
	}
	
	// Sort by timestamp (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].timestamp.After(entries[j].timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	// Remove oldest 10% or at least 1 entry
	numToRemove := len(c.cache.data) / 10
	if numToRemove < 1 {
		numToRemove = 1
	}
	
	for i := 0; i < numToRemove && i < len(entries); i++ {
		delete(c.cache.data, entries[i].key)
	}
}

// EHRAdapter provides a unified interface for different EHR systems
type EHRAdapter struct {
	clients map[EHRSystem]*EHRClient
}

// NewEHRAdapter creates a new EHR adapter
func NewEHRAdapter() *EHRAdapter {
	return &EHRAdapter{
		clients: make(map[EHRSystem]*EHRClient),
	}
}

// RegisterClient registers an EHR client with the adapter
func (a *EHRAdapter) RegisterClient(system EHRSystem, client *EHRClient) {
	a.clients[system] = client
}

// GetClient gets an EHR client for a specific system
func (a *EHRAdapter) GetClient(system EHRSystem) (*EHRClient, error) {
	client, found := a.clients[system]
	if !found {
		return nil, fmt.Errorf("no client registered for EHR system: %s", system)
	}
	return client, nil
}

// GetPatientAcrossSystems retrieves patient information from all registered EHR systems
func (a *EHRAdapter) GetPatientAcrossSystems(ctx context.Context, patientID string) (map[EHRSystem]map[string]interface{}, error) {
	results := make(map[EHRSystem]map[string]interface{})
	var firstError error
	
	// Query each system
	for system, client := range a.clients {
		patient, err := client.GetPatient(ctx, patientID)
		if err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("error from %s: %w", system, err)
			}
			continue
		}
		results[system] = patient
	}
	
	if len(results) == 0 && firstError != nil {
		return nil, firstError
	}
	
	return results, nil
}

// MergePatientsAcrossSystems merges patient data from all registered EHR systems
func (a *EHRAdapter) MergePatientsAcrossSystems(ctx context.Context, patientID string) (map[string]interface{}, error) {
	// Get patient data from all systems
	patientsData, err := a.GetPatientAcrossSystems(ctx, patientID)
	if err != nil && len(patientsData) == 0 {
		return nil, err
	}
	
	// Merge the data
	mergedPatient := make(map[string]interface{})
	
	// Simple merge strategy - take non-empty fields from any system
	// In a real implementation, this would be more sophisticated
	for _, patientData := range patientsData {
		for field, value := range patientData {
			if _, exists := mergedPatient[field]; !exists {
				// Field doesn't exist in merged data yet
				mergedPatient[field] = value
			} else if valueStr, ok := value.(string); ok && valueStr != "" {
				// Field exists but is empty in merged data and non-empty in this system
				if existingStr, ok := mergedPatient[field].(string); ok && existingStr == "" {
					mergedPatient[field] = value
				}
			}
		}
	}
	
	// Add a source field to indicate this is merged data
	mergedPatient["source"] = "merged"
	
	return mergedPatient, nil
}
