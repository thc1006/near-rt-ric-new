// Package dashboard provides centralized type definitions for the O-RAN Near-RT RIC Dashboard
// This file contains all shared types to avoid redeclaration errors
package dashboard

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
	"unsafe"
	
	"github.com/sirupsen/logrus"
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
	HealthStatusUnknown   HealthStatus = "unknown"
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

// ServiceModelRegistry manages service model registration

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

// Service Model Management Types - centralized to avoid redeclarations

// ProtocolIE represents a protocol information element
type ProtocolIE struct {
	ID          uint32      `json:"id"`
	Criticality string      `json:"criticality"`
	Value       interface{} `json:"value"`
	TypeName    string      `json:"typeName"`
}

// ServiceModelDefinition represents a complete service model definition
type ServiceModelDefinition struct {
	OID           string                    `json:"oid"`
	Name          string                    `json:"name"`
	Type          ServiceModelType          `json:"type"`
	Version       string                    `json:"version"`
	Description   string                    `json:"description"`
	Capabilities  []ServiceModelCapability  `json:"capabilities"`
	RANFunctions  []RANFunction            `json:"ranFunctions"`
	LastUpdated   time.Time                `json:"lastUpdated"`
}

// ServiceModelCapability represents a capability of a service model
type ServiceModelCapability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Supported   bool   `json:"supported"`
}

// ServiceModelCapabilities represents the capabilities of a service model
type ServiceModelCapabilities struct {
	ServiceModelType     ServiceModelType `json:"serviceModelType"`
	Version              string           `json:"version"`
	SupportedOperations  []string         `json:"supportedOperations"`
	SupportedMessageTypes []string        `json:"supportedMessageTypes"`
	SupportsIndications  bool             `json:"supportsIndications"`
	SupportsControl      bool             `json:"supportsControl"`
	MaxConcurrentOps     int              `json:"maxConcurrentOps"`
	LastUpdated          time.Time        `json:"lastUpdated"`
}

// ServiceModelStatistics represents statistics for a service model
type ServiceModelStatistics struct {
	ServiceModelType      ServiceModelType `json:"serviceModelType"`
	IndicationsProcessed  uint64           `json:"indicationsProcessed"`
	ControlsProcessed     uint64           `json:"controlsProcessed"`
	ValidationErrors      uint64           `json:"validationErrors"`
	ProcessingErrors      uint64           `json:"processingErrors"`
	AverageProcessingTime time.Duration    `json:"averageProcessingTime"`
	LastProcessedAt       time.Time        `json:"lastProcessedAt"`
	TotalProcessingTime   time.Duration    `json:"totalProcessingTime"`
}

// ServiceModelAPI interface for service model implementations
type ServiceModelAPI interface {
	GetServiceModelType() ServiceModelType
	GetSupportedOperations() []string
	ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error)
	ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error)
	ValidateMessage(messageType string, data []byte) error
}

// ServiceModelRegistry manages service model definitions and capabilities
// This is the centralized definition to avoid redeclarations
type ServiceModelRegistry struct {
	// From service_models.go
	models map[string]*ServiceModelDefinition
	
	// From service_model_registry.go
	mu           sync.RWMutex
	serviceModels map[ServiceModelType]ServiceModelAPI
	capabilities  map[ServiceModelType]*ServiceModelCapabilities
	statistics    map[ServiceModelType]*ServiceModelStatistics
	supportedTypes   []ServiceModelType
}

// NewServiceModelRegistry creates a new service model registry
// This is the centralized constructor to avoid redeclarations
func NewServiceModelRegistry() *ServiceModelRegistry {
	return &ServiceModelRegistry{
		models: make(map[string]*ServiceModelDefinition),
		serviceModels: make(map[ServiceModelType]ServiceModelAPI),
		capabilities:  make(map[ServiceModelType]*ServiceModelCapabilities),
		statistics:    make(map[ServiceModelType]*ServiceModelStatistics),
		supportedTypes: []ServiceModelType{ServiceModelTypeKPM, ServiceModelTypeRC, ServiceModelTypeNI, ServiceModelTypeTS, ServiceModelTypeQoE, ServiceModelTypeSON},
	}
}

// Policy Management Types - centralized to avoid redeclarations

// PolicyTypeID represents a policy type identifier
type PolicyTypeID string

// PolicyInstanceID represents a policy instance identifier
type PolicyInstanceID string

