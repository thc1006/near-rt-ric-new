package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/api"
)

// MockE2ManagerClient for testing
type MockE2ManagerClient struct{}

func (m *MockE2ManagerClient) GetNodes() ([]E2Node, error) {
	return []E2Node{}, nil
}

// MockSubscriptionManagerClient for testing
type MockSubscriptionManagerClient struct{}

func (m *MockSubscriptionManagerClient) GetSubscriptions() ([]Subscription, error) {
	return []Subscription{}, nil
}

// MockPrometheusClient for testing
type MockPrometheusClient struct{}

func (m *MockPrometheusClient) URL(ep string, args map[string]string) *api.URL {
	return &api.URL{}
}

func (m *MockPrometheusClient) Do(ctx context.Context, req *api.Request) (*api.Response, []byte, api.Warnings, error) {
	return &api.Response{}, []byte{}, nil, nil
}

func TestNewPerformanceTestSuite(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)

	if suite == nil {
		t.Fatal("Expected performance test suite to be created")
	}

	if suite.config.MaxE2Nodes != 100 {
		t.Errorf("Expected MaxE2Nodes to be 100, got %d", suite.config.MaxE2Nodes)
	}

	if suite.config.TargetThroughput != 10000 {
		t.Errorf("Expected TargetThroughput to be 10000, got %d", suite.config.TargetThroughput)
	}

	if suite.config.MaxLatencyMs != 10 {
		t.Errorf("Expected MaxLatencyMs to be 10, got %d", suite.config.MaxLatencyMs)
	}
}

func TestLoadTestManager(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewLoadTestManager(e2Manager, subManager)

	if manager == nil {
		t.Fatal("Expected load test manager to be created")
	}

	if manager.simulatedNodes == nil {
		t.Error("Expected simulatedNodes map to be initialized")
	}

	if manager.activeTests == nil {
		t.Error("Expected activeTests map to be initialized")
	}
}

func TestThroughputTestManager(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewThroughputTestManager(e2Manager, subManager)

	if manager == nil {
		t.Fatal("Expected throughput test manager to be created")
	}

	if manager.indicationProcessor == nil {
		t.Error("Expected indication processor to be initialized")
	}

	if manager.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

func TestLatencyTestManager(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewLatencyTestManager(e2Manager, subManager)

	if manager == nil {
		t.Fatal("Expected latency test manager to be created")
	}

	if manager.latencyTracker == nil {
		t.Error("Expected latency tracker to be initialized")
	}

	if manager.testMetrics == nil {
		t.Error("Expected test metrics to be initialized")
	}
}

func TestStressTestManager(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewStressTestManager(e2Manager, subManager)

	if manager == nil {
		t.Fatal("Expected stress test manager to be created")
	}

	if manager.resourceMonitor == nil {
		t.Error("Expected resource monitor to be initialized")
	}

	if manager.failureInjector == nil {
		t.Error("Expected failure injector to be initialized")
	}
}

func TestStabilityTestManager(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewStabilityTestManager(e2Manager, subManager)

	if manager == nil {
		t.Fatal("Expected stability test manager to be created")
	}

	if manager.memoryTracker == nil {
		t.Error("Expected memory tracker to be initialized")
	}

	if manager.connectionTracker == nil {
		t.Error("Expected connection tracker to be initialized")
	}

	if manager.performanceTracker == nil {
		t.Error("Expected performance tracker to be initialized")
	}
}

func TestResourceConsumptionTracking(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)

	usage := suite.GetCurrentResourceUsage()

	if usage == nil {
		t.Fatal("Expected resource usage to be returned")
	}

	if usage.MemoryUsageMB < 0 {
		t.Error("Expected memory usage to be non-negative")
	}

	if usage.GoroutineCount < 0 {
		t.Error("Expected goroutine count to be non-negative")
	}
}

func TestResourceMonitoring(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resourceChan := suite.MonitorResourceUsage(ctx, 100*time.Millisecond)

	sampleCount := 0
	for range resourceChan {
		sampleCount++
		if sampleCount >= 5 {
			break
		}
	}

	if sampleCount == 0 {
		t.Error("Expected to receive resource usage samples")
	}
}

