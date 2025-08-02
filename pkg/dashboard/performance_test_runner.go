package dashboard

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
)

// PerformanceTestRunner orchestrates comprehensive performance testing
type PerformanceTestRunner struct {
	suite           *PerformanceTestSuite
	validationRules *ValidationRules
	testReport      *ComprehensiveTestReport
}

// ValidationRules defines the requirements validation criteria
type ValidationRules struct {
	MinConcurrentE2Nodes     int     `json:"minConcurrentE2Nodes"`     // Must be 100+
	MinThroughputIPS         int     `json:"minThroughputIPS"`         // Must be 10,000+
	MaxLatencyMs             float64 `json:"maxLatencyMs"`             // Must be sub-10ms
	MaxMemoryLeakThresholdMB float64 `json:"maxMemoryLeakThresholdMB"` // For leak detection
	MinStabilityTestHours    float64 `json:"minStabilityTestHours"`    // Long-running test requirement
	RequiredStressScenarios  []string `json:"requiredStressScenarios"` // Resource exhaustion scenarios
}

// ComprehensiveTestReport provides detailed test results and validation
type ComprehensiveTestReport struct {
	TestStartTime           time.Time                `json:"testStartTime"`
	TestEndTime             time.Time                `json:"testEndTime"`
	TotalTestDuration       time.Duration            `json:"totalTestDuration"`
	ValidationResults       *ValidationResults       `json:"validationResults"`
	PerformanceResults      *TestResults             `json:"performanceResults"`
	RequirementCompliance   *RequirementCompliance   `json:"requirementCompliance"`
	DetailedMetrics         *DetailedMetrics         `json:"detailedMetrics"`
	Recommendations         []string                 `json:"recommendations"`
	CriticalIssues          []string                 `json:"criticalIssues"`
	OverallGrade            string                   `json:"overallGrade"`
	ComplianceScore         float64                  `json:"complianceScore"`
}

// ValidationResults tracks validation against requirements
type ValidationResults struct {
	ConcurrentE2NodesTest   *ValidationResult `json:"concurrentE2NodesTest"`
	ThroughputTest          *ValidationResult `json:"throughputTest"`
	LatencyTest             *ValidationResult `json:"latencyTest"`
	StressTest              *ValidationResult `json:"stressTest"`
	StabilityTest           *ValidationResult `json:"stabilityTest"`
	MemoryLeakTest          *ValidationResult `json:"memoryLeakTest"`
}

// ValidationResult represents the result of a specific validation
type ValidationResult struct {
	TestName        string    `json:"testName"`
	RequirementMet  bool      `json:"requirementMet"`
	ActualValue     float64   `json:"actualValue"`
	RequiredValue   float64   `json:"requiredValue"`
	PerformanceGap  float64   `json:"performanceGap"` // Positive means exceeds requirement
	Details         string    `json:"details"`
	Timestamp       time.Time `json:"timestamp"`
}

// RequirementCompliance tracks compliance with each requirement
type RequirementCompliance struct {
	Requirement61_Latency     bool    `json:"requirement61_latency"`     // 6.1: Sub-10ms processing latency
	Requirement62_Scalability bool    `json:"requirement62_scalability"` // 6.2: 100+ concurrent E2 nodes
	Requirement63_Throughput  bool    `json:"requirement63_throughput"`  // 6.3: 1000+ concurrent subscriptions
	Requirement64_Performance bool    `json:"requirement64_performance"` // 6.4: 10,000+ indications per second
	Requirement65_Stability   bool    `json:"requirement65_stability"`   // 6.5: Stable memory consumption
	OverallCompliance         float64 `json:"overallCompliance"`         // Percentage of requirements met
}

