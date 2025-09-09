/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// PolicyValidationHandler handles policy validation requests
func (s *Server) PolicyValidationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	policyTypeID := vars["policyTypeId"]
	if policyTypeID == "" {
		http.Error(w, "Policy type ID is required", http.StatusBadRequest)
		return
	}

	var request PolicyInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	validationResult, err := policyManager.ValidatePolicyInstance(PolicyTypeID(policyTypeID), request.Policy)
	if err != nil {
		log.Printf("Failed to validate policy for type %s: %v", policyTypeID, err)
		http.Error(w, "Failed to validate policy", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(validationResult); err != nil {
		log.Printf("Failed to encode policy validation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// PolicyTypeValidationHandler handles policy type schema validation requests
func (s *Server) PolicyTypeValidationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	policyTypeID := vars["policyTypeId"]
	if policyTypeID == "" {
		http.Error(w, "Policy type ID is required", http.StatusBadRequest)
		return
	}

	var request PolicyTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	validationResult, err := policyManager.ValidatePolicyType(PolicyTypeID(policyTypeID), request.Schema)
	if err != nil {
		log.Printf("Failed to validate policy type schema %s: %v", policyTypeID, err)
		http.Error(w, "Failed to validate policy type schema", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(validationResult); err != nil {
		log.Printf("Failed to encode policy type validation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// PolicyConflictsHandler handles requests for policy conflicts
func (s *Server) PolicyConflictsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	conflicts := policyManager.GetPolicyConflicts()

	response := struct {
		Conflicts map[string]*PolicyConflict `json:"conflicts"`
		Total     int                        `json:"total"`
	}{
		Conflicts: conflicts,
		Total:     len(conflicts),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode policy conflicts response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// PolicyConflictResolutionHandler handles policy conflict resolution requests
func (s *Server) PolicyConflictResolutionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	conflictID := vars["conflictId"]
	if conflictID == "" {
		http.Error(w, "Conflict ID is required", http.StatusBadRequest)
		return
	}

	var request struct {
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Resolution == "" {
		http.Error(w, "Resolution is required", http.StatusBadRequest)
		return
	}

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	if err := policyManager.ResolveConflict(conflictID, request.Resolution); err != nil {
		log.Printf("Failed to resolve conflict %s: %v", conflictID, err)
		if err.Error() == "conflict "+conflictID+" not found" {
			http.Error(w, "Conflict not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to resolve conflict", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"status":     "success",
		"message":    "Conflict resolved successfully",
		"conflictId": conflictID,
		"resolution": request.Resolution,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode conflict resolution response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// PolicyDistributionStatusHandler handles requests for policy distribution status
func (s *Server) PolicyDistributionStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	policyInstanceID := vars["policyInstanceId"]
	if policyInstanceID == "" {
		http.Error(w, "Policy instance ID is required", http.StatusBadRequest)
		return
	}

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	distributionStatus := policyManager.GetPolicyDistributionStatus(PolicyInstanceID(policyInstanceID))

	response := struct {
		PolicyInstanceID   PolicyInstanceID                        `json:"policy_instance_id"`
		DistributionStatus map[string]*PolicyDistributionStatus    `json:"distribution_status"`
		TotalXApps         int                                     `json:"total_xapps"`
		SuccessfulXApps    int                                     `json:"successful_xapps"`
		FailedXApps        int                                     `json:"failed_xapps"`
		PendingXApps       int                                     `json:"pending_xapps"`
	}{
		PolicyInstanceID:   PolicyInstanceID(policyInstanceID),
		DistributionStatus: distributionStatus,
		TotalXApps:         len(distributionStatus),
	}

	// Count status types
	for _, status := range distributionStatus {
		switch status.Status {
		case string(PolicyDistributionStatusDeployed):
			response.SuccessfulXApps++
		case string(PolicyDistributionStatusFailed):
			response.FailedXApps++
		case string(PolicyDistributionStatusPending):
			response.PendingXApps++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode policy distribution status response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// PolicyComplianceReportsHandler handles requests for policy compliance reports
func (s *Server) PolicyComplianceReportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	policyInstanceID := vars["policyInstanceId"]
	if policyInstanceID == "" {
		http.Error(w, "Policy instance ID is required", http.StatusBadRequest)
		return
	}

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	complianceReports := policyManager.GetPolicyComplianceReports(PolicyInstanceID(policyInstanceID))

	response := struct {
		PolicyInstanceID  PolicyInstanceID                       `json:"policy_instance_id"`
		ComplianceReports map[string]*PolicyComplianceReport     `json:"compliance_reports"`
		TotalXApps        int                                    `json:"total_xapps"`
		CompliantXApps    int                                    `json:"compliant_xapps"`
		NonCompliantXApps int                                    `json:"non_compliant_xapps"`
		UnknownXApps      int                                    `json:"unknown_xapps"`
	}{
		PolicyInstanceID:  PolicyInstanceID(policyInstanceID),
		ComplianceReports: complianceReports,
		TotalXApps:        len(complianceReports),
	}

	// Count compliance status types
	for _, report := range complianceReports {
		switch report.ComplianceStatus {
		case string(PolicyComplianceStatusCompliant):
			response.CompliantXApps++
		case string(PolicyComplianceStatusNonCompliant):
			response.NonCompliantXApps++
		case string(PolicyComplianceStatusUnknown):
			response.UnknownXApps++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode policy compliance reports response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// XAppRegistrationHandler handles xApp registration for policy distribution
func (s *Server) XAppRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleXAppRegistration(w, r)
	case http.MethodGet:
		s.handleGetRegisteredXApps(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleXAppRegistration handles POST requests to register an xApp
func (s *Server) handleXAppRegistration(w http.ResponseWriter, r *http.Request) {
	var request struct {
		XAppID   string `json:"xapp_id"`
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.XAppID == "" {
		http.Error(w, "xApp ID is required", http.StatusBadRequest)
		return
	}

	if request.Endpoint == "" {
		http.Error(w, "Endpoint is required", http.StatusBadRequest)
		return
	}

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	policyManager.RegisterXApp(request.XAppID, request.Endpoint)

	response := map[string]interface{}{
		"status":   "success",
		"message":  "xApp registered successfully",
		"xappId":   request.XAppID,
		"endpoint": request.Endpoint,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode xApp registration response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleGetRegisteredXApps handles GET requests to list registered xApps
func (s *Server) handleGetRegisteredXApps(w http.ResponseWriter, r *http.Request) {
	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	xapps := policyManager.GetRegisteredXApps()

	response := struct {
		XApps []string `json:"xapps"`
		Total int      `json:"total"`
	}{
		XApps: xapps,
		Total: len(xapps),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode registered xApps response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// XAppUnregistrationHandler handles xApp unregistration
func (s *Server) XAppUnregistrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	xappID := vars["xappId"]
	if xappID == "" {
		http.Error(w, "xApp ID is required", http.StatusBadRequest)
		return
	}

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		http.Error(w, "Policy manager not available", http.StatusServiceUnavailable)
		return
	}

	policyManager.UnregisterXApp(xappID)

	response := map[string]interface{}{
		"status":  "success",
		"message": "xApp unregistered successfully",
		"xappId":  xappID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode xApp unregistration response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Enhanced A1PolicyInstanceHandler with policy manager integration
func (s *Server) EnhancedA1PolicyInstanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyTypeID := vars["policyTypeId"]
	policyInstanceID := vars["policyInstanceId"]
	
	if policyTypeID == "" {
		http.Error(w, "Policy type ID is required", http.StatusBadRequest)
		return
	}
	
	if policyInstanceID == "" {
		http.Error(w, "Policy instance ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	policyManager := s.getPolicyManager()
	if policyManager == nil {
		// Fall back to basic A1 handler
		s.A1PolicyInstanceHandler(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.A1PolicyInstanceHandler(w, r) // Use existing handler for GET
	case http.MethodPut:
		s.handleEnhancedCreateOrUpdatePolicyInstance(w, r, ctx, policyManager, PolicyTypeID(policyTypeID), PolicyInstanceID(policyInstanceID))
	case http.MethodDelete:
		s.handleEnhancedDeletePolicyInstance(w, r, ctx, policyManager, PolicyTypeID(policyTypeID), PolicyInstanceID(policyInstanceID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleEnhancedCreateOrUpdatePolicyInstance handles PUT requests with policy manager integration
func (s *Server) handleEnhancedCreateOrUpdatePolicyInstance(w http.ResponseWriter, r *http.Request, ctx context.Context, policyManager *PolicyManager, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) {
	var request PolicyInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if policy instance already exists
	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	_, err := a1Client.GetPolicyInstance(ctx, policyTypeID, policyInstanceID)
	isUpdate := err == nil

	if isUpdate {
		// Update existing policy instance using policy manager
		if err := policyManager.UpdatePolicyInstance(ctx, policyTypeID, policyInstanceID, request.Policy); err != nil {
			log.Printf("Failed to update policy instance %s: %v", policyInstanceID, err)
			http.Error(w, "Failed to update policy instance", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"status":           "success",
			"message":          "Policy instance updated successfully with enhanced management",
			"policyTypeId":     policyTypeID,
			"policyInstanceId": policyInstanceID,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode policy instance update response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	} else {
		// Create new policy instance using policy manager
		if err := policyManager.CreatePolicyInstance(ctx, policyTypeID, policyInstanceID, request.Policy); err != nil {
			log.Printf("Failed to create policy instance %s: %v", policyInstanceID, err)
			http.Error(w, "Failed to create policy instance", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"status":           "success",
			"message":          "Policy instance created successfully with enhanced management",
			"policyTypeId":     policyTypeID,
			"policyInstanceId": policyInstanceID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode policy instance creation response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleEnhancedDeletePolicyInstance handles DELETE requests with policy manager integration
func (s *Server) handleEnhancedDeletePolicyInstance(w http.ResponseWriter, r *http.Request, ctx context.Context, policyManager *PolicyManager, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) {
	if err := policyManager.DeletePolicyInstance(ctx, policyTypeID, policyInstanceID); err != nil {
		log.Printf("Failed to delete policy instance %s: %v", policyInstanceID, err)
		if err.Error() == "policy instance "+string(policyInstanceID)+" not found" {
			http.Error(w, "Policy instance not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete policy instance", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"status":           "success",
		"message":          "Policy instance deleted successfully with enhanced management",
		"policyTypeId":     policyTypeID,
		"policyInstanceId": policyInstanceID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode policy instance deletion response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getPolicyManager returns the policy manager instance
func (s *Server) getPolicyManager() *PolicyManager {
	return s.policyManager
}