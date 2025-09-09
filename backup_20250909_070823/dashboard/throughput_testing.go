package dashboard

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ThroughputTestManager manages throughput testing scenarios
type ThroughputTestManager struct {
	e2Manager           *E2ManagerClient
	subManager          *SubscriptionManagerClient
	indicationProcessor *IndicationProcessor
	metrics             *ThroughputTestMetrics
	mu                  sync.RWMutex
}

// IndicationProcessor simulates indication processing pipeline
type IndicationProcessor struct {
	inputQueue       chan *Indication
	processingQueue  chan *ProcessedIndication
	outputQueue      chan *ProcessedIndication
	processedCount   int64
	droppedCount     int64
	processingTimeMs []float64
	queueDepths      []int
	backpressureEvents int64
	mu               sync.RWMutex
}

// Indication type is now defined in types.go to avoid redeclaration

// ProcessedIndication represents a processed indication
type ProcessedIndication struct {
	*Indication
	ProcessingEnd   time.Time
	ProcessingTimeMs float64
	QueueWaitTimeMs float64
}

// ThroughputTestMetrics captures detailed throughput metrics
type ThroughputTestMetrics struct {
	TargetThroughput       int                    `json:"targetThroughput"`
	AchievedThroughput     int                    `json:"achievedThroughput"`
	PeakThroughput         int                    `json:"peakThroughput"`
	SustainedThroughput    int                    `json:"sustainedThroughput"`
	ThroughputSamples      []ThroughputSample     `json:"throughputSamples"`
	ProcessingLatencies    []float64              `json:"processingLatencies"`
	QueueMetrics           *QueueMetrics          `json:"queueMetrics"`
	BackpressureEvents     int64                  `json:"backpressureEvents"`
	DroppedIndications     int64                  `json:"droppedIndications"`
	ProcessedIndications   int64                  `json:"processedIndications"`
	ErrorRate              float64                `json:"errorRate"`
	ResourceUtilization    []*ResourceConsumption `json:"resourceUtilization"`
	TestDuration           time.Duration          `json:"testDuration"`
	mu                     sync.RWMutex
}

// ThroughputTestScenario defines a throughput test scenario
type ThroughputTestScenario struct {
	Name                   string        `json:"name"`
	TargetThroughputIPS    int           `json:"targetThroughputIPS"`
	RampUpDuration         time.Duration `json:"rampUpDuration"`
	SustainDuration        time.Duration `json:"sustainDuration"`
	RampDownDuration       time.Duration `json:"rampDownDuration"`
	MaxQueueDepth          int           `json:"maxQueueDepth"`
	ProcessingComplexity   string        `json:"processingComplexity"` // "simple", "medium", "complex"
	BackpressureThreshold  float64       `json:"backpressureThreshold"`
}

// NewThroughputTestManager creates a new throughput test manager
func NewThroughputTestManager(e2Manager *E2ManagerClient, subManager *SubscriptionManagerClient) *ThroughputTestManager {
	processor := &IndicationProcessor{
		inputQueue:      make(chan *Indication, 10000),
		processingQueue: make(chan *ProcessedIndication, 10000),
		outputQueue:     make(chan *ProcessedIndication, 10000),
		processingTimeMs: make([]float64, 0),
		queueDepths:     make([]int, 0),
	}

	return &ThroughputTestManager{
		e2Manager:           e2Manager,
		subManager:          subManager,
		indicationProcessor: processor,
		metrics: &ThroughputTestMetrics{
			ThroughputSamples: make([]ThroughputSample, 0),
			ProcessingLatencies: make([]float64, 0),
			ResourceUtilization: make([]*ResourceConsumption, 0),
		},
	}
}

