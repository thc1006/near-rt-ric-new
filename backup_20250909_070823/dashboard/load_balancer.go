/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancer manages load distribution across multiple instances
type LoadBalancer struct {
	algorithm    LoadBalancingAlgorithm
	backends     []*Backend
	healthChecker *HealthCheckerImpl
	metrics      *LoadBalancerMetrics
	mu           sync.RWMutex
}



// HealthCheckerImpl is the concrete implementation of HealthChecker interface
type HealthCheckerImpl struct {
	interval     time.Duration
	timeout      time.Duration
	retries      int
	healthChecks map[string]*HealthCheck
	mu           sync.RWMutex
}

// HealthCheck represents a health check configuration
type HealthCheck struct {
	Path           string
	Method         string
	ExpectedStatus int
	Headers        map[string]string
	Body           string
}

// LoadBalancerMetrics tracks load balancer performance
type LoadBalancerMetrics struct {
	TotalRequests     uint64
	SuccessfulRequests uint64
	FailedRequests    uint64
	AverageLatency    time.Duration
	BackendStats      map[string]*BackendStats
	mu                sync.RWMutex
}

// BackendStats tracks individual backend statistics
type BackendStats struct {
	Requests      uint64
	Errors        uint64
	TotalLatency  time.Duration
	LastUsed      time.Time
}

// SubscriptionDistributor manages subscription load balancing
type SubscriptionDistributor struct {
	loadBalancer    *LoadBalancer
	subscriptions   *FastHashMap // subscription ID -> backend mapping
	nodeDistribution map[string]string // E2 node ID -> backend mapping
	mu              sync.RWMutex
}

// ConnectionPool manages connection pooling and reuse
type ConnectionPool struct {
	pools       map[string]*Pool // backend ID -> connection pool
	maxIdle     int
	maxActive   int
	idleTimeout time.Duration
	mu          sync.RWMutex
}

// Pool represents a connection pool for a specific backend
type Pool struct {
	connections chan *Connection
	factory     ConnectionFactory
	maxIdle     int
	maxActive   int
	activeCount int64
	mu          sync.RWMutex
}

// Connection represents a pooled connection
type Connection struct {
	conn      interface{}
	createdAt time.Time
	lastUsed  time.Time
	inUse     int32 // atomic boolean
}

// ConnectionFactory creates new connections
type ConnectionFactory func(address string) (interface{}, error)

// BackpressureManager handles flow control and backpressure
type BackpressureManager struct {
	queues          map[string]*BackpressureQueue
	globalThreshold int64
	backendLimits   map[string]int64
	mu              sync.RWMutex
}

// BackpressureQueue manages queuing with backpressure
type BackpressureQueue struct {
	queue       *LockFreeQueue
	maxSize     int64
	currentSize int64
	dropRate    float64
	mu          sync.RWMutex
}

// FailoverManager handles component redundancy and failover
type FailoverManager struct {
	primaryBackends   []*Backend
	secondaryBackends []*Backend
	failoverPolicy    FailoverPolicy
	circuitBreakers   map[string]*CircuitBreaker
	mu                sync.RWMutex
}

// FailoverPolicy defines failover behavior
type FailoverPolicy struct {
	MaxFailures     int
	FailureWindow   time.Duration
	RecoveryTimeout time.Duration
	AutoFailback    bool
}


// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(algorithm LoadBalancingAlgorithm) *LoadBalancer {
	return &LoadBalancer{
		algorithm:     algorithm,
		backends:      make([]*Backend, 0),
		healthChecker: NewHealthCheckerImpl(),
		metrics:       NewLoadBalancerMetrics(),
	}
}

// AddBackend adds a backend to the load balancer
func (lb *LoadBalancer) AddBackend(backend *Backend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	lb.backends = append(lb.backends, backend)
	lb.metrics.BackendStats[backend.ID] = &BackendStats{}
	
	// Start health checking for the new backend
	lb.healthChecker.AddBackend(backend)
}

// RemoveBackend removes a backend from the load balancer
func (lb *LoadBalancer) RemoveBackend(backendID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	for i, backend := range lb.backends {
		if backend.ID == backendID {
			lb.backends = append(lb.backends[:i], lb.backends[i+1:]...)
			delete(lb.metrics.BackendStats, backendID)
			lb.healthChecker.RemoveBackend(backendID)
			break
		}
	}
}

