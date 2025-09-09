/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// EnhancedDashboardAPI provides production-hardened API for 100+ concurrent E2 nodes
type EnhancedDashboardAPI struct {
	// Core components
	smoOptimizer        *SMOPerformanceOptimizer
	messageProcessor    *HighPerformanceMessageProcessor
	connectionManager   *ConnectionManager
	
	// API server components
	server              *http.Server
	router              *mux.Router
	upgrader            websocket.Upgrader
	
	// Connection pools and management
	e2NodeManager       *E2NodeManager
	subscriptionManager *SubscriptionManager
	wsConnectionPool    *WebSocketConnectionPool
	
	// Rate limiting and throttling
	rateLimiter         *AdaptiveRateLimiter
	circuitBreaker      *CircuitBreaker
	
	// Caching and performance
	responseCache       *ResponseCache
	compressionManager  *CompressionManager
	
	// Monitoring and metrics
	metricsRegistry     *prometheus.Registry
	performanceMetrics  *DashboardMetrics
	healthChecker       *HealthChecker
	
	// Security and hardening
	authManager         *AuthManager
	tlsConfig           *TLSConfig
	securityHeaders     *SecurityHeaders
	
	// Configuration
	config              *DashboardConfig
	
	// State management
	running             int32
	activeConnections   int64
	requestCount        uint64
	mu                  sync.RWMutex
}

// DashboardConfig defines enhanced dashboard configuration
type DashboardConfig struct {
	// Server configuration
	ListenAddress       string        `json:"listenAddress"`
	ListenPort          int           `json:"listenPort"`
	TLSEnabled          bool          `json:"tlsEnabled"`
	CertFile            string        `json:"certFile"`
	KeyFile             string        `json:"keyFile"`
	
	// Performance settings
	MaxConcurrentE2Nodes    int       `json:"maxConcurrentE2Nodes"`    // 100+ target
	MaxConcurrentConnections int      `json:"maxConcurrentConnections"` // WebSocket connections
	RequestTimeout          time.Duration `json:"requestTimeout"`
	KeepAliveTimeout        time.Duration `json:"keepAliveTimeout"`
	
	// Rate limiting
	RateLimit               int       `json:"rateLimit"`              // Requests per second
	BurstSize               int       `json:"burstSize"`
	AdaptiveRateLimiting    bool      `json:"adaptiveRateLimiting"`
	
	// Caching
	EnableResponseCache     bool      `json:"enableResponseCache"`
	CacheTTL                time.Duration `json:"cacheTTL"`
	MaxCacheSize            int       `json:"maxCacheSize"`
	
	// Compression
	EnableCompression       bool      `json:"enableCompression"`
	CompressionLevel        int       `json:"compressionLevel"`
	MinCompressionSize      int       `json:"minCompressionSize"`
	
	// WebSocket settings
	WSReadBufferSize        int       `json:"wsReadBufferSize"`
	WSWriteBufferSize       int       `json:"wsWriteBufferSize"`
	WSPingInterval          time.Duration `json:"wsPingInterval"`
	WSPongTimeout           time.Duration `json:"wsPongTimeout"`
	
	// Circuit breaker
	CircuitBreakerEnabled   bool      `json:"circuitBreakerEnabled"`
	FailureThreshold        int       `json:"failureThreshold"`
	RecoveryTimeout         time.Duration `json:"recoveryTimeout"`
	
	// Security
	EnableJWTAuth           bool      `json:"enableJWTAuth"`
	JWTSecret               string    `json:"jwtSecret"`
	CORSEnabled             bool      `json:"corsEnabled"`
	CORSOrigins             []string  `json:"corsOrigins"`
}

// DashboardMetrics tracks dashboard performance metrics
type DashboardMetrics struct {
	// HTTP metrics
	HTTPRequestsTotal       *prometheus.CounterVec
	HTTPRequestDuration     *prometheus.HistogramVec
	HTTPResponseSize        *prometheus.HistogramVec
	ActiveHTTPConnections   prometheus.Gauge
	
	// WebSocket metrics
	WSConnectionsTotal      *prometheus.CounterVec
	WSActiveConnections     prometheus.Gauge
	WSMessagesSent          *prometheus.CounterVec
	WSMessagesReceived      *prometheus.CounterVec
	
	// E2 Node metrics
	E2NodesConnected        prometheus.Gauge
	E2NodesActive           prometheus.Gauge
	E2SubscriptionsActive   prometheus.Gauge
	E2IndicationsPerSecond  prometheus.Gauge
	
	// Performance metrics
	APILatency              *prometheus.HistogramVec
	CacheHitRatio           prometheus.Gauge
	CPUUtilization          prometheus.Gauge
	MemoryUtilization       prometheus.Gauge
	
	// Error metrics
	ErrorsTotal             *prometheus.CounterVec
	CircuitBreakerState     prometheus.Gauge
}

