/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"time"
)

// O1ManagedObjectID represents a managed object identifier
type O1ManagedObjectID string

// O1ConfigurationID represents a configuration identifier
type O1ConfigurationID string

// O1AlarmID represents an alarm identifier
type O1AlarmID string

// O1KPIID represents a KPI identifier
type O1KPIID string

// O1ManagedObject represents an O1 managed object
type O1ManagedObject struct {
	ID          O1ManagedObjectID `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Attributes  json.RawMessage   `json:"attributes"`
	State       string            `json:"state"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// O1Configuration represents an O1 configuration
type O1Configuration struct {
	ID          O1ConfigurationID `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Config      json.RawMessage   `json:"config"`
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// O1Alarm represents an O1 alarm
type O1Alarm struct {
	ID               O1AlarmID   `json:"id"`
	ManagedObjectID  string      `json:"managed_object_id"`
	AlarmType        string      `json:"alarm_type"`
	Severity         string      `json:"severity"`
	ProbableCause    string      `json:"probable_cause"`
	SpecificProblem  string      `json:"specific_problem"`
	AdditionalText   string      `json:"additional_text,omitempty"`
	AlarmState       string      `json:"alarm_state"`
	EventTime        time.Time   `json:"event_time"`
	NotificationID   uint64      `json:"notification_id"`
	CorrelatedAlarms []O1AlarmID `json:"correlated_alarms,omitempty"`
	AckState         string      `json:"ack_state"`
	AckTime          *time.Time  `json:"ack_time,omitempty"`
	AckUser          string      `json:"ack_user,omitempty"`
	ClearTime        *time.Time  `json:"clear_time,omitempty"`
}

// O1KPI represents an O1 Key Performance Indicator
type O1KPI struct {
	ID              O1KPIID         `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	MeasurementType string          `json:"measurement_type"`
	Unit            string          `json:"unit"`
	Value           float64         `json:"value"`
	Threshold       *O1KPIThreshold `json:"threshold,omitempty"`
	Timestamp       time.Time       `json:"timestamp"`
	ManagedObjectID string          `json:"managed_object_id"`
}

// O1KPIThreshold represents KPI threshold configuration
type O1KPIThreshold struct {
	WarningMin  *float64 `json:"warning_min,omitempty"`
	WarningMax  *float64 `json:"warning_max,omitempty"`
	CriticalMin *float64 `json:"critical_min,omitempty"`
	CriticalMax *float64 `json:"critical_max,omitempty"`
}

// O1Health represents health information from O1 Mediator
type O1Health struct {
	IsHealthy       bool      `json:"is_healthy"`
	StatusMessage   string    `json:"status_message,omitempty"`
	LastHealthCheck time.Time `json:"last_health_check"`
	Version         string    `json:"version,omitempty"`
	Capabilities    []string  `json:"capabilities,omitempty"`
}

// O1Stats represents statistics from O1 Mediator
type O1Stats struct {
	ManagedObjectsByType    map[string]uint32 `json:"managed_objects_by_type"`
	ConfigurationsByStatus  map[string]uint32 `json:"configurations_by_status"`
	AlarmsBySeverity        map[string]uint32 `json:"alarms_by_severity"`
	KPIsByType              map[string]uint32 `json:"kpis_by_type"`
	TotalManagedObjects     uint32            `json:"total_managed_objects"`
	TotalConfigurations     uint32            `json:"total_configurations"`
	TotalActiveAlarms       uint32            `json:"total_active_alarms"`
	TotalKPIs               uint32            `json:"total_kpis"`
	LastUpdated             time.Time         `json:"last_updated"`
}

// O1ManagedObjectListResponse represents the response for listing managed objects
type O1ManagedObjectListResponse struct {
	ManagedObjects []O1ManagedObject `json:"managed_objects"`
	Total          uint32            `json:"total"`
}

// O1ConfigurationListResponse represents the response for listing configurations
type O1ConfigurationListResponse struct {
	Configurations []O1Configuration `json:"configurations"`
	Total          uint32            `json:"total"`
}

// O1AlarmListResponse represents the response for listing alarms
type O1AlarmListResponse struct {
	Alarms []O1Alarm `json:"alarms"`
	Total  uint32    `json:"total"`
}

// O1KPIListResponse represents the response for listing KPIs
type O1KPIListResponse struct {
	KPIs  []O1KPI `json:"kpis"`
	Total uint32  `json:"total"`
}

// O1ConfigurationRequest represents a request to create or update a configuration
type O1ConfigurationRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
}