func TestLoadTestScenario(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:             "Test Scenario",
		MaxE2Nodes:       50,
		MaxSubscriptions: 100,
		RampUpDuration:   1 * time.Minute,
		SustainDuration:  2 * time.Minute,
		RampDownDuration: 30 * time.Second,
		ConnectionPattern: "linear",
	}

	if scenario.Name != "Test Scenario" {
		t.Error("Expected scenario name to be set correctly")
	}

	if scenario.MaxE2Nodes != 50 {
		t.Error("Expected MaxE2Nodes to be 50")
	}

	if scenario.ConnectionPattern != "linear" {
		t.Error("Expected connection pattern to be linear")
	}
}

func TestThroughputTestScenario(t *testing.T) {
	scenario := &ThroughputTestScenario{
		Name:                  "Test Throughput",
		TargetThroughputIPS:   5000,
		RampUpDuration:        2 * time.Minute,
		SustainDuration:       5 * time.Minute,
		RampDownDuration:      1 * time.Minute,
		MaxQueueDepth:         1000,
		ProcessingComplexity:  "medium",
		BackpressureThreshold: 0.8,
	}

	if scenario.TargetThroughputIPS != 5000 {
		t.Error("Expected target throughput to be 5000")
	}

	if scenario.ProcessingComplexity != "medium" {
		t.Error("Expected processing complexity to be medium")
	}

	if scenario.BackpressureThreshold != 0.8 {
		t.Error("Expected backpressure threshold to be 0.8")
	}
}

func TestLatencyTestScenario(t *testing.T) {
	scenario := &LatencyTestScenario{
		Name:                 "Test Latency",
		MaxLatencyMs:         10.0,
		OperationRate:        100,
		TestDuration:         5 * time.Minute,
		OperationTypes:       []string{"e2_setup", "subscription"},
		ConcurrentOperations: 10,
		LatencyTargets: map[string]float64{
			"e2_setup":     10.0,
			"subscription": 5.0,
		},
	}

	if scenario.MaxLatencyMs != 10.0 {
		t.Error("Expected max latency to be 10.0ms")
	}

	if len(scenario.OperationTypes) != 2 {
		t.Error("Expected 2 operation types")
	}

	if scenario.LatencyTargets["e2_setup"] != 10.0 {
		t.Error("Expected e2_setup latency target to be 10.0ms")
	}
}

func TestStressTestScenario(t *testing.T) {
	scenario := &StressTestScenario{
		Name:         "Test Stress",
		TestDuration: 15 * time.Minute,
		ResourceTargets: &ResourceTargets{
			MaxCPUPercent:    90.0,
			MaxMemoryMB:      2000,
			MaxGoroutines:    5000,
			MaxConnections:   200,
			MaxThroughputIPS: 15000,
		},
		FailureTypes:          []string{"cpu_spike", "memory_leak"},
		FailureRate:           2.0,
		RecoveryTimeout:       30 * time.Second,
		MaxConcurrentFailures: 3,
		LoadIncreaseRate:      1.2,
	}

	if scenario.ResourceTargets.MaxCPUPercent != 90.0 {
		t.Error("Expected max CPU percent to be 90.0")
	}

	if len(scenario.FailureTypes) != 2 {
		t.Error("Expected 2 failure types")
	}

	if scenario.FailureRate != 2.0 {
		t.Error("Expected failure rate to be 2.0")
	}
}

func TestStabilityTestScenario(t *testing.T) {
	scenario := &StabilityTestScenario{
		Name:                            "Test Stability",
		TestDurationHours:               24.0,
		SamplingIntervalSeconds:         30,
		MemoryLeakThresholdMB:           100.0,
		PerformanceDegradationThreshold: 20.0,
		ConnectionStabilityThreshold:    95.0,
		BaselineStabilizationMinutes:    30,
		ContinuousLoad:                  true,
		LoadVariation:                   false,
	}

	if scenario.TestDurationHours != 24.0 {
		t.Error("Expected test duration to be 24 hours")
	}

	if scenario.MemoryLeakThresholdMB != 100.0 {
		t.Error("Expected memory leak threshold to be 100MB")
	}

	if !scenario.ContinuousLoad {
		t.Error("Expected continuous load to be enabled")
	}
}