// E2NodeManager manages concurrent E2 node connections
type E2NodeManager struct {
	nodes               map[string]*E2NodeConnection
	nodeStats           map[string]*E2NodeStats
	maxConcurrentNodes  int
	connectionPool      *ConnectionPool
	healthMonitor       *E2HealthMonitor
	loadBalancer        *E2LoadBalancer
	mu                  sync.RWMutex
}

// E2NodeConnection represents an enhanced E2 node connection
type E2NodeConnection struct {
	NodeID              string
	GlobalE2NodeID      string
	PLMNIdentity        string
	Address             string
	Port                int
	ConnectionState     E2ConnectionState
	LastHeartbeat       time.Time
	Subscriptions       map[string]*E2Subscription
	Statistics          *E2NodeStats
	RateLimiter         *NodeRateLimiter
	CircuitBreaker      *NodeCircuitBreaker
	Context             context.Context
	Cancel              context.CancelFunc
	mu                  sync.RWMutex
}

// E2NodeStats tracks per-node statistics
type E2NodeStats struct {
	ConnectionTime      time.Time
	MessagesReceived    uint64
	MessagesSent        uint64
	IndicationsReceived uint64
	ControlMessagesSent uint64
	LastMessageTime     time.Time
	AverageLatencyMs    float64
	ErrorCount          uint64
	HealthScore         float64
}

// E2ConnectionState represents E2 node connection state
type E2ConnectionState int

const (
	E2StateDisconnected E2ConnectionState = iota
	E2StateConnecting
	E2StateConnected
	E2StateSetupComplete
	E2StateError
)

// WebSocketConnectionPool manages WebSocket connections efficiently
type WebSocketConnectionPool struct {
	connections         map[string]*WSConnection
	connectionsByType   map[string][]*WSConnection
	maxConnections      int
	cleanupInterval     time.Duration
	stats               WSPoolStats
	mu                  sync.RWMutex
}

// WSConnection represents an enhanced WebSocket connection
type WSConnection struct {
	ID                  string
	Conn                *websocket.Conn
	Type                string // dashboard, monitoring, admin
	LastActivity        time.Time
	MessageCount        uint64
	BytesSent           uint64
	BytesReceived       uint64
	RateLimiter         *ConnectionRateLimiter
	Context             context.Context
	Cancel              context.CancelFunc
	mu                  sync.RWMutex
}

// WSPoolStats tracks WebSocket pool statistics
type WSPoolStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	MessagesSent        uint64
	MessagesReceived    uint64
	BytesTransferred    uint64
	ConnectionErrors    uint64
}

// AdaptiveRateLimiter provides adaptive rate limiting
type AdaptiveRateLimiter struct {
	baseLimit           int
	currentLimit        int
	burstSize           int
	window              time.Duration
	requests            map[string]*ClientRequestTracker
	systemLoad          *SystemLoadTracker
	adaptationEnabled   bool
	mu                  sync.RWMutex
}

// ClientRequestTracker tracks requests per client
type ClientRequestTracker struct {
	requests            []time.Time
	violations          int
	lastRequest         time.Time
	rateLimitHits       int
}

// SystemLoadTracker tracks system load for adaptive limiting
type SystemLoadTracker struct {
	cpuUsage            float64
	memoryUsage         float64
	activeConnections   int64
	requestsPerSecond   float64
	lastUpdate          time.Time
}

// ResponseCache provides intelligent response caching
type ResponseCache struct {
	cache               map[string]*CacheEntry
	maxSize             int
	ttl                 time.Duration
	hits                uint64
	misses              uint64
	evictions           uint64
	mu                  sync.RWMutex
}

// CacheEntry represents a cached response
type CacheEntry struct {
	Data                []byte
	ContentType         string
	Timestamp           time.Time
	AccessCount         int
	LastAccess          time.Time
	ETag                string
}

