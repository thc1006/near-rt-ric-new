/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

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

// XAppHealthMonitor monitors the health of xApp instances
type XAppHealthMonitor struct {
	mu              sync.RWMutex
	healthChecks    map[string]*XAppHealthStatus
	isRunning       bool
	ctx             context.Context
	cancel          context.CancelFunc
	httpClient      *http.Client
	checkInterval   time.Duration
	alertThreshold  int
	alertCallbacks  []HealthAlertCallback
}

// XAppHealthStatus represents the health status of an xApp instance
type XAppHealthStatus struct {
	InstanceID       string                 `json:"instanceId"`
	XAppName         string                 `json:"xappName"`
	XAppVersion      string                 `json:"xappVersion"`
	Status           HealthStatus           `json:"status"`
	LastCheckTime    time.Time              `json:"lastCheckTime"`
	LastHealthyTime  time.Time              `json:"lastHealthyTime"`
	ConsecutiveFails int                    `json:"consecutiveFails"`
	TotalChecks      uint64                 `json:"totalChecks"`
	SuccessfulChecks uint64                 `json:"successfulChecks"`
	FailedChecks     uint64                 `json:"failedChecks"`
	ResponseTime     time.Duration          `json:"responseTime"`
	ErrorMessage     string                 `json:"errorMessage,omitempty"`
	HealthData       map[string]interface{} `json:"healthData,omitempty"`
	Metrics          *XAppHealthMetrics     `json:"metrics,omitempty"`
}

// HealthStatus represents the health status of an xApp
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "HEALTHY"
	HealthStatusUnhealthy HealthStatus = "UNHEALTHY"
	HealthStatusUnknown   HealthStatus = "UNKNOWN"
	HealthStatusStarting  HealthStatus = "STARTING"
	HealthStatusStopping  HealthStatus = "STOPPING"
)

// XAppHealthMetrics represents health-related metrics for an xApp
type XAppHealthMetrics struct {
	CPUUsage         float64   `json:"cpuUsage"`
	MemoryUsage      int64     `json:"memoryUsage"`
	MemoryLimit      int64     `json:"memoryLimit"`
	NetworkRxBytes   int64     `json:"networkRxBytes"`
	NetworkTxBytes   int64     `json:"networkTxBytes"`
	ActiveConnections int       `json:"activeConnections"`
	ProcessedMessages int64     `json:"processedMessages"`
	ErrorCount       int64     `json:"errorCount"`
	Uptime           time.Duration `json:"uptime"`
	LastUpdated      time.Time `json:"lastUpdated"`
}

// HealthAlertCallback defines the callback function for health alerts
type HealthAlertCallback func(instanceID string, status *XAppHealthStatus, alertType HealthAlertType)

// HealthAlertType represents the type of health alert
type HealthAlertType string

const (
	HealthAlertUnhealthy HealthAlertType = "UNHEALTHY"
	HealthAlertRecovered HealthAlertType = "RECOVERED"
	HealthAlertTimeout   HealthAlertType = "TIMEOUT"
	HealthAlertError     HealthAlertType = "ERROR"
)

// NewXAppHealthMonitor creates a new xApp health monitor
func NewXAppHealthMonitor() *XAppHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &XAppHealthMonitor{
		healthChecks:   make(map[string]*XAppHealthStatus),
		ctx:            ctx,
		cancel:         cancel,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		checkInterval:  30 * time.Second,
		alertThreshold: 3,
		alertCallbacks: make([]HealthAlertCallback, 0),
	}
}

// Start starts the health monitor
func (hm *XAppHealthMonitor) Start() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	if hm.isRunning {
		return fmt.Errorf("health monitor is already running")
	}
	
	// Start health check routine
	go hm.healthCheckRoutine()
	
	// Start metrics collection routine
	go hm.metricsCollectionRoutine()
	
	hm.isRunning = true
	log.Println("xApp health monitor started")
	return nil
}

// Stop stops the health monitor
func (hm *XAppHealthMonitor) Stop() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	if !hm.isRunning {
		return nil
	}
	
	hm.cancel()
	hm.isRunning = false
	log.Println("xApp health monitor stopped")
	return nil
}

// RegisterXApp registers an xApp instance for health monitoring
func (hm *XAppHealthMonitor) RegisterXApp(instance *XAppInstance) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	healthStatus := &XAppHealthStatus{
		InstanceID:       instance.ID,
		XAppName:         instance.Descriptor.Name,
		XAppVersion:      instance.Descriptor.Version,
		Status:           HealthStatusStarting,
		LastCheckTime:    time.Now(),
		LastHealthyTime:  time.Now(),
		ConsecutiveFails: 0,
		TotalChecks:      0,
		SuccessfulChecks: 0,
		FailedChecks:     0,
		HealthData:       make(map[string]interface{}),
		Metrics:          &XAppHealthMetrics{},
	}
	
	hm.healthChecks[instance.ID] = healthStatus
	
	log.Printf("Registered xApp %s (instance: %s) for health monitoring", 
		instance.Descriptor.Name, instance.ID)
	return nil
}

