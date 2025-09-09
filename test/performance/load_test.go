package performance

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// LoadTestSuite provides comprehensive performance and load testing
type LoadTestSuite struct {
	suite.Suite
	testTimeout     time.Duration
	maxConcurrency  int
	testResults     *LoadTestResults
	e2NodePool      []*E2NodeSimulator
	requestPool     chan *PerformanceRequest
	responsePool    chan *PerformanceResponse
	metricsCollector *MetricsCollector
}

// LoadTestResults aggregates all load test results
type LoadTestResults struct {
	StartTime             time.Time
	EndTime               time.Time
	TotalRequests         int64
	SuccessfulRequests    int64
	FailedRequests        int64
	AverageLatency        time.Duration
	P50Latency           time.Duration
	P95Latency           time.Duration
	P99Latency           time.Duration
	MaxLatency           time.Duration
	MinLatency           time.Duration
	ThroughputRPS        float64
	ErrorRate            float64
	ConcurrentConnections int
	ResourceUtilization  *ResourceMetrics
	NetworkMetrics       *NetworkMetrics
	LatencyDistribution  map[string]int64
	ErrorDistribution    map[string]int64
}

// E2NodeSimulator represents a simulated E2 node for load testing
type E2NodeSimulator struct {
	ID           string
	GlobalNodeID string
	Connected    bool
	RequestCount int64
	ErrorCount   int64
	LastActivity time.Time
	Latency      time.Duration
}

// PerformanceRequest represents a performance test request
type PerformanceRequest struct {
	ID          string
	Type        RequestType
	Timestamp   time.Time
	NodeID      string
	Payload     []byte
	ExpectedRTT time.Duration
}

// PerformanceResponse represents a performance test response
type PerformanceResponse struct {
	RequestID   string
	Success     bool
	Latency     time.Duration
	Error       error
	Timestamp   time.Time
	PayloadSize int
}

// RequestType defines different types of performance requests
type RequestType int

const (
	E2SetupRequest RequestType = iota
	RICSubscriptionRequest
	RICIndicationMessage
	RICControlRequest
	PolicyUpdateRequest
	PolicyDeleteRequest
	HealthCheckRequest
)

// ResourceMetrics tracks system resource utilization
type ResourceMetrics struct {
	CPUUsagePercent    float64
	MemoryUsageMB      int64
	NetworkThroughput  float64
	DiskIOPS          float64
	GoroutineCount    int
	HeapSize          int64
	GCPauseTime       time.Duration
}

// NetworkMetrics tracks network-related metrics
type NetworkMetrics struct {
	BytesSent        int64
	BytesReceived    int64
	PacketsSent      int64
	PacketsReceived  int64
	ConnectionsActive int64
	ConnectionsTotal  int64
	RetransmissionRate float64
}

// MetricsCollector collects and aggregates performance metrics
type MetricsCollector struct {
	mutex               sync.RWMutex
	requestMetrics      map[string]*RequestMetrics
	responseMetrics     map[string]*ResponseMetrics
	systemMetrics       *SystemMetrics
	collectionInterval  time.Duration
	isCollecting       bool
	stopChannel        chan struct{}
}

// RequestMetrics tracks metrics for specific request types
type RequestMetrics struct {
	RequestType     RequestType
	TotalCount      int64
	SuccessCount    int64
	ErrorCount      int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	LatencyBuckets  map[time.Duration]int64
}

// ResponseMetrics tracks response-specific metrics
type ResponseMetrics struct {
	ResponseCode    int
	Count           int64
	AverageSize     int64
	TotalSize       int64
	ProcessingTime  time.Duration
}

// SystemMetrics tracks overall system performance
type SystemMetrics struct {
	Timestamp       time.Time
	CPUUtilization  float64
	MemoryUtilization float64
	NetworkUtilization float64
	DiskUtilization   float64
	ActiveConnections int64
}

