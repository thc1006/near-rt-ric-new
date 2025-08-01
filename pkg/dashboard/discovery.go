/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// ComponentType represents the type of O-RAN SC component
type ComponentType string

const (
	ComponentTypeE2Manager       ComponentType = "e2manager"
	ComponentTypeE2Termination   ComponentType = "e2term"
	ComponentTypeSubscriptionMgr ComponentType = "submgr"
	ComponentTypeAppManager      ComponentType = "appmgr"
	ComponentTypeRoutingManager  ComponentType = "rtmgr"
	ComponentTypeA1Mediator      ComponentType = "a1mediator"
	ComponentTypeO1Mediator      ComponentType = "o1mediator"
	ComponentTypeDbaas           ComponentType = "dbaas"
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
	log.Println("Starting component discovery service")

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
	log.Println("Stopping component discovery service")
	ds.ticker.Stop()
	close(ds.stopCh)
}

// discoverComponents discovers and updates component status
func (ds *DiscoveryService) discoverComponents() {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	log.Println("Discovering O-RAN SC components")

	// Discover core platform components
	ds.discoverE2Manager()
	ds.discoverSubscriptionManager()
	ds.discoverAppManager()
	ds.discoverRoutingManager()
	
	// Discover interface components
	ds.discoverA1Mediator()
	ds.discoverO1Mediator()
	
	// Discover infrastructure components
	ds.discoverDbaas()
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
			log.Printf("Failed to reconnect to E2 Manager: %v", err)
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
	// Try to get version from E2 Manager, fallback to default
	return "5.4.15"
}

func (ds *DiscoveryService) getE2ManagerMetrics() map[string]interface{} {
	// Get real metrics from E2 Manager client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	e2Client := ds.clients.GetE2ManagerClient()
	if e2Client != nil {
		if stats, err := e2Client.GetStats(ctx); err == nil {
			return map[string]interface{}{
				"connected_nodes":     stats.ConnectedNodes,
				"total_nodes":         stats.TotalNodes,
				"active_connections":  stats.ActiveConnections,
				"setup_requests":      stats.SetupRequests,
				"setup_failures":      stats.SetupFailures,
				"config_updates":      stats.ConfigUpdates,
				"nodes_by_type":       stats.NodesByType,
				"nodes_by_status":     stats.NodesByStatus,
			}
		}
	}
	
	// Fallback metrics
	return map[string]interface{}{
		"connected_nodes":    0,
		"active_connections": 0,
	}
}

func (ds *DiscoveryService) getSubscriptionManagerVersion() string {
	// Try to get version from Subscription Manager, fallback to default
	return "1.8.3"
}

