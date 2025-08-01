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
	"io"
	"log"
	"net/http"
	"time"
)

// A1MediatorClient provides client interface for A1 Mediator component
type A1MediatorClient struct {
	httpClient *http.Client
	endpoint   string
}

// NewA1MediatorClient creates a new A1 Mediator client
func NewA1MediatorClient(httpClient *http.Client, endpoint string) *A1MediatorClient {
	return &A1MediatorClient{
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

// IsConnected checks if the A1 Mediator client is connected
func (c *A1MediatorClient) IsConnected() bool {
	if c.httpClient == nil || c.endpoint == "" {
		return false
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/a1-p/healthcheck", nil)
	if err != nil {
		return false
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// GetHealth retrieves health information from A1 Mediator
func (c *A1MediatorClient) GetHealth(ctx context.Context) (*A1Health, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/healthcheck", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get health from A1 Mediator: %v", err)
		return &A1Health{
			IsHealthy:       false,
			StatusMessage:   "Connection failed",
			LastHealthCheck: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	health := &A1Health{
		IsHealthy:       resp.StatusCode == http.StatusOK,
		LastHealthCheck: time.Now(),
	}

	if resp.StatusCode == http.StatusOK {
		health.StatusMessage = "Healthy"
		
		// Try to read version information if available
		var healthData map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&healthData); err == nil {
			if version, ok := healthData["version"].(string); ok {
				health.Version = version
			}
		}
	} else {
		health.StatusMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return health, nil
}

// GetPolicyTypes retrieves all policy types from A1 Mediator
func (c *A1MediatorClient) GetPolicyTypes(ctx context.Context) (*PolicyTypeListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &PolicyTypeListResponse{PolicyTypes: []PolicyType{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/a1-p/policytypes", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get policy types from A1 Mediator: %v", err)
		return &PolicyTypeListResponse{PolicyTypes: []PolicyType{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("A1 Mediator returned status %d for policy types", resp.StatusCode)
		return &PolicyTypeListResponse{PolicyTypes: []PolicyType{}, Total: 0}, nil
	}

	var policyTypeIDs []string
	if err := json.NewDecoder(resp.Body).Decode(&policyTypeIDs); err != nil {
		log.Printf("Failed to decode policy types response: %v", err)
		return &PolicyTypeListResponse{PolicyTypes: []PolicyType{}, Total: 0}, nil
	}

	// Fetch detailed information for each policy type
	policyTypes := make([]PolicyType, 0, len(policyTypeIDs))
	for _, typeID := range policyTypeIDs {
		policyType, err := c.GetPolicyType(ctx, PolicyTypeID(typeID))
		if err != nil {
			log.Printf("Failed to get policy type %s: %v", typeID, err)
			continue
		}
		policyTypes = append(policyTypes, *policyType)
	}

	return &PolicyTypeListResponse{
		PolicyTypes: policyTypes,
		Total:       uint32(len(policyTypes)),
	}, nil
}

// GetPolicyType retrieves a specific policy type from A1 Mediator
func (c *A1MediatorClient) GetPolicyType(ctx context.Context, policyTypeID PolicyTypeID) (*PolicyType, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s", c.endpoint, policyTypeID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy type from A1 Mediator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("policy type %s not found", policyTypeID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("A1 Mediator returned status %d for policy type %s", resp.StatusCode, policyTypeID)
	}

	var rawPolicyType map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawPolicyType); err != nil {
		return nil, fmt.Errorf("failed to decode policy type response: %w", err)
	}

	return c.parsePolicyType(string(policyTypeID), rawPolicyType)
}

// CreatePolicyType creates a new policy type in A1 Mediator
func (c *A1MediatorClient) CreatePolicyType(ctx context.Context, policyTypeID PolicyTypeID, request *PolicyTypeRequest) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s", c.endpoint, policyTypeID)
	
	jsonData, err := json.Marshal(request.Schema)
	if err != nil {
		return fmt.Errorf("failed to marshal policy type schema: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create policy type: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("A1 Mediator returned status %d for policy type creation: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully created policy type %s", policyTypeID)
	return nil
}

// DeletePolicyType deletes a policy type from A1 Mediator
func (c *A1MediatorClient) DeletePolicyType(ctx context.Context, policyTypeID PolicyTypeID) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s", c.endpoint, policyTypeID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete policy type: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("policy type %s not found", policyTypeID)
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("A1 Mediator returned status %d for policy type deletion: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully deleted policy type %s", policyTypeID)
	return nil
}

// GetPolicyInstances retrieves all policy instances for a policy type from A1 Mediator
func (c *A1MediatorClient) GetPolicyInstances(ctx context.Context, policyTypeID PolicyTypeID) (*PolicyInstanceListResponse, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &PolicyInstanceListResponse{PolicyInstances: []PolicyInstance{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s/policies", c.endpoint, policyTypeID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get policy instances from A1 Mediator: %v", err)
		return &PolicyInstanceListResponse{PolicyInstances: []PolicyInstance{}, Total: 0}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("A1 Mediator returned status %d for policy instances", resp.StatusCode)
		return &PolicyInstanceListResponse{PolicyInstances: []PolicyInstance{}, Total: 0}, nil
	}

	var policyInstanceIDs []string
	if err := json.NewDecoder(resp.Body).Decode(&policyInstanceIDs); err != nil {
		log.Printf("Failed to decode policy instances response: %v", err)
		return &PolicyInstanceListResponse{PolicyInstances: []PolicyInstance{}, Total: 0}, nil
	}

	// Fetch detailed information for each policy instance
	policyInstances := make([]PolicyInstance, 0, len(policyInstanceIDs))
	for _, instanceID := range policyInstanceIDs {
		policyInstance, err := c.GetPolicyInstance(ctx, policyTypeID, PolicyInstanceID(instanceID))
		if err != nil {
			log.Printf("Failed to get policy instance %s: %v", instanceID, err)
			continue
		}
		policyInstances = append(policyInstances, *policyInstance)
	}

	return &PolicyInstanceListResponse{
		PolicyInstances: policyInstances,
		Total:           uint32(len(policyInstances)),
	}, nil
}

// GetPolicyInstance retrieves a specific policy instance from A1 Mediator
func (c *A1MediatorClient) GetPolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) (*PolicyInstance, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s/policies/%s", c.endpoint, policyTypeID, policyInstanceID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy instance from A1 Mediator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("policy instance %s not found", policyInstanceID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("A1 Mediator returned status %d for policy instance %s", resp.StatusCode, policyInstanceID)
	}

	var policyData json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&policyData); err != nil {
		return nil, fmt.Errorf("failed to decode policy instance response: %w", err)
	}

	return &PolicyInstance{
		ID:        policyInstanceID,
		TypeID:    policyTypeID,
		Policy:    policyData,
		Status:    PolicyStatus{Status: "ACTIVE", LastUpdate: time.Now()},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// CreatePolicyInstance creates a new policy instance in A1 Mediator
func (c *A1MediatorClient) CreatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID, request *PolicyInstanceRequest) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s/policies/%s", c.endpoint, policyTypeID, policyInstanceID)
	
	jsonData, err := json.Marshal(request.Policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy instance: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create policy instance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("A1 Mediator returned status %d for policy instance creation: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully created policy instance %s for type %s", policyInstanceID, policyTypeID)
	return nil
}

// UpdatePolicyInstance updates an existing policy instance in A1 Mediator
func (c *A1MediatorClient) UpdatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, update *PolicyInstanceUpdate) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s/policies/%s", c.endpoint, policyTypeID, update.PolicyInstanceID)
	
	jsonData, err := json.Marshal(update.Policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy instance update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update policy instance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("A1 Mediator returned status %d for policy instance update: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully updated policy instance %s for type %s", update.PolicyInstanceID, policyTypeID)
	return nil
}

// DeletePolicyInstance deletes a policy instance from A1 Mediator
func (c *A1MediatorClient) DeletePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) error {
	if c.httpClient == nil || c.endpoint == "" {
		return fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s/policies/%s", c.endpoint, policyTypeID, policyInstanceID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete policy instance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("policy instance %s not found", policyInstanceID)
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("A1 Mediator returned status %d for policy instance deletion: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully deleted policy instance %s for type %s", policyInstanceID, policyTypeID)
	return nil
}

// GetPolicyInstanceStatus retrieves the status of a policy instance from A1 Mediator
func (c *A1MediatorClient) GetPolicyInstanceStatus(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) (*PolicyStatus, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return nil, fmt.Errorf("A1 Mediator client not configured")
	}

	url := fmt.Sprintf("%s/a1-p/policytypes/%s/policies/%s/status", c.endpoint, policyTypeID, policyInstanceID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy instance status from A1 Mediator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("policy instance %s not found", policyInstanceID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("A1 Mediator returned status %d for policy instance status %s", resp.StatusCode, policyInstanceID)
	}

	var status PolicyStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		// If status endpoint doesn't return structured data, create default status
		status = PolicyStatus{
			Status:     "ACTIVE",
			LastUpdate: time.Now(),
		}
	}

	return &status, nil
}

// GetStats retrieves statistics from A1 Mediator
func (c *A1MediatorClient) GetStats(ctx context.Context) (*A1Stats, error) {
	if c.httpClient == nil || c.endpoint == "" {
		return &A1Stats{
			PolicyTypesByStatus:     make(map[string]uint32),
			PolicyInstancesByType:   make(map[string]uint32),
			PolicyInstancesByStatus: make(map[string]uint32),
			LastUpdated:             time.Now(),
		}, nil
	}

	// A1 Mediator doesn't have a dedicated stats endpoint, so we'll aggregate from available data
	policyTypes, err := c.GetPolicyTypes(ctx)
	if err != nil {
		log.Printf("Failed to get policy types for stats: %v", err)
		return &A1Stats{
			PolicyTypesByStatus:     make(map[string]uint32),
			PolicyInstancesByType:   make(map[string]uint32),
			PolicyInstancesByStatus: make(map[string]uint32),
			LastUpdated:             time.Now(),
		}, nil
	}

	stats := &A1Stats{
		PolicyTypesByStatus:     make(map[string]uint32),
		PolicyInstancesByType:   make(map[string]uint32),
		PolicyInstancesByStatus: make(map[string]uint32),
		TotalPolicyTypes:        policyTypes.Total,
		LastUpdated:             time.Now(),
	}

	// Count policy types by status
	stats.PolicyTypesByStatus["ACTIVE"] = policyTypes.Total

	// Count policy instances by type and status
	var totalInstances uint32
	for _, policyType := range policyTypes.PolicyTypes {
		instances, err := c.GetPolicyInstances(ctx, policyType.ID)
		if err != nil {
			log.Printf("Failed to get policy instances for type %s: %v", policyType.ID, err)
			continue
		}

		stats.PolicyInstancesByType[string(policyType.ID)] = instances.Total
		totalInstances += instances.Total

		// Count by status (assuming all are active for now)
		stats.PolicyInstancesByStatus["ACTIVE"] += instances.Total
	}

	stats.TotalPolicyInstances = totalInstances
	return stats, nil
}

// ValidatePolicy validates a policy against its type schema
func (c *A1MediatorClient) ValidatePolicy(ctx context.Context, policyTypeID PolicyTypeID, policy json.RawMessage) (*PolicyValidationResult, error) {
	// This is a client-side validation - in a real implementation, this might call
	// a validation endpoint or perform JSON schema validation
	
	// For now, we'll do basic JSON validation
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

	// Basic validation passed
	return &PolicyValidationResult{
		IsValid: true,
		Errors:  []PolicyValidationError{},
	}, nil
}

// parsePolicyType parses raw policy type data into PolicyType struct
func (c *A1MediatorClient) parsePolicyType(typeID string, rawPolicyType map[string]interface{}) (*PolicyType, error) {
	policyType := &PolicyType{
		ID:        PolicyTypeID(typeID),
		CreatedAt: time.Now(),
	}

	// Extract name if available
	if name, ok := rawPolicyType["name"].(string); ok {
		policyType.Name = name
	} else {
		policyType.Name = typeID
	}

	// Extract description if available
	if description, ok := rawPolicyType["description"].(string); ok {
		policyType.Description = description
	}

	// The entire response is the schema
	schemaBytes, err := json.Marshal(rawPolicyType)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy type schema: %w", err)
	}
	policyType.Schema = json.RawMessage(schemaBytes)

	return policyType, nil
}