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

// GlobalE2NodeID represents the global E2 node identifier
type GlobalE2NodeID struct {
	PLMNIdentity []byte `json:"plmnIdentity"`
	E2NodeID     []byte `json:"e2NodeId"`
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
	
	// Additional fields needed by connection_pool_manager.go
	MemoryUsage float64   `json:"memoryUsage"` // Memory usage percentage (0-100)
	CPUUsage    float64   `json:"cpuUsage"`    // CPU usage percentage (0-100)
	
	// Additional fields needed by horizontal_scaler.go
	AverageLatency time.Duration `json:"averageLatency"`
	RequestRate    float64       `json:"requestRate"`
}

// ResourceConsumption represents resource consumption metrics for testing
type ResourceConsumption struct {
	CPUUsage       float64   `json:"cpuUsage"`       // CPU usage percentage (0-100)
	MemoryUsage    uint64    `json:"memoryUsage"`    // Memory usage in bytes
	NetworkIOBytes uint64    `json:"networkIOBytes"` // Network I/O in bytes
	DiskIOBytes    uint64    `json:"diskIOBytes"`    // Disk I/O in bytes
	Timestamp      time.Time `json:"timestamp"`      // When the measurement was taken
	NodeID         string    `json:"nodeId"`         // Node identifier
	ProcessID      int       `json:"processId"`      // Process ID
	ThreadCount    int       `json:"threadCount"`    // Number of threads
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

// LatencyMetrics represents latency measurements
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
	Count                   int64         `json:"count"`
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	TestName        string    `json:"testName"`
	RequirementMet  bool      `json:"requirementMet"`
	ActualValue     float64   `json:"actualValue"`
	RequiredValue   float64   `json:"requiredValue"`
	PerformanceGap  float64   `json:"performanceGap"`
	Details         string    `json:"details"`
	Timestamp       time.Time `json:"timestamp"`
	Rule     string      `json:"rule"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
	Status   string      `json:"status"`
	Message  string      `json:"message"`
	Valid      bool                       `json:"valid"`
	Errors     []string                   `json:"errors"`
	Warnings   []string                   `json:"warnings"`
}

// ResourceMonitor monitors system resources - concrete type needed
type ResourceMonitor struct {
	mu               sync.RWMutex
	cpuUsage         float64
	memoryUsage      float64
	networkUsage     float64
	diskUsage        float64
	connectionCount  int64
	lastCheck        time.Time
}

// Load Balancing Types

// LoadBalancingAlgorithm represents different load balancing algorithms
type LoadBalancingAlgorithm int

const (
	LoadBalancingAlgorithmRoundRobin LoadBalancingAlgorithm = iota
	LoadBalancingAlgorithmLeastConnections
	LoadBalancingAlgorithmWeightedRoundRobin
	LoadBalancingAlgorithmIPHash
	LoadBalancingAlgorithmLeastResponseTime
	LoadBalancingAlgorithmResource
)

// Backend represents a backend server for load balancing
type Backend struct {
	ID               string            `json:"id"`
	Address          string            `json:"address"`
	Port             int               `json:"port"`
	Weight           int               `json:"weight"`
	IsHealthy        bool              `json:"isHealthy"`
	ActiveConnections int              `json:"activeConnections"`
	ResponseTime     time.Duration     `json:"responseTime"`
	LastHealthCheck  time.Time         `json:"lastHealthCheck"`
	HealthCheckURL   string            `json:"healthCheckUrl"`
	Metadata         map[string]string `json:"metadata"`
	TotalRequests    uint64            `json:"totalRequests"`
	FailedRequests   uint64            `json:"failedRequests"`
	mu               sync.RWMutex      `json:"-"`
}

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

// RICComponentHealth represents the health status of a RIC component
type RICComponentHealth struct {
	ComponentName string    `json:"componentName"`
	Endpoint      string    `json:"endpoint"`
	Status        string    `json:"status"`
	Error         string    `json:"error,omitempty"`
	LastChecked   time.Time `json:"lastChecked"`
	ResponseTime  time.Duration `json:"responseTime,omitempty"`
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

// ActionType represents the type of subscription action
type ActionType uint32

const (
	ActionTypeReport ActionType = iota
	ActionTypeInsert
	ActionTypePolicy
)

// SubscriptionAction represents an action within a subscription
type SubscriptionAction struct {
	ActionID     uint32      `json:"actionId"`
	ActionType   ActionType  `json:"actionType"`
	Definition   []byte      `json:"definition"`
	SubsequentAction *SubscriptionAction `json:"subsequentAction,omitempty"`
}

// Priority defines work priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// WorkType defines the type of work
type WorkType int

const (
	WorkTypeE2APMessage WorkType = iota
	WorkTypeSubscription
	WorkTypeIndication
	WorkTypeControl
	WorkTypePolicyUpdate
)

// MessageType represents the type of message
type MessageType int

const (
	MessageTypeE2AP MessageType = iota
	MessageTypeA1
	MessageTypeO1
)

// Testing and Validation Types

// TestSeverity indicates the importance of a test
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

// TestSummary represents a summary of test results
type TestSummary struct {
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration time.Duration `json:"duration"`
	Coverage float64       `json:"coverage"`
}

// BenchmarkResult represents benchmark test results
type BenchmarkResult struct {
	Name           string  `json:"name"`
	Package        string  `json:"package"`
	Iterations     int64   `json:"iterations"`
	NsPerOp        int64   `json:"nsPerOp"`
	BytesPerOp     int64   `json:"bytesPerOp,omitempty"`
	AllocsPerOp    int64   `json:"allocsPerOp,omitempty"`
	MBPerSec       float64 `json:"mbPerSec,omitempty"`
}

// Logger provides structured logging with correlation IDs (slog-based)
type Logger struct {
	*slog.Logger
	component string
}

// LogrusLogger is the global structured logger instance (logrus-based)
var LogrusLogger *logrus.Logger

// ProcessedMessage represents a processed message
type ProcessedMessage struct {
	Original        *OptimizedMessage
	Result          unsafe.Pointer  
	ResultSize      uint32
	ProcessingTime  time.Duration
	Success         bool
	Error           error
	Metadata        map[string]interface{}
}

// OptimizedMessage represents an optimized message
type OptimizedMessage struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	Payload   []byte      `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	Priority  Priority    `json:"priority"`
	Metadata  map[string]interface{} `json:"metadata"`
}

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

