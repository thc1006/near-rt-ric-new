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
	"strings"
	"testing"
	"time"
)

// MockA1MediatorClient for testing
type MockA1MediatorClient struct {
	policyTypes     map[PolicyTypeID]*PolicyType
	policyInstances map[PolicyInstanceID]*PolicyInstance
}

func NewMockA1MediatorClient() *MockA1MediatorClient {
	return &MockA1MediatorClient{
		policyTypes:     make(map[PolicyTypeID]*PolicyType),
		policyInstances: make(map[PolicyInstanceID]*PolicyInstance),
	}
}

func (m *MockA1MediatorClient) IsConnected() bool {
	return true
}

func (m *MockA1MediatorClient) GetHealth(ctx context.Context) (*A1Health, error) {
	return &A1Health{
		IsHealthy:       true,
		StatusMessage:   "Healthy",
		LastHealthCheck: time.Now(),
		Version:         "1.0.0",
	}, nil
}

func (m *MockA1MediatorClient) GetPolicyTypes(ctx context.Context) (*PolicyTypeListResponse, error) {
	var policyTypes []PolicyType
	for _, pt := range m.policyTypes {
		policyTypes = append(policyTypes, *pt)
	}
	return &PolicyTypeListResponse{
		PolicyTypes: policyTypes,
		Total:       uint32(len(policyTypes)),
	}, nil
}

func (m *MockA1MediatorClient) GetPolicyType(ctx context.Context, policyTypeID PolicyTypeID) (*PolicyType, error) {
	if pt, exists := m.policyTypes[policyTypeID]; exists {
		return pt, nil
	}
	return nil, &A1ErrorResponse{
		Title:  "Not Found",
		Status: 404,
		Detail: "Policy type not found",
	}
}

func (m *MockA1MediatorClient) CreatePolicyType(ctx context.Context, policyTypeID PolicyTypeID, request *PolicyTypeRequest) error {
	m.policyTypes[policyTypeID] = &PolicyType{
		ID:          policyTypeID,
		Name:        request.Name,
		Description: request.Description,
		Schema:      request.Schema,
		CreatedAt:   time.Now(),
	}
	return nil
}

func (m *MockA1MediatorClient) DeletePolicyType(ctx context.Context, policyTypeID PolicyTypeID) error {
	if _, exists := m.policyTypes[policyTypeID]; !exists {
		return &A1ErrorResponse{
			Title:  "Not Found",
			Status: 404,
			Detail: "Policy type not found",
		}
	}
	delete(m.policyTypes, policyTypeID)
	return nil
}

func (m *MockA1MediatorClient) GetPolicyInstances(ctx context.Context, policyTypeID PolicyTypeID) (*PolicyInstanceListResponse, error) {
	var instances []PolicyInstance
	for _, pi := range m.policyInstances {
		if pi.TypeID == policyTypeID {
			instances = append(instances, *pi)
		}
	}
	return &PolicyInstanceListResponse{
		PolicyInstances: instances,
		Total:           uint32(len(instances)),
	}, nil
}

func (m *MockA1MediatorClient) GetPolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) (*PolicyInstance, error) {
	if pi, exists := m.policyInstances[policyInstanceID]; exists && pi.TypeID == policyTypeID {
		return pi, nil
	}
	return nil, &A1ErrorResponse{
		Title:  "Not Found",
		Status: 404,
		Detail: "Policy instance not found",
	}
}

func (m *MockA1MediatorClient) CreatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID, request *PolicyInstanceRequest) error {
	m.policyInstances[policyInstanceID] = &PolicyInstance{
		ID:        policyInstanceID,
		TypeID:    policyTypeID,
		Policy:    request.Policy,
		Status:    PolicyStatus{Status: "ACTIVE", LastUpdate: time.Now()},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

func (m *MockA1MediatorClient) UpdatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, update *PolicyInstanceUpdate) error {
	if pi, exists := m.policyInstances[update.PolicyInstanceID]; exists && pi.TypeID == policyTypeID {
		pi.Policy = update.Policy
		pi.UpdatedAt = time.Now()
		return nil
	}
	return &A1ErrorResponse{
		Title:  "Not Found",
		Status: 404,
		Detail: "Policy instance not found",
	}
}

func (m *MockA1MediatorClient) DeletePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) error {
	if pi, exists := m.policyInstances[policyInstanceID]; exists && pi.TypeID == policyTypeID {
		delete(m.policyInstances, policyInstanceID)
		return nil
	}
	return &A1ErrorResponse{
		Title:  "Not Found",
		Status: 404,
		Detail: "Policy instance not found",
	}
}

func (m *MockA1MediatorClient) GetPolicyInstanceStatus(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) (*PolicyStatus, error) {
	if pi, exists := m.policyInstances[policyInstanceID]; exists && pi.TypeID == policyTypeID {
		return &pi.Status, nil
	}
	return nil, &A1ErrorResponse{
		Title:  "Not Found",
		Status: 404,
		Detail: "Policy instance not found",
	}
}

