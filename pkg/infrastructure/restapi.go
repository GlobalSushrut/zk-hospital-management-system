package infrastructure

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// RESTServer provides HTTP API access to infrastructure components
type RESTServer struct {
	Manager *InfrastructureManager
	Router  *mux.Router
}

// NewRESTServer creates a new REST API server for infrastructure components
func NewRESTServer(manager *InfrastructureManager) *RESTServer {
	server := &RESTServer{
		Manager: manager,
		Router:  mux.NewRouter(),
	}

	// Initialize routes
	server.setupRoutes()
	return server
}

// Start starts the REST API server
func (s *RESTServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.Manager.Config.Host, s.Manager.Config.Port+1)
	log.Printf("Starting infrastructure REST API server on %s", addr)
	return http.ListenAndServe(addr, s.Router)
}

// setupRoutes initializes the API routes
func (s *RESTServer) setupRoutes() {
	// Health check
	s.Router.HandleFunc("/health", s.healthCheck).Methods("GET")

	// ZK Circuit endpoints
	s.Router.HandleFunc("/zkcircuit/execute", s.executeZKCircuit).Methods("POST")
	s.Router.HandleFunc("/zkcircuit/verify", s.verifyZKCircuit).Methods("POST")
	s.Router.HandleFunc("/zkcircuit/list", s.listZKCircuits).Methods("GET")

	// Scaling endpoints
	s.Router.HandleFunc("/scaling/status", s.scalingStatus).Methods("GET")
	s.Router.HandleFunc("/scaling/scale", s.scaleNodes).Methods("POST")
	s.Router.HandleFunc("/scaling/route", s.routeRequest).Methods("GET")
	s.Router.HandleFunc("/scaling/nodes", s.getNodes).Methods("GET")

	// Security endpoints
	s.Router.HandleFunc("/security/status", s.securityStatus).Methods("GET")
	s.Router.HandleFunc("/security/token", s.generateToken).Methods("POST")
	s.Router.HandleFunc("/security/verify", s.verifyToken).Methods("GET")

	// Monitoring endpoints
	s.Router.HandleFunc("/monitoring/health", s.monitoringHealth).Methods("GET")
	s.Router.HandleFunc("/monitoring/metrics", s.getMetrics).Methods("GET")
	s.Router.HandleFunc("/monitoring/circuit/status", s.circuitStatus).Methods("GET")
	s.Router.HandleFunc("/monitoring/circuit/test", s.testCircuitBreaker).Methods("GET")

	// Interop endpoints
	s.Router.HandleFunc("/interop/fhir/status", s.fhirStatus).Methods("GET")
	s.Router.HandleFunc("/interop/fhir/Patient", s.fhirPatient).Methods("POST")
	s.Router.HandleFunc("/interop/ehr/status", s.ehrStatus).Methods("GET")
	s.Router.HandleFunc("/interop/ehr/patient/{id}", s.getEHRPatient).Methods("GET")
}

// healthCheck handles the /health endpoint
func (s *RESTServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	respondJSON(w, http.StatusOK, response)
}

