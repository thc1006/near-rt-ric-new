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
	"crypto/rand"
	"encoding/hex"
)

// XAppLifecycleManager manages the lifecycle of xApp instances
type XAppLifecycleManager struct {
	instances     map[string]*XAppInstance
	clientManager *ClientManager
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewXAppLifecycleManager creates a new xApp lifecycle manager
func NewXAppLifecycleManager(clientManager *ClientManager) *XAppLifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &XAppLifecycleManager{
		instances:     make(map[string]*XAppInstance),
		clientManager: clientManager,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the lifecycle manager
func (lm *XAppLifecycleManager) Start(ctx context.Context) error {
	log.Println("Starting xApp Lifecycle Manager...")
	
	// Start health monitoring
	go lm.healthMonitor()
	
	log.Println("xApp Lifecycle Manager started")
	return nil
}

// Stop stops the lifecycle manager
func (lm *XAppLifecycleManager) Stop() {
	log.Println("Stopping xApp Lifecycle Manager...")
	
	if lm.cancel != nil {
		lm.cancel()
	}
	
	log.Println("xApp Lifecycle Manager stopped")
}

// Deploy deploys a new xApp instance
func (lm *XAppLifecycleManager) Deploy(descriptor *XAppDescriptor, config map[string]interface{}) (*XAppInstance, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	// Generate instance ID
	instanceID, err := lm.generateInstanceID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate instance ID: %w", err)
	}
	
	// Create instance
	instance := &XAppInstance{
		ID:            instanceID,
		Descriptor:    descriptor,
		Status:        XAppStatusPending,
		StartedAt:     time.Now().UTC(),
		LastHealthy:   time.Now().UTC(),
		Subscriptions: make(map[string]*Subscription),
		Metrics: XAppMetrics{
			LastUpdated: time.Now().UTC(),
		},
		Environment: make(map[string]string),
	}
	
	// Merge configuration into environment
	for k, v := range config {
		instance.Environment[k] = fmt.Sprintf("%v", v)
	}
	
	// Add standard environment variables
	instance.Environment["XAPP_NAME"] = descriptor.Name
	instance.Environment["XAPP_VERSION"] = descriptor.Version
	instance.Environment["XAPP_INSTANCE_ID"] = instanceID
	instance.Environment["RMR_DATA_PORT"] = fmt.Sprintf("%d", descriptor.Endpoints.RMRData)
	instance.Environment["RMR_ROUTE_PORT"] = fmt.Sprintf("%d", descriptor.Endpoints.RMRRoute)
	
	// Store instance
	lm.instances[instanceID] = instance
	
	// Start deployment process
	go lm.deployInstance(instance)
	
	log.Printf("xApp deployment initiated for %s v%s (instance: %s)", 
		descriptor.Name, descriptor.Version, instanceID)
	
	return instance, nil
}

// Undeploy undeploys an xApp instance
func (lm *XAppLifecycleManager) Undeploy(instanceID string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	instance, exists := lm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	
	// Update status
	instance.Status = XAppStatusStopped
	
	// Clean up subscriptions
	for subID := range instance.Subscriptions {
		if err := lm.cleanupSubscription(instanceID, subID); err != nil {
			log.Printf("Warning: failed to cleanup subscription %s: %v", subID, err)
		}
	}
	
	// Remove from instances
	delete(lm.instances, instanceID)
	
	log.Printf("xApp instance %s undeployed", instanceID)
	return nil
}

// GetInstance returns an xApp instance by ID
func (lm *XAppLifecycleManager) GetInstance(instanceID string) (*XAppInstance, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	instance, exists := lm.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}
	
	return instance, nil
}

// ListInstances returns all xApp instances
func (lm *XAppLifecycleManager) ListInstances() ([]*XAppInstance, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	instances := make([]*XAppInstance, 0, len(lm.instances))
	for _, instance := range lm.instances {
		instances = append(instances, instance)
	}
	
	return instances, nil
}

// UpdateInstanceStatus updates the status of an xApp instance
func (lm *XAppLifecycleManager) UpdateInstanceStatus(instanceID string, status XAppStatus) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	instance, exists := lm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	
	instance.Status = status
	if status == XAppStatusRunning {
		instance.LastHealthy = time.Now().UTC()
	}
	
	log.Printf("xApp instance %s status updated to %s", instanceID, status)
	return nil
}

// AddSubscription adds a subscription to an xApp instance
func (lm *XAppLifecycleManager) AddSubscription(instanceID string, subscription *Subscription) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	instance, exists := lm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	
	instance.Subscriptions[subscription.ID] = subscription
	instance.Metrics.ActiveSubscriptions = len(instance.Subscriptions)
	instance.Metrics.LastUpdated = time.Now().UTC()
	
	log.Printf("Subscription %s added to xApp instance %s", subscription.ID, instanceID)
	return nil
}

