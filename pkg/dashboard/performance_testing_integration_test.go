package dashboard

import (
	"context"
	"testing"
	"time"
)

// TestPerformanceTestSuiteIntegration tests the complete performance test suite integration
func TestPerformanceTestSuiteIntegration(t *testing.T) {
	// Create mock clients
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	// Create performance test suite
	suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)
	if suite == nil {
		t.Fatal("Failed to create performance test suite")
	}

	// Verify enhanced configuration meets requirements
	if suite.config.MaxE2Nodes < 100 {
		t.Errorf("MaxE2Nodes should be at least 100 for requirement compliance, got %d", suite.config.MaxE2Nodes)
	}

	if suite.config.TargetThroughput < 10000 {
		t.Errorf("TargetThroughput should be at least 10000 for requirement compliance, got %d", suite.config.TargetThroughput)
	}

	if suite.config.MaxLatencyMs > 10 {
		t.Errorf("MaxLatencyMs should be at most 10 for sub-10ms requirement, got %d", suite.config.MaxLatencyMs)
	}

	// Test that all test managers can be created
	loadManager := NewLoadTestManager(e2Manager, subManager)
	if loadManager == nil {
		t.Error("Failed to create load test manager")
	}

	throughputManager := NewThroughputTestManager(e2Manager, subManager)
	if throughputManager == nil {
		t.Error("Failed to create throughput test manager")
	}

	latencyManager := NewLatencyTestManager(e2Manager, subManager)
	if latencyManager == nil {
		t.Error("Failed to create latency test manager")
	}

	stressManager := NewStressTestManager(e2Manager, subManager)
	if stressManager == nil {
		t.Error("Failed to create stress test manager")
	}

	stabilityManager := NewStabilityTestManager(e2Manager, subManager)
	if stabilityManager == nil {
		t.Error("Failed to create stability test manager")
	}
}

// TestPerformanceTestRunnerIntegration tests the performance test runner integration
func TestPerformanceTestRunnerIntegration(t *testing.T) {
	// Create mock clients
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	// Create performance test runner
	runner := NewPerformanceTestRunner(e2Manager, subManager, prometheusClient)
	if runner == nil {
		t.Fatal("Failed to create performance test runner")
	}

	// Verify validation rules meet requirements
	if runner.validationRules.MinConcurrentE2Nodes != 100 {
		t.Errorf("MinConcurrentE2Nodes should be 100, got %d", runner.validationRules.MinConcurrentE2Nodes)
	}

	if runner.validationRules.MinThroughputIPS != 10000 {
		t.Errorf("MinThroughputIPS should be 10000, got %d", runner.validationRules.MinThroughputIPS)
	}

	if runner.validationRules.MaxLatencyMs != 10.0 {
		t.Errorf("MaxLatencyMs should be 10.0, got %f", runner.validationRules.MaxLatencyMs)
	}

	// Test validation functions with mock data
	loadResults := &LoadTestResults{
		MaxConcurrentE2Nodes: 150, // Above requirement
	}
	validation := runner.validateConcurrentE2Nodes(loadResults)
	if !validation.RequirementMet {
		t.Error("Expected concurrent E2 nodes requirement to be met with 150 nodes")
	}

	throughputResults := &ThroughputResults{
		MaxIndicationsPerSecond: 15000, // Above requirement
	}
	validation = runner.validateThroughput(throughputResults)
	if !validation.RequirementMet {
		t.Error("Expected throughput requirement to be met with 15000 IPS")
	}

	latencyResults := &LatencyResults{
		EndToEndLatencyMs: &LatencyMetrics{
			P99: 8.5, // Below requirement (good)
		},
	}
	validation = runner.validateLatency(latencyResults)
	if !validation.RequirementMet {
		t.Error("Expected latency requirement to be met with 8.5ms P99")
	}
}

// TestLoadTestScenarios tests load test scenario configuration
func TestLoadTestScenarios(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewLoadTestManager(e2Manager, subManager)

	// Test scenario creation
	scenario := &LoadTestScenario{
		Name:             "Test 100+ Nodes",
		MaxE2Nodes:       120, // Above requirement
		MaxSubscriptions: 1000,
		RampUpDuration:   5 * time.Minute,
		SustainDuration:  10 * time.Minute,
		RampDownDuration: 2 * time.Minute,
		ConnectionPattern: "linear",
	}

	if scenario.MaxE2Nodes < 100 {
		t.Errorf("Test scenario should have at least 100 E2 nodes, got %d", scenario.MaxE2Nodes)
	}

	// Test response time metrics calculation
	times := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	metrics := manager.calculateResponseTimeMetrics(times)

	if metrics.Mean != 5.5 {
		t.Errorf("Expected mean response time to be 5.5, got %f", metrics.Mean)
	}

	if metrics.P99 != 10.0 {
		t.Errorf("Expected P99 response time to be 10.0, got %f", metrics.P99)
	}
}

