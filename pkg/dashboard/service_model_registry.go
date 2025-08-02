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

// ServiceModelRegistry manages all service model implementations
type ServiceModelRegistry struct {
	mu           sync.RWMutex
	serviceModels map[ServiceModelType]ServiceModelAPI
	capabilities  map[ServiceModelType]*ServiceModelCapabilities
	statistics    map[ServiceModelType]*ServiceModelStatistics
}

// ServiceModelAPI defines the interface for service model implementations
type ServiceModelAPI interface {
	GetServiceModelType() ServiceModelType
	ValidateMessage(messageType string, data []byte) error
	ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error)
	ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error)
	GetSupportedOperations() []string
	GetMessageSchema(messageType string) (map[string]interface{}, error)
}

// ServiceModelType represents the type of service model
type ServiceModelType string

const (
	ServiceModelTypeKPM ServiceModelType = "E2SM-KPM"
	ServiceModelTypeRC  ServiceModelType = "E2SM-RC"
	ServiceModelTypeNI  ServiceModelType = "E2SM-NI"
)

// ServiceModelCapabilities represents the capabilities of a service model
type ServiceModelCapabilities struct {
	ServiceModelType     ServiceModelType `json:"serviceModelType"`
	Version              string           `json:"version"`
	SupportedOperations  []string         `json:"supportedOperations"`
	SupportedMessageTypes []string        `json:"supportedMessageTypes"`
	SupportsIndications  bool             `json:"supportsIndications"`
	SupportsControl      bool             `json:"supportsControl"`
	MaxConcurrentOps     int              `json:"maxConcurrentOps"`
	LastUpdated          time.Time        `json:"lastUpdated"`
}

// ServiceModelStatistics represents statistics for a service model
type ServiceModelStatistics struct {
	ServiceModelType      ServiceModelType `json:"serviceModelType"`
	IndicationsProcessed  uint64           `json:"indicationsProcessed"`
	ControlsProcessed     uint64           `json:"controlsProcessed"`
	ValidationErrors      uint64           `json:"validationErrors"`
	ProcessingErrors      uint64           `json:"processingErrors"`
	AverageProcessingTime time.Duration    `json:"averageProcessingTime"`
	LastProcessedAt       time.Time        `json:"lastProcessedAt"`
	TotalProcessingTime   time.Duration    `json:"totalProcessingTime"`
}

// NewServiceModelRegistry creates a new service model registry
func NewServiceModelRegistry() *ServiceModelRegistry {
	registry := &ServiceModelRegistry{
		serviceModels: make(map[ServiceModelType]ServiceModelAPI),
		capabilities:  make(map[ServiceModelType]*ServiceModelCapabilities),
		statistics:    make(map[ServiceModelType]*ServiceModelStatistics),
	}
	
	// Initialize service models
	registry.initializeServiceModels()
	
	return registry
}

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
	startTime := time.Now()
	
	api, err := r.GetServiceModel(serviceModelType)
	if err != nil {
		r.updateStatistics(serviceModelType, false, false, time.Since(startTime))
		return nil, err
	}
	
	result, err := api.ProcessIndication(ctx, header, message)
	if err != nil {
		r.updateStatistics(serviceModelType, false, false, time.Since(startTime))
		return nil, fmt.Errorf("failed to process indication: %w", err)
	}
	
	r.updateStatistics(serviceModelType, true, false, time.Since(startTime))
	return result, nil
}

// ProcessControl processes a control message using the appropriate service model
func (r *ServiceModelRegistry) ProcessControl(ctx context.Context, serviceModelType ServiceModelType, header []byte, message []byte) (interface{}, error) {
	startTime := time.Now()
	
	api, err := r.GetServiceModel(serviceModelType)
	if err != nil {
		r.updateStatistics(serviceModelType, false, true, time.Since(startTime))
		return nil, err
	}
	
	result, err := api.ProcessControl(ctx, header, message)
	if err != nil {
		r.updateStatistics(serviceModelType, false, true, time.Since(startTime))
		return nil, fmt.Errorf("failed to process control: %w", err)
	}
	
	r.updateStatistics(serviceModelType, true, true, time.Since(startTime))
	return result, nil
}

