/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SMONephioIntegrationLayer provides comprehensive integration with SMO and Nephio R5
type SMONephioIntegrationLayer struct {
	// SMO Integration components
	smoClient               *EnhancedSMOClient
	nonRTRICClient          *NonRTRICClient
	policyManagerClient     *PolicyManagerClient
	rAppManagerClient       *RAppManagerClient
	
	// Nephio R5 Integration components
	porchClient             *PorchAPIClient
	oCloudClient            *OCloudClient
	packageManagerClient    *PackageManagerClient
	resourceProvisionerClient *ResourceProvisionerClient
	
	// Performance optimization
	smoLoadBalancer         *SMOLoadBalancer
	nephioLoadBalancer      *NephioLoadBalancer
	requestCache            *IntegrationCache
	circuitBreaker          *IntegrationCircuitBreaker
	
	// Monitoring and observability
	integrationMetrics      *IntegrationMetrics
	performanceTracker      *IntegrationPerformanceTracker
	healthMonitor           *IntegrationHealthMonitor
	
	// Configuration
	config                  *IntegrationConfig
	
	// State management
	smoConnectionStatus     int32
	nephioConnectionStatus  int32
	running                 int32
	
	// Statistics
	stats                   IntegrationStats
	mu                      sync.RWMutex
}

// IntegrationConfig defines SMO and Nephio R5 integration configuration
type IntegrationConfig struct {
	// SMO Configuration
	SMOEndpoint             string        `json:"smoEndpoint"`
	NonRTRICEndpoint        string        `json:"nonRTRICEndpoint"`
	PolicyManagerEndpoint   string        `json:"policyManagerEndpoint"`
	RAppManagerEndpoint     string        `json:"rAppManagerEndpoint"`
	
	// Nephio R5 Configuration
	PorchAPIEndpoint        string        `json:"porchAPIEndpoint"`
	OCloudEndpoint          string        `json:"oCloudEndpoint"`
	PackageManagerEndpoint  string        `json:"packageManagerEndpoint"`
	ResourceProvisionerEndpoint string    `json:"resourceProvisionerEndpoint"`
	
	// Kubernetes Configuration
	KubeConfigPath          string        `json:"kubeConfigPath"`
	Namespace               string        `json:"namespace"`
	
	// Performance Configuration
	MaxConcurrentRequests   int           `json:"maxConcurrentRequests"`
	RequestTimeout          time.Duration `json:"requestTimeout"`
	RetryAttempts           int           `json:"retryAttempts"`
	RetryBackoff            time.Duration `json:"retryBackoff"`
	
	// Cache Configuration
	CacheEnabled            bool          `json:"cacheEnabled"`
	CacheTTL                time.Duration `json:"cacheTTL"`
	CacheMaxSize            int           `json:"cacheMaxSize"`
	
	// Circuit Breaker Configuration
	CircuitBreakerEnabled   bool          `json:"circuitBreakerEnabled"`
	FailureThreshold        int           `json:"failureThreshold"`
	RecoveryTimeout         time.Duration `json:"recoveryTimeout"`
	
	// Health Check Configuration
	HealthCheckInterval     time.Duration `json:"healthCheckInterval"`
	HealthCheckTimeout      time.Duration `json:"healthCheckTimeout"`
	
	// Load Balancing Configuration
	LoadBalancingEnabled    bool          `json:"loadBalancingEnabled"`
	LoadBalancingStrategy   string        `json:"loadBalancingStrategy"`
}

