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
	"runtime"
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
	config                  *Config
	httpServer              *http.Server
	clients                 *ClientManager
	discovery               *DiscoveryService
	wsHub                   *WebSocketHub
	serviceModelRegistry    *ServiceModelRegistry
	serviceModelAPIManager  *ServiceModelAPIManager
	policyManager           *PolicyManager
	authService             *AuthService
	rbacManager             *RBACManager
	auditLogger             *AuditLogger
	jwtManager              *JWTManager
	serviceAccountManager   *ServiceAccountManager
	authMiddleware          *AuthMiddleware
	authHandlers            *AuthHandlers
	serviceAccountHandlers  *ServiceAccountHandlers
	securityMonitor         *SecurityMonitor
	securityHandlers        *SecurityHandlers
	tlsManager              *TLSManager
	tlsHandlers             *TLSHandlers
	metricsManager          *MetricsManager
	tracingManager          *TracingManager
	logger                  *Logger
	performanceOptimizer    *PerformanceOptimizer
	loadBalancer            *LoadBalancer
	horizontalScaler        *HorizontalScaler
}

// NewServer creates a new dashboard server instance
func NewServer(config *Config) (*Server, error) {
	// Initialize observability components first
	logger := NewLogger("dashboard-server")
	
	// Initialize performance optimizer
	performanceOptimizer := NewPerformanceOptimizer()
	
	// Initialize metrics
	metricsManager := NewMetricsManager()
	
	// Initialize tracing
	tracingConfig := TracingConfig{
		ServiceName:     "oran-ric-dashboard",
		ServiceVersion:  "1.0.0",
		JaegerEndpoint:  "http://jaeger-collector:14268/api/traces",
		SamplingRate:    1.0, // 100% sampling for development
		Environment:     "development",
	}
	
	tracingManager, err := NewTracingManager(tracingConfig)
	if err != nil {
		logger.WithError(err).Warn("Failed to initialize tracing, continuing without tracing")
		tracingManager = nil
	}

	// Initialize TLS management
	tlsManager, err := NewTLSManager(nil) // Use default config
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS manager: %w", err)
	}

	// Initialize gRPC clients
	clients, err := NewClientManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client manager: %w", err)
	}
	
	// Set TLS config for client connections
	clientTLSConfig := tlsManager.GetClientTLSConfig()
	if clientTLSConfig != nil {
		clients.SetTLSConfig(clientTLSConfig)
	}

	// Initialize discovery service
	discovery := NewDiscoveryService(clients)

	// Initialize WebSocket hub
	wsHub := NewWebSocketHub()

	// Initialize service model registry
	serviceModelRegistry := NewServiceModelRegistry()

	// Initialize service model API manager
	serviceModelAPIManager := NewServiceModelAPIManager(serviceModelRegistry)

	// Initialize policy manager
	var policyManager *PolicyManager
	if a1Client := clients.GetA1MediatorClient(); a1Client != nil {
		policyManager = NewPolicyManager(a1Client)
	}

	// Initialize authentication system
	jwtManager, err := NewJWTManager("oran-ric-dashboard", 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT manager: %w", err)
	}

	rbacManager := NewRBACManager()
	
	auditLogger, err := NewAuditLogger("audit.log", 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	authService := NewAuthService(jwtManager, rbacManager, auditLogger)
	serviceAccountManager := NewServiceAccountManager(jwtManager, rbacManager, auditLogger)
	authMiddleware := NewAuthMiddleware(authService, auditLogger)
	authHandlers := NewAuthHandlers(authService, rbacManager, auditLogger)
	serviceAccountHandlers := NewServiceAccountHandlers(serviceAccountManager, auditLogger)
	
	// Initialize security monitoring
	securityMonitor := NewSecurityMonitor(auditLogger)
	securityHandlers := NewSecurityHandlers(securityMonitor, auditLogger)
	
	// TLS handlers (tlsManager already initialized above)
	tlsHandlers := NewTLSHandlers(tlsManager, auditLogger)

	server := &Server{
		config:                 config,
		clients:                clients,
		discovery:              discovery,
		wsHub:                  wsHub,
		serviceModelRegistry:   serviceModelRegistry,
		serviceModelAPIManager: serviceModelAPIManager,
		policyManager:          policyManager,
		authService:            authService,
		rbacManager:            rbacManager,
		auditLogger:            auditLogger,
		jwtManager:             jwtManager,
		serviceAccountManager:  serviceAccountManager,
		authMiddleware:         authMiddleware,
		authHandlers:           authHandlers,
		serviceAccountHandlers: serviceAccountHandlers,
		securityMonitor:        securityMonitor,
		securityHandlers:       securityHandlers,
		tlsManager:             tlsManager,
		tlsHandlers:            tlsHandlers,
		metricsManager:         metricsManager,
		tracingManager:         tracingManager,
		logger:                 logger,
		performanceOptimizer:   performanceOptimizer,
		loadBalancer:           NewLoadBalancer(RoundRobin),
		horizontalScaler:       NewHorizontalScaler(nil), // TODO: Add Kubernetes client
	}

	// Setup HTTP router with observability middleware
	router := server.setupRoutes()
	
	// Add metrics middleware
	handler := metricsManager.HTTPMiddleware(router)
	
	// Add tracing middleware if available
	if tracingManager != nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Extract trace context from headers
			headers := make(map[string]string)
			for k, v := range r.Header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}
			ctx = ExtractTraceContext(ctx, headers)
			
			// Add correlation ID
			correlationID := r.Header.Get("X-Correlation-ID")
			ctx = WithCorrelationID(ctx, correlationID)
			
			// Start HTTP span
			ctx, span := tracingManager.StartHTTPSpan(ctx, r.Method, r.URL.Path)
			defer span.End()
			
			// Update request context
			r = r.WithContext(ctx)
			
			handler.ServeHTTP(w, r)
		})
	}
	
	// Configure HTTPS server with TLS
	tlsConfig := tlsManager.GetServerTLSConfig()
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		TLSConfig:    tlsConfig,
	}

	return server, nil
}

