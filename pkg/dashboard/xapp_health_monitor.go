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

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	XAppID     string                 `json:"xappId"`
	Status     HealthStatus          `json:"status"`
	Message    string                `json:"message,omitempty"`
	Metrics    *XAppHealthMetrics    `json:"metrics,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	CheckTime  time.Time             `json:"checkTime"`
	Duration   time.Duration         `json:"duration"`
}

// NewXAppHealthMonitor creates a new xApp health monitor
func NewXAppHealthMonitor(config *XAppHealthConfig) *XAppHealthMonitor {
	if config == nil {
		config = &XAppHealthConfig{
			CheckInterval:      30 * time.Second,
			Timeout:           10 * time.Second,
			RetryAttempts:     3,
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
			EnableMetrics:      true,
		}
	}

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

// RegisterXApp registers an xApp for health monitoring
func (monitor *XAppHealthMonitor) RegisterXApp(xappID, name, namespace, healthEndpoint string) error {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	if _, exists := monitor.xapps[xappID]; exists {
		return fmt.Errorf("xApp %s already registered", xappID)
	}

	xappHealth := &XAppHealth{
		XAppID:        xappID,
		Name:          name,
		Namespace:     namespace,
		Status:        HealthStatusUnknown,
		LastCheckTime: time.Now(),
		HealthData:    make(map[string]interface{}),
		Metrics:       &XAppHealthMetrics{},
	}

	monitor.xapps[xappID] = xappHealth

	// Create health checker
	checker := &HealthChecker{
		ComponentID:   xappID,
		Status:        HealthStatusUnknown,
		LastCheck:     time.Now(),
		CheckInterval: monitor.checkInterval,
		Metrics:       make(map[string]interface{}),
	}

	monitor.healthCheckers[xappID] = checker

	log.Printf("Registered xApp %s (%s) for health monitoring", xappID, name)
	return nil
}

// UnregisterXApp removes an xApp from health monitoring
func (monitor *XAppHealthMonitor) UnregisterXApp(xappID string) error {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	if _, exists := monitor.xapps[xappID]; !exists {
		return fmt.Errorf("xApp %s not found", xappID)
	}

	delete(monitor.xapps, xappID)
	delete(monitor.healthCheckers, xappID)

	log.Printf("Unregistered xApp %s from health monitoring", xappID)
	return nil
}

// CheckHealth performs a health check for a specific xApp
func (monitor *XAppHealthMonitor) CheckHealth(xappID string) (*HealthCheckResult, error) {
	monitor.mu.RLock()
	xapp, exists := monitor.xapps[xappID]
	monitor.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("xApp %s not found", xappID)
	}

	start := time.Now()
	
	// Perform health check (simplified implementation)
	result := &HealthCheckResult{
		XAppID:    xappID,
		Status:    HealthStatusHealthy, // Default to healthy for now
		Message:   "Health check completed successfully",
		CheckTime: start,
		Duration:  time.Since(start),
		Details:   make(map[string]interface{}),
	}

	// Update xApp health status
	monitor.mu.Lock()
	xapp.Status = result.Status
	xapp.LastCheckTime = result.CheckTime
	monitor.mu.Unlock()

	return result, nil
}

// CheckAllHealth performs health checks for all registered xApps
func (monitor *XAppHealthMonitor) CheckAllHealth() map[string]*HealthCheckResult {
	monitor.mu.RLock()
	xappIds := make([]string, 0, len(monitor.xapps))
	for id := range monitor.xapps {
		xappIds = append(xappIds, id)
	}
	monitor.mu.RUnlock()

	results := make(map[string]*HealthCheckResult)
	
	for _, xappID := range xappIds {
		if result, err := monitor.CheckHealth(xappID); err == nil {
			results[xappID] = result
		} else {
			log.Printf("Health check failed for xApp %s: %v", xappID, err)
		}
	}

	return results
}

// GetXAppHealth returns the current health status of an xApp
func (monitor *XAppHealthMonitor) GetXAppHealth(xappID string) (*XAppHealth, error) {
	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	xapp, exists := monitor.xapps[xappID]
	if !exists {
		return nil, fmt.Errorf("xApp %s not found", xappID)
	}

	// Return a copy to avoid race conditions
	xappCopy := *xapp
	if xapp.Metrics != nil {
		metricsCopy := *xapp.Metrics
		xappCopy.Metrics = &metricsCopy
	}

	return &xappCopy, nil
}

// GetAllXAppsHealth returns the health status of all registered xApps
func (monitor *XAppHealthMonitor) GetAllXAppsHealth() map[string]*XAppHealth {
	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	result := make(map[string]*XAppHealth)
	for id, xapp := range monitor.xapps {
		xappCopy := *xapp
		if xapp.Metrics != nil {
			metricsCopy := *xapp.Metrics
			xappCopy.Metrics = &metricsCopy
		}
		result[id] = &xappCopy
	}

	return result
}

// GetHealthySummary returns a summary of healthy vs unhealthy xApps
func (monitor *XAppHealthMonitor) GetHealthySummary() map[string]int {
	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	summary := map[string]int{
		"total":     0,
		"healthy":   0,
		"degraded":  0,
		"unhealthy": 0,
		"unknown":   0,
	}

	for _, xapp := range monitor.xapps {
		summary["total"]++
		switch xapp.Status {
		case HealthStatusHealthy:
			summary["healthy"]++
		case HealthStatusDegraded:
			summary["degraded"]++
		case HealthStatusUnhealthy:
			summary["unhealthy"]++
		default:
			summary["unknown"]++
		}
	}

	return summary
}

// Start begins the health monitoring process
func (monitor *XAppHealthMonitor) Start() error {
	log.Println("Starting xApp health monitor...")

	// Start periodic health checks
	go monitor.periodicHealthCheck()

	// Start metrics collection (if enabled)
	go monitor.metricsCollection()

	log.Println("xApp health monitor started successfully")
	return nil
}

// Stop stops the health monitoring process
func (monitor *XAppHealthMonitor) Stop() error {
	log.Println("Stopping xApp health monitor...")

	monitor.cancel()

	log.Println("xApp health monitor stopped")
	return nil
}

// periodicHealthCheck runs periodic health checks for all xApps
func (monitor *XAppHealthMonitor) periodicHealthCheck() {
	ticker := time.NewTicker(monitor.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-monitor.ctx.Done():
			return
		case <-ticker.C:
			monitor.CheckAllHealth()
		}
	}
}

// metricsCollection collects detailed metrics from xApps
func (monitor *XAppHealthMonitor) metricsCollection() {
	ticker := time.NewTicker(monitor.checkInterval / 2) // Collect metrics more frequently
	defer ticker.Stop()

	for {
		select {
		case <-monitor.ctx.Done():
			return
		case <-ticker.C:
			monitor.collectMetrics()
		}
	}
}

// collectMetrics collects metrics from all registered xApps
func (monitor *XAppHealthMonitor) collectMetrics() {
	monitor.mu.RLock()
	xappIds := make([]string, 0, len(monitor.xapps))
	for id := range monitor.xapps {
		xappIds = append(xappIds, id)
	}
	monitor.mu.RUnlock()

	for _, xappID := range xappIds {
		if metrics, err := monitor.getXAppMetrics(xappID); err == nil {
			monitor.mu.Lock()
			if xapp, exists := monitor.xapps[xappID]; exists {
				xapp.Metrics = metrics
			}
			monitor.mu.Unlock()
		}
	}
}

// getXAppMetrics retrieves metrics for a specific xApp
func (monitor *XAppHealthMonitor) getXAppMetrics(xappID string) (*XAppHealthMetrics, error) {
	// In a real implementation, this would query the xApp's metrics endpoint
	// For now, return mock metrics
	return &XAppHealthMetrics{
		CPUUsage:          25.5,
		MemoryUsage:       134217728, // 128MB
		MemoryLimit:       536870912, // 512MB
		NetworkIn:         1024000,
		NetworkOut:        2048000,
		RequestsPerSec:    50.0,
		ResponseTime:      15.5,
		ErrorRate:         0.1,
		ActiveConnections: 10,
		GoroutineCount:    25,
		HeapSize:          67108864, // 64MB
		StackInUse:        1048576,  // 1MB
	}, nil
}

// SetXAppStatus manually sets the health status of an xApp
func (monitor *XAppHealthMonitor) SetXAppStatus(xappID string, status HealthStatus, message string) error {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	xapp, exists := monitor.xapps[xappID]
	if !exists {
		return fmt.Errorf("xApp %s not found", xappID)
	}

	oldStatus := xapp.Status
	xapp.Status = status
	xapp.LastCheckTime = time.Now()

	if xapp.HealthData == nil {
		xapp.HealthData = make(map[string]interface{})
	}
	xapp.HealthData["manualStatusChange"] = message

	log.Printf("xApp %s status changed from %s to %s: %s", xappID, oldStatus, status, message)
	return nil
}

// GetHealthHistory returns the health check history for an xApp
func (monitor *XAppHealthMonitor) GetHealthHistory(xappID string) ([]HealthCheckResult, error) {
	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	if _, exists := monitor.xapps[xappID]; !exists {
		return nil, fmt.Errorf("xApp %s not found", xappID)
	}

	// In a real implementation, this would return stored history
	// For now, return empty history
	return []HealthCheckResult{}, nil
}

// ExportHealthData exports all health data as JSON
func (monitor *XAppHealthMonitor) ExportHealthData() ([]byte, error) {
	monitor.mu.RLock()
	defer monitor.mu.RUnlock()

	data := struct {
		XApps   map[string]*XAppHealth `json:"xapps"`
		Summary map[string]int         `json:"summary"`
		Timestamp time.Time            `json:"timestamp"`
	}{
		XApps:     monitor.xapps,
		Summary:   monitor.GetHealthySummary(),
		Timestamp: time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}

// HandleHealthWebhook handles incoming health status webhooks from xApps
func (monitor *XAppHealthMonitor) HandleHealthWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook struct {
		XAppID   string                 `json:"xappId"`
		Status   HealthStatus          `json:"status"`
		Message  string                `json:"message,omitempty"`
		Metrics  *XAppHealthMetrics    `json:"metrics,omitempty"`
		Details  map[string]interface{} `json:"details,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	monitor.mu.Lock()
	xapp, exists := monitor.xapps[webhook.XAppID]
	if !exists {
		monitor.mu.Unlock()
		http.Error(w, "xApp not found", http.StatusNotFound)
		return
	}

	xapp.Status = webhook.Status
	xapp.LastCheckTime = time.Now()
	if webhook.Metrics != nil {
		xapp.Metrics = webhook.Metrics
	}
	if webhook.Details != nil {
		xapp.HealthData = webhook.Details
	}
	monitor.mu.Unlock()

	log.Printf("Received health webhook for xApp %s: %s", webhook.XAppID, webhook.Status)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}