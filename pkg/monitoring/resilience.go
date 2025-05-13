package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	// CircuitClosed indicates the circuit is closed (allowing requests)
	CircuitClosed CircuitBreakerState = iota
	
	// CircuitOpen indicates the circuit is open (blocking requests)
	CircuitOpen
	
	// CircuitHalfOpen indicates the circuit is half-open (allowing test requests)
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures
type CircuitBreaker struct {
	name                 string
	failureThreshold     int
	resetTimeout         time.Duration
	halfOpenSuccessThreshold int
	
	state                CircuitBreakerState
	failures             int
	halfOpenSuccesses    int
	lastStateChange      time.Time
	mutex                sync.RWMutex
	
	onStateChange        func(name string, from, to CircuitBreakerState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:                 name,
		failureThreshold:     failureThreshold,
		resetTimeout:         resetTimeout,
		halfOpenSuccessThreshold: 5, // Default: 5 successful requests to close circuit
		state:                CircuitClosed,
		lastStateChange:      time.Now(),
	}
}

// SetOnStateChangeHandler sets a callback for state changes
func (cb *CircuitBreaker) SetOnStateChangeHandler(handler func(name string, from, to CircuitBreakerState)) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.onStateChange = handler
}

// Execute executes the given function if the circuit is closed or half-open
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	state := cb.State()
	
	switch state {
	case CircuitOpen:
		// Check if reset timeout has elapsed
		cb.mutex.RLock()
		elapsed := time.Since(cb.lastStateChange)
		cb.mutex.RUnlock()
		
		if elapsed >= cb.resetTimeout {
			// Transition to half-open
			cb.setState(CircuitHalfOpen)
		} else {
			// Circuit is open, reject the request
			return fmt.Errorf("circuit breaker %s is open", cb.name)
		}
		
	case CircuitClosed:
		// Normal operation
		err := fn()
		if err != nil {
			cb.recordFailure()
		}
		return err
		
	case CircuitHalfOpen:
		// Test requests allowed
		err := fn()
		if err != nil {
			// Failed test request, back to open
			cb.mutex.Lock()
			cb.failures++
			cb.halfOpenSuccesses = 0
			cb.setState(CircuitOpen)
			cb.mutex.Unlock()
			return err
		}
		
		// Successful test request
		cb.mutex.Lock()
		cb.halfOpenSuccesses++
		
		// Check if we've had enough successes to close the circuit
		if cb.halfOpenSuccesses >= cb.halfOpenSuccessThreshold {
			cb.setState(CircuitClosed)
			cb.failures = 0
			cb.halfOpenSuccesses = 0
		}
		cb.mutex.Unlock()
		return nil
	}
	
	// Should never reach here
	return fmt.Errorf("unexpected circuit breaker state: %v", state)
}

// recordFailure records a failure and possibly opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.failures++
	
	if cb.state == CircuitClosed && cb.failures >= cb.failureThreshold {
		cb.setState(CircuitOpen)
		cb.halfOpenSuccesses = 0
	}
}

// setState changes the circuit breaker state with a callback
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()
	
	if cb.onStateChange != nil {
		go cb.onStateChange(cb.name, oldState, newState)
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.failures = 0
	cb.halfOpenSuccesses = 0
	cb.setState(CircuitClosed)
}

// Health returns the health status of the circuit breaker
func (cb *CircuitBreaker) Health() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	var stateName string
	switch cb.state {
	case CircuitClosed:
		stateName = "CLOSED"
	case CircuitOpen:
		stateName = "OPEN"
	case CircuitHalfOpen:
		stateName = "HALF_OPEN"
	}
	
	return map[string]interface{}{
		"name":               cb.name,
		"state":              stateName,
		"failures":           cb.failures,
		"failureThreshold":   cb.failureThreshold,
		"halfOpenSuccesses":  cb.halfOpenSuccesses,
		"lastStateChange":    cb.lastStateChange,
		"resetTimeout":       cb.resetTimeout.String(),
	}
}

// HealthChecker manages health checks for services
type HealthChecker struct {
	checks          map[string]HealthCheck
	checksMutex     sync.RWMutex
	checkInterval   time.Duration
	stopChan        chan struct{}
	isRunning       bool
	overallHealth   bool
	healthListeners []func(bool)
}

