/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// ProductionDashboardAPI provides production-hardened API for 100+ concurrent E2 nodes
type ProductionDashboardAPI struct {
	// Core performance components
	performanceOptimizer    *AdvancedSMOPerformanceOptimizer
	connectionCluster       *ConnectionPoolCluster
	
	// API server and routing
	httpServer              *http.Server
	httpsServer             *http.Server
	router                  *mux.Router
	
	// WebSocket management
	wsUpgrader              websocket.Upgrader
	wsConnectionManager     *WebSocketConnectionManager
	
	// Load balancing and scaling
	loadBalancer            *APILoadBalancer
	horizontalScaler        *HorizontalScaler
	rateLimiter             *AdaptiveRateLimiter
	
	// Caching and compression
	responseCache           *ResponseCache
	compressionHandler      *CompressionHandler
	
	// Security and authentication
	authManager             *AuthenticationManager
	rateLimitManager        *RateLimitManager
	
	// E2 node management
	e2NodeManager           *ProductionE2NodeManager
	subscriptionManager     *HighPerformanceSubscriptionManager
	indicationProcessor     *RealTimeIndicationProcessor
	
	// Monitoring and observability
	metricsRegistry         *prometheus.Registry
	dashboardMetrics        *DashboardMetrics
	performanceMonitor      *APIPerformanceMonitor
	healthChecker           *APIHealthChecker
	
	// Configuration and state
	config                  *ProductionAPIConfig
	stats                   ProductionAPIStats
	running                 int32
	
	// Connection tracking
	activeConnections       int64
	concurrentRequests      int64
	totalRequests           uint64
	
	mu                      sync.RWMutex
}

// ProductionAPIConfig defines production-grade API configuration
type ProductionAPIConfig struct {
	// Server configuration
	HTTPPort                int           `json:"httpPort"`
	HTTPSPort               int           `json:"httpsPort"`
	ListenAddress           string        `json:"listenAddress"`
	TLSCertFile             string        `json:"tlsCertFile"`
	TLSKeyFile              string        `json:"tlsKeyFile"`
	
	// Performance targets
	MaxConcurrentE2Nodes    int           `json:"maxConcurrentE2Nodes"`    // 100+
	MaxConcurrentUsers      int           `json:"maxConcurrentUsers"`      // 100+
	MaxRequestsPerSecond    int           `json:"maxRequestsPerSecond"`    // High throughput
	TargetAPILatencyMs      int           `json:"targetAPILatencyMs"`      // <100ms
	
	// Connection management
	MaxConnections          int           `json:"maxConnections"`
	ConnectionTimeout       time.Duration `json:"connectionTimeout"`
	KeepAliveTimeout        time.Duration `json:"keepAliveTimeout"`
	ReadTimeout             time.Duration `json:"readTimeout"`
	WriteTimeout            time.Duration `json:"writeTimeout"`
	IdleTimeout             time.Duration `json:"idleTimeout"`
	
	// WebSocket configuration
	WSReadBufferSize        int           `json:"wsReadBufferSize"`
	WSWriteBufferSize       int           `json:"wsWriteBufferSize"`
	WSMaxMessageSize        int64         `json:"wsMaxMessageSize"`
	WSPingInterval          time.Duration `json:"wsPingInterval"`
	WSPongTimeout           time.Duration `json:"wsPongTimeout"`
	WSCompressionEnabled    bool          `json:"wsCompressionEnabled"`
	
	// Rate limiting
	GlobalRateLimit         int           `json:"globalRateLimit"`          // Requests per second
	PerUserRateLimit        int           `json:"perUserRateLimit"`
	BurstSize               int           `json:"burstSize"`
	RateLimitWindowSize     time.Duration `json:"rateLimitWindowSize"`
	
	// Caching
	EnableResponseCache     bool          `json:"enableResponseCache"`
	CacheTTL                time.Duration `json:"cacheTTL"`
	MaxCacheSize            int           `json:"maxCacheSize"`
	CacheCompressionEnabled bool          `json:"cacheCompressionEnabled"`
	
	// Compression
	EnableCompression       bool          `json:"enableCompression"`
	CompressionLevel        int           `json:"compressionLevel"`
	CompressionMinSize      int           `json:"compressionMinSize"`
	CompressionTypes        []string      `json:"compressionTypes"`
	
	// Load balancing
	EnableLoadBalancing     bool          `json:"enableLoadBalancing"`
	LoadBalancingStrategy   string        `json:"loadBalancingStrategy"`
	HealthCheckInterval     time.Duration `json:"healthCheckInterval"`
	
	// Horizontal scaling
	EnableAutoScaling       bool          `json:"enableAutoScaling"`
	ScaleUpThreshold        float64       `json:"scaleUpThreshold"`
	ScaleDownThreshold      float64       `json:"scaleDownThreshold"`
	MinInstances            int           `json:"minInstances"`
	MaxInstances            int           `json:"maxInstances"`
	
	// Security
	EnableAuthentication    bool          `json:"enableAuthentication"`
	JWTSecretKey            string        `json:"jwtSecretKey"`
	TokenExpirationTime     time.Duration `json:"tokenExpirationTime"`
	EnableCORS              bool          `json:"enableCORS"`
	CORSAllowedOrigins      []string      `json:"corsAllowedOrigins"`
	
	// Monitoring
	EnableMetrics           bool          `json:"enableMetrics"`
	MetricsPath             string        `json:"metricsPath"`
	EnableProfiling         bool          `json:"enableProfiling"`
	ProfilingEnabled        bool          `json:"profilingEnabled"`
	
	// Circuit breaker
	EnableCircuitBreaker    bool          `json:"enableCircuitBreaker"`
	CircuitBreakerThreshold int           `json:"circuitBreakerThreshold"`
	CircuitBreakerTimeout   time.Duration `json:"circuitBreakerTimeout"`
}