// RunThroughputTesting executes throughput testing scenarios
func (pts *PerformanceTestSuite) RunThroughputTesting(ctx context.Context) error {
	log.Println("Starting throughput testing scenarios...")

	throughputManager := NewThroughputTestManager(pts.e2Manager, pts.subManager)

	// Define throughput test scenarios - Enhanced for 10,000+ indications per second
	scenarios := []*ThroughputTestScenario{
		{
			Name:                  "Linear Ramp-up Test - 15K IPS",
			TargetThroughputIPS:   pts.config.TargetThroughput,
			RampUpDuration:        6 * time.Minute,
			SustainDuration:       15 * time.Minute, // Extended sustain
			RampDownDuration:      3 * time.Minute,
			MaxQueueDepth:         2000, // Increased queue depth
			ProcessingComplexity:  "simple",
			BackpressureThreshold: 0.8,
		},
		{
			Name:                  "High Burst Test - 25K IPS",
			TargetThroughputIPS:   25000, // Well above 10K requirement
			RampUpDuration:        45 * time.Second,
			SustainDuration:       5 * time.Minute,
			RampDownDuration:      45 * time.Second,
			MaxQueueDepth:         5000, // Large queue for burst
			ProcessingComplexity:  "simple", // Simple for high throughput
			BackpressureThreshold: 0.9,
		},
		{
			Name:                  "Sustained 10K+ Test",
			TargetThroughputIPS:   12000, // Above 10K requirement
			RampUpDuration:        4 * time.Minute,
			SustainDuration:       25 * time.Minute, // Long sustain test
			RampDownDuration:      2 * time.Minute,
			MaxQueueDepth:         3000,
			ProcessingComplexity:  "medium",
			BackpressureThreshold: 0.85,
		},
		{
			Name:                  "Complex Processing - 8K IPS",
			TargetThroughputIPS:   8000, // Lower but with complex processing
			RampUpDuration:        5 * time.Minute,
			SustainDuration:       12 * time.Minute,
			RampDownDuration:      3 * time.Minute,
			MaxQueueDepth:         1500,
			ProcessingComplexity:  "complex",
			BackpressureThreshold: 0.7,
		},
		{
			Name:                  "Peak Performance Test - 30K IPS",
			TargetThroughputIPS:   30000, // Maximum throughput test
			RampUpDuration:        3 * time.Minute,
			SustainDuration:       8 * time.Minute,
			RampDownDuration:      2 * time.Minute,
			MaxQueueDepth:         8000, // Very large queue
			ProcessingComplexity:  "simple",
			BackpressureThreshold: 0.95, // High threshold for peak test
		},
		{
			Name:                  "Variable Load Test - 10-20K IPS",
			TargetThroughputIPS:   15000, // Mid-range with variation
			RampUpDuration:        8 * time.Minute,
			SustainDuration:       18 * time.Minute,
			RampDownDuration:      4 * time.Minute,
			MaxQueueDepth:         4000,
			ProcessingComplexity:  "medium",
			BackpressureThreshold: 0.8,
		},
	}

	results := make([]*ThroughputResults, len(scenarios))

	// Run scenarios sequentially
	for i, scenario := range scenarios {
		log.Printf("Executing throughput test scenario: %s", scenario.Name)
		
		result, err := throughputManager.ExecuteScenario(ctx, scenario)
		if err != nil {
			log.Printf("Throughput test scenario %s failed: %v", scenario.Name, err)
			continue
		}
		results[i] = result
		
		// Wait between scenarios for system to stabilize
		time.Sleep(1 * time.Minute)
	}

	// Aggregate results
	pts.testResults.ThroughputResults = pts.aggregateThroughputResults(results)
	
	log.Println("Throughput testing completed")
	return nil
}

// ExecuteScenario runs a specific throughput test scenario
func (ttm *ThroughputTestManager) ExecuteScenario(ctx context.Context, scenario *ThroughputTestScenario) (*ThroughputResults, error) {
	testCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Reset metrics
	ttm.metrics = &ThroughputTestMetrics{
		TargetThroughput:    scenario.TargetThroughputIPS,
		ThroughputSamples:   make([]ThroughputSample, 0),
		ProcessingLatencies: make([]float64, 0),
		ResourceUtilization: make([]*ResourceConsumption, 0),
		QueueMetrics: &QueueMetrics{
			MaxDepth:     0,
			AvgDepth:     0,
			OverflowRate: 0,
		},
	}

	startTime := time.Now()

	// Start indication processing pipeline
	if err := ttm.startProcessingPipeline(testCtx, scenario); err != nil {
		return nil, fmt.Errorf("failed to start processing pipeline: %w", err)
	}

	// Start resource monitoring
	resourceChan := ttm.monitorResources(testCtx, 2*time.Second)

	// Execute test phases
	if err := ttm.executeRampUp(testCtx, scenario); err != nil {
		return nil, fmt.Errorf("ramp-up phase failed: %w", err)
	}

	if err := ttm.executeSustain(testCtx, scenario); err != nil {
		return nil, fmt.Errorf("sustain phase failed: %w", err)
	}

	if err := ttm.executeRampDown(testCtx, scenario); err != nil {
		return nil, fmt.Errorf("ramp-down phase failed: %w", err)
	}

	// Collect final resource samples
	go func() {
		for resource := range resourceChan {
			ttm.metrics.mu.Lock()
			ttm.metrics.ResourceUtilization = append(ttm.metrics.ResourceUtilization, resource)
			ttm.metrics.mu.Unlock()
		}
	}()

	ttm.metrics.TestDuration = time.Since(startTime)

	return ttm.generateThroughputResults(), nil
}

