/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// PerformanceOptimizer manages high-performance message processing
type PerformanceOptimizer struct {
	// Zero-copy message pools
	messagePool     *MessagePool
	bufferPool      *BufferPool
	
	// CPU affinity and thread management
	cpuManager      *CPUAffinityManager
	threadPool      *ThreadPool
	
	// Memory management
	memoryPool      *MemoryPool
	gcOptimizer     *GCOptimizer
	
	// Performance monitoring
	profiler        *PerformanceProfiler
	bottleneckDetector *BottleneckDetector
	
	// Metrics
	processedMessages uint64
	averageLatency    uint64 // nanoseconds
	throughput        uint64 // messages per second
	
	mu sync.RWMutex
}

// MessagePool provides zero-copy message handling
type MessagePool struct {
	pool sync.Pool
	size int
}

// BufferPool manages reusable byte buffers
type BufferPool struct {
	pools map[int]*sync.Pool // size -> pool
	mu    sync.RWMutex
}

// CPUAffinityManager handles CPU core assignment
type CPUAffinityManager struct {
	coreCount     int
	assignments   map[string]int // thread ID -> core ID
	criticalCores []int          // cores reserved for critical paths
	mu            sync.RWMutex
}

// ThreadPool manages optimized worker threads
type ThreadPool struct {
	workers    []*Worker
	workQueue  chan WorkItem
	resultChan chan WorkResult
	size       int
	running    int32
}

// Worker represents a high-performance worker thread
type Worker struct {
	id       int
	coreID   int
	workChan chan WorkItem
	quit     chan bool
	stats    WorkerStats
}

// WorkItem represents work to be processed
type WorkItem struct {
	ID        uint64
	Type      WorkType
	Data      unsafe.Pointer // Zero-copy data pointer
	Size      int
	Priority  Priority
	Timestamp time.Time
	Callback  func(result WorkResult)
}

// WorkResult contains processing results
type WorkResult struct {
	ID        uint64
	Success   bool
	Data      unsafe.Pointer
	Size      int
	Duration  time.Duration
	Error     error
}

// WorkType defines the type of work
type WorkType int

const (
	WorkTypeE2APMessage WorkType = iota
	WorkTypeSubscription
	WorkTypeIndication
	WorkTypeControl
	WorkTypePolicyUpdate
)

// Priority defines work priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// WorkerStats tracks worker performance
type WorkerStats struct {
	ProcessedItems uint64
	TotalDuration  time.Duration
	ErrorCount     uint64
	LastActive     time.Time
}

// MemoryPool manages memory allocation optimization
type MemoryPool struct {
	pools map[int]*sync.Pool // size -> pool
	stats MemoryStats
	mu    sync.RWMutex
}

// MemoryStats tracks memory usage
type MemoryStats struct {
	AllocatedBytes   uint64
	PoolHits         uint64
	PoolMisses       uint64
	GCPauses         uint64
	LastGCDuration   time.Duration
}

// GCOptimizer manages garbage collection optimization
type GCOptimizer struct {
	targetGCPercent int
	gcStats         runtime.MemStats
	lastOptimization time.Time
	mu              sync.RWMutex
}

// PerformanceProfiler provides runtime profiling
type PerformanceProfiler struct {
	enabled       bool
	samplingRate  time.Duration
	profiles      map[string]*ProfileData
	mu            sync.RWMutex
}

// ProfileData contains profiling information
type ProfileData struct {
	FunctionName   string
	CallCount      uint64
	TotalDuration  time.Duration
	AverageDuration time.Duration
	MaxDuration    time.Duration
	MinDuration    time.Duration
	LastCalled     time.Time
}

// BottleneckDetector identifies performance bottlenecks
type BottleneckDetector struct {
	thresholds    map[string]time.Duration
	violations    map[string]uint64
	alerts        chan BottleneckAlert
	enabled       bool
	mu            sync.RWMutex
}

// BottleneckAlert represents a performance bottleneck
type BottleneckAlert struct {
	Component   string
	Threshold   time.Duration
	Actual      time.Duration
	Severity    AlertSeverity
	Timestamp   time.Time
	Suggestions []string
}

// AlertSeverity defines alert severity levels
type AlertSeverity int

