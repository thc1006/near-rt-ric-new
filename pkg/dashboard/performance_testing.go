package dashboard

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// PerformanceTestSuite manages all performance testing scenarios
type PerformanceTestSuite struct {
	e2Manager     *E2ManagerClient
	subManager    *SubscriptionManagerClient
	prometheusAPI v1.API
	testResults   *TestResults
	config        *PerformanceTestConfig
	mu            sync.RWMutex
}

// PerformanceTestConfig defines test parameters
type PerformanceTestConfig struct {
	MaxE2Nodes           int           `json:"maxE2Nodes"`
	MaxConcurrentSubs    int           `json:"maxConcurrentSubs"`
	TargetThroughput     int           `json:"targetThroughput"` // indications per second
	MaxLatencyMs         int           `json:"maxLatencyMs"`
	TestDurationMinutes  int           `json:"testDurationMinutes"`
	StabilityTestHours   int           `json:"stabilityTestHours"`
	MemoryLeakThresholdMB int          `json:"memoryLeakThresholdMB"`
	CPUThresholdPercent  float64       `json:"cpuThresholdPercent"`
	LoadRampUpDuration   time.Duration `json:"loadRampUpDuration"`
}

// TestResults stores comprehensive test metrics
type TestResults struct {
	LoadTestResults      *LoadTestResults      `json:"loadTestResults"`
	ThroughputResults    *ThroughputResults    `json:"throughputResults"`
	LatencyResults       *LatencyResults       `json:"latencyResults"`
	StressTestResults    *StressTestResults    `json:"stressTestResults"`
	StabilityResults     *StabilityResults     `json:"stabilityResults"`
	ResourceUtilization  *ResourceUtilization  `json:"resourceUtilization"`
	TestSummary          *TestSummary          `json:"testSummary"`
}

// LoadTestResults captures load testing metrics
type LoadTestResults struct {
	MaxConcurrentE2Nodes    int                    `json:"maxConcurrentE2Nodes"`
	MaxConcurrentSubs       int                    `json:"maxConcurrentSubs"`
	ConnectionSuccessRate   float64                `json:"connectionSuccessRate"`
	SubscriptionSuccessRate float64                `json:"subscriptionSuccessRate"`
	ErrorRates              map[string]float64     `json:"errorRates"`
	ResponseTimes           *ResponseTimeMetrics   `json:"responseTimes"`
	ResourceConsumption     *ResourceConsumption   `json:"resourceConsumption"`
	Timestamp               time.Time              `json:"timestamp"`
}

// ThroughputResults captures throughput testing metrics
type ThroughputResults struct {
	MaxIndicationsPerSecond    int                 `json:"maxIndicationsPerSecond"`
	SustainedThroughput        int                 `json:"sustainedThroughput"`
	ThroughputOverTime         []ThroughputSample  `json:"throughputOverTime"`
	ProcessingLatencyMs        []float64           `json:"processingLatencyMs"`
	QueueDepthMetrics          *QueueMetrics       `json:"queueDepthMetrics"`
	BackpressureEvents         int                 `json:"backpressureEvents"`
	Timestamp                  time.Time           `json:"timestamp"`
}

// LatencyResults captures latency testing metrics
type LatencyResults struct {
	E2SetupLatencyMs        *LatencyMetrics `json:"e2SetupLatencyMs"`
	SubscriptionLatencyMs   *LatencyMetrics `json:"subscriptionLatencyMs"`
	IndicationLatencyMs     *LatencyMetrics `json:"indicationLatencyMs"`
	ControlLatencyMs        *LatencyMetrics `json:"controlLatencyMs"`
	EndToEndLatencyMs       *LatencyMetrics `json:"endToEndLatencyMs"`
	LatencyDistribution     []LatencyBucket `json:"latencyDistribution"`
	Timestamp               time.Time       `json:"timestamp"`
}

// StressTestResults captures stress testing metrics
type StressTestResults struct {
	ResourceExhaustionPoint *ResourceExhaustionPoint `json:"resourceExhaustionPoint"`
	FailureScenarios        []FailureScenario        `json:"failureScenarios"`
	RecoveryMetrics         *RecoveryMetrics         `json:"recoveryMetrics"`
	SystemLimits            *SystemLimits            `json:"systemLimits"`
	Timestamp               time.Time                `json:"timestamp"`
}

// StabilityResults captures long-running stability metrics
type StabilityResults struct {
	TestDurationHours       float64              `json:"testDurationHours"`
	MemoryLeakDetected      bool                 `json:"memoryLeakDetected"`
	MemoryUsageOverTime     []MemorySample       `json:"memoryUsageOverTime"`
	CPUUsageOverTime        []CPUSample          `json:"cpuUsageOverTime"`
	ConnectionStability     *ConnectionStability `json:"connectionStability"`
	PerformanceDegradation  *PerformanceDegradation `json:"performanceDegradation"`
	ErrorRateOverTime       []ErrorRateSample    `json:"errorRateOverTime"`
	Timestamp               time.Time            `json:"timestamp"`
}

