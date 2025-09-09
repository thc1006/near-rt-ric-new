/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"time"
)

// E2NodeConnectionStatus type is now defined in types.go to avoid redeclaration

// E2NodeType represents the type of E2 node
type E2NodeType string

const (
	E2NodeTypeGNB   E2NodeType = "GNB"
	E2NodeTypeENB   E2NodeType = "ENB"
	E2NodeTypeOCU   E2NodeType = "O_CU"
	E2NodeTypeODU   E2NodeType = "O_DU"
	E2NodeTypeOCUCP E2NodeType = "O_CU_CP"
	E2NodeTypeOCUUP E2NodeType = "O_CU_UP"
)

// GlobalE2NodeID represents the global E2 node identifier
// GlobalE2NodeID type is now defined in types.go to avoid redeclaration

// RANFunction type is now defined in types.go to avoid redeclaration

// ServiceModel type is now defined in types.go to avoid redeclaration

// E2SetupRequest represents the E2 setup request information
type E2SetupRequest struct {
	TransactionID   uint32        `json:"transactionId"`
	GlobalE2NodeID  GlobalE2NodeID `json:"globalE2NodeId"`
	RANFunctions    []RANFunction `json:"ranFunctions"`
	E2NodeComponentConfigAddList []E2NodeComponentConfig `json:"e2NodeComponentConfigAddList"`
}

// E2NodeComponentConfig represents E2 node component configuration
type E2NodeComponentConfig struct {
	E2NodeComponentInterfaceType string `json:"e2NodeComponentInterfaceType"`
	E2NodeComponentID            string `json:"e2NodeComponentId"`
	E2NodeComponentConfiguration []byte `json:"e2NodeComponentConfiguration"`
}


// SubscriptionInfo represents subscription information for an E2 node
type SubscriptionInfo struct {
	ID            string    `json:"id"`
	XAppID        string    `json:"xappId"`
	RANFunctionID uint32    `json:"ranFunctionId"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

// E2NodeHealth represents health information for an E2 node
type E2NodeHealth struct {
	NodeID           string    `json:"nodeId"`
	IsHealthy        bool      `json:"isHealthy"`
	LastHealthCheck  time.Time `json:"lastHealthCheck"`
	HealthCheckCount uint64    `json:"healthCheckCount"`
	ErrorCount       uint64    `json:"errorCount"`
	LastError        string    `json:"lastError,omitempty"`
}

// E2ManagerStats represents statistics from E2 Manager
type E2ManagerStats struct {
	ConnectedNodes     uint32            `json:"connectedNodes"`
	TotalNodes         uint32            `json:"totalNodes"`
	ActiveConnections  uint32            `json:"activeConnections"`
	SetupRequests      uint64            `json:"setupRequests"`
	SetupFailures      uint64            `json:"setupFailures"`
	ConfigUpdates      uint64            `json:"configUpdates"`
	NodesByType        map[string]uint32 `json:"nodesByType"`
	NodesByStatus      map[string]uint32 `json:"nodesByStatus"`
	LastUpdated        time.Time         `json:"lastUpdated"`
}

// E2NodeListResponse represents the response for listing E2 nodes
type E2NodeListResponse struct {
	Nodes []E2Node `json:"nodes"`
	Total uint32   `json:"total"`
}

// E2NodeConfigurationUpdate type is now defined in types.go to avoid redeclaration

// E2ConnectionUpdate represents a connection update event
type E2ConnectionUpdate struct {
	NodeID    string                 `json:"nodeId"`
	Status    E2NodeConnectionStatus `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Reason    string                 `json:"reason,omitempty"`
}