// Package dashboard provides subscription models and types
// for the O-RAN Near-RT RIC platform E2 subscription management
package dashboard

import (
	"encoding/json"
	"time"
)

// Subscription management types and interfaces

// SubscriptionManager interface defines subscription management operations
type SubscriptionManager interface {
	CreateSubscription(req *SubscriptionRequest) (*SubscriptionResponse, error)
	DeleteSubscription(id SubscriptionID) error
	GetSubscription(id SubscriptionID) (*ManagedSubscription, error)
	ListSubscriptions() ([]*ManagedSubscription, error)
	UpdateSubscription(id SubscriptionID, req *SubscriptionRequest) (*SubscriptionResponse, error)
}

// SubscriptionRepository interface defines subscription storage operations
type SubscriptionRepository interface {
	Save(subscription *ManagedSubscription) error
	FindByID(id SubscriptionID) (*ManagedSubscription, error)
	FindByXAppID(xappID string) ([]*ManagedSubscription, error)
	FindByE2NodeID(e2NodeID string) ([]*ManagedSubscription, error)
	Delete(id SubscriptionID) error
	List() ([]*ManagedSubscription, error)
}

// NOTE: EventTriggerType type moved to types.go to avoid redeclaration

// NOTE: ActionType type is now defined in types.go to avoid redeclaration

// EventTrigger represents the event trigger definition
type EventTrigger struct {
	Type       EventTriggerType `json:"type"`
	Period     time.Duration    `json:"period,omitempty"`
	Definition []byte          `json:"definition,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// EventTriggerDefinition contains the detailed event trigger specification
type EventTriggerDefinition struct {
	InterfaceID         uint32                 `json:"interfaceId,omitempty"`
	InterfaceName       string                 `json:"interfaceName,omitempty"`
	InterfaceDirection  string                 `json:"interfaceDirection,omitempty"`
	InterfaceType       string                 `json:"interfaceType,omitempty"`
	ReportingPeriod     time.Duration         `json:"reportingPeriod,omitempty"`
	EventConditions     []EventCondition      `json:"eventConditions,omitempty"`
	GranularityPeriod   time.Duration         `json:"granularityPeriod,omitempty"`
	MeasurementLabels   map[string]string     `json:"measurementLabels,omitempty"`
	RANParameterList    []RANParameter        `json:"ranParameterList,omitempty"`
	CustomParameters    map[string]interface{} `json:"customParameters,omitempty"`
}

// EventCondition represents a condition for triggering events
type EventCondition struct {
	Type              EventConditionType     `json:"type"`
	Threshold         float64               `json:"threshold,omitempty"`
	HysteresisValue   float64               `json:"hysteresisValue,omitempty"`
	TimeToTrigger     time.Duration         `json:"timeToTrigger,omitempty"`
	MeasurementType   string                `json:"measurementType,omitempty"`
	ComparisonType    string                `json:"comparisonType,omitempty"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
}

// EventConditionType represents the type of event condition
type EventConditionType string

const (
	EventConditionTypeThreshold        EventConditionType = "THRESHOLD"
	EventConditionTypeChange          EventConditionType = "CHANGE"
	EventConditionTypePresence        EventConditionType = "PRESENCE"
	EventConditionTypeHysteresis      EventConditionType = "HYSTERESIS"
	EventConditionTypePeriodicReport  EventConditionType = "PERIODIC_REPORT"
)

// SubscriptionFilter allows filtering subscriptions based on various criteria
type SubscriptionFilter struct {
	XAppID         string                  `json:"xappId,omitempty"`
	E2NodeID       string                  `json:"e2NodeId,omitempty"`
	RANFunctionID  *uint32                 `json:"ranFunctionId,omitempty"`
	Status         *SubscriptionStatus     `json:"status,omitempty"`
	ServiceModel   string                  `json:"serviceModel,omitempty"`
	EventTrigger   *EventTriggerType       `json:"eventTrigger,omitempty"`
	CreatedAfter   *time.Time              `json:"createdAfter,omitempty"`
	CreatedBefore  *time.Time              `json:"createdBefore,omitempty"`
	ActiveOnly     bool                    `json:"activeOnly,omitempty"`
}