// PolicyType represents an A1 policy type
// This is the centralized definition to avoid conflicts between smo_components.go and a1_models.go
type PolicyType struct {
	// From a1_models.go - standard A1 policy type
	ID          PolicyTypeID    `json:"policy_type_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
	CreatedAt   time.Time       `json:"created_at"`
	
	// From smo_components.go - additional fields for SMO integration
	PolicyTypeID  string      `json:"policyTypeId"`
	Schema2       interface{} `json:"policy_type_schema,omitempty"`
}

// PolicyStatus represents policy status
// This is the centralized definition to avoid conflicts between smo_components.go and a1_models.go
type PolicyStatus struct {
	// From a1_models.go - standard policy status
	Status     string    `json:"status"`
	Reason     string    `json:"reason,omitempty"`
	LastUpdate time.Time `json:"last_update"`
}

// Policy status constants - centralized
const (
	PolicyStatusActive   = "ACTIVE"
	PolicyStatusInactive = "INACTIVE"
	PolicyStatusError    = "ERROR"
	PolicyStatusDeleted  = "DELETED"
)

// PolicyManager manages policy lifecycle, validation, conflicts, and distribution
// This is the centralized definition to avoid conflicts between smo_components.go and policy_manager.go
type PolicyManager struct {
	// From policy_manager.go - more complete implementation
	a1Client           A1MediatorClient
	xappClients        map[string]*XAppClient
	policyTypes        map[PolicyTypeID]*PolicyType
	policyInstances    map[PolicyInstanceID]*PolicyInstance
	conflicts          map[string]*PolicyConflict
	distributionStatus map[PolicyInstanceID]map[string]*PolicyDistributionStatus
	complianceReports  map[PolicyInstanceID]map[string]*PolicyComplianceReport
	mutex              sync.RWMutex
	distributionChan   chan *PolicyDistributionRequest
	complianceChan     chan *PolicyComplianceRequest
	stopChan           chan struct{}
	
	// From smo_components.go - additional fields for SMO integration
	policies        map[string]*PolicyDefinition
	activeJobs      map[string]*EnrichmentJob
	nonRTRICClient  *NonRTRICClient
	stats           PolicyManagerStats
	mu              sync.RWMutex
}

// PolicyDefinition represents an A1 policy (from smo_components.go)
type PolicyDefinition struct {
	PolicyID      string                 `json:"policyId"`
	PolicyTypeID  string                 `json:"policyTypeId"`
	ServiceID     string                 `json:"serviceId"`
	RICInstance   string                 `json:"ricInstance"`
	PolicyData    map[string]interface{} `json:"policyData"`
	Status        PolicyStatus           `json:"status"`
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`
}

// PolicyManagerStats tracks policy manager statistics (from smo_components.go)
type PolicyManagerStats struct {
	TotalPolicies       uint64 `json:"totalPolicies"`
	ActivePolicies      uint64 `json:"activePolicies"`
	PolicyViolations    uint64 `json:"policyViolations"`
	EnrichmentJobs      uint64 `json:"enrichmentJobs"`
	PolicyUpdateRate    float64 `json:"policyUpdateRate"`
}

// NewPolicyManager creates a new policy manager
// This is the centralized constructor to avoid conflicts
func NewPolicyManager(a1Client ...A1MediatorClient) *PolicyManager {
	pm := &PolicyManager{
		xappClients:        make(map[string]*XAppClient),
		policyTypes:        make(map[PolicyTypeID]*PolicyType),
		policyInstances:    make(map[PolicyInstanceID]*PolicyInstance),
		conflicts:          make(map[string]*PolicyConflict),
		distributionStatus: make(map[PolicyInstanceID]map[string]*PolicyDistributionStatus),
		complianceReports:  make(map[PolicyInstanceID]map[string]*PolicyComplianceReport),
		distributionChan:   make(chan *PolicyDistributionRequest, 100),
		complianceChan:     make(chan *PolicyComplianceRequest, 100),
		stopChan:           make(chan struct{}),
		policies:           make(map[string]*PolicyDefinition),
		activeJobs:         make(map[string]*EnrichmentJob),
		stats:              PolicyManagerStats{},
	}
	
	if len(a1Client) > 0 {
		pm.a1Client = a1Client[0]
	}
	
	return pm
}

// Supporting Policy Types