// SetupSuite initializes the load testing environment
func (suite *LoadTestSuite) SetupSuite() {
	log.Println("Setting up performance load testing environment...")
	
	suite.testTimeout = 60 * time.Minute
	suite.maxConcurrency = 1000
	suite.testResults = &LoadTestResults{
		StartTime:           time.Now(),
		LatencyDistribution: make(map[string]int64),
		ErrorDistribution:   make(map[string]int64),
		ResourceUtilization: &ResourceMetrics{},
		NetworkMetrics:      &NetworkMetrics{},
	}
	
	// Initialize request and response pools for high throughput
	suite.requestPool = make(chan *PerformanceRequest, suite.maxConcurrency)
	suite.responsePool = make(chan *PerformanceResponse, suite.maxConcurrency*2)
	
	// Initialize metrics collector
	suite.metricsCollector = &MetricsCollector{
		requestMetrics:     make(map[string]*RequestMetrics),
		responseMetrics:    make(map[string]*ResponseMetrics),
		systemMetrics:      &SystemMetrics{},
		collectionInterval: 1 * time.Second,
		isCollecting:      false,
		stopChannel:       make(chan struct{}),
	}
	
	// Setup E2 node simulators for load testing
	suite.setupE2NodeSimulators()
	
	log.Println("Performance load testing environment setup completed")
}

// TearDownSuite cleans up the load testing environment
func (suite *LoadTestSuite) TearDownSuite() {
	log.Println("Cleaning up performance load testing environment...")
	
	suite.testResults.EndTime = time.Now()
	
	// Stop metrics collection
	suite.stopMetricsCollection()
	
	// Close channels
	close(suite.requestPool)
	close(suite.responsePool)
	
	// Generate comprehensive performance report
	suite.generatePerformanceReport()
	
	log.Println("Performance load testing environment cleanup completed")
}

// setupE2NodeSimulators creates a pool of E2 node simulators for load testing
func (suite *LoadTestSuite) setupE2NodeSimulators() {
	log.Println("Setting up E2 node simulators for load testing...")
	
	// Create a pool of 100 E2 node simulators
	nodeCount := 100
	suite.e2NodePool = make([]*E2NodeSimulator, nodeCount)
	
	for i := 0; i < nodeCount; i++ {
		simulator := &E2NodeSimulator{
			ID:           fmt.Sprintf("loadtest-node-%03d", i),
			GlobalNodeID: fmt.Sprintf("001-001-%03d", i),
			Connected:    false,
			RequestCount: 0,
			ErrorCount:   0,
			LastActivity: time.Now(),
		}
		
		suite.e2NodePool[i] = simulator
	}
	
	log.Printf("Created %d E2 node simulators for load testing", nodeCount)
}

// startMetricsCollection starts collecting system and application metrics
func (suite *LoadTestSuite) startMetricsCollection() {
	log.Println("Starting metrics collection...")
	
	suite.metricsCollector.isCollecting = true
	
	go func() {
		ticker := time.NewTicker(suite.metricsCollector.collectionInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				suite.collectMetrics()
			case <-suite.metricsCollector.stopChannel:
				return
			}
		}
	}()
}

// stopMetricsCollection stops collecting metrics
func (suite *LoadTestSuite) stopMetricsCollection() {
	if suite.metricsCollector.isCollecting {
		log.Println("Stopping metrics collection...")
		suite.metricsCollector.isCollecting = false
		close(suite.metricsCollector.stopChannel)
	}
}

// collectMetrics collects current system and application metrics
func (suite *LoadTestSuite) collectMetrics() {
	suite.metricsCollector.mutex.Lock()
	defer suite.metricsCollector.mutex.Unlock()
	
	// Collect system metrics (implementation would interface with actual system)
	suite.metricsCollector.systemMetrics = &SystemMetrics{
		Timestamp:         time.Now(),
		CPUUtilization:    suite.getCurrentCPUUtilization(),
		MemoryUtilization: suite.getCurrentMemoryUtilization(),
		NetworkUtilization: suite.getCurrentNetworkUtilization(),
		DiskUtilization:   suite.getCurrentDiskUtilization(),
		ActiveConnections: suite.getActiveConnections(),
	}
	
	// Update test results with latest metrics
	suite.testResults.ResourceUtilization.CPUUsagePercent = suite.metricsCollector.systemMetrics.CPUUtilization
	suite.testResults.ResourceUtilization.MemoryUsageMB = int64(suite.metricsCollector.systemMetrics.MemoryUtilization)
	suite.testResults.ResourceUtilization.NetworkThroughput = suite.metricsCollector.systemMetrics.NetworkUtilization
}

