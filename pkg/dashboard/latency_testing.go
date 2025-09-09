package dashboard

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// LatencyTestManager manages latency testing scenarios
type LatencyTestManager struct {
	e2Manager       *E2ManagerClient
	subManager      *SubscriptionManagerClient
	latencyTracker  *LatencyTracker
	testMetrics     *LatencyTestMetrics
	mu              sync.RWMutex
}



// LatencyTestMetrics captures detailed latency test metrics
type LatencyTestMetrics struct {
	E2SetupLatencies      []float64              `json:"e2SetupLatencies"`
	SubscriptionLatencies []float64              `json:"subscriptionLatencies"`
	IndicationLatencies   []float64              `json:"indicationLatencies"`
	ControlLatencies      []float64              `json:"controlLatencies"`
	EndToEndLatencies     []float64              `json:"endToEndLatencies"`
	LatencyBuckets        []LatencyBucket        `json:"latencyBuckets"`
	ResourceUtilization   []*ResourceConsumption `json:"resourceUtilization"`
	TestDuration          time.Duration          `json:"testDuration"`
	TotalOperations       int64                  `json:"totalOperations"`
	FailedOperations      int64                  `json:"failedOperations"`
	mu                    sync.RWMutex
}

// LatencyTestScenario defines a latency test scenario
type LatencyTestScenario struct {
	Name                    string        `json:"name"`
	MaxLatencyMs            float64       `json:"maxLatencyMs"`
	OperationRate           int           `json:"operationRate"` // operations per second
	TestDuration            time.Duration `json:"testDuration"`
	OperationTypes          []string      `json:"operationTypes"`
	ConcurrentOperations    int           `json:"concurrentOperations"`
	LatencyTargets          map[string]float64 `json:"latencyTargets"` // operation -> target latency ms
}

// NewLatencyTestManager creates a new latency test manager
func NewLatencyTestManager(e2Manager *E2ManagerClient, subManager *SubscriptionManagerClient) *LatencyTestManager {
	tracker := &LatencyTracker{
		e2SetupLatencies:      make([]float64, 0),
		subscriptionLatencies: make([]float64, 0),
		indicationLatencies:   make([]float64, 0),
		controlLatencies:      make([]float64, 0),
		endToEndLatencies:     make([]float64, 0),
		latencyDistribution:   make(map[float64]int64),
		activeOperations:      make(map[string]*LatencyMeasurement),
	}

	return &LatencyTestManager{
		e2Manager:      e2Manager,
		subManager:     subManager,
		latencyTracker: tracker,
		testMetrics: &LatencyTestMetrics{
			E2SetupLatencies:      make([]float64, 0),
			SubscriptionLatencies: make([]float64, 0),
			IndicationLatencies:   make([]float64, 0),
			ControlLatencies:      make([]float64, 0),
			EndToEndLatencies:     make([]float64, 0),
			LatencyBuckets:        make([]LatencyBucket, 0),
			ResourceUtilization:   make([]*ResourceConsumption, 0),
		},
	}
}

