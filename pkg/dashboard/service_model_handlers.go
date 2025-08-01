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

// ServiceModelHandler handles requests for service models
func (s *Server) ServiceModelHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetServiceModels(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ServiceModelByOIDHandler handles requests for a specific service model
func (s *Server) ServiceModelByOIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	oid := vars["oid"]
	if oid == "" {
		http.Error(w, "Service model OID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetServiceModel(w, r, oid)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ServiceModelCapabilitiesHandler handles requests for service model capabilities
func (s *Server) ServiceModelCapabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	capabilities := s.serviceModelRegistry.GetSupportedCapabilities()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"capabilities": capabilities,
		"total":        len(capabilities),
		"timestamp":    time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode capabilities response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ServiceModelStatsHandler handles requests for service model statistics
func (s *Server) ServiceModelStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.serviceModelRegistry.GetServiceModelStats()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"stats":     stats,
		"timestamp": time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleGetServiceModels handles GET requests for all service models
func (s *Server) handleGetServiceModels(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	modelType := r.URL.Query().Get("type")

	var models []*ServiceModelDefinition
	if modelType != "" {
		models = s.serviceModelRegistry.GetServiceModelsByType(ServiceModelType(modelType))
	} else {
		allModels := s.serviceModelRegistry.GetAllServiceModels()
		models = make([]*ServiceModelDefinition, 0, len(allModels))
		for _, model := range allModels {
			models = append(models, model)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"serviceModels": models,
		"total":         len(models),
		"timestamp":     time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode service models response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleGetServiceModel handles GET requests for a specific service model
func (s *Server) handleGetServiceModel(w http.ResponseWriter, r *http.Request, oid string) {
	model, exists := s.serviceModelRegistry.GetServiceModel(oid)
	if !exists {
		http.Error(w, "Service model not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(model); err != nil {
		log.Printf("Failed to encode service model response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ProcessIndicationHandler handles processing of indication messages
func (s *Server) ProcessIndicationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ServiceModelOID string `json:"serviceModelOid"`
		Header          []byte `json:"header"`
		Message         []byte `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.ServiceModelOID == "" {
		http.Error(w, "Service model OID is required", http.StatusBadRequest)
		return
	}

	model, exists := s.serviceModelRegistry.GetServiceModel(request.ServiceModelOID)
	if !exists {
		http.Error(w, "Service model not found", http.StatusNotFound)
		return
	}

	var response map[string]interface{}
	var err error

	switch model.Type {
	case ServiceModelTypeKPM:
		header, message, processErr := s.serviceModelRegistry.ProcessKPMIndication(request.Header, request.Message)
		if processErr != nil {
			log.Printf("Failed to process KPM indication: %v", processErr)
			http.Error(w, "Failed to process KPM indication", http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{
			"type":    "KPM",
			"header":  header,
			"message": message,
		}

	case ServiceModelTypeNI:
		header, message, processErr := s.serviceModelRegistry.ProcessNIIndication(request.Header, request.Message)
		if processErr != nil {
			log.Printf("Failed to process NI indication: %v", processErr)
			http.Error(w, "Failed to process NI indication", http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{
			"type":    "NI",
			"header":  header,
			"message": message,
		}

	default:
		http.Error(w, "Unsupported service model type for indication processing", http.StatusBadRequest)
		return
	}

	response["serviceModelOid"] = request.ServiceModelOID
	response["timestamp"] = time.Now()

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode indication response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ProcessControlHandler handles processing of control messages
func (s *Server) ProcessControlHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ServiceModelOID string `json:"serviceModelOid"`
		Header          []byte `json:"header"`
		Message         []byte `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.ServiceModelOID == "" {
		http.Error(w, "Service model OID is required", http.StatusBadRequest)
		return
	}

	model, exists := s.serviceModelRegistry.GetServiceModel(request.ServiceModelOID)
	if !exists {
		http.Error(w, "Service model not found", http.StatusNotFound)
		return
	}

	if model.Type != ServiceModelTypeRC {
		http.Error(w, "Service model does not support control operations", http.StatusBadRequest)
		return
	}

	header, message, err := s.serviceModelRegistry.ProcessRCControl(request.Header, request.Message)
	if err != nil {
		log.Printf("Failed to process RC control: %v", err)
		http.Error(w, "Failed to process RC control", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"type":             "RC",
		"header":           header,
		"message":          message,
		"serviceModelOid":  request.ServiceModelOID,
		"timestamp":        time.Now(),
		"status":           "processed",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode control response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}