func TestResponseTimeMetricsCalculation(t *testing.T) {
	manager := &LoadTestManager{}
	
	times := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	
	metrics := manager.calculateResponseTimeMetrics(times)
	
	if metrics == nil {
		t.Fatal("Expected response time metrics to be calculated")
	}
	
	if metrics.Mean != 5.5 {
		t.Errorf("Expected mean to be 5.5, got %f", metrics.Mean)
	}
	
	if metrics.Min != 1.0 {
		t.Errorf("Expected min to be 1.0, got %f", metrics.Min)
	}
	
	if metrics.Max != 10.0 {
		t.Errorf("Expected max to be 10.0, got %f", metrics.Max)
	}
}

func TestLatencyMetricsCalculation(t *testing.T) {
	manager := &LatencyTestManager{}
	
	latencies := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	
	metrics := manager.calculateLatencyMetrics(latencies)
	
	if metrics == nil {
		t.Fatal("Expected latency metrics to be calculated")
	}
	
	if metrics.Count != 10 {
		t.Errorf("Expected count to be 10, got %d", metrics.Count)
	}
	
	if metrics.Mean != 5.5 {
		t.Errorf("Expected mean to be 5.5, got %f", metrics.Mean)
	}
	
	if metrics.Min != 1.0 {
		t.Errorf("Expected min to be 1.0, got %f", metrics.Min)
	}
	
	if metrics.Max != 10.0 {
		t.Errorf("Expected max to be 10.0, got %f", metrics.Max)
	}
}

func TestLatencyBucketCreation(t *testing.T) {
	manager := &LatencyTestManager{}
	
	distribution := map[float64]int64{
		1.0:  10,
		5.0:  20,
		10.0: 30,
		50.0: 5,
	}
	
	buckets := manager.createLatencyBuckets(distribution)
	
	if len(buckets) != 4 {
		t.Errorf("Expected 4 buckets, got %d", len(buckets))
	}
	
	totalCount := int64(0)
	for _, bucket := range buckets {
		totalCount += bucket.Count
	}
	
	if totalCount != 65 {
		t.Errorf("Expected total count to be 65, got %d", totalCount)
	}
	
	// Check that buckets are sorted
	for i := 1; i < len(buckets); i++ {
		if buckets[i-1].UpperBoundMs > buckets[i].UpperBoundMs {
			t.Error("Expected buckets to be sorted by upper bound")
		}
	}
}

func TestPerformanceTestConfig(t *testing.T) {
	config := &PerformanceTestConfig{
		MaxE2Nodes:           100,
		MaxConcurrentSubs:    1000,
		TargetThroughput:     10000,
		MaxLatencyMs:         10,
		TestDurationMinutes:  30,
		StabilityTestHours:   24,
		MemoryLeakThresholdMB: 100,
		CPUThresholdPercent:  80.0,
		LoadRampUpDuration:   5 * time.Minute,
	}

	if config.MaxE2Nodes != 100 {
		t.Error("Expected MaxE2Nodes to be 100")
	}

	if config.TargetThroughput != 10000 {
		t.Error("Expected TargetThroughput to be 10000")
	}

	if config.LoadRampUpDuration != 5*time.Minute {
		t.Error("Expected LoadRampUpDuration to be 5 minutes")
	}
}

func TestTestResultsStructure(t *testing.T) {
	results := &TestResults{
		LoadTestResults:     &LoadTestResults{},
		ThroughputResults:   &ThroughputResults{},
		LatencyResults:      &LatencyResults{},
		StressTestResults:   &StressTestResults{},
		StabilityResults:    &StabilityResults{},
		ResourceUtilization: &ResourceUtilization{},
		TestSummary:         &TestSummary{},
	}

	if results.LoadTestResults == nil {
		t.Error("Expected LoadTestResults to be initialized")
	}

	if results.ThroughputResults == nil {
		t.Error("Expected ThroughputResults to be initialized")
	}

	if results.LatencyResults == nil {
		t.Error("Expected LatencyResults to be initialized")
	}

	if results.StressTestResults == nil {
		t.Error("Expected StressTestResults to be initialized")
	}

	if results.StabilityResults == nil {
		t.Error("Expected StabilityResults to be initialized")
	}
}

