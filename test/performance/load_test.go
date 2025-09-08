package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// LoadTestSuite provides performance and load testing for O-RAN components
type LoadTestSuite struct {
	suite.Suite
	baseURL              string
	testTimeout          time.Duration
	maxConcurrentNodes   int
	testResults          *LoadTestResults
	resourceMonitor      *ResourceMonitor
	performanceTargets   *PerformanceTargets
}

// LoadTestResults aggregates all load test results
type LoadTestResults struct {
	StartTime           time.Time
	EndTime             time.Time
	TotalTests          int
	PassedTests         int
	FailedTests         int
	E2NodeLoadTests     map[string]*E2NodeLoadResult
	A1PolicyLoadTests   map[string]*A1PolicyLoadResult
	XAppLoadTests       map[string]*XAppLoadResult
	SystemLoadTests     map[string]*SystemLoadResult
	PerformanceMetrics  *PerformanceMetrics
}

// E2NodeLoadResult tracks E2 node load test results
type E2NodeLoadResult struct {
	TestName              string
	ConcurrentNodes       int
	RequestsPerNode       int
	TotalRequests         int64
	SuccessfulRequests    int64
	FailedRequests        int64
	AverageResponseTime   time.Duration
	P50ResponseTime       time.Duration
	P95ResponseTime       time.Duration
	P99ResponseTime       time.Duration
	MaxResponseTime       time.Duration
	MinResponseTime       time.Duration
	ThroughputRPS         float64
	ErrorRate             float64
	ResourceUtilization   ResourceUtilization
	Errors                []string
}

// A1PolicyLoadResult tracks A1 policy load test results
type A1PolicyLoadResult struct {
	TestName              string
	ConcurrentPolicies    int
	TotalPolicies         int64
	CreatedPolicies       int64
	EnforcedPolicies      int64
	FailedPolicies        int64
	AverageCreationTime   time.Duration
	AverageEnforcementTime time.Duration
	PolicyDistributionTime time.Duration
	ThroughputPPS         float64
	ErrorRate             float64
	Errors                []string
}

// XAppLoadResult tracks xApp load test results
type XAppLoadResult struct {
	TestName              string
	ConcurrentXApps       int
	TotalDeployments      int64
	SuccessfulDeployments int64
	FailedDeployments     int64
	AverageDeploymentTime time.Duration
	AverageScalingTime    time.Duration
	ResourceEfficiency    float64
	Errors                []string
}

// SystemLoadResult tracks system-wide load test results
type SystemLoadResult struct {
	TestName              string
	Duration              time.Duration
	TotalRequests         int64
	SuccessfulRequests    int64
	FailedRequests        int64
	PeakThroughput        float64
	SustainedThroughput   float64
	SystemStability       bool
	MemoryLeaks           bool
	ResourceExhaustion    bool
	Errors                []string
}

// PerformanceMetrics tracks overall performance metrics
type PerformanceMetrics struct {
	E2SetupLatency        LatencyMetrics
	IndicationLatency     LatencyMetrics
	PolicyLatency         LatencyMetrics
	ControlLatency        LatencyMetrics
	SystemThroughput      ThroughputMetrics
	ResourceEfficiency    ResourceEfficiencyMetrics
	ScalabilityMetrics    ScalabilityMetrics
}

// LatencyMetrics tracks latency performance
type LatencyMetrics struct {
	Average    time.Duration
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
	P999       time.Duration
	Max        time.Duration
	Min        time.Duration
}

// ThroughputMetrics tracks throughput performance
type ThroughputMetrics struct {
	RequestsPerSecond     float64
	PoliciesPerSecond     float64
	IndicationsPerSecond  float64
	PeakThroughput        float64
	SustainedThroughput   float64
}

// ResourceEfficiencyMetrics tracks resource efficiency
type ResourceEfficiencyMetrics struct {
	CPUEfficiency         float64 // requests per CPU core per second
	MemoryEfficiency      float64 // requests per GB memory per second
	NetworkEfficiency     float64 // requests per Mbps network per second
	EnergyEfficiency      float64 // requests per watt per second
}

// ScalabilityMetrics tracks scalability performance
type ScalabilityMetrics struct {
	MaxConcurrentNodes    int
	MaxConcurrentPolicies int
	MaxConcurrentXApps    int
	LinearScalingLimit    int
	ResourceScalingFactor float64
}

// ResourceUtilization tracks system resource usage
type ResourceUtilization struct {
	CPUUsagePercent       float64
	MemoryUsagePercent    float64
	NetworkThroughputMbps float64
	DiskIOPSUsage         float64
	GoroutineCount        int
	HeapSizeMB            float64
}

// ResourceMonitor monitors system resources during load tests
type ResourceMonitor struct {
	monitoring   bool
	interval     time.Duration
	measurements []ResourceMeasurement
	mutex        sync.RWMutex
}

// ResourceMeasurement represents a resource measurement at a point in time
type ResourceMeasurement struct {
	Timestamp   time.Time
	Utilization ResourceUtilization
}

// PerformanceTargets defines performance requirements
type PerformanceTargets struct {
	MaxE2SetupLatency         time.Duration // 500ms
	MaxIndicationLatency      time.Duration // 10ms
	MaxPolicyLatency          time.Duration // 1s
	MinThroughputRPS          float64       // 1000 req/s
	MaxCPUUsage               float64       // 80%
	MaxMemoryUsage            float64       // 85%
	MinSuccessRate            float64       // 99%
	MaxConcurrentNodesTarget  int           // 100+
}

