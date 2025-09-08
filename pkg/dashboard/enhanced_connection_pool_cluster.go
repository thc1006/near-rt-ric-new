/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// ConnectionPoolCluster manages multiple connection pools with advanced optimization
type ConnectionPoolCluster struct {
	// Connection pools by type
	e2Pools         map[string]*EnhancedE2ConnectionPool
	httpPools       map[string]*HTTPConnectionPool
	webSocketPools  map[string]*WebSocketConnectionPool
	sctpPools       map[string]*SCTPConnectionPool
	
	// CPU affinity and thread management
	cpuController   *AdvancedCPUController
	threadScheduler *ThreadScheduler
	
	// Memory optimization
	memoryManager   *HugePagesMemoryManager
	bufferPool      *ZeroCopyBufferPool
	
	// Load balancing and health
	loadBalancer    *ConnectionLoadBalancer
	healthMonitor   *ConnectionHealthMonitor
	
	// Performance monitoring
	metrics         ConnectionClusterMetrics
	profiler        *ConnectionProfiler
	
	// Configuration
	config          *ConnectionClusterConfig
	
	// State management
	running         int32
	totalPools      int32
	activeConnections int64
	mu              sync.RWMutex
}

// ConnectionClusterConfig defines advanced connection pool configuration
type ConnectionClusterConfig struct {
	// Pool sizing
	E2PoolSize              int           `json:"e2PoolSize"`
	HTTPPoolSize            int           `json:"httpPoolSize"`
	WebSocketPoolSize       int           `json:"webSocketPoolSize"`
	SCTPPoolSize            int           `json:"sctpPoolSize"`
	
	// Connection parameters
	MaxConnectionsPerPool   int           `json:"maxConnectionsPerPool"`
	MinIdleConnections      int           `json:"minIdleConnections"`
	MaxIdleConnections      int           `json:"maxIdleConnections"`
	ConnectionTimeout       time.Duration `json:"connectionTimeout"`
	IdleTimeout             time.Duration `json:"idleTimeout"`
	MaxConnectionLifetime   time.Duration `json:"maxConnectionLifetime"`
	
	// Performance optimization
	EnableCPUAffinity       bool          `json:"enableCPUAffinity"`
	EnableHugePages         bool          `json:"enableHugePages"`
	EnableZeroCopy          bool          `json:"enableZeroCopy"`
	BufferSize              int           `json:"bufferSize"`
	ReadBufferSize          int           `json:"readBufferSize"`
	WriteBufferSize         int           `json:"writeBufferSize"`
	
	// TCP optimization
	EnableTCPNoDelay        bool          `json:"enableTCPNoDelay"`
	EnableTCPKeepAlive      bool          `json:"enableTCPKeepAlive"`
	TCPKeepAliveInterval    time.Duration `json:"tcpKeepAliveInterval"`
	EnableTCPFastOpen       bool          `json:"enableTCPFastOpen"`
	SocketRecvBuffer        int           `json:"socketRecvBuffer"`
	SocketSendBuffer        int           `json:"socketSendBuffer"`
	
	// Health checking
	EnableHealthCheck       bool          `json:"enableHealthCheck"`
	HealthCheckInterval     time.Duration `json:"healthCheckInterval"`
	HealthCheckTimeout      time.Duration `json:"healthCheckTimeout"`
	MaxHealthCheckFailures  int           `json:"maxHealthCheckFailures"`
	
	// Load balancing
	LoadBalancingStrategy   string        `json:"loadBalancingStrategy"`
	EnableConnectionReuse   bool          `json:"enableConnectionReuse"`
	ConnectionReuseTimeout  time.Duration `json:"connectionReuseTimeout"`
	
	// Monitoring
	EnableMetrics           bool          `json:"enableMetrics"`
	MetricsInterval         time.Duration `json:"metricsInterval"`
	EnableProfiling         bool          `json:"enableProfiling"`
	ProfilingInterval       time.Duration `json:"profilingInterval"`
}

// EnhancedE2ConnectionPool manages high-performance E2 connections
type EnhancedE2ConnectionPool struct {
	poolID          string
	target          string
	connections     chan *E2Connection
	activeConns     int64
	createdConns    int64
	failedConns     int64
	
	// Performance optimization
	cpuAffinity     int
	memoryPool      *E2MemoryPool
	
	// Connection factory and validation
	factory         E2ConnectionFactory
	validator       E2ConnectionValidator
	
	// Metrics and monitoring
	stats           E2PoolStats
	healthChecker   *E2HealthChecker
	
	mu              sync.RWMutex
	closed          int32
}