// XAppClient represents a client for communicating with xApps
type XAppClient struct {
	ID       string
	Endpoint string
}

// PolicyInstance represents an A1 policy instance
type PolicyInstance struct {
	ID       PolicyInstanceID `json:"policy_instance_id"`
	TypeID   PolicyTypeID     `json:"policy_type_id"`
	Policy   json.RawMessage  `json:"policy"`
	Status   PolicyStatus     `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// PolicyConflict represents a policy conflict
type PolicyConflict struct {
	ConflictID          string             `json:"conflict_id"`
	PolicyInstanceID    PolicyInstanceID   `json:"policy_instance_id"`
	ConflictingPolicyID PolicyInstanceID   `json:"conflicting_policy_id"`
	ConflictType        string             `json:"conflict_type"`
	Description         string             `json:"description"`
	Resolution          string             `json:"resolution,omitempty"`
	DetectedAt          time.Time          `json:"detected_at"`
}

// PolicyDistributionStatus represents the distribution status of a policy
type PolicyDistributionStatus struct {
	PolicyInstanceID PolicyInstanceID `json:"policy_instance_id"`
	XAppID           string           `json:"xapp_id"`
	Status           string           `json:"status"`
	Message          string           `json:"message,omitempty"`
	LastUpdate       time.Time        `json:"last_update"`
}

// PolicyComplianceReport represents a policy compliance report
type PolicyComplianceReport struct {
	PolicyInstanceID PolicyInstanceID `json:"policy_instance_id"`
	XAppID           string           `json:"xapp_id"`
	ComplianceStatus string           `json:"compliance_status"`
	Violations       []string         `json:"violations,omitempty"`
	LastCheck        time.Time        `json:"last_check"`
}

// PolicyDistributionRequest represents a request to distribute a policy to xApps
type PolicyDistributionRequest struct {
	PolicyInstanceID PolicyInstanceID
	PolicyTypeID     PolicyTypeID
	Policy           json.RawMessage
	TargetXApps      []string
}

// PolicyComplianceRequest represents a request to check policy compliance
type PolicyComplianceRequest struct {
	PolicyInstanceID PolicyInstanceID
	XAppID           string
}

// Additional supporting types for SMO integration
type A1MediatorClient interface {
	IsConnected() bool
	GetHealth(ctx context.Context) (*A1Health, error)
	GetPolicyTypes(ctx context.Context) (*PolicyTypeListResponse, error)
	GetPolicyType(ctx context.Context, policyTypeID PolicyTypeID) (*PolicyType, error)
	CreatePolicyType(ctx context.Context, policyTypeID PolicyTypeID, request *PolicyTypeRequest) error
	DeletePolicyType(ctx context.Context, policyTypeID PolicyTypeID) error
	GetPolicyInstances(ctx context.Context, policyTypeID PolicyTypeID) (*PolicyInstanceListResponse, error)
	GetPolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) (*PolicyInstance, error)
	CreatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID, request *PolicyInstanceRequest) error
	UpdatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, update *PolicyInstanceUpdate) error
	DeletePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) error
	GetPolicyInstanceStatus(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) (*PolicyStatus, error)
	GetStats(ctx context.Context) (*A1Stats, error)
	ValidatePolicy(ctx context.Context, policyTypeID PolicyTypeID, policy json.RawMessage) (*PolicyValidationResult, error)
}

// A1MediatorClientImpl provides concrete implementation of A1MediatorClient interface
type A1MediatorClientImpl struct {
	httpClient *http.Client
	endpoint   string
}

// Scaling Types - centralized to avoid redeclarations

// ScalingPolicy defines scaling behavior for a component
// This is the centralized definition to avoid conflicts between smo_components.go and horizontal_scaler.go
type ScalingPolicy struct {
	// From horizontal_scaler.go - complete scaling policy
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
	
	// From smo_components.go - additional energy-aware fields
	Metric        string  `json:"metric,omitempty"`        // gbps_per_watt
	Threshold     float64 `json:"threshold,omitempty"`
	Action        string  `json:"action,omitempty"`        // scale_down_idle, optimize
}

// Additional Centralized Types to resolve remaining conflicts

// KubernetesClient interface for Kubernetes operations
// Centralized to avoid conflicts between horizontal_scaler.go and smo_components.go
type KubernetesClient interface {
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error
	GetDeploymentStatus(ctx context.Context, namespace, name string) (*DeploymentStatus, error)
	CreateDeployment(ctx context.Context, spec *DeploymentSpec) error
	DeleteDeployment(ctx context.Context, namespace, name string) error
}

// ResourceRequirements defines resource requirements for a deployment
// Centralized to avoid conflicts between horizontal_scaler.go and smo_nephio_integration_layer.go
type ResourceRequirements struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
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

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	ReadyReplicas     int32
	AvailableReplicas int32
	UnavailableReplicas int32
	UpdatedReplicas   int32
}

// PackageTask represents a package task
// Centralized to avoid conflicts between smo_components.go and smo_nephio_integration_layer.go
type PackageTask struct {
	Type    string                 `json:"type"`
	Function string                `json:"function"`
	Config  map[string]interface{} `json:"config"`
}

// HighPerformanceMessageProcessor processes messages with high performance
// Centralized to avoid conflicts between high_performance_processor.go and smo_performance_optimizer.go
type HighPerformanceMessageProcessor struct {
	// Common fields from both implementations
	numWorkers          int
	numThreads          int
	cpuCount           int
	workerPool         []Worker
	messageBuffer      []ProcessedMessage
	bufferSize         int
	processingStats    ProcessingStats
	memoryManager      *MemoryManager
	latencyTracker     *LatencyTracker
	batchProcessor     *BatchProcessor
	mu                 sync.RWMutex
}

// MessageType represents the type of message
// Centralized to avoid conflicts between common_types.go and smo_performance_optimizer.go
type MessageType int

const (
	MessageTypeE2AP MessageType = iota
	MessageTypeA1
	MessageTypeO1
)

// ErrorHandler handles errors in processing
// Centralized to avoid conflicts between error_handling.go and smo_performance_optimizer.go
type ErrorHandler struct {
	handlers map[string]ErrorHandlerFunc
	mu       sync.RWMutex
}

type ErrorHandlerFunc func(error) error

// OptimizedMessage represents an optimized message
// Centralized to avoid conflicts between common_types.go and smo_performance_optimizer.go
type OptimizedMessage struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	Payload   []byte      `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	Priority  Priority    `json:"priority"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ProcessingStats tracks processing statistics
type ProcessingStats struct {
	MessagesProcessed uint64
	ProcessingTime    time.Duration
	ErrorCount        uint64
	ThroughputMbps    float64
}

// MemoryManager manages memory allocation
type MemoryManager struct {
	mu sync.RWMutex
}

// BatchProcessor processes messages in batches
type BatchProcessor struct {
	batchSize int
	mu        sync.RWMutex
}

// Additional Types - to resolve remaining conflicts

// ProcessedMessage represents a processed message (most comprehensive version)
type ProcessedMessage struct {
	Original        *OptimizedMessage
	Result          unsafe.Pointer  // More flexible than []byte
	ResultSize      uint32
	ProcessingTime  time.Duration
	Success         bool
	Error           error
	Metadata        map[string]interface{}
}

// MessagePriority defines message priority levels
type MessagePriority int

const (
	MessagePriorityLow MessagePriority = iota
	MessagePriorityNormal
	MessagePriorityHigh
	MessagePriorityCritical
)

// SMOClient represents Enhanced SMO Client for L Release integration
type SMOClient struct {
	endpoint        string
	client          *OptimizedHTTPClient
	circuitBreaker  *CircuitBreaker
	rateLimiter     *RateLimiter
	cache           *ResponseCache
	stats           SMOClientStats
	mu              sync.RWMutex
}

// NonRTRICClient manages Non-RT RIC integration
type NonRTRICClient struct {
	endpoint        string
	client          *OptimizedHTTPClient
	policyAPI       *PolicyAPI
	enrichmentAPI   *EnrichmentAPI
	dmaapClient     *DMAAPClient
	stats           NonRTRICStats
	mu              sync.RWMutex
}

// PorchClient represents Nephio R5 Porch integration
type PorchClient struct {
	endpoint        string
	kubeClient      interface{} // k8s client
	packageRepo     *PackageRepository
	packageRevision *PackageRevisionManager
	validator       *PackageValidator
	stats           PorchStats
	mu              sync.RWMutex
}

// OCloudManager manages O-Cloud resource management
type OCloudManager struct {
	endpoint        string
	resourcePools   map[string]*ResourcePool
	energyManager   *EnergyManager
	scalingPolicy   *ScalingPolicy
	stats           OCloudStats
	mu              sync.RWMutex
}

// ResourceMonitor monitors system resources (most comprehensive version)
type ResourceMonitor struct {
	mu               sync.RWMutex
	cpuUsage         float64
	memoryUsage      float64
	networkUsage     float64
	diskUsage        float64
	connectionCount  int64
	
	// Thresholds
	cpuThreshold     float64
	memoryThreshold  float64
	networkThreshold float64
	
	// Stress testing specific fields
	cpuSamples        []float64
	memorySamples     []float64
	networkSamples    []int64
	diskSamples       []int64
	goroutineSamples  []int
	exhaustionPoint   *ResourceExhaustionPoint
}

// Logger provides structured logging with correlation IDs (slog-based)
type Logger struct {
	*slog.Logger
	component string
}

// LogrusLogger is the global structured logger instance (logrus-based)
var LogrusLogger *logrus.Logger

// TestResults aggregates all test execution results (most comprehensive version)
type TestResults struct {
	// Basic test results
	TotalTests      int           `json:"totalTests"`
	PassedTests     int           `json:"passedTests"`
	FailedTests     int           `json:"failedTests"`
	SkippedTests    int           `json:"skippedTests"`
	Duration        time.Duration `json:"duration"`
	Coverage        float64       `json:"coverage"`
	PackageResults  []PackageTestResult `json:"packageResults"`
	BenchmarkResults []BenchmarkResult  `json:"benchmarkResults"`
	
	// Performance testing specific fields
	LoadTestResults      *LoadTestResults      `json:"loadTestResults,omitempty"`
	ThroughputResults    *ThroughputResults    `json:"throughputResults,omitempty"`
	LatencyResults       *LatencyResults       `json:"latencyResults,omitempty"`
	StressTestResults    *StressTestResults    `json:"stressTestResults,omitempty"`
	StabilityResults     *StabilityResults     `json:"stabilityResults,omitempty"`
	ResourceUtilization  *ResourceUtilization  `json:"resourceUtilization,omitempty"`
	TestSummary          *TestSummary          `json:"testSummary,omitempty"`
}

// BenchmarkResult represents benchmark test results (most comprehensive version)
type BenchmarkResult struct {
	// Basic benchmark fields
	Name           string  `json:"name"`
	Package        string  `json:"package"`
	Iterations     int64   `json:"iterations"`
	NsPerOp        int64   `json:"nsPerOp"`
	BytesPerOp     int64   `json:"bytesPerOp,omitempty"`
	AllocsPerOp    int64   `json:"allocsPerOp,omitempty"`
	MBPerSec       float64 `json:"mbPerSec,omitempty"`
	
	// Comprehensive benchmark fields
	ID                      string        `json:"id"`
	Timestamp               time.Time     `json:"timestamp"`
	Duration                time.Duration `json:"duration"`
	BenchmarkType           string        `json:"benchmarkType"`
	Configuration           interface{}   `json:"configuration"`
	LatencyResults          *LatencyBenchmarkResult     `json:"latencyResults,omitempty"`
	ThroughputResults       *ThroughputBenchmarkResult  `json:"throughputResults,omitempty"`
}

// Supporting types for the comprehensive structures above

// SMOClientStats represents statistics for SMO client
type SMOClientStats struct {
	RequestCount    int64
	SuccessCount    int64
	ErrorCount      int64
	AvgResponseTime time.Duration
}

// NonRTRICStats represents statistics for Non-RT RIC client
type NonRTRICStats struct {
	PolicyRequests    int64
	EnrichmentRequests int64
	DMAAPMessages     int64
	ErrorCount        int64
}

// PorchStats represents statistics for Porch client
type PorchStats struct {
	PackageOperations int64
	ValidationCount   int64
	ErrorCount        int64
}

// OCloudStats represents statistics for O-Cloud manager
type OCloudStats struct {
	ResourceOperations int64
	EnergyOptimizations int64
	ScalingActions     int64
}

// ResourceExhaustionPoint represents a resource exhaustion point
type ResourceExhaustionPoint struct {
	Timestamp time.Time
	CPUUsage  float64
	MemUsage  float64
	Goroutines int
}

// PackageTestResult represents test results for a single package
type PackageTestResult struct {
	Package      string        `json:"package"`
	Tests        int          `json:"tests"`
	Passed       int          `json:"passed"`
	Failed       int          `json:"failed"`
	Skipped      int          `json:"skipped"`
	Duration     time.Duration `json:"duration"`
	Coverage     float64       `json:"coverage"`
	TestCases    []TestCase    `json:"testCases"`
}

// TestCase represents an individual test case result
type TestCase struct {
	Name     string        `json:"name"`
	Package  string        `json:"package"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
}