// HealthCheck represents a health check function and its metadata
type HealthCheck struct {
	Name        string
	Description string
	CheckFn     func() (bool, string)
	IsCritical  bool
	LastStatus  bool
	LastMessage string
	LastChecked time.Time
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(checkInterval time.Duration) *HealthChecker {
	return &HealthChecker{
		checks:        make(map[string]HealthCheck),
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
		isRunning:     false,
		overallHealth: true,
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(name, description string, checkFn func() (bool, string), isCritical bool) {
	hc.checksMutex.Lock()
	defer hc.checksMutex.Unlock()
	
	hc.checks[name] = HealthCheck{
		Name:        name,
		Description: description,
		CheckFn:     checkFn,
		IsCritical:  isCritical,
		LastStatus:  true, // Assume initially healthy
		LastMessage: "Not checked yet",
	}
}

// RemoveCheck removes a health check
func (hc *HealthChecker) RemoveCheck(name string) {
	hc.checksMutex.Lock()
	defer hc.checksMutex.Unlock()
	
	delete(hc.checks, name)
}

// AddHealthListener adds a listener for health state changes
func (hc *HealthChecker) AddHealthListener(listener func(bool)) {
	hc.checksMutex.Lock()
	defer hc.checksMutex.Unlock()
	
	hc.healthListeners = append(hc.healthListeners, listener)
}

// Start begins the health checking process
func (hc *HealthChecker) Start(ctx context.Context) {
	if hc.isRunning {
		return
	}
	
	hc.isRunning = true
	ticker := time.NewTicker(hc.checkInterval)
	
	go func() {
		// Perform initial health check
		hc.performHealthChecks()
		
		for {
			select {
			case <-ticker.C:
				hc.performHealthChecks()
			case <-hc.stopChan:
				ticker.Stop()
				hc.isRunning = false
				return
			case <-ctx.Done():
				ticker.Stop()
				hc.isRunning = false
				return
			}
		}
	}()
}

// Stop stops the health checking process
func (hc *HealthChecker) Stop() {
	if !hc.isRunning {
		return
	}
	
	close(hc.stopChan)
}

// performHealthChecks runs all registered health checks
func (hc *HealthChecker) performHealthChecks() {
	hc.checksMutex.Lock()
	defer hc.checksMutex.Unlock()
	
	wasHealthy := hc.overallHealth
	hc.overallHealth = true
	
	for name, check := range hc.checks {
		healthy, message := check.CheckFn()
		
		// Update check status
		updatedCheck := check
		updatedCheck.LastStatus = healthy
		updatedCheck.LastMessage = message
		updatedCheck.LastChecked = time.Now()
		hc.checks[name] = updatedCheck
		
		// Update overall health
		if check.IsCritical && !healthy {
			hc.overallHealth = false
		}
	}
	
	// If health state changed, notify listeners
	if wasHealthy != hc.overallHealth {
		for _, listener := range hc.healthListeners {
			go listener(hc.overallHealth)
		}
	}
}

// HealthStatus returns the current health status
func (hc *HealthChecker) HealthStatus() map[string]interface{} {
	hc.checksMutex.RLock()
	defer hc.checksMutex.RUnlock()
	
	details := make(map[string]interface{})
	for name, check := range hc.checks {
		details[name] = map[string]interface{}{
			"healthy":     check.LastStatus,
			"message":     check.LastMessage,
			"lastChecked": check.LastChecked,
			"isCritical":  check.IsCritical,
			"description": check.Description,
		}
	}
	
	return map[string]interface{}{
		"healthy": hc.overallHealth,
		"checks":  details,
	}
}

// MetricsCollector collects and maintains system metrics
type MetricsCollector struct {
	metrics       map[string]*Metric
	metricsMutex  sync.RWMutex
	collectTicker *time.Ticker
	stopChan      chan struct{}
	isRunning     bool
}

// Metric represents a collected metric with its time series data
type Metric struct {
	Name        string
	Description string
	Unit        string
	CollectFn   func() float64
	Values      []MetricValue
	ValuesLock  sync.RWMutex
	MaxPoints   int
}

// MetricValue represents a single data point with timestamp
type MetricValue struct {
	Timestamp time.Time
	Value     float64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics:  make(map[string]*Metric),
		stopChan: make(chan struct{}),
	}
}

// AddMetric adds a new metric to collect
func (mc *MetricsCollector) AddMetric(name, description, unit string, collectFn func() float64, maxPoints int) {
	mc.metricsMutex.Lock()
	defer mc.metricsMutex.Unlock()
	
	if maxPoints <= 0 {
		maxPoints = 100 // Default to 100 data points
	}
	
	mc.metrics[name] = &Metric{
		Name:        name,
		Description: description,
		Unit:        unit,
		CollectFn:   collectFn,
		Values:      make([]MetricValue, 0, maxPoints),
		MaxPoints:   maxPoints,
	}
}

// RemoveMetric removes a metric
func (mc *MetricsCollector) RemoveMetric(name string) {
	mc.metricsMutex.Lock()
	defer mc.metricsMutex.Unlock()
	
	delete(mc.metrics, name)
}

// Start begins collecting metrics at the specified interval
func (mc *MetricsCollector) Start(interval time.Duration) {
	if mc.isRunning {
		return
	}
	
	mc.isRunning = true
	mc.collectTicker = time.NewTicker(interval)
	
	go func() {
		// Collect initial metrics
		mc.collectMetrics()
		
		for {
			select {
			case <-mc.collectTicker.C:
				mc.collectMetrics()
			case <-mc.stopChan:
				mc.collectTicker.Stop()
				mc.isRunning = false
				return
			}
		}
	}()
}

// Stop stops collecting metrics
func (mc *MetricsCollector) Stop() {
	if !mc.isRunning {
		return
	}
	
	close(mc.stopChan)
}

// collectMetrics collects all registered metrics
func (mc *MetricsCollector) collectMetrics() {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()
	
	now := time.Now()
	
	for _, metric := range mc.metrics {
		if metric.CollectFn != nil {
			value := metric.CollectFn()
			
			metric.ValuesLock.Lock()
			metric.Values = append(metric.Values, MetricValue{
				Timestamp: now,
				Value:     value,
			})
			
			// Prune old values if exceeding max points
			if len(metric.Values) > metric.MaxPoints {
				metric.Values = metric.Values[len(metric.Values)-metric.MaxPoints:]
			}
			metric.ValuesLock.Unlock()
		}
	}
}

// GetMetricData returns the time series data for a metric
func (mc *MetricsCollector) GetMetricData(name string) ([]MetricValue, error) {
	mc.metricsMutex.RLock()
	metric, exists := mc.metrics[name]
	mc.metricsMutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("metric %s not found", name)
	}
	
	metric.ValuesLock.RLock()
	defer metric.ValuesLock.RUnlock()
	
	// Return a copy to avoid concurrent access issues
	values := make([]MetricValue, len(metric.Values))
	copy(values, metric.Values)
	
	return values, nil
}

// GetAllMetrics returns data for all metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]interface{} {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()
	
	result := make(map[string]interface{})
	
	for name, metric := range mc.metrics {
		metric.ValuesLock.RLock()
		
		metricData := map[string]interface{}{
			"description": metric.Description,
			"unit":        metric.Unit,
		}
		
		// Get the most recent value if available
		if len(metric.Values) > 0 {
			lastValue := metric.Values[len(metric.Values)-1]
			metricData["latest_value"] = lastValue.Value
			metricData["latest_time"] = lastValue.Timestamp
		}
		
		result[name] = metricData
		metric.ValuesLock.RUnlock()
	}
	
	return result
}