// E2Connection represents an optimized E2 connection
type E2Connection struct {
	conn            net.Conn
	sctpConn        *SCTPConnection
	id              string
	poolID          string
	created         time.Time
	lastUsed        time.Time
	useCount        int64
	
	// Buffers for zero-copy operations
	readBuffer      []byte
	writeBuffer     []byte
	messageBuffer   []byte
	
	// Performance counters
	bytesSent       uint64
	bytesReceived   uint64
	messagesProcessed uint64
	
	// State management
	inUse           int32
	healthy         int32
	mu              sync.RWMutex
}

// SCTPConnection handles SCTP-specific optimizations
type SCTPConnection struct {
	conn            *net.SCTPConn
	streams         map[uint16]*SCTPStream
	maxInStreams    uint16
	maxOutStreams   uint16
	
	// SCTP-specific buffers
	sctpBuffer      []byte
	chunks          []*SCTPChunk
	
	stats           SCTPStats
	mu              sync.RWMutex
}

// SCTPStream represents an SCTP stream
type SCTPStream struct {
	id              uint16
	direction       StreamDirection
	sequenceNumber  uint32
	buffer          []byte
	
	stats           StreamStats
	mu              sync.RWMutex
}

// SCTPChunk represents an SCTP chunk for zero-copy processing
type SCTPChunk struct {
	data            unsafe.Pointer
	size            uint32
	streamID        uint16
	sequenceNumber  uint32
	processed       bool
}

// AdvancedCPUController manages CPU affinity for connections
type AdvancedCPUController struct {
	coreCount       int
	coreAssignments map[string]int
	affinityMask    []uint64
	criticalCores   []int
	isolatedCores   []int
	
	// Performance monitoring
	coreUtilization map[int]float64
	loadBalancer    *CPULoadBalancer
	
	mu              sync.RWMutex
}

// ThreadScheduler manages thread scheduling optimization
type ThreadScheduler struct {
	pools           map[string]*WorkerPool
	scheduler       *RealTimeScheduler
	priorityQueues  map[Priority]*PriorityQueue
	
	// CPU optimization
	cpuAffinity     map[string]int
	numaNodes       []NUMANode
	
	stats           SchedulerStats
	mu              sync.RWMutex
}

// WorkerPool manages a pool of worker threads
type WorkerPool struct {
	id              string
	workers         []*Worker
	workQueue       *LockFreeQueue
	resultQueue     *LockFreeQueue
	
	// CPU affinity
	cpuCore         int
	numaNode        int
	
	// Performance tuning
	batchSize       int
	prefetchSize    int
	
	stats           WorkerPoolStats
	running         int32
	mu              sync.RWMutex
}

// Worker represents a high-performance worker thread
type Worker struct {
	id              int
	poolID          string
	thread          *WorkerThread
	
	// CPU affinity
	cpuCore         int
	threadID        int
	
	// Performance optimization
	cache           *WorkerCache
	localQueue      *LocalWorkQueue
	
	stats           WorkerStats
	state           WorkerState
	mu              sync.RWMutex
}

// WorkerThread represents the actual thread implementation
type WorkerThread struct {
	tid             int
	priority        int
	cpuAffinity     uint64
	stackSize       int
	
	// Real-time scheduling
	policy          SchedulingPolicy
	rtPriority      int
	
	context         *ThreadContext
	mu              sync.RWMutex
}

// ThreadContext holds thread-specific context
type ThreadContext struct {
	registers       ThreadRegisters
	stack           []byte
	tlsData         map[string]interface{}
	
	// Performance counters
	instructions    uint64
	cycles          uint64
	cacheMisses     uint64
	
	mu              sync.RWMutex
}

// ConnectionLoadBalancer distributes connections across pools
type ConnectionLoadBalancer struct {
	strategy        LoadBalancingStrategy
	pools           []*ConnectionPool
	weights         map[string]int
	
	// Health awareness
	healthScores    map[string]float64
	
	// Performance tracking
	routingTable    *RoutingTable
	metrics         LoadBalancerMetrics
	
	mu              sync.RWMutex
}

// ConnectionHealthMonitor monitors connection health
type ConnectionHealthMonitor struct {
	healthCheckers  map[string]*HealthChecker
	healthScores    map[string]*HealthScore
	alertManager    *HealthAlertManager
	
	// Monitoring configuration
	interval        time.Duration
	timeout         time.Duration
	retries         int
	
	// Circuit breaker integration
	circuitBreakers map[string]*CircuitBreaker
	
	running         int32
	mu              sync.RWMutex
}

