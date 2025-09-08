/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// ConnectionPoolConfig contains configuration for connection pooling
type ConnectionPoolConfig struct {
	MaxIdle           int           `json:"maxIdle"`
	MaxActive         int           `json:"maxActive"`
	IdleTimeout       time.Duration `json:"idleTimeout"`
	MaxLifetime       time.Duration `json:"maxLifetime"`
	HealthCheckInterval time.Duration `json:"healthCheckInterval"`
	EnableMetrics     bool          `json:"enableMetrics"`
	EnableHealthCheck bool          `json:"enableHealthCheck"`
}

// ConnectionPoolMetrics tracks connection pool performance
type ConnectionPoolMetrics struct {
	mu                sync.RWMutex
	ActiveConnections int64         `json:"activeConnections"`
	PooledConnections int64         `json:"pooledConnections"`
	ConnectionFailures int64        `json:"connectionFailures"`
	PoolHits          int64         `json:"poolHits"`
	PoolMisses        int64         `json:"poolMisses"`
	HealthCheckFails  int64         `json:"healthCheckFails"`
	CreatedCount      int64         `json:"createdCount"`
	DestroyedCount    int64         `json:"destroyedCount"`
	LastUpdated       time.Time     `json:"lastUpdated"`
}

// PoolHealthStatus represents the health status of a connection pool
type PoolHealthStatus int

const (
	PoolHealthy PoolHealthStatus = iota
	PoolDegraded
	PoolUnhealthy
)

// String returns the string representation of pool health status
func (phs PoolHealthStatus) String() string {
	switch phs {
	case PoolHealthy:
		return "HEALTHY"
	case PoolDegraded:
		return "DEGRADED"
	case PoolUnhealthy:
		return "UNHEALTHY"
	default:
		return "UNKNOWN"
	}
}

// PoolMetrics tracks individual pool metrics
type PoolMetrics struct {
	mu            sync.RWMutex
	Requests      int64         `json:"requests"`
	Hits          int64         `json:"hits"`
	Misses        int64         `json:"misses"`
	Timeouts      int64         `json:"timeouts"`
	Errors        int64         `json:"errors"`
	AvgWaitTime   time.Duration `json:"avgWaitTime"`
	MaxWaitTime   time.Duration `json:"maxWaitTime"`
	LastActivity  time.Time     `json:"lastActivity"`
}

// PoolHealthChecker performs health checks on connection pools
type PoolHealthChecker struct {
	mu           sync.RWMutex
	pools        map[string]*EnhancedConnectionPool
	interval     time.Duration
	timeout      time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	isRunning    bool
}


// NewConnectionPoolManager creates a new connection pool manager
func NewConnectionPoolManager(config ConnectionPoolConfig) *ConnectionPoolManager {
	return &ConnectionPoolManager{
		pools:   make(map[string]*EnhancedConnectionPool),
		config:  config,
		metrics: &ConnectionPoolMetrics{LastUpdated: time.Now()},
		resourceMonitor: NewResourceMonitor(),
		healthChecker:   NewPoolHealthChecker(30 * time.Second),
	}
}

// Start starts the connection pool manager
func (cpm *ConnectionPoolManager) Start() error {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()
	
	// Start resource monitor
	if err := cpm.resourceMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start resource monitor: %w", err)
	}
	
	// Start health checker
	if cpm.config.EnableHealthCheck {
		if err := cpm.healthChecker.Start(); err != nil {
			return fmt.Errorf("failed to start health checker: %w", err)
		}
	}
	
	log.Println("Connection pool manager started")
	return nil
}

// Stop stops the connection pool manager
func (cpm *ConnectionPoolManager) Stop() error {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()
	
	// Stop health checker
	if cpm.healthChecker != nil {
		cpm.healthChecker.Stop()
	}
	
	// Stop resource monitor
	if cpm.resourceMonitor != nil {
		cpm.resourceMonitor.Stop()
	}
	
	// Close all pools
	for name, pool := range cpm.pools {
		if err := pool.Close(); err != nil {
			log.Printf("Error closing pool %s: %v", name, err)
		}
	}
	
	log.Println("Connection pool manager stopped")
	return nil
}