const (
	SeverityInfo AlertSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer() *PerformanceOptimizer {
	coreCount := runtime.NumCPU()
	
	optimizer := &PerformanceOptimizer{
		messagePool:        NewMessagePool(1024),
		bufferPool:         NewBufferPool(),
		cpuManager:         NewCPUAffinityManager(coreCount),
		threadPool:         NewThreadPool(coreCount * 2), // 2x cores for optimal performance
		memoryPool:         NewMemoryPool(),
		gcOptimizer:        NewGCOptimizer(),
		profiler:           NewPerformanceProfiler(),
		bottleneckDetector: NewBottleneckDetector(),
	}
	
	return optimizer
}

// NewMessagePool creates a new message pool
func NewMessagePool(size int) *MessagePool {
	return &MessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
		size: size,
	}
}

// GetMessage gets a message buffer from the pool (zero-copy)
func (mp *MessagePool) GetMessage() []byte {
	return mp.pool.Get().([]byte)
}

// PutMessage returns a message buffer to the pool
func (mp *MessagePool) PutMessage(msg []byte) {
	// Clear the buffer before returning to pool
	for i := range msg {
		msg[i] = 0
	}
	mp.pool.Put(msg)
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	bp := &BufferPool{
		pools: make(map[int]*sync.Pool),
	}
	
	// Pre-create pools for common sizes
	commonSizes := []int{64, 128, 256, 512, 1024, 2048, 4096, 8192}
	for _, size := range commonSizes {
		bp.pools[size] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		}
	}
	
	return bp
}

// GetBuffer gets a buffer of the specified size
func (bp *BufferPool) GetBuffer(size int) []byte {
	bp.mu.RLock()
	pool, exists := bp.pools[size]
	bp.mu.RUnlock()
	
	if !exists {
		// Create new pool for this size
		bp.mu.Lock()
		if _, exists := bp.pools[size]; !exists {
			bp.pools[size] = &sync.Pool{
				New: func() interface{} {
					return make([]byte, size)
				},
			}
		}
		pool = bp.pools[size]
		bp.mu.Unlock()
	}
	
	return pool.Get().([]byte)
}

// PutBuffer returns a buffer to the appropriate pool
func (bp *BufferPool) PutBuffer(buf []byte) {
	size := cap(buf)
	bp.mu.RLock()
	pool, exists := bp.pools[size]
	bp.mu.RUnlock()
	
	if exists {
		// Clear buffer before returning
		for i := range buf {
			buf[i] = 0
		}
		pool.Put(buf[:size]) // Reset length to capacity
	}
}

// NewCPUAffinityManager creates a new CPU affinity manager
func NewCPUAffinityManager(coreCount int) *CPUAffinityManager {
	// Reserve half the cores for critical paths
	criticalCores := make([]int, coreCount/2)
	for i := 0; i < coreCount/2; i++ {
		criticalCores[i] = i
	}
	
	return &CPUAffinityManager{
		coreCount:     coreCount,
		assignments:   make(map[string]int),
		criticalCores: criticalCores,
	}
}

// AssignCore assigns a CPU core to a thread
func (cam *CPUAffinityManager) AssignCore(threadID string, critical bool) int {
	cam.mu.Lock()
	defer cam.mu.Unlock()
	
	var coreID int
	if critical && len(cam.criticalCores) > 0 {
		// Assign from critical cores
		coreID = cam.criticalCores[len(cam.assignments)%len(cam.criticalCores)]
	} else {
		// Assign from all cores
		coreID = len(cam.assignments) % cam.coreCount
	}
	
	cam.assignments[threadID] = coreID
	return coreID
}

// GetAssignment gets the core assignment for a thread
func (cam *CPUAffinityManager) GetAssignment(threadID string) (int, bool) {
	cam.mu.RLock()
	defer cam.mu.RUnlock()
	
	coreID, exists := cam.assignments[threadID]
	return coreID, exists
}

// NewThreadPool creates a new optimized thread pool
func NewThreadPool(size int) *ThreadPool {
	tp := &ThreadPool{
		workers:    make([]*Worker, size),
		workQueue:  make(chan WorkItem, size*10), // Buffer for work items
		resultChan: make(chan WorkResult, size*10),
		size:       size,
	}
	
	// Create workers
	for i := 0; i < size; i++ {
		worker := &Worker{
			id:       i,
			workChan: make(chan WorkItem, 10),
			quit:     make(chan bool),
		}
		tp.workers[i] = worker
	}
	
	return tp
}

