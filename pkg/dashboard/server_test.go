/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServerLifecycle(t *testing.T) {
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
	defer server.clients.Close()

	// Test server creation
	if server.config != config {
		t.Error("Expected server config to match")
	}

	if server.clients == nil {
		t.Error("Expected clients to be initialized")
	}

	if server.wsHub == nil {
		t.Error("Expected WebSocket hub to be initialized")
	}

	if server.discovery == nil {
		t.Error("Expected discovery service to be initialized")
	}

	// Test shutdown without starting (should not panic)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		t.Logf("Shutdown error (expected): %v", err)
	}
}

func TestCORSMiddleware(t *testing.T) {
	server := createTestServer(t)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	// Wrap with CORS middleware
	corsHandler := server.corsMiddleware(testHandler)

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	rr := httptest.NewRecorder()
	corsHandler.ServeHTTP(rr, req)

	// Check CORS headers
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin header")
	}

	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}

	if rr.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Expected Access-Control-Allow-Headers header")
	}

	// Test regular request
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr = httptest.NewRecorder()
	corsHandler.ServeHTTP(rr, req)

	// Check CORS headers are still present
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin header on regular request")
	}

	// Check response body
	if rr.Body.String() != "test" {
		t.Error("Expected response body to be 'test'")
	}
}

func TestSetupRoutes(t *testing.T) {
	server := createTestServer(t)

	router := server.setupRoutes()
	if router == nil {
		t.Error("Expected router to be created")
	}

	// Test that routes are registered by making requests
	testRoutes := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"GET", "/api/v1/components"},
		{"GET", "/api/v1/e2nodes"},
		{"GET", "/api/v1/subscriptions"},
		{"GET", "/api/v1/xapps"},
	}

	for _, route := range testRoutes {
		req := httptest.NewRequest(route.method, route.path, nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		// Should not return 404 (route not found)
		if rr.Code == http.StatusNotFound {
			t.Errorf("Route %s %s not found", route.method, route.path)
		}
	}
}

func TestWebSocketUpgradeError(t *testing.T) {
	server := createTestServer(t)

	// Create a regular HTTP request (not WebSocket upgrade)
	req := httptest.NewRequest("GET", "/ws", nil)
	rr := httptest.NewRecorder()

	server.handleWebSocket(rr, req)

	// Should return an error since it's not a proper WebSocket upgrade request
	if rr.Code == http.StatusOK {
		t.Error("Expected WebSocket upgrade to fail for regular HTTP request")
	}
}
