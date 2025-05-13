package infrastructure

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/telemedicine/zkhealth/pkg/interop"
	"github.com/telemedicine/zkhealth/pkg/monitoring"
	"github.com/telemedicine/zkhealth/pkg/scaling"
	"github.com/telemedicine/zkhealth/pkg/security"
	"github.com/telemedicine/zkhealth/pkg/zkcircuit"
)

// Config contains configuration for all infrastructure components
type Config struct {
	// Server settings
	Port int `json:"port"`
	Host string `json:"host"`

	// Scaling settings
	MinNodes int    `json:"min_nodes"`
	MaxNodes int    `json:"max_nodes"`
	Strategy string `json:"load_balancing_strategy"`

	// Security settings
	KeyRotationDays int    `json:"key_rotation_days"`
	SecurityLevel   string `json:"security_level"`

	// Monitoring settings
	MonitoringPort       int    `json:"monitoring_port"`
	HealthCheckInterval  string `json:"health_check_interval"`
	MetricsCollectPeriod string `json:"metrics_collect_period"`

	// Interoperability settings
	FHIREndpoint      string `json:"fhir_endpoint"`
	EHRSystemEndpoint string `json:"ehr_system_endpoint"`
	EHRSystemType     string `json:"ehr_system_type"`
	HL7Port           int    `json:"hl7_port"`
	DICOMPort         int    `json:"dicom_port"`

	// ZK Circuit settings
	CircuitTemplatesDir string `json:"circuit_templates_dir"`
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *Config {
	config := &Config{
		// Default values
		Port:                8080,
		Host:                "localhost",
		MinNodes:            2,
		MaxNodes:            10,
		Strategy:            "round-robin",
		KeyRotationDays:     90,
		SecurityLevel:       "high",
		MonitoringPort:      8081,
		HealthCheckInterval: "30s",
		MetricsCollectPeriod: "15s",
		FHIREndpoint:        "http://localhost:8082/fhir/",
		EHRSystemEndpoint:   "http://localhost:8083/api/",
		EHRSystemType:       "Epic",
		HL7Port:             2575,
		DICOMPort:           4242,
		CircuitTemplatesDir: "./templates/circuits",
	}

	// Server settings
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}
	if host := os.Getenv("HOST"); host != "" {
		config.Host = host
	}

	// Scaling settings
	if minNodes := os.Getenv("MIN_NODES"); minNodes != "" {
		if n, err := strconv.Atoi(minNodes); err == nil {
			config.MinNodes = n
		}
	}
	if maxNodes := os.Getenv("MAX_NODES"); maxNodes != "" {
		if n, err := strconv.Atoi(maxNodes); err == nil {
			config.MaxNodes = n
		}
	}
	if strategy := os.Getenv("LOAD_BALANCING_STRATEGY"); strategy != "" {
		config.Strategy = strategy
	}

	// Security settings
	if days := os.Getenv("KEY_ROTATION_DAYS"); days != "" {
		if d, err := strconv.Atoi(days); err == nil {
			config.KeyRotationDays = d
		}
	}
	if level := os.Getenv("SECURITY_LEVEL"); level != "" {
		config.SecurityLevel = level
	}

	// Monitoring settings
	if port := os.Getenv("MONITORING_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.MonitoringPort = p
		}
	}
	if interval := os.Getenv("HEALTH_CHECK_INTERVAL"); interval != "" {
		config.HealthCheckInterval = interval
	}
	if period := os.Getenv("METRICS_COLLECT_PERIOD"); period != "" {
		config.MetricsCollectPeriod = period
	}

	// Interoperability settings
	if endpoint := os.Getenv("FHIR_ENDPOINT"); endpoint != "" {
		config.FHIREndpoint = endpoint
	}
	if endpoint := os.Getenv("EHR_SYSTEM_ENDPOINT"); endpoint != "" {
		config.EHRSystemEndpoint = endpoint
	}
	if sysType := os.Getenv("EHR_SYSTEM_TYPE"); sysType != "" {
		config.EHRSystemType = sysType
	}
	if port := os.Getenv("HL7_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.HL7Port = p
		}
	}
	if port := os.Getenv("DICOM_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.DICOMPort = p
		}
	}

	// ZK Circuit settings
	if dir := os.Getenv("CIRCUIT_TEMPLATES_DIR"); dir != "" {
		config.CircuitTemplatesDir = dir
	}

	return config
}

// InfrastructureManager manages all infrastructure components
type InfrastructureManager struct {
	Config            *Config
	LoadBalancer      *scaling.LoadBalancer
	AutoScaler        *scaling.AutoScaler
	SecurityManager   *security.SecurityManager
	HealthChecker     *monitoring.HealthChecker
	MetricsCollector  *monitoring.MetricsCollector
	MonitoringServer  *monitoring.MonitoringServer
	ZKCircuitCompiler *zkcircuit.CircuitCompiler
	ZKCircuitExecutor *zkcircuit.CircuitExecutor
	FHIRClient        *interop.FHIRClient
	EHRClient         *interop.EHRClient
	CircuitBreakers   map[string]*monitoring.CircuitBreaker
	RESTServer        *RESTServer
}

