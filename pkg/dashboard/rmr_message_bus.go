// Package dashboard provides RMR message bus implementation
// for high-performance message routing in the O-RAN Near-RT RIC
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// RMR Message Types - O-RAN Near-RT RIC specific
const (
	// Subscription messages
	RMR_MSG_E2_SUBSCRIPTION_REQ      = 12010
	RMR_MSG_E2_SUBSCRIPTION_RESP     = 12011  
	RMR_MSG_E2_SUBSCRIPTION_FAILURE  = 12012
	RMR_MSG_E2_SUBSCRIPTION_DEL_REQ  = 12020
	RMR_MSG_E2_SUBSCRIPTION_DEL_RESP = 12021
	RMR_MSG_E2_SUBSCRIPTION_DEL_FAILURE = 12022

	// Indication messages
	RMR_MSG_E2_INDICATION            = 12050

	// Control messages  
	RMR_MSG_E2_CONTROL_REQ           = 12040
	RMR_MSG_E2_CONTROL_ACK           = 12041
	RMR_MSG_E2_CONTROL_FAILURE       = 12042

	// Setup messages
	RMR_MSG_E2_SETUP_REQ             = 12001
	RMR_MSG_E2_SETUP_RESP            = 12002
	RMR_MSG_E2_SETUP_FAILURE         = 12003

	// Node configuration messages
	RMR_MSG_E2_NODE_CONFIG_UPDATE    = 12004
	RMR_MSG_E2_NODE_CONFIG_UPDATE_ACK = 12005
	RMR_MSG_E2_NODE_CONFIG_UPDATE_FAILURE = 12006

	// Error indication
	RMR_MSG_E2_ERROR_INDICATION      = 12007

	// Reset messages
	RMR_MSG_E2_RESET_REQ             = 12008
	RMR_MSG_E2_RESET_RESP            = 12009

	// Service update messages
	RMR_MSG_E2_SERVICE_UPDATE        = 12030
	RMR_MSG_E2_SERVICE_QUERY         = 12031

	// Custom dashboard messages
	RMR_MSG_DASHBOARD_HEALTH_CHECK   = 30001
	RMR_MSG_ROUTING_UPDATE           = 30002
	RMR_MSG_COMPONENT_STATUS         = 30003
)

// NOTE: RMRMessage type moved to types.go to avoid redeclaration

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

// RMRMessageBus provides high-performance message bus functionality
type RMRMessageBus struct {
	config       *RMRConfig
	conn         net.Conn
	handlers     map[uint32][]MessageHandler
	endpoints    map[string]*RMREndpoint
	routingTable map[uint32]*RMRRoutingEntry
	stats        *RMRStats
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex

	// Metrics
	messagesReceived prometheus.Counter
	messagesSent     prometheus.Counter
	messageErrors    prometheus.Counter
	latencyHistogram prometheus.Histogram
	activeEndpoints  prometheus.Gauge
}

// NOTE: MessageHandler type moved to types.go to avoid redeclaration

// RMRConfig represents RMR configuration
type RMRConfig struct {
	ListenAddress string                     `json:"listenAddress"`
	ListenPort    uint32                     `json:"listenPort"`
	RoutingTable  map[uint32][]string       `json:"routingTable"`
	Endpoints     map[string]string         `json:"endpoints"`
	BufferSize    int                       `json:"bufferSize"`
	Workers       int                       `json:"workers"`
	RetryAttempts int                       `json:"retryAttempts"`
	RetryDelay    time.Duration             `json:"retryDelay"`
	Timeout       time.Duration             `json:"timeout"`
}

// RMRStats tracks RMR message bus statistics
type RMRStats struct {
	MessagesReceived uint64 `json:"messagesReceived"`
	MessagesSent     uint64 `json:"messagesSent"`
	MessagesDropped  uint64 `json:"messagesDropped"`
	Errors           uint64 `json:"errors"`
	Latency          RMRLatencyStats `json:"latency"`
	Throughput       RMRThroughputStats `json:"throughput"`
}

// RMRLatencyStats tracks latency statistics
type RMRLatencyStats struct {
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	P95     float64 `json:"p95"`
	P99     float64 `json:"p99"`
}

// RMRThroughputStats tracks throughput statistics
type RMRThroughputStats struct {
	MessagesPerSecond float64 `json:"messagesPerSecond"`
	BytesPerSecond    float64 `json:"bytesPerSecond"`
	Peak              float64 `json:"peak"`
	Average           float64 `json:"average"`
}