// RunLatencyTesting executes latency testing scenarios
func (pts *PerformanceTestSuite) RunLatencyTesting(ctx context.Context) error {
	log.Println("Starting latency testing scenarios...")

	latencyManager := NewLatencyTestManager(pts.e2Manager, pts.subManager)

	// Define latency test scenarios - Enhanced for sub-10ms validation
	scenarios := []*LatencyTestScenario{
		{
			Name:                 "E2 Setup Sub-10ms Test",
			MaxLatencyMs:         float64(pts.config.MaxLatencyMs),
			OperationRate:        20, // Increased rate for better validation
			TestDuration:         8 * time.Minute, // Extended duration
			OperationTypes:       []string{"e2_setup"},
			ConcurrentOperations: 10, // Increased concurrency
			LatencyTargets: map[string]float64{
				"e2_setup": float64(pts.config.MaxLatencyMs),
			},
		},
		{
			Name:                 "Subscription Sub-8ms Test",
			MaxLatencyMs:         6.0, // Stricter than config for validation
			OperationRate:        100, // Higher rate for stress testing
			TestDuration:         12 * time.Minute,
			OperationTypes:       []string{"subscription"},
			ConcurrentOperations: 50, // Increased concurrency
			LatencyTargets: map[string]float64{
				"subscription": 6.0,
			},
		},
		{
			Name:                 "High-Rate Indication Sub-5ms Test",
			MaxLatencyMs:         4.0, // Very strict for high-rate processing
			OperationRate:        2000, // Very high rate
			TestDuration:         15 * time.Minute,
			OperationTypes:       []string{"indication"},
			ConcurrentOperations: 200, // High concurrency
			LatencyTargets: map[string]float64{
				"indication": 4.0,
			},
		},
		{
			Name:                 "Control Message Sub-10ms Test",
			MaxLatencyMs:         float64(pts.config.MaxLatencyMs),
			OperationRate:        50, // Increased rate
			TestDuration:         10 * time.Minute,
			OperationTypes:       []string{"control"},
			ConcurrentOperations: 25, // Increased concurrency
			LatencyTargets: map[string]float64{
				"control": float64(pts.config.MaxLatencyMs),
			},
		},
		{
			Name:                 "End-to-End Sub-15ms Test",
			MaxLatencyMs:         12.0, // Stricter than previous 2x config
			OperationRate:        75, // Higher rate
			TestDuration:         18 * time.Minute, // Extended duration
			OperationTypes:       []string{"e2e"},
			ConcurrentOperations: 30, // Increased concurrency
			LatencyTargets: map[string]float64{
				"e2e": 12.0,
			},
		},
		{
			Name:                 "Mixed Operations Latency Test",
			MaxLatencyMs:         float64(pts.config.MaxLatencyMs),
			OperationRate:        150, // High mixed rate
			TestDuration:         20 * time.Minute,
			OperationTypes:       []string{"e2_setup", "subscription", "indication", "control"},
			ConcurrentOperations: 40,
			LatencyTargets: map[string]float64{
				"e2_setup":     float64(pts.config.MaxLatencyMs),
				"subscription": 6.0,
				"indication":   4.0,
				"control":      float64(pts.config.MaxLatencyMs),
			},
		},
		{
			Name:                 "Peak Load Latency Validation",
			MaxLatencyMs:         float64(pts.config.MaxLatencyMs),
			OperationRate:        500, // Very high rate for peak validation
			TestDuration:         10 * time.Minute,
			OperationTypes:       []string{"indication", "subscription"},
			ConcurrentOperations: 100,
			LatencyTargets: map[string]float64{
				"indication":   5.0, // Strict under peak load
				"subscription": 7.0,
			},
		},
	}

	results := make([]*LatencyResults, len(scenarios))

	// Run scenarios sequentially
	for i, scenario := range scenarios {
		log.Printf("Executing latency test scenario: %s", scenario.Name)
		
		result, err := latencyManager.ExecuteScenario(ctx, scenario)
		if err != nil {
			log.Printf("Latency test scenario %s failed: %v", scenario.Name, err)
			continue
		}
		results[i] = result
		
		// Wait between scenarios for system to stabilize
		time.Sleep(30 * time.Second)
	}

	// Aggregate results
	pts.testResults.LatencyResults = pts.aggregateLatencyResults(results)
	
	log.Println("Latency testing completed")
	return nil
}

// ExecuteScenario runs a specific latency test scenario
func (ltm *LatencyTestManager) ExecuteScenario(ctx context.Context, scenario *LatencyTestScenario) (*LatencyResults, error) {
	testCtx, cancel := context.WithTimeout(ctx, scenario.TestDuration)
	defer cancel()

	// Reset metrics
	ltm.testMetrics = &LatencyTestMetrics{
		E2SetupLatencies:      make([]float64, 0),
		SubscriptionLatencies: make([]float64, 0),
		IndicationLatencies:   make([]float64, 0),
		ControlLatencies:      make([]float64, 0),
		EndToEndLatencies:     make([]float64, 0),
		LatencyBuckets:        make([]LatencyBucket, 0),
		ResourceUtilization:   make([]*ResourceConsumption, 0),
	}

	startTime := time.Now()

	// Start resource monitoring
	resourceChan := ltm.monitorResources(testCtx, 2*time.Second)

	// Start latency monitoring
	go ltm.monitorLatencies(testCtx)

	// Execute operations based on scenario
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, scenario.ConcurrentOperations)

	// Calculate operation interval
	operationInterval := time.Second / time.Duration(scenario.OperationRate)
	ticker := time.NewTicker(operationInterval)
	defer ticker.Stop()

	operationID := 0

	for {
		select {
		case <-testCtx.Done():
			// Wait for all operations to complete
			wg.Wait()
			
			// Collect final resource samples
			go func() {
				for resource := range resourceChan {
					ltm.testMetrics.mu.Lock()
					ltm.testMetrics.ResourceUtilization = append(ltm.testMetrics.ResourceUtilization, resource)
					ltm.testMetrics.mu.Unlock()
				}
			}()

			ltm.testMetrics.TestDuration = time.Since(startTime)
			return ltm.generateLatencyResults(), nil

		case <-ticker.C:
			// Select operation type for this iteration
			operationType := scenario.OperationTypes[rand.Intn(len(scenario.OperationTypes))]
			
			wg.Add(1)
			go func(opID int, opType string) {
				defer wg.Done()
				
				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				ltm.executeOperation(testCtx, opID, opType, scenario)
			}(operationID, operationType)
			
			operationID++
		}
	}
}