// UnregisterXApp unregisters an xApp instance from health monitoring
func (hm *XAppHealthMonitor) UnregisterXApp(instanceID string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	delete(hm.healthChecks, instanceID)
	
	log.Printf("Unregistered xApp instance %s from health monitoring", instanceID)
	return nil
}

// GetHealthStatus returns the health status of an xApp instance
func (hm *XAppHealthMonitor) GetHealthStatus(instanceID string) (*XAppHealthStatus, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	status, exists := hm.healthChecks[instanceID]
	if !exists {
		return nil, fmt.Errorf("xApp instance %s not found", instanceID)
	}
	
	return status, nil
}

// GetAllHealthStatuses returns health statuses for all monitored xApp instances
func (hm *XAppHealthMonitor) GetAllHealthStatuses() map[string]*XAppHealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	statuses := make(map[string]*XAppHealthStatus)
	for instanceID, status := range hm.healthChecks {
		statuses[instanceID] = &XAppHealthStatus{
			InstanceID:       status.InstanceID,
			XAppName:         status.XAppName,
			XAppVersion:      status.XAppVersion,
			Status:           status.Status,
			LastCheckTime:    status.LastCheckTime,
			LastHealthyTime:  status.LastHealthyTime,
			ConsecutiveFails: status.ConsecutiveFails,
			TotalChecks:      status.TotalChecks,
			SuccessfulChecks: status.SuccessfulChecks,
			FailedChecks:     status.FailedChecks,
			ResponseTime:     status.ResponseTime,
			ErrorMessage:     status.ErrorMessage,
			HealthData:       make(map[string]interface{}),
			Metrics:          status.Metrics,
		}
		// Copy health data
		for k, v := range status.HealthData {
			statuses[instanceID].HealthData[k] = v
		}
	}
	
	return statuses
}

// AddAlertCallback adds a callback function for health alerts
func (hm *XAppHealthMonitor) AddAlertCallback(callback HealthAlertCallback) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	hm.alertCallbacks = append(hm.alertCallbacks, callback)
}

// SetCheckInterval sets the health check interval
func (hm *XAppHealthMonitor) SetCheckInterval(interval time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	hm.checkInterval = interval
}

// SetAlertThreshold sets the threshold for consecutive failures before alerting
func (hm *XAppHealthMonitor) SetAlertThreshold(threshold int) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	hm.alertThreshold = threshold
}

// Private methods

func (hm *XAppHealthMonitor) healthCheckRoutine() {
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			hm.performHealthChecks()
		}
	}
}

func (hm *XAppHealthMonitor) metricsCollectionRoutine() {
	ticker := time.NewTicker(15 * time.Second) // Collect metrics more frequently
	defer ticker.Stop()
	
	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			hm.collectMetrics()
		}
	}
}

func (hm *XAppHealthMonitor) performHealthChecks() {
	hm.mu.RLock()
	instances := make(map[string]*XAppHealthStatus)
	for id, status := range hm.healthChecks {
		instances[id] = status
	}
	hm.mu.RUnlock()
	
	for instanceID, status := range instances {
		go hm.checkInstanceHealth(instanceID, status)
	}
}

func (hm *XAppHealthMonitor) checkInstanceHealth(instanceID string, status *XAppHealthStatus) {
	startTime := time.Now()
	
	// Perform HTTP health check
	healthy, healthData, err := hm.performHTTPHealthCheck(instanceID)
	
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	// Update status
	status.LastCheckTime = time.Now()
	status.ResponseTime = time.Since(startTime)
	status.TotalChecks++
	
	if healthy {
		previousStatus := status.Status
		status.Status = HealthStatusHealthy
		status.LastHealthyTime = time.Now()
		status.ConsecutiveFails = 0
		status.SuccessfulChecks++
		status.ErrorMessage = ""
		status.HealthData = healthData
		
		// Check if this is a recovery
		if previousStatus == HealthStatusUnhealthy {
			hm.triggerAlert(instanceID, status, HealthAlertRecovered)
		}
	} else {
		status.Status = HealthStatusUnhealthy
		status.ConsecutiveFails++
		status.FailedChecks++
		if err != nil {
			status.ErrorMessage = err.Error()
		}
		
		// Trigger alert if threshold reached
		if status.ConsecutiveFails >= hm.alertThreshold {
			hm.triggerAlert(instanceID, status, HealthAlertUnhealthy)
		}
	}
}

