/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

// PolicyManager type is now defined in types.go to avoid redeclaration

// PolicyDistributionRequest type is now defined in types.go to avoid redeclaration

// PolicyComplianceRequest type is now defined in types.go to avoid redeclaration

// XAppClient represents a client for communicating with xApps
// XAppClient is now defined in types.go to avoid redeclaration

// NewPolicyManager function is now defined in types.go to avoid redeclaration

// Stop stops the policy manager and its background workers
func (pm *PolicyManager) Stop() {
	close(pm.stopChan)
}

// ValidatePolicyType validates a policy type schema using JSON Schema validation
func (pm *PolicyManager) ValidatePolicyType(policyTypeID PolicyTypeID, schema json.RawMessage) (*PolicyValidationResult, error) {
	// Load the schema
	schemaLoader := gojsonschema.NewBytesLoader(schema)

	// Validate that the schema itself is valid JSON Schema
	_, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return &PolicyValidationResult{
			IsValid: false,
			Errors: []PolicyValidationError{
				{
					Field:   "schema",
					Message: fmt.Sprintf("Invalid JSON Schema: %v", err),
					Value:   string(schema),
				},
			},
		}, nil
	}

	return &PolicyValidationResult{
		IsValid: true,
		Errors:  []PolicyValidationError{},
	}, nil
}

// ValidatePolicyInstance validates a policy instance against its type schema
func (pm *PolicyManager) ValidatePolicyInstance(policyTypeID PolicyTypeID, policy json.RawMessage) (*PolicyValidationResult, error) {
	pm.mutex.RLock()
	policyType, exists := pm.policyTypes[policyTypeID]
	pm.mutex.RUnlock()

	if !exists {
		// Try to fetch from A1 Mediator
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		policyType, err = pm.a1Client.GetPolicyType(ctx, policyTypeID)
		if err != nil {
			return &PolicyValidationResult{
				IsValid: false,
				Errors: []PolicyValidationError{
					{
						Field:   "policyTypeId",
						Message: fmt.Sprintf("Policy type %s not found", policyTypeID),
						Value:   string(policyTypeID),
					},
				},
			}, nil
		}

		// Cache the policy type
		pm.mutex.Lock()
		pm.policyTypes[policyTypeID] = policyType
		pm.mutex.Unlock()
	}

	// Validate policy against schema
	schemaLoader := gojsonschema.NewBytesLoader(policyType.Schema)
	documentLoader := gojsonschema.NewBytesLoader(policy)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return &PolicyValidationResult{
			IsValid: false,
			Errors: []PolicyValidationError{
				{
					Field:   "policy",
					Message: fmt.Sprintf("Validation error: %v", err),
					Value:   string(policy),
				},
			},
		}, nil
	}

	if result.Valid() {
		return &PolicyValidationResult{
			IsValid: true,
			Errors:  []PolicyValidationError{},
		}, nil
	}

	// Convert validation errors
	var validationErrors []PolicyValidationError
	for _, desc := range result.Errors() {
		validationErrors = append(validationErrors, PolicyValidationError{
			Field:   desc.Field(),
			Message: desc.Description(),
			Value:   fmt.Sprintf("%v", desc.Value()),
		})
	}

	return &PolicyValidationResult{
		IsValid: false,
		Errors:  validationErrors,
	}, nil
}