// NewConnectionPoolCluster creates a new connection pool cluster
func NewConnectionPoolCluster(config *ConnectionClusterConfig) *ConnectionPoolCluster {
	if config == nil {
		config = getDefaultConnectionClusterConfig()
	}

	cluster := &ConnectionPoolCluster{
		e2Pools:        make(map[string]*EnhancedE2ConnectionPool),
		httpPools:      make(map[string]*HTTPConnectionPool),
		webSocketPools: make(map[string]*WebSocketConnectionPool),
		sctpPools:      make(map[string]*SCTPConnectionPool),
		config:         config,
		metrics:        ConnectionClusterMetrics{},
	}

	// Initialize CPU controller
	if config.EnableCPUAffinity {
		cluster.cpuController = NewAdvancedCPUController()
	}

	// Initialize thread scheduler
	cluster.threadScheduler = NewThreadScheduler(config)

	// Initialize memory manager
	if config.EnableHugePages {
		cluster.memoryManager = NewHugePagesMemoryManager(&AdvancedPerformanceConfig{
			EnableHugePages:     true,
			HugePageSize:        2048, // 2MB pages
			MemoryPoolSizeMB:    config.SocketRecvBuffer / 1024,
		})
	}

	// Initialize buffer pool
	if config.EnableZeroCopy {
		cluster.bufferPool = NewZeroCopyBufferPool(config.BufferSize, 1000)
	}

	// Initialize load balancer
	cluster.loadBalancer = NewConnectionLoadBalancer(config.LoadBalancingStrategy)

	// Initialize health monitor
	if config.EnableHealthCheck {
		cluster.healthMonitor = NewConnectionHealthMonitor(config)
	}

	// Initialize profiler
	if config.EnableProfiling {
		cluster.profiler = NewConnectionProfiler(config.ProfilingInterval)
	}

	return cluster
}

// Start starts the connection pool cluster
func (cpc *ConnectionPoolCluster) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&cpc.running, 0, 1) {
		return fmt.Errorf("connection pool cluster already running")
	}

	logrus.WithFields(logrus.Fields{
		"e2PoolSize":       cpc.config.E2PoolSize,
		"httpPoolSize":     cpc.config.HTTPPoolSize,
		"webSocketPoolSize": cpc.config.WebSocketPoolSize,
		"sctpPoolSize":     cpc.config.SCTPPoolSize,
		"cpuAffinity":      cpc.config.EnableCPUAffinity,
		"hugePages":        cpc.config.EnableHugePages,
		"zeroCopy":         cpc.config.EnableZeroCopy,
	}).Info("Starting Connection Pool Cluster")

	// Start CPU controller
	if cpc.cpuController != nil {
		if err := cpc.cpuController.Start(ctx); err != nil {
			return fmt.Errorf("failed to start CPU controller: %w", err)
		}
	}

	// Start thread scheduler
	if err := cpc.threadScheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start thread scheduler: %w", err)
	}

	// Start memory manager
	if cpc.memoryManager != nil {
		if err := cpc.memoryManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start memory manager: %w", err)
		}
	}

	// Start load balancer
	if err := cpc.loadBalancer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start load balancer: %w", err)
	}

	// Start health monitor
	if cpc.healthMonitor != nil {
		if err := cpc.healthMonitor.Start(ctx); err != nil {
			return fmt.Errorf("failed to start health monitor: %w", err)
		}
	}

	// Initialize connection pools
	if err := cpc.initializeConnectionPools(); err != nil {
		return fmt.Errorf("failed to initialize connection pools: %w", err)
	}

	// Start monitoring goroutines
	go cpc.metricsCollector(ctx)
	go cpc.poolOptimizer(ctx)
	go cpc.connectionReaper(ctx)

	logrus.Info("Connection Pool Cluster started successfully")
	return nil
}

