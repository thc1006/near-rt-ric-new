/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// ComprehensivePerformanceMonitor provides real-time performance monitoring
// and benchmarking for O-RAN L Release with Nephio R5 integration
type ComprehensivePerformanceMonitor struct {
	// Core monitoring components
	realTimeProfiler        *RealTimeProfiler
	latencyAnalyzer         *LatencyAnalyzer
	throughputMonitor       *ThroughputMonitor
	resourceMonitor         *ResourceMonitor
	
	// SMO and Nephio R5 monitoring
	smoPerformanceMonitor   *SMOPerformanceMonitor
	nephioPerformanceMonitor *NephioPerformanceMonitor
	
	// E2 interface monitoring
	e2InterfaceMonitor      *E2InterfaceMonitor
	indicationMonitor       *IndicationMonitor
	subscriptionMonitor     *SubscriptionMonitor
	
	// Dashboard API monitoring
	apiPerformanceMonitor   *APIPerformanceMonitor
	connectionMonitor       *ConnectionMonitor
	
	// Performance analysis
	performanceAnalyzer     *PerformanceAnalyzer
	bottleneckDetector      *BottleneckDetector
	performancePredictor    *PerformancePredictor
	
	// Benchmarking system
	benchmarkRunner         *BenchmarkRunner
	loadTester              *LoadTester
	stressTester            *StressTester
	
	// Configuration and state
	config                  *PerformanceMonitorConfig
	stats                   ComprehensivePerformanceStats
	benchmarkHistory        []*BenchmarkResult
	
	// Control
	running                 int32
	mu                      sync.RWMutex
}

// PerformanceMonitorConfig defines monitoring configuration
type PerformanceMonitorConfig struct {
	// Monitoring intervals
	RealTimeInterval        time.Duration `json:"realTimeInterval"`
	LatencyAnalysisInterval time.Duration `json:"latencyAnalysisInterval"`
	ThroughputInterval      time.Duration `json:"throughputInterval"`
	ResourceInterval        time.Duration `json:"resourceInterval"`
	
	// Performance targets
	LatencyTargetMs         float64       `json:"latencyTargetMs"`
	ThroughputTargetIPS     int64         `json:"throughputTargetIPS"`
	E2NodeTargetCount       int           `json:"e2NodeTargetCount"`
	DashboardUserTarget     int           `json:"dashboardUserTarget"`
	
	// Analysis parameters
	LatencyPercentiles      []float64     `json:"latencyPercentiles"`
	ThroughputWindowSize    time.Duration `json:"throughputWindowSize"`
	ResourceThresholds      map[string]float64 `json:"resourceThresholds"`
	
	// Alerting thresholds
	LatencyAlertThreshold   float64       `json:"latencyAlertThreshold"`
	ThroughputAlertThreshold float64      `json:"throughputAlertThreshold"`
	ErrorRateThreshold      float64       `json:"errorRateThreshold"`
	
	// Benchmarking settings
	BenchmarkInterval       time.Duration `json:"benchmarkInterval"`
	LoadTestDuration        time.Duration `json:"loadTestDuration"`
	StressTestDuration      time.Duration `json:"stressTestDuration"`
	
	// Data retention
	MetricsRetentionPeriod  time.Duration `json:"metricsRetentionPeriod"`
	BenchmarkRetentionCount int           `json:"benchmarkRetentionCount"`
}

// ComprehensivePerformanceStats contains all performance metrics
type ComprehensivePerformanceStats struct {
	// Timestamp
	Timestamp               time.Time     `json:"timestamp"`
	
	// Latency metrics (nanoseconds converted to milliseconds for readability)
	LatencyStats            LatencyStats  `json:"latencyStats"`
	
	// Throughput metrics
	ThroughputStats         ThroughputStats `json:"throughputStats"`
	
	// Resource utilization
	ResourceStats           ResourceStats `json:"resourceStats"`
	
	// E2 interface performance
	E2InterfaceStats        E2InterfaceStats `json:"e2InterfaceStats"`
	
	// Dashboard API performance
	DashboardAPIStats       DashboardAPIStats `json:"dashboardAPIStats"`
	
	// SMO integration performance
	SMOIntegrationStats     SMOIntegrationStats `json:"smoIntegrationStats"`
	
	// Nephio R5 performance
	NephioPerformanceStats  NephioPerformanceStats `json:"nephioPerformanceStats"`
	
	// Error and availability metrics
	ErrorStats              ErrorStats `json:"errorStats"`
	AvailabilityStats       AvailabilityStats `json:"availabilityStats"`
	
	// Performance targets compliance
	TargetCompliance        TargetCompliance `json:"targetCompliance"`
	
	// Benchmark results
	LatestBenchmarkResult   *BenchmarkResult `json:"latestBenchmarkResult,omitempty"`
}

