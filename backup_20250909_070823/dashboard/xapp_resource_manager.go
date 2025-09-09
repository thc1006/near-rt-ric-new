/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// XAppResourceManager manages resource allocation and isolation for xApps
type XAppResourceManager struct {
	allocations    map[string]*XAppResourceAllocation
	quotas         map[string]*XAppResourceQuota
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	totalResources *XAppResourceQuota
}

// XAppResourceAllocation represents allocated resources for an xApp instance
type XAppResourceAllocation struct {
	InstanceID      string                 `json:"instanceId"`
	XAppName        string                 `json:"xappName"`
	Version         string                 `json:"version"`
	AllocatedCPU    int64                  `json:"allocatedCpu"`    // CPU in millicores
	AllocatedMemory int64                  `json:"allocatedMemory"` // Memory in bytes
	AllocatedStorage int64                 `json:"allocatedStorage"` // Storage in bytes
	NetworkPorts    []int                  `json:"networkPorts"`
	Subscriptions   int                    `json:"subscriptions"`
	Connections     int                    `json:"connections"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	Status          XAppResourceStatus     `json:"status"`
	Metrics         *XAppResourceMetrics   `json:"metrics,omitempty"`
}

// XAppResourceQuota represents resource quotas and limits
type XAppResourceQuota struct {
	MaxCPU           int64 `json:"maxCpu"`           // CPU in millicores
	MaxMemory        int64 `json:"maxMemory"`        // Memory in bytes
	MaxStorage       int64 `json:"maxStorage"`       // Storage in bytes
	MaxInstances     int   `json:"maxInstances"`
	MaxSubscriptions int   `json:"maxSubscriptions"`
	MaxConnections   int   `json:"maxConnections"`
	MaxNetworkPorts  int   `json:"maxNetworkPorts"`
}

// XAppResourceStatus represents the status of resource allocation
type XAppResourceStatus string

const (
	XAppResourceStatusAllocated XAppResourceStatus = "ALLOCATED"
	XAppResourceStatusPending   XAppResourceStatus = "PENDING"
	XAppResourceStatusFailed    XAppResourceStatus = "FAILED"
	XAppResourceStatusReleased  XAppResourceStatus = "RELEASED"
)

// XAppResourceMetrics contains real-time resource usage metrics
type XAppResourceMetrics struct {
	CPUUsage        float64   `json:"cpuUsage"`        // CPU usage percentage
	MemoryUsage     int64     `json:"memoryUsage"`     // Memory usage in bytes
	StorageUsage    int64     `json:"storageUsage"`    // Storage usage in bytes
	NetworkBytesIn  int64     `json:"networkBytesIn"`  // Network bytes received
	NetworkBytesOut int64     `json:"networkBytesOut"` // Network bytes sent
	ActiveSubs      int       `json:"activeSubs"`      // Active subscriptions
	ActiveConns     int       `json:"activeConns"`     // Active connections
	LastUpdated     time.Time `json:"lastUpdated"`
}

// XAppResourceRequest represents a resource allocation request
type XAppResourceRequest struct {
	InstanceID       string            `json:"instanceId"`
	XAppName         string            `json:"xappName"`
	Version          string            `json:"version"`
	ResourceLimits   XAppResourceLimits `json:"resourceLimits"`
	RequiredPorts    []int             `json:"requiredPorts,omitempty"`
	PreferredPorts   []int             `json:"preferredPorts,omitempty"`
	Isolation        XAppIsolationLevel `json:"isolation"`
}

// XAppIsolationLevel defines the level of resource isolation
type XAppIsolationLevel string

const (
	XAppIsolationNone   XAppIsolationLevel = "NONE"
	XAppIsolationBasic  XAppIsolationLevel = "BASIC"
	XAppIsolationStrict XAppIsolationLevel = "STRICT"
)

// NewXAppResourceManager creates a new resource manager
func NewXAppResourceManager() *XAppResourceManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Default total resources (can be configured)
	totalResources := &XAppResourceQuota{
		MaxCPU:           8000,  // 8 CPU cores
		MaxMemory:        16 * 1024 * 1024 * 1024, // 16 GB
		MaxStorage:       100 * 1024 * 1024 * 1024, // 100 GB
		MaxInstances:     50,
		MaxSubscriptions: 1000,
		MaxConnections:   10000,
		MaxNetworkPorts:  1000,
	}
	
	return &XAppResourceManager{
		allocations:    make(map[string]*XAppResourceAllocation),
		quotas:         make(map[string]*XAppResourceQuota),
		ctx:            ctx,
		cancel:         cancel,
		totalResources: totalResources,
	}
}

// Start starts the resource manager
func (rm *XAppResourceManager) Start(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	log.Println("Starting xApp Resource Manager...")
	
	// Start resource monitoring
	go rm.monitorResources()
	
	// Start cleanup task
	go rm.cleanupTask()
	
	log.Println("xApp Resource Manager started successfully")
	return nil
}

// Stop stops the resource manager
func (rm *XAppResourceManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	log.Println("Stopping xApp Resource Manager...")
	
	// Cancel context to stop background tasks
	rm.cancel()
	
	log.Println("xApp Resource Manager stopped")
}

// AllocateResources allocates resources for an xApp instance
func (rm *XAppResourceManager) AllocateResources(request *XAppResourceRequest) (*XAppResourceAllocation, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Check if resources are available
	if err := rm.checkResourceAvailability(request); err != nil {
		return nil, fmt.Errorf("resource allocation failed: %w", err)
	}
	
	// Parse resource limits
	cpuMillicores, err := rm.parseCPULimit(request.ResourceLimits.CPU)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU limit: %w", err)
	}
	
	memoryBytes, err := rm.parseMemoryLimit(request.ResourceLimits.Memory)
	if err != nil {
		return nil, fmt.Errorf("invalid memory limit: %w", err)
	}
	
	storageBytes, err := rm.parseStorageLimit(request.ResourceLimits.Storage)
	if err != nil {
		return nil, fmt.Errorf("invalid storage limit: %w", err)
	}
	
	// Allocate network ports
	allocatedPorts, err := rm.allocateNetworkPorts(request.RequiredPorts, request.PreferredPorts)
	if err != nil {
		return nil, fmt.Errorf("port allocation failed: %w", err)
	}
	
	// Create allocation
	allocation := &XAppResourceAllocation{
		InstanceID:       request.InstanceID,
		XAppName:         request.XAppName,
		Version:          request.Version,
		AllocatedCPU:     cpuMillicores,
		AllocatedMemory:  memoryBytes,
		AllocatedStorage: storageBytes,
		NetworkPorts:     allocatedPorts,
		Subscriptions:    request.ResourceLimits.MaxSubscriptions,
		Connections:      request.ResourceLimits.MaxConnections,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Status:           XAppResourceStatusAllocated,
	}
	
	// Store allocation
	rm.allocations[request.InstanceID] = allocation
	
	log.Printf("Allocated resources for xApp instance %s: CPU=%dm, Memory=%dMB, Storage=%dMB, Ports=%v",
		request.InstanceID, cpuMillicores, memoryBytes/(1024*1024), storageBytes/(1024*1024), allocatedPorts)
	
	return allocation, nil
}

// ReleaseResources releases resources for an xApp instance
func (rm *XAppResourceManager) ReleaseResources(instanceID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	allocation, exists := rm.allocations[instanceID]
	if !exists {
		return fmt.Errorf("no resource allocation found for instance %s", instanceID)
	}
	
	// Mark as released
	allocation.Status = XAppResourceStatusReleased
	allocation.UpdatedAt = time.Now()
	
	// Remove from active allocations
	delete(rm.allocations, instanceID)
	
	log.Printf("Released resources for xApp instance %s", instanceID)
	return nil
}

// GetResourceAllocation returns the resource allocation for an instance
func (rm *XAppResourceManager) GetResourceAllocation(instanceID string) (*XAppResourceAllocation, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	allocation, exists := rm.allocations[instanceID]
	if !exists {
		return nil, fmt.Errorf("no resource allocation found for instance %s", instanceID)
	}
	
	// Return a copy to prevent external modification
	allocationCopy := *allocation
	return &allocationCopy, nil
}

// ListResourceAllocations returns all active resource allocations
func (rm *XAppResourceManager) ListResourceAllocations() ([]*XAppResourceAllocation, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	var allocations []*XAppResourceAllocation
	for _, allocation := range rm.allocations {
		// Return a copy to prevent external modification
		allocationCopy := *allocation
		allocations = append(allocations, &allocationCopy)
	}
	
	return allocations, nil
}

// UpdateResourceMetrics updates the resource usage metrics for an instance
func (rm *XAppResourceManager) UpdateResourceMetrics(instanceID string, metrics *XAppResourceMetrics) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	allocation, exists := rm.allocations[instanceID]
	if !exists {
		return fmt.Errorf("no resource allocation found for instance %s", instanceID)
	}
	
	allocation.Metrics = metrics
	allocation.UpdatedAt = time.Now()
	
	return nil
}

// GetResourceUsage returns current resource usage statistics
func (rm *XAppResourceManager) GetResourceUsage() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	var totalCPU, totalMemory, totalStorage int64
	var totalInstances, totalSubscriptions, totalConnections int
	var totalPorts []int
	
	for _, allocation := range rm.allocations {
		if allocation.Status == XAppResourceStatusAllocated {
			totalCPU += allocation.AllocatedCPU
			totalMemory += allocation.AllocatedMemory
			totalStorage += allocation.AllocatedStorage
			totalInstances++
			totalSubscriptions += allocation.Subscriptions
			totalConnections += allocation.Connections
			totalPorts = append(totalPorts, allocation.NetworkPorts...)
		}
	}
	
	return map[string]interface{}{
		"cpu": map[string]interface{}{
			"allocated": totalCPU,
			"total":     rm.totalResources.MaxCPU,
			"usage":     float64(totalCPU) / float64(rm.totalResources.MaxCPU) * 100,
		},
		"memory": map[string]interface{}{
			"allocated": totalMemory,
			"total":     rm.totalResources.MaxMemory,
			"usage":     float64(totalMemory) / float64(rm.totalResources.MaxMemory) * 100,
		},
		"storage": map[string]interface{}{
			"allocated": totalStorage,
			"total":     rm.totalResources.MaxStorage,
			"usage":     float64(totalStorage) / float64(rm.totalResources.MaxStorage) * 100,
		},
		"instances": map[string]interface{}{
			"active": totalInstances,
			"total":  rm.totalResources.MaxInstances,
		},
		"subscriptions": map[string]interface{}{
			"active": totalSubscriptions,
			"total":  rm.totalResources.MaxSubscriptions,
		},
		"connections": map[string]interface{}{
			"active": totalConnections,
			"total":  rm.totalResources.MaxConnections,
		},
		"ports": map[string]interface{}{
			"allocated": len(totalPorts),
			"total":     rm.totalResources.MaxNetworkPorts,
		},
		"timestamp": time.Now().UTC(),
	}
}

// SetResourceQuota sets resource quota for a specific xApp
func (rm *XAppResourceManager) SetResourceQuota(xappName string, quota *XAppResourceQuota) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.quotas[xappName] = quota
	log.Printf("Set resource quota for xApp %s", xappName)
	return nil
}

// GetResourceQuota returns the resource quota for a specific xApp
func (rm *XAppResourceManager) GetResourceQuota(xappName string) (*XAppResourceQuota, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	quota, exists := rm.quotas[xappName]
	if !exists {
		// Return default quota
		return &XAppResourceQuota{
			MaxCPU:           1000, // 1 CPU core
			MaxMemory:        1024 * 1024 * 1024, // 1 GB
			MaxStorage:       10 * 1024 * 1024 * 1024, // 10 GB
			MaxInstances:     5,
			MaxSubscriptions: 100,
			MaxConnections:   1000,
			MaxNetworkPorts:  10,
		}, nil
	}
	
	// Return a copy to prevent external modification
	quotaCopy := *quota
	return &quotaCopy, nil
}

// checkResourceAvailability checks if requested resources are available
func (rm *XAppResourceManager) checkResourceAvailability(request *XAppResourceRequest) error {
	// Get current usage
	usage := rm.getResourceUsageInternal()
	
	// Parse requested resources
	cpuMillicores, err := rm.parseCPULimit(request.ResourceLimits.CPU)
	if err != nil {
		return err
	}
	
	memoryBytes, err := rm.parseMemoryLimit(request.ResourceLimits.Memory)
	if err != nil {
		return err
	}
	
	storageBytes, err := rm.parseStorageLimit(request.ResourceLimits.Storage)
	if err != nil {
		return err
	}
	
	// Check CPU availability
	if usage.totalCPU+cpuMillicores > rm.totalResources.MaxCPU {
		return fmt.Errorf("insufficient CPU resources: requested %dm, available %dm",
			cpuMillicores, rm.totalResources.MaxCPU-usage.totalCPU)
	}
	
	// Check memory availability
	if usage.totalMemory+memoryBytes > rm.totalResources.MaxMemory {
		return fmt.Errorf("insufficient memory resources: requested %dMB, available %dMB",
			memoryBytes/(1024*1024), (rm.totalResources.MaxMemory-usage.totalMemory)/(1024*1024))
	}
	
	// Check storage availability
	if usage.totalStorage+storageBytes > rm.totalResources.MaxStorage {
		return fmt.Errorf("insufficient storage resources: requested %dMB, available %dMB",
			storageBytes/(1024*1024), (rm.totalResources.MaxStorage-usage.totalStorage)/(1024*1024))
	}
	
	// Check instance limit
	if usage.totalInstances >= rm.totalResources.MaxInstances {
		return fmt.Errorf("maximum number of instances reached: %d", rm.totalResources.MaxInstances)
	}
	
	return nil
}

// getResourceUsageInternal returns current resource usage (internal method)
func (rm *XAppResourceManager) getResourceUsageInternal() struct {
	totalCPU, totalMemory, totalStorage int64
	totalInstances                      int
} {
	var usage struct {
		totalCPU, totalMemory, totalStorage int64
		totalInstances                      int
	}
	
	for _, allocation := range rm.allocations {
		if allocation.Status == XAppResourceStatusAllocated {
			usage.totalCPU += allocation.AllocatedCPU
			usage.totalMemory += allocation.AllocatedMemory
			usage.totalStorage += allocation.AllocatedStorage
			usage.totalInstances++
		}
	}
	
	return usage
}

// parseCPULimit parses CPU limit string (e.g., "500m", "1", "1.5")
func (rm *XAppResourceManager) parseCPULimit(cpuLimit string) (int64, error) {
	if strings.HasSuffix(cpuLimit, "m") {
		// Millicores
		value, err := strconv.ParseInt(strings.TrimSuffix(cpuLimit, "m"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid CPU limit format: %s", cpuLimit)
		}
		return value, nil
	} else {
		// Cores
		value, err := strconv.ParseFloat(cpuLimit, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid CPU limit format: %s", cpuLimit)
		}
		return int64(value * 1000), nil
	}
}

// parseMemoryLimit parses memory limit string (e.g., "512Mi", "1Gi", "1024")
func (rm *XAppResourceManager) parseMemoryLimit(memoryLimit string) (int64, error) {
	if strings.HasSuffix(memoryLimit, "Mi") {
		value, err := strconv.ParseInt(strings.TrimSuffix(memoryLimit, "Mi"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid memory limit format: %s", memoryLimit)
		}
		return value * 1024 * 1024, nil
	} else if strings.HasSuffix(memoryLimit, "Gi") {
		value, err := strconv.ParseInt(strings.TrimSuffix(memoryLimit, "Gi"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid memory limit format: %s", memoryLimit)
		}
		return value * 1024 * 1024 * 1024, nil
	} else {
		// Assume bytes
		value, err := strconv.ParseInt(memoryLimit, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid memory limit format: %s", memoryLimit)
		}
		return value, nil
	}
}

// parseStorageLimit parses storage limit string (e.g., "10Gi", "1024Mi")
func (rm *XAppResourceManager) parseStorageLimit(storageLimit string) (int64, error) {
	if storageLimit == "" {
		return 1024 * 1024 * 1024, nil // Default 1GB
	}
	
	if strings.HasSuffix(storageLimit, "Mi") {
		value, err := strconv.ParseInt(strings.TrimSuffix(storageLimit, "Mi"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid storage limit format: %s", storageLimit)
		}
		return value * 1024 * 1024, nil
	} else if strings.HasSuffix(storageLimit, "Gi") {
		value, err := strconv.ParseInt(strings.TrimSuffix(storageLimit, "Gi"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid storage limit format: %s", storageLimit)
		}
		return value * 1024 * 1024 * 1024, nil
	} else {
		// Assume bytes
		value, err := strconv.ParseInt(storageLimit, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid storage limit format: %s", storageLimit)
		}
		return value, nil
	}
}

// allocateNetworkPorts allocates network ports for an xApp
func (rm *XAppResourceManager) allocateNetworkPorts(requiredPorts, preferredPorts []int) ([]int, error) {
	allocatedPorts := make([]int, 0)
	usedPorts := make(map[int]bool)
	
	// Collect currently used ports
	for _, allocation := range rm.allocations {
		for _, port := range allocation.NetworkPorts {
			usedPorts[port] = true
		}
	}
	
	// Allocate required ports
	for _, port := range requiredPorts {
		if usedPorts[port] {
			return nil, fmt.Errorf("required port %d is already in use", port)
		}
		allocatedPorts = append(allocatedPorts, port)
		usedPorts[port] = true
	}
	
	// Allocate preferred ports if available
	for _, port := range preferredPorts {
		if !usedPorts[port] {
			allocatedPorts = append(allocatedPorts, port)
			usedPorts[port] = true
		}
	}
	
	return allocatedPorts, nil
}

// monitorResources monitors resource usage
func (rm *XAppResourceManager) monitorResources() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.performResourceMonitoring()
		}
	}
}

// performResourceMonitoring performs resource monitoring
func (rm *XAppResourceManager) performResourceMonitoring() {
	rm.mu.RLock()
	activeAllocations := len(rm.allocations)
	rm.mu.RUnlock()
	
	log.Printf("Resource monitoring: %d active allocations", activeAllocations)
	
	// TODO: Collect actual resource metrics from Kubernetes/containers
	// For now, just log the monitoring activity
}

// cleanupTask performs periodic cleanup
func (rm *XAppResourceManager) cleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.performCleanup()
		}
	}
}

// performCleanup performs cleanup of stale allocations
func (rm *XAppResourceManager) performCleanup() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Clean up allocations that have been released for more than 1 hour
	cutoff := time.Now().Add(-1 * time.Hour)
	
	for instanceID, allocation := range rm.allocations {
		if allocation.Status == XAppResourceStatusReleased && allocation.UpdatedAt.Before(cutoff) {
			delete(rm.allocations, instanceID)
			log.Printf("Cleaned up stale resource allocation for instance %s", instanceID)
		}
	}
}