// Package dashboard provides centralized type definitions for the O-RAN Near-RT RIC Dashboard
// This file contains all shared types to avoid redeclaration errors
package dashboard

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Core E2 Types - centralized definitions to avoid redeclarations

// E2Node represents an E2 Node in the O-RAN architecture
type E2Node struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Address           string              `json:"address"`
	Port              int                 `json:"port"`
	Type              string              `json:"type"`
	Status            string              `json:"status"`
	ConnectionStatus  E2NodeConnectionStatus `json:"connectionStatus"`
	LastSeen          time.Time           `json:"lastSeen"`
	Version           string              `json:"version"`
	SupportedRANFunctions []RANFunction   `json:"supportedRanFunctions"`
	GlobalRICID       GlobalRICID         `json:"globalRicId"`
	ConfigurationUpdate E2NodeConfigurationUpdate `json:"configurationUpdate,omitempty"`
}

// GlobalRICID represents the Global RIC Identifier
type GlobalRICID struct {
	PLMNIdentity []byte `json:"plmnIdentity"`
	RICId        []byte `json:"ricId"`
}

// E2APMessage represents an E2AP protocol message
type E2APMessage struct {
	MessageType    E2APMessageType `json:"messageType"`
	TransactionID  uint32          `json:"transactionId"`
	Payload        []byte          `json:"payload"`
	Timestamp      time.Time       `json:"timestamp"`
	Source         string          `json:"source"`
	Destination    string          `json:"destination"`
}

// E2APMessageType represents the type of E2AP message
type E2APMessageType uint32

const (
	E2APMessageTypeSetupRequest E2APMessageType = iota + 1
	E2APMessageTypeSetupResponse
	E2APMessageTypeSetupFailure
	E2APMessageTypeConfigurationUpdate
	E2APMessageTypeConfigurationUpdateAck
	E2APMessageTypeConfigurationUpdateFailure
)

// E2NodeComponentConfigUpdateAck represents the acknowledgment for configuration updates
type E2NodeComponentConfigUpdateAck struct {
	ComponentID    E2NodeComponentID `json:"componentId"`
	ConfigAck      ConfigAckType     `json:"configAck"`
	UpdateOutcome  string           `json:"updateOutcome"`
}

// ConfigAckType represents the type of configuration acknowledgment
type ConfigAckType uint32

const (
	ConfigAckSuccess ConfigAckType = iota
	ConfigAckFailure
)

// Performance and Monitoring Types

// SIMDOperation represents SIMD (Single Instruction, Multiple Data) operations for performance
type SIMDOperation struct {
	Operation string        `json:"operation"`
	DataSize  int          `json:"dataSize"`
	Execution time.Duration `json:"execution"`
	Result    []float64    `json:"result"`
}

// ResourceUsage represents system resource utilization metrics
type ResourceUsage struct {
	CPU        float64   `json:"cpu"`
	Memory     uint64    `json:"memory"`
	Network    uint64    `json:"network"`
	Disk       uint64    `json:"disk"`
	Timestamp  time.Time `json:"timestamp"`
	NodeID     string    `json:"nodeId"`
}

// LatencyTracker tracks latency metrics across different operations
type LatencyTracker struct {
	Operation     string        `json:"operation"`
	StartTime     time.Time     `json:"startTime"`
	EndTime       time.Time     `json:"endTime"`
	Duration      time.Duration `json:"duration"`
	TargetLatency time.Duration `json:"targetLatency"`
	Success       bool          `json:"success"`
}

// Load Balancing Types

// RoundRobin represents the round-robin load balancing algorithm
type RoundRobin struct {
	Servers []string `json:"servers"`
	Current int      `json:"current"`
	Mutex   sync.Mutex `json:"-"`
}

// HealthChecker monitors the health of system components
type HealthChecker struct {
	ComponentID   string            `json:"componentId"`
	Status        HealthStatus      `json:"status"`
	LastCheck     time.Time         `json:"lastCheck"`
	CheckInterval time.Duration     `json:"checkInterval"`
	Metrics       map[string]interface{} `json:"metrics"`
}

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// CircuitBreaker implements the circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	mu                sync.RWMutex
	name              string
	state             CircuitState
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	lastSuccessTime   time.Time
	nextAttempt       time.Time
	
	// Configuration
	maxFailures       int
	timeout           time.Duration
	resetTimeout      time.Duration
	halfOpenMaxCalls  int
	
	// Callbacks
	onStateChange     func(name string, from, to CircuitState)
	
	// Statistics
	totalCalls        int64
	totalFailures     int64
	totalSuccesses    int64
	totalTimeouts     int64
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateHalfOpen
	StateOpen
)

