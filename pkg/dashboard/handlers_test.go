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
	"testing"

	"github.com/gorilla/mux"
)

func TestHandleGetE2Nodes(t *testing.T) {
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

	if response["ranFunctionId"] != subscriptionReq["ranFunctionId"] {
		t.Errorf("Expected ranFunctionId to be %v, got %v", subscriptionReq["ranFunctionId"], response["ranFunctionId"])
	}
}

func TestHandleCreateSubscriptionInvalidJSON(t *testing.T) {
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

func TestHandleGetXApp(t *testing.T) {
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

func TestHandleGetComponentNotFound(t *testing.T) {
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
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	server, err := NewServer(config)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	router := server.setupRoutes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/e2nodes", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

func BenchmarkHandleGetSubscriptions(b *testing.B) {
	config := &Config{
		Port:           8080,
		E2MgrEndpoint:  "localhost:3800",
		SubmgrEndpoint: "localhost:3801",
		AppmgrEndpoint: "localhost:8080",
	}

	server, err := NewServer(config)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	router := server.setupRoutes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/subscriptions", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}
