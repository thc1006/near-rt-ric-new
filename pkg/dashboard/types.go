// Package dashboard provides centralized type definitions for the O-RAN Near-RT RIC Dashboard
// This file contains all shared types to avoid redeclaration errors
package dashboard

import (
	"context"
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
	Name          string        `json:"name"`
	State         CircuitState  `json:"state"`
	FailureCount  int          `json:"failureCount"`
	Threshold     int          `json:"threshold"`
	Timeout       time.Duration `json:"timeout"`
	LastFailTime  time.Time     `json:"lastFailTime"`
	Mutex         sync.RWMutex  `json:"-"`
}

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	CircuitStateClosed    CircuitState = "closed"
	CircuitStateOpen      CircuitState = "open"
	CircuitStateHalfOpen  CircuitState = "half-open"
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