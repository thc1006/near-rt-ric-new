/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"sync"
	"time"
)

// ComponentType represents the type of O-RAN SC component
type ComponentType string

const (
	ComponentTypeE2Manager       ComponentType = "e2manager"
	ComponentTypeSubscriptionMgr ComponentType = "submgr"
	ComponentTypeAppManager      ComponentType = "appmgr"
	ComponentTypeRoutingManager  ComponentType = "rtmgr"
	ComponentTypeXApp            ComponentType = "xapp"
)

// ComponentStatus represents the status of a component
type ComponentStatus string

const (
	ComponentStatusRunning ComponentStatus = "running"
	ComponentStatusStopped ComponentStatus = "stopped"
	ComponentStatusError   ComponentStatus = "error"
	ComponentStatusUnknown ComponentStatus = "unknown"
)

// Component represents an O-RAN SC component
type Component struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        ComponentType          `json:"type"`
	Status      ComponentStatus        `json:"status"`
	Version     string                 `json:"version"`
	Endpoint    string                 `json:"endpoint"`
	Metrics     map[string]interface{} `json:"metrics"`
	LastUpdated time.Time              `json:"lastUpdated"`
}

// DiscoveryService handles auto-discovery of O-RAN SC components
type DiscoveryService struct {
	clients    *ClientManager
	components map[string]*Component
	mutex      sync.RWMutex
	stopCh     chan struct{}
	ticker     *time.Ticker
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(clients *ClientManager) *DiscoveryService {
	return &DiscoveryService{
		clients:    clients,
		components: make(map[string]*Component),
		stopCh:     make(chan struct{}),
		ticker:     time.NewTicker(30 * time.Second), // Check every 30 seconds
	}
}

// Start starts the discovery service
func (ds *DiscoveryService) Start(wsHub *WebSocketHub) {
	log.Info("Starting component discovery service")

	// Initial discovery
	ds.discoverComponents()

	// Periodic discovery
	go func() {
		for {
			select {
			case <-ds.ticker.C:
				ds.discoverComponents()
				// Broadcast updates to WebSocket clients
				if wsHub != nil {
					ds.broadcastComponentUpdates(wsHub)
				}
			case <-ds.stopCh:
				return
			}
		}
	}()
}

// Stop stops the discovery service
func (ds *DiscoveryService) Stop() {
	log.Info("Stopping component discovery service")
	ds.ticker.Stop()
	close(ds.stopCh)
}

// discoverComponents discovers and updates component status
func (ds *DiscoveryService) discoverComponents() {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	log.Debug("Discovering O-RAN SC components")

	// Discover E2 Manager
	ds.discoverE2Manager()

	// Discover Subscription Manager
	ds.discoverSubscriptionManager()

	// Discover App Manager
	ds.discoverAppManager()

	// TODO: Add discovery for other components like Routing Manager
}

// discoverE2Manager discovers E2 Manager component
func (ds *DiscoveryService) discoverE2Manager() {
	componentID := "e2manager"

	component := &Component{
		ID:          componentID,
		Name:        "E2 Manager",
		Type:        ComponentTypeE2Manager,
		Endpoint:    ds.clients.config.E2MgrEndpoint,
		LastUpdated: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	if ds.clients.IsE2ManagerConnected() {
		component.Status = ComponentStatusRunning
		component.Version = ds.getE2ManagerVersion()
		component.Metrics = ds.getE2ManagerMetrics()
	} else {
		component.Status = ComponentStatusError
		// Try to reconnect
		if err := ds.clients.Reconnect(); err != nil {
			log.Debugf("Failed to reconnect to E2 Manager: %v", err)
		}
	}

	ds.components[componentID] = component
}

// discoverSubscriptionManager discovers Subscription Manager component
func (ds *DiscoveryService) discoverSubscriptionManager() {
	componentID := "submgr"

	component := &Component{
		ID:          componentID,
		Name:        "Subscription Manager",
		Type:        ComponentTypeSubscriptionMgr,
		Endpoint:    ds.clients.config.SubmgrEndpoint,
		LastUpdated: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	if ds.clients.IsSubscriptionManagerConnected() {
		component.Status = ComponentStatusRunning
		component.Version = ds.getSubscriptionManagerVersion()
		component.Metrics = ds.getSubscriptionManagerMetrics()
	} else {
		component.Status = ComponentStatusError
	}

	ds.components[componentID] = component
}

// discoverAppManager discovers App Manager component
func (ds *DiscoveryService) discoverAppManager() {
	componentID := "appmgr"

	component := &Component{
		ID:          componentID,
		Name:        "App Manager",
		Type:        ComponentTypeAppManager,
		Endpoint:    ds.clients.config.AppmgrEndpoint,
		LastUpdated: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	// Try to reach App Manager via HTTP
	if ds.isAppManagerReachable() {
		component.Status = ComponentStatusRunning
		component.Version = ds.getAppManagerVersion()
		component.Metrics = ds.getAppManagerMetrics()
	} else {
		component.Status = ComponentStatusError
	}

	ds.components[componentID] = component
}

// Helper methods to get component information
func (ds *DiscoveryService) getE2ManagerVersion() string {
	// TODO: Implement actual version retrieval from E2 Manager
	return "1.0.0"
}

func (ds *DiscoveryService) getE2ManagerMetrics() map[string]interface{} {
	// TODO: Implement actual metrics retrieval from E2 Manager
	return map[string]interface{}{
		"connected_nodes":    0,
		"active_connections": 0,
	}
}

func (ds *DiscoveryService) getSubscriptionManagerVersion() string {
	// TODO: Implement actual version retrieval from Subscription Manager
	return "1.0.0"
}

func (ds *DiscoveryService) getSubscriptionManagerMetrics() map[string]interface{} {
	// TODO: Implement actual metrics retrieval from Subscription Manager
	return map[string]interface{}{
		"active_subscriptions": 0,
		"total_indications":    0,
	}
}

func (ds *DiscoveryService) isAppManagerReachable() bool {
	// TODO: Implement actual health check for App Manager
	return false // Placeholder
}

func (ds *DiscoveryService) getAppManagerVersion() string {
	// TODO: Implement actual version retrieval from App Manager
	return "1.0.0"
}

func (ds *DiscoveryService) getAppManagerMetrics() map[string]interface{} {
	// TODO: Implement actual metrics retrieval from App Manager
	return map[string]interface{}{
		"deployed_xapps": 0,
		"running_xapps":  0,
	}
}

// GetComponents returns all discovered components
func (ds *DiscoveryService) GetComponents() map[string]*Component {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	// Create a copy to avoid race conditions
	components := make(map[string]*Component)
	for k, v := range ds.components {
		components[k] = v
	}
	return components
}

// GetComponent returns a specific component by ID
func (ds *DiscoveryService) GetComponent(id string) (*Component, bool) {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	component, exists := ds.components[id]
	return component, exists
}

// GetComponentStatus returns the status of all components
func (ds *DiscoveryService) GetComponentStatus() map[string]ComponentStatus {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	status := make(map[string]ComponentStatus)
	for id, component := range ds.components {
		status[id] = component.Status
	}
	return status
}

// broadcastComponentUpdates sends component updates to WebSocket clients
func (ds *DiscoveryService) broadcastComponentUpdates(wsHub *WebSocketHub) {
	components := ds.GetComponents()

	message := map[string]interface{}{
		"type":      "component_update",
		"data":      components,
		"timestamp": time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Errorf("Failed to marshal component update message: %v", err)
		return
	}

	wsHub.broadcast <- data
}