// ProductionAPIStats tracks comprehensive API performance
type ProductionAPIStats struct {
	// Request metrics
	TotalRequests           uint64        `json:"totalRequests"`
	SuccessfulRequests      uint64        `json:"successfulRequests"`
	FailedRequests          uint64        `json:"failedRequests"`
	RequestsPerSecond       uint64        `json:"requestsPerSecond"`
	
	// Latency metrics
	AverageLatencyMs        float64       `json:"averageLatencyMs"`
	P50LatencyMs            float64       `json:"p50LatencyMs"`
	P95LatencyMs            float64       `json:"p95LatencyMs"`
	P99LatencyMs            float64       `json:"p99LatencyMs"`
	MaxLatencyMs            float64       `json:"maxLatencyMs"`
	
	// Connection metrics
	ActiveConnections       uint64        `json:"activeConnections"`
	TotalConnections        uint64        `json:"totalConnections"`
	ConnectionFailures      uint64        `json:"connectionFailures"`
	ConcurrentUsers         uint64        `json:"concurrentUsers"`
	
	// WebSocket metrics
	WSConnections           uint64        `json:"wsConnections"`
	WSMessagesSent          uint64        `json:"wsMessagesSent"`
	WSMessagesReceived      uint64        `json:"wsMessagesReceived"`
	WSErrors                uint64        `json:"wsErrors"`
	
	// E2 node metrics
	ConnectedE2Nodes        uint64        `json:"connectedE2Nodes"`
	ActiveSubscriptions     uint64        `json:"activeSubscriptions"`
	IndicationsPerSecond    uint64        `json:"indicationsPerSecond"`
	E2NodeErrors            uint64        `json:"e2NodeErrors"`
	
	// Cache metrics
	CacheHits               uint64        `json:"cacheHits"`
	CacheMisses             uint64        `json:"cacheMisses"`
	CacheHitRatio           float64       `json:"cacheHitRatio"`
	CacheEvictions          uint64        `json:"cacheEvictions"`
	
	// Performance metrics
	CPUUtilization          float64       `json:"cpuUtilization"`
	MemoryUtilizationMB     uint64        `json:"memoryUtilizationMB"`
	GoroutineCount          uint64        `json:"goroutineCount"`
	
	// Error metrics
	HTTPErrors              map[int]uint64 `json:"httpErrors"`
	RateLimitExceeded       uint64        `json:"rateLimitExceeded"`
	AuthenticationFailures  uint64        `json:"authenticationFailures"`
	
	LastUpdated             time.Time     `json:"lastUpdated"`
}

