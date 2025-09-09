/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ServiceModelRegistry type is now defined in types.go to avoid redeclaration

// ServiceModelInterface is defined in service_model_api.go to avoid redeclaration

// ServiceModelType and constants are now defined in types.go to avoid redeclaration

// ServiceModelCapabilities is now defined in types.go to avoid redeclaration

// ServiceModelStatistics is now defined in types.go to avoid redeclaration

// NewServiceModelRegistry function is now defined in types.go to avoid redeclaration

// RegisterServiceModel registers a service model implementation
func (r *ServiceModelRegistry) RegisterServiceModel(api ServiceModelAPI) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	serviceModelType := api.GetServiceModelType()
	
	// Register the API
	r.serviceModels[serviceModelType] = api
	
	// Initialize capabilities
	r.capabilities[serviceModelType] = &ServiceModelCapabilities{
		ServiceModelType:      serviceModelType,
		Version:               r.getServiceModelVersion(serviceModelType),
		SupportedOperations:   api.GetSupportedOperations(),
		SupportedMessageTypes: r.getSupportedMessageTypes(serviceModelType),
		SupportsIndications:   true, // All service models support indications
		SupportsControl:       r.supportsControl(serviceModelType),
		MaxConcurrentOps:      100, // Default limit
		LastUpdated:           time.Now(),
	}
	
	// Initialize statistics
	r.statistics[serviceModelType] = &ServiceModelStatistics{
		ServiceModelType: serviceModelType,
		LastProcessedAt:  time.Now(),
	}
	
	log.Printf("Registered service model: %s", serviceModelType)
	return nil
}

// GetServiceModel returns a service model implementation
func (r *ServiceModelRegistry) GetServiceModel(serviceModelType ServiceModelType) (ServiceModelAPI, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	api, exists := r.serviceModels[serviceModelType]
	if !exists {
		return nil, fmt.Errorf("service model %s not found", serviceModelType)
	}
	
	return api, nil
}

// ProcessIndication processes an indication message using the appropriate service model
func (r *ServiceModelRegistry) ProcessIndication(ctx context.Context, serviceModelType ServiceModelType, header []byte, message []byte) (interface{}, error) {
	api, err := r.GetServiceModel(serviceModelType)
	if err != nil {
		return nil, err
	}
	
	// Update statistics
	r.updateStatistics(serviceModelType, "indication")
	
	// Process the indication
	startTime := time.Now()
	result, err := api.ProcessIndication(ctx, header, message)
	processingTime := time.Since(startTime)
	
	// Update processing time statistics
	r.updateProcessingTime(serviceModelType, processingTime)
	
	if err != nil {
		r.incrementErrorCount(serviceModelType, "processing")
		return nil, fmt.Errorf("failed to process indication for %s: %w", serviceModelType, err)
	}
	
	return result, nil
}

// ProcessControl processes a control message using the appropriate service model
func (r *ServiceModelRegistry) ProcessControl(ctx context.Context, serviceModelType ServiceModelType, header []byte, message []byte) (interface{}, error) {
	api, err := r.GetServiceModel(serviceModelType)
	if err != nil {
		return nil, err
	}
	
	// Check if service model supports control
	if !r.supportsControl(serviceModelType) {
		return nil, fmt.Errorf("service model %s does not support control messages", serviceModelType)
	}
	
	// Update statistics
	r.updateStatistics(serviceModelType, "control")
	
	// Process the control message
	startTime := time.Now()
	result, err := api.ProcessControl(ctx, header, message)
	processingTime := time.Since(startTime)
	
	// Update processing time statistics
	r.updateProcessingTime(serviceModelType, processingTime)
	
	if err != nil {
		r.incrementErrorCount(serviceModelType, "processing")
		return nil, fmt.Errorf("failed to process control for %s: %w", serviceModelType, err)
	}
	
	return result, nil
}

// ValidateMessage validates a message against a service model
func (r *ServiceModelRegistry) ValidateMessage(serviceModelType ServiceModelType, messageType string, data []byte) error {
	api, err := r.GetServiceModel(serviceModelType)
	if err != nil {
		return err
	}
	
	err = api.ValidateMessage(messageType, data)
	if err != nil {
		r.incrementErrorCount(serviceModelType, "validation")
		return fmt.Errorf("validation failed for %s/%s: %w", serviceModelType, messageType, err)
	}
	
	return nil
}

