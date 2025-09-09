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
	"sync"
	"time"
)

// XAppFramework provides the core framework for xApp development and runtime
type XAppFramework struct {
	registry         *XAppRegistry
	resourceManager  *XAppResourceManager
	communicationAPI *XAppCommunicationAPI
	configManager    *XAppConfigManager
	lifecycleManager *XAppLifecycleManager
	clientManager    *ClientManager
	messageBus       *RMRMessageBus
	serviceRegistry  *ServiceModelRegistry
	subscriptionMgr  *XAppSubscriptionManager
	healthMonitor    *XAppHealthMonitor
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

// XAppDescriptor describes an xApp and its capabilities
type XAppDescriptor struct {
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description"`
	ServiceModels   []string               `json:"serviceModels"`
	Capabilities    []string               `json:"capabilities"`
	ResourceLimits  XAppResourceLimits     `json:"resourceLimits"`
	Configuration   map[string]interface{} `json:"configuration"`
	Endpoints       XAppEndpoints          `json:"endpoints"`
	HealthCheck     XAppHealthCheck        `json:"healthCheck"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	Status          XAppStatus             `json:"status"`
}

// XAppResourceLimits defines resource constraints for an xApp
type XAppResourceLimits struct {
	CPU              string `json:"cpu"`
	Memory           string `json:"memory"`
	Storage          string `json:"storage"`
	MaxSubscriptions int    `json:"maxSubscriptions"`
	MaxConnections   int    `json:"maxConnections"`
}

// XAppEndpoints defines the communication endpoints for an xApp
type XAppEndpoints struct {
	HTTP    string `json:"http"`
	RMRData int    `json:"rmrData"`
	RMRRoute int   `json:"rmrRoute"`
	Metrics string `json:"metrics"`
}

// XAppHealthCheck defines health check configuration
type XAppHealthCheck struct {
	Enabled             bool          `json:"enabled"`
	Path                string        `json:"path"`
	InitialDelaySeconds int           `json:"initialDelaySeconds"`
	PeriodSeconds       int           `json:"periodSeconds"`
	TimeoutSeconds      int           `json:"timeoutSeconds"`
	FailureThreshold    int           `json:"failureThreshold"`
}

// XAppStatus represents the current status of an xApp
type XAppStatus string

const (
	XAppStatusPending   XAppStatus = "PENDING"
	XAppStatusRunning   XAppStatus = "RUNNING"
	XAppStatusStopped   XAppStatus = "STOPPED"
	XAppStatusFailed    XAppStatus = "FAILED"
	XAppStatusUpdating  XAppStatus = "UPDATING"
)


// XAppMetrics contains runtime metrics for an xApp
type XAppMetrics struct {
	CPUUsage           float64   `json:"cpuUsage"`
	MemoryUsage        int64     `json:"memoryUsage"`
	NetworkBytesIn     int64     `json:"networkBytesIn"`
	NetworkBytesOut    int64     `json:"networkBytesOut"`
	ActiveSubscriptions int       `json:"activeSubscriptions"`
	MessagesProcessed  int64     `json:"messagesProcessed"`
	ErrorCount         int64     `json:"errorCount"`
	LastUpdated        time.Time `json:"lastUpdated"`
}

// NewXAppFramework creates a new xApp framework instance
func NewXAppFramework(clientManager *ClientManager, messageBus *RMRMessageBus, serviceRegistry *ServiceModelRegistry) *XAppFramework {
	ctx, cancel := context.WithCancel(context.Background())
	
	framework := &XAppFramework{
		clientManager:   clientManager,
		messageBus:      messageBus,
		serviceRegistry: serviceRegistry,
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Initialize framework components
	framework.registry = NewXAppRegistry()
	framework.resourceManager = NewXAppResourceManager()
	framework.communicationAPI = NewXAppCommunicationAPI(clientManager, messageBus)
	framework.configManager = NewXAppConfigManager()
	framework.lifecycleManager = NewXAppLifecycleManager(clientManager)
	framework.subscriptionMgr = NewXAppSubscriptionManager(clientManager, messageBus)
	framework.healthMonitor = NewXAppHealthMonitor()
	
	return framework
}

// Start starts the xApp framework
func (f *XAppFramework) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	log.Println("Starting xApp Framework...")
	
	// Start registry
	if err := f.registry.Start(f.ctx); err != nil {
		return fmt.Errorf("failed to start xApp registry: %w", err)
	}
	
	// Start resource manager
	if err := f.resourceManager.Start(f.ctx); err != nil {
		return fmt.Errorf("failed to start resource manager: %w", err)
	}
	
	// Start communication API
	if err := f.communicationAPI.Start(f.ctx); err != nil {
		return fmt.Errorf("failed to start communication API: %w", err)
	}
	
	// Start configuration manager
	if err := f.configManager.Start(f.ctx); err != nil {
		return fmt.Errorf("failed to start config manager: %w", err)
	}
	
	// Start lifecycle manager
	if err := f.lifecycleManager.Start(f.ctx); err != nil {
		return fmt.Errorf("failed to start lifecycle manager: %w", err)
	}
	
	log.Println("xApp Framework started successfully")
	return nil
}

// Stop stops the xApp framework
func (f *XAppFramework) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	log.Println("Stopping xApp Framework...")
	
	// Cancel context to stop all components
	f.cancel()
	
	// Stop components in reverse order
	if f.lifecycleManager != nil {
		f.lifecycleManager.Stop()
	}
	if f.configManager != nil {
		f.configManager.Stop()
	}
	if f.communicationAPI != nil {
		f.communicationAPI.Stop()
	}
	if f.resourceManager != nil {
		f.resourceManager.Stop()
	}
	if f.registry != nil {
		f.registry.Stop()
	}
	
	log.Println("xApp Framework stopped")
	return nil
}

// RegisterXApp registers a new xApp with the framework
func (f *XAppFramework) RegisterXApp(descriptor *XAppDescriptor) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Validate descriptor
	if err := f.validateXAppDescriptor(descriptor); err != nil {
		return fmt.Errorf("invalid xApp descriptor: %w", err)
	}
	
	// Register with registry
	if err := f.registry.Register(descriptor); err != nil {
		return fmt.Errorf("failed to register xApp: %w", err)
	}
	
	log.Printf("xApp %s v%s registered successfully", descriptor.Name, descriptor.Version)
	return nil
}

// DeployXApp deploys an xApp instance
func (f *XAppFramework) DeployXApp(name, version string, config map[string]interface{}) (*XAppInstance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Get xApp descriptor
	descriptor, err := f.registry.GetXApp(name, version)
	if err != nil {
		return nil, fmt.Errorf("xApp not found: %w", err)
	}
	
	// Deploy using lifecycle manager
	instance, err := f.lifecycleManager.Deploy(descriptor, config)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy xApp: %w", err)
	}
	
	log.Printf("xApp %s v%s deployed with instance ID %s", name, version, instance.ID)
	return instance, nil
}

// UndeployXApp undeploys an xApp instance
func (f *XAppFramework) UndeployXApp(instanceID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if err := f.lifecycleManager.Undeploy(instanceID); err != nil {
		return fmt.Errorf("failed to undeploy xApp: %w", err)
	}
	
	log.Printf("xApp instance %s undeployed successfully", instanceID)
	return nil
}

// GetXAppInstance returns an xApp instance by ID
func (f *XAppFramework) GetXAppInstance(instanceID string) (*XAppInstance, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.lifecycleManager.GetInstance(instanceID)
}

// ListXAppInstances returns all xApp instances
func (f *XAppFramework) ListXAppInstances() ([]*XAppInstance, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.lifecycleManager.ListInstances()
}

// GetXAppRegistry returns the xApp registry
func (f *XAppFramework) GetXAppRegistry() *XAppRegistry {
	return f.registry
}

// GetResourceManager returns the resource manager
func (f *XAppFramework) GetResourceManager() *XAppResourceManager {
	return f.resourceManager
}

// GetCommunicationAPI returns the communication API
func (f *XAppFramework) GetCommunicationAPI() *XAppCommunicationAPI {
	return f.communicationAPI
}

// GetConfigManager returns the configuration manager
func (f *XAppFramework) GetConfigManager() *XAppConfigManager {
	return f.configManager
}

// GetLifecycleManager returns the lifecycle manager
func (f *XAppFramework) GetLifecycleManager() *XAppLifecycleManager {
	return f.lifecycleManager
}

// validateXAppDescriptor validates an xApp descriptor
func (f *XAppFramework) validateXAppDescriptor(descriptor *XAppDescriptor) error {
	if descriptor.Name == "" {
		return fmt.Errorf("xApp name is required")
	}
	if descriptor.Version == "" {
		return fmt.Errorf("xApp version is required")
	}
	if len(descriptor.ServiceModels) == 0 {
		return fmt.Errorf("at least one service model is required")
	}
	if descriptor.ResourceLimits.CPU == "" {
		return fmt.Errorf("CPU limit is required")
	}
	if descriptor.ResourceLimits.Memory == "" {
		return fmt.Errorf("memory limit is required")
	}
	if descriptor.Endpoints.HTTP == "" {
		return fmt.Errorf("HTTP endpoint is required")
	}
	if descriptor.Endpoints.RMRData == 0 {
		return fmt.Errorf("RMR data port is required")
	}
	
	return nil
}

// GetFrameworkStatus returns the current status of the framework
func (f *XAppFramework) GetFrameworkStatus() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	registeredApps, _ := f.registry.ListXApps()
	runningInstances, _ := f.lifecycleManager.ListInstances()
	
	var runningCount int
	for _, instance := range runningInstances {
		if instance.Status == XAppStatusRunning {
			runningCount++
		}
	}
	
	return map[string]interface{}{
		"status":           "running",
		"registeredApps":   len(registeredApps),
		"runningInstances": runningCount,
		"totalInstances":   len(runningInstances),
		"timestamp":        time.Now().UTC(),
	}
}