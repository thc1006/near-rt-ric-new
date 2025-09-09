/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/oran/near-rt-ric-new/pkg/dashboard"
	"github.com/sirupsen/logrus"
)

// Version information
var (
	version   = "1.0.0"
	buildTime = "2024-09-08"
	gitCommit = "dev"
)

// Command line flags
var (
	configFile        = flag.String("config", "config/performance-optimizer.json", "Configuration file path")
	logLevel          = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	enableProfiling   = flag.Bool("enable-profiling", false, "Enable performance profiling")
	runBenchmark      = flag.Bool("benchmark", false, "Run comprehensive performance benchmark")
	optimizeOnly      = flag.Bool("optimize-only", false, "Run optimization only without starting services")
	showVersion       = flag.Bool("version", false, "Show version information")
	
	// Performance targets from command line
	targetLatencyMs   = flag.Int("target-latency-ms", 10, "Target processing latency in milliseconds")
	targetThroughput  = flag.Int("target-throughput", 10000, "Target throughput in indications per second")
	targetE2Nodes     = flag.Int("target-e2-nodes", 100, "Target concurrent E2 nodes")
	targetDashUsers   = flag.Int("target-dashboard-users", 100, "Target concurrent dashboard users")
	
	// SMO and Nephio integration
	smoEndpoint       = flag.String("smo-endpoint", "", "SMO endpoint URL")
	nephioEndpoint    = flag.String("nephio-endpoint", "", "Nephio R5 Porch endpoint URL")
	oCloudEndpoint    = flag.String("ocloud-endpoint", "", "O-Cloud endpoint URL")
)

// PerformanceOptimizerConfig represents the main configuration
type PerformanceOptimizerConfig struct {
	// Core performance configuration
	AdvancedPerformanceConfig *dashboard.AdvancedPerformanceConfig `json:"advancedPerformanceConfig"`
	
	// API configuration
	ProductionAPIConfig       *dashboard.ProductionAPIConfig       `json:"productionAPIConfig"`
	
	// Monitoring configuration
	PerformanceMonitorConfig  *dashboard.PerformanceMonitorConfig  `json:"performanceMonitorConfig"`
	
	// Integration configuration
	IntegrationConfig         *dashboard.IntegrationConfig         `json:"integrationConfig"`
	
	// Connection pool configuration
	ConnectionClusterConfig   *dashboard.ConnectionClusterConfig   `json:"connectionClusterConfig"`
	
	// General settings
	EnableSMOIntegration      bool                                 `json:"enableSMOIntegration"`
	EnableNephioIntegration   bool                                 `json:"enableNephioIntegration"`
	EnableRealTimeMonitoring  bool                                 `json:"enableRealTimeMonitoring"`
	EnableAutoBenchmarking    bool                                 `json:"enableAutoBenchmarking"`
	
	// Logging and observability
	LogLevel                  string                               `json:"logLevel"`
	EnableMetrics             bool                                 `json:"enableMetrics"`
	MetricsPort               int                                  `json:"metricsPort"`
}

// PerformanceOptimizer is the main application structure
type PerformanceOptimizer struct {
	config                    *PerformanceOptimizerConfig
	
	// Core components
	advancedOptimizer         *dashboard.AdvancedSMOPerformanceOptimizer
	productionAPI             *dashboard.ProductionDashboardAPI
	performanceMonitor        *dashboard.ComprehensivePerformanceMonitor
	integrationLayer          *dashboard.SMONephioIntegrationLayer
	connectionCluster         *dashboard.ConnectionPoolCluster
	
	// Context and lifecycle
	ctx                       context.Context
	cancel                    context.CancelFunc
	
	// Performance metrics
	startTime                 time.Time
	benchmarkResults          []*dashboard.BenchmarkResult
}