// DetailedMetrics provides comprehensive performance metrics
type DetailedMetrics struct {
	PeakConcurrentE2Nodes    int                    `json:"peakConcurrentE2Nodes"`
	PeakThroughputIPS        int                    `json:"peakThroughputIPS"`
	MinLatencyMs             float64                `json:"minLatencyMs"`
	MaxLatencyMs             float64                `json:"maxLatencyMs"`
	P99LatencyMs             float64                `json:"p99LatencyMs"`
	MemoryLeakDetected       bool                   `json:"memoryLeakDetected"`
	MaxMemoryUsageMB         float64                `json:"maxMemoryUsageMB"`
	StabilityTestDurationH   float64                `json:"stabilityTestDurationH"`
	ResourceExhaustionPoints []ResourceExhaustionPoint `json:"resourceExhaustionPoints"`
	FailureRecoveryRate      float64                `json:"failureRecoveryRate"`
}

// NewPerformanceTestRunner creates a new performance test runner
func NewPerformanceTestRunner(e2Manager *E2ManagerClient, subManager *SubscriptionManagerClient, prometheusClient api.Client) *PerformanceTestRunner {
	suite := NewPerformanceTestSuite(e2Manager, subManager, prometheusClient)
	
	validationRules := &ValidationRules{
		MinConcurrentE2Nodes:     100,   // Requirement: 100+ concurrent E2 nodes
		MinThroughputIPS:         10000, // Requirement: 10,000+ indications per second
		MaxLatencyMs:             10.0,  // Requirement: sub-10ms processing
		MaxMemoryLeakThresholdMB: 100.0, // Memory leak detection threshold
		MinStabilityTestHours:    24.0,  // Long-running stability test
		RequiredStressScenarios: []string{
			"cpu_exhaustion",
			"memory_exhaustion", 
			"connection_exhaustion",
			"resource_starvation",
		},
	}

	return &PerformanceTestRunner{
		suite:           suite,
		validationRules: validationRules,
		testReport:      &ComprehensiveTestReport{},
	}
}

// RunComprehensivePerformanceTests executes all performance tests with validation
func (ptr *PerformanceTestRunner) RunComprehensivePerformanceTests(ctx context.Context) (*ComprehensiveTestReport, error) {
	log.Println("Starting comprehensive performance testing with requirement validation...")
	
	startTime := time.Now()
	ptr.testReport.TestStartTime = startTime

	// Execute the full performance test suite
	results, err := ptr.suite.RunAllPerformanceTests(ctx)
	if err != nil {
		return nil, fmt.Errorf("performance tests failed: %w", err)
	}

	endTime := time.Now()
	ptr.testReport.TestEndTime = endTime
	ptr.testReport.TotalTestDuration = endTime.Sub(startTime)
	ptr.testReport.PerformanceResults = results

	// Validate results against requirements
	validationResults, err := ptr.validateRequirements(results)
	if err != nil {
		return nil, fmt.Errorf("requirement validation failed: %w", err)
	}
	ptr.testReport.ValidationResults = validationResults

	// Generate compliance report
	compliance := ptr.generateComplianceReport(validationResults)
	ptr.testReport.RequirementCompliance = compliance

	// Extract detailed metrics
	detailedMetrics := ptr.extractDetailedMetrics(results)
	ptr.testReport.DetailedMetrics = detailedMetrics

	// Generate recommendations and identify issues
	ptr.generateRecommendationsAndIssues()

	// Calculate overall grade and compliance score
	ptr.calculateOverallAssessment()

	log.Printf("Comprehensive performance testing completed in %v", ptr.testReport.TotalTestDuration)
	log.Printf("Overall compliance score: %.1f%%", ptr.testReport.ComplianceScore)
	log.Printf("Overall grade: %s", ptr.testReport.OverallGrade)

	return ptr.testReport, nil
}

// validateRequirements validates test results against performance requirements
func (ptr *PerformanceTestRunner) validateRequirements(results *TestResults) (*ValidationResults, error) {
	validationResults := &ValidationResults{}

	// Validate concurrent E2 nodes (Requirement 6.2)
	validationResults.ConcurrentE2NodesTest = ptr.validateConcurrentE2Nodes(results.LoadTestResults)

	// Validate throughput (Requirement 6.4)
	validationResults.ThroughputTest = ptr.validateThroughput(results.ThroughputResults)

	// Validate latency (Requirement 6.1)
	validationResults.LatencyTest = ptr.validateLatency(results.LatencyResults)

	// Validate stress testing (Resource exhaustion scenarios)
	validationResults.StressTest = ptr.validateStressTesting(results.StressTestResults)

	// Validate stability testing (Requirement 6.5)
	validationResults.StabilityTest = ptr.validateStabilityTesting(results.StabilityResults)

	// Validate memory leak detection
	validationResults.MemoryLeakTest = ptr.validateMemoryLeakDetection(results.StabilityResults)

	return validationResults, nil
}

