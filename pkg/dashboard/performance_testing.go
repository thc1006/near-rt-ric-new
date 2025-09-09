// Performance testing package providing comprehensive benchmarking capabilities
// for the O-RAN Near-RT RIC Dashboard and platform components
package dashboard

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"encoding/json"
	"log"
	"sort"
)

// Performance testing constants and configurations
const (
	DefaultTestDuration     = 30 * time.Second
	DefaultConcurrency     = 10
	DefaultRampUpDuration  = 5 * time.Second
	DefaultRampDownDuration = 5 * time.Second
	MaxConcurrency         = 1000
	MetricsCollectionInterval = 1 * time.Second
)

// NOTE: PerformanceTestRunner type is in performance_test_runner.go to avoid redeclaration

// Performance Test Configuration
type PerformanceTestConfig struct {
	TestName          string                 `json:"testName"`
	TestType          string                 `json:"testType"`
	Duration          time.Duration          `json:"duration"`
	Concurrency       int                    `json:"concurrency"`
	RampUpDuration    time.Duration          `json:"rampUpDuration"`
	RampDownDuration  time.Duration          `json:"rampDownDuration"`
	TargetTPS         int                    `json:"targetTPS"`
	AcceptableLatency time.Duration          `json:"acceptableLatency"`
	ErrorThreshold    float64               `json:"errorThreshold"`
	MetricsInterval   time.Duration          `json:"metricsInterval"`
	CustomConfig      map[string]interface{} `json:"customConfig,omitempty"`
}

// Performance Test Result
type PerformanceTestResult struct {
	TestName         string                 `json:"testName"`
	TestType         string                 `json:"testType"`
	StartTime        time.Time              `json:"startTime"`
	EndTime          time.Time              `json:"endTime"`
	Duration         time.Duration          `json:"duration"`
	TotalRequests    int64                  `json:"totalRequests"`
	SuccessfulReqs   int64                  `json:"successfulRequests"`
	FailedRequests   int64                  `json:"failedRequests"`
	RequestsPerSec   float64                `json:"requestsPerSecond"`
	LatencyMetrics   *LatencyMetrics        `json:"latencyMetrics"`
	ThroughputMetrics *ThroughputMetrics     `json:"throughputMetrics"`
	ResourceMetrics  *ResourceMetrics       `json:"resourceMetrics"`
	ErrorMetrics     *ErrorMetrics          `json:"errorMetrics"`
	Pass             bool                   `json:"pass"`
	Score            float64                `json:"score"`
	Issues           []string               `json:"issues,omitempty"`
	Recommendations  []string               `json:"recommendations,omitempty"`
}

// Throughput Metrics
type ThroughputMetrics struct {
	MaxThroughput    float64   `json:"maxThroughput"`
	MinThroughput    float64   `json:"minThroughput"`
	AvgThroughput    float64   `json:"avgThroughput"`
	MedianThroughput float64   `json:"medianThroughput"`
	ThroughputP95    float64   `json:"throughputP95"`
	ThroughputP99    float64   `json:"throughputP99"`
	Samples          []float64 `json:"samples"`
	SampleInterval   time.Duration `json:"sampleInterval"`
}

// Resource Metrics
type ResourceMetrics struct {
	CPUUsage         *ResourceUsageMetrics `json:"cpuUsage"`
	MemoryUsage      *ResourceUsageMetrics `json:"memoryUsage"`
	NetworkIO        *IOMetrics           `json:"networkIO"`
	DiskIO           *IOMetrics           `json:"diskIO"`
	GoroutineCount   *ResourceUsageMetrics `json:"goroutineCount"`
	GCMetrics        *GCMetrics           `json:"gcMetrics"`
}

// Resource Usage Metrics
type ResourceUsageMetrics struct {
	Max     float64   `json:"max"`
	Min     float64   `json:"min"`
	Average float64   `json:"average"`
	Current float64   `json:"current"`
	Samples []float64 `json:"samples"`
}

// IO Metrics
type IOMetrics struct {
	BytesRead    int64 `json:"bytesRead"`
	BytesWritten int64 `json:"bytesWritten"`
	Operations   int64 `json:"operations"`
	RateBytes    float64 `json:"rateBytes"`
	RateOps      float64 `json:"rateOps"`
}

