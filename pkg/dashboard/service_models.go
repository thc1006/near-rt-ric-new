/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ServiceModelType represents the type of service model
type ServiceModelType string

const (
	ServiceModelTypeKPM ServiceModelType = "E2SM-KPM"
	ServiceModelTypeRC  ServiceModelType = "E2SM-RC"
	ServiceModelTypeNI  ServiceModelType = "E2SM-NI"
)

// ServiceModelCapability represents a capability of a service model
type ServiceModelCapability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Supported   bool   `json:"supported"`
}

// ServiceModelDefinition represents a complete service model definition
type ServiceModelDefinition struct {
	OID           string                    `json:"oid"`
	Name          string                    `json:"name"`
	Type          ServiceModelType          `json:"type"`
	Version       string                    `json:"version"`
	Description   string                    `json:"description"`
	Capabilities  []ServiceModelCapability  `json:"capabilities"`
	RANFunctions  []RANFunction            `json:"ranFunctions"`
	LastUpdated   time.Time                `json:"lastUpdated"`
}

// E2SMKPMMetrics represents KPI measurement metrics
type E2SMKPMMetrics struct {
	MeasurementName   string                 `json:"measurementName"`
	MeasurementType   string                 `json:"measurementType"`
	MeasurementValue  interface{}            `json:"measurementValue"`
	MeasurementUnit   string                 `json:"measurementUnit"`
	Timestamp         time.Time              `json:"timestamp"`
	CellID            string                 `json:"cellId,omitempty"`
	AdditionalInfo    map[string]interface{} `json:"additionalInfo,omitempty"`
}

// E2SMKPMIndicationHeader represents KPM indication header
type E2SMKPMIndicationHeader struct {
	CollectionStartTime time.Time `json:"collectionStartTime"`
	FileFormatVersion   string    `json:"fileFormatVersion"`
	SenderName          string    `json:"senderName"`
	SenderType          string    `json:"senderType"`
	VendorName          string    `json:"vendorName"`
}

// E2SMKPMIndicationMessage represents KPM indication message
type E2SMKPMIndicationMessage struct {
	MeasurementData []E2SMKPMMetrics `json:"measurementData"`
	GranularityPeriod uint32         `json:"granularityPeriod"`
	MeasurementInfoList []MeasurementInfo `json:"measurementInfoList"`
}

// MeasurementInfo represents measurement information
type MeasurementInfo struct {
	MeasurementTypeID   uint32 `json:"measurementTypeId"`
	MeasurementTypeName string `json:"measurementTypeName"`
}

// E2SMRCControlHeader represents RC control header
type E2SMRCControlHeader struct {
	RICControlHeaderFormat uint32 `json:"ricControlHeaderFormat"`
	UEIdentity            string `json:"ueIdentity,omitempty"`
	RANParameterID        uint32 `json:"ranParameterId,omitempty"`
	RANParameterName      string `json:"ranParameterName,omitempty"`
}

// E2SMRCControlMessage represents RC control message
type E2SMRCControlMessage struct {
	RICControlMessageFormat uint32                 `json:"ricControlMessageFormat"`
	RANParameters          []RANParameter         `json:"ranParameters"`
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
	InterfaceType      string    `json:"interfaceType"`
	InterfaceID        string    `json:"interfaceId"`
	InterfaceDirection string    `json:"interfaceDirection"`
	Timestamp          time.Time `json:"timestamp"`
}

// E2SMNIIndicationMessage represents NI indication message
type E2SMNIIndicationMessage struct {
	InterfaceMessage []byte                 `json:"interfaceMessage"`
	MessageType      string                 `json:"messageType"`
	ProtocolIEs      []ProtocolIE          `json:"protocolIEs,omitempty"`
	AdditionalInfo   map[string]interface{} `json:"additionalInfo,omitempty"`
}

// ProtocolIE represents a protocol information element
type ProtocolIE struct {
	ID           uint32      `json:"id"`
	Criticality  string      `json:"criticality"`
	Value        interface{} `json:"value"`
	TypeName     string      `json:"typeName"`
}

// ServiceModelRegistry manages service model definitions and capabilities
type ServiceModelRegistry struct {
	models map[string]*ServiceModelDefinition
}

// NewServiceModelRegistry creates a new service model registry
func NewServiceModelRegistry() *ServiceModelRegistry {
	registry := &ServiceModelRegistry{
		models: make(map[string]*ServiceModelDefinition),
	}
	
	// Initialize with standard O-RAN service models
	registry.initializeStandardModels()
	
	return registry
}