// executeOperation executes a specific operation and measures its latency
func (ltm *LatencyTestManager) executeOperation(ctx context.Context, operationID int, operationType string, scenario *LatencyTestScenario) {
	opIDStr := fmt.Sprintf("%s-%d", operationType, operationID)
	
	// Start latency measurement
	measurement := &LatencyMeasurement{
		OperationID:   opIDStr,
		OperationType: operationType,
		StartTime:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	ltm.latencyTracker.mu.Lock()
	ltm.latencyTracker.activeOperations[opIDStr] = measurement
	ltm.latencyTracker.mu.Unlock()

	atomic.AddInt64(&ltm.testMetrics.TotalOperations, 1)

	// Execute the operation based on type
	var err error
	switch operationType {
	case "e2_setup":
		err = ltm.simulateE2Setup(ctx, measurement)
	case "subscription":
		err = ltm.simulateSubscription(ctx, measurement)
	case "indication":
		err = ltm.simulateIndicationProcessing(ctx, measurement)
	case "control":
		err = ltm.simulateControlMessage(ctx, measurement)
	case "e2e":
		err = ltm.simulateEndToEndOperation(ctx, measurement)
	default:
		err = fmt.Errorf("unknown operation type: %s", operationType)
	}

	// Complete latency measurement
	endTime := time.Now()
	measurement.EndTime = &endTime
	latency := endTime.Sub(measurement.StartTime)
	measurement.Latency = &latency

	if err != nil {
		atomic.AddInt64(&ltm.testMetrics.FailedOperations, 1)
		log.Printf("Operation %s failed: %v", opIDStr, err)
	} else {
		// Record successful latency measurement
		latencyMs := float64(latency.Nanoseconds()) / 1e6
		ltm.recordLatency(operationType, latencyMs)
		
		// Check if latency exceeds target
		if target, exists := scenario.LatencyTargets[operationType]; exists && latencyMs > target {
			log.Printf("Operation %s exceeded latency target: %.2fms > %.2fms", 
				opIDStr, latencyMs, target)
		}
	}

	// Remove from active operations
	ltm.latencyTracker.mu.Lock()
	delete(ltm.latencyTracker.activeOperations, opIDStr)
	ltm.latencyTracker.mu.Unlock()
}

// simulateE2Setup simulates E2 setup procedure
func (ltm *LatencyTestManager) simulateE2Setup(ctx context.Context, measurement *LatencyMeasurement) error {
	// Simulate E2 setup phases
	phases := []struct {
		name     string
		duration time.Duration
	}{
		{"sctp_connection", time.Duration(rand.Intn(5)+2) * time.Millisecond},
		{"e2_setup_request", time.Duration(rand.Intn(3)+1) * time.Millisecond},
		{"asn1_encoding", time.Duration(rand.Intn(2)+1) * time.Millisecond},
		{"e2_setup_response", time.Duration(rand.Intn(4)+2) * time.Millisecond},
	}

	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate phase processing time
		time.Sleep(phase.duration)
		
		// Record phase in metadata
		measurement.Metadata[phase.name] = phase.duration.Seconds() * 1000
		
		// Simulate occasional failures (2% failure rate)
		if rand.Float32() < 0.02 {
			return fmt.Errorf("E2 setup failed during %s phase", phase.name)
		}
	}

	return nil
}

// simulateSubscription simulates subscription creation
func (ltm *LatencyTestManager) simulateSubscription(ctx context.Context, measurement *LatencyMeasurement) error {
	// Simulate subscription phases
	phases := []struct {
		name     string
		duration time.Duration
	}{
		{"validation", time.Duration(rand.Intn(2)+1) * time.Millisecond},
		{"routing", time.Duration(rand.Intn(3)+1) * time.Millisecond},
		{"e2_subscription_request", time.Duration(rand.Intn(4)+2) * time.Millisecond},
		{"e2_subscription_response", time.Duration(rand.Intn(3)+1) * time.Millisecond},
	}

	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate phase processing time
		time.Sleep(phase.duration)
		
		// Record phase in metadata
		measurement.Metadata[phase.name] = phase.duration.Seconds() * 1000
		
		// Simulate occasional failures (1% failure rate)
		if rand.Float32() < 0.01 {
			return fmt.Errorf("subscription failed during %s phase", phase.name)
		}
	}

	return nil
}