// CreatePolicyInstance creates a new policy instance with validation and conflict detection
func (pm *PolicyManager) CreatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID, policy json.RawMessage) error {
	// Validate policy instance
	validationResult, err := pm.ValidatePolicyInstance(policyTypeID, policy)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if !validationResult.IsValid {
		return fmt.Errorf("policy validation failed: %v", validationResult.Errors)
	}

	// Check for conflicts
	conflicts, err := pm.DetectConflicts(policyTypeID, policyInstanceID, policy)
	if err != nil {
		return fmt.Errorf("conflict detection failed: %w", err)
	}

	// If there are conflicts, handle them
	if len(conflicts) > 0 {
		for _, conflict := range conflicts {
			log.Printf("Policy conflict detected: %s", conflict.Description)

			// Store conflict for resolution
			pm.mutex.Lock()
			pm.conflicts[conflict.ConflictID] = conflict
			pm.mutex.Unlock()
		}

		// For now, we'll allow creation but log conflicts
		// In a production system, you might want to reject or require explicit resolution
		log.Printf("Creating policy instance %s with %d conflicts", policyInstanceID, len(conflicts))
	}

	// Create policy instance via A1 Mediator
	request := &PolicyInstanceRequest{Policy: policy}
	if err := pm.a1Client.CreatePolicyInstance(ctx, policyTypeID, policyInstanceID, request); err != nil {
		return fmt.Errorf("failed to create policy instance: %w", err)
	}

	// Cache policy instance
	policyInstance := &PolicyInstance{
		ID:        policyInstanceID,
		TypeID:    policyTypeID,
		Policy:    policy,
		Status:    PolicyStatus{Status: string(PolicyInstanceStatusActive), LastUpdate: time.Now()},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	pm.mutex.Lock()
	pm.policyInstances[policyInstanceID] = policyInstance
	pm.mutex.Unlock()

	// Initiate distribution to xApps
	pm.distributionChan <- &PolicyDistributionRequest{
		PolicyInstanceID: policyInstanceID,
		PolicyTypeID:     policyTypeID,
		Policy:           policy,
		TargetXApps:      pm.getTargetXApps(policyTypeID),
	}

	return nil
}

// UpdatePolicyInstance updates an existing policy instance
func (pm *PolicyManager) UpdatePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID, policy json.RawMessage) error {
	// Validate policy instance
	validationResult, err := pm.ValidatePolicyInstance(policyTypeID, policy)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if !validationResult.IsValid {
		return fmt.Errorf("policy validation failed: %v", validationResult.Errors)
	}

	// Check for conflicts
	conflicts, err := pm.DetectConflicts(policyTypeID, policyInstanceID, policy)
	if err != nil {
		return fmt.Errorf("conflict detection failed: %w", err)
	}

	// Handle conflicts
	if len(conflicts) > 0 {
		for _, conflict := range conflicts {
			log.Printf("Policy conflict detected during update: %s", conflict.Description)

			pm.mutex.Lock()
			pm.conflicts[conflict.ConflictID] = conflict
			pm.mutex.Unlock()
		}
	}

	// Update policy instance via A1 Mediator
	update := &PolicyInstanceUpdate{
		PolicyInstanceID: policyInstanceID,
		Policy:           policy,
	}

	if err := pm.a1Client.UpdatePolicyInstance(ctx, policyTypeID, update); err != nil {
		return fmt.Errorf("failed to update policy instance: %w", err)
	}

	// Update cached policy instance
	pm.mutex.Lock()
	if policyInstance, exists := pm.policyInstances[policyInstanceID]; exists {
		policyInstance.Policy = policy
		policyInstance.UpdatedAt = time.Now()
		policyInstance.Status.LastUpdate = time.Now()
	}
	pm.mutex.Unlock()

	// Initiate redistribution to xApps
	pm.distributionChan <- &PolicyDistributionRequest{
		PolicyInstanceID: policyInstanceID,
		PolicyTypeID:     policyTypeID,
		Policy:           policy,
		TargetXApps:      pm.getTargetXApps(policyTypeID),
	}

	return nil
}

// DeletePolicyInstance deletes a policy instance
func (pm *PolicyManager) DeletePolicyInstance(ctx context.Context, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) error {
	// Delete from A1 Mediator
	if err := pm.a1Client.DeletePolicyInstance(ctx, policyTypeID, policyInstanceID); err != nil {
		return fmt.Errorf("failed to delete policy instance: %w", err)
	}

	// Remove from cache
	pm.mutex.Lock()
	delete(pm.policyInstances, policyInstanceID)
	delete(pm.distributionStatus, policyInstanceID)
	delete(pm.complianceReports, policyInstanceID)

	// Remove related conflicts
	for conflictID, conflict := range pm.conflicts {
		if conflict.PolicyInstanceID == policyInstanceID || conflict.ConflictingPolicyID == policyInstanceID {
			delete(pm.conflicts, conflictID)
		}
	}
	pm.mutex.Unlock()

	// Notify xApps about policy withdrawal
	pm.distributionChan <- &PolicyDistributionRequest{
		PolicyInstanceID: policyInstanceID,
		PolicyTypeID:     policyTypeID,
		Policy:           nil, // nil indicates withdrawal
		TargetXApps:      pm.getTargetXApps(policyTypeID),
	}

	return nil
}