// CreateE2ConnectionPool creates a new E2 connection pool
func (cpc *ConnectionPoolCluster) CreateE2ConnectionPool(target string, size int) (*EnhancedE2ConnectionPool, error) {
	cpc.mu.Lock()
	defer cpc.mu.Unlock()

	poolID := fmt.Sprintf("e2-%s", target)
	if _, exists := cpc.e2Pools[poolID]; exists {
		return nil, fmt.Errorf("E2 pool for target %s already exists", target)
	}

	pool := &EnhancedE2ConnectionPool{
		poolID:      poolID,
		target:      target,
		connections: make(chan *E2Connection, size),
		factory:     NewE2ConnectionFactory(target),
		validator:   NewE2ConnectionValidator(),
		stats:       E2PoolStats{},
	}

	// Assign CPU affinity if enabled
	if cpc.config.EnableCPUAffinity && cpc.cpuController != nil {
		cpuCore := cpc.cpuController.AssignCore(poolID)
		pool.cpuAffinity = cpuCore
	}

	// Create memory pool for this E2 pool
	pool.memoryPool = NewE2MemoryPool(cpc.config.BufferSize)

	// Pre-populate the pool
	if err := pool.prepopulate(size); err != nil {
		return nil, fmt.Errorf("failed to prepopulate E2 pool: %w", err)
	}

	cpc.e2Pools[poolID] = pool
	atomic.AddInt32(&cpc.totalPools, 1)

	logrus.WithFields(logrus.Fields{
		"poolID":     poolID,
		"target":     target,
		"size":       size,
		"cpuCore":    pool.cpuAffinity,
	}).Info("Created E2 connection pool")

	return pool, nil
}

// GetE2Connection gets a connection from the E2 pool
func (cpc *ConnectionPoolCluster) GetE2Connection(target string) (*E2Connection, error) {
	poolID := fmt.Sprintf("e2-%s", target)
	
	cpc.mu.RLock()
	pool, exists := cpc.e2Pools[poolID]
	cpc.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("no E2 pool found for target %s", target)
	}

	return pool.GetConnection()
}

// ReturnE2Connection returns a connection to the E2 pool
func (cpc *ConnectionPoolCluster) ReturnE2Connection(conn *E2Connection) error {
	cpc.mu.RLock()
	pool, exists := cpc.e2Pools[conn.poolID]
	cpc.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("no E2 pool found for connection %s", conn.id)
	}

	return pool.ReturnConnection(conn)
}

// ScaleToSize scales the connection pools to the specified size
func (cpc *ConnectionPoolCluster) ScaleToSize(totalSize int) error {
	cpc.mu.Lock()
	defer cpc.mu.Unlock()

	logrus.WithField("targetSize", totalSize).Info("Scaling connection pools")

	// Calculate size per pool type
	e2Size := totalSize * 40 / 100      // 40% for E2
	httpSize := totalSize * 30 / 100    // 30% for HTTP
	wsSize := totalSize * 20 / 100      // 20% for WebSocket
	sctpSize := totalSize * 10 / 100    // 10% for SCTP

	// Scale E2 pools
	for _, pool := range cpc.e2Pools {
		if err := pool.ScaleToSize(e2Size); err != nil {
			return fmt.Errorf("failed to scale E2 pool %s: %w", pool.poolID, err)
		}
	}

	// Scale HTTP pools
	for _, pool := range cpc.httpPools {
		if err := pool.ScaleToSize(httpSize); err != nil {
			return fmt.Errorf("failed to scale HTTP pool: %w", err)
		}
	}

	// Scale WebSocket pools
	for _, pool := range cpc.webSocketPools {
		if err := pool.ScaleToSize(wsSize); err != nil {
			return fmt.Errorf("failed to scale WebSocket pool: %w", err)
		}
	}

	// Scale SCTP pools
	for _, pool := range cpc.sctpPools {
		if err := pool.ScaleToSize(sctpSize); err != nil {
			return fmt.Errorf("failed to scale SCTP pool: %w", err)
		}
	}

	return nil
}

// ScaleWebSocketPool scales WebSocket pool specifically
func (cpc *ConnectionPoolCluster) ScaleWebSocketPool(size int) error {
	cpc.mu.Lock()
	defer cpc.mu.Unlock()

	for poolID, pool := range cpc.webSocketPools {
		if err := pool.ScaleToSize(size); err != nil {
			return fmt.Errorf("failed to scale WebSocket pool %s: %w", poolID, err)
		}
	}

	return nil
}

// ScaleUp scales up all connection pools
func (cpc *ConnectionPoolCluster) ScaleUp() error {
	currentSize := cpc.getTotalConnectionCapacity()
	newSize := int(float64(currentSize) * 1.2) // Scale up by 20%
	
	logrus.WithFields(logrus.Fields{
		"currentSize": currentSize,
		"newSize":     newSize,
	}).Info("Scaling up connection pools")

	return cpc.ScaleToSize(newSize)
}

