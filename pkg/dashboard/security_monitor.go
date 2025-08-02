/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// SecurityMonitor handles security event monitoring and anomaly detection
type SecurityMonitor struct {
	auditLogger         *AuditLogger
	alertManager        *SecurityAlertManager
	anomalyDetector     *AnomalyDetector
	complianceValidator *ComplianceValidator
	isRunning           bool
	stopChan            chan struct{}
	mutex               sync.RWMutex
}

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      string                 `json:"status"`
	Actions     []string               `json:"actions"`
}

// SecurityMetrics represents security-related metrics
type SecurityMetrics struct {
	LoginAttempts         int64     `json:"loginAttempts"`
	FailedLogins          int64     `json:"failedLogins"`
	SuccessfulLogins      int64     `json:"successfulLogins"`
	AccessDeniedEvents    int64     `json:"accessDeniedEvents"`
	PasswordChanges       int64     `json:"passwordChanges"`
	UserCreations         int64     `json:"userCreations"`
	RoleModifications     int64     `json:"roleModifications"`
	SuspiciousActivities  int64     `json:"suspiciousActivities"`
	ActiveSessions        int64     `json:"activeSessions"`
	LastUpdated           time.Time `json:"lastUpdated"`
}

// AnomalyPattern represents patterns for anomaly detection
type AnomalyPattern struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	EventType   string        `json:"eventType"`
	Threshold   int           `json:"threshold"`
	TimeWindow  time.Duration `json:"timeWindow"`
	Enabled     bool          `json:"enabled"`
}

// NewSecurityMonitor creates a new security monitor
func NewSecurityMonitor(auditLogger *AuditLogger) *SecurityMonitor {
	alertManager := NewSecurityAlertManager()
	anomalyDetector := NewAnomalyDetector()
	complianceValidator := NewComplianceValidator()

	return &SecurityMonitor{
		auditLogger:         auditLogger,
		alertManager:        alertManager,
		anomalyDetector:     anomalyDetector,
		complianceValidator: complianceValidator,
		stopChan:            make(chan struct{}),
	}
}

// Start starts the security monitoring
func (sm *SecurityMonitor) Start(ctx context.Context) error {
	sm.mutex.Lock()
	if sm.isRunning {
		sm.mutex.Unlock()
		return fmt.Errorf("security monitor is already running")
	}
	sm.isRunning = true
	sm.mutex.Unlock()

	log.Println("Starting security monitor")

	// Start monitoring routines
	go sm.monitorSecurityEvents(ctx)
	go sm.detectAnomalies(ctx)
	go sm.validateCompliance(ctx)
	go sm.generateSecurityReports(ctx)

	return nil
}

// Stop stops the security monitoring
func (sm *SecurityMonitor) Stop() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if !sm.isRunning {
		return
	}

	log.Println("Stopping security monitor")
	close(sm.stopChan)
	sm.isRunning = false
}

// monitorSecurityEvents monitors security events from audit logs
func (sm *SecurityMonitor) monitorSecurityEvents(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.processSecurityEvents()
		}
	}
}

// processSecurityEvents processes recent security events
func (sm *SecurityMonitor) processSecurityEvents() {
	// Get recent security events
	events := sm.auditLogger.GetSecurityEvents(100)
	
	for _, event := range events {
		// Check for suspicious patterns
		if sm.isSuspiciousEvent(&event) {
			alert := &SecurityAlert{
				ID:          sm.generateAlertID(),
				Type:        "suspicious_activity",
				Severity:    "medium",
				Title:       "Suspicious Activity Detected",
				Description: fmt.Sprintf("Suspicious %s activity from user %s", event.EventType, event.Username),
				Source:      "security_monitor",
				Details: map[string]interface{}{
					"eventType": event.EventType,
					"userId":    event.UserID,
					"username":  event.Username,
					"ipAddress": event.IPAddress,
					"resource":  event.Resource,
					"action":    event.Action,
				},
				Timestamp: time.Now(),
				Status:    "active",
				Actions:   []string{"investigate", "block_user", "require_mfa"},
			}
			
			sm.alertManager.CreateAlert(alert)
		}

		// Check for failed login patterns
		if event.EventType == EventTypeLoginFailed {
			sm.checkFailedLoginPattern(&event)
		}

		// Check for privilege escalation
		if event.EventType == EventTypePermissionGranted {
			sm.checkPrivilegeEscalation(&event)
		}
	}
}