// DetectConflicts detects conflicts between policies
func (pm *PolicyManager) DetectConflicts(policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID, policy json.RawMessage) ([]*PolicyConflict, error) {
	var conflicts []*PolicyConflict

	// Parse the policy to extract relevant fields for conflict detection
	var policyData map[string]interface{}
	if err := json.Unmarshal(policy, &policyData); err != nil {
		return nil, fmt.Errorf("failed to parse policy: %w", err)
	}

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Check against existing policies of the same type
	for existingID, existingInstance := range pm.policyInstances {
		if existingID == policyInstanceID || existingInstance.TypeID != policyTypeID {
			continue
		}

		var existingData map[string]interface{}
		if err := json.Unmarshal(existingInstance.Policy, &existingData); err != nil {
			log.Printf("Failed to parse existing policy %s: %v", existingID, err)
			continue
		}

		// Detect different types of conflicts
		if conflict := pm.detectResourceConflict(policyInstanceID, existingID, policyData, existingData); conflict != nil {
			conflicts = append(conflicts, conflict)
		}

		if conflict := pm.detectParameterConflict(policyInstanceID, existingID, policyData, existingData); conflict != nil {
			conflicts = append(conflicts, conflict)
		}

		if conflict := pm.detectPriorityConflict(policyInstanceID, existingID, policyData, existingData); conflict != nil {
			conflicts = append(conflicts, conflict)
		}

		if conflict := pm.detectExclusiveConflict(policyInstanceID, existingID, policyData, existingData); conflict != nil {
			conflicts = append(conflicts, conflict)
		}
	}

	return conflicts, nil
}

// detectResourceConflict detects resource-based conflicts
func (pm *PolicyManager) detectResourceConflict(policyID, existingID PolicyInstanceID, policy, existing map[string]interface{}) *PolicyConflict {
	// Check if policies target the same resources
	policyResources := pm.extractResources(policy)
	existingResources := pm.extractResources(existing)

	for _, resource := range policyResources {
		for _, existingResource := range existingResources {
			if resource == existingResource {
				return &PolicyConflict{
					ConflictID:          fmt.Sprintf("resource-%s-%s", policyID, existingID),
					PolicyInstanceID:    policyID,
					ConflictingPolicyID: existingID,
					ConflictType:        string(PolicyConflictTypeResource),
					Description:         fmt.Sprintf("Resource conflict on %s", resource),
					DetectedAt:          time.Now(),
				}
			}
		}
	}

	return nil
}

// detectParameterConflict detects parameter-based conflicts
func (pm *PolicyManager) detectParameterConflict(policyID, existingID PolicyInstanceID, policy, existing map[string]interface{}) *PolicyConflict {
	// Check for conflicting parameter values
	if policyParams, ok := policy["parameters"].(map[string]interface{}); ok {
		if existingParams, ok := existing["parameters"].(map[string]interface{}); ok {
			for key, value := range policyParams {
				if existingValue, exists := existingParams[key]; exists {
					if pm.areParametersConflicting(value, existingValue) {
						return &PolicyConflict{
							ConflictID:          fmt.Sprintf("parameter-%s-%s-%s", policyID, existingID, key),
							PolicyInstanceID:    policyID,
							ConflictingPolicyID: existingID,
							ConflictType:        string(PolicyConflictTypeParameter),
							Description:         fmt.Sprintf("Parameter conflict on %s", key),
							DetectedAt:          time.Now(),
						}
					}
				}
			}
		}
	}

	return nil
}