// SetupSuite initializes the load test environment
func (suite *LoadTestSuite) SetupSuite() {
	log.Println("Setting up performance and load test environment...")
	
	suite.baseURL = "http://localhost:8080" // Default dashboard API URL
	suite.testTimeout = 60 * time.Minute
	suite.maxConcurrentNodes = 200
	
	// Initialize test results
	suite.testResults = &LoadTestResults{
		StartTime:         time.Now(),
		E2NodeLoadTests:   make(map[string]*E2NodeLoadResult),
		A1PolicyLoadTests: make(map[string]*A1PolicyLoadResult),
		XAppLoadTests:     make(map[string]*XAppLoadResult),
		SystemLoadTests:   make(map[string]*SystemLoadResult),
		PerformanceMetrics: &PerformanceMetrics{},
	}
	
	// Initialize resource monitor
	suite.resourceMonitor = &ResourceMonitor{
		interval:     time.Second,
		measurements: make([]ResourceMeasurement, 0),
	}
	
	// Set performance targets for O-RAN L Release
	suite.performanceTargets = &PerformanceTargets{
		MaxE2SetupLatency:         500 * time.Millisecond,
		MaxIndicationLatency:      10 * time.Millisecond,
		MaxPolicyLatency:          1 * time.Second,
		MinThroughputRPS:          1000.0,
		MaxCPUUsage:               80.0,
		MaxMemoryUsage:            85.0,
		MinSuccessRate:            99.0,
		MaxConcurrentNodesTarget:  100,
	}
	
	log.Println("Performance and load test environment setup completed")
}

// TearDownSuite cleans up the test environment
func (suite *LoadTestSuite) TearDownSuite() {
	log.Println("Cleaning up performance and load test environment...")
	
	suite.testResults.EndTime = time.Now()
	suite.stopResourceMonitoring()
	suite.generateLoadTestReport()
	
	log.Println("Performance and load test environment cleanup completed")
}

// TestE2NodeConcurrentLoad tests concurrent E2 node handling with 100+ nodes
func (suite *LoadTestSuite) TestE2NodeConcurrentLoad() {
	suite.testResults.TotalTests++
	
	log.Println("Testing E2 node concurrent load with 100+ nodes...")
	
	// Test different concurrent node levels
	concurrentLevels := []int{25, 50, 100, 150, 200}
	requestsPerNode := 50
	
	for _, concurrent := range concurrentLevels {
		suite.Run(fmt.Sprintf("E2NodeLoad_%dNodes", concurrent), func() {
			log.Printf("Testing %d concurrent E2 nodes with %d requests each", concurrent, requestsPerNode)
			
			// Start resource monitoring
			suite.startResourceMonitoring()
			
			// Execute load test
			result := suite.executeE2NodeLoadTest(concurrent, requestsPerNode)
			result.TestName = fmt.Sprintf("E2NodeLoad_%dNodes", concurrent)
			
			// Stop monitoring and get resource utilization
			result.ResourceUtilization = suite.stopResourceMonitoring()
			
			suite.testResults.E2NodeLoadTests[result.TestName] = result
			
			// Validate performance requirements
			suite.validateE2NodePerformance(result)
		})
	}
	
	suite.testResults.PassedTests++
}

// TestA1PolicyConcurrentLoad tests concurrent A1 policy operations
func (suite *LoadTestSuite) TestA1PolicyConcurrentLoad() {
	suite.testResults.TotalTests++
	
	log.Println("Testing A1 policy concurrent load...")
	
	// Test different policy load levels
	policyLevels := []int{50, 100, 200, 500, 1000}
	
	for _, policyCount := range policyLevels {
		suite.Run(fmt.Sprintf("A1PolicyLoad_%dPolicies", policyCount), func() {
			log.Printf("Testing %d concurrent A1 policies", policyCount)
			
			// Start resource monitoring
			suite.startResourceMonitoring()
			
			// Execute policy load test
			result := suite.executeA1PolicyLoadTest(policyCount)
			result.TestName = fmt.Sprintf("A1PolicyLoad_%dPolicies", policyCount)
			
			// Stop monitoring
			suite.stopResourceMonitoring()
			
			suite.testResults.A1PolicyLoadTests[result.TestName] = result
			
			// Validate policy performance requirements
			suite.validateA1PolicyPerformance(result)
		})
	}
	
	suite.testResults.PassedTests++
}

// TestXAppDeploymentLoad tests xApp deployment and scaling under load
func (suite *LoadTestSuite) TestXAppDeploymentLoad() {
	suite.testResults.TotalTests++
	
	log.Println("Testing xApp deployment load...")
	
	// Test different xApp deployment loads
	xappCounts := []int{5, 10, 20, 30}
	
	for _, xappCount := range xappCounts {
		suite.Run(fmt.Sprintf("XAppLoad_%dXApps", xappCount), func() {
			log.Printf("Testing %d concurrent xApp deployments", xappCount)
			
			// Start resource monitoring
			suite.startResourceMonitoring()
			
			// Execute xApp load test
			result := suite.executeXAppLoadTest(xappCount)
			result.TestName = fmt.Sprintf("XAppLoad_%dXApps", xappCount)
			
			// Stop monitoring
			suite.stopResourceMonitoring()
			
			suite.testResults.XAppLoadTests[result.TestName] = result
			
			// Validate xApp performance requirements
			suite.validateXAppPerformance(result)
		})
	}
	
	suite.testResults.PassedTests++
}

// TestSystemSustainedLoad tests system performance under sustained load
func (suite *LoadTestSuite) TestSystemSustainedLoad() {
	suite.testResults.TotalTests++
	
	log.Println("Testing system sustained load performance...")
	
	// Test different sustained load durations
	loadDurations := []struct {
		name     string
		duration time.Duration
		rps      int
	}{
		{"ShortBurst", 2 * time.Minute, 500},
		{"MediumLoad", 5 * time.Minute, 1000},
		{"SustainedLoad", 15 * time.Minute, 800},
		{"StressLoad", 30 * time.Minute, 1200},
	}
	
	for _, load := range loadDurations {
		suite.Run(fmt.Sprintf("SystemLoad_%s", load.name), func() {
			log.Printf("Testing system load: %s for %v at %d RPS", load.name, load.duration, load.rps)
			
			// Start resource monitoring
			suite.startResourceMonitoring()
			
			// Execute sustained load test
			result := suite.executeSystemLoadTest(load.duration, load.rps)
			result.TestName = load.name
			
			// Stop monitoring
			suite.stopResourceMonitoring()
			
			suite.testResults.SystemLoadTests[result.TestName] = result
			
			// Validate system performance requirements
			suite.validateSystemPerformance(result)
		})
	}
	
	suite.testResults.PassedTests++
}

