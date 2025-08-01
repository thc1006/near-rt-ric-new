/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientManager manages gRPC clients for O-RAN SC components
type ClientManager struct {
	e2mgrConn      *grpc.ClientConn
	submgrConn     *grpc.ClientConn
	a1mediatorConn *grpc.ClientConn
	o1mediatorConn *grpc.ClientConn
	rtmgrConn      *grpc.ClientConn
	e2tConn        *grpc.ClientConn
	httpClient     *http.Client
	config         *Config
	
	// High-level client interfaces
	e2ManagerClient       *E2ManagerClient
	subscriptionMgrClient *SubscriptionManagerClient
	routingMgrClient      *RoutingManagerClient
	e2tClient             *E2TClient
	a1MediatorClient      *A1MediatorClient
	o1MediatorClient      *O1MediatorClient
	sctpManager           *SCTPConnectionManager
}

// NewClientManager creates a new client manager
func NewClientManager(config *Config) (*ClientManager, error) {
	cm := &ClientManager{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}

	// Initialize SCTP connection manager
	sctpConfig := &SCTPConfig{
		ListenAddress:     "0.0.0.0",
		ListenPort:        36422, // Standard E2 port
		MaxAssociations:   1000,
		HeartbeatInterval: 30 * time.Second,
		ConnectTimeout:    10 * time.Second,
		MaxRetries:        3,
		MaxInStreams:      10,
		MaxOutStreams:     10,
		BufferSize:        65536,
	}
	cm.sctpManager = NewSCTPConnectionManager(sctpConfig, &DefaultSCTPEventHandler{})

	// Initialize gRPC connections
	if err := cm.initializeConnections(); err != nil {
		return nil, err
	}

	// Initialize high-level clients
	cm.initializeHighLevelClients()

	return cm, nil
}

// StartSCTPManager starts the SCTP connection manager
func (cm *ClientManager) StartSCTPManager(ctx context.Context) error {
	if cm.sctpManager != nil {
		return cm.sctpManager.Start(ctx)
	}
	return nil
}

// initializeConnections establishes gRPC connections to O-RAN SC components
func (cm *ClientManager) initializeConnections() error {
	// Connect to E2 Manager
	if cm.config.E2MgrEndpoint != "" {
		e2mgrConn, err := cm.createGRPCConnection(cm.config.E2MgrEndpoint)
		if err != nil {
			log.Printf("Failed to connect to E2 Manager at %s: %v", cm.config.E2MgrEndpoint, err)
		} else {
			cm.e2mgrConn = e2mgrConn
			log.Printf("Connected to E2 Manager at %s", cm.config.E2MgrEndpoint)
		}
	}

	// Connect to Subscription Manager
	if cm.config.SubmgrEndpoint != "" {
		submgrConn, err := cm.createGRPCConnection(cm.config.SubmgrEndpoint)
		if err != nil {
			log.Printf("Failed to connect to Subscription Manager at %s: %v", cm.config.SubmgrEndpoint, err)
		} else {
			cm.submgrConn = submgrConn
			log.Printf("Connected to Subscription Manager at %s", cm.config.SubmgrEndpoint)
		}
	}

	// Connect to A1 Mediator (HTTP-based, no gRPC connection needed)
	if cm.config.A1MediatorEndpoint != "" {
		log.Printf("A1 Mediator endpoint configured at %s", cm.config.A1MediatorEndpoint)
	}

	// Connect to O1 Mediator (HTTP-based, no gRPC connection needed)
	if cm.config.O1MediatorEndpoint != "" {
		log.Printf("O1 Mediator endpoint configured at %s", cm.config.O1MediatorEndpoint)
	}

	// Connect to Routing Manager
	if cm.config.RtmgrEndpoint != "" {
		rtmgrConn, err := cm.createGRPCConnection(cm.config.RtmgrEndpoint)
		if err != nil {
			log.Printf("Failed to connect to Routing Manager at %s: %v", cm.config.RtmgrEndpoint, err)
		} else {
			cm.rtmgrConn = rtmgrConn
			log.Printf("Connected to Routing Manager at %s", cm.config.RtmgrEndpoint)
		}
	}

	// Connect to E2 Termination
	if cm.config.E2TermEndpoint != "" {
		e2tConn, err := cm.createGRPCConnection(cm.config.E2TermEndpoint)
		if err != nil {
			log.Printf("Failed to connect to E2 Termination at %s: %v", cm.config.E2TermEndpoint, err)
		} else {
			cm.e2tConn = e2tConn
			log.Printf("Connected to E2 Termination at %s", cm.config.E2TermEndpoint)
		}
	}

	return nil
}