// Test comprehensive performance test runner
func TestPerformanceTestRunner(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	runner := NewPerformanceTestRunner(e2Manager, subManager, prometheusClient)

	if runner == nil {
		t.Fatal("Expected performance test runner to be created")
	}

	if runner.validationRules.MinConcurrentE2Nodes != 100 {
		t.Errorf("Expected MinConcurrentE2Nodes to be 100, got %d", runner.validationRules.MinConcurrentE2Nodes)
	}

	if runner.validationRules.MinThroughputIPS != 10000 {
		t.Errorf("Expected MinThroughputIPS to be 10000, got %d", runner.validationRules.MinThroughputIPS)
	}

	if runner.validationRules.MaxLatencyMs != 10.0 {
		t.Errorf("Expected MaxLatencyMs to be 10.0, got %f", runner.validationRules.MaxLatencyMs)
	}
}

func TestValidationRules(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	runner := NewPerformanceTestRunner(e2Manager, subManager, prometheusClient)

	// Test concurrent E2 nodes validation
	loadResults := &LoadTestResults{
		MaxConcurrentE2Nodes: 120, // Above requirement
	}
	
	validation := runner.validateConcurrentE2Nodes(loadResults)
	if !validation.RequirementMet {
		t.Error("Expected concurrent E2 nodes requirement to be met")
	}
	if validation.PerformanceGap != 20.0 {
		t.Errorf("Expected performance gap to be 20.0, got %f", validation.PerformanceGap)
	}

	// Test throughput validation
	throughputResults := &ThroughputResults{
		MaxIndicationsPerSecond: 15000, // Above requirement
	}
	
	validation = runner.validateThroughput(throughputResults)
	if !validation.RequirementMet {
		t.Error("Expected throughput requirement to be met")
	}
	if validation.PerformanceGap != 5000.0 {
		t.Errorf("Expected performance gap to be 5000.0, got %f", validation.PerformanceGap)
	}

	// Test latency validation
	latencyResults := &LatencyResults{
		EndToEndLatencyMs: &LatencyMetrics{
			P99: 8.5, // Below requirement (good)
		},
	}
	
	validation = runner.validateLatency(latencyResults)
	if !validation.RequirementMet {
		t.Error("Expected latency requirement to be met")
	}
	if validation.PerformanceGap != 1.5 {
		t.Errorf("Expected performance gap to be 1.5, got %f", validation.PerformanceGap)
	}
}

func TestRequirementCompliance(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	runner := NewPerformanceTestRunner(e2Manager, subManager, prometheusClient)

	// Create mock validation results
	validationResults := &ValidationResults{
		ConcurrentE2NodesTest: &ValidationResult{RequirementMet: true},
		ThroughputTest:        &ValidationResult{RequirementMet: true},
		LatencyTest:          &ValidationResult{RequirementMet: true},
		StressTest:           &ValidationResult{RequirementMet: true},
		StabilityTest:        &ValidationResult{RequirementMet: true},
		MemoryLeakTest:       &ValidationResult{RequirementMet: true},
	}

	compliance := runner.generateComplianceReport(validationResults)

	if !compliance.Requirement61_Latency {
		t.Error("Expected Requirement 6.1 (Latency) to be met")
	}

	if !compliance.Requirement62_Scalability {
		t.Error("Expected Requirement 6.2 (Scalability) to be met")
	}

	if compliance.OverallCompliance != 100.0 {
		t.Errorf("Expected overall compliance to be 100%%, got %.1f%%", compliance.OverallCompliance)
	}
}

func TestEnhancedLoadTestScenarios(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)

	// Verify enhanced configuration
	if suite.config.MaxE2Nodes < 100 {
		t.Errorf("Expected MaxE2Nodes to be at least 100 for requirement compliance, got %d", suite.config.MaxE2Nodes)
	}

	if suite.config.TargetThroughput < 10000 {
		t.Errorf("Expected TargetThroughput to be at least 10000 for requirement compliance, got %d", suite.config.TargetThroughput)
	}

	if suite.config.MaxLatencyMs > 10 {
		t.Errorf("Expected MaxLatencyMs to be at most 10 for sub-10ms requirement, got %d", suite.config.MaxLatencyMs)
	}
}