// startProcessingPipeline initializes the indication processing pipeline
func (ttm *ThroughputTestManager) startProcessingPipeline(ctx context.Context, scenario *ThroughputTestScenario) error {
	processor := ttm.indicationProcessor

	// Start processing workers
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		go ttm.processingWorker(ctx, scenario, i)
	}

	// Start queue monitoring
	go ttm.monitorQueues(ctx)

	// Start metrics collection
	go ttm.collectThroughputMetrics(ctx)

	return nil
}

// processingWorker simulates indication processing
func (ttm *ThroughputTestManager) processingWorker(ctx context.Context, scenario *ThroughputTestScenario, workerID int) {
	processor := ttm.indicationProcessor

	for {
		select {
		case <-ctx.Done():
			return
		case indication := <-processor.inputQueue:
			startProcessing := time.Now()
			
			// Simulate processing based on complexity
			processingTime := ttm.simulateProcessing(scenario.ProcessingComplexity)
			time.Sleep(processingTime)
			
			endProcessing := time.Now()
			processingDuration := endProcessing.Sub(startProcessing)
			queueWaitTime := startProcessing.Sub(indication.ProcessingStart)

			processed := &ProcessedIndication{
				Indication:      indication,
				ProcessingEnd:   endProcessing,
				ProcessingTimeMs: float64(processingDuration.Nanoseconds()) / 1e6,
				QueueWaitTimeMs: float64(queueWaitTime.Nanoseconds()) / 1e6,
			}

			// Try to send to output queue
			select {
			case processor.outputQueue <- processed:
				atomic.AddInt64(&processor.processedCount, 1)
				
				processor.mu.Lock()
				processor.processingTimeMs = append(processor.processingTimeMs, processed.ProcessingTimeMs)
				processor.mu.Unlock()
				
			default:
				// Output queue full, drop indication
				atomic.AddInt64(&processor.droppedCount, 1)
			}
		}
	}
}

// simulateProcessing simulates different processing complexities
func (ttm *ThroughputTestManager) simulateProcessing(complexity string) time.Duration {
	switch complexity {
	case "simple":
		// Simple processing: 0.1-0.5ms
		return time.Duration(rand.Intn(400)+100) * time.Microsecond
	case "medium":
		// Medium processing: 0.5-2ms
		return time.Duration(rand.Intn(1500)+500) * time.Microsecond
	case "complex":
		// Complex processing: 2-10ms
		return time.Duration(rand.Intn(8000)+2000) * time.Microsecond
	default:
		return time.Duration(rand.Intn(1000)+100) * time.Microsecond
	}
}

// monitorQueues monitors queue depths and backpressure
func (ttm *ThroughputTestManager) monitorQueues(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	processor := ttm.indicationProcessor

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			inputDepth := len(processor.inputQueue)
			processingDepth := len(processor.processingQueue)
			outputDepth := len(processor.outputQueue)

			processor.mu.Lock()
			processor.queueDepths = append(processor.queueDepths, inputDepth)
			processor.mu.Unlock()

			// Check for backpressure
			if float64(inputDepth)/float64(cap(processor.inputQueue)) > 0.8 {
				atomic.AddInt64(&processor.backpressureEvents, 1)
			}

			// Update queue metrics
			ttm.metrics.mu.Lock()
			if inputDepth > ttm.metrics.QueueMetrics.MaxDepth {
				ttm.metrics.QueueMetrics.MaxDepth = inputDepth
			}
			ttm.metrics.mu.Unlock()

			log.Printf("Queue depths - Input: %d, Processing: %d, Output: %d", 
				inputDepth, processingDepth, outputDepth)
		}
	}
}

