package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/telemedicine/zkhealth/pkg/api"
	"github.com/telemedicine/zkhealth/pkg/cassandra"
	"github.com/telemedicine/zkhealth/pkg/consent"
	"github.com/telemedicine/zkhealth/pkg/eventlog"
	"github.com/telemedicine/zkhealth/pkg/yag"
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
	log.Printf("✓ API server initialized, starting on port %s", apiPort)

	// Start API server
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