// Start starts the thread pool
func (tp *ThreadPool) Start(ctx context.Context, cpuManager *CPUAffinityManager) error {
	if atomic.LoadInt32(&tp.running) == 1 {
		return nil // Already running
	}
	
	atomic.StoreInt32(&tp.running, 1)
	
	// Start dispatcher
	go tp.dispatch(ctx)
	
	// Start workers
	for i, worker := range tp.workers {
		// Assign CPU core
		threadID := runtime.Goid() // Get goroutine ID
		coreID := cpuManager.AssignCore(string(rune(threadID)), i < len(tp.workers)/2) // First half are critical
		worker.coreID = coreID
		
		go worker.start(ctx, tp.resultChan)
	}
	
	return nil
}

// Stop stops the thread pool
func (tp *ThreadPool) Stop() {
	if atomic.LoadInt32(&tp.running) == 0 {
		return // Already stopped
	}
	
	atomic.StoreInt32(&tp.running, 0)
	
	// Stop all workers
	for _, worker := range tp.workers {
		worker.quit <- true
	}
	
	close(tp.workQueue)
	close(tp.resultChan)
}

// Submit submits work to the thread pool
func (tp *ThreadPool) Submit(item WorkItem) error {
	if atomic.LoadInt32(&tp.running) == 0 {
		return ErrThreadPoolStopped
	}
	
	select {
	case tp.workQueue <- item:
		return nil
	default:
		return ErrThreadPoolFull
	}
}

// dispatch distributes work to workers
func (tp *ThreadPool) dispatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-tp.workQueue:
			if !ok {
				return
			}
			
			// Find best worker based on priority and load
			worker := tp.selectWorker(item.Priority)
			if worker != nil {
				select {
				case worker.workChan <- item:
				default:
					// Worker is busy, try next available
					tp.workQueue <- item // Put back in queue
				}
			}
		}
	}
}

// selectWorker selects the best worker for the given priority
func (tp *ThreadPool) selectWorker(priority Priority) *Worker {
	var bestWorker *Worker
	var minLoad uint64 = ^uint64(0) // Max uint64
	
	for _, worker := range tp.workers {
		load := atomic.LoadUint64(&worker.stats.ProcessedItems)
		if load < minLoad {
			minLoad = load
			bestWorker = worker
		}
	}
	
	return bestWorker
}

// start starts a worker
func (w *Worker) start(ctx context.Context, resultChan chan<- WorkResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.quit:
			return
		case item := <-w.workChan:
			result := w.processItem(item)
			
			// Update stats
			atomic.AddUint64(&w.stats.ProcessedItems, 1)
			w.stats.TotalDuration += result.Duration
			w.stats.LastActive = time.Now()
			
			if result.Error != nil {
				atomic.AddUint64(&w.stats.ErrorCount, 1)
			}
			
			// Send result
			select {
			case resultChan <- result:
			default:
				// Result channel is full, handle callback directly
				if item.Callback != nil {
					item.Callback(result)
				}
			}
		}
	}
}

// processItem processes a work item
func (w *Worker) processItem(item WorkItem) WorkResult {
	start := time.Now()
	
	result := WorkResult{
		ID:        item.ID,
		Success:   true,
		Timestamp: start,
	}
	
	// Process based on work type
	switch item.Type {
	case WorkTypeE2APMessage:
		result.Data, result.Size, result.Error = w.processE2APMessage(item.Data, item.Size)
	case WorkTypeSubscription:
		result.Data, result.Size, result.Error = w.processSubscription(item.Data, item.Size)
	case WorkTypeIndication:
		result.Data, result.Size, result.Error = w.processIndication(item.Data, item.Size)
	case WorkTypeControl:
		result.Data, result.Size, result.Error = w.processControl(item.Data, item.Size)
	case WorkTypePolicyUpdate:
		result.Data, result.Size, result.Error = w.processPolicyUpdate(item.Data, item.Size)
	default:
		result.Error = ErrUnknownWorkType
	}
	
	result.Duration = time.Since(start)
	result.Success = result.Error == nil
	
	return result
}