// GetConnection gets a connection from the specified pool
func (cpm *ConnectionPoolManager) GetConnection(ctx context.Context, poolName, address string) (*PooledConnection, error) {
	cpm.mu.RLock()
	pool, exists := cpm.pools[poolName]
	cpm.mu.RUnlock()
	
	if !exists {
		// Create new pool
		pool = cpm.createPool(poolName, address)
		cpm.mu.Lock()
		cpm.pools[poolName] = pool
		cpm.mu.Unlock()
		
		// Register with health checker
		if cpm.healthChecker != nil {
			cpm.healthChecker.AddPool(poolName, pool)
		}
	}
	
	// Check resource constraints
	if cpm.resourceMonitor != nil {
		usage := cpm.resourceMonitor.GetResourceUsage()
		if usage.MemoryUsage > 90.0 || usage.CPUUsage > 95.0 {
			return nil, fmt.Errorf("resource constraints exceeded: CPU=%.1f%%, Memory=%.1f%%", 
				usage.CPUUsage, usage.MemoryUsage)
		}
	}
	
	// Get connection from pool
	conn, err := pool.GetConnection(ctx, address)
	if err != nil {
		cpm.updateMetrics(false, false)
		return nil, err
	}
	
	cpm.updateMetrics(true, conn.id != "")
	return conn, nil
}

// ReturnConnection returns a connection to the pool
func (cpm *ConnectionPoolManager) ReturnConnection(poolName string, conn *PooledConnection) error {
	cpm.mu.RLock()
	pool, exists := cpm.pools[poolName]
	cpm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("pool %s not found", poolName)
	}
	
	return pool.ReturnConnection(conn)
}

// GetMetrics returns connection pool metrics
func (cpm *ConnectionPoolManager) GetMetrics() *ConnectionPoolMetrics {
	cpm.metrics.mu.RLock()
	defer cpm.metrics.mu.RUnlock()
	
	// Update active connections
	totalActive := int64(0)
	totalPooled := int64(0)
	
	cpm.mu.RLock()
	for _, pool := range cpm.pools {
		poolMetrics := pool.GetMetrics()
		totalActive += int64(len(pool.activeConnections))
		totalPooled += int64(len(pool.connections))
	}
	cpm.mu.RUnlock()
	
	metrics := &ConnectionPoolMetrics{
		ActiveConnections: totalActive,
		PooledConnections: totalPooled,
		ConnectionFailures: cpm.metrics.ConnectionFailures,
		PoolHits:          cpm.metrics.PoolHits,
		PoolMisses:        cpm.metrics.PoolMisses,
		HealthCheckFails:  cpm.metrics.HealthCheckFails,
		CreatedCount:      cpm.metrics.CreatedCount,
		DestroyedCount:    cpm.metrics.DestroyedCount,
		LastUpdated:      time.Now(),
	}
	
	return metrics
}

// createPool creates a new enhanced connection pool
func (cpm *ConnectionPoolManager) createPool(poolName, address string) *EnhancedConnectionPool {
	pool := &EnhancedConnectionPool{
		connections:      make(chan *PooledConnection, cpm.config.MaxIdle),
		activeConnections: make(map[string]*PooledConnection),
		config:           cpm.config,
		healthStatus:     PoolHealthy,
		lastHealthCheck:  time.Now(),
		metrics:          &PoolMetrics{LastActivity: time.Now()},
	}
	
	// Set appropriate factory based on address/protocol
	pool.factory = cpm.createConnectionFactory(address)
	
	log.Printf("Created new connection pool: %s for address: %s", poolName, address)
	return pool
}

// createConnectionFactory creates a connection factory for the given address
func (cpm *ConnectionPoolManager) createConnectionFactory(address string) ConnectionFactory {
	return func(addr string) (interface{}, error) {
		// For production, this would create appropriate connection types
		// (gRPC, HTTP, TCP, etc.) based on the address format
		
		conn, err := net.DialTimeout("tcp", addr, cpm.config.IdleTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection to %s: %w", addr, err)
		}
		
		log.Printf("Created new connection to %s", addr)
		return conn, nil
	}
}

