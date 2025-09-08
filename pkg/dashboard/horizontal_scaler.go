/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// HorizontalScaler type is now defined in types.go to avoid redeclaration

// ScalingPolicy defines scaling behavior for a component
type ScalingPolicy struct {
	ComponentName    string
	MinInstances     int
	MaxInstances     int
	TargetCPU        float64 // Target CPU utilization (0.0-1.0)
	TargetMemory     float64 // Target memory utilization (0.0-1.0)
	TargetLatency    time.Duration
	TargetThroughput int64 // Requests per second
	ScaleUpCooldown  time.Duration
	ScaleDownCooldown time.Duration
	ScaleUpThreshold  float64 // Threshold to trigger scale up
	ScaleDownThreshold float64 // Threshold to trigger scale down
	Enabled          bool
}

// InstanceGroup manages a group of instances for a component
type InstanceGroup struct {
	ComponentName   string
	Instances       []*Instance
	DesiredCount    int32
	RunningCount    int32
	PendingCount    int32
	LastScaleUp     time.Time
	LastScaleDown   time.Time
	mu              sync.RWMutex
}

// Instance represents a single instance of a component
type Instance struct {
	ID           string
	ComponentName string
	Status       InstanceStatus
	StartTime    time.Time
	CPUUsage     float64
	MemoryUsage  float64
	RequestCount int64
	ErrorCount   int64
	LastUpdate   time.Time
	mu           sync.RWMutex
}

// InstanceStatus represents the status of an instance
type InstanceStatus int

const (
	InstancePending InstanceStatus = iota
	InstanceRunning
	InstanceTerminating
	InstanceFailed
)

// ScalingMetrics tracks scaling-related metrics
type ScalingMetrics struct {
	ScaleUpEvents   uint64
	ScaleDownEvents uint64
	TotalInstances  map[string]int32
	ResourceUsage   map[string]*ResourceUsage
	mu              sync.RWMutex
}


// ScaleExecutor handles the actual scaling operations
type ScaleExecutor struct {
	kubernetesClient KubernetesClient
	deploymentSpecs  map[string]*DeploymentSpec
	mu               sync.RWMutex
}

// KubernetesClient interface for Kubernetes operations
type KubernetesClient interface {
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error
	GetDeploymentStatus(ctx context.Context, namespace, name string) (*DeploymentStatus, error)
	CreateDeployment(ctx context.Context, spec *DeploymentSpec) error
	DeleteDeployment(ctx context.Context, namespace, name string) error
}

// DeploymentSpec defines a Kubernetes deployment specification
type DeploymentSpec struct {
	Name      string
	Namespace string
	Image     string
	Replicas  int32
	Resources *ResourceRequirements
	Labels    map[string]string
	Env       map[string]string
}

// ResourceRequirements defines resource requirements for a deployment
type ResourceRequirements struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	ReadyReplicas     int32
	AvailableReplicas int32
	UnavailableReplicas int32
	UpdatedReplicas   int32
}

// AutoScaler manages automatic scaling decisions
type AutoScaler struct {
	scaler          *HorizontalScaler
	decisionEngine  *ScalingDecisionEngine
	metricsCollector *MetricsCollector
	enabled         int32 // atomic boolean
	interval        time.Duration
}

// ScalingDecisionEngine makes scaling decisions based on metrics
type ScalingDecisionEngine struct {
	algorithms map[string]ScalingAlgorithm
	mu         sync.RWMutex
}

// ScalingAlgorithm defines different scaling algorithms
type ScalingAlgorithm interface {
	CalculateDesiredReplicas(current int32, metrics *ResourceUsage, policy *ScalingPolicy) int32
	GetName() string
}

// MetricsCollector collects metrics from various sources
type MetricsCollector struct {
	sources    map[string]MetricsSource
	aggregator *MetricsAggregator
	mu         sync.RWMutex
}

// MetricsSource interface for different metrics sources
type MetricsSource interface {
	CollectMetrics(ctx context.Context, componentName string) (*ResourceUsage, error)
	GetName() string
}

