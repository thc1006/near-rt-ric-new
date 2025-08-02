package dashboard

import (
	"context"
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// StabilityTestManager manages long-running stability tests
type StabilityTestManager struct {
	e2Manager           *E2ManagerClient
	subManager          *SubscriptionManagerClient
	memoryTracker       *MemoryTracker
	connectionTracker   *ConnectionTracker
	performanceTracker  *PerformanceTracker
	testMetrics         *StabilityTestMetrics
	mu                  sync.RWMutex
}

// MemoryTracker tracks memory usage over time for leak detection
type MemoryTracker struct {
	samples             []MemorySample
	baselineMemoryMB    float64
	peakMemoryMB        float64
	leakDetected        bool
	leakThresholdMB     float64
	samplingInterval    time.Duration
	mu                  sync.RWMutex
}

// ConnectionTracker tracks connection stability over time
type ConnectionTracker struct {
	totalConnections    int64
	droppedConnections  int64
	reconnections       int64
	connectionEvents    []ConnectionEvent
	stabilityScore      float64
	mu                  sync.RWMutex
}

// PerformanceTracker tracks performance degradation over time
type PerformanceTracker struct {
	initialThroughput   int
	currentThroughput   int
	initialLatencyMs    float64
	currentLatencyMs    float64
	throughputSamples   []ThroughputSample
	latencySamples      []LatencySample
	degradationDetected bool
	mu                  sync.RWMutex
}

// ConnectionEvent represents a connection-related event
type ConnectionEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"eventType"` // "connect", "disconnect", "reconnect", "timeout"
	NodeID      string    `json:"nodeId"`
	Details     string    `json:"details"`
}

// LatencySample represents a latency measurement sample
type LatencySample struct {
	Timestamp   time.Time `json:"timestamp"`
	LatencyMs   float64   `json:"latencyMs"`
	Operation   string    `json:"operation"`
}

// StabilityTestMetrics captures detailed stability test metrics
type StabilityTestMetrics struct {
	TestStartTime           time.Time                `json:"testStartTime"`
	TestDuration            time.Duration            `json:"testDuration"`
	MemorySamples           []MemorySample           `json:"memorySamples"`
	CPUSamples              []CPUSample              `json:"cpuSamples"`
	ConnectionEvents        []ConnectionEvent        `json:"connectionEvents"`
	ThroughputSamples       []ThroughputSample       `json:"throughputSamples"`
	LatencySamples          []LatencySample          `json:"latencySamples"`
	ErrorRateSamples        []ErrorRateSample        `json:"errorRateSamples"`
	MemoryLeakDetected      bool                     `json:"memoryLeakDetected"`
	PerformanceDegradation  *PerformanceDegradation  `json:"performanceDegradation"`
	ConnectionStability     *ConnectionStability     `json:"connectionStability"`
	ResourceUtilization     []*ResourceConsumption   `json:"resourceUtilization"`
	mu                      sync.RWMutex
}

// StabilityTestScenario defines a stability test scenario
type StabilityTestScenario struct {
	Name                    string        `json:"name"`
	TestDurationHours       float64       `json:"testDurationHours"`
	SamplingIntervalSeconds int           `json:"samplingIntervalSeconds"`
	MemoryLeakThresholdMB   float64       `json:"memoryLeakThresholdMB"`
	PerformanceDegradationThreshold float64 `json:"performanceDegradationThreshold"` // percentage
	ConnectionStabilityThreshold    float64 `json:"connectionStabilityThreshold"`    // percentage
	BaselineStabilizationMinutes    int     `json:"baselineStabilizationMinutes"`
	ContinuousLoad                  bool    `json:"continuousLoad"`
	LoadVariation                   bool    `json:"loadVariation"`
}

// NewStabilityTestManager creates a new stability test manager
func NewStabilityTestManager(e2Manager *E2ManagerClient, subManager *SubscriptionManagerClient) *StabilityTestManager {
	memoryTracker := &MemoryTracker{
		samples:          make([]MemorySample, 0),
		samplingInterval: 30 * time.Second,
		leakThresholdMB:  100, // 100MB increase indicates potential leak
	}

	connectionTracker := &ConnectionTracker{
		connectionEvents: make([]ConnectionEvent, 0),
	}

	performanceTracker := &PerformanceTracker{
		throughputSamples: make([]ThroughputSample, 0),
		latencySamples:    make([]LatencySample, 0),
	}

	return &StabilityTestManager{
		e2Manager:          e2Manager,
		subManager:         subManager,
		memoryTracker:      memoryTracker,
		connectionTracker:  connectionTracker,
		performanceTracker: performanceTracker,
		testMetrics: &StabilityTestMetrics{
			MemorySamples:     make([]MemorySample, 0),
			CPUSamples:        make([]CPUSample, 0),
			ConnectionEvents:  make([]ConnectionEvent, 0),
			ThroughputSamples: make([]ThroughputSample, 0),
			LatencySamples:    make([]LatencySample, 0),
			ErrorRateSamples:  make([]ErrorRateSample, 0),
			ResourceUtilization: make([]*ResourceConsumption, 0),
		},
	}
}

// RunStabilityTesting executes long-running stability tests
func (pts *PerformanceTestSuite) RunStabilityTesting(ctx context.Context) error {
	log.Println("Starting stability testing scenarios...")

	stabilityManager := NewStabilityTestManager(pts.e2Manager, pts.subManager)

	// Define stability test scenarios - Enhanced for memory leak detection
	scenarios := []*StabilityTestScenario{
		{
			Name:                            "Extended Stability Test - 48h",
			TestDurationHours:               float64(pts.config.StabilityTestHours),
			SamplingIntervalSeconds:         15, // More frequent sampling
			MemoryLeakThresholdMB:           float64(pts.config.MemoryLeakThresholdMB),
			PerformanceDegradationThreshold: 15.0, // Stricter degradation threshold
			ConnectionStabilityThreshold:    97.0, // Higher stability requirement
			BaselineStabilizationMinutes:    45,   // Longer baseline establishment
			ContinuousLoad:                  true,
			LoadVariation:                   false,
		},
		{
			Name:                            "Memory Leak Detection Test - 24h",
			TestDurationHours:               24.0, // Full day test
			SamplingIntervalSeconds:         10,   // Very frequent sampling for leak detection
			MemoryLeakThresholdMB:           25.0, // Very sensitive threshold
			PerformanceDegradationThreshold: 20.0,
			ConnectionStabilityThreshold:    95.0,
			BaselineStabilizationMinutes:    60, // Longer baseline for accurate leak detection
			ContinuousLoad:                  true,
			LoadVariation:                   false,
		},
		{
			Name:                            "Variable Load Stability Test - 36h",
			TestDurationHours:               36.0, // Extended variable load test
			SamplingIntervalSeconds:         30,
			MemoryLeakThresholdMB:           float64(pts.config.MemoryLeakThresholdMB),
			PerformanceDegradationThreshold: 25.0, // More lenient for variable load
			ConnectionStabilityThreshold:    92.0, // More lenient for variable load
			BaselineStabilizationMinutes:    30,
			ContinuousLoad:                  true,
			LoadVariation:                   true,
		},
		{
			Name:                            "High Load Stability Test - 12h",
			TestDurationHours:               12.0, // Shorter but high intensity
			SamplingIntervalSeconds:         20,
			MemoryLeakThresholdMB:           75.0, // Higher threshold for high load
			PerformanceDegradationThreshold: 30.0, // More lenient under high load
			ConnectionStabilityThreshold:    90.0,
			BaselineStabilizationMinutes:    20,
			ContinuousLoad:                  true,
			LoadVariation:                   false,
		},
		{
			Name:                            "Micro-Leak Detection Test - 72h",
			TestDurationHours:               72.0, // Very long test for micro-leaks
			SamplingIntervalSeconds:         5,    // Very frequent sampling
			MemoryLeakThresholdMB:           10.0, // Very sensitive for micro-leaks
			PerformanceDegradationThreshold: 10.0, // Very strict degradation detection
			ConnectionStabilityThreshold:    98.0, // Very high stability requirement
			BaselineStabilizationMinutes:    120,  // Very long baseline (2 hours)
			ContinuousLoad:                  true,
			LoadVariation:                   false,
		},
	}

	results := make([]*StabilityResults, len(scenarios))

	// Run scenarios sequentially (though each is very long)
	for i, scenario := range scenarios {
		log.Printf("Executing stability test scenario: %s (duration: %.1f hours)", 
			scenario.Name, scenario.TestDurationHours)
		
		result, err := stabilityManager.ExecuteScenario(ctx, scenario)
		if err != nil {
			log.Printf("Stability test scenario %s failed: %v", scenario.Name, err)
			continue
		}
		results[i] = result
		
		// Brief pause between scenarios
		time.Sleep(5 * time.Minute)
	}

	// Aggregate results
	pts.testResults.StabilityResults = pts.aggregateStabilityResults(results)
	
	log.Println("Stability testing completed")
	return nil
}

// ExecuteScenario runs a specific stability test scenario
func (stm *StabilityTestManager) ExecuteScenario(ctx context.Context, scenario *StabilityTestScenario) (*StabilityResults, error) {
	testDuration := time.Duration(scenario.TestDurationHours * float64(time.Hour))
	testCtx, cancel := context.WithTimeout(ctx, testDuration)
	defer cancel()

	// Reset metrics
	stm.testMetrics = &StabilityTestMetrics{
		TestStartTime:       time.Now(),
		MemorySamples:       make([]MemorySample, 0),
		CPUSamples:          make([]CPUSample, 0),
		ConnectionEvents:    make([]ConnectionEvent, 0),
		ThroughputSamples:   make([]ThroughputSample, 0),
		LatencySamples:      make([]LatencySample, 0),
		ErrorRateSamples:    make([]ErrorRateSample, 0),
		ResourceUtilization: make([]*ResourceConsumption, 0),
	}

	log.Printf("Starting stability test: %s", scenario.Name)

	// Start monitoring goroutines
	var wg sync.WaitGroup

	// Memory monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		stm.monitorMemoryUsage(testCtx, scenario)
	}()

	// CPU monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		stm.monitorCPUUsage(testCtx, scenario)
	}()

	// Connection monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		stm.monitorConnections(testCtx, scenario)
	}()

	// Performance monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		stm.monitorPerformance(testCtx, scenario)
	}()

	// Error rate monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		stm.monitorErrorRates(testCtx, scenario)
	}()

	// Resource utilization monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		stm.monitorResourceUtilization(testCtx, scenario)
	}()

	// Generate continuous load if required
	if scenario.ContinuousLoad {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stm.generateContinuousLoad(testCtx, scenario)
		}()
	}

	// Wait for test completion
	<-testCtx.Done()

	// Wait for all monitoring goroutines to complete
	wg.Wait()

	stm.testMetrics.TestDuration = time.Since(stm.testMetrics.TestStartTime)

	log.Printf("Stability test %s completed after %v", scenario.Name, stm.testMetrics.TestDuration)

	return stm.generateStabilityResults(scenario), nil
}

// monitorMemoryUsage monitors memory usage for leak detection
func (stm *StabilityTestManager) monitorMemoryUsage(ctx context.Context, scenario *StabilityTestScenario) {
	interval := time.Duration(scenario.SamplingIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	baselineEstablished := false
	baselineTimer := time.NewTimer(time.Duration(scenario.BaselineStabilizationMinutes) * time.Minute)
	defer baselineTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			sample := MemorySample{
				Timestamp: time.Now(),
				UsageMB:   float64(m.Alloc) / 1024 / 1024,
				HeapMB:    float64(m.HeapAlloc) / 1024 / 1024,
				StackMB:   float64(m.StackInuse) / 1024 / 1024,
			}

			stm.memoryTracker.mu.Lock()
			stm.memoryTracker.samples = append(stm.memoryTracker.samples, sample)

			// Update peak memory
			if sample.UsageMB > stm.memoryTracker.peakMemoryMB {
				stm.memoryTracker.peakMemoryMB = sample.UsageMB
			}

			// Establish baseline after stabilization period
			if !baselineEstablished {
				select {
				case <-baselineTimer.C:
					if len(stm.memoryTracker.samples) > 0 {
						// Calculate baseline as average of recent samples
						recentSamples := stm.memoryTracker.samples
						if len(recentSamples) > 10 {
							recentSamples = recentSamples[len(recentSamples)-10:]
						}
						
						sum := 0.0
						for _, s := range recentSamples {
							sum += s.UsageMB
						}
						stm.memoryTracker.baselineMemoryMB = sum / float64(len(recentSamples))
						baselineEstablished = true
						
						log.Printf("Memory baseline established: %.2f MB", stm.memoryTracker.baselineMemoryMB)
					}
				default:
				}
			}

			// Check for memory leak
			if baselineEstablished && !stm.memoryTracker.leakDetected {
				memoryIncrease := sample.UsageMB - stm.memoryTracker.baselineMemoryMB
				if memoryIncrease > scenario.MemoryLeakThresholdMB {
					stm.memoryTracker.leakDetected = true
					stm.testMetrics.MemoryLeakDetected = true
					log.Printf("Memory leak detected: %.2f MB increase from baseline", memoryIncrease)
				}
			}

			stm.memoryTracker.mu.Unlock()

			// Add to test metrics
			stm.testMetrics.mu.Lock()
			stm.testMetrics.MemorySamples = append(stm.testMetrics.MemorySamples, sample)
			stm.testMetrics.mu.Unlock()

		}
	}
}

// monitorCPUUsage monitors CPU usage over time
func (stm *StabilityTestManager) monitorCPUUsage(ctx context.Context, scenario *StabilityTestScenario) {
	interval := time.Duration(scenario.SamplingIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate CPU usage monitoring
			// In a real implementation, this would use system monitoring tools
			cpuPercent := stm.getCurrentCPUUsage()

			sample := CPUSample{
				Timestamp: time.Now(),
				Percent:   cpuPercent,
			}

			stm.testMetrics.mu.Lock()
			stm.testMetrics.CPUSamples = append(stm.testMetrics.CPUSamples, sample)
			stm.testMetrics.mu.Unlock()
		}
	}
}

// monitorConnections monitors connection stability
func (stm *StabilityTestManager) monitorConnections(ctx context.Context, scenario *StabilityTestScenario) {
	interval := time.Duration(scenario.SamplingIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate connection monitoring
			stm.simulateConnectionEvents()
		}
	}
}

// monitorPerformance monitors performance metrics for degradation detection
func (stm *StabilityTestManager) monitorPerformance(ctx context.Context, scenario *StabilityTestScenario) {
	interval := time.Duration(scenario.SamplingIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	baselineEstablished := false
	baselineTimer := time.NewTimer(time.Duration(scenario.BaselineStabilizationMinutes) * time.Minute)
	defer baselineTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate performance measurements
			currentThroughput := stm.getCurrentThroughput()
			currentLatency := stm.getCurrentLatency()

			throughputSample := ThroughputSample{
				Timestamp:           time.Now(),
				IndicationsPerSec:   currentThroughput,
				ProcessingLatencyMs: currentLatency,
			}

			latencySample := LatencySample{
				Timestamp: time.Now(),
				LatencyMs: currentLatency,
				Operation: "indication_processing",
			}

			stm.performanceTracker.mu.Lock()
			stm.performanceTracker.throughputSamples = append(stm.performanceTracker.throughputSamples, throughputSample)
			stm.performanceTracker.latencySamples = append(stm.performanceTracker.latencySamples, latencySample)
			stm.performanceTracker.currentThroughput = currentThroughput
			stm.performanceTracker.currentLatencyMs = currentLatency

			// Establish baseline
			if !baselineEstablished {
				select {
				case <-baselineTimer.C:
					stm.performanceTracker.initialThroughput = currentThroughput
					stm.performanceTracker.initialLatencyMs = currentLatency
					baselineEstablished = true
					log.Printf("Performance baseline established: %d IPS, %.2f ms latency", 
						currentThroughput, currentLatency)
				default:
				}
			}

			// Check for performance degradation
			if baselineEstablished && !stm.performanceTracker.degradationDetected {
				throughputDegradation := float64(stm.performanceTracker.initialThroughput-currentThroughput) / 
					float64(stm.performanceTracker.initialThroughput) * 100
				latencyDegradation := (currentLatency - stm.performanceTracker.initialLatencyMs) / 
					stm.performanceTracker.initialLatencyMs * 100

				if throughputDegradation > scenario.PerformanceDegradationThreshold ||
					latencyDegradation > scenario.PerformanceDegradationThreshold {
					stm.performanceTracker.degradationDetected = true
					log.Printf("Performance degradation detected: %.1f%% throughput, %.1f%% latency", 
						throughputDegradation, latencyDegradation)
				}
			}

			stm.performanceTracker.mu.Unlock()

			// Add to test metrics
			stm.testMetrics.mu.Lock()
			stm.testMetrics.ThroughputSamples = append(stm.testMetrics.ThroughputSamples, throughputSample)
			stm.testMetrics.LatencySamples = append(stm.testMetrics.LatencySamples, latencySample)
			stm.testMetrics.mu.Unlock()
		}
	}
}

// monitorErrorRates monitors error rates over time
func (stm *StabilityTestManager) monitorErrorRates(ctx context.Context, scenario *StabilityTestScenario) {
	interval := time.Duration(scenario.SamplingIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate error rate monitoring
			errorRate := stm.getCurrentErrorRate()

			sample := ErrorRateSample{
				Timestamp: time.Now(),
				ErrorRate: errorRate,
			}

			stm.testMetrics.mu.Lock()
			stm.testMetrics.ErrorRateSamples = append(stm.testMetrics.ErrorRateSamples, sample)
			stm.testMetrics.mu.Unlock()
		}
	}
}

// monitorResourceUtilization monitors overall resource utilization
func (stm *StabilityTestManager) monitorResourceUtilization(ctx context.Context, scenario *StabilityTestScenario) {
	interval := time.Duration(scenario.SamplingIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			utilization := &ResourceConsumption{
				CPUUsagePercent:    stm.getCurrentCPUUsage(),
				MemoryUsageMB:      float64(m.Alloc) / 1024 / 1024,
				NetworkBytesPerSec: int64(stm.getCurrentNetworkUsage()),
				DiskIOPerSec:       int64(stm.getCurrentDiskUsage()),
				GoroutineCount:     runtime.NumGoroutine(),
				GCPauseMs:          float64(m.PauseNs[(m.NumGC+255)%256]) / 1000000,
			}

			stm.testMetrics.mu.Lock()
			stm.testMetrics.ResourceUtilization = append(stm.testMetrics.ResourceUtilization, utilization)
			stm.testMetrics.mu.Unlock()
		}
	}
}

// generateContinuousLoad generates continuous load during stability testing
func (stm *StabilityTestManager) generateContinuousLoad(ctx context.Context, scenario *StabilityTestScenario) {
	baseLoad := 1000 // Base indications per second
	
	if scenario.LoadVariation {
		// Variable load pattern
		stm.generateVariableLoad(ctx, baseLoad)
	} else {
		// Constant load pattern
		stm.generateConstantLoad(ctx, baseLoad)
	}
}

// generateConstantLoad generates constant load
func (stm *StabilityTestManager) generateConstantLoad(ctx context.Context, targetIPS int) {
	interval := time.Second / time.Duration(targetIPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate processing work
			go func() {
				// Simulate indication processing
				time.Sleep(time.Millisecond)
			}()
		}
	}
}

// generateVariableLoad generates variable load with patterns
func (stm *StabilityTestManager) generateVariableLoad(ctx context.Context, baseLoad int) {
	ticker := time.NewTicker(5 * time.Minute) // Change load every 5 minutes
	defer ticker.Stop()

	currentLoad := baseLoad
	loadTicker := time.NewTicker(time.Second / time.Duration(currentLoad))
	defer loadTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Vary load between 50% and 150% of base load
			variation := 0.5 + (1.0 * float64(time.Now().Unix()%100) / 100.0)
			currentLoad = int(float64(baseLoad) * variation)
			
			loadTicker.Stop()
			loadTicker = time.NewTicker(time.Second / time.Duration(currentLoad))
			
			log.Printf("Load variation: %d IPS (%.1f%% of base)", currentLoad, variation*100)
			
		case <-loadTicker.C:
			// Generate load
			go func() {
				time.Sleep(time.Millisecond)
			}()
		}
	}
}

// Simulation helper methods
func (stm *StabilityTestManager) getCurrentCPUUsage() float64 {
	// Simulate CPU usage with some variation
	base := 30.0 + (20.0 * math.Sin(float64(time.Now().Unix())/3600.0)) // Hourly variation
	noise := (float64(time.Now().UnixNano()%1000) / 1000.0) * 10.0 // Random noise
	return math.Max(0, math.Min(100, base+noise))
}

func (stm *StabilityTestManager) getCurrentThroughput() int {
	// Simulate throughput with gradual degradation over time
	elapsed := time.Since(stm.testMetrics.TestStartTime).Hours()
	degradation := math.Max(0, elapsed*0.5) // 0.5% degradation per hour
	baseThroughput := 5000.0
	return int(baseThroughput * (1.0 - degradation/100.0))
}

func (stm *StabilityTestManager) getCurrentLatency() float64 {
	// Simulate latency with gradual increase over time
	elapsed := time.Since(stm.testMetrics.TestStartTime).Hours()
	increase := elapsed * 0.1 // 0.1ms increase per hour
	baseLatency := 2.0
	return baseLatency + increase
}

func (stm *StabilityTestManager) getCurrentErrorRate() float64 {
	// Simulate error rate with occasional spikes
	base := 0.1 // 0.1% base error rate
	if time.Now().Unix()%3600 < 60 { // Spike once per hour for 1 minute
		return base * 10
	}
	return base
}

func (stm *StabilityTestManager) getCurrentNetworkUsage() int {
	// Simulate network usage
	return 1000000 + int(time.Now().UnixNano()%500000) // 1-1.5 MB/s
}

func (stm *StabilityTestManager) getCurrentDiskUsage() int {
	// Simulate disk usage
	return 10000 + int(time.Now().UnixNano()%5000) // 10-15K IOPS
}

func (stm *StabilityTestManager) simulateConnectionEvents() {
	// Simulate occasional connection events
	if time.Now().Unix()%300 == 0 { // Every 5 minutes
		event := ConnectionEvent{
			Timestamp: time.Now(),
			EventType: "reconnect",
			NodeID:    fmt.Sprintf("node-%d", time.Now().Unix()%100),
			Details:   "Simulated connection reconnection",
		}

		stm.connectionTracker.mu.Lock()
		stm.connectionTracker.connectionEvents = append(stm.connectionTracker.connectionEvents, event)
		atomic.AddInt64(&stm.connectionTracker.reconnections, 1)
		stm.connectionTracker.mu.Unlock()

		stm.testMetrics.mu.Lock()
		stm.testMetrics.ConnectionEvents = append(stm.testMetrics.ConnectionEvents, event)
		stm.testMetrics.mu.Unlock()
	}
}

// generateStabilityResults creates comprehensive stability test results
func (stm *StabilityTestManager) generateStabilityResults(scenario *StabilityTestScenario) *StabilityResults {
	stm.testMetrics.mu.RLock()
	defer stm.testMetrics.mu.RUnlock()

	// Calculate performance degradation
	performanceDegradation := &PerformanceDegradation{}
	stm.performanceTracker.mu.RLock()
	if stm.performanceTracker.initialThroughput > 0 {
		performanceDegradation.InitialThroughput = stm.performanceTracker.initialThroughput
		performanceDegradation.FinalThroughput = stm.performanceTracker.currentThroughput
		performanceDegradation.DegradationPercent = float64(stm.performanceTracker.initialThroughput-stm.performanceTracker.currentThroughput) / 
			float64(stm.performanceTracker.initialThroughput) * 100
		performanceDegradation.InitialLatencyMs = stm.performanceTracker.initialLatencyMs
		performanceDegradation.FinalLatencyMs = stm.performanceTracker.currentLatencyMs
	}
	stm.performanceTracker.mu.RUnlock()

	// Calculate connection stability
	connectionStability := &ConnectionStability{}
	stm.connectionTracker.mu.RLock()
	totalConnections := atomic.LoadInt64(&stm.connectionTracker.totalConnections)
	droppedConnections := atomic.LoadInt64(&stm.connectionTracker.droppedConnections)
	reconnections := atomic.LoadInt64(&stm.connectionTracker.reconnections)

	if totalConnections > 0 {
		connectionStability.TotalConnections = totalConnections
		connectionStability.DroppedConnections = droppedConnections
		connectionStability.ReconnectionRate = float64(reconnections) / float64(totalConnections) * 100
		connectionStability.StabilityScore = (1.0 - float64(droppedConnections)/float64(totalConnections)) * 100
	} else {
		// Simulate connection stability metrics
		connectionStability.TotalConnections = 1000
		connectionStability.DroppedConnections = 5
		connectionStability.ReconnectionRate = 0.5
		connectionStability.StabilityScore = 99.5
	}
	stm.connectionTracker.mu.RUnlock()

	return &StabilityResults{
		TestDurationHours:      stm.testMetrics.TestDuration.Hours(),
		MemoryLeakDetected:     stm.testMetrics.MemoryLeakDetected,
		MemoryUsageOverTime:    stm.testMetrics.MemorySamples,
		CPUUsageOverTime:       stm.testMetrics.CPUSamples,
		ConnectionStability:    connectionStability,
		PerformanceDegradation: performanceDegradation,
		ErrorRateOverTime:      stm.testMetrics.ErrorRateSamples,
		Timestamp:              time.Now(),
	}
}

// aggregateStabilityResults combines results from multiple scenarios
func (pts *PerformanceTestSuite) aggregateStabilityResults(results []*StabilityResults) *StabilityResults {
	if len(results) == 0 {
		return &StabilityResults{Timestamp: time.Now()}
	}

	aggregated := &StabilityResults{
		MemoryUsageOverTime: make([]MemorySample, 0),
		CPUUsageOverTime:    make([]CPUSample, 0),
		ErrorRateOverTime:   make([]ErrorRateSample, 0),
		Timestamp:           time.Now(),
	}

	totalDuration := 0.0
	memoryLeakDetected := false
	
	// Initialize aggregated metrics
	aggregated.ConnectionStability = &ConnectionStability{}
	aggregated.PerformanceDegradation = &PerformanceDegradation{}

	for _, result := range results {
		if result == nil {
			continue
		}

		// Aggregate duration
		totalDuration += result.TestDurationHours

		// Check for memory leaks
		if result.MemoryLeakDetected {
			memoryLeakDetected = true
		}

		// Aggregate samples
		aggregated.MemoryUsageOverTime = append(aggregated.MemoryUsageOverTime, result.MemoryUsageOverTime...)
		aggregated.CPUUsageOverTime = append(aggregated.CPUUsageOverTime, result.CPUUsageOverTime...)
		aggregated.ErrorRateOverTime = append(aggregated.ErrorRateOverTime, result.ErrorRateOverTime...)

		// Take worst-case connection stability
		if result.ConnectionStability != nil {
			if aggregated.ConnectionStability.StabilityScore == 0 || 
				result.ConnectionStability.StabilityScore < aggregated.ConnectionStability.StabilityScore {
				aggregated.ConnectionStability = result.ConnectionStability
			}
		}

		// Take worst-case performance degradation
		if result.PerformanceDegradation != nil {
			if aggregated.PerformanceDegradation.DegradationPercent == 0 ||
				result.PerformanceDegradation.DegradationPercent > aggregated.PerformanceDegradation.DegradationPercent {
				aggregated.PerformanceDegradation = result.PerformanceDegradation
			}
		}
	}

	aggregated.TestDurationHours = totalDuration
	aggregated.MemoryLeakDetected = memoryLeakDetected

	return aggregated
}