// ValidateMessage validates a message using the appropriate service model
func (r *ServiceModelRegistry) ValidateMessage(serviceModelType ServiceModelType, messageType string, data []byte) error {
	api, err := r.GetServiceModel(serviceModelType)
	if err != nil {
		return err
	}
	
	if err := api.ValidateMessage(messageType, data); err != nil {
		r.incrementValidationErrors(serviceModelType)
		return fmt.Errorf("message validation failed: %w", err)
	}
	
	return nil
}

// GetCapabilities returns capabilities for a service model
func (r *ServiceModelRegistry) GetCapabilities(serviceModelType ServiceModelType) (*ServiceModelCapabilities, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	capabilities, exists := r.capabilities[serviceModelType]
	if !exists {
		return nil, fmt.Errorf("capabilities for service model %s not found", serviceModelType)
	}
	
	return capabilities, nil
}

// GetAllCapabilities returns capabilities for all registered service models
func (r *ServiceModelRegistry) GetAllCapabilities() map[ServiceModelType]*ServiceModelCapabilities {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	capabilities := make(map[ServiceModelType]*ServiceModelCapabilities)
	for serviceModelType, cap := range r.capabilities {
		capabilities[serviceModelType] = &ServiceModelCapabilities{
			ServiceModelType:      cap.ServiceModelType,
			Version:               cap.Version,
			SupportedOperations:   make([]string, len(cap.SupportedOperations)),
			SupportedMessageTypes: make([]string, len(cap.SupportedMessageTypes)),
			SupportsIndications:   cap.SupportsIndications,
			SupportsControl:       cap.SupportsControl,
			MaxConcurrentOps:      cap.MaxConcurrentOps,
			LastUpdated:           cap.LastUpdated,
		}
		copy(capabilities[serviceModelType].SupportedOperations, cap.SupportedOperations)
		copy(capabilities[serviceModelType].SupportedMessageTypes, cap.SupportedMessageTypes)
	}
	
	return capabilities
}

// GetStatistics returns statistics for a service model
func (r *ServiceModelRegistry) GetStatistics(serviceModelType ServiceModelType) (*ServiceModelStatistics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	stats, exists := r.statistics[serviceModelType]
	if !exists {
		return nil, fmt.Errorf("statistics for service model %s not found", serviceModelType)
	}
	
	return stats, nil
}

// GetAllStatistics returns statistics for all registered service models
func (r *ServiceModelRegistry) GetAllStatistics() map[ServiceModelType]*ServiceModelStatistics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	statistics := make(map[ServiceModelType]*ServiceModelStatistics)
	for serviceModelType, stats := range r.statistics {
		statistics[serviceModelType] = &ServiceModelStatistics{
			ServiceModelType:      stats.ServiceModelType,
			IndicationsProcessed:  stats.IndicationsProcessed,
			ControlsProcessed:     stats.ControlsProcessed,
			ValidationErrors:      stats.ValidationErrors,
			ProcessingErrors:      stats.ProcessingErrors,
			AverageProcessingTime: stats.AverageProcessingTime,
			LastProcessedAt:       stats.LastProcessedAt,
			TotalProcessingTime:   stats.TotalProcessingTime,
		}
	}
	
	return statistics
}

// GetRegisteredServiceModels returns list of registered service model types
func (r *ServiceModelRegistry) GetRegisteredServiceModels() []ServiceModelType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	serviceModels := make([]ServiceModelType, 0, len(r.serviceModels))
	for serviceModelType := range r.serviceModels {
		serviceModels = append(serviceModels, serviceModelType)
	}
	
	return serviceModels
}

// ProcessKPMIndication processes KPM indication (legacy method for compatibility)
func (r *ServiceModelRegistry) ProcessKPMIndication(header []byte, message []byte) (*E2SMKPMIndicationHeader, *E2SMKPMIndicationMessage, error) {
	// This is a compatibility method - in practice, use ProcessIndication
	return &E2SMKPMIndicationHeader{}, &E2SMKPMIndicationMessage{}, nil
}

// ProcessRCIndication processes RC indication (legacy method for compatibility)
func (r *ServiceModelRegistry) ProcessRCIndication(header []byte, message []byte) (*E2SMRCIndicationHeader, *E2SMRCIndicationMessage, error) {
	// This is a compatibility method - in practice, use ProcessIndication
	return &E2SMRCIndicationHeader{}, &E2SMRCIndicationMessage{}, nil
}