// Additional Supporting Types

// E2NodeConnectionStatus represents the connection status of an E2 Node
type E2NodeConnectionStatus string

const (
	E2NodeConnectionStatusConnected    E2NodeConnectionStatus = "connected"
	E2NodeConnectionStatusDisconnected E2NodeConnectionStatus = "disconnected"
	E2NodeConnectionStatusConnecting   E2NodeConnectionStatus = "connecting"
)

// RANFunction represents a RAN function supported by an E2 Node
type RANFunction struct {
	ID          uint32 `json:"id"`
	Revision    uint32 `json:"revision"`
	OID         string `json:"oid"`
	Description string `json:"description"`
}

// E2NodeConfigurationUpdate represents configuration update information
type E2NodeConfigurationUpdate struct {
	UpdateID    string    `json:"updateId"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requestedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// E2NodeComponentID represents the identifier for an E2 Node component
type E2NodeComponentID struct {
	Type        E2NodeComponentType `json:"type"`
	Identifier  string             `json:"identifier"`
}

// E2NodeComponentType represents the type of E2 Node component
type E2NodeComponentType string

const (
	E2NodeComponentTypeNG E2NodeComponentType = "ng"
	E2NodeComponentTypeXn E2NodeComponentType = "xn"
	E2NodeComponentTypeE1 E2NodeComponentType = "e1"
	E2NodeComponentTypeF1 E2NodeComponentType = "f1"
	E2NodeComponentTypeW1 E2NodeComponentType = "w1"
	E2NodeComponentTypeS1 E2NodeComponentType = "s1"
	E2NodeComponentTypeX2 E2NodeComponentType = "x2"
)

// Context wrapper for operations
type OperationContext struct {
	Context context.Context
	Cancel  context.CancelFunc
	Timeout time.Duration
}

// Simulation and Testing Types

// E2NodeSimulator represents a simulated E2 node for testing and development
// Using the more complete definition from e2_node_simulator.go
type E2NodeSimulator struct {
	mu                sync.RWMutex
	nodeID            string
	globalE2NodeID    GlobalE2NodeID
	ricAddress        string
	ricPort           uint32
	localAddress      string
	localPort         uint32
	
	// SCTP connection
	conn              interface{} // *sctp.SCTPConn - using interface{} to avoid import issues
	isConnected       bool
	
	// E2AP protocol handler
	protocolHandler   interface{} // *E2APProcedureHandler
	encoder           interface{} // *E2APEncoder
	
	// Simulation state
	isRunning         bool
	ctx               context.Context
	cancel            context.CancelFunc
	
	// RAN Functions
	ranFunctions      []RANFunction
	serviceModels     []ServiceModel
	
	// Subscription management
	subscriptions     map[string]*SimulatedSubscription
	
	// Load testing fields from load_testing.go
	ID               string
	ConnectionTime   time.Time
	Subscriptions_   []string // renamed to avoid conflict
	IndicationCount  int64
	LastActivity     time.Time
	ErrorCount       int64
}

// GlobalE2NodeID represents the global E2 node identifier
type GlobalE2NodeID struct {
	PLMNIdentity []byte `json:"plmnIdentity"`
	E2NodeID     []byte `json:"e2NodeId"`
}

// ServiceModel represents an E2 service model
type ServiceModel struct {
	OID         string `json:"oid"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// SimulatedSubscription represents a subscription in the simulator
type SimulatedSubscription struct {
	SubscriptionID   string            `json:"subscriptionId"`
	ServiceModelOID  string            `json:"serviceModelOid"`
	E2NodeID         string            `json:"e2NodeId"`
	Actions          []SubscriptionAction `json:"actions"`
	ReportingPeriod  time.Duration     `json:"reportingPeriod"`
	IsActive         bool              `json:"isActive"`
	CreatedAt        time.Time         `json:"createdAt"`
}

// SubscriptionAction represents an action within a subscription
type SubscriptionAction struct {
	ActionID     uint32      `json:"actionId"`
	ActionType   ActionType  `json:"actionType"`
	Definition   []byte      `json:"definition"`
	SubsequentAction *SubscriptionAction `json:"subsequentAction,omitempty"`
}

// ActionType represents the type of subscription action
type ActionType uint32

const (
	ActionTypeReport ActionType = iota
	ActionTypeInsert
	ActionTypePolicy
)

// Worker Performance Types

// Worker represents a high-performance worker thread
// Using the more complete definition from enhanced_connection_pool_cluster.go
type Worker struct {
	id              int
	poolID          string
	thread          *WorkerThread
	
	// CPU affinity
	cpuCore         int
	threadID        int
	
	// Performance optimization
	cache           *WorkerCache
	
	// From performance_optimizer.go fields
	coreID_         int // renamed to avoid conflict  
	workChan        chan WorkItem
	quit            chan bool
	stats           WorkerStats
}

// WorkerThread represents a worker thread with real-time scheduling
type WorkerThread struct {
	tid             int
	priority        int
	cpuAffinity     uint64
	stackSize       int
	
	// Real-time scheduling
	policy          SchedulingPolicy
	rtPriority      int
	
	context         *ThreadContext
}

// WorkerCache represents worker-specific cache
type WorkerCache struct {
	size     int
	data     map[string]interface{}
	hitCount int64
	mu       sync.RWMutex
}

// SchedulingPolicy represents thread scheduling policy
type SchedulingPolicy int

const (
	SchedulingPolicyNormal SchedulingPolicy = iota
	SchedulingPolicyFIFO
	SchedulingPolicyRR
)

// ThreadContext represents thread execution context
type ThreadContext struct {
	startTime time.Time
	cpuTime   time.Duration
	userTime  time.Duration
	systTime  time.Duration
}

// WorkItem represents work to be processed
type WorkItem struct {
	ID        uint64
	Type      WorkType
	Data      interface{} // Changed from unsafe.Pointer to interface{} for safety
	Size      int
	Priority  Priority
	Timestamp time.Time
	Callback  func(result WorkResult)
}

// WorkResult contains processing results
type WorkResult struct {
	ID        uint64
	Success   bool
	Data      interface{} // Changed from unsafe.Pointer to interface{} for safety
	Size      int
	Duration  time.Duration
	Error     error
}

// WorkType defines the type of work
type WorkType int

const (
	WorkTypeE2APMessage WorkType = iota
	WorkTypeSubscription
	WorkTypeIndication
	WorkTypeControl
	WorkTypePolicyUpdate
)

// Priority defines work priority levels - consolidated from multiple files
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// WorkerStats tracks worker performance
// Using the more complete definition from high_performance_processor.go
type WorkerStats struct {
	MessagesProcessed   uint64
	ProcessingTime      time.Duration
	IdleTime            time.Duration
	ErrorCount          uint64
	LastActivity        time.Time
	CPUTime             time.Duration
	
	// From performance_optimizer.go additional fields
	ProcessedItems uint64 // alias for MessagesProcessed
	TotalDuration  time.Duration // alias for ProcessingTime
}

// MemoryStats tracks memory usage
// Using the more complete definition from high_performance_processor.go
type MemoryStats struct {
	TotalAllocated      uint64
	ByNUMANode          map[int]uint64
	HitRatio            float64
	FragmentationRatio  float64
	GCPressure          float64
	
	// From performance_optimizer.go additional fields
	AllocatedBytes   uint64 // alias for TotalAllocated
	PoolHits         uint64
	PoolMisses       uint64
	GCPauses         uint64
	LastGCDuration   time.Duration
}

// Testing and Validation Types

// TestSeverity indicates the importance of a test - consolidated
type TestSeverity string

const (
	SeverityCritical TestSeverity = "critical"
	SeverityHigh     TestSeverity = "high"
	SeverityMedium   TestSeverity = "medium"
	SeverityLow      TestSeverity = "low"
)

// TestPriority defines test priorities
type TestPriority string

const (
	TestPriorityCritical TestPriority = "critical"
	TestPriorityHigh     TestPriority = "high"
	TestPriorityMedium   TestPriority = "medium"
	TestPriorityLow      TestPriority = "low"
)

// ValidationResult represents the result of a validation
// Using the more complete definition from performance_test_runner.go
type ValidationResult struct {
	TestName        string    `json:"testName"`
	RequirementMet  bool      `json:"requirementMet"`
	ActualValue     float64   `json:"actualValue"`
	RequiredValue   float64   `json:"requiredValue"`
	PerformanceGap  float64   `json:"performanceGap"` // Positive means exceeds requirement
	Details         string    `json:"details"`
	Timestamp       time.Time `json:"timestamp"`
	
	// From e2e_testing_suite.go additional fields
	Rule     string      `json:"rule"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
	Status   string      `json:"status"`
	Message  string      `json:"message"`
}

// LatencyMetrics represents latency measurements
// Using the more complete definition from comprehensive_performance_monitor.go
type LatencyMetrics struct {
	Mean                    float64       `json:"mean"`
	Median                  float64       `json:"median"`
	P50                     float64       `json:"p50"`
	P95                     float64       `json:"p95"`
	P99                     float64       `json:"p99"`
	P999                    float64       `json:"p999"`
	Min                     float64       `json:"min"`
	Max                     float64       `json:"max"`
	StandardDeviation       float64       `json:"standardDeviation"`
	SampleCount             int64         `json:"sampleCount"`
	Count                   int64         `json:"count"` // alias for SampleCount
}

// Additional Consolidated Types

// TestSummary represents a summary of test results - consolidated from multiple files
type TestSummary struct {
	// From compliance_testing.go
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration time.Duration `json:"duration"`
	Coverage float64       `json:"coverage"`
	
	// From performance_testing.go additional fields
	TotalTestDuration   time.Duration `json:"totalTestDuration"` // alias for Duration
	TestsExecuted       int           `json:"testsExecuted"`     // alias for Total
	TestsPassed         int           `json:"testsPassed"`       // alias for Passed
	TestsFailed         int           `json:"testsFailed"`       // alias for Failed
	OverallScore        float64       `json:"overallScore"`
	Recommendations     []string      `json:"recommendations"`
	CriticalIssues      []string      `json:"criticalIssues"`
	PerformanceGrade    string        `json:"performanceGrade"`
}

// HorizontalScaler manages horizontal scaling of components
type HorizontalScaler struct {
	mu                  sync.RWMutex
	metrics             *HorizontalScalingMetrics
	config              *HorizontalScalingConfig
	decisions           []ScalingDecision
	lastScalingAction   time.Time
	cooldownPeriod      time.Duration
	scalingHistory      []ScalingEvent
	predictiveModel     *PredictiveScalingModel
}

// HorizontalScalingMetrics tracks scaling metrics
type HorizontalScalingMetrics struct {
	CurrentReplicas     int32     `json:"currentReplicas"`
	TargetReplicas      int32     `json:"targetReplicas"`
	CPUUtilization      float64   `json:"cpuUtilization"`
	MemoryUtilization   float64   `json:"memoryUtilization"`
	RequestRate         float64   `json:"requestRate"`
	ResponseTime        float64   `json:"responseTime"`
	LastScalingEvent    time.Time `json:"lastScalingEvent"`
}

// HorizontalScalingConfig defines scaling configuration
type HorizontalScalingConfig struct {
	MinReplicas             int32   `json:"minReplicas"`
	MaxReplicas             int32   `json:"maxReplicas"`
	TargetCPUUtilization    float64 `json:"targetCPUUtilization"`
	TargetMemoryUtilization float64 `json:"targetMemoryUtilization"`
	ScaleUpCooldown         time.Duration `json:"scaleUpCooldown"`
	ScaleDownCooldown       time.Duration `json:"scaleDownCooldown"`
	EnablePredictiveScaling bool    `json:"enablePredictiveScaling"`
}

// ScalingDecision represents a scaling decision
type ScalingDecision struct {
	Timestamp       time.Time         `json:"timestamp"`
	Action          ScalingAction     `json:"action"`
	Reason          string           `json:"reason"`
	MetricsSnapshot interface{}      `json:"metricsSnapshot"`
	Confidence      float64          `json:"confidence"`
}

// ScalingAction represents the type of scaling action
type ScalingAction string

const (
	ScalingActionUp   ScalingAction = "scale_up"
	ScalingActionDown ScalingAction = "scale_down"
	ScalingActionNone ScalingAction = "none"
)

// ScalingEvent represents a scaling event
type ScalingEvent struct {
	Timestamp     time.Time     `json:"timestamp"`
	Action        ScalingAction `json:"action"`
	FromReplicas  int32         `json:"fromReplicas"`
	ToReplicas    int32         `json:"toReplicas"`
	Reason        string        `json:"reason"`
	Success       bool          `json:"success"`
	Duration      time.Duration `json:"duration"`
}

// PredictiveScalingModel represents a model for predictive scaling
type PredictiveScalingModel struct {
	ModelType       string                 `json:"modelType"`
	Parameters      map[string]interface{} `json:"parameters"`
	Accuracy        float64               `json:"accuracy"`
	LastTraining    time.Time             `json:"lastTraining"`
	PredictionHorizon time.Duration       `json:"predictionHorizon"`
}

// LogEntry represents a structured log entry - consolidated
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component"`
	RequestID   string                 `json:"requestId,omitempty"`
	UserID      string                 `json:"userId,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	
	// Additional fields from different log implementations
	Logger      string                 `json:"logger,omitempty"`
	Thread      string                 `json:"thread,omitempty"`
	Function    string                 `json:"function,omitempty"`
	Line        int                    `json:"line,omitempty"`
	Stack       string                 `json:"stack,omitempty"`
}

// RouteEntry represents a routing entry - consolidated
type RouteEntry struct {
	ID           string            `json:"id"`
	Path         string            `json:"path"`
	Method       string            `json:"method"`
	Handler      string            `json:"handler"`
	Middleware   []string          `json:"middleware"`
	Metadata     map[string]string `json:"metadata"`
	
	// Performance optimization fields
	Priority     int               `json:"priority"`
	LoadBalancer *LoadBalancerConfig `json:"loadBalancer,omitempty"`
	RateLimit    *RateLimitConfig  `json:"rateLimit,omitempty"`
	Cache        *CacheConfig      `json:"cache,omitempty"`
}

// LoadBalancerConfig defines load balancing configuration
type LoadBalancerConfig struct {
	Strategy    string   `json:"strategy"` // round_robin, least_connections, etc.
	Targets     []string `json:"targets"`
	HealthCheck bool     `json:"healthCheck"`
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int           `json:"requestsPerSecond"`
	BurstSize         int           `json:"burstSize"`
	Window            time.Duration `json:"window"`
}

// CacheConfig defines caching configuration
type CacheConfig struct {
	Enabled   bool          `json:"enabled"`
	TTL       time.Duration `json:"ttl"`
	MaxSize   int           `json:"maxSize"`
	Strategy  string        `json:"strategy"` // LRU, LFU, etc.
}

// ServiceModelAPI defines the API for service models - consolidated
type ServiceModelAPI struct {
	mu               sync.RWMutex
	registry         *ServiceModelRegistry
	clients          map[string]*ServiceModelClient
	subscriptions    map[string]*ServiceModelSubscription
	eventHandlers    map[ServiceModelEventType][]ServiceModelEventHandler
	config           *ServiceModelAPIConfig
}

// ServiceModelRegistry manages service model registration
type ServiceModelRegistry struct {
	models           map[string]*ServiceModel
	supportedTypes   []ServiceModelType
	mu               sync.RWMutex
}

// ServiceModelClient represents a client for a service model
type ServiceModelClient struct {
	ModelOID         string
	Version          string
	Connection       interface{}
	LastActivity     time.Time
	IsActive         bool
}

// ServiceModelSubscription represents a subscription to a service model
type ServiceModelSubscription struct {
	ID               string
	ModelOID         string
	SubscriberID     string
	EventTypes       []ServiceModelEventType
	CreatedAt        time.Time
	IsActive         bool
}

// ServiceModelEventType represents types of service model events
type ServiceModelEventType string

const (
	ServiceModelEventRegistered   ServiceModelEventType = "registered"
	ServiceModelEventUnregistered ServiceModelEventType = "unregistered"
	ServiceModelEventUpdated      ServiceModelEventType = "updated"
	ServiceModelEventError        ServiceModelEventType = "error"
)

// ServiceModelEventHandler handles service model events
type ServiceModelEventHandler func(event ServiceModelEvent)

// ServiceModelEvent represents a service model event
type ServiceModelEvent struct {
	Type        ServiceModelEventType `json:"type"`
	ModelOID    string               `json:"modelOid"`
	Timestamp   time.Time            `json:"timestamp"`
	Data        interface{}          `json:"data"`
	Error       error                `json:"error,omitempty"`
}

// ServiceModelAPIConfig configures the service model API
type ServiceModelAPIConfig struct {
	RegistryEnabled     bool          `json:"registryEnabled"`
	AutoDiscovery       bool          `json:"autoDiscovery"`
	EventBufferSize     int           `json:"eventBufferSize"`
	ClientTimeout       time.Duration `json:"clientTimeout"`
	HeartbeatInterval   time.Duration `json:"heartbeatInterval"`
}

// ServiceModelType represents the type of service model - consolidated
type ServiceModelType string

const (
	ServiceModelTypeKPM ServiceModelType = "kpm"      // Key Performance Measurement
	ServiceModelTypeRC  ServiceModelType = "rc"       // RAN Control
	ServiceModelTypeNI  ServiceModelType = "ni"       // Network Intelligence
	ServiceModelTypeTS  ServiceModelType = "ts"       // Traffic Steering
	ServiceModelTypeQoE ServiceModelType = "qoe"      // Quality of Experience
	ServiceModelTypeSON ServiceModelType = "son"      // Self-Organizing Networks
)

// ResponseWriterWrapper wraps http.ResponseWriter to avoid conflicts
type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// WriteHeader sets the status code
func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write writes the response data
func (rw *ResponseWriterWrapper) Write(data []byte) (int, error) {
	rw.size += len(data)
	return rw.ResponseWriter.Write(data)
}

// E2 Service Model Types - consolidated to avoid redeclaration

// E2SMKPMMetrics represents KPM service model metrics
type E2SMKPMMetrics struct {
	E2NodeID            string                 `json:"e2NodeId"`
	MeasurementData     []MeasurementData      `json:"measurementData"`
	GranularityPeriod   int64                 `json:"granularityPeriod"`
	Timestamp           time.Time             `json:"timestamp"`
	SubscriptionID      string                `json:"subscriptionId"`
}

// MeasurementData represents measurement data point
type MeasurementData struct {
	MeasurementID       uint32                `json:"measurementId"`
	MeasurementValue    interface{}           `json:"measurementValue"`
	MeasurementType     string               `json:"measurementType"`
	Labels              map[string]string    `json:"labels"`
}

// E2SMKPMIndicationHeader represents KPM indication header
type E2SMKPMIndicationHeader struct {
	IndicationHeaderFormat  int32             `json:"indicationHeaderFormat"`
	CollectionStartTime     time.Time         `json:"collectionStartTime"`
	FileFormatVersion       string            `json:"fileFormatVersion"`
	SenderName              string            `json:"senderName"`
	SenderType              string            `json:"senderType"`
	VendorName              string            `json:"vendorName"`
}

// E2SMKPMIndicationMessage represents KMP indication message
type E2SMKPMIndicationMessage struct {
	IndicationMessageFormat int32             `json:"indicationMessageFormat"`
	MeasurementData         []MeasurementData `json:"measurementData"`
	GranularityPeriod       int64            `json:"granularityPeriod"`
	MeasurementInfoList     []MeasurementInfo `json:"measurementInfoList"`
}

// MeasurementInfo represents measurement information
type MeasurementInfo struct {
	MeasurementName         string            `json:"measurementName"`
	MeasurementID           uint32            `json:"measurementId"`
	MeasurementDescription  string            `json:"measurementDescription"`
	Units                   string            `json:"units"`
}

// E2SMRCControlHeader represents RC control header
type E2SMRCControlHeader struct {
	ControlHeaderFormat     int32             `json:"controlHeaderFormat"`
	UEId                    string            `json:"ueId"`
	RANFunctionID           uint32            `json:"ranFunctionId"`
	RICControlMessageType   int32             `json:"ricControlMessageType"`
	RICControlAckRequest    bool              `json:"ricControlAckRequest"`
}

// E2SMRCControlMessage represents RC control message
type E2SMRCControlMessage struct {
	ControlMessageFormat    int32             `json:"controlMessageFormat"`
	RANParameters           []RANParameter    `json:"ranParameters"`
	ControlAction           string            `json:"controlAction"`
	ControlOutcome          string            `json:"controlOutcome"`
	CallProcessID           uint32            `json:"callProcessId"`
}

// RANParameter represents a RAN parameter
type RANParameter struct {
	ParameterID             uint32            `json:"parameterId"`
	ParameterName           string            `json:"parameterName"`
	ParameterValue          interface{}       `json:"parameterValue"`
	ParameterType           string            `json:"parameterType"`
}

// E2SMNIIndicationHeader represents NI indication header
type E2SMNIIndicationHeader struct {
	IndicationHeaderFormat  int32             `json:"indicationHeaderFormat"`
	InterfaceType           string            `json:"interfaceType"`
	InterfaceDirection      string            `json:"interfaceDirection"`
	Timestamp               time.Time         `json:"timestamp"`
	EventType               string            `json:"eventType"`
}

// E2SMNIIndicationMessage represents NI indication message
type E2SMNIIndicationMessage struct {
	IndicationMessageFormat int32             `json:"indicationMessageFormat"`
	InterfaceMessage        []byte            `json:"interfaceMessage"`
	EventTriggerDefinition  []byte            `json:"eventTriggerDefinition"`
	ActionDefinition        []byte            `json:"actionDefinition"`
}