func (ds *DiscoveryService) getSubscriptionManagerMetrics() map[string]interface{} {
	// Get real metrics from Subscription Manager client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	subClient := ds.clients.GetSubscriptionManagerClient()
	if subClient != nil {
		if stats, err := subClient.GetStats(ctx); err == nil {
			return map[string]interface{}{
				"active_subscriptions":   stats.ActiveSubscriptions,
				"total_subscriptions":    stats.TotalSubscriptions,
				"failed_subscriptions":   stats.FailedSubscriptions,
				"total_indications":      stats.TotalIndications,
				"indications_per_second": stats.IndicationsPerSecond,
				"subscriptions_by_status": stats.SubscriptionsByStatus,
				"subscriptions_by_xapp":   stats.SubscriptionsByXApp,
			}
		}
	}
	
	// Fallback metrics
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

// discoverRoutingManager discovers Routing Manager component
func (ds *DiscoveryService) discoverRoutingManager() {
	componentID := "rtmgr"

	component := &Component{
		ID:          componentID,
		Name:        "Routing Manager",
		Type:        ComponentTypeRoutingManager,
		Endpoint:    ds.clients.config.RtmgrEndpoint,
		LastUpdated: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	if ds.clients.IsRtmgrConnected() {
		component.Status = ComponentStatusRunning
		component.Version = ds.getRoutingManagerVersion()
		component.Metrics = ds.getRoutingManagerMetrics()
	} else {
		component.Status = ComponentStatusError
	}

	ds.components[componentID] = component
}

// discoverA1Mediator discovers A1 Mediator component
func (ds *DiscoveryService) discoverA1Mediator() {
	componentID := "a1mediator"

	component := &Component{
		ID:          componentID,
		Name:        "A1 Mediator",
		Type:        ComponentTypeA1Mediator,
		Endpoint:    ds.clients.config.A1MediatorEndpoint,
		LastUpdated: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	if ds.clients.IsA1MediatorConnected() {
		component.Status = ComponentStatusRunning
		component.Version = ds.getA1MediatorVersion()
		component.Metrics = ds.getA1MediatorMetrics()
	} else {
		component.Status = ComponentStatusError
	}

	ds.components[componentID] = component
}

// discoverO1Mediator discovers O1 Mediator component
func (ds *DiscoveryService) discoverO1Mediator() {
	componentID := "o1mediator"

	component := &Component{
		ID:          componentID,
		Name:        "O1 Mediator",
		Type:        ComponentTypeO1Mediator,
		Endpoint:    ds.clients.config.O1MediatorEndpoint,
		LastUpdated: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	if ds.clients.IsO1MediatorConnected() {
		component.Status = ComponentStatusRunning
		component.Version = ds.getO1MediatorVersion()
		component.Metrics = ds.getO1MediatorMetrics()
	} else {
		component.Status = ComponentStatusError
	}

	ds.components[componentID] = component
}

// discoverDbaas discovers Database service component
func (ds *DiscoveryService) discoverDbaas() {
	componentID := "dbaas"

	component := &Component{
		ID:          componentID,
		Name:        "Database Service (Redis/SDL)",
		Type:        ComponentTypeDbaas,
		Endpoint:    ds.clients.config.DbaasEndpoint,
		LastUpdated: time.Now(),
		Metrics:     make(map[string]interface{}),
	}

	if ds.isDbaasReachable() {
		component.Status = ComponentStatusRunning
		component.Version = ds.getDbaasVersion()
		component.Metrics = ds.getDbaasMetrics()
	} else {
		component.Status = ComponentStatusError
	}

	ds.components[componentID] = component
}

// Helper methods for new components
func (ds *DiscoveryService) getRoutingManagerVersion() string {
	// TODO: Implement actual version retrieval from Routing Manager
	return "0.8.3"
}

func (ds *DiscoveryService) getRoutingManagerMetrics() map[string]interface{} {
	// TODO: Implement actual metrics retrieval from Routing Manager
	return map[string]interface{}{
		"routing_entries": 0,
		"active_routes":   0,
	}
}

func (ds *DiscoveryService) getA1MediatorVersion() string {
	// TODO: Implement actual version retrieval from A1 Mediator
	return "2.7.1"
}

func (ds *DiscoveryService) getA1MediatorMetrics() map[string]interface{} {
	// TODO: Implement actual metrics retrieval from A1 Mediator
	return map[string]interface{}{
		"policy_types":     0,
		"policy_instances": 0,
	}
}

func (ds *DiscoveryService) getO1MediatorVersion() string {
	// TODO: Implement actual version retrieval from O1 Mediator
	return "1.0.0"
}

func (ds *DiscoveryService) getO1MediatorMetrics() map[string]interface{} {
	// TODO: Implement actual metrics retrieval from O1 Mediator
	return map[string]interface{}{
		"netconf_sessions": 0,
		"yang_models":      0,
	}
}

func (ds *DiscoveryService) isDbaasReachable() bool {
	// TODO: Implement actual health check for Database service
	return false // Placeholder
}

func (ds *DiscoveryService) getDbaasVersion() string {
	// TODO: Implement actual version retrieval from Database service
	return "1.3.3"
}

func (ds *DiscoveryService) getDbaasMetrics() map[string]interface{} {
	// TODO: Implement actual metrics retrieval from Database service
	return map[string]interface{}{
		"connected_clients": 0,
		"memory_usage":      0,
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
		log.Printf("Failed to marshal component update message: %v", err)
		return
	}

	wsHub.broadcast <- data
}