// NewEnhancedDashboardAPI creates a new enhanced dashboard API
func NewEnhancedDashboardAPI(config *DashboardConfig, smoOptimizer *SMOPerformanceOptimizer) *EnhancedDashboardAPI {
	if config == nil {
		config = &DashboardConfig{
			ListenAddress:           "0.0.0.0",
			ListenPort:              8080,
			MaxConcurrentE2Nodes:    200,  // Exceed 100+ requirement
			MaxConcurrentConnections: 1000,
			RequestTimeout:          time.Second * 30,
			KeepAliveTimeout:        time.Second * 60,
			RateLimit:              1000,  // 1000 RPS
			BurstSize:              2000,
			AdaptiveRateLimiting:   true,
			EnableResponseCache:    true,
			CacheTTL:               time.Minute * 5,
			MaxCacheSize:           10000,
			EnableCompression:      true,
			CompressionLevel:       6,
			MinCompressionSize:     1024,
			WSReadBufferSize:       4096,
			WSWriteBufferSize:      4096,
			WSPingInterval:         time.Second * 30,
			WSPongTimeout:          time.Second * 10,
			CircuitBreakerEnabled:  true,
			FailureThreshold:       50,
			RecoveryTimeout:        time.Second * 30,
		}
	}

	// Initialize metrics registry
	metricsRegistry := prometheus.NewRegistry()
	
	api := &EnhancedDashboardAPI{
		smoOptimizer:        smoOptimizer,
		messageProcessor:    smoOptimizer.messageProcessor,
		config:              config,
		metricsRegistry:     metricsRegistry,
		performanceMetrics:  NewDashboardMetrics(metricsRegistry),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.WSReadBufferSize,
			WriteBufferSize: config.WSWriteBufferSize,
			CheckOrigin:     func(r *http.Request) bool { return true }, // Configure properly in production
		},
	}

	// Initialize components
	api.e2NodeManager = NewE2NodeManager(config.MaxConcurrentE2Nodes)
	api.subscriptionManager = NewSubscriptionManager()
	api.wsConnectionPool = NewWebSocketConnectionPool(config.MaxConcurrentConnections)
	api.connectionManager = NewConnectionManager(config.MaxConcurrentConnections)

	// Initialize performance components
	api.rateLimiter = NewAdaptiveRateLimiter(config.RateLimit, config.BurstSize, config.AdaptiveRateLimiting)
	api.responseCache = NewResponseCache(config.MaxCacheSize, config.CacheTTL)
	api.compressionManager = NewCompressionManager(config.CompressionLevel, config.MinCompressionSize)

	if config.CircuitBreakerEnabled {
		api.circuitBreaker = NewCircuitBreaker(config.FailureThreshold, config.RecoveryTimeout)
	}

	// Initialize security components
	api.authManager = NewAuthManager(config.EnableJWTAuth, config.JWTSecret)
	api.securityHeaders = NewSecurityHeaders()
	api.healthChecker = NewHealthChecker()

	// Setup router with middleware
	api.setupRouter()

	return api
}

// Start starts the enhanced dashboard API server
func (eda *EnhancedDashboardAPI) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&eda.running, 0, 1) {
		return fmt.Errorf("dashboard API already running")
	}

	logrus.WithFields(logrus.Fields{
		"address":           fmt.Sprintf("%s:%d", eda.config.ListenAddress, eda.config.ListenPort),
		"maxE2Nodes":        eda.config.MaxConcurrentE2Nodes,
		"maxConnections":    eda.config.MaxConcurrentConnections,
		"rateLimitRPS":      eda.config.RateLimit,
		"cacheEnabled":      eda.config.EnableResponseCache,
		"compressionEnabled": eda.config.EnableCompression,
	}).Info("Starting Enhanced Dashboard API")

	// Create HTTP server
	eda.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", eda.config.ListenAddress, eda.config.ListenPort),
		Handler:        eda.router,
		ReadTimeout:    eda.config.RequestTimeout,
		WriteTimeout:   eda.config.RequestTimeout,
		IdleTimeout:    eda.config.KeepAliveTimeout,
		MaxHeaderBytes: 1 << 16, // 64KB
	}

	// Start background services
	go eda.performanceMonitoringLoop(ctx)
	go eda.connectionCleanupLoop(ctx)
	go eda.cacheCleanupLoop(ctx)
	go eda.metricsCollectionLoop(ctx)

	// Start server
	errChan := make(chan error, 1)
	
	go func() {
		if eda.config.TLSEnabled {
			errChan <- eda.server.ListenAndServeTLS(eda.config.CertFile, eda.config.KeyFile)
		} else {
			errChan <- eda.server.ListenAndServe()
		}
	}()

	// Wait for startup completion or error
	select {
	case err := <-errChan:
		if err != http.ErrServerClosed {
			return fmt.Errorf("failed to start server: %w", err)
		}
	case <-time.After(time.Second * 2):
		// Server started successfully
	}

	logrus.Info("Enhanced Dashboard API started successfully")
	return nil
}

