/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// SecurityHandlers handles security monitoring HTTP requests
type SecurityHandlers struct {
	securityMonitor *SecurityMonitor
	auditLogger     *AuditLogger
}

// NewSecurityHandlers creates new security handlers
func NewSecurityHandlers(securityMonitor *SecurityMonitor, auditLogger *AuditLogger) *SecurityHandlers {
	return &SecurityHandlers{
		securityMonitor: securityMonitor,
		auditLogger:     auditLogger,
	}
}

// GetSecurityMetricsHandler returns security metrics
func (sh *SecurityHandlers) GetSecurityMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := sh.securityMonitor.GetSecurityMetrics()
	sh.respondWithJSON(w, http.StatusOK, metrics)
}

// GetSecurityAlertsHandler returns security alerts
func (sh *SecurityHandlers) GetSecurityAlertsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	
	var alerts []*SecurityAlert
	
	if alertType := query.Get("type"); alertType != "" {
		alerts = sh.securityMonitor.alertManager.GetAlertsByType(alertType)
	} else if severity := query.Get("severity"); severity != "" {
		alerts = sh.securityMonitor.alertManager.GetAlertsBySeverity(severity)
	} else if status := query.Get("status"); status != "" {
		if status == "active" {
			alerts = sh.securityMonitor.alertManager.GetActiveAlerts()
		} else {
			// Filter by status
			allAlerts := sh.securityMonitor.alertManager.GetAllAlerts()
			for _, alert := range allAlerts {
				if alert.Status == status {
					alerts = append(alerts, alert)
				}
			}
		}
	} else {
		alerts = sh.securityMonitor.GetSecurityAlerts()
	}

	sh.respondWithJSON(w, http.StatusOK, alerts)
}

// GetSecurityAlertHandler returns a specific security alert
func (sh *SecurityHandlers) GetSecurityAlertHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["alertId"]

	alert, err := sh.securityMonitor.alertManager.GetAlert(alertID)
	if err != nil {
		sh.respondWithError(w, http.StatusNotFound, "Alert not found", err.Error())
		return
	}

	sh.respondWithJSON(w, http.StatusOK, alert)
}

// AcknowledgeSecurityAlertHandler acknowledges a security alert
func (sh *SecurityHandlers) AcknowledgeSecurityAlertHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sh.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	alertID := vars["alertId"]

	if err := sh.securityMonitor.alertManager.AcknowledgeAlert(alertID, claims.Username); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Failed to acknowledge alert", err.Error())
		return
	}

	sh.auditLogger.LogEvent(&AuditEvent{
		EventType: "security_alert_acknowledged",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "security_alerts",
		Action:    "acknowledge",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"alertId": alertID,
		},
		Timestamp: time.Now(),
	})

	sh.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Alert acknowledged successfully"})
}