// LatencyStats provides detailed latency analysis
type LatencyStats struct {
	// Processing latency
	E2ProcessingLatencyMs   LatencyMetrics `json:"e2ProcessingLatencyMs"`
	A1ProcessingLatencyMs   LatencyMetrics `json:"a1ProcessingLatencyMs"`
	O1ProcessingLatencyMs   LatencyMetrics `json:"o1ProcessingLatencyMs"`
	
	// API latency
	DashboardAPILatencyMs   LatencyMetrics `json:"dashboardAPILatencyMs"`
	WebSocketLatencyMs      LatencyMetrics `json:"webSocketLatencyMs"`
	
	// End-to-end latency
	E2EndToEndLatencyMs     LatencyMetrics `json:"e2EndToEndLatencyMs"`
	IndicationLatencyMs     LatencyMetrics `json:"indicationLatencyMs"`
	
	// SMO integration latency
	SMORequestLatencyMs     LatencyMetrics `json:"smoRequestLatencyMs"`
	PolicyUpdateLatencyMs   LatencyMetrics `json:"policyUpdateLatencyMs"`
	
	// Network latency
	NetworkLatencyMs        LatencyMetrics `json:"networkLatencyMs"`
}

// LatencyMetrics type is now defined in types.go to avoid redeclaration

// ThroughputStats provides throughput analysis
type ThroughputStats struct {
	// Message throughput
	E2IndicationsPerSecond  int64         `json:"e2IndicationsPerSecond"`
	A1PolicyUpdatesPerSecond int64        `json:"a1PolicyUpdatesPerSecond"`
	O1ConfigUpdatesPerSecond int64        `json:"o1ConfigUpdatesPerSecond"`
	
	// API throughput
	DashboardRequestsPerSecond int64      `json:"dashboardRequestsPerSecond"`
	WebSocketMessagesPerSecond int64      `json:"webSocketMessagesPerSecond"`
	
	// SMO integration throughput
	SMORequestsPerSecond    int64         `json:"smoRequestsPerSecond"`
	RAppDeploymentsPerSecond int64        `json:"rAppDeploymentsPerSecond"`
	
	// Nephio R5 throughput
	PackageDeploymentsPerSecond int64     `json:"packageDeploymentsPerSecond"`
	ResourceProvisioningPerSecond int64   `json:"resourceProvisioningPerSecond"`
	
	// Peak measurements
	PeakThroughputAchieved  int64         `json:"peakThroughputAchieved"`
	PeakThroughputTimestamp time.Time     `json:"peakThroughputTimestamp"`
	
	// Efficiency metrics
	ThroughputEfficiency    float64       `json:"throughputEfficiency"`
	ProcessingUtilization   float64       `json:"processingUtilization"`
}

// ResourceStats provides resource utilization analysis
type ResourceStats struct {
	// CPU metrics
	CPUUtilization          CPUMetrics    `json:"cpuUtilization"`
	
	// Memory metrics
	MemoryUtilization       MemoryMetrics `json:"memoryUtilization"`
	
	// Network metrics
	NetworkUtilization      NetworkMetrics `json:"networkUtilization"`
	
	// Disk I/O metrics
	DiskUtilization         DiskMetrics   `json:"diskUtilization"`
	
	// Go runtime metrics
	RuntimeMetrics          RuntimeMetrics `json:"runtimeMetrics"`
	
	// Container metrics (if running in containers)
	ContainerMetrics        ContainerMetrics `json:"containerMetrics,omitempty"`
}

