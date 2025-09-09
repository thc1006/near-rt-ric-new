/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProductionHardeningManager coordinates all production hardening features
type ProductionHardeningManager struct {
	mu                        sync.RWMutex
	errorHandler             *ErrorHandler
	circuitBreakerManager    *CircuitBreakerManager
	connectionPoolManager    *ConnectionPoolManager
	gracefulDegradationMgr   *GracefulDegradationManager
	structuredLogger         *StructuredLogger
	ctx                      context.Context
	cancel                   context.CancelFunc
	isRunning                bool
	
	// Configuration
	config                   *ProductionHardeningConfig
	
	// Metrics
	metrics                  *ProductionMetrics
}

// ProductionHardeningConfig contains configuration for production hardening
type ProductionHardeningConfig struct {
	// Error handling configuration
	MaxRetries              int           `json:"maxRetries"`
	RetryBackoff           time.Duration `json:"retryBackoff"`
	ErrorThreshold         int           `json:"errorThreshold"`
	
	// Circuit breaker configuration
	CircuitBreaker         CircuitBreakerConfig `json:"circuitBreaker"`
	
	// Connection pooling configuration
	ConnectionPool         ConnectionPoolConfig `json:"connectionPool"`
	
	// Graceful degradation configuration
	HealthCheckInterval    time.Duration `json:"healthCheckInterval"`
	DegradationThreshold   float64       `json:"degradationThreshold"`
	
	// Logging configuration
	LogLevel               string        `json:"logLevel"`
	LogFormat              string        `json:"logFormat"`
	EnableCorrelationIDs   bool          `json:"enableCorrelationIDs"`
	
	// Resource management
	MaxConnections         int           `json:"maxConnections"`
	ConnectionTimeout      time.Duration `json:"connectionTimeout"`
	IdleTimeout           time.Duration `json:"idleTimeout"`
}

// ProductionMetrics tracks production hardening metrics
type ProductionMetrics struct {
	mu                     sync.RWMutex
	
	// Error handling metrics
	TotalErrors           int64         `json:"totalErrors"`
	RecoveredErrors       int64         `json:"recoveredErrors"`
	ErrorsByComponent     map[string]int64 `json:"errorsByComponent"`
	
	// Circuit breaker metrics
	CircuitBreakerTrips   int64         `json:"circuitBreakerTrips"`
	CircuitBreakerResets  int64         `json:"circuitBreakerResets"`
	
	// Connection metrics
	ActiveConnections     int64         `json:"activeConnections"`
	PooledConnections     int64         `json:"pooledConnections"`
	ConnectionFailures    int64         `json:"connectionFailures"`
	
	// Degradation metrics
	ServicesInDegradation int64         `json:"servicesInDegradation"`
	FallbackExecutions    int64         `json:"fallbackExecutions"`
	
	// Resource metrics
	ResourceExhaustion    int64         `json:"resourceExhaustion"`
	RecoveryOperations    int64         `json:"recoveryOperations"`
	
	LastUpdated           time.Time     `json:"lastUpdated"`
}

// ConnectionPoolManager manages connection pools with production-grade features
type ConnectionPoolManager struct {
	mu            sync.RWMutex
	pools         map[string]*EnhancedConnectionPool
	config        ConnectionPoolConfig
	metrics       *ConnectionPoolMetrics
	
	// Resource monitoring
	resourceMonitor *ResourceMonitor
	
	// Health checking
	healthChecker   *PoolHealthChecker
}

// EnhancedConnectionPool provides production-grade connection pooling
type EnhancedConnectionPool struct {
	mu                 sync.RWMutex
	connections        chan *PooledConnection
	activeConnections  map[string]*PooledConnection
	factory           ConnectionFactory
	config            ConnectionPoolConfig
	
	// Monitoring
	metrics           *PoolMetrics
	
	// Health management
	healthStatus      PoolHealthStatus
	lastHealthCheck   time.Time
	
	// Resource management
	createdCount      int64
	destroyedCount    int64
	requestCount      int64
	errorCount        int64
}

// PooledConnection represents a connection in the pool
type PooledConnection struct {
	conn          interface{}
	id            string
	createdAt     time.Time
	lastUsed      time.Time
	useCount      int64
	isHealthy     bool
	metadata      map[string]interface{}
	
	// Context for request correlation
	correlationID string
}

// StructuredLogger provides production-grade logging with correlation IDs
type StructuredLogger struct {
	mu               sync.RWMutex
	baseLogger       *Logger
	correlationIDGen *CorrelationIDGenerator
	logBuffer        *CircularLogBuffer
	
	// Configuration
	enableCorrelation bool
	logLevel         LogLevel
	logFormat        LogFormat
	
	// Metrics
	logMetrics       *LoggingMetrics
}

