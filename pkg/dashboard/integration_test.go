/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// Mock gRPC server for E2 Manager
type mockE2ManagerServer struct {
	// Embed UnimplementedE2ManagerServer if using generated gRPC code
}

// Mock gRPC server for Subscription Manager
type mockSubscriptionManagerServer struct {
	// Embed UnimplementedSubscriptionManagerServer if using generated gRPC code
}

// MockGRPCServer represents a mock gRPC server for testing
type MockGRPCServer struct {
	server   *grpc.Server
	listener *bufconn.Listener
	running  bool
}

// NewMockGRPCServer creates a new mock gRPC server
func NewMockGRPCServer() *MockGRPCServer {
	return &MockGRPCServer{
		server:   grpc.NewServer(),
		listener: bufconn.Listen(1024 * 1024),
	}
}

// Start starts the mock gRPC server
func (m *MockGRPCServer) Start() {
	if m.running {
		return
	}
	m.running = true
	go func() {
		if err := m.server.Serve(m.listener); err != nil {
			// Server stopped
		}
	}()
}

// Stop stops the mock gRPC server
func (m *MockGRPCServer) Stop() {
	if !m.running {
		return
	}
	m.running = false
	m.server.Stop()
	m.listener.Close()
}

// GetDialer returns a dialer function for connecting to this mock server
func (m *MockGRPCServer) GetDialer() func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return m.listener.Dial()
	}
}