// Supporting types for benchmark results
type LatencyBenchmarkResult struct {
	Min     time.Duration `json:"min"`
	Max     time.Duration `json:"max"`
	Mean    time.Duration `json:"mean"`
	Median  time.Duration `json:"median"`
	P95     time.Duration `json:"p95"`
	P99     time.Duration `json:"p99"`
}

type ThroughputBenchmarkResult struct {
	RequestsPerSecond float64 `json:"requestsPerSecond"`
	BytesPerSecond    float64 `json:"bytesPerSecond"`
	MaxThroughput     float64 `json:"maxThroughput"`
}

// Additional types from remaining conflicts

// Indication represents an indication message from an E2 node (comprehensive version)
type Indication struct {
	// From subscription_models.go (more comprehensive)
	SubscriptionID SubscriptionID `json:"subscriptionId"`
	E2NodeID       string         `json:"e2NodeId"`
	RANFunctionID  uint32         `json:"ranFunctionId"`
	ActionID       uint32         `json:"actionId"`
	IndicationSN   uint32         `json:"indicationSn"`
	IndicationHeader []byte       `json:"indicationHeader"`
	IndicationMessage []byte      `json:"indicationMessage"`
	CallProcessID  []byte         `json:"callProcessId,omitempty"`
	Timestamp      time.Time      `json:"timestamp"`
	
	// From throughput_testing.go (additional fields)
	ID            string         `json:"id,omitempty"`
	NodeID        string         `json:"nodeId,omitempty"`
	Data          []byte         `json:"data,omitempty"`
	ProcessingStart time.Time    `json:"processingStart,omitempty"`
}