// MetricsAggregator aggregates metrics from multiple sources
type MetricsAggregator struct {
	windowSize time.Duration
	history    map[string]*MetricsHistory
	mu         sync.RWMutex
}

// MetricsHistory maintains historical metrics data
type MetricsHistory struct {
	Samples    []MetricsSample
	MaxSamples int
	mu         sync.RWMutex
}

// MetricsSample represents a single metrics sample
type MetricsSample struct {
	Timestamp time.Time
	Usage     *ResourceUsage
}

// NewHorizontalScaler creates a new horizontal scaler
func NewHorizontalScaler(kubernetesClient KubernetesClient) *HorizontalScaler {
	return &HorizontalScaler{
		scalingPolicies: make(map[string]*ScalingPolicy),
		instances:       make(map[string]*InstanceGroup),
		metrics:         NewScalingMetrics(),
		scaleExecutor:   NewScaleExecutor(kubernetesClient),
	}
}

// AddScalingPolicy adds a scaling policy for a component
func (hs *HorizontalScaler) AddScalingPolicy(policy *ScalingPolicy) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	
	hs.scalingPolicies[policy.ComponentName] = policy
	
	// Initialize instance group if it doesn't exist
	if _, exists := hs.instances[policy.ComponentName]; !exists {
		hs.instances[policy.ComponentName] = &InstanceGroup{
			ComponentName: policy.ComponentName,
			Instances:     make([]*Instance, 0),
			DesiredCount:  int32(policy.MinInstances),
		}
	}
}

// ScaleUp increases the number of instances for a component
func (hs *HorizontalScaler) ScaleUp(ctx context.Context, componentName string, targetReplicas int32) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	
	policy, exists := hs.scalingPolicies[componentName]
	if !exists {
		return fmt.Errorf("no scaling policy found for component %s", componentName)
	}
	
	instanceGroup, exists := hs.instances[componentName]
	if !exists {
		return fmt.Errorf("no instance group found for component %s", componentName)
	}
	
	// Check cooldown period
	if time.Since(instanceGroup.LastScaleUp) < policy.ScaleUpCooldown {
		return fmt.Errorf("scale up cooldown period not elapsed for component %s", componentName)
	}
	
	// Enforce max instances limit
	if targetReplicas > int32(policy.MaxInstances) {
		targetReplicas = int32(policy.MaxInstances)
	}
	
	currentReplicas := atomic.LoadInt32(&instanceGroup.RunningCount)
	if targetReplicas <= currentReplicas {
		return nil // No scaling needed
	}
	
	// Execute scaling
	err := hs.scaleExecutor.ScaleUp(ctx, componentName, targetReplicas)
	if err != nil {
		return fmt.Errorf("failed to scale up component %s: %w", componentName, err)
	}
	
	// Update state
	atomic.StoreInt32(&instanceGroup.DesiredCount, targetReplicas)
	instanceGroup.LastScaleUp = time.Now()
	atomic.AddUint64(&hs.metrics.ScaleUpEvents, 1)
	
	return nil
}

// ScaleDown decreases the number of instances for a component
func (hs *HorizontalScaler) ScaleDown(ctx context.Context, componentName string, targetReplicas int32) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	
	policy, exists := hs.scalingPolicies[componentName]
	if !exists {
		return fmt.Errorf("no scaling policy found for component %s", componentName)
	}
	
	instanceGroup, exists := hs.instances[componentName]
	if !exists {
		return fmt.Errorf("no instance group found for component %s", componentName)
	}
	
	// Check cooldown period
	if time.Since(instanceGroup.LastScaleDown) < policy.ScaleDownCooldown {
		return fmt.Errorf("scale down cooldown period not elapsed for component %s", componentName)
	}
	
	// Enforce min instances limit
	if targetReplicas < int32(policy.MinInstances) {
		targetReplicas = int32(policy.MinInstances)
	}
	
	currentReplicas := atomic.LoadInt32(&instanceGroup.RunningCount)
	if targetReplicas >= currentReplicas {
		return nil // No scaling needed
	}
	
	// Execute scaling
	err := hs.scaleExecutor.ScaleDown(ctx, componentName, targetReplicas)
	if err != nil {
		return fmt.Errorf("failed to scale down component %s: %w", componentName, err)
	}
	
	// Update state
	atomic.StoreInt32(&instanceGroup.DesiredCount, targetReplicas)
	instanceGroup.LastScaleDown = time.Now()
	atomic.AddUint64(&hs.metrics.ScaleDownEvents, 1)
	
	return nil
}