// Policy Management Types - needed by a1_models.go

// PolicyTypeID represents a policy type identifier
type PolicyTypeID string

// PolicyInstanceID represents a policy instance identifier
type PolicyInstanceID string

// PolicyType represents an A1 policy type
type PolicyType struct {
	ID          PolicyTypeID    `json:"policy_type_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
	CreatedAt   time.Time       `json:"created_at"`
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

// PolicyStatus represents policy status
type PolicyStatus struct {
	Status     string    `json:"status"`
	Reason     string    `json:"reason,omitempty"`
	LastUpdate time.Time `json:"last_update"`
}

// Subscription Management Types

// SubscriptionID represents a subscription identifier
type SubscriptionID string

// EventTriggerType represents the type of event trigger
type EventTriggerType string

const (
	EventTriggerTypePeriodic EventTriggerType = "PERIODIC"
	EventTriggerTypeOnChange EventTriggerType = "ON_CHANGE"  
	EventTriggerTypeOnDemand EventTriggerType = "ON_DEMAND"
)

// RANParameter represents a RAN parameter
type RANParameter struct {
	ParameterID    uint32      `json:"parameterId"`
	ParameterName  string      `json:"parameterName"`
	ParameterValue interface{} `json:"parameterValue"`
	ParameterType  string      `json:"parameterType"`
}

// E2NodeSimulator represents a simulated E2 node for testing
type E2NodeSimulator struct {
	mu                sync.RWMutex
	nodeID            string
	globalE2NodeID    GlobalE2NodeID
	ricAddress        string
	ricPort           uint32
	localAddress      string
	localPort         uint32
	conn              interface{}
	isConnected       bool
	protocolHandler   interface{}
	encoder           interface{}
	isRunning         bool
	ctx               context.Context
	cancel            context.CancelFunc
	ranFunctions      []RANFunction
	serviceModels     []ServiceModel
	subscriptions     map[string]*SimulatedSubscription
	ID               string
	ConnectionTime   time.Time
	Subscriptions_   []string
	IndicationCount  int64
	LastActivity     time.Time
	ErrorCount       int64
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

// ServiceModelRegistry manages service model definitions 
type ServiceModelRegistry struct {
	models map[string]*ServiceModelDefinition
	mu           sync.RWMutex
	serviceModels map[ServiceModelType]ServiceModelAPI
	capabilities  map[ServiceModelType]*ServiceModelCapabilities
	statistics    map[ServiceModelType]*ServiceModelStatistics
	supportedTypes   []ServiceModelType
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

// ServiceModelType represents the type of service model
type ServiceModelType string

const (
	ServiceModelTypeKPM ServiceModelType = "kpm"      
	ServiceModelTypeRC  ServiceModelType = "rc"       
	ServiceModelTypeNI  ServiceModelType = "ni"       
	ServiceModelTypeTS  ServiceModelType = "ts"       
	ServiceModelTypeQoE ServiceModelType = "qoe"      
	ServiceModelTypeSON ServiceModelType = "son"      
)

// ServiceModelAPI interface for service model implementations
type ServiceModelAPI interface {
	GetServiceModelType() ServiceModelType
	GetSupportedOperations() []string
	ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error)
	ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error)
	ValidateMessage(messageType string, data []byte) error
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

// E2SMNIIndicationHeader represents NI indication header
type E2SMNIIndicationHeader struct {
	IndicationHeaderFormat  int32     `json:"indicationHeaderFormat"`
	InterfaceType           string    `json:"interfaceType"`
	InterfaceID             string    `json:"interfaceId"`
	Timestamp               time.Time `json:"timestamp"`
}

// E2SMNIIndicationMessage represents NI indication message
type E2SMNIIndicationMessage struct {
	IndicationMessageFormat int32     `json:"indicationMessageFormat"`
	InterfaceMessage        []byte    `json:"interfaceMessage"`
	InterfaceMetadata       map[string]interface{} `json:"interfaceMetadata"`
}

// E2SMRCControlHeader represents RC control header
type E2SMRCControlHeader struct {
	ControlHeaderFormat     int32     `json:"controlHeaderFormat"`
	UEIdentifier           string    `json:"ueIdentifier"`
	ControlMessageType     string    `json:"controlMessageType"`
	Timestamp              time.Time `json:"timestamp"`
}

// E2SMRCControlMessage represents RC control message
type E2SMRCControlMessage struct {
	ControlMessageFormat    int32     `json:"controlMessageFormat"`
	ControlCommand          []byte    `json:"controlCommand"`
	ControlParameters       map[string]interface{} `json:"controlParameters"`
}

// MeasurementData represents measurement data point
type MeasurementData struct {
	MeasurementID       uint32                `json:"measurementId"`
	MeasurementValue    interface{}           `json:"measurementValue"`
	MeasurementType     string               `json:"measurementType"`
	Labels              map[string]string    `json:"labels"`
}

// MeasurementInfo represents measurement information
type MeasurementInfo struct {
	MeasurementName         string            `json:"measurementName"`
	MeasurementID           uint32            `json:"measurementId"`
	MeasurementDescription  string            `json:"measurementDescription"`
	Units                   string            `json:"units"`
}

// SCTPStats represents SCTP connection statistics
type SCTPStats struct {
	AssociationID       uint32    `json:"associationId"`
	BytesSent          uint64    `json:"bytesSent"`
	BytesReceived      uint64    `json:"bytesReceived"`
	MessagesSent       uint64    `json:"messagesSent"`
	MessagesReceived   uint64    `json:"messagesReceived"`
	ErrorCount         uint64    `json:"errorCount"`
	LastActivity       time.Time `json:"lastActivity"`
	ConnectionState    string    `json:"connectionState"`
}

// StreamDirection represents the direction of a stream
type StreamDirection int

const (
	StreamDirectionInbound StreamDirection = iota
	StreamDirectionOutbound
	StreamDirectionBidirectional
)

// StreamStats represents statistics for a stream
type StreamStats struct {
	StreamID            uint16          `json:"streamId"`
	Direction           StreamDirection `json:"direction"`
	BytesSent          uint64          `json:"bytesSent"`
	BytesReceived      uint64          `json:"bytesReceived"`
	MessagesSent       uint64          `json:"messagesSent"`
	MessagesReceived   uint64          `json:"messagesReceived"`
	LastActivity       time.Time       `json:"lastActivity"`
}

// Missing types needed for other files - using interfaces to avoid redeclaration
type ResponseTimeMetrics interface{}
type SubscriptionListResponse interface{}
type SubscriptionUpdate interface{}
type Action interface{}

// MessageHandler defines the interface for handling RMR messages 
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *RMRMessage) error
	GetMessageTypes() []uint32
}

// RMRMessage represents an RMR message 
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

// Advanced SMO types needed by various files
type OCloudManager interface{}
type WeightedLoadBalancer interface{}
type RoutingCache interface{}
type RoutingMetrics interface{}
type E2ConnectionPool interface{}
type E2LoadBalancer interface{}
type E2NodeMetrics interface{}
type PerformanceEngine interface{}

// Additional interfaces for advanced SMO performance optimizer
type ConnectionMultiplexer interface{}
type IntelligentLoadDistributor interface{}
type SubscriptionOptimizer interface{}
type HighSpeedIndicationProcessor interface{}
type DynamicResourceAllocator interface{}
type AdaptiveBackpressureManager interface{}
type CircuitBreakerCluster interface{}
type ComprehensiveHealthMonitor interface{}

// A1 Mediator Client interface
type A1MediatorClient interface{}

// Performance monitoring interfaces
type LatencyAnalyzer interface{}
type ThroughputMonitor interface{}
type SMOPerformanceMonitor interface{}
type NephioPerformanceMonitor interface{}
type E2InterfaceMonitor interface{}
type IndicationMonitor interface{}
type SubscriptionMonitor interface{}
type APIPerformanceMonitor interface{}
type ConnectionMonitor interface{}

// Additional performance testing interfaces
type PerformancePredictor interface{}
type LoadTester interface{}
type StressTester interface{}

// External client interfaces
type E2TerminationClient interface{}
type O2CloudClient interface{}
type PorchClient interface{}
type KubernetesClient interface{}

// Enhanced connection pool interfaces
type E2MemoryPool interface{}
type E2ConnectionFactory interface{}
type E2ConnectionValidator interface{}
type HTTPConnectionPool interface{}
type SCTPConnectionPool interface{}
type E2PoolStats interface{}
type E2HealthChecker interface{}
type CPULoadBalancer interface{}
type RealTimeScheduler interface{}
type PriorityQueue interface{}
type Worker interface{}
type WorkerPoolStats interface{}
type ConnectionRateLimiter interface{}

// Additional interface types for complete resolution
type HugePagesMemoryManager interface{}
type ConnectionClusterMetrics interface{}
type ConnectionProfiler interface{}
type LoadBalancingStrategy interface{}
type HealthScore interface{}
type HealthAlertManager interface{}
type RouteEntry interface{}
type SMOClient interface{}
type NonRTRICClient interface{}
type PolicyManager interface{}

// Final interface types to resolve all conflicts 
type HighPerformanceMessageProcessor interface{}
type EnhancedLoadBalancer interface{}
type HorizontalScaler interface{}
type ThroughputSample interface{}
type ConnectionManager interface{}
type E2Subscription interface{}

// Additional interface types for complete resolution - final batch
type ResponseCache interface{}
type CompressionManager interface{}
type AuthManager interface{}
type SecurityHeaders interface{}
type E2HealthMonitor interface{}
type NodeRateLimiter interface{}
type NodeCircuitBreaker interface{}

// MISSING CONCRETE TYPES - Adding required struct types

// WorkItem represents a unit of work to be processed
type WorkItem struct {
	ID          string                 `json:"id"`
	Type        WorkType               `json:"type"`
	Priority    Priority               `json:"priority"`
	Payload     []byte                 `json:"payload"`
	Metadata    map[string]interface{} `json:"metadata"`
	SubmittedAt time.Time              `json:"submittedAt"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	RetryCount  int                    `json:"retryCount"`
	MaxRetries  int                    `json:"maxRetries"`
	Timeout     time.Duration          `json:"timeout"`
}