// initializeHighLevelClients initializes high-level client interfaces
func (cm *ClientManager) initializeHighLevelClients() {
	// Initialize E2 Manager client
	cm.e2ManagerClient = NewE2ManagerClient(cm.e2mgrConn, cm.httpClient, cm.config.E2MgrEndpoint)
	
	// Initialize Subscription Manager client
	cm.subscriptionMgrClient = NewSubscriptionManagerClient(cm.submgrConn, cm.httpClient, cm.config.SubmgrEndpoint)
	
	// Initialize Routing Manager client
	cm.routingMgrClient = NewRoutingManagerClient(cm.rtmgrConn, cm.httpClient, cm.config.RtmgrEndpoint)
	
	// Initialize E2 Termination client
	cm.e2tClient = NewE2TClient(cm.e2tConn, cm.httpClient, cm.config.E2TermEndpoint, cm.sctpManager)
	
	// Initialize A1 Mediator client
	cm.a1MediatorClient = NewA1MediatorClient(cm.httpClient, cm.config.A1MediatorEndpoint)
	
	// Initialize O1 Mediator client
	cm.o1MediatorClient = NewO1MediatorClient(cm.httpClient, cm.config.O1MediatorEndpoint)
}

// createGRPCConnection creates a gRPC connection with appropriate credentials
func (cm *ClientManager) createGRPCConnection(endpoint string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try secure connection first, fallback to insecure
	opts := []grpc.DialOption{
		grpc.WithBlock(),
	}

	// For development, use insecure connections
	// In production, this should use proper TLS credentials
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", endpoint, err)
	}

	return conn, nil
}

// GetE2ManagerConnection returns the E2 Manager gRPC connection
func (cm *ClientManager) GetE2ManagerConnection() *grpc.ClientConn {
	return cm.e2mgrConn
}

// GetSubscriptionManagerConnection returns the Subscription Manager gRPC connection
func (cm *ClientManager) GetSubscriptionManagerConnection() *grpc.ClientConn {
	return cm.submgrConn
}

// GetHTTPClient returns the HTTP client for REST API calls
func (cm *ClientManager) GetHTTPClient() *http.Client {
	return cm.httpClient
}

// GetAppManagerEndpoint returns the App Manager endpoint
func (cm *ClientManager) GetAppManagerEndpoint() string {
	return cm.config.AppmgrEndpoint
}

// GetA1MediatorEndpoint returns the A1 Mediator endpoint
func (cm *ClientManager) GetA1MediatorEndpoint() string {
	return cm.config.A1MediatorEndpoint
}

// GetO1MediatorEndpoint returns the O1 Mediator endpoint
func (cm *ClientManager) GetO1MediatorEndpoint() string {
	return cm.config.O1MediatorEndpoint
}

// GetDbaasEndpoint returns the Database service endpoint
func (cm *ClientManager) GetDbaasEndpoint() string {
	return cm.config.DbaasEndpoint
}

// GetRtmgrConnection returns the Routing Manager gRPC connection
func (cm *ClientManager) GetRtmgrConnection() *grpc.ClientConn {
	return cm.rtmgrConn
}

// GetE2ManagerClient returns the E2 Manager client
func (cm *ClientManager) GetE2ManagerClient() *E2ManagerClient {
	return cm.e2ManagerClient
}

// GetSubscriptionManagerClient returns the Subscription Manager client
func (cm *ClientManager) GetSubscriptionManagerClient() *SubscriptionManagerClient {
	return cm.subscriptionMgrClient
}

// GetRoutingManagerClient returns the Routing Manager client
func (cm *ClientManager) GetRoutingManagerClient() *RoutingManagerClient {
	return cm.routingMgrClient
}

// GetE2TClient returns the E2 Termination client
func (cm *ClientManager) GetE2TClient() *E2TClient {
	return cm.e2tClient
}

// GetA1MediatorClient returns the A1 Mediator client
func (cm *ClientManager) GetA1MediatorClient() *A1MediatorClient {
	return cm.a1MediatorClient
}

