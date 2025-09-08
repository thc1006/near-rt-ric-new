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

// E2Node type definition moved to types.go to avoid redeclaration

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

// GlobalRICID type definition moved to types.go to avoid redeclaration

// =====================================================================
// E2AP Message Types
// =====================================================================

// E2APMessage type definition moved to types.go to avoid redeclaration

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

// E2NodeComponentConfigUpdateAck type definition moved to types.go to avoid redeclaration

// =====================================================================
// SIMD Operation Types
// =====================================================================

// SIMDOperation type definition moved to types.go to avoid redeclaration

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

// ResourceUsage type definition moved to types.go to avoid redeclaration

// =====================================================================
// Latency Tracker Types
// =====================================================================

// LatencyTracker type definition moved to types.go to avoid redeclaration

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

// RoundRobin and other load balancing algorithm constants moved to types.go to avoid redeclaration
const (
	WeightedRoundRobin LoadBalancingAlgorithm = iota + 1
	LeastConnections
	WeightedLeastConnections
	ConsistentHashing
	ResourceBased
	LatencyBased
)

// =====================================================================
// Health Checker Types
// =====================================================================

// HealthChecker interface definition moved to types.go to avoid redeclaration

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