// WorkResult represents the result of processing a work item
type WorkResult struct {
	WorkItemID      string                 `json:"workItemId"`
	Success         bool                   `json:"success"`
	Result          interface{}            `json:"result,omitempty"`
	Error           error                  `json:"error,omitempty"`
	ProcessingTime  time.Duration          `json:"processingTime"`
	WorkerID        string                 `json:"workerId"`
	CompletedAt     time.Time              `json:"completedAt"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// WorkerStats represents statistics for a worker
type WorkerStats struct {
	WorkerID           string        `json:"workerId"`
	TasksProcessed     uint64        `json:"tasksProcessed"`
	TasksCompleted     uint64        `json:"tasksCompleted"`
	TasksFailed        uint64        `json:"tasksFailed"`
	AverageProcessTime time.Duration `json:"averageProcessTime"`
	TotalProcessTime   time.Duration `json:"totalProcessTime"`
	LastActivity       time.Time     `json:"lastActivity"`
	Status             string        `json:"status"`
	QueueSize          int           `json:"queueSize"`
	ErrorRate          float64       `json:"errorRate"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Alloc          uint64    `json:"alloc"`          // bytes allocated and still in use
	TotalAlloc     uint64    `json:"totalAlloc"`     // bytes allocated (even if freed)
	Sys            uint64    `json:"sys"`            // bytes obtained from system
	Lookups        uint64    `json:"lookups"`        // number of pointer lookups
	Mallocs        uint64    `json:"mallocs"`        // number of mallocs
	Frees          uint64    `json:"frees"`          // number of frees
	HeapAlloc      uint64    `json:"heapAlloc"`      // bytes allocated and still in use
	HeapSys        uint64    `json:"heapSys"`        // bytes obtained from system
	HeapIdle       uint64    `json:"heapIdle"`       // bytes in idle (unused) spans
	HeapInuse      uint64    `json:"heapInuse"`      // bytes in in-use spans
	HeapReleased   uint64    `json:"heapReleased"`   // bytes released to OS
	HeapObjects    uint64    `json:"heapObjects"`    // total number of allocated objects
	StackInuse     uint64    `json:"stackInuse"`     // bytes in stack spans
	StackSys       uint64    `json:"stackSys"`       // bytes obtained from system for stack
	MSpanInuse     uint64    `json:"mSpanInuse"`     // bytes of allocated mspan structures
	MSpanSys       uint64    `json:"mSpanSys"`       // bytes of memory obtained from OS for mspan
	MCacheInuse    uint64    `json:"mCacheInuse"`    // bytes of allocated mcache structures  
	MCacheSys      uint64    `json:"mCacheSys"`      // bytes of memory obtained from OS for mcache
	BuckHashSys    uint64    `json:"buckHashSys"`    // bytes of memory in profiling bucket hash tables
	GCSys          uint64    `json:"gcSys"`          // bytes of memory in garbage collection metadata
	OtherSys       uint64    `json:"otherSys"`       // bytes of memory in miscellaneous off-heap runtime allocations
	NextGC         uint64    `json:"nextGC"`         // target heap size of the next GC cycle
	LastGC         uint64    `json:"lastGC"`         // time the last garbage collection finished
	PauseTotalNs   uint64    `json:"pauseTotalNs"`   // cumulative nanoseconds in GC stop-the-world pauses
	PauseNs        [256]uint64 `json:"pauseNs"`      // circular buffer of recent GC stop-the-world pause times
	PauseEnd       [256]uint64 `json:"pauseEnd"`     // circular buffer of recent GC stop-the-world pause end times
	NumGC          uint32    `json:"numGC"`          // number of completed GC cycles
	NumForcedGC    uint32    `json:"numForcedGC"`    // number of GC cycles that were forced by the application
	GCCPUFraction  float64   `json:"gcCPUFraction"`  // fraction of this program's available CPU time used by the GC
	EnableGC       bool      `json:"enableGC"`       // boolean that reports whether GC is enabled
	DebugGC        bool      `json:"debugGC"`        // boolean that reports whether debugGC is enabled
	Timestamp      time.Time `json:"timestamp"`      // when these stats were collected
}