// IntegrationStats tracks integration performance metrics
type IntegrationStats struct {
	// SMO Statistics
	SMORequestsTotal        uint64        `json:"smoRequestsTotal"`
	SMORequestsSuccessful   uint64        `json:"smoRequestsSuccessful"`
	SMORequestsFailed       uint64        `json:"smoRequestsFailed"`
	SMOAverageLatencyMs     float64       `json:"smoAverageLatencyMs"`
	SMOConnectionStatus     string        `json:"smoConnectionStatus"`
	
	// Non-RT RIC Statistics
	NonRTRICPolicyUpdates   uint64        `json:"nonRTRICPolicyUpdates"`
	NonRTRICRAppDeployments uint64        `json:"nonRTRICRAppDeployments"`
	NonRTRICEnrichmentInfo  uint64        `json:"nonRTRICEnrichmentInfo"`
	
	// Nephio R5 Statistics
	NephioRequestsTotal     uint64        `json:"nephioRequestsTotal"`
	NephioRequestsSuccessful uint64       `json:"nephioRequestsSuccessful"`
	NephioRequestsFailed    uint64        `json:"nephioRequestsFailed"`
	NephioAverageLatencyMs  float64       `json:"nephioAverageLatencyMs"`
	NephioConnectionStatus  string        `json:"nephioConnectionStatus"`
	
	// Porch Statistics
	PackageRevisions        uint64        `json:"packageRevisions"`
	PackageDeployments      uint64        `json:"packageDeployments"`
	PackageValidations      uint64        `json:"packageValidations"`
	
	// O-Cloud Statistics
	ResourcePools           uint64        `json:"resourcePools"`
	ResourceAllocations     uint64        `json:"resourceAllocations"`
	EnergyEfficiencyScore   float64       `json:"energyEfficiencyScore"`
	
	// Performance Statistics
	CacheHitRatio           float64       `json:"cacheHitRatio"`
	CircuitBreakerTrips     uint64        `json:"circuitBreakerTrips"`
	LoadBalancerSwitches    uint64        `json:"loadBalancerSwitches"`
	
	// Error Statistics
	TimeoutErrors           uint64        `json:"timeoutErrors"`
	ConnectionErrors        uint64        `json:"connectionErrors"`
	AuthenticationErrors    uint64        `json:"authenticationErrors"`
	
	LastUpdated             time.Time     `json:"lastUpdated"`
}

// EnhancedSMOClient provides high-performance SMO integration
type EnhancedSMOClient struct {
	endpoint                string
	httpClient              *http.Client
	
	// Performance optimization
	connectionPool          *HTTPConnectionPool
	requestCache            *RequestCache
	circuitBreaker          *CircuitBreaker
	
	// Authentication and security
	authToken               string
	tokenExpiry             time.Time
	
	// Metrics
	requestCount            uint64
	errorCount              uint64
	averageLatency          uint64
	
	mu                      sync.RWMutex
}

// PorchAPIClient provides high-performance Porch integration
type PorchAPIClient struct {
	endpoint                string
	kubeClient              kubernetes.Interface
	
	// Package management
	packageCache            *PackageCache
	packageValidator        *PackageValidator
	packageDeployer         *PackageDeployer
	
	// Performance optimization
	batchProcessor          *PackageBatchProcessor
	asyncProcessor          *AsyncPackageProcessor
	
	// Metrics
	packageOperations       uint64
	deploymentSuccess       uint64
	validationErrors        uint64
	
	mu                      sync.RWMutex
}

// OCloudClient provides O-Cloud resource management integration
type OCloudClient struct {
	endpoint                string
	httpClient              *http.Client
	
	// Resource management
	resourcePoolManager     *ResourcePoolManager
	energyManager           *EnergyManager
	capacityPlanner         *CapacityPlanner
	
	// Performance tracking
	resourceUtilization     map[string]float64
	energyEfficiency        float64
	allocationLatency       time.Duration
	
	mu                      sync.RWMutex
}

// SMO Integration Types

// SMOPolicyRequest represents an SMO policy management request
type SMOPolicyRequest struct {
	PolicyID                string                 `json:"policyId"`
	PolicyType              string                 `json:"policyType"`
	PolicyContent           map[string]interface{} `json:"policyContent"`
	TargetE2Nodes           []string               `json:"targetE2Nodes"`
	Priority                int                    `json:"priority"`
	Timestamp               time.Time              `json:"timestamp"`
}

// SMOPolicyResponse represents an SMO policy management response
type SMOPolicyResponse struct {
	PolicyID                string                 `json:"policyId"`
	Status                  string                 `json:"status"`
	ValidationResults       []ValidationResult    `json:"validationResults"`
	DeploymentStatus        string                 `json:"deploymentStatus"`
	AffectedNodes           []string               `json:"affectedNodes"`
	Timestamp               time.Time              `json:"timestamp"`
}

// RAppDeploymentRequest represents an rApp deployment request
type RAppDeploymentRequest struct {
	RAppID                  string                 `json:"rAppId"`
	RAppName                string                 `json:"rAppName"`
	RAppVersion             string                 `json:"rAppVersion"`
	DeploymentConfig        map[string]interface{} `json:"deploymentConfig"`
	ResourceRequirements    ResourceRequirements   `json:"resourceRequirements"`
	TargetClusters          []string               `json:"targetClusters"`
}