// AddStandardSystemMetrics adds standard system metrics to the collector
func (mc *MetricsCollector) AddStandardSystemMetrics() {
	// CPU Usage
	mc.AddMetric("cpu_usage", "CPU Usage Percentage", "%", func() float64 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return float64(m.Sys) / 1024 / 1024
	}, 100)
	
	// Memory Usage
	mc.AddMetric("memory_usage", "Memory Usage in MB", "MB", func() float64 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return float64(m.Alloc) / 1024 / 1024
	}, 100)
	
	// Goroutines
	mc.AddMetric("goroutines", "Number of Goroutines", "count", func() float64 {
		return float64(runtime.NumGoroutine())
	}, 100)
}

// MonitoringServer is an HTTP server that provides health and metrics endpoints
type MonitoringServer struct {
	healthChecker    *HealthChecker
	metricsCollector *MetricsCollector
	circuitBreakers  map[string]*CircuitBreaker
	cbMutex          sync.RWMutex
	port             int
	server           *http.Server
}

// NewMonitoringServer creates a new monitoring server
func NewMonitoringServer(healthChecker *HealthChecker, metricsCollector *MetricsCollector, port int) *MonitoringServer {
	return &MonitoringServer{
		healthChecker:    healthChecker,
		metricsCollector: metricsCollector,
		circuitBreakers:  make(map[string]*CircuitBreaker),
		port:             port,
	}
}

