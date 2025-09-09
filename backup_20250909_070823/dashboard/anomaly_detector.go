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

// AnomalyDetector detects anomalous behavior patterns
type AnomalyDetector struct {
	patterns map[string]*AnomalyPattern
	mutex    sync.RWMutex
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector() *AnomalyDetector {
	detector := &AnomalyDetector{
		patterns: make(map[string]*AnomalyPattern),
	}

	// Initialize with default patterns
	detector.initializeDefaultPatterns()

	return detector
}

// initializeDefaultPatterns initializes default anomaly detection patterns
func (ad *AnomalyDetector) initializeDefaultPatterns() {
	defaultPatterns := []*AnomalyPattern{
		{
			Name:        "excessive_failed_logins",
			Description: "Excessive failed login attempts detected",
			EventType:   EventTypeLoginFailed,
			Threshold:   10,
			TimeWindow:  15 * time.Minute,
			Enabled:     true,
		},
		{
			Name:        "unusual_access_denied",
			Description: "Unusual number of access denied events",
			EventType:   EventTypeAccessDenied,
			Threshold:   20,
			TimeWindow:  30 * time.Minute,
			Enabled:     true,
		},
		{
			Name:        "bulk_user_creation",
			Description: "Bulk user creation activity detected",
			EventType:   EventTypeUserCreated,
			Threshold:   5,
			TimeWindow:  10 * time.Minute,
			Enabled:     true,
		},
		{
			Name:        "rapid_role_changes",
			Description: "Rapid role modification activity detected",
			EventType:   EventTypeRoleUpdated,
			Threshold:   3,
			TimeWindow:  5 * time.Minute,
			Enabled:     true,
		},
		{
			Name:        "excessive_permission_grants",
			Description: "Excessive permission grants detected",
			EventType:   EventTypePermissionGranted,
			Threshold:   10,
			TimeWindow:  30 * time.Minute,
			Enabled:     true,
		},
		{
			Name:        "unusual_logout_pattern",
			Description: "Unusual logout pattern detected",
			EventType:   EventTypeLogout,
			Threshold:   50,
			TimeWindow:  1 * time.Hour,
			Enabled:     false, // Disabled by default as it might be noisy
		},
		{
			Name:        "service_account_abuse",
			Description: "Potential service account abuse detected",
			EventType:   EventTypeServiceAccountCreated,
			Threshold:   3,
			TimeWindow:  1 * time.Hour,
			Enabled:     true,
		},
	}

	for _, pattern := range defaultPatterns {
		ad.patterns[pattern.Name] = pattern
	}
}

// AddPattern adds a new anomaly detection pattern
func (ad *AnomalyDetector) AddPattern(pattern *AnomalyPattern) {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	ad.patterns[pattern.Name] = pattern
}

// UpdatePattern updates an existing anomaly detection pattern
func (ad *AnomalyDetector) UpdatePattern(name string, pattern *AnomalyPattern) error {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	if _, exists := ad.patterns[name]; !exists {
		return fmt.Errorf("pattern %s not found", name)
	}

	pattern.Name = name // Ensure name consistency
	ad.patterns[name] = pattern
	return nil
}

// RemovePattern removes an anomaly detection pattern
func (ad *AnomalyDetector) RemovePattern(name string) error {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	if _, exists := ad.patterns[name]; !exists {
		return fmt.Errorf("pattern %s not found", name)
	}

	delete(ad.patterns, name)
	return nil
}

// GetPattern retrieves a specific pattern
func (ad *AnomalyDetector) GetPattern(name string) (*AnomalyPattern, error) {
	ad.mutex.RLock()
	defer ad.mutex.RUnlock()

	pattern, exists := ad.patterns[name]
	if !exists {
		return nil, fmt.Errorf("pattern %s not found", name)
	}

	return pattern, nil
}

// GetAllPatterns retrieves all patterns
func (ad *AnomalyDetector) GetAllPatterns() []*AnomalyPattern {
	ad.mutex.RLock()
	defer ad.mutex.RUnlock()

	patterns := make([]*AnomalyPattern, 0, len(ad.patterns))
	for _, pattern := range ad.patterns {
		patterns = append(patterns, pattern)
	}

	return patterns
}

// GetEnabledPatterns retrieves all enabled patterns
func (ad *AnomalyDetector) GetEnabledPatterns() []*AnomalyPattern {
	ad.mutex.RLock()
	defer ad.mutex.RUnlock()

	var enabledPatterns []*AnomalyPattern
	for _, pattern := range ad.patterns {
		if pattern.Enabled {
			enabledPatterns = append(enabledPatterns, pattern)
		}
	}

	return enabledPatterns
}

// EnablePattern enables a pattern
func (ad *AnomalyDetector) EnablePattern(name string) error {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	pattern, exists := ad.patterns[name]
	if !exists {
		return fmt.Errorf("pattern %s not found", name)
	}

	pattern.Enabled = true
	return nil
}

// DisablePattern disables a pattern
func (ad *AnomalyDetector) DisablePattern(name string) error {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	pattern, exists := ad.patterns[name]
	if !exists {
		return fmt.Errorf("pattern %s not found", name)
	}

	pattern.Enabled = false
	return nil
}

// GetPatternStats returns statistics about patterns
func (ad *AnomalyDetector) GetPatternStats() map[string]interface{} {
	ad.mutex.RLock()
	defer ad.mutex.RUnlock()

	stats := map[string]interface{}{
		"total":   len(ad.patterns),
		"enabled": 0,
		"disabled": 0,
		"byEventType": make(map[string]int),
	}

	byEventType := stats["byEventType"].(map[string]int)
	enabled := 0
	disabled := 0

	for _, pattern := range ad.patterns {
		if pattern.Enabled {
			enabled++
		} else {
			disabled++
		}
		byEventType[pattern.EventType]++
	}

	stats["enabled"] = enabled
	stats["disabled"] = disabled

	return stats
}