// updateMetrics updates connection pool metrics
func (cpm *ConnectionPoolManager) updateMetrics(success, fromPool bool) {
	cpm.metrics.mu.Lock()
	defer cpm.metrics.mu.Unlock()
	
	if success {
		if fromPool {
			cpm.metrics.PoolHits++
		} else {
			cpm.metrics.PoolMisses++
			cpm.metrics.CreatedCount++
		}
	} else {
		cpm.metrics.ConnectionFailures++
	}
	
	cpm.metrics.LastUpdated = time.Now()
}

// EnhancedConnectionPool implementation

// GetConnection gets a connection from the pool with production-grade features
func (ecp *EnhancedConnectionPool) GetConnection(ctx context.Context, address string) (*PooledConnection, error) {
	startTime := time.Now()
	
	// Update metrics
	ecp.metrics.mu.Lock()
	ecp.metrics.Requests++
	ecp.metrics.LastActivity = time.Now()
	ecp.metrics.mu.Unlock()
	
	// Try to get from pool first
	select {
	case conn := <-ecp.connections:
		ecp.metrics.mu.Lock()
		ecp.metrics.Hits++
		ecp.metrics.mu.Unlock()
		
		// Check if connection is still healthy and not expired
		if ecp.isConnectionValid(conn) {
			conn.lastUsed = time.Now()
			conn.useCount++
			
			// Add to active connections
			ecp.mu.Lock()
			ecp.activeConnections[conn.id] = conn
			ecp.mu.Unlock()
			
			// Add correlation ID from context
			conn.correlationID = GetCorrelationID(ctx)
			
			return conn, nil
		}
		
		// Connection is invalid, destroy it
		ecp.destroyConnection(conn)
		
	case <-time.After(5 * time.Second):
		// Timeout waiting for pooled connection
		ecp.metrics.mu.Lock()
		ecp.metrics.Timeouts++
		waitTime := time.Since(startTime)
		if waitTime > ecp.metrics.MaxWaitTime {
			ecp.metrics.MaxWaitTime = waitTime
		}
		ecp.metrics.mu.Unlock()
		
	default:
		// Pool is empty, create new connection
	}
	
	// Check if we can create new connection
	ecp.mu.RLock()
	activeCount := len(ecp.activeConnections)
	ecp.mu.RUnlock()
	
	if activeCount >= ecp.config.MaxActive {
		return nil, fmt.Errorf("connection pool exhausted: %d active connections", activeCount)
	}
	
	// Create new connection
	rawConn, err := ecp.factory(address)
	if err != nil {
		ecp.metrics.mu.Lock()
		ecp.metrics.Errors++
		ecp.metrics.mu.Unlock()
		
		atomic.AddInt64(&ecp.errorCount, 1)
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}
	
	conn := &PooledConnection{
		conn:          rawConn,
		id:            uuid.New().String(),
		createdAt:     time.Now(),
		lastUsed:      time.Now(),
		useCount:      1,
		isHealthy:     true,
		metadata:      make(map[string]interface{}),
		correlationID: GetCorrelationID(ctx),
	}
	
	// Add to active connections
	ecp.mu.Lock()
	ecp.activeConnections[conn.id] = conn
	ecp.mu.Unlock()
	
	atomic.AddInt64(&ecp.createdCount, 1)
	
	ecp.metrics.mu.Lock()
	ecp.metrics.Misses++
	ecp.metrics.mu.Unlock()
	
	log.Printf("Created new pooled connection %s for address %s", conn.id, address)
	return conn, nil
}

// ReturnConnection returns a connection to the pool
func (ecp *EnhancedConnectionPool) ReturnConnection(conn *PooledConnection) error {
	if conn == nil {
		return fmt.Errorf("cannot return nil connection")
	}
	
	// Remove from active connections
	ecp.mu.Lock()
	delete(ecp.activeConnections, conn.id)
	ecp.mu.Unlock()
	
	// Check if connection is still healthy
	if !ecp.isConnectionValid(conn) {
		ecp.destroyConnection(conn)
		return nil
	}
	
	// Try to return to pool
	select {
	case ecp.connections <- conn:
		log.Printf("Returned connection %s to pool", conn.id)
		return nil
		
	default:
		// Pool is full, destroy connection
		ecp.destroyConnection(conn)
		return nil
	}
}

