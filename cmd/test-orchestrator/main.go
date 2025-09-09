package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/oran/near-rt-ric-new/pkg/dashboard"
	"github.com/sirupsen/logrus"
)

var (
	configFile     = flag.String("config", "test-config.json", "Test configuration file path")
	logLevel       = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	outputDir      = flag.String("output-dir", "./test-results", "Output directory for test results")
	reportFormat   = flag.String("report-format", "json,html", "Comma-separated list of report formats")
	parallel       = flag.Bool("parallel", true, "Run test suites in parallel")
	continueOnErr  = flag.Bool("continue-on-error", true, "Continue execution even if some tests fail")
	testTimeout    = flag.Duration("timeout", 2*time.Hour, "Maximum test execution timeout")
	coverage       = flag.Float64("min-coverage", 80.0, "Minimum required test coverage percentage")
	dryRun         = flag.Bool("dry-run", false, "Perform a dry run without executing tests")
	verbose        = flag.Bool("verbose", false, "Enable verbose logging")
	testSuites     = flag.String("test-suites", "all", "Comma-separated list of test suites to run (e2e,load,nephio,compliance,all)")
)

func main() {
	flag.Parse()

	// Setup logging
	logger := logrus.New()
	level, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.Fatalf("Invalid log level: %v", err)
	}
	logger.SetLevel(level)

	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.Info("Starting O-RAN L Release and Nephio R5 Test Orchestrator")
	logger.Info("Configuration", 
		"configFile", *configFile,
		"outputDir", *outputDir,
		"parallel", *parallel,
		"testSuites", *testSuites,
		"timeout", *testTimeout)

	// Load configuration
	config, err := loadConfiguration(*configFile)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Override configuration with command line flags
	applyCommandLineOverrides(config)

	// Validate configuration
	if err := validateConfiguration(config); err != nil {
		logger.Fatalf("Configuration validation failed: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		logger.Fatalf("Failed to create output directory: %v", err)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), *testTimeout)
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		logger.Warn("Received interrupt signal, initiating graceful shutdown...")
		cancel()
	}()

	// Create test orchestrator
	orchestrator, err := dashboard.NewTestOrchestrator(config, logger)
	if err != nil {
		logger.Fatalf("Failed to create test orchestrator: %v", err)
	}

	if *dryRun {
		logger.Info("Dry run mode - configuration validated successfully")
		printConfigurationSummary(config, logger)
		return
	}

	// Run comprehensive test suite
	logger.Info("Starting comprehensive test execution")
	results, err := orchestrator.RunComprehensiveTestSuite(ctx)
	if err != nil {
		logger.Errorf("Test execution failed: %v", err)
		os.Exit(1)
	}

	// Print summary
	printTestSummary(results, logger)

	// Determine exit code based on results
	exitCode := determineExitCode(results)
	if exitCode != 0 {
		logger.Error("Test execution completed with failures")
	} else {
		logger.Info("Test execution completed successfully")
	}

	os.Exit(exitCode)
}