// ResourceRequirements defines resource requirements for rApp deployment
type ResourceRequirements struct {
	CPU                     string                 `json:"cpu"`
	Memory                  string                 `json:"memory"`
	Storage                 string                 `json:"storage"`
	NetworkBandwidth        string                 `json:"networkBandwidth"`
	GPURequirement          bool                   `json:"gpuRequirement"`
}

// Nephio R5 Integration Types

// PackageRevisionRequest represents a package revision request to Porch
type PackageRevisionRequest struct {
	PackageName             string                 `json:"packageName"`
	Revision                string                 `json:"revision"`
	Repository              string                 `json:"repository"`
	Workspace               string                 `json:"workspace"`
	PackageContent          map[string]interface{} `json:"packageContent"`
	Lifecycle               string                 `json:"lifecycle"`
	Tasks                   []PackageTask          `json:"tasks"`
}

// PackageTask represents a task to be performed on the package
type PackageTask struct {
	Type                    string                 `json:"type"`
	Function                string                 `json:"function"`
	ConfigMap               map[string]interface{} `json:"configMap"`
}

// OCloudResourceRequest represents an O-Cloud resource provisioning request
type OCloudResourceRequest struct {
	ResourceType            string                 `json:"resourceType"`
	ResourceSpec            ResourceSpec           `json:"resourceSpec"`
	PlacementConstraints    []PlacementConstraint  `json:"placementConstraints"`
	EnergyRequirements      EnergyRequirements     `json:"energyRequirements"`
	SLARequirements         SLARequirements        `json:"slaRequirements"`
}

// ResourceSpec defines the specification for a resource
type ResourceSpec struct {
	ComputeResources        ComputeResources       `json:"computeResources"`
	NetworkResources        NetworkResources       `json:"networkResources"`
	StorageResources        StorageResources       `json:"storageResources"`
	AcceleratorResources    AcceleratorResources   `json:"acceleratorResources"`
}

// PlacementConstraint defines constraints for resource placement
type PlacementConstraint struct {
	Type                    string                 `json:"type"`
	Values                  []string               `json:"values"`
	Weight                  int                    `json:"weight"`
}

// EnergyRequirements defines energy efficiency requirements
type EnergyRequirements struct {
	MaxPowerConsumption     float64                `json:"maxPowerConsumption"`
	EnergyEfficiencyTarget  float64                `json:"energyEfficiencyTarget"`
	CarbonFootprintLimit    float64                `json:"carbonFootprintLimit"`
}

// NewSMONephioIntegrationLayer creates a new integration layer
func NewSMONephioIntegrationLayer(config *IntegrationConfig) *SMONephioIntegrationLayer {
	if config == nil {
		config = getDefaultIntegrationConfig()
	}

	layer := &SMONephioIntegrationLayer{
		config: config,
		stats:  IntegrationStats{},
	}

	// Initialize SMO clients
	layer.smoClient = NewEnhancedSMOClient(config.SMOEndpoint)
	layer.nonRTRICClient = NewNonRTRICClient(config.NonRTRICEndpoint)
	layer.policyManagerClient = NewPolicyManagerClient(config.PolicyManagerEndpoint)
	layer.rAppManagerClient = NewRAppManagerClient(config.RAppManagerEndpoint)

	// Initialize Nephio R5 clients
	layer.porchClient = NewPorchAPIClient(config.PorchAPIEndpoint, config.KubeConfigPath)
	layer.oCloudClient = NewOCloudClient(config.OCloudEndpoint)
	layer.packageManagerClient = NewPackageManagerClient(config.PackageManagerEndpoint)
	layer.resourceProvisionerClient = NewResourceProvisionerClient(config.ResourceProvisionerEndpoint)

	// Initialize performance optimization components
	if config.LoadBalancingEnabled {
		layer.smoLoadBalancer = NewSMOLoadBalancer(config.LoadBalancingStrategy)
		layer.nephioLoadBalancer = NewNephioLoadBalancer(config.LoadBalancingStrategy)
	}

	if config.CacheEnabled {
		layer.requestCache = NewIntegrationCache(config.CacheMaxSize, config.CacheTTL)
	}

	if config.CircuitBreakerEnabled {
		layer.circuitBreaker = NewIntegrationCircuitBreaker(config.FailureThreshold, config.RecoveryTimeout)
	}

	// Initialize monitoring
	layer.integrationMetrics = NewIntegrationMetrics()
	layer.performanceTracker = NewIntegrationPerformanceTracker()
	layer.healthMonitor = NewIntegrationHealthMonitor(config)

	return layer
}