// collectThroughputMetrics collects throughput metrics
func (ttm *ThroughputTestManager) collectThroughputMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	processor := ttm.indicationProcessor
	lastProcessedCount := int64(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentProcessedCount := atomic.LoadInt64(&processor.processedCount)
			throughputIPS := int(currentProcessedCount - lastProcessedCount)
			lastProcessedCount = currentProcessedCount

			// Calculate average processing latency for this interval
			processor.mu.RLock()
			avgLatency := 0.0
			if len(processor.processingTimeMs) > 0 {
				sum := 0.0
				for _, latency := range processor.processingTimeMs {
					sum += latency
				}
				avgLatency = sum / float64(len(processor.processingTimeMs))
			}
			processor.mu.RUnlock()

			sample := ThroughputSample{
				Timestamp:           time.Now(),
				IndicationsPerSec:   throughputIPS,
				ProcessingLatencyMs: avgLatency,
			}

			ttm.metrics.mu.Lock()
			ttm.metrics.ThroughputSamples = append(ttm.metrics.ThroughputSamples, sample)
			
			// Update peak throughput
			if throughputIPS > ttm.metrics.PeakThroughput {
				ttm.metrics.PeakThroughput = throughputIPS
			}
			ttm.metrics.mu.Unlock()

			log.Printf("Current throughput: %d IPS, Avg latency: %.2f ms", throughputIPS, avgLatency)
		}
	}
}

// executeRampUp gradually increases indication generation rate
func (ttm *ThroughputTestManager) executeRampUp(ctx context.Context, scenario *ThroughputTestScenario) error {
	log.Printf("Starting throughput ramp-up phase for %s", scenario.Name)

	rampUpSteps := 20
	stepDuration := scenario.RampUpDuration / time.Duration(rampUpSteps)
	processor := ttm.indicationProcessor

	for step := 1; step <= rampUpSteps; step++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate target throughput for this step
		targetThroughput := (scenario.TargetThroughputIPS * step) / rampUpSteps
		indicationInterval := time.Second / time.Duration(targetThroughput)

		log.Printf("Ramp-up step %d/%d: Target throughput %d IPS", step, rampUpSteps, targetThroughput)

		// Generate indications at target rate for this step
		stepCtx, stepCancel := context.WithTimeout(ctx, stepDuration)
		go ttm.generateIndications(stepCtx, targetThroughput, processor)

		<-stepCtx.Done()
		stepCancel()
	}

	log.Printf("Throughput ramp-up phase completed")
	return nil
}

// executeSustain maintains target throughput for specified duration
func (ttm *ThroughputTestManager) executeSustain(ctx context.Context, scenario *ThroughputTestScenario) error {
	log.Printf("Starting throughput sustain phase for %s (duration: %v, target: %d IPS)", 
		scenario.Name, scenario.SustainDuration, scenario.TargetThroughputIPS)

	processor := ttm.indicationProcessor

	// Generate indications at target rate
	sustainCtx, sustainCancel := context.WithTimeout(ctx, scenario.SustainDuration)
	defer sustainCancel()

	go ttm.generateIndications(sustainCtx, scenario.TargetThroughputIPS, processor)

	// Monitor for backpressure and adjust if necessary
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	sustainedThroughputSamples := make([]int, 0)

	for {
		select {
		case <-sustainCtx.Done():
			// Calculate sustained throughput (average of middle 80% of samples)
			if len(sustainedThroughputSamples) > 0 {
				ttm.calculateSustainedThroughput(sustainedThroughputSamples)
			}
			log.Printf("Throughput sustain phase completed")
			return nil
		case <-ticker.C:
			// Check current throughput
			ttm.metrics.mu.RLock()
			if len(ttm.metrics.ThroughputSamples) > 0 {
				lastSample := ttm.metrics.ThroughputSamples[len(ttm.metrics.ThroughputSamples)-1]
				sustainedThroughputSamples = append(sustainedThroughputSamples, lastSample.IndicationsPerSec)
				
				log.Printf("Sustain phase: Current throughput %d IPS, Queue depth %d", 
					lastSample.IndicationsPerSec, len(processor.inputQueue))
			}
			ttm.metrics.mu.RUnlock()

			// Check for backpressure
			if float64(len(processor.inputQueue))/float64(cap(processor.inputQueue)) > scenario.BackpressureThreshold {
				log.Printf("Backpressure detected: Queue utilization %.2f%%", 
					float64(len(processor.inputQueue))/float64(cap(processor.inputQueue))*100)
			}
		}
	}
}