// WebSocketConnectionManager manages WebSocket connections efficiently
type WebSocketConnectionManager struct {
	connections             map[string]*WebSocketConnection
	connectionsByUser       map[string][]*WebSocketConnection
	connectionPool          *WebSocketPool
	
	// Broadcasting
	broadcastChannel        chan *BroadcastMessage
	
	// Performance optimization
	messageQueue            *LockFreeQueue
	compressionEnabled      bool
	
	// Monitoring
	stats                   WSManagerStats
	
	mu                      sync.RWMutex
}

// WebSocketConnection represents an optimized WebSocket connection
type WebSocketConnection struct {
	id                      string
	userID                  string
	conn                    *websocket.Conn
	send                    chan []byte
	receive                 chan []byte
	
	// Performance tracking
	messagesReceived        uint64
	messagesSent            uint64
	bytesReceived           uint64
	bytesSent               uint64
	lastActivity            time.Time
	
	// Connection state
	authenticated           bool
	subscriptions           map[string]bool
	
	mu                      sync.RWMutex
}

// APILoadBalancer handles load balancing for API requests
type APILoadBalancer struct {
	strategy                LoadBalancingStrategy
	backends                []*APIBackend
	healthChecker           *BackendHealthChecker
	
	// Request routing
	router                  *RequestRouter
	stickySession           *StickySessionManager
	
	// Performance tracking
	metrics                 LoadBalancerMetrics
	
	mu                      sync.RWMutex
}

// APIBackend represents an API backend instance
type APIBackend struct {
	id                      string
	address                 string
	port                    int
	weight                  int
	
	// Health and performance
	healthy                 bool
	responseTime            time.Duration
	activeConnections       int64
	requestCount            uint64
	errorCount              uint64
	
	// CPU and memory usage
	cpuUsage                float64
	memoryUsage             float64
	
	lastHealthCheck         time.Time
	mu                      sync.RWMutex
}

// HorizontalScaler type is now defined in types.go to avoid redeclaration

// ProductionE2NodeManager handles 100+ concurrent E2 nodes
type ProductionE2NodeManager struct {
	nodes                   *ShardedE2NodeMap
	connectionManager       *E2ConnectionManager
	
	// Performance optimization
	nodeRouter              *E2NodeRouter
	loadDistributor         *E2LoadDistributor
	
	// Health monitoring
	healthMonitor           *E2HealthMonitor
	alertManager            *E2AlertManager
	
	// Metrics
	stats                   E2NodeManagerStats
	
	mu                      sync.RWMutex
}

// ShardedE2NodeMap provides high-performance E2 node storage
type ShardedE2NodeMap struct {
	shards                  []*E2NodeMapShard
	shardCount              int
	shardMask               uint32
	
	// Hash function for sharding
	hasher                  func(string) uint32
	
	// Global statistics
	totalNodes              int64
	
	mu                      sync.RWMutex
}

// RealTimeIndicationProcessor handles high-speed indication processing
type RealTimeIndicationProcessor struct {
	processingPipeline      *IndicationPipeline
	batchProcessor          *IndicationBatchProcessor
	
	// Performance optimization
	simdProcessor           *IndicationSIMDProcessor
	compressionHandler      *IndicationCompressionHandler
	
	// Routing and distribution  
	routingEngine           *IndicationRoutingEngine
	subscriptionMatcher     *FastSubscriptionMatcher
	
	// Monitoring
	stats                   IndicationProcessorStats
	
	mu                      sync.RWMutex
}