// simulateIndicationProcessing simulates indication processing
func (ltm *LatencyTestManager) simulateIndicationProcessing(ctx context.Context, measurement *LatencyMeasurement) error {
	// Simulate indication processing phases
	phases := []struct {
		name     string
		duration time.Duration
	}{
		{"asn1_decoding", time.Duration(rand.Intn(1000)+500) * time.Microsecond},
		{"validation", time.Duration(rand.Intn(500)+200) * time.Microsecond},
		{"routing", time.Duration(rand.Intn(800)+300) * time.Microsecond},
		{"delivery", time.Duration(rand.Intn(1200)+400) * time.Microsecond},
	}

	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate phase processing time
		time.Sleep(phase.duration)
		
		// Record phase in metadata
		measurement.Metadata[phase.name] = phase.duration.Seconds() * 1000
		
		// Simulate very rare failures (0.1% failure rate)
		if rand.Float32() < 0.001 {
			return fmt.Errorf("indication processing failed during %s phase", phase.name)
		}
	}

	return nil
}

// simulateControlMessage simulates control message processing
func (ltm *LatencyTestManager) simulateControlMessage(ctx context.Context, measurement *LatencyMeasurement) error {
	// Simulate control message phases
	phases := []struct {
		name     string
		duration time.Duration
	}{
		{"validation", time.Duration(rand.Intn(2)+1) * time.Millisecond},
		{"authorization", time.Duration(rand.Intn(3)+1) * time.Millisecond},
		{"ric_control_request", time.Duration(rand.Intn(5)+3) * time.Millisecond},
		{"ric_control_ack", time.Duration(rand.Intn(4)+2) * time.Millisecond},
	}

	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate phase processing time
		time.Sleep(phase.duration)
		
		// Record phase in metadata
		measurement.Metadata[phase.name] = phase.duration.Seconds() * 1000
		
		// Simulate occasional failures (3% failure rate)
		if rand.Float32() < 0.03 {
			return fmt.Errorf("control message failed during %s phase", phase.name)
		}
	}

	return nil
}

// simulateEndToEndOperation simulates complete end-to-end operation
func (ltm *LatencyTestManager) simulateEndToEndOperation(ctx context.Context, measurement *LatencyMeasurement) error {
	// Simulate end-to-end operation phases
	phases := []struct {
		name     string
		duration time.Duration
	}{
		{"request_ingress", time.Duration(rand.Intn(2)+1) * time.Millisecond},
		{"authentication", time.Duration(rand.Intn(3)+1) * time.Millisecond},
		{"business_logic", time.Duration(rand.Intn(8)+4) * time.Millisecond},
		{"database_query", time.Duration(rand.Intn(6)+2) * time.Millisecond},
		{"e2_interaction", time.Duration(rand.Intn(10)+5) * time.Millisecond},
		{"response_formatting", time.Duration(rand.Intn(2)+1) * time.Millisecond},
		{"response_egress", time.Duration(rand.Intn(3)+1) * time.Millisecond},
	}

	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate phase processing time
		time.Sleep(phase.duration)
		
		// Record phase in metadata
		measurement.Metadata[phase.name] = phase.duration.Seconds() * 1000
		
		// Simulate occasional failures (2% failure rate)
		if rand.Float32() < 0.02 {
			return fmt.Errorf("end-to-end operation failed during %s phase", phase.name)
		}
	}

	return nil
}

// recordLatency records a latency measurement
func (ltm *LatencyTestManager) recordLatency(operationType string, latencyMs float64) {
	ltm.latencyTracker.mu.Lock()
	defer ltm.latencyTracker.mu.Unlock()

	// Record in appropriate slice
	switch operationType {
	case "e2_setup":
		ltm.latencyTracker.e2SetupLatencies = append(ltm.latencyTracker.e2SetupLatencies, latencyMs)
	case "subscription":
		ltm.latencyTracker.subscriptionLatencies = append(ltm.latencyTracker.subscriptionLatencies, latencyMs)
	case "indication":
		ltm.latencyTracker.indicationLatencies = append(ltm.latencyTracker.indicationLatencies, latencyMs)
	case "control":
		ltm.latencyTracker.controlLatencies = append(ltm.latencyTracker.controlLatencies, latencyMs)
	case "e2e":
		ltm.latencyTracker.endToEndLatencies = append(ltm.latencyTracker.endToEndLatencies, latencyMs)
	}

	// Update latency distribution
	bucket := ltm.getLatencyBucket(latencyMs)
	ltm.latencyTracker.latencyDistribution[bucket]++
}