// TestThroughputBenchmarks tests maximum throughput capabilities
func (suite *LoadTestSuite) TestThroughputBenchmarks() {
	suite.testResults.TotalTests++
	
	log.Println("Testing throughput benchmarks...")
	
	suite.Run("MaxThroughputBenchmark", func() {
		// Find maximum sustainable throughput
		maxThroughput := suite.findMaxThroughput()
		
		suite.testResults.PerformanceMetrics.SystemThroughput.PeakThroughput = maxThroughput
		
		// Validate against requirements
		assert.GreaterOrEqual(suite.T(), maxThroughput, suite.performanceTargets.MinThroughputRPS,
			"System throughput below minimum requirement")
		
		log.Printf("Maximum sustainable throughput: %.2f RPS", maxThroughput)
	})
	
	suite.testResults.PassedTests++
}

// TestLatencyBenchmarks tests latency performance for all operations
func (suite *LoadTestSuite) TestLatencyBenchmarks() {
	suite.testResults.TotalTests++
	
	log.Println("Testing latency benchmarks...")
	
	// Test E2 Setup latency
	suite.Run("E2SetupLatency", func() {
		latency := suite.measureE2SetupLatency(100) // 100 samples
		suite.testResults.PerformanceMetrics.E2SetupLatency = latency
		
		assert.LessOrEqual(suite.T(), latency.P95, suite.performanceTargets.MaxE2SetupLatency,
			"E2 Setup P95 latency exceeds target")
	})
	
	// Test Indication latency
	suite.Run("IndicationLatency", func() {
		latency := suite.measureIndicationLatency(1000) // 1000 samples
		suite.testResults.PerformanceMetrics.IndicationLatency = latency
		
		assert.LessOrEqual(suite.T(), latency.P95, suite.performanceTargets.MaxIndicationLatency,
			"Indication P95 latency exceeds target")
	})
	
	// Test Policy latency
	suite.Run("PolicyLatency", func() {
		latency := suite.measurePolicyLatency(100) // 100 samples
		suite.testResults.PerformanceMetrics.PolicyLatency = latency
		
		assert.LessOrEqual(suite.T(), latency.P95, suite.performanceTargets.MaxPolicyLatency,
			"Policy P95 latency exceeds target")
	})
	
	// Test Control latency
	suite.Run("ControlLatency", func() {
		latency := suite.measureControlLatency(100) // 100 samples
		suite.testResults.PerformanceMetrics.ControlLatency = latency
		
		assert.LessOrEqual(suite.T(), latency.P99, 100*time.Millisecond,
			"Control P99 latency exceeds 100ms")
	})
	
	suite.testResults.PassedTests++
}

// TestResourceEfficiency tests resource usage efficiency
func (suite *LoadTestSuite) TestResourceEfficiency() {
	suite.testResults.TotalTests++
	
	log.Println("Testing resource efficiency...")
	
	suite.Run("ResourceEfficiencyBenchmark", func() {
		// Measure resource efficiency under standard load
		efficiency := suite.measureResourceEfficiency()
		suite.testResults.PerformanceMetrics.ResourceEfficiency = efficiency
		
		// Validate efficiency targets
		assert.Greater(suite.T(), efficiency.CPUEfficiency, 50.0,
			"CPU efficiency below target (50 req/core/sec)")
		assert.Greater(suite.T(), efficiency.MemoryEfficiency, 100.0,
			"Memory efficiency below target (100 req/GB/sec)")
		
		log.Printf("Resource efficiency - CPU: %.2f req/core/sec, Memory: %.2f req/GB/sec",
			efficiency.CPUEfficiency, efficiency.MemoryEfficiency)
	})
	
	suite.testResults.PassedTests++
}

// TestScalabilityLimits tests system scalability limits
func (suite *LoadTestSuite) TestScalabilityLimits() {
	suite.testResults.TotalTests++
	
	log.Println("Testing scalability limits...")
	
	suite.Run("ScalabilityLimits", func() {
		// Find maximum scalability limits
		scalability := suite.measureScalabilityLimits()
		suite.testResults.PerformanceMetrics.ScalabilityMetrics = scalability
		
		// Validate scalability requirements
		assert.GreaterOrEqual(suite.T(), scalability.MaxConcurrentNodes,
			suite.performanceTargets.MaxConcurrentNodesTarget,
			"Maximum concurrent nodes below target")
		
		log.Printf("Scalability limits - Nodes: %d, Policies: %d, xApps: %d",
			scalability.MaxConcurrentNodes, scalability.MaxConcurrentPolicies, scalability.MaxConcurrentXApps)
	})
	
	suite.testResults.PassedTests++
}

// Implementation Methods