// NewProductionDashboardAPI creates a new production dashboard API
func NewProductionDashboardAPI(config *ProductionAPIConfig) *ProductionDashboardAPI {
	if config == nil {
		config = getDefaultProductionAPIConfig()
	}

	api := &ProductionDashboardAPI{
		config:      config,
		router:      mux.NewRouter(),
		stats:       ProductionAPIStats{HTTPErrors: make(map[int]uint64)},
	}

	// Initialize performance optimizer
	perfConfig := &AdvancedPerformanceConfig{
		MaxConcurrentE2Nodes:     config.MaxConcurrentE2Nodes,
		DashboardConcurrentUsers: config.MaxConcurrentUsers,
		TargetThroughputIPS:      config.MaxRequestsPerSecond,
		MaxProcessingLatencyMs:   config.TargetAPILatencyMs,
	}
	api.performanceOptimizer = NewAdvancedSMOPerformanceOptimizer(perfConfig)

	// Initialize connection cluster
	connConfig := &ConnectionClusterConfig{
		HTTPPoolSize:      config.MaxConnections,
		WebSocketPoolSize: config.MaxConcurrentUsers * 2,
	}
	api.connectionCluster = NewConnectionPoolCluster(connConfig)

	// Initialize WebSocket connection manager
	api.wsConnectionManager = NewWebSocketConnectionManager(config)

	// Initialize load balancer
	if config.EnableLoadBalancing {
		api.loadBalancer = NewAPILoadBalancer(config.LoadBalancingStrategy)
	}

	// Initialize horizontal scaler
	if config.EnableAutoScaling {
		api.horizontalScaler = NewHorizontalScaler(config)
	}

	// Initialize rate limiter
	api.rateLimiter = NewAdaptiveRateLimiter(config.GlobalRateLimit, config.BurstSize)

	// Initialize caching
	if config.EnableResponseCache {
		api.responseCache = NewResponseCache(config.MaxCacheSize, config.CacheTTL)
	}

	// Initialize compression
	if config.EnableCompression {
		api.compressionHandler = NewCompressionHandler(config.CompressionLevel)
	}

	// Initialize E2 node manager
	api.e2NodeManager = NewProductionE2NodeManager(config.MaxConcurrentE2Nodes)

	// Initialize subscription manager
	api.subscriptionManager = NewHighPerformanceSubscriptionManager()

	// Initialize indication processor
	api.indicationProcessor = NewRealTimeIndicationProcessor()

	// Initialize monitoring
	if config.EnableMetrics {
		api.metricsRegistry = prometheus.NewRegistry()
		api.dashboardMetrics = NewDashboardMetrics(api.metricsRegistry)
	}

	api.performanceMonitor = NewAPIPerformanceMonitor()
	api.healthChecker = NewAPIHealthChecker()

	// Setup WebSocket upgrader
	api.wsUpgrader = websocket.Upgrader{
		ReadBufferSize:    config.WSReadBufferSize,
		WriteBufferSize:   config.WSWriteBufferSize,
		EnableCompression: config.WSCompressionEnabled,
		CheckOrigin: func(r *http.Request) bool {
			if config.EnableCORS {
				return api.validateOrigin(r.Header.Get("Origin"))
			}
			return true
		},
	}

	// Setup authentication
	if config.EnableAuthentication {
		api.authManager = NewAuthenticationManager(config.JWTSecretKey)
	}

	// Setup rate limiting
	api.rateLimitManager = NewRateLimitManager(config)

	// Setup routes
	api.setupRoutes()

	return api
}

// Start starts the production dashboard API
func (api *ProductionDashboardAPI) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&api.running, 0, 1) {
		return fmt.Errorf("dashboard API already running")
	}

	logrus.WithFields(logrus.Fields{
		"httpPort":        api.config.HTTPPort,
		"httpsPort":       api.config.HTTPSPort,
		"maxE2Nodes":      api.config.MaxConcurrentE2Nodes,
		"maxUsers":        api.config.MaxConcurrentUsers,
		"maxRPS":          api.config.MaxRequestsPerSecond,
		"targetLatencyMs": api.config.TargetAPILatencyMs,
	}).Info("Starting Production Dashboard API")

	// Start performance optimizer
	if err := api.performanceOptimizer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start performance optimizer: %w", err)
	}

	// Start connection cluster
	if err := api.connectionCluster.Start(ctx); err != nil {
		return fmt.Errorf("failed to start connection cluster: %w", err)
	}

	// Start WebSocket connection manager
	if err := api.wsConnectionManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start WebSocket manager: %w", err)
	}

	// Start load balancer
	if api.loadBalancer != nil {
		if err := api.loadBalancer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start load balancer: %w", err)
		}
	}

	// Start horizontal scaler
	if api.horizontalScaler != nil {
		if err := api.horizontalScaler.Start(ctx); err != nil {
			return fmt.Errorf("failed to start horizontal scaler: %w", err)
		}
	}

	// Start E2 components
	if err := api.e2NodeManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start E2 node manager: %w", err)
	}

	if err := api.subscriptionManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start subscription manager: %w", err)
	}

	if err := api.indicationProcessor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start indication processor: %w", err)
	}

	// Start HTTP servers
	if err := api.startHTTPServers(); err != nil {
		return fmt.Errorf("failed to start HTTP servers: %w", err)
	}

	// Start monitoring and optimization goroutines
	go api.performanceMonitoringLoop(ctx)
	go api.metricsCollectionLoop(ctx)
	go api.healthCheckLoop(ctx)
	go api.autoOptimizationLoop(ctx)

	logrus.Info("Production Dashboard API started successfully")
	return nil
}

