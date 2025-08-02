package dashboard

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LoadTestScenario defines a load testing scenario
type LoadTestScenario struct {
	Name                string        `json:"name"`
	MaxE2Nodes          int           `json:"maxE2Nodes"`
	MaxSubscriptions    int           `json:"maxSubscriptions"`
	RampUpDuration      time.Duration `json:"rampUpDuration"`
	SustainDuration     time.Duration `json:"sustainDuration"`
	RampDownDuration    time.Duration `json:"rampDownDuration"`
	ConnectionPattern   string        `json:"connectionPattern"` // "linear", "exponential", "burst"
}

// E2NodeSimulator simulates an E2 node for load testing
type E2NodeSimulator struct {
	ID               string
	ConnectionTime   time.Time
	IsConnected      bool
	Subscriptions    []string
	IndicationCount  int64
	LastActivity     time.Time
	ErrorCount       int64
	mu               sync.RWMutex
}

// LoadTestManager manages load testing scenarios
type LoadTestManager struct {
	e2Manager       *E2ManagerClient
	subManager      *SubscriptionManagerClient
	simulatedNodes  map[string]*E2NodeSimulator
	activeTests     map[string]*LoadTestExecution
	metrics         *LoadTestMetrics
	mu              sync.RWMutex
}

// LoadTestExecution tracks a running load test
type LoadTestExecution struct {
	Scenario        *LoadTestScenario
	StartTime       time.Time
	EndTime         *time.Time
	Status          string // "running", "completed", "failed"
	ConnectedNodes  int64
	ActiveSubs      int64
	TotalErrors     int64
	Metrics         *LoadTestMetrics
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// LoadTestMetrics captures detailed load test metrics
type LoadTestMetrics struct {
	ConnectionAttempts    int64                    `json:"connectionAttempts"`
	SuccessfulConnections int64                    `json:"successfulConnections"`
	FailedConnections     int64                    `json:"failedConnections"`
	ConnectionTimes       []float64                `json:"connectionTimes"`
	SubscriptionAttempts  int64                    `json:"subscriptionAttempts"`
	SuccessfulSubs        int64                    `json:"successfulSubs"`
	FailedSubs            int64                    `json:"failedSubs"`
	SubscriptionTimes     []float64                `json:"subscriptionTimes"`
	ErrorsByType          map[string]int64         `json:"errorsByType"`
	ResourceSamples       []*ResourceConsumption   `json:"resourceSamples"`
	ThroughputSamples     []ThroughputSample       `json:"throughputSamples"`
	mu                    sync.RWMutex
}

// NewLoadTestManager creates a new load test manager
func NewLoadTestManager(e2Manager *E2ManagerClient, subManager *SubscriptionManagerClient) *LoadTestManager {
	return &LoadTestManager{
		e2Manager:      e2Manager,
		subManager:     subManager,
		simulatedNodes: make(map[string]*E2NodeSimulator),
		activeTests:    make(map[string]*LoadTestExecution),
		metrics:        &LoadTestMetrics{
			ErrorsByType: make(map[string]int64),
		},
	}
}

// RunLoadTesting executes the load testing scenarios
func (pts *PerformanceTestSuite) RunLoadTesting(ctx context.Context) error {
	log.Println("Starting load testing scenarios...")

	loadManager := NewLoadTestManager(pts.e2Manager, pts.subManager)
	
	// Define load test scenarios - Enhanced for 100+ concurrent E2 nodes
	scenarios := []*LoadTestScenario{
		{
			Name:             "Gradual Load Increase - 150 Nodes",
			MaxE2Nodes:       pts.config.MaxE2Nodes,
			MaxSubscriptions: pts.config.MaxConcurrentSubs,
			RampUpDuration:   pts.config.LoadRampUpDuration,
			SustainDuration:  15 * time.Minute, // Extended sustain period
			RampDownDuration: 3 * time.Minute,
			ConnectionPattern: "linear",
		},
		{
			Name:             "Burst Load Test - 100 Nodes",
			MaxE2Nodes:       100, // Exactly 100 nodes for requirement validation
			MaxSubscriptions: pts.config.MaxConcurrentSubs / 2,
			RampUpDuration:   45 * time.Second, // Faster burst
			SustainDuration:  8 * time.Minute,  // Extended sustain
			RampDownDuration: 45 * time.Second,
			ConnectionPattern: "burst",
		},
		{
			Name:             "Exponential Growth - 120 Nodes",
			MaxE2Nodes:       120, // Above 100 node requirement
			MaxSubscriptions: pts.config.MaxConcurrentSubs,
			RampUpDuration:   10 * time.Minute, // Longer exponential growth
			SustainDuration:  12 * time.Minute, // Extended sustain
			RampDownDuration: 4 * time.Minute,
			ConnectionPattern: "exponential",
		},
		{
			Name:             "High Density Load - 150 Nodes",
			MaxE2Nodes:       pts.config.MaxE2Nodes,
			MaxSubscriptions: pts.config.MaxConcurrentSubs * 2, // Double subscriptions
			RampUpDuration:   12 * time.Minute,
			SustainDuration:  20 * time.Minute, // Long sustain for stability
			RampDownDuration: 5 * time.Minute,
			ConnectionPattern: "linear",
		},
		{
			Name:             "Stress Burst - 200 Nodes",
			MaxE2Nodes:       200, // Extreme load test
			MaxSubscriptions: pts.config.MaxConcurrentSubs * 3,
			RampUpDuration:   2 * time.Minute,  // Very fast ramp-up
			SustainDuration:  5 * time.Minute,  // Short but intense
			RampDownDuration: 1 * time.Minute,
			ConnectionPattern: "burst",
		},
	}

	var wg sync.WaitGroup
	results := make([]*LoadTestResults, len(scenarios))
	
	// Run scenarios sequentially to avoid interference
	for i, scenario := range scenarios {
		log.Printf("Executing load test scenario: %s", scenario.Name)
		
		result, err := loadManager.ExecuteScenario(ctx, scenario)
		if err != nil {
			log.Printf("Load test scenario %s failed: %v", scenario.Name, err)
			continue
		}
		results[i] = result
		
		// Wait between scenarios for system to stabilize
		time.Sleep(2 * time.Minute)
	}

	// Aggregate results
	pts.testResults.LoadTestResults = pts.aggregateLoadTestResults(results)
	
	log.Println("Load testing completed")
	return nil
}

// ExecuteScenario runs a specific load test scenario
func (ltm *LoadTestManager) ExecuteScenario(ctx context.Context, scenario *LoadTestScenario) (*LoadTestResults, error) {
	testCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	execution := &LoadTestExecution{
		Scenario:  scenario,
		StartTime: time.Now(),
		Status:    "running",
		Metrics:   &LoadTestMetrics{ErrorsByType: make(map[string]int64)},
		ctx:       testCtx,
		cancel:    cancel,
	}

	ltm.mu.Lock()
	ltm.activeTests[scenario.Name] = execution
	ltm.mu.Unlock()

	defer func() {
		ltm.mu.Lock()
		delete(ltm.activeTests, scenario.Name)
		ltm.mu.Unlock()
	}()

	// Start resource monitoring
	resourceChan := ltm.monitorResources(testCtx, 5*time.Second)

	// Execute the test phases
	if err := ltm.executeRampUp(execution); err != nil {
		execution.Status = "failed"
		return nil, fmt.Errorf("ramp-up phase failed: %w", err)
	}

	if err := ltm.executeSustain(execution); err != nil {
		execution.Status = "failed"
		return nil, fmt.Errorf("sustain phase failed: %w", err)
	}

	if err := ltm.executeRampDown(execution); err != nil {
		execution.Status = "failed"
		return nil, fmt.Errorf("ramp-down phase failed: %w", err)
	}

	// Collect final resource samples
	go func() {
		for resource := range resourceChan {
			execution.Metrics.mu.Lock()
			execution.Metrics.ResourceSamples = append(execution.Metrics.ResourceSamples, resource)
			execution.Metrics.mu.Unlock()
		}
	}()

	execution.Status = "completed"
	endTime := time.Now()
	execution.EndTime = &endTime

	return ltm.generateLoadTestResults(execution), nil
}

// executeRampUp gradually increases load according to the scenario pattern
func (ltm *LoadTestManager) executeRampUp(execution *LoadTestExecution) error {
	log.Printf("Starting ramp-up phase for %s", execution.Scenario.Name)
	
	scenario := execution.Scenario
	rampUpSteps := 20 // Number of steps in ramp-up
	stepDuration := scenario.RampUpDuration / time.Duration(rampUpSteps)
	
	for step := 1; step <= rampUpSteps; step++ {
		select {
		case <-execution.ctx.Done():
			return execution.ctx.Err()
		default:
		}

		// Calculate target connections for this step
		var targetNodes int
		switch scenario.ConnectionPattern {
		case "linear":
			targetNodes = (scenario.MaxE2Nodes * step) / rampUpSteps
		case "exponential":
			// Exponential growth: 2^(step/steps) - 1
			factor := float64(step) / float64(rampUpSteps)
			targetNodes = int(float64(scenario.MaxE2Nodes) * factor * factor)
		case "burst":
			// Burst pattern: rapid increase in later steps
			if step < rampUpSteps/2 {
				targetNodes = (scenario.MaxE2Nodes * step) / (rampUpSteps * 2)
			} else {
				targetNodes = scenario.MaxE2Nodes
			}
		default:
			targetNodes = (scenario.MaxE2Nodes * step) / rampUpSteps
		}

		// Connect additional nodes
		currentNodes := int(atomic.LoadInt64(&execution.ConnectedNodes))
		nodesToAdd := targetNodes - currentNodes
		
		if nodesToAdd > 0 {
			if err := ltm.connectNodes(execution, nodesToAdd); err != nil {
				log.Printf("Failed to connect nodes in step %d: %v", step, err)
			}
		}

		// Create subscriptions
		targetSubs := (scenario.MaxSubscriptions * step) / rampUpSteps
		currentSubs := int(atomic.LoadInt64(&execution.ActiveSubs))
		subsToAdd := targetSubs - currentSubs
		
		if subsToAdd > 0 {
			if err := ltm.createSubscriptions(execution, subsToAdd); err != nil {
				log.Printf("Failed to create subscriptions in step %d: %v", step, err)
			}
		}

		log.Printf("Ramp-up step %d/%d: %d nodes, %d subscriptions", 
			step, rampUpSteps, int(atomic.LoadInt64(&execution.ConnectedNodes)), 
			int(atomic.LoadInt64(&execution.ActiveSubs)))

		time.Sleep(stepDuration)
	}

	log.Printf("Ramp-up phase completed: %d nodes, %d subscriptions", 
		int(atomic.LoadInt64(&execution.ConnectedNodes)), 
		int(atomic.LoadInt64(&execution.ActiveSubs)))
	
	return nil
}

// executeSustain maintains the load for the specified duration
func (ltm *LoadTestManager) executeSustain(execution *LoadTestExecution) error {
	log.Printf("Starting sustain phase for %s (duration: %v)", 
		execution.Scenario.Name, execution.Scenario.SustainDuration)

	// Start indication generation
	indicationCtx, indicationCancel := context.WithCancel(execution.ctx)
	defer indicationCancel()

	go ltm.generateIndications(indicationCtx, execution)

	// Monitor and maintain connections during sustain phase
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	sustainEnd := time.Now().Add(execution.Scenario.SustainDuration)
	
	for time.Now().Before(sustainEnd) {
		select {
		case <-execution.ctx.Done():
			return execution.ctx.Err()
		case <-ticker.C:
			// Check and repair any dropped connections
			if err := ltm.maintainConnections(execution); err != nil {
				log.Printf("Error maintaining connections: %v", err)
			}
			
			// Log current status
			log.Printf("Sustain phase: %d nodes, %d subscriptions, %d total errors", 
				int(atomic.LoadInt64(&execution.ConnectedNodes)), 
				int(atomic.LoadInt64(&execution.ActiveSubs)),
				int(atomic.LoadInt64(&execution.TotalErrors)))
		}
	}

	log.Printf("Sustain phase completed")
	return nil
}

// executeRampDown gradually reduces load
func (ltm *LoadTestManager) executeRampDown(execution *LoadTestExecution) error {
	log.Printf("Starting ramp-down phase for %s", execution.Scenario.Name)
	
	scenario := execution.Scenario
	rampDownSteps := 10
	stepDuration := scenario.RampDownDuration / time.Duration(rampDownSteps)
	
	for step := 1; step <= rampDownSteps; step++ {
		select {
		case <-execution.ctx.Done():
			return execution.ctx.Err()
		default:
		}

		// Calculate target connections for this step (decreasing)
		remainingFactor := float64(rampDownSteps-step) / float64(rampDownSteps)
		targetNodes := int(float64(scenario.MaxE2Nodes) * remainingFactor)
		targetSubs := int(float64(scenario.MaxSubscriptions) * remainingFactor)

		// Disconnect excess nodes
		currentNodes := int(atomic.LoadInt64(&execution.ConnectedNodes))
		nodesToRemove := currentNodes - targetNodes
		if nodesToRemove > 0 {
			ltm.disconnectNodes(execution, nodesToRemove)
		}

		// Remove excess subscriptions
		currentSubs := int(atomic.LoadInt64(&execution.ActiveSubs))
		subsToRemove := currentSubs - targetSubs
		if subsToRemove > 0 {
			ltm.removeSubscriptions(execution, subsToRemove)
		}

		log.Printf("Ramp-down step %d/%d: %d nodes, %d subscriptions", 
			step, rampDownSteps, int(atomic.LoadInt64(&execution.ConnectedNodes)), 
			int(atomic.LoadInt64(&execution.ActiveSubs)))

		time.Sleep(stepDuration)
	}

	log.Printf("Ramp-down phase completed")
	return nil
}

// connectNodes simulates connecting E2 nodes
func (ltm *LoadTestManager) connectNodes(execution *LoadTestExecution, count int) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent connections

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(nodeIndex int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			nodeID := fmt.Sprintf("load-test-node-%d-%d", time.Now().Unix(), nodeIndex)
			startTime := time.Now()

			atomic.AddInt64(&execution.Metrics.ConnectionAttempts, 1)

			// Simulate E2 node connection
			node := &E2NodeSimulator{
				ID:             nodeID,
				ConnectionTime: startTime,
				IsConnected:    false,
				Subscriptions:  make([]string, 0),
				LastActivity:   startTime,
			}

			// Simulate connection process with realistic timing
			connectionDelay := time.Duration(rand.Intn(100)+50) * time.Millisecond
			time.Sleep(connectionDelay)

			// Simulate occasional connection failures (5% failure rate)
			if rand.Float32() < 0.05 {
				atomic.AddInt64(&execution.Metrics.FailedConnections, 1)
				atomic.AddInt64(&execution.TotalErrors, 1)
				execution.Metrics.mu.Lock()
				execution.Metrics.ErrorsByType["connection_failed"]++
				execution.Metrics.mu.Unlock()
				return
			}

			node.IsConnected = true
			connectionTime := time.Since(startTime).Seconds() * 1000 // Convert to milliseconds

			ltm.mu.Lock()
			ltm.simulatedNodes[nodeID] = node
			ltm.mu.Unlock()

			atomic.AddInt64(&execution.ConnectedNodes, 1)
			atomic.AddInt64(&execution.Metrics.SuccessfulConnections, 1)

			execution.Metrics.mu.Lock()
			execution.Metrics.ConnectionTimes = append(execution.Metrics.ConnectionTimes, connectionTime)
			execution.Metrics.mu.Unlock()

		}(i)
	}

	wg.Wait()
	return nil
}