// executeRampDown gradually decreases indication generation rate
func (ttm *ThroughputTestManager) executeRampDown(ctx context.Context, scenario *ThroughputTestScenario) error {
	log.Printf("Starting throughput ramp-down phase for %s", scenario.Name)

	rampDownSteps := 10
	stepDuration := scenario.RampDownDuration / time.Duration(rampDownSteps)
	processor := ttm.indicationProcessor

	for step := 1; step <= rampDownSteps; step++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate target throughput for this step (decreasing)
		remainingFactor := float64(rampDownSteps-step) / float64(rampDownSteps)
		targetThroughput := int(float64(scenario.TargetThroughputIPS) * remainingFactor)

		if targetThroughput > 0 {
			log.Printf("Ramp-down step %d/%d: Target throughput %d IPS", step, rampDownSteps, targetThroughput)

			// Generate indications at reduced rate for this step
			stepCtx, stepCancel := context.WithTimeout(ctx, stepDuration)
			go ttm.generateIndications(stepCtx, targetThroughput, processor)

			<-stepCtx.Done()
			stepCancel()
		} else {
			// No more indications to generate
			time.Sleep(stepDuration)
		}
	}

	log.Printf("Throughput ramp-down phase completed")
	return nil
}

// generateIndications generates indications at specified rate
func (ttm *ThroughputTestManager) generateIndications(ctx context.Context, targetThroughputIPS int, processor *IndicationProcessor) {
	if targetThroughputIPS <= 0 {
		return
	}

	interval := time.Second / time.Duration(targetThroughputIPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	indicationID := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			indication := &Indication{
				ID:              fmt.Sprintf("indication-%d-%d", time.Now().Unix(), indicationID),
				NodeID:          fmt.Sprintf("node-%d", rand.Intn(100)),
				SubscriptionID:  fmt.Sprintf("sub-%d", rand.Intn(1000)),
				Timestamp:       time.Now(),
				Data:            make([]byte, rand.Intn(1000)+100), // Random payload size
				ProcessingStart: time.Now(),
			}

			// Try to send to input queue
			select {
			case processor.inputQueue <- indication:
				indicationID++
			default:
				// Queue full, drop indication
				atomic.AddInt64(&processor.droppedCount, 1)
			}
		}
	}
}

// calculateSustainedThroughput calculates sustained throughput from samples
func (ttm *ThroughputTestManager) calculateSustainedThroughput(samples []int) {
	if len(samples) < 5 {
		return
	}

	// Sort samples
	sortedSamples := make([]int, len(samples))
	copy(sortedSamples, samples)
	
	for i := 0; i < len(sortedSamples); i++ {
		for j := i + 1; j < len(sortedSamples); j++ {
			if sortedSamples[i] > sortedSamples[j] {
				sortedSamples[i], sortedSamples[j] = sortedSamples[j], sortedSamples[i]
			}
		}
	}

	// Take middle 80% of samples
	start := len(sortedSamples) / 10
	end := len(sortedSamples) - start

	sum := 0
	count := 0
	for i := start; i < end; i++ {
		sum += sortedSamples[i]
		count++
	}

	if count > 0 {
		ttm.metrics.mu.Lock()
		ttm.metrics.SustainedThroughput = sum / count
		ttm.metrics.mu.Unlock()
	}
}

