package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/telemedicine/zkhealth/pkg/api"
	"github.com/telemedicine/zkhealth/pkg/cassandra"
	"github.com/telemedicine/zkhealth/pkg/consent"
	"github.com/telemedicine/zkhealth/pkg/eventlog"
	"github.com/telemedicine/zkhealth/pkg/infrastructure"
	"github.com/telemedicine/zkhealth/pkg/monitoring"
	"github.com/telemedicine/zkhealth/pkg/yag"
	"github.com/telemedicine/zkhealth/pkg/zkcircuit"
	"github.com/telemedicine/zkhealth/pkg/zkproof"
)

const (
	defaultMongoURI       = "mongodb://localhost:27018"
	defaultCassandraHosts = "localhost"
	defaultAPIPort        = ":8080"
	appVersion            = "1.0.0"
)

func main() {
	log.Printf("Starting ZK-Proof-Based Decentralized Healthcare Infrastructure v%s...", appVersion)

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("Shutdown signal received, closing connections...")
		cancel()
		// Allow some time for cleanup
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	// Get configuration from environment or use defaults
	mongoURI := getEnv("MONGO_URI", defaultMongoURI)
	cassandraHost := getEnv("CASSANDRA_HOST", defaultCassandraHosts)
	apiPort := getEnv("API_PORT", defaultAPIPort)

	// Initialize ZK Identity module
	log.Println("Initializing ZK Identity module...")
	zkIdentity, err := zkproof.NewZKIdentity(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Failed to initialize ZK Identity: %v", err)
	}
	defer zkIdentity.Close(ctx)
	log.Println("✓ ZK Identity module initialized")

	// Initialize Cassandra Archive
	log.Println("Initializing Cassandra Archive...")
	cassandraArchive, err := cassandra.NewCassandraArchive([]string{cassandraHost}, "healthcare")
	if err != nil {
		log.Fatalf("Failed to initialize Cassandra Archive: %v", err)
	}
	defer cassandraArchive.Close()
	log.Println("✓ Cassandra Archive module initialized")

	// Initialize Event Logger
	log.Println("Initializing Event Logger...")
	eventLogger, err := eventlog.NewEventLogger(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Failed to initialize Event Logger: %v", err)
	}
	defer eventLogger.Close(ctx)
	log.Println("✓ Event Logger module initialized")

	// Initialize YAG Updater
	log.Println("Initializing YAG Updater...")
	yagUpdater, err := yag.NewYAGUpdater(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Failed to initialize YAG Updater: %v", err)
	}
	defer yagUpdater.Close(ctx)
	log.Println("✓ YAG Updater module initialized")

	// Initialize Consent Manager
	log.Println("Initializing Consent Manager...")
	consentManager, err := consent.NewConsentManager(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Failed to initialize Consent Manager: %v", err)
	}
	defer consentManager.Close(ctx)
	log.Println("✓ Consent Manager module initialized")

	// Initialize Misalignment Tracker
	log.Println("Initializing Treatment Vector Misalignment Tracker...")
	misalignmentTracker, err := yag.NewMisalignmentTracker(ctx, mongoURI, yagUpdater)
	if err != nil {
		log.Fatalf("Failed to initialize Misalignment Tracker: %v", err)
	}
	defer misalignmentTracker.Close(ctx)
	log.Println("✓ Misalignment Tracker module initialized")

	// Initialize infrastructure components
	log.Println("Initializing infrastructure components...")
	infrConfig := infrastructure.LoadConfigFromEnv()
	infrastructureManager, err := infrastructure.NewInfrastructureManager(infrConfig)
	if err != nil {
		log.Fatalf("Failed to initialize infrastructure manager: %v", err)
	}
	
	// Start infrastructure components
	if err := infrastructureManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start infrastructure components: %v", err)
	}
	defer infrastructureManager.Stop()
	log.Println("✓ Infrastructure components initialized")

	// Initialize security components
	log.Println("Initializing security components...")
	securityManager := infrastructureManager.SecurityManager
	// Start key rotation
	securityManager.StartKeyRotation()
	defer securityManager.StopKeyRotation()
	log.Println("✓ Security components initialized")

	// Initialize monitoring components
	log.Println("Initializing monitoring components...")
	healthChecker := infrastructureManager.HealthChecker
	metricsCollector := infrastructureManager.MetricsCollector

	// Start monitoring server if initialized
	if infrastructureManager.MonitoringServer != nil {
		log.Println("Monitoring server initialized")
	}
	
	// Register the services in health check
	healthChecker.AddCheck("zkidentity", "ZK Identity Service", func() (bool, string) {
		// Check if the service is operational
		return zkIdentity != nil, "ZK Identity service is initialized"
	}, true)
	
	healthChecker.AddCheck("cassandra", "Cassandra Archive Service", func() (bool, string) {
		// Check if the service is operational
		return cassandraArchive != nil, "Cassandra Archive service is initialized"
	}, true)
	
	log.Println("✓ Monitoring components initialized")

	// Initialize ZK Circuit toolkit
	log.Println("Initializing ZK Circuit toolkit...")
	circuitCompiler := infrastructureManager.ZKCircuitCompiler

	// Ensure circuit executor is ready
	if infrastructureManager.ZKCircuitExecutor != nil {
		log.Println("ZK Circuit executor initialized")
	}
	
	// Load healthcare-specific circuit templates
	templateManager := zkcircuit.NewTemplateManager()
	for _, templateName := range templateManager.ListTemplates() {
		template, found := templateManager.GetTemplate(templateName)
		if found {
			_, err := circuitCompiler.Compile(ctx, template)
			if err != nil {
				log.Printf("Warning: Failed to compile template %s: %v", templateName, err)
			} else {
				log.Printf("Successfully compiled circuit template: %s", templateName)
			}
		}
	}
	log.Println("✓ ZK Circuit toolkit initialized")

	// Initialize API server
	log.Println("Initializing API server...")
	server := api.NewHealthServer(
		zkIdentity,
		cassandraArchive,
		eventLogger,
		yagUpdater,
		consentManager,
		misalignmentTracker,
	)

	// We'll use the infrastructure components directly
	log.Println("Configuring server with infrastructure components...")
	
	// Instead of using middleware directly, we'll wrap the handlers as needed
	// The Router is already initialized in the server
	
	// We don't need to set the FHIR/EHR clients since they're used by the infrastructure manager
	if infrastructureManager.FHIRClient != nil {
		log.Println("FHIR interoperability initialized and available")
	}
	
	if infrastructureManager.EHRClient != nil {
		log.Println("EHR interoperability initialized and available")
	}
	
	log.Printf("✓ API server initialized, starting on port %s", apiPort)

	// Add request tracking middleware
	requestTrackingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			
			// Create a custom response writer to track status code
			wrapper := NewResponseWriterWrapper(w)
			
			// Call the next handler
			next.ServeHTTP(wrapper, r)
			
			// Record metrics
			duration := time.Since(startTime)
			path := r.URL.Path
			method := r.Method
			statusCode := wrapper.StatusCode()
			
			// Log the request
			log.Printf("%s %s %d %s", method, path, statusCode, duration)
			
			// Track metrics
			TrackRequest(metricsCollector, path, method, statusCode, duration)
		})
	}
	// We'll apply middleware using the Router directly
	server.Router.Use(requestTrackingMiddleware)

	// Start API server with enhanced middleware
	if err := server.Start(apiPort); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// TrackRequest tracks HTTP request metrics
func TrackRequest(m *monitoring.MetricsCollector, path, method string, statusCode int, duration time.Duration) {
	// In a real implementation, this would use the metrics collector to record metrics
	// For this demo implementation, we'll just log
	log.Printf("METRIC: request_duration path=%s method=%s status=%d duration=%s", 
		path, method, statusCode, duration)
}

// NewResponseWriterWrapper creates a new response writer wrapper
func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status code
	}
}

// ResponseWriterWrapper wraps a ResponseWriter to capture the status code
type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rww *ResponseWriterWrapper) WriteHeader(statusCode int) {
	rww.statusCode = statusCode
	rww.ResponseWriter.WriteHeader(statusCode)
}

// StatusCode returns the captured status code
func (rww *ResponseWriterWrapper) StatusCode() int {
	return rww.statusCode
}