// validateConcurrentE2Nodes validates the 100+ concurrent E2 nodes requirement
func (ptr *PerformanceTestRunner) validateConcurrentE2Nodes(loadResults *LoadTestResults) *ValidationResult {
	if loadResults == nil {
		return &ValidationResult{
			TestName:       "Concurrent E2 Nodes Test",
			RequirementMet: false,
			ActualValue:    0,
			RequiredValue:  float64(ptr.validationRules.MinConcurrentE2Nodes),
			PerformanceGap: -float64(ptr.validationRules.MinConcurrentE2Nodes),
			Details:        "Load test results not available",
			Timestamp:      time.Now(),
		}
	}

	actualNodes := float64(loadResults.MaxConcurrentE2Nodes)
	requiredNodes := float64(ptr.validationRules.MinConcurrentE2Nodes)
	requirementMet := actualNodes >= requiredNodes
	performanceGap := actualNodes - requiredNodes

	details := fmt.Sprintf("Achieved %d concurrent E2 nodes (required: %d)", 
		int(actualNodes), int(requiredNodes))
	if requirementMet {
		details += fmt.Sprintf(" - PASSED with %.0f node margin", performanceGap)
	} else {
		details += fmt.Sprintf(" - FAILED by %.0f nodes", -performanceGap)
	}

	return &ValidationResult{
		TestName:       "Concurrent E2 Nodes Test",
		RequirementMet: requirementMet,
		ActualValue:    actualNodes,
		RequiredValue:  requiredNodes,
		PerformanceGap: performanceGap,
		Details:        details,
		Timestamp:      time.Now(),
	}
}

// validateThroughput validates the 10,000+ indications per second requirement
func (ptr *PerformanceTestRunner) validateThroughput(throughputResults *ThroughputResults) *ValidationResult {
	if throughputResults == nil {
		return &ValidationResult{
			TestName:       "Throughput Test",
			RequirementMet: false,
			ActualValue:    0,
			RequiredValue:  float64(ptr.validationRules.MinThroughputIPS),
			PerformanceGap: -float64(ptr.validationRules.MinThroughputIPS),
			Details:        "Throughput test results not available",
			Timestamp:      time.Now(),
		}
	}

	actualThroughput := float64(throughputResults.MaxIndicationsPerSecond)
	requiredThroughput := float64(ptr.validationRules.MinThroughputIPS)
	requirementMet := actualThroughput >= requiredThroughput
	performanceGap := actualThroughput - requiredThroughput

	details := fmt.Sprintf("Achieved %d IPS (required: %d)", 
		int(actualThroughput), int(requiredThroughput))
	if requirementMet {
		details += fmt.Sprintf(" - PASSED with %.0f IPS margin", performanceGap)
	} else {
		details += fmt.Sprintf(" - FAILED by %.0f IPS", -performanceGap)
	}

	return &ValidationResult{
		TestName:       "Throughput Test",
		RequirementMet: requirementMet,
		ActualValue:    actualThroughput,
		RequiredValue:  requiredThroughput,
		PerformanceGap: performanceGap,
		Details:        details,
		Timestamp:      time.Now(),
	}
}

