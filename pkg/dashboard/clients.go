/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientManager manages gRPC clients for O-RAN SC components
type ClientManager struct {
	e2mgrConn  *grpc.ClientConn
	submgrConn *grpc.ClientConn
	httpClient *http.Client
	config     *Config
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

	// Initialize gRPC connections
	if err := cm.initializeConnections(); err != nil {
		return nil, err
	}

	return cm, nil
}

// initializeConnections establishes gRPC connections to O-RAN SC components
func (cm *ClientManager) initializeConnections() error {
	// Connect to E2 Manager
	e2mgrConn, err := cm.createGRPCConnection(cm.config.E2MgrEndpoint)
	if err != nil {
		log.Warnf("Failed to connect to E2 Manager at %s: %v", cm.config.E2MgrEndpoint, err)
		// Don't fail completely, component might not be deployed yet
	} else {
		cm.e2mgrConn = e2mgrConn
		log.Infof("Connected to E2 Manager at %s", cm.config.E2MgrEndpoint)
	}

	// Connect to Subscription Manager
	submgrConn, err := cm.createGRPCConnection(cm.config.SubmgrEndpoint)
	if err != nil {
		log.Warnf("Failed to connect to Subscription Manager at %s: %v", cm.config.SubmgrEndpoint, err)
		// Don't fail completely, component might not be deployed yet
	} else {
		cm.submgrConn = submgrConn
		log.Infof("Connected to Subscription Manager at %s", cm.config.SubmgrEndpoint)
	}

	return nil
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

// IsE2ManagerConnected checks if E2 Manager connection is available
func (cm *ClientManager) IsE2ManagerConnected() bool {
	return cm.e2mgrConn != nil && cm.e2mgrConn.GetState().String() == "READY"
}

// IsSubscriptionManagerConnected checks if Subscription Manager connection is available
func (cm *ClientManager) IsSubscriptionManagerConnected() bool {
	return cm.submgrConn != nil && cm.submgrConn.GetState().String() == "READY"
}

// Reconnect attempts to reconnect to all components
func (cm *ClientManager) Reconnect() error {
	log.Info("Attempting to reconnect to O-RAN SC components")

	// Close existing connections
	if cm.e2mgrConn != nil {
		cm.e2mgrConn.Close()
		cm.e2mgrConn = nil
	}
	if cm.submgrConn != nil {
		cm.submgrConn.Close()
		cm.submgrConn = nil
	}

	// Reinitialize connections
	return cm.initializeConnections()
}

// Close closes all gRPC connections
func (cm *ClientManager) Close() {
	if cm.e2mgrConn != nil {
		cm.e2mgrConn.Close()
		log.Info("Closed E2 Manager connection")
	}
	if cm.submgrConn != nil {
		cm.submgrConn.Close()
		log.Info("Closed Subscription Manager connection")
	}
}