// setupRouter configures the HTTP router with all endpoints and middleware
func (eda *EnhancedDashboardAPI) setupRouter() {
	eda.router = mux.NewRouter()

	// Add middleware
	eda.router.Use(eda.loggingMiddleware)
	eda.router.Use(eda.metricsMiddleware)
	eda.router.Use(eda.securityHeadersMiddleware)
	eda.router.Use(eda.rateLimitingMiddleware)
	eda.router.Use(eda.compressionMiddleware)

	if eda.config.CORSEnabled {
		eda.router.Use(eda.corsMiddleware)
	}

	// Health check endpoints
	eda.router.HandleFunc("/health", eda.handleHealth).Methods("GET")
	eda.router.HandleFunc("/ready", eda.handleReadiness).Methods("GET")
	eda.router.HandleFunc("/metrics", promhttp.HandlerFor(eda.metricsRegistry, promhttp.HandlerOpts{}).ServeHTTP).Methods("GET")

	// E2 Node management endpoints
	e2Router := eda.router.PathPrefix("/api/v1/e2").Subrouter()
	e2Router.HandleFunc("/nodes", eda.handleGetE2Nodes).Methods("GET")
	e2Router.HandleFunc("/nodes/{nodeId}", eda.handleGetE2Node).Methods("GET")
	e2Router.HandleFunc("/nodes/{nodeId}/stats", eda.handleGetE2NodeStats).Methods("GET")
	e2Router.HandleFunc("/nodes/{nodeId}/subscriptions", eda.handleGetE2NodeSubscriptions).Methods("GET")
	e2Router.HandleFunc("/nodes/{nodeId}/control", eda.handleE2Control).Methods("POST")

	// Subscription management endpoints
	subRouter := eda.router.PathPrefix("/api/v1/subscriptions").Subrouter()
	subRouter.HandleFunc("", eda.handleGetSubscriptions).Methods("GET")
	subRouter.HandleFunc("", eda.handleCreateSubscription).Methods("POST")
	subRouter.HandleFunc("/{subId}", eda.handleGetSubscription).Methods("GET")
	subRouter.HandleFunc("/{subId}", eda.handleUpdateSubscription).Methods("PUT")
	subRouter.HandleFunc("/{subId}", eda.handleDeleteSubscription).Methods("DELETE")

	// Performance monitoring endpoints
	perfRouter := eda.router.PathPrefix("/api/v1/performance").Subrouter()
	perfRouter.HandleFunc("/metrics", eda.handleGetPerformanceMetrics).Methods("GET")
	perfRouter.HandleFunc("/dashboard", eda.handleGetDashboardMetrics).Methods("GET")
	perfRouter.HandleFunc("/e2-stats", eda.handleGetE2Stats).Methods("GET")
	perfRouter.HandleFunc("/system-stats", eda.handleGetSystemStats).Methods("GET")

	// SMO integration endpoints
	smoRouter := eda.router.PathPrefix("/api/v1/smo").Subrouter()
	smoRouter.HandleFunc("/status", eda.handleGetSMOStatus).Methods("GET")
	smoRouter.HandleFunc("/policies", eda.handleGetSMOPolicies).Methods("GET")
	smoRouter.HandleFunc("/rapps", eda.handleGetSMORApps).Methods("GET")

	// WebSocket endpoint
	eda.router.HandleFunc("/ws", eda.handleWebSocket)

	// Admin endpoints with authentication
	adminRouter := eda.router.PathPrefix("/api/v1/admin").Subrouter()
	if eda.config.EnableJWTAuth {
		adminRouter.Use(eda.authMiddleware)
	}
	adminRouter.HandleFunc("/config", eda.handleGetConfig).Methods("GET")
	adminRouter.HandleFunc("/config", eda.handleUpdateConfig).Methods("PUT")
	adminRouter.HandleFunc("/cache/clear", eda.handleClearCache).Methods("POST")
	adminRouter.HandleFunc("/connections", eda.handleGetConnections).Methods("GET")
}