// setupRoutes configures all API routes with middleware
func (api *ProductionDashboardAPI) setupRoutes() {
	// Apply global middleware
	api.router.Use(api.loggingMiddleware)
	api.router.Use(api.metricsMiddleware)
	api.router.Use(api.compressionMiddleware)
	api.router.Use(api.rateLimitMiddleware)
	api.router.Use(api.authenticationMiddleware)
	api.router.Use(api.corsMiddleware)

	// Health and monitoring endpoints
	api.router.HandleFunc("/health", api.healthHandler).Methods("GET")
	api.router.HandleFunc("/ready", api.readinessHandler).Methods("GET")
	api.router.HandleFunc("/metrics", promhttp.HandlerFor(api.metricsRegistry, promhttp.HandlerOpts{}).ServeHTTP).Methods("GET")

	// API versioning
	v1 := api.router.PathPrefix("/api/v1").Subrouter()

	// E2 Node management endpoints
	v1.HandleFunc("/e2nodes", api.getE2NodesHandler).Methods("GET")
	v1.HandleFunc("/e2nodes/{nodeId}", api.getE2NodeHandler).Methods("GET")
	v1.HandleFunc("/e2nodes/{nodeId}/connect", api.connectE2NodeHandler).Methods("POST")
	v1.HandleFunc("/e2nodes/{nodeId}/disconnect", api.disconnectE2NodeHandler).Methods("POST")
	v1.HandleFunc("/e2nodes/{nodeId}/status", api.getE2NodeStatusHandler).Methods("GET")

	// Subscription management endpoints
	v1.HandleFunc("/subscriptions", api.getSubscriptionsHandler).Methods("GET")
	v1.HandleFunc("/subscriptions", api.createSubscriptionHandler).Methods("POST")
	v1.HandleFunc("/subscriptions/{subscriptionId}", api.getSubscriptionHandler).Methods("GET")
	v1.HandleFunc("/subscriptions/{subscriptionId}", api.updateSubscriptionHandler).Methods("PUT")
	v1.HandleFunc("/subscriptions/{subscriptionId}", api.deleteSubscriptionHandler).Methods("DELETE")

	// RIC Control endpoints
	v1.HandleFunc("/control", api.ricControlHandler).Methods("POST")
	v1.HandleFunc("/control/{controlId}/status", api.getControlStatusHandler).Methods("GET")

	// Performance and metrics endpoints
	v1.HandleFunc("/performance/metrics", api.getPerformanceMetricsHandler).Methods("GET")
	v1.HandleFunc("/performance/optimize", api.optimizePerformanceHandler).Methods("POST")
	v1.HandleFunc("/performance/benchmark", api.runBenchmarkHandler).Methods("POST")

	// SMO integration endpoints
	v1.HandleFunc("/smo/policies", api.getSMOPoliciesHandler).Methods("GET")
	v1.HandleFunc("/smo/policies", api.createSMOPolicyHandler).Methods("POST")
	v1.HandleFunc("/smo/rapps", api.getSMORAppsHandler).Methods("GET")
	v1.HandleFunc("/smo/rapps/{rAppId}/deploy", api.deploySMORAppHandler).Methods("POST")

	// Nephio R5 integration endpoints
	v1.HandleFunc("/nephio/packages", api.getNephioPackagesHandler).Methods("GET")
	v1.HandleFunc("/nephio/packages/{packageId}/deploy", api.deployNephioPackageHandler).Methods("POST")
	v1.HandleFunc("/nephio/ocloud/resources", api.getOCloudResourcesHandler).Methods("GET")

	// Real-time WebSocket endpoint
	v1.HandleFunc("/ws", api.websocketHandler)

	// Static file serving (for dashboard UI)
	api.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./ui/build/")))
}