// GetConnectionClusterMetrics returns comprehensive cluster metrics
func (cpc *ConnectionPoolCluster) GetConnectionClusterMetrics() ConnectionClusterMetrics {
	cpc.mu.RLock()
	defer cpc.mu.RUnlock()

	metrics := cpc.metrics

	// Update real-time metrics
	metrics.TotalPools = atomic.LoadInt32(&cpc.totalPools)
	metrics.ActiveConnections = atomic.LoadInt64(&cpc.activeConnections)

	// Aggregate pool metrics
	var totalCapacity int64
	var totalUtilization float64
	poolCount := 0

	for _, pool := range cpc.e2Pools {
		poolMetrics := pool.GetMetrics()
		totalCapacity += int64(poolMetrics.Capacity)
		totalUtilization += poolMetrics.Utilization
		poolCount++
	}

	if poolCount > 0 {
		metrics.AverageUtilization = totalUtilization / float64(poolCount)
		metrics.TotalCapacity = totalCapacity
	}

	metrics.LastUpdated = time.Now()
	return metrics
}

// Private helper methods

func (cpc *ConnectionPoolCluster) initializeConnectionPools() error {
	// Create default E2 pools
	if cpc.config.E2PoolSize > 0 {
		if _, err := cpc.CreateE2ConnectionPool("default", cpc.config.E2PoolSize); err != nil {
			return fmt.Errorf("failed to create default E2 pool: %w", err)
		}
	}

	// Initialize other pool types as needed
	// HTTP, WebSocket, SCTP pools...

	return nil
}

func (cpc *ConnectionPoolCluster) metricsCollector(ctx context.Context) {
	ticker := time.NewTicker(cpc.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpc.collectMetrics()
		}
	}
}

func (cpc *ConnectionPoolCluster) poolOptimizer(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpc.optimizePools()
		}
	}
}

func (cpc *ConnectionPoolCluster) connectionReaper(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpc.reapIdleConnections()
		}
	}
}

func (cpc *ConnectionPoolCluster) collectMetrics() {
	// Collect metrics from all pools
	for _, pool := range cpc.e2Pools {
		pool.updateMetrics()
	}

	// Update cluster-level metrics
	cpc.updateClusterMetrics()
}

func (cpc *ConnectionPoolCluster) optimizePools() {
	// Optimize pool sizes based on utilization
	for _, pool := range cpc.e2Pools {
		metrics := pool.GetMetrics()
		if metrics.Utilization > 0.8 {
			// Scale up if utilization is high
			pool.ScaleUp(0.2)
		} else if metrics.Utilization < 0.3 {
			// Scale down if utilization is low
			pool.ScaleDown(0.1)
		}
	}
}

func (cpc *ConnectionPoolCluster) reapIdleConnections() {
	// Remove idle connections that exceed max lifetime
	for _, pool := range cpc.e2Pools {
		pool.ReapIdleConnections(cpc.config.MaxConnectionLifetime)
	}
}

func (cpc *ConnectionPoolCluster) getTotalConnectionCapacity() int {
	capacity := 0
	for _, pool := range cpc.e2Pools {
		capacity += pool.GetCapacity()
	}
	for _, pool := range cpc.httpPools {
		capacity += pool.GetCapacity()
	}
	for _, pool := range cpc.webSocketPools {
		capacity += pool.GetCapacity()
	}
	return capacity
}

func (cpc *ConnectionPoolCluster) updateClusterMetrics() {
	// Update cluster-wide performance metrics
	cpc.metrics.LastUpdated = time.Now()
}

// Enhanced E2 Connection Pool Implementation

func (pool *EnhancedE2ConnectionPool) GetConnection() (*E2Connection, error) {
	// Try to get an existing connection first
	select {
	case conn := <-pool.connections:
		if pool.validator.ValidateConnection(conn) {
			atomic.StoreInt32(&conn.inUse, 1)
			conn.lastUsed = time.Now()
			atomic.AddInt64(&conn.useCount, 1)
			atomic.AddInt64(&pool.activeConns, 1)
			pool.stats.PoolHits++
			return conn, nil
		}
		// Connection is invalid, close it
		conn.Close()
	default:
		// No available connections
	}

	// Create a new connection
	conn, err := pool.factory.CreateConnection()
	if err != nil {
		atomic.AddInt64(&pool.failedConns, 1)
		pool.stats.PoolMisses++
		return nil, fmt.Errorf("failed to create E2 connection: %w", err)
	}

	conn.poolID = pool.poolID
	atomic.AddInt64(&pool.createdConns, 1)
	atomic.AddInt64(&pool.activeConns, 1)
	atomic.StoreInt32(&conn.inUse, 1)

	// Apply CPU affinity if configured
	if pool.cpuAffinity >= 0 {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		
		// Set CPU affinity for this connection's processing
		if err := setCPUAffinity(pool.cpuAffinity); err != nil {
			logrus.WithError(err).Warn("Failed to set CPU affinity")
		}
	}

	return conn, nil
}

