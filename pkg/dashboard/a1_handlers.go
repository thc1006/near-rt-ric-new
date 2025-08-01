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

// A1HealthHandler handles requests for A1 Mediator health
func (s *Server) A1HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	health, err := a1Client.GetHealth(ctx)
	if err != nil {
		log.Printf("Failed to get A1 Mediator health: %v", err)
		http.Error(w, "Failed to retrieve A1 Mediator health", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Failed to encode A1 health response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// A1PolicyTypesHandler handles requests for A1 policy types
func (s *Server) A1PolicyTypesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPolicyTypes(w, r, ctx, a1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPolicyTypes handles GET requests for policy types
func (s *Server) handleGetPolicyTypes(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient) {
	policyTypes, err := a1Client.GetPolicyTypes(ctx)
	if err != nil {
		log.Printf("Failed to get policy types: %v", err)
		http.Error(w, "Failed to retrieve policy types", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policyTypes); err != nil {
		log.Printf("Failed to encode policy types response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// A1PolicyTypeHandler handles requests for a specific A1 policy type
func (s *Server) A1PolicyTypeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyTypeID := vars["policyTypeId"]
	if policyTypeID == "" {
		http.Error(w, "Policy type ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPolicyType(w, r, ctx, a1Client, PolicyTypeID(policyTypeID))
	case http.MethodPost:
		s.handleCreatePolicyType(w, r, ctx, a1Client, PolicyTypeID(policyTypeID))
	case http.MethodDelete:
		s.handleDeletePolicyType(w, r, ctx, a1Client, PolicyTypeID(policyTypeID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPolicyType handles GET requests for a specific policy type
func (s *Server) handleGetPolicyType(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient, policyTypeID PolicyTypeID) {
	policyType, err := a1Client.GetPolicyType(ctx, policyTypeID)
	if err != nil {
		log.Printf("Failed to get policy type %s: %v", policyTypeID, err)
		if err.Error() == "policy type "+string(policyTypeID)+" not found" {
			http.Error(w, "Policy type not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve policy type", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policyType); err != nil {
		log.Printf("Failed to encode policy type response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCreatePolicyType handles POST requests to create a policy type
func (s *Server) handleCreatePolicyType(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient, policyTypeID PolicyTypeID) {
	var request PolicyTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := a1Client.CreatePolicyType(ctx, policyTypeID, &request); err != nil {
		log.Printf("Failed to create policy type %s: %v", policyTypeID, err)
		http.Error(w, "Failed to create policy type", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]interface{}{
		"status":         "success",
		"message":        "Policy type created successfully",
		"policyTypeId":   policyTypeID,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode policy type creation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleDeletePolicyType handles DELETE requests for a policy type
func (s *Server) handleDeletePolicyType(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient, policyTypeID PolicyTypeID) {
	if err := a1Client.DeletePolicyType(ctx, policyTypeID); err != nil {
		log.Printf("Failed to delete policy type %s: %v", policyTypeID, err)
		if err.Error() == "policy type "+string(policyTypeID)+" not found" {
			http.Error(w, "Policy type not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete policy type", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":         "success",
		"message":        "Policy type deleted successfully",
		"policyTypeId":   policyTypeID,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode policy type deletion response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// A1PolicyInstancesHandler handles requests for A1 policy instances
func (s *Server) A1PolicyInstancesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyTypeID := vars["policyTypeId"]
	if policyTypeID == "" {
		http.Error(w, "Policy type ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPolicyInstances(w, r, ctx, a1Client, PolicyTypeID(policyTypeID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPolicyInstances handles GET requests for policy instances
func (s *Server) handleGetPolicyInstances(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient, policyTypeID PolicyTypeID) {
	policyInstances, err := a1Client.GetPolicyInstances(ctx, policyTypeID)
	if err != nil {
		log.Printf("Failed to get policy instances for type %s: %v", policyTypeID, err)
		http.Error(w, "Failed to retrieve policy instances", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policyInstances); err != nil {
		log.Printf("Failed to encode policy instances response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// A1PolicyInstanceHandler handles requests for a specific A1 policy instance
func (s *Server) A1PolicyInstanceHandler(w http.ResponseWriter, r *http.Request) {
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

	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPolicyInstance(w, r, ctx, a1Client, PolicyTypeID(policyTypeID), PolicyInstanceID(policyInstanceID))
	case http.MethodPut:
		s.handleCreateOrUpdatePolicyInstance(w, r, ctx, a1Client, PolicyTypeID(policyTypeID), PolicyInstanceID(policyInstanceID))
	case http.MethodDelete:
		s.handleDeletePolicyInstance(w, r, ctx, a1Client, PolicyTypeID(policyTypeID), PolicyInstanceID(policyInstanceID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPolicyInstance handles GET requests for a specific policy instance
func (s *Server) handleGetPolicyInstance(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) {
	policyInstance, err := a1Client.GetPolicyInstance(ctx, policyTypeID, policyInstanceID)
	if err != nil {
		log.Printf("Failed to get policy instance %s: %v", policyInstanceID, err)
		if err.Error() == "policy instance "+string(policyInstanceID)+" not found" {
			http.Error(w, "Policy instance not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve policy instance", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policyInstance); err != nil {
		log.Printf("Failed to encode policy instance response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCreateOrUpdatePolicyInstance handles PUT requests to create or update a policy instance
func (s *Server) handleCreateOrUpdatePolicyInstance(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) {
	var request PolicyInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if policy instance already exists to determine if this is create or update
	_, err := a1Client.GetPolicyInstance(ctx, policyTypeID, policyInstanceID)
	isUpdate := err == nil

	if isUpdate {
		// Update existing policy instance
		update := &PolicyInstanceUpdate{
			PolicyInstanceID: policyInstanceID,
			Policy:           request.Policy,
		}
		
		if err := a1Client.UpdatePolicyInstance(ctx, policyTypeID, update); err != nil {
			log.Printf("Failed to update policy instance %s: %v", policyInstanceID, err)
			http.Error(w, "Failed to update policy instance", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":           "success",
			"message":          "Policy instance updated successfully",
			"policyTypeId":     policyTypeID,
			"policyInstanceId": policyInstanceID,
		}
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode policy instance update response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	} else {
		// Create new policy instance
		if err := a1Client.CreatePolicyInstance(ctx, policyTypeID, policyInstanceID, &request); err != nil {
			log.Printf("Failed to create policy instance %s: %v", policyInstanceID, err)
			http.Error(w, "Failed to create policy instance", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := map[string]interface{}{
			"status":           "success",
			"message":          "Policy instance created successfully",
			"policyTypeId":     policyTypeID,
			"policyInstanceId": policyInstanceID,
		}
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode policy instance creation response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// handleDeletePolicyInstance handles DELETE requests for a policy instance
func (s *Server) handleDeletePolicyInstance(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) {
	if err := a1Client.DeletePolicyInstance(ctx, policyTypeID, policyInstanceID); err != nil {
		log.Printf("Failed to delete policy instance %s: %v", policyInstanceID, err)
		if err.Error() == "policy instance "+string(policyInstanceID)+" not found" {
			http.Error(w, "Policy instance not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete policy instance", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":           "success",
		"message":          "Policy instance deleted successfully",
		"policyTypeId":     policyTypeID,
		"policyInstanceId": policyInstanceID,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode policy instance deletion response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// A1PolicyInstanceStatusHandler handles requests for A1 policy instance status
func (s *Server) A1PolicyInstanceStatusHandler(w http.ResponseWriter, r *http.Request) {
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPolicyInstanceStatus(w, r, ctx, a1Client, PolicyTypeID(policyTypeID), PolicyInstanceID(policyInstanceID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPolicyInstanceStatus handles GET requests for policy instance status
func (s *Server) handleGetPolicyInstanceStatus(w http.ResponseWriter, r *http.Request, ctx context.Context, a1Client *A1MediatorClient, policyTypeID PolicyTypeID, policyInstanceID PolicyInstanceID) {
	status, err := a1Client.GetPolicyInstanceStatus(ctx, policyTypeID, policyInstanceID)
	if err != nil {
		log.Printf("Failed to get policy instance status %s: %v", policyInstanceID, err)
		if err.Error() == "policy instance "+string(policyInstanceID)+" not found" {
			http.Error(w, "Policy instance not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve policy instance status", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode policy instance status response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// A1StatsHandler handles requests for A1 Mediator statistics
func (s *Server) A1StatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	stats, err := a1Client.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to get A1 Mediator stats: %v", err)
		http.Error(w, "Failed to retrieve A1 Mediator stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode A1 stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// A1PolicyValidationHandler handles requests for policy validation
func (s *Server) A1PolicyValidationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyTypeID := vars["policyTypeId"]
	if policyTypeID == "" {
		http.Error(w, "Policy type ID is required", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	a1Client := s.clients.GetA1MediatorClient()
	if a1Client == nil {
		http.Error(w, "A1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	var request PolicyInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	validationResult, err := a1Client.ValidatePolicy(ctx, PolicyTypeID(policyTypeID), request.Policy)
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