// monitorResources monitors system resources during throughput testing
func (ttm *ThroughputTestManager) monitorResources(ctx context.Context, interval time.Duration) <-chan *ResourceConsumption {
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
				// Simulate resource monitoring with realistic values
				usage := &ResourceConsumption{
					CPUUsagePercent:    float64(rand.Intn(90) + 10),
					MemoryUsageMB:      float64(rand.Intn(2000) + 1000),
					NetworkBytesPerSec: int64(rand.Intn(5000000) + 1000000),
					DiskIOPerSec:       int64(rand.Intn(50000) + 10000),
					GoroutineCount:     rand.Intn(2000) + 500,
					GCPauseMs:          float64(rand.Intn(20) + 1),
				}

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

// generateThroughputResults creates comprehensive throughput test results
func (ttm *ThroughputTestManager) generateThroughputResults() *ThroughputResults {
	processor := ttm.indicationProcessor
	metrics := ttm.metrics

	// Calculate queue metrics
	processor.mu.RLock()
	avgQueueDepth := 0.0
	if len(processor.queueDepths) > 0 {
		sum := 0
		for _, depth := range processor.queueDepths {
			sum += depth
		}
		avgQueueDepth = float64(sum) / float64(len(processor.queueDepths))
	}
	processor.mu.RUnlock()

	queueMetrics := &QueueMetrics{
		MaxDepth:     metrics.QueueMetrics.MaxDepth,
		AvgDepth:     avgQueueDepth,
		OverflowRate: 0.0,
	}

	// Calculate overflow rate
	totalIndications := atomic.LoadInt64(&processor.processedCount) + atomic.LoadInt64(&processor.droppedCount)
	if totalIndications > 0 {
		queueMetrics.OverflowRate = float64(atomic.LoadInt64(&processor.droppedCount)) / float64(totalIndications) * 100
	}

	// Calculate achieved throughput (average of all samples)
	achievedThroughput := 0
	if len(metrics.ThroughputSamples) > 0 {
		sum := 0
		for _, sample := range metrics.ThroughputSamples {
			sum += sample.IndicationsPerSec
		}
		achievedThroughput = sum / len(metrics.ThroughputSamples)
	}

	// Copy processing latencies
	processor.mu.RLock()
	processingLatencies := make([]float64, len(processor.processingTimeMs))
	copy(processingLatencies, processor.processingTimeMs)
	processor.mu.RUnlock()

	return &ThroughputResults{
		MaxIndicationsPerSecond: metrics.PeakThroughput,
		SustainedThroughput:     metrics.SustainedThroughput,
		ThroughputOverTime:      metrics.ThroughputSamples,
		ProcessingLatencyMs:     processingLatencies,
		QueueDepthMetrics:       queueMetrics,
		BackpressureEvents:      int(atomic.LoadInt64(&processor.backpressureEvents)),
		Timestamp:               time.Now(),
	}
}

// aggregateThroughputResults combines results from multiple scenarios
func (pts *PerformanceTestSuite) aggregateThroughputResults(results []*ThroughputResults) *ThroughputResults {
	if len(results) == 0 {
		return &ThroughputResults{Timestamp: time.Now()}
	}

	aggregated := &ThroughputResults{
		ThroughputOverTime:  make([]ThroughputSample, 0),
		ProcessingLatencyMs: make([]float64, 0),
		QueueDepthMetrics:   &QueueMetrics{},
		Timestamp:           time.Now(),
	}

	maxThroughput := 0
	totalSustained := 0
	totalBackpressure := 0
	validResults := 0

	for _, result := range results {
		if result == nil {
			continue
		}
		
		validResults++
		
		if result.MaxIndicationsPerSecond > maxThroughput {
			maxThroughput = result.MaxIndicationsPerSecond
		}
		
		totalSustained += result.SustainedThroughput
		totalBackpressure += result.BackpressureEvents
		
		// Aggregate throughput samples
		aggregated.ThroughputOverTime = append(aggregated.ThroughputOverTime, result.ThroughputOverTime...)
		
		// Aggregate processing latencies
		aggregated.ProcessingLatencyMs = append(aggregated.ProcessingLatencyMs, result.ProcessingLatencyMs...)
		
		// Update queue metrics
		if result.QueueDepthMetrics.MaxDepth > aggregated.QueueDepthMetrics.MaxDepth {
			aggregated.QueueDepthMetrics.MaxDepth = result.QueueDepthMetrics.MaxDepth
		}
		aggregated.QueueDepthMetrics.AvgDepth += result.QueueDepthMetrics.AvgDepth
		aggregated.QueueDepthMetrics.OverflowRate += result.QueueDepthMetrics.OverflowRate
	}

	if validResults > 0 {
		aggregated.MaxIndicationsPerSecond = maxThroughput
		aggregated.SustainedThroughput = totalSustained / validResults
		aggregated.BackpressureEvents = totalBackpressure
		aggregated.QueueDepthMetrics.AvgDepth /= float64(validResults)
		aggregated.QueueDepthMetrics.OverflowRate /= float64(validResults)
	}

	return aggregated
}