// CPUMetrics provides CPU utilization details
type CPUMetrics struct {
	OverallUtilization      float64       `json:"overallUtilization"`
	PerCoreUtilization      []float64     `json:"perCoreUtilization"`
	LoadAverage             LoadAverage   `json:"loadAverage"`
	ContextSwitchesPerSec   int64         `json:"contextSwitchesPerSec"`
	CPUAffinityEffectiveness float64      `json:"cpuAffinityEffectiveness"`
}

// MemoryMetrics provides memory utilization details
type MemoryMetrics struct {
	TotalMemoryMB           int64         `json:"totalMemoryMB"`
	UsedMemoryMB            int64         `json:"usedMemoryMB"`
	AvailableMemoryMB       int64         `json:"availableMemoryMB"`
	MemoryUtilizationPercent float64      `json:"memoryUtilizationPercent"`
	
	// Go heap metrics
	HeapSizeMB              int64         `json:"heapSizeMB"`
	HeapUsedMB              int64         `json:"heapUsedMB"`
	
	// GC metrics
	GCPausesMs              []float64     `json:"gcPausesMs"`
	GCFrequency             float64       `json:"gcFrequency"`
	
	// Memory pool metrics
	PoolHitRatio            float64       `json:"poolHitRatio"`
	ZeroCopyEffectiveness   float64       `json:"zeroCopyEffectiveness"`
}

// E2InterfaceStats provides E2 interface performance metrics
type E2InterfaceStats struct {
	// Connection metrics
	ConnectedE2Nodes        int64         `json:"connectedE2Nodes"`
	ActiveConnections       int64         `json:"activeConnections"`
	ConnectionFailures      int64         `json:"connectionFailures"`
	ConnectionLatencyMs     float64       `json:"connectionLatencyMs"`
	
	// Subscription metrics
	ActiveSubscriptions     int64         `json:"activeSubscriptions"`
	SubscriptionSetupRate   int64         `json:"subscriptionSetupRate"`
	SubscriptionFailureRate float64       `json:"subscriptionFailureRate"`
	
	// Indication metrics
	IndicationsReceived     int64         `json:"indicationsReceived"`
	IndicationsProcessed    int64         `json:"indicationsProcessed"`
	IndicationProcessingRate int64        `json:"indicationProcessingRate"`
	IndicationLatencyMs     float64       `json:"indicationLatencyMs"`
	
	// Control message metrics
	ControlMessagesSent     int64         `json:"controlMessagesSent"`
	ControlMessageLatencyMs float64       `json:"controlMessageLatencyMs"`
	ControlSuccessRate      float64       `json:"controlSuccessRate"`
}

// BenchmarkRunner executes comprehensive performance benchmarks
type BenchmarkRunner struct {
	// Benchmark types
	latencyBenchmark        *LatencyBenchmark
	throughputBenchmark     *ThroughputBenchmark
	scalabilityBenchmark    *ScalabilityBenchmark
	stressBenchmark         *StressBenchmark
	enduranceBenchmark      *EnduranceBenchmark
	
	// Test configuration
	config                  *BenchmarkConfig
	
	// Result storage
	results                 []*BenchmarkResult
	
	// State management
	running                 int32
	currentBenchmark        string
	mu                      sync.RWMutex
}

// BenchmarkConfig defines benchmark parameters
type BenchmarkConfig struct {
	// Latency benchmark
	LatencyTestDuration     time.Duration `json:"latencyTestDuration"`
	LatencyTargetMs         float64       `json:"latencyTargetMs"`
	LatencyTestMessageCount int           `json:"latencyTestMessageCount"`
	
	// Throughput benchmark
	ThroughputTestDuration  time.Duration `json:"throughputTestDuration"`
	ThroughputTargetIPS     int           `json:"throughputTargetIPS"`
	ThroughputRampUpTime    time.Duration `json:"throughputRampUpTime"`
	
	// Scalability benchmark
	MaxE2Nodes              int           `json:"maxE2Nodes"`
	MaxConcurrentUsers      int           `json:"maxConcurrentUsers"`
	ScalabilityStepSize     int           `json:"scalabilityStepSize"`
	ScalabilityStepDuration time.Duration `json:"scalabilityStepDuration"`
	
	// Stress benchmark
	StressTestDuration      time.Duration `json:"stressTestDuration"`
	StressTestMultiplier    float64       `json:"stressTestMultiplier"`
	ResourceLimitTests      bool          `json:"resourceLimitTests"`
	
	// Endurance benchmark
	EnduranceTestDuration   time.Duration `json:"enduranceTestDuration"`
	MemoryLeakDetection     bool          `json:"memoryLeakDetection"`
	PerformanceDegradation  bool          `json:"performanceDegradation"`
}