func main() {
	flag.Parse()
	
	// Show version and exit if requested
	if *showVersion {
		fmt.Printf("O-RAN Near-RT RIC Performance Optimizer\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}
	
	// Setup logging
	setupLogging(*logLevel)
	
	logrus.WithFields(logrus.Fields{
		"version":    version,
		"buildTime":  buildTime,
		"gitCommit":  gitCommit,
		"goVersion":  runtime.Version(),
	}).Info("Starting O-RAN Near-RT RIC Performance Optimizer")
	
	// Load configuration
	config, err := loadConfiguration(*configFile)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}
	
	// Override configuration with command line flags
	overrideConfigFromFlags(config)
	
	// Create performance optimizer
	optimizer := NewPerformanceOptimizer(config)
	
	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	optimizer.ctx = ctx
	optimizer.cancel = cancel
	
	// Setup signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-signalChan
		logrus.Info("Received shutdown signal")
		cancel()
	}()
	
	// Run based on command line options
	if *runBenchmark {
		// Run benchmark mode
		if err := optimizer.RunBenchmarkMode(ctx); err != nil {
			logrus.WithError(err).Fatal("Benchmark mode failed")
		}
	} else if *optimizeOnly {
		// Run optimization only
		if err := optimizer.RunOptimizationMode(ctx); err != nil {
			logrus.WithError(err).Fatal("Optimization mode failed")
		}
	} else {
		// Run full service mode
		if err := optimizer.RunServiceMode(ctx); err != nil {
			logrus.WithError(err).Fatal("Service mode failed")
		}
	}
	
	logrus.Info("O-RAN Near-RT RIC Performance Optimizer shutting down")
}

// NewPerformanceOptimizer creates a new performance optimizer instance
func NewPerformanceOptimizer(config *PerformanceOptimizerConfig) *PerformanceOptimizer {
	optimizer := &PerformanceOptimizer{
		config:           config,
		startTime:        time.Now(),
		benchmarkResults: make([]*dashboard.BenchmarkResult, 0),
	}
	
	// Initialize core components
	optimizer.advancedOptimizer = dashboard.NewAdvancedSMOPerformanceOptimizer(config.AdvancedPerformanceConfig)
	optimizer.productionAPI = dashboard.NewProductionDashboardAPI(config.ProductionAPIConfig)
	optimizer.performanceMonitor = dashboard.NewComprehensivePerformanceMonitor(config.PerformanceMonitorConfig)
	optimizer.connectionCluster = dashboard.NewConnectionPoolCluster(config.ConnectionClusterConfig)
	
	// Initialize integration layer if enabled
	if config.EnableSMOIntegration || config.EnableNephioIntegration {
		optimizer.integrationLayer = dashboard.NewSMONephioIntegrationLayer(config.IntegrationConfig)
	}
	
	return optimizer
}

// RunServiceMode runs the optimizer in full service mode
func (po *PerformanceOptimizer) RunServiceMode(ctx context.Context) error {
	logrus.Info("Starting Performance Optimizer in Service Mode")
	
	// Start core components
	if err := po.startCoreComponents(ctx); err != nil {
		return fmt.Errorf("failed to start core components: %w", err)
	}
	
	// Start integration layer if configured
	if po.integrationLayer != nil {
		if err := po.integrationLayer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start integration layer: %w", err)
		}
	}
	
	// Start performance monitoring
	if po.config.EnableRealTimeMonitoring {
		go po.realTimeMonitoringLoop(ctx)
	}
	
	// Start auto benchmarking if enabled
	if po.config.EnableAutoBenchmarking {
		go po.autoBenchmarkingLoop(ctx)
	}
	
	// Start performance reporting
	go po.performanceReportingLoop(ctx)
	
	logrus.WithFields(logrus.Fields{
		"targetLatencyMs":    po.config.AdvancedPerformanceConfig.MaxProcessingLatencyMs,
		"targetThroughput":   po.config.AdvancedPerformanceConfig.TargetThroughputIPS,
		"maxE2Nodes":         po.config.AdvancedPerformanceConfig.MaxConcurrentE2Nodes,
		"maxDashUsers":       po.config.AdvancedPerformanceConfig.DashboardConcurrentUsers,
		"smoIntegration":     po.config.EnableSMOIntegration,
		"nephioIntegration":  po.config.EnableNephioIntegration,
	}).Info("Performance Optimizer service mode started successfully")
	
	// Wait for shutdown signal
	<-ctx.Done()
	
	// Graceful shutdown
	return po.gracefulShutdown()
}