// TestLightLoad tests system performance under light load (25 concurrent connections)
func (suite *LoadTestSuite) TestLightLoad() {
	suite.testResults.TotalRequests++
	
	log.Println("Starting light load test (25 concurrent connections)...")
	
	concurrency := 25
	duration := 2 * time.Minute
	requestsPerSecond := 100
	
	suite.runLoadTest("Light Load Test", concurrency, duration, requestsPerSecond)
	
	// Verify performance requirements for light load
	assert.Less(suite.T(), suite.testResults.AverageLatency, 50*time.Millisecond, 
		"Average latency should be under 50ms for light load")
	assert.Less(suite.T(), suite.testResults.P95Latency, 100*time.Millisecond,
		"P95 latency should be under 100ms for light load")
	assert.Greater(suite.T(), suite.testResults.ThroughputRPS, 
		float64(requestsPerSecond)*0.8, "Throughput should be at least 80% of target RPS")
	assert.Less(suite.T(), suite.testResults.ErrorRate, 0.1,
		"Error rate should be less than 0.1% for light load")
}

// TestMediumLoad tests system performance under medium load (50 concurrent connections)
func (suite *LoadTestSuite) TestMediumLoad() {
	suite.testResults.TotalRequests++
	
	log.Println("Starting medium load test (50 concurrent connections)...")
	
	concurrency := 50
	duration := 5 * time.Minute
	requestsPerSecond := 500
	
	suite.runLoadTest("Medium Load Test", concurrency, duration, requestsPerSecond)
	
	// Verify performance requirements for medium load
	assert.Less(suite.T(), suite.testResults.AverageLatency, 100*time.Millisecond,
		"Average latency should be under 100ms for medium load")
	assert.Less(suite.T(), suite.testResults.P95Latency, 200*time.Millisecond,
		"P95 latency should be under 200ms for medium load")
	assert.Greater(suite.T(), suite.testResults.ThroughputRPS, 
		float64(requestsPerSecond)*0.7, "Throughput should be at least 70% of target RPS")
	assert.Less(suite.T(), suite.testResults.ErrorRate, 0.5,
		"Error rate should be less than 0.5% for medium load")
}

// TestHeavyLoad tests system performance under heavy load (100 concurrent connections)
func (suite *LoadTestSuite) TestHeavyLoad() {
	suite.testResults.TotalRequests++
	
	log.Println("Starting heavy load test (100 concurrent connections)...")
	
	concurrency := 100
	duration := 10 * time.Minute
	requestsPerSecond := 1000
	
	suite.runLoadTest("Heavy Load Test", concurrency, duration, requestsPerSecond)
	
	// Verify performance requirements for heavy load
	assert.Less(suite.T(), suite.testResults.AverageLatency, 200*time.Millisecond,
		"Average latency should be under 200ms for heavy load")
	assert.Less(suite.T(), suite.testResults.P95Latency, 500*time.Millisecond,
		"P95 latency should be under 500ms for heavy load")
	assert.Greater(suite.T(), suite.testResults.ThroughputRPS, 
		float64(requestsPerSecond)*0.6, "Throughput should be at least 60% of target RPS")
	assert.Less(suite.T(), suite.testResults.ErrorRate, 1.0,
		"Error rate should be less than 1% for heavy load")
}

// TestStressLoad tests system performance under stress conditions (200+ concurrent connections)
func (suite *LoadTestSuite) TestStressLoad() {
	suite.testResults.TotalRequests++
	
	log.Println("Starting stress load test (200+ concurrent connections)...")
	
	concurrency := 250
	duration := 15 * time.Minute
	requestsPerSecond := 2000
	
	suite.runLoadTest("Stress Load Test", concurrency, duration, requestsPerSecond)
	
	// Verify system graceful degradation under stress
	assert.Less(suite.T(), suite.testResults.AverageLatency, 1000*time.Millisecond,
		"Average latency should be under 1000ms even under stress")
	assert.Less(suite.T(), suite.testResults.P99Latency, 2000*time.Millisecond,
		"P99 latency should be under 2000ms under stress")
	assert.Greater(suite.T(), suite.testResults.ThroughputRPS, 
		float64(requestsPerSecond)*0.4, "Throughput should be at least 40% of target RPS under stress")
	assert.Less(suite.T(), suite.testResults.ErrorRate, 5.0,
		"Error rate should be less than 5% even under stress")
	
	// Verify system doesn't crash under stress
	assert.Less(suite.T(), suite.testResults.ResourceUtilization.CPUUsagePercent, 95.0,
		"CPU usage should not exceed 95% for extended periods")
	assert.Less(suite.T(), suite.testResults.ResourceUtilization.MemoryUsageMB, 8192,
		"Memory usage should not exceed 8GB under stress")
}