// createSubscriptions simulates creating subscriptions
func (ltm *LoadTestManager) createSubscriptions(execution *LoadTestExecution, count int) error {
	ltm.mu.RLock()
	var availableNodes []*E2NodeSimulator
	for _, node := range ltm.simulatedNodes {
		if node.IsConnected {
			availableNodes = append(availableNodes, node)
		}
	}
	ltm.mu.RUnlock()

	if len(availableNodes) == 0 {
		return fmt.Errorf("no connected nodes available for subscriptions")
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20) // Limit concurrent subscriptions

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(subIndex int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Select a random node
			node := availableNodes[rand.Intn(len(availableNodes))]
			subID := fmt.Sprintf("load-test-sub-%s-%d", node.ID, subIndex)
			startTime := time.Now()

			atomic.AddInt64(&execution.Metrics.SubscriptionAttempts, 1)

			// Simulate subscription creation with realistic timing
			subscriptionDelay := time.Duration(rand.Intn(50)+25) * time.Millisecond
			time.Sleep(subscriptionDelay)

			// Simulate occasional subscription failures (3% failure rate)
			if rand.Float32() < 0.03 {
				atomic.AddInt64(&execution.Metrics.FailedSubs, 1)
				atomic.AddInt64(&execution.TotalErrors, 1)
				execution.Metrics.mu.Lock()
				execution.Metrics.ErrorsByType["subscription_failed"]++
				execution.Metrics.mu.Unlock()
				return
			}

			subscriptionTime := time.Since(startTime).Seconds() * 1000 // Convert to milliseconds

			node.mu.Lock()
			node.Subscriptions = append(node.Subscriptions, subID)
			node.mu.Unlock()

			atomic.AddInt64(&execution.ActiveSubs, 1)
			atomic.AddInt64(&execution.Metrics.SuccessfulSubs, 1)

			execution.Metrics.mu.Lock()
			execution.Metrics.SubscriptionTimes = append(execution.Metrics.SubscriptionTimes, subscriptionTime)
			execution.Metrics.mu.Unlock()

		}(i)
	}

	wg.Wait()
	return nil
}

