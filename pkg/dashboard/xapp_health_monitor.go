// Package dashboard provides xApp health monitoring capabilities
// for the O-RAN Near-RT RIC platform
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// XAppHealthMonitor monitors the health of xApps in the RIC platform
type XAppHealthMonitor struct {
	xapps          map[string]*XAppHealth
	healthCheckers map[string]*HealthChecker
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	checkInterval  time.Duration
	timeout        time.Duration
	client         *http.Client
}

// XAppHealth represents the health information of an xApp
type XAppHealth struct {
	XAppID           string                     `json:"xappId"`
	Name             string                     `json:"name"`
	Namespace        string                     `json:"namespace"`
	Version          string                     `json:"version"`
	Status           HealthStatus              `json:"status"`
	LastCheckTime    time.Time                  `json:"lastCheckTime"`
	Uptime           time.Duration              `json:"uptime"`
	RestartCount     int                        `json:"restartCount"`
	HealthData       map[string]interface{}     `json:"healthData,omitempty"`
	Metrics          *XAppHealthMetrics         `json:"metrics,omitempty"`
}

// NOTE: HealthStatus type moved to types.go to avoid redeclaration

// XAppHealthMetrics represents health-related metrics for an xApp
type XAppHealthMetrics struct {
	CPUUsage         float64   `json:"cpuUsage"`
	MemoryUsage      int64     `json:"memoryUsage"`
	MemoryLimit      int64     `json:"memoryLimit"`
	NetworkIn        int64     `json:"networkIn"`
	NetworkOut       int64     `json:"networkOut"`
	RequestsPerSec   float64   `json:"requestsPerSec"`
	ResponseTime     float64   `json:"responseTime"`
	ErrorRate        float64   `json:"errorRate"`
	ActiveConnections int      `json:"activeConnections"`
	GoroutineCount   int       `json:"goroutineCount"`
	HeapSize         uint64    `json:"heapSize"`
	StackInUse       uint64    `json:"stackInUse"`
}

// XAppHealthConfig defines configuration for health monitoring
type XAppHealthConfig struct {
	CheckInterval    time.Duration `json:"checkInterval"`
	Timeout          time.Duration `json:"timeout"`
	RetryAttempts    int          `json:"retryAttempts"`
	UnhealthyThreshold int        `json:"unhealthyThreshold"`
	HealthyThreshold   int        `json:"healthyThreshold"`
	EnableMetrics      bool       `json:"enableMetrics"`
}

// NOTE: HealthCheckResult type moved to types.go to avoid redeclaration

// NewXAppHealthMonitor creates a new xApp health monitor
func NewXAppHealthMonitor(config *XAppHealthConfig) *XAppHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &XAppHealthMonitor{
		xapps:          make(map[string]*XAppHealth),
		healthCheckers: make(map[string]*HealthChecker),
		ctx:            ctx,
		cancel:         cancel,
		checkInterval:  config.CheckInterval,
		timeout:        config.Timeout,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Start begins the health monitoring process
func (h *XAppHealthMonitor) Start() error {
	go h.monitoringLoop()
	return nil
}

// Stop stops the health monitoring process
func (h *XAppHealthMonitor) Stop() {
	h.cancel()
}

// RegisterXApp registers a new xApp for health monitoring
func (h *XAppHealthMonitor) RegisterXApp(xapp *XAppHealth) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.xapps[xapp.XAppID] = xapp
	
	// Create a health checker for this xApp
	checker := &HealthChecker{
		ComponentID:   xapp.XAppID,
		Status:        HealthStatusUnknown,
		LastCheck:     time.Now(),
		CheckInterval: h.checkInterval,
		Metrics:       make(map[string]interface{}),
	}
	
	h.healthCheckers[xapp.XAppID] = checker
	
	log.Printf("Registered xApp %s for health monitoring", xapp.XAppID)
	return nil
}

// UnregisterXApp removes an xApp from health monitoring
func (h *XAppHealthMonitor) UnregisterXApp(xappID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	delete(h.xapps, xappID)
	delete(h.healthCheckers, xappID)
	
	log.Printf("Unregistered xApp %s from health monitoring", xappID)
	return nil
}

// GetXAppHealth returns the current health status of an xApp
func (h *XAppHealthMonitor) GetXAppHealth(xappID string) (*XAppHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	xapp, exists := h.xapps[xappID]
	if !exists {
		return nil, fmt.Errorf("xApp %s not found", xappID)
	}
	
	return xapp, nil
}

// GetAllXAppHealth returns the health status of all registered xApps
func (h *XAppHealthMonitor) GetAllXAppHealth() map[string]*XAppHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	result := make(map[string]*XAppHealth)
	for id, xapp := range h.xapps {
		result[id] = xapp
	}
	
	return result
}