// O1ConfigurationUpdate represents an update to a configuration
type O1ConfigurationUpdate struct {
	ConfigurationID O1ConfigurationID `json:"configuration_id"`
	Config          json.RawMessage   `json:"config"`
	Description     string            `json:"description,omitempty"`
}

// O1AlarmAcknowledgment represents an alarm acknowledgment request
type O1AlarmAcknowledgment struct {
	AlarmID O1AlarmID `json:"alarm_id"`
	User    string    `json:"user"`
	Comment string    `json:"comment,omitempty"`
}

// O1Filter represents filtering options for O1 queries
type O1Filter struct {
	Type     string    `json:"type,omitempty"`
	Status   string    `json:"status,omitempty"`
	Severity string    `json:"severity,omitempty"`
	Since    time.Time `json:"since,omitempty"`
	Until    time.Time `json:"until,omitempty"`
	Limit    uint32    `json:"limit,omitempty"`
	Offset   uint32    `json:"offset,omitempty"`
}

// O1ValidationError represents an O1 validation error
type O1ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// O1ValidationResult represents the result of O1 validation
type O1ValidationResult struct {
	IsValid bool                 `json:"is_valid"`
	Errors  []O1ValidationError  `json:"errors,omitempty"`
}

// O1ErrorResponse represents an error response from O1 Mediator
type O1ErrorResponse struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Type   string `json:"type,omitempty"`
}

// Error implements the error interface for O1ErrorResponse
func (e O1ErrorResponse) Error() string {
	return e.Detail
}

// O1ManagedObjectType represents the type of managed object
type O1ManagedObjectType string

const (
	O1ManagedObjectTypeRIC      O1ManagedObjectType = "RIC"
	O1ManagedObjectTypeE2Node   O1ManagedObjectType = "E2_NODE"
	O1ManagedObjectTypeXApp     O1ManagedObjectType = "XAPP"
	O1ManagedObjectTypeService  O1ManagedObjectType = "SERVICE"
)

// O1ConfigurationStatus represents the status of a configuration
type O1ConfigurationStatus string

const (
	O1ConfigurationStatusActive   O1ConfigurationStatus = "ACTIVE"
	O1ConfigurationStatusInactive O1ConfigurationStatus = "INACTIVE"
	O1ConfigurationStatusPending  O1ConfigurationStatus = "PENDING"
	O1ConfigurationStatusError    O1ConfigurationStatus = "ERROR"
)

// O1AlarmSeverity represents the severity of an alarm
type O1AlarmSeverity string

const (
	O1AlarmSeverityCritical O1AlarmSeverity = "CRITICAL"
	O1AlarmSeverityMajor    O1AlarmSeverity = "MAJOR"
	O1AlarmSeverityMinor    O1AlarmSeverity = "MINOR"
	O1AlarmSeverityWarning  O1AlarmSeverity = "WARNING"
	O1AlarmSeverityCleared  O1AlarmSeverity = "CLEARED"
)

// O1AlarmState represents the state of an alarm
type O1AlarmState string

const (
	O1AlarmStateActive  O1AlarmState = "ACTIVE"
	O1AlarmStateCleared O1AlarmState = "CLEARED"
)

// O1AlarmAckState represents the acknowledgment state of an alarm
type O1AlarmAckState string

const (
	O1AlarmAckStateUnacknowledged O1AlarmAckState = "UNACKNOWLEDGED"
	O1AlarmAckStateAcknowledged   O1AlarmAckState = "ACKNOWLEDGED"
)

// O1KPIMeasurementType represents the type of KPI measurement
type O1KPIMeasurementType string

const (
	O1KPIMeasurementTypeCounter   O1KPIMeasurementType = "COUNTER"
	O1KPIMeasurementTypeGauge     O1KPIMeasurementType = "GAUGE"
	O1KPIMeasurementTypeHistogram O1KPIMeasurementType = "HISTOGRAM"
)

// O1BackupRequest represents a configuration backup request
type O1BackupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IncludeAll  bool   `json:"include_all"`
	ObjectTypes []string `json:"object_types,omitempty"`
}

// O1BackupResponse represents a configuration backup response
type O1BackupResponse struct {
	BackupID    string    `json:"backup_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Size        uint64    `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
}