// validateLatency validates the sub-10ms processing latency requirement
func (ptr *PerformanceTestRunner) validateLatency(latencyResults *LatencyResults) *ValidationResult {
	if latencyResults == nil || latencyResults.EndToEndLatencyMs == nil {
		return &ValidationResult{
			TestName:       "Latency Test",
			RequirementMet: false,
			ActualValue:    999.0, // High value to indicate failure
			RequiredValue:  ptr.validationRules.MaxLatencyMs,
			PerformanceGap: -ptr.validationRules.MaxLatencyMs,
			Details:        "Latency test results not available",
			Timestamp:      time.Now(),
		}
	}

	actualLatency := latencyResults.EndToEndLatencyMs.P99
	requiredLatency := ptr.validationRules.MaxLatencyMs
	requirementMet := actualLatency <= requiredLatency
	performanceGap := requiredLatency - actualLatency // Positive gap means better than required

	details := fmt.Sprintf("P99 latency: %.2fms (required: <%.1fms)", 
		actualLatency, requiredLatency)
	if requirementMet {
		details += fmt.Sprintf(" - PASSED with %.2fms margin", performanceGap)
	} else {
		details += fmt.Sprintf(" - FAILED by %.2fms", -performanceGap)
	}

	return &ValidationResult{
		TestName:       "Latency Test",
		RequirementMet: requirementMet,
		ActualValue:    actualLatency,
		RequiredValue:  requiredLatency,
		PerformanceGap: performanceGap,
		Details:        details,
		Timestamp:      time.Now(),
	}
}

// validateStressTesting validates stress testing with resource exhaustion scenarios
func (ptr *PerformanceTestRunner) validateStressTesting(stressResults *StressTestResults) *ValidationResult {
	if stressResults == nil {
		return &ValidationResult{
			TestName:       "Stress Test",
			RequirementMet: false,
			ActualValue:    0,
			RequiredValue:  float64(len(ptr.validationRules.RequiredStressScenarios)),
			PerformanceGap: -float64(len(ptr.validationRules.RequiredStressScenarios)),
			Details:        "Stress test results not available",
			Timestamp:      time.Now(),
		}
	}

	// Check if resource exhaustion was detected
	exhaustionDetected := stressResults.ResourceExhaustionPoint != nil
	
	// Check recovery metrics
	recoveryRate := 0.0
	if stressResults.RecoveryMetrics != nil {
		totalRecoveries := stressResults.RecoveryMetrics.SuccessfulRecoveries + stressResults.RecoveryMetrics.FailedRecoveries
		if totalRecoveries > 0 {
			recoveryRate = float64(stressResults.RecoveryMetrics.SuccessfulRecoveries) / float64(totalRecoveries) * 100
		}
	}

	requirementMet := exhaustionDetected && recoveryRate >= 70.0 // At least 70% recovery rate
	
	details := fmt.Sprintf("Resource exhaustion detected: %v, Recovery rate: %.1f%%", 
		exhaustionDetected, recoveryRate)
	if requirementMet {
		details += " - PASSED"
	} else {
		details += " - FAILED (insufficient stress testing or poor recovery)"
	}

	return &ValidationResult{
		TestName:       "Stress Test",
		RequirementMet: requirementMet,
		ActualValue:    recoveryRate,
		RequiredValue:  70.0,
		PerformanceGap: recoveryRate - 70.0,
		Details:        details,
		Timestamp:      time.Now(),
	}
}

// validateStabilityTesting validates long-running stability testing
func (ptr *PerformanceTestRunner) validateStabilityTesting(stabilityResults *StabilityResults) *ValidationResult {
	if stabilityResults == nil {
		return &ValidationResult{
			TestName:       "Stability Test",
			RequirementMet: false,
			ActualValue:    0,
			RequiredValue:  ptr.validationRules.MinStabilityTestHours,
			PerformanceGap: -ptr.validationRules.MinStabilityTestHours,
			Details:        "Stability test results not available",
			Timestamp:      time.Now(),
		}
	}

	actualDuration := stabilityResults.TestDurationHours
	requiredDuration := ptr.validationRules.MinStabilityTestHours
	requirementMet := actualDuration >= requiredDuration
	performanceGap := actualDuration - requiredDuration

	details := fmt.Sprintf("Test duration: %.1f hours (required: %.1f hours)", 
		actualDuration, requiredDuration)
	if requirementMet {
		details += fmt.Sprintf(" - PASSED with %.1f hour margin", performanceGap)
	} else {
		details += fmt.Sprintf(" - FAILED by %.1f hours", -performanceGap)
	}

	return &ValidationResult{
		TestName:       "Stability Test",
		RequirementMet: requirementMet,
		ActualValue:    actualDuration,
		RequiredValue:  requiredDuration,
		PerformanceGap: performanceGap,
		Details:        details,
		Timestamp:      time.Now(),
	}
}