// TestThroughputTestScenarios tests throughput test scenario configuration
func TestThroughputTestScenarios(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewThroughputTestManager(e2Manager, subManager)

	// Test scenario creation
	scenario := &ThroughputTestScenario{
		Name:                  "Test 10K+ IPS",
		TargetThroughputIPS:   15000, // Above requirement
		RampUpDuration:        3 * time.Minute,
		SustainDuration:       10 * time.Minute,
		RampDownDuration:      2 * time.Minute,
		MaxQueueDepth:         2000,
		ProcessingComplexity:  "medium",
		BackpressureThreshold: 0.8,
	}

	if scenario.TargetThroughputIPS < 10000 {
		t.Errorf("Test scenario should have at least 10000 IPS, got %d", scenario.TargetThroughputIPS)
	}

	// Test indication processor initialization
	if manager.indicationProcessor == nil {
		t.Error("Indication processor should be initialized")
	}

	if manager.indicationProcessor.inputQueue == nil {
		t.Error("Input queue should be initialized")
	}
}

// TestLatencyTestScenarios tests latency test scenario configuration
func TestLatencyTestScenarios(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewLatencyTestManager(e2Manager, subManager)

	// Test scenario creation
	scenario := &LatencyTestScenario{
		Name:                 "Test Sub-10ms",
		MaxLatencyMs:         8.0, // Stricter than requirement
		OperationRate:        100,
		TestDuration:         5 * time.Minute,
		OperationTypes:       []string{"e2_setup", "subscription", "indication"},
		ConcurrentOperations: 20,
		LatencyTargets: map[string]float64{
			"e2_setup":     10.0,
			"subscription": 8.0,
			"indication":   5.0,
		},
	}

	if scenario.MaxLatencyMs > 10.0 {
		t.Errorf("Test scenario should have max latency ≤10ms, got %f", scenario.MaxLatencyMs)
	}

	// Test latency bucket creation
	distribution := map[float64]int64{
		1.0:  100,
		5.0:  200,
		10.0: 50,
	}

	buckets := manager.createLatencyBuckets(distribution)
	if len(buckets) != 3 {
		t.Errorf("Expected 3 latency buckets, got %d", len(buckets))
	}

	totalCount := int64(0)
	for _, bucket := range buckets {
		totalCount += bucket.Count
	}

	if totalCount != 350 {
		t.Errorf("Expected total count to be 350, got %d", totalCount)
	}
}

// TestStressTestScenarios tests stress test scenario configuration
func TestStressTestScenarios(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewStressTestManager(e2Manager, subManager)

	// Test scenario creation
	scenario := &StressTestScenario{
		Name:         "Resource Exhaustion Test",
		TestDuration: 20 * time.Minute,
		ResourceTargets: &ResourceTargets{
			MaxCPUPercent:    95.0,
			MaxMemoryMB:      4000,
			MaxGoroutines:    8000,
			MaxConnections:   500,
			MaxThroughputIPS: 20000,
		},
		FailureTypes:          []string{"cpu_spike", "memory_leak", "connection_leak"},
		FailureRate:           3.0,
		RecoveryTimeout:       30 * time.Second,
		MaxConcurrentFailures: 5,
		LoadIncreaseRate:      1.2,
	}

	// Test failure descriptions
	for _, failureType := range scenario.FailureTypes {
		desc := manager.getFailureDescription(failureType)
		if desc == "Unknown failure type" {
			t.Errorf("Expected description for failure type %s", failureType)
		}
	}

	// Test resource monitor initialization
	if manager.resourceMonitor == nil {
		t.Error("Resource monitor should be initialized")
	}

	if manager.failureInjector == nil {
		t.Error("Failure injector should be initialized")
	}
}

// TestStabilityTestScenarios tests stability test scenario configuration
func TestStabilityTestScenarios(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}

	manager := NewStabilityTestManager(e2Manager, subManager)

	// Test scenario creation
	scenario := &StabilityTestScenario{
		Name:                            "Memory Leak Detection Test",
		TestDurationHours:               24.0, // Long-running test
		SamplingIntervalSeconds:         30,
		MemoryLeakThresholdMB:           50.0,
		PerformanceDegradationThreshold: 20.0,
		ConnectionStabilityThreshold:    95.0,
		BaselineStabilizationMinutes:    30,
		ContinuousLoad:                  true,
		LoadVariation:                   false,
	}

	if scenario.TestDurationHours < 1.0 {
		t.Errorf("Stability test should run for at least 1 hour, got %f", scenario.TestDurationHours)
	}

	// Test memory tracker initialization
	if manager.memoryTracker == nil {
		t.Error("Memory tracker should be initialized")
	}

	if manager.connectionTracker == nil {
		t.Error("Connection tracker should be initialized")
	}

	if manager.performanceTracker == nil {
		t.Error("Performance tracker should be initialized")
	}
}