// SubscriptionQuery represents a query for subscriptions with pagination
type SubscriptionQuery struct {
	Filter   *SubscriptionFilter `json:"filter,omitempty"`
	Offset   int                 `json:"offset,omitempty"`
	Limit    int                 `json:"limit,omitempty"`
	SortBy   string             `json:"sortBy,omitempty"`
	SortDesc bool               `json:"sortDesc,omitempty"`
}

// SubscriptionQueryResult represents the result of a subscription query
type SubscriptionQueryResult struct {
	Subscriptions []*ManagedSubscription `json:"subscriptions"`
	TotalCount    int                    `json:"totalCount"`
	Offset        int                    `json:"offset"`
	Limit         int                    `json:"limit"`
	HasMore       bool                   `json:"hasMore"`
}

// SubscriptionValidation provides validation for subscription requests
type SubscriptionValidation struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// SubscriptionValidator interface defines subscription validation operations
type SubscriptionValidator interface {
	ValidateRequest(req *SubscriptionRequest) *SubscriptionValidation
	ValidateEventTrigger(trigger *EventTrigger) *SubscriptionValidation
	ValidateActions(actions []SubscriptionAction) *SubscriptionValidation
	ValidateConfiguration(config map[string]interface{}) *SubscriptionValidation
}

// SubscriptionEvent represents events that occur during subscription lifecycle
type SubscriptionEvent struct {
	ID             string                 `json:"id"`
	SubscriptionID SubscriptionID         `json:"subscriptionId"`
	EventType      SubscriptionEventType  `json:"eventType"`
	Timestamp      time.Time              `json:"timestamp"`
	Data           map[string]interface{} `json:"data,omitempty"`
	Source         string                 `json:"source"`
	Message        string                 `json:"message,omitempty"`
}

// SubscriptionEventType represents the type of subscription event
type SubscriptionEventType string

const (
	SubscriptionEventCreated     SubscriptionEventType = "CREATED"
	SubscriptionEventActivated   SubscriptionEventType = "ACTIVATED"
	SubscriptionEventDeactivated SubscriptionEventType = "DEACTIVATED"
	SubscriptionEventDeleted     SubscriptionEventType = "DELETED"
	SubscriptionEventFailed      SubscriptionEventType = "FAILED"
	SubscriptionEventIndication  SubscriptionEventType = "INDICATION_RECEIVED"
	SubscriptionEventTimeout     SubscriptionEventType = "TIMEOUT"
	SubscriptionEventError       SubscriptionEventType = "ERROR"
)

// SubscriptionMetrics contains metrics and statistics for subscriptions
type SubscriptionMetrics struct {
	SubscriptionID      SubscriptionID `json:"subscriptionId"`
	IndicationsReceived int64         `json:"indicationsReceived"`
	LastIndicationTime  *time.Time    `json:"lastIndicationTime,omitempty"`
	AverageLatency      time.Duration `json:"averageLatency"`
	ErrorCount          int64         `json:"errorCount"`
	TimeoutCount        int64         `json:"timeoutCount"`
	SuccessRate         float64       `json:"successRate"`
	Throughput          float64       `json:"throughput"` // indications per second
	DataVolume          int64         `json:"dataVolume"` // bytes processed
	LastErrorTime       *time.Time    `json:"lastErrorTime,omitempty"`
	LastError           string        `json:"lastError,omitempty"`
}