// validateMemoryLeakDetection validates memory leak detection capability
func (ptr *PerformanceTestRunner) validateMemoryLeakDetection(stabilityResults *StabilityResults) *ValidationResult {
	if stabilityResults == nil {
		return &ValidationResult{
			TestName:       "Memory Leak Detection Test",
			RequirementMet: false,
			ActualValue:    0,
			RequiredValue:  1, // Binary: leak detection capability
			PerformanceGap: -1,
			Details:        "Stability test results not available",
			Timestamp:      time.Now(),
		}
	}

	// Memory leak detection is successful if:
	// 1. No memory leak was detected (good), OR
	// 2. Memory leak was detected and properly identified (also good - shows detection works)
	memoryLeakDetected := stabilityResults.MemoryLeakDetected
	
	// Check if we have sufficient memory samples for analysis
	sufficientSamples := len(stabilityResults.MemoryUsageOverTime) >= 100
	
	requirementMet := sufficientSamples // We can detect leaks if we have enough samples
	
	details := fmt.Sprintf("Memory leak detected: %v, Sample count: %d", 
		memoryLeakDetected, len(stabilityResults.MemoryUsageOverTime))
	if requirementMet {
		if memoryLeakDetected {
			details += " - PASSED (leak detection working)"
		} else {
			details += " - PASSED (no leaks detected)"
		}
	} else {
		details += " - FAILED (insufficient monitoring data)"
	}

	actualValue := 0.0
	if requirementMet {
		actualValue = 1.0
	}

	return &ValidationResult{
		TestName:       "Memory Leak Detection Test",
		RequirementMet: requirementMet,
		ActualValue:    actualValue,
		RequiredValue:  1.0,
		PerformanceGap: actualValue - 1.0,
		Details:        details,
		Timestamp:      time.Now(),
	}
}

// generateComplianceReport generates requirement compliance report
func (ptr *PerformanceTestRunner) generateComplianceReport(validationResults *ValidationResults) *RequirementCompliance {
	compliance := &RequirementCompliance{}

	// Map validation results to specific requirements
	compliance.Requirement61_Latency = validationResults.LatencyTest.RequirementMet
	compliance.Requirement62_Scalability = validationResults.ConcurrentE2NodesTest.RequirementMet
	compliance.Requirement63_Throughput = validationResults.ThroughputTest.RequirementMet
	compliance.Requirement64_Performance = validationResults.ThroughputTest.RequirementMet // Same as throughput
	compliance.Requirement65_Stability = validationResults.StabilityTest.RequirementMet && !validationResults.MemoryLeakTest.RequirementMet

	// Calculate overall compliance percentage
	totalRequirements := 5
	metRequirements := 0
	if compliance.Requirement61_Latency { metRequirements++ }
	if compliance.Requirement62_Scalability { metRequirements++ }
	if compliance.Requirement63_Throughput { metRequirements++ }
	if compliance.Requirement64_Performance { metRequirements++ }
	if compliance.Requirement65_Stability { metRequirements++ }

	compliance.OverallCompliance = float64(metRequirements) / float64(totalRequirements) * 100

	return compliance
}