// TestRequirementValidation tests requirement validation logic
func TestRequirementValidation(t *testing.T) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	runner := NewPerformanceTestRunner(e2Manager, subManager, prometheusClient)

	// Create test results that meet all requirements
	results := &TestResults{
		LoadTestResults: &LoadTestResults{
			MaxConcurrentE2Nodes:    150, // Above 100 requirement
			ConnectionSuccessRate:   98.5,
			SubscriptionSuccessRate: 97.2,
		},
		ThroughputResults: &ThroughputResults{
			MaxIndicationsPerSecond: 18000, // Above 10000 requirement
			SustainedThroughput:     15000,
		},
		LatencyResults: &LatencyResults{
			EndToEndLatencyMs: &LatencyMetrics{
				P99: 7.8, // Below 10ms requirement
				P95: 6.2,
				Mean: 4.1,
			},
		},
		StressTestResults: &StressTestResults{
			ResourceExhaustionPoint: &ResourceExhaustionPoint{
				CPUPercent: 95.0,
				MemoryMB:   4000.0,
			},
			RecoveryMetrics: &RecoveryMetrics{
				SuccessfulRecoveries: 8,
				FailedRecoveries:     2, // 80% recovery rate
			},
		},
		StabilityResults: &StabilityResults{
			TestDurationHours:  48.0, // Above 24h requirement
			MemoryLeakDetected: false, // No leaks detected
		},
	}

	// Validate requirements
	validationResults, err := runner.validateRequirements(results)
	if err != nil {
		t.Fatalf("Requirement validation failed: %v", err)
	}

	// Check that all requirements are met
	if !validationResults.ConcurrentE2NodesTest.RequirementMet {
		t.Error("Expected concurrent E2 nodes requirement to be met")
	}

	if !validationResults.ThroughputTest.RequirementMet {
		t.Error("Expected throughput requirement to be met")
	}

	if !validationResults.LatencyTest.RequirementMet {
		t.Error("Expected latency requirement to be met")
	}

	if !validationResults.StressTest.RequirementMet {
		t.Error("Expected stress test requirement to be met")
	}

	if !validationResults.StabilityTest.RequirementMet {
		t.Error("Expected stability test requirement to be met")
	}

	// Generate compliance report
	compliance := runner.generateComplianceReport(validationResults)
	if compliance.OverallCompliance != 100.0 {
		t.Errorf("Expected 100%% compliance, got %.1f%%", compliance.OverallCompliance)
	}
}

// TestPerformanceTestExecution tests a short performance test execution
func TestPerformanceTestExecution(t *testing.T) {
	// Skip this test in short mode as it takes time
	if testing.Short() {
		t.Skip("Skipping performance test execution in short mode")
	}

	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)

	// Override configuration for faster testing
	suite.config.TestDurationMinutes = 1    // 1 minute instead of 60
	suite.config.StabilityTestHours = 0.1   // 6 minutes instead of 72 hours
	suite.config.LoadRampUpDuration = 30 * time.Second // 30 seconds instead of 10 minutes

	// Create a short-duration context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run a subset of tests
	results, err := suite.RunAllPerformanceTests(ctx)
	if err != nil {
		t.Fatalf("Performance tests failed: %v", err)
	}

	if results == nil {
		t.Fatal("Expected test results to be returned")
	}

	// Verify that test results contain expected data
	if results.TestSummary == nil {
		t.Error("Expected test summary to be generated")
	}

	if results.TestSummary.TestsExecuted == 0 {
		t.Error("Expected some tests to be executed")
	}

	// Check that the test took a reasonable amount of time
	if results.TestSummary.TotalTestDuration < 30*time.Second {
		t.Error("Expected test to take at least 30 seconds")
	}

	if results.TestSummary.TotalTestDuration > 10*time.Minute {
		t.Error("Expected test to complete within 10 minutes")
	}
}

// BenchmarkPerformanceTestSuite benchmarks the performance test suite creation
func BenchmarkPerformanceTestSuite(b *testing.B) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)
		_ = suite
	}
}

// BenchmarkResourceUsageTracking benchmarks resource usage tracking
func BenchmarkResourceUsageTracking(b *testing.B) {
	e2Manager := &MockE2ManagerClient{}
	subManager := &MockSubscriptionManagerClient{}
	prometheusClient := &MockPrometheusClient{}

	suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage := suite.GetCurrentResourceUsage()
		_ = usage
	}
}

// BenchmarkLatencyMeasurement benchmarks latency measurement operations
func BenchmarkLatencyMeasurement(b *testing.B) {
	manager := NewLatencyTestManager(&MockE2ManagerClient{}, &MockSubscriptionManagerClient{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.recordLatency("test_operation", float64(i%100))
	}
}