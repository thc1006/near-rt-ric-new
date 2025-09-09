/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"net/http"
	"testing"
	"time"
)

// createTestServer creates a server for testing without gRPC connections
func createTestServer(t *testing.T) *Server {
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	// Create a minimal server without gRPC connections for testing
	server := &Server{
		config: config,
		clients: &ClientManager{
			config:     config,
			httpClient: &http.Client{},
		},
		wsHub: NewWebSocketHub(),
	}

	// Initialize discovery service
	server.discovery = NewDiscoveryService(server.clients)

	// Manually populate test components without triggering network calls
	server.discovery.components = map[string]*Component{
		"e2manager": {
			ID:          "e2manager",
			Name:        "E2 Manager",
			Type:        ComponentTypeE2Manager,
			Status:      ComponentStatusError,
			Version:     "1.0.0",
			Endpoint:    config.E2MgrEndpoint,
			Metrics:     make(map[string]interface{}),
			LastUpdated: time.Now(),
		},
		"submgr": {
			ID:          "submgr",
			Name:        "Subscription Manager",
			Type:        ComponentTypeSubscriptionMgr,
			Status:      ComponentStatusError,
			Version:     "1.0.0",
			Endpoint:    config.SubmgrEndpoint,
			Metrics:     make(map[string]interface{}),
			LastUpdated: time.Now(),
		},
		"appmgr": {
			ID:          "appmgr",
			Name:        "App Manager",
			Type:        ComponentTypeAppManager,
			Status:      ComponentStatusError,
			Version:     "1.0.0",
			Endpoint:    config.AppmgrEndpoint,
			Metrics:     make(map[string]interface{}),
			LastUpdated: time.Now(),
		},
	}

	return server
}

// startTestWebSocketHub starts the WebSocket hub for tests that need it
func startTestWebSocketHub(server *Server) func() {
	go server.wsHub.Run()
	return func() {
		server.wsHub.Stop()
	}
}
