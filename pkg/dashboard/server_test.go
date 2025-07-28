/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthEndpoint(t *testing.T) {
	// Create a test server
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create a request to the health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status to be 'healthy', got %v", response["status"])
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp in response")
	}

	if _, ok := response["components"]; !ok {
		t.Error("Expected components in response")
	}
}

func TestGetComponentsEndpoint(t *testing.T) {
	// Create a test server
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Trigger initial discovery
	server.discovery.discoverComponents()

	// Create a request to the components endpoint
	req, err := http.NewRequest("GET", "/api/v1/components", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if _, ok := response["components"]; !ok {
		t.Error("Expected components in response")
	}

	if _, ok := response["count"]; !ok {
		t.Error("Expected count in response")
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp in response")
	}
}

func TestCORSMiddleware(t *testing.T) {
	// Create a test server
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create an OPTIONS request
	req, err := http.NewRequest("OPTIONS", "/api/v1/components", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check CORS headers
	if header := rr.Header().Get("Access-Control-Allow-Origin"); header != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin to be '*', got %v", header)
	}

	if header := rr.Header().Get("Access-Control-Allow-Methods"); header == "" {
		t.Error("Expected Access-Control-Allow-Methods header to be set")
	}

	if header := rr.Header().Get("Access-Control-Allow-Headers"); header == "" {
		t.Error("Expected Access-Control-Allow-Headers header to be set")
	}
}

func TestDiscoveryService(t *testing.T) {
	// Create a test client manager
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	clients, err := NewClientManager(config)
	if err != nil {
		t.Fatalf("Failed to create client manager: %v", err)
	}
	defer clients.Close()

	// Create discovery service
	discovery := NewDiscoveryService(clients)

	// Test initial state
	components := discovery.GetComponents()
	if len(components) != 0 {
		t.Errorf("Expected 0 components initially, got %d", len(components))
	}

	// Run discovery
	discovery.discoverComponents()

	// Check that components were discovered
	components = discovery.GetComponents()
	if len(components) == 0 {
		t.Error("Expected components to be discovered")
	}

	// Check that expected components exist
	expectedComponents := []string{"e2manager", "submgr", "appmgr"}
	for _, expectedID := range expectedComponents {
		if component, exists := discovery.GetComponent(expectedID); !exists {
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
	status := discovery.GetComponentStatus()
	if len(status) == 0 {
		t.Error("Expected component status to be available")
	}
}

func TestWebSocketHub(t *testing.T) {
	hub := NewWebSocketHub()

	// Test initial state
	if len(hub.clients) != 0 {
		t.Error("Expected no clients initially")
	}

	// Test broadcast message
	go hub.Run()
	defer hub.Stop()

	// Give the hub time to start
	time.Sleep(100 * time.Millisecond)

	// Test broadcasting a message (should not panic even with no clients)
	hub.BroadcastMessage("test", map[string]string{"message": "hello"})

	// Give time for message processing
	time.Sleep(100 * time.Millisecond)
}