// ProcessRCControl processes RC control (legacy method for compatibility)
func (r *ServiceModelRegistry) ProcessRCControl(header []byte, message []byte) (*E2SMRCControlHeader, *E2SMRCControlMessage, error) {
	// This is a compatibility method - in practice, use ProcessControl
	return &E2SMRCControlHeader{}, &E2SMRCControlMessage{}, nil
}

// ProcessNIIndication processes NI indication (legacy method for compatibility)
func (r *ServiceModelRegistry) ProcessNIIndication(header []byte, message []byte) (*E2SMNIIndicationHeader, *E2SMNIIndicationMessage, error) {
	// This is a compatibility method - in practice, use ProcessIndication
	return &E2SMNIIndicationHeader{}, &E2SMNIIndicationMessage{}, nil
}

// Private methods

func (r *ServiceModelRegistry) initializeServiceModels() {
	// Register KPM service model
	kmpAPI := NewE2SMKPMApi(r)
	if err := r.RegisterServiceModel(kmpAPI); err != nil {
		log.Printf("Failed to register E2SM-KPM: %v", err)
	}
	
	// Register RC service model
	rcAPI := NewE2SMRCApi(r)
	if err := r.RegisterServiceModel(rcAPI); err != nil {
		log.Printf("Failed to register E2SM-RC: %v", err)
	}
	
	// Register NI service model
	niAPI := NewE2SMNIApi(r)
	if err := r.RegisterServiceModel(niAPI); err != nil {
		log.Printf("Failed to register E2SM-NI: %v", err)
	}
	
	log.Printf("Initialized %d service models", len(r.serviceModels))
}

func (r *ServiceModelRegistry) updateStatistics(serviceModelType ServiceModelType, success bool, isControl bool, processingTime time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	stats, exists := r.statistics[serviceModelType]
	if !exists {
		return
	}
	
	if success {
		if isControl {
			stats.ControlsProcessed++
		} else {
			stats.IndicationsProcessed++
		}
	} else {
		stats.ProcessingErrors++
	}
	
	// Update processing time statistics
	stats.TotalProcessingTime += processingTime
	totalOps := stats.IndicationsProcessed + stats.ControlsProcessed
	if totalOps > 0 {
		stats.AverageProcessingTime = stats.TotalProcessingTime / time.Duration(totalOps)
	}
	
	stats.LastProcessedAt = time.Now()
}

func (r *ServiceModelRegistry) incrementValidationErrors(serviceModelType ServiceModelType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if stats, exists := r.statistics[serviceModelType]; exists {
		stats.ValidationErrors++
	}
}

func (r *ServiceModelRegistry) getServiceModelVersion(serviceModelType ServiceModelType) string {
	switch serviceModelType {
	case ServiceModelTypeKPM:
		return "v2.0"
	case ServiceModelTypeRC:
		return "v1.0"
	case ServiceModelTypeNI:
		return "v1.0"
	default:
		return "v1.0"
	}
}

func (r *ServiceModelRegistry) getSupportedMessageTypes(serviceModelType ServiceModelType) []string {
	switch serviceModelType {
	case ServiceModelTypeKPM:
		return []string{
			"kmp-indication-header",
			"kmp-indication-message",
		}
	case ServiceModelTypeRC:
		return []string{
			"rc-indication-header",
			"rc-indication-message",
			"rc-control-header",
			"rc-control-message",
		}
	case ServiceModelTypeNI:
		return []string{
			"ni-indication-header",
			"ni-indication-message",
		}
	default:
		return []string{}
	}
}

func (r *ServiceModelRegistry) supportsControl(serviceModelType ServiceModelType) bool {
	switch serviceModelType {
	case ServiceModelTypeKPM:
		return false // KPM is indication-only
	case ServiceModelTypeRC:
		return true // RC supports control operations
	case ServiceModelTypeNI:
		return false // NI is primarily indication-only
	default:
		return false
	}
}

// Data structures for compatibility