// BenchmarkResult contains comprehensive benchmark results
type BenchmarkResult struct {
	// Metadata
	ID                      string        `json:"id"`
	Timestamp               time.Time     `json:"timestamp"`
	Duration                time.Duration `json:"duration"`
	BenchmarkType           string        `json:"benchmarkType"`
	
	// Test configuration
	Configuration           interface{}   `json:"configuration"`
	
	// Performance results
	LatencyResults          *LatencyBenchmarkResult     `json:"latencyResults,omitempty"`
	ThroughputResults       *ThroughputBenchmarkResult  `json:"throughputResults,omitempty"`
	ScalabilityResults      *ScalabilityBenchmarkResult `json:"scalabilityResults,omitempty"`
	StressResults           *StressBenchmarkResult      `json:"stressResults,omitempty"`
	EnduranceResults        *EnduranceBenchmarkResult   `json:"enduranceResults,omitempty"`
	
	// Compliance assessment
	TargetsMet              map[string]bool `json:"targetsMet"`
	PerformanceScore        float64        `json:"performanceScore"`
	Grade                   string         `json:"grade"`
	
	// Recommendations
	Recommendations         []string       `json:"recommendations"`
	
	// System state during test
	SystemMetrics           SystemMetrics  `json:"systemMetrics"`
}

// LatencyBenchmarkResult contains latency benchmark results
type LatencyBenchmarkResult struct {
	// E2 interface latency
	E2SetupLatencyMs        LatencyMetrics `json:"e2SetupLatencyMs"`
	SubscriptionLatencyMs   LatencyMetrics `json:"subscriptionLatencyMs"`
	IndicationLatencyMs     LatencyMetrics `json:"indicationLatencyMs"`
	ControlLatencyMs        LatencyMetrics `json:"controlLatencyMs"`
	
	// Dashboard API latency
	APILatencyMs            LatencyMetrics `json:"apiLatencyMs"`
	WebSocketLatencyMs      LatencyMetrics `json:"webSocketLatencyMs"`
	
	// SMO integration latency
	SMOLatencyMs            LatencyMetrics `json:"smoLatencyMs"`
	
	// Target compliance
	TargetLatencyMs         float64        `json:"targetLatencyMs"`
	CompliancePercentage    float64        `json:"compliancePercentage"`
	OutlierCount            int64          `json:"outlierCount"`
}

// ThroughputBenchmarkResult contains throughput benchmark results
type ThroughputBenchmarkResult struct {
	// Peak throughput achieved
	PeakThroughputIPS       int64          `json:"peakThroughputIPS"`
	SustainedThroughputIPS  int64          `json:"sustainedThroughputIPS"`
	AverageThroughputIPS    int64          `json:"averageThroughputIPS"`
	
	// Breakdown by component
	E2ThroughputIPS         int64          `json:"e2ThroughputIPS"`
	APIthroughputIPS        int64          `json:"apiThroughputIPS"`
	SMOThroughputIPS        int64          `json:"smoThroughputIPS"`
	
	// Efficiency metrics
	CPUEfficiency           float64        `json:"cpuEfficiency"`
	MemoryEfficiency        float64        `json:"memoryEfficiency"`
	NetworkEfficiency       float64        `json:"networkEfficiency"`
	
	// Target compliance
	TargetThroughputIPS     int64          `json:"targetThroughputIPS"`
	TargetAchieved          bool           `json:"targetAchieved"`
	PerformanceRatio        float64        `json:"performanceRatio"`
}