// Supporting metric structures
type ResponseTimeMetrics struct {
	Mean   float64 `json:"mean"`
	P50    float64 `json:"p50"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	P999   float64 `json:"p999"`
	Max    float64 `json:"max"`
	Min    float64 `json:"min"`
	StdDev float64 `json:"stdDev"`
}

type ResourceConsumption struct {
	CPUUsagePercent    float64 `json:"cpuUsagePercent"`
	MemoryUsageMB      float64 `json:"memoryUsageMB"`
	NetworkBytesPerSec int64   `json:"networkBytesPerSec"`
	DiskIOPerSec       int64   `json:"diskIOPerSec"`
	GoroutineCount     int     `json:"goroutineCount"`
	GCPauseMs          float64 `json:"gcPauseMs"`
}

type ThroughputSample struct {
	Timestamp           time.Time `json:"timestamp"`
	IndicationsPerSec   int       `json:"indicationsPerSec"`
	ProcessingLatencyMs float64   `json:"processingLatencyMs"`
}

type QueueMetrics struct {
	MaxDepth     int     `json:"maxDepth"`
	AvgDepth     float64 `json:"avgDepth"`
	OverflowRate float64 `json:"overflowRate"`
}

type LatencyMetrics struct {
	Mean   float64 `json:"mean"`
	P50    float64 `json:"p50"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	Max    float64 `json:"max"`
	Min    float64 `json:"min"`
	Count  int64   `json:"count"`
}

type LatencyBucket struct {
	UpperBoundMs float64 `json:"upperBoundMs"`
	Count        int64   `json:"count"`
	Percentage   float64 `json:"percentage"`
}

type ResourceExhaustionPoint struct {
	CPUPercent     float64 `json:"cpuPercent"`
	MemoryMB       float64 `json:"memoryMB"`
	ActiveE2Nodes  int     `json:"activeE2Nodes"`
	ActiveSubs     int     `json:"activeSubs"`
	ThroughputIPS  int     `json:"throughputIPS"`
}

type FailureScenario struct {
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	TriggerPoint    string    `json:"triggerPoint"`
	RecoveryTimeMs  int64     `json:"recoveryTimeMs"`
	DataLoss        bool      `json:"dataLoss"`
	Timestamp       time.Time `json:"timestamp"`
}

type RecoveryMetrics struct {
	MeanRecoveryTimeMs float64 `json:"meanRecoveryTimeMs"`
	MaxRecoveryTimeMs  int64   `json:"maxRecoveryTimeMs"`
	SuccessfulRecoveries int   `json:"successfulRecoveries"`
	FailedRecoveries   int     `json:"failedRecoveries"`
}

type SystemLimits struct {
	MaxE2Connections    int `json:"maxE2Connections"`
	MaxSubscriptions    int `json:"maxSubscriptions"`
	MaxThroughputIPS    int `json:"maxThroughputIPS"`
	MaxMemoryUsageMB    int `json:"maxMemoryUsageMB"`
	MaxCPUUsagePercent  int `json:"maxCPUUsagePercent"`
}

type MemorySample struct {
	Timestamp time.Time `json:"timestamp"`
	UsageMB   float64   `json:"usageMB"`
	HeapMB    float64   `json:"heapMB"`
	StackMB   float64   `json:"stackMB"`
}

type CPUSample struct {
	Timestamp time.Time `json:"timestamp"`
	Percent   float64   `json:"percent"`
}

type ConnectionStability struct {
	TotalConnections    int64   `json:"totalConnections"`
	DroppedConnections  int64   `json:"droppedConnections"`
	ReconnectionRate    float64 `json:"reconnectionRate"`
	StabilityScore      float64 `json:"stabilityScore"`
}

type PerformanceDegradation struct {
	InitialThroughput   int     `json:"initialThroughput"`
	FinalThroughput     int     `json:"finalThroughput"`
	DegradationPercent  float64 `json:"degradationPercent"`
	InitialLatencyMs    float64 `json:"initialLatencyMs"`
	FinalLatencyMs      float64 `json:"finalLatencyMs"`
}

type ErrorRateSample struct {
	Timestamp time.Time `json:"timestamp"`
	ErrorRate float64   `json:"errorRate"`
}