// Enhanced E2 Node Management Handlers

func (eda *EnhancedDashboardAPI) handleGetE2Nodes(w http.ResponseWriter, r *http.Request) {
	// Check cache first
	if eda.config.EnableResponseCache {
		if cached := eda.responseCache.Get("e2_nodes"); cached != nil {
			eda.writeCachedResponse(w, cached)
			return
		}
	}

	nodes := eda.e2NodeManager.GetAllNodes()
	response := map[string]interface{}{
		"nodes":       nodes,
		"totalCount":  len(nodes),
		"activeCount": eda.e2NodeManager.GetActiveNodeCount(),
		"timestamp":   time.Now(),
	}

	data, err := json.Marshal(response)
	if err != nil {
		eda.writeErrorResponse(w, http.StatusInternalServerError, "Failed to marshal response")
		return
	}

	// Cache the response
	if eda.config.EnableResponseCache {
		eda.responseCache.Set("e2_nodes", data, "application/json")
	}

	eda.writeJSONResponse(w, http.StatusOK, data)
}

func (eda *EnhancedDashboardAPI) handleGetE2Node(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	node, exists := eda.e2NodeManager.GetNode(nodeID)
	if !exists {
		eda.writeErrorResponse(w, http.StatusNotFound, "E2 node not found")
		return
	}

	response := map[string]interface{}{
		"node":      node,
		"timestamp": time.Now(),
	}

	eda.writeJSONResponse(w, http.StatusOK, response)
}

func (eda *EnhancedDashboardAPI) handleGetE2NodeStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	stats, exists := eda.e2NodeManager.GetNodeStats(nodeID)
	if !exists {
		eda.writeErrorResponse(w, http.StatusNotFound, "E2 node stats not found")
		return
	}

	// Add real-time statistics
	realtimeStats := map[string]interface{}{
		"basic":          stats,
		"realtimeMetrics": eda.smoOptimizer.GetPerformanceMetrics(),
		"connectionInfo": eda.connectionManager.GetConnectionInfo(nodeID),
		"timestamp":      time.Now(),
	}

	eda.writeJSONResponse(w, http.StatusOK, realtimeStats)
}

// Performance Monitoring Handlers

func (eda *EnhancedDashboardAPI) handleGetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	smoMetrics := eda.smoOptimizer.GetPerformanceMetrics()
	processorStats := eda.messageProcessor.GetProcessingStats()
	
	response := map[string]interface{}{
		"smoMetrics":      smoMetrics,
		"processorStats":  processorStats,
		"dashboardStats": eda.getDashboardStats(),
		"systemStats":    eda.getSystemStats(),
		"timestamp":      time.Now(),
	}

	eda.writeJSONResponse(w, http.StatusOK, response)
}

func (eda *EnhancedDashboardAPI) handleGetDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	stats := eda.getDashboardStats()
	eda.writeJSONResponse(w, http.StatusOK, stats)
}

// WebSocket Handler for real-time updates
func (eda *EnhancedDashboardAPI) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	conn, err := eda.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).Error("WebSocket upgrade failed")
		return
	}

	// Create WebSocket connection
	wsConn := &WSConnection{
		ID:           generateConnectionID(),
		Conn:         conn,
		Type:         r.URL.Query().Get("type"),
		LastActivity: time.Now(),
		Context:      context.Background(),
	}

	// Add to connection pool
	eda.wsConnectionPool.AddConnection(wsConn)
	atomic.AddInt64(&eda.activeConnections, 1)

	// Handle connection
	go eda.handleWebSocketConnection(wsConn)
}

// Middleware

func (eda *EnhancedDashboardAPI) rateLimitingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		
		if !eda.rateLimiter.Allow(clientIP) {
			eda.writeErrorResponse(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (eda *EnhancedDashboardAPI) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		atomic.AddUint64(&eda.requestCount, 1)

		// Wrap response writer to capture status code and size
		wrapped := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		eda.performanceMetrics.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(wrapped.statusCode)).Observe(duration.Seconds())
		eda.performanceMetrics.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(wrapped.statusCode)).Inc()
		
		if wrapped.size > 0 {
			eda.performanceMetrics.HTTPResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(wrapped.size))
		}
	})
}