// NewRMRMessageBus creates a new RMR message bus
func NewRMRMessageBus(config *RMRConfig) *RMRMessageBus {
	if config == nil {
		config = &RMRConfig{
			ListenAddress: "0.0.0.0",
			ListenPort:    4560,
			BufferSize:    1024,
			Workers:       4,
			RetryAttempts: 3,
			RetryDelay:    100 * time.Millisecond,
			Timeout:       5 * time.Second,
			RoutingTable:  make(map[uint32][]string),
			Endpoints:     make(map[string]string),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &RMRMessageBus{
		config:       config,
		handlers:     make(map[uint32][]MessageHandler),
		endpoints:    make(map[string]*RMREndpoint),
		routingTable: make(map[uint32]*RMRRoutingEntry),
		stats:        &RMRStats{},
		ctx:          ctx,
		cancel:       cancel,
	}

	bus.initMetrics()
	bus.loadRoutingTable()

	return bus
}

// initMetrics initializes Prometheus metrics
func (bus *RMRMessageBus) initMetrics() {
	bus.messagesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rmr_messages_received_total",
		Help: "Total number of RMR messages received",
	})

	bus.messagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rmr_messages_sent_total", 
		Help: "Total number of RMR messages sent",
	})

	bus.messageErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rmr_message_errors_total",
		Help: "Total number of RMR message errors",
	})

	bus.latencyHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "rmr_message_latency_seconds",
		Help:    "RMR message processing latency",
		Buckets: prometheus.DefBuckets,
	})

	bus.activeEndpoints = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rmr_active_endpoints",
		Help: "Number of active RMR endpoints",
	})

	// Register metrics
	prometheus.MustRegister(bus.messagesReceived, bus.messagesSent, 
		bus.messageErrors, bus.latencyHistogram, bus.activeEndpoints)
}

// loadRoutingTable loads the routing table from configuration
func (bus *RMRMessageBus) loadRoutingTable() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	for msgType, endpoints := range bus.config.RoutingTable {
		bus.routingTable[msgType] = &RMRRoutingEntry{
			MessageType: msgType,
			Endpoints:   endpoints,
			Policy:      "round_robin", // default policy
		}
	}

	// Load endpoints
	for name, address := range bus.config.Endpoints {
		bus.endpoints[name] = &RMREndpoint{
			Name:     name,
			Address:  address,
			Port:     bus.config.ListenPort,
			IsActive: true,
			LastSeen: time.Now(),
		}
	}

	bus.activeEndpoints.Set(float64(len(bus.endpoints)))
}

// RegisterHandler registers a message handler for specific message types
func (bus *RMRMessageBus) RegisterHandler(messageType uint32, handler MessageHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.handlers[messageType] == nil {
		bus.handlers[messageType] = make([]MessageHandler, 0)
	}
	bus.handlers[messageType] = append(bus.handlers[messageType], handler)

	log.Printf("Registered handler for message type %d", messageType)
}

// Send sends an RMR message
func (bus *RMRMessageBus) Send(msg *RMRMessage) error {
	start := time.Now()
	defer func() {
		bus.latencyHistogram.Observe(time.Since(start).Seconds())
	}()

	bus.mu.RLock()
	routingEntry, exists := bus.routingTable[msg.MessageType]
	bus.mu.RUnlock()

	if !exists {
		bus.messageErrors.Inc()
		return fmt.Errorf("no routing entry for message type %d", msg.MessageType)
	}

	// Select target endpoint based on routing policy
	targetEndpoint, err := bus.selectEndpoint(routingEntry)
	if err != nil {
		bus.messageErrors.Inc()
		return fmt.Errorf("failed to select endpoint: %w", err)
	}

	// Send message to target endpoint
	if err := bus.sendToEndpoint(msg, targetEndpoint); err != nil {
		bus.messageErrors.Inc()
		return fmt.Errorf("failed to send to endpoint %s: %w", targetEndpoint, err)
	}

	bus.messagesSent.Inc()
	bus.stats.MessagesSent++

	log.Printf("Sent message type %d to endpoint %s", msg.MessageType, targetEndpoint)
	return nil
}

// selectEndpoint selects target endpoint based on routing policy
func (bus *RMRMessageBus) selectEndpoint(entry *RMRRoutingEntry) (string, error) {
	if len(entry.Endpoints) == 0 {
		return "", fmt.Errorf("no endpoints available for message type %d", entry.MessageType)
	}

	// For now, use simple round-robin (in production, implement proper load balancing)
	return entry.Endpoints[0], nil
}

// sendToEndpoint sends message to specific endpoint
func (bus *RMRMessageBus) sendToEndpoint(msg *RMRMessage, endpoint string) error {
	// In a real implementation, this would serialize the message and send over network
	// For now, we'll just log it
	
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	log.Printf("Sending %d bytes to endpoint %s", len(payload), endpoint)
	
	// Update endpoint last seen time
	bus.mu.Lock()
	if ep, exists := bus.endpoints[endpoint]; exists {
		ep.LastSeen = time.Now()
	}
	bus.mu.Unlock()

	return nil
}