func (pool *EnhancedE2ConnectionPool) ReturnConnection(conn *E2Connection) error {
	if conn == nil {
		return fmt.Errorf("cannot return nil connection")
	}

	atomic.StoreInt32(&conn.inUse, 0)
	conn.lastUsed = time.Now()
	atomic.AddInt64(&pool.activeConns, -1)

	// Validate connection before returning to pool
	if !pool.validator.ValidateConnection(conn) {
		conn.Close()
		return nil
	}

	// Reset connection state
	conn.reset()

	// Return to pool
	select {
	case pool.connections <- conn:
		return nil
	default:
		// Pool is full, close the connection
		conn.Close()
		return nil
	}
}

func (pool *EnhancedE2ConnectionPool) ScaleToSize(newSize int) error {
	currentSize := cap(pool.connections)
	if newSize == currentSize {
		return nil
	}

	if newSize > currentSize {
		// Scale up
		additional := newSize - currentSize
		return pool.addConnections(additional)
	} else {
		// Scale down
		toRemove := currentSize - newSize
		return pool.removeConnections(toRemove)
	}
}

// CPU Affinity and Thread Management

func setCPUAffinity(coreID int) error {
	if coreID < 0 {
		return nil
	}

	var cpuset unix.CPUSet
	cpuset.Zero()
	cpuset.Set(coreID)

	return unix.SchedSetaffinity(0, &cpuset)
}

func (ctrl *AdvancedCPUController) AssignCore(poolID string) int {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	// Find the least loaded core
	minLoad := 1.0
	selectedCore := 0

	for core, load := range ctrl.coreUtilization {
		if load < minLoad {
			minLoad = load
			selectedCore = core
		}
	}

	ctrl.coreAssignments[poolID] = selectedCore
	return selectedCore
}

// Default configuration

func getDefaultConnectionClusterConfig() *ConnectionClusterConfig {
	return &ConnectionClusterConfig{
		E2PoolSize:              1000,
		HTTPPoolSize:            500,
		WebSocketPoolSize:       200,
		SCTPPoolSize:            100,
		
		MaxConnectionsPerPool:   2000,
		MinIdleConnections:      10,
		MaxIdleConnections:      100,
		ConnectionTimeout:       time.Second * 30,
		IdleTimeout:             time.Minute * 5,
		MaxConnectionLifetime:   time.Hour,
		
		EnableCPUAffinity:       true,
		EnableHugePages:         true,
		EnableZeroCopy:          true,
		BufferSize:              65536,
		ReadBufferSize:          32768,
		WriteBufferSize:         32768,
		
		EnableTCPNoDelay:        true,
		EnableTCPKeepAlive:      true,
		TCPKeepAliveInterval:    time.Second * 30,
		SocketRecvBuffer:        1048576,
		SocketSendBuffer:        1048576,
		
		EnableHealthCheck:       true,
		HealthCheckInterval:     time.Second * 30,
		HealthCheckTimeout:      time.Second * 5,
		MaxHealthCheckFailures:  3,
		
		LoadBalancingStrategy:   "weighted_round_robin",
		EnableConnectionReuse:   true,
		ConnectionReuseTimeout:  time.Second * 30,
		
		EnableMetrics:           true,
		MetricsInterval:         time.Second * 5,
		EnableProfiling:         true,
		ProfilingInterval:       time.Second * 10,
	}
}

// Stop gracefully stops the connection pool cluster
func (cpc *ConnectionPoolCluster) Stop() error {
	if !atomic.CompareAndSwapInt32(&cpc.running, 1, 0) {
		return fmt.Errorf("connection pool cluster not running")
	}

	logrus.Info("Stopping Connection Pool Cluster")

	// Close all connection pools
	for _, pool := range cpc.e2Pools {
		pool.Close()
	}

	// Stop other components
	if cpc.healthMonitor != nil {
		cpc.healthMonitor.Stop()
	}

	if cpc.cpuController != nil {
		cpc.cpuController.Stop()
	}

	logrus.Info("Connection Pool Cluster stopped successfully")
	return nil
}