// initializeStandardModels initializes the registry with standard O-RAN service models
func (r *ServiceModelRegistry) initializeStandardModels() {
	// E2SM-KPM Service Model
	kpmModel := &ServiceModelDefinition{
		OID:         "1.3.6.1.4.1.53148.1.2.2.2",
		Name:        "E2SM-KPM",
		Type:        ServiceModelTypeKPM,
		Version:     "2.0",
		Description: "E2 Service Model for Key Performance Measurement",
		Capabilities: []ServiceModelCapability{
			{
				Name:        "Measurement Reporting",
				Description: "Periodic and event-triggered measurement reporting",
				Version:     "2.0",
				Supported:   true,
			},
			{
				Name:        "KPI Collection",
				Description: "Collection of Key Performance Indicators",
				Version:     "2.0",
				Supported:   true,
			},
			{
				Name:        "Cell-level Measurements",
				Description: "Cell-specific performance measurements",
				Version:     "2.0",
				Supported:   true,
			},
			{
				Name:        "UE-level Measurements",
				Description: "User Equipment specific measurements",
				Version:     "2.0",
				Supported:   true,
			},
		},
		RANFunctions: []RANFunction{
			{
				ID:          1,
				OID:         "1.3.6.1.4.1.53148.1.2.2.2",
				Description: "KPM Measurement Reporting",
				Revision:    1,
			},
		},
		LastUpdated: time.Now(),
	}
	r.models[kpmModel.OID] = kpmModel

	// E2SM-RC Service Model
	rcModel := &ServiceModelDefinition{
		OID:         "1.3.6.1.4.1.53148.1.2.2.3",
		Name:        "E2SM-RC",
		Type:        ServiceModelTypeRC,
		Version:     "1.0",
		Description: "E2 Service Model for RAN Control",
		Capabilities: []ServiceModelCapability{
			{
				Name:        "RAN Control",
				Description: "Control of RAN functions and parameters",
				Version:     "1.0",
				Supported:   true,
			},
			{
				Name:        "Policy Enforcement",
				Description: "Enforcement of RAN policies",
				Version:     "1.0",
				Supported:   true,
			},
			{
				Name:        "Resource Management",
				Description: "Management of RAN resources",
				Version:     "1.0",
				Supported:   true,
			},
			{
				Name:        "QoS Control",
				Description: "Quality of Service control mechanisms",
				Version:     "1.0",
				Supported:   true,
			},
		},
		RANFunctions: []RANFunction{
			{
				ID:          2,
				OID:         "1.3.6.1.4.1.53148.1.2.2.3",
				Description: "RAN Control Functions",
				Revision:    1,
			},
		},
		LastUpdated: time.Now(),
	}
	r.models[rcModel.OID] = rcModel

	// E2SM-NI Service Model
	niModel := &ServiceModelDefinition{
		OID:         "1.3.6.1.4.1.53148.1.2.2.4",
		Name:        "E2SM-NI",
		Type:        ServiceModelTypeNI,
		Version:     "1.0",
		Description: "E2 Service Model for Network Interface",
		Capabilities: []ServiceModelCapability{
			{
				Name:        "Interface Monitoring",
				Description: "Monitoring of network interfaces",
				Version:     "1.0",
				Supported:   true,
			},
			{
				Name:        "Protocol Analysis",
				Description: "Analysis of protocol messages",
				Version:     "1.0",
				Supported:   true,
			},
			{
				Name:        "Traffic Inspection",
				Description: "Deep packet inspection capabilities",
				Version:     "1.0",
				Supported:   true,
			},
			{
				Name:        "Performance Monitoring",
				Description: "Interface performance monitoring",
				Version:     "1.0",
				Supported:   true,
			},
		},
		RANFunctions: []RANFunction{
			{
				ID:          3,
				OID:         "1.3.6.1.4.1.53148.1.2.2.4",
				Description: "Network Interface Functions",
				Revision:    1,
			},
		},
		LastUpdated: time.Now(),
	}
	r.models[niModel.OID] = niModel
}

// RegisterServiceModel registers a new service model
func (r *ServiceModelRegistry) RegisterServiceModel(model *ServiceModelDefinition) error {
	if model.OID == "" {
		return fmt.Errorf("service model OID cannot be empty")
	}
	
	model.LastUpdated = time.Now()
	r.models[model.OID] = model
	
	log.Printf("Registered service model: %s (%s)", model.Name, model.OID)
	return nil
}

// GetServiceModel retrieves a service model by OID
func (r *ServiceModelRegistry) GetServiceModel(oid string) (*ServiceModelDefinition, bool) {
	model, exists := r.models[oid]
	return model, exists
}

// GetAllServiceModels returns all registered service models
func (r *ServiceModelRegistry) GetAllServiceModels() map[string]*ServiceModelDefinition {
	// Return a copy to prevent external modification
	result := make(map[string]*ServiceModelDefinition)
	for oid, model := range r.models {
		result[oid] = model
	}
	return result
}