// NewComprehensivePerformanceMonitor creates a new performance monitor
func NewComprehensivePerformanceMonitor(config *PerformanceMonitorConfig) *ComprehensivePerformanceMonitor {
	if config == nil {
		config = getDefaultPerformanceMonitorConfig()
	}

	monitor := &ComprehensivePerformanceMonitor{
		config:           config,
		benchmarkHistory: make([]*BenchmarkResult, 0),
	}

	// Initialize monitoring components
	monitor.realTimeProfiler = NewRealTimeProfiler(config.RealTimeInterval)
	monitor.latencyAnalyzer = NewLatencyAnalyzer(config.LatencyPercentiles)
	monitor.throughputMonitor = NewThroughputMonitor(config.ThroughputWindowSize)
	monitor.resourceMonitor = NewResourceMonitor(config.ResourceInterval)

	// Initialize interface monitors
	monitor.e2InterfaceMonitor = NewE2InterfaceMonitor()
	monitor.indicationMonitor = NewIndicationMonitor()
	monitor.subscriptionMonitor = NewSubscriptionMonitor()

	// Initialize SMO and Nephio monitors
	monitor.smoPerformanceMonitor = NewSMOPerformanceMonitor()
	monitor.nephioPerformanceMonitor = NewNephioPerformanceMonitor()

	// Initialize API monitor
	monitor.apiPerformanceMonitor = NewAPIPerformanceMonitor()
	monitor.connectionMonitor = NewConnectionMonitor()

	// Initialize analysis components
	monitor.performanceAnalyzer = NewPerformanceAnalyzer()
	monitor.bottleneckDetector = NewBottleneckDetector(config.LatencyAlertThreshold)
	monitor.performancePredictor = NewPerformancePredictor()

	// Initialize benchmarking system
	benchmarkConfig := &BenchmarkConfig{
		LatencyTestDuration:     time.Minute * 5,
		LatencyTargetMs:         config.LatencyTargetMs,
		ThroughputTestDuration:  time.Minute * 10,
		ThroughputTargetIPS:     int(config.ThroughputTargetIPS),
		MaxE2Nodes:              config.E2NodeTargetCount,
		MaxConcurrentUsers:      config.DashboardUserTarget,
		EnduranceTestDuration:   time.Hour,
	}
	monitor.benchmarkRunner = NewBenchmarkRunner(benchmarkConfig)
	monitor.loadTester = NewLoadTester()
	monitor.stressTester = NewStressTester()

	return monitor
}

// Start starts the comprehensive performance monitor
func (cpm *ComprehensivePerformanceMonitor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&cpm.running, 0, 1) {
		return fmt.Errorf("performance monitor already running")
	}

	logrus.WithFields(logrus.Fields{
		"latencyTargetMs":      cpm.config.LatencyTargetMs,
		"throughputTargetIPS":  cpm.config.ThroughputTargetIPS,
		"e2NodeTarget":         cpm.config.E2NodeTargetCount,
		"dashboardUserTarget":  cpm.config.DashboardUserTarget,
	}).Info("Starting Comprehensive Performance Monitor")

	// Start monitoring components
	components := []interface{ Start(context.Context) error }{
		cpm.realTimeProfiler,
		cpm.latencyAnalyzer,
		cpm.throughputMonitor,
		cpm.resourceMonitor,
		cpm.e2InterfaceMonitor,
		cpm.indicationMonitor,
		cpm.subscriptionMonitor,
		cpm.smoPerformanceMonitor,
		cpm.nephioPerformanceMonitor,
		cpm.apiPerformanceMonitor,
		cpm.connectionMonitor,
		cpm.performanceAnalyzer,
		cpm.bottleneckDetector,
		cpm.performancePredictor,
	}

	for _, component := range components {
		if component != nil {
			if err := component.Start(ctx); err != nil {
				return fmt.Errorf("failed to start monitoring component: %w", err)
			}
		}
	}

	// Start monitoring loops
	go cpm.realTimeMonitoringLoop(ctx)
	go cpm.performanceAnalysisLoop(ctx)
	go cpm.benchmarkSchedulingLoop(ctx)
	go cpm.alertingLoop(ctx)

	logrus.Info("Comprehensive Performance Monitor started successfully")
	return nil
}