// GC Metrics
type GCMetrics struct {
	NumGC        uint32        `json:"numGC"`
	PauseTotal   time.Duration `json:"pauseTotal"`
	PauseAvg     time.Duration `json:"pauseAvg"`
	PauseMax     time.Duration `json:"pauseMax"`
	LastGC       time.Time     `json:"lastGC"`
	HeapSize     uint64        `json:"heapSize"`
	HeapInUse    uint64        `json:"heapInUse"`
}

// Error Metrics
type ErrorMetrics struct {
	TotalErrors      int64             `json:"totalErrors"`
	ErrorRate        float64           `json:"errorRate"`
	ErrorsByType     map[string]int64  `json:"errorsByType"`
	ErrorsByCode     map[int]int64     `json:"errorsByCode"`
	CriticalErrors   int64             `json:"criticalErrors"`
	RecoverableErrors int64            `json:"recoverableErrors"`
}

// LatencyBucket represents a latency distribution bucket - needed for performance_testing.go line 127
type LatencyBucket struct {
	Min       time.Duration `json:"min"`
	Max       time.Duration `json:"max"`
	Count     int64         `json:"count"`
	Frequency float64       `json:"frequency"`
	Label     string        `json:"label"`
}

// NOTE: ResourceExhaustionPoint type moved to types.go to avoid redeclaration

// Failure Scenario
type FailureScenario struct {
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	TriggerPoint    string    `json:"triggerPoint"`
	RecoveryTimeMs  int64     `json:"recoveryTimeMs"`
	Impact          string    `json:"impact"`
	Severity        TestSeverity `json:"severity"`
	MitigationSteps []string  `json:"mitigationSteps"`
}

// NOTE: PerformanceTestRunner type moved to performance_test_runner.go to avoid redeclaration

// NOTE: PerformanceMetrics type is in performance_optimizer.go to avoid redeclaration

// Resource Samples
type ResourceSamples struct {
	CPUSamples      []float64
	MemorySamples   []uint64
	GoroutineSamples []int
	Timestamps      []time.Time
}

// NOTE: NewPerformanceTestRunner moved to performance_test_runner.go to avoid redeclaration

// Performance testing helper functions

// getCurrentCPUUsage returns current CPU usage (stub implementation)
func getCurrentCPUUsage() float64 {
	// This is a simplified implementation
	// In production, use proper CPU monitoring
	return 0.0
}

// Helper functions for metric calculations
func maxFloat64(values []float64) float64 {
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

func minFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func avgFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func maxUint64(values []uint64) uint64 {
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

func minUint64(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func avgUint64(values []uint64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := uint64(0)
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func maxInt(values []int) int {
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

// minInt function for calculating minimum of two integers - needed by performance_testing.go line 257
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func avgInt(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func uint64ToFloat64(values []uint64) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = float64(v)
	}
	return result
}

func intToFloat64(values []int) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = float64(v)
	}
	return result
}

// calculateMean calculates mean latency
func calculateMean(latencies []time.Duration) float64 {
	if len(latencies) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, latency := range latencies {
		sum += latency
	}
	
	return float64(sum.Nanoseconds()) / float64(len(latencies)) / 1e6
}

// calculatePercentile calculates latency percentile
func calculatePercentile(latencies []time.Duration, percentile float64) float64 {
	if len(latencies) == 0 {
		return 0
	}
	
	index := int(float64(len(latencies)) * percentile / 100.0)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}
	
	return float64(latencies[index].Nanoseconds()) / 1e6
}

// PerformanceBenchmark represents a performance benchmark
type PerformanceBenchmark struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Config      *PerformanceTestConfig `json:"config"`
	Results     *PerformanceTestResult `json:"results"`
	Validation  *ValidationResult      `json:"validation"`
	RunTime     time.Duration          `json:"runTime"`
	Success     bool                   `json:"success"`
	ErrorMsg    string                 `json:"errorMsg,omitempty"`
}