// GetO1MediatorClient returns the O1 Mediator client
func (cm *ClientManager) GetO1MediatorClient() *O1MediatorClient {
	return cm.o1MediatorClient
}

// GetSCTPManager returns the SCTP connection manager
func (cm *ClientManager) GetSCTPManager() *SCTPConnectionManager {
	return cm.sctpManager
}

// IsE2ManagerConnected checks if E2 Manager connection is available
func (cm *ClientManager) IsE2ManagerConnected() bool {
	return cm.e2mgrConn != nil && cm.e2mgrConn.GetState().String() == "READY"
}

// IsSubscriptionManagerConnected checks if Subscription Manager connection is available
func (cm *ClientManager) IsSubscriptionManagerConnected() bool {
	return cm.submgrConn != nil && cm.submgrConn.GetState().String() == "READY"
}

// IsA1MediatorConnected checks if A1 Mediator is available via HTTP
func (cm *ClientManager) IsA1MediatorConnected() bool {
	if cm.config.A1MediatorEndpoint == "" {
		return false
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", cm.config.A1MediatorEndpoint+"/a1-p/healthcheck", nil)
	if err != nil {
		return false
	}
	
	resp, err := cm.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// IsO1MediatorConnected checks if O1 Mediator is available via HTTP
func (cm *ClientManager) IsO1MediatorConnected() bool {
	if cm.o1MediatorClient == nil {
		return false
	}
	return cm.o1MediatorClient.IsConnected()
}

// IsRtmgrConnected checks if Routing Manager connection is available
func (cm *ClientManager) IsRtmgrConnected() bool {
	return cm.rtmgrConn != nil && cm.rtmgrConn.GetState().String() == "READY"
}

// IsE2TConnected checks if E2 Termination connection is available
func (cm *ClientManager) IsE2TConnected() bool {
	return cm.e2tConn != nil && cm.e2tConn.GetState().String() == "READY"
}

// Reconnect attempts to reconnect to all components
func (cm *ClientManager) Reconnect() error {
	log.Println("Attempting to reconnect to O-RAN SC components")

	// Close existing connections
	if cm.e2mgrConn != nil {
		cm.e2mgrConn.Close()
		cm.e2mgrConn = nil
	}
	if cm.submgrConn != nil {
		cm.submgrConn.Close()
		cm.submgrConn = nil
	}
	if cm.a1mediatorConn != nil {
		cm.a1mediatorConn.Close()
		cm.a1mediatorConn = nil
	}
	if cm.o1mediatorConn != nil {
		cm.o1mediatorConn.Close()
		cm.o1mediatorConn = nil
	}
	if cm.rtmgrConn != nil {
		cm.rtmgrConn.Close()
		cm.rtmgrConn = nil
	}
	if cm.e2tConn != nil {
		cm.e2tConn.Close()
		cm.e2tConn = nil
	}

	// Stop SCTP manager
	if cm.sctpManager != nil {
		cm.sctpManager.Stop()
	}

	// Reinitialize connections
	if err := cm.initializeConnections(); err != nil {
		return err
	}
	
	// Reinitialize high-level clients
	cm.initializeHighLevelClients()
	return nil
}

// Close closes all gRPC connections
func (cm *ClientManager) Close() {
	if cm.e2mgrConn != nil {
		cm.e2mgrConn.Close()
		log.Println("Closed E2 Manager connection")
	}
	if cm.submgrConn != nil {
		cm.submgrConn.Close()
		log.Println("Closed Subscription Manager connection")
	}
	if cm.a1mediatorConn != nil {
		cm.a1mediatorConn.Close()
		log.Println("Closed A1 Mediator connection")
	}
	if cm.o1mediatorConn != nil {
		cm.o1mediatorConn.Close()
		log.Println("Closed O1 Mediator connection")
	}
	if cm.rtmgrConn != nil {
		cm.rtmgrConn.Close()
		log.Println("Closed Routing Manager connection")
	}
	if cm.e2tConn != nil {
		cm.e2tConn.Close()
		log.Println("Closed E2 Termination connection")
	}
	if cm.sctpManager != nil {
		cm.sctpManager.Stop()
		log.Println("Stopped SCTP connection manager")
	}
}