func (m *MockA1MediatorClient) GetStats(ctx context.Context) (*A1Stats, error) {
	return &A1Stats{
		PolicyTypesByStatus:     map[string]uint32{"ACTIVE": uint32(len(m.policyTypes))},
		PolicyInstancesByType:   make(map[string]uint32),
		PolicyInstancesByStatus: map[string]uint32{"ACTIVE": uint32(len(m.policyInstances))},
		TotalPolicyTypes:        uint32(len(m.policyTypes)),
		TotalPolicyInstances:    uint32(len(m.policyInstances)),
		LastUpdated:             time.Now(),
	}, nil
}

func (m *MockA1MediatorClient) ValidatePolicy(ctx context.Context, policyTypeID PolicyTypeID, policy json.RawMessage) (*PolicyValidationResult, error) {
	// Basic validation - check if it's valid JSON
	var policyData interface{}
	if err := json.Unmarshal(policy, &policyData); err != nil {
		return &PolicyValidationResult{
			IsValid: false,
			Errors: []PolicyValidationError{
				{
					Field:   "policy",
					Message: "Invalid JSON format",
					Value:   string(policy),
				},
			},
		}, nil
	}
	
	return &PolicyValidationResult{
		IsValid: true,
		Errors:  []PolicyValidationError{},
	}, nil
}

func TestPolicyManager_ValidatePolicyType(t *testing.T) {
	mockClient := NewMockA1MediatorClient()
	pm := NewPolicyManager(mockClient)
	defer pm.Stop()

	// Test valid JSON schema
	validSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"priority": {"type": "integer"},
			"action": {"type": "string"}
		},
		"required": ["priority", "action"]
	}`)

	result, err := pm.ValidatePolicyType("test-type", validSchema)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected valid schema, got invalid with errors: %v", result.Errors)
	}

	// Test invalid JSON schema
	invalidSchema := json.RawMessage(`{
		"type": "invalid-type"
	}`)

	result, err = pm.ValidatePolicyType("test-type", invalidSchema)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.IsValid {
		t.Errorf("Expected invalid schema, got valid")
	}
}

func TestPolicyManager_ValidatePolicyInstance(t *testing.T) {
	mockClient := NewMockA1MediatorClient()
	pm := NewPolicyManager(mockClient)
	defer pm.Stop()

	// Create a policy type first
	policyTypeID := PolicyTypeID("test-type")
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"priority": {"type": "integer"},
			"action": {"type": "string"}
		},
		"required": ["priority", "action"]
	}`)

	mockClient.CreatePolicyType(context.Background(), policyTypeID, &PolicyTypeRequest{
		Name:        "Test Policy Type",
		Description: "Test policy type for validation",
		Schema:      schema,
	})

	// Test valid policy instance
	validPolicy := json.RawMessage(`{
		"priority": 10,
		"action": "allow"
	}`)

	result, err := pm.ValidatePolicyInstance(policyTypeID, validPolicy)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected valid policy, got invalid with errors: %v", result.Errors)
	}

	// Test invalid policy instance (missing required field)
	invalidPolicy := json.RawMessage(`{
		"priority": 10
	}`)

	result, err = pm.ValidatePolicyInstance(policyTypeID, invalidPolicy)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.IsValid {
		t.Errorf("Expected invalid policy, got valid")
	}
}