// NewInfrastructureManager creates a new infrastructure manager with all components
func NewInfrastructureManager(config *Config) (*InfrastructureManager, error) {
	// Validate configuration
	if config == nil {
		config = LoadConfigFromEnv()
	}

	// Parse durations from config
	healthCheckInterval, err := time.ParseDuration(config.HealthCheckInterval)
	if err != nil {
		healthCheckInterval = 30 * time.Second
	}
	
	// We'll use the config's metrics collection period elsewhere
	_, err = time.ParseDuration(config.MetricsCollectPeriod)
	if err != nil {
		// Use default in case of error
		config.MetricsCollectPeriod = "15s"
	}

	// Initialize core infrastructure components
	loadBalancingAlgorithm := parseBalancingAlgorithm(config.Strategy)
	
	loadBalancer := scaling.NewLoadBalancer(loadBalancingAlgorithm, nil)
	autoScaler := scaling.NewAutoScaler(loadBalancer, config.MinNodes, config.MaxNodes, 0.7)
	
	securityManager := security.NewSecurityManager()
	if config.KeyRotationDays > 0 {
		keyManager := securityManager.GetKeyManager()
		keyManager.SetRotationPeriod(time.Duration(config.KeyRotationDays) * 24 * time.Hour)
	}
	
	healthChecker := monitoring.NewHealthChecker(healthCheckInterval)
	metricsCollector := monitoring.NewMetricsCollector()
	monitoringServer := monitoring.NewMonitoringServer(healthChecker, metricsCollector, config.MonitoringPort)
	
	circuitCompiler := zkcircuit.NewCircuitCompiler()
	circuitExecutor := zkcircuit.NewCircuitExecutor(circuitCompiler)
	
	// Initialize interoperability components
	var fhirClient *interop.FHIRClient
	if config.FHIREndpoint != "" {
		fhirClient = interop.NewFHIRClient(config.FHIREndpoint, interop.FHIRR4)
	}
	
	var ehrClient *interop.EHRClient
	if config.EHRSystemEndpoint != "" {
		var ehrSystem interop.EHRSystem
		switch strings.ToLower(config.EHRSystemType) {
		case "epic":
			ehrSystem = interop.EHRSystemEpic
		case "cerner":
			ehrSystem = interop.EHRSystemCerner
		case "allscripts":
			ehrSystem = interop.EHRSystemAllscripts
		default:
			ehrSystem = interop.EHRSystemEpic
		}
		
		ehrClient = interop.NewEHRClient(config.EHRSystemEndpoint, ehrSystem)
	}
	
	// Create circuit breakers for critical services
	circuitBreakers := make(map[string]*monitoring.CircuitBreaker)
	circuitBreakers["database"] = monitoring.NewCircuitBreaker("database", 5, 30*time.Second)
	circuitBreakers["fhir"] = monitoring.NewCircuitBreaker("fhir", 3, 10*time.Second)
	circuitBreakers["ehr"] = monitoring.NewCircuitBreaker("ehr", 3, 10*time.Second)
	
	for _, cb := range circuitBreakers {
		monitoringServer.RegisterCircuitBreaker(cb)
	}
	
	return &InfrastructureManager{
		Config:            config,
		LoadBalancer:      loadBalancer,
		AutoScaler:        autoScaler,
		SecurityManager:   securityManager,
		HealthChecker:     healthChecker,
		MetricsCollector:  metricsCollector,
		MonitoringServer:  monitoringServer,
		ZKCircuitCompiler: circuitCompiler,
		ZKCircuitExecutor: circuitExecutor,
		FHIRClient:        fhirClient,
		EHRClient:         ehrClient,
		CircuitBreakers:   circuitBreakers,
	}, nil
}