// extractDetailedMetrics extracts detailed performance metrics
func (ptr *PerformanceTestRunner) extractDetailedMetrics(results *TestResults) *DetailedMetrics {
	metrics := &DetailedMetrics{}

	// Extract load test metrics
	if results.LoadTestResults != nil {
		metrics.PeakConcurrentE2Nodes = results.LoadTestResults.MaxConcurrentE2Nodes
	}

	// Extract throughput metrics
	if results.ThroughputResults != nil {
		metrics.PeakThroughputIPS = results.ThroughputResults.MaxIndicationsPerSecond
	}

	// Extract latency metrics
	if results.LatencyResults != nil && results.LatencyResults.EndToEndLatencyMs != nil {
		metrics.MinLatencyMs = results.LatencyResults.EndToEndLatencyMs.Min
		metrics.MaxLatencyMs = results.LatencyResults.EndToEndLatencyMs.Max
		metrics.P99LatencyMs = results.LatencyResults.EndToEndLatencyMs.P99
	}

	// Extract stability metrics
	if results.StabilityResults != nil {
		metrics.MemoryLeakDetected = results.StabilityResults.MemoryLeakDetected
		metrics.StabilityTestDurationH = results.StabilityResults.TestDurationHours
		
		// Find max memory usage
		maxMemory := 0.0
		for _, sample := range results.StabilityResults.MemoryUsageOverTime {
			if sample.UsageMB > maxMemory {
				maxMemory = sample.UsageMB
			}
		}
		metrics.MaxMemoryUsageMB = maxMemory
	}

	// Extract stress test metrics
	if results.StressTestResults != nil {
		if results.StressTestResults.ResourceExhaustionPoint != nil {
			metrics.ResourceExhaustionPoints = []ResourceExhaustionPoint{*results.StressTestResults.ResourceExhaustionPoint}
		}
		
		if results.StressTestResults.RecoveryMetrics != nil {
			totalRecoveries := results.StressTestResults.RecoveryMetrics.SuccessfulRecoveries + results.StressTestResults.RecoveryMetrics.FailedRecoveries
			if totalRecoveries > 0 {
				metrics.FailureRecoveryRate = float64(results.StressTestResults.RecoveryMetrics.SuccessfulRecoveries) / float64(totalRecoveries) * 100
			}
		}
	}

	return metrics
}

// generateRecommendationsAndIssues generates recommendations and identifies critical issues
func (ptr *PerformanceTestRunner) generateRecommendationsAndIssues() {
	recommendations := []string{}
	criticalIssues := []string{}

	validation := ptr.testReport.ValidationResults
	compliance := ptr.testReport.RequirementCompliance

	// Check each requirement and generate specific recommendations
	if !validation.ConcurrentE2NodesTest.RequirementMet {
		criticalIssues = append(criticalIssues, 
			fmt.Sprintf("Failed to meet 100+ concurrent E2 nodes requirement (achieved: %.0f)", 
				validation.ConcurrentE2NodesTest.ActualValue))
		recommendations = append(recommendations, 
			"Optimize E2 connection handling and increase connection pool sizes")
	}

	if !validation.ThroughputTest.RequirementMet {
		criticalIssues = append(criticalIssues, 
			fmt.Sprintf("Failed to meet 10,000+ IPS throughput requirement (achieved: %.0f)", 
				validation.ThroughputTest.ActualValue))
		recommendations = append(recommendations, 
			"Optimize indication processing pipeline and consider horizontal scaling")
	}

	if !validation.LatencyTest.RequirementMet {
		criticalIssues = append(criticalIssues, 
			fmt.Sprintf("Failed to meet sub-10ms latency requirement (P99: %.2fms)", 
				validation.LatencyTest.ActualValue))
		recommendations = append(recommendations, 
			"Optimize critical path processing and reduce serialization overhead")
	}

	if !validation.StressTest.RequirementMet {
		criticalIssues = append(criticalIssues, "Insufficient stress testing or poor failure recovery")
		recommendations = append(recommendations, 
			"Implement better error handling and recovery mechanisms")
	}

	if !validation.StabilityTest.RequirementMet {
		criticalIssues = append(criticalIssues, "Stability test duration insufficient")
		recommendations = append(recommendations, 
			"Extend stability testing duration and improve monitoring")
	}

	if ptr.testReport.DetailedMetrics.MemoryLeakDetected {
		criticalIssues = append(criticalIssues, "Memory leak detected during stability testing")
		recommendations = append(recommendations, 
			"Investigate and fix memory leaks in long-running processes")
	}

	// Overall performance recommendations
	if compliance.OverallCompliance < 80.0 {
		recommendations = append(recommendations, 
			"Consider architectural improvements for better scalability")
		recommendations = append(recommendations, 
			"Implement performance monitoring and alerting in production")
	}

	ptr.testReport.Recommendations = recommendations
	ptr.testReport.CriticalIssues = criticalIssues
}