// Close closes the connection pool
func (ecp *EnhancedConnectionPool) Close() error {
	ecp.mu.Lock()
	defer ecp.mu.Unlock()
	
	// Close all pooled connections
	close(ecp.connections)
	for conn := range ecp.connections {
		ecp.destroyConnection(conn)
	}
	
	// Close all active connections
	for _, conn := range ecp.activeConnections {
		ecp.destroyConnection(conn)
	}
	
	log.Printf("Closed connection pool with %d destroyed connections", 
		atomic.LoadInt64(&ecp.destroyedCount))
	return nil
}

// GetMetrics returns pool-specific metrics
func (ecp *EnhancedConnectionPool) GetMetrics() *PoolMetrics {
	ecp.metrics.mu.RLock()
	defer ecp.metrics.mu.RUnlock()
	
	return &PoolMetrics{
		Requests:     ecp.metrics.Requests,
		Hits:         ecp.metrics.Hits,
		Misses:       ecp.metrics.Misses,
		Timeouts:     ecp.metrics.Timeouts,
		Errors:       ecp.metrics.Errors,
		AvgWaitTime:  ecp.metrics.AvgWaitTime,
		MaxWaitTime:  ecp.metrics.MaxWaitTime,
		LastActivity: ecp.metrics.LastActivity,
	}
}

// isConnectionValid checks if a connection is still valid and healthy
func (ecp *EnhancedConnectionPool) isConnectionValid(conn *PooledConnection) bool {
	if conn == nil {
		return false
	}
	
	// Check if connection is too old
	if ecp.config.MaxLifetime > 0 && time.Since(conn.createdAt) > ecp.config.MaxLifetime {
		return false
	}
	
	// Check if connection has been idle too long
	if time.Since(conn.lastUsed) > ecp.config.IdleTimeout {
		return false
	}
	
	// Check health flag
	if !conn.isHealthy {
		return false
	}
	
	// Perform simple connectivity check for network connections
	if netConn, ok := conn.conn.(net.Conn); ok {
		// Set a very short deadline for health check
		netConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		
		// Try to read one byte without blocking
		buffer := make([]byte, 1)
		_, err := netConn.Read(buffer)
		
		// Reset deadline
		netConn.SetReadDeadline(time.Time{})
		
		// If we get EOF or timeout, connection might be closed
		if err != nil {
			return false
		}
	}
	
	return true
}

// destroyConnection properly destroys a connection and updates metrics
func (ecp *EnhancedConnectionPool) destroyConnection(conn *PooledConnection) {
	if conn == nil {
		return
	}
	
	// Close the underlying connection
	if closer, ok := conn.conn.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			log.Printf("Error closing connection %s: %v", conn.id, err)
		}
	}
	
	atomic.AddInt64(&ecp.destroyedCount, 1)
	log.Printf("Destroyed connection %s (used %d times, age: %v)", 
		conn.id, conn.useCount, time.Since(conn.createdAt))
}

// ResourceMonitor implementation

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{
		cpuThreshold:     80.0,
		memoryThreshold:  85.0,
		networkThreshold: 90.0,
		interval:         5 * time.Second,
		lastCheck:        time.Now(),
	}
}

// Start starts the resource monitor
func (rm *ResourceMonitor) Start() error {
	// In a production environment, this would start actual resource monitoring
	// For now, we'll simulate resource monitoring
	go rm.monitoringLoop()
	
	log.Println("Resource monitor started")
	return nil
}

// Stop stops the resource monitor
func (rm *ResourceMonitor) Stop() error {
	log.Println("Resource monitor stopped")
	return nil
}

// GetResourceUsage returns current resource usage
func (rm *ResourceMonitor) GetResourceUsage() ResourceUsage {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	return ResourceUsage{
		CPUUsage:     rm.cpuUsage,
		MemoryUsage:  rm.memoryUsage,
		NetworkUsage: rm.networkUsage,
		DiskUsage:    rm.diskUsage,
	}
}

// monitoringLoop performs periodic resource monitoring
func (rm *ResourceMonitor) monitoringLoop() {
	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()
	
	for range ticker.C {
		rm.updateResourceUsage()
	}
}