// DeploymentSpec represents a Kubernetes deployment specification
type DeploymentSpec struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Replicas    int32             `json:"replicas"`
	Image       string            `json:"image"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Resources   ResourceRequirements `json:"resources"`
	Ports       []ContainerPort   `json:"ports"`
	Env         []EnvVar          `json:"env"`
	VolumeMounts []VolumeMount    `json:"volumeMounts"`
	Strategy    DeploymentStrategy `json:"strategy"`
}

// ResourceRequirements represents resource requirements for a container
type ResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty"`
}

// ResourceList represents a set of named resource quantities
type ResourceList map[string]string

// ContainerPort represents a port exposed by a container
type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

// EnvVar represents an environment variable present in a Container
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// VolumeMount describes a mounting of a Volume within a container
type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// DeploymentStrategy describes how to replace existing pods with new ones
type DeploymentStrategy struct {
	Type          string                 `json:"type"`
	RollingUpdate *RollingUpdateDeployment `json:"rollingUpdate,omitempty"`
}

// RollingUpdateDeployment represents rolling update configuration
type RollingUpdateDeployment struct {
	MaxUnavailable interface{} `json:"maxUnavailable,omitempty"`
	MaxSurge       interface{} `json:"maxSurge,omitempty"`
}

// ScalingPolicy represents a scaling policy for auto-scaling
type ScalingPolicy struct {
	ComponentName     string        `json:"componentName"`
	Enabled          bool          `json:"enabled"`
	MinReplicas      int32         `json:"minReplicas"`
	MaxReplicas      int32         `json:"maxReplicas"`
	TargetCPU        float64       `json:"targetCPU"`        // Target CPU utilization percentage (0-100)
	TargetMemory     float64       `json:"targetMemory"`     // Target memory utilization percentage (0-100)
	TargetLatency    time.Duration `json:"targetLatency"`    // Target latency threshold
	TargetThroughput int32         `json:"targetThroughput"` // Target throughput (requests/sec)
	ScaleUpCooldown  time.Duration `json:"scaleUpCooldown"`  // Cooldown period after scaling up
	ScaleDownCooldown time.Duration `json:"scaleDownCooldown"` // Cooldown period after scaling down
	ScaleUpSteps     int32         `json:"scaleUpSteps"`     // Number of replicas to add when scaling up
	ScaleDownSteps   int32         `json:"scaleDownSteps"`   // Number of replicas to remove when scaling down
	Algorithm        string        `json:"algorithm"`        // Scaling algorithm to use (cpu, memory, latency, throughput, composite)
	Weights          ScalingWeights `json:"weights"`         // Weights for composite scaling algorithm
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
}