// TestSpikeLoad tests system performance under sudden load spikes
func (suite *LoadTestSuite) TestSpikeLoad() {
	suite.testResults.TotalRequests++
	
	log.Println("Starting spike load test (sudden traffic spikes)...")
	
	// Test normal load followed by sudden spikes
	normalConcurrency := 50
	spikeConcurrency := 500
	normalRPS := 200
	spikeRPS := 5000
	
	// Start with normal load
	suite.runLoadTest("Spike Test - Normal Phase", normalConcurrency, 2*time.Minute, normalRPS)
	baselineLatency := suite.testResults.AverageLatency
	
	// Apply sudden spike
	suite.runLoadTest("Spike Test - Spike Phase", spikeConcurrency, 1*time.Minute, spikeRPS)
	spikeLatency := suite.testResults.AverageLatency
	
	// Return to normal load
	suite.runLoadTest("Spike Test - Recovery Phase", normalConcurrency, 2*time.Minute, normalRPS)
	recoveryLatency := suite.testResults.AverageLatency
	
	// Verify system handles spikes gracefully
	assert.Less(suite.T(), spikeLatency, baselineLatency*5,
		"Spike latency should not exceed 5x baseline latency")
	assert.Less(suite.T(), recoveryLatency, 
		time.Duration(float64(baselineLatency)*1.2), "Recovery latency should return close to baseline")
	assert.Less(suite.T(), suite.testResults.ErrorRate, 2.0,
		"Error rate should remain reasonable during spikes")
}

// TestEnduranceLoad tests system performance over extended periods
func (suite *LoadTestSuite) TestEnduranceLoad() {
	suite.testResults.TotalRequests++
	
	log.Println("Starting endurance load test (extended duration)...")
	
	concurrency := 75
	duration := 30 * time.Minute
	requestsPerSecond := 300
	
	suite.runLoadTest("Endurance Load Test", concurrency, duration, requestsPerSecond)
	
	// Verify system stability over time
	assert.Less(suite.T(), suite.testResults.AverageLatency, 150*time.Millisecond,
		"Average latency should remain stable over extended periods")
	assert.Less(suite.T(), suite.testResults.ErrorRate, 0.5,
		"Error rate should remain low during extended testing")
	
	// Check for memory leaks
	assert.Less(suite.T(), suite.testResults.ResourceUtilization.MemoryUsageMB, 4096,
		"Memory usage should not continuously grow (memory leak check)")
	
	// Check for resource exhaustion
	assert.Less(suite.T(), suite.testResults.ResourceUtilization.GoroutineCount, 10000,
		"Goroutine count should not continuously grow")
}

// runLoadTest executes a load test with specified parameters
func (suite *LoadTestSuite) runLoadTest(testName string, concurrency int, duration time.Duration, requestsPerSecond int) {
	log.Printf("Running %s: concurrency=%d, duration=%v, target_rps=%d", 
		testName, concurrency, duration, requestsPerSecond)
	
	// Start metrics collection
	suite.startMetricsCollection()
	
	// Reset test results for this run
	startTime := time.Now()
	suite.testResults.ConcurrentConnections = concurrency
	
	// Create worker goroutines for load generation
	var wg sync.WaitGroup
	requestInterval := time.Duration(1000/requestsPerSecond) * time.Millisecond
	
	// Create response collector goroutine
	responseCollector := make(chan *PerformanceResponse, concurrency*100)
	go suite.collectResponses(responseCollector)
	
	// Launch load generator workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			suite.loadGeneratorWorker(workerID, duration, requestInterval, responseCollector)
		}(i)
	}
	
	// Wait for all workers to complete
	wg.Wait()
	close(responseCollector)
	
	// Calculate final metrics
	testDuration := time.Since(startTime)
	suite.calculateMetrics(testDuration)
	
	// Stop metrics collection
	suite.stopMetricsCollection()
	
	log.Printf("Completed %s in %v", testName, testDuration)
	suite.logTestSummary()
}