// CorrelationIDGenerator generates correlation IDs for request tracing
type CorrelationIDGenerator struct {
	mu       sync.RWMutex
	prefix   string
	counter  int64
	nodeID   string
}

// CircularLogBuffer provides efficient log buffering
type CircularLogBuffer struct {
	mu       sync.RWMutex
	buffer   []LogEntry
	size     int
	head     int
	tail     int
	count    int
}

// LogEntry type is now defined in types.go to avoid redeclaration

// ResourceMonitor type is now defined in types.go to avoid redeclaration

// NewProductionHardeningManager creates a new production hardening manager
func NewProductionHardeningManager(config *ProductionHardeningConfig) *ProductionHardeningManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	phm := &ProductionHardeningManager{
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		metrics: &ProductionMetrics{
			ErrorsByComponent: make(map[string]int64),
			LastUpdated:      time.Now(),
		},
	}
	
	// Initialize components
	phm.initializeComponents()
	
	return phm
}

// Start starts the production hardening manager
func (phm *ProductionHardeningManager) Start() error {
	phm.mu.Lock()
	defer phm.mu.Unlock()
	
	if phm.isRunning {
		return fmt.Errorf("production hardening manager is already running")
	}
	
	// Start error handler
	if err := phm.errorHandler.Start(); err != nil {
		return fmt.Errorf("failed to start error handler: %w", err)
	}
	
	// Start graceful degradation manager
	if err := phm.gracefulDegradationMgr.Start(); err != nil {
		return fmt.Errorf("failed to start graceful degradation manager: %w", err)
	}
	
	// Start connection pool manager
	if err := phm.connectionPoolManager.Start(); err != nil {
		return fmt.Errorf("failed to start connection pool manager: %w", err)
	}
	
	// Start monitoring routines
	go phm.metricsCollectionRoutine()
	go phm.healthMonitoringRoutine()
	go phm.resourceMonitoringRoutine()
	
	phm.isRunning = true
	
	log.Println("Production hardening manager started successfully")
	return nil
}

// Stop stops the production hardening manager
func (phm *ProductionHardeningManager) Stop() error {
	phm.mu.Lock()
	defer phm.mu.Unlock()
	
	if !phm.isRunning {
		return nil
	}
	
	// Stop all components
	phm.cancel()
	
	if phm.errorHandler != nil {
		phm.errorHandler.Stop()
	}
	
	if phm.gracefulDegradationMgr != nil {
		phm.gracefulDegradationMgr.Stop()
	}
	
	if phm.connectionPoolManager != nil {
		phm.connectionPoolManager.Stop()
	}
	
	phm.isRunning = false
	
	log.Println("Production hardening manager stopped")
	return nil
}

// ExecuteWithHardening executes a function with full production hardening
func (phm *ProductionHardeningManager) ExecuteWithHardening(ctx context.Context, component, operation string, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	// Add correlation ID to context
	correlationID := phm.structuredLogger.GenerateCorrelationID()
	ctx = WithCorrelationID(ctx, correlationID)
	
	// Log operation start
	phm.structuredLogger.InfoCtx(ctx, "Starting operation",
		"component", component,
		"operation", operation,
		"correlation_id", correlationID)
	
	// Execute with circuit breaker protection
	circuitBreaker := phm.circuitBreakerManager.GetCircuitBreaker(fmt.Sprintf("%s_%s", component, operation))
	
	result, err := circuitBreaker.Execute(ctx, func() error {
		// Execute with graceful degradation
		res, execErr := phm.gracefulDegradationMgr.ExecuteWithDegradation(ctx, component, func() (interface{}, error) {
			return fn(ctx)
		})
		
		if execErr != nil {
			return execErr
		}
		
		// Store result in context for potential use
		return nil
	})
	
	if err != nil {
		// Handle error with comprehensive error handling
		errorRecord := phm.errorHandler.HandleError(ctx, 
			ErrorTypeOperation, 
			ErrorSeverityMedium, 
			component, 
			operation, 
			err.Error(), 
			err, 
			map[string]interface{}{
				"correlation_id": correlationID,
			})
		
		phm.structuredLogger.ErrorCtx(ctx, "Operation failed",
			"component", component,
			"operation", operation,
			"error", err.Error(),
			"error_id", errorRecord.ID)
		
		// Update metrics
		phm.updateErrorMetrics(component, err)
		
		return nil, err
	}
	
	phm.structuredLogger.InfoCtx(ctx, "Operation completed successfully",
		"component", component,
		"operation", operation,
		"correlation_id", correlationID)
	
	return result, nil
}