// ScalingWeights represents weights for different metrics in composite scaling
type ScalingWeights struct {
	CPU        float64 `json:"cpu"`
	Memory     float64 `json:"memory"`
	Latency    float64 `json:"latency"`
	Throughput float64 `json:"throughput"`
}

// TestResults represents comprehensive test results
type TestResults struct {
	TestSuite          string             `json:"testSuite"`
	StartTime          time.Time          `json:"startTime"`
	EndTime            time.Time          `json:"endTime"`
	Duration           time.Duration      `json:"duration"`
	TestsRun           int                `json:"testsRun"`
	TestsPassed        int                `json:"testsPassed"`
	TestsFailed        int                `json:"testsFailed"`
	TestsSkipped       int                `json:"testsSkipped"`
	PassRate           float64            `json:"passRate"`
	FailureRate        float64            `json:"failureRate"`
	Coverage           float64            `json:"coverage"`
	
	// Performance test results
	LoadTestResults    *LoadTestResults   `json:"loadTestResults,omitempty"`
	StressTestResults  *StressTestResults `json:"stressTestResults,omitempty"`
	LatencyTestResults *LatencyTestResults `json:"latencyTestResults,omitempty"`
	
	// Individual test results
	TestCases          []TestCaseResult   `json:"testCases"`
	
	// Benchmark results
	BenchmarkResults   []BenchmarkResult  `json:"benchmarkResults,omitempty"`
	
	// Resource consumption during tests
	ResourceUsage      *ResourceConsumption `json:"resourceUsage,omitempty"`
	
	// Errors and warnings
	Errors             []string           `json:"errors,omitempty"`
	Warnings           []string           `json:"warnings,omitempty"`
	
	// Additional fields needed by performance_test_runner.go
	ThroughputResults  *ThroughputResults `json:"throughputResults,omitempty"`
	LatencyResults     *LatencyResults    `json:"latencyResults,omitempty"`
	StabilityResults   *StabilityResults  `json:"stabilityResults,omitempty"`
}