// Start starts the dashboard server
func (s *Server) Start() error {
	ctx := context.Background()
	
	// Initialize and start metrics WebSocket hub
	InitializeMetricsWebSocket()
	
	// Start metrics collection goroutine
	go s.startMetricsCollection(ctx)
	
	// Start WebSocket hub
	go s.wsHub.Run()

	// Start discovery service
	go s.discovery.Start(s.wsHub)

	// Start SCTP connection manager
	if err := s.clients.StartSCTPManager(ctx); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to start SCTP manager", "error", err)
	}

	// Start security monitoring
	if err := s.securityMonitor.Start(ctx); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to start security monitor", "error", err)
	}

	// Start performance optimizer
	if err := s.performanceOptimizer.Start(ctx); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to start performance optimizer", "error", err)
	}

	// Start HTTPS server with TLS
	if s.httpServer.TLSConfig != nil {
		s.logger.InfoCtx(ctx, "Starting HTTPS server with TLS 1.3", "port", s.config.Port)
		return s.httpServer.ListenAndServeTLS("", "") // Certificates are in TLSConfig
	} else {
		s.logger.WarnCtx(ctx, "Starting HTTP server without TLS", "port", s.config.Port)
		return s.httpServer.ListenAndServe()
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoCtx(ctx, "Shutting down dashboard server")
	
	// Stop discovery service
	s.discovery.Stop()

	// Stop WebSocket hub
	s.wsHub.Stop()

	// Stop policy manager
	if s.policyManager != nil {
		s.policyManager.Stop()
	}

	// Stop security monitor
	if s.securityMonitor != nil {
		s.securityMonitor.Stop()
	}

	// Close audit logger
	if s.auditLogger != nil {
		s.auditLogger.Close()
	}

	// Close tracing
	if s.tracingManager != nil {
		if err := s.tracingManager.Close(ctx); err != nil {
			s.logger.ErrorCtx(ctx, "Failed to close tracing", "error", err)
		}
	}

	// Stop performance optimizer
	s.performanceOptimizer.Stop()

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

	// Authentication routes (no auth required)
	api.HandleFunc("/auth/login", s.authHandlers.LoginHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/refresh", s.authHandlers.RefreshTokenHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/service-account/token", s.serviceAccountHandlers.ServiceAccountTokenHandler).Methods("POST", "OPTIONS")

	// Protected authentication routes
	authAPI := api.PathPrefix("/auth").Subrouter()
	authAPI.Use(s.authMiddleware.RequireAuth)
	authAPI.HandleFunc("/logout", s.authHandlers.LogoutHandler).Methods("POST", "OPTIONS")
	authAPI.HandleFunc("/me", s.authHandlers.GetCurrentUserHandler).Methods("GET", "OPTIONS")
	authAPI.HandleFunc("/change-password", s.authHandlers.ChangePasswordHandler).Methods("POST", "OPTIONS")

	// User management routes (admin only)
	userAPI := api.PathPrefix("/users").Subrouter()
	userAPI.Use(s.authMiddleware.AdminOnly)
	userAPI.HandleFunc("", s.authHandlers.GetUsersHandler).Methods("GET", "OPTIONS")
	userAPI.HandleFunc("", s.authHandlers.CreateUserHandler).Methods("POST", "OPTIONS")
	userAPI.HandleFunc("/{userId}", s.authHandlers.UpdateUserHandler).Methods("PUT", "OPTIONS")
	userAPI.HandleFunc("/{userId}", s.authHandlers.DeleteUserHandler).Methods("DELETE", "OPTIONS")
	userAPI.HandleFunc("/{userId}/permissions", s.authHandlers.GetUserPermissionsHandler).Methods("GET", "OPTIONS")

	// Role management routes (admin only)
	roleAPI := api.PathPrefix("/roles").Subrouter()
	roleAPI.Use(s.authMiddleware.AdminOnly)
	roleAPI.HandleFunc("", s.authHandlers.GetRolesHandler).Methods("GET", "OPTIONS")
	roleAPI.HandleFunc("", s.authHandlers.CreateRoleHandler).Methods("POST", "OPTIONS")
	roleAPI.HandleFunc("/{roleId}", s.authHandlers.UpdateRoleHandler).Methods("PUT", "OPTIONS")
	roleAPI.HandleFunc("/{roleId}", s.authHandlers.DeleteRoleHandler).Methods("DELETE", "OPTIONS")

	// Permission routes (admin only)
	permAPI := api.PathPrefix("/permissions").Subrouter()
	permAPI.Use(s.authMiddleware.AdminOnly)
	permAPI.HandleFunc("", s.authHandlers.GetPermissionsHandler).Methods("GET", "OPTIONS")

	// Service account management routes (admin only)
	saAPI := api.PathPrefix("/service-accounts").Subrouter()
	saAPI.Use(s.authMiddleware.AdminOnly)
	saAPI.HandleFunc("", s.serviceAccountHandlers.GetServiceAccountsHandler).Methods("GET", "OPTIONS")
	saAPI.HandleFunc("", s.serviceAccountHandlers.CreateServiceAccountHandler).Methods("POST", "OPTIONS")
	saAPI.HandleFunc("/{serviceAccountId}", s.serviceAccountHandlers.GetServiceAccountHandler).Methods("GET", "OPTIONS")
	saAPI.HandleFunc("/{serviceAccountId}", s.serviceAccountHandlers.UpdateServiceAccountHandler).Methods("PUT", "OPTIONS")
	saAPI.HandleFunc("/{serviceAccountId}", s.serviceAccountHandlers.DeleteServiceAccountHandler).Methods("DELETE", "OPTIONS")
	saAPI.HandleFunc("/{serviceAccountId}/rotate-secret", s.serviceAccountHandlers.RotateServiceAccountSecretHandler).Methods("POST", "OPTIONS")
	saAPI.HandleFunc("/{serviceAccountId}/credentials", s.serviceAccountHandlers.GetServiceAccountCredentialsHandler).Methods("GET", "OPTIONS")

	// Audit routes (admin only)
	auditAPI := api.PathPrefix("/audit").Subrouter()
	auditAPI.Use(s.authMiddleware.AdminOnly)
	auditAPI.HandleFunc("/events", s.authHandlers.GetAuditEventsHandler).Methods("GET", "OPTIONS")
	auditAPI.HandleFunc("/stats", s.authHandlers.GetAuditStatsHandler).Methods("GET", "OPTIONS")

	// Security monitoring routes (admin only)
	securityAPI := api.PathPrefix("/security").Subrouter()
	securityAPI.Use(s.authMiddleware.AdminOnly)
	securityAPI.HandleFunc("/metrics", s.securityHandlers.GetSecurityMetricsHandler).Methods("GET", "OPTIONS")
	securityAPI.HandleFunc("/alerts", s.securityHandlers.GetSecurityAlertsHandler).Methods("GET", "OPTIONS")
	securityAPI.HandleFunc("/alerts/{alertId}", s.securityHandlers.GetSecurityAlertHandler).Methods("GET", "OPTIONS")
	securityAPI.HandleFunc("/alerts/{alertId}/acknowledge", s.securityHandlers.AcknowledgeSecurityAlertHandler).Methods("POST", "OPTIONS")
	securityAPI.HandleFunc("/alerts/{alertId}/resolve", s.securityHandlers.ResolveSecurityAlertHandler).Methods("POST", "OPTIONS")
	securityAPI.HandleFunc("/alerts/stats", s.securityHandlers.GetAlertStatsHandler).Methods("GET", "OPTIONS")
	securityAPI.HandleFunc("/compliance/status", s.securityHandlers.GetComplianceStatusHandler).Methods("GET", "OPTIONS")
	securityAPI.HandleFunc("/compliance/rules", s.securityHandlers.GetComplianceRulesHandler).Methods("GET", "OPTIONS")
	securityAPI.HandleFunc("/compliance/rules/{ruleId}", s.securityHandlers.UpdateComplianceRuleHandler).Methods("PUT", "OPTIONS")
	securityAPI.HandleFunc("/compliance/validate", s.securityHandlers.RunComplianceValidationHandler).Methods("POST", "OPTIONS")
	securityAPI.HandleFunc("/anomaly/patterns", s.securityHandlers.GetAnomalyPatternsHandler).Methods("GET", "OPTIONS")
	securityAPI.HandleFunc("/anomaly/patterns", s.securityHandlers.CreateAnomalyPatternHandler).Methods("POST", "OPTIONS")
	securityAPI.HandleFunc("/anomaly/patterns/{patternName}", s.securityHandlers.UpdateAnomalyPatternHandler).Methods("PUT", "OPTIONS")
	securityAPI.HandleFunc("/anomaly/patterns/{patternName}/enable", s.securityHandlers.EnableAnomalyPatternHandler).Methods("POST", "OPTIONS")
	securityAPI.HandleFunc("/anomaly/patterns/{patternName}/disable", s.securityHandlers.DisableAnomalyPatternHandler).Methods("POST", "OPTIONS")
	securityAPI.HandleFunc("/anomaly/stats", s.securityHandlers.GetPatternStatsHandler).Methods("GET", "OPTIONS")

	// TLS and certificate management routes (admin only)
	tlsAPI := api.PathPrefix("/tls").Subrouter()
	tlsAPI.Use(s.authMiddleware.AdminOnly)
	tlsAPI.HandleFunc("/certificates", s.tlsHandlers.GetAllCertificatesHandler).Methods("GET", "OPTIONS")
	tlsAPI.HandleFunc("/certificates/{certType}", s.tlsHandlers.GetCertificateInfoHandler).Methods("GET", "OPTIONS")
	tlsAPI.HandleFunc("/certificates/{certType}/regenerate", s.tlsHandlers.RegenerateCertificateHandler).Methods("POST", "OPTIONS")
	tlsAPI.HandleFunc("/certificates/rotate", s.tlsHandlers.RotateCertificatesHandler).Methods("POST", "OPTIONS")
	tlsAPI.HandleFunc("/certificates/expiry", s.tlsHandlers.CheckCertificateExpiryHandler).Methods("GET", "OPTIONS")
	tlsAPI.HandleFunc("/certificates/stats", s.tlsHandlers.GetCertificateStatsHandler).Methods("GET", "OPTIONS")
	tlsAPI.HandleFunc("/config", s.tlsHandlers.GetTLSConfigHandler).Methods("GET", "OPTIONS")
	tlsAPI.HandleFunc("/validate", s.tlsHandlers.ValidateTLSConfigHandler).Methods("GET", "OPTIONS")

	// Protected API endpoints (require authentication)
	protectedAPI := api.PathPrefix("").Subrouter()
	protectedAPI.Use(s.authMiddleware.RequireAuth)

	// Component discovery endpoints
	protectedAPI.HandleFunc("/components", s.handleGetComponents).Methods("GET", "OPTIONS")
	protectedAPI.HandleFunc("/components/{id}", s.handleGetComponent).Methods("GET", "OPTIONS")

	// E2 Manager endpoints (require operator or admin role)
	e2API := protectedAPI.PathPrefix("/e2nodes").Subrouter()
	e2API.Use(s.authMiddleware.OperatorOrAdmin)
	e2API.HandleFunc("", s.E2NodesHandler).Methods("GET", "OPTIONS")
	e2API.HandleFunc("/{nodeId}", s.E2NodeHandler).Methods("GET", "OPTIONS")
	e2API.HandleFunc("/{nodeId}/health", s.E2NodeHealthHandler).Methods("GET", "OPTIONS")
	e2API.HandleFunc("/{nodeId}/configuration", s.E2NodeConfigurationHandler).Methods("PUT", "OPTIONS")

	// Subscription Manager endpoints (require operator or admin role)
	subAPI := protectedAPI.PathPrefix("/subscriptions").Subrouter()
	subAPI.Use(s.authMiddleware.OperatorOrAdmin)
	subAPI.HandleFunc("", s.SubscriptionsHandler).Methods("GET", "POST", "OPTIONS")
	subAPI.HandleFunc("/{subscriptionId}", s.SubscriptionHandler).Methods("GET", "PUT", "DELETE", "OPTIONS")
	subAPI.HandleFunc("/{subscriptionId}/indications", s.SubscriptionIndicationsHandler).Methods("GET", "OPTIONS")

	// Service Model endpoints (require operator or admin role)
	smAPI := protectedAPI.PathPrefix("/servicemodels").Subrouter()
	smAPI.Use(s.authMiddleware.OperatorOrAdmin)
	smAPI.HandleFunc("", s.ServiceModelHandler).Methods("GET", "OPTIONS")
	smAPI.HandleFunc("/{oid}", s.ServiceModelByOIDHandler).Methods("GET", "OPTIONS")
	smAPI.HandleFunc("/capabilities", s.ServiceModelCapabilitiesHandler).Methods("GET", "OPTIONS")
	smAPI.HandleFunc("/stats", s.ServiceModelStatsHandler).Methods("GET", "OPTIONS")
	smAPI.HandleFunc("/process/indication", s.ProcessIndicationHandler).Methods("POST", "OPTIONS")
	smAPI.HandleFunc("/process/control", s.ProcessControlHandler).Methods("POST", "OPTIONS")
	
	// Enhanced Service Model API endpoints
	smAPI.HandleFunc("/operations", s.ServiceModelOperationsHandler).Methods("GET", "OPTIONS")
	smAPI.HandleFunc("/{type}/schema", s.ServiceModelSchemaHandler).Methods("GET", "OPTIONS")
	smAPI.HandleFunc("/validate", s.ValidateMessageHandler).Methods("POST", "OPTIONS")
	smAPI.HandleFunc("/kpi/definitions", s.KPIDefinitionsHandler).Methods("GET", "OPTIONS")
	smAPI.HandleFunc("/control/definitions", s.ControlActionDefinitionsHandler).Methods("GET", "OPTIONS")
	smAPI.HandleFunc("/interface/definitions", s.InterfaceDefinitionsHandler).Methods("GET", "OPTIONS")

	// SCTP Connection endpoints (require operator or admin role)
	sctpAPI := protectedAPI.PathPrefix("/sctp").Subrouter()
	sctpAPI.Use(s.authMiddleware.OperatorOrAdmin)
	sctpAPI.HandleFunc("/connections", s.SCTPConnectionsHandler).Methods("GET", "OPTIONS")
	sctpAPI.HandleFunc("/connections/{associationId}", s.SCTPConnectionHandler).Methods("GET", "DELETE", "OPTIONS")
	sctpAPI.HandleFunc("/stats", s.SCTPStatsHandler).Methods("GET", "OPTIONS")

	// E2 Termination endpoints (require operator or admin role)
	e2tAPI := protectedAPI.PathPrefix("/e2t").Subrouter()
	e2tAPI.Use(s.authMiddleware.OperatorOrAdmin)
	e2tAPI.HandleFunc("/stats", s.E2TStatsHandler).Methods("GET", "OPTIONS")
	e2tAPI.HandleFunc("/messages", s.E2APMessagesHandler).Methods("GET", "OPTIONS")

	// A1 Mediator endpoints (require policy-manager role or admin)
	a1API := protectedAPI.PathPrefix("/a1").Subrouter()
	a1API.Use(s.authMiddleware.RequirePermission("policies", "read"))
	a1API.HandleFunc("/health", s.A1HealthHandler).Methods("GET", "OPTIONS")
	a1API.HandleFunc("/stats", s.A1StatsHandler).Methods("GET", "OPTIONS")
	a1API.HandleFunc("/policytypes", s.A1PolicyTypesHandler).Methods("GET", "OPTIONS")
	a1API.HandleFunc("/policytypes/{policyTypeId}", s.A1PolicyTypeHandler).Methods("GET", "OPTIONS")
	a1API.HandleFunc("/policytypes/{policyTypeId}/policies", s.A1PolicyInstancesHandler).Methods("GET", "OPTIONS")
	a1API.HandleFunc("/policytypes/{policyTypeId}/policies/{policyInstanceId}", s.EnhancedA1PolicyInstanceHandler).Methods("GET", "OPTIONS")
	a1API.HandleFunc("/policytypes/{policyTypeId}/policies/{policyInstanceId}/status", s.A1PolicyInstanceStatusHandler).Methods("GET", "OPTIONS")
	
	// A1 write operations (require policy write permission)
	a1WriteAPI := protectedAPI.PathPrefix("/a1").Subrouter()
	a1WriteAPI.Use(s.authMiddleware.RequirePermission("policies", "write"))
	a1WriteAPI.HandleFunc("/policytypes/{policyTypeId}", s.A1PolicyTypeHandler).Methods("POST", "DELETE", "OPTIONS")
	a1WriteAPI.HandleFunc("/policytypes/{policyTypeId}/policies/{policyInstanceId}", s.EnhancedA1PolicyInstanceHandler).Methods("PUT", "DELETE", "OPTIONS")
	a1WriteAPI.HandleFunc("/policytypes/{policyTypeId}/validate", s.PolicyValidationHandler).Methods("POST", "OPTIONS")
	a1WriteAPI.HandleFunc("/policytypes/{policyTypeId}/validate-schema", s.PolicyTypeValidationHandler).Methods("POST", "OPTIONS")

	// Policy Management Framework endpoints (require policy read permission)
	policyAPI := protectedAPI.PathPrefix("/policies").Subrouter()
	policyAPI.Use(s.authMiddleware.RequirePermission("policies", "read"))
	policyAPI.HandleFunc("/conflicts", s.PolicyConflictsHandler).Methods("GET", "OPTIONS")
	policyAPI.HandleFunc("/{policyInstanceId}/distribution", s.PolicyDistributionStatusHandler).Methods("GET", "OPTIONS")
	policyAPI.HandleFunc("/{policyInstanceId}/compliance", s.PolicyComplianceReportsHandler).Methods("GET", "OPTIONS")
	
	// Policy write operations
	policyWriteAPI := protectedAPI.PathPrefix("/policies").Subrouter()
	policyWriteAPI.Use(s.authMiddleware.RequirePermission("policies", "write"))
	policyWriteAPI.HandleFunc("/conflicts/{conflictId}/resolve", s.PolicyConflictResolutionHandler).Methods("POST", "OPTIONS")
	
	// xApp Management endpoints for policy distribution (require xapp read permission)
	xappRegAPI := protectedAPI.PathPrefix("/xapps/registration").Subrouter()
	xappRegAPI.Use(s.authMiddleware.RequirePermission("xapps", "read"))
	xappRegAPI.HandleFunc("", s.XAppRegistrationHandler).Methods("GET", "OPTIONS")
	
	xappRegWriteAPI := protectedAPI.PathPrefix("/xapps/registration").Subrouter()
	xappRegWriteAPI.Use(s.authMiddleware.RequirePermission("xapps", "write"))
	xappRegWriteAPI.HandleFunc("", s.XAppRegistrationHandler).Methods("POST", "OPTIONS")
	xappRegWriteAPI.HandleFunc("/{xappId}", s.XAppUnregistrationHandler).Methods("DELETE", "OPTIONS")

	// App Manager endpoints (require xapp permissions)
	xappAPI := protectedAPI.PathPrefix("/xapps").Subrouter()
	xappAPI.Use(s.authMiddleware.RequirePermission("xapps", "read"))
	xappAPI.HandleFunc("", s.handleGetXApps).Methods("GET", "OPTIONS")
	xappAPI.HandleFunc("/{name}", s.handleGetXApp).Methods("GET", "OPTIONS")
	
	// xApp deployment operations (require xapp write permission)
	xappWriteAPI := protectedAPI.PathPrefix("/xapps").Subrouter()
	xappWriteAPI.Use(s.authMiddleware.RequirePermission("xapps", "deploy"))
	xappWriteAPI.HandleFunc("", s.handleDeployXApp).Methods("POST", "OPTIONS")
	xappWriteAPI.HandleFunc("/{name}", s.handleUndeployXApp).Methods("DELETE", "OPTIONS")

	// O1 Mediator endpoints (require o1 read permission)
	o1API := protectedAPI.PathPrefix("/o1").Subrouter()
	o1API.Use(s.authMiddleware.RequirePermission("o1", "read"))
	o1API.HandleFunc("/health", s.O1HealthHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/stats", s.O1StatsHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/managed-objects", s.O1ManagedObjectsHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/managed-objects/{objectId}", s.O1ManagedObjectHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/configurations", s.O1ConfigurationsHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/alarms", s.O1AlarmsHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/kpis", s.O1KPIsHandler).Methods("GET", "OPTIONS")
	
	// O1 write operations (require o1 write permission)
	o1WriteAPI := protectedAPI.PathPrefix("/o1").Subrouter()
	o1WriteAPI.Use(s.authMiddleware.RequirePermission("o1", "write"))
	o1WriteAPI.HandleFunc("/configurations/{configId}", s.O1ConfigurationHandler).Methods("POST", "PUT", "OPTIONS")
	o1WriteAPI.HandleFunc("/alarms/{alarmId}/acknowledge", s.O1AlarmHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/backup", s.O1BackupHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/restore", s.O1RestoreHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/validate", s.O1ValidationHandler).Methods("POST", "OPTIONS")

	// O1 Management Operations endpoints (continue with o1 permissions)
	o1API.HandleFunc("/backups", s.O1BackupsHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/certificates", s.O1CertificatesHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/resource-usage", s.O1ResourceUsageHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/access-control/policies", s.O1AccessControlHandler).Methods("GET", "OPTIONS")
	o1API.HandleFunc("/access-control/policies/{policyId}", s.O1AccessControlPolicyHandler).Methods("GET", "OPTIONS")
	
	// O1 write operations (continue)
	o1WriteAPI.HandleFunc("/backups/{backupId}", s.O1BackupHandler).Methods("DELETE", "OPTIONS")
	o1WriteAPI.HandleFunc("/alarms/generate", s.O1AlarmGenerationHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/alarms/{alarmId}/clear", s.O1AlarmClearHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/alarms/correlate", s.O1AlarmCorrelationHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/kpis/manage", s.O1KPIManagementHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/kpis/{kpiId}", s.O1KPIHandler).Methods("PUT", "OPTIONS")
	o1WriteAPI.HandleFunc("/kpis/collect", s.O1KPICollectionHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/certificates", s.O1CertificatesHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/certificates/{certId}/revoke", s.O1CertificateHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/resource-usage", s.O1ResourceUsageHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/access-control/policies", s.O1AccessControlHandler).Methods("POST", "OPTIONS")
	o1WriteAPI.HandleFunc("/access-control/policies/{policyId}", s.O1AccessControlPolicyHandler).Methods("PUT", "DELETE", "OPTIONS")

	// WebSocket endpoints
	router.HandleFunc("/ws", s.handleWebSocket)
	router.HandleFunc("/ws/metrics", s.handleMetricsWebSocket)

	// Health check endpoint
	router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")
	
	// Metrics endpoint
	router.Handle("/metrics", s.metricsManager.Handler()).Methods("GET")
	
	// Performance monitoring endpoints
	router.HandleFunc("/api/v1/performance/metrics", s.handlePerformanceMetrics).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/performance/bottlenecks", s.handleBottleneckAlerts).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/performance/profiles", s.handlePerformanceProfiles).Methods("GET", "OPTIONS")

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

// startMetricsCollection starts periodic metrics collection
func (s *Server) startMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectMetrics(ctx)
		}
	}
}