// HTTP endpoint handlers

func (api *ProductionDashboardAPI) getE2NodesHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		api.updateLatencyMetrics("get_e2_nodes", time.Since(startTime))
	}()

	// Check cache first
	if api.responseCache != nil {
		if cached := api.responseCache.Get("e2_nodes"); cached != nil {
			api.writeJSONResponse(w, http.StatusOK, cached)
			atomic.AddUint64(&api.stats.CacheHits, 1)
			return
		}
	}

	nodes := api.e2NodeManager.GetAllNodes()
	
	// Cache the response
	if api.responseCache != nil {
		api.responseCache.Set("e2_nodes", nodes, api.config.CacheTTL)
	}

	api.writeJSONResponse(w, http.StatusOK, nodes)
	atomic.AddUint64(&api.stats.CacheMisses, 1)
}

func (api *ProductionDashboardAPI) connectE2NodeHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		api.updateLatencyMetrics("connect_e2_node", time.Since(startTime))
	}()

	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	var connectRequest E2NodeConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&connectRequest); err != nil {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Connect E2 node with performance optimization
	result, err := api.e2NodeManager.ConnectNode(nodeID, &connectRequest)
	if err != nil {
		api.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		atomic.AddUint64(&api.stats.E2NodeErrors, 1)
		return
	}

	api.writeJSONResponse(w, http.StatusOK, result)
}

func (api *ProductionDashboardAPI) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket connection
	conn, err := api.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to upgrade WebSocket connection")
		atomic.AddUint64(&api.stats.WSErrors, 1)
		return
	}

	// Get user ID from authentication context
	userID := api.getUserIDFromContext(r.Context())
	if userID == "" {
		conn.Close()
		return
	}

	// Create WebSocket connection
	wsConn := &WebSocketConnection{
		id:            generateConnectionID(),
		userID:        userID,
		conn:          conn,
		send:          make(chan []byte, 256),
		receive:       make(chan []byte, 256),
		authenticated: true,
		subscriptions: make(map[string]bool),
		lastActivity:  time.Now(),
	}

	// Register connection
	api.wsConnectionManager.RegisterConnection(wsConn)
	atomic.AddInt64(&api.activeConnections, 1)
	atomic.AddUint64(&api.stats.WSConnections, 1)

	// Start connection handlers
	go wsConn.writePump()
	go wsConn.readPump(api.wsConnectionManager)
}

// Middleware functions

func (api *ProductionDashboardAPI) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Increment request counter
		atomic.AddUint64(&api.stats.TotalRequests, 1)
		atomic.AddInt64(&api.concurrentRequests, 1)
		
		// Wrap response writer to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(ww, r)
		
		// Update metrics
		duration := time.Since(start)
		api.updateLatencyMetrics(r.URL.Path, duration)
		
		if ww.statusCode >= 400 {
			atomic.AddUint64(&api.stats.FailedRequests, 1)
			api.stats.HTTPErrors[ww.statusCode]++
		} else {
			atomic.AddUint64(&api.stats.SuccessfulRequests, 1)
		}
		
		atomic.AddInt64(&api.concurrentRequests, -1)
	})
}