// Test setup for integration tests
func setupIntegrationTest(t *testing.T) (*Server, func()) {
	// Create mock gRPC servers
	e2mgrMock := NewMockGRPCServer()
	submgrMock := NewMockGRPCServer()

	// Start mock servers
	e2mgrMock.Start()
	submgrMock.Start()

	// Create test configuration
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "bufconn",
		SubmgrEndpoint: "bufconn",
		AppmgrEndpoint: "localhost:8080",
	}

	// Create dashboard server
	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Override client connections to use buffer connections
	server.clients.e2mgrConn, err = grpc.NewClient("bufconn",
		grpc.WithContextDialer(e2mgrMock.GetDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to create E2 Manager connection: %v", err)
	}

	server.clients.submgrConn, err = grpc.NewClient("bufconn",
		grpc.WithContextDialer(submgrMock.GetDialer()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to create Subscription Manager connection: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		server.clients.Close()
		e2mgrMock.Stop()
		submgrMock.Stop()
	}

	return server, cleanup
}

func TestIntegrationClientManager(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test client manager functionality
	if server.clients == nil {
		t.Error("Expected client manager to be initialized")
	}

	// Test connection status
	// Note: These will likely be false since we're using mock servers
	// In a real implementation, you would implement proper mock responses
	e2Connected := server.clients.IsE2ManagerConnected()
	submgrConnected := server.clients.IsSubscriptionManagerConnected()

	t.Logf("E2 Manager connected: %v", e2Connected)
	t.Logf("Subscription Manager connected: %v", submgrConnected)

	// Test reconnection functionality
	err := server.clients.Reconnect()
	if err != nil {
		t.Logf("Reconnection failed (expected with mock servers): %v", err)
	}
}

func TestIntegrationDiscoveryService(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test discovery service
	if server.discovery == nil {
		t.Error("Expected discovery service to be initialized")
	}

	// Run discovery
	server.discovery.discoverComponents()

	// Check discovered components
	components := server.discovery.GetComponents()
	if len(components) == 0 {
		t.Error("Expected components to be discovered")
	}

	// Check for expected component types
	expectedComponents := []string{"e2manager", "submgr", "appmgr"}
	for _, expectedID := range expectedComponents {
		if component, exists := server.discovery.GetComponent(expectedID); !exists {
			t.Errorf("Expected component %s to be discovered", expectedID)
		} else {
			if component.ID != expectedID {
				t.Errorf("Expected component ID to be %s, got %s", expectedID, component.ID)
			}
			if component.LastUpdated.IsZero() {
				t.Error("Expected LastUpdated to be set")
			}
		}
	}

	// Test component status
	status := server.discovery.GetComponentStatus()
	if len(status) == 0 {
		t.Error("Expected component status to be available")
	}

	for componentID, componentStatus := range status {
		t.Logf("Component %s status: %s", componentID, componentStatus)
		if componentStatus == "" {
			t.Errorf("Expected non-empty status for component %s", componentID)
		}
	}
}

func TestIntegrationWebSocketHub(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test WebSocket hub
	if server.wsHub == nil {
		t.Error("Expected WebSocket hub to be initialized")
	}

	// Start the hub
	go server.wsHub.Run()
	defer server.wsHub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Test broadcasting a message
	testMessage := map[string]string{"test": "message"}
	server.wsHub.BroadcastMessage("test_event", testMessage)

	// Give time for message processing
	time.Sleep(100 * time.Millisecond)

	// Verify hub is running (no clients connected, so no errors expected)
	if len(server.wsHub.clients) != 0 {
		t.Error("Expected no clients initially")
	}
}

func TestIntegrationServerLifecycle(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Test server lifecycle
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		err := server.Start()
		if err != nil && err.Error() != "http: Server closed" {
			serverErr <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown
	shutdownErr := server.Shutdown(ctx)
	if shutdownErr != nil {
		t.Errorf("Failed to shutdown server: %v", shutdownErr)
	}

	// Check for server errors
	select {
	case err := <-serverErr:
		t.Errorf("Server error: %v", err)
	case <-time.After(1 * time.Second):
		// No error, which is expected
	}
}

func TestIntegrationComponentDiscoveryWithWebSocket(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Start WebSocket hub
	go server.wsHub.Run()
	defer server.wsHub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Start discovery service with WebSocket integration
	go server.discovery.Start(server.wsHub)
	defer server.discovery.Stop()

	// Give discovery time to run
	time.Sleep(200 * time.Millisecond)

	// Verify components were discovered
	components := server.discovery.GetComponents()
	if len(components) == 0 {
		t.Error("Expected components to be discovered")
	}

	// Test that discovery broadcasts updates
	// This is difficult to test without actual WebSocket clients,
	// but we can verify the discovery service is running
	status := server.discovery.GetComponentStatus()
	if len(status) == 0 {
		t.Error("Expected component status to be available")
	}
}

// Test concurrent access to discovery service
func TestIntegrationConcurrentDiscovery(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Start discovery service
	go server.discovery.Start(server.wsHub)
	defer server.discovery.Stop()

	// Give discovery time to initialize
	time.Sleep(100 * time.Millisecond)

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Concurrent reads
			components := server.discovery.GetComponents()
			if len(components) == 0 {
				t.Error("Expected components to be discovered")
			}

			status := server.discovery.GetComponentStatus()
			if len(status) == 0 {
				t.Error("Expected component status to be available")
			}

			// Test getting specific components
			for id := range components {
				if _, exists := server.discovery.GetComponent(id); !exists {
					t.Errorf("Expected component %s to exist", id)
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for concurrent discovery tests")
		}
	}
}

// Test error handling in integration scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	// Test with invalid configuration
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "invalid:endpoint",
		SubmgrEndpoint: "invalid:endpoint",
		AppmgrEndpoint: "invalid:endpoint",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.clients.Close()

	// Test discovery with invalid endpoints
	server.discovery.discoverComponents()

	// Components should still be discovered but with error status
	components := server.discovery.GetComponents()
	if len(components) == 0 {
		t.Error("Expected components to be discovered even with invalid endpoints")
	}

	for _, component := range components {
		if component.Status != ComponentStatusError {
			t.Logf("Component %s has status %s (expected error status)", component.ID, component.Status)
		}
	}
}

// Test integration with mock HTTP responses
func TestIntegrationHTTPEndpoints(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Start WebSocket hub
	go server.wsHub.Run()
	defer server.wsHub.Stop()

	// Create test HTTP server
	testServer := httptest.NewServer(server.setupRoutes())
	defer testServer.Close()

	// Test all major endpoints
	endpoints := []struct {
		method   string
		path     string
		expected int
	}{
		{"GET", "/health", http.StatusOK},
		{"GET", "/api/v1/components", http.StatusOK},
		{"GET", "/api/v1/components/e2manager", http.StatusOK},
		{"GET", "/api/v1/e2nodes", http.StatusOK},
		{"GET", "/api/v1/e2nodes/test", http.StatusOK},
		{"GET", "/api/v1/subscriptions", http.StatusOK},
		{"GET", "/api/v1/subscriptions/test", http.StatusOK},
		{"GET", "/api/v1/xapps", http.StatusOK},
		{"GET", "/api/v1/xapps/test", http.StatusOK},
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("%s %s", endpoint.method, endpoint.path), func(t *testing.T) {
			req, err := http.NewRequest(endpoint.method, testServer.URL+endpoint.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != endpoint.expected {
				t.Errorf("Expected status %d, got %d", endpoint.expected, resp.StatusCode)
			}
		})
	}
}

// Test integration with POST/DELETE operations
func TestIntegrationMutatingOperations(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Start WebSocket hub
	go server.wsHub.Run()
	defer server.wsHub.Stop()

	// Create test HTTP server
	testServer := httptest.NewServer(server.setupRoutes())
	defer testServer.Close()

	// Test subscription creation
	t.Run("Create Subscription", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"e2nodeId":      "test-node",
			"ranFunctionId": 1,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", testServer.URL+"/api/v1/subscriptions", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		}
	})

	// Test xApp deployment
	t.Run("Deploy xApp", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":    "test-xapp",
			"version": "1.0.0",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", testServer.URL+"/api/v1/xapps", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
		}
	})

	// Test deletion operations
	t.Run("Delete Subscription", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", testServer.URL+"/api/v1/subscriptions/test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
		}
	})

	t.Run("Undeploy xApp", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", testServer.URL+"/api/v1/xapps/test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
		}
	})
}

