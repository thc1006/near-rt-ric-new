/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrorHandler type is now defined in types.go to avoid redeclaration

// ErrorRecord represents a recorded error with context
type ErrorRecord struct {
	ID          string                 `json:"id"`
	Type        ErrorType              `json:"type"`
	Severity    ErrorSeverity          `json:"severity"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	StackTrace  string                 `json:"stackTrace,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	RetryCount  int                    `json:"retryCount"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolvedAt,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeConnection    ErrorType = "CONNECTION"
	ErrorTypeProtocol      ErrorType = "PROTOCOL"
	ErrorTypeValidation    ErrorType = "VALIDATION"
	ErrorTypeTimeout       ErrorType = "TIMEOUT"
	ErrorTypeResource      ErrorType = "RESOURCE"
	ErrorTypeConfiguration ErrorType = "CONFIGURATION"
	ErrorTypeInternal      ErrorType = "INTERNAL"
	ErrorTypeExternal      ErrorType = "EXTERNAL"
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity string

const (
	ErrorSeverityCritical ErrorSeverity = "CRITICAL"
	ErrorSeverityHigh     ErrorSeverity = "HIGH"
	ErrorSeverityMedium   ErrorSeverity = "MEDIUM"
	ErrorSeverityLow      ErrorSeverity = "LOW"
	ErrorSeverityInfo     ErrorSeverity = "INFO"
)

// RecoveryHandler defines the interface for error recovery handlers
type RecoveryHandler interface {
	CanRecover(error *ErrorRecord) bool
	Recover(ctx context.Context, error *ErrorRecord) error
	GetRecoveryStrategy() string
}

// ErrorAlertCallback defines the callback function for error alerts
type ErrorAlertCallback func(error *ErrorRecord)

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	ctx, cancel := context.WithCancel(context.Background())
	
	handler := &ErrorHandler{
		errorRecords:     make(map[string]*ErrorRecord),
		recoveryHandlers: make(map[ErrorType]RecoveryHandler),
		alertCallbacks:   make([]ErrorAlertCallback, 0),
		maxRetries:       3,
		retryBackoff:     time.Second,
		errorThreshold:   10,
		timeWindow:       5 * time.Minute,
		ctx:              ctx,
		cancel:           cancel,
	}
	
	// Register default recovery handlers
	handler.registerDefaultRecoveryHandlers()
	
	return handler
}

// Start starts the error handler
func (eh *ErrorHandler) Start() error {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	
	if eh.isRunning {
		return fmt.Errorf("error handler is already running")
	}
	
	// Start error monitoring routine
	go eh.errorMonitoringRoutine()
	
	// Start cleanup routine
	go eh.cleanupRoutine()
	
	eh.isRunning = true
	log.Println("Error handler started")
	return nil
}

// Stop stops the error handler
func (eh *ErrorHandler) Stop() error {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	
	if !eh.isRunning {
		return nil
	}
	
	eh.cancel()
	eh.isRunning = false
	log.Println("Error handler stopped")
	return nil
}

// HandleError handles an error with automatic recovery attempts
func (eh *ErrorHandler) HandleError(ctx context.Context, errorType ErrorType, severity ErrorSeverity, component, operation, message string, err error, details map[string]interface{}) *ErrorRecord {
	errorRecord := &ErrorRecord{
		ID:         uuid.New().String(),
		Type:       errorType,
		Severity:   severity,
		Component:  component,
		Operation:  operation,
		Message:    message,
		Details:    details,
		Timestamp:  time.Now(),
		RetryCount: 0,
		Resolved:   false,
		Context:    make(map[string]interface{}),
	}
	
	// Add error details if provided
	if err != nil {
		errorRecord.Details["error"] = err.Error()
	}
	
	// Capture stack trace for critical errors
	if severity == ErrorSeverityCritical || severity == ErrorSeverityHigh {
		errorRecord.StackTrace = eh.captureStackTrace()
	}
	
	// Add context information
	errorRecord.Context["goroutine_id"] = eh.getGoroutineID()
	errorRecord.Context["timestamp_unix"] = errorRecord.Timestamp.Unix()
	
	// Store error record
	eh.mu.Lock()
	eh.errorRecords[errorRecord.ID] = errorRecord
	eh.mu.Unlock()
	
	// Log error
	eh.logError(errorRecord)
	
	// Trigger alerts
	eh.triggerAlerts(errorRecord)
	
	// Attempt recovery
	go eh.attemptRecovery(ctx, errorRecord)
	
	return errorRecord
}