// processE2APMessage processes E2AP messages with zero-copy techniques
func (w *Worker) processE2APMessage(data unsafe.Pointer, size int) (unsafe.Pointer, int, error) {
	// Zero-copy processing - work directly with the memory
	bytes := (*[1 << 30]byte)(data)[:size:size]
	
	// Process message in-place
	// This is a placeholder for actual E2AP message processing
	// In real implementation, this would parse ASN.1 PER encoding
	
	return data, size, nil
}

// processSubscription processes subscription requests
func (w *Worker) processSubscription(data unsafe.Pointer, size int) (unsafe.Pointer, int, error) {
	// Process subscription data
	return data, size, nil
}

// processIndication processes indication messages
func (w *Worker) processIndication(data unsafe.Pointer, size int) (unsafe.Pointer, int, error) {
	// Process indication data
	return data, size, nil
}

// processControl processes control messages
func (w *Worker) processControl(data unsafe.Pointer, size int) (unsafe.Pointer, int, error) {
	// Process control data
	return data, size, nil
}

// processPolicyUpdate processes policy updates
func (w *Worker) processPolicyUpdate(data unsafe.Pointer, size int) (unsafe.Pointer, int, error) {
	// Process policy update data
	return data, size, nil
}

// NewMemoryPool creates a new memory pool
func NewMemoryPool() *MemoryPool {
	mp := &MemoryPool{
		pools: make(map[int]*sync.Pool),
	}
	
	// Pre-create pools for common sizes
	commonSizes := []int{32, 64, 128, 256, 512, 1024, 2048, 4096}
	for _, size := range commonSizes {
		mp.pools[size] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		}
	}
	
	return mp
}

// Allocate allocates memory from the pool
func (mp *MemoryPool) Allocate(size int) []byte {
	mp.mu.RLock()
	pool, exists := mp.pools[size]
	mp.mu.RUnlock()
	
	if exists {
		atomic.AddUint64(&mp.stats.PoolHits, 1)
		return pool.Get().([]byte)
	}
	
	atomic.AddUint64(&mp.stats.PoolMisses, 1)
	atomic.AddUint64(&mp.stats.AllocatedBytes, uint64(size))
	return make([]byte, size)
}

// Free returns memory to the pool
func (mp *MemoryPool) Free(buf []byte) {
	size := cap(buf)
	mp.mu.RLock()
	pool, exists := mp.pools[size]
	mp.mu.RUnlock()
	
	if exists {
		// Clear buffer
		for i := range buf {
			buf[i] = 0
		}
		pool.Put(buf[:size])
	}
}

// GetStats returns memory pool statistics
func (mp *MemoryPool) GetStats() MemoryStats {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.stats
}

// NewGCOptimizer creates a new GC optimizer
func NewGCOptimizer() *GCOptimizer {
	return &GCOptimizer{
		targetGCPercent: 100, // Default Go GC target
	}
}

// OptimizeGC optimizes garbage collection settings
func (gco *GCOptimizer) OptimizeGC() {
	gco.mu.Lock()
	defer gco.mu.Unlock()
	
	// Read current memory stats
	runtime.ReadMemStats(&gco.gcStats)
	
	// Adjust GC target based on memory pressure
	if gco.gcStats.HeapAlloc > gco.gcStats.HeapSys/2 {
		// High memory usage, be more aggressive
		gco.targetGCPercent = 50
	} else {
		// Low memory usage, be less aggressive
		gco.targetGCPercent = 200
	}
	
	runtime.SetGCPercent(gco.targetGCPercent)
	gco.lastOptimization = time.Now()
}

// GetGCStats returns GC statistics
func (gco *GCOptimizer) GetGCStats() runtime.MemStats {
	gco.mu.RLock()
	defer gco.mu.RUnlock()
	return gco.gcStats
}

// NewPerformanceProfiler creates a new performance profiler
func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		enabled:      true,
		samplingRate: time.Millisecond * 100, // Sample every 100ms
		profiles:     make(map[string]*ProfileData),
	}
}

// StartProfiling starts profiling a function
func (pp *PerformanceProfiler) StartProfiling(functionName string) func() {
	if !pp.enabled {
		return func() {}
	}
	
	start := time.Now()
	
	return func() {
		duration := time.Since(start)
		pp.recordProfile(functionName, duration)
	}
}