// SelectBackend selects a backend based on the configured algorithm
func (lb *LoadBalancer) SelectBackend(key string) (*Backend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	healthyBackends := lb.getHealthyBackends()
	if len(healthyBackends) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}
	
	switch lb.algorithm {
	case RoundRobin:
		return lb.roundRobin(healthyBackends), nil
	case WeightedRoundRobin:
		return lb.weightedRoundRobin(healthyBackends), nil
	case LeastConnections:
		return lb.leastConnections(healthyBackends), nil
	case WeightedLeastConnections:
		return lb.weightedLeastConnections(healthyBackends), nil
	case ConsistentHashing:
		return lb.consistentHashing(healthyBackends, key), nil
	case ResourceBased:
		return lb.resourceBased(healthyBackends), nil
	case LatencyBased:
		return lb.latencyBased(healthyBackends), nil
	default:
		return lb.roundRobin(healthyBackends), nil
	}
}

// getHealthyBackends returns only healthy backends
func (lb *LoadBalancer) getHealthyBackends() []*Backend {
	var healthy []*Backend
	for _, backend := range lb.backends {
		if atomic.LoadInt32(&backend.IsHealthy) == 1 {
			healthy = append(healthy, backend)
		}
	}
	return healthy
}

// roundRobin implements round-robin load balancing
func (lb *LoadBalancer) roundRobin(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}
	
	index := int(atomic.AddUint64(&lb.metrics.TotalRequests, 1) - 1)
	return backends[index%len(backends)]
}

// weightedRoundRobin implements weighted round-robin load balancing
func (lb *LoadBalancer) weightedRoundRobin(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}
	
	totalWeight := 0
	for _, backend := range backends {
		totalWeight += backend.Weight
	}
	
	if totalWeight == 0 {
		return lb.roundRobin(backends)
	}
	
	requestNum := int(atomic.AddUint64(&lb.metrics.TotalRequests, 1))
	weightedIndex := requestNum % totalWeight
	
	currentWeight := 0
	for _, backend := range backends {
		currentWeight += backend.Weight
		if weightedIndex < currentWeight {
			return backend
		}
	}
	
	return backends[0] // Fallback
}

// leastConnections implements least connections load balancing
func (lb *LoadBalancer) leastConnections(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}
	
	var selected *Backend
	minConnections := int64(^uint64(0) >> 1) // Max int64
	
	for _, backend := range backends {
		connections := atomic.LoadInt64(&backend.CurrentConns)
		if connections < minConnections {
			minConnections = connections
			selected = backend
		}
	}
	
	return selected
}

// weightedLeastConnections implements weighted least connections
func (lb *LoadBalancer) weightedLeastConnections(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}
	
	var selected *Backend
	minRatio := float64(^uint64(0) >> 1) // Max float64
	
	for _, backend := range backends {
		if backend.Weight == 0 {
			continue
		}
		
		connections := atomic.LoadInt64(&backend.CurrentConns)
		ratio := float64(connections) / float64(backend.Weight)
		
		if ratio < minRatio {
			minRatio = ratio
			selected = backend
		}
	}
	
	return selected
}

// consistentHashing implements consistent hashing
func (lb *LoadBalancer) consistentHashing(backends []*Backend, key string) *Backend {
	if len(backends) == 0 {
		return nil
	}
	
	// Simple hash-based selection (in production, use proper consistent hashing)
	hash := lb.hash(key)
	index := int(hash) % len(backends)
	return backends[index]
}

// resourceBased selects backend based on resource utilization
func (lb *LoadBalancer) resourceBased(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}
	
	var selected *Backend
	minLoad := float64(1.0) // 100% load
	
	for _, backend := range backends {
		// Calculate combined load (CPU + Memory)
		load := (backend.CPUUsage + backend.MemoryUsage) / 2.0
		if load < minLoad {
			minLoad = load
			selected = backend
		}
	}
	
	return selected
}

// latencyBased selects backend with lowest latency
func (lb *LoadBalancer) latencyBased(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}
	
	var selected *Backend
	minLatency := time.Duration(^uint64(0) >> 1) // Max duration
	
	for _, backend := range backends {
		if backend.ResponseTime < minLatency {
			minLatency = backend.ResponseTime
			selected = backend
		}
	}
	
	return selected
}