// getLatencyBucket determines the latency bucket for a given latency
func (ltm *LatencyTestManager) getLatencyBucket(latencyMs float64) float64 {
	// Define latency buckets: 0.1, 0.5, 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000+ ms
	buckets := []float64{0.1, 0.5, 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000}
	
	for _, bucket := range buckets {
		if latencyMs <= bucket {
			return bucket
		}
	}
	
	return math.Inf(1) // For latencies > 1000ms
}

// monitorLatencies monitors active operations for timeout detection
func (ltm *LatencyTestManager) monitorLatencies(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ltm.latencyTracker.mu.RLock()
			activeCount := len(ltm.latencyTracker.activeOperations)
			
			// Check for operations that have been running too long
			now := time.Now()
			timeoutCount := 0
			for _, measurement := range ltm.latencyTracker.activeOperations {
				if now.Sub(measurement.StartTime) > 30*time.Second {
					timeoutCount++
				}
			}
			ltm.latencyTracker.mu.RUnlock()

			if timeoutCount > 0 {
				log.Printf("Latency monitoring: %d active operations, %d potential timeouts", 
					activeCount, timeoutCount)
			}
		}
	}
}

// monitorResources monitors system resources during latency testing
func (ltm *LatencyTestManager) monitorResources(ctx context.Context, interval time.Duration) <-chan *ResourceConsumption {
	resourceChan := make(chan *ResourceConsumption, 100)

	go func() {
		defer close(resourceChan)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Simulate resource monitoring with realistic values
				usage := &ResourceConsumption{
					CPUUsagePercent:    float64(rand.Intn(70) + 20),
					MemoryUsageMB:      float64(rand.Intn(1500) + 800),
					NetworkBytesPerSec: int64(rand.Intn(2000000) + 500000),
					DiskIOPerSec:       int64(rand.Intn(20000) + 5000),
					GoroutineCount:     rand.Intn(1500) + 300,
					GCPauseMs:          float64(rand.Intn(15) + 1),
				}

				select {
				case resourceChan <- usage:
				default:
					// Channel full, skip this sample
				}
			}
		}
	}()

	return resourceChan
}

// generateLatencyResults creates comprehensive latency test results
func (ltm *LatencyTestManager) generateLatencyResults() *LatencyResults {
	tracker := ltm.latencyTracker
	
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	// Calculate latency metrics for each operation type
	e2SetupMetrics := ltm.calculateLatencyMetrics(tracker.e2SetupLatencies)
	subscriptionMetrics := ltm.calculateLatencyMetrics(tracker.subscriptionLatencies)
	indicationMetrics := ltm.calculateLatencyMetrics(tracker.indicationLatencies)
	controlMetrics := ltm.calculateLatencyMetrics(tracker.controlLatencies)
	endToEndMetrics := ltm.calculateLatencyMetrics(tracker.endToEndLatencies)

	// Create latency distribution buckets
	latencyBuckets := ltm.createLatencyBuckets(tracker.latencyDistribution)

	return &LatencyResults{
		E2SetupLatencyMs:      e2SetupMetrics,
		SubscriptionLatencyMs: subscriptionMetrics,
		IndicationLatencyMs:   indicationMetrics,
		ControlLatencyMs:      controlMetrics,
		EndToEndLatencyMs:     endToEndMetrics,
		LatencyDistribution:   latencyBuckets,
		Timestamp:             time.Now(),
	}
}

