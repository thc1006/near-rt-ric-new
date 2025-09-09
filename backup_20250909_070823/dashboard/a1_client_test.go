/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestA1MediatorClientHealth(t *testing.T) {
	// Create a mock A1 Mediator server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/a1-p/healthcheck" && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version": "1.0.0",
				"status":  "healthy",
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create A1 Mediator client
	client := NewA1MediatorClient(&http.Client{Timeout: 5 * time.Second}, server.URL)

	// Test IsConnected
	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}

	// Test GetHealth
	ctx := context.Background()
	health, err := client.GetHealth(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !health.IsHealthy {
		t.Error("Expected health to be healthy")
	}

	if health.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", health.Version)
	}
}

func TestA1MediatorClientPolicyTypes(t *testing.T) {
	// Create a mock A1 Mediator server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/a1-p/policytypes" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]string{"policy-type-1", "policy-type-2"})
		case r.URL.Path == "/a1-p/policytypes/policy-type-1" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":        "Test Policy Type",
				"description": "A test policy type",
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"priority": map[string]interface{}{
							"type": "integer",
						},
					},
				},
			})
		case r.URL.Path == "/a1-p/policytypes/policy-type-2" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":        "Another Policy Type",
				"description": "Another test policy type",
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"threshold": map[string]interface{}{
							"type": "number",
						},
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create A1 Mediator client
	client := NewA1MediatorClient(&http.Client{Timeout: 5 * time.Second}, server.URL)

	// Test GetPolicyTypes
	ctx := context.Background()
	policyTypes, err := client.GetPolicyTypes(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if policyTypes.Total != 2 {
		t.Errorf("Expected 2 policy types, got %d", policyTypes.Total)
	}

	if len(policyTypes.PolicyTypes) != 2 {
		t.Errorf("Expected 2 policy types in list, got %d", len(policyTypes.PolicyTypes))
	}

	// Verify first policy type
	policyType1 := policyTypes.PolicyTypes[0]
	if policyType1.ID != "policy-type-1" {
		t.Errorf("Expected policy type ID 'policy-type-1', got %s", policyType1.ID)
	}
	if policyType1.Name != "Test Policy Type" {
		t.Errorf("Expected policy type name 'Test Policy Type', got %s", policyType1.Name)
	}
}

func TestA1MediatorClientPolicyInstances(t *testing.T) {
	// Create a mock A1 Mediator server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/a1-p/policytypes/policy-type-1/policies" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]string{"policy-instance-1", "policy-instance-2"})
		case r.URL.Path == "/a1-p/policytypes/policy-type-1/policies/policy-instance-1" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"priority": 10,
				"scope":    "cell-1",
			})
		case r.URL.Path == "/a1-p/policytypes/policy-type-1/policies/policy-instance-2" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"priority": 20,
				"scope":    "cell-2",
			})
		case r.URL.Path == "/a1-p/policytypes/policy-type-1/policies/policy-instance-1/status" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":      "ACTIVE",
				"last_update": time.Now().Format(time.RFC3339),
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create A1 Mediator client
	client := NewA1MediatorClient(&http.Client{Timeout: 5 * time.Second}, server.URL)

	// Test GetPolicyInstances
	ctx := context.Background()
	policyInstances, err := client.GetPolicyInstances(ctx, "policy-type-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if policyInstances.Total != 2 {
		t.Errorf("Expected 2 policy instances, got %d", policyInstances.Total)
	}

	if len(policyInstances.PolicyInstances) != 2 {
		t.Errorf("Expected 2 policy instances in list, got %d", len(policyInstances.PolicyInstances))
	}

	// Test GetPolicyInstance
	policyInstance, err := client.GetPolicyInstance(ctx, "policy-type-1", "policy-instance-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if policyInstance.ID != "policy-instance-1" {
		t.Errorf("Expected policy instance ID 'policy-instance-1', got %s", policyInstance.ID)
	}

	if policyInstance.TypeID != "policy-type-1" {
		t.Errorf("Expected policy type ID 'policy-type-1', got %s", policyInstance.TypeID)
	}

	// Test GetPolicyInstanceStatus
	status, err := client.GetPolicyInstanceStatus(ctx, "policy-type-1", "policy-instance-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if status.Status != "ACTIVE" {
		t.Errorf("Expected status 'ACTIVE', got %s", status.Status)
	}
}

func TestA1MediatorClientNotConnected(t *testing.T) {
	// Create A1 Mediator client with invalid endpoint
	client := NewA1MediatorClient(&http.Client{Timeout: 1 * time.Second}, "http://invalid-endpoint:9999")

	// Test IsConnected
	if client.IsConnected() {
		t.Error("Expected client to not be connected")
	}

	// Test GetHealth with connection failure
	ctx := context.Background()
	health, err := client.GetHealth(ctx)
	if err != nil {
		t.Fatalf("Expected no error (graceful degradation), got %v", err)
	}

	if health.IsHealthy {
		t.Error("Expected health to be unhealthy")
	}

	if health.StatusMessage != "Connection failed" {
		t.Errorf("Expected status message 'Connection failed', got %s", health.StatusMessage)
	}
}

func TestA1MediatorClientValidatePolicy(t *testing.T) {
	// Create A1 Mediator client
	client := NewA1MediatorClient(&http.Client{Timeout: 5 * time.Second}, "http://test-endpoint")

	ctx := context.Background()

	// Test valid JSON policy
	validPolicy := json.RawMessage(`{"priority": 10, "scope": "cell-1"}`)
	result, err := client.ValidatePolicy(ctx, "policy-type-1", validPolicy)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.IsValid {
		t.Error("Expected policy to be valid")
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no validation errors, got %d", len(result.Errors))
	}

	// Test invalid JSON policy
	invalidPolicy := json.RawMessage(`{"priority": 10, "scope": "cell-1"`)
	result, err = client.ValidatePolicy(ctx, "policy-type-1", invalidPolicy)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.IsValid {
		t.Error("Expected policy to be invalid")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors")
	}

	if result.Errors[0].Field != "policy" {
		t.Errorf("Expected error field 'policy', got %s", result.Errors[0].Field)
	}
}