// TestCaseResult represents the result of an individual test case
type TestCaseResult struct {
	Name        string        `json:"name"`
	Package     string        `json:"package"`
	Status      string        `json:"status"` // pass, fail, skip
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	Output      string        `json:"output,omitempty"`
	Assertions  int           `json:"assertions"`
}

// LoadTestResults represents results from load testing
type LoadTestResults struct {
	MaxConcurrentE2Nodes   int           `json:"maxConcurrentE2Nodes"`
	AverageResponseTime    time.Duration `json:"averageResponseTime"`
	RequestsPerSecond      float64       `json:"requestsPerSecond"`
	ErrorRate              float64       `json:"errorRate"`
	ThroughputMbps         float64       `json:"throughputMbps"`
	ResourceUtilization    ResourceConsumption `json:"resourceUtilization"`
}

// StressTestResults represents results from stress testing
type StressTestResults struct {
	BreakingPoint          int                      `json:"breakingPoint"`
	RecoveryTime           time.Duration            `json:"recoveryTime"`
	ResourceExhaustionPoint *ResourceExhaustionPoint `json:"resourceExhaustionPoint,omitempty"`
	RecoveryMetrics        *RecoveryMetrics         `json:"recoveryMetrics,omitempty"`
	MaxStressLevel         float64                  `json:"maxStressLevel"`
	SystemStability        string                   `json:"systemStability"`
}