// executeZKCircuit handles the /zkcircuit/execute endpoint
func (s *RESTServer) executeZKCircuit(w http.ResponseWriter, r *http.Request) {
	var request struct {
		CircuitType   string                 `json:"circuit_type"`
		PublicInputs  map[string]interface{} `json:"public_inputs,omitempty"`
		PrivateInputs map[string]interface{} `json:"private_inputs,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	startTime := time.Now()
	result, err := s.Manager.ExecuteZKCircuit(r.Context(), request.CircuitType, request.PublicInputs, request.PrivateInputs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Error executing circuit: %v", err))
		return
	}

	response := map[string]interface{}{
		"success":            true,
		"proof":              result.Proof,
		"public_outputs":     result.PublicOutputs,
		"execution_time_ms":  float64(time.Since(startTime)) / float64(time.Millisecond),
		"circuit_complexity": result.CircuitComplexity,
	}
	respondJSON(w, http.StatusOK, response)
}

// verifyZKCircuit handles the /zkcircuit/verify endpoint
func (s *RESTServer) verifyZKCircuit(w http.ResponseWriter, r *http.Request) {
	var request struct {
		CircuitType  string                 `json:"circuit_type"`
		Proof        string                 `json:"proof"`
		PublicInputs map[string]interface{} `json:"public_inputs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Simulate proof verification since we don't have the actual implementation
	valid := true
	if len(request.Proof) < 10 {
		valid = false
	}

	response := map[string]interface{}{
		"valid":            valid,
		"verification_time_ms": float64(time.Millisecond * 5) / float64(time.Millisecond),
	}
	respondJSON(w, http.StatusOK, response)
}

// listZKCircuits handles the /zkcircuit/list endpoint
func (s *RESTServer) listZKCircuits(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, we would get these from the circuit compiler
	circuits := []string{
		"patient-consent",
		"medical-credential",
		"prescription-validity",
		"insurance-eligibility", 
		"anonymized-research",
	}

	response := map[string]interface{}{
		"circuits": circuits,
	}
	respondJSON(w, http.StatusOK, response)
}

// scalingStatus handles the /scaling/status endpoint
func (s *RESTServer) scalingStatus(w http.ResponseWriter, r *http.Request) {
	// Get nodes from load balancer
	nodes := s.Manager.LoadBalancer.GetNodes()

	nodeInfos := make([]map[string]interface{}, 0)
	for _, node := range nodes {
		nodeInfos = append(nodeInfos, map[string]interface{}{
			"id":          node.ID,
			"address":     node.Address,
			"port":        node.Port,
			"health":      node.Health,
			"load":        node.Load,
			"connections": node.Connections,
			"capabilities": node.Capabilities,
		})
	}

	response := map[string]interface{}{
		"auto_scaling_enabled": true,
		"min_nodes":            s.Manager.Config.MinNodes,
		"max_nodes":            s.Manager.Config.MaxNodes,
		"current_node_count":   len(nodes),
		"algorithm":            s.Manager.Config.Strategy,
		"nodes":                nodeInfos,
	}
	respondJSON(w, http.StatusOK, response)
}

// scaleNodes handles the /scaling/scale endpoint
func (s *RESTServer) scaleNodes(w http.ResponseWriter, r *http.Request) {
	var request struct {
		DesiredNodes int    `json:"desired_nodes"`
		Reason       string `json:"reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get current nodes
	currentNodes := s.Manager.LoadBalancer.GetNodes()
	currentCount := len(currentNodes)

	// Cap to min/max range
	if request.DesiredNodes < s.Manager.Config.MinNodes {
		request.DesiredNodes = s.Manager.Config.MinNodes
	}
	if request.DesiredNodes > s.Manager.Config.MaxNodes {
		request.DesiredNodes = s.Manager.Config.MaxNodes
	}

	// Add or remove nodes as needed (simulation)
	if request.DesiredNodes > currentCount {
		// Simulate adding nodes
		for i := currentCount; i < request.DesiredNodes; i++ {
			nodeID := fmt.Sprintf("node-%d", i+1)
			nodePort := s.Manager.Config.Port + 100 + i
			s.Manager.LoadBalancer.AddNode(&scaling.Node{
				ID:       nodeID,
				Address:  s.Manager.Config.Host,
				Port:     nodePort,
				Health:   1.0,
				Load:     0.0,
				Capabilities: []string{"identity", "document", "policy"},
			})
		}
	} else if request.DesiredNodes < currentCount {
		// Simulate removing nodes
		for i := currentCount - 1; i >= request.DesiredNodes; i-- {
			if i < len(currentNodes) {
				s.Manager.LoadBalancer.RemoveNode(currentNodes[i].ID)
			}
		}
	}

	// Get new node list
	newNodes := s.Manager.LoadBalancer.GetNodes()
	
	response := map[string]interface{}{
		"success":          true,
		"previous_count":   currentCount,
		"new_count":        len(newNodes),
		"requested_count":  request.DesiredNodes,
		"reason":           request.Reason,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}
	respondJSON(w, http.StatusOK, response)
}

// routeRequest handles the /scaling/route endpoint 
func (s *RESTServer) routeRequest(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	capability := r.URL.Query().Get("capability")
	if capability == "" {
		capability = "general"
	}

	node, err := s.Manager.LoadBalancer.GetNextNode(r.Context(), clientIP, capability)
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, "No nodes available")
		return
	}

	response := map[string]interface{}{
		"node_id":     node.ID,
		"address":     node.Address,
		"port":        node.Port,
		"capability":  capability,
		"client_ip":   clientIP,
		"algorithm":   s.Manager.Config.Strategy,
	}
	respondJSON(w, http.StatusOK, response)
}

// getNodes handles the /scaling/nodes endpoint
func (s *RESTServer) getNodes(w http.ResponseWriter, r *http.Request) {
	nodes := s.Manager.LoadBalancer.GetNodes()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// securityStatus handles the /security/status endpoint
func (s *RESTServer) securityStatus(w http.ResponseWriter, r *http.Request) {
	keyManager := s.Manager.SecurityManager.GetKeyManager()
	
	response := map[string]interface{}{
		"status":              "active",
		"security_level":      s.Manager.Config.SecurityLevel,
		"last_key_rotation":   time.Now().Add(-24 * time.Hour).Format(time.RFC3339), // Simulated
		"key_id":              "key-" + time.Now().Format("20060102"),
		"rate_limiting_enabled": true,
	}
	respondJSON(w, http.StatusOK, response)
}

// generateToken handles the /security/token endpoint
func (s *RESTServer) generateToken(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Simple simulation for demo
	if request.Username == "" || request.Password == "" {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate a simulated token
	expiry := time.Now().Add(24 * time.Hour)
	token := fmt.Sprintf("zk.%s.%d", request.Username, expiry.Unix())

	response := map[string]interface{}{
		"token":      token,
		"expires_at": expiry.Format(time.RFC3339),
		"issued_at":  time.Now().Format(time.RFC3339),
		"type":       "Bearer",
	}
	respondJSON(w, http.StatusOK, response)
}

// verifyToken handles the /security/verify endpoint
func (s *RESTServer) verifyToken(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth == "" || len(auth) < 8 || !strings.HasPrefix(auth, "Bearer ") {
		respondError(w, http.StatusUnauthorized, "Invalid or missing token")
		return
	}

	token := auth[7:] // Remove "Bearer " prefix
	
	// Simple simulation for demo
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "zk" {
		respondError(w, http.StatusUnauthorized, "Invalid token format")
		return
	}

	username := parts[1]
	expiryStr := parts[2]
	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid token expiry")
		return
	}

	if time.Now().Unix() > expiry {
		respondError(w, http.StatusUnauthorized, "Token expired")
		return
	}

	response := map[string]interface{}{
		"valid":      true,
		"username":   username,
		"expires_at": time.Unix(expiry, 0).Format(time.RFC3339),
		"issued_at":  time.Now().Add(-1 * time.Hour).Format(time.RFC3339), // Simulated issue time
	}
	respondJSON(w, http.StatusOK, response)
}

// monitoringHealth handles the /monitoring/health endpoint
func (s *RESTServer) monitoringHealth(w http.ResponseWriter, r *http.Request) {
	components := map[string]map[string]interface{}{
		"database": {
			"status":           "healthy",
			"last_check":       time.Now().Format(time.RFC3339),
			"response_time_ms": 5,
		},
		"identity_service": {
			"status":           "healthy",
			"last_check":       time.Now().Format(time.RFC3339),
			"response_time_ms": 3,
		},
		"document_service": {
			"status":           "healthy",
			"last_check":       time.Now().Format(time.RFC3339),
			"response_time_ms": 7,
		},
		"policy_service": {
			"status":           "healthy",
			"last_check":       time.Now().Format(time.RFC3339),
			"response_time_ms": 2,
		},
		"zkproof_service": {
			"status":           "healthy",
			"last_check":       time.Now().Format(time.RFC3339),
			"response_time_ms": 12,
		},
		"fhir_interop": {
			"status":           "healthy",
			"last_check":       time.Now().Format(time.RFC3339),
			"response_time_ms": 15,
		},
		"ehr_interop": {
			"status":           "healthy",
			"last_check":       time.Now().Format(time.RFC3339),
			"response_time_ms": 18,
		},
	}

	response := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"components":  components,
	}
	respondJSON(w, http.StatusOK, response)
}

// getMetrics handles the /monitoring/metrics endpoint
func (s *RESTServer) getMetrics(w http.ResponseWriter, r *http.Request) {
	// Simulate metrics collection
	metrics := map[string]interface{}{
		"requests": map[string]interface{}{
			"total":         1542,
			"success":       1487,
			"error":         55,
			"success_rate":  96.4,
			"avg_latency_ms": 8.3,
		},
		"resources": map[string]interface{}{
			"cpu_usage":       42.5,
			"memory_usage_mb": 128.7,
			"disk_usage_gb":   0.8,
		},
		"components": map[string]interface{}{
			"identity": map[string]interface{}{
				"requests":       450,
				"avg_latency_ms": 3.2,
			},
			"document": map[string]interface{}{
				"requests":       380,
				"avg_latency_ms": 8.1,
			},
			"policy": map[string]interface{}{
				"requests":       512,
				"avg_latency_ms": 2.8,
			},
			"zkproof": map[string]interface{}{
				"requests":       200,
				"avg_latency_ms": 15.4,
			},
		},
	}

	response := map[string]interface{}{
		"metrics":    metrics,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"period":     "last_hour",
	}
	respondJSON(w, http.StatusOK, response)
}

// circuitStatus handles the /monitoring/circuit/status endpoint
func (s *RESTServer) circuitStatus(w http.ResponseWriter, r *http.Request) {
	// Get all circuit breakers
	circuitBreakers := s.Manager.CircuitBreakers
	
	status := make(map[string]string)
	for name, cb := range circuitBreakers {
		status[name] = string(cb.GetState())
	}
	
	// Add a test circuit for the test endpoint
	if _, ok := status["test"]; !ok {
		status["test"] = "closed"
	}

	respondJSON(w, http.StatusOK, status)
}

// testCircuitBreaker handles the /monitoring/circuit/test endpoint
func (s *RESTServer) testCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		service = "test"
	}

	var state string
	cb, exists := s.Manager.CircuitBreakers[service]
	if exists {
		// For test purposes, randomly fail to simulate circuit opening
		if service == "test" && time.Now().UnixNano()%3 == 0 {
			cb.RecordFailure()
			state = string(cb.GetState())
		} else {
			cb.RecordSuccess()
			state = string(cb.GetState())
		}
	} else {
		state = "unknown"
	}

	response := map[string]interface{}{
		"service": service,
		"state":   state,
	}
	respondJSON(w, http.StatusOK, response)
}

// fhirStatus handles the /interop/fhir/status endpoint
func (s *RESTServer) fhirStatus(w http.ResponseWriter, r *http.Request) {
	status := "connected"
	version := "R4"
	
	if s.Manager.FHIRClient == nil {
		status = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"version":   version,
		"endpoint":  s.Manager.Config.FHIREndpoint,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	respondJSON(w, http.StatusOK, response)
}

// fhirPatient handles the /interop/fhir/Patient endpoint
func (s *RESTServer) fhirPatient(w http.ResponseWriter, r *http.Request) {
	var patient map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patient); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Add resource metadata
	if _, ok := patient["resourceType"]; !ok {
		patient["resourceType"] = "Patient"
	}
	if _, ok := patient["id"]; !ok {
		patient["id"] = fmt.Sprintf("generated-%d", time.Now().UnixNano())
	}
	
	patient["meta"] = map[string]interface{}{
		"versionId":   "1",
		"lastUpdated": time.Now().UTC().Format(time.RFC3339),
	}

	respondJSON(w, http.StatusCreated, patient)
}

// ehrStatus handles the /interop/ehr/status endpoint
func (s *RESTServer) ehrStatus(w http.ResponseWriter, r *http.Request) {
	status := "connected"
	
	if s.Manager.EHRClient == nil {
		status = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"system":    s.Manager.Config.EHRSystemType,
		"endpoint":  s.Manager.Config.EHRSystemEndpoint,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	respondJSON(w, http.StatusOK, response)
}

// getEHRPatient handles the /interop/ehr/patient/{id} endpoint
func (s *RESTServer) getEHRPatient(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	patientID := vars["id"]
	
	if patientID == "" {
		respondError(w, http.StatusBadRequest, "Patient ID is required")
		return
	}

	// Simulate retrieving patient data
	patient := map[string]interface{}{
		"id":           patientID,
		"firstName":    "John",
		"lastName":     "Doe",
		"dateOfBirth":  "1970-01-01",
		"gender":       "male",
		"medicalRecordNumber": fmt.Sprintf("MRN-%s", patientID),
		"lastVisit":    time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, patient)
}

// Helper functions for HTTP responses
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{"error": message})
}