// O1RestoreRequest represents a configuration restore request
type O1RestoreRequest struct {
	BackupID    string `json:"backup_id"`
	RestoreAll  bool   `json:"restore_all"`
	ObjectTypes []string `json:"object_types,omitempty"`
}

// O1RestoreResponse represents a configuration restore response
type O1RestoreResponse struct {
	RestoreID string    `json:"restore_id"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// O1Certificate represents a certificate for secure communications
type O1Certificate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	Fingerprint string    `json:"fingerprint"`
	Status      string    `json:"status"`
}

// O1CertificateListResponse represents the response for listing certificates
type O1CertificateListResponse struct {
	Certificates []O1Certificate `json:"certificates"`
	Total        uint32          `json:"total"`
}

// O1CertificateRequest represents a certificate request
type O1CertificateRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Subject     string `json:"subject"`
	KeySize     int    `json:"key_size,omitempty"`
	ValidityDays int   `json:"validity_days,omitempty"`
}

// O1BackupInfo represents backup information
type O1BackupInfo struct {
	BackupID    string    `json:"backup_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Size        uint64    `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
	ObjectTypes []string  `json:"object_types"`
}

// O1BackupListResponse represents the response for listing backups
type O1BackupListResponse struct {
	Backups []O1BackupInfo `json:"backups"`
	Total   uint32         `json:"total"`
}

// O1AlarmRequest represents a request to generate an alarm
type O1AlarmRequest struct {
	ManagedObjectID string    `json:"managed_object_id"`
	AlarmType       string    `json:"alarm_type"`
	Severity        string    `json:"severity"`
	ProbableCause   string    `json:"probable_cause"`
	SpecificProblem string    `json:"specific_problem"`
	AdditionalText  string    `json:"additional_text,omitempty"`
	EventTime       time.Time `json:"event_time"`
}

// O1AlarmClearRequest represents a request to clear an alarm
type O1AlarmClearRequest struct {
	AlarmID   O1AlarmID `json:"alarm_id"`
	User      string    `json:"user"`
	Reason    string    `json:"reason,omitempty"`
	ClearTime time.Time `json:"clear_time"`
}

// O1AlarmCorrelationRequest represents a request to correlate alarms
type O1AlarmCorrelationRequest struct {
	AlarmIDs        []O1AlarmID `json:"alarm_ids"`
	CorrelationType string      `json:"correlation_type"`
	RootCause       string      `json:"root_cause,omitempty"`
	Description     string      `json:"description,omitempty"`
}

// O1AlarmCorrelationResponse represents the response for alarm correlation
type O1AlarmCorrelationResponse struct {
	CorrelationID   string      `json:"correlation_id"`
	AlarmIDs        []O1AlarmID `json:"alarm_ids"`
	CorrelationType string      `json:"correlation_type"`
	RootCause       string      `json:"root_cause,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
}

// O1KPIRequest represents a request to create a KPI
type O1KPIRequest struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	MeasurementType string          `json:"measurement_type"`
	Unit            string          `json:"unit"`
	ManagedObjectID string          `json:"managed_object_id"`
	Threshold       *O1KPIThreshold `json:"threshold,omitempty"`
}

// O1KPIUpdate represents an update to a KPI
type O1KPIUpdate struct {
	KPIID       O1KPIID         `json:"kpi_id"`
	Value       float64         `json:"value"`
	Threshold   *O1KPIThreshold `json:"threshold,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

// O1KPICollectionRequest represents a request to collect KPI data
type O1KPICollectionRequest struct {
	KPIIDs    []O1KPIID `json:"kpi_ids,omitempty"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Interval  string    `json:"interval,omitempty"`
}

// O1KPICollectionResponse represents the response for KPI data collection
type O1KPICollectionResponse struct {
	CollectedKPIs []O1KPIDataPoint `json:"collected_kpis"`
	StartTime     time.Time        `json:"start_time"`
	EndTime       time.Time        `json:"end_time"`
	TotalPoints   uint32           `json:"total_points"`
}

// O1KPIDataPoint represents a single KPI data point
type O1KPIDataPoint struct {
	KPIID     O1KPIID   `json:"kpi_id"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Quality   string    `json:"quality,omitempty"`
}

// O1ResourceUsage represents resource usage information
type O1ResourceUsage struct {
	ID              string                 `json:"id"`
	ResourceType    string                 `json:"resource_type"`
	ResourceID      string                 `json:"resource_id"`
	UsageMetrics    map[string]interface{} `json:"usage_metrics"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	Duration        string                 `json:"duration"`
	Cost            *O1ResourceCost        `json:"cost,omitempty"`
	ManagedObjectID string                 `json:"managed_object_id"`
}

// O1ResourceCost represents the cost associated with resource usage
type O1ResourceCost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Unit     string  `json:"unit"`
}

// O1ResourceUsageResponse represents the response for resource usage queries
type O1ResourceUsageResponse struct {
	ResourceUsage []O1ResourceUsage `json:"resource_usage"`
	Total         uint32            `json:"total"`
}

// O1ResourceUsageRequest represents a request to create a resource usage record
type O1ResourceUsageRequest struct {
	ResourceType    string                 `json:"resource_type"`
	ResourceID      string                 `json:"resource_id"`
	UsageMetrics    map[string]interface{} `json:"usage_metrics"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	ManagedObjectID string                 `json:"managed_object_id"`
	Cost            *O1ResourceCost        `json:"cost,omitempty"`
}

// O1AccessControlPolicy represents an access control policy
type O1AccessControlPolicy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	PolicyType  string                 `json:"policy_type"`
	Rules       []O1AccessControlRule  `json:"rules"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// O1AccessControlRule represents a single access control rule
type O1AccessControlRule struct {
	ID          string                 `json:"id"`
	Subject     O1AccessControlSubject `json:"subject"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	Effect      string                 `json:"effect"`
	Conditions  map[string]interface{} `json:"conditions,omitempty"`
}