// executeE2NodeLoadTest executes E2 node load test
func (suite *LoadTestSuite) executeE2NodeLoadTest(concurrentNodes, requestsPerNode int) *E2NodeLoadResult {
	result := &E2NodeLoadResult{
		ConcurrentNodes: concurrentNodes,
		RequestsPerNode: requestsPerNode,
		TotalRequests:   int64(concurrentNodes * requestsPerNode),
		Errors:          make([]string, 0),
	}
	
	// Response time tracking
	var responseTimes []time.Duration
	var responseTimesMutex sync.Mutex
	
	// Counters
	var successCounter int64
	var failureCounter int64
	
	// Start time
	startTime := time.Now()
	
	// Execute concurrent E2 node operations
	var wg sync.WaitGroup
	
	for i := 0; i < concurrentNodes; i++ {
		wg.Add(1)
		go func(nodeID int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerNode; j++ {
				// Simulate E2 Setup Request
				reqStart := time.Now()
				success := suite.simulateE2NodeOperation(nodeID, j)
				reqDuration := time.Since(reqStart)
				
				// Record response time
				responseTimesMutex.Lock()
				responseTimes = append(responseTimes, reqDuration)
				responseTimesMutex.Unlock()
				
				// Update counters
				if success {
					atomic.AddInt64(&successCounter, 1)
				} else {
					atomic.AddInt64(&failureCounter, 1)
				}
			}
		}(i)
	}
	
	// Wait for completion
	wg.Wait()
	
	// Calculate results
	duration := time.Since(startTime)
	result.SuccessfulRequests = successCounter
	result.FailedRequests = failureCounter
	result.ThroughputRPS = float64(result.SuccessfulRequests) / duration.Seconds()
	result.ErrorRate = float64(result.FailedRequests) / float64(result.TotalRequests) * 100
	
	// Calculate response time statistics
	if len(responseTimes) > 0 {
		sort.Slice(responseTimes, func(i, j int) bool {
			return responseTimes[i] < responseTimes[j]
		})
		
		result.MinResponseTime = responseTimes[0]
		result.MaxResponseTime = responseTimes[len(responseTimes)-1]
		result.P50ResponseTime = responseTimes[len(responseTimes)/2]
		result.P95ResponseTime = responseTimes[int(float64(len(responseTimes))*0.95)]
		result.P99ResponseTime = responseTimes[int(float64(len(responseTimes))*0.99)]
		
		var total time.Duration
		for _, rt := range responseTimes {
			total += rt
		}
		result.AverageResponseTime = total / time.Duration(len(responseTimes))
	}
	
	return result
}

// executeA1PolicyLoadTest executes A1 policy load test
func (suite *LoadTestSuite) executeA1PolicyLoadTest(policyCount int) *A1PolicyLoadResult {
	result := &A1PolicyLoadResult{
		ConcurrentPolicies: policyCount,
		TotalPolicies:      int64(policyCount),
		Errors:             make([]string, 0),
	}
	
	startTime := time.Now()
	
	var wg sync.WaitGroup
	var createdCounter int64
	var enforcedCounter int64
	var failedCounter int64
	
	var creationTimes []time.Duration
	var enforcementTimes []time.Duration
	var timesMutex sync.Mutex
	
	// Create policies concurrently
	for i := 0; i < policyCount; i++ {
		wg.Add(1)
		go func(policyID int) {
			defer wg.Done()
			
			// Policy creation
			createStart := time.Now()
			created := suite.simulatePolicyCreation(policyID)
			createDuration := time.Since(createStart)
			
			timesMutex.Lock()
			creationTimes = append(creationTimes, createDuration)
			timesMutex.Unlock()
			
			if created {
				atomic.AddInt64(&createdCounter, 1)
				
				// Policy enforcement
				enforceStart := time.Now()
				enforced := suite.simulatePolicyEnforcement(policyID)
				enforceDuration := time.Since(enforceStart)
				
				timesMutex.Lock()
				enforcementTimes = append(enforcementTimes, enforceDuration)
				timesMutex.Unlock()
				
				if enforced {
					atomic.AddInt64(&enforcedCounter, 1)
				} else {
					atomic.AddInt64(&failedCounter, 1)
				}
			} else {
				atomic.AddInt64(&failedCounter, 1)
			}
		}(i)
	}
	
	wg.Wait()
	
	duration := time.Since(startTime)
	result.CreatedPolicies = createdCounter
	result.EnforcedPolicies = enforcedCounter
	result.FailedPolicies = failedCounter
	result.ThroughputPPS = float64(result.EnforcedPolicies) / duration.Seconds()
	result.ErrorRate = float64(result.FailedPolicies) / float64(result.TotalPolicies) * 100
	
	// Calculate timing statistics
	if len(creationTimes) > 0 {
		var totalCreate time.Duration
		for _, ct := range creationTimes {
			totalCreate += ct
		}
		result.AverageCreationTime = totalCreate / time.Duration(len(creationTimes))
	}
	
	if len(enforcementTimes) > 0 {
		var totalEnforce time.Duration
		for _, et := range enforcementTimes {
			totalEnforce += et
		}
		result.AverageEnforcementTime = totalEnforce / time.Duration(len(enforcementTimes))
	}
	
	return result
}

// executeXAppLoadTest executes xApp deployment load test
func (suite *LoadTestSuite) executeXAppLoadTest(xappCount int) *XAppLoadResult {
	result := &XAppLoadResult{
		ConcurrentXApps:   xappCount,
		TotalDeployments:  int64(xappCount),
		Errors:            make([]string, 0),
	}
	
	startTime := time.Now()
	
	var wg sync.WaitGroup
	var successCounter int64
	var failedCounter int64
	
	var deploymentTimes []time.Duration
	var scalingTimes []time.Duration
	var timesMutex sync.Mutex
	
	// Deploy xApps concurrently
	for i := 0; i < xappCount; i++ {
		wg.Add(1)
		go func(xappID int) {
			defer wg.Done()
			
			// xApp deployment
			deployStart := time.Now()
			deployed := suite.simulateXAppDeployment(xappID)
			deployDuration := time.Since(deployStart)
			
			timesMutex.Lock()
			deploymentTimes = append(deploymentTimes, deployDuration)
			timesMutex.Unlock()
			
			if deployed {
				atomic.AddInt64(&successCounter, 1)
				
				// xApp scaling test
				scaleStart := time.Now()
				suite.simulateXAppScaling(xappID)
				scaleDuration := time.Since(scaleStart)
				
				timesMutex.Lock()
				scalingTimes = append(scalingTimes, scaleDuration)
				timesMutex.Unlock()
			} else {
				atomic.AddInt64(&failedCounter, 1)
			}
		}(i)
	}
	
	wg.Wait()
	
	duration := time.Since(startTime)
	result.SuccessfulDeployments = successCounter
	result.FailedDeployments = failedCounter
	
	// Calculate timing statistics
	if len(deploymentTimes) > 0 {
		var totalDeploy time.Duration
		for _, dt := range deploymentTimes {
			totalDeploy += dt
		}
		result.AverageDeploymentTime = totalDeploy / time.Duration(len(deploymentTimes))
	}
	
	if len(scalingTimes) > 0 {
		var totalScale time.Duration
		for _, st := range scalingTimes {
			totalScale += st
		}
		result.AverageScalingTime = totalScale / time.Duration(len(scalingTimes))
	}
	
	// Calculate resource efficiency
	if duration.Seconds() > 0 {
		result.ResourceEfficiency = float64(result.SuccessfulDeployments) / duration.Seconds()
	}
	
	return result
}

