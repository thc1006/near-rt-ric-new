/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"time"
)

// PolicyTypeID represents a policy type identifier
type PolicyTypeID string

// PolicyInstanceID represents a policy instance identifier
type PolicyInstanceID string

// PolicyType represents an A1 policy type
type PolicyType struct {
	ID          PolicyTypeID    `json:"policy_type_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"policy_type_schema"`
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

// PolicyStatus represents the status of a policy instance
type PolicyStatus struct {
	Status     string    `json:"status"`
	Reason     string    `json:"reason,omitempty"`
	LastUpdate time.Time `json:"last_update"`
}

// PolicyTypeListResponse represents the response for listing policy types
type PolicyTypeListResponse struct {
	PolicyTypes []PolicyType `json:"policy_types"`
	Total       uint32       `json:"total"`
}

// PolicyInstanceListResponse represents the response for listing policy instances
type PolicyInstanceListResponse struct {
	PolicyInstances []PolicyInstance `json:"policy_instances"`
	Total           uint32           `json:"total"`
}

// PolicyTypeRequest represents a request to create or update a policy type
type PolicyTypeRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"policy_type_schema"`
}

// PolicyInstanceRequest represents a request to create or update a policy instance
type PolicyInstanceRequest struct {
	Policy json.RawMessage `json:"policy"`
}

// PolicyInstanceUpdate represents an update to a policy instance
type PolicyInstanceUpdate struct {
	PolicyInstanceID PolicyInstanceID `json:"policy_instance_id"`
	Policy           json.RawMessage  `json:"policy"`
}

// PolicyFilter represents filtering options for policy queries
type PolicyFilter struct {
	PolicyTypeID PolicyTypeID `json:"policy_type_id,omitempty"`
	Status       string       `json:"status,omitempty"`
	Limit        uint32       `json:"limit,omitempty"`
	Offset       uint32       `json:"offset,omitempty"`
}

// A1Stats represents statistics from A1 Mediator
type A1Stats struct {
	PolicyTypesByStatus    map[string]uint32 `json:"policy_types_by_status"`
	PolicyInstancesByType  map[string]uint32 `json:"policy_instances_by_type"`
	PolicyInstancesByStatus map[string]uint32 `json:"policy_instances_by_status"`
	TotalPolicyTypes       uint32            `json:"total_policy_types"`
	TotalPolicyInstances   uint32            `json:"total_policy_instances"`
	LastUpdated            time.Time         `json:"last_updated"`
}

// A1Health represents health information from A1 Mediator
type A1Health struct {
	IsHealthy       bool      `json:"is_healthy"`
	StatusMessage   string    `json:"status_message,omitempty"`
	LastHealthCheck time.Time `json:"last_health_check"`
	Version         string    `json:"version,omitempty"`
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

// PolicyValidationError represents a policy validation error
type PolicyValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// PolicyValidationResult represents the result of policy validation
type PolicyValidationResult struct {
	IsValid bool                    `json:"is_valid"`
	Errors  []PolicyValidationError `json:"errors,omitempty"`
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

// A1ErrorResponse represents an error response from A1 Mediator
type A1ErrorResponse struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Type   string `json:"type,omitempty"`
}

// Error implements the error interface for A1ErrorResponse
func (e A1ErrorResponse) Error() string {
	return e.Detail
}

// PolicyTypeStatus represents the status of a policy type
type PolicyTypeStatus string

const (
	PolicyTypeStatusActive   PolicyTypeStatus = "ACTIVE"
	PolicyTypeStatusInactive PolicyTypeStatus = "INACTIVE"
	PolicyTypeStatusDeleted  PolicyTypeStatus = "DELETED"
)

// PolicyInstanceStatus represents the status of a policy instance
type PolicyInstanceStatus string

const (
	PolicyInstanceStatusActive    PolicyInstanceStatus = "ACTIVE"
	PolicyInstanceStatusInactive  PolicyInstanceStatus = "INACTIVE"
	PolicyInstanceStatusDeleted   PolicyInstanceStatus = "DELETED"
	PolicyInstanceStatusPending   PolicyInstanceStatus = "PENDING"
	PolicyInstanceStatusError     PolicyInstanceStatus = "ERROR"
)

// PolicyConflictType represents the type of policy conflict
type PolicyConflictType string

const (
	PolicyConflictTypeResource   PolicyConflictType = "RESOURCE"
	PolicyConflictTypeParameter  PolicyConflictType = "PARAMETER"
	PolicyConflictTypePriority   PolicyConflictType = "PRIORITY"
	PolicyConflictTypeExclusive  PolicyConflictType = "EXCLUSIVE"
)

// PolicyDistributionStatusType represents the distribution status type
type PolicyDistributionStatusType string

const (
	PolicyDistributionStatusPending    PolicyDistributionStatusType = "PENDING"
	PolicyDistributionStatusDeployed   PolicyDistributionStatusType = "DEPLOYED"
	PolicyDistributionStatusFailed     PolicyDistributionStatusType = "FAILED"
	PolicyDistributionStatusWithdrawn  PolicyDistributionStatusType = "WITHDRAWN"
)

// PolicyComplianceStatusType represents the compliance status type
type PolicyComplianceStatusType string

const (
	PolicyComplianceStatusCompliant    PolicyComplianceStatusType = "COMPLIANT"
	PolicyComplianceStatusNonCompliant PolicyComplianceStatusType = "NON_COMPLIANT"
	PolicyComplianceStatusUnknown      PolicyComplianceStatusType = "UNKNOWN"
)