func (hm *XAppHealthMonitor) performHTTPHealthCheck(instanceID string) (bool, map[string]interface{}, error) {
	// This is a simplified health check - in a real implementation,
	// you would get the actual endpoint from the xApp instance configuration
	healthEndpoint := fmt.Sprintf("http://xapp-%s:8080/health", instanceID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthEndpoint, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := hm.httpClient.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	
	// Parse health data
	var healthData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthData); err != nil {
		// If we can't parse the response, still consider it healthy if status is OK
		return true, map[string]interface{}{"status": "ok"}, nil
	}
	
	return true, healthData, nil
}

func (hm *XAppHealthMonitor) collectMetrics() {
	hm.mu.RLock()
	instances := make(map[string]*XAppHealthStatus)
	for id, status := range hm.healthChecks {
		instances[id] = status
	}
	hm.mu.RUnlock()
	
	for instanceID, status := range instances {
		go hm.collectInstanceMetrics(instanceID, status)
	}
}

func (hm *XAppHealthMonitor) collectInstanceMetrics(instanceID string, status *XAppHealthStatus) {
	// This is a simplified metrics collection - in a real implementation,
	// you would collect actual metrics from Prometheus or the xApp's metrics endpoint
	metricsEndpoint := fmt.Sprintf("http://xapp-%s:8080/metrics", instanceID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", metricsEndpoint, nil)
	if err != nil {
		log.Printf("Failed to create metrics request for %s: %v", instanceID, err)
		return
	}
	
	resp, err := hm.httpClient.Do(req)
	if err != nil {
		log.Printf("Metrics request failed for %s: %v", instanceID, err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("Metrics request returned status %d for %s", resp.StatusCode, instanceID)
		return
	}
	
	// Parse metrics data
	var metricsData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metricsData); err != nil {
		log.Printf("Failed to parse metrics for %s: %v", instanceID, err)
		return
	}
	
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	// Update metrics
	if status.Metrics == nil {
		status.Metrics = &XAppHealthMetrics{}
	}
	
	// Extract metrics from the response
	if cpuUsage, ok := metricsData["cpu_usage"].(float64); ok {
		status.Metrics.CPUUsage = cpuUsage
	}
	if memoryUsage, ok := metricsData["memory_usage"].(float64); ok {
		status.Metrics.MemoryUsage = int64(memoryUsage)
	}
	if memoryLimit, ok := metricsData["memory_limit"].(float64); ok {
		status.Metrics.MemoryLimit = int64(memoryLimit)
	}
	if networkRx, ok := metricsData["network_rx_bytes"].(float64); ok {
		status.Metrics.NetworkRxBytes = int64(networkRx)
	}
	if networkTx, ok := metricsData["network_tx_bytes"].(float64); ok {
		status.Metrics.NetworkTxBytes = int64(networkTx)
	}
	if activeConns, ok := metricsData["active_connections"].(float64); ok {
		status.Metrics.ActiveConnections = int(activeConns)
	}
	if processedMsgs, ok := metricsData["processed_messages"].(float64); ok {
		status.Metrics.ProcessedMessages = int64(processedMsgs)
	}
	if errorCount, ok := metricsData["error_count"].(float64); ok {
		status.Metrics.ErrorCount = int64(errorCount)
	}
	if uptime, ok := metricsData["uptime_seconds"].(float64); ok {
		status.Metrics.Uptime = time.Duration(uptime) * time.Second
	}
	
	status.Metrics.LastUpdated = time.Now()
}

func (hm *XAppHealthMonitor) triggerAlert(instanceID string, status *XAppHealthStatus, alertType HealthAlertType) {
	log.Printf("Health alert for xApp %s (instance: %s): %s", 
		status.XAppName, instanceID, alertType)
	
	// Call all registered alert callbacks
	for _, callback := range hm.alertCallbacks {
		go callback(instanceID, status, alertType)
	}
}

// GetHealthSummary returns a summary of health statuses
func (hm *XAppHealthMonitor) GetHealthSummary() map[string]interface{} {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	summary := map[string]interface{}{
		"totalInstances": len(hm.healthChecks),
		"byStatus":       make(map[string]int),
		"byXApp":         make(map[string]int),
	}
	
	statusCounts := make(map[string]int)
	xappCounts := make(map[string]int)
	
	var totalChecks uint64
	var totalSuccessful uint64
	var totalFailed uint64
	
	for _, status := range hm.healthChecks {
		statusCounts[string(status.Status)]++
		xappCounts[status.XAppName]++
		totalChecks += status.TotalChecks
		totalSuccessful += status.SuccessfulChecks
		totalFailed += status.FailedChecks
	}
	
	summary["byStatus"] = statusCounts
	summary["byXApp"] = xappCounts
	summary["totalChecks"] = totalChecks
	summary["totalSuccessful"] = totalSuccessful
	summary["totalFailed"] = totalFailed
	
	if totalChecks > 0 {
		summary["successRate"] = float64(totalSuccessful) / float64(totalChecks) * 100
	} else {
		summary["successRate"] = 0.0
	}
	
	return summary
}