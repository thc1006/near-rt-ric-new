/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"fmt"
	"sync"
	"time"
)

// SecurityAlertManager manages security alerts
type SecurityAlertManager struct {
	alerts    map[string]*SecurityAlert
	mutex     sync.RWMutex
	maxAlerts int
}

// NewSecurityAlertManager creates a new security alert manager
func NewSecurityAlertManager() *SecurityAlertManager {
	return &SecurityAlertManager{
		alerts:    make(map[string]*SecurityAlert),
		maxAlerts: 1000,
	}
}

// CreateAlert creates a new security alert
func (sam *SecurityAlertManager) CreateAlert(alert *SecurityAlert) error {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	if alert.ID == "" {
		alert.ID = sam.generateAlertID()
	}

	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	if alert.Status == "" {
		alert.Status = "active"
	}

	sam.alerts[alert.ID] = alert

	// Maintain max alerts limit
	if len(sam.alerts) > sam.maxAlerts {
		sam.cleanupOldAlerts()
	}

	return nil
}

// GetAlert retrieves an alert by ID
func (sam *SecurityAlertManager) GetAlert(alertID string) (*SecurityAlert, error) {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	alert, exists := sam.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert not found")
	}

	return alert, nil
}

// GetAllAlerts retrieves all alerts
func (sam *SecurityAlertManager) GetAllAlerts() []*SecurityAlert {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	alerts := make([]*SecurityAlert, 0, len(sam.alerts))
	for _, alert := range sam.alerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// GetActiveAlerts retrieves all active alerts
func (sam *SecurityAlertManager) GetActiveAlerts() []*SecurityAlert {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	var activeAlerts []*SecurityAlert
	for _, alert := range sam.alerts {
		if alert.Status == "active" {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts
}

// GetAlertsByType retrieves alerts by type
func (sam *SecurityAlertManager) GetAlertsByType(alertType string) []*SecurityAlert {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	var typeAlerts []*SecurityAlert
	for _, alert := range sam.alerts {
		if alert.Type == alertType {
			typeAlerts = append(typeAlerts, alert)
		}
	}

	return typeAlerts
}

// GetAlertsBySeverity retrieves alerts by severity
func (sam *SecurityAlertManager) GetAlertsBySeverity(severity string) []*SecurityAlert {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	var severityAlerts []*SecurityAlert
	for _, alert := range sam.alerts {
		if alert.Severity == severity {
			severityAlerts = append(severityAlerts, alert)
		}
	}

	return severityAlerts
}

// UpdateAlertStatus updates the status of an alert
func (sam *SecurityAlertManager) UpdateAlertStatus(alertID, status string) error {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	alert, exists := sam.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found")
	}

	alert.Status = status
	return nil
}

// AcknowledgeAlert acknowledges an alert
func (sam *SecurityAlertManager) AcknowledgeAlert(alertID, acknowledgedBy string) error {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	alert, exists := sam.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found")
	}

	alert.Status = "acknowledged"
	if alert.Details == nil {
		alert.Details = make(map[string]interface{})
	}
	alert.Details["acknowledgedBy"] = acknowledgedBy
	alert.Details["acknowledgedAt"] = time.Now()

	return nil
}

// ResolveAlert resolves an alert
func (sam *SecurityAlertManager) ResolveAlert(alertID, resolvedBy, resolution string) error {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	alert, exists := sam.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found")
	}

	alert.Status = "resolved"
	if alert.Details == nil {
		alert.Details = make(map[string]interface{})
	}
	alert.Details["resolvedBy"] = resolvedBy
	alert.Details["resolvedAt"] = time.Now()
	alert.Details["resolution"] = resolution

	return nil
}

// DeleteAlert deletes an alert
func (sam *SecurityAlertManager) DeleteAlert(alertID string) error {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	if _, exists := sam.alerts[alertID]; !exists {
		return fmt.Errorf("alert not found")
	}

	delete(sam.alerts, alertID)
	return nil
}

// GetAlertStats returns statistics about alerts
func (sam *SecurityAlertManager) GetAlertStats() map[string]interface{} {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	stats := map[string]interface{}{
		"total":  len(sam.alerts),
		"byType": make(map[string]int),
		"bySeverity": make(map[string]int),
		"byStatus": make(map[string]int),
	}

	byType := stats["byType"].(map[string]int)
	bySeverity := stats["bySeverity"].(map[string]int)
	byStatus := stats["byStatus"].(map[string]int)

	for _, alert := range sam.alerts {
		byType[alert.Type]++
		bySeverity[alert.Severity]++
		byStatus[alert.Status]++
	}

	return stats
}

// cleanupOldAlerts removes old resolved alerts to maintain the limit
func (sam *SecurityAlertManager) cleanupOldAlerts() {
	// Find oldest resolved alerts
	var oldestResolved []*SecurityAlert
	for _, alert := range sam.alerts {
		if alert.Status == "resolved" {
			oldestResolved = append(oldestResolved, alert)
		}
	}

	// Sort by timestamp and remove oldest
	if len(oldestResolved) > 0 {
		// Simple cleanup - remove 10% of resolved alerts
		removeCount := len(oldestResolved) / 10
		if removeCount == 0 {
			removeCount = 1
		}

		for i := 0; i < removeCount && i < len(oldestResolved); i++ {
			delete(sam.alerts, oldestResolved[i].ID)
		}
	}
}

// generateAlertID generates a unique alert ID
func (sam *SecurityAlertManager) generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}