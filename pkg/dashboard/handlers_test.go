/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func TestHandleGetE2Nodes(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to the E2 nodes endpoint
	req, err := http.NewRequest("GET", "/api/v1/e2nodes", nil)
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

	// Verify response structure
	if _, ok := response["e2nodes"]; !ok {
		t.Error("Expected e2nodes in response")
	}

	if _, ok := response["count"]; !ok {
		t.Error("Expected count in response")
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp in response")
	}

	// Check that we have at least one mock E2 node
	e2nodes, ok := response["e2nodes"].([]interface{})
	if !ok {
		t.Error("Expected e2nodes to be an array")
	}

	if len(e2nodes) == 0 {
		t.Error("Expected at least one E2 node in mock data")
	}

	// Verify first E2 node structure
	if len(e2nodes) > 0 {
		node, ok := e2nodes[0].(map[string]interface{})
		if !ok {
			t.Error("Expected E2 node to be an object")
		}

		expectedFields := []string{"id", "name", "type", "status", "plmnId", "connectionTime"}
		for _, field := range expectedFields {
			if _, exists := node[field]; !exists {
				t.Errorf("Expected field %s in E2 node", field)
			}
		}
	}
}

func TestHandleGetE2Node(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to get a specific E2 node
	req, err := http.NewRequest("GET", "/api/v1/e2nodes/001", nil)
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

	// Verify response structure
	expectedFields := []string{"id", "name", "type", "status", "plmnId", "connectionTime", "supportedFunctions"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field %s in E2 node response", field)
		}
	}

	// Verify the ID matches the requested one
	if response["id"] != "001" {
		t.Errorf("Expected node ID to be '001', got %v", response["id"])
	}
}

func TestHandleGetSubscriptions(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to the subscriptions endpoint
	req, err := http.NewRequest("GET", "/api/v1/subscriptions", nil)
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

	// Verify response structure
	if _, ok := response["subscriptions"]; !ok {
		t.Error("Expected subscriptions in response")
	}

	if _, ok := response["count"]; !ok {
		t.Error("Expected count in response")
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp in response")
	}
}

func TestHandleCreateSubscription(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Start WebSocket hub to prevent blocking
	cleanup := startTestWebSocketHub(server)
	defer cleanup()

	// Create subscription request
	subscriptionReq := map[string]interface{}{
		"e2nodeId":      "gnb-001",
		"ranFunctionId": 1,
	}

	reqBody, err := json.Marshal(subscriptionReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create a request to create a subscription
	req, err := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Verify response structure
	expectedFields := []string{"id", "e2nodeId", "ranFunctionId", "status", "createdTime"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field %s in subscription response", field)
		}
	}

	// Verify the subscription data matches request
	if response["e2nodeId"] != subscriptionReq["e2nodeId"] {
		t.Errorf("Expected e2nodeId to be %v, got %v", subscriptionReq["e2nodeId"], response["e2nodeId"])
	}

	// Handle float64 conversion for JSON numbers
	if ranFunctionId, ok := response["ranFunctionId"].(float64); ok {
		if int(ranFunctionId) != subscriptionReq["ranFunctionId"].(int) {
			t.Errorf("Expected ranFunctionId to be %v, got %v", subscriptionReq["ranFunctionId"], int(ranFunctionId))
		}
	} else {
		t.Errorf("Expected ranFunctionId to be a number, got %T", response["ranFunctionId"])
	}
}

func TestHandleCreateSubscriptionInvalidJSON(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request with invalid JSON
	req, err := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestHandleGetSubscription(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to get a specific subscription
	req, err := http.NewRequest("GET", "/api/v1/subscriptions/sub-001", nil)
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

	// Verify response structure
	expectedFields := []string{"id", "e2nodeId", "ranFunctionId", "status", "createdTime"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field %s in subscription response", field)
		}
	}

	// Verify the ID matches the requested one
	if response["id"] != "sub-001" {
		t.Errorf("Expected subscription ID to be 'sub-001', got %v", response["id"])
	}
}

