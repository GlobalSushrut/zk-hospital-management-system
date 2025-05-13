package interop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

// FHIRVersion represents a FHIR specification version
type FHIRVersion string

const (
	// FHIR versions
	FHIRR2 FHIRVersion = "R2"
	FHIRR3 FHIRVersion = "R3"
	FHIRR4 FHIRVersion = "R4"
	FHIRR5 FHIRVersion = "R5"
	
	// Default timeout for FHIR API calls
	defaultTimeout = 30 * time.Second
)

// FHIRClient represents a client for interacting with FHIR servers
type FHIRClient struct {
	baseURL   string
	version   FHIRVersion
	authToken string
	client    *http.Client
	headers   map[string]string
	cacheEnabled bool
	cache     *ResourceCache
}

// ResourceCache provides caching for FHIR resources
type ResourceCache struct {
	resources  map[string]map[string]CachedResource
	mutex      sync.RWMutex
	maxSize    int
	defaultTTL time.Duration
}

// CachedResource represents a cached FHIR resource
type CachedResource struct {
	Data      map[string]interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// NewFHIRClient creates a new FHIR client
func NewFHIRClient(baseURL string, version FHIRVersion) *FHIRClient {
	// Ensure baseURL ends with a slash
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	
	return &FHIRClient{
		baseURL: baseURL,
		version: version,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		headers: map[string]string{
			"Accept":       "application/fhir+json",
			"Content-Type": "application/fhir+json",
		},
		cacheEnabled: true,
		cache: &ResourceCache{
			resources:  make(map[string]map[string]CachedResource),
			maxSize:    1000,
			defaultTTL: 5 * time.Minute,
		},
	}
}

// SetAuthToken sets the authorization token for the client
func (fc *FHIRClient) SetAuthToken(token string) {
	fc.authToken = token
	fc.headers["Authorization"] = "Bearer " + token
}

// SetTimeout sets the timeout for HTTP requests
func (fc *FHIRClient) SetTimeout(timeout time.Duration) {
	fc.client.Timeout = timeout
}

// SetHeader sets a custom HTTP header for requests
func (fc *FHIRClient) SetHeader(key, value string) {
	fc.headers[key] = value
}

// EnableCache enables or disables the resource cache
func (fc *FHIRClient) EnableCache(enabled bool) {
	fc.cacheEnabled = enabled
}

// SetCacheTTL sets the default time-to-live for cached resources
func (fc *FHIRClient) SetCacheTTL(ttl time.Duration) {
	fc.cache.defaultTTL = ttl
}

// ClearCache clears all cached resources
func (fc *FHIRClient) ClearCache() {
	fc.cache.mutex.Lock()
	defer fc.cache.mutex.Unlock()
	fc.cache.resources = make(map[string]map[string]CachedResource)
}

// GetPatient gets a patient by ID
func (fc *FHIRClient) GetPatient(ctx context.Context, id string) (map[string]interface{}, error) {
	return fc.GetResource(ctx, "Patient", id)
}

// GetObservation gets an observation by ID
func (fc *FHIRClient) GetObservation(ctx context.Context, id string) (map[string]interface{}, error) {
	return fc.GetResource(ctx, "Observation", id)
}

// GetResource gets a resource by type and ID
func (fc *FHIRClient) GetResource(ctx context.Context, resourceType, id string) (map[string]interface{}, error) {
	// Check cache first if enabled
	if fc.cacheEnabled {
		if resource, found := fc.getFromCache(resourceType, id); found {
			return resource, nil
		}
	}
	
	url := fmt.Sprintf("%s%s/%s", fc.baseURL, resourceType, id)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	// Add headers
	for key, value := range fc.headers {
		req.Header.Set(key, value)
	}
	
	resp, err := fc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response (status %d): %s", resp.StatusCode, body)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	
	// Add to cache if enabled
	if fc.cacheEnabled {
		fc.addToCache(resourceType, id, result)
	}
	
	return result, nil
}

// SearchPatients searches for patients based on parameters
func (fc *FHIRClient) SearchPatients(ctx context.Context, params map[string]string) ([]map[string]interface{}, error) {
	return fc.SearchResources(ctx, "Patient", params)
}

// SearchResources searches for resources based on type and parameters
func (fc *FHIRClient) SearchResources(ctx context.Context, resourceType string, params map[string]string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s", fc.baseURL, resourceType)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	// Add query parameters
	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()
	
	// Add headers
	for key, value := range fc.headers {
		req.Header.Set(key, value)
	}
	
	resp, err := fc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response (status %d): %s", resp.StatusCode, body)
	}
	
	var bundle map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	
	// Extract entries from bundle
	entries, ok := bundle["entry"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}
	
	resources := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		
		resource, ok := entryMap["resource"].(map[string]interface{})
		if !ok {
			continue
		}
		
		resources = append(resources, resource)
	}
	
	return resources, nil
}