// RunComprehensiveBenchmark executes a full performance benchmark suite
func (cpm *ComprehensivePerformanceMonitor) RunComprehensiveBenchmark() (*BenchmarkResult, error) {
	if !atomic.CompareAndSwapInt32(&cpm.benchmarkRunner.running, 0, 1) {
		return nil, fmt.Errorf("benchmark already running")
	}
	defer atomic.StoreInt32(&cpm.benchmarkRunner.running, 0)

	logrus.Info("Starting comprehensive performance benchmark")

	startTime := time.Now()
	result := &BenchmarkResult{
		ID:            fmt.Sprintf("benchmark_%d", startTime.Unix()),
		Timestamp:     startTime,
		BenchmarkType: "comprehensive",
		TargetsMet:    make(map[string]bool),
	}

	// Run latency benchmark
	logrus.Info("Running latency benchmark")
	latencyResults, err := cpm.runLatencyBenchmark()
	if err != nil {
		logrus.WithError(err).Error("Latency benchmark failed")
	} else {
		result.LatencyResults = latencyResults
		result.TargetsMet["latency"] = latencyResults.CompliancePercentage >= 95.0
	}

	// Run throughput benchmark
	logrus.Info("Running throughput benchmark")
	throughputResults, err := cpm.runThroughputBenchmark()
	if err != nil {
		logrus.WithError(err).Error("Throughput benchmark failed")
	} else {
		result.ThroughputResults = throughputResults
		result.TargetsMet["throughput"] = throughputResults.TargetAchieved
	}

	// Run scalability benchmark
	logrus.Info("Running scalability benchmark")
	scalabilityResults, err := cpm.runScalabilityBenchmark()
	if err != nil {
		logrus.WithError(err).Error("Scalability benchmark failed")
	} else {
		result.ScalabilityResults = scalabilityResults
		result.TargetsMet["scalability"] = scalabilityResults.MaxE2NodesAchieved >= cpm.config.E2NodeTargetCount
	}

	// Run stress benchmark
	logrus.Info("Running stress benchmark")
	stressResults, err := cpm.runStressBenchmark()
	if err != nil {
		logrus.WithError(err).Error("Stress benchmark failed")
	} else {
		result.StressResults = stressResults
		result.TargetsMet["stress"] = stressResults.SystemStabilityScore >= 0.8
	}

	result.Duration = time.Since(startTime)
	result.PerformanceScore = cpm.calculatePerformanceScore(result)
	result.Grade = cpm.calculateGrade(result.PerformanceScore)
	result.Recommendations = cpm.generateRecommendations(result)

	// Store result
	cpm.mu.Lock()
	cpm.benchmarkHistory = append(cpm.benchmarkHistory, result)
	if len(cpm.benchmarkHistory) > cpm.config.BenchmarkRetentionCount {
		cpm.benchmarkHistory = cpm.benchmarkHistory[1:]
	}
	cpm.mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"duration":         result.Duration,
		"performanceScore": result.PerformanceScore,
		"grade":            result.Grade,
		"latencyCompliant": result.TargetsMet["latency"],
		"throughputMet":    result.TargetsMet["throughput"],
		"scalabilityMet":   result.TargetsMet["scalability"],
	}).Info("Comprehensive benchmark completed")

	return result, nil
}

// GetRealTimePerformanceStats returns current performance statistics
func (cpm *ComprehensivePerformanceMonitor) GetRealTimePerformanceStats() ComprehensivePerformanceStats {
	cpm.mu.RLock()
	defer cpm.mu.RUnlock()

	stats := ComprehensivePerformanceStats{
		Timestamp: time.Now(),
	}

	// Collect latency statistics
	stats.LatencyStats = cpm.collectLatencyStats()

	// Collect throughput statistics
	stats.ThroughputStats = cpm.collectThroughputStats()

	// Collect resource statistics
	stats.ResourceStats = cpm.collectResourceStats()

	// Collect E2 interface statistics
	stats.E2InterfaceStats = cpm.collectE2InterfaceStats()

	// Collect Dashboard API statistics
	stats.DashboardAPIStats = cpm.collectDashboardAPIStats()

	// Collect SMO integration statistics
	stats.SMOIntegrationStats = cpm.collectSMOIntegrationStats()

	// Collect Nephio performance statistics
	stats.NephioPerformanceStats = cpm.collectNephioPerformanceStats()

	// Calculate target compliance
	stats.TargetCompliance = cpm.calculateTargetCompliance(stats)

	// Add latest benchmark result if available
	if len(cpm.benchmarkHistory) > 0 {
		stats.LatestBenchmarkResult = cpm.benchmarkHistory[len(cpm.benchmarkHistory)-1]
	}

	return stats
}