// updateResourceUsage updates current resource usage metrics
func (rm *ResourceMonitor) updateResourceUsage() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Get memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Calculate memory usage percentage (simplified)
	rm.memoryUsage = float64(memStats.Alloc) / float64(memStats.Sys) * 100
	
	// Simulate CPU usage (in production, use system calls)
	rm.cpuUsage = 15.0 + (float64(time.Now().Unix()%10) * 2.0) // Simulated 15-35%
	
	// Simulate network usage
	rm.networkUsage = 5.0 + (float64(time.Now().Unix()%5) * 3.0) // Simulated 5-20%
	
	// Simulate disk usage
	rm.diskUsage = 30.0 + (float64(time.Now().Unix()%8) * 1.5) // Simulated 30-42%
	
	rm.lastCheck = time.Now()
}

// PoolHealthChecker implementation

// NewPoolHealthChecker creates a new pool health checker
func NewPoolHealthChecker(interval time.Duration) *PoolHealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PoolHealthChecker{
		pools:    make(map[string]*EnhancedConnectionPool),
		interval: interval,
		timeout:  10 * time.Second,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the pool health checker
func (phc *PoolHealthChecker) Start() error {
	phc.mu.Lock()
	defer phc.mu.Unlock()
	
	if phc.isRunning {
		return fmt.Errorf("pool health checker is already running")
	}
	
	go phc.healthCheckLoop()
	phc.isRunning = true
	
	log.Println("Pool health checker started")
	return nil
}

// Stop stops the pool health checker
func (phc *PoolHealthChecker) Stop() error {
	phc.mu.Lock()
	defer phc.mu.Unlock()
	
	if !phc.isRunning {
		return nil
	}
	
	phc.cancel()
	phc.isRunning = false
	
	log.Println("Pool health checker stopped")
	return nil
}

// AddPool adds a pool to health monitoring
func (phc *PoolHealthChecker) AddPool(name string, pool *EnhancedConnectionPool) {
	phc.mu.Lock()
	defer phc.mu.Unlock()
	
	phc.pools[name] = pool
	log.Printf("Added pool %s to health monitoring", name)
}

// healthCheckLoop performs periodic health checks on all pools
func (phc *PoolHealthChecker) healthCheckLoop() {
	ticker := time.NewTicker(phc.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-phc.ctx.Done():
			return
		case <-ticker.C:
			phc.performHealthChecks()
		}
	}
}

// performHealthChecks performs health checks on all registered pools
func (phc *PoolHealthChecker) performHealthChecks() {
	phc.mu.RLock()
	pools := make(map[string]*EnhancedConnectionPool)
	for name, pool := range phc.pools {
		pools[name] = pool
	}
	phc.mu.RUnlock()
	
	for name, pool := range pools {
		go phc.checkPoolHealth(name, pool)
	}
}

// checkPoolHealth checks the health of a specific pool
func (phc *PoolHealthChecker) checkPoolHealth(name string, pool *EnhancedConnectionPool) {
	pool.mu.RLock()
	activeCount := len(pool.activeConnections)
	pooledCount := len(pool.connections)
	errorCount := atomic.LoadInt64(&pool.errorCount)
	pool.mu.RUnlock()
	
	// Calculate health score
	metrics := pool.GetMetrics()
	errorRate := float64(0)
	if metrics.Requests > 0 {
		errorRate = float64(metrics.Errors) / float64(metrics.Requests) * 100
	}
	
	// Determine health status
	previousStatus := pool.healthStatus
	
	if errorRate > 50 || errorCount > 10 {
		pool.healthStatus = PoolUnhealthy
	} else if errorRate > 20 || activeCount >= int(float64(pool.config.MaxActive)*0.9) {
		pool.healthStatus = PoolDegraded
	} else {
		pool.healthStatus = PoolHealthy
	}
	
	// Log status changes
	if pool.healthStatus != previousStatus {
		log.Printf("Pool %s health changed from %s to %s (error_rate=%.1f%%, active=%d/%d)", 
			name, previousStatus.String(), pool.healthStatus.String(), 
			errorRate, activeCount, pool.config.MaxActive)
	}
	
	pool.lastHealthCheck = time.Now()
}