// GetCapabilities returns the capabilities of a service model
func (r *ServiceModelRegistry) GetCapabilities(serviceModelType ServiceModelType) (*ServiceModelCapabilities, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	capabilities, exists := r.capabilities[serviceModelType]
	if !exists {
		return nil, fmt.Errorf("capabilities not found for service model %s", serviceModelType)
	}
	
	return capabilities, nil
}

// GetStatistics returns the statistics of a service model
func (r *ServiceModelRegistry) GetStatistics(serviceModelType ServiceModelType) (*ServiceModelStatistics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	stats, exists := r.statistics[serviceModelType]
	if !exists {
		return nil, fmt.Errorf("statistics not found for service model %s", serviceModelType)
	}
	
	return stats, nil
}

// GetAllStatistics returns statistics for all registered service models
func (r *ServiceModelRegistry) GetAllStatistics() map[ServiceModelType]*ServiceModelStatistics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[ServiceModelType]*ServiceModelStatistics)
	for modelType, stats := range r.statistics {
		// Create a copy to prevent external modification
		statsCopy := *stats
		result[modelType] = &statsCopy
	}
	
	return result
}

// GetSupportedServiceModels returns a list of supported service model types
func (r *ServiceModelRegistry) GetSupportedServiceModels() []ServiceModelType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.supportedTypes
}

// updateStatistics updates processing statistics for a service model
func (r *ServiceModelRegistry) updateStatistics(serviceModelType ServiceModelType, operationType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	stats, exists := r.statistics[serviceModelType]
	if !exists {
		return
	}
	
	switch operationType {
	case "indication":
		stats.IndicationsProcessed++
	case "control":
		stats.ControlsProcessed++
	}
	
	stats.LastProcessedAt = time.Now()
}

// updateProcessingTime updates the average processing time for a service model
func (r *ServiceModelRegistry) updateProcessingTime(serviceModelType ServiceModelType, processingTime time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	stats, exists := r.statistics[serviceModelType]
	if !exists {
		return
	}
	
	// Update total processing time
	stats.TotalProcessingTime += processingTime
	
	// Calculate new average
	totalOperations := stats.IndicationsProcessed + stats.ControlsProcessed
	if totalOperations > 0 {
		stats.AverageProcessingTime = stats.TotalProcessingTime / time.Duration(totalOperations)
	}
}

// incrementErrorCount increments the error count for a service model
func (r *ServiceModelRegistry) incrementErrorCount(serviceModelType ServiceModelType, errorType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	stats, exists := r.statistics[serviceModelType]
	if !exists {
		return
	}
	
	switch errorType {
	case "validation":
		stats.ValidationErrors++
	case "processing":
		stats.ProcessingErrors++
	}
}

// getServiceModelVersion returns the version of a service model
func (r *ServiceModelRegistry) getServiceModelVersion(serviceModelType ServiceModelType) string {
	switch serviceModelType {
	case ServiceModelTypeKPM:
		return "2.0"
	case ServiceModelTypeRC:
		return "1.0"
	case ServiceModelTypeNI:
		return "1.0"
	default:
		return "1.0"
	}
}

// getSupportedMessageTypes returns the supported message types for a service model
func (r *ServiceModelRegistry) getSupportedMessageTypes(serviceModelType ServiceModelType) []string {
	switch serviceModelType {
	case ServiceModelTypeKPM:
		return []string{"E2SM-KPM-IndicationHeader", "E2SM-KPM-IndicationMessage"}
	case ServiceModelTypeRC:
		return []string{"E2SM-RC-ControlHeader", "E2SM-RC-ControlMessage", "E2SM-RC-ControlOutcome"}
	case ServiceModelTypeNI:
		return []string{"E2SM-NI-IndicationHeader", "E2SM-NI-IndicationMessage"}
	default:
		return []string{}
	}
}

// supportsControl checks if a service model supports control messages
func (r *ServiceModelRegistry) supportsControl(serviceModelType ServiceModelType) bool {
	switch serviceModelType {
	case ServiceModelTypeRC:
		return true
	case ServiceModelTypeKPM, ServiceModelTypeNI:
		return false
	default:
		return false
	}
}