// RegisterCircuitBreaker registers a circuit breaker for monitoring
func (ms *MonitoringServer) RegisterCircuitBreaker(cb *CircuitBreaker) {
	ms.cbMutex.Lock()
	defer ms.cbMutex.Unlock()
	
	ms.circuitBreakers[cb.name] = cb
}

// Start starts the monitoring server
func (ms *MonitoringServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		health := ms.healthChecker.HealthStatus()
		
		// Set appropriate status code
		if !health["healthy"].(bool) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})
	
	// Liveness probe
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Readiness probe
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		health := ms.healthChecker.HealthStatus()
		
		if !health["healthy"].(bool) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})
	
	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := ms.metricsCollector.GetAllMetrics()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})
	
	// Circuit breaker status endpoint
	mux.HandleFunc("/circuitbreakers", func(w http.ResponseWriter, r *http.Request) {
		ms.cbMutex.RLock()
		defer ms.cbMutex.RUnlock()
		
		cbStatus := make(map[string]interface{})
		for name, cb := range ms.circuitBreakers {
			cbStatus[name] = cb.Health()
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cbStatus)
	})
	
	// Detailed metric data endpoint
	mux.HandleFunc("/metrics/", func(w http.ResponseWriter, r *http.Request) {
		metricName := r.URL.Path[len("/metrics/"):]
		if metricName == "" {
			http.NotFound(w, r)
			return
		}
		
		values, err := ms.metricsCollector.GetMetricData(metricName)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(values)
	})
	
	// Create server
	ms.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ms.port),
		Handler: mux,
	}
	
	// Start server in goroutine
	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Monitoring server error: %v", err)
		}
	}()
	
	// Wait for context to be done
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := ms.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Monitoring server shutdown error: %v", err)
		}
	}()
	
	return nil
}

// Logger provides structured logging with different levels
type Logger struct {
	component string
	level     LogLevel
}

// LogLevel represents a logging level
type LogLevel int

const (
	// LogLevelDebug is for detailed debugging information
	LogLevelDebug LogLevel = iota
	
	// LogLevelInfo is for general information
	LogLevelInfo
	
	// LogLevelWarn is for warning information
	LogLevelWarn
	
	// LogLevelError is for error information
	LogLevelError
	
	// LogLevelFatal is for fatal errors that cause the app to exit
	LogLevelFatal
)

var logLevelNames = map[LogLevel]string{
	LogLevelDebug: "DEBUG",
	LogLevelInfo:  "INFO",
	LogLevelWarn:  "WARN",
	LogLevelError: "ERROR",
	LogLevelFatal: "FATAL",
}

// NewLogger creates a new logger
func NewLogger(component string, level LogLevel) *Logger {
	return &Logger{
		component: component,
		level:     level,
	}
}

// log logs a message at the specified level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	
	levelStr := logLevelNames[level]
	msg := fmt.Sprintf(format, args...)
	
	logMsg := fmt.Sprintf(
		"%s [%s] %s: %s",
		time.Now().Format(time.RFC3339),
		levelStr,
		l.component,
		msg,
	)
	
	if level == LogLevelFatal {
		fmt.Fprintln(os.Stderr, logMsg)
		os.Exit(1)
	} else if level == LogLevelError {
		fmt.Fprintln(os.Stderr, logMsg)
	} else {
		fmt.Println(logMsg)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogLevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogLevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogLevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogLevelError, format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LogLevelFatal, format, args...)
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}