// RunBenchmarkMode runs comprehensive benchmarks
func (po *PerformanceOptimizer) RunBenchmarkMode(ctx context.Context) error {
	logrus.Info("Starting Performance Optimizer in Benchmark Mode")
	
	// Start minimal components for benchmarking
	if err := po.startCoreComponents(ctx); err != nil {
		return fmt.Errorf("failed to start core components: %w", err)
	}
	
	// Run comprehensive benchmark
	result, err := po.performanceMonitor.RunComprehensiveBenchmark()
	if err != nil {
		return fmt.Errorf("comprehensive benchmark failed: %w", err)
	}
	
	// Store result
	po.benchmarkResults = append(po.benchmarkResults, result)
	
	// Generate detailed report
	report := po.generateBenchmarkReport(result)
	
	// Save report to file
	reportFile := fmt.Sprintf("benchmark-report-%d.json", time.Now().Unix())
	if err := po.saveBenchmarkReport(report, reportFile); err != nil {
		logrus.WithError(err).Error("Failed to save benchmark report")
	}
	
	// Print summary to console
	po.printBenchmarkSummary(result)
	
	return nil
}

// RunOptimizationMode runs optimization only
func (po *PerformanceOptimizer) RunOptimizationMode(ctx context.Context) error {
	logrus.Info("Starting Performance Optimizer in Optimization Mode")
	
	// Start advanced optimizer
	if err := po.advancedOptimizer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start advanced optimizer: %w", err)
	}
	
	// Perform optimizations
	optimizations := []struct {
		name string
		fn   func() error
	}{
		{"E2 Nodes", func() error {
			return po.advancedOptimizer.OptimizeForE2Nodes(*targetE2Nodes)
		}},
		{"Throughput", func() error {
			return po.advancedOptimizer.OptimizeForThroughput(*targetThroughput)
		}},
		{"Dashboard API", func() error {
			return po.advancedOptimizer.OptimizeDashboardAPI(*targetDashUsers)
		}},
	}
	
	for _, opt := range optimizations {
		logrus.WithField("optimization", opt.name).Info("Running optimization")
		if err := opt.fn(); err != nil {
			logrus.WithError(err).WithField("optimization", opt.name).Error("Optimization failed")
		} else {
			logrus.WithField("optimization", opt.name).Info("Optimization completed successfully")
		}
	}
	
	// Get final performance metrics
	metrics := po.advancedOptimizer.GetAdvancedPerformanceMetrics()
	po.printOptimizationResults(metrics)
	
	return nil
}

// Private helper methods

func (po *PerformanceOptimizer) startCoreComponents(ctx context.Context) error {
	components := []struct {
		name      string
		component interface{ Start(context.Context) error }
	}{
		{"Advanced Optimizer", po.advancedOptimizer},
		{"Production API", po.productionAPI},
		{"Performance Monitor", po.performanceMonitor},
		{"Connection Cluster", po.connectionCluster},
	}
	
	for _, comp := range components {
		if comp.component != nil {
			logrus.WithField("component", comp.name).Info("Starting component")
			if err := comp.component.Start(ctx); err != nil {
				return fmt.Errorf("failed to start %s: %w", comp.name, err)
			}
		}
	}
	
	return nil
}

func (po *PerformanceOptimizer) realTimeMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := po.performanceMonitor.GetRealTimePerformanceStats()
			po.logPerformanceStats(stats)
		}
	}
}

func (po *PerformanceOptimizer) autoBenchmarkingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 4) // Run benchmark every 4 hours
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logrus.Info("Running scheduled comprehensive benchmark")
			result, err := po.performanceMonitor.RunComprehensiveBenchmark()
			if err != nil {
				logrus.WithError(err).Error("Scheduled benchmark failed")
			} else {
				po.benchmarkResults = append(po.benchmarkResults, result)
				logrus.WithFields(logrus.Fields{
					"performanceScore": result.PerformanceScore,
					"grade":            result.Grade,
				}).Info("Scheduled benchmark completed")
			}
		}
	}
}

func (po *PerformanceOptimizer) performanceReportingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5) // Generate report every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			report := po.generatePerformanceReport()
			logrus.WithFields(logrus.Fields{
				"uptime":               time.Since(po.startTime),
				"avgLatencyMs":         report.AverageLatencyMs,
				"currentThroughput":    report.CurrentThroughput,
				"connectedE2Nodes":     report.ConnectedE2Nodes,
				"activeDashboardUsers": report.ActiveDashboardUsers,
			}).Info("Performance report")
		}
	}
}