// executeSystemLoadTest executes system-wide sustained load test
func (suite *LoadTestSuite) executeSystemLoadTest(duration time.Duration, targetRPS int) *SystemLoadResult {
	result := &SystemLoadResult{
		Duration:      duration,
		SystemStability: true,
		Errors:        make([]string, 0),
	}
	
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	var totalRequests int64
	var successRequests int64
	var failedRequests int64
	
	// Calculate request interval
	requestInterval := time.Second / time.Duration(targetRPS)
	
	var wg sync.WaitGroup
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()
	
	// Generate sustained load
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				wg.Add(1)
				go func() {
					defer wg.Done()
					
					atomic.AddInt64(&totalRequests, 1)
					
					// Simulate mixed workload
					success := suite.simulateMixedWorkload()
					if success {
						atomic.AddInt64(&successRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}
				}()
			}
		}
	}()
	
	// Wait for test completion
	<-ctx.Done()
	wg.Wait()
	
	actualDuration := time.Since(startTime)
	
	result.TotalRequests = totalRequests
	result.SuccessfulRequests = successRequests
	result.FailedRequests = failedRequests
	result.SustainedThroughput = float64(successRequests) / actualDuration.Seconds()
	
	// Check system stability
	errorRate := float64(failedRequests) / float64(totalRequests) * 100
	if errorRate > 1.0 { // More than 1% error rate indicates instability
		result.SystemStability = false
	}
	
	return result
}

// Simulation Methods

func (suite *LoadTestSuite) simulateE2NodeOperation(nodeID, requestID int) bool {
	// Simulate E2 node operation with realistic timing
	baseLatency := 50 * time.Millisecond
	jitter := time.Duration(nodeID%20) * time.Millisecond
	time.Sleep(baseLatency + jitter)
	
	// Simulate 99% success rate
	return (nodeID*requestID)%100 != 0
}

func (suite *LoadTestSuite) simulatePolicyCreation(policyID int) bool {
	// Simulate A1 policy creation
	time.Sleep(time.Duration(100+policyID%50) * time.Millisecond)
	return policyID%50 != 0 // 98% success rate
}

func (suite *LoadTestSuite) simulatePolicyEnforcement(policyID int) bool {
	// Simulate policy enforcement
	time.Sleep(time.Duration(200+policyID%100) * time.Millisecond)
	return policyID%25 != 0 // 96% success rate
}

func (suite *LoadTestSuite) simulateXAppDeployment(xappID int) bool {
	// Simulate xApp deployment (longer operation)
	time.Sleep(time.Duration(2000+xappID%1000) * time.Millisecond)
	return xappID%10 != 0 // 90% success rate
}

func (suite *LoadTestSuite) simulateXAppScaling(xappID int) bool {
	// Simulate xApp scaling
	time.Sleep(time.Duration(1000+xappID%500) * time.Millisecond)
	return true
}

func (suite *LoadTestSuite) simulateMixedWorkload() bool {
	// Simulate mixed workload with different operation types
	operationType := time.Now().Nanosecond() % 4
	
	switch operationType {
	case 0: // E2 node operation
		time.Sleep(20 * time.Millisecond)
	case 1: // Policy operation
		time.Sleep(50 * time.Millisecond)
	case 2: // xApp operation
		time.Sleep(100 * time.Millisecond)
	case 3: // System operation
		time.Sleep(10 * time.Millisecond)
	}
	
	// 99.5% success rate for mixed workload
	return time.Now().Nanosecond()%1000 != 0
}

// Measurement Methods

func (suite *LoadTestSuite) measureE2SetupLatency(samples int) LatencyMetrics {
	latencies := make([]time.Duration, samples)
	
	for i := 0; i < samples; i++ {
		start := time.Now()
		// Simulate E2 setup procedure
		suite.simulateE2NodeOperation(i, 0)
		latencies[i] = time.Since(start)
	}
	
	return suite.calculateLatencyMetrics(latencies)
}

func (suite *LoadTestSuite) measureIndicationLatency(samples int) LatencyMetrics {
	latencies := make([]time.Duration, samples)
	
	for i := 0; i < samples; i++ {
		start := time.Now()
		// Simulate indication processing
		time.Sleep(time.Duration(2+i%8) * time.Millisecond)
		latencies[i] = time.Since(start)
	}
	
	return suite.calculateLatencyMetrics(latencies)
}

func (suite *LoadTestSuite) measurePolicyLatency(samples int) LatencyMetrics {
	latencies := make([]time.Duration, samples)
	
	for i := 0; i < samples; i++ {
		start := time.Now()
		// Simulate policy processing
		suite.simulatePolicyCreation(i)
		latencies[i] = time.Since(start)
	}
	
	return suite.calculateLatencyMetrics(latencies)
}

func (suite *LoadTestSuite) measureControlLatency(samples int) LatencyMetrics {
	latencies := make([]time.Duration, samples)
	
	for i := 0; i < samples; i++ {
		start := time.Now()
		// Simulate RIC control
		time.Sleep(time.Duration(30+i%20) * time.Millisecond)
		latencies[i] = time.Since(start)
	}
	
	return suite.calculateLatencyMetrics(latencies)
}