// Private methods for benchmark execution

func (cpm *ComprehensivePerformanceMonitor) runLatencyBenchmark() (*LatencyBenchmarkResult, error) {
	result := &LatencyBenchmarkResult{
		TargetLatencyMs: cpm.config.LatencyTargetMs,
	}

	// Test E2 setup latency
	e2SetupLatencies := cpm.measureE2SetupLatency(1000)
	result.E2SetupLatencyMs = cpm.calculateLatencyMetrics(e2SetupLatencies)

	// Test subscription latency
	subscriptionLatencies := cpm.measureSubscriptionLatency(1000)
	result.SubscriptionLatencyMs = cpm.calculateLatencyMetrics(subscriptionLatencies)

	// Test indication processing latency
	indicationLatencies := cpm.measureIndicationLatency(10000)
	result.IndicationLatencyMs = cpm.calculateLatencyMetrics(indicationLatencies)

	// Test RIC control latency
	controlLatencies := cpm.measureControlLatency(500)
	result.ControlLatencyMs = cpm.calculateLatencyMetrics(controlLatencies)

	// Test API latency
	apiLatencies := cpm.measureAPILatency(2000)
	result.APILatencyMs = cpm.calculateLatencyMetrics(apiLatencies)

	// Calculate compliance
	result.CompliancePercentage = cpm.calculateLatencyCompliance(result)

	return result, nil
}

func (cpm *ComprehensivePerformanceMonitor) runThroughputBenchmark() (*ThroughputBenchmarkResult, error) {
	result := &ThroughputBenchmarkResult{
		TargetThroughputIPS: cpm.config.ThroughputTargetIPS,
	}

	// Run sustained throughput test
	throughputSamples := cpm.measureSustainedThroughput(cpm.config.ThroughputTargetIPS)
	
	if len(throughputSamples) > 0 {
		result.PeakThroughputIPS = maxInt64(throughputSamples)
		result.AverageThroughputIPS = averageInt64(throughputSamples)
		result.SustainedThroughputIPS = result.AverageThroughputIPS
		result.TargetAchieved = result.SustainedThroughputIPS >= cpm.config.ThroughputTargetIPS
		result.PerformanceRatio = float64(result.SustainedThroughputIPS) / float64(cpm.config.ThroughputTargetIPS)
	}

	// Measure component-specific throughput
	result.E2ThroughputIPS = cpm.measureE2Throughput()
	result.APIthroughputIPS = cpm.measureAPIThroughput()
	result.SMOThroughputIPS = cpm.measureSMOThroughput()

	// Calculate efficiency metrics
	result.CPUEfficiency = cpm.calculateCPUEfficiency()
	result.MemoryEfficiency = cpm.calculateMemoryEfficiency()
	result.NetworkEfficiency = cpm.calculateNetworkEfficiency()

	return result, nil
}

func (cpm *ComprehensivePerformanceMonitor) runScalabilityBenchmark() (*ScalabilityBenchmarkResult, error) {
	result := &ScalabilityBenchmarkResult{}

	// Test E2 node scalability
	maxE2Nodes := cpm.testE2NodeScalability()
	result.MaxE2NodesAchieved = maxE2Nodes

	// Test concurrent user scalability
	maxUsers := cpm.testConcurrentUserScalability()
	result.MaxConcurrentUsers = maxUsers

	// Test throughput scalability
	scalabilityFactor := cpm.testThroughputScalability()
	result.ThroughputScalabilityFactor = scalabilityFactor

	return result, nil
}

func (cpm *ComprehensivePerformanceMonitor) runStressBenchmark() (*StressBenchmarkResult, error) {
	result := &StressBenchmarkResult{}

	// Test under high load
	stabilityScore := cpm.testSystemStability()
	result.SystemStabilityScore = stabilityScore

	// Test resource exhaustion
	recoveryScore := cpm.testResourceExhaustionRecovery()
	result.RecoveryScore = recoveryScore

	// Test error handling
	errorHandlingScore := cpm.testErrorHandling()
	result.ErrorHandlingScore = errorHandlingScore

	return result, nil
}

