package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/oran/near-rt-ric-new/pkg/dashboard"
)

// PerformanceTestExecutor orchestrates comprehensive performance testing
type PerformanceTestExecutor struct {
	testRunner *dashboard.PerformanceTestRunner
	config     *TestExecutionConfig
}

// TestExecutionConfig defines test execution parameters
type TestExecutionConfig struct {
	RunLoadTests       bool
	RunThroughputTests bool
	RunLatencyTests    bool
	RunStressTests     bool
	RunStabilityTests  bool
	OutputFormat       string // "json", "text", "html"
	OutputFile         string
	Verbose            bool
	ContinueOnFailure  bool
}

func main() {
	// Parse command line flags
	config := parseFlags()

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, gracefully stopping tests...")
		cancel()
	}()

	// Initialize test executor
	executor, err := NewPerformanceTestExecutor(config)
	if err != nil {
		log.Fatalf("Failed to initialize performance test executor: %v", err)
	}

	// Execute comprehensive performance tests
	if err := executor.ExecuteComprehensiveTests(ctx); err != nil {
		log.Fatalf("Performance tests failed: %v", err)
	}

	log.Println("Performance testing completed successfully")
}

// parseFlags parses command line flags
func parseFlags() *TestExecutionConfig {
	config := &TestExecutionConfig{}

	flag.BoolVar(&config.RunLoadTests, "load", true, "Run load testing scenarios (100+ concurrent E2 nodes)")
	flag.BoolVar(&config.RunThroughputTests, "throughput", true, "Run throughput testing scenarios (10,000+ IPS)")
	flag.BoolVar(&config.RunLatencyTests, "latency", true, "Run latency testing scenarios (sub-10ms)")
	flag.BoolVar(&config.RunStressTests, "stress", true, "Run stress testing scenarios (resource exhaustion)")
	flag.BoolVar(&config.RunStabilityTests, "stability", true, "Run stability testing scenarios (memory leak detection)")
	flag.StringVar(&config.OutputFormat, "format", "text", "Output format: json, text, html")
	flag.StringVar(&config.OutputFile, "output", "", "Output file (default: stdout)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&config.ContinueOnFailure, "continue", false, "Continue testing even if some tests fail")

	flag.Parse()

	return config
}

// NewPerformanceTestExecutor creates a new performance test executor
func NewPerformanceTestExecutor(config *TestExecutionConfig) (*PerformanceTestExecutor, error) {
	// Initialize mock clients for testing
	e2Manager := &dashboard.MockE2ManagerClient{}
	subManager := &dashboard.MockSubscriptionManagerClient{}
	prometheusClient := &dashboard.MockPrometheusClient{}

	// Create performance test runner
	testRunner := dashboard.NewPerformanceTestRunner(e2Manager, subManager, prometheusClient)

	return &PerformanceTestExecutor{
		testRunner: testRunner,
		config:     config,
	}, nil
}

// ExecuteComprehensiveTests executes all performance tests based on configuration
func (pte *PerformanceTestExecutor) ExecuteComprehensiveTests(ctx context.Context) error {
	log.Println("Starting comprehensive performance testing...")
	log.Println(strings.Repeat("=", 80))
	
	// Print test configuration
	pte.printTestConfiguration()

	startTime := time.Now()

	// Execute the comprehensive test suite
	report, err := pte.testRunner.RunComprehensivePerformanceTests(ctx)
	if err != nil {
		return fmt.Errorf("comprehensive performance tests failed: %w", err)
	}

	totalDuration := time.Since(startTime)
	log.Printf("Total test execution time: %v", totalDuration)

	// Generate and output results
	if err := pte.generateTestReport(report); err != nil {
		return fmt.Errorf("failed to generate test report: %w", err)
	}

	// Validate requirements compliance
	if err := pte.validateRequirementsCompliance(report); err != nil {
		if !pte.config.ContinueOnFailure {
			return fmt.Errorf("requirements validation failed: %w", err)
		}
		log.Printf("Requirements validation failed but continuing: %v", err)
	}

	return nil
}

// printTestConfiguration prints the test configuration
func (pte *PerformanceTestExecutor) printTestConfiguration() {
	log.Println("TEST CONFIGURATION:")
	log.Println(strings.Repeat("-", 40))
	log.Printf("Load Tests (100+ E2 nodes): %v", pte.config.RunLoadTests)
	log.Printf("Throughput Tests (10,000+ IPS): %v", pte.config.RunThroughputTests)
	log.Printf("Latency Tests (sub-10ms): %v", pte.config.RunLatencyTests)
	log.Printf("Stress Tests (resource exhaustion): %v", pte.config.RunStressTests)
	log.Printf("Stability Tests (memory leak detection): %v", pte.config.RunStabilityTests)
	log.Printf("Output Format: %s", pte.config.OutputFormat)
	log.Printf("Verbose Logging: %v", pte.config.Verbose)
	log.Printf("Continue on Failure: %v", pte.config.ContinueOnFailure)
	log.Println(strings.Repeat("=", 80))
}

// generateTestReport generates and outputs the test report
func (pte *PerformanceTestExecutor) generateTestReport(report *dashboard.ComprehensiveTestReport) error {
	switch pte.config.OutputFormat {
	case "json":
		return pte.generateJSONReport(report)
	case "html":
		return pte.generateHTMLReport(report)
	default:
		return pte.generateTextReport(report)
	}
}

// generateTextReport generates a text-based test report
func (pte *PerformanceTestExecutor) generateTextReport(report *dashboard.ComprehensiveTestReport) error {
	// Print summary report to console
	pte.testRunner.PrintSummaryReport()

	// Generate detailed report
	detailedReport := pte.generateDetailedTextReport(report)

	if pte.config.OutputFile != "" {
		// Write to file
		file, err := os.Create(pte.config.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		if _, err := file.WriteString(detailedReport); err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}

		log.Printf("Detailed report written to: %s", pte.config.OutputFile)
	} else {
		// Print to console
		fmt.Println(detailedReport)
	}

	return nil
}

// generateDetailedTextReport generates a detailed text report
func (pte *PerformanceTestExecutor) generateDetailedTextReport(report *dashboard.ComprehensiveTestReport) string {
	var output string

	output += "\n" + strings.Repeat("=", 100) + "\n"
	output += "COMPREHENSIVE PERFORMANCE TEST REPORT - DETAILED RESULTS\n"
	output += strings.Repeat("=", 100) + "\n"

	// Test Summary
	output += fmt.Sprintf("Test Start Time: %s\n", report.TestStartTime.Format(time.RFC3339))
	output += fmt.Sprintf("Test End Time: %s\n", report.TestEndTime.Format(time.RFC3339))
	output += fmt.Sprintf("Total Duration: %v\n", report.TotalTestDuration)
	output += fmt.Sprintf("Overall Grade: %s\n", report.OverallGrade)
	output += fmt.Sprintf("Compliance Score: %.1f%%\n\n", report.ComplianceScore)

	// Requirement Validation Details
	output += "REQUIREMENT VALIDATION DETAILS:\n"
	output += strings.Repeat("-", 50) + "\n"

	if report.ValidationResults != nil {
		output += pte.formatValidationResult("Load Testing (100+ E2 Nodes)", report.ValidationResults.ConcurrentE2NodesTest)
		output += pte.formatValidationResult("Throughput Testing (10,000+ IPS)", report.ValidationResults.ThroughputTest)
		output += pte.formatValidationResult("Latency Testing (sub-10ms)", report.ValidationResults.LatencyTest)
		output += pte.formatValidationResult("Stress Testing", report.ValidationResults.StressTest)
		output += pte.formatValidationResult("Stability Testing", report.ValidationResults.StabilityTest)
		output += pte.formatValidationResult("Memory Leak Detection", report.ValidationResults.MemoryLeakTest)
	}

	// Detailed Performance Metrics
	output += "\nDETAILED PERFORMANCE METRICS:\n"
	output += strings.Repeat("-", 50) + "\n"

	if report.DetailedMetrics != nil {
		metrics := report.DetailedMetrics
		output += fmt.Sprintf("Peak Concurrent E2 Nodes: %d\n", metrics.PeakConcurrentE2Nodes)
		output += fmt.Sprintf("Peak Throughput: %d IPS\n", metrics.PeakThroughputIPS)
		output += fmt.Sprintf("Minimum Latency: %.2f ms\n", metrics.MinLatencyMs)
		output += fmt.Sprintf("Maximum Latency: %.2f ms\n", metrics.MaxLatencyMs)
		output += fmt.Sprintf("P99 Latency: %.2f ms\n", metrics.P99LatencyMs)
		output += fmt.Sprintf("Memory Leak Detected: %v\n", metrics.MemoryLeakDetected)
		output += fmt.Sprintf("Maximum Memory Usage: %.1f MB\n", metrics.MaxMemoryUsageMB)
		output += fmt.Sprintf("Stability Test Duration: %.1f hours\n", metrics.StabilityTestDurationH)
		output += fmt.Sprintf("Failure Recovery Rate: %.1f%%\n", metrics.FailureRecoveryRate)
	}

	// Resource Exhaustion Points
	if report.DetailedMetrics != nil && len(report.DetailedMetrics.ResourceExhaustionPoints) > 0 {
		output += "\nRESOURCE EXHAUSTION POINTS:\n"
		output += strings.Repeat("-", 50) + "\n"
		for i, point := range report.DetailedMetrics.ResourceExhaustionPoints {
			output += fmt.Sprintf("Exhaustion Point %d:\n", i+1)
			output += fmt.Sprintf("  CPU: %.1f%%\n", point.CPUPercent)
			output += fmt.Sprintf("  Memory: %.1f MB\n", point.MemoryMB)
			output += fmt.Sprintf("  Active E2 Nodes: %d\n", point.ActiveE2Nodes)
			output += fmt.Sprintf("  Active Subscriptions: %d\n", point.ActiveSubs)
			output += fmt.Sprintf("  Throughput: %d IPS\n", point.ThroughputIPS)
		}
	}

	// Critical Issues
	if len(report.CriticalIssues) > 0 {
		output += "\nCRITICAL ISSUES:\n"
		output += strings.Repeat("-", 30) + "\n"
		for i, issue := range report.CriticalIssues {
			output += fmt.Sprintf("%d. %s\n", i+1, issue)
		}
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		output += "\nRECOMMENDATIONS:\n"
		output += strings.Repeat("-", 30) + "\n"
		for i, rec := range report.Recommendations {
			output += fmt.Sprintf("%d. %s\n", i+1, rec)
		}
	}

	// Requirement Compliance Summary
	output += "\nREQUIREMENT COMPLIANCE SUMMARY:\n"
	output += strings.Repeat("-", 50) + "\n"
	if report.RequirementCompliance != nil {
		compliance := report.RequirementCompliance
		output += fmt.Sprintf("6.1 - Latency (sub-10ms): %s\n", pte.formatComplianceStatus(compliance.Requirement61_Latency))
		output += fmt.Sprintf("6.2 - Scalability (100+ nodes): %s\n", pte.formatComplianceStatus(compliance.Requirement62_Scalability))
		output += fmt.Sprintf("6.3 - Throughput (1000+ subs): %s\n", pte.formatComplianceStatus(compliance.Requirement63_Throughput))
		output += fmt.Sprintf("6.4 - Performance (10,000+ IPS): %s\n", pte.formatComplianceStatus(compliance.Requirement64_Performance))
		output += fmt.Sprintf("6.5 - Stability (memory): %s\n", pte.formatComplianceStatus(compliance.Requirement65_Stability))
		output += fmt.Sprintf("Overall Compliance: %.1f%%\n", compliance.OverallCompliance)
	}

	output += "\n" + strings.Repeat("=", 100) + "\n"

	return output
}

// formatValidationResult formats a validation result for text output
func (pte *PerformanceTestExecutor) formatValidationResult(testName string, result *dashboard.ValidationResult) string {
	if result == nil {
		return fmt.Sprintf("%s: NO DATA\n", testName)
	}

	status := "FAIL"
	if result.RequirementMet {
		status = "PASS"
	}

	return fmt.Sprintf("%s: %s\n  Actual: %.2f, Required: %.2f, Gap: %.2f\n  Details: %s\n\n",
		testName, status, result.ActualValue, result.RequiredValue, result.PerformanceGap, result.Details)
}

// formatComplianceStatus formats compliance status
func (pte *PerformanceTestExecutor) formatComplianceStatus(compliant bool) string {
	if compliant {
		return "✓ COMPLIANT"
	}
	return "✗ NON-COMPLIANT"
}

// generateJSONReport generates a JSON-based test report
func (pte *PerformanceTestExecutor) generateJSONReport(report *dashboard.ComprehensiveTestReport) error {
	// Implementation for JSON report generation
	log.Println("JSON report generation not yet implemented")
	return nil
}

// generateHTMLReport generates an HTML-based test report
func (pte *PerformanceTestExecutor) generateHTMLReport(report *dashboard.ComprehensiveTestReport) error {
	// Implementation for HTML report generation
	log.Println("HTML report generation not yet implemented")
	return nil
}

// validateRequirementsCompliance validates that all requirements are met
func (pte *PerformanceTestExecutor) validateRequirementsCompliance(report *dashboard.ComprehensiveTestReport) error {
	if report.RequirementCompliance == nil {
		return fmt.Errorf("requirement compliance data not available")
	}

	compliance := report.RequirementCompliance
	failedRequirements := []string{}

	// Check each requirement
	if !compliance.Requirement61_Latency {
		failedRequirements = append(failedRequirements, "6.1 - Sub-10ms processing latency")
	}
	if !compliance.Requirement62_Scalability {
		failedRequirements = append(failedRequirements, "6.2 - 100+ concurrent E2 nodes")
	}
	if !compliance.Requirement63_Throughput {
		failedRequirements = append(failedRequirements, "6.3 - 1000+ concurrent subscriptions")
	}
	if !compliance.Requirement64_Performance {
		failedRequirements = append(failedRequirements, "6.4 - 10,000+ indications per second")
	}
	if !compliance.Requirement65_Stability {
		failedRequirements = append(failedRequirements, "6.5 - Stable memory consumption")
	}

	if len(failedRequirements) > 0 {
		return fmt.Errorf("failed requirements: %v (overall compliance: %.1f%%)", 
			failedRequirements, compliance.OverallCompliance)
	}

	log.Printf("All requirements met! Overall compliance: %.1f%%", compliance.OverallCompliance)
	return nil
}