// loadConfiguration loads the test configuration from file
func loadConfiguration(configPath string) (*dashboard.OrchestratorConfig, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfiguration(), nil
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config dashboard.OrchestratorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// createDefaultConfiguration creates a default configuration
func createDefaultConfiguration() *dashboard.OrchestratorConfig {
	return &dashboard.OrchestratorConfig{
		E2EConfig: &dashboard.E2ETestConfig{
			E2TermEndpoint:       "http://e2term:36421",
			E2MgrEndpoint:        "http://e2mgr:3800",
			SubMgrEndpoint:       "http://submgr:3801",
			A1MediatorURL:        "http://a1mediator:9001",
			O1MediatorURL:        "http://o1mediator:8080",
			O2CloudAPI:           "http://ocloud:8080",
			MaxConcurrentE2Nodes: 100,
			TestDuration:         30 * time.Minute,
			CoverageThreshold:    80.0,
			Namespace:           "oran",
			ReportOutputDir:     "./test-results",
		},
		LoadTestConfig: &dashboard.LoadTestConfig{
			MaxConcurrentE2Nodes:  100,
			TestDuration:         15 * time.Minute,
			RampUpDuration:       5 * time.Minute,
			RampDownDuration:     2 * time.Minute,
			RequestsPerSecond:    1000,
			MaxBurstSize:         2000,
			MaxLatencyP99:        100 * time.Millisecond,
			MaxLatencyP95:        50 * time.Millisecond,
			MinThroughputMbps:    1000.0,
			MaxErrorRate:         1.0,
			MaxCpuUtilization:    80.0,
			MaxMemoryUtilization: 80.0,
			E2TermEndpoint:       "http://e2term:36421",
			E2MgrEndpoint:        "http://e2mgr:3800",
			DashboardEndpoint:    "http://dashboard:3000",
			ReportPath:          "./test-results/load-test-report.json",
		},
		NephioR5Config: &dashboard.NephioR5Config{
			PorchAPIEndpoint:      "http://porch-server:7007",
			PackageRegistryURL:    "https://github.com/nephio-project/catalog",
			GitOpsRepoURL:         "https://github.com/nephio-project/nephio-test",
			ConfigSyncEndpoint:    "http://config-sync:8080",
			KubeConfig:           "", // Use in-cluster config
			TargetNamespace:      "nephio-system",
			WorkloadClusters: []dashboard.WorkloadClusterConfig{
				{
					Name:         "edge-cluster-1",
					Endpoint:     "https://edge-1.example.com",
					Region:       "us-west-1",
					Provider:     "aws",
					Capabilities: []string{"5g-ran", "edge-compute"},
					Labels: map[string]string{
						"cluster-type": "edge",
						"region":       "us-west-1",
					},
					ResourceQuotas: dashboard.ResourceQuotas{
						CPU:         "32",
						Memory:      "128Gi",
						Storage:     "1Ti",
						MaxPods:     1000,
						MaxServices: 100,
					},
				},
			},
			PackageRepository:    "oci://registry.nephio.org/packages",
			PackageCatalogURL:    "https://catalog.nephio.org",
			TestTimeout:          30 * time.Minute,
			DeploymentTimeout:    10 * time.Minute,
		},
		ComplianceConfig: &dashboard.ComplianceConfig{
			E2TermEndpoint:    "http://e2term:36421",
			E2MgrEndpoint:     "http://e2mgr:3800",
			SubMgrEndpoint:    "http://submgr:3801",
			A1MediatorURL:     "http://a1mediator:9001",
			O1MediatorURL:     "http://o1mediator:8080",
			Timeout:           30 * time.Second,
			RetryAttempts:     3,
			TestDataPath:      "./test-data",
			ReportOutputPath:  "./test-results/compliance-report.json",
		},
		ParallelExecution:  true,
		ContinueOnFailure:  true,
		MaxRetries:         3,
		TestTimeout:        2 * time.Hour,
		MinCoveragePercent: 80.0,
		MaxFailureRate:     5.0,
		QualityGates: []dashboard.QualityGate{
			{
				Name:          "Test Pass Rate",
				Type:          "threshold",
				Metric:        "overall_pass_rate",
				Threshold:     95.0,
				Operator:      ">=",
				Severity:      "critical",
				FailureAction: "fail_build",
			},
			{
				Name:          "Test Coverage",
				Type:          "threshold",
				Metric:        "coverage_percent",
				Threshold:     80.0,
				Operator:      ">=",
				Severity:      "high",
				FailureAction: "warning",
			},
			{
				Name:          "Error Rate",
				Type:          "threshold",
				Metric:        "error_rate",
				Threshold:     1.0,
				Operator:      "<=",
				Severity:      "high",
				FailureAction: "warning",
			},
			{
				Name:          "P99 Latency",
				Type:          "threshold",
				Metric:        "p99_latency_ms",
				Threshold:     100.0,
				Operator:      "<=",
				Severity:      "medium",
				FailureAction: "warning",
			},
		},
		OutputDirectory:   "./test-results",
		ReportFormats:     []string{"json", "html"},
		ArtifactRetention: 30 * 24 * time.Hour, // 30 days
		EnvironmentType:   "test",
		TestLabels: map[string]string{
			"release":     "o-ran-l",
			"nephio":      "r5",
			"environment": "test",
		},
	}
}

// applyCommandLineOverrides applies command line flag overrides to the configuration
func applyCommandLineOverrides(config *dashboard.OrchestratorConfig) {
	config.OutputDirectory = *outputDir
	config.ParallelExecution = *parallel
	config.ContinueOnFailure = *continueOnErr
	config.TestTimeout = *testTimeout
	config.MinCoveragePercent = *coverage

	// Parse report formats
	if *reportFormat != "" {
		config.ReportFormats = strings.Split(*reportFormat, ",")
		for i, format := range config.ReportFormats {
			config.ReportFormats[i] = strings.TrimSpace(format)
		}
	}

	// Parse test suites to run
	if *testSuites != "" && *testSuites != "all" {
		suites := strings.Split(*testSuites, ",")
		suitesMap := make(map[string]bool)
		for _, suite := range suites {
			suitesMap[strings.TrimSpace(suite)] = true
		}

		// Disable test suites not specified
		if !suitesMap["e2e"] {
			config.E2EConfig = nil
		}
		if !suitesMap["load"] {
			config.LoadTestConfig = nil
		}
		if !suitesMap["nephio"] {
			config.NephioR5Config = nil
		}
		if !suitesMap["compliance"] {
			config.ComplianceConfig = nil
		}
	}
}

// validateConfiguration validates the test configuration
func validateConfiguration(config *dashboard.OrchestratorConfig) error {
	if config.OutputDirectory == "" {
		return fmt.Errorf("output directory must be specified")
	}

	if config.TestTimeout <= 0 {
		return fmt.Errorf("test timeout must be positive")
	}

	if config.MinCoveragePercent < 0 || config.MinCoveragePercent > 100 {
		return fmt.Errorf("minimum coverage percent must be between 0 and 100")
	}

	// Validate that at least one test suite is enabled
	if config.E2EConfig == nil && config.LoadTestConfig == nil && 
	   config.NephioR5Config == nil && config.ComplianceConfig == nil {
		return fmt.Errorf("at least one test suite must be enabled")
	}

	return nil
}

// printConfigurationSummary prints a summary of the configuration
func printConfigurationSummary(config *dashboard.OrchestratorConfig, logger *logrus.Logger) {
	logger.Info("=== Test Configuration Summary ===")
	logger.Info("General Settings",
		"parallelExecution", config.ParallelExecution,
		"continueOnFailure", config.ContinueOnFailure,
		"testTimeout", config.TestTimeout,
		"minCoverage", config.MinCoveragePercent)

	if config.E2EConfig != nil {
		logger.Info("E2E Test Configuration",
			"maxE2Nodes", config.E2EConfig.MaxConcurrentE2Nodes,
			"duration", config.E2EConfig.TestDuration,
			"coverageThreshold", config.E2EConfig.CoverageThreshold)
	}

	if config.LoadTestConfig != nil {
		logger.Info("Load Test Configuration",
			"maxE2Nodes", config.LoadTestConfig.MaxConcurrentE2Nodes,
			"rps", config.LoadTestConfig.RequestsPerSecond,
			"duration", config.LoadTestConfig.TestDuration,
			"maxLatencyP99", config.LoadTestConfig.MaxLatencyP99)
	}

	if config.NephioR5Config != nil {
		logger.Info("Nephio R5 Test Configuration",
			"porchEndpoint", config.NephioR5Config.PorchAPIEndpoint,
			"workloadClusters", len(config.NephioR5Config.WorkloadClusters),
			"testTimeout", config.NephioR5Config.TestTimeout)
	}

	if config.ComplianceConfig != nil {
		logger.Info("Compliance Test Configuration",
			"timeout", config.ComplianceConfig.Timeout,
			"retryAttempts", config.ComplianceConfig.RetryAttempts)
	}

	logger.Info("Quality Gates", "count", len(config.QualityGates))
	for _, gate := range config.QualityGates {
		logger.Debug("Quality Gate",
			"name", gate.Name,
			"metric", gate.Metric,
			"threshold", gate.Threshold,
			"operator", gate.Operator,
			"severity", gate.Severity)
	}

	logger.Info("Output Configuration",
		"directory", config.OutputDirectory,
		"formats", config.ReportFormats,
		"retention", config.ArtifactRetention)
}

// printTestSummary prints a summary of the test results
func printTestSummary(results *dashboard.CombinedTestResults, logger *logrus.Logger) {
	logger.Info("=== Test Execution Summary ===")

	if results.OverallMetrics != nil {
		metrics := results.OverallMetrics
		logger.Info("Overall Test Results",
			"totalTests", metrics.TotalTests,
			"passedTests", metrics.PassedTests,
			"failedTests", metrics.FailedTests,
			"skippedTests", metrics.SkippedTests,
			"passRate", fmt.Sprintf("%.2f%%", metrics.OverallPassRate),
			"executionTime", metrics.TotalExecutionTime)

		if metrics.AverageLatencyMs > 0 {
			logger.Info("Performance Metrics",
				"avgLatency", fmt.Sprintf("%.2fms", metrics.AverageLatencyMs),
				"p99Latency", fmt.Sprintf("%.2fms", metrics.P99LatencyMs),
				"throughput", fmt.Sprintf("%.2f RPS", metrics.ThroughputRPS),
				"errorRate", fmt.Sprintf("%.2f%%", metrics.ErrorRate))
		}

		if metrics.E2NodesConnected > 0 {
			logger.Info("O-RAN Metrics",
				"e2NodesConnected", metrics.E2NodesConnected,
				"activeSubscriptions", metrics.ActiveSubscriptions,
				"policyEnforcements", metrics.PolicyEnforcements,
				"xAppsDeployed", metrics.XAppsDeployed)
		}
	}

	if results.CoverageAnalysis != nil {
		logger.Info("Coverage Analysis",
			"overallCoverage", fmt.Sprintf("%.2f%%", results.CoverageAnalysis.OverallCoverage),
			"codeCoverage", fmt.Sprintf("%.2f%%", results.CoverageAnalysis.CodeCoverage),
			"testCoverage", fmt.Sprintf("%.2f%%", results.CoverageAnalysis.TestCoverage),
			"coverageGaps", len(results.CoverageAnalysis.CoverageGaps))
	}

	// Print quality gate results
	if len(results.QualityGateResults) > 0 {
		logger.Info("Quality Gate Results")
		passedGates := 0
		for _, gate := range results.QualityGateResults {
			status := "FAIL"
			if gate.Passed {
				status = "PASS"
				passedGates++
			}
			logger.Info("Quality Gate",
				"name", gate.GateName,
				"status", status,
				"actual", fmt.Sprintf("%.2f", gate.ActualValue),
				"threshold", fmt.Sprintf("%.2f", gate.ThresholdValue),
				"deviation", fmt.Sprintf("%.2f", gate.Deviation))
		}
		logger.Info("Quality Gates Summary",
			"passed", passedGates,
			"total", len(results.QualityGateResults),
			"passRate", fmt.Sprintf("%.2f%%", float64(passedGates)/float64(len(results.QualityGateResults))*100))
	}

	// Print recommendations
	if len(results.Recommendations) > 0 {
		logger.Info("Test Recommendations", "count", len(results.Recommendations))
		for _, rec := range results.Recommendations {
			logger.Info("Recommendation",
				"category", rec.Category,
				"priority", rec.Priority,
				"title", rec.Title,
				"impact", rec.Impact,
				"effort", rec.Effort)
		}
	}

	// Print action items
	if len(results.ActionItems) > 0 {
		logger.Info("Action Items", "count", len(results.ActionItems))
		for _, item := range results.ActionItems {
			logger.Info("Action Item",
				"id", item.ID,
				"category", item.Category,
				"priority", item.Priority,
				"title", item.Title,
				"dueDate", item.DueDate.Format("2006-01-02"))
		}
	}
}

// determineExitCode determines the appropriate exit code based on test results
func determineExitCode(results *dashboard.CombinedTestResults) int {
	if results == nil {
		return 1
	}

	// Check if any critical quality gates failed
	for _, gate := range results.QualityGateResults {
		if !gate.Passed && gate.ImpactLevel == "high" {
			return 1
		}
	}

	// Check overall pass rate
	if results.OverallMetrics != nil && results.OverallMetrics.OverallPassRate < 95.0 {
		return 1
	}

	// Check for failed test suites
	if results.OverallMetrics != nil && results.OverallMetrics.FailedTests > 0 {
		// Allow some failures if continue on failure is enabled
		failureRate := float64(results.OverallMetrics.FailedTests) / float64(results.OverallMetrics.TotalTests) * 100
		if failureRate > 5.0 { // More than 5% failure rate
			return 1
		}
	}

	return 0
}