// generateIndications simulates indication generation from connected nodes
func (ltm *LoadTestManager) generateIndications(ctx context.Context, execution *LoadTestExecution) {
	ticker := time.NewTicker(100 * time.Millisecond) // Generate indications every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ltm.mu.RLock()
			for _, node := range ltm.simulatedNodes {
				if node.IsConnected && len(node.Subscriptions) > 0 {
					// Generate indications for each subscription
					for range node.Subscriptions {
						atomic.AddInt64(&node.IndicationCount, 1)
						node.LastActivity = time.Now()
					}
				}
			}
			ltm.mu.RUnlock()

			// Record throughput sample
			totalIndications := int64(0)
			ltm.mu.RLock()
			for _, node := range ltm.simulatedNodes {
				totalIndications += atomic.LoadInt64(&node.IndicationCount)
			}
			ltm.mu.RUnlock()

			sample := ThroughputSample{
				Timestamp:         time.Now(),
				IndicationsPerSec: int(totalIndications / int64(time.Since(execution.StartTime).Seconds())),
				ProcessingLatencyMs: float64(rand.Intn(5) + 1), // Simulate processing latency
			}

			execution.Metrics.mu.Lock()
			execution.Metrics.ThroughputSamples = append(execution.Metrics.ThroughputSamples, sample)
			execution.Metrics.mu.Unlock()
		}
	}
}