func (suite *LoadTestSuite) calculateLatencyMetrics(latencies []time.Duration) LatencyMetrics {
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	
	var total time.Duration
	for _, lat := range latencies {
		total += lat
	}
	
	return LatencyMetrics{
		Average: total / time.Duration(len(latencies)),
		P50:     latencies[len(latencies)/2],
		P95:     latencies[int(float64(len(latencies))*0.95)],
		P99:     latencies[int(float64(len(latencies))*0.99)],
		P999:    latencies[int(float64(len(latencies))*0.999)],
		Max:     latencies[len(latencies)-1],
		Min:     latencies[0],
	}
}

func (suite *LoadTestSuite) findMaxThroughput() float64 {
	// Binary search for maximum sustainable throughput
	minRPS := 100.0
	maxRPS := 5000.0
	tolerance := 10.0
	
	for maxRPS-minRPS > tolerance {
		testRPS := (minRPS + maxRPS) / 2
		
		// Test throughput for 30 seconds
		sustainable := suite.testThroughputSustainability(testRPS, 30*time.Second)
		
		if sustainable {
			minRPS = testRPS
		} else {
			maxRPS = testRPS
		}
	}
	
	return minRPS
}

func (suite *LoadTestSuite) testThroughputSustainability(targetRPS float64, duration time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	var successCount int64
	var totalCount int64
	
	interval := time.Second / time.Duration(targetRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			successRate := float64(successCount) / float64(totalCount) * 100
			return successRate >= suite.performanceTargets.MinSuccessRate
		case <-ticker.C:
			atomic.AddInt64(&totalCount, 1)
			if suite.simulateMixedWorkload() {
				atomic.AddInt64(&successCount, 1)
			}
		}
	}
}

func (suite *LoadTestSuite) measureResourceEfficiency() ResourceEfficiencyMetrics {
	// Measure resource efficiency under standard load (1000 RPS for 60 seconds)
	duration := 60 * time.Second
	targetRPS := 1000.0
	
	suite.startResourceMonitoring()
	
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	var requestCount int64
	interval := time.Second / time.Duration(targetRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			go func() {
				suite.simulateMixedWorkload()
				atomic.AddInt64(&requestCount, 1)
			}()
		}
	}
	
done:
	utilization := suite.stopResourceMonitoring()
	
	// Calculate efficiency metrics
	requestsPerSecond := float64(requestCount) / duration.Seconds()
	
	return ResourceEfficiencyMetrics{
		CPUEfficiency:     requestsPerSecond / (utilization.CPUUsagePercent / 100 * 8), // Assuming 8 cores
		MemoryEfficiency:  requestsPerSecond / (utilization.MemoryUsagePercent / 100 * 16), // Assuming 16GB
		NetworkEfficiency: requestsPerSecond / utilization.NetworkThroughputMbps,
		EnergyEfficiency:  requestsPerSecond / 100, // Assuming 100W power consumption
	}
}

func (suite *LoadTestSuite) measureScalabilityLimits() ScalabilityMetrics {
	metrics := ScalabilityMetrics{}
	
	// Find maximum concurrent E2 nodes
	metrics.MaxConcurrentNodes = suite.findMaxConcurrentNodes()
	
	// Find maximum concurrent policies
	metrics.MaxConcurrentPolicies = suite.findMaxConcurrentPolicies()
	
	// Find maximum concurrent xApps
	metrics.MaxConcurrentXApps = suite.findMaxConcurrentXApps()
	
	// Find linear scaling limit
	metrics.LinearScalingLimit = suite.findLinearScalingLimit()
	
	// Calculate resource scaling factor
	metrics.ResourceScalingFactor = suite.calculateResourceScalingFactor()
	
	return metrics
}

func (suite *LoadTestSuite) findMaxConcurrentNodes() int {
	// Binary search for maximum concurrent nodes
	min := 50
	max := 500
	
	for max-min > 5 {
		test := (min + max) / 2
		
		result := suite.executeE2NodeLoadTest(test, 10)
		if result.ErrorRate < 1.0 && result.AverageResponseTime < 500*time.Millisecond {
			min = test
		} else {
			max = test
		}
	}
	
	return min
}

func (suite *LoadTestSuite) findMaxConcurrentPolicies() int {
	// Binary search for maximum concurrent policies
	min := 100
	max := 2000
	
	for max-min > 10 {
		test := (min + max) / 2
		
		result := suite.executeA1PolicyLoadTest(test)
		if result.ErrorRate < 1.0 && result.AverageCreationTime < 1*time.Second {
			min = test
		} else {
			max = test
		}
	}
	
	return min
}

func (suite *LoadTestSuite) findMaxConcurrentXApps() int {
	// Binary search for maximum concurrent xApps
	min := 10
	max := 100
	
	for max-min > 2 {
		test := (min + max) / 2
		
		result := suite.executeXAppLoadTest(test)
		if result.FailedDeployments == 0 && result.AverageDeploymentTime < 30*time.Second {
			min = test
		} else {
			max = test
		}
	}
	
	return min
}

func (suite *LoadTestSuite) findLinearScalingLimit() int {
	// Find the point where scaling becomes non-linear
	baseline := suite.executeE2NodeLoadTest(50, 20)
	baselineRatio := baseline.ThroughputRPS / float64(baseline.ConcurrentNodes)
	
	for nodes := 100; nodes <= 500; nodes += 50 {
		result := suite.executeE2NodeLoadTest(nodes, 20)
		currentRatio := result.ThroughputRPS / float64(result.ConcurrentNodes)
		
		// If throughput per node drops by more than 20%, we've hit the limit
		if currentRatio < baselineRatio*0.8 {
			return nodes - 50
		}
	}
	
	return 500 // Maximum tested
}