// recordProfile records profiling data
func (pp *PerformanceProfiler) recordProfile(functionName string, duration time.Duration) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	
	profile, exists := pp.profiles[functionName]
	if !exists {
		profile = &ProfileData{
			FunctionName:  functionName,
			MinDuration:   duration,
			MaxDuration:   duration,
		}
		pp.profiles[functionName] = profile
	}
	
	profile.CallCount++
	profile.TotalDuration += duration
	profile.AverageDuration = time.Duration(int64(profile.TotalDuration) / int64(profile.CallCount))
	profile.LastCalled = time.Now()
	
	if duration < profile.MinDuration {
		profile.MinDuration = duration
	}
	if duration > profile.MaxDuration {
		profile.MaxDuration = duration
	}
}

// GetProfiles returns all profiling data
func (pp *PerformanceProfiler) GetProfiles() map[string]*ProfileData {
	pp.mu.RLock()
	defer pp.mu.RUnlock()
	
	profiles := make(map[string]*ProfileData)
	for k, v := range pp.profiles {
		profileCopy := *v
		profiles[k] = &profileCopy
	}
	
	return profiles
}

// NewBottleneckDetector creates a new bottleneck detector
func NewBottleneckDetector() *BottleneckDetector {
	bd := &BottleneckDetector{
		thresholds: map[string]time.Duration{
			"e2ap_processing":    time.Millisecond * 10,  // 10ms threshold
			"subscription_mgmt":  time.Millisecond * 50,  // 50ms threshold
			"indication_routing": time.Millisecond * 5,   // 5ms threshold
			"policy_validation":  time.Millisecond * 100, // 100ms threshold
		},
		violations: make(map[string]uint64),
		alerts:     make(chan BottleneckAlert, 100),
		enabled:    true,
	}
	
	return bd
}

// CheckBottleneck checks for performance bottlenecks
func (bd *BottleneckDetector) CheckBottleneck(component string, duration time.Duration) {
	if !bd.enabled {
		return
	}
	
	bd.mu.RLock()
	threshold, exists := bd.thresholds[component]
	bd.mu.RUnlock()
	
	if !exists {
		return
	}
	
	if duration > threshold {
		bd.mu.Lock()
		bd.violations[component]++
		violationCount := bd.violations[component]
		bd.mu.Unlock()
		
		// Determine severity based on violation count and duration
		severity := bd.calculateSeverity(duration, threshold, violationCount)
		
		alert := BottleneckAlert{
			Component:   component,
			Threshold:   threshold,
			Actual:      duration,
			Severity:    severity,
			Timestamp:   time.Now(),
			Suggestions: bd.generateSuggestions(component, duration, threshold),
		}
		
		select {
		case bd.alerts <- alert:
		default:
			// Alert channel is full, drop oldest alert
		}
	}
}

// calculateSeverity calculates alert severity
func (bd *BottleneckDetector) calculateSeverity(actual, threshold time.Duration, violations uint64) AlertSeverity {
	ratio := float64(actual) / float64(threshold)
	
	if violations > 100 || ratio > 10 {
		return SeverityCritical
	} else if violations > 50 || ratio > 5 {
		return SeverityError
	} else if violations > 10 || ratio > 2 {
		return SeverityWarning
	}
	
	return SeverityInfo
}

// generateSuggestions generates optimization suggestions
func (bd *BottleneckDetector) generateSuggestions(component string, actual, threshold time.Duration) []string {
	suggestions := []string{}
	
	switch component {
	case "e2ap_processing":
		suggestions = append(suggestions, "Consider optimizing ASN.1 encoding/decoding")
		suggestions = append(suggestions, "Increase thread pool size for E2AP processing")
		suggestions = append(suggestions, "Enable zero-copy message processing")
	case "subscription_mgmt":
		suggestions = append(suggestions, "Optimize subscription lookup data structures")
		suggestions = append(suggestions, "Consider caching subscription metadata")
		suggestions = append(suggestions, "Implement subscription batching")
	case "indication_routing":
		suggestions = append(suggestions, "Optimize message routing algorithms")
		suggestions = append(suggestions, "Increase indication processing threads")
		suggestions = append(suggestions, "Consider message prioritization")
	case "policy_validation":
		suggestions = append(suggestions, "Cache policy validation results")
		suggestions = append(suggestions, "Optimize JSON schema validation")
		suggestions = append(suggestions, "Consider policy pre-compilation")
	}
	
	return suggestions
}