func (eda *EnhancedDashboardAPI) compressionMiddleware(next http.Handler) http.Handler {
	if !eda.config.EnableCompression {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if eda.compressionManager.ShouldCompress(r) {
			compressed := eda.compressionManager.CompressResponse(w)
			next.ServeHTTP(compressed, r)
			compressed.Close()
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// Helper methods

func (eda *EnhancedDashboardAPI) getDashboardStats() map[string]interface{} {
	return map[string]interface{}{
		"activeConnections":    atomic.LoadInt64(&eda.activeConnections),
		"totalRequests":        atomic.LoadUint64(&eda.requestCount),
		"cacheHitRatio":        eda.responseCache.GetHitRatio(),
		"e2NodesConnected":     eda.e2NodeManager.GetActiveNodeCount(),
		"activeSubscriptions":  eda.subscriptionManager.GetActiveCount(),
		"wsConnections":        eda.wsConnectionPool.GetActiveCount(),
		"rateLimitViolations":  eda.rateLimiter.GetViolationCount(),
		"circuitBreakerState":  eda.getCircuitBreakerState(),
		"uptime":              time.Since(eda.getStartTime()),
	}
}

func (eda *EnhancedDashboardAPI) getSystemStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"memoryUsageMB":    m.Alloc / 1024 / 1024,
		"goroutines":       runtime.NumGoroutine(),
		"cpuCores":         runtime.NumCPU(),
		"gcPauseMs":        float64(m.PauseNs[(m.NumGC+255)%256]) / 1000000,
		"heapObjects":      m.HeapObjects,
		"nextGCMB":         m.NextGC / 1024 / 1024,
	}
}

func (eda *EnhancedDashboardAPI) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			logrus.WithError(err).Error("Failed to encode JSON response")
		}
	}
}

func (eda *EnhancedDashboardAPI) writeErrorResponse(w http.ResponseWriter, status int, message string) {
	error := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now(),
		"status":    status,
	}
	eda.writeJSONResponse(w, status, error)
}

// Background monitoring loops

func (eda *EnhancedDashboardAPI) performanceMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eda.updatePerformanceMetrics()
		}
	}
}

func (eda *EnhancedDashboardAPI) updatePerformanceMetrics() {
	// Update Prometheus metrics
	eda.performanceMetrics.ActiveHTTPConnections.Set(float64(atomic.LoadInt64(&eda.activeConnections)))
	eda.performanceMetrics.E2NodesConnected.Set(float64(eda.e2NodeManager.GetActiveNodeCount()))
	eda.performanceMetrics.E2SubscriptionsActive.Set(float64(eda.subscriptionManager.GetActiveCount()))
	eda.performanceMetrics.WSActiveConnections.Set(float64(eda.wsConnectionPool.GetActiveCount()))
	eda.performanceMetrics.CacheHitRatio.Set(eda.responseCache.GetHitRatio())

	// Update system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	eda.performanceMetrics.MemoryUtilization.Set(float64(m.Alloc) / 1024 / 1024)

	// Update SMO-specific metrics
	smoStats := eda.smoOptimizer.GetPerformanceMetrics()
	eda.performanceMetrics.E2IndicationsPerSecond.Set(float64(smoStats.CurrentThroughputIPS))
}

// Connection management loops
func (eda *EnhancedDashboardAPI) connectionCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eda.wsConnectionPool.CleanupStaleConnections()
			eda.e2NodeManager.CleanupStaleNodes()
		}
	}
}

func (eda *EnhancedDashboardAPI) cacheCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eda.responseCache.CleanupExpiredEntries()
		}
	}
}

func (eda *EnhancedDashboardAPI) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eda.collectDetailedMetrics()
		}
	}
}

// Stop gracefully stops the enhanced dashboard API
func (eda *EnhancedDashboardAPI) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&eda.running, 1, 0) {
		return fmt.Errorf("dashboard API not running")
	}

	logrus.Info("Stopping Enhanced Dashboard API")

	// Shutdown HTTP server
	if eda.server != nil {
		if err := eda.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}

	// Close WebSocket connections
	eda.wsConnectionPool.CloseAllConnections()

	// Stop components
	eda.e2NodeManager.Stop()

	logrus.Info("Enhanced Dashboard API stopped")
	return nil
}

// Utility functions and additional helper methods would be implemented here...

// responseWriterWrapper wraps http.ResponseWriter to capture metrics
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

// Additional helper functions...
func getClientIP(r *http.Request) string {
	// Extract client IP from request
	return r.RemoteAddr
}

func generateConnectionID() string {
	// Generate unique connection ID
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}