// GetServiceModelsByType returns service models of a specific type
func (r *ServiceModelRegistry) GetServiceModelsByType(modelType ServiceModelType) []*ServiceModelDefinition {
	var result []*ServiceModelDefinition
	for _, model := range r.models {
		if model.Type == modelType {
			result = append(result, model)
		}
	}
	return result
}

// ValidateServiceModel validates a service model definition
func (r *ServiceModelRegistry) ValidateServiceModel(model *ServiceModelDefinition) error {
	if model.OID == "" {
		return fmt.Errorf("service model OID is required")
	}
	
	if model.Name == "" {
		return fmt.Errorf("service model name is required")
	}
	
	if model.Version == "" {
		return fmt.Errorf("service model version is required")
	}
	
	if len(model.RANFunctions) == 0 {
		return fmt.Errorf("service model must have at least one RAN function")
	}
	
	// Validate RAN functions
	for _, ranFunc := range model.RANFunctions {
		if ranFunc.OID == "" {
			return fmt.Errorf("RAN function OID is required")
		}
		if ranFunc.ID == 0 {
			return fmt.Errorf("RAN function ID must be greater than 0")
		}
	}
	
	return nil
}

// ProcessKPMIndication processes a KPM indication message
func (r *ServiceModelRegistry) ProcessKPMIndication(header []byte, message []byte) (*E2SMKPMIndicationHeader, *E2SMKPMIndicationMessage, error) {
	var indicationHeader E2SMKPMIndicationHeader
	var indicationMessage E2SMKPMIndicationMessage
	
	// Parse header
	if err := json.Unmarshal(header, &indicationHeader); err != nil {
		return nil, nil, fmt.Errorf("failed to parse KMP indication header: %w", err)
	}
	
	// Parse message
	if err := json.Unmarshal(message, &indicationMessage); err != nil {
		return nil, nil, fmt.Errorf("failed to parse KMP indication message: %w", err)
	}
	
	log.Printf("Processed KPM indication with %d measurements", len(indicationMessage.MeasurementData))
	return &indicationHeader, &indicationMessage, nil
}

// ProcessRCControl processes an RC control message
func (r *ServiceModelRegistry) ProcessRCControl(header []byte, message []byte) (*E2SMRCControlHeader, *E2SMRCControlMessage, error) {
	var controlHeader E2SMRCControlHeader
	var controlMessage E2SMRCControlMessage
	
	// Parse header
	if err := json.Unmarshal(header, &controlHeader); err != nil {
		return nil, nil, fmt.Errorf("failed to parse RC control header: %w", err)
	}
	
	// Parse message
	if err := json.Unmarshal(message, &controlMessage); err != nil {
		return nil, nil, fmt.Errorf("failed to parse RC control message: %w", err)
	}
	
	log.Printf("Processed RC control with %d parameters", len(controlMessage.RANParameters))
	return &controlHeader, &controlMessage, nil
}

// ProcessNIIndication processes an NI indication message
func (r *ServiceModelRegistry) ProcessNIIndication(header []byte, message []byte) (*E2SMNIIndicationHeader, *E2SMNIIndicationMessage, error) {
	var indicationHeader E2SMNIIndicationHeader
	var indicationMessage E2SMNIIndicationMessage
	
	// Parse header
	if err := json.Unmarshal(header, &indicationHeader); err != nil {
		return nil, nil, fmt.Errorf("failed to parse NI indication header: %w", err)
	}
	
	// Parse message
	if err := json.Unmarshal(message, &indicationMessage); err != nil {
		return nil, nil, fmt.Errorf("failed to parse NI indication message: %w", err)
	}
	
	log.Printf("Processed NI indication for interface %s", indicationHeader.InterfaceType)
	return &indicationHeader, &indicationMessage, nil
}

// GetSupportedCapabilities returns all supported capabilities across all service models
func (r *ServiceModelRegistry) GetSupportedCapabilities() []ServiceModelCapability {
	var capabilities []ServiceModelCapability
	
	for _, model := range r.models {
		for _, capability := range model.Capabilities {
			if capability.Supported {
				capabilities = append(capabilities, capability)
			}
		}
	}
	
	return capabilities
}

// GetServiceModelStats returns statistics about registered service models
func (r *ServiceModelRegistry) GetServiceModelStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_models":      len(r.models),
		"models_by_type":    make(map[string]int),
		"total_functions":   0,
		"total_capabilities": 0,
	}
	
	modelsByType := stats["models_by_type"].(map[string]int)
	
	for _, model := range r.models {
		modelsByType[string(model.Type)]++
		stats["total_functions"] = stats["total_functions"].(int) + len(model.RANFunctions)
		stats["total_capabilities"] = stats["total_capabilities"].(int) + len(model.Capabilities)
	}
	
	return stats
}