// Monitoring loop implementations

func (cpm *ComprehensivePerformanceMonitor) realTimeMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(cpm.config.RealTimeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpm.collectRealTimeMetrics()
		}
	}
}

func (cpm *ComprehensivePerformanceMonitor) performanceAnalysisLoop(ctx context.Context) {
	ticker := time.NewTicker(cpm.config.LatencyAnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpm.performAnalysis()
		}
	}
}

func (cpm *ComprehensivePerformanceMonitor) benchmarkSchedulingLoop(ctx context.Context) {
	ticker := time.NewTicker(cpm.config.BenchmarkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if cpm.shouldRunBenchmark() {
				go func() {
					_, err := cpm.RunComprehensiveBenchmark()
					if err != nil {
						logrus.WithError(err).Error("Scheduled benchmark failed")
					}
				}()
			}
		}
	}
}

func (cpm *ComprehensivePerformanceMonitor) alertingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cpm.checkPerformanceAlerts()
		}
	}
}

// Utility functions

func (cpm *ComprehensivePerformanceMonitor) calculateLatencyMetrics(samples []float64) LatencyMetrics {
	if len(samples) == 0 {
		return LatencyMetrics{}
	}

	sort.Float64s(samples)
	n := len(samples)

	metrics := LatencyMetrics{
		SampleCount: int64(n),
		Min:         samples[0],
		Max:         samples[n-1],
	}

	// Calculate mean
	sum := 0.0
	for _, sample := range samples {
		sum += sample
	}
	metrics.Mean = sum / float64(n)

	// Calculate percentiles
	metrics.Median = samples[n/2]
	metrics.P95 = samples[int(float64(n)*0.95)]
	metrics.P99 = samples[int(float64(n)*0.99)]
	metrics.P999 = samples[int(float64(n)*0.999)]

	// Calculate standard deviation
	varianceSum := 0.0
	for _, sample := range samples {
		diff := sample - metrics.Mean
		varianceSum += diff * diff
	}
	metrics.StandardDeviation = varianceSum / float64(n-1)

	return metrics
}

func getDefaultPerformanceMonitorConfig() *PerformanceMonitorConfig {
	return &PerformanceMonitorConfig{
		RealTimeInterval:        time.Second,
		LatencyAnalysisInterval: time.Second * 5,
		ThroughputInterval:      time.Second * 10,
		ResourceInterval:        time.Second * 5,
		
		LatencyTargetMs:         10.0,
		ThroughputTargetIPS:     10000,
		E2NodeTargetCount:       100,
		DashboardUserTarget:     100,
		
		LatencyPercentiles:      []float64{0.5, 0.90, 0.95, 0.99, 0.999},
		ThroughputWindowSize:    time.Minute,
		
		LatencyAlertThreshold:   15.0,
		ThroughputAlertThreshold: 8000.0,
		ErrorRateThreshold:      0.01,
		
		BenchmarkInterval:       time.Hour * 4,
		LoadTestDuration:        time.Minute * 10,
		StressTestDuration:      time.Minute * 5,
		
		MetricsRetentionPeriod:  time.Hour * 24,
		BenchmarkRetentionCount: 100,
	}
}

func maxInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func averageInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	sum := int64(0)
	for _, v := range values {
		sum += v
	}
	return sum / int64(len(values))
}

// Stop gracefully stops the performance monitor
func (cpm *ComprehensivePerformanceMonitor) Stop() error {
	if !atomic.CompareAndSwapInt32(&cpm.running, 1, 0) {
		return fmt.Errorf("performance monitor not running")
	}

	logrus.Info("Stopping Comprehensive Performance Monitor")

	// Stop all monitoring components
	components := []interface{ Stop() error }{
		cpm.realTimeProfiler,
		cpm.latencyAnalyzer,
		cpm.throughputMonitor,
		cpm.resourceMonitor,
		cpm.e2InterfaceMonitor,
		cpm.benchmarkRunner,
	}

	for _, component := range components {
		if component != nil {
			if err := component.Stop(); err != nil {
				logrus.WithError(err).Error("Failed to stop monitoring component")
			}
		}
	}

	logrus.Info("Comprehensive Performance Monitor stopped successfully")
	return nil
}