// detectPriorityConflict detects priority-based conflicts
func (pm *PolicyManager) detectPriorityConflict(policyID, existingID PolicyInstanceID, policy, existing map[string]interface{}) *PolicyConflict {
	policyPriority := pm.extractPriority(policy)
	existingPriority := pm.extractPriority(existing)

	// Check if policies have the same priority but different actions
	if policyPriority == existingPriority && policyPriority > 0 {
		if !pm.areActionsCompatible(policy, existing) {
			return &PolicyConflict{
				ConflictID:          fmt.Sprintf("priority-%s-%s", policyID, existingID),
				PolicyInstanceID:    policyID,
				ConflictingPolicyID: existingID,
				ConflictType:        string(PolicyConflictTypePriority),
				Description:         fmt.Sprintf("Priority conflict at level %d", policyPriority),
				DetectedAt:          time.Now(),
			}
		}
	}

	return nil
}

// detectExclusiveConflict detects exclusive policy conflicts
func (pm *PolicyManager) detectExclusiveConflict(policyID, existingID PolicyInstanceID, policy, existing map[string]interface{}) *PolicyConflict {
	// Check if either policy is marked as exclusive
	if pm.isExclusive(policy) || pm.isExclusive(existing) {
		// Check if they target overlapping scopes
		if pm.haveScopeOverlap(policy, existing) {
			return &PolicyConflict{
				ConflictID:          fmt.Sprintf("exclusive-%s-%s", policyID, existingID),
				PolicyInstanceID:    policyID,
				ConflictingPolicyID: existingID,
				ConflictType:        string(PolicyConflictTypeExclusive),
				Description:         "Exclusive policy conflict",
				DetectedAt:          time.Now(),
			}
		}
	}

	return nil
}

// Helper methods for conflict detection
func (pm *PolicyManager) extractResources(policy map[string]interface{}) []string {
	var resources []string
	if scope, ok := policy["scope"].(map[string]interface{}); ok {
		if resourceList, ok := scope["resources"].([]interface{}); ok {
			for _, resource := range resourceList {
				if resourceStr, ok := resource.(string); ok {
					resources = append(resources, resourceStr)
				}
			}
		}
	}
	return resources
}

func (pm *PolicyManager) extractPriority(policy map[string]interface{}) int {
	if priority, ok := policy["priority"].(float64); ok {
		return int(priority)
	}
	return 0
}

func (pm *PolicyManager) areParametersConflicting(value1, value2 interface{}) bool {
	// Simple conflict detection - in practice, this would be more sophisticated
	return value1 != value2
}

func (pm *PolicyManager) areActionsCompatible(policy1, policy2 map[string]interface{}) bool {
	// Check if the actions in both policies are compatible
	action1 := pm.extractAction(policy1)
	action2 := pm.extractAction(policy2)
	return action1 == action2
}

func (pm *PolicyManager) extractAction(policy map[string]interface{}) string {
	if action, ok := policy["action"].(string); ok {
		return action
	}
	return ""
}

func (pm *PolicyManager) isExclusive(policy map[string]interface{}) bool {
	if exclusive, ok := policy["exclusive"].(bool); ok {
		return exclusive
	}
	return false
}

func (pm *PolicyManager) haveScopeOverlap(policy1, policy2 map[string]interface{}) bool {
	resources1 := pm.extractResources(policy1)
	resources2 := pm.extractResources(policy2)

	for _, r1 := range resources1 {
		for _, r2 := range resources2 {
			if r1 == r2 {
				return true
			}
		}
	}
	return false
}

// getTargetXApps returns the list of xApps that should receive this policy type
func (pm *PolicyManager) getTargetXApps(policyTypeID PolicyTypeID) []string {
	// In a real implementation, this would be based on xApp subscriptions or policy type metadata
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var targetXApps []string
	for xappID := range pm.xappClients {
		targetXApps = append(targetXApps, xappID)
	}

	return targetXApps
}