// Start starts all infrastructure components
func (im *InfrastructureManager) Start(ctx context.Context) error {
	// Start security components
	im.SecurityManager.StartKeyRotation()
	
	// Add standard system metrics
	im.MetricsCollector.AddStandardSystemMetrics()
	
	// Start metrics collection
	metricsCollectPeriod, err := time.ParseDuration(im.Config.MetricsCollectPeriod)
	if err != nil {
		metricsCollectPeriod = 15 * time.Second
	}
	im.MetricsCollector.Start(metricsCollectPeriod)
	
	// Start health checks
	im.setupHealthChecks()
	im.HealthChecker.Start(ctx)
	
	// Start monitoring server
	if err := im.MonitoringServer.Start(ctx); err != nil {
		return err
	}
	
	// Let's check the load on each active node
	nodes = im.LoadBalancer.GetNodes()
	if len(nodes) == 0 {
		log.Println("No nodes registered with load balancer yet")
		log.Println("Adding initial node...")
		im.initializeLoadBalancer()
	}
	
	// Start health checks for load balancer
	im.LoadBalancer.StartHealthCheck(ctx)
	
	// Configure auto-scaler functions
	im.configureAutoScaler()
	
	// Load ZKCircuit templates
	im.loadZKCircuitTemplates()

	// Initialize REST API server if not already initialized
	if im.RESTServer == nil {
		im.RESTServer = NewRESTServer(im)
		
		// Start REST API server in a goroutine
		go func() {
			if err := im.RESTServer.Start(); err != nil {
				log.Printf("Error starting infrastructure REST API server: %v", err)
			}
		}()
		log.Println("REST API server for infrastructure components initialized")
	}
	
	log.Println("Infrastructure components started successfully")
	return nil
}

// Stop stops all infrastructure components
func (im *InfrastructureManager) Stop() {
	// Stop auto-scaler and load balancer
	im.LoadBalancer.StopHealthCheck()
	
	// Stop security components
	im.SecurityManager.StopKeyRotation()
	
	// Stop metrics collector
	im.MetricsCollector.Stop()
	
	// Stop health checker
	im.HealthChecker.Stop()
	
	log.Println("Infrastructure components stopped")
}

// setupHealthChecks sets up health checks for critical services
func (im *InfrastructureManager) setupHealthChecks() {
	// Database health check
	im.HealthChecker.AddCheck("database", "Cassandra database connection", func() (bool, string) {
		// Implement actual database health check here
		// For now, returning a simulated success
		return true, "Database connection is healthy"
	}, true)
	
	// FHIR API health check
	if im.FHIRClient != nil {
		im.HealthChecker.AddCheck("fhir-api", "FHIR API connectivity", func() (bool, string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			// Use the circuit breaker pattern to prevent cascading failures
			healthy := false
			message := "FHIR API check failed"
			
			err := im.CircuitBreakers["fhir"].Execute(ctx, func() error {
				// Simple check - try to get capability statement
				_, err := im.FHIRClient.ExecuteOperation(ctx, "metadata", nil)
				return err
			})
			
			if err == nil {
				healthy = true
				message = "FHIR API connection is healthy"
			}
			
			return healthy, message
		}, true)
	}
	
	// EHR system health check
	if im.EHRClient != nil {
		im.HealthChecker.AddCheck("ehr-system", "EHR system connectivity", func() (bool, string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			healthy := false
			message := "EHR system check failed"
			
			err := im.CircuitBreakers["ehr"].Execute(ctx, func() error {
				// For demonstration, we'll just make a simple request
				// In a real system, this would check the actual EHR API
				return im.EHRClient.Ping(ctx)
			})
			
			if err == nil {
				healthy = true
				message = "EHR system connection is healthy"
			}
			
			return healthy, message
		}, false) // Not critical, as we can fall back to other data sources
	}
}

// initializeLoadBalancer sets up initial nodes for the load balancer
func (im *InfrastructureManager) initializeLoadBalancer() {
	// In a real system, this would discover actual nodes to add
	// For demonstration, we'll add simulated nodes
	for i := 1; i <= im.Config.MinNodes; i++ {
		node := &scaling.Node{
			ID:           fmt.Sprintf("node-%d", i),
			Address:      fmt.Sprintf("127.0.0.%d", i),
			Port:         im.Config.Port,
			Weight:       100, // Equal weight for each node
			IsActive:     true,
			LastSeen:     time.Now(),
			Capabilities: []string{"identity", "document", "policy"},
			Tags: map[string]string{
				"region": "us-east",
				"env":    "production",
			},
		}
		
		im.LoadBalancer.AddNode(node)
	}
}

// configureAutoScaler sets up the auto-scaler functions
func (im *InfrastructureManager) configureAutoScaler() {
	// Set up scale up function
	scaleUp := func(numNodes int) error {
		log.Printf("Auto-scaling: Adding %d nodes", numNodes)
		
		// In a real system, this would provision actual nodes
		// For demonstration, we'll add simulated nodes
		currentCount := len(im.LoadBalancer.GetNodes())
		for i := 1; i <= numNodes; i++ {
			nodeNum := currentCount + i
			node := &scaling.Node{
				ID:           fmt.Sprintf("node-%d", nodeNum),
				Address:      fmt.Sprintf("127.0.0.%d", nodeNum),
				Port:         im.Config.Port,
				Weight:       100,
				IsActive:     true,
				LastSeen:     time.Now(),
				Capabilities: []string{"identity", "document", "policy"},
				Tags: map[string]string{
					"region": "us-east",
					"env":    "production",
				},
			}
			
			im.LoadBalancer.AddNode(node)
		}
		
		return nil
	}
	
	// Set up scale down function
	scaleDown := func(nodeIDs []string) error {
		log.Printf("Auto-scaling: Removing %d nodes", len(nodeIDs))
		
		// In a real system, this would properly decommission nodes
		for _, nodeID := range nodeIDs {
			im.LoadBalancer.RemoveNode(nodeID)
		}
		
		return nil
	}
	
	// Set the scale functions on the auto-scaler
	im.AutoScaler.SetScaleFunctions(scaleUp, scaleDown)
}