// loadGeneratorWorker generates load for a specific duration
func (suite *LoadTestSuite) loadGeneratorWorker(workerID int, duration time.Duration, 
	requestInterval time.Duration, responseCollector chan<- *PerformanceResponse) {
	
	endTime := time.Now().Add(duration)
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()
	
	requestID := 0
	
	for time.Now().Before(endTime) {
		select {
		case <-ticker.C:
			// Select a random E2 node for this request
			nodeIndex := workerID % len(suite.e2NodePool)
			node := suite.e2NodePool[nodeIndex]
			
			// Create performance request
			request := &PerformanceRequest{
				ID:          fmt.Sprintf("worker-%d-req-%d", workerID, requestID),
				Type:        suite.getRandomRequestType(),
				Timestamp:   time.Now(),
				NodeID:      node.ID,
				Payload:     suite.generateRequestPayload(),
				ExpectedRTT: 100 * time.Millisecond,
			}
			
			// Execute request and measure performance
			response := suite.executeRequest(request, node)
			
			// Send response to collector
			select {
			case responseCollector <- response:
			default:
				// Collector channel full, drop response
			}
			
			requestID++
		}
	}
}

// collectResponses collects and aggregates response metrics
func (suite *LoadTestSuite) collectResponses(responseCollector <-chan *PerformanceResponse) {
	var latencies []time.Duration
	var mutex sync.Mutex
	
	for response := range responseCollector {
		mutex.Lock()
		
		suite.testResults.TotalRequests++
		if response.Success {
			suite.testResults.SuccessfulRequests++
		} else {
			suite.testResults.FailedRequests++
			if response.Error != nil {
				suite.testResults.ErrorDistribution[response.Error.Error()]++
			}
		}
		
		// Collect latency data
		latencies = append(latencies, response.Latency)
		
		// Update latency distribution
		latencyBucket := suite.getLatencyBucket(response.Latency)
		suite.testResults.LatencyDistribution[latencyBucket]++
		
		mutex.Unlock()
	}
	
	// Calculate latency percentiles
	if len(latencies) > 0 {
		suite.calculateLatencyPercentiles(latencies)
	}
}

// executeRequest executes a performance request and measures latency
func (suite *LoadTestSuite) executeRequest(request *PerformanceRequest, node *E2NodeSimulator) *PerformanceResponse {
	startTime := time.Now()
	
	// Simulate request execution (replace with actual implementation)
	success, err := suite.simulateRequestExecution(request, node)
	
	latency := time.Since(startTime)
	
	// Update node metrics
	node.RequestCount++
	node.LastActivity = time.Now()
	node.Latency = latency
	
	if !success {
		node.ErrorCount++
	}
	
	return &PerformanceResponse{
		RequestID:   request.ID,
		Success:     success,
		Latency:     latency,
		Error:       err,
		Timestamp:   time.Now(),
		PayloadSize: len(request.Payload),
	}
}

// simulateRequestExecution simulates the execution of different request types
func (suite *LoadTestSuite) simulateRequestExecution(request *PerformanceRequest, node *E2NodeSimulator) (bool, error) {
	// Simulate processing time based on request type
	var processingTime time.Duration
	
	switch request.Type {
	case E2SetupRequest:
		processingTime = 50*time.Millisecond + time.Duration(len(request.Payload)/1000)*time.Millisecond
	case RICSubscriptionRequest:
		processingTime = 30*time.Millisecond + time.Duration(len(request.Payload)/2000)*time.Millisecond
	case RICIndicationMessage:
		processingTime = 10*time.Millisecond + time.Duration(len(request.Payload)/5000)*time.Millisecond
	case RICControlRequest:
		processingTime = 40*time.Millisecond + time.Duration(len(request.Payload)/1500)*time.Millisecond
	case PolicyUpdateRequest:
		processingTime = 35*time.Millisecond + time.Duration(len(request.Payload)/1200)*time.Millisecond
	case PolicyDeleteRequest:
		processingTime = 25*time.Millisecond + time.Duration(len(request.Payload)/3000)*time.Millisecond
	case HealthCheckRequest:
		processingTime = 5*time.Millisecond + time.Duration(len(request.Payload)/10000)*time.Millisecond
	}
	
	// Add some randomness to simulate real-world variability
	jitter := time.Duration(float64(processingTime) * 0.2 * (suite.getRandomFloat() - 0.5))
	processingTime += jitter
	
	// Simulate actual processing delay
	time.Sleep(processingTime)
	
	// Simulate occasional failures (2% error rate under normal conditions)
	errorRate := 0.02
	if suite.getRandomFloat() < errorRate {
		return false, fmt.Errorf("simulated request failure for %s", request.Type)
	}
	
	return true, nil
}