// RegisterRecoveryHandler registers a recovery handler for a specific error type
func (eh *ErrorHandler) RegisterRecoveryHandler(errorType ErrorType, handler RecoveryHandler) {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	
	eh.recoveryHandlers[errorType] = handler
	log.Printf("Registered recovery handler for error type: %s", errorType)
}

// AddAlertCallback adds a callback function for error alerts
func (eh *ErrorHandler) AddAlertCallback(callback ErrorAlertCallback) {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	
	eh.alertCallbacks = append(eh.alertCallbacks, callback)
}

// GetErrorRecord returns an error record by ID
func (eh *ErrorHandler) GetErrorRecord(errorID string) (*ErrorRecord, error) {
	eh.mu.RLock()
	defer eh.mu.RUnlock()
	
	record, exists := eh.errorRecords[errorID]
	if !exists {
		return nil, fmt.Errorf("error record %s not found", errorID)
	}
	
	return record, nil
}

// GetErrorRecords returns all error records with optional filtering
func (eh *ErrorHandler) GetErrorRecords(filter *ErrorFilter) []*ErrorRecord {
	eh.mu.RLock()
	defer eh.mu.RUnlock()
	
	var records []*ErrorRecord
	for _, record := range eh.errorRecords {
		if filter == nil || eh.matchesFilter(record, filter) {
			records = append(records, record)
		}
	}
	
	return records
}

// GetErrorStatistics returns error statistics
func (eh *ErrorHandler) GetErrorStatistics() *ErrorStatistics {
	eh.mu.RLock()
	defer eh.mu.RUnlock()
	
	stats := &ErrorStatistics{
		TotalErrors:    len(eh.errorRecords),
		ByType:         make(map[string]int),
		BySeverity:     make(map[string]int),
		ByComponent:    make(map[string]int),
		ResolvedErrors: 0,
		LastUpdated:    time.Now(),
	}
	
	for _, record := range eh.errorRecords {
		stats.ByType[string(record.Type)]++
		stats.BySeverity[string(record.Severity)]++
		stats.ByComponent[record.Component]++
		
		if record.Resolved {
			stats.ResolvedErrors++
		}
	}
	
	return stats
}

// MarkResolved marks an error as resolved
func (eh *ErrorHandler) MarkResolved(errorID string) error {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	
	record, exists := eh.errorRecords[errorID]
	if !exists {
		return fmt.Errorf("error record %s not found", errorID)
	}
	
	record.Resolved = true
	now := time.Now()
	record.ResolvedAt = &now
	
	log.Printf("Marked error %s as resolved", errorID)
	return nil
}

// Private methods

func (eh *ErrorHandler) registerDefaultRecoveryHandlers() {
	// Connection recovery handler
	eh.recoveryHandlers[ErrorTypeConnection] = &ConnectionRecoveryHandler{}
	
	// Protocol recovery handler
	eh.recoveryHandlers[ErrorTypeProtocol] = &ProtocolRecoveryHandler{}
	
	// Timeout recovery handler
	eh.recoveryHandlers[ErrorTypeTimeout] = &TimeoutRecoveryHandler{}
	
	// Resource recovery handler
	eh.recoveryHandlers[ErrorTypeResource] = &ResourceRecoveryHandler{}
	
	log.Printf("Registered %d default recovery handlers", len(eh.recoveryHandlers))
}