// ResolveSecurityAlertHandler resolves a security alert
func (sh *SecurityHandlers) ResolveSecurityAlertHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sh.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	alertID := vars["alertId"]

	var request struct {
		Resolution string `json:"resolution"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := sh.securityMonitor.alertManager.ResolveAlert(alertID, claims.Username, request.Resolution); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Failed to resolve alert", err.Error())
		return
	}

	sh.auditLogger.LogEvent(&AuditEvent{
		EventType: "security_alert_resolved",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "security_alerts",
		Action:    "resolve",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"alertId":    alertID,
			"resolution": request.Resolution,
		},
		Timestamp: time.Now(),
	})

	sh.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Alert resolved successfully"})
}

// GetAlertStatsHandler returns alert statistics
func (sh *SecurityHandlers) GetAlertStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := sh.securityMonitor.alertManager.GetAlertStats()
	sh.respondWithJSON(w, http.StatusOK, stats)
}

// GetComplianceStatusHandler returns compliance status
func (sh *SecurityHandlers) GetComplianceStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := sh.securityMonitor.GetComplianceStatus()
	sh.respondWithJSON(w, http.StatusOK, status)
}

// GetComplianceRulesHandler returns compliance rules
func (sh *SecurityHandlers) GetComplianceRulesHandler(w http.ResponseWriter, r *http.Request) {
	rules := sh.securityMonitor.complianceValidator.GetComplianceRules()
	sh.respondWithJSON(w, http.StatusOK, rules)
}

// UpdateComplianceRuleHandler updates a compliance rule
func (sh *SecurityHandlers) UpdateComplianceRuleHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sh.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	ruleID := vars["ruleId"]

	var rule ComplianceRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := sh.securityMonitor.complianceValidator.UpdateComplianceRule(ruleID, &rule); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Failed to update compliance rule", err.Error())
		return
	}

	sh.auditLogger.LogEvent(&AuditEvent{
		EventType: "compliance_rule_updated",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "compliance_rules",
		Action:    "update",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"ruleId": ruleID,
		},
		Timestamp: time.Now(),
	})

	sh.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Compliance rule updated successfully"})
}

// GetAnomalyPatternsHandler returns anomaly detection patterns
func (sh *SecurityHandlers) GetAnomalyPatternsHandler(w http.ResponseWriter, r *http.Request) {
	patterns := sh.securityMonitor.anomalyDetector.GetAllPatterns()
	sh.respondWithJSON(w, http.StatusOK, patterns)
}

// CreateAnomalyPatternHandler creates a new anomaly detection pattern
func (sh *SecurityHandlers) CreateAnomalyPatternHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sh.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	var pattern AnomalyPattern
	if err := json.NewDecoder(r.Body).Decode(&pattern); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	sh.securityMonitor.anomalyDetector.AddPattern(&pattern)

	sh.auditLogger.LogEvent(&AuditEvent{
		EventType: "anomaly_pattern_created",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "anomaly_patterns",
		Action:    "create",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"patternName": pattern.Name,
		},
		Timestamp: time.Now(),
	})

	sh.respondWithJSON(w, http.StatusCreated, map[string]string{"message": "Anomaly pattern created successfully"})
}

// UpdateAnomalyPatternHandler updates an anomaly detection pattern
func (sh *SecurityHandlers) UpdateAnomalyPatternHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sh.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	patternName := vars["patternName"]

	var pattern AnomalyPattern
	if err := json.NewDecoder(r.Body).Decode(&pattern); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := sh.securityMonitor.anomalyDetector.UpdatePattern(patternName, &pattern); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Failed to update anomaly pattern", err.Error())
		return
	}

	sh.auditLogger.LogEvent(&AuditEvent{
		EventType: "anomaly_pattern_updated",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "anomaly_patterns",
		Action:    "update",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"patternName": patternName,
		},
		Timestamp: time.Now(),
	})

	sh.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Anomaly pattern updated successfully"})
}

// EnableAnomalyPatternHandler enables an anomaly detection pattern
func (sh *SecurityHandlers) EnableAnomalyPatternHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sh.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	patternName := vars["patternName"]

	if err := sh.securityMonitor.anomalyDetector.EnablePattern(patternName); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Failed to enable anomaly pattern", err.Error())
		return
	}

	sh.auditLogger.LogEvent(&AuditEvent{
		EventType: "anomaly_pattern_enabled",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "anomaly_patterns",
		Action:    "enable",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"patternName": patternName,
		},
		Timestamp: time.Now(),
	})

	sh.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Anomaly pattern enabled successfully"})
}

// DisableAnomalyPatternHandler disables an anomaly detection pattern
func (sh *SecurityHandlers) DisableAnomalyPatternHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sh.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	vars := mux.Vars(r)
	patternName := vars["patternName"]

	if err := sh.securityMonitor.anomalyDetector.DisablePattern(patternName); err != nil {
		sh.respondWithError(w, http.StatusBadRequest, "Failed to disable anomaly pattern", err.Error())
		return
	}

	sh.auditLogger.LogEvent(&AuditEvent{
		EventType: "anomaly_pattern_disabled",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "anomaly_patterns",
		Action:    "disable",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"patternName": patternName,
		},
		Timestamp: time.Now(),
	})

	sh.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Anomaly pattern disabled successfully"})
}

// GetPatternStatsHandler returns anomaly pattern statistics
func (sh *SecurityHandlers) GetPatternStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := sh.securityMonitor.anomalyDetector.GetPatternStats()
	sh.respondWithJSON(w, http.StatusOK, stats)
}

// RunComplianceValidationHandler triggers a compliance validation
func (sh *SecurityHandlers) RunComplianceValidationHandler(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		sh.respondWithError(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}

	violations := sh.securityMonitor.complianceValidator.ValidateCompliance()

	sh.auditLogger.LogEvent(&AuditEvent{
		EventType: "compliance_validation_run",
		UserID:    claims.UserID,
		Username:  claims.Username,
		Resource:  "compliance",
		Action:    "validate",
		Result:    ResultSuccess,
		Details: map[string]interface{}{
			"violationsFound": len(violations),
		},
		Timestamp: time.Now(),
	})

	response := map[string]interface{}{
		"message":    "Compliance validation completed",
		"violations": violations,
		"count":      len(violations),
	}

	sh.respondWithJSON(w, http.StatusOK, response)
}

// Helper methods

func (sh *SecurityHandlers) respondWithError(w http.ResponseWriter, statusCode int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   message,
		"details": details,
		"status":  statusCode,
	}

	json.NewEncoder(w).Encode(response)
}

func (sh *SecurityHandlers) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}