// GetInstanceGroup returns the instance group for a component
func (hs *HorizontalScaler) GetInstanceGroup(componentName string) (*InstanceGroup, bool) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	
	group, exists := hs.instances[componentName]
	return group, exists
}

// UpdateInstanceMetrics updates metrics for an instance
func (hs *HorizontalScaler) UpdateInstanceMetrics(instanceID string, cpuUsage, memoryUsage float64, requestCount, errorCount int64) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	
	for _, group := range hs.instances {
		group.mu.RLock()
		for _, instance := range group.Instances {
			if instance.ID == instanceID {
				instance.mu.Lock()
				instance.CPUUsage = cpuUsage
				instance.MemoryUsage = memoryUsage
				instance.RequestCount = requestCount
				instance.ErrorCount = errorCount
				instance.LastUpdate = time.Now()
				instance.mu.Unlock()
				break
			}
		}
		group.mu.RUnlock()
	}
}

// NewScalingMetrics creates new scaling metrics
func NewScalingMetrics() *ScalingMetrics {
	return &ScalingMetrics{
		TotalInstances: make(map[string]int32),
		ResourceUsage:  make(map[string]*ResourceUsage),
	}
}

// UpdateResourceUsage updates resource usage metrics
func (sm *ScalingMetrics) UpdateResourceUsage(componentName string, usage *ResourceUsage) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.ResourceUsage[componentName] = usage
}

// GetResourceUsage gets resource usage for a component
func (sm *ScalingMetrics) GetResourceUsage(componentName string) (*ResourceUsage, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	usage, exists := sm.ResourceUsage[componentName]
	return usage, exists
}

// NewScaleExecutor creates a new scale executor
func NewScaleExecutor(kubernetesClient KubernetesClient) *ScaleExecutor {
	return &ScaleExecutor{
		kubernetesClient: kubernetesClient,
		deploymentSpecs:  make(map[string]*DeploymentSpec),
	}
}

// ScaleUp scales up a component
func (se *ScaleExecutor) ScaleUp(ctx context.Context, componentName string, targetReplicas int32) error {
	se.mu.RLock()
	spec, exists := se.deploymentSpecs[componentName]
	se.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("no deployment spec found for component %s", componentName)
	}
	
	return se.kubernetesClient.ScaleDeployment(ctx, spec.Namespace, spec.Name, targetReplicas)
}

// ScaleDown scales down a component
func (se *ScaleExecutor) ScaleDown(ctx context.Context, componentName string, targetReplicas int32) error {
	se.mu.RLock()
	spec, exists := se.deploymentSpecs[componentName]
	se.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("no deployment spec found for component %s", componentName)
	}
	
	return se.kubernetesClient.ScaleDeployment(ctx, spec.Namespace, spec.Name, targetReplicas)
}

// AddDeploymentSpec adds a deployment specification
func (se *ScaleExecutor) AddDeploymentSpec(componentName string, spec *DeploymentSpec) {
	se.mu.Lock()
	defer se.mu.Unlock()
	
	se.deploymentSpecs[componentName] = spec
}

// NewAutoScaler creates a new auto scaler
func NewAutoScaler(scaler *HorizontalScaler, interval time.Duration) *AutoScaler {
	return &AutoScaler{
		scaler:          scaler,
		decisionEngine:  NewScalingDecisionEngine(),
		metricsCollector: NewMetricsCollector(),
		interval:        interval,
	}
}