// LatencyTestResults represents results from latency testing
type LatencyTestResults struct {
	AverageLatency    time.Duration `json:"averageLatency"`
	MedianLatency     time.Duration `json:"medianLatency"`
	P95Latency        time.Duration `json:"p95Latency"`
	P99Latency        time.Duration `json:"p99Latency"`
	MaxLatency        time.Duration `json:"maxLatency"`
	MinLatency        time.Duration `json:"minLatency"`
	LatencyDistribution []LatencyBucket `json:"latencyDistribution"` // Note: LatencyBucket is in performance_testing.go, not duplicated here
}

// ResourceExhaustionPoint represents the point at which resources are exhausted
type ResourceExhaustionPoint struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	NetworkIO   float64 `json:"networkIO"`
	DiskIO      float64 `json:"diskIO"`
	Connections int     `json:"connections"`
}

// RecoveryMetrics represents recovery metrics after failure
type RecoveryMetrics struct {
	RecoveryTime         time.Duration `json:"recoveryTime"`
	SuccessfulRecoveries int           `json:"successfulRecoveries"`
	FailedRecoveries     int           `json:"failedRecoveries"`
	RecoveryRate         float64       `json:"recoveryRate"`
}

// Additional types needed by performance_test_runner.go - note: ValidationResults, RequirementCompliance, and DetailedMetrics are in performance_test_runner.go, not duplicated here
type ThroughputResults struct {
	MaxIndicationsPerSecond int     `json:"maxIndicationsPerSecond"`
	AverageIPS             int     `json:"averageIPS"`
	PeakThroughput         float64 `json:"peakThroughput"`
}

type LatencyResults struct {
	EndToEndLatencyMs *LatencyMetrics `json:"endToEndLatencyMs"`
	SubscriptionLatency *LatencyMetrics `json:"subscriptionLatency"`
	IndicationProcessing *LatencyMetrics `json:"indicationProcessing"`
}

