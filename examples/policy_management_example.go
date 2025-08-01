/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0

This example demonstrates the Policy Management Framework functionality
including JSON schema validation, conflict detection, and policy distribution.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/oran/near-rt-ric-new/pkg/dashboard"
)

func main() {
	fmt.Println("Policy Management Framework Example")
	fmt.Println("===================================")

	// Create a mock A1 Mediator client for demonstration
	mockClient := &MockA1Client{}
	
	// Initialize the policy manager
	policyManager := dashboard.NewPolicyManager(mockClient)
	defer policyManager.Stop()

	// Example 1: Policy Type Schema Validation
	fmt.Println("\n1. Policy Type Schema Validation")
	fmt.Println("---------------------------------")
	
	policyTypeID := dashboard.PolicyTypeID("qos-policy-type")
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"priority": {
				"type": "integer",
				"minimum": 1,
				"maximum": 100
			},
			"action": {
				"type": "string",
				"enum": ["allow", "deny", "throttle"]
			},
			"scope": {
				"type": "object",
				"properties": {
					"resources": {
						"type": "array",
						"items": {"type": "string"}
					}
				},
				"required": ["resources"]
			}
		},
		"required": ["priority", "action", "scope"]
	}`)

	validationResult, err := policyManager.ValidatePolicyType(policyTypeID, schema)
	if err != nil {
		log.Printf("Error validating policy type: %v", err)
	} else if validationResult.IsValid {
		fmt.Printf("✓ Policy type schema is valid\n")
	} else {
		fmt.Printf("✗ Policy type schema is invalid: %v\n", validationResult.Errors)
	}

	// Example 2: Policy Instance Validation
	fmt.Println("\n2. Policy Instance Validation")
	fmt.Println("------------------------------")

	// First, create the policy type in our mock client
	mockClient.AddPolicyType(policyTypeID, &dashboard.PolicyType{
		ID:          policyTypeID,
		Name:        "QoS Policy Type",
		Description: "Policy type for QoS management",
		Schema:      schema,
		CreatedAt:   time.Now(),
	})

	// Valid policy instance
	validPolicy := json.RawMessage(`{
		"priority": 10,
		"action": "allow",
		"scope": {
			"resources": ["cell-1", "cell-2"]
		}
	}`)

	validationResult, err = policyManager.ValidatePolicyInstance(policyTypeID, validPolicy)
	if err != nil {
		log.Printf("Error validating policy instance: %v", err)
	} else if validationResult.IsValid {
		fmt.Printf("✓ Valid policy instance\n")
	} else {
		fmt.Printf("✗ Invalid policy instance: %v\n", validationResult.Errors)
	}

	// Invalid policy instance (missing required field)
	invalidPolicy := json.RawMessage(`{
		"priority": 10,
		"action": "allow"
	}`)

	validationResult, err = policyManager.ValidatePolicyInstance(policyTypeID, invalidPolicy)
	if err != nil {
		log.Printf("Error validating policy instance: %v", err)
	} else if !validationResult.IsValid {
		fmt.Printf("✓ Correctly identified invalid policy: %s\n", validationResult.Errors[0].Message)
	}

	// Example 3: xApp Registration and Policy Distribution
	fmt.Println("\n3. xApp Registration and Policy Distribution")
	fmt.Println("--------------------------------------------")

	// Register some xApps
	policyManager.RegisterXApp("qos-xapp", "http://qos-xapp:8080")
	policyManager.RegisterXApp("scheduler-xapp", "http://scheduler-xapp:8080")

	registeredXApps := policyManager.GetRegisteredXApps()
	fmt.Printf("Registered xApps: %v\n", registeredXApps)

	// Create a policy instance (this will trigger distribution)
	ctx := context.Background()
	policyInstanceID := dashboard.PolicyInstanceID("qos-policy-1")
	
	err = policyManager.CreatePolicyInstance(ctx, policyTypeID, policyInstanceID, validPolicy)
	if err != nil {
		log.Printf("Error creating policy instance: %v", err)
	} else {
		fmt.Printf("✓ Policy instance created and distribution initiated\n")
	}

	// Wait a bit for async distribution
	time.Sleep(200 * time.Millisecond)

	// Check distribution status
	distributionStatus := policyManager.GetPolicyDistributionStatus(policyInstanceID)
	fmt.Printf("Distribution status:\n")
	for xappID, status := range distributionStatus {
		fmt.Printf("  %s: %s (%s)\n", xappID, status.Status, status.Message)
	}

	// Example 4: Conflict Detection
	fmt.Println("\n4. Policy Conflict Detection")
	fmt.Println("-----------------------------")

	// Create a conflicting policy (same resources, different action)
	conflictingPolicy := json.RawMessage(`{
		"priority": 20,
		"action": "deny",
		"scope": {
			"resources": ["cell-1", "cell-3"]
		}
	}`)

	conflictingPolicyID := dashboard.PolicyInstanceID("qos-policy-2")
	err = policyManager.CreatePolicyInstance(ctx, policyTypeID, conflictingPolicyID, conflictingPolicy)
	if err != nil {
		log.Printf("Error creating conflicting policy: %v", err)
	} else {
		fmt.Printf("✓ Conflicting policy created\n")
	}

	// Check for conflicts
	conflicts := policyManager.GetPolicyConflicts()
	if len(conflicts) > 0 {
		fmt.Printf("Detected %d conflict(s):\n", len(conflicts))
		for _, conflict := range conflicts {
			fmt.Printf("  %s: %s (Type: %s)\n", conflict.ConflictID, conflict.Description, conflict.ConflictType)
		}

		// Resolve a conflict
		for conflictID := range conflicts {
			err = policyManager.ResolveConflict(conflictID, "Higher priority policy takes precedence")
			if err != nil {
				log.Printf("Error resolving conflict: %v", err)
			} else {
				fmt.Printf("✓ Conflict %s resolved\n", conflictID)
			}
			break // Just resolve the first one for demo
		}
	} else {
		fmt.Printf("No conflicts detected\n")
	}

	// Example 5: Compliance Monitoring
	fmt.Println("\n5. Policy Compliance Monitoring")
	fmt.Println("--------------------------------")

	// Wait for compliance checks to run
	time.Sleep(1 * time.Second)

	complianceReports := policyManager.GetPolicyComplianceReports(policyInstanceID)
	if len(complianceReports) > 0 {
		fmt.Printf("Compliance reports for policy %s:\n", policyInstanceID)
		for xappID, report := range complianceReports {
			fmt.Printf("  %s: %s", xappID, report.ComplianceStatus)
			if len(report.Violations) > 0 {
				fmt.Printf(" (Violations: %v)", report.Violations)
			}
			fmt.Printf("\n")
		}
	} else {
		fmt.Printf("No compliance reports available yet\n")
	}

	fmt.Println("\nPolicy Management Framework demonstration completed!")
}

// MockA1Client implements the A1MediatorClient interface for demonstration
type MockA1Client struct {
	policyTypes     map[dashboard.PolicyTypeID]*dashboard.PolicyType
	policyInstances map[dashboard.PolicyInstanceID]*dashboard.PolicyInstance
}

func (m *MockA1Client) AddPolicyType(id dashboard.PolicyTypeID, policyType *dashboard.PolicyType) {
	if m.policyTypes == nil {
		m.policyTypes = make(map[dashboard.PolicyTypeID]*dashboard.PolicyType)
	}
	m.policyTypes[id] = policyType
}

func (m *MockA1Client) IsConnected() bool {
	return true
}

func (m *MockA1Client) GetHealth(ctx context.Context) (*dashboard.A1Health, error) {
	return &dashboard.A1Health{
		IsHealthy:       true,
		StatusMessage:   "Healthy",
		LastHealthCheck: time.Now(),
		Version:         "1.0.0",
	}, nil
}

func (m *MockA1Client) GetPolicyTypes(ctx context.Context) (*dashboard.PolicyTypeListResponse, error) {
	var policyTypes []dashboard.PolicyType
	for _, pt := range m.policyTypes {
		policyTypes = append(policyTypes, *pt)
	}
	return &dashboard.PolicyTypeListResponse{
		PolicyTypes: policyTypes,
		Total:       uint32(len(policyTypes)),
	}, nil
}

func (m *MockA1Client) GetPolicyType(ctx context.Context, policyTypeID dashboard.PolicyTypeID) (*dashboard.PolicyType, error) {
	if pt, exists := m.policyTypes[policyTypeID]; exists {
		return pt, nil
	}
	return nil, fmt.Errorf("policy type %s not found", policyTypeID)
}

func (m *MockA1Client) CreatePolicyType(ctx context.Context, policyTypeID dashboard.PolicyTypeID, request *dashboard.PolicyTypeRequest) error {
	if m.policyTypes == nil {
		m.policyTypes = make(map[dashboard.PolicyTypeID]*dashboard.PolicyType)
	}
	m.policyTypes[policyTypeID] = &dashboard.PolicyType{
		ID:          policyTypeID,
		Name:        request.Name,
		Description: request.Description,
		Schema:      request.Schema,
		CreatedAt:   time.Now(),
	}
	return nil
}

func (m *MockA1Client) DeletePolicyType(ctx context.Context, policyTypeID dashboard.PolicyTypeID) error {
	if _, exists := m.policyTypes[policyTypeID]; !exists {
		return fmt.Errorf("policy type %s not found", policyTypeID)
	}
	delete(m.policyTypes, policyTypeID)
	return nil
}

func (m *MockA1Client) GetPolicyInstances(ctx context.Context, policyTypeID dashboard.PolicyTypeID) (*dashboard.PolicyInstanceListResponse, error) {
	var instances []dashboard.PolicyInstance
	for _, pi := range m.policyInstances {
		if pi.TypeID == policyTypeID {
			instances = append(instances, *pi)
		}
	}
	return &dashboard.PolicyInstanceListResponse{
		PolicyInstances: instances,
		Total:           uint32(len(instances)),
	}, nil
}

func (m *MockA1Client) GetPolicyInstance(ctx context.Context, policyTypeID dashboard.PolicyTypeID, policyInstanceID dashboard.PolicyInstanceID) (*dashboard.PolicyInstance, error) {
	if pi, exists := m.policyInstances[policyInstanceID]; exists && pi.TypeID == policyTypeID {
		return pi, nil
	}
	return nil, fmt.Errorf("policy instance %s not found", policyInstanceID)
}

func (m *MockA1Client) CreatePolicyInstance(ctx context.Context, policyTypeID dashboard.PolicyTypeID, policyInstanceID dashboard.PolicyInstanceID, request *dashboard.PolicyInstanceRequest) error {
	if m.policyInstances == nil {
		m.policyInstances = make(map[dashboard.PolicyInstanceID]*dashboard.PolicyInstance)
	}
	m.policyInstances[policyInstanceID] = &dashboard.PolicyInstance{
		ID:        policyInstanceID,
		TypeID:    policyTypeID,
		Policy:    request.Policy,
		Status:    dashboard.PolicyStatus{Status: "ACTIVE", LastUpdate: time.Now()},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return nil
}

func (m *MockA1Client) UpdatePolicyInstance(ctx context.Context, policyTypeID dashboard.PolicyTypeID, update *dashboard.PolicyInstanceUpdate) error {
	if pi, exists := m.policyInstances[update.PolicyInstanceID]; exists && pi.TypeID == policyTypeID {
		pi.Policy = update.Policy
		pi.UpdatedAt = time.Now()
		return nil
	}
	return fmt.Errorf("policy instance %s not found", update.PolicyInstanceID)
}

func (m *MockA1Client) DeletePolicyInstance(ctx context.Context, policyTypeID dashboard.PolicyTypeID, policyInstanceID dashboard.PolicyInstanceID) error {
	if pi, exists := m.policyInstances[policyInstanceID]; exists && pi.TypeID == policyTypeID {
		delete(m.policyInstances, policyInstanceID)
		return nil
	}
	return fmt.Errorf("policy instance %s not found", policyInstanceID)
}

func (m *MockA1Client) GetPolicyInstanceStatus(ctx context.Context, policyTypeID dashboard.PolicyTypeID, policyInstanceID dashboard.PolicyInstanceID) (*dashboard.PolicyStatus, error) {
	if pi, exists := m.policyInstances[policyInstanceID]; exists && pi.TypeID == policyTypeID {
		return &pi.Status, nil
	}
	return nil, fmt.Errorf("policy instance %s not found", policyInstanceID)
}

func (m *MockA1Client) GetStats(ctx context.Context) (*dashboard.A1Stats, error) {
	return &dashboard.A1Stats{
		PolicyTypesByStatus:     map[string]uint32{"ACTIVE": uint32(len(m.policyTypes))},
		PolicyInstancesByType:   make(map[string]uint32),
		PolicyInstancesByStatus: map[string]uint32{"ACTIVE": uint32(len(m.policyInstances))},
		TotalPolicyTypes:        uint32(len(m.policyTypes)),
		TotalPolicyInstances:    uint32(len(m.policyInstances)),
		LastUpdated:             time.Now(),
	}, nil
}

func (m *MockA1Client) ValidatePolicy(ctx context.Context, policyTypeID dashboard.PolicyTypeID, policy json.RawMessage) (*dashboard.PolicyValidationResult, error) {
	var policyData interface{}
	if err := json.Unmarshal(policy, &policyData); err != nil {
		return &dashboard.PolicyValidationResult{
			IsValid: false,
			Errors: []dashboard.PolicyValidationError{
				{
					Field:   "policy",
					Message: "Invalid JSON format",
					Value:   string(policy),
				},
			},
		}, nil
	}
	
	return &dashboard.PolicyValidationResult{
		IsValid: true,
		Errors:  []dashboard.PolicyValidationError{},
	}, nil
}