// Start starts the auto scaler
func (as *AutoScaler) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&as.enabled, 0, 1) {
		return fmt.Errorf("auto scaler is already running")
	}
	
	go as.scalingLoop(ctx)
	return nil
}

// Stop stops the auto scaler
func (as *AutoScaler) Stop() {
	atomic.StoreInt32(&as.enabled, 0)
}

// scalingLoop runs the main scaling loop
func (as *AutoScaler) scalingLoop(ctx context.Context) {
	ticker := time.NewTicker(as.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if atomic.LoadInt32(&as.enabled) == 0 {
				return
			}
			
			as.evaluateScaling(ctx)
		}
	}
}

// evaluateScaling evaluates scaling decisions for all components
func (as *AutoScaler) evaluateScaling(ctx context.Context) {
	as.scaler.mu.RLock()
	policies := make(map[string]*ScalingPolicy)
	for k, v := range as.scaler.scalingPolicies {
		if v.Enabled {
			policies[k] = v
		}
	}
	as.scaler.mu.RUnlock()
	
	for componentName, policy := range policies {
		// Collect current metrics
		metrics, err := as.metricsCollector.CollectComponentMetrics(ctx, componentName)
		if err != nil {
			continue // Skip this component
		}
		
		// Get current instance count
		instanceGroup, exists := as.scaler.GetInstanceGroup(componentName)
		if !exists {
			continue
		}
		
		currentReplicas := atomic.LoadInt32(&instanceGroup.RunningCount)
		
		// Calculate desired replicas
		desiredReplicas := as.decisionEngine.CalculateDesiredReplicas(
			currentReplicas, metrics, policy)
		
		// Execute scaling if needed
		if desiredReplicas > currentReplicas {
			as.scaler.ScaleUp(ctx, componentName, desiredReplicas)
		} else if desiredReplicas < currentReplicas {
			as.scaler.ScaleDown(ctx, componentName, desiredReplicas)
		}
	}
}

// NewScalingDecisionEngine creates a new scaling decision engine
func NewScalingDecisionEngine() *ScalingDecisionEngine {
	engine := &ScalingDecisionEngine{
		algorithms: make(map[string]ScalingAlgorithm),
	}
	
	// Register default algorithms
	engine.RegisterAlgorithm(&CPUBasedScaling{})
	engine.RegisterAlgorithm(&MemoryBasedScaling{})
	engine.RegisterAlgorithm(&LatencyBasedScaling{})
	engine.RegisterAlgorithm(&ThroughputBasedScaling{})
	engine.RegisterAlgorithm(&CompositeScaling{})
	
	return engine
}

// RegisterAlgorithm registers a scaling algorithm
func (sde *ScalingDecisionEngine) RegisterAlgorithm(algorithm ScalingAlgorithm) {
	sde.mu.Lock()
	defer sde.mu.Unlock()
	
	sde.algorithms[algorithm.GetName()] = algorithm
}

// CalculateDesiredReplicas calculates desired replicas using composite algorithm
func (sde *ScalingDecisionEngine) CalculateDesiredReplicas(current int32, metrics *ResourceUsage, policy *ScalingPolicy) int32 {
	sde.mu.RLock()
	algorithm, exists := sde.algorithms["composite"]
	sde.mu.RUnlock()
	
	if !exists {
		return current // No algorithm available
	}
	
	return algorithm.CalculateDesiredReplicas(current, metrics, policy)
}

// CPUBasedScaling implements CPU-based scaling
type CPUBasedScaling struct{}

func (cbs *CPUBasedScaling) GetName() string {
	return "cpu"
}