type ResourceUtilization struct {
	PeakCPUPercent     float64 `json:"peakCPUPercent"`
	PeakMemoryMB       float64 `json:"peakMemoryMB"`
	AvgCPUPercent      float64 `json:"avgCPUPercent"`
	AvgMemoryMB        float64 `json:"avgMemoryMB"`
	NetworkUtilization int64   `json:"networkUtilization"`
	DiskUtilization    int64   `json:"diskUtilization"`
}

type TestSummary struct {
	TotalTestDuration   time.Duration `json:"totalTestDuration"`
	TestsExecuted       int           `json:"testsExecuted"`
	TestsPassed         int           `json:"testsPassed"`
	TestsFailed         int           `json:"testsFailed"`
	OverallScore        float64       `json:"overallScore"`
	Recommendations     []string      `json:"recommendations"`
	CriticalIssues      []string      `json:"criticalIssues"`
	PerformanceGrade    string        `json:"performanceGrade"`
}

// NewPerformanceTestSuite creates a new performance test suite
func NewPerformanceTestSuite(e2Manager *E2ManagerClient, subManager *SubscriptionManagerClient, prometheusClient api.Client) *PerformanceTestSuite {
	prometheusAPI := v1.NewAPI(prometheusClient)
	
	return &PerformanceTestSuite{
		e2Manager:     e2Manager,
		subManager:    subManager,
		prometheusAPI: prometheusAPI,
		testResults:   &TestResults{},
		config: &PerformanceTestConfig{
			MaxE2Nodes:           200,  // Well above 100+ requirement for comprehensive testing
			MaxConcurrentSubs:    3000, // Increased for better scalability testing
			TargetThroughput:     20000, // Well above 10,000+ requirement for comprehensive validation
			MaxLatencyMs:         8,    // Stricter than 10ms for sub-10ms validation
			TestDurationMinutes:  60,   // Extended for more comprehensive testing
			StabilityTestHours:   72,   // Extended for better memory leak detection (3 days)
			MemoryLeakThresholdMB: 25,  // More sensitive threshold for early detection
			CPUThresholdPercent:  90.0, // Higher threshold for stress testing
			LoadRampUpDuration:   10 * time.Minute, // Longer ramp-up for gradual load increase
		},
	}
}

// RunAllPerformanceTests executes the complete performance test suite
func (pts *PerformanceTestSuite) RunAllPerformanceTests(ctx context.Context) (*TestResults, error) {
	log.Println("Starting comprehensive performance test suite...")
	log.Printf("Configuration: MaxE2Nodes=%d, TargetThroughput=%d IPS, MaxLatency=%dms", 
		pts.config.MaxE2Nodes, pts.config.TargetThroughput, pts.config.MaxLatencyMs)
	
	startTime := time.Now()

	// Initialize test results
	pts.testResults = &TestResults{
		LoadTestResults:     &LoadTestResults{},
		ThroughputResults:   &ThroughputResults{},
		LatencyResults:      &LatencyResults{},
		StressTestResults:   &StressTestResults{},
		StabilityResults:    &StabilityResults{},
		ResourceUtilization: &ResourceUtilization{},
		TestSummary:         &TestSummary{},
	}

	// Execute tests in optimal order for comprehensive validation
	log.Println("Phase 1: Load Testing (100+ concurrent E2 nodes validation)")
	if err := pts.RunLoadTesting(ctx); err != nil {
		log.Printf("Load testing failed: %v", err)
		// Continue with other tests even if load testing fails
	}

	log.Println("Phase 2: Throughput Testing (10,000+ IPS validation)")
	if err := pts.RunThroughputTesting(ctx); err != nil {
		log.Printf("Throughput testing failed: %v", err)
		// Continue with other tests
	}

	log.Println("Phase 3: Latency Testing (sub-10ms validation)")
	if err := pts.RunLatencyTesting(ctx); err != nil {
		log.Printf("Latency testing failed: %v", err)
		// Continue with other tests
	}

	log.Println("Phase 4: Stress Testing (resource exhaustion scenarios)")
	if err := pts.RunStressTesting(ctx); err != nil {
		log.Printf("Stress testing failed: %v", err)
		// Continue with other tests
	}

	log.Println("Phase 5: Stability Testing (long-running with memory leak detection)")
	if err := pts.RunStabilityTesting(ctx); err != nil {
		log.Printf("Stability testing failed: %v", err)
		// Continue to generate results even if stability testing fails
	}

	// Generate comprehensive test summary
	pts.generateTestSummary(time.Since(startTime))

	log.Printf("Performance test suite completed in %v", time.Since(startTime))
	log.Printf("Test Summary: %d tests executed, %d passed, %d failed, Overall Score: %.1f%%", 
		pts.testResults.TestSummary.TestsExecuted,
		pts.testResults.TestSummary.TestsPassed,
		pts.testResults.TestSummary.TestsFailed,
		pts.testResults.TestSummary.OverallScore)

	return pts.testResults, nil
}