// O1AccessControlSubject represents the subject of an access control rule
type O1AccessControlSubject struct {
	Type       string   `json:"type"`
	Identifier string   `json:"identifier"`
	Attributes []string `json:"attributes,omitempty"`
}

// O1AccessControlPolicyListResponse represents the response for listing access control policies
type O1AccessControlPolicyListResponse struct {
	Policies []O1AccessControlPolicy `json:"policies"`
	Total    uint32                  `json:"total"`
}

// O1AccessControlPolicyRequest represents a request to create an access control policy
type O1AccessControlPolicyRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	PolicyType  string                `json:"policy_type"`
	Rules       []O1AccessControlRule `json:"rules"`
}

// O1ResourceType represents the type of resource for usage tracking
type O1ResourceType string

const (
	O1ResourceTypeCPU     O1ResourceType = "CPU"
	O1ResourceTypeMemory  O1ResourceType = "MEMORY"
	O1ResourceTypeStorage O1ResourceType = "STORAGE"
	O1ResourceTypeNetwork O1ResourceType = "NETWORK"
	O1ResourceTypeXApp    O1ResourceType = "XAPP"
	O1ResourceTypeE2Node  O1ResourceType = "E2_NODE"
)

// O1AccessControlEffect represents the effect of an access control rule
type O1AccessControlEffect string

const (
	O1AccessControlEffectAllow O1AccessControlEffect = "ALLOW"
	O1AccessControlEffectDeny  O1AccessControlEffect = "DENY"
)

// O1AccessControlAction represents actions in access control
type O1AccessControlAction string

const (
	O1AccessControlActionRead   O1AccessControlAction = "READ"
	O1AccessControlActionWrite  O1AccessControlAction = "WRITE"
	O1AccessControlActionDelete O1AccessControlAction = "DELETE"
	O1AccessControlActionExecute O1AccessControlAction = "EXECUTE"
)

// O1CertificateStatus represents the status of a certificate
type O1CertificateStatus string

const (
	O1CertificateStatusActive   O1CertificateStatus = "ACTIVE"
	O1CertificateStatusExpired  O1CertificateStatus = "EXPIRED"
	O1CertificateStatusRevoked  O1CertificateStatus = "REVOKED"
	O1CertificateStatusPending  O1CertificateStatus = "PENDING"
)

// O1BackupStatus represents the status of a backup
type O1BackupStatus string

const (
	O1BackupStatusCompleted  O1BackupStatus = "COMPLETED"
	O1BackupStatusInProgress O1BackupStatus = "IN_PROGRESS"
	O1BackupStatusFailed     O1BackupStatus = "FAILED"
	O1BackupStatusCorrupted  O1BackupStatus = "CORRUPTED"
)