func (po *PerformanceOptimizer) gracefulShutdown() error {
	logrus.Info("Initiating graceful shutdown")
	
	// Stop components in reverse order
	components := []struct {
		name      string
		component interface{ Stop() error }
	}{
		{"Integration Layer", po.integrationLayer},
		{"Connection Cluster", po.connectionCluster},
		{"Performance Monitor", po.performanceMonitor},
		{"Production API", po.productionAPI},
		{"Advanced Optimizer", po.advancedOptimizer},
	}
	
	for _, comp := range components {
		if comp.component != nil {
			logrus.WithField("component", comp.name).Info("Stopping component")
			if err := comp.component.Stop(); err != nil {
				logrus.WithError(err).WithField("component", comp.name).Error("Failed to stop component")
			}
		}
	}
	
	// Generate final performance summary
	po.generateFinalSummary()
	
	logrus.Info("Graceful shutdown completed")
	return nil
}

// Configuration and utility functions

func setupLogging(level string) {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.WithError(err).Warn("Invalid log level, using info")
		logLevel = logrus.InfoLevel
	}
	
	logrus.SetLevel(logLevel)
	logrus.SetOutput(os.Stdout)
}

func loadConfiguration(configFile string) (*PerformanceOptimizerConfig, error) {
	// Try to load from file
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		logrus.WithField("configFile", configFile).Warn("Configuration file not found, using defaults")
		return getDefaultConfiguration(), nil
	}
	
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()
	
	var config PerformanceOptimizerConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}
	
	// Validate and fill in defaults
	validateAndFillDefaults(&config)
	
	return &config, nil
}

func overrideConfigFromFlags(config *PerformanceOptimizerConfig) {
	// Override performance targets
	if *targetLatencyMs != 10 {
		config.AdvancedPerformanceConfig.MaxProcessingLatencyMs = *targetLatencyMs
		config.PerformanceMonitorConfig.LatencyTargetMs = float64(*targetLatencyMs)
	}
	
	if *targetThroughput != 10000 {
		config.AdvancedPerformanceConfig.TargetThroughputIPS = *targetThroughput
		config.PerformanceMonitorConfig.ThroughputTargetIPS = int64(*targetThroughput)
	}
	
	if *targetE2Nodes != 100 {
		config.AdvancedPerformanceConfig.MaxConcurrentE2Nodes = *targetE2Nodes
		config.PerformanceMonitorConfig.E2NodeTargetCount = *targetE2Nodes
	}
	
	if *targetDashUsers != 100 {
		config.AdvancedPerformanceConfig.DashboardConcurrentUsers = *targetDashUsers
		config.PerformanceMonitorConfig.DashboardUserTarget = *targetDashUsers
	}
	
	// Override integration endpoints
	if *smoEndpoint != "" {
		config.IntegrationConfig.SMOEndpoint = *smoEndpoint
		config.EnableSMOIntegration = true
	}
	
	if *nephioEndpoint != "" {
		config.IntegrationConfig.PorchAPIEndpoint = *nephioEndpoint
		config.EnableNephioIntegration = true
	}
	
	if *oCloudEndpoint != "" {
		config.IntegrationConfig.OCloudEndpoint = *oCloudEndpoint
		config.EnableNephioIntegration = true
	}
	
	// Override log level
	if *logLevel != "info" {
		config.LogLevel = *logLevel
	}
}

func getDefaultConfiguration() *PerformanceOptimizerConfig {
	return &PerformanceOptimizerConfig{
		AdvancedPerformanceConfig: &dashboard.AdvancedPerformanceConfig{
			MaxProcessingLatencyMs:      10,
			TargetThroughputIPS:         10000,
			MaxConcurrentE2Nodes:        100,
			DashboardConcurrentUsers:    100,
			
			EnableZeroCopy:              true,
			EnableSIMDAcceleration:      true,
			EnableCPUAffinity:           true,
			EnableHugePages:             true,
			EnableNUMAAwareness:         true,
			
			WorkerThreadCount:           runtime.NumCPU() * 2,
			E2ConnectionPoolSize:        1000,
			HTTPConnectionPoolSize:      500,
			WebSocketPoolSize:           200,
		},
		
		ProductionAPIConfig: &dashboard.ProductionAPIConfig{
			HTTPPort:                8080,
			HTTPSPort:               8443,
			MaxConcurrentE2Nodes:    100,
			MaxConcurrentUsers:      100,
			MaxRequestsPerSecond:    10000,
			TargetAPILatencyMs:      50,
			
			EnableResponseCache:     true,
			EnableCompression:       true,
			EnableLoadBalancing:     true,
			EnableAutoScaling:       true,
		},
		
		PerformanceMonitorConfig: &dashboard.PerformanceMonitorConfig{
			LatencyTargetMs:         10.0,
			ThroughputTargetIPS:     10000,
			E2NodeTargetCount:       100,
			DashboardUserTarget:     100,
			
			RealTimeInterval:        time.Second,
			BenchmarkInterval:       time.Hour * 4,
		},
		
		IntegrationConfig: &dashboard.IntegrationConfig{
			CacheEnabled:            true,
			CircuitBreakerEnabled:   true,
			LoadBalancingEnabled:    true,
		},
		
		ConnectionClusterConfig: &dashboard.ConnectionClusterConfig{
			E2PoolSize:              1000,
			HTTPPoolSize:            500,
			WebSocketPoolSize:       200,
			EnableCPUAffinity:       true,
			EnableHugePages:         true,
			EnableZeroCopy:          true,
		},
		
		EnableRealTimeMonitoring:  true,
		EnableAutoBenchmarking:    true,
		LogLevel:                  "info",
		EnableMetrics:             true,
		MetricsPort:               9090,
	}
}