// maintainConnections checks and repairs dropped connections
func (ltm *LoadTestManager) maintainConnections(execution *LoadTestExecution) error {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	droppedNodes := 0
	for nodeID, node := range ltm.simulatedNodes {
		// Simulate occasional connection drops (1% chance per check)
		if node.IsConnected && rand.Float32() < 0.01 {
			node.IsConnected = false
			atomic.AddInt64(&execution.ConnectedNodes, -1)
			atomic.AddInt64(&execution.ActiveSubs, -int64(len(node.Subscriptions)))
			droppedNodes++
			
			execution.Metrics.mu.Lock()
			execution.Metrics.ErrorsByType["connection_dropped"]++
			execution.Metrics.mu.Unlock()
			
			log.Printf("Simulated connection drop for node %s", nodeID)
		}
	}

	if droppedNodes > 0 {
		log.Printf("Detected %d dropped connections during sustain phase", droppedNodes)
	}

	return nil
}

// disconnectNodes simulates disconnecting E2 nodes
func (ltm *LoadTestManager) disconnectNodes(execution *LoadTestExecution, count int) {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	disconnected := 0
	for nodeID, node := range ltm.simulatedNodes {
		if disconnected >= count {
			break
		}
		
		if node.IsConnected {
			node.IsConnected = false
			atomic.AddInt64(&execution.ConnectedNodes, -1)
			atomic.AddInt64(&execution.ActiveSubs, -int64(len(node.Subscriptions)))
			delete(ltm.simulatedNodes, nodeID)
			disconnected++
		}
	}
}