type StabilityResults struct {
	TestDurationHours     float64           `json:"testDurationHours"`
	MemoryLeakDetected    bool              `json:"memoryLeakDetected"`
	MemoryUsageOverTime   []MemoryUsageSample `json:"memoryUsageOverTime"`
	SystemStability       string            `json:"systemStability"`
	ResourceLeaks         []ResourceLeak    `json:"resourceLeaks"`
}

type MemoryUsageSample struct {
	Timestamp time.Time `json:"timestamp"`
	UsageMB   float64   `json:"usageMB"`
	HeapMB    float64   `json:"heapMB"`
}

type ResourceLeak struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	DetectedAt  time.Time `json:"detectedAt"`
	Severity    string    `json:"severity"`
}

// ResourceEfficiencyMetrics represents resource efficiency metrics
type ResourceEfficiencyMetrics struct {
	CPUEfficiency      float64 `json:"cpuEfficiency"`
	MemoryEfficiency   float64 `json:"memoryEfficiency"`
	NetworkEfficiency  float64 `json:"networkEfficiency"`
	OverallEfficiency  float64 `json:"overallEfficiency"`
}

// Missing types needed for enhanced dashboard API - WebSocket Pool - note: WebSocketConnection is in production_dashboard_api.go, not duplicated here
type WebSocketPool struct {
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex
	stats       WSManagerStats
	config      WebSocketPoolConfig
}

type WebSocketPoolConfig struct {
	MaxConnections   int
	PingInterval     time.Duration
	WriteTimeout     time.Duration
	ReadTimeout      time.Duration
	EnableHeartbeat  bool
}

type BroadcastMessage struct {
	Type      string                 `json:"type"`
	Channel   string                 `json:"channel"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
	TargetIDs []string               `json:"targetIds,omitempty"`
}

type WSManagerStats struct {
	ActiveConnections    int64     `json:"activeConnections"`
	TotalConnections     int64     `json:"totalConnections"`
	MessagesSent         int64     `json:"messagesSent"`
	MessagesReceived     int64     `json:"messagesReceived"`
	BytesSent           int64     `json:"bytesSent"`
	BytesReceived       int64     `json:"bytesReceived"`
	LastUpdate          time.Time `json:"lastUpdate"`
	ConnectionsByChannel map[string]int `json:"connectionsByChannel"`
}

type BackendHealthChecker struct {
	backends     map[string]*Backend
	mu           sync.RWMutex
	checkInterval time.Duration
	timeout       time.Duration
	isRunning     bool
	stopChan      chan struct{}
	healthChecks  map[string]*HealthCheckResult
}

// Note: HealthCheckResult is in xapp_health_monitor.go, not duplicated here

type RequestRouter struct {
	routes      map[string]*Route
	middleware  []MiddlewareFunc
	fallback    http.Handler
	stats       RouterStats
	mu          sync.RWMutex
}

// Route is defined in routing_models.go, not duplicated here

type MiddlewareFunc func(http.Handler) http.Handler

type RouterStats struct {
	TotalRequests     int64                    `json:"totalRequests"`
	RequestsByRoute   map[string]int64         `json:"requestsByRoute"`
	RequestsByMethod  map[string]int64         `json:"requestsByMethod"`
	AverageLatency    time.Duration            `json:"averageLatency"`
	ErrorCount        int64                    `json:"errorCount"`
	LastUpdate        time.Time                `json:"lastUpdate"`
}

type StickySessionManager struct {
	sessions map[string]*SessionInfo
	mu       sync.RWMutex
	ttl      time.Duration
	cleanup  *time.Ticker
}

type SessionInfo struct {
	SessionID   string                 `json:"sessionId"`
	BackendID   string                 `json:"backendId"`
	UserID      string                 `json:"userId"`
	CreatedAt   time.Time              `json:"createdAt"`
	LastAccess  time.Time              `json:"lastAccess"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Note: AuditEvent is in security_compliance_test.go, not duplicated here

// Note: WorkItem and WorkResult are defined as concrete types but used as interfaces in zero_copy_components.go
// This creates a conflict that would need refactoring to resolve properly