// SubscriptionTemplate provides pre-defined subscription templates
type SubscriptionTemplate struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	ServiceModel     string                 `json:"serviceModel"`
	RANFunctionOID   string                 `json:"ranFunctionOid"`
	EventTrigger     *EventTrigger          `json:"eventTrigger"`
	Actions          []SubscriptionAction   `json:"actions"`
	DefaultConfig    map[string]interface{} `json:"defaultConfig,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	CreatedAt        time.Time             `json:"createdAt"`
	Version          string                 `json:"version"`
}

// SubscriptionTemplateManager manages subscription templates
type SubscriptionTemplateManager interface {
	CreateTemplate(template *SubscriptionTemplate) error
	GetTemplate(id string) (*SubscriptionTemplate, error)
	ListTemplates(tags []string) ([]*SubscriptionTemplate, error)
	UpdateTemplate(id string, template *SubscriptionTemplate) error
	DeleteTemplate(id string) error
	ApplyTemplate(templateID string, params map[string]interface{}) (*SubscriptionRequest, error)
}

// SubscriptionPolicy defines policies for subscription management
type SubscriptionPolicy struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	Description           string                 `json:"description"`
	MaxSubscriptionsPerXApp int                  `json:"maxSubscriptionsPerXApp"`
	MaxSubscriptionsPerNode int                  `json:"maxSubscriptionsPerNode"`
	AllowedEventTriggers    []EventTriggerType   `json:"allowedEventTriggers"`
	AllowedActionTypes      []ActionType         `json:"allowedActionTypes"`
	RequiredPermissions     []string             `json:"requiredPermissions"`
	RateLimits             map[string]int        `json:"rateLimits,omitempty"`
	RestrictedRANFunctions  []uint32             `json:"restrictedRanFunctions,omitempty"`
	EnableAuditLogging      bool                 `json:"enableAuditLogging"`
	MaxIndicationRate       float64              `json:"maxIndicationRate,omitempty"`
	CustomRules             map[string]interface{} `json:"customRules,omitempty"`
}

// SubscriptionPolicyManager manages subscription policies
type SubscriptionPolicyManager interface {
	CreatePolicy(policy *SubscriptionPolicy) error
	GetPolicy(id string) (*SubscriptionPolicy, error)
	ListPolicies() ([]*SubscriptionPolicy, error)
	UpdatePolicy(id string, policy *SubscriptionPolicy) error
	DeletePolicy(id string) error
	EvaluateRequest(req *SubscriptionRequest, xappID string) *SubscriptionValidation
}

// SubscriptionAudit represents audit information for subscription operations
type SubscriptionAudit struct {
	ID             string                 `json:"id"`
	SubscriptionID SubscriptionID         `json:"subscriptionId"`
	Operation      string                 `json:"operation"`
	UserID         string                 `json:"userId"`
	XAppID         string                 `json:"xappId"`
	Timestamp      time.Time              `json:"timestamp"`
	Changes        map[string]interface{} `json:"changes,omitempty"`
	Success        bool                   `json:"success"`
	ErrorMessage   string                 `json:"errorMessage,omitempty"`
	IPAddress      string                 `json:"ipAddress,omitempty"`
	UserAgent      string                 `json:"userAgent,omitempty"`
}

// SubscriptionBatch represents a batch operation for multiple subscriptions
type SubscriptionBatch struct {
	ID          string                   `json:"id"`
	XAppID      string                   `json:"xappId"`
	Operation   SubscriptionBatchOperation `json:"operation"`
	Requests    []*SubscriptionRequest   `json:"requests,omitempty"`
	IDs         []SubscriptionID         `json:"ids,omitempty"`
	Status      SubscriptionBatchStatus  `json:"status"`
	CreatedAt   time.Time                `json:"createdAt"`
	CompletedAt *time.Time               `json:"completedAt,omitempty"`
	Results     []*SubscriptionBatchResult `json:"results,omitempty"`
	ErrorCount  int                      `json:"errorCount"`
	SuccessCount int                     `json:"successCount"`
}

// SubscriptionBatchOperation represents the type of batch operation
type SubscriptionBatchOperation string

const (
	BatchOperationCreate SubscriptionBatchOperation = "CREATE"
	BatchOperationDelete SubscriptionBatchOperation = "DELETE"
	BatchOperationUpdate SubscriptionBatchOperation = "UPDATE"
)

// SubscriptionBatchStatus represents the status of a batch operation
type SubscriptionBatchStatus string

const (
	BatchStatusPending   SubscriptionBatchStatus = "PENDING"
	BatchStatusRunning   SubscriptionBatchStatus = "RUNNING"
	BatchStatusCompleted SubscriptionBatchStatus = "COMPLETED"
	BatchStatusFailed    SubscriptionBatchStatus = "FAILED"
	BatchStatusCancelled SubscriptionBatchStatus = "CANCELLED"
)

// SubscriptionBatchResult represents the result of a single operation in a batch
type SubscriptionBatchResult struct {
	Index          int               `json:"index"`
	SubscriptionID *SubscriptionID   `json:"subscriptionId,omitempty"`
	Success        bool              `json:"success"`
	Error          string            `json:"error,omitempty"`
	Response       *SubscriptionResponse `json:"response,omitempty"`
}

// Utility functions for subscription models

// NewSubscriptionRequest creates a new subscription request with default values
func NewSubscriptionRequest(xappID, e2NodeID string, ranFunctionID uint32) *SubscriptionRequest {
	return &SubscriptionRequest{
		XAppID:        xappID,
		E2NodeID:      e2NodeID,
		RANFunctionID: ranFunctionID,
		Configuration: make(map[string]interface{}),
	}
}

// AddAction adds an action to a subscription request
func (req *SubscriptionRequest) AddAction(actionID uint32, actionType ActionType, definition []byte) {
	action := SubscriptionAction{
		ActionID:   actionID,
		ActionType: actionType,
		Definition: definition,
	}
	req.Actions = append(req.Actions, action)
}

// SetEventTrigger sets the event trigger for a subscription request
func (req *SubscriptionRequest) SetEventTrigger(triggerType EventTriggerType, period time.Duration) {
	req.EventTrigger = &EventTrigger{
		Type:   triggerType,
		Period: period,
	}
}

// Validate performs basic validation on a subscription request
func (req *SubscriptionRequest) Validate() *SubscriptionValidation {
	validation := &SubscriptionValidation{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	if req.XAppID == "" {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "xAppID is required")
	}

	if req.E2NodeID == "" {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "e2NodeID is required")
	}

	if req.RANFunctionID == 0 {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "ranFunctionID is required")
	}

	if req.EventTrigger == nil {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "eventTrigger is required")
	}

	if len(req.Actions) == 0 {
		validation.Valid = false
		validation.Errors = append(validation.Errors, "at least one action is required")
	}

	return validation
}

// ToJSON converts a subscription request to JSON
func (req *SubscriptionRequest) ToJSON() ([]byte, error) {
	return json.MarshalIndent(req, "", "  ")
}

// FromJSON creates a subscription request from JSON
func (req *SubscriptionRequest) FromJSON(data []byte) error {
	return json.Unmarshal(data, req)
}

// IsActive checks if a subscription is active
func (sub *ManagedSubscription) IsActive() bool {
	return sub.Status == SubscriptionStatusActive
}

// IsExpired checks if a subscription has expired (no activity for a long time)
func (sub *ManagedSubscription) IsExpired(threshold time.Duration) bool {
	return time.Since(sub.LastActivity) > threshold
}

// GetAge returns the age of the subscription
func (sub *ManagedSubscription) GetAge() time.Duration {
	return time.Since(sub.CreatedAt)
}

// HasErrored checks if the subscription has errors
func (sub *ManagedSubscription) HasErrored() bool {
	return sub.ErrorCount > 0
}

// GetIndicationRate calculates the indication rate (indications per minute)
func (sub *ManagedSubscription) GetIndicationRate() float64 {
	if sub.IndicationCount == 0 {
		return 0
	}
	
	age := sub.GetAge()
	if age.Minutes() == 0 {
		return 0
	}
	
	return float64(sub.IndicationCount) / age.Minutes()
}