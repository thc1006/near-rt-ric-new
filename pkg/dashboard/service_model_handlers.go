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
		NodeID          string `json:"nodeId,omitempty"`
		SubscriptionID  string `json:"subscriptionId,omitempty"`
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

	// Use the service model API manager for processing
	ctx := context.Background()
	result, err := s.serviceModelAPIManager.ProcessIndication(ctx, model.Type, request.Header, request.Message)
	if err != nil {
		log.Printf("Failed to process indication: %v", err)
		http.Error(w, "Failed to process indication", http.StatusInternalServerError)
		return
	}

	response := ServiceModelResponse{
		ServiceModelOID: request.ServiceModelOID,
		MessageType:     "indication",
		Status:          "success",
		Result:          result,
		Timestamp:       time.Now(),
		Metadata: map[string]interface{}{
			"nodeId":         request.NodeID,
			"subscriptionId": request.SubscriptionID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
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
		NodeID          string `json:"nodeId,omitempty"`
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

	// Use the service model API manager for processing
	ctx := context.Background()
	result, err := s.serviceModelAPIManager.ProcessControl(ctx, model.Type, request.Header, request.Message)
	if err != nil {
		log.Printf("Failed to process control: %v", err)
		http.Error(w, "Failed to process control", http.StatusInternalServerError)
		return
	}

	response := ServiceModelResponse{
		ServiceModelOID: request.ServiceModelOID,
		MessageType:     "control",
		Status:          "success",
		Result:          result,
		Timestamp:       time.Now(),
		Metadata: map[string]interface{}{
			"nodeId": request.NodeID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode control response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
//
 ServiceModelOperationsHandler handles requests for supported operations
func (s *Server) ServiceModelOperationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	operations := s.serviceModelAPIManager.GetSupportedOperations()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"operations": operations,
		"timestamp":  time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode operations response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ServiceModelSchemaHandler handles requests for message schemas
func (s *Server) ServiceModelSchemaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	modelType := vars["type"]
	messageType := r.URL.Query().Get("messageType")

	if modelType == "" {
		http.Error(w, "Service model type is required", http.StatusBadRequest)
		return
	}

	if messageType == "" {
		http.Error(w, "Message type is required", http.StatusBadRequest)
		return
	}

	schema, err := s.serviceModelAPIManager.GetMessageSchema(ServiceModelType(modelType), messageType)
	if err != nil {
		log.Printf("Failed to get message schema: %v", err)
		http.Error(w, "Failed to get message schema", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"serviceModelType": modelType,
		"messageType":      messageType,
		"schema":           schema,
		"timestamp":        time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode schema response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ValidateMessageHandler handles message validation requests
func (s *Server) ValidateMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ServiceModelType ServiceModelType `json:"serviceModelType"`
		MessageType      string           `json:"messageType"`
		Data             []byte           `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.ServiceModelType == "" {
		http.Error(w, "Service model type is required", http.StatusBadRequest)
		return
	}

	if request.MessageType == "" {
		http.Error(w, "Message type is required", http.StatusBadRequest)
		return
	}

	err := s.serviceModelAPIManager.ValidateMessage(request.ServiceModelType, request.MessageType, request.Data)
	
	var status string
	var errorMsg string
	if err != nil {
		status = "invalid"
		errorMsg = err.Error()
	} else {
		status = "valid"
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"serviceModelType": request.ServiceModelType,
		"messageType":      request.MessageType,
		"status":           status,
		"timestamp":        time.Now(),
	}

	if errorMsg != "" {
		response["error"] = errorMsg
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode validation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// KPIDefinitionsHandler handles requests for KPI definitions
func (s *Server) KPIDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get KPM API
	api, err := s.serviceModelAPIManager.GetAPI(ServiceModelTypeKPM)
	if err != nil {
		http.Error(w, "KPM API not available", http.StatusNotFound)
		return
	}

	kmpAPI, ok := api.(*E2SMKPMApi)
	if !ok {
		http.Error(w, "Invalid KPM API type", http.StatusInternalServerError)
		return
	}

	definitions := kmpAPI.GetKPIDefinitions()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"kpiDefinitions": definitions,
		"total":          len(definitions),
		"timestamp":      time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode KPI definitions response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ControlActionDefinitionsHandler handles requests for control action definitions
func (s *Server) ControlActionDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get RC API
	api, err := s.serviceModelAPIManager.GetAPI(ServiceModelTypeRC)
	if err != nil {
		http.Error(w, "RC API not available", http.StatusNotFound)
		return
	}

	rcAPI, ok := api.(*E2SMRCApi)
	if !ok {
		http.Error(w, "Invalid RC API type", http.StatusInternalServerError)
		return
	}

	definitions := rcAPI.GetControlActionDefinitions()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"controlActionDefinitions": definitions,
		"total":                    len(definitions),
		"timestamp":                time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode control action definitions response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// InterfaceDefinitionsHandler handles requests for interface definitions
func (s *Server) InterfaceDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get NI API
	api, err := s.serviceModelAPIManager.GetAPI(ServiceModelTypeNI)
	if err != nil {
		http.Error(w, "NI API not available", http.StatusNotFound)
		return
	}

	niAPI, ok := api.(*E2SMNIApi)
	if !ok {
		http.Error(w, "Invalid NI API type", http.StatusInternalServerError)
		return
	}

	interfaceDefinitions := niAPI.GetInterfaceDefinitions()
	protocolDefinitions := niAPI.GetSupportedProtocols()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"interfaceDefinitions": interfaceDefinitions,
		"protocolDefinitions":  protocolDefinitions,
		"timestamp":            time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode interface definitions response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}