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

// O1HealthHandler handles requests for O1 Mediator health
func (s *Server) O1HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	health, err := o1Client.GetHealth(ctx)
	if err != nil {
		log.Printf("Failed to get O1 Mediator health: %v", err)
		http.Error(w, "Failed to retrieve O1 Mediator health", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Failed to encode O1 health response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1ManagedObjectsHandler handles requests for O1 managed objects
func (s *Server) O1ManagedObjectsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetManagedObjects(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetManagedObjects handles GET requests for managed objects
func (s *Server) handleGetManagedObjects(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	// Parse query parameters for filtering
	filter := &O1Filter{}
	if objectType := r.URL.Query().Get("type"); objectType != "" {
		filter.Type = objectType
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	}

	managedObjects, err := o1Client.GetManagedObjects(ctx, filter)
	if err != nil {
		log.Printf("Failed to get managed objects: %v", err)
		http.Error(w, "Failed to retrieve managed objects", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(managedObjects); err != nil {
		log.Printf("Failed to encode managed objects response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1ManagedObjectHandler handles requests for a specific O1 managed object
func (s *Server) O1ManagedObjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objectID := vars["objectId"]
	if objectID == "" {
		http.Error(w, "Object ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetManagedObject(w, r, ctx, o1Client, O1ManagedObjectID(objectID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetManagedObject handles GET requests for a specific managed object
func (s *Server) handleGetManagedObject(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, objectID O1ManagedObjectID) {
	managedObject, err := o1Client.GetManagedObject(ctx, objectID)
	if err != nil {
		log.Printf("Failed to get managed object %s: %v", objectID, err)
		if err.Error() == "managed object "+string(objectID)+" not found" {
			http.Error(w, "Managed object not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve managed object", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(managedObject); err != nil {
		log.Printf("Failed to encode managed object response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1ConfigurationsHandler handles requests for O1 configurations
func (s *Server) O1ConfigurationsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetConfigurations(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetConfigurations handles GET requests for configurations
func (s *Server) handleGetConfigurations(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	// Parse query parameters for filtering
	filter := &O1Filter{}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	}

	configurations, err := o1Client.GetConfigurations(ctx, filter)
	if err != nil {
		log.Printf("Failed to get configurations: %v", err)
		http.Error(w, "Failed to retrieve configurations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(configurations); err != nil {
		log.Printf("Failed to encode configurations response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1ConfigurationHandler handles requests for a specific O1 configuration
func (s *Server) O1ConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	configID := vars["configId"]
	if configID == "" {
		http.Error(w, "Configuration ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleCreateConfiguration(w, r, ctx, o1Client, O1ConfigurationID(configID))
	case http.MethodPut:
		s.handleUpdateConfiguration(w, r, ctx, o1Client, O1ConfigurationID(configID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreateConfiguration handles POST requests to create a configuration
func (s *Server) handleCreateConfiguration(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, configID O1ConfigurationID) {
	var request O1ConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := o1Client.CreateConfiguration(ctx, configID, &request); err != nil {
		log.Printf("Failed to create configuration %s: %v", configID, err)
		http.Error(w, "Failed to create configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]interface{}{
		"status":          "success",
		"message":         "Configuration created successfully",
		"configurationId": configID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode configuration creation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleUpdateConfiguration handles PUT requests to update a configuration
func (s *Server) handleUpdateConfiguration(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, configID O1ConfigurationID) {
	var update O1ConfigurationUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	update.ConfigurationID = configID

	if err := o1Client.UpdateConfiguration(ctx, configID, &update); err != nil {
		log.Printf("Failed to update configuration %s: %v", configID, err)
		http.Error(w, "Failed to update configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":          "success",
		"message":         "Configuration updated successfully",
		"configurationId": configID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode configuration update response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1AlarmsHandler handles requests for O1 alarms
func (s *Server) O1AlarmsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetAlarms(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetAlarms handles GET requests for alarms
func (s *Server) handleGetAlarms(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	// Parse query parameters for filtering
	filter := &O1Filter{}
	if severity := r.URL.Query().Get("severity"); severity != "" {
		filter.Severity = severity
	}

	alarms, err := o1Client.GetAlarms(ctx, filter)
	if err != nil {
		log.Printf("Failed to get alarms: %v", err)
		http.Error(w, "Failed to retrieve alarms", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(alarms); err != nil {
		log.Printf("Failed to encode alarms response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1AlarmHandler handles requests for a specific O1 alarm
func (s *Server) O1AlarmHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alarmID := vars["alarmId"]
	if alarmID == "" {
		http.Error(w, "Alarm ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleAcknowledgeAlarm(w, r, ctx, o1Client, O1AlarmID(alarmID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAcknowledgeAlarm handles POST requests to acknowledge an alarm
func (s *Server) handleAcknowledgeAlarm(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, alarmID O1AlarmID) {
	var ack O1AlarmAcknowledgment
	if err := json.NewDecoder(r.Body).Decode(&ack); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ack.AlarmID = alarmID

	if err := o1Client.AcknowledgeAlarm(ctx, alarmID, &ack); err != nil {
		log.Printf("Failed to acknowledge alarm %s: %v", alarmID, err)
		http.Error(w, "Failed to acknowledge alarm", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Alarm acknowledged successfully",
		"alarmId": alarmID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode alarm acknowledgment response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1KPIsHandler handles requests for O1 KPIs
func (s *Server) O1KPIsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetKPIs(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetKPIs handles GET requests for KPIs
func (s *Server) handleGetKPIs(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	// Parse query parameters for filtering
	filter := &O1Filter{}
	if kpiType := r.URL.Query().Get("type"); kpiType != "" {
		filter.Type = kpiType
	}

	kpis, err := o1Client.GetKPIs(ctx, filter)
	if err != nil {
		log.Printf("Failed to get KPIs: %v", err)
		http.Error(w, "Failed to retrieve KPIs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(kpis); err != nil {
		log.Printf("Failed to encode KPIs response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1StatsHandler handles requests for O1 Mediator statistics
func (s *Server) O1StatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	stats, err := o1Client.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to get O1 Mediator stats: %v", err)
		http.Error(w, "Failed to retrieve O1 Mediator stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Failed to encode O1 stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1BackupHandler handles requests for configuration backup
func (s *Server) O1BackupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	var request O1BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	backup, err := o1Client.BackupConfiguration(ctx, &request)
	if err != nil {
		log.Printf("Failed to create backup: %v", err)
		http.Error(w, "Failed to create backup", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(backup); err != nil {
		log.Printf("Failed to encode backup response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1RestoreHandler handles requests for configuration restore
func (s *Server) O1RestoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	var request O1RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	restore, err := o1Client.RestoreConfiguration(ctx, &request)
	if err != nil {
		log.Printf("Failed to restore configuration: %v", err)
		http.Error(w, "Failed to restore configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(restore); err != nil {
		log.Printf("Failed to encode restore response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1ValidationHandler handles requests for configuration validation
func (s *Server) O1ValidationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	var request O1ConfigurationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	validationResult, err := o1Client.ValidateConfiguration(ctx, request.Config)
	if err != nil {
		log.Printf("Failed to validate configuration: %v", err)
		http.Error(w, "Failed to validate configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(validationResult); err != nil {
		log.Printf("Failed to encode validation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1BackupsHandler handles requests for configuration backups
func (s *Server) O1BackupsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetBackups(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetBackups handles GET requests for backups
func (s *Server) handleGetBackups(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	filter := &O1Filter{}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	}

	backups, err := o1Client.GetBackups(ctx, filter)
	if err != nil {
		log.Printf("Failed to get backups: %v", err)
		http.Error(w, "Failed to retrieve backups", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(backups); err != nil {
		log.Printf("Failed to encode backups response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}


// End of file - orphaned code removed for clean build