// GetConnection gets a managed connection with production-grade features
func (phm *ProductionHardeningManager) GetConnection(ctx context.Context, poolName, address string) (*PooledConnection, error) {
	return phm.connectionPoolManager.GetConnection(ctx, poolName, address)
}

// ReturnConnection returns a connection to the pool
func (phm *ProductionHardeningManager) ReturnConnection(poolName string, conn *PooledConnection) error {
	return phm.connectionPoolManager.ReturnConnection(poolName, conn)
}

// GetMetrics returns production hardening metrics
func (phm *ProductionHardeningManager) GetMetrics() *ProductionMetrics {
	phm.metrics.mu.RLock()
	defer phm.metrics.mu.RUnlock()
	
	// Create a copy of metrics
	metrics := &ProductionMetrics{
		TotalErrors:           phm.metrics.TotalErrors,
		RecoveredErrors:       phm.metrics.RecoveredErrors,
		CircuitBreakerTrips:   phm.metrics.CircuitBreakerTrips,
		CircuitBreakerResets:  phm.metrics.CircuitBreakerResets,
		ActiveConnections:     phm.metrics.ActiveConnections,
		PooledConnections:     phm.metrics.PooledConnections,
		ConnectionFailures:    phm.metrics.ConnectionFailures,
		ServicesInDegradation: phm.metrics.ServicesInDegradation,
		FallbackExecutions:    phm.metrics.FallbackExecutions,
		ResourceExhaustion:    phm.metrics.ResourceExhaustion,
		RecoveryOperations:    phm.metrics.RecoveryOperations,
		ErrorsByComponent:     make(map[string]int64),
		LastUpdated:          time.Now(),
	}
	
	// Copy error by component map
	for k, v := range phm.metrics.ErrorsByComponent {
		metrics.ErrorsByComponent[k] = v
	}
	
	return metrics
}

// Private methods

func (phm *ProductionHardeningManager) initializeComponents() {
	// Initialize error handler
	phm.errorHandler = NewErrorHandler()
	
	// Initialize circuit breaker manager
	phm.circuitBreakerManager = NewCircuitBreakerManager()
	phm.circuitBreakerManager.SetDefaultConfig(&phm.config.CircuitBreaker)
	
	// Initialize connection pool manager
	phm.connectionPoolManager = NewConnectionPoolManager(phm.config.ConnectionPool)
	
	// Initialize graceful degradation manager
	phm.gracefulDegradationMgr = NewGracefulDegradationManager()
	
	// Initialize structured logger
	phm.structuredLogger = NewStructuredLogger(&StructuredLoggerConfig{
		EnableCorrelation: phm.config.EnableCorrelationIDs,
		LogLevel:         LogLevel(phm.config.LogLevel),
		LogFormat:        LogFormat(phm.config.LogFormat),
	})
	
	// Register production error handlers
	phm.registerProductionErrorHandlers()
	
	// Register circuit breaker callbacks
	phm.registerCircuitBreakerCallbacks()
}

func (phm *ProductionHardeningManager) registerProductionErrorHandlers() {
	// Connection recovery handler with circuit breaker integration
	phm.errorHandler.RegisterRecoveryHandler(ErrorTypeConnection, &ProductionConnectionRecoveryHandler{
		circuitBreakerManager: phm.circuitBreakerManager,
		connectionPoolManager: phm.connectionPoolManager,
	})
	
	// Resource recovery handler
	phm.errorHandler.RegisterRecoveryHandler(ErrorTypeResource, &ProductionResourceRecoveryHandler{
		resourceMonitor: phm.connectionPoolManager.resourceMonitor,
	})
	
	// Protocol recovery handler with enhanced logic
	phm.errorHandler.RegisterRecoveryHandler(ErrorTypeProtocol, &ProductionProtocolRecoveryHandler{
		circuitBreakerManager: phm.circuitBreakerManager,
	})
}

func (phm *ProductionHardeningManager) registerCircuitBreakerCallbacks() {
	// Set state change callback for all circuit breakers
	phm.circuitBreakerManager.SetGlobalStateChangeCallback(func(name string, from, to CircuitState) {
		phm.structuredLogger.InfoCtx(phm.ctx, "Circuit breaker state changed",
			"circuit_breaker", name,
			"from_state", from.String(),
			"to_state", to.String())
		
		// Update metrics
		phm.metrics.mu.Lock()
		if to == StateOpen {
			phm.metrics.CircuitBreakerTrips++
		} else if from == StateOpen && to == StateClosed {
			phm.metrics.CircuitBreakerResets++
		}
		phm.metrics.mu.Unlock()
	})
}

