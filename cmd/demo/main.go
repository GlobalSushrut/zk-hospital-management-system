package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/telemedicine/zkhealth/pkg/api"
	"github.com/telemedicine/zkhealth/pkg/infrastructure"
	"github.com/telemedicine/zkhealth/pkg/interop"
	"github.com/telemedicine/zkhealth/pkg/monitoring"
	"github.com/telemedicine/zkhealth/pkg/scaling"
	"github.com/telemedicine/zkhealth/pkg/security"
	"github.com/telemedicine/zkhealth/pkg/zkcircuit"
)

// Configuration for the demo app
type Config struct {
	Port         string
	StaticDir    string
	TemplatesDir string
}

// Demo application that showcases the ZK-Proof Healthcare System
type DemoApp struct {
	config           Config
	router           *mux.Router
	server           *http.Server
	infrastructureMgr *infrastructure.Manager
	zkCircuitManager *zkcircuit.Manager
	securityManager  *security.Manager
	monitoringManager *monitoring.Manager
	scalingManager   *scaling.Manager
	fhirClient       *interop.FHIRClient
	ehrClient        *interop.EHRClient
}

// NewDemoApp creates a new instance of the demo application
func NewDemoApp(config Config) (*DemoApp, error) {
	app := &DemoApp{
		config: config,
		router: mux.NewRouter(),
	}

	// Initialize infrastructure components
	infraConfig := infrastructure.LoadConfigFromEnv()
	infrastructureMgr, err := infrastructure.NewManager(infraConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize infrastructure manager: %w", err)
	}
	app.infrastructureMgr = infrastructureMgr

	// Initialize ZK circuit manager
	zkConfig := zkcircuit.LoadConfigFromEnv()
	zkCircuitManager, err := zkcircuit.NewManager(zkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ZK circuit manager: %w", err)
	}
	app.zkCircuitManager = zkCircuitManager

	// Initialize security manager
	secConfig := security.LoadConfigFromEnv()
	securityManager, err := security.NewManager(secConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize security manager: %w", err)
	}
	app.securityManager = securityManager

	// Initialize monitoring manager
	monConfig := monitoring.LoadConfigFromEnv()
	monitoringManager, err := monitoring.NewManager(monConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring manager: %w", err)
	}
	app.monitoringManager = monitoringManager

	// Initialize scaling manager
	scaleConfig := scaling.LoadConfigFromEnv()
	scalingManager, err := scaling.NewManager(scaleConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scaling manager: %w", err)
	}
	app.scalingManager = scalingManager

	// Initialize interoperability clients
	fhirClient, err := interop.NewFHIRClient(interop.LoadFHIRConfigFromEnv())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize FHIR client: %w", err)
	}
	app.fhirClient = fhirClient

	ehrClient, err := interop.NewEHRClient(interop.LoadEHRConfigFromEnv())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize EHR client: %w", err)
	}
	app.ehrClient = ehrClient

	// Set up the HTTP server
	app.server = &http.Server{
		Addr:         ":" + app.config.Port,
		Handler:      app.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return app, nil
}

// SetupRoutes configures the API endpoints and static file handlers
func (app *DemoApp) SetupRoutes() {
	// API endpoints
	apiRouter := app.router.PathPrefix("/api").Subrouter()
	
	// Health check endpoint
	apiRouter.HandleFunc("/health", app.handleHealth).Methods("GET")
	
	// ZK Circuit API endpoints
	apiRouter.HandleFunc("/zk/compile", app.handleZKCompile).Methods("POST")
	apiRouter.HandleFunc("/zk/execute", app.handleZKExecute).Methods("POST")
	apiRouter.HandleFunc("/zk/templates", app.handleZKTemplates).Methods("GET")
	
	// Security API endpoints
	apiRouter.HandleFunc("/security/encrypt", app.handleEncrypt).Methods("POST")
	apiRouter.HandleFunc("/security/decrypt", app.handleDecrypt).Methods("POST")
	apiRouter.HandleFunc("/security/status", app.handleSecurityStatus).Methods("GET")
	
	// Interoperability API endpoints
	apiRouter.HandleFunc("/interop/fhir/{resource}", app.handleFHIR).Methods("GET")
	apiRouter.HandleFunc("/interop/ehr/{system}/{operation}/{id}", app.handleEHR).Methods("GET")
	
	// Monitoring API endpoints
	apiRouter.HandleFunc("/monitoring/metrics", app.handleMetrics).Methods("GET")
	apiRouter.HandleFunc("/monitoring/health", app.handleSystemHealth).Methods("GET")
	
	// Scaling API endpoints
	apiRouter.HandleFunc("/scaling/status", app.handleScalingStatus).Methods("GET")
	
	// Serve static files for the web interface
	fs := http.FileServer(http.Dir(app.config.StaticDir))
	app.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	
	// Serve the main index page for all other routes
	app.router.PathPrefix("/").HandlerFunc(app.serveIndex)
}