func (cbs *CPUBasedScaling) CalculateDesiredReplicas(current int32, metrics *ResourceUsage, policy *ScalingPolicy) int32 {
	if metrics.CPUUsage == 0 {
		return current
	}
	
	utilizationRatio := metrics.CPUUsage / policy.TargetCPU
	desiredReplicas := int32(math.Ceil(float64(current) * utilizationRatio))
	
	// Apply bounds
	if desiredReplicas < int32(policy.MinInstances) {
		desiredReplicas = int32(policy.MinInstances)
	}
	if desiredReplicas > int32(policy.MaxInstances) {
		desiredReplicas = int32(policy.MaxInstances)
	}
	
	return desiredReplicas
}

// MemoryBasedScaling implements memory-based scaling
type MemoryBasedScaling struct{}

func (mbs *MemoryBasedScaling) GetName() string {
	return "memory"
}

func (mbs *MemoryBasedScaling) CalculateDesiredReplicas(current int32, metrics *ResourceUsage, policy *ScalingPolicy) int32 {
	if metrics.MemoryUsage == 0 {
		return current
	}
	
	utilizationRatio := metrics.MemoryUsage / policy.TargetMemory
	desiredReplicas := int32(math.Ceil(float64(current) * utilizationRatio))
	
	// Apply bounds
	if desiredReplicas < int32(policy.MinInstances) {
		desiredReplicas = int32(policy.MinInstances)
	}
	if desiredReplicas > int32(policy.MaxInstances) {
		desiredReplicas = int32(policy.MaxInstances)
	}
	
	return desiredReplicas
}

// LatencyBasedScaling implements latency-based scaling
type LatencyBasedScaling struct{}

func (lbs *LatencyBasedScaling) GetName() string {
	return "latency"
}

func (lbs *LatencyBasedScaling) CalculateDesiredReplicas(current int32, metrics *ResourceUsage, policy *ScalingPolicy) int32 {
	if metrics.AverageLatency == 0 || policy.TargetLatency == 0 {
		return current
	}
	
	latencyRatio := float64(metrics.AverageLatency) / float64(policy.TargetLatency)
	if latencyRatio > 1.0 {
		// Scale up if latency is too high
		desiredReplicas := int32(math.Ceil(float64(current) * latencyRatio))
		if desiredReplicas > int32(policy.MaxInstances) {
			desiredReplicas = int32(policy.MaxInstances)
		}
		return desiredReplicas
	}
	
	return current
}

// ThroughputBasedScaling implements throughput-based scaling
type ThroughputBasedScaling struct{}

func (tbs *ThroughputBasedScaling) GetName() string {
	return "throughput"
}

func (tbs *ThroughputBasedScaling) CalculateDesiredReplicas(current int32, metrics *ResourceUsage, policy *ScalingPolicy) int32 {
	if metrics.RequestRate == 0 || policy.TargetThroughput == 0 {
		return current
	}
	
	throughputRatio := metrics.RequestRate / float64(policy.TargetThroughput)
	desiredReplicas := int32(math.Ceil(float64(current) * throughputRatio))
	
	// Apply bounds
	if desiredReplicas < int32(policy.MinInstances) {
		desiredReplicas = int32(policy.MinInstances)
	}
	if desiredReplicas > int32(policy.MaxInstances) {
		desiredReplicas = int32(policy.MaxInstances)
	}
	
	return desiredReplicas
}

// CompositeScaling implements composite scaling using multiple metrics
type CompositeScaling struct{}

func (cs *CompositeScaling) GetName() string {
	return "composite"
}

