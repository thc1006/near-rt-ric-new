/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"net"
	"sync"
	"time"
	"unsafe"
)

// =====================================================================
// E2 Node and Related Types
// =====================================================================

// E2Node represents an E2 node in the system (from e2_models.go - most comprehensive)
// E2Node type moved to types.go to avoid redeclaration
// type E2Node struct {
	ID                string                 `json:"id"`
	GlobalE2NodeID    GlobalE2NodeID         `json:"globalE2NodeId"`
	ConnectionStatus  E2NodeConnectionStatus `json:"connectionStatus"`
	SetupRequest      *E2SetupRequest        `json:"setupRequest,omitempty"`
	ServiceModels     []ServiceModel         `json:"serviceModels"`
	RANFunctions      []RANFunction          `json:"ranFunctions"`
	LastUpdate        time.Time              `json:"lastUpdate"`
	Subscriptions     []SubscriptionInfo     `json:"subscriptions"`
	IPAddress         string                 `json:"ipAddress"`
	Port              uint32                 `json:"port"`
	AssociationID     string                 `json:"associationId"`
	SCTPStreams       uint32                 `json:"sctpStreams"`
	Address           string                 `json:"address,omitempty"`
	Metrics           NodeMetrics            `json:"metrics,omitempty"`
	Connection        *OptimizedConnection   `json:"connection,omitempty"`
	mu                sync.RWMutex           `json:"-"`
}

// NodeMetrics represents performance metrics for an E2 node
type NodeMetrics struct {
	RequestCount     uint64        `json:"requestCount"`
	ErrorCount       uint64        `json:"errorCount"`
	AverageLatency   time.Duration `json:"averageLatency"`
	ThroughputIPS    uint64        `json:"throughputIPS"`
	LastHeartbeat    time.Time     `json:"lastHeartbeat"`
	ConnectionUptime time.Duration `json:"connectionUptime"`
}

// OptimizedConnection represents a high-performance connection
type OptimizedConnection struct {
	conn            net.Conn
	readBuffer      []byte
	writeBuffer     []byte
	lastActivity    time.Time
	stats           ConnectionStats
	mu              sync.RWMutex
}

// ConnectionStats tracks connection statistics
type ConnectionStats struct {
	BytesReceived    uint64    `json:"bytesReceived"`
	BytesSent        uint64    `json:"bytesSent"`
	MessagesReceived uint64    `json:"messagesReceived"`
	MessagesSent     uint64    `json:"messagesSent"`
	ErrorCount       uint64    `json:"errorCount"`
	LastActivity     time.Time `json:"lastActivity"`
}

// NodeStatus represents the status of an E2 node
type NodeStatus int

const (
	NodeStatusConnected NodeStatus = iota
	NodeStatusDisconnected
	NodeStatusConnecting
	NodeStatusError
)

// =====================================================================
// Global RIC ID Type
// =====================================================================

// GlobalRICID represents the global RIC identifier (from e2t_client.go - most comprehensive)
type GlobalRICID struct {
	PlmnID string `json:"plmnId"`
	RicID  string `json:"ricId"`
}

// =====================================================================
// E2AP Message Types
// =====================================================================

// E2APMessage represents an E2AP protocol message (from e2t_client.go - most comprehensive)
type E2APMessage struct {
	MessageType    string    `json:"messageType"`
	ProcedureCode  uint8     `json:"procedureCode"`
	Criticality    string    `json:"criticality"`
	TransactionID  uint32    `json:"transactionId"`
	Payload        []byte    `json:"payload"`
	Timestamp      time.Time `json:"timestamp"`
	SourceAddress  string    `json:"sourceAddress"`
	DestAddress    string    `json:"destAddress"`
	AssociationID  string    `json:"associationId"`
	PDUType        uint8     `json:"pduType,omitempty"`
	Value          map[string]interface{} `json:"value,omitempty"`
	RawData        []byte    `json:"rawData,omitempty"`
}

// E2MessageType represents different E2 message types
type E2MessageType int

const (
	E2MessageTypeSetup E2MessageType = iota
	E2MessageTypeIndication
	E2MessageTypeControl
	E2MessageTypeSubscription
	E2MessageTypeConfiguration
)

// =====================================================================
// E2 Node Component Config Update Acknowledgement
// =====================================================================

// E2NodeComponentConfigUpdateAck represents acknowledgment of component config update
type E2NodeComponentConfigUpdateAck struct {
	E2NodeComponentInterfaceType   E2NodeComponentInterfaceType    `json:"e2NodeComponentInterfaceType"`
	E2NodeComponentID              E2NodeComponentID               `json:"e2NodeComponentId"`
	E2NodeComponentConfigAck       E2NodeComponentConfigAckType    `json:"e2NodeComponentConfigAck"`
	E2NodeComponentConfigUpdateAck *E2NodeComponentConfigUpdateAck `json:"e2NodeComponentConfigUpdateAck,omitempty"`
	UpdateOutcome                  string                          `json:"updateOutcome,omitempty"`
}