// Receive receives and processes RMR messages
func (bus *RMRMessageBus) Receive(msg *RMRMessage) error {
	start := time.Now()
	defer func() {
		bus.latencyHistogram.Observe(time.Since(start).Seconds())
	}()

	bus.messagesReceived.Inc()
	bus.stats.MessagesReceived++

	bus.mu.RLock()
	handlers, exists := bus.handlers[msg.MessageType]
	bus.mu.RUnlock()

	if !exists {
		log.Printf("No handlers registered for message type %d", msg.MessageType)
		return nil
	}

	// Process message with all registered handlers
	for _, handler := range handlers {
		if err := handler.HandleMessage(bus.ctx, msg); err != nil {
			log.Printf("Handler error for message type %d: %v", msg.MessageType, err)
			bus.messageErrors.Inc()
		}
	}

	return nil
}

// Start starts the RMR message bus
func (bus *RMRMessageBus) Start() error {
	log.Printf("Starting RMR message bus on %s:%d", bus.config.ListenAddress, bus.config.ListenPort)

	// Start worker goroutines
	for i := 0; i < bus.config.Workers; i++ {
		bus.wg.Add(1)
		go bus.worker(i)
	}

	// Start metrics collector
	bus.wg.Add(1)
	go bus.metricsCollector()

	// Start health checker
	bus.wg.Add(1)
	go bus.healthChecker()

	log.Println("RMR message bus started successfully")
	return nil
}

// Stop stops the RMR message bus
func (bus *RMRMessageBus) Stop() error {
	log.Println("Stopping RMR message bus...")

	bus.cancel()
	
	if bus.conn != nil {
		bus.conn.Close()
	}

	bus.wg.Wait()

	log.Println("RMR message bus stopped")
	return nil
}

// worker processes messages in background
func (bus *RMRMessageBus) worker(workerID int) {
	defer bus.wg.Done()

	log.Printf("RMR worker %d started", workerID)

	for {
		select {
		case <-bus.ctx.Done():
			log.Printf("RMR worker %d stopping", workerID)
			return
		default:
			// In a real implementation, this would read from message queues
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// metricsCollector collects performance metrics
func (bus *RMRMessageBus) metricsCollector() {
	defer bus.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bus.ctx.Done():
			return
		case <-ticker.C:
			bus.collectMetrics()
		}
	}
}

// collectMetrics collects and updates performance metrics
func (bus *RMRMessageBus) collectMetrics() {
	bus.mu.RLock()
	activeCount := 0
	for _, endpoint := range bus.endpoints {
		if endpoint.IsActive {
			activeCount++
		}
	}
	bus.mu.RUnlock()

	bus.activeEndpoints.Set(float64(activeCount))
}

// healthChecker monitors endpoint health
func (bus *RMRMessageBus) healthChecker() {
	defer bus.wg.Done()

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

// checkEndpointHealth checks the health of all endpoints
func (bus *RMRMessageBus) checkEndpointHealth() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	now := time.Now()
	for name, endpoint := range bus.endpoints {
		// Mark endpoint inactive if not seen for 2 minutes
		if now.Sub(endpoint.LastSeen) > 2*time.Minute {
			if endpoint.IsActive {
				log.Printf("Endpoint %s marked as inactive", name)
				endpoint.IsActive = false
			}
		}
	}
}

// GetStats returns current message bus statistics
func (bus *RMRMessageBus) GetStats() *RMRStats {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// Return a copy of stats
	statsCopy := *bus.stats
	return &statsCopy
}

// UpdateRoutingTable updates the routing table with new entries
func (bus *RMRMessageBus) UpdateRoutingTable(entries map[uint32]*RMRRoutingEntry) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	for msgType, entry := range entries {
		bus.routingTable[msgType] = entry
		log.Printf("Updated routing entry for message type %d", msgType)
	}
}

// GetEndpoints returns information about all endpoints
func (bus *RMRMessageBus) GetEndpoints() map[string]*RMREndpoint {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// Return a copy of endpoints
	endpoints := make(map[string]*RMREndpoint)
	for name, endpoint := range bus.endpoints {
		endpointCopy := *endpoint
		endpoints[name] = &endpointCopy
	}

	return endpoints
}

// AddEndpoint adds a new endpoint
func (bus *RMRMessageBus) AddEndpoint(name, address string, port uint32) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.endpoints[name] = &RMREndpoint{
		Name:     name,
		Address:  address,
		Port:     port,
		IsActive: true,
		LastSeen: time.Now(),
	}

	log.Printf("Added endpoint %s at %s:%d", name, address, port)
}

// RemoveEndpoint removes an endpoint
func (bus *RMRMessageBus) RemoveEndpoint(name string) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if _, exists := bus.endpoints[name]; exists {
		delete(bus.endpoints, name)
		log.Printf("Removed endpoint %s", name)
	}
}