// Start starts the SMO and Nephio integration layer
func (layer *SMONephioIntegrationLayer) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&layer.running, 0, 1) {
		return fmt.Errorf("integration layer already running")
	}

	logrus.WithFields(logrus.Fields{
		"smoEndpoint":      layer.config.SMOEndpoint,
		"porchEndpoint":    layer.config.PorchAPIEndpoint,
		"oCloudEndpoint":   layer.config.OCloudEndpoint,
		"cacheEnabled":     layer.config.CacheEnabled,
		"circuitBreaker":   layer.config.CircuitBreakerEnabled,
		"loadBalancing":    layer.config.LoadBalancingEnabled,
	}).Info("Starting SMO and Nephio R5 Integration Layer")

	// Start SMO connections
	if err := layer.initializeSMOConnections(ctx); err != nil {
		return fmt.Errorf("failed to initialize SMO connections: %w", err)
	}

	// Start Nephio R5 connections
	if err := layer.initializeNephioConnections(ctx); err != nil {
		return fmt.Errorf("failed to initialize Nephio connections: %w", err)
	}

	// Start performance optimization components
	if layer.smoLoadBalancer != nil {
		if err := layer.smoLoadBalancer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start SMO load balancer: %w", err)
		}
	}

	if layer.nephioLoadBalancer != nil {
		if err := layer.nephioLoadBalancer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start Nephio load balancer: %w", err)
		}
	}

	// Start monitoring and health checks
	go layer.healthMonitoringLoop(ctx)
	go layer.performanceMonitoringLoop(ctx)
	go layer.metricsCollectionLoop(ctx)

	logrus.Info("SMO and Nephio R5 Integration Layer started successfully")
	return nil
}

// SMO Integration Methods

// DeploySMOPolicy deploys a policy through SMO integration
func (layer *SMONephioIntegrationLayer) DeploySMOPolicy(ctx context.Context, request *SMOPolicyRequest) (*SMOPolicyResponse, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		atomic.AddUint64(&layer.stats.SMORequestsTotal, 1)
		layer.updateSMOLatency(latency)
	}()

	// Check circuit breaker
	if layer.circuitBreaker != nil && layer.circuitBreaker.IsOpen("smo") {
		atomic.AddUint64(&layer.stats.SMORequestsFailed, 1)
		return nil, fmt.Errorf("SMO circuit breaker is open")
	}

	// Check cache
	if layer.requestCache != nil {
		if cached := layer.requestCache.GetPolicyResponse(request.PolicyID); cached != nil {
			return cached, nil
		}
	}

	// Deploy policy through Non-RT RIC
	response, err := layer.deployPolicyThroughNonRTRIC(ctx, request)
	if err != nil {
		atomic.AddUint64(&layer.stats.SMORequestsFailed, 1)
		if layer.circuitBreaker != nil {
			layer.circuitBreaker.RecordFailure("smo")
		}
		return nil, fmt.Errorf("failed to deploy policy through Non-RT RIC: %w", err)
	}

	// Cache successful response
	if layer.requestCache != nil {
		layer.requestCache.SetPolicyResponse(request.PolicyID, response, layer.config.CacheTTL)
	}

	atomic.AddUint64(&layer.stats.SMORequestsSuccessful, 1)
	atomic.AddUint64(&layer.stats.NonRTRICPolicyUpdates, 1)
	if layer.circuitBreaker != nil {
		layer.circuitBreaker.RecordSuccess("smo")
	}

	return response, nil
}

// DeploySMORApp deploys an rApp through SMO integration
func (layer *SMONephioIntegrationLayer) DeploySMORApp(ctx context.Context, request *RAppDeploymentRequest) (*RAppDeploymentResponse, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		atomic.AddUint64(&layer.stats.SMORequestsTotal, 1)
		layer.updateSMOLatency(latency)
	}()

	// Deploy through rApp Manager
	response, err := layer.rAppManagerClient.DeployRApp(ctx, request)
	if err != nil {
		atomic.AddUint64(&layer.stats.SMORequestsFailed, 1)
		return nil, fmt.Errorf("failed to deploy rApp: %w", err)
	}

	atomic.AddUint64(&layer.stats.SMORequestsSuccessful, 1)
	atomic.AddUint64(&layer.stats.NonRTRICRAppDeployments, 1)

	return response, nil
}

