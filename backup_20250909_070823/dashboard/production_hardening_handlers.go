/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// ProductionHardeningHandlers provides HTTP handlers for production hardening features
type ProductionHardeningHandlers struct {
	hardeningManager *ProductionHardeningManager
	auditLogger      *AuditLogger
}

// NewProductionHardeningHandlers creates new production hardening handlers
func NewProductionHardeningHandlers(hardeningManager *ProductionHardeningManager, auditLogger *AuditLogger) *ProductionHardeningHandlers {
	return &ProductionHardeningHandlers{
		hardeningManager: hardeningManager,
		auditLogger:      auditLogger,
	}
}

// GetMetrics returns production hardening metrics
func (phh *ProductionHardeningHandlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Execute with production hardening
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "get_metrics", 
		func(ctx context.Context) (interface{}, error) {
			return phh.hardeningManager.GetMetrics(), nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get metrics: %v", err), http.StatusInternalServerError)
		return
	}
	
	metrics := result.(*ProductionMetrics)
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode metrics: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Log audit event
	if phh.auditLogger != nil {
		phh.auditLogger.LogEvent(AuditEvent{
			Timestamp: time.Now(),
			UserID:    getUserIDFromContext(ctx),
			Action:    "VIEW_PRODUCTION_METRICS",
			Resource:  "production_hardening",
			Success:   true,
			Details: map[string]interface{}{
				"metrics_retrieved": true,
			},
		})
	}
}

// GetConnectionPoolMetrics returns connection pool metrics
func (phh *ProductionHardeningHandlers) GetConnectionPoolMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "get_connection_metrics", 
		func(ctx context.Context) (interface{}, error) {
			if phh.hardeningManager.connectionPoolManager == nil {
				return nil, fmt.Errorf("connection pool manager not available")
			}
			return phh.hardeningManager.connectionPoolManager.GetMetrics(), nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get connection pool metrics: %v", err), http.StatusInternalServerError)
		return
	}
	
	metrics := result.(*ConnectionPoolMetrics)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GetCircuitBreakerStatus returns circuit breaker status
func (phh *ProductionHardeningHandlers) GetCircuitBreakerStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "get_circuit_breaker_status", 
		func(ctx context.Context) (interface{}, error) {
			if phh.hardeningManager.circuitBreakerManager == nil {
				return nil, fmt.Errorf("circuit breaker manager not available")
			}
			return phh.hardeningManager.circuitBreakerManager.GetStatistics(), nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get circuit breaker status: %v", err), http.StatusInternalServerError)
		return
	}
	
	stats := result.(map[string]*CircuitBreakerStatistics)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ResetCircuitBreaker resets a specific circuit breaker
func (phh *ProductionHardeningHandlers) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]
	
	if name == "" {
		http.Error(w, "Circuit breaker name is required", http.StatusBadRequest)
		return
	}
	
	_, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "reset_circuit_breaker", 
		func(ctx context.Context) (interface{}, error) {
			if phh.hardeningManager.circuitBreakerManager == nil {
				return nil, fmt.Errorf("circuit breaker manager not available")
			}
			
			cb := phh.hardeningManager.circuitBreakerManager.GetCircuitBreaker(name)
			cb.Reset()
			return map[string]string{"status": "reset", "circuit_breaker": name}, nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset circuit breaker: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
	
	// Log audit event
	if phh.auditLogger != nil {
		phh.auditLogger.LogEvent(AuditEvent{
			Timestamp: time.Now(),
			UserID:    getUserIDFromContext(ctx),
			Action:    "RESET_CIRCUIT_BREAKER",
			Resource:  fmt.Sprintf("circuit_breaker:%s", name),
			Success:   true,
			Details: map[string]interface{}{
				"circuit_breaker_name": name,
			},
		})
	}
}

// GetServiceHealth returns service health status from graceful degradation
func (phh *ProductionHardeningHandlers) GetServiceHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "get_service_health", 
		func(ctx context.Context) (interface{}, error) {
			if phh.hardeningManager.gracefulDegradationMgr == nil {
				return nil, fmt.Errorf("graceful degradation manager not available")
			}
			return phh.hardeningManager.gracefulDegradationMgr.GetAllServiceHealth(), nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get service health: %v", err), http.StatusInternalServerError)
		return
	}
	
	health := result.(map[string]*ServiceHealth)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// GetResourceUsage returns current resource usage
