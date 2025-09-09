// Package dashboard provides common types and utilities
// for the O-RAN Near-RT RIC Dashboard
package dashboard

import (
	"context"
	"encoding/json"
	"time"
)

// Common interfaces and utilities

// NOTE: ClientManager type moved to clients.go to avoid redeclaration

// Client represents a generic client
type Client struct {
	ID       string
	Type     string
	Endpoint string
	Active   bool
}

// NOTE: SubscriptionManagerClient type is in its own file

// NOTE: E2TClient type moved to e2t_client.go to avoid redeclaration

// NOTE: E2ManagerClient type moved to e2_manager_client.go to avoid redeclaration

// NOTE: NodeStatus type moved to types.go to avoid redeclaration

// APIResponse represents a generic API response
type APIResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"requestId,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalPages int `json:"totalPages"`
	TotalItems int `json:"totalItems"`
	HasNext    bool `json:"hasNext"`
	HasPrev    bool `json:"hasPrev"`
}

// FilterOptions represents generic filtering options
type FilterOptions struct {
	Search    string                 `json:"search,omitempty"`
	StartDate *time.Time             `json:"startDate,omitempty"`
	EndDate   *time.Time             `json:"endDate,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Type      string                 `json:"type,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

// SortOptions represents sorting options
type SortOptions struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "asc" or "desc"
}

// QueryRequest represents a generic query request
type QueryRequest struct {
	Filter     *FilterOptions `json:"filter,omitempty"`
	Sort       *SortOptions   `json:"sort,omitempty"`
	Pagination *Pagination    `json:"pagination,omitempty"`
}

// BatchRequest represents a batch operation request
type BatchRequest struct {
	Operations []BatchOperation `json:"operations"`
	Options    *BatchOptions    `json:"options,omitempty"`
}

// BatchOperation represents a single operation in a batch
type BatchOperation struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// BatchOptions represents options for batch operations
type BatchOptions struct {
	ContinueOnError bool          `json:"continueOnError"`
	Timeout         time.Duration `json:"timeout"`
	MaxConcurrency  int           `json:"maxConcurrency"`
	RetryPolicy     *RetryPolicy  `json:"retryPolicy,omitempty"`
}

// BatchResponse represents a batch operation response
type BatchResponse struct {
	Results   []BatchOperationResult `json:"results"`
	Summary   BatchSummary  `json:"summary"`
	Timestamp time.Time     `json:"timestamp"`
}

// BatchOperationResult represents the result of a single batch operation
type BatchOperationResult struct {
	ID      string      `json:"id"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// BatchSummary represents a summary of batch operation results
type BatchSummary struct {
	Total     int           `json:"total"`
	Succeeded int           `json:"succeeded"`
	Failed    int           `json:"failed"`
	Duration  time.Duration `json:"duration"`
}

// NOTE: RetryPolicy type moved to e2e_testing_suite.go to avoid redeclaration

// NOTE: ConfigurationManager type moved to o1_management_service.go to avoid redeclaration

// NOTE: CacheEntry type moved to enhanced_dashboard_api.go to avoid redeclaration

// EventBus provides event publishing and subscription capabilities
type EventBus struct {
	subscribers map[string][]EventHandler
	active      bool
}

// EventHandler defines the interface for handling events
type EventHandler interface {
	HandleEvent(ctx context.Context, event *Event) error
}

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Subject   string                 `json:"subject,omitempty"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		active:      false,
	}
}

// Subscribe adds an event handler for a specific event type
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(ctx context.Context, event *Event) error {
	if !eb.active {
		return nil
	}

	handlers, exists := eb.subscribers[event.Type]
	if !exists {
		return nil
	}

	for _, handler := range handlers {
		if err := handler.HandleEvent(ctx, event); err != nil {
			// Log error but continue processing
			continue
		}
	}

	return nil
}

// Start starts the event bus
func (eb *EventBus) Start() error {
	eb.active = true
	return nil
}

// Stop stops the event bus
func (eb *EventBus) Stop() error {
	eb.active = false
	return nil
}

// Utility functions

// ToJSON converts any value to JSON string
func ToJSON(v interface{}) (string, error) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON converts JSON string to a value
func FromJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}

// Contains checks if a slice contains a specific value
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Unique removes duplicate strings from a slice
func Unique(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// Map applies a function to each string in a slice
func Map(slice []string, fn func(string) string) []string {
	result := make([]string, len(slice))
	for i, item := range slice {
		result[i] = fn(item)
	}
	return result
}

// Filter filters strings in a slice based on a predicate
func Filter(slice []string, predicate func(string) bool) []string {
	result := []string{}
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Reduce reduces a slice of strings to a single value
func Reduce(slice []string, initial string, reducer func(string, string) string) string {
	result := initial
	for _, item := range slice {
		result = reducer(result, item)
	}
	return result
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Duration returns the duration of the time range
func (tr *TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// Contains checks if a time is within the range
func (tr *TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && !t.After(tr.End)
}

// Overlaps checks if this range overlaps with another
func (tr *TimeRange) Overlaps(other *TimeRange) bool {
	return tr.Start.Before(other.End) && other.Start.Before(tr.End)
}

// StatusCode represents HTTP status codes commonly used
type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusCreated            StatusCode = 201
	StatusAccepted           StatusCode = 202
	StatusNoContent          StatusCode = 204
	StatusBadRequest         StatusCode = 400
	StatusUnauthorized       StatusCode = 401
	StatusForbidden          StatusCode = 403
	StatusNotFound           StatusCode = 404
	StatusMethodNotAllowed   StatusCode = 405
	StatusConflict           StatusCode = 409
	StatusUnprocessableEntity StatusCode = 422
	StatusInternalServerError StatusCode = 500
	StatusBadGateway         StatusCode = 502
	StatusServiceUnavailable StatusCode = 503
	StatusGatewayTimeout     StatusCode = 504
)

// String returns the string representation of the status code
func (sc StatusCode) String() string {
	switch sc {
		case StatusOK:
			return "OK"
		case StatusCreated:
			return "Created"
		case StatusAccepted:
			return "Accepted"
		case StatusNoContent:
			return "No Content"
		case StatusBadRequest:
			return "Bad Request"
		case StatusUnauthorized:
			return "Unauthorized"
		case StatusForbidden:
			return "Forbidden"
		case StatusNotFound:
			return "Not Found"
		case StatusMethodNotAllowed:
			return "Method Not Allowed"
		case StatusConflict:
			return "Conflict"
		case StatusUnprocessableEntity:
			return "Unprocessable Entity"
		case StatusInternalServerError:
			return "Internal Server Error"
		case StatusBadGateway:
			return "Bad Gateway"
		case StatusServiceUnavailable:
			return "Service Unavailable"
		case StatusGatewayTimeout:
			return "Gateway Timeout"
		default:
			return "Unknown"
	}
}