// NodeStatus represents the status of an E2 node
type NodeStatus int

const (
	NodeStatusConnected NodeStatus = iota
	NodeStatusDisconnected
	NodeStatusConnecting
	NodeStatusError
)

// MessageHandler defines the interface for handling RMR messages - CENTRALIZED
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *RMRMessage) error
	GetMessageTypes() []uint32
}

// RMRMessage represents an RMR message - CENTRALIZED 
type RMRMessage struct {
	MessageType   uint32            `json:"messageType"`
	SubscriptionID string           `json:"subscriptionId,omitempty"`
	TransactionID string            `json:"transactionId"`
	Payload       []byte            `json:"payload"`
	Source        string            `json:"source"`
	Target        string            `json:"target,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// EventTriggerType represents the type of event trigger - CENTRALIZED
type EventTriggerType string

const (
	EventTriggerTypePeriodic EventTriggerType = "PERIODIC"
	EventTriggerTypeOnChange EventTriggerType = "ON_CHANGE"
	EventTriggerTypeOnDemand EventTriggerType = "ON_DEMAND"
)

// Supporting types that were missing from the list

// SubscriptionID represents a subscription identifier
type SubscriptionID string

// Additional interface types needed for compilation
type OptimizedHTTPClient interface{}
type RateLimiter interface{}
type ResponseCache interface{}
type PolicyAPI interface{}
type EnrichmentAPI interface{}
type DMAAPClient interface{}
type EnrichmentJob interface{}
type PackageRepository interface{}
type PackageRevisionManager interface{}
type PackageValidator interface{}
type ResourcePool interface{}
type EnergyManager interface{}

// Additional result types for comprehensive testing
type LoadTestResults interface{}
type ThroughputResults interface{}
type LatencyResults interface{}
type StressTestResults interface{}
type StabilityResults interface{}
type ResourceUtilization interface{}
// Advanced SMO Performance Optimizer types
type SMOIntegration interface{}
type NephioR5Integration interface{}
type PerformanceEngine interface{}
type OptimizedBatchProcessor interface{}
type ConnectionMultiplexer interface{}
type IntelligentLoadDistributor interface{}
type WeightedLoadBalancer interface{}
type RoutingCache interface{}
type RoutingMetrics interface{}
type E2ConnectionPool interface{}


// Additional missing types
type SubscriptionOptimizer interface{}
type HighSpeedIndicationProcessor interface{}
type HugePagesMemoryManager interface{}
type ProductionGCOptimizer interface{}
type E2LoadBalancer interface{}
type E2NodeMetrics interface{}
type StreamDirection interface{}
type StreamStats interface{}
type CPULoadBalancer interface{}


// Connection pool types
type HTTPConnectionPool interface{}
type SCTPConnectionPool interface{}
type E2MemoryPool interface{}
type E2ConnectionFactory interface{}
type E2ConnectionValidator interface{}
type E2PoolStats interface{}
type E2HealthChecker interface{}
type SCTPStats interface{}
type WorkerPoolStats interface{}
type ConnectionRateLimiter interface{}


// Additional performance and connection types
type ConnectionClusterMetrics interface{}
type ConnectionProfiler interface{}
type RealTimeScheduler interface{}
type PriorityQueue interface{}
type LoadBalancingStrategy interface{}
type HealthScore interface{}
type HealthAlertManager interface{}
type DynamicResourceAllocator interface{}
type AdaptiveBackpressureManager interface{}
type RealTimeProfiler interface{}


// Performance monitoring types
type PerformanceAnalyzer interface{}
type AutoPerformanceTuner interface{}
type CircuitBreakerCluster interface {
	IsOpen(nodeID string) bool
	RecordFailure(nodeID string)
	RecordSuccess(nodeID string)
}
type ComprehensiveHealthMonitor interface{}
type LatencyAnalyzer interface{}
type ThroughputMonitor interface{}
type SMOPerformanceMonitor interface{}
type NephioPerformanceMonitor interface{}
type E2InterfaceMonitor interface{}
type IndicationMonitor interface{}


// Final monitoring and benchmark types
type SubscriptionMonitor interface{}
type APIPerformanceMonitor interface{}
type ConnectionMonitor interface{}
type PerformancePredictor interface{}
type LoadTester interface{}
type LatencyBenchmark interface{}
type ThroughputBenchmark interface{}
type ScalabilityBenchmark interface{}
type StressBenchmark interface{}
type EnduranceBenchmark interface{}


// System metrics and stats types
type StressTester interface{}
type DashboardAPIStats interface{}
type SMOIntegrationStats interface{}
type NephioPerformanceStats interface{}
type ErrorStats interface{}
type NetworkMetrics interface{}
type DiskMetrics interface{}
type RuntimeMetrics interface{}
type ContainerMetrics interface{}
type LoadAverage interface{}


// Final remaining types
type AvailabilityStats interface{}
type TargetCompliance interface{}
type E2TerminationClient interface{}
type O2CloudClient interface{}
type EnhancedLoadBalancer interface{}
type ThroughputSample interface{}
type ConnectionManager interface{}


// Final API and dashboard types
type CompressionManager interface{}
type AuthManager interface{}
type SecurityHeaders interface{}
type E2HealthMonitor interface{}
type E2Subscription interface{}
type NodeRateLimiter interface{}
type NodeCircuitBreaker interface{}
type ResourceConsumption interface{}
type LoadBalancingAlgorithm interface{}
type Backend interface{}


// Final 10 missing types to complete the build
type PerformanceTestSuite interface{}
type CompressionHandler interface{}
type AuthenticationManager interface{}
type RateLimitManager interface{}
type WebSocketPool interface{}
type BroadcastMessage interface{}
type WSManagerStats interface{}
type BackendHealthChecker interface{}
type RequestRouter interface{}
type StickySessionManager interface{}


// Production dashboard API types
type HighPerformanceSubscriptionManager interface{}
type E2ConnectionManager interface{}
type E2NodeRouter interface{}
type E2LoadDistributor interface{}
type E2AlertManager interface{}
type E2NodeManagerStats interface{}
type E2NodeMapShard interface{}
type IndicationPipeline interface{}
type IndicationBatchProcessor interface{}
type IndicationSIMDProcessor interface{}


// Final 5 types to complete zero errors
type APIHealthChecker interface{}
type FastSubscriptionMatcher interface{}
type IndicationCompressionHandler interface{}
type IndicationProcessorStats interface{}
type IndicationRoutingEngine interface{}


// Final completion types - already defined, but now adding the missing ones from smo_nephio_integration_layer.go
type PolicyManagerClient interface{}
type RAppManagerClient interface{}
type RequestCache interface{}
type PackageCache interface{}

// Final missing types from smo_nephio_integration_layer.go build errors
type PackageManagerClient interface{}
type ResourceProvisionerClient interface{}
type SMOLoadBalancer interface{}
type NephioLoadBalancer interface{}
type IntegrationCache interface{}
type IntegrationCircuitBreaker interface{}
type IntegrationMetrics interface{}
type IntegrationPerformanceTracker interface{}
type IntegrationHealthMonitor interface{}

// Additional supporting types for SMO Nephio Integration Layer
type PackageRevisionResponse interface{}
type OCloudResourceResponse interface{}
type RAppDeploymentResponse interface{}

// Additional missing types from smo_nephio_integration_layer.go compilation errors
type PackageDeployer interface{}
type PackageBatchProcessor interface{}
type AsyncPackageProcessor interface{}
type ResourcePoolManager interface{}
type CapacityPlanner interface{}

// Supporting struct types from slice_e2_integration.go - CENTRALIZED TO AVOID REDECLARATION
type ServiceProfile struct {
	MaxDataRate *DataRate `json:"maxDataRate"`
	Latency     int32     `json:"latency"`
	Reliability float64   `json:"reliability"`
}

type DataRate struct {
	Value int64  `json:"value"`
	Unit  string `json:"unit"`
}

type ComputeResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type NetworkResources struct {
	Bandwidth int64 `json:"bandwidth"`
}

type StorageResources struct {
	Size string `json:"size"`
}

type AcceleratorResources struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type SLARequirements struct {
	Availability  float64       `json:"availability"`
	Latency      time.Duration `json:"latency"`
	Throughput   int64         `json:"throughput"`
}

// NetworkSlice represents a network slice - CENTRALIZED TO AVOID REDECLARATION
type NetworkSlice struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Throughput  float64 `json:"throughput"`
	Latency     float64 `json:"latency"`
	Reliability float64 `json:"reliability"`
	CPU         int     `json:"cpu"`
	Memory      int     `json:"memory"`
	Bandwidth   int64   `json:"bandwidth"`
	Priority    int     `json:"priority"`
}

// Missing types from stability_testing.go
type MemorySample interface{}
type CPUSample interface{}
type ErrorRateSample interface{}

// Additional performance testing types - CENTRALIZED TO AVOID REDECLARATION

// Additional missing types from testing files
type PerformanceDegradation interface{}
type ConnectionStability interface{}
type RecoveryMetrics interface{}
type SystemLimits interface{}

// Test orchestrator types
type ComprehensiveLoadTest interface{}
type NephioR5IntegrationTest interface{}
type LoadTestConfig interface{}
type NephioR5Config interface{}
type LoadTestReport interface{}

// Additional missing types from build errors
type NephioTestReport interface{}
type QueueMetrics interface{}
type Subscription interface{}
type E2MessageType int

const (
	E2MessageTypeIndication E2MessageType = iota
	E2MessageTypeSubscription
	E2MessageTypePolicy
)
type ProcessingResult struct {
	NodeID           string
	MessageType      E2MessageType
	ProcessedData    []byte
	Success          bool
	ProcessingTimeNs int64
}
type ScalabilityBenchmarkResult interface{}
type StressBenchmarkResult interface{}
type LatencyMeasurement interface{}
// NOTE: PerformanceTestRunner moved from performance_test_runner.go to avoid redeclaration
type PerformanceTestRunner struct {
	Suite           interface{} // *PerformanceTestSuite
	ValidationRules interface{} // *ValidationRules  
	TestReport      interface{} // *ComprehensiveTestReport
}

// NOTE: PerformanceMetrics moved from performance_optimizer.go to avoid redeclaration
type PerformanceMetrics struct {
	ProcessedMessages uint64
	AverageLatency    time.Duration
	Throughput        uint64
	MemoryStats       interface{} // MemoryStats
	GCStats           interface{} // runtime.MemStats
	ProfileData       map[string]interface{} // map[string]*ProfileData
}

// Missing types identified in final build test
type ResponseTimeMetrics interface{}
type SubscriptionListResponse interface{}
type SubscriptionUpdate interface{}
type Action interface{}