// distributionWorker handles policy distribution to xApps
func (pm *PolicyManager) distributionWorker() {
	for {
		select {
		case req := <-pm.distributionChan:
			pm.handlePolicyDistribution(req)
		case <-pm.stopChan:
			return
		}
	}
}

// handlePolicyDistribution distributes a policy to target xApps
func (pm *PolicyManager) handlePolicyDistribution(req *PolicyDistributionRequest) {
	pm.mutex.Lock()
	if pm.distributionStatus[req.PolicyInstanceID] == nil {
		pm.distributionStatus[req.PolicyInstanceID] = make(map[string]*PolicyDistributionStatus)
	}
	pm.mutex.Unlock()

	for _, xappID := range req.TargetXApps {
		status := &PolicyDistributionStatus{
			PolicyInstanceID: req.PolicyInstanceID,
			XAppID:           xappID,
			Status:           string(PolicyDistributionStatusPending),
			LastUpdate:       time.Now(),
		}

		pm.mutex.Lock()
		pm.distributionStatus[req.PolicyInstanceID][xappID] = status
		pm.mutex.Unlock()

		// Distribute policy to xApp
		go pm.distributeToXApp(req, xappID)
	}
}

// distributeToXApp distributes a policy to a specific xApp
func (pm *PolicyManager) distributeToXApp(req *PolicyDistributionRequest, xappID string) {
	pm.mutex.RLock()
	xappClient, exists := pm.xappClients[xappID]
	pm.mutex.RUnlock()

	if !exists {
		pm.updateDistributionStatus(req.PolicyInstanceID, xappID, string(PolicyDistributionStatusFailed), "xApp client not found")
		return
	}

	var err error
	if req.Policy == nil {
		// Policy withdrawal
		err = pm.withdrawPolicyFromXApp(xappClient, req.PolicyInstanceID)
	} else {
		// Policy deployment
		err = pm.deployPolicyToXApp(xappClient, req.PolicyTypeID, req.PolicyInstanceID, req.Policy)
	}

	if err != nil {
		pm.updateDistributionStatus(req.PolicyInstanceID, xappID, string(PolicyDistributionStatusFailed), err.Error())
		log.Printf("Failed to distribute policy %s to xApp %s: %v", req.PolicyInstanceID, xappID, err)
	} else {
		status := string(PolicyDistributionStatusDeployed)
		if req.Policy == nil {
			status = string(PolicyDistributionStatusWithdrawn)
		}
		pm.updateDistributionStatus(req.PolicyInstanceID, xappID, status, "Successfully distributed")

		// Schedule compliance check
		pm.complianceChan <- &PolicyComplianceRequest{
			PolicyInstanceID: req.PolicyInstanceID,
			XAppID:           xappID,
		}
	}
}

// deployPolicyToXApp deploys a policy to an xApp
func (pm *PolicyManager) deployPolicyToXApp(xappClient *XAppClient, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID, policy json.RawMessage) error {
	// In a real implementation, this would make HTTP/gRPC calls to the xApp
	// For now, we'll simulate the deployment
	log.Printf("Deploying policy %s (type %s) to xApp %s", policyInstanceID, policyTypeID, xappClient.ID)

	// Simulate deployment delay
	time.Sleep(100 * time.Millisecond)

	// Simulate occasional failures for testing
	if time.Now().UnixNano()%10 == 0 {
		return fmt.Errorf("simulated deployment failure")
	}

	return nil
}

// withdrawPolicyFromXApp withdraws a policy from an xApp
func (pm *PolicyManager) withdrawPolicyFromXApp(xappClient *XAppClient, policyInstanceID PolicyInstanceID) error {
	// In a real implementation, this would make HTTP/gRPC calls to the xApp
	log.Printf("Withdrawing policy %s from xApp %s", policyInstanceID, xappClient.ID)

	// Simulate withdrawal delay
	time.Sleep(50 * time.Millisecond)

	return nil
}