// CheckXAppHealth performs a health check on a specific xApp
func (h *XAppHealthMonitor) CheckXAppHealth(xappID string) (*HealthCheckResult, error) {
	h.mu.RLock()
	xapp, exists := h.xapps[xappID]
	h.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("xApp %s not found", xappID)
	}
	
	startTime := time.Now()
	
	// Perform health check via HTTP endpoint
	healthEndpoint := fmt.Sprintf("http://%s:%d/health", xapp.Name, 8080) // Assuming default health endpoint
	
	ctx, cancel := context.WithTimeout(h.ctx, h.timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthEndpoint, nil)
	if err != nil {
		return &HealthCheckResult{
			ComponentName: xapp.XAppID,
			Status:        HealthStatusUnhealthy,
			Message:       fmt.Sprintf("Failed to create health check request: %v", err),
			Error:         err,
			CheckTime:     startTime,
			Duration:      time.Since(startTime),
		}, nil
	}
	
	resp, err := h.client.Do(req)
	if err != nil {
		return &HealthCheckResult{
			ComponentName: xapp.XAppID,
			Status:        HealthStatusUnhealthy,
			Message:       fmt.Sprintf("Health check request failed: %v", err),
			Error:         err,
			CheckTime:     startTime,
			Duration:      time.Since(startTime),
		}, nil
	}
	defer resp.Body.Close()
	
	duration := time.Since(startTime)
	
	if resp.StatusCode == http.StatusOK {
		// Parse response for metrics if available
		var healthData map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&healthData); err == nil {
			// Update xApp health data
			h.mu.Lock()
			xapp.HealthData = healthData
			xapp.Status = HealthStatusHealthy
			xapp.LastCheckTime = time.Now()
			h.mu.Unlock()
		}
		
		return &HealthCheckResult{
			ComponentName: xapp.XAppID,
			Status:        HealthStatusHealthy,
			Message:       "Health check successful",
			CheckTime:     startTime,
			Duration:      duration,
			Details:       healthData,
		}, nil
	}
	
	// Handle unhealthy status
	h.mu.Lock()
	xapp.Status = HealthStatusUnhealthy
	xapp.LastCheckTime = time.Now()
	h.mu.Unlock()
	
	return &HealthCheckResult{
		ComponentName: xapp.XAppID,
		Status:        HealthStatusUnhealthy,
		Message:       fmt.Sprintf("Health check failed with status: %d", resp.StatusCode),
		CheckTime:     startTime,
		Duration:      duration,
	}, nil
}

// monitoringLoop runs the continuous health monitoring
func (h *XAppHealthMonitor) monitoringLoop() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.performHealthChecks()
		}
	}
}

// performHealthChecks checks the health of all registered xApps
func (h *XAppHealthMonitor) performHealthChecks() {
	h.mu.RLock()
	xappIDs := make([]string, 0, len(h.xapps))
	for id := range h.xapps {
		xappIDs = append(xappIDs, id)
	}
	h.mu.RUnlock()
	
	// Perform health checks concurrently
	for _, xappID := range xappIDs {
		go func(id string) {
			if _, err := h.CheckXAppHealth(id); err != nil {
				log.Printf("Error checking health for xApp %s: %v", id, err)
			}
		}(xappID)
	}
}

// GetHealthSummary returns a summary of all xApp health statuses
func (h *XAppHealthMonitor) GetHealthSummary() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	healthyCount := 0
	unhealthyCount := 0
	unknownCount := 0
	degradedCount := 0
	
	for _, xapp := range h.xapps {
		switch xapp.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusDegraded:
			degradedCount++
		default:
			unknownCount++
		}
	}
	
	return map[string]interface{}{
		"total":     len(h.xapps),
		"healthy":   healthyCount,
		"unhealthy": unhealthyCount,
		"degraded":  degradedCount,
		"unknown":   unknownCount,
		"timestamp": time.Now(),
	}
}

// GetXAppMetrics returns metrics for a specific xApp
func (h *XAppHealthMonitor) GetXAppMetrics(xappID string) (*XAppHealthMetrics, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	xapp, exists := h.xapps[xappID]
	if !exists {
		return nil, fmt.Errorf("xApp %s not found", xappID)
	}
	
	return xapp.Metrics, nil
}

// UpdateXAppMetrics updates metrics for a specific xApp
func (h *XAppHealthMonitor) UpdateXAppMetrics(xappID string, metrics *XAppHealthMetrics) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	xapp, exists := h.xapps[xappID]
	if !exists {
		return fmt.Errorf("xApp %s not found", xappID)
	}
	
	xapp.Metrics = metrics
	xapp.LastCheckTime = time.Now()
	
	return nil
}