// detectAnomalies detects anomalous behavior patterns
func (sm *SecurityMonitor) detectAnomalies(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.runAnomalyDetection()
		}
	}
}

// runAnomalyDetection runs anomaly detection algorithms
func (sm *SecurityMonitor) runAnomalyDetection() {
	patterns := sm.anomalyDetector.GetEnabledPatterns()
	
	for _, pattern := range patterns {
		if sm.detectPattern(&pattern) {
			alert := &SecurityAlert{
				ID:          sm.generateAlertID(),
				Type:        "anomaly_detected",
				Severity:    "high",
				Title:       fmt.Sprintf("Anomaly Detected: %s", pattern.Name),
				Description: pattern.Description,
				Source:      "anomaly_detector",
				Details: map[string]interface{}{
					"pattern":    pattern.Name,
					"threshold":  pattern.Threshold,
					"timeWindow": pattern.TimeWindow.String(),
				},
				Timestamp: time.Now(),
				Status:    "active",
				Actions:   []string{"investigate", "increase_monitoring", "notify_admin"},
			}
			
			sm.alertManager.CreateAlert(alert)
		}
	}
}

// validateCompliance validates security compliance
func (sm *SecurityMonitor) validateCompliance(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.runComplianceValidation()
		}
	}
}

// runComplianceValidation runs compliance validation checks
func (sm *SecurityMonitor) runComplianceValidation() {
	violations := sm.complianceValidator.ValidateCompliance()
	
	for _, violation := range violations {
		alert := &SecurityAlert{
			ID:          sm.generateAlertID(),
			Type:        "compliance_violation",
			Severity:    violation.Severity,
			Title:       fmt.Sprintf("Compliance Violation: %s", violation.Rule),
			Description: violation.Description,
			Source:      "compliance_validator",
			Details: map[string]interface{}{
				"rule":        violation.Rule,
				"standard":    violation.Standard,
				"requirement": violation.Requirement,
			},
			Timestamp: time.Now(),
			Status:    "active",
			Actions:   []string{"remediate", "document_exception", "notify_compliance"},
		}
		
		sm.alertManager.CreateAlert(alert)
	}
}

// generateSecurityReports generates periodic security reports
func (sm *SecurityMonitor) generateSecurityReports(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.generateDailySecurityReport()
		}
	}
}

// generateDailySecurityReport generates a daily security report
func (sm *SecurityMonitor) generateDailySecurityReport() {
	report := sm.createSecurityReport()
	log.Printf("Daily Security Report: %+v", report)
	
	// In production, this would be sent to administrators
	// or stored in a reporting system
}

// GetSecurityMetrics returns current security metrics
func (sm *SecurityMonitor) GetSecurityMetrics() *SecurityMetrics {
	stats := sm.auditLogger.GetEventStats()
	
	return &SecurityMetrics{
		LoginAttempts:        int64(stats.EventsByType[EventTypeLogin] + stats.EventsByType[EventTypeLoginFailed]),
		FailedLogins:         int64(stats.EventsByType[EventTypeLoginFailed]),
		SuccessfulLogins:     int64(stats.EventsByType[EventTypeLogin]),
		AccessDeniedEvents:   int64(stats.EventsByType[EventTypeAccessDenied]),
		PasswordChanges:      int64(stats.EventsByType[EventTypePasswordChanged]),
		UserCreations:        int64(stats.EventsByType[EventTypeUserCreated]),
		RoleModifications:    int64(stats.EventsByType[EventTypeRoleCreated] + stats.EventsByType[EventTypeRoleUpdated]),
		SuspiciousActivities: int64(len(sm.alertManager.GetAlertsByType("suspicious_activity"))),
		ActiveSessions:       0, // Would be calculated from session manager
		LastUpdated:          time.Now(),
	}
}