// GetAlerts returns bottleneck alerts
func (bd *BottleneckDetector) GetAlerts() <-chan BottleneckAlert {
	return bd.alerts
}

// ProcessMessage processes a message with performance optimization
func (po *PerformanceOptimizer) ProcessMessage(ctx context.Context, msgType WorkType, data []byte, priority Priority) error {
	defer po.profiler.StartProfiling("ProcessMessage")()
	
	start := time.Now()
	
	// Get buffer from pool for zero-copy processing
	buffer := po.bufferPool.GetBuffer(len(data))
	defer po.bufferPool.PutBuffer(buffer)
	
	// Copy data to buffer (this could be optimized further with true zero-copy)
	copy(buffer, data)
	
	// Create work item
	workItem := WorkItem{
		ID:        atomic.AddUint64(&po.processedMessages, 1),
		Type:      msgType,
		Data:      unsafe.Pointer(&buffer[0]),
		Size:      len(data),
		Priority:  priority,
		Timestamp: start,
		Callback: func(result WorkResult) {
			// Update metrics
			duration := result.Duration
			atomic.StoreUint64(&po.averageLatency, uint64(duration.Nanoseconds()))
			
			// Check for bottlenecks
			component := po.getComponentName(msgType)
			po.bottleneckDetector.CheckBottleneck(component, duration)
		},
	}
	
	// Submit to thread pool
	if err := po.threadPool.Submit(workItem); err != nil {
		return err
	}
	
	return nil
}

// getComponentName maps work type to component name
func (po *PerformanceOptimizer) getComponentName(workType WorkType) string {
	switch workType {
	case WorkTypeE2APMessage:
		return "e2ap_processing"
	case WorkTypeSubscription:
		return "subscription_mgmt"
	case WorkTypeIndication:
		return "indication_routing"
	case WorkTypePolicyUpdate:
		return "policy_validation"
	default:
		return "unknown"
	}
}

// GetMetrics returns performance metrics
func (po *PerformanceOptimizer) GetMetrics() PerformanceMetrics {
	po.mu.RLock()
	defer po.mu.RUnlock()
	
	return PerformanceMetrics{
		ProcessedMessages: atomic.LoadUint64(&po.processedMessages),
		AverageLatency:    time.Duration(atomic.LoadUint64(&po.averageLatency)),
		Throughput:        atomic.LoadUint64(&po.throughput),
		MemoryStats:       po.memoryPool.GetStats(),
		GCStats:           po.gcOptimizer.GetGCStats(),
		ProfileData:       po.profiler.GetProfiles(),
	}
}

// PerformanceMetrics contains performance metrics
type PerformanceMetrics struct {
	ProcessedMessages uint64
	AverageLatency    time.Duration
	Throughput        uint64
	MemoryStats       MemoryStats
	GCStats           runtime.MemStats
	ProfileData       map[string]*ProfileData
}

// Start starts the performance optimizer
func (po *PerformanceOptimizer) Start(ctx context.Context) error {
	// Start thread pool
	if err := po.threadPool.Start(ctx, po.cpuManager); err != nil {
		return err
	}
	
	// Start GC optimization
	go po.gcOptimizationLoop(ctx)
	
	// Start throughput calculation
	go po.throughputCalculationLoop(ctx)
	
	return nil
}

// Stop stops the performance optimizer
func (po *PerformanceOptimizer) Stop() {
	po.threadPool.Stop()
}

// gcOptimizationLoop runs GC optimization periodically
func (po *PerformanceOptimizer) gcOptimizationLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5) // Optimize every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			po.gcOptimizer.OptimizeGC()
		}
	}
}

// throughputCalculationLoop calculates throughput periodically
func (po *PerformanceOptimizer) throughputCalculationLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	var lastCount uint64
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentCount := atomic.LoadUint64(&po.processedMessages)
			throughput := currentCount - lastCount
			atomic.StoreUint64(&po.throughput, throughput)
			lastCount = currentCount
		}
	}
}

// Error definitions
var (
	ErrThreadPoolStopped = fmt.Errorf("thread pool is stopped")
	ErrThreadPoolFull    = fmt.Errorf("thread pool is full")
	ErrUnknownWorkType   = fmt.Errorf("unknown work type")
)