func (eh *ErrorHandler) attemptRecovery(ctx context.Context, errorRecord *ErrorRecord) {
	handler, exists := eh.recoveryHandlers[errorRecord.Type]
	if !exists {
		log.Printf("No recovery handler available for error type: %s", errorRecord.Type)
		return
	}
	
	if !handler.CanRecover(errorRecord) {
		log.Printf("Error %s cannot be recovered by handler", errorRecord.ID)
		return
	}
	
	// Implement retry logic with exponential backoff
	for errorRecord.RetryCount < eh.maxRetries {
		errorRecord.RetryCount++
		
		// Wait before retry
		backoffDuration := eh.retryBackoff * time.Duration(errorRecord.RetryCount)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoffDuration):
		}
		
		log.Printf("Attempting recovery for error %s (attempt %d/%d)", 
			errorRecord.ID, errorRecord.RetryCount, eh.maxRetries)
		
		if err := handler.Recover(ctx, errorRecord); err != nil {
			log.Printf("Recovery attempt %d failed for error %s: %v", 
				errorRecord.RetryCount, errorRecord.ID, err)
			continue
		}
		
		// Recovery successful
		eh.MarkResolved(errorRecord.ID)
		log.Printf("Successfully recovered from error %s using strategy: %s", 
			errorRecord.ID, handler.GetRecoveryStrategy())
		return
	}
	
	log.Printf("Failed to recover from error %s after %d attempts", 
		errorRecord.ID, eh.maxRetries)
}

func (eh *ErrorHandler) logError(errorRecord *ErrorRecord) {
	logLevel := "INFO"
	switch errorRecord.Severity {
	case ErrorSeverityCritical:
		logLevel = "CRITICAL"
	case ErrorSeverityHigh:
		logLevel = "ERROR"
	case ErrorSeverityMedium:
		logLevel = "WARN"
	case ErrorSeverityLow:
		logLevel = "INFO"
	}
	
	log.Printf("[%s] %s/%s: %s - %s", 
		logLevel, errorRecord.Component, errorRecord.Operation, 
		errorRecord.Type, errorRecord.Message)
}

func (eh *ErrorHandler) triggerAlerts(errorRecord *ErrorRecord) {
	for _, callback := range eh.alertCallbacks {
		go callback(errorRecord)
	}
}

func (eh *ErrorHandler) captureStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (eh *ErrorHandler) getGoroutineID() int {
	// This is a simplified implementation
	// In production, you might want to use a more robust method
	return runtime.NumGoroutine()
}

func (eh *ErrorHandler) matchesFilter(record *ErrorRecord, filter *ErrorFilter) bool {
	if filter.Type != "" && record.Type != filter.Type {
		return false
	}
	if filter.Severity != "" && record.Severity != filter.Severity {
		return false
	}
	if filter.Component != "" && record.Component != filter.Component {
		return false
	}
	if !filter.Since.IsZero() && record.Timestamp.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && record.Timestamp.After(filter.Until) {
		return false
	}
	if filter.Resolved != nil && record.Resolved != *filter.Resolved {
		return false
	}
	
	return true
}

func (eh *ErrorHandler) errorMonitoringRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-eh.ctx.Done():
			return
		case <-ticker.C:
			eh.monitorErrorPatterns()
		}
	}
}

func (eh *ErrorHandler) monitorErrorPatterns() {
	eh.mu.RLock()
	defer eh.mu.RUnlock()
	
	now := time.Now()
	windowStart := now.Add(-eh.timeWindow)
	
	// Count errors in the time window
	errorCounts := make(map[ErrorType]int)
	for _, record := range eh.errorRecords {
		if record.Timestamp.After(windowStart) && !record.Resolved {
			errorCounts[record.Type]++
		}
	}
	
	// Check for error patterns that exceed threshold
	for errorType, count := range errorCounts {
		if count >= eh.errorThreshold {
			log.Printf("Error pattern detected: %s errors (%d) exceeded threshold (%d) in %v window", 
				errorType, count, eh.errorThreshold, eh.timeWindow)
		}
	}
}

func (eh *ErrorHandler) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-eh.ctx.Done():
			return
		case <-ticker.C:
			eh.cleanupOldErrors()
		}
	}
}

func (eh *ErrorHandler) cleanupOldErrors() {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour) // Keep errors for 24 hours
	var toDelete []string
	
	for id, record := range eh.errorRecords {
		if record.Resolved && record.ResolvedAt != nil && record.ResolvedAt.Before(cutoff) {
			toDelete = append(toDelete, id)
		} else if !record.Resolved && record.Timestamp.Before(cutoff) {
			// Keep unresolved errors longer, but clean up very old ones
			veryOldCutoff := time.Now().Add(-7 * 24 * time.Hour)
			if record.Timestamp.Before(veryOldCutoff) {
				toDelete = append(toDelete, id)
			}
		}
	}
	
	for _, id := range toDelete {
		delete(eh.errorRecords, id)
	}
	
	if len(toDelete) > 0 {
		log.Printf("Cleaned up %d old error records", len(toDelete))
	}
}