// Helper methods for load testing

func (suite *LoadTestSuite) getRandomRequestType() RequestType {
	types := []RequestType{
		E2SetupRequest,
		RICSubscriptionRequest,
		RICIndicationMessage,
		RICControlRequest,
		PolicyUpdateRequest,
		PolicyDeleteRequest,
		HealthCheckRequest,
	}
	return types[int(suite.getRandomFloat()*float64(len(types)))]
}

func (suite *LoadTestSuite) generateRequestPayload() []byte {
	// Generate random payload between 100-1000 bytes
	size := 100 + int(suite.getRandomFloat()*900)
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = byte(65 + int(suite.getRandomFloat()*26)) // Random A-Z
	}
	return payload
}

func (suite *LoadTestSuite) getRandomFloat() float64 {
	// Simple pseudo-random number generator for testing
	// In real implementation, use crypto/rand for better randomness
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

func (suite *LoadTestSuite) getLatencyBucket(latency time.Duration) string {
	switch {
	case latency < 10*time.Millisecond:
		return "<10ms"
	case latency < 50*time.Millisecond:
		return "10-50ms"
	case latency < 100*time.Millisecond:
		return "50-100ms"
	case latency < 200*time.Millisecond:
		return "100-200ms"
	case latency < 500*time.Millisecond:
		return "200-500ms"
	case latency < 1000*time.Millisecond:
		return "500ms-1s"
	default:
		return ">1s"
	}
}

func (suite *LoadTestSuite) calculateLatencyPercentiles(latencies []time.Duration) {
	// Sort latencies for percentile calculation
	// Simple bubble sort for demonstration (use proper sorting in production)
	n := len(latencies)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if latencies[j] > latencies[j+1] {
				latencies[j], latencies[j+1] = latencies[j+1], latencies[j]
			}
		}
	}
	
	// Calculate percentiles
	suite.testResults.MinLatency = latencies[0]
	suite.testResults.MaxLatency = latencies[n-1]
	suite.testResults.P50Latency = latencies[n*50/100]
	suite.testResults.P95Latency = latencies[n*95/100]
	suite.testResults.P99Latency = latencies[n*99/100]
	
	// Calculate average latency
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	suite.testResults.AverageLatency = total / time.Duration(n)
}

func (suite *LoadTestSuite) calculateMetrics(testDuration time.Duration) {
	if suite.testResults.TotalRequests > 0 {
		suite.testResults.ThroughputRPS = float64(suite.testResults.SuccessfulRequests) / testDuration.Seconds()
		suite.testResults.ErrorRate = float64(suite.testResults.FailedRequests) / float64(suite.testResults.TotalRequests) * 100
	}
}

func (suite *LoadTestSuite) logTestSummary() {
	log.Printf("Test Summary:")
	log.Printf("  Total Requests: %d", suite.testResults.TotalRequests)
	log.Printf("  Successful Requests: %d", suite.testResults.SuccessfulRequests)
	log.Printf("  Failed Requests: %d", suite.testResults.FailedRequests)
	log.Printf("  Success Rate: %.2f%%", 100-suite.testResults.ErrorRate)
	log.Printf("  Throughput: %.2f RPS", suite.testResults.ThroughputRPS)
	log.Printf("  Average Latency: %v", suite.testResults.AverageLatency)
	log.Printf("  P95 Latency: %v", suite.testResults.P95Latency)
	log.Printf("  P99 Latency: %v", suite.testResults.P99Latency)
}

// Metric collection helper methods

func (suite *LoadTestSuite) getCurrentCPUUtilization() float64 {
	// Mock implementation - replace with actual system metrics
	return 45.0 + suite.getRandomFloat()*20.0
}

