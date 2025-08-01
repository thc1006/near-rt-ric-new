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
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Config holds the configuration for the dashboard server
type Config struct {
	Port             int
	E2MgrEndpoint    string
	E2TermEndpoint   string
	SubmgrEndpoint   string
	AppmgrEndpoint   string
	A1MediatorEndpoint string
	O1MediatorEndpoint string
	DbaasEndpoint    string
	RtmgrEndpoint    string
}

// Server represents the dashboard API gateway server
type Server struct {
	config               *Config
	httpServer           *http.Server
	clients              *ClientManager
	discovery            *DiscoveryService
	wsHub                *WebSocketHub
	serviceModelRegistry *ServiceModelRegistry
	policyManager        *PolicyManager
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

	// Initialize service model registry
	serviceModelRegistry := NewServiceModelRegistry()

	// Initialize policy manager
	var policyManager *PolicyManager
	if a1Client := clients.GetA1MediatorClient(); a1Client != nil {
		policyManager = NewPolicyManager(a1Client)
	}

	server := &Server{
		config:               config,
		clients:              clients,
		discovery:            discovery,
		wsHub:                wsHub,
		serviceModelRegistry: serviceModelRegistry,
		policyManager:        policyManager,
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

	// Start SCTP connection manager
	ctx := context.Background()
	if err := s.clients.StartSCTPManager(ctx); err != nil {
		log.Printf("Failed to start SCTP manager: %v", err)
	}

	// Start HTTP server
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop discovery service
	s.discovery.Stop()

	// Stop WebSocket hub
	s.wsHub.Stop()

	// Stop policy manager
	if s.policyManager != nil {
		s.policyManager.Stop()
	}

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
	api.HandleFunc("/e2nodes", s.E2NodesHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/e2nodes/{nodeId}", s.E2NodeHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/e2nodes/{nodeId}/health", s.E2NodeHealthHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/e2nodes/{nodeId}/configuration", s.E2NodeConfigurationHandler).Methods("PUT", "OPTIONS")

	// Subscription Manager endpoints
	api.HandleFunc("/subscriptions", s.SubscriptionsHandler).Methods("GET", "POST", "OPTIONS")
	api.HandleFunc("/subscriptions/{subscriptionId}", s.SubscriptionHandler).Methods("GET", "PUT", "DELETE", "OPTIONS")
	api.HandleFunc("/subscriptions/{subscriptionId}/indications", s.SubscriptionIndicationsHandler).Methods("GET", "OPTIONS")

	// Service Model endpoints
	api.HandleFunc("/servicemodels", s.ServiceModelHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/servicemodels/{oid}", s.ServiceModelByOIDHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/servicemodels/capabilities", s.ServiceModelCapabilitiesHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/servicemodels/stats", s.ServiceModelStatsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/servicemodels/process/indication", s.ProcessIndicationHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/servicemodels/process/control", s.ProcessControlHandler).Methods("POST", "OPTIONS")

	// SCTP Connection endpoints
	api.HandleFunc("/sctp/connections", s.SCTPConnectionsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/sctp/connections/{associationId}", s.SCTPConnectionHandler).Methods("GET", "DELETE", "OPTIONS")
	api.HandleFunc("/sctp/stats", s.SCTPStatsHandler).Methods("GET", "OPTIONS")

	// E2 Termination endpoints
	api.HandleFunc("/e2t/stats", s.E2TStatsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/e2t/messages", s.E2APMessagesHandler).Methods("GET", "OPTIONS")

	// A1 Mediator endpoints
	api.HandleFunc("/a1/health", s.A1HealthHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/a1/stats", s.A1StatsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/a1/policytypes", s.A1PolicyTypesHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/a1/policytypes/{policyTypeId}", s.A1PolicyTypeHandler).Methods("GET", "POST", "DELETE", "OPTIONS")
	api.HandleFunc("/a1/policytypes/{policyTypeId}/policies", s.A1PolicyInstancesHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/a1/policytypes/{policyTypeId}/policies/{policyInstanceId}", s.EnhancedA1PolicyInstanceHandler).Methods("GET", "PUT", "DELETE", "OPTIONS")
	api.HandleFunc("/a1/policytypes/{policyTypeId}/policies/{policyInstanceId}/status", s.A1PolicyInstanceStatusHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/a1/policytypes/{policyTypeId}/validate", s.PolicyValidationHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/a1/policytypes/{policyTypeId}/validate-schema", s.PolicyTypeValidationHandler).Methods("POST", "OPTIONS")

	// Policy Management Framework endpoints
	api.HandleFunc("/policies/conflicts", s.PolicyConflictsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/policies/conflicts/{conflictId}/resolve", s.PolicyConflictResolutionHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/policies/{policyInstanceId}/distribution", s.PolicyDistributionStatusHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/policies/{policyInstanceId}/compliance", s.PolicyComplianceReportsHandler).Methods("GET", "OPTIONS")
	
	// xApp Management endpoints for policy distribution
	api.HandleFunc("/xapps/registration", s.XAppRegistrationHandler).Methods("GET", "POST", "OPTIONS")
	api.HandleFunc("/xapps/registration/{xappId}", s.XAppUnregistrationHandler).Methods("DELETE", "OPTIONS")

	// App Manager endpoints
	api.HandleFunc("/xapps", s.handleGetXApps).Methods("GET", "OPTIONS")
	api.HandleFunc("/xapps", s.handleDeployXApp).Methods("POST", "OPTIONS")
	api.HandleFunc("/xapps/{name}", s.handleGetXApp).Methods("GET", "OPTIONS")
	api.HandleFunc("/xapps/{name}", s.handleUndeployXApp).Methods("DELETE", "OPTIONS")

	// O1 Mediator endpoints
	api.HandleFunc("/o1/health", s.O1HealthHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/o1/stats", s.O1StatsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/o1/managed-objects", s.O1ManagedObjectsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/o1/managed-objects/{objectId}", s.O1ManagedObjectHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/o1/configurations", s.O1ConfigurationsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/o1/configurations/{configId}", s.O1ConfigurationHandler).Methods("POST", "PUT", "OPTIONS")
	api.HandleFunc("/o1/alarms", s.O1AlarmsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/o1/alarms/{alarmId}/acknowledge", s.O1AlarmHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/kpis", s.O1KPIsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/o1/backup", s.O1BackupHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/restore", s.O1RestoreHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/validate", s.O1ValidationHandler).Methods("POST", "OPTIONS")

	// O1 Management Operations endpoints
	api.HandleFunc("/o1/backups", s.O1BackupsHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/o1/backups/{backupId}", s.O1BackupHandler).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/o1/alarms/generate", s.O1AlarmGenerationHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/alarms/{alarmId}/clear", s.O1AlarmClearHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/alarms/correlate", s.O1AlarmCorrelationHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/kpis/manage", s.O1KPIManagementHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/kpis/{kpiId}", s.O1KPIHandler).Methods("PUT", "OPTIONS")
	api.HandleFunc("/o1/kpis/collect", s.O1KPICollectionHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/certificates", s.O1CertificatesHandler).Methods("GET", "POST", "OPTIONS")
	api.HandleFunc("/o1/certificates/{certId}/revoke", s.O1CertificateHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/o1/resource-usage", s.O1ResourceUsageHandler).Methods("GET", "POST", "OPTIONS")
	api.HandleFunc("/o1/access-control/policies", s.O1AccessControlHandler).Methods("GET", "POST", "OPTIONS")
	api.HandleFunc("/o1/access-control/policies/{policyId}", s.O1AccessControlPolicyHandler).Methods("GET", "PUT", "DELETE", "OPTIONS")

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
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
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
