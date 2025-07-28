/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("dashboard-server")

// Config holds the configuration for the dashboard server
type Config struct {
	Port           int
	E2MgrEndpoint  string
	SubmgrEndpoint string
	AppmgrEndpoint string
}

// Server represents the dashboard API gateway server
type Server struct {
	config     *Config
	httpServer *http.Server
	clients    *ClientManager
	discovery  *DiscoveryService
	wsHub      *WebSocketHub
}

// NewServer creates a new dashboard server instance
func NewServer(config *Config) (*Server, error) {
	// Initialize gRPC clients
	clients, err := NewClientManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client manager: %w", err)
	}

	// Initialize discovery service
	discovery := NewDiscoveryService(clients)

	// Initialize WebSocket hub
	wsHub := NewWebSocketHub()

	server := &Server{
		config:    config,
		clients:   clients,
		discovery: discovery,
		wsHub:     wsHub,
	}

	// Setup HTTP router
	router := server.setupRoutes()
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return server, nil
}

// Start starts the dashboard server
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Start discovery service
	go s.discovery.Start(s.wsHub)

	// Start HTTP server
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop discovery service
	s.discovery.Stop()

	// Stop WebSocket hub
	s.wsHub.Stop()

	// Close gRPC clients
	s.clients.Close()

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Enable CORS
	router.Use(s.corsMiddleware)

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Component discovery endpoints
	api.HandleFunc("/components", s.handleGetComponents).Methods("GET", "OPTIONS")
	api.HandleFunc("/components/{id}", s.handleGetComponent).Methods("GET", "OPTIONS")

	// E2 Manager endpoints
	api.HandleFunc("/e2nodes", s.handleGetE2Nodes).Methods("GET", "OPTIONS")
	api.HandleFunc("/e2nodes/{id}", s.handleGetE2Node).Methods("GET", "OPTIONS")

	// Subscription Manager endpoints
	api.HandleFunc("/subscriptions", s.handleGetSubscriptions).Methods("GET", "OPTIONS")
	api.HandleFunc("/subscriptions", s.handleCreateSubscription).Methods("POST", "OPTIONS")
	api.HandleFunc("/subscriptions/{id}", s.handleGetSubscription).Methods("GET", "OPTIONS")
	api.HandleFunc("/subscriptions/{id}", s.handleDeleteSubscription).Methods("DELETE", "OPTIONS")

	// App Manager endpoints
	api.HandleFunc("/xapps", s.handleGetXApps).Methods("GET", "OPTIONS")
	api.HandleFunc("/xapps", s.handleDeployXApp).Methods("POST", "OPTIONS")
	api.HandleFunc("/xapps/{name}", s.handleGetXApp).Methods("GET", "OPTIONS")
	api.HandleFunc("/xapps/{name}", s.handleUndeployXApp).Methods("DELETE", "OPTIONS")

	// WebSocket endpoint
	router.HandleFunc("/ws", s.handleWebSocket)

	// Health check endpoint
	router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")

	return router
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now().UTC(),
		"components": s.discovery.GetComponentStatus(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	client := &WebSocketClient{
		hub:  s.wsHub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.wsHub.register <- client

	// Start goroutines for handling the connection
	go client.writePump()
	go client.readPump()
}