// Report generation functions

func (po *PerformanceOptimizer) generateBenchmarkReport(result *dashboard.BenchmarkResult) *BenchmarkReport {
	return &BenchmarkReport{
		Timestamp:        result.Timestamp,
		Duration:         result.Duration,
		PerformanceScore: result.PerformanceScore,
		Grade:            result.Grade,
		TargetsMet:       result.TargetsMet,
		Recommendations:  result.Recommendations,
		
		LatencyResults:    result.LatencyResults,
		ThroughputResults: result.ThroughputResults,
		ScalabilityResults: result.ScalabilityResults,
		
		SystemInfo: SystemInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			GOMAXPROCS:   runtime.GOMAXPROCS(0),
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
		},
	}
}

func (po *PerformanceOptimizer) printBenchmarkSummary(result *dashboard.BenchmarkResult) {
	fmt.Printf("\n=== O-RAN Near-RT RIC Performance Benchmark Results ===\n")
	fmt.Printf("Benchmark ID: %s\n", result.ID)
	fmt.Printf("Timestamp: %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("Performance Score: %.2f/100\n", result.PerformanceScore)
	fmt.Printf("Grade: %s\n", result.Grade)
	fmt.Printf("\n")
	
	fmt.Printf("Target Compliance:\n")
	for target, met := range result.TargetsMet {
		status := "❌ FAILED"
		if met {
			status = "✅ PASSED"
		}
		fmt.Printf("  %s: %s\n", target, status)
	}
	fmt.Printf("\n")
	
	if result.LatencyResults != nil {
		fmt.Printf("Latency Results:\n")
		fmt.Printf("  E2 Processing P99: %.2fms (target: <%.0fms)\n", 
			result.LatencyResults.E2ProcessingLatencyMs.P99, result.LatencyResults.TargetLatencyMs)
		fmt.Printf("  API Response P99: %.2fms\n", result.LatencyResults.APILatencyMs.P99)
		fmt.Printf("  Compliance: %.1f%%\n", result.LatencyResults.CompliancePercentage)
		fmt.Printf("\n")
	}
	
	if result.ThroughputResults != nil {
		fmt.Printf("Throughput Results:\n")
		fmt.Printf("  Peak Throughput: %d IPS\n", result.ThroughputResults.PeakThroughputIPS)
		fmt.Printf("  Sustained Throughput: %d IPS (target: %d IPS)\n", 
			result.ThroughputResults.SustainedThroughputIPS, result.ThroughputResults.TargetThroughputIPS)
		fmt.Printf("  Target Achieved: %t\n", result.ThroughputResults.TargetAchieved)
		fmt.Printf("\n")
	}
	
	if len(result.Recommendations) > 0 {
		fmt.Printf("Recommendations:\n")
		for _, rec := range result.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
		fmt.Printf("\n")
	}
	
	fmt.Printf("=== End of Benchmark Results ===\n\n")
}

// Additional types for reporting
type BenchmarkReport struct {
	Timestamp          time.Time                               `json:"timestamp"`
	Duration           time.Duration                           `json:"duration"`
	PerformanceScore   float64                                 `json:"performanceScore"`
	Grade              string                                  `json:"grade"`
	TargetsMet         map[string]bool                         `json:"targetsMet"`
	Recommendations    []string                                `json:"recommendations"`
	LatencyResults     *dashboard.LatencyBenchmarkResult       `json:"latencyResults"`
	ThroughputResults  *dashboard.ThroughputBenchmarkResult    `json:"throughputResults"`
	ScalabilityResults *dashboard.ScalabilityBenchmarkResult   `json:"scalabilityResults"`
	SystemInfo         SystemInfo                              `json:"systemInfo"`
}

type SystemInfo struct {
	GoVersion    string `json:"goVersion"`
	NumCPU       int    `json:"numCPU"`
	GOMAXPROCS   int    `json:"gomaxprocs"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

type PerformanceReport struct {
	Timestamp            time.Time `json:"timestamp"`
	Uptime               time.Duration `json:"uptime"`
	AverageLatencyMs     float64   `json:"averageLatencyMs"`
	CurrentThroughput    int64     `json:"currentThroughput"`
	ConnectedE2Nodes     int64     `json:"connectedE2Nodes"`
	ActiveDashboardUsers int64     `json:"activeDashboardUsers"`
}

func (po *PerformanceOptimizer) generatePerformanceReport() PerformanceReport {
	stats := po.performanceMonitor.GetRealTimePerformanceStats()
	
	return PerformanceReport{
		Timestamp:            time.Now(),
		Uptime:               time.Since(po.startTime),
		AverageLatencyMs:     stats.LatencyStats.E2ProcessingLatencyMs.Mean,
		CurrentThroughput:    stats.ThroughputStats.E2IndicationsPerSecond,
		ConnectedE2Nodes:     int64(stats.E2InterfaceStats.ConnectedE2Nodes),
		ActiveDashboardUsers: int64(stats.DashboardAPIStats.ConcurrentUsers),
	}
}

func (po *PerformanceOptimizer) saveBenchmarkReport(report *BenchmarkReport, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func validateAndFillDefaults(config *PerformanceOptimizerConfig) {
	// Validate and fill defaults - implementation would go here
}

func (po *PerformanceOptimizer) logPerformanceStats(stats dashboard.ComprehensivePerformanceStats) {
	logrus.WithFields(logrus.Fields{
		"latencyP99Ms":      stats.LatencyStats.E2ProcessingLatencyMs.P99,
		"throughputIPS":     stats.ThroughputStats.E2IndicationsPerSecond,
		"connectedE2Nodes":  stats.E2InterfaceStats.ConnectedE2Nodes,
		"cpuUtilization":    stats.ResourceStats.CPUUtilization.OverallUtilization,
		"memoryUtilizationMB": stats.ResourceStats.MemoryUtilization.UsedMemoryMB,
	}).Debug("Real-time performance statistics")
}

func (po *PerformanceOptimizer) printOptimizationResults(metrics dashboard.AdvancedPerformanceStats) {
	fmt.Printf("\n=== Optimization Results ===\n")
	fmt.Printf("Average Processing Latency: %.2fms\n", float64(metrics.AverageProcessingTimeNs)/1e6)
	fmt.Printf("Current Throughput: %d IPS\n", metrics.CurrentThroughputIPS)
	fmt.Printf("Peak Throughput: %d IPS\n", metrics.PeakThroughputIPS)
	fmt.Printf("Connected E2 Nodes: %d\n", metrics.ConnectedE2Nodes)
	fmt.Printf("Dashboard Users: %d\n", metrics.ConcurrentDashboardUsers)
	fmt.Printf("CPU Utilization: %.1f%%\n", metrics.CPUUtilization)
	fmt.Printf("Memory Utilization: %d MB\n", metrics.MemoryUtilizationMB)
	fmt.Printf("============================\n\n")
}

func (po *PerformanceOptimizer) generateFinalSummary() {
	uptime := time.Since(po.startTime)
	
	logrus.WithFields(logrus.Fields{
		"uptime":           uptime,
		"benchmarksRun":    len(po.benchmarkResults),
		"version":          version,
	}).Info("Performance Optimizer final summary")
	
	if len(po.benchmarkResults) > 0 {
		lastResult := po.benchmarkResults[len(po.benchmarkResults)-1]
		logrus.WithFields(logrus.Fields{
			"lastBenchmarkScore": lastResult.PerformanceScore,
			"lastBenchmarkGrade": lastResult.Grade,
		}).Info("Last benchmark results")
	}
}