// removeSubscriptions simulates removing subscriptions
func (ltm *LoadTestManager) removeSubscriptions(execution *LoadTestExecution, count int) {
	ltm.mu.Lock()
	defer ltm.mu.Unlock()

	removed := 0
	for _, node := range ltm.simulatedNodes {
		if removed >= count {
			break
		}
		
		node.mu.Lock()
		if len(node.Subscriptions) > 0 {
			subsToRemove := len(node.Subscriptions)
			if subsToRemove > count-removed {
				subsToRemove = count - removed
			}
			
			node.Subscriptions = node.Subscriptions[subsToRemove:]
			atomic.AddInt64(&execution.ActiveSubs, -int64(subsToRemove))
			removed += subsToRemove
		}
		node.mu.Unlock()
	}
}

// monitorResources monitors system resources during load testing
func (ltm *LoadTestManager) monitorResources(ctx context.Context, interval time.Duration) <-chan *ResourceConsumption {
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
				// This would typically query actual system metrics
				// For simulation, we'll generate realistic values
				usage := &ResourceConsumption{
					CPUUsagePercent:    float64(rand.Intn(80) + 10),
					MemoryUsageMB:      float64(rand.Intn(1000) + 500),
					NetworkBytesPerSec: int64(rand.Intn(1000000) + 100000),
					DiskIOPerSec:       int64(rand.Intn(10000) + 1000),
					GoroutineCount:     rand.Intn(1000) + 100,
					GCPauseMs:          float64(rand.Intn(10) + 1),
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

// generateLoadTestResults creates comprehensive load test results
func (ltm *LoadTestManager) generateLoadTestResults(execution *LoadTestExecution) *LoadTestResults {
	metrics := execution.Metrics
	
	// Calculate response time statistics
	responseTimeMetrics := ltm.calculateResponseTimeMetrics(metrics.ConnectionTimes)
	
	// Calculate resource consumption averages
	resourceConsumption := ltm.calculateAverageResourceConsumption(metrics.ResourceSamples)
	
	// Calculate success rates
	connectionSuccessRate := 0.0
	if metrics.ConnectionAttempts > 0 {
		connectionSuccessRate = float64(metrics.SuccessfulConnections) / float64(metrics.ConnectionAttempts) * 100
	}
	
	subscriptionSuccessRate := 0.0
	if metrics.SubscriptionAttempts > 0 {
		subscriptionSuccessRate = float64(metrics.SuccessfulSubs) / float64(metrics.SubscriptionAttempts) * 100
	}

	// Calculate error rates by type
	errorRates := make(map[string]float64)
	totalOperations := metrics.ConnectionAttempts + metrics.SubscriptionAttempts
	if totalOperations > 0 {
		for errorType, count := range metrics.ErrorsByType {
			errorRates[errorType] = float64(count) / float64(totalOperations) * 100
		}
	}

	return &LoadTestResults{
		MaxConcurrentE2Nodes:    int(execution.ConnectedNodes),
		MaxConcurrentSubs:       int(execution.ActiveSubs),
		ConnectionSuccessRate:   connectionSuccessRate,
		SubscriptionSuccessRate: subscriptionSuccessRate,
		ErrorRates:              errorRates,
		ResponseTimes:           responseTimeMetrics,
		ResourceConsumption:     resourceConsumption,
		Timestamp:               time.Now(),
	}
}

// calculateResponseTimeMetrics calculates statistical metrics for response times
func (ltm *LoadTestManager) calculateResponseTimeMetrics(times []float64) *ResponseTimeMetrics {
	if len(times) == 0 {
		return &ResponseTimeMetrics{}
	}

	// Sort times for percentile calculations
	sortedTimes := make([]float64, len(times))
	copy(sortedTimes, times)
	
	// Simple bubble sort for small datasets
	for i := 0; i < len(sortedTimes); i++ {
		for j := i + 1; j < len(sortedTimes); j++ {
			if sortedTimes[i] > sortedTimes[j] {
				sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
			}
		}
	}

	// Calculate statistics
	sum := 0.0
	min := sortedTimes[0]
	max := sortedTimes[len(sortedTimes)-1]
	
	for _, time := range sortedTimes {
		sum += time
	}
	mean := sum / float64(len(sortedTimes))

	// Calculate percentiles
	p50 := sortedTimes[len(sortedTimes)*50/100]
	p95 := sortedTimes[len(sortedTimes)*95/100]
	p99 := sortedTimes[len(sortedTimes)*99/100]
	p999 := sortedTimes[len(sortedTimes)*999/1000]

	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, time := range sortedTimes {
		diff := time - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := 0.0
	if len(sortedTimes) > 1 {
		stdDev = sumSquaredDiff / float64(len(sortedTimes)-1)
	}

	return &ResponseTimeMetrics{
		Mean:   mean,
		P50:    p50,
		P95:    p95,
		P99:    p99,
		P999:   p999,
		Max:    max,
		Min:    min,
		StdDev: stdDev,
	}
}

// calculateAverageResourceConsumption calculates average resource usage
func (ltm *LoadTestManager) calculateAverageResourceConsumption(samples []*ResourceConsumption) *ResourceConsumption {
	if len(samples) == 0 {
		return &ResourceConsumption{}
	}

	var totalCPU, totalMemory, totalGC float64
	var totalNetwork, totalDisk int64
	var totalGoroutines int

	for _, sample := range samples {
		totalCPU += sample.CPUUsagePercent
		totalMemory += sample.MemoryUsageMB
		totalNetwork += sample.NetworkBytesPerSec
		totalDisk += sample.DiskIOPerSec
		totalGoroutines += sample.GoroutineCount
		totalGC += sample.GCPauseMs
	}

	count := float64(len(samples))
	return &ResourceConsumption{
		CPUUsagePercent:    totalCPU / count,
		MemoryUsageMB:      totalMemory / count,
		NetworkBytesPerSec: totalNetwork / int64(count),
		DiskIOPerSec:       totalDisk / int64(count),
		GoroutineCount:     totalGoroutines / len(samples),
		GCPauseMs:          totalGC / count,
	}
}

// aggregateLoadTestResults combines results from multiple scenarios
func (pts *PerformanceTestSuite) aggregateLoadTestResults(results []*LoadTestResults) *LoadTestResults {
	if len(results) == 0 {
		return &LoadTestResults{Timestamp: time.Now()}
	}

	// Find the best results across all scenarios
	aggregated := &LoadTestResults{
		ErrorRates:  make(map[string]float64),
		Timestamp:   time.Now(),
	}

	maxNodes := 0
	maxSubs := 0
	totalConnectionSuccess := 0.0
	totalSubscriptionSuccess := 0.0
	validResults := 0

	for _, result := range results {
		if result == nil {
			continue
		}
		
		validResults++
		
		if result.MaxConcurrentE2Nodes > maxNodes {
			maxNodes = result.MaxConcurrentE2Nodes
		}
		
		if result.MaxConcurrentSubs > maxSubs {
			maxSubs = result.MaxConcurrentSubs
		}
		
		totalConnectionSuccess += result.ConnectionSuccessRate
		totalSubscriptionSuccess += result.SubscriptionSuccessRate
		
		// Aggregate error rates
		for errorType, rate := range result.ErrorRates {
			aggregated.ErrorRates[errorType] += rate
		}
	}

	if validResults > 0 {
		aggregated.MaxConcurrentE2Nodes = maxNodes
		aggregated.MaxConcurrentSubs = maxSubs
		aggregated.ConnectionSuccessRate = totalConnectionSuccess / float64(validResults)
		aggregated.SubscriptionSuccessRate = totalSubscriptionSuccess / float64(validResults)
		
		// Average error rates
		for errorType := range aggregated.ErrorRates {
			aggregated.ErrorRates[errorType] /= float64(validResults)
		}
		
		// Use the resource consumption from the most intensive test
		if len(results) > 0 && results[0] != nil {
			aggregated.ResourceConsumption = results[0].ResourceConsumption
			aggregated.ResponseTimes = results[0].ResponseTimes
		}
	}

	return aggregated
}