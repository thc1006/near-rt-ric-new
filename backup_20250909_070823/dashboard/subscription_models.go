/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"time"
)

// SubscriptionID represents a unique subscription identifier
type SubscriptionID string

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "ACTIVE"
	SubscriptionStatusPending   SubscriptionStatus = "PENDING"
	SubscriptionStatusFailed    SubscriptionStatus = "FAILED"
	SubscriptionStatusDeleted   SubscriptionStatus = "DELETED"
	SubscriptionStatusCompleted SubscriptionStatus = "COMPLETED"
)

// EventTriggerType represents the type of event trigger
type EventTriggerType string

const (
	EventTriggerTypePeriodic EventTriggerType = "PERIODIC"
	EventTriggerTypeOnChange EventTriggerType = "ON_CHANGE"
	EventTriggerTypeOnDemand EventTriggerType = "ON_DEMAND"
)

// ActionType type is now defined in types.go to avoid redeclaration

// EventTrigger represents the event trigger definition
type EventTrigger struct {
	Type       EventTriggerType `json:"type"`
	Definition []byte           `json:"definition"`
	Period     *time.Duration   `json:"period,omitempty"`
}

// SubsequentAction represents a subsequent action to be taken
type SubsequentAction struct {
	Type       ActionType `json:"type"`
	TimeToWait uint32     `json:"timeToWait"`
}

// Action represents an action in a subscription
type Action struct {
	ID               uint32            `json:"id"`
	Type             ActionType        `json:"type"`
	Definition       []byte            `json:"definition"`
	SubsequentAction *SubsequentAction `json:"subsequentAction,omitempty"`
}

// Subscription represents a subscription in the system
type Subscription struct {
	ID            SubscriptionID     `json:"id"`
	E2NodeID      string             `json:"e2NodeId"`
	XAppID        string             `json:"xappId"`
	RANFunctionID uint32             `json:"ranFunctionId"`
	EventTrigger  EventTrigger       `json:"eventTrigger"`
	Actions       []Action           `json:"actions"`
	Status        SubscriptionStatus `json:"status"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
	ErrorMessage  string             `json:"errorMessage,omitempty"`
}

// SubscriptionRequest represents a request to create a subscription
type SubscriptionRequest struct {
	E2NodeID      string       `json:"e2NodeId"`
	XAppID        string       `json:"xappId"`
	RANFunctionID uint32       `json:"ranFunctionId"`
	EventTrigger  EventTrigger `json:"eventTrigger"`
	Actions       []Action     `json:"actions"`
}

// SubscriptionResponse represents the response to a subscription request
type SubscriptionResponse struct {
	SubscriptionID SubscriptionID `json:"subscriptionId"`
	Status         string         `json:"status"`
	Message        string         `json:"message,omitempty"`
}

// SubscriptionListResponse represents the response for listing subscriptions
type SubscriptionListResponse struct {
	Subscriptions []Subscription `json:"subscriptions"`
	Total         uint32         `json:"total"`
}

// SubscriptionStats represents statistics from Subscription Manager
type SubscriptionStats struct {
	ActiveSubscriptions   uint32            `json:"activeSubscriptions"`
	TotalSubscriptions    uint32            `json:"totalSubscriptions"`
	FailedSubscriptions   uint32            `json:"failedSubscriptions"`
	TotalIndications      uint64            `json:"totalIndications"`
	IndicationsPerSecond  float64           `json:"indicationsPerSecond"`
	SubscriptionsByStatus map[string]uint32 `json:"subscriptionsByStatus"`
	SubscriptionsByXApp   map[string]uint32 `json:"subscriptionsByXApp"`
	LastUpdated           time.Time         `json:"lastUpdated"`
}

// Indication type is now defined in types.go to avoid redeclaration

// SubscriptionUpdate represents an update to a subscription
type SubscriptionUpdate struct {
	SubscriptionID SubscriptionID     `json:"subscriptionId"`
	Status         SubscriptionStatus `json:"status"`
	EventTrigger   *EventTrigger      `json:"eventTrigger,omitempty"`
	Actions        []Action           `json:"actions,omitempty"`
}

// SubscriptionFilter represents filters for subscription queries
type SubscriptionFilter struct {
	E2NodeID      string             `json:"e2NodeId,omitempty"`
	XAppID        string             `json:"xappId,omitempty"`
	RANFunctionID *uint32            `json:"ranFunctionId,omitempty"`
	Status        SubscriptionStatus `json:"status,omitempty"`
	Limit         uint32             `json:"limit,omitempty"`
	Offset        uint32             `json:"offset,omitempty"`
}