func TestStabilityTestConfiguration(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewStabilityTestManager(e2Manager, subManager)

	if manager.memoryTracker.leakThresholdMB != 100 {
		t.Errorf("Expected memory leak threshold to be 100MB, got %.1f", manager.memoryTracker.leakThresholdMB)
	}

	if manager.memoryTracker.samplingInterval != 30*time.Second {
		t.Errorf("Expected sampling interval to be 30s, got %v", manager.memoryTracker.samplingInterval)
	}
}

func TestDetailedMetricsExtraction(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	runner := NewPerformanceTestRunner(e2Manager, subManager, prometheusClient)

	// Create mock test results
	results := &TestResults{
		LoadTestResults: &LoadTestResults{
			MaxConcurrentE2Nodes: 150,
		},
		ThroughputResults: &ThroughputResults{
			MaxIndicationsPerSecond: 25000,
		},
		LatencyResults: &LatencyResults{
			EndToEndLatencyMs: &LatencyMetrics{
				Min: 1.2,
				Max: 9.8,
				P99: 7.5,
			},
		},
		StabilityResults: &StabilityResults{
			MemoryLeakDetected:   false,
			TestDurationHours:    48.0,
			MemoryUsageOverTime: []MemorySample{
				{UsageMB: 500.0},
				{UsageMB: 750.0},
				{UsageMB: 600.0},
			},
		},
		StressTestResults: &StressTestResults{
			ResourceExhaustionPoint: &ResourceExhaustionPoint{
				CPUPercent: 95.0,
				MemoryMB:   4000.0,
			},
			RecoveryMetrics: &RecoveryMetrics{
				SuccessfulRecoveries: 8,
				FailedRecoveries:     2,
			},
		},
	}

	metrics := runner.extractDetailedMetrics(results)

	if metrics.PeakConcurrentE2Nodes != 150 {
		t.Errorf("Expected peak concurrent E2 nodes to be 150, got %d", metrics.PeakConcurrentE2Nodes)
	}

	if metrics.PeakThroughputIPS != 25000 {
		t.Errorf("Expected peak throughput to be 25000, got %d", metrics.PeakThroughputIPS)
	}

	if metrics.P99LatencyMs != 7.5 {
		t.Errorf("Expected P99 latency to be 7.5ms, got %.1f", metrics.P99LatencyMs)
	}

	if metrics.MaxMemoryUsageMB != 750.0 {
		t.Errorf("Expected max memory usage to be 750MB, got %.1f", metrics.MaxMemoryUsageMB)
	}

	if metrics.FailureRecoveryRate != 80.0 {
		t.Errorf("Expected failure recovery rate to be 80%%, got %.1f", metrics.FailureRecoveryRate)
	}
}

func TestGradeCalculation(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	runner := NewPerformanceTestRunner(e2Manager, subManager, prometheusClient)

	// Test different compliance scores
	testCases := []struct {
		compliance float64
		expectedGrade string
	}{
		{100.0, "A+"},
		{95.0, "A+"},
		{92.0, "A"},
		{87.0, "A-"},
		{82.0, "B+"},
		{77.0, "B"},
		{72.0, "B-"},
		{67.0, "C+"},
		{62.0, "C"},
		{57.0, "C-"},
		{52.0, "D"},
		{40.0, "F"},
	}

	for _, tc := range testCases {
		runner.testReport = &ComprehensiveTestReport{
			RequirementCompliance: &RequirementCompliance{
				OverallCompliance: tc.compliance,
			},
		}
		
		runner.calculateOverallAssessment()
		
		if runner.testReport.OverallGrade != tc.expectedGrade {
			t.Errorf("For compliance %.1f%%, expected grade %s, got %s", 
				tc.compliance, tc.expectedGrade, runner.testReport.OverallGrade)
		}
	}
}