// InitializeDefaultServiceModels initializes the registry with default service models
func (r *ServiceModelRegistry) InitializeDefaultServiceModels() error {
	// Register E2SM-KPM
	kpmAPI := &E2SMKPMServiceModel{}
	if err := r.RegisterServiceModel(kpmAPI); err != nil {
		return fmt.Errorf("failed to register E2SM-KPM: %w", err)
	}
	
	// Register E2SM-RC
	rcAPI := &E2SMRCServiceModel{}
	if err := r.RegisterServiceModel(rcAPI); err != nil {
		return fmt.Errorf("failed to register E2SM-RC: %w", err)
	}
	
	// Register E2SM-NI
	niAPI := &E2SMNIServiceModel{}
	if err := r.RegisterServiceModel(niAPI); err != nil {
		return fmt.Errorf("failed to register E2SM-NI: %w", err)
	}
	
	// Update supported types
	r.mu.Lock()
	r.supportedTypes = []ServiceModelType{
		ServiceModelTypeKPM,
		ServiceModelTypeRC,
		ServiceModelTypeNI,
	}
	r.mu.Unlock()
	
	log.Println("Initialized default service models")
	return nil
}

// ResetStatistics resets the statistics for a specific service model or all models
func (r *ServiceModelRegistry) ResetStatistics(serviceModelType ...ServiceModelType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if len(serviceModelType) == 0 {
		// Reset all statistics
		for modelType := range r.statistics {
			r.statistics[modelType] = &ServiceModelStatistics{
				ServiceModelType: modelType,
				LastProcessedAt:  time.Now(),
			}
		}
	} else {
		// Reset specific service model statistics
		for _, modelType := range serviceModelType {
			if _, exists := r.statistics[modelType]; exists {
				r.statistics[modelType] = &ServiceModelStatistics{
					ServiceModelType: modelType,
					LastProcessedAt:  time.Now(),
				}
			}
		}
	}
}

// GetServiceModelCount returns the number of registered service models
func (r *ServiceModelRegistry) GetServiceModelCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return len(r.serviceModels)
}

// IsServiceModelRegistered checks if a service model is registered
func (r *ServiceModelRegistry) IsServiceModelRegistered(serviceModelType ServiceModelType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	_, exists := r.serviceModels[serviceModelType]
	return exists
}

// E2SMKPMServiceModel implements ServiceModelAPI for E2SM-KPM
type E2SMKPMServiceModel struct{}

func (m *E2SMKPMServiceModel) GetServiceModelType() ServiceModelType {
	return ServiceModelTypeKPM
}

func (m *E2SMKPMServiceModel) GetSupportedOperations() []string {
	return []string{"report", "subscribe", "indication"}
}

func (m *E2SMKPMServiceModel) ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	// Process KPM indication
	return map[string]interface{}{
		"type": "kpm_indication",
		"data": string(message),
	}, nil
}

func (m *E2SMKPMServiceModel) ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	return nil, fmt.Errorf("E2SM-KPM does not support control messages")
}

func (m *E2SMKPMServiceModel) ValidateMessage(messageType string, data []byte) error {
	// Validate KPM message
	if len(data) == 0 {
		return fmt.Errorf("empty message data")
	}
	return nil
}

// E2SMRCServiceModel implements ServiceModelAPI for E2SM-RC
type E2SMRCServiceModel struct{}

func (m *E2SMRCServiceModel) GetServiceModelType() ServiceModelType {
	return ServiceModelTypeRC
}

func (m *E2SMRCServiceModel) GetSupportedOperations() []string {
	return []string{"control", "policy", "indication"}
}

func (m *E2SMRCServiceModel) ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	// Process RC indication
	return map[string]interface{}{
		"type": "rc_indication",
		"data": string(message),
	}, nil
}

func (m *E2SMRCServiceModel) ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	// Process RC control
	return map[string]interface{}{
		"type": "rc_control",
		"data": string(message),
	}, nil
}

func (m *E2SMRCServiceModel) ValidateMessage(messageType string, data []byte) error {
	// Validate RC message
	if len(data) == 0 {
		return fmt.Errorf("empty message data")
	}
	return nil
}

// E2SMNIServiceModel implements ServiceModelAPI for E2SM-NI
type E2SMNIServiceModel struct{}

func (m *E2SMNIServiceModel) GetServiceModelType() ServiceModelType {
	return ServiceModelTypeNI
}

func (m *E2SMNIServiceModel) GetSupportedOperations() []string {
	return []string{"monitor", "analyze", "indication"}
}

func (m *E2SMNIServiceModel) ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	// Process NI indication
	return map[string]interface{}{
		"type": "ni_indication",
		"data": string(message),
	}, nil
}

func (m *E2SMNIServiceModel) ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	return nil, fmt.Errorf("E2SM-NI does not support control messages")
}

func (m *E2SMNIServiceModel) ValidateMessage(messageType string, data []byte) error {
	// Validate NI message
	if len(data) == 0 {
		return fmt.Errorf("empty message data")
	}
	return nil
}