func (cs *CompositeScaling) CalculateDesiredReplicas(current int32, metrics *ResourceUsage, policy *ScalingPolicy) int32 {
	// Calculate desired replicas based on different metrics
	cpuScaling := &CPUBasedScaling{}
	memoryScaling := &MemoryBasedScaling{}
	latencyScaling := &LatencyBasedScaling{}
	throughputScaling := &ThroughputBasedScaling{}
	
	cpuReplicas := cpuScaling.CalculateDesiredReplicas(current, metrics, policy)
	memoryReplicas := memoryScaling.CalculateDesiredReplicas(current, metrics, policy)
	latencyReplicas := latencyScaling.CalculateDesiredReplicas(current, metrics, policy)
	throughputReplicas := throughputScaling.CalculateDesiredReplicas(current, metrics, policy)
	
	// Take the maximum to ensure all constraints are satisfied
	desiredReplicas := current
	if cpuReplicas > desiredReplicas {
		desiredReplicas = cpuReplicas
	}
	if memoryReplicas > desiredReplicas {
		desiredReplicas = memoryReplicas
	}
	if latencyReplicas > desiredReplicas {
		desiredReplicas = latencyReplicas
	}
	if throughputReplicas > desiredReplicas {
		desiredReplicas = throughputReplicas
	}
	
	return desiredReplicas
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		sources:    make(map[string]MetricsSource),
		aggregator: NewMetricsAggregator(time.Minute * 5), // 5-minute window
	}
}

// AddMetricsSource adds a metrics source
func (mc *MetricsCollector) AddMetricsSource(source MetricsSource) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.sources[source.GetName()] = source
}

// CollectComponentMetrics collects metrics for a component
func (mc *MetricsCollector) CollectComponentMetrics(ctx context.Context, componentName string) (*ResourceUsage, error) {
	mc.mu.RLock()
	sources := make([]MetricsSource, 0, len(mc.sources))
	for _, source := range mc.sources {
		sources = append(sources, source)
	}
	mc.mu.RUnlock()
	
	// Collect from all sources
	var allUsage []*ResourceUsage
	for _, source := range sources {
		usage, err := source.CollectMetrics(ctx, componentName)
		if err == nil && usage != nil {
			allUsage = append(allUsage, usage)
		}
	}
	
	if len(allUsage) == 0 {
		return nil, fmt.Errorf("no metrics available for component %s", componentName)
	}
	
	// Aggregate metrics
	aggregated := mc.aggregator.AggregateMetrics(componentName, allUsage)
	return aggregated, nil
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(windowSize time.Duration) *MetricsAggregator {
	return &MetricsAggregator{
		windowSize: windowSize,
		history:    make(map[string]*MetricsHistory),
	}
}

// AggregateMetrics aggregates multiple resource usage metrics
func (ma *MetricsAggregator) AggregateMetrics(componentName string, usages []*ResourceUsage) *ResourceUsage {
	if len(usages) == 0 {
		return nil
	}
	
	// Simple averaging for now
	aggregated := &ResourceUsage{
		LastUpdated: time.Now(),
	}
	
	for _, usage := range usages {
		aggregated.CPUUsage += usage.CPUUsage
		aggregated.MemoryUsage += usage.MemoryUsage
		aggregated.NetworkIn += usage.NetworkIn
		aggregated.NetworkOut += usage.NetworkOut
		aggregated.RequestRate += usage.RequestRate
		aggregated.ErrorRate += usage.ErrorRate
		aggregated.AverageLatency += usage.AverageLatency
	}
	
	count := float64(len(usages))
	aggregated.CPUUsage /= count
	aggregated.MemoryUsage /= count
	aggregated.RequestRate /= count
	aggregated.ErrorRate /= count
	aggregated.AverageLatency = time.Duration(float64(aggregated.AverageLatency) / count)
	
	// Store in history
	ma.addToHistory(componentName, aggregated)
	
	return aggregated
}

// addToHistory adds metrics to historical data
func (ma *MetricsAggregator) addToHistory(componentName string, usage *ResourceUsage) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	
	history, exists := ma.history[componentName]
	if !exists {
		history = &MetricsHistory{
			Samples:    make([]MetricsSample, 0),
			MaxSamples: 100, // Keep last 100 samples
		}
		ma.history[componentName] = history
	}
	
	history.mu.Lock()
	defer history.mu.Unlock()
	
	sample := MetricsSample{
		Timestamp: time.Now(),
		Usage:     usage,
	}
	
	history.Samples = append(history.Samples, sample)
	
	// Remove old samples if we exceed the limit
	if len(history.Samples) > history.MaxSamples {
		history.Samples = history.Samples[1:]
	}
}