// =====================================================================
// SIMD Operation Types
// =====================================================================

// SIMDOperation represents a SIMD-optimized operation (from high_performance_processor.go - most comprehensive)
type SIMDOperation struct {
	Name            string
	Function        unsafe.Pointer
	InputTypes      []SIMDDataType
	OutputType      SIMDDataType
	VectorWidth     int
	Fallback        unsafe.Pointer
}

// SIMDDataType represents data types for SIMD operations
type SIMDDataType int

const (
	SIMDInt8 SIMDDataType = iota
	SIMDInt16
	SIMDInt32
	SIMDInt64
	SIMDFloat32
	SIMDFloat64
	SIMDUint8
	SIMDUint16
	SIMDUint32
	SIMDUint64
)

// =====================================================================
// Resource Usage Types
// =====================================================================

// ResourceUsage represents current resource usage (from horizontal_scaler.go - most comprehensive)
type ResourceUsage struct {
	CPUUsage       float64       `json:"cpuUsage"`
	MemoryUsage    float64       `json:"memoryUsage"`
	NetworkIn      int64         `json:"networkIn"`
	NetworkOut     int64         `json:"networkOut"`
	RequestRate    float64       `json:"requestRate"`
	ErrorRate      float64       `json:"errorRate"`
	AverageLatency time.Duration `json:"averageLatency"`
	LastUpdated    time.Time     `json:"lastUpdated"`
	NetworkUsage   float64       `json:"networkUsage,omitempty"`
	DiskUsage      float64       `json:"diskUsage,omitempty"`
}

// =====================================================================
// Latency Tracker Types
// =====================================================================

// LatencyTracker tracks latency measurements across different operations (from latency_testing.go - most comprehensive)
type LatencyTracker struct {
	e2SetupLatencies        []float64
	subscriptionLatencies   []float64
	indicationLatencies     []float64
	controlLatencies        []float64
	endToEndLatencies       []float64
	latencyDistribution     map[float64]int64 // bucket -> count
	activeOperations        map[string]*LatencyMeasurement
	samples                 []time.Duration
	sampleIndex             int
	sampleCount             uint64
	histogram               map[time.Duration]uint64
	percentiles             LatencyPercentiles
	mu                      sync.RWMutex
}

// LatencyMeasurement tracks a single operation's latency
type LatencyMeasurement struct {
	OperationID   string
	OperationType string
	StartTime     time.Time
	EndTime       *time.Time
	Latency       *time.Duration
	Metadata      map[string]interface{}
}

// LatencyPercentiles stores calculated percentiles
type LatencyPercentiles struct {
	P50     time.Duration
	P90     time.Duration
	P95     time.Duration
	P99     time.Duration
	P999    time.Duration
	P9999   time.Duration
}

// =====================================================================
// Load Balancing Algorithm Types
// =====================================================================

// LoadBalancingAlgorithm defines load balancing strategies
type LoadBalancingAlgorithm int

// RoundRobin represents the round-robin load balancing algorithm
const (
	RoundRobin LoadBalancingAlgorithm = iota
	WeightedRoundRobin
	LeastConnections
	WeightedLeastConnections
	ConsistentHashing
	ResourceBased
	LatencyBased
)

// =====================================================================
// Health Checker Types
// =====================================================================

// HealthChecker defines the interface for health checkers (from graceful_degradation.go - interface definition)
type HealthChecker interface {
	CheckHealth(ctx context.Context) (*HealthCheckResult, error)
	GetServiceName() string
	AddBackend(backend *Backend)
	RemoveBackend(backendID string)
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Healthy      bool                   `json:"healthy"`
	ResponseTime time.Duration          `json:"responseTime"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Backend represents a backend service instance
type Backend struct {
	ID           string
	Address      string
	Port         int
	Weight       int
	MaxConns     int
	CurrentConns int64
	IsHealthy    int32 // atomic boolean
	LastCheck    time.Time
	ResponseTime time.Duration
	ErrorRate    float64
	CPUUsage     float64
	MemoryUsage  float64
	mu           sync.RWMutex
}


// =====================================================================
// Message Processing Types
// =====================================================================

// MessageType represents different types of messages
type MessageType int

const (
	MessageTypeE2AP MessageType = iota
	MessageTypeA1
	MessageTypeO1
)

// OptimizedMessage represents an optimized message for processing
type OptimizedMessage struct {
	Type     MessageType
	Data     []byte
	Size     int
	Metadata map[string]interface{}
}

// ProcessedMessage represents the result of message processing
type ProcessedMessage struct {
	Original       *OptimizedMessage
	Result         []byte
	ResultSize     int
	ProcessingTime time.Duration
	Success        bool
	Metadata       map[string]interface{}
}

// ProcessingResult represents the result of processing operations
type ProcessingResult struct {
	Success        bool
	Data           []byte
	ProcessingTime time.Duration
	Metadata       map[string]interface{}
}

// =====================================================================
// Context Helper Functions
// =====================================================================

// GetCorrelationID gets correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value("correlation_id").(string); ok {
		return id
	}
	return ""
}