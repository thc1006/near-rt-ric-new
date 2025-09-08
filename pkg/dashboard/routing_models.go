/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"time"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Route represents a routing rule in the system
type Route struct {
	ID             string       `json:"id"`
	SourceXApp     string       `json:"sourceXapp"`
	TargetXApp     string       `json:"targetXapp"`
	MessageType    uint32       `json:"messageType"`
	SubscriptionID string       `json:"subscriptionId"`
	Policy         *RoutePolicy `json:"policy,omitempty"`
	CreatedAt      time.Time    `json:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

// RoutePolicy defines routing policy parameters
type RoutePolicy struct {
	Type       string            `json:"type"` // "round_robin", "priority", "load_balanced"
	Parameters map[string]string `json:"parameters"`
}

// RouteEntry type is now defined in types.go to avoid redeclaration

// RoutingTable represents the complete routing table
type RoutingTable struct {
	Entries     []RouteEntry `json:"entries"`
	Version     uint32       `json:"version"`
	LastUpdated time.Time    `json:"lastUpdated"`
}

// XAppInfo represents information about a registered xApp
type XAppInfo struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Namespace     string            `json:"namespace"`
	Endpoints     []string          `json:"endpoints"`
	Config        map[string]string `json:"config"`
	Status        string            `json:"status"`
	RegisteredAt  time.Time         `json:"registeredAt"`
	LastHeartbeat time.Time         `json:"lastHeartbeat"`
}

// RoutingStats represents routing manager statistics
type RoutingStats struct {
	TotalRoutes         uint32            `json:"totalRoutes"`
	ActiveXApps         uint32            `json:"activeXapps"`
	RoutesByMessageType map[string]uint32 `json:"routesByMessageType"`
	XAppsByStatus       map[string]uint32 `json:"xappsByStatus"`
	MessagesRouted      uint64            `json:"messagesRouted"`
	RoutingErrors       uint64            `json:"routingErrors"`
	LastUpdated         time.Time         `json:"lastUpdated"`
}

// RoutingHealth represents routing manager health status
type RoutingHealth struct {
	IsHealthy         bool      `json:"isHealthy"`
	StatusMessage     string    `json:"statusMessage"`
	ActiveConnections uint32    `json:"activeConnections"`
	FailedRoutes      uint32    `json:"failedRoutes"`
	LastHealthCheck   time.Time `json:"lastHealthCheck"`
}

// Request/Response types for routing operations

// RouteResponse represents the response from creating a route
type RouteResponse struct {
	RouteID string `json:"routeId"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RouteListResponse represents a list of routes
type RouteListResponse struct {
	Routes []Route `json:"routes"`
	Total  uint32  `json:"total"`
}

// RouteFilter represents filter criteria for route queries
type RouteFilter struct {
	SourceXApp  string  `json:"sourceXapp,omitempty"`
	TargetXApp  string  `json:"targetXapp,omitempty"`
	MessageType *uint32 `json:"messageType,omitempty"`
	Limit       *uint32 `json:"limit,omitempty"`
	Offset      *uint32 `json:"offset,omitempty"`
}

// XAppListResponse represents a list of xApps
type XAppListResponse struct {
	XApps []XAppInfo `json:"xapps"`
	Total uint32     `json:"total"`
}

// XAppRegistrationRequest represents a request to register an xApp
type XAppRegistrationRequest struct {
	XAppInfo *XAppInfo `json:"xappInfo"`
}

// XAppRegistrationResponse represents the response from xApp registration
type XAppRegistrationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RoutingTableUpdateRequest represents a request to update the routing table
type RoutingTableUpdateRequest struct {
	RoutingTable *RoutingTable `json:"routingTable"`
}

// RoutingTableUpdateResponse represents the response from routing table update
type RoutingTableUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Helper functions for protobuf conversion

// ToTimestampPB converts time.Time to protobuf timestamp
func ToTimestampPB(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// FromTimestampPB converts protobuf timestamp to time.Time
func FromTimestampPB(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}