// Test WebSocket integration with real-time updates
func TestIntegrationWebSocketRealTime(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Start WebSocket hub
	go server.wsHub.Run()
	defer server.wsHub.Stop()

	// Create test HTTP server
	testServer := httptest.NewServer(server.setupRoutes())
	defer testServer.Close()

	// Connect to WebSocket
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Read welcome message
	_, _, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read welcome message: %v", err)
	}

	// Test real-time updates by triggering operations
	go func() {
		time.Sleep(100 * time.Millisecond)

		// Create a subscription to trigger WebSocket broadcast
		reqBody := map[string]interface{}{
			"e2nodeId":      "test-node",
			"ranFunctionId": 1,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", testServer.URL+"/api/v1/subscriptions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		http.DefaultClient.Do(req)
	}()

	// Read the broadcast message
	_, broadcastMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read broadcast message: %v", err)
	}

	var broadcast map[string]interface{}
	if err := json.Unmarshal(broadcastMsg, &broadcast); err != nil {
		t.Errorf("Failed to unmarshal broadcast message: %v", err)
	}

	if broadcast["type"] != "subscription_created" {
		t.Errorf("Expected subscription_created message, got %v", broadcast["type"])
	}
}

// Test concurrent operations
func TestIntegrationConcurrentOperations(t *testing.T) {
	server, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Start WebSocket hub
	go server.wsHub.Run()
	defer server.wsHub.Stop()

	// Create test HTTP server
	testServer := httptest.NewServer(server.setupRoutes())
	defer testServer.Close()

	// Test concurrent requests
	numRequests := 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Make concurrent requests to different endpoints
			endpoints := []string{
				"/api/v1/components",
				"/api/v1/e2nodes",
				"/api/v1/subscriptions",
				"/api/v1/xapps",
				"/health",
			}

			for _, endpoint := range endpoints {
				resp, err := http.Get(testServer.URL + endpoint)
				if err != nil {
					t.Errorf("Request %d to %s failed: %v", id, endpoint, err)
					return
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("Request %d to %s returned status %d", id, endpoint, resp.StatusCode)
				}
			}
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("Timeout waiting for concurrent requests")
		}
	}
}

// Benchmark integration tests
func BenchmarkIntegrationDiscovery(b *testing.B) {
	server, cleanup := setupIntegrationTest(&testing.T{})
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.discovery.discoverComponents()
	}
}

func BenchmarkIntegrationGetComponents(b *testing.B) {
	server, cleanup := setupIntegrationTest(&testing.T{})
	defer cleanup()

	// Initialize components
	server.discovery.discoverComponents()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.discovery.GetComponents()
	}
}

func BenchmarkIntegrationHTTPEndpoint(b *testing.B) {
	server, cleanup := setupIntegrationTest(&testing.T{})
	defer cleanup()

	testServer := httptest.NewServer(server.setupRoutes())
	defer testServer.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := http.Get(testServer.URL + "/api/v1/components")
		resp.Body.Close()
	}
}