// Start begins the HTTP server and listens for shutdown signals
func (app *DemoApp) Start() error {
	// Start the HTTP server in a goroutine
	go func() {
		log.Printf("Starting demo web application on port %s", app.config.Port)
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Set up channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received
	<-stop
	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := app.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server gracefully stopped")
	return nil
}

// Handler functions for API endpoints

func (app *DemoApp) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]string{"status": "healthy", "timestamp": time.Now().Format(time.RFC3339)}
	respondWithJSON(w, http.StatusOK, health)
}

func (app *DemoApp) handleZKCompile(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TemplateName string `json:"template_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	result, err := app.zkCircuitManager.CompileCircuit(request.TemplateName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error compiling circuit: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func (app *DemoApp) handleZKExecute(w http.ResponseWriter, r *http.Request) {
	var request struct {
		CircuitName  string                 `json:"circuit_name"`
		PublicInputs map[string]interface{} `json:"public_inputs"`
		PrivateInputs map[string]interface{} `json:"private_inputs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	result, err := app.zkCircuitManager.ExecuteCircuit(request.CircuitName, request.PublicInputs, request.PrivateInputs)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error executing circuit: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func (app *DemoApp) handleZKTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := app.zkCircuitManager.ListTemplates()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error listing templates: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, templates)
}

func (app *DemoApp) handleEncrypt(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Data string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	encrypted, keyID, err := app.securityManager.EncryptData([]byte(request.Data))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error encrypting data: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"encrypted": encrypted,
		"key_id": keyID,
	})
}

func (app *DemoApp) handleDecrypt(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Encrypted string `json:"encrypted"`
		KeyID     string `json:"key_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	decrypted, err := app.securityManager.DecryptData(request.Encrypted, request.KeyID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error decrypting data: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"decrypted": string(decrypted),
	})
}

func (app *DemoApp) handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	status := app.securityManager.GetStatus()
	respondWithJSON(w, http.StatusOK, status)
}

func (app *DemoApp) handleFHIR(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceType := vars["resource"]

	params := map[string]string{}
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	result, err := app.fhirClient.GetResource(resourceType, params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error getting FHIR resource: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func (app *DemoApp) handleEHR(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	system := vars["system"]
	operation := vars["operation"]
	id := vars["id"]

	result, err := app.ehrClient.PerformOperation(system, operation, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error performing EHR operation: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func (app *DemoApp) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := app.monitoringManager.GetMetrics()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error getting metrics: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, metrics)
}

func (app *DemoApp) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	health, err := app.monitoringManager.CheckHealth()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error checking health: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, health)
}

func (app *DemoApp) handleScalingStatus(w http.ResponseWriter, r *http.Request) {
	status, err := app.scalingManager.GetStatus()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error getting scaling status: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, status)
}

// serveIndex serves the main index.html file for the demo UI
func (app *DemoApp) serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, app.config.StaticDir+"/index.html")
}

// Helper functions for HTTP responses

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error marshalling JSON response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	// Load configuration from environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static" // Default static files directory
	}

	templatesDir := os.Getenv("TEMPLATES_DIR")
	if templatesDir == "" {
		templatesDir = "./templates" // Default templates directory
	}

	config := Config{
		Port:         port,
		StaticDir:    staticDir,
		TemplatesDir: templatesDir,
	}

	// Create and initialize the demo application
	app, err := NewDemoApp(config)
	if err != nil {
		log.Fatalf("Failed to initialize demo application: %v", err)
	}

	// Set up the routes
	app.SetupRoutes()

	// Start the server
	if err := app.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