// GetSecurityAlerts returns current security alerts
func (sm *SecurityMonitor) GetSecurityAlerts() []*SecurityAlert {
	return sm.alertManager.GetActiveAlerts()
}

// GetComplianceStatus returns current compliance status
func (sm *SecurityMonitor) GetComplianceStatus() *ComplianceStatus {
	return sm.complianceValidator.GetComplianceStatus()
}

// Helper methods

func (sm *SecurityMonitor) isSuspiciousEvent(event *AuditEvent) bool {
	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"multiple_failed_logins",
		"unusual_access_time",
		"privilege_escalation",
		"bulk_operations",
	}

	// Simple pattern matching - in production, use more sophisticated ML algorithms
	if event.Result == ResultFailure || event.Result == ResultDenied {
		return true
	}

	// Check for unusual IP addresses or user agents
	if event.IPAddress != "" && sm.isUnusualIP(event.IPAddress) {
		return true
	}

	return false
}

func (sm *SecurityMonitor) checkFailedLoginPattern(event *AuditEvent) {
	// Check for multiple failed logins from same IP or user
	recentEvents := sm.auditLogger.GetEventsByUser(event.UserID, 10)
	
	failedCount := 0
	for _, e := range recentEvents {
		if e.EventType == EventTypeLoginFailed && 
		   time.Since(e.Timestamp) < 15*time.Minute {
			failedCount++
		}
	}

	if failedCount >= 5 {
		alert := &SecurityAlert{
			ID:          sm.generateAlertID(),
			Type:        "brute_force_attack",
			Severity:    "high",
			Title:       "Potential Brute Force Attack",
			Description: fmt.Sprintf("Multiple failed login attempts for user %s", event.Username),
			Source:      "security_monitor",
			Details: map[string]interface{}{
				"userId":      event.UserID,
				"username":    event.Username,
				"ipAddress":   event.IPAddress,
				"failedCount": failedCount,
			},
			Timestamp: time.Now(),
			Status:    "active",
			Actions:   []string{"block_ip", "lock_account", "notify_admin"},
		}
		
		sm.alertManager.CreateAlert(alert)
	}
}

func (sm *SecurityMonitor) checkPrivilegeEscalation(event *AuditEvent) {
	// Check for unusual privilege grants
	if details, ok := event.Details["permission"].(string); ok {
		if details == "system:*" || details == "admin" {
			alert := &SecurityAlert{
				ID:          sm.generateAlertID(),
				Type:        "privilege_escalation",
				Severity:    "critical",
				Title:       "Privilege Escalation Detected",
				Description: fmt.Sprintf("High-level privileges granted to user %s", event.Username),
				Source:      "security_monitor",
				Details: map[string]interface{}{
					"userId":     event.UserID,
					"username":   event.Username,
					"permission": details,
					"grantedBy":  event.UserID,
				},
				Timestamp: time.Now(),
				Status:    "active",
				Actions:   []string{"investigate", "verify_authorization", "notify_admin"},
			}
			
			sm.alertManager.CreateAlert(alert)
		}
	}
}

func (sm *SecurityMonitor) detectPattern(pattern *AnomalyPattern) bool {
	// Get events matching the pattern
	filter := &AuditEventFilter{
		EventType: pattern.EventType,
		StartTime: time.Now().Add(-pattern.TimeWindow),
	}
	
	events := sm.auditLogger.GetEvents(filter)
	return len(events) >= pattern.Threshold
}

func (sm *SecurityMonitor) isUnusualIP(ipAddress string) bool {
	// Simple check for private vs public IPs
	// In production, use IP reputation services and geolocation
	return false
}

func (sm *SecurityMonitor) generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}

func (sm *SecurityMonitor) createSecurityReport() map[string]interface{} {
	metrics := sm.GetSecurityMetrics()
	alerts := sm.GetSecurityAlerts()
	compliance := sm.GetComplianceStatus()

	return map[string]interface{}{
		"date":       time.Now().Format("2006-01-02"),
		"metrics":    metrics,
		"alerts":     len(alerts),
		"compliance": compliance,
	}
}