// E2SMKPMIndicationHeader represents KPM indication header
type E2SMKPMIndicationHeader struct {
	CollectionStartTime string `json:"collectionStartTime"`
	FileFormatVersion   string `json:"fileFormatVersion"`
	SenderName          string `json:"senderName"`
	SenderType          string `json:"senderType"`
	VendorName          string `json:"vendorName,omitempty"`
}

// E2SMKPMIndicationMessage represents KPM indication message
type E2SMKPMIndicationMessage struct {
	MeasurementData     []E2SMKPMMetrics `json:"measurementData"`
	GranularityPeriod   uint32           `json:"granularityPeriod"`
	MeasurementInfoList []MeasurementInfo `json:"measurementInfoList,omitempty"`
}

// E2SMKPMMetrics represents KPM measurement data
type E2SMKPMMetrics struct {
	MeasurementName  string      `json:"measurementName"`
	MeasurementType  string      `json:"measurementType"`
	MeasurementValue interface{} `json:"measurementValue"`
	MeasurementUnit  string      `json:"measurementUnit,omitempty"`
	Timestamp        time.Time   `json:"timestamp"`
	CellID           string      `json:"cellId,omitempty"`
	AdditionalInfo   map[string]interface{} `json:"additionalInfo,omitempty"`
}

// MeasurementInfo represents measurement information
type MeasurementInfo struct {
	MeasurementTypeID   uint32 `json:"measurementTypeId"`
	MeasurementTypeName string `json:"measurementTypeName"`
}

// E2SMRCIndicationHeader represents RC indication header
type E2SMRCIndicationHeader struct {
	RICIndicationHeaderFormat uint32 `json:"ricIndicationHeaderFormat"`
	UEIdentity               string `json:"ueIdentity,omitempty"`
	RANParameterID           uint32 `json:"ranParameterId,omitempty"`
	RANParameterName         string `json:"ranParameterName,omitempty"`
}

// E2SMRCIndicationMessage represents RC indication message
type E2SMRCIndicationMessage struct {
	RICIndicationMessageFormat uint32                 `json:"ricIndicationMessageFormat"`
	RANParameterList          []RANParameter         `json:"ranParameterList"`
	AdditionalInfo            map[string]interface{} `json:"additionalInfo,omitempty"`
}

// E2SMRCControlHeader represents RC control header
type E2SMRCControlHeader struct {
	RICControlHeaderFormat uint32 `json:"ricControlHeaderFormat"`
	UEIdentity            string `json:"ueIdentity,omitempty"`
	RANParameterID        uint32 `json:"ranParameterId,omitempty"`
	RANParameterName      string `json:"ranParameterName,omitempty"`
	ControlType           string `json:"controlType"`
	ControlAction         string `json:"controlAction"`
}

// E2SMRCControlMessage represents RC control message
type E2SMRCControlMessage struct {
	RICControlMessageFormat uint32                 `json:"ricControlMessageFormat"`
	RANParameterList       []RANParameter         `json:"ranParameterList"`
	ControlAction          string                 `json:"controlAction"`
	ControlOutcome         string                 `json:"controlOutcome,omitempty"`
	AdditionalInfo         map[string]interface{} `json:"additionalInfo,omitempty"`
}

// RANParameter represents a RAN parameter
type RANParameter struct {
	ID    uint32      `json:"id"`
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}

// E2SMNIIndicationHeader represents NI indication header
type E2SMNIIndicationHeader struct {
	InterfaceType      string `json:"interfaceType"`
	InterfaceID        string `json:"interfaceId"`
	InterfaceDirection string `json:"interfaceDirection"`
	Timestamp          string `json:"timestamp"`
}

// E2SMNIIndicationMessage represents NI indication message
type E2SMNIIndicationMessage struct {
	InterfaceMessage string                 `json:"interfaceMessage"`
	MessageType      string                 `json:"messageType"`
	ProtocolIEs      []ProtocolIE          `json:"protocolIEs,omitempty"`
	AdditionalInfo   map[string]interface{} `json:"additionalInfo,omitempty"`
}

// ProtocolIE represents a protocol information element
type ProtocolIE struct {
	ID          uint32      `json:"id"`
	Criticality string      `json:"criticality"`
	Value       interface{} `json:"value"`
	TypeName    string      `json:"typeName"`
}