// hash computes a hash for consistent hashing
func (lb *LoadBalancer) hash(key string) uint64 {
	h := uint64(0)
	for _, c := range key {
		h = h*31 + uint64(c)
	}
	return h
}

// NewHealthCheckerImpl creates a new health checker implementation
func NewHealthCheckerImpl() *HealthCheckerImpl {
	return &HealthCheckerImpl{
		interval:     time.Second * 30,
		timeout:      time.Second * 5,
		retries:      3,
		healthChecks: make(map[string]*HealthCheck),
	}
}

// AddBackend adds a backend for health checking
func (hc *HealthCheckerImpl) AddBackend(backend *Backend) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.healthChecks[backend.ID] = &HealthCheck{
		Path:           "/health",
		Method:         "GET",
		ExpectedStatus: 200,
	}
	
	// Start health checking goroutine
	go hc.checkBackendHealth(backend)
}

// RemoveBackend removes a backend from health checking
func (hc *HealthCheckerImpl) RemoveBackend(backendID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	delete(hc.healthChecks, backendID)
}

// checkBackendHealth continuously checks backend health
func (hc *HealthCheckerImpl) checkBackendHealth(backend *Backend) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()
	
	for range ticker.C {
		hc.mu.RLock()
		healthCheck, exists := hc.healthChecks[backend.ID]
		hc.mu.RUnlock()
		
		if !exists {
			return // Backend was removed
		}
		
		healthy := hc.performHealthCheck(backend, healthCheck)
		
		if healthy {
			atomic.StoreInt32(&backend.IsHealthy, 1)
		} else {
			atomic.StoreInt32(&backend.IsHealthy, 0)
		}
		
		backend.LastCheck = time.Now()
	}
}

// performHealthCheck performs actual health check
func (hc *HealthCheckerImpl) performHealthCheck(backend *Backend, check *HealthCheck) bool {
	// This is a simplified health check
	// In production, this would make actual HTTP requests
	
	// Simulate health check with some randomness
	return rand.Float64() > 0.1 // 90% success rate
}

// CheckHealth implements HealthChecker interface
func (hc *HealthCheckerImpl) CheckHealth(ctx context.Context) (*HealthCheckResult, error) {
	// Simple health check implementation
	return &HealthCheckResult{
		Healthy:      true,
		ResponseTime: time.Millisecond * 10,
	}, nil
}

// GetServiceName implements HealthChecker interface
func (hc *HealthCheckerImpl) GetServiceName() string {
	return "load-balancer-health-checker"
}

// NewLoadBalancerMetrics creates new load balancer metrics
func NewLoadBalancerMetrics() *LoadBalancerMetrics {
	return &LoadBalancerMetrics{
		BackendStats: make(map[string]*BackendStats),
	}
}

// RecordRequest records a request to the metrics
func (lbm *LoadBalancerMetrics) RecordRequest(backendID string, latency time.Duration, success bool) {
	atomic.AddUint64(&lbm.TotalRequests, 1)
	
	if success {
		atomic.AddUint64(&lbm.SuccessfulRequests, 1)
	} else {
		atomic.AddUint64(&lbm.FailedRequests, 1)
	}
	
	lbm.mu.Lock()
	defer lbm.mu.Unlock()
	
	stats, exists := lbm.BackendStats[backendID]
	if !exists {
		stats = &BackendStats{}
		lbm.BackendStats[backendID] = stats
	}
	
	atomic.AddUint64(&stats.Requests, 1)
	if !success {
		atomic.AddUint64(&stats.Errors, 1)
	}
	
	stats.TotalLatency += latency
	stats.LastUsed = time.Now()
}

// NewSubscriptionDistributor creates a new subscription distributor
func NewSubscriptionDistributor(loadBalancer *LoadBalancer) *SubscriptionDistributor {
	return &SubscriptionDistributor{
		loadBalancer:     loadBalancer,
		subscriptions:    NewFastHashMap(1000),
		nodeDistribution: make(map[string]string),
	}
}

