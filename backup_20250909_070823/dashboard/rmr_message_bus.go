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
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RMR Message Types per O-RAN SC specifications
const (
	RMR_MSG_E2AP_SETUP_REQ           = 12001
	RMR_MSG_E2AP_SETUP_RESP          = 12002
	RMR_MSG_E2AP_SETUP_FAILURE       = 12003
	RMR_MSG_E2AP_CONFIG_UPDATE_REQ   = 12004
	RMR_MSG_E2AP_CONFIG_UPDATE_RESP  = 12005
	RMR_MSG_E2AP_RESET_REQ           = 12006
	RMR_MSG_E2AP_RESET_RESP          = 12007
	RMR_MSG_E2AP_INDICATION          = 12008
	RMR_MSG_E2AP_CONTROL_REQ         = 12009
	RMR_MSG_E2AP_CONTROL_ACK         = 12010
	RMR_MSG_E2AP_CONTROL_FAILURE     = 12011
	RMR_MSG_E2AP_SUBSCRIPTION_REQ    = 12012
	RMR_MSG_E2AP_SUBSCRIPTION_RESP   = 12013
	RMR_MSG_E2AP_SUBSCRIPTION_FAILURE = 12014
	RMR_MSG_E2AP_SUBSCRIPTION_DELETE_REQ = 12015
	RMR_MSG_E2AP_SUBSCRIPTION_DELETE_RESP = 12016
	RMR_MSG_E2AP_SUBSCRIPTION_DELETE_FAILURE = 12017
	
	// xApp communication message types
	RMR_MSG_XAPP_REGISTER            = 20001
	RMR_MSG_XAPP_UNREGISTER          = 20002
	RMR_MSG_XAPP_CONFIG_UPDATE       = 20003
	RMR_MSG_XAPP_HEALTH_CHECK        = 20004
	
	// Platform internal message types
	RMR_MSG_PLATFORM_HEALTH          = 30001
	RMR_MSG_ROUTING_UPDATE           = 30002
	RMR_MSG_COMPONENT_STATUS         = 30003
)