// collectMetrics collects and updates various platform metrics
func (s *Server) collectMetrics(ctx context.Context) {
	// Collect system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	s.metricsManager.SetMemoryUsage("dashboard", "alloc", int64(m.Alloc))
	s.metricsManager.SetMemoryUsage("dashboard", "sys", int64(m.Sys))
	s.metricsManager.SetMemoryUsage("dashboard", "heap_alloc", int64(m.HeapAlloc))
	s.metricsManager.SetMemoryUsage("dashboard", "heap_sys", int64(m.HeapSys))
	s.metricsManager.SetGoroutinesCount(runtime.NumGoroutine())

	// Collect component health metrics
	componentStatus := s.discovery.GetComponentStatus()
	for component, status := range componentStatus {
		healthy := status == "healthy"
		s.metricsManager.SetComponentHealth(component, healthy)
	}

	// Collect E2 metrics if E2 manager client is available
	if e2mgrClient := s.clients.GetE2ManagerClient(); e2mgrClient != nil {
		// This would typically call the E2 manager to get node count
		// For now, we'll use a placeholder
		s.metricsManager.SetE2NodesConnected(0) // Would be actual count from E2M
	}

	// Collect subscription metrics if subscription manager client is available
	if submgrClient := s.clients.GetSubscriptionManagerClient(); submgrClient != nil {
		// This would typically call the subscription manager to get active subscriptions
		// For now, we'll use a placeholder
		s.metricsManager.SetE2SubscriptionsActive(0) // Would be actual count from SubMgr
	}

	// Collect A1 policy metrics if policy manager is available
	if s.policyManager != nil {
		// This would typically get policy counts from the policy manager
		// For now, we'll use placeholders
		s.metricsManager.SetA1PolicyTypesTotal(0)     // Would be actual count
		s.metricsManager.SetA1PolicyInstancesTotal(0) // Would be actual count
	}

	// Log metrics collection
	s.logger.DebugCtx(ctx, "Collected platform metrics",
		"goroutines", runtime.NumGoroutine(),
		"memory_alloc", m.Alloc,
		"memory_sys", m.Sys,
	)
}// h
andleMetricsWebSocket handles metrics WebSocket connections
func (s *Server) handleMetricsWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.ErrorCtx(r.Context(), "Failed to upgrade metrics WebSocket connection", "error", err)
		return
	}

	if GlobalMetricsHub == nil {
		s.logger.ErrorCtx(r.Context(), "Metrics WebSocket hub not initialized")
		conn.Close()
		return
	}

	client := NewMetricsWebSocketClient(GlobalMetricsHub, conn)
	GlobalMetricsHub.register <- client

	// Start goroutines for handling the connection
	go client.writePump()
	go client.readPump()
}
// ha
ndlePerformanceMetrics handles requests for performance metrics
func (s *Server) handlePerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := s.performanceOptimizer.GetMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Failed to encode performance metrics: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleBottleneckAlerts handles requests for bottleneck alerts
func (s *Server) handleBottleneckAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := make([]BottleneckAlert, 0)
	
	// Collect all available alerts
	alertChan := s.performanceOptimizer.bottleneckDetector.GetAlerts()
	for {
		select {
		case alert := <-alertChan:
			alerts = append(alerts, alert)
		default:
			// No more alerts available
			goto done
		}
	}
	
done:
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(alerts); err != nil {
		log.Printf("Failed to encode bottleneck alerts: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handlePerformanceProfiles handles requests for performance profiles
func (s *Server) handlePerformanceProfiles(w http.ResponseWriter, r *http.Request) {
	profiles := s.performanceOptimizer.profiler.GetProfiles()
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profiles); err != nil {
		log.Printf("Failed to encode performance profiles: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}