// CreateResource creates a new resource
func (fc *FHIRClient) CreateResource(ctx context.Context, resourceType string, resource map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s", fc.baseURL, resourceType)
	
	body, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("error encoding resource: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	// Add headers
	for key, value := range fc.headers {
		req.Header.Set(key, value)
	}
	
	resp, err := fc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response (status %d): %s", resp.StatusCode, body)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	
	// Add to cache if enabled
	if fc.cacheEnabled {
		if id, ok := result["id"].(string); ok {
			fc.addToCache(resourceType, id, result)
		}
	}
	
	return result, nil
}

// UpdateResource updates an existing resource
func (fc *FHIRClient) UpdateResource(ctx context.Context, resourceType string, id string, resource map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s/%s", fc.baseURL, resourceType, id)
	
	body, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("error encoding resource: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	// Add headers
	for key, value := range fc.headers {
		req.Header.Set(key, value)
	}
	
	resp, err := fc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response (status %d): %s", resp.StatusCode, body)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	
	// Update cache if enabled
	if fc.cacheEnabled {
		fc.addToCache(resourceType, id, result)
	}
	
	return result, nil
}

// DeleteResource deletes a resource
func (fc *FHIRClient) DeleteResource(ctx context.Context, resourceType string, id string) error {
	url := fmt.Sprintf("%s%s/%s", fc.baseURL, resourceType, id)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	
	// Add headers
	for key, value := range fc.headers {
		req.Header.Set(key, value)
	}
	
	resp, err := fc.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("error response (status %d): %s", resp.StatusCode, body)
	}
	
	// Remove from cache if enabled
	if fc.cacheEnabled {
		fc.removeFromCache(resourceType, id)
	}
	
	return nil
}

// ExecuteOperation executes a FHIR operation
func (fc *FHIRClient) ExecuteOperation(ctx context.Context, operation string, params map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s$%s", fc.baseURL, operation)
	
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error encoding parameters: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	// Add headers
	for key, value := range fc.headers {
		req.Header.Set(key, value)
	}
	
	resp, err := fc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response (status %d): %s", resp.StatusCode, body)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	
	return result, nil
}

// getFromCache tries to get a resource from cache
func (fc *FHIRClient) getFromCache(resourceType, id string) (map[string]interface{}, bool) {
	fc.cache.mutex.RLock()
	defer fc.cache.mutex.RUnlock()
	
	resources, ok := fc.cache.resources[resourceType]
	if !ok {
		return nil, false
	}
	
	cached, ok := resources[id]
	if !ok {
		return nil, false
	}
	
	// Check if expired
	if time.Since(cached.Timestamp) > cached.TTL {
		return nil, false
	}
	
	return cached.Data, true
}

// addToCache adds a resource to the cache
func (fc *FHIRClient) addToCache(resourceType, id string, data map[string]interface{}) {
	fc.cache.mutex.Lock()
	defer fc.cache.mutex.Unlock()
	
	// Initialize map for resource type if needed
	if _, ok := fc.cache.resources[resourceType]; !ok {
		fc.cache.resources[resourceType] = make(map[string]CachedResource)
	}
	
	// Add to cache
	fc.cache.resources[resourceType][id] = CachedResource{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       fc.cache.defaultTTL,
	}
	
	// Check cache size
	if fc.getCacheSize() > fc.cache.maxSize {
		fc.evictOldestCache()
	}
}

// removeFromCache removes a resource from the cache
func (fc *FHIRClient) removeFromCache(resourceType, id string) {
	fc.cache.mutex.Lock()
	defer fc.cache.mutex.Unlock()
	
	resources, ok := fc.cache.resources[resourceType]
	if !ok {
		return
	}
	
	delete(resources, id)
	
	// Cleanup empty maps
	if len(resources) == 0 {
		delete(fc.cache.resources, resourceType)
	}
}

// getCacheSize gets the total number of cached resources
func (fc *FHIRClient) getCacheSize() int {
	size := 0
	for _, resources := range fc.cache.resources {
		size += len(resources)
	}
	return size
}

// evictOldestCache removes the oldest entries from the cache
func (fc *FHIRClient) evictOldestCache() {
	type cacheEntry struct {
		resourceType string
		id           string
		timestamp    time.Time
	}
	
	// Collect all entries
	entries := make([]cacheEntry, 0)
	for resourceType, resources := range fc.cache.resources {
		for id, resource := range resources {
			entries = append(entries, cacheEntry{
				resourceType: resourceType,
				id:           id,
				timestamp:    resource.Timestamp,
			})
		}
	}
	
	// Sort by timestamp (oldest first)
	numToRemove := fc.getCacheSize() - fc.cache.maxSize + 10 // Remove extra to avoid frequent evictions
	if numToRemove <= 0 || len(entries) <= numToRemove {
		return
	}
	
	// Sort entries by timestamp
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].timestamp.After(entries[j].timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	// Remove oldest entries
	for i := 0; i < numToRemove && i < len(entries); i++ {
		entry := entries[i]
		delete(fc.cache.resources[entry.resourceType], entry.id)
		
		// Cleanup empty maps
		if len(fc.cache.resources[entry.resourceType]) == 0 {
			delete(fc.cache.resources, entry.resourceType)
		}
	}
}

// FHIRBundle represents a FHIR bundle
type FHIRBundle struct {
	ResourceType string          `json:"resourceType"`
	Type         string          `json:"type"`
	Entry        []FHIRBundleEntry `json:"entry"`
}

// FHIRBundleEntry represents an entry in a FHIR bundle
type FHIRBundleEntry struct {
	Resource map[string]interface{} `json:"resource"`
}

// CreateBundle creates a new FHIR bundle
func CreateBundle(bundleType string, resources []map[string]interface{}) FHIRBundle {
	entries := make([]FHIRBundleEntry, len(resources))
	for i, resource := range resources {
		entries[i] = FHIRBundleEntry{
			Resource: resource,
		}
	}
	
	return FHIRBundle{
		ResourceType: "Bundle",
		Type:         bundleType,
		Entry:        entries,
	}
}

// PatientBuilder helps build a FHIR Patient resource
type PatientBuilder struct {
	patient map[string]interface{}
}

// NewPatientBuilder creates a new patient builder
func NewPatientBuilder() *PatientBuilder {
	return &PatientBuilder{
		patient: map[string]interface{}{
			"resourceType": "Patient",
			"active":       true,
		},
	}
}

// WithID sets the ID
func (pb *PatientBuilder) WithID(id string) *PatientBuilder {
	pb.patient["id"] = id
	return pb
}

// WithName sets the name
func (pb *PatientBuilder) WithName(family string, given ...string) *PatientBuilder {
	name := map[string]interface{}{
		"family": family,
		"given":  given,
	}
	pb.patient["name"] = []interface{}{name}
	return pb
}

// WithBirthDate sets the birth date
func (pb *PatientBuilder) WithBirthDate(date string) *PatientBuilder {
	pb.patient["birthDate"] = date
	return pb
}

// WithGender sets the gender
func (pb *PatientBuilder) WithGender(gender string) *PatientBuilder {
	pb.patient["gender"] = gender
	return pb
}

// WithAddress sets the address
func (pb *PatientBuilder) WithAddress(line []string, city, state, postalCode, country string) *PatientBuilder {
	address := map[string]interface{}{
		"line":        line,
		"city":        city,
		"state":       state,
		"postalCode":  postalCode,
		"country":     country,
	}
	pb.patient["address"] = []interface{}{address}
	return pb
}

// WithTelecom adds a telecom (contact point)
func (pb *PatientBuilder) WithTelecom(system, value, use string) *PatientBuilder {
	telecom := map[string]interface{}{
		"system": system,
		"value":  value,
		"use":    use,
	}
	
	var telecoms []interface{}
	if existing, ok := pb.patient["telecom"].([]interface{}); ok {
		telecoms = existing
	} else {
		telecoms = make([]interface{}, 0)
	}
	
	telecoms = append(telecoms, telecom)
	pb.patient["telecom"] = telecoms
	return pb
}

// WithIdentifier adds an identifier
func (pb *PatientBuilder) WithIdentifier(system, value string) *PatientBuilder {
	identifier := map[string]interface{}{
		"system": system,
		"value":  value,
	}
	
	var identifiers []interface{}
	if existing, ok := pb.patient["identifier"].([]interface{}); ok {
		identifiers = existing
	} else {
		identifiers = make([]interface{}, 0)
	}
	
	identifiers = append(identifiers, identifier)
	pb.patient["identifier"] = identifiers
	return pb
}

// Build returns the completed patient resource
func (pb *PatientBuilder) Build() map[string]interface{} {
	return pb.patient
}

// FHIRValidator validates FHIR resources
type FHIRValidator struct {
	validationServer string
	client           *http.Client
}

// NewFHIRValidator creates a new FHIR validator
func NewFHIRValidator(validationServer string) *FHIRValidator {
	return &FHIRValidator{
		validationServer: validationServer,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ValidateResource validates a FHIR resource
func (fv *FHIRValidator) ValidateResource(ctx context.Context, resource map[string]interface{}) (bool, []string, error) {
	body, err := json.Marshal(resource)
	if err != nil {
		return false, nil, fmt.Errorf("error encoding resource: %w", err)
	}
	
	url := fmt.Sprintf("%s/validate", fv.validationServer)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return false, nil, fmt.Errorf("error creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/fhir+json")
	
	resp, err := fv.client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil, fmt.Errorf("error decoding response: %w", err)
	}
	
	// Parse validation results
	issues, ok := result["issue"].([]interface{})
	if !ok {
		return true, nil, nil
	}
	
	errors := make([]string, 0)
	hasError := false
	
	for _, issue := range issues {
		issueMap, ok := issue.(map[string]interface{})
		if !ok {
			continue
		}
		
		severity, _ := issueMap["severity"].(string)
		details, _ := issueMap["details"].(map[string]interface{})
		text, _ := details["text"].(string)
		
		if severity == "error" {
			hasError = true
			errors = append(errors, text)
		}
	}
	
	return !hasError, errors, nil
}
