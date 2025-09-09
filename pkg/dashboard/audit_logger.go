/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// AuditLogger handles security audit logging
type AuditLogger struct {
	events    []AuditEvent
	logFile   *os.File
	maxEvents int
	mutex     sync.RWMutex
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logFilePath string, maxEvents int) (*AuditLogger, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	return &AuditLogger{
		events:    make([]AuditEvent, 0),
		logFile:   logFile,
		maxEvents: maxEvents,
	}, nil
}

// LogEvent logs a security audit event
func (al *AuditLogger) LogEvent(event *AuditEvent) {
	if event.ID == "" {
		event.ID = al.generateEventID()
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	al.mutex.Lock()
	defer al.mutex.Unlock()

	// Add to in-memory storage
	al.events = append(al.events, *event)

	// Maintain max events limit
	if len(al.events) > al.maxEvents {
		al.events = al.events[len(al.events)-al.maxEvents:]
	}

	// Write to log file
	al.writeToFile(event)

	// Log to standard logger for immediate visibility
	log.Printf("AUDIT: %s - User: %s, Resource: %s, Action: %s, Result: %s",
		event.EventType, event.Username, event.Resource, event.Action, event.Result)
}

// GetEvents retrieves audit events with optional filtering
func (al *AuditLogger) GetEvents(filter *AuditEventFilter) []AuditEvent {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	if filter == nil {
		// Return all events
		result := make([]AuditEvent, len(al.events))
		copy(result, al.events)
		return result
	}

	var filtered []AuditEvent
	for _, event := range al.events {
		if al.matchesFilter(&event, filter) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// GetEventsByUser retrieves audit events for a specific user
func (al *AuditLogger) GetEventsByUser(userID string, limit int) []AuditEvent {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	var userEvents []AuditEvent
	count := 0

	// Search from most recent events
	for i := len(al.events) - 1; i >= 0 && count < limit; i-- {
		if al.events[i].UserID == userID {
			userEvents = append([]AuditEvent{al.events[i]}, userEvents...)
			count++
		}
	}

	return userEvents
}

// GetEventsByResource retrieves audit events for a specific resource
func (al *AuditLogger) GetEventsByResource(resource string, limit int) []AuditEvent {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	var resourceEvents []AuditEvent
	count := 0

	// Search from most recent events
	for i := len(al.events) - 1; i >= 0 && count < limit; i-- {
		if al.events[i].Resource == resource {
			resourceEvents = append([]AuditEvent{al.events[i]}, resourceEvents...)
			count++
		}
	}

	return resourceEvents
}

// GetFailedEvents retrieves failed audit events
func (al *AuditLogger) GetFailedEvents(limit int) []AuditEvent {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	var failedEvents []AuditEvent
	count := 0

	// Search from most recent events
	for i := len(al.events) - 1; i >= 0 && count < limit; i-- {
		if al.events[i].Result == ResultFailure || al.events[i].Result == ResultDenied {
			failedEvents = append([]AuditEvent{al.events[i]}, failedEvents...)
			count++
		}
	}

	return failedEvents
}

// GetSecurityEvents retrieves security-related events
func (al *AuditLogger) GetSecurityEvents(limit int) []AuditEvent {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	securityEventTypes := map[string]bool{
		EventTypeLogin:             true,
		EventTypeLogout:            true,
		EventTypeLoginFailed:       true,
		EventTypePasswordChanged:   true,
		EventTypeAccessDenied:      true,
		EventTypePermissionGranted: true,
		EventTypePermissionRevoked: true,
	}

	var securityEvents []AuditEvent
	count := 0

	// Search from most recent events
	for i := len(al.events) - 1; i >= 0 && count < limit; i-- {
		if securityEventTypes[al.events[i].EventType] {
			securityEvents = append([]AuditEvent{al.events[i]}, securityEvents...)
			count++
		}
	}

	return securityEvents
}

// GetEventStats returns statistics about audit events
func (al *AuditLogger) GetEventStats() *AuditEventStats {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	stats := &AuditEventStats{
		TotalEvents:    len(al.events),
		EventsByType:   make(map[string]int),
		EventsByResult: make(map[string]int),
		EventsByUser:   make(map[string]int),
	}

	for _, event := range al.events {
		stats.EventsByType[event.EventType]++
		stats.EventsByResult[event.Result]++
		if event.Username != "" {
			stats.EventsByUser[event.Username]++
		}
	}

	return stats
}

// Close closes the audit logger and its resources
func (al *AuditLogger) Close() error {
	al.mutex.Lock()
	defer al.mutex.Unlock()

	if al.logFile != nil {
		return al.logFile.Close()
	}

	return nil
}

// writeToFile writes an audit event to the log file
func (al *AuditLogger) writeToFile(event *AuditEvent) {
	if al.logFile == nil {
		return
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal audit event: %v", err)
		return
	}

	if _, err := al.logFile.WriteString(string(eventJSON) + "\n"); err != nil {
		log.Printf("Failed to write audit event to file: %v", err)
	}

	// Ensure data is written to disk
	al.logFile.Sync()
}

// matchesFilter checks if an event matches the given filter
func (al *AuditLogger) matchesFilter(event *AuditEvent, filter *AuditEventFilter) bool {
	if filter.EventType != "" && event.EventType != filter.EventType {
		return false
	}

	if filter.UserID != "" && event.UserID != filter.UserID {
		return false
	}

	if filter.Username != "" && event.Username != filter.Username {
		return false
	}

	if filter.Resource != "" && event.Resource != filter.Resource {
		return false
	}

	if filter.Action != "" && event.Action != filter.Action {
		return false
	}

	if filter.Result != "" && event.Result != filter.Result {
		return false
	}

	if !filter.StartTime.IsZero() && event.Timestamp.Before(filter.StartTime) {
		return false
	}

	if !filter.EndTime.IsZero() && event.Timestamp.After(filter.EndTime) {
		return false
	}

	return true
}

// generateEventID generates a unique event ID
func (al *AuditLogger) generateEventID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// AuditEventFilter represents filters for audit event queries
type AuditEventFilter struct {
	EventType string    `json:"eventType,omitempty"`
	UserID    string    `json:"userId,omitempty"`
	Username  string    `json:"username,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Action    string    `json:"action,omitempty"`
	Result    string    `json:"result,omitempty"`
	StartTime time.Time `json:"startTime,omitempty"`
	EndTime   time.Time `json:"endTime,omitempty"`
}

// AuditEventStats represents statistics about audit events
type AuditEventStats struct {
	TotalEvents    int            `json:"totalEvents"`
	EventsByType   map[string]int `json:"eventsByType"`
	EventsByResult map[string]int `json:"eventsByResult"`
	EventsByUser   map[string]int `json:"eventsByUser"`
}

// LogAccessDenied logs an access denied event
func (al *AuditLogger) LogAccessDenied(userID, username, resource, action, ipAddress, userAgent string) {
	al.LogEvent(&AuditEvent{
		EventType: EventTypeAccessDenied,
		UserID:    userID,
		Username:  username,
		Resource:  resource,
		Action:    action,
		Result:    ResultDenied,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details: map[string]interface{}{
			"reason": "insufficient permissions",
		},
		Timestamp: time.Now(),
	})
}

// LogPermissionGranted logs a permission granted event
func (al *AuditLogger) LogPermissionGranted(granterID, granterUsername, targetUserID, targetUsername, permission string) {
	al.LogEvent(&AuditEvent{
		EventType: EventTypePermissionGranted,
		UserID:    granterID,
		Username:  granterUsername,
		Resource:  "permissions",
		Action:    "grant",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"targetUserId":   targetUserID,
			"targetUsername": targetUsername,
			"permission":     permission,
		},
		Timestamp: time.Now(),
	})
}

// LogPermissionRevoked logs a permission revoked event
func (al *AuditLogger) LogPermissionRevoked(revokerID, revokerUsername, targetUserID, targetUsername, permission string) {
	al.LogEvent(&AuditEvent{
		EventType: EventTypePermissionRevoked,
		UserID:    revokerID,
		Username:  revokerUsername,
		Resource:  "permissions",
		Action:    "revoke",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"targetUserId":   targetUserID,
			"targetUsername": targetUsername,
			"permission":     permission,
		},
		Timestamp: time.Now(),
	})
}