// Data structures

// ErrorFilter represents filter criteria for error records
type ErrorFilter struct {
	Type      ErrorType     `json:"type,omitempty"`
	Severity  ErrorSeverity `json:"severity,omitempty"`
	Component string        `json:"component,omitempty"`
	Since     time.Time     `json:"since,omitempty"`
	Until     time.Time     `json:"until,omitempty"`
	Resolved  *bool         `json:"resolved,omitempty"`
}

// ErrorStatistics represents error statistics
type ErrorStatistics struct {
	TotalErrors    int            `json:"totalErrors"`
	ResolvedErrors int            `json:"resolvedErrors"`
	ByType         map[string]int `json:"byType"`
	BySeverity     map[string]int `json:"bySeverity"`
	ByComponent    map[string]int `json:"byComponent"`
	LastUpdated    time.Time      `json:"lastUpdated"`
}

// Recovery handler implementations

// ConnectionRecoveryHandler handles connection-related errors
type ConnectionRecoveryHandler struct{}

func (h *ConnectionRecoveryHandler) CanRecover(error *ErrorRecord) bool {
	return error.Type == ErrorTypeConnection && error.Severity != ErrorSeverityCritical
}

func (h *ConnectionRecoveryHandler) Recover(ctx context.Context, error *ErrorRecord) error {
	log.Printf("Attempting connection recovery for error %s", error.ID)
	
	// Simulate connection recovery logic
	time.Sleep(100 * time.Millisecond)
	
	// In a real implementation, this would attempt to re-establish connections
	return nil
}

func (h *ConnectionRecoveryHandler) GetRecoveryStrategy() string {
	return "connection_retry"
}

// ProtocolRecoveryHandler handles protocol-related errors
type ProtocolRecoveryHandler struct{}

func (h *ProtocolRecoveryHandler) CanRecover(error *ErrorRecord) bool {
	return error.Type == ErrorTypeProtocol && error.RetryCount < 2
}

func (h *ProtocolRecoveryHandler) Recover(ctx context.Context, error *ErrorRecord) error {
	log.Printf("Attempting protocol recovery for error %s", error.ID)
	
	// Simulate protocol recovery logic
	time.Sleep(50 * time.Millisecond)
	
	// In a real implementation, this would reset protocol state
	return nil
}

func (h *ProtocolRecoveryHandler) GetRecoveryStrategy() string {
	return "protocol_reset"
}

// TimeoutRecoveryHandler handles timeout-related errors
type TimeoutRecoveryHandler struct{}

func (h *TimeoutRecoveryHandler) CanRecover(error *ErrorRecord) bool {
	return error.Type == ErrorTypeTimeout
}

func (h *TimeoutRecoveryHandler) Recover(ctx context.Context, error *ErrorRecord) error {
	log.Printf("Attempting timeout recovery for error %s", error.ID)
	
	// Simulate timeout recovery logic
	time.Sleep(200 * time.Millisecond)
	
	// In a real implementation, this would retry with increased timeout
	return nil
}

func (h *TimeoutRecoveryHandler) GetRecoveryStrategy() string {
	return "timeout_retry"
}

// ResourceRecoveryHandler handles resource-related errors
type ResourceRecoveryHandler struct{}

func (h *ResourceRecoveryHandler) CanRecover(error *ErrorRecord) bool {
	return error.Type == ErrorTypeResource && error.Severity != ErrorSeverityCritical
}

func (h *ResourceRecoveryHandler) Recover(ctx context.Context, error *ErrorRecord) error {
	log.Printf("Attempting resource recovery for error %s", error.ID)
	
	// Simulate resource recovery logic
	time.Sleep(300 * time.Millisecond)
	
	// In a real implementation, this would free up resources or scale up
	return nil
}

func (h *ResourceRecoveryHandler) GetRecoveryStrategy() string {
	return "resource_cleanup"
}