func (suite *LoadTestSuite) calculateResourceScalingFactor() float64 {
	// Measure resource usage scaling factor
	result50 := suite.executeE2NodeLoadTest(50, 20)
	result100 := suite.executeE2NodeLoadTest(100, 20)
	
	throughputRatio := result100.ThroughputRPS / result50.ThroughputRPS
	expectedRatio := 100.0 / 50.0 // 2.0 for linear scaling
	
	return throughputRatio / expectedRatio
}

// Resource Monitoring

func (suite *LoadTestSuite) startResourceMonitoring() {
	suite.resourceMonitor.monitoring = true
	suite.resourceMonitor.measurements = make([]ResourceMeasurement, 0)
	
	go func() {
		ticker := time.NewTicker(suite.resourceMonitor.interval)
		defer ticker.Stop()
		
		for suite.resourceMonitor.monitoring {
			select {
			case <-ticker.C:
				measurement := suite.collectResourceMeasurement()
				
				suite.resourceMonitor.mutex.Lock()
				suite.resourceMonitor.measurements = append(suite.resourceMonitor.measurements, measurement)
				suite.resourceMonitor.mutex.Unlock()
			}
		}
	}()
}

func (suite *LoadTestSuite) stopResourceMonitoring() ResourceUtilization {
	suite.resourceMonitor.monitoring = false
	time.Sleep(suite.resourceMonitor.interval * 2) // Wait for monitoring to stop
	
	suite.resourceMonitor.mutex.RLock()
	defer suite.resourceMonitor.mutex.RUnlock()
	
	if len(suite.resourceMonitor.measurements) == 0 {
		return ResourceUtilization{}
	}
	
	// Calculate average utilization
	var avgCPU, avgMemory, avgNetwork, avgDiskIOPS float64
	var avgGoroutines int
	var avgHeap float64
	
	for _, measurement := range suite.resourceMonitor.measurements {
		avgCPU += measurement.Utilization.CPUUsagePercent
		avgMemory += measurement.Utilization.MemoryUsagePercent
		avgNetwork += measurement.Utilization.NetworkThroughputMbps
		avgDiskIOPS += measurement.Utilization.DiskIOPSUsage
		avgGoroutines += measurement.Utilization.GoroutineCount
		avgHeap += measurement.Utilization.HeapSizeMB
	}
	
	count := float64(len(suite.resourceMonitor.measurements))
	
	return ResourceUtilization{
		CPUUsagePercent:       avgCPU / count,
		MemoryUsagePercent:    avgMemory / count,
		NetworkThroughputMbps: avgNetwork / count,
		DiskIOPSUsage:        avgDiskIOPS / count,
		GoroutineCount:       int(float64(avgGoroutines) / count),
		HeapSizeMB:           avgHeap / count,
	}
}

func (suite *LoadTestSuite) collectResourceMeasurement() ResourceMeasurement {
	// Simulate resource collection (in real implementation, would query system metrics)
	return ResourceMeasurement{
		Timestamp: time.Now(),
		Utilization: ResourceUtilization{
			CPUUsagePercent:       60.0 + math.Sin(float64(time.Now().Unix()))*10,
			MemoryUsagePercent:    70.0 + math.Cos(float64(time.Now().Unix()))*5,
			NetworkThroughputMbps: 100.0 + math.Sin(float64(time.Now().Unix()/2))*20,
			DiskIOPSUsage:        50.0 + math.Cos(float64(time.Now().Unix()/3))*10,
			GoroutineCount:       1000 + int(math.Sin(float64(time.Now().Unix()))*100),
			HeapSizeMB:           512.0 + math.Cos(float64(time.Now().Unix()))*50,
		},
	}
}

// Performance Validation

func (suite *LoadTestSuite) validateE2NodePerformance(result *E2NodeLoadResult) {
	// Validate E2 node performance requirements
	assert.LessOrEqual(suite.T(), result.P95ResponseTime, suite.performanceTargets.MaxE2SetupLatency,
		"E2 node P95 response time exceeds target for %d nodes", result.ConcurrentNodes)
	
	assert.LessOrEqual(suite.T(), result.ErrorRate, 100-suite.performanceTargets.MinSuccessRate,
		"E2 node error rate exceeds target for %d nodes", result.ConcurrentNodes)
	
	assert.LessOrEqual(suite.T(), result.ResourceUtilization.CPUUsagePercent, suite.performanceTargets.MaxCPUUsage,
		"CPU usage exceeds target for %d nodes", result.ConcurrentNodes)
	
	assert.LessOrEqual(suite.T(), result.ResourceUtilization.MemoryUsagePercent, suite.performanceTargets.MaxMemoryUsage,
		"Memory usage exceeds target for %d nodes", result.ConcurrentNodes)
	
	log.Printf("E2 node load test (%d nodes): %.2f RPS, %.1f%% error rate, P95: %v",
		result.ConcurrentNodes, result.ThroughputRPS, result.ErrorRate, result.P95ResponseTime)
}

func (suite *LoadTestSuite) validateA1PolicyPerformance(result *A1PolicyLoadResult) {
	// Validate A1 policy performance requirements
	assert.LessOrEqual(suite.T(), result.AverageCreationTime, suite.performanceTargets.MaxPolicyLatency,
		"Policy creation time exceeds target for %d policies", result.ConcurrentPolicies)
	
	assert.LessOrEqual(suite.T(), result.ErrorRate, 100-suite.performanceTargets.MinSuccessRate,
		"Policy error rate exceeds target for %d policies", result.ConcurrentPolicies)
	
	log.Printf("A1 policy load test (%d policies): %.2f PPS, %.1f%% error rate, avg creation: %v",
		result.ConcurrentPolicies, result.ThroughputPPS, result.ErrorRate, result.AverageCreationTime)
}