// RMRMessage represents an RMR message
type RMRMessage struct {
	MessageType   uint32            `json:"messageType"`
	SubscriptionID string           `json:"subscriptionId,omitempty"`
	TransactionID string            `json:"transactionId"`
	Payload       []byte            `json:"payload"`
	Source        string            `json:"source"`
	Target        string            `json:"target,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// RMREndpoint represents an RMR endpoint
type RMREndpoint struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Port      uint32 `json:"port"`
	IsActive  bool   `json:"isActive"`
	LastSeen  time.Time `json:"lastSeen"`
}

// RMRRoutingEntry represents a routing table entry
type RMRRoutingEntry struct {
	MessageType uint32   `json:"messageType"`
	Endpoints   []string `json:"endpoints"`
	Policy      string   `json:"policy"` // "round_robin", "broadcast", "first_available"
}

// RMRRoutingTable represents the complete routing table
type RMRRoutingTable struct {
	Version   uint32                         `json:"version"`
	Entries   map[uint32]*RMRRoutingEntry   `json:"entries"`
	UpdatedAt time.Time                     `json:"updatedAt"`
}

// RMRMessageBus provides RMR message routing functionality
type RMRMessageBus struct {
	mu              sync.RWMutex
	endpoints       map[string]*RMREndpoint
	routingTable    *RMRRoutingTable
	messageHandlers map[uint32][]MessageHandler
	listeners       map[string]net.Listener
	connections     map[string]net.Conn
	isRunning       bool
	ctx             context.Context
	cancel          context.CancelFunc
	
	// Metrics
	messagesSent     prometheus.Counter
	messagesReceived prometheus.Counter
	routingErrors    prometheus.Counter
	activeEndpoints  prometheus.Gauge
}

// MessageHandler defines the interface for handling RMR messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *RMRMessage) error
	GetMessageTypes() []uint32
}

// RMRConfig represents RMR configuration
type RMRConfig struct {
	ListenAddress string                     `json:"listenAddress"`
	ListenPort    uint32                     `json:"listenPort"`
	RoutingTable  map[uint32][]string       `json:"routingTable"`
	Endpoints     map[string]string         `json:"endpoints"`
}

// NewRMRMessageBus creates a new RMR message bus
func NewRMRMessageBus(config *RMRConfig) *RMRMessageBus {
	ctx, cancel := context.WithCancel(context.Background())
	
	bus := &RMRMessageBus{
		endpoints:       make(map[string]*RMREndpoint),
		messageHandlers: make(map[uint32][]MessageHandler),
		listeners:       make(map[string]net.Listener),
		connections:     make(map[string]net.Conn),
		ctx:             ctx,
		cancel:          cancel,
		routingTable: &RMRRoutingTable{
			Version: 1,
			Entries: make(map[uint32]*RMRRoutingEntry),
			UpdatedAt: time.Now(),
		},
		
		// Initialize metrics
		messagesSent: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rmr_messages_sent_total",
			Help: "Total number of RMR messages sent",
		}),
		messagesReceived: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rmr_messages_received_total",
			Help: "Total number of RMR messages received",
		}),
		routingErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rmr_routing_errors_total",
			Help: "Total number of RMR routing errors",
		}),
		activeEndpoints: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rmr_active_endpoints",
			Help: "Number of active RMR endpoints",
		}),
	}
	
	// Initialize routing table from config
	if config != nil {
		bus.initializeRoutingTable(config)
		bus.initializeEndpoints(config)
	}
	
	return bus
}

// Start starts the RMR message bus
func (bus *RMRMessageBus) Start() error {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	if bus.isRunning {
		return fmt.Errorf("RMR message bus is already running")
	}
	
	// Start message processing goroutine
	go bus.messageProcessor()
	
	// Start health monitoring
	go bus.healthMonitor()
	
	bus.isRunning = true
	log.Println("RMR message bus started successfully")
	return nil
}

// Stop stops the RMR message bus
func (bus *RMRMessageBus) Stop() error {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	if !bus.isRunning {
		return nil
	}
	
	bus.cancel()
	
	// Close all listeners
	for name, listener := range bus.listeners {
		if err := listener.Close(); err != nil {
			log.Printf("Error closing listener %s: %v", name, err)
		}
	}
	
	// Close all connections
	for name, conn := range bus.connections {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection %s: %v", name, err)
		}
	}
	
	bus.isRunning = false
	log.Println("RMR message bus stopped")
	return nil
}

// RegisterEndpoint registers a new RMR endpoint
func (bus *RMRMessageBus) RegisterEndpoint(name, address string, port uint32) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	endpoint := &RMREndpoint{
		Name:     name,
		Address:  address,
		Port:     port,
		IsActive: true,
		LastSeen: time.Now(),
	}
	
	bus.endpoints[name] = endpoint
	bus.activeEndpoints.Set(float64(len(bus.endpoints)))
	
	log.Printf("Registered RMR endpoint: %s at %s:%d", name, address, port)
	return nil
}

// UnregisterEndpoint unregisters an RMR endpoint
func (bus *RMRMessageBus) UnregisterEndpoint(name string) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	if _, exists := bus.endpoints[name]; !exists {
		return fmt.Errorf("endpoint %s not found", name)
	}
	
	delete(bus.endpoints, name)
	bus.activeEndpoints.Set(float64(len(bus.endpoints)))
	
	log.Printf("Unregistered RMR endpoint: %s", name)
	return nil
}

// SendMessage sends an RMR message
func (bus *RMRMessageBus) SendMessage(msg *RMRMessage) error {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	
	if !bus.isRunning {
		return fmt.Errorf("RMR message bus is not running")
	}
	
	// Set transaction ID if not provided
	if msg.TransactionID == "" {
		msg.TransactionID = uuid.New().String()
	}
	
	// Set timestamp
	msg.Timestamp = time.Now()
	
	// Find routing entry for message type
	entry, exists := bus.routingTable.Entries[msg.MessageType]
	if !exists {
		bus.routingErrors.Inc()
		return fmt.Errorf("no routing entry found for message type %d", msg.MessageType)
	}
	
	// Route message based on policy
	switch entry.Policy {
	case "round_robin":
		return bus.sendRoundRobin(msg, entry.Endpoints)
	case "broadcast":
		return bus.sendBroadcast(msg, entry.Endpoints)
	case "first_available":
		return bus.sendFirstAvailable(msg, entry.Endpoints)
	default:
		return bus.sendFirstAvailable(msg, entry.Endpoints)
	}
}

// RegisterMessageHandler registers a message handler for specific message types
func (bus *RMRMessageBus) RegisterMessageHandler(handler MessageHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	for _, msgType := range handler.GetMessageTypes() {
		if bus.messageHandlers[msgType] == nil {
			bus.messageHandlers[msgType] = make([]MessageHandler, 0)
		}
		bus.messageHandlers[msgType] = append(bus.messageHandlers[msgType], handler)
	}
	
	log.Printf("Registered message handler for types: %v", handler.GetMessageTypes())
}

// UpdateRoutingTable updates the routing table
func (bus *RMRMessageBus) UpdateRoutingTable(table *RMRRoutingTable) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	table.Version = bus.routingTable.Version + 1
	table.UpdatedAt = time.Now()
	bus.routingTable = table
	
	log.Printf("Updated routing table to version %d", table.Version)
	return nil
}

// GetRoutingTable returns the current routing table
func (bus *RMRMessageBus) GetRoutingTable() *RMRRoutingTable {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	
	// Return a copy to prevent external modifications
	tableCopy := &RMRRoutingTable{
		Version:   bus.routingTable.Version,
		Entries:   make(map[uint32]*RMRRoutingEntry),
		UpdatedAt: bus.routingTable.UpdatedAt,
	}
	
	for msgType, entry := range bus.routingTable.Entries {
		entryCopy := &RMRRoutingEntry{
			MessageType: entry.MessageType,
			Endpoints:   make([]string, len(entry.Endpoints)),
			Policy:      entry.Policy,
		}
		copy(entryCopy.Endpoints, entry.Endpoints)
		tableCopy.Entries[msgType] = entryCopy
	}
	
	return tableCopy
}

// GetEndpoints returns all registered endpoints
func (bus *RMRMessageBus) GetEndpoints() map[string]*RMREndpoint {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	
	endpoints := make(map[string]*RMREndpoint)
	for name, endpoint := range bus.endpoints {
		endpoints[name] = &RMREndpoint{
			Name:     endpoint.Name,
			Address:  endpoint.Address,
			Port:     endpoint.Port,
			IsActive: endpoint.IsActive,
			LastSeen: endpoint.LastSeen,
		}
	}
	
	return endpoints
}

// GetStats returns RMR message bus statistics
func (bus *RMRMessageBus) GetStats() map[string]interface{} {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	
	return map[string]interface{}{
		"endpoints_count":     len(bus.endpoints),
		"routing_entries":     len(bus.routingTable.Entries),
		"routing_version":     bus.routingTable.Version,
		"is_running":          bus.isRunning,
		"last_table_update":   bus.routingTable.UpdatedAt,
	}
}

// Private methods

func (bus *RMRMessageBus) initializeRoutingTable(config *RMRConfig) {
	for msgType, endpoints := range config.RoutingTable {
		entry := &RMRRoutingEntry{
			MessageType: msgType,
			Endpoints:   endpoints,
			Policy:      "round_robin", // Default policy
		}
		bus.routingTable.Entries[msgType] = entry
	}
}

func (bus *RMRMessageBus) initializeEndpoints(config *RMRConfig) {
	for name, address := range config.Endpoints {
		// Parse address:port
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			log.Printf("Invalid endpoint address %s: %v", address, err)
			continue
		}
		
		portNum := uint32(0)
		if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
			log.Printf("Invalid port in address %s: %v", address, err)
			continue
		}
		
		endpoint := &RMREndpoint{
			Name:     name,
			Address:  host,
			Port:     portNum,
			IsActive: false, // Will be activated when connection is established
			LastSeen: time.Now(),
		}
		
		bus.endpoints[name] = endpoint
	}
}

func (bus *RMRMessageBus) sendRoundRobin(msg *RMRMessage, endpoints []string) error {
	// Simple round-robin implementation
	if len(endpoints) == 0 {
		return fmt.Errorf("no endpoints available")
	}
	
	// Use timestamp as seed for round-robin selection
	index := int(msg.Timestamp.UnixNano()) % len(endpoints)
	targetEndpoint := endpoints[index]
	
	return bus.sendToEndpoint(msg, targetEndpoint)
}

func (bus *RMRMessageBus) sendBroadcast(msg *RMRMessage, endpoints []string) error {
	if len(endpoints) == 0 {
		return fmt.Errorf("no endpoints available")
	}
	
	var lastErr error
	successCount := 0
	
	for _, endpoint := range endpoints {
		if err := bus.sendToEndpoint(msg, endpoint); err != nil {
			lastErr = err
			log.Printf("Failed to send message to endpoint %s: %v", endpoint, err)
		} else {
			successCount++
		}
	}
	
	if successCount == 0 {
		return fmt.Errorf("failed to send message to any endpoint: %v", lastErr)
	}
	
	return nil
}

func (bus *RMRMessageBus) sendFirstAvailable(msg *RMRMessage, endpoints []string) error {
	if len(endpoints) == 0 {
		return fmt.Errorf("no endpoints available")
	}
	
	for _, endpoint := range endpoints {
		if err := bus.sendToEndpoint(msg, endpoint); err != nil {
			log.Printf("Failed to send message to endpoint %s: %v", endpoint, err)
			continue
		}
		return nil
	}
	
	return fmt.Errorf("no available endpoints for message delivery")
}

func (bus *RMRMessageBus) sendToEndpoint(msg *RMRMessage, endpointName string) error {
	endpoint, exists := bus.endpoints[endpointName]
	if !exists {
		bus.routingErrors.Inc()
		return fmt.Errorf("endpoint %s not found", endpointName)
	}
	
	if !endpoint.IsActive {
		bus.routingErrors.Inc()
		return fmt.Errorf("endpoint %s is not active", endpointName)
	}
	
	// Serialize message
	msgData, err := json.Marshal(msg)
	if err != nil {
		bus.routingErrors.Inc()
		return fmt.Errorf("failed to serialize message: %w", err)
	}
	
	// For now, simulate message sending
	// In a real implementation, this would send via TCP/UDP to the endpoint
	log.Printf("Sending RMR message type %d to endpoint %s (%s:%d)", 
		msg.MessageType, endpointName, endpoint.Address, endpoint.Port)
	
	bus.messagesSent.Inc()
	endpoint.LastSeen = time.Now()
	
	return nil
}

func (bus *RMRMessageBus) messageProcessor() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-bus.ctx.Done():
			return
		case <-ticker.C:
			// Process incoming messages
			// In a real implementation, this would read from network connections
		}
	}
}

func (bus *RMRMessageBus) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-bus.ctx.Done():
			return
		case <-ticker.C:
			bus.checkEndpointHealth()
		}
	}
}

func (bus *RMRMessageBus) checkEndpointHealth() {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	now := time.Now()
	activeCount := 0
	
	for name, endpoint := range bus.endpoints {
		// Mark endpoint as inactive if not seen for 2 minutes
		if now.Sub(endpoint.LastSeen) > 2*time.Minute {
			if endpoint.IsActive {
				endpoint.IsActive = false
				log.Printf("Endpoint %s marked as inactive", name)
			}
		} else {
			if !endpoint.IsActive {
				endpoint.IsActive = true
				log.Printf("Endpoint %s marked as active", name)
			}
			activeCount++
		}
	}
	
	bus.activeEndpoints.Set(float64(activeCount))
}