// updateDistributionStatus updates the distribution status for a policy-xApp pair
func (pm *PolicyManager) updateDistributionStatus(policyInstanceID PolicyInstanceID, xappID, status, message string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.distributionStatus[policyInstanceID] == nil {
		pm.distributionStatus[policyInstanceID] = make(map[string]*PolicyDistributionStatus)
	}

	if pm.distributionStatus[policyInstanceID][xappID] == nil {
		pm.distributionStatus[policyInstanceID][xappID] = &PolicyDistributionStatus{
			PolicyInstanceID: policyInstanceID,
			XAppID:           xappID,
		}
	}

	pm.distributionStatus[policyInstanceID][xappID].Status = status
	pm.distributionStatus[policyInstanceID][xappID].Message = message
	pm.distributionStatus[policyInstanceID][xappID].LastUpdate = time.Now()
}

// complianceWorker handles policy compliance monitoring
func (pm *PolicyManager) complianceWorker() {
	ticker := time.NewTicker(30 * time.Second) // Check compliance every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case req := <-pm.complianceChan:
			pm.checkPolicyCompliance(req)
		case <-ticker.C:
			pm.performPeriodicComplianceCheck()
		case <-pm.stopChan:
			return
		}
	}
}

// checkPolicyCompliance checks compliance for a specific policy-xApp pair
func (pm *PolicyManager) checkPolicyCompliance(req *PolicyComplianceRequest) {
	pm.mutex.RLock()
	xappClient, exists := pm.xappClients[req.XAppID]
	pm.mutex.RUnlock()

	if !exists {
		log.Printf("xApp client %s not found for compliance check", req.XAppID)
		return
	}

	// Perform compliance check
	complianceStatus, violations, err := pm.performComplianceCheck(xappClient, req.PolicyInstanceID)
	if err != nil {
		log.Printf("Failed to check compliance for policy %s on xApp %s: %v", req.PolicyInstanceID, req.XAppID, err)
		complianceStatus = string(PolicyComplianceStatusUnknown)
	}

	// Update compliance report
	report := &PolicyComplianceReport{
		PolicyInstanceID: req.PolicyInstanceID,
		XAppID:           req.XAppID,
		ComplianceStatus: complianceStatus,
		Violations:       violations,
		LastCheck:        time.Now(),
	}

	pm.mutex.Lock()
	if pm.complianceReports[req.PolicyInstanceID] == nil {
		pm.complianceReports[req.PolicyInstanceID] = make(map[string]*PolicyComplianceReport)
	}
	pm.complianceReports[req.PolicyInstanceID][req.XAppID] = report
	pm.mutex.Unlock()

	log.Printf("Compliance check for policy %s on xApp %s: %s", req.PolicyInstanceID, req.XAppID, complianceStatus)
}

// performComplianceCheck performs the actual compliance check with an xApp
func (pm *PolicyManager) performComplianceCheck(xappClient *XAppClient, policyInstanceID PolicyInstanceID) (string, []string, error) {
	// In a real implementation, this would query the xApp for compliance status
	// For now, we'll simulate compliance checking

	// Simulate check delay
	time.Sleep(50 * time.Millisecond)

	// Simulate different compliance outcomes
	rand := time.Now().UnixNano() % 100

	if rand < 80 {
		return string(PolicyComplianceStatusCompliant), []string{}, nil
	} else if rand < 95 {
		violations := []string{
			"Parameter threshold exceeded",
			"Resource utilization outside policy bounds",
		}
		return string(PolicyComplianceStatusNonCompliant), violations, nil
	} else {
		return string(PolicyComplianceStatusUnknown), []string{}, fmt.Errorf("compliance check failed")
	}
}

// performPeriodicComplianceCheck performs periodic compliance checks for all active policies
func (pm *PolicyManager) performPeriodicComplianceCheck() {
	pm.mutex.RLock()
	var requests []*PolicyComplianceRequest

	for policyInstanceID := range pm.distributionStatus {
		for xappID, status := range pm.distributionStatus[policyInstanceID] {
			if status.Status == string(PolicyDistributionStatusDeployed) {
				requests = append(requests, &PolicyComplianceRequest{
					PolicyInstanceID: policyInstanceID,
					XAppID:           xappID,
				})
			}
		}
	}
	pm.mutex.RUnlock()

	// Queue compliance checks
	for _, req := range requests {
		select {
		case pm.complianceChan <- req:
		default:
			// Channel full, skip this check
			log.Printf("Compliance check queue full, skipping check for policy %s on xApp %s", req.PolicyInstanceID, req.XAppID)
		}
	}
}