func (phm *ProductionHardeningManager) metricsCollectionRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-phm.ctx.Done():
			return
		case <-ticker.C:
			phm.collectMetrics()
		}
	}
}

func (phm *ProductionHardeningManager) healthMonitoringRoutine() {
	ticker := time.NewTicker(phm.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-phm.ctx.Done():
			return
		case <-ticker.C:
			phm.performHealthChecks()
		}
	}
}

func (phm *ProductionHardeningManager) resourceMonitoringRoutine() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-phm.ctx.Done():
			return
		case <-ticker.C:
			phm.monitorResources()
		}
	}
}

func (phm *ProductionHardeningManager) collectMetrics() {
	phm.metrics.mu.Lock()
	defer phm.metrics.mu.Unlock()
	
	// Collect connection pool metrics
	if phm.connectionPoolManager != nil {
		poolMetrics := phm.connectionPoolManager.GetMetrics()
		phm.metrics.ActiveConnections = poolMetrics.ActiveConnections
		phm.metrics.PooledConnections = poolMetrics.PooledConnections
		phm.metrics.ConnectionFailures = poolMetrics.ConnectionFailures
	}
	
	// Collect graceful degradation metrics
	if phm.gracefulDegradationMgr != nil {
		serviceHealth := phm.gracefulDegradationMgr.GetAllServiceHealth()
		degradedCount := int64(0)
		for _, health := range serviceHealth {
			if health.Status == ServiceStatusDegraded {
				degradedCount++
			}
		}
		phm.metrics.ServicesInDegradation = degradedCount
	}
	
	phm.metrics.LastUpdated = time.Now()
}

func (phm *ProductionHardeningManager) performHealthChecks() {
	// Perform circuit breaker health assessment
	circuitBreakers := phm.circuitBreakerManager.GetAllCircuitBreakers()
	for name, cb := range circuitBreakers {
		stats := cb.GetStatistics()
		if stats.FailureRate > 50.0 && stats.State == "OPEN" {
			phm.structuredLogger.WarnCtx(phm.ctx, "Circuit breaker in poor health",
				"circuit_breaker", name,
				"failure_rate", stats.FailureRate,
				"state", stats.State)
		}
	}
}

func (phm *ProductionHardeningManager) monitorResources() {
	if phm.connectionPoolManager != nil && phm.connectionPoolManager.resourceMonitor != nil {
		resources := phm.connectionPoolManager.resourceMonitor.GetResourceUsage()
		
		// Check for resource exhaustion
		if resources.MemoryUsage > 90.0 || resources.CPUUsage > 95.0 {
			phm.metrics.mu.Lock()
			phm.metrics.ResourceExhaustion++
			phm.metrics.mu.Unlock()
			
			phm.structuredLogger.WarnCtx(phm.ctx, "High resource usage detected",
				"cpu_usage", resources.CPUUsage,
				"memory_usage", resources.MemoryUsage,
				"network_usage", resources.NetworkUsage)
		}
	}
}

func (phm *ProductionHardeningManager) updateErrorMetrics(component string, err error) {
	phm.metrics.mu.Lock()
	defer phm.metrics.mu.Unlock()
	
	phm.metrics.TotalErrors++
	phm.metrics.ErrorsByComponent[component]++
}

// Default production hardening configuration
func DefaultProductionHardeningConfig() *ProductionHardeningConfig {
	return &ProductionHardeningConfig{
		MaxRetries:           3,
		RetryBackoff:        time.Second,
		ErrorThreshold:      10,
		CircuitBreaker: CircuitBreakerConfig{
			MaxFailures:      5,
			Timeout:          30 * time.Second,
			ResetTimeout:     60 * time.Second,
			HalfOpenMaxCalls: 3,
		},
		ConnectionPool: ConnectionPoolConfig{
			MaxIdle:     10,
			MaxActive:   50,
			IdleTimeout: 5 * time.Minute,
		},
		HealthCheckInterval:  30 * time.Second,
		DegradationThreshold: 75.0,
		LogLevel:            "INFO",
		LogFormat:           "JSON",
		EnableCorrelationIDs: true,
		MaxConnections:      100,
		ConnectionTimeout:   10 * time.Second,
		IdleTimeout:        5 * time.Minute,
	}
}
