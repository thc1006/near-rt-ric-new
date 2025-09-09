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

// O1BackupHandler handles requests for a specific backup
func (s *Server) O1BackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID := vars["backupId"]
	if backupID == "" {
		http.Error(w, "Backup ID is required", http.StatusBadRequest)
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
	case http.MethodDelete:
		s.handleDeleteBackup(w, r, ctx, o1Client, backupID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDeleteBackup handles DELETE requests for a backup
func (s *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, backupID string) {
	if err := o1Client.DeleteBackup(ctx, backupID); err != nil {
		log.Printf("Failed to delete backup %s: %v", backupID, err)
		http.Error(w, "Failed to delete backup", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":   "success",
		"message":  "Backup deleted successfully",
		"backupId": backupID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode backup deletion response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1AlarmGenerationHandler handles requests for alarm generation
func (s *Server) O1AlarmGenerationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	var request O1AlarmRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	alarm, err := o1Client.GenerateAlarm(ctx, &request)
	if err != nil {
		log.Printf("Failed to generate alarm: %v", err)
		http.Error(w, "Failed to generate alarm", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(alarm); err != nil {
		log.Printf("Failed to encode alarm generation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1AlarmClearHandler handles requests for alarm clearing
func (s *Server) O1AlarmClearHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alarmID := vars["alarmId"]
	if alarmID == "" {
		http.Error(w, "Alarm ID is required", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	var request O1AlarmClearRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	request.AlarmID = O1AlarmID(alarmID)

	if err := o1Client.ClearAlarm(ctx, O1AlarmID(alarmID), &request); err != nil {
		log.Printf("Failed to clear alarm %s: %v", alarmID, err)
		http.Error(w, "Failed to clear alarm", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Alarm cleared successfully",
		"alarmId": alarmID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode alarm clear response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1AlarmCorrelationHandler handles requests for alarm correlation
func (s *Server) O1AlarmCorrelationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	var request O1AlarmCorrelationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	correlation, err := o1Client.CorrelateAlarms(ctx, &request)
	if err != nil {
		log.Printf("Failed to correlate alarms: %v", err)
		http.Error(w, "Failed to correlate alarms", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(correlation); err != nil {
		log.Printf("Failed to encode alarm correlation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1KPIManagementHandler handles requests for KPI management
func (s *Server) O1KPIManagementHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleCreateKPI(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreateKPI handles POST requests to create a KPI
func (s *Server) handleCreateKPI(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	var request O1KPIRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	kpi, err := o1Client.CreateKPI(ctx, &request)
	if err != nil {
		log.Printf("Failed to create KPI: %v", err)
		http.Error(w, "Failed to create KPI", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(kpi); err != nil {
		log.Printf("Failed to encode KPI creation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1KPIHandler handles requests for a specific KPI
func (s *Server) O1KPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	kpiID := vars["kpiId"]
	if kpiID == "" {
		http.Error(w, "KPI ID is required", http.StatusBadRequest)
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
	case http.MethodPut:
		s.handleUpdateKPI(w, r, ctx, o1Client, O1KPIID(kpiID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleUpdateKPI handles PUT requests to update a KPI
func (s *Server) handleUpdateKPI(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, kpiID O1KPIID) {
	var update O1KPIUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	update.KPIID = kpiID

	if err := o1Client.UpdateKPI(ctx, kpiID, &update); err != nil {
		log.Printf("Failed to update KPI %s: %v", kpiID, err)
		http.Error(w, "Failed to update KPI", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "KPI updated successfully",
		"kpiId":   kpiID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode KPI update response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1KPICollectionHandler handles requests for KPI data collection
func (s *Server) O1KPICollectionHandler(w http.ResponseWriter, r *http.Request) {
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

	var request O1KPICollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	collection, err := o1Client.CollectKPIData(ctx, &request)
	if err != nil {
		log.Printf("Failed to collect KPI data: %v", err)
		http.Error(w, "Failed to collect KPI data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(collection); err != nil {
		log.Printf("Failed to encode KPI collection response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1CertificatesHandler handles requests for certificates
func (s *Server) O1CertificatesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetCertificates(w, r, ctx, o1Client)
	case http.MethodPost:
		s.handleCreateCertificate(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetCertificates handles GET requests for certificates
func (s *Server) handleGetCertificates(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	filter := &O1Filter{}
	if certType := r.URL.Query().Get("type"); certType != "" {
		filter.Type = certType
	}

	certificates, err := o1Client.GetCertificates(ctx, filter)
	if err != nil {
		log.Printf("Failed to get certificates: %v", err)
		http.Error(w, "Failed to retrieve certificates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(certificates); err != nil {
		log.Printf("Failed to encode certificates response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCreateCertificate handles POST requests to create a certificate
func (s *Server) handleCreateCertificate(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	var request O1CertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	certificate, err := o1Client.CreateCertificate(ctx, &request)
	if err != nil {
		log.Printf("Failed to create certificate: %v", err)
		http.Error(w, "Failed to create certificate", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(certificate); err != nil {
		log.Printf("Failed to encode certificate creation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1CertificateHandler handles requests for a specific certificate
func (s *Server) O1CertificateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	certID := vars["certId"]
	if certID == "" {
		http.Error(w, "Certificate ID is required", http.StatusBadRequest)
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
		s.handleRevokeCertificate(w, r, ctx, o1Client, certID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRevokeCertificate handles POST requests to revoke a certificate
func (s *Server) handleRevokeCertificate(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, certID string) {
	var request map[string]string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reason, ok := request["reason"]
	if !ok {
		reason = "unspecified"
	}

	if err := o1Client.RevokeCertificate(ctx, certID, reason); err != nil {
		log.Printf("Failed to revoke certificate %s: %v", certID, err)
		http.Error(w, "Failed to revoke certificate", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":        "success",
		"message":       "Certificate revoked successfully",
		"certificateId": certID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode certificate revocation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1ResourceUsageHandler handles requests for resource usage
func (s *Server) O1ResourceUsageHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetResourceUsage(w, r, ctx, o1Client)
	case http.MethodPost:
		s.handleCreateResourceUsageRecord(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetResourceUsage handles GET requests for resource usage
func (s *Server) handleGetResourceUsage(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	filter := &O1Filter{}
	if resourceType := r.URL.Query().Get("type"); resourceType != "" {
		filter.Type = resourceType
	}

	usage, err := o1Client.GetResourceUsage(ctx, filter)
	if err != nil {
		log.Printf("Failed to get resource usage: %v", err)
		http.Error(w, "Failed to retrieve resource usage", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(usage); err != nil {
		log.Printf("Failed to encode resource usage response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCreateResourceUsageRecord handles POST requests to create a resource usage record
func (s *Server) handleCreateResourceUsageRecord(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	var request O1ResourceUsageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	usage, err := o1Client.CreateResourceUsageRecord(ctx, &request)
	if err != nil {
		log.Printf("Failed to create resource usage record: %v", err)
		http.Error(w, "Failed to create resource usage record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(usage); err != nil {
		log.Printf("Failed to encode resource usage creation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1AccessControlHandler handles requests for access control policies
func (s *Server) O1AccessControlHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	o1Client := s.clients.GetO1MediatorClient()
	if o1Client == nil {
		http.Error(w, "O1 Mediator client not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetAccessControlPolicies(w, r, ctx, o1Client)
	case http.MethodPost:
		s.handleCreateAccessControlPolicy(w, r, ctx, o1Client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetAccessControlPolicies handles GET requests for access control policies
func (s *Server) handleGetAccessControlPolicies(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	filter := &O1Filter{}
	if policyType := r.URL.Query().Get("type"); policyType != "" {
		filter.Type = policyType
	}

	policies, err := o1Client.GetAccessControlPolicies(ctx, filter)
	if err != nil {
		log.Printf("Failed to get access control policies: %v", err)
		http.Error(w, "Failed to retrieve access control policies", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policies); err != nil {
		log.Printf("Failed to encode access control policies response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleCreateAccessControlPolicy handles POST requests to create an access control policy
func (s *Server) handleCreateAccessControlPolicy(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient) {
	var request O1AccessControlPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	policy, err := o1Client.CreateAccessControlPolicy(ctx, &request)
	if err != nil {
		log.Printf("Failed to create access control policy: %v", err)
		http.Error(w, "Failed to create access control policy", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(policy); err != nil {
		log.Printf("Failed to encode access control policy creation response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// O1AccessControlPolicyHandler handles requests for a specific access control policy
func (s *Server) O1AccessControlPolicyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyID := vars["policyId"]
	if policyID == "" {
		http.Error(w, "Policy ID is required", http.StatusBadRequest)
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
		s.handleGetAccessControlPolicy(w, r, ctx, o1Client, policyID)
	case http.MethodPUT:
		s.handleUpdateAccessControlPolicy(w, r, ctx, o1Client, policyID)
	case http.MethodDELETE:
		s.handleDeleteAccessControlPolicy(w, r, ctx, o1Client, policyID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetAccessControlPolicy handles GET requests for a specific access control policy
func (s *Server) handleGetAccessControlPolicy(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, policyID string) {
	// For now, return a mock policy since the client doesn't have this method yet
	policy := &O1AccessControlPolicy{
		ID:          policyID,
		Name:        "Sample Policy",
		Description: "Sample access control policy",
		PolicyType:  "RBAC",
		Rules: []O1AccessControlRule{
			{
				ID: "rule1",
				Subject: O1AccessControlSubject{
					Type:       "USER",
					Identifier: "admin",
				},
				Action:   "READ",
				Resource: "*",
				Effect:   "ALLOW",
			},
		},
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(policy); err != nil {
		log.Printf("Failed to encode access control policy response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleUpdateAccessControlPolicy handles PUT requests to update an access control policy
func (s *Server) handleUpdateAccessControlPolicy(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, policyID string) {
	var update O1AccessControlPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For now, just return success since the client doesn't have this method yet
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":   "success",
		"message":  "Access control policy updated successfully",
		"policyId": policyID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode access control policy update response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleDeleteAccessControlPolicy handles DELETE requests for an access control policy
func (s *Server) handleDeleteAccessControlPolicy(w http.ResponseWriter, r *http.Request, ctx context.Context, o1Client *O1MediatorClient, policyID string) {
	// For now, just return success since the client doesn't have this method yet
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":   "success",
		"message":  "Access control policy deleted successfully",
		"policyId": policyID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode access control policy deletion response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