func (api *ProductionDashboardAPI) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check global rate limit
		if !api.rateLimiter.Allow() {
			atomic.AddUint64(&api.stats.RateLimitExceeded, 1)
			api.writeErrorResponse(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}
		
		// Check per-user rate limit
		userID := api.getUserIDFromContext(r.Context())
		if userID != "" && !api.rateLimitManager.AllowUser(userID) {
			atomic.AddUint64(&api.stats.RateLimitExceeded, 1)
			api.writeErrorResponse(w, http.StatusTooManyRequests, "User rate limit exceeded")
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (api *ProductionDashboardAPI) compressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !api.config.EnableCompression {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		
		// Apply compression
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		
		gz := gzip.NewWriter(w)
		defer gz.Close()
		
		gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

// Performance monitoring and optimization

func (api *ProductionDashboardAPI) performanceMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			api.updatePerformanceMetrics()
			api.checkPerformanceThresholds()
		}
	}
}

func (api *ProductionDashboardAPI) updatePerformanceMetrics() {
	// Update request rate
	currentTime := time.Now()
	if !api.stats.LastUpdated.IsZero() {
		timeDelta := currentTime.Sub(api.stats.LastUpdated).Seconds()
		if timeDelta > 0 {
			totalRequests := atomic.LoadUint64(&api.stats.TotalRequests)
			api.stats.RequestsPerSecond = uint64(float64(totalRequests) / timeDelta)
		}
	}
	
	// Update connection metrics
	api.stats.ActiveConnections = uint64(atomic.LoadInt64(&api.activeConnections))
	api.stats.ConcurrentUsers = api.wsConnectionManager.GetActiveUserCount()
	
	// Update E2 node metrics
	api.stats.ConnectedE2Nodes = uint64(api.e2NodeManager.GetConnectedNodeCount())
	api.stats.ActiveSubscriptions = uint64(api.subscriptionManager.GetActiveSubscriptionCount())
	
	// Update resource utilization
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	api.stats.MemoryUtilizationMB = m.Alloc / 1024 / 1024
	api.stats.GoroutineCount = uint64(runtime.NumGoroutine())
	
	// Update cache metrics
	if api.responseCache != nil {
		cacheStats := api.responseCache.GetStats()
		api.stats.CacheHits = cacheStats.Hits
		api.stats.CacheMisses = cacheStats.Misses
		if cacheStats.Hits+cacheStats.Misses > 0 {
			api.stats.CacheHitRatio = float64(cacheStats.Hits) / float64(cacheStats.Hits+cacheStats.Misses)
		}
	}
	
	api.stats.LastUpdated = currentTime
}

func (api *ProductionDashboardAPI) checkPerformanceThresholds() {
	// Check latency threshold
	if api.stats.P99LatencyMs > float64(api.config.TargetAPILatencyMs) {
		logrus.WithFields(logrus.Fields{
			"p99LatencyMs": api.stats.P99LatencyMs,
			"threshold":    api.config.TargetAPILatencyMs,
		}).Warn("API latency threshold exceeded")
		
		go api.optimizeLatency()
	}
	
	// Check request rate threshold
	if api.stats.RequestsPerSecond > uint64(api.config.MaxRequestsPerSecond)*8/10 {
		logrus.WithFields(logrus.Fields{
			"currentRPS": api.stats.RequestsPerSecond,
			"threshold":  api.config.MaxRequestsPerSecond,
		}).Info("High request rate detected - scaling up")
		
		go api.scaleUp()
	}
	
	// Check connection threshold
	if api.stats.ActiveConnections > uint64(api.config.MaxConnections)*9/10 {
		logrus.WithFields(logrus.Fields{
			"activeConnections": api.stats.ActiveConnections,
			"maxConnections":    api.config.MaxConnections,
		}).Warn("Connection limit approaching - scaling up")
		
		go api.scaleUp()
	}
}

func (api *ProductionDashboardAPI) optimizeLatency() {
	// Optimize for lower latency
	if api.performanceOptimizer != nil {
		api.performanceOptimizer.OptimizeForThroughput(api.config.MaxRequestsPerSecond)
	}
}

func (api *ProductionDashboardAPI) scaleUp() {
	// Trigger horizontal scaling
	if api.horizontalScaler != nil {
		api.horizontalScaler.ScaleUp()
	}
}

// Utility functions

func (api *ProductionDashboardAPI) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logrus.WithError(err).Error("Failed to encode JSON response")
	}
}