// loadZKCircuitTemplates loads all available ZK circuit templates
func (im *InfrastructureManager) loadZKCircuitTemplates() {
	// Create a template manager
	templateManager := zkcircuit.NewTemplateManager()
	
	// In a real implementation, this would load from disk or database
	templatePaths := map[string]string{
		"consent":      "/templates/consent.zkc",
		"identity":     "/templates/identity.zkc",
		"prescription": "/templates/prescription.zkc",
		"diagnosis":    "/templates/diagnosis.zkc",
	}
	
	for name, _ := range templatePaths {
		log.Printf("Loading ZK circuit template: %s\n", name)
		// In a real implementation, this would load and compile the template
	}
	
	// Compile some templates for immediate use
	templates := templateManager.ListTemplates()
	for _, templateName := range templates {
		template, found := templateManager.GetTemplate(templateName)
		if found {
			_, err := im.ZKCircuitCompiler.Compile(context.Background(), template)
			if err != nil {
				log.Printf("Failed to compile template %s: %v", templateName, err)
			} else {
				log.Printf("Successfully compiled template: %s", templateName)
			}
		}
	}
}

// GetZKCircuit gets a compiled ZK circuit by name
func (im *InfrastructureManager) GetZKCircuit(name string) (*zkcircuit.CompiledCircuit, bool) {
	return im.ZKCircuitCompiler.GetCircuit(name)
}

// ExecuteZKCircuit executes a ZK circuit with inputs
func (im *InfrastructureManager) ExecuteZKCircuit(
	ctx context.Context, 
	circuitName string, 
	publicInputs, 
	privateInputs map[string]interface{},
) (*zkcircuit.ExecutionResult, error) {
	return im.ZKCircuitExecutor.Execute(ctx, circuitName, publicInputs, privateInputs)
}

// GetServiceNode gets a node for a specific service capability
func (im *InfrastructureManager) GetServiceNode(ctx context.Context, capability string, clientIP string) (*scaling.Node, error) {
	return im.LoadBalancer.GetNextNode(ctx, clientIP, capability)
}

// CreateRequestHandler creates an HTTP middleware handler that adds infrastructure capabilities
func (im *InfrastructureManager) CreateRequestHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add infrastructure context to the request
		ctx := r.Context()
		
		// Rate limit check
		clientIP := getClientIP(r)
		if !im.SecurityManager.RateLimitAllowed(clientIP) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		// Determine which capability this request needs
		capability := ""
		path := r.URL.Path
		
		if strings.HasPrefix(path, "/identity") {
			capability = "identity"
		} else if strings.HasPrefix(path, "/document") {
			capability = "document"
		} else if strings.HasPrefix(path, "/policy") {
			capability = "policy"
		}
		
		// If capability routing is needed, get an appropriate node
		if capability != "" {
			node, err := im.GetServiceNode(ctx, capability, clientIP)
			if err != nil {
				http.Error(w, "No available service nodes", http.StatusServiceUnavailable)
				return
			}
			
			// If the node isn't this server, proxy the request
			if node.Address != im.Config.Host || node.Port != im.Config.Port {
				im.proxyRequest(w, r, node)
				return
			}
		}
		
		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		// Continue to the next handler
		next.ServeHTTP(w, r)
	})
}

// proxyRequest proxies a request to another node
func (im *InfrastructureManager) proxyRequest(w http.ResponseWriter, r *http.Request, node *scaling.Node) {
	// In a real implementation, this would forward the request to the node
	// For this demo, we'll just return a node info message
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := fmt.Sprintf(`{"proxied":true,"node":"%s","address":"%s","port":%d}`,
		node.ID, node.Address, node.Port)
	w.Write([]byte(response))
}

// parseBalancingAlgorithm converts a string algorithm name to the LoadBalancer type
func parseBalancingAlgorithm(algorithm string) scaling.BalancingAlgorithm {
	switch strings.ToLower(algorithm) {
	case "round-robin":
		return scaling.RoundRobin
	case "least-connections":
		return scaling.LeastConnections
	case "weighted-round-robin":
		return scaling.WeightedRoundRobin
	case "ip-hash":
		return scaling.IPHash
	default:
		return scaling.RoundRobin
	}
}

// getClientIP gets the client IP from a request
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check for X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