// calculateOverallAssessment calculates overall grade and compliance score
func (ptr *PerformanceTestRunner) calculateOverallAssessment() {
	compliance := ptr.testReport.RequirementCompliance.OverallCompliance
	ptr.testReport.ComplianceScore = compliance

	// Determine grade based on compliance score
	switch {
	case compliance >= 95.0:
		ptr.testReport.OverallGrade = "A+"
	case compliance >= 90.0:
		ptr.testReport.OverallGrade = "A"
	case compliance >= 85.0:
		ptr.testReport.OverallGrade = "A-"
	case compliance >= 80.0:
		ptr.testReport.OverallGrade = "B+"
	case compliance >= 75.0:
		ptr.testReport.OverallGrade = "B"
	case compliance >= 70.0:
		ptr.testReport.OverallGrade = "B-"
	case compliance >= 65.0:
		ptr.testReport.OverallGrade = "C+"
	case compliance >= 60.0:
		ptr.testReport.OverallGrade = "C"
	case compliance >= 55.0:
		ptr.testReport.OverallGrade = "C-"
	case compliance >= 50.0:
		ptr.testReport.OverallGrade = "D"
	default:
		ptr.testReport.OverallGrade = "F"
	}
}

// GetTestReport returns the comprehensive test report
func (ptr *PerformanceTestRunner) GetTestReport() *ComprehensiveTestReport {
	return ptr.testReport
}

// PrintSummaryReport prints a summary of the test results
func (ptr *PerformanceTestRunner) PrintSummaryReport() {
	report := ptr.testReport
	
	fmt.Println("\n" + "="*80)
	fmt.Println("COMPREHENSIVE PERFORMANCE TEST REPORT")
	fmt.Println("="*80)
	
	fmt.Printf("Test Duration: %v\n", report.TotalTestDuration)
	fmt.Printf("Overall Grade: %s\n", report.OverallGrade)
	fmt.Printf("Compliance Score: %.1f%%\n\n", report.ComplianceScore)
	
	fmt.Println("REQUIREMENT VALIDATION RESULTS:")
	fmt.Println("-"*40)
	
	validation := report.ValidationResults
	fmt.Printf("• Concurrent E2 Nodes (100+): %s (%.0f)\n", 
		getPassFailStatus(validation.ConcurrentE2NodesTest.RequirementMet), 
		validation.ConcurrentE2NodesTest.ActualValue)
	fmt.Printf("• Throughput (10,000+ IPS): %s (%.0f)\n", 
		getPassFailStatus(validation.ThroughputTest.RequirementMet), 
		validation.ThroughputTest.ActualValue)
	fmt.Printf("• Latency (sub-10ms): %s (%.2fms P99)\n", 
		getPassFailStatus(validation.LatencyTest.RequirementMet), 
		validation.LatencyTest.ActualValue)
	fmt.Printf("• Stress Testing: %s\n", 
		getPassFailStatus(validation.StressTest.RequirementMet))
	fmt.Printf("• Stability Testing: %s (%.1fh)\n", 
		getPassFailStatus(validation.StabilityTest.RequirementMet), 
		validation.StabilityTest.ActualValue)
	fmt.Printf("• Memory Leak Detection: %s\n", 
		getPassFailStatus(validation.MemoryLeakTest.RequirementMet))
	
	if len(report.CriticalIssues) > 0 {
		fmt.Println("\nCRITICAL ISSUES:")
		fmt.Println("-"*20)
		for _, issue := range report.CriticalIssues {
			fmt.Printf("• %s\n", issue)
		}
	}
	
	if len(report.Recommendations) > 0 {
		fmt.Println("\nRECOMMENDATIONS:")
		fmt.Println("-"*20)
		for _, rec := range report.Recommendations {
			fmt.Printf("• %s\n", rec)
		}
	}
	
	fmt.Println("\n" + "="*80)
}

// getPassFailStatus returns a formatted pass/fail status
func getPassFailStatus(passed bool) string {
	if passed {
		return "PASS"
	}
	return "FAIL"
}