// generateTestSummary creates a comprehensive test summary
func (pts *PerformanceTestSuite) generateTestSummary(duration time.Duration) {
	pts.mu.Lock()
	defer pts.mu.Unlock()

	summary := pts.testResults.TestSummary
	summary.TotalTestDuration = duration
	summary.TestsExecuted = 5 // Load, Throughput, Latency, Stress, Stability

	// Calculate overall score based on test results
	score := 0.0
	passed := 0

	// Load test scoring
	if pts.testResults.LoadTestResults.MaxConcurrentE2Nodes >= pts.config.MaxE2Nodes {
		score += 20
		passed++
	}

	// Throughput test scoring
	if pts.testResults.ThroughputResults.MaxIndicationsPerSecond >= pts.config.TargetThroughput {
		score += 20
		passed++
	}

	// Latency test scoring
	if pts.testResults.LatencyResults.EndToEndLatencyMs.P99 <= float64(pts.config.MaxLatencyMs) {
		score += 20
		passed++
	}

	// Stress test scoring
	if pts.testResults.StressTestResults.RecoveryMetrics.SuccessfulRecoveries > 0 {
		score += 20
		passed++
	}

	// Stability test scoring
	if !pts.testResults.StabilityResults.MemoryLeakDetected {
		score += 20
		passed++
	}

	summary.TestsPassed = passed
	summary.TestsFailed = summary.TestsExecuted - passed
	summary.OverallScore = score

	// Determine performance grade
	switch {
	case score >= 90:
		summary.PerformanceGrade = "A"
	case score >= 80:
		summary.PerformanceGrade = "B"
	case score >= 70:
		summary.PerformanceGrade = "C"
	case score >= 60:
		summary.PerformanceGrade = "D"
	default:
		summary.PerformanceGrade = "F"
	}

	// Generate recommendations
	pts.generateRecommendations(summary)
}

// generateRecommendations provides performance optimization recommendations
func (pts *PerformanceTestSuite) generateRecommendations(summary *TestSummary) {
	recommendations := []string{}
	criticalIssues := []string{}

	// Check load test results
	if pts.testResults.LoadTestResults.MaxConcurrentE2Nodes < pts.config.MaxE2Nodes {
		recommendations = append(recommendations, "Consider optimizing E2 connection handling for better scalability")
		if pts.testResults.LoadTestResults.MaxConcurrentE2Nodes < pts.config.MaxE2Nodes/2 {
			criticalIssues = append(criticalIssues, "E2 node connection capacity is critically low")
		}
	}

	// Check throughput results
	if pts.testResults.ThroughputResults.MaxIndicationsPerSecond < pts.config.TargetThroughput {
		recommendations = append(recommendations, "Optimize indication processing pipeline for higher throughput")
		if pts.testResults.ThroughputResults.MaxIndicationsPerSecond < pts.config.TargetThroughput/2 {
			criticalIssues = append(criticalIssues, "Indication processing throughput is critically low")
		}
	}

	// Check latency results
	if pts.testResults.LatencyResults.EndToEndLatencyMs.P99 > float64(pts.config.MaxLatencyMs) {
		recommendations = append(recommendations, "Reduce processing latency through code optimization and caching")
		if pts.testResults.LatencyResults.EndToEndLatencyMs.P99 > float64(pts.config.MaxLatencyMs*2) {
			criticalIssues = append(criticalIssues, "End-to-end latency exceeds acceptable limits")
		}
	}

	// Check memory leak
	if pts.testResults.StabilityResults.MemoryLeakDetected {
		criticalIssues = append(criticalIssues, "Memory leak detected during stability testing")
		recommendations = append(recommendations, "Investigate and fix memory leaks in long-running processes")
	}

	// Check resource utilization
	if pts.testResults.ResourceUtilization.PeakCPUPercent > pts.config.CPUThresholdPercent {
		recommendations = append(recommendations, "Consider CPU optimization or horizontal scaling")
	}

	summary.Recommendations = recommendations
	summary.CriticalIssues = criticalIssues
}

// GetCurrentResourceUsage returns current system resource usage
func (pts *PerformanceTestSuite) GetCurrentResourceUsage() *ResourceConsumption {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &ResourceConsumption{
		MemoryUsageMB:  float64(m.Alloc) / 1024 / 1024,
		GoroutineCount: runtime.NumGoroutine(),
		GCPauseMs:      float64(m.PauseNs[(m.NumGC+255)%256]) / 1000000,
	}
}

// MonitorResourceUsage continuously monitors resource usage during tests
func (pts *PerformanceTestSuite) MonitorResourceUsage(ctx context.Context, interval time.Duration) <-chan *ResourceConsumption {
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
				usage := pts.GetCurrentResourceUsage()
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