// Nephio R5 Integration Methods

// DeployNephioPackage deploys a package through Porch
func (layer *SMONephioIntegrationLayer) DeployNephioPackage(ctx context.Context, request *PackageRevisionRequest) (*PackageRevisionResponse, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		atomic.AddUint64(&layer.stats.NephioRequestsTotal, 1)
		layer.updateNephioLatency(latency)
	}()

	// Validate package
	if err := layer.porchClient.ValidatePackage(request); err != nil {
		atomic.AddUint64(&layer.stats.NephioRequestsFailed, 1)
		return nil, fmt.Errorf("package validation failed: %w", err)
	}

	// Deploy package through Porch
	response, err := layer.porchClient.DeployPackage(ctx, request)
	if err != nil {
		atomic.AddUint64(&layer.stats.NephioRequestsFailed, 1)
		return nil, fmt.Errorf("failed to deploy package: %w", err)
	}

	atomic.AddUint64(&layer.stats.NephioRequestsSuccessful, 1)
	atomic.AddUint64(&layer.stats.PackageDeployments, 1)

	return response, nil
}

// ProvisionOCloudResources provisions resources through O-Cloud
func (layer *SMONephioIntegrationLayer) ProvisionOCloudResources(ctx context.Context, request *OCloudResourceRequest) (*OCloudResourceResponse, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		atomic.AddUint64(&layer.stats.NephioRequestsTotal, 1)
		layer.updateNephioLatency(latency)
	}()

	// Optimize resource placement for energy efficiency
	optimizedPlacement, err := layer.oCloudClient.OptimizeResourcePlacement(request)
	if err != nil {
		atomic.AddUint64(&layer.stats.NephioRequestsFailed, 1)
		return nil, fmt.Errorf("resource placement optimization failed: %w", err)
	}

	// Provision resources
	response, err := layer.oCloudClient.ProvisionResources(ctx, optimizedPlacement)
	if err != nil {
		atomic.AddUint64(&layer.stats.NephioRequestsFailed, 1)
		return nil, fmt.Errorf("failed to provision O-Cloud resources: %w", err)
	}

	atomic.AddUint64(&layer.stats.NephioRequestsSuccessful, 1)
	atomic.AddUint64(&layer.stats.ResourceAllocations, 1)
	
	// Update energy efficiency score
	layer.updateEnergyEfficiencyScore(response.EnergyMetrics)

	return response, nil
}

// Performance optimization methods

func (layer *SMONephioIntegrationLayer) OptimizeForSMOIntegration() error {
	// Optimize SMO client connections
	if err := layer.smoClient.OptimizeConnections(); err != nil {
		return fmt.Errorf("failed to optimize SMO connections: %w", err)
	}

	// Optimize load balancing
	if layer.smoLoadBalancer != nil {
		layer.smoLoadBalancer.OptimizeForLatency()
	}

	// Optimize caching
	if layer.requestCache != nil {
		layer.requestCache.OptimizeForHitRatio()
	}

	return nil
}

func (layer *SMONephioIntegrationLayer) OptimizeForNephioR5Integration() error {
	// Optimize Porch client performance
	if err := layer.porchClient.OptimizeForThroughput(); err != nil {
		return fmt.Errorf("failed to optimize Porch client: %w", err)
	}

	// Optimize O-Cloud resource management
	if err := layer.oCloudClient.OptimizeForEnergyEfficiency(); err != nil {
		return fmt.Errorf("failed to optimize O-Cloud client: %w", err)
	}

	return nil
}

// Monitoring and health check methods

func (layer *SMONephioIntegrationLayer) healthMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(layer.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			layer.performHealthChecks()
		}
	}
}