func (suite *LoadTestSuite) getCurrentMemoryUtilization() float64 {
	// Mock implementation - replace with actual system metrics
	return 1024 + suite.getRandomFloat()*512
}

func (suite *LoadTestSuite) getCurrentNetworkUtilization() float64 {
	// Mock implementation - replace with actual network metrics
	return 100.0 + suite.getRandomFloat()*200.0
}

func (suite *LoadTestSuite) getCurrentDiskUtilization() float64 {
	// Mock implementation - replace with actual disk metrics
	return 10.0 + suite.getRandomFloat()*30.0
}

func (suite *LoadTestSuite) getActiveConnections() int64 {
	// Mock implementation - replace with actual connection count
	return int64(suite.testResults.ConcurrentConnections + int(suite.getRandomFloat()*10))
}

// generatePerformanceReport generates a comprehensive performance test report
func (suite *LoadTestSuite) generatePerformanceReport() {
	duration := suite.testResults.EndTime.Sub(suite.testResults.StartTime)
	
	report := fmt.Sprintf(`
================================================================
O-RAN SC Performance Load Test Report
================================================================
Test Duration: %v
Total Requests: %d
Successful Requests: %d
Failed Requests: %d
Success Rate: %.2f%%
Error Rate: %.2f%%

Performance Metrics:
- Throughput: %.2f RPS
- Average Latency: %v
- Median (P50) Latency: %v
- P95 Latency: %v
- P99 Latency: %v
- Min Latency: %v
- Max Latency: %v

Resource Utilization:
- CPU Usage: %.2f%%
- Memory Usage: %d MB
- Network Throughput: %.2f MB/s
- Disk IOPS: %.2f
- Active Connections: %d

Latency Distribution:
`, duration,
		suite.testResults.TotalRequests,
		suite.testResults.SuccessfulRequests,
		suite.testResults.FailedRequests,
		(1-suite.testResults.ErrorRate/100)*100,
		suite.testResults.ErrorRate,
		suite.testResults.ThroughputRPS,
		suite.testResults.AverageLatency,
		suite.testResults.P50Latency,
		suite.testResults.P95Latency,
		suite.testResults.P99Latency,
		suite.testResults.MinLatency,
		suite.testResults.MaxLatency,
		suite.testResults.ResourceUtilization.CPUUsagePercent,
		suite.testResults.ResourceUtilization.MemoryUsageMB,
		suite.testResults.ResourceUtilization.NetworkThroughput,
		suite.testResults.ResourceUtilization.DiskIOPS,
		suite.testResults.ConcurrentConnections)
	
	for bucket, count := range suite.testResults.LatencyDistribution {
		report += fmt.Sprintf("- %s: %d requests\n", bucket, count)
	}
	
	if len(suite.testResults.ErrorDistribution) > 0 {
		report += "\nError Distribution:\n"
		for errorType, count := range suite.testResults.ErrorDistribution {
			report += fmt.Sprintf("- %s: %d occurrences\n", errorType, count)
		}
	}
	
	report += "\n================================================================\n"
	
	// Write report to file
	reportFile := fmt.Sprintf("test-results/performance_load_test_report_%s.txt", 
		time.Now().Format("20060102_150405"))
	
	err := os.WriteFile(reportFile, []byte(report), 0644)
	if err != nil {
		log.Printf("Failed to write performance report: %v", err)
	} else {
		log.Printf("Performance load test report written to: %s", reportFile)
	}
	
	// Also log to console
	fmt.Println(report)
}

// String method for RequestType
func (rt RequestType) String() string {
	switch rt {
	case E2SetupRequest:
		return "E2SetupRequest"
	case RICSubscriptionRequest:
		return "RICSubscriptionRequest"
	case RICIndicationMessage:
		return "RICIndicationMessage"
	case RICControlRequest:
		return "RICControlRequest"
	case PolicyUpdateRequest:
		return "PolicyUpdateRequest"
	case PolicyDeleteRequest:
		return "PolicyDeleteRequest"
	case HealthCheckRequest:
		return "HealthCheckRequest"
	default:
		return "UnknownRequest"
	}
}

// TestLoadTestSuite runs the performance load test suite
func TestLoadTestSuite(t *testing.T) {
	suite.Run(t, new(LoadTestSuite))
}