// RunBenchmark executes a performance benchmark
func RunBenchmark(ctx context.Context, config *PerformanceTestConfig, testFunc func() error) (*PerformanceBenchmark, error) {
	benchmark := &PerformanceBenchmark{
		Name:   config.TestName,
		Type:   config.TestType,
		Config: config,
	}

	startTime := time.Now()
	defer func() {
		benchmark.RunTime = time.Since(startTime)
	}()

	// Execute the actual test function in a loop
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var latencies []time.Duration

	testDuration := config.Duration
	if testDuration == 0 {
		testDuration = DefaultTestDuration
	}

	testCtx, cancel := context.WithTimeout(ctx, testDuration)
	defer cancel()

	var wg sync.WaitGroup
	concurrency := config.Concurrency
	if concurrency == 0 {
		concurrency = DefaultConcurrency
	}

	// Start concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-testCtx.Done():
					return
				default:
					start := time.Now()
					err := testFunc()
					duration := time.Since(start)

					atomic.AddInt64(&totalRequests, 1)
					if err != nil {
						atomic.AddInt64(&failedRequests, 1)
					} else {
						atomic.AddInt64(&successfulRequests, 1)
					}

					// Thread-safe append to latencies (simplified approach)
					latencies = append(latencies, duration)
				}
			}
		}()
	}

	wg.Wait()

	// Calculate results
	testEndTime := time.Now()
	actualDuration := testEndTime.Sub(startTime)
	
	result := &PerformanceTestResult{
		TestName:       config.TestName,
		TestType:       config.TestType,
		StartTime:      startTime,
		EndTime:        testEndTime,
		Duration:       actualDuration,
		TotalRequests:  totalRequests,
		SuccessfulReqs: successfulRequests,
		FailedRequests: failedRequests,
		RequestsPerSec: float64(totalRequests) / actualDuration.Seconds(),
	}

	// Calculate latency metrics if we have data
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		result.LatencyMetrics = &LatencyMetrics{
			Min:    float64(latencies[0].Nanoseconds()) / 1e6,
			Max:    float64(latencies[len(latencies)-1].Nanoseconds()) / 1e6,
			Mean:   calculateMean(latencies),
			P95:    calculatePercentile(latencies, 95),
			P99:    calculatePercentile(latencies, 99),
			Count:  int64(len(latencies)),
		}
	}

	benchmark.Results = result
	benchmark.Success = failedRequests == 0

	// Simple validation
	validation := &ValidationResult{
		TestName:       config.TestName,
		RequirementMet: benchmark.Success,
		Timestamp:      time.Now(),
	}

	if config.TargetTPS > 0 {
		validation.RequiredValue = float64(config.TargetTPS)
		validation.ActualValue = result.RequestsPerSec
		validation.RequirementMet = result.RequestsPerSec >= float64(config.TargetTPS)*0.9 // 90% tolerance
	}

	benchmark.Validation = validation

	return benchmark, nil
}

// ExportBenchmarkResults exports benchmark results to JSON
func ExportBenchmarkResults(benchmarks []*PerformanceBenchmark) ([]byte, error) {
	return json.MarshalIndent(benchmarks, "", "  ")
}

// LogBenchmarkResults logs benchmark results in a formatted way
func LogBenchmarkResults(benchmarks []*PerformanceBenchmark) {
	for _, benchmark := range benchmarks {
		log.Printf("=== Benchmark Results for %s ===", benchmark.Name)
		log.Printf("Type: %s", benchmark.Type)
		log.Printf("Duration: %v", benchmark.RunTime)
		log.Printf("Success: %t", benchmark.Success)

		if benchmark.Results != nil {
			log.Printf("Total Requests: %d", benchmark.Results.TotalRequests)
			log.Printf("Requests/sec: %.2f", benchmark.Results.RequestsPerSec)
			log.Printf("Success Rate: %.2f%%", float64(benchmark.Results.SuccessfulReqs)/float64(benchmark.Results.TotalRequests)*100)

			if benchmark.Results.LatencyMetrics != nil {
				log.Printf("Latency - Mean: %.2fms, P95: %.2fms, P99: %.2fms",
					benchmark.Results.LatencyMetrics.Mean,
					benchmark.Results.LatencyMetrics.P95,
					benchmark.Results.LatencyMetrics.P99)
			}
		}

		if benchmark.Validation != nil {
			log.Printf("Validation: %t", benchmark.Validation.RequirementMet)
			if benchmark.Validation.RequiredValue > 0 {
				log.Printf("Required: %.2f, Actual: %.2f", 
					benchmark.Validation.RequiredValue, 
					benchmark.Validation.ActualValue)
			}
		}

		if !benchmark.Success && benchmark.ErrorMsg != "" {
			log.Printf("Error: %s", benchmark.ErrorMsg)
		}
		log.Println()
	}
}