func TestHandleDeleteSubscription(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Start WebSocket hub to prevent blocking
	cleanup := startTestWebSocketHub(server)
	defer cleanup()

	// Create a request to delete a subscription
	req, err := http.NewRequest("DELETE", "/api/v1/subscriptions/sub-001", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}

func TestHandleGetXApps(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to the xApps endpoint
	req, err := http.NewRequest("GET", "/api/v1/xapps", nil)
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

	// Verify response structure
	if _, ok := response["xapps"]; !ok {
		t.Error("Expected xapps in response")
	}

	if _, ok := response["count"]; !ok {
		t.Error("Expected count in response")
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp in response")
	}
}

func TestHandleDeployXApp(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Start WebSocket hub to prevent blocking
	cleanup := startTestWebSocketHub(server)
	defer cleanup()

	// Create xApp deployment request
	xappReq := map[string]interface{}{
		"name":    "test-xapp",
		"version": "1.0.0",
	}

	reqBody, err := json.Marshal(xappReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create a request to deploy an xApp
	req, err := http.NewRequest("POST", "/api/v1/xapps", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Verify response structure
	expectedFields := []string{"name", "version", "status", "instances", "deployedTime"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field %s in xApp response", field)
		}
	}

	// Verify the xApp data matches request
	if response["name"] != xappReq["name"] {
		t.Errorf("Expected name to be %v, got %v", xappReq["name"], response["name"])
	}

	if response["version"] != xappReq["version"] {
		t.Errorf("Expected version to be %v, got %v", xappReq["version"], response["version"])
	}
}

func TestHandleDeployXAppInvalidJSON(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request with invalid JSON
	req, err := http.NewRequest("POST", "/api/v1/xapps", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestHandleGetXApp(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to get a specific xApp
	req, err := http.NewRequest("GET", "/api/v1/xapps/hello-world", nil)
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

	// Verify response structure
	expectedFields := []string{"name", "version", "status", "instances", "deployedTime", "configuration"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field %s in xApp response", field)
		}
	}

	// Verify the name matches the requested one
	if response["name"] != "hello-world" {
		t.Errorf("Expected xApp name to be 'hello-world', got %v", response["name"])
	}
}

func TestHandleUndeployXApp(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Start WebSocket hub to prevent blocking
	cleanup := startTestWebSocketHub(server)
	defer cleanup()

	// Create a request to undeploy an xApp
	req, err := http.NewRequest("DELETE", "/api/v1/xapps/hello-world", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}

func TestHandleGetComponents(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

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

	// Verify response structure
	if _, ok := response["components"]; !ok {
		t.Error("Expected components in response")
	}

	if _, ok := response["count"]; !ok {
		t.Error("Expected count in response")
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp in response")
	}

	// Check that we have the expected components
	components, ok := response["components"].(map[string]interface{})
	if !ok {
		t.Error("Expected components to be an object")
	}

	// Should have at least the basic components
	expectedComponents := []string{"e2manager", "submgr", "appmgr"}
	for _, expectedID := range expectedComponents {
		if _, exists := components[expectedID]; !exists {
			t.Errorf("Expected component %s to be present", expectedID)
		}
	}
}

func TestHandleGetComponent(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to get a specific component
	req, err := http.NewRequest("GET", "/api/v1/components/e2manager", nil)
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

	// Verify response structure
	expectedFields := []string{"id", "name", "type", "status", "endpoint", "lastUpdated"}
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field %s in component response", field)
		}
	}

	// Verify the ID matches the requested one
	if response["id"] != "e2manager" {
		t.Errorf("Expected component ID to be 'e2manager', got %v", response["id"])
	}
}

func TestHandleGetComponentNotFound(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to get a non-existent component
	req, err := http.NewRequest("GET", "/api/v1/components/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

// Test helper function to create a test router with mux variables
func createTestRouter(handler http.HandlerFunc, path string) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc(path, handler)
	return router
}

// Benchmark tests for performance
func BenchmarkHandleGetE2Nodes(b *testing.B) {
	server := createTestServer(&testing.T{})

	router := server.setupRoutes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/e2nodes", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

func TestHandleHealth(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

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

	// Verify response structure
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

func TestHandleWebSocketUpgrade(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Start WebSocket hub
	go server.wsHub.Run()
	defer server.wsHub.Stop()

	// Create test HTTP server
	testServer := httptest.NewServer(server.setupRoutes())
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Give time for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Verify we can read the welcome message
	_, welcomeMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read welcome message: %v", err)
	}

	var welcome map[string]interface{}
	if err := json.Unmarshal(welcomeMsg, &welcome); err != nil {
		t.Errorf("Failed to unmarshal welcome message: %v", err)
	}

	if welcome["type"] != "welcome" {
		t.Errorf("Expected welcome message, got %v", welcome["type"])
	}
}

// Additional error handling tests
func TestHandleGetE2NodeNotFound(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to get a non-existent E2 node
	req, err := http.NewRequest("GET", "/api/v1/e2nodes/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code - should still return 200 with mock data
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Verify the ID matches the requested one (mock returns the requested ID)
	if response["id"] != "nonexistent" {
		t.Errorf("Expected node ID to be 'nonexistent', got %v", response["id"])
	}
}

func TestHandleGetSubscriptionNotFound(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to get a non-existent subscription
	req, err := http.NewRequest("GET", "/api/v1/subscriptions/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code - should still return 200 with mock data
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Verify the ID matches the requested one (mock returns the requested ID)
	if response["id"] != "nonexistent" {
		t.Errorf("Expected subscription ID to be 'nonexistent', got %v", response["id"])
	}
}

func TestHandleGetXAppNotFound(t *testing.T) {
	// Create a test server
	server := createTestServer(t)

	// Create a request to get a non-existent xApp
	req, err := http.NewRequest("GET", "/api/v1/xapps/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	router := server.setupRoutes()

	// Serve the HTTP request
	router.ServeHTTP(rr, req)

	// Check the status code - should still return 200 with mock data
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	// Verify the name matches the requested one (mock returns the requested name)
	if response["name"] != "nonexistent" {
		t.Errorf("Expected xApp name to be 'nonexistent', got %v", response["name"])
	}
}