// RemoveSubscription removes a subscription from an xApp instance
func (lm *XAppLifecycleManager) RemoveSubscription(instanceID, subscriptionID string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	instance, exists := lm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	
	delete(instance.Subscriptions, subscriptionID)
	instance.Metrics.ActiveSubscriptions = len(instance.Subscriptions)
	instance.Metrics.LastUpdated = time.Now().UTC()
	
	log.Printf("Subscription %s removed from xApp instance %s", subscriptionID, instanceID)
	return nil
}

// UpdateMetrics updates the metrics for an xApp instance
func (lm *XAppLifecycleManager) UpdateMetrics(instanceID string, metrics XAppMetrics) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	instance, exists := lm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	
	metrics.LastUpdated = time.Now().UTC()
	instance.Metrics = metrics
	
	return nil
}

// deployInstance handles the actual deployment of an xApp instance
func (lm *XAppLifecycleManager) deployInstance(instance *XAppInstance) {
	// Simulate deployment process
	time.Sleep(2 * time.Second)
	
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	// Check if instance still exists (might have been undeployed)
	if _, exists := lm.instances[instance.ID]; !exists {
		return
	}
	
	// Update status to running
	instance.Status = XAppStatusRunning
	instance.LastHealthy = time.Now().UTC()
	
	// Set pod and service names (simulated)
	instance.PodName = fmt.Sprintf("%s-%s-pod", instance.Descriptor.Name, instance.ID[:8])
	instance.ServiceName = fmt.Sprintf("%s-%s-svc", instance.Descriptor.Name, instance.ID[:8])
	
	log.Printf("xApp instance %s deployed successfully", instance.ID)
}

// healthMonitor monitors the health of xApp instances
func (lm *XAppLifecycleManager) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			lm.performHealthCheck()
		case <-lm.ctx.Done():
			return
		}
	}
}

// performHealthCheck performs health checks on all running instances
func (lm *XAppLifecycleManager) performHealthCheck() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	for instanceID, instance := range lm.instances {
		if instance.Status != XAppStatusRunning {
			continue
		}
		
		// Simulate health check
		if lm.isInstanceHealthy(instance) {
			instance.LastHealthy = time.Now().UTC()
		} else {
			log.Printf("Health check failed for xApp instance %s", instanceID)
			instance.Status = XAppStatusFailed
		}
	}
}

// isInstanceHealthy checks if an instance is healthy
func (lm *XAppLifecycleManager) isInstanceHealthy(instance *XAppInstance) bool {
	// Simulate health check logic
	// In a real implementation, this would check HTTP endpoints, metrics, etc.
	
	// Consider instance unhealthy if it hasn't been updated in 5 minutes
	if time.Since(instance.LastHealthy) > 5*time.Minute {
		return false
	}
	
	return true
}

// cleanupSubscription cleans up a subscription when an instance is undeployed
func (lm *XAppLifecycleManager) cleanupSubscription(instanceID, subscriptionID string) error {
	// In a real implementation, this would call the subscription manager
	// to properly clean up the subscription
	log.Printf("Cleaning up subscription %s for instance %s", subscriptionID, instanceID)
	return nil
}

// generateInstanceID generates a unique instance ID
func (lm *XAppLifecycleManager) generateInstanceID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RestartInstance restarts an xApp instance
func (lm *XAppLifecycleManager) RestartInstance(instanceID string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	instance, exists := lm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}
	
	// Update status to updating
	instance.Status = XAppStatusUpdating
	
	// Start restart process
	go func() {
		// Simulate restart delay
		time.Sleep(5 * time.Second)
		
		lm.mu.Lock()
		defer lm.mu.Unlock()
		
		// Check if instance still exists
		if _, exists := lm.instances[instanceID]; exists {
			instance.Status = XAppStatusRunning
			instance.StartedAt = time.Now().UTC()
			instance.LastHealthy = time.Now().UTC()
			log.Printf("xApp instance %s restarted successfully", instanceID)
		}
	}()
	
	log.Printf("xApp instance %s restart initiated", instanceID)
	return nil
}

// GetInstancesByXApp returns all instances of a specific xApp
func (lm *XAppLifecycleManager) GetInstancesByXApp(name, version string) ([]*XAppInstance, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	var instances []*XAppInstance
	for _, instance := range lm.instances {
		if instance.Descriptor.Name == name && (version == "" || instance.Descriptor.Version == version) {
			instances = append(instances, instance)
		}
	}
	
	return instances, nil
}