func (phh *ProductionHardeningHandlers) GetResourceUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "get_resource_usage", 
		func(ctx context.Context) (interface{}, error) {
			if phh.hardeningManager.connectionPoolManager == nil || 
			   phh.hardeningManager.connectionPoolManager.resourceMonitor == nil {
				return nil, fmt.Errorf("resource monitor not available")
			}
			return phh.hardeningManager.connectionPoolManager.resourceMonitor.GetResourceUsage(), nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resource usage: %v", err), http.StatusInternalServerError)
		return
	}
	
	usage := result.(ResourceUsage)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

// GetLoggingMetrics returns structured logging metrics
func (phh *ProductionHardeningHandlers) GetLoggingMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "get_logging_metrics", 
		func(ctx context.Context) (interface{}, error) {
			if phh.hardeningManager.structuredLogger == nil {
				return nil, fmt.Errorf("structured logger not available")
			}
			return phh.hardeningManager.structuredLogger.GetMetrics(), nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get logging metrics: %v", err), http.StatusInternalServerError)
		return
	}
	
	metrics := result.(*LoggingMetrics)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// TestConnection tests a connection with production hardening
func (phh *ProductionHardeningHandlers) TestConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse request
	var request struct {
		Component string `json:"component"`
		Address   string `json:"address"`
		Timeout   int    `json:"timeout,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if request.Component == "" || request.Address == "" {
		http.Error(w, "Component and address are required", http.StatusBadRequest)
		return
	}
	
	if request.Timeout == 0 {
		request.Timeout = 10 // Default 10 seconds
	}
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, request.Component, "test_connection", 
		func(ctx context.Context) (interface{}, error) {
			// Test connection using connection pool
			timeout := time.Duration(request.Timeout) * time.Second
			connCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			
			conn, err := phh.hardeningManager.GetConnection(connCtx, request.Component, request.Address)
			if err != nil {
				return nil, fmt.Errorf("connection test failed: %w", err)
			}
			
			// Return connection immediately
			if returnErr := phh.hardeningManager.ReturnConnection(request.Component, conn); returnErr != nil {
				// Log but don't fail the test
				log.Printf("Warning: Failed to return test connection: %v", returnErr)
			}
			
			return map[string]interface{}{
				"status":     "success",
				"component":  request.Component,
				"address":    request.Address,
				"connection_id": conn.id,
				"response_time": time.Since(conn.createdAt),
			}, nil
		})
	
	if err != nil {
		response := map[string]interface{}{
			"status":    "failure",
			"component": request.Component,
			"address":   request.Address,
			"error":     err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
	
	// Log audit event
	if phh.auditLogger != nil {
		phh.auditLogger.LogEvent(AuditEvent{
			Timestamp: time.Now(),
			UserID:    getUserIDFromContext(ctx),
			Action:    "TEST_CONNECTION",
			Resource:  fmt.Sprintf("connection:%s:%s", request.Component, request.Address),
			Success:   true,
			Details: map[string]interface{}{
				"component": request.Component,
				"address":   request.Address,
				"timeout":   request.Timeout,
			},
		})
	}
}

// GetErrorStatistics returns error handling statistics
func (phh *ProductionHardeningHandlers) GetErrorStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "get_error_statistics", 
		func(ctx context.Context) (interface{}, error) {
			if phh.hardeningManager.errorHandler == nil {
				return nil, fmt.Errorf("error handler not available")
			}
			return phh.hardeningManager.errorHandler.GetErrorStatistics(), nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get error statistics: %v", err), http.StatusInternalServerError)
		return
	}
	
	stats := result.(*ErrorStatistics)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetRecentErrors returns recent error records
func (phh *ProductionHardeningHandlers) GetRecentErrors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "get_recent_errors", 
		func(ctx context.Context) (interface{}, error) {
			if phh.hardeningManager.errorHandler == nil {
				return nil, fmt.Errorf("error handler not available")
			}
			
			// Get all error records and return the most recent ones
			filter := &ErrorFilter{
				Since: time.Now().Add(-24 * time.Hour), // Last 24 hours
			}
			
			errors := phh.hardeningManager.errorHandler.GetErrorRecords(filter)
			
			// Limit results
			if len(errors) > limit {
				errors = errors[:limit]
			}
			
			return errors, nil
		})
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get recent errors: %v", err), http.StatusInternalServerError)
		return
	}
	
	errors := result.([]*ErrorRecord)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(errors)
}

// HealthCheck provides a comprehensive health check endpoint
func (phh *ProductionHardeningHandlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	result, err := phh.hardeningManager.ExecuteWithHardening(ctx, "hardening", "health_check", 
		func(ctx context.Context) (interface{}, error) {
			health := map[string]interface{}{
				"status":    "healthy",
				"timestamp": time.Now(),
				"components": map[string]interface{}{},
			}
			
			overallHealthy := true
			
			// Check error handler
			if phh.hardeningManager.errorHandler != nil {
				stats := phh.hardeningManager.errorHandler.GetErrorStatistics()
				errorRate := float64(0)
				if stats.TotalErrors > 0 {
					errorRate = float64(stats.TotalErrors-stats.ResolvedErrors) / float64(stats.TotalErrors) * 100
				}
				
				componentHealth := "healthy"
				if errorRate > 20 {
					componentHealth = "degraded"
					overallHealthy = false
				}
				if errorRate > 50 {
					componentHealth = "unhealthy"
				}
				
				health["components"].(map[string]interface{})["error_handler"] = map[string]interface{}{
					"status":     componentHealth,
					"error_rate": errorRate,
					"total_errors": stats.TotalErrors,
					"resolved_errors": stats.ResolvedErrors,
				}
			}
			
			// Check circuit breakers
			if phh.hardeningManager.circuitBreakerManager != nil {
				cbStats := phh.hardeningManager.circuitBreakerManager.GetStatistics()
				openCircuits := 0
				totalCircuits := len(cbStats)
				
				for _, stats := range cbStats {
					if stats.State == "OPEN" {
						openCircuits++
					}
				}
				
				componentHealth := "healthy"
				if openCircuits > 0 {
					componentHealth = "degraded"
					if totalCircuits > 0 && float64(openCircuits)/float64(totalCircuits) > 0.5 {
						componentHealth = "unhealthy"
						overallHealthy = false
					}
				}
				
				health["components"].(map[string]interface{})["circuit_breakers"] = map[string]interface{}{
					"status":        componentHealth,
					"open_circuits": openCircuits,
					"total_circuits": totalCircuits,
				}
			}
			
			// Check connection pools
			if phh.hardeningManager.connectionPoolManager != nil {
				poolMetrics := phh.hardeningManager.connectionPoolManager.GetMetrics()
				
				componentHealth := "healthy"
				if poolMetrics.ConnectionFailures > 10 {
					componentHealth = "degraded"
				}
				if poolMetrics.ConnectionFailures > 50 {
					componentHealth = "unhealthy"
					overallHealthy = false
				}
				
				health["components"].(map[string]interface{})["connection_pools"] = map[string]interface{}{
					"status":             componentHealth,
					"active_connections": poolMetrics.ActiveConnections,
					"pooled_connections": poolMetrics.PooledConnections,
					"connection_failures": poolMetrics.ConnectionFailures,
				}
			}
			
			// Check resource usage
			if phh.hardeningManager.connectionPoolManager != nil && 
			   phh.hardeningManager.connectionPoolManager.resourceMonitor != nil {
				usage := phh.hardeningManager.connectionPoolManager.resourceMonitor.GetResourceUsage()
				
				componentHealth := "healthy"
				if usage.CPUUsage > 80 || usage.MemoryUsage > 85 {
					componentHealth = "degraded"
				}
				if usage.CPUUsage > 95 || usage.MemoryUsage > 95 {
					componentHealth = "unhealthy"
					overallHealthy = false
				}
				
				health["components"].(map[string]interface{})["resources"] = map[string]interface{}{
					"status":       componentHealth,
					"cpu_usage":    usage.CPUUsage,
					"memory_usage": usage.MemoryUsage,
					"network_usage": usage.NetworkUsage,
					"disk_usage":   usage.DiskUsage,
				}
			}
			
			if !overallHealthy {
				health["status"] = "degraded"
			}
			
			return health, nil
		})
	
	if err != nil {
		// Return unhealthy status
		health := map[string]interface{}{
			"status":    "unhealthy",
			"timestamp": time.Now(),
			"error":     err.Error(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(health)
		return
	}
	
	health := result.(map[string]interface{})
	
	// Set appropriate HTTP status
	statusCode := http.StatusOK
	if status, ok := health["status"].(string); ok {
		switch status {
		case "degraded":
			statusCode = http.StatusPartialContent
		case "unhealthy":
			statusCode = http.StatusServiceUnavailable
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// Helper function to get user ID from context
func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "system" // Default for system operations
}