// GetPolicyDistributionStatus returns the distribution status for a policy
func (pm *PolicyManager) GetPolicyDistributionStatus(policyInstanceID PolicyInstanceID) map[string]*PolicyDistributionStatus {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	status := make(map[string]*PolicyDistributionStatus)
	if pm.distributionStatus[policyInstanceID] != nil {
		for xappID, s := range pm.distributionStatus[policyInstanceID] {
			status[xappID] = &PolicyDistributionStatus{
				PolicyInstanceID: s.PolicyInstanceID,
				XAppID:           s.XAppID,
				Status:           s.Status,
				Message:          s.Message,
				LastUpdate:       s.LastUpdate,
			}
		}
	}

	return status
}

// GetPolicyComplianceReports returns compliance reports for a policy
func (pm *PolicyManager) GetPolicyComplianceReports(policyInstanceID PolicyInstanceID) map[string]*PolicyComplianceReport {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	reports := make(map[string]*PolicyComplianceReport)
	if pm.complianceReports[policyInstanceID] != nil {
		for xappID, report := range pm.complianceReports[policyInstanceID] {
			reports[xappID] = &PolicyComplianceReport{
				PolicyInstanceID: report.PolicyInstanceID,
				XAppID:           report.XAppID,
				ComplianceStatus: report.ComplianceStatus,
				Violations:       append([]string{}, report.Violations...),
				LastCheck:        report.LastCheck,
			}
		}
	}

	return reports
}

// GetPolicyConflicts returns all detected policy conflicts
func (pm *PolicyManager) GetPolicyConflicts() map[string]*PolicyConflict {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	conflicts := make(map[string]*PolicyConflict)
	for id, conflict := range pm.conflicts {
		conflicts[id] = &PolicyConflict{
			ConflictID:          conflict.ConflictID,
			PolicyInstanceID:    conflict.PolicyInstanceID,
			ConflictingPolicyID: conflict.ConflictingPolicyID,
			ConflictType:        conflict.ConflictType,
			Description:         conflict.Description,
			Resolution:          conflict.Resolution,
			DetectedAt:          conflict.DetectedAt,
		}
	}

	return conflicts
}

// ResolveConflict resolves a policy conflict
func (pm *PolicyManager) ResolveConflict(conflictID, resolution string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	conflict, exists := pm.conflicts[conflictID]
	if !exists {
		return fmt.Errorf("conflict %s not found", conflictID)
	}

	conflict.Resolution = resolution
	log.Printf("Resolved conflict %s: %s", conflictID, resolution)

	return nil
}

// RegisterXApp registers an xApp client for policy distribution
func (pm *PolicyManager) RegisterXApp(xappID, endpoint string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.xappClients[xappID] = &XAppClient{
		ID:       xappID,
		Endpoint: endpoint,
	}

	log.Printf("Registered xApp %s at endpoint %s", xappID, endpoint)
}

// UnregisterXApp unregisters an xApp client
func (pm *PolicyManager) UnregisterXApp(xappID string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	delete(pm.xappClients, xappID)

	// Clean up distribution status and compliance reports
	for policyInstanceID := range pm.distributionStatus {
		delete(pm.distributionStatus[policyInstanceID], xappID)
	}

	for policyInstanceID := range pm.complianceReports {
		delete(pm.complianceReports[policyInstanceID], xappID)
	}

	log.Printf("Unregistered xApp %s", xappID)
}

// GetRegisteredXApps returns the list of registered xApps
func (pm *PolicyManager) GetRegisteredXApps() []string {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var xapps []string
	for xappID := range pm.xappClients {
		xapps = append(xapps, xappID)
	}

	return xapps
}
