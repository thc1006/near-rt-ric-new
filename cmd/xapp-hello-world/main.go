/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// XAppConfig holds the configuration for the xApp
type XAppConfig struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	E2MgrEndpoint  string `json:"e2mgr_endpoint"`
	SubmgrEndpoint string `json:"submgr_endpoint"`
	RMRPort        int    `json:"rmr_port"`
	HTTPPort       int    `json:"http_port"`
}

// E2Node represents an E2 node
type E2Node struct {
	ID               string                 `json:"id"`
	GlobalE2NodeID   string                 `json:"globalE2NodeId"`
	ConnectionStatus string                 `json:"connectionStatus"`
	RANFunctions     []RANFunction          `json:"ranFunctions"`
	ServiceModels    []ServiceModel         `json:"serviceModels"`
	LastUpdate       time.Time              `json:"lastUpdate"`
}

// RANFunction represents a RAN function
type RANFunction struct {
	ID          uint32 `json:"id"`
	OID         string `json:"oid"`
	Definition  []byte `json:"definition"`
	Revision    uint32 `json:"revision"`
}

// ServiceModel represents a service model
type ServiceModel struct {
	OID       string        `json:"oid"`
	Name      string        `json:"name"`
	Version   string        `json:"version"`
	Functions []RANFunction `json:"functions"`
}

// Subscription represents an E2 subscription
type Subscription struct {
	ID            string    `json:"id"`
	E2NodeID      string    `json:"e2NodeId"`
	RANFunctionID uint32    `json:"ranFunctionId"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

// XAppManager manages the hello-world xApp
type XAppManager struct {
	config        *XAppConfig
	httpServer    *http.Server
	subscriptions map[string]*Subscription
	e2nodes       map[string]*E2Node
}

// NewXAppManager creates a new xApp manager
func NewXAppManager(config *XAppConfig) *XAppManager {
	return &XAppManager{
		config:        config,
		subscriptions: make(map[string]*Subscription),
		e2nodes:       make(map[string]*E2Node),
	}
}

// Start starts the xApp manager
func (m *XAppManager) Start() error {
	log.Printf("Starting xApp %s version %s", m.config.Name, m.config.Version)

	// Setup HTTP server for health checks and metrics
	m.setupHTTPServer()

	// Start E2 node discovery
	go m.discoverE2Nodes()

	// Start subscription management
	go m.manageSubscriptions()

	// Start HTTP server
	go func() {
		if err := m.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	log.Printf("xApp started successfully on port %d", m.config.HTTPPort)
	return nil
}

// setupHTTPServer configures the HTTP server
func (m *XAppManager) setupHTTPServer() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", m.handleHealth)
	mux.HandleFunc("/ready", m.handleReady)

	// Metrics endpoint
	mux.HandleFunc("/metrics", m.handleMetrics)

	// E2 nodes endpoint
	mux.HandleFunc("/e2nodes", m.handleE2Nodes)

	// Subscriptions endpoint
	mux.HandleFunc("/subscriptions", m.handleSubscriptions)

	m.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.HTTPPort),
		Handler: mux,
	}
}

// discoverE2Nodes discovers E2 nodes from E2 Manager
func (m *XAppManager) discoverE2Nodes() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.fetchE2Nodes()
		}
	}
}

// fetchE2Nodes fetches E2 nodes from E2 Manager
func (m *XAppManager) fetchE2Nodes() {
	if m.config.E2MgrEndpoint == "" {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/v1/nodeb", m.config.E2MgrEndpoint))
	if err != nil {
		log.Printf("Failed to fetch E2 nodes: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("E2 Manager returned status %d", resp.StatusCode)
		return
	}

	var nodes []E2Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		log.Printf("Failed to decode E2 nodes response: %v", err)
		return
	}

	// Update local E2 nodes cache
	for _, node := range nodes {
		m.e2nodes[node.ID] = &node
		log.Printf("Discovered E2 node: %s (status: %s)", node.ID, node.ConnectionStatus)
	}
}

// manageSubscriptions manages E2 subscriptions
func (m *XAppManager) manageSubscriptions() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.createSubscriptionsForNodes()
		}
	}
}

// createSubscriptionsForNodes creates subscriptions for connected E2 nodes
func (m *XAppManager) createSubscriptionsForNodes() {
	for nodeID, node := range m.e2nodes {
		if node.ConnectionStatus != "CONNECTED" {
			continue
		}

		// Check if we already have a subscription for this node
		subID := fmt.Sprintf("hello-world-%s", nodeID)
		if _, exists := m.subscriptions[subID]; exists {
			continue
		}

		// Create subscription for KPM service model (if available)
		for _, sm := range node.ServiceModels {
			if sm.Name == "E2SM-KPM" {
				if err := m.createSubscription(nodeID, sm.Functions[0].ID); err != nil {
					log.Printf("Failed to create subscription for node %s: %v", nodeID, err)
				} else {
					log.Printf("Created subscription %s for node %s", subID, nodeID)
				}
				break
			}
		}
	}
}

// createSubscription creates an E2 subscription
func (m *XAppManager) createSubscription(nodeID string, ranFunctionID uint32) error {
	if m.config.SubmgrEndpoint == "" {
		return fmt.Errorf("subscription manager endpoint not configured")
	}

	subID := fmt.Sprintf("hello-world-%s", nodeID)
	subscription := &Subscription{
		ID:            subID,
		E2NodeID:      nodeID,
		RANFunctionID: ranFunctionID,
		Status:        "PENDING",
		CreatedAt:     time.Now(),
	}

	// TODO: Send actual subscription request to Subscription Manager
	// For now, just store it locally
	m.subscriptions[subID] = subscription

	log.Printf("Subscription %s created for node %s", subID, nodeID)
	return nil
}

// HTTP handlers
func (m *XAppManager) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"xapp":      m.config.Name,
		"version":   m.config.Version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *XAppManager) handleReady(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *XAppManager) handleMetrics(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"e2_nodes_discovered":    len(m.e2nodes),
		"active_subscriptions":   len(m.subscriptions),
		"timestamp":              time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *XAppManager) handleE2Nodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.e2nodes)
}

func (m *XAppManager) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.subscriptions)
}

// Shutdown gracefully shuts down the xApp
func (m *XAppManager) Shutdown(ctx context.Context) error {
	log.Println("Shutting down xApp...")

	// Shutdown HTTP server
	if err := m.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	log.Println("xApp shutdown complete")
	return nil
}

// loadConfig loads configuration from environment variables
func loadConfig() *XAppConfig {
	config := &XAppConfig{
		Name:           getEnv("XAPP_NAME", "hello-world"),
		Version:        getEnv("XAPP_VERSION", "1.0.0"),
		E2MgrEndpoint:  getEnv("E2MGR_ENDPOINT", ""),
		SubmgrEndpoint: getEnv("SUBMGR_ENDPOINT", ""),
		RMRPort:        getEnvInt("RMR_PORT", 4560),
		HTTPPort:       getEnvInt("HTTP_PORT", 8080),
	}

	log.Printf("Loaded configuration: %+v", config)
	return config
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets environment variable as integer with default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := fmt.Sscanf(value, "%d", &defaultValue); err == nil && intValue == 1 {
			return defaultValue
		}
	}
	return defaultValue
}

func main() {
	// Load configuration
	config := loadConfig()

	// Create xApp manager
	manager := NewXAppManager(config)

	// Start xApp
	if err := manager.Start(); err != nil {
		log.Fatalf("Failed to start xApp: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := manager.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown xApp: %v", err)
	}
}