func (api *ProductionDashboardAPI) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	api.writeJSONResponse(w, statusCode, map[string]string{"error": message})
}

func (api *ProductionDashboardAPI) updateLatencyMetrics(endpoint string, duration time.Duration) {
	latencyMs := float64(duration.Nanoseconds()) / 1e6
	
	// Update average latency (simple moving average)
	currentAvg := api.stats.AverageLatencyMs
	if currentAvg == 0 {
		api.stats.AverageLatencyMs = latencyMs
	} else {
		api.stats.AverageLatencyMs = (currentAvg*0.9 + latencyMs*0.1)
	}
	
	// Update max latency
	if latencyMs > api.stats.MaxLatencyMs {
		api.stats.MaxLatencyMs = latencyMs
	}
}

// generateConnectionID is defined in enhanced_dashboard_api.go to avoid redeclaration

func getDefaultProductionAPIConfig() *ProductionAPIConfig {
	return &ProductionAPIConfig{
		HTTPPort:                 8080,
		HTTPSPort:                8443,
		ListenAddress:            "0.0.0.0",
		
		MaxConcurrentE2Nodes:     200,
		MaxConcurrentUsers:       500,
		MaxRequestsPerSecond:     10000,
		TargetAPILatencyMs:       50,
		
		MaxConnections:           5000,
		ConnectionTimeout:        time.Second * 30,
		KeepAliveTimeout:         time.Second * 60,
		ReadTimeout:              time.Second * 30,
		WriteTimeout:             time.Second * 30,
		IdleTimeout:              time.Minute * 5,
		
		WSReadBufferSize:         4096,
		WSWriteBufferSize:        4096,
		WSMaxMessageSize:         65536,
		WSPingInterval:           time.Second * 30,
		WSPongTimeout:            time.Second * 10,
		WSCompressionEnabled:     true,
		
		GlobalRateLimit:          10000,
		PerUserRateLimit:         100,
		BurstSize:                1000,
		RateLimitWindowSize:      time.Minute,
		
		EnableResponseCache:      true,
		CacheTTL:                 time.Minute * 5,
		MaxCacheSize:             10000,
		CacheCompressionEnabled:  true,
		
		EnableCompression:        true,
		CompressionLevel:         6,
		CompressionMinSize:       1024,
		CompressionTypes:         []string{"application/json", "text/html", "text/css", "text/javascript"},
		
		EnableLoadBalancing:      true,
		LoadBalancingStrategy:    "weighted_round_robin",
		HealthCheckInterval:      time.Second * 30,
		
		EnableAutoScaling:        true,
		ScaleUpThreshold:         0.8,
		ScaleDownThreshold:       0.3,
		MinInstances:             2,
		MaxInstances:             10,
		
		EnableAuthentication:     true,
		TokenExpirationTime:      time.Hour * 24,
		EnableCORS:               true,
		
		EnableMetrics:            true,
		MetricsPath:              "/metrics",
		EnableProfiling:          true,
		
		EnableCircuitBreaker:     true,
		CircuitBreakerThreshold:  10,
		CircuitBreakerTimeout:    time.Second * 30,
	}
}

// Stop gracefully stops the production dashboard API
func (api *ProductionDashboardAPI) Stop() error {
	if !atomic.CompareAndSwapInt32(&api.running, 1, 0) {
		return fmt.Errorf("dashboard API not running")
	}
	
	logrus.Info("Stopping Production Dashboard API")
	
	// Stop HTTP servers
	if api.httpServer != nil {
		api.httpServer.Shutdown(context.Background())
	}
	if api.httpsServer != nil {
		api.httpsServer.Shutdown(context.Background())
	}
	
	// Stop other components
	if api.performanceOptimizer != nil {
		api.performanceOptimizer.Stop()
	}
	
	if api.connectionCluster != nil {
		api.connectionCluster.Stop()
	}
	
	logrus.Info("Production Dashboard API stopped successfully")
	return nil
}

// Response writer wrapper for compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Response writer wrapper for metrics
// responseWriter is now using the ResponseWriterWrapper from types.go to avoid redeclaration

// WriteHeader method is now defined in types.go with ResponseWriterWrapper