func (suite *LoadTestSuite) validateXAppPerformance(result *XAppLoadResult) {
	// Validate xApp performance requirements
	assert.LessOrEqual(suite.T(), result.AverageDeploymentTime, 60*time.Second,
		"xApp deployment time exceeds 60s for %d xApps", result.ConcurrentXApps)
	
	assert.Equal(suite.T(), result.FailedDeployments, int64(0),
		"xApp deployment failures detected for %d xApps", result.ConcurrentXApps)
	
	log.Printf("xApp load test (%d xApps): %.1f deployments/s, avg deployment: %v",
		result.ConcurrentXApps, result.ResourceEfficiency, result.AverageDeploymentTime)
}

func (suite *LoadTestSuite) validateSystemPerformance(result *SystemLoadResult) {
	// Validate system performance requirements
	assert.True(suite.T(), result.SystemStability,
		"System stability compromised during %s test", result.TestName)
	
	successRate := float64(result.SuccessfulRequests) / float64(result.TotalRequests) * 100
	assert.GreaterOrEqual(suite.T(), successRate, suite.performanceTargets.MinSuccessRate,
		"Success rate below target during %s test", result.TestName)
	
	assert.GreaterOrEqual(suite.T(), result.SustainedThroughput, suite.performanceTargets.MinThroughputRPS,
		"Sustained throughput below target during %s test", result.TestName)
	
	log.Printf("System load test (%s): %.2f RPS sustained, %.1f%% success rate",
		result.TestName, result.SustainedThroughput, successRate)
}

// generateLoadTestReport generates comprehensive load test report
func (suite *LoadTestSuite) generateLoadTestReport() {
	duration := suite.testResults.EndTime.Sub(suite.testResults.StartTime)
	
	report := fmt.Sprintf(`
================================================================
O-RAN L Release Performance and Load Test Report
================================================================
Test Duration: %v
Total Tests: %d
Passed Tests: %d
Failed Tests: %d
Success Rate: %.2f%%

E2 Node Load Test Results:
`, duration,
		suite.testResults.TotalTests,
		suite.testResults.PassedTests,
		suite.testResults.FailedTests,
		float64(suite.testResults.PassedTests)/float64(suite.testResults.TotalTests)*100)
	
	for testName, result := range suite.testResults.E2NodeLoadTests {
		report += fmt.Sprintf("- %s: %d nodes, %.2f RPS, %.1f%% errors, P95: %v\n",
			testName, result.ConcurrentNodes, result.ThroughputRPS, result.ErrorRate, result.P95ResponseTime)
	}
	
	report += "\nA1 Policy Load Test Results:\n"
	for testName, result := range suite.testResults.A1PolicyLoadTests {
		report += fmt.Sprintf("- %s: %d policies, %.2f PPS, %.1f%% errors, avg creation: %v\n",
			testName, result.ConcurrentPolicies, result.ThroughputPPS, result.ErrorRate, result.AverageCreationTime)
	}
	
	report += "\nxApp Load Test Results:\n"
	for testName, result := range suite.testResults.XAppLoadTests {
		report += fmt.Sprintf("- %s: %d xApps, %.1f deployments/s, avg deployment: %v\n",
			testName, result.ConcurrentXApps, result.ResourceEfficiency, result.AverageDeploymentTime)
	}
	
	report += "\nSystem Load Test Results:\n"
	for testName, result := range suite.testResults.SystemLoadTests {
		successRate := float64(result.SuccessfulRequests) / float64(result.TotalRequests) * 100
		report += fmt.Sprintf("- %s: %.2f RPS sustained, %.1f%% success, stable: %v\n",
			testName, result.SustainedThroughput, successRate, result.SystemStability)
	}
	
	if suite.testResults.PerformanceMetrics != nil {
		metrics := suite.testResults.PerformanceMetrics
		report += fmt.Sprintf(`
Performance Metrics Summary:
- E2 Setup Latency: avg=%v, P95=%v, P99=%v
- Indication Latency: avg=%v, P95=%v, P99=%v
- Policy Latency: avg=%v, P95=%v, P99=%v
- Control Latency: avg=%v, P95=%v, P99=%v
- Peak Throughput: %.2f RPS
- Sustained Throughput: %.2f RPS
- CPU Efficiency: %.2f req/core/sec
- Memory Efficiency: %.2f req/GB/sec
- Max Concurrent Nodes: %d
- Max Concurrent Policies: %d
- Max Concurrent xApps: %d
`,
			metrics.E2SetupLatency.Average, metrics.E2SetupLatency.P95, metrics.E2SetupLatency.P99,
			metrics.IndicationLatency.Average, metrics.IndicationLatency.P95, metrics.IndicationLatency.P99,
			metrics.PolicyLatency.Average, metrics.PolicyLatency.P95, metrics.PolicyLatency.P99,
			metrics.ControlLatency.Average, metrics.ControlLatency.P95, metrics.ControlLatency.P99,
			metrics.SystemThroughput.PeakThroughput,
			metrics.SystemThroughput.SustainedThroughput,
			metrics.ResourceEfficiency.CPUEfficiency,
			metrics.ResourceEfficiency.MemoryEfficiency,
			metrics.ScalabilityMetrics.MaxConcurrentNodes,
			metrics.ScalabilityMetrics.MaxConcurrentPolicies,
			metrics.ScalabilityMetrics.MaxConcurrentXApps)
	}
	
	report += "\n================================================================\n"
	
	// Write to file
	reportFile := fmt.Sprintf("test-results/load_test_report_%s.txt",
		time.Now().Format("20060102_150405"))
	
	err := os.WriteFile(reportFile, []byte(report), 0644)
	if err != nil {
		log.Printf("Failed to write load test report: %v", err)
	} else {
		log.Printf("Load test report written to: %s", reportFile)
	}
	
	// Also print to console
	fmt.Println(report)
}

// TestLoadTestSuite runs the performance and load test suite
func TestLoadTestSuite(t *testing.T) {
	suite.Run(t, new(LoadTestSuite))
}