func TestPolicyManager_CreatePolicyInstance(t *testing.T) {
	mockClient := NewMockA1MediatorClient()
	pm := NewPolicyManager(mockClient)
	defer pm.Stop()

	// Create a policy type first
	policyTypeID := PolicyTypeID("test-type")
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"priority": {"type": "integer"},
			"action": {"type": "string"}
		},
		"required": ["priority", "action"]
	}`)

	mockClient.CreatePolicyType(context.Background(), policyTypeID, &PolicyTypeRequest{
		Name:        "Test Policy Type",
		Description: "Test policy type",
		Schema:      schema,
	})

	// Register a test xApp
	pm.RegisterXApp("test-xapp", "http://test-xapp:8080")

	// Create a valid policy instance
	policyInstanceID := PolicyInstanceID("test-instance")
	policy := json.RawMessage(`{
		"priority": 10,
		"action": "allow"
	}`)

	ctx := context.Background()
	err := pm.CreatePolicyInstance(ctx, policyTypeID, policyInstanceID, policy)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify policy was created in mock client
	instance, err := mockClient.GetPolicyInstance(ctx, policyTypeID, policyInstanceID)
	if err != nil {
		t.Fatalf("Expected policy instance to be created, got error: %v", err)
	}

	if instance.ID != policyInstanceID {
		t.Errorf("Expected policy instance ID %s, got %s", policyInstanceID, instance.ID)
	}

	// Check distribution status
	time.Sleep(200 * time.Millisecond) // Allow time for async distribution
	distributionStatus := pm.GetPolicyDistributionStatus(policyInstanceID)
	if len(distributionStatus) == 0 {
		t.Errorf("Expected distribution status to be recorded")
	}
}

func TestPolicyManager_ConflictDetection(t *testing.T) {
	mockClient := NewMockA1MediatorClient()
	pm := NewPolicyManager(mockClient)
	defer pm.Stop()

	// Create a policy type
	policyTypeID := PolicyTypeID("test-type")
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"priority": {"type": "integer"},
			"action": {"type": "string"},
			"scope": {
				"type": "object",
				"properties": {
					"resources": {"type": "array", "items": {"type": "string"}}
				}
			}
		}
	}`)

	mockClient.CreatePolicyType(context.Background(), policyTypeID, &PolicyTypeRequest{
		Name:        "Test Policy Type",
		Description: "Test policy type",
		Schema:      schema,
	})

	// Create first policy instance
	policy1 := json.RawMessage(`{
		"priority": 10,
		"action": "allow",
		"scope": {
			"resources": ["cell-1", "cell-2"]
		}
	}`)

	ctx := context.Background()
	err := pm.CreatePolicyInstance(ctx, policyTypeID, "policy-1", policy1)
	if err != nil {
		t.Fatalf("Expected no error creating first policy, got %v", err)
	}

	// Create second policy instance with resource conflict
	policy2 := json.RawMessage(`{
		"priority": 20,
		"action": "deny",
		"scope": {
			"resources": ["cell-1", "cell-3"]
		}
	}`)

	err = pm.CreatePolicyInstance(ctx, policyTypeID, "policy-2", policy2)
	if err != nil {
		t.Fatalf("Expected no error creating second policy, got %v", err)
	}

	// Check for conflicts
	conflicts := pm.GetPolicyConflicts()
	if len(conflicts) == 0 {
		t.Errorf("Expected conflicts to be detected")
	}

	// Verify conflict type
	for _, conflict := range conflicts {
		if conflict.ConflictType != string(PolicyConflictTypeResource) {
			t.Errorf("Expected resource conflict, got %s", conflict.ConflictType)
		}
	}
}

func TestPolicyManager_XAppRegistration(t *testing.T) {
	mockClient := NewMockA1MediatorClient()
	pm := NewPolicyManager(mockClient)
	defer pm.Stop()

	// Register xApps
	pm.RegisterXApp("xapp-1", "http://xapp-1:8080")
	pm.RegisterXApp("xapp-2", "http://xapp-2:8080")

	// Check registered xApps
	xapps := pm.GetRegisteredXApps()
	if len(xapps) != 2 {
		t.Errorf("Expected 2 registered xApps, got %d", len(xapps))
	}

	// Unregister an xApp
	pm.UnregisterXApp("xapp-1")

	xapps = pm.GetRegisteredXApps()
	if len(xapps) != 1 {
		t.Errorf("Expected 1 registered xApp after unregistration, got %d", len(xapps))
	}
}

func TestPolicyValidationHandler(t *testing.T) {
	// Create a test server with mock client
	mockClient := NewMockA1MediatorClient()
	
	// Create a mock client manager
	clientManager := &ClientManager{
		a1MediatorClient: mockClient,
	}
	
	server := &Server{
		clients:       clientManager,
		policyManager: NewPolicyManager(mockClient),
	}
	defer server.policyManager.Stop()

	// Create a policy type for testing
	policyTypeID := PolicyTypeID("test-type")
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"priority": {"type": "integer"},
			"action": {"type": "string"}
		},
		"required": ["priority", "action"]
	}`)

	mockClient.CreatePolicyType(context.Background(), policyTypeID, &PolicyTypeRequest{
		Name:        "Test Policy Type",
		Description: "Test policy type",
		Schema:      schema,
	})

	// Test valid policy validation
	validPolicyJSON := `{"policy": {"priority": 10, "action": "allow"}}`
	req := httptest.NewRequest("POST", "/api/v1/a1/policytypes/test-type/validate", 
		http.NoBody)
	req.Body = http.NoBody
	req.Header.Set("Content-Type", "application/json")
	
	// Create a proper request body
	req = httptest.NewRequest("POST", "/api/v1/a1/policytypes/test-type/validate", 
		http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	
	// Set up mux vars
	req = mux.SetURLVars(req, map[string]string{"policyTypeId": "test-type"})
	
	w := httptest.NewRecorder()
	
	// We need to create the request body properly
	req = httptest.NewRequest("POST", "/api/v1/a1/policytypes/test-type/validate", 
		strings.NewReader(validPolicyJSON))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"policyTypeId": "test-type"})
	
	server.PolicyValidationHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response PolicyValidationResult
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.IsValid {
		t.Errorf("Expected valid policy, got invalid with errors: %v", response.Errors)
	}
}