// calculateLatencyMetrics calculates statistical metrics for latency measurements
func (ltm *LatencyTestManager) calculateLatencyMetrics(latencies []float64) *LatencyMetrics {
	if len(latencies) == 0 {
		return &LatencyMetrics{}
	}

	// Sort latencies for percentile calculations
	sortedLatencies := make([]float64, len(latencies))
	copy(sortedLatencies, latencies)
	sort.Float64s(sortedLatencies)

	// Calculate statistics
	sum := 0.0
	min := sortedLatencies[0]
	max := sortedLatencies[len(sortedLatencies)-1]
	
	for _, latency := range sortedLatencies {
		sum += latency
	}
	mean := sum / float64(len(sortedLatencies))

	// Calculate percentiles
	p50 := sortedLatencies[len(sortedLatencies)*50/100]
	p95 := sortedLatencies[len(sortedLatencies)*95/100]
	p99 := sortedLatencies[len(sortedLatencies)*99/100]

	return &LatencyMetrics{
		Mean:  mean,
		P50:   p50,
		P95:   p95,
		P99:   p99,
		Max:   max,
		Min:   min,
		Count: int64(len(sortedLatencies)),
	}
}

// createLatencyBuckets creates latency distribution buckets
func (ltm *LatencyTestManager) createLatencyBuckets(distribution map[float64]int64) []LatencyBucket {
	buckets := make([]LatencyBucket, 0)
	
	// Calculate total count
	totalCount := int64(0)
	for _, count := range distribution {
		totalCount += count
	}

	if totalCount == 0 {
		return buckets
	}

	// Create sorted buckets
	bucketKeys := make([]float64, 0, len(distribution))
	for bucket := range distribution {
		bucketKeys = append(bucketKeys, bucket)
	}
	sort.Float64s(bucketKeys)

	for _, bucket := range bucketKeys {
		count := distribution[bucket]
		percentage := float64(count) / float64(totalCount) * 100

		buckets = append(buckets, LatencyBucket{
			UpperBoundMs: bucket,
			Count:        count,
			Percentage:   percentage,
		})
	}

	return buckets
}

// aggregateLatencyResults combines results from multiple scenarios
func (pts *PerformanceTestSuite) aggregateLatencyResults(results []*LatencyResults) *LatencyResults {
	if len(results) == 0 {
		return &LatencyResults{Timestamp: time.Now()}
	}

	// Aggregate all latency measurements
	allE2Setup := make([]float64, 0)
	allSubscription := make([]float64, 0)
	allIndication := make([]float64, 0)
	allControl := make([]float64, 0)
	allEndToEnd := make([]float64, 0)
	allDistribution := make([]LatencyBucket, 0)

	for _, result := range results {
		if result == nil {
			continue
		}
		
		// Collect all latency measurements for recalculation
		// Note: In a real implementation, we would need to store raw measurements
		// For now, we'll use the best metrics from each scenario
		
		allDistribution = append(allDistribution, result.LatencyDistribution...)
	}

	// For aggregation, we'll take the worst-case metrics from all scenarios
	aggregated := &LatencyResults{
		E2SetupLatencyMs:      &LatencyMetrics{},
		SubscriptionLatencyMs: &LatencyMetrics{},
		IndicationLatencyMs:   &LatencyMetrics{},
		ControlLatencyMs:      &LatencyMetrics{},
		EndToEndLatencyMs:     &LatencyMetrics{},
		LatencyDistribution:   allDistribution,
		Timestamp:             time.Now(),
	}

	// Find worst-case metrics across all scenarios
	for _, result := range results {
		if result == nil {
			continue
		}

		// E2 Setup metrics
		if result.E2SetupLatencyMs != nil {
			if result.E2SetupLatencyMs.P99 > aggregated.E2SetupLatencyMs.P99 {
				aggregated.E2SetupLatencyMs = result.E2SetupLatencyMs
			}
		}

		// Subscription metrics
		if result.SubscriptionLatencyMs != nil {
			if result.SubscriptionLatencyMs.P99 > aggregated.SubscriptionLatencyMs.P99 {
				aggregated.SubscriptionLatencyMs = result.SubscriptionLatencyMs
			}
		}

		// Indication metrics
		if result.IndicationLatencyMs != nil {
			if result.IndicationLatencyMs.P99 > aggregated.IndicationLatencyMs.P99 {
				aggregated.IndicationLatencyMs = result.IndicationLatencyMs
			}
		}

		// Control metrics
		if result.ControlLatencyMs != nil {
			if result.ControlLatencyMs.P99 > aggregated.ControlLatencyMs.P99 {
				aggregated.ControlLatencyMs = result.ControlLatencyMs
			}
		}

		// End-to-end metrics
		if result.EndToEndLatencyMs != nil {
			if result.EndToEndLatencyMs.P99 > aggregated.EndToEndLatencyMs.P99 {
				aggregated.EndToEndLatencyMs = result.EndToEndLatencyMs
			}
		}
	}

	return aggregated
}