func (layer *SMONephioIntegrationLayer) performHealthChecks() {
	// Check SMO connection health
	smoHealthy := layer.smoClient.CheckHealth()
	if smoHealthy {
		atomic.StoreInt32(&layer.smoConnectionStatus, 1)
		layer.stats.SMOConnectionStatus = "healthy"
	} else {
		atomic.StoreInt32(&layer.smoConnectionStatus, 0)
		layer.stats.SMOConnectionStatus = "unhealthy"
	}

	// Check Nephio R5 connection health
	nephioHealthy := layer.porchClient.CheckHealth() && layer.oCloudClient.CheckHealth()
	if nephioHealthy {
		atomic.StoreInt32(&layer.nephioConnectionStatus, 1)
		layer.stats.NephioConnectionStatus = "healthy"
	} else {
		atomic.StoreInt32(&layer.nephioConnectionStatus, 0)
		layer.stats.NephioConnectionStatus = "unhealthy"
	}
}

func (layer *SMONephioIntegrationLayer) performanceMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			layer.updatePerformanceMetrics()
		}
	}
}

func (layer *SMONephioIntegrationLayer) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			layer.collectDetailedMetrics()
		}
	}
}

// Utility methods

func (layer *SMONephioIntegrationLayer) updateSMOLatency(latency time.Duration) {
	latencyMs := float64(latency.Nanoseconds()) / 1e6
	currentAvg := layer.stats.SMOAverageLatencyMs
	if currentAvg == 0 {
		layer.stats.SMOAverageLatencyMs = latencyMs
	} else {
		layer.stats.SMOAverageLatencyMs = (currentAvg*0.9 + latencyMs*0.1)
	}
}

func (layer *SMONephioIntegrationLayer) updateNephioLatency(latency time.Duration) {
	latencyMs := float64(latency.Nanoseconds()) / 1e6
	currentAvg := layer.stats.NephioAverageLatencyMs
	if currentAvg == 0 {
		layer.stats.NephioAverageLatencyMs = latencyMs
	} else {
		layer.stats.NephioAverageLatencyMs = (currentAvg*0.9 + latencyMs*0.1)
	}
}

func (layer *SMONephioIntegrationLayer) updateEnergyEfficiencyScore(energyMetrics *EnergyMetrics) {
	if energyMetrics != nil {
		layer.stats.EnergyEfficiencyScore = energyMetrics.EfficiencyScore
	}
}

// GetIntegrationStats returns current integration statistics
func (layer *SMONephioIntegrationLayer) GetIntegrationStats() IntegrationStats {
	layer.mu.RLock()
	defer layer.mu.RUnlock()

	layer.stats.LastUpdated = time.Now()
	
	// Update cache hit ratio
	if layer.requestCache != nil {
		layer.stats.CacheHitRatio = layer.requestCache.GetHitRatio()
	}

	// Update circuit breaker statistics
	if layer.circuitBreaker != nil {
		layer.stats.CircuitBreakerTrips = layer.circuitBreaker.GetTripCount()
	}

	return layer.stats
}

func getDefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		SMOEndpoint:             "http://localhost:8080",
		NonRTRICEndpoint:        "http://localhost:8081",
		PorchAPIEndpoint:        "http://localhost:8082",
		OCloudEndpoint:          "http://localhost:8083",
		
		MaxConcurrentRequests:   1000,
		RequestTimeout:          time.Second * 30,
		RetryAttempts:           3,
		RetryBackoff:            time.Second,
		
		CacheEnabled:            true,
		CacheTTL:                time.Minute * 5,
		CacheMaxSize:            10000,
		
		CircuitBreakerEnabled:   true,
		FailureThreshold:        10,
		RecoveryTimeout:         time.Second * 30,
		
		HealthCheckInterval:     time.Second * 30,
		HealthCheckTimeout:      time.Second * 5,
		
		LoadBalancingEnabled:    true,
		LoadBalancingStrategy:   "weighted_round_robin",
	}
}

// Stop gracefully stops the integration layer
func (layer *SMONephioIntegrationLayer) Stop() error {
	if !atomic.CompareAndSwapInt32(&layer.running, 1, 0) {
		return fmt.Errorf("integration layer not running")
	}

	logrus.Info("Stopping SMO and Nephio R5 Integration Layer")

	// Stop clients and components
	if layer.smoClient != nil {
		layer.smoClient.Close()
	}

	if layer.porchClient != nil {
		layer.porchClient.Close()
	}

	if layer.oCloudClient != nil {
		layer.oCloudClient.Close()
	}

	logrus.Info("SMO and Nephio R5 Integration Layer stopped successfully")
	return nil
}