// DistributeSubscription distributes a subscription to a backend
func (sd *SubscriptionDistributor) DistributeSubscription(subscriptionID, e2NodeID string) (*Backend, error) {
	// Check if E2 node already has an assigned backend
	sd.mu.RLock()
	if backendID, exists := sd.nodeDistribution[e2NodeID]; exists {
		sd.mu.RUnlock()
		
		// Find the backend
		backend := sd.findBackend(backendID)
		if backend != nil && atomic.LoadInt32(&backend.IsHealthy) == 1 {
			// Store subscription mapping
			sd.subscriptions.Put(subscriptionID, unsafe.Pointer(&backendID))
			return backend, nil
		}
	}
	sd.mu.RUnlock()
	
	// Select new backend for the E2 node
	backend, err := sd.loadBalancer.SelectBackend(e2NodeID)
	if err != nil {
		return nil, err
	}
	
	// Update mappings
	sd.mu.Lock()
	sd.nodeDistribution[e2NodeID] = backend.ID
	sd.mu.Unlock()
	
	sd.subscriptions.Put(subscriptionID, unsafe.Pointer(&backend.ID))
	
	return backend, nil
}

// GetSubscriptionBackend gets the backend for a subscription
func (sd *SubscriptionDistributor) GetSubscriptionBackend(subscriptionID string) *Backend {
	if backendIDPtr, exists := sd.subscriptions.Get(subscriptionID); exists {
		backendID := *(*string)(backendIDPtr)
		return sd.findBackend(backendID)
	}
	return nil
}

// findBackend finds a backend by ID
func (sd *SubscriptionDistributor) findBackend(backendID string) *Backend {
	sd.loadBalancer.mu.RLock()
	defer sd.loadBalancer.mu.RUnlock()
	
	for _, backend := range sd.loadBalancer.backends {
		if backend.ID == backendID {
			return backend
		}
	}
	return nil
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxIdle, maxActive int, idleTimeout time.Duration) *ConnectionPool {
	return &ConnectionPool{
		pools:       make(map[string]*Pool),
		maxIdle:     maxIdle,
		maxActive:   maxActive,
		idleTimeout: idleTimeout,
	}
}

// GetConnection gets a connection from the pool
func (cp *ConnectionPool) GetConnection(backendID string, factory ConnectionFactory, address string) (*Connection, error) {
	cp.mu.RLock()
	pool, exists := cp.pools[backendID]
	cp.mu.RUnlock()
	
	if !exists {
		cp.mu.Lock()
		if pool, exists = cp.pools[backendID]; !exists {
			pool = &Pool{
				connections: make(chan *Connection, cp.maxIdle),
				factory:     factory,
				maxIdle:     cp.maxIdle,
				maxActive:   cp.maxActive,
			}
			cp.pools[backendID] = pool
		}
		cp.mu.Unlock()
	}
	
	return pool.GetConnection(address)
}

// ReturnConnection returns a connection to the pool
func (cp *ConnectionPool) ReturnConnection(backendID string, conn *Connection) {
	cp.mu.RLock()
	pool, exists := cp.pools[backendID]
	cp.mu.RUnlock()
	
	if exists {
		pool.ReturnConnection(conn)
	}
}

// GetConnection gets a connection from the pool
func (p *Pool) GetConnection(address string) (*Connection, error) {
	// Try to get from pool first
	select {
	case conn := <-p.connections:
		if time.Since(conn.lastUsed) < time.Minute*5 { // Connection is still fresh
			atomic.StoreInt32(&conn.inUse, 1)
			return conn, nil
		}
		// Connection is stale, create new one
	default:
		// Pool is empty
	}
	
	// Check if we can create new connection
	if atomic.LoadInt64(&p.activeCount) >= int64(p.maxActive) {
		return nil, fmt.Errorf("connection pool exhausted")
	}
	
	// Create new connection
	rawConn, err := p.factory(address)
	if err != nil {
		return nil, err
	}
	
	conn := &Connection{
		conn:      rawConn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		inUse:     1,
	}
	
	atomic.AddInt64(&p.activeCount, 1)
	return conn, nil
}

// ReturnConnection returns a connection to the pool
func (p *Pool) ReturnConnection(conn *Connection) {
	atomic.StoreInt32(&conn.inUse, 0)
	conn.lastUsed = time.Now()
	
	select {
	case p.connections <- conn:
		// Successfully returned to pool
	default:
		// Pool is full, close connection
		atomic.AddInt64(&p.activeCount, -1)
	}
}

// NewBackpressureManager creates a new backpressure manager
func NewBackpressureManager(globalThreshold int64) *BackpressureManager {
	return &BackpressureManager{
		queues:          make(map[string]*BackpressureQueue),
		globalThreshold: globalThreshold,
		backendLimits:   make(map[string]int64),
	}
}

// AddQueue adds a backpressure queue for a backend
func (bm *BackpressureManager) AddQueue(backendID string, maxSize int64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	bm.queues[backendID] = &BackpressureQueue{
		queue:   NewLockFreeQueue(),
		maxSize: maxSize,
	}
	bm.backendLimits[backendID] = maxSize
}

// Enqueue adds an item to the queue with backpressure handling
func (bm *BackpressureManager) Enqueue(backendID string, item unsafe.Pointer) error {
	bm.mu.RLock()
	queue, exists := bm.queues[backendID]
	bm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("queue not found for backend %s", backendID)
	}
	
	// Check if queue is full
	if atomic.LoadInt64(&queue.currentSize) >= queue.maxSize {
		// Apply drop policy
		if rand.Float64() < queue.dropRate {
			return fmt.Errorf("item dropped due to backpressure")
		}
	}
	
	queue.queue.Enqueue(item)
	atomic.AddInt64(&queue.currentSize, 1)
	
	return nil
}

// Dequeue removes an item from the queue
func (bm *BackpressureManager) Dequeue(backendID string) (unsafe.Pointer, bool) {
	bm.mu.RLock()
	queue, exists := bm.queues[backendID]
	bm.mu.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	item, ok := queue.queue.Dequeue()
	if ok {
		atomic.AddInt64(&queue.currentSize, -1)
	}
	
	return item, ok
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager(policy FailoverPolicy) *FailoverManager {
	return &FailoverManager{
		primaryBackends:   make([]*Backend, 0),
		secondaryBackends: make([]*Backend, 0),
		failoverPolicy:    policy,
		circuitBreakers:   make(map[string]*CircuitBreaker),
	}
}

// AddPrimaryBackend adds a primary backend
func (fm *FailoverManager) AddPrimaryBackend(backend *Backend) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	fm.primaryBackends = append(fm.primaryBackends, backend)
	fm.circuitBreakers[backend.ID] = NewCircuitBreaker(
		int64(fm.failoverPolicy.MaxFailures),
		fm.failoverPolicy.RecoveryTimeout,
	)
}

// AddSecondaryBackend adds a secondary backend
func (fm *FailoverManager) AddSecondaryBackend(backend *Backend) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	fm.secondaryBackends = append(fm.secondaryBackends, backend)
	fm.circuitBreakers[backend.ID] = NewCircuitBreaker(
		int64(fm.failoverPolicy.MaxFailures),
		fm.failoverPolicy.RecoveryTimeout,
	)
}

// SelectBackend selects a backend with failover logic
func (fm *FailoverManager) SelectBackend() *Backend {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	// Try primary backends first
	for _, backend := range fm.primaryBackends {
		if cb := fm.circuitBreakers[backend.ID]; cb != nil {
			if cb.CanExecute() && atomic.LoadInt32(&backend.IsHealthy) == 1 {
				return backend
			}
		}
	}
	
	// Fallback to secondary backends
	for _, backend := range fm.secondaryBackends {
		if cb := fm.circuitBreakers[backend.ID]; cb != nil {
			if cb.CanExecute() && atomic.LoadInt32(&backend.IsHealthy) == 1 {
				return backend
			}
		}
	}
	
	return nil
}

// RecordSuccess records a successful operation
func (fm *FailoverManager) RecordSuccess(backendID string) {
	fm.mu.RLock()
	cb, exists := fm.circuitBreakers[backendID]
	fm.mu.RUnlock()
	
	if exists {
		cb.RecordSuccess()
	}
}

// RecordFailure records a failed operation
func (fm *FailoverManager) RecordFailure(backendID string) {
	fm.mu.RLock()
	cb, exists := fm.circuitBreakers[backendID]
	fm.mu.RUnlock()
	
	if exists {
		cb.RecordFailure()
	}
}

