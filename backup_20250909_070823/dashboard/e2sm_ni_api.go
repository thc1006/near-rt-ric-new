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
	"time"
)

// E2SMNIApi implements the ServiceModelAPI interface for E2SM-NI
type E2SMNIApi struct {
	registry  *ServiceModelRegistry
	validator *MessageValidator
}

// NewE2SMNIApi creates a new E2SM-NI API instance
func NewE2SMNIApi(registry *ServiceModelRegistry) *E2SMNIApi {
	api := &E2SMNIApi{
		registry:  registry,
		validator: NewMessageValidator(),
	}
	
	// Register message schemas
	api.registerSchemas()
	
	return api
}

// GetServiceModelType returns the service model type
func (api *E2SMNIApi) GetServiceModelType() ServiceModelType {
	return ServiceModelTypeNI
}

// ValidateMessage validates an NI message
func (api *E2SMNIApi) ValidateMessage(messageType string, data []byte) error {
	return api.validator.ValidateMessage(messageType, data)
}

// ProcessIndication processes an NI indication message
func (api *E2SMNIApi) ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	startTime := time.Now()
	
	// Validate header
	if err := api.ValidateMessage("ni-indication-header", header); err != nil {
		return nil, fmt.Errorf("invalid NI indication header: %w", err)
	}
	
	// Validate message
	if err := api.ValidateMessage("ni-indication-message", message); err != nil {
		return nil, fmt.Errorf("invalid NI indication message: %w", err)
	}
	
	// Parse header and message
	indicationHeader, indicationMessage, err := api.registry.ProcessNIIndication(header, message)
	if err != nil {
		return nil, fmt.Errorf("failed to process NI indication: %w", err)
	}
	
	// Process interface message
	processedData := api.processInterfaceMessage(indicationHeader, indicationMessage)
	
	// Create response
	response := &NIIndicationResponse{
		Header:         indicationHeader,
		Message:        indicationMessage,
		ProcessedData:  processedData,
		ProcessingTime: time.Since(startTime),
		Timestamp:      time.Now(),
	}
	
	log.Printf("Processed NI indication for interface %s in %v", 
		indicationHeader.InterfaceType, response.ProcessingTime)
	
	return response, nil
}

// ProcessControl processes an NI control message (not supported for NI)
func (api *E2SMNIApi) ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	return nil, fmt.Errorf("control operations not supported for E2SM-NI service model")
}

// GetSupportedOperations returns supported operations for NI
func (api *E2SMNIApi) GetSupportedOperations() []string {
	return []string{
		"indication-processing",
		"interface-monitoring",
		"protocol-analysis",
		"traffic-inspection",
		"performance-monitoring",
		"packet-capture",
		"flow-analysis",
		"latency-measurement",
		"throughput-analysis",
		"error-detection",
		"quality-assessment",
		"network-diagnostics",
	}
}

// GetMessageSchema returns the JSON schema for NI message validation
func (api *E2SMNIApi) GetMessageSchema(messageType string) (map[string]interface{}, error) {
	schema, exists := api.validator.GetSchema(messageType)
	if !exists {
		return nil, fmt.Errorf("no schema found for message type: %s", messageType)
	}
	
	return schema, nil
}

// registerSchemas registers JSON schemas for NI message validation
func (api *E2SMNIApi) registerSchemas() {
	// NI Indication Header Schema
	niHeaderSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"interfaceType", "interfaceId", "interfaceDirection", "timestamp"},
		"properties": map[string]interface{}{
			"interfaceType": map[string]interface{}{
				"type": "string",
				"enum": []string{"E1", "F1-C", "F1-U", "Xn-C", "Xn-U", "NG-C", "NG-U", "X2-C", "X2-U"},
			},
			"interfaceId": map[string]interface{}{
				"type": "string",
			},
			"interfaceDirection": map[string]interface{}{
				"type": "string",
				"enum": []string{"INGRESS", "EGRESS", "BIDIRECTIONAL"},
			},
			"timestamp": map[string]interface{}{
				"type": "string",
				"format": "date-time",
			},
		},
	}
	api.validator.RegisterSchema("ni-indication-header", niHeaderSchema)
	
	// NI Indication Message Schema
	niMessageSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"interfaceMessage", "messageType"},
		"properties": map[string]interface{}{
			"interfaceMessage": map[string]interface{}{
				"type": "string",
				"format": "base64",
			},
			"messageType": map[string]interface{}{
				"type": "string",
			},
			"protocolIEs": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"required": []string{"id", "criticality", "value", "typeName"},
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "number",
						},
						"criticality": map[string]interface{}{
							"type": "string",
							"enum": []string{"reject", "ignore", "notify"},
						},
						"value": map[string]interface{}{
							"type": "number",
						},
						"typeName": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			"additionalInfo": map[string]interface{}{
				"type": "object",
			},
		},
	}
	api.validator.RegisterSchema("ni-indication-message", niMessageSchema)
	
	log.Println("Registered E2SM-NI message schemas")
}

// processInterfaceMessage processes interface message data
func (api *E2SMNIApi) processInterfaceMessage(header *E2SMNIIndicationHeader, message *E2SMNIIndicationMessage) *NIProcessedData {
	processedData := &NIProcessedData{
		InterfaceType:      header.InterfaceType,
		InterfaceID:        header.InterfaceID,
		InterfaceDirection: header.InterfaceDirection,
		MessageType:        message.MessageType,
		ProcessedAt:        time.Now(),
		Statistics:         make(map[string]interface{}),
		ProtocolInfo:       make(map[string]interface{}),
	}
	
	// Process protocol IEs
	if len(message.ProtocolIEs) > 0 {
		processedData.ProtocolInfo["ie_count"] = len(message.ProtocolIEs)
		
		// Categorize IEs by criticality
		criticalityCount := make(map[string]int)
		for _, ie := range message.ProtocolIEs {
			criticalityCount[ie.Criticality]++
			
			// Store IE information
			ieKey := fmt.Sprintf("ie_%d_%s", ie.ID, ie.TypeName)
			processedData.ProtocolInfo[ieKey] = map[string]interface{}{
				"id":          ie.ID,
				"criticality": ie.Criticality,
				"value":       ie.Value,
				"typeName":    ie.TypeName,
			}
		}
		processedData.ProtocolInfo["criticality_distribution"] = criticalityCount
	}
	
	// Calculate message statistics
	processedData.Statistics["message_size"] = len(message.InterfaceMessage)
	processedData.Statistics["processing_timestamp"] = time.Now().Unix()
	
	// Analyze interface type specific metrics
	switch header.InterfaceType {
	case "F1-C", "F1-U":
		processedData.Statistics["interface_family"] = "F1"
		processedData.Statistics["is_control_plane"] = header.InterfaceType == "F1-C"
		
	case "Xn-C", "Xn-U":
		processedData.Statistics["interface_family"] = "Xn"
		processedData.Statistics["is_control_plane"] = header.InterfaceType == "Xn-C"
		
	case "NG-C", "NG-U":
		processedData.Statistics["interface_family"] = "NG"
		processedData.Statistics["is_control_plane"] = header.InterfaceType == "NG-C"
		
	case "X2-C", "X2-U":
		processedData.Statistics["interface_family"] = "X2"
		processedData.Statistics["is_control_plane"] = header.InterfaceType == "X2-C"
		
	case "E1":
		processedData.Statistics["interface_family"] = "E1"
		processedData.Statistics["is_control_plane"] = true
	}
	
	// Add direction-specific processing
	processedData.Statistics["is_ingress"] = header.InterfaceDirection == "INGRESS"
	processedData.Statistics["is_egress"] = header.InterfaceDirection == "EGRESS"
	processedData.Statistics["is_bidirectional"] = header.InterfaceDirection == "BIDIRECTIONAL"
	
	return processedData
}

// GetInterfaceDefinitions returns standard interface definitions for E2SM-NI
func (api *E2SMNIApi) GetInterfaceDefinitions() []InterfaceDefinition {
	return []InterfaceDefinition{
		{
			InterfaceType: "E1",
			Description:   "Interface between gNB-CU-CP and gNB-CU-UP",
			Protocol:      "E1AP",
			Direction:     "BIDIRECTIONAL",
			Category:      "Internal gNB",
		},
		{
			InterfaceType: "F1-C",
			Description:   "Control plane interface between gNB-CU-CP and gNB-DU",
			Protocol:      "F1AP",
			Direction:     "BIDIRECTIONAL",
			Category:      "Control Plane",
		},
		{
			InterfaceType: "F1-U",
			Description:   "User plane interface between gNB-CU-UP and gNB-DU",
			Protocol:      "GTP-U",
			Direction:     "BIDIRECTIONAL",
			Category:      "User Plane",
		},
		{
			InterfaceType: "Xn-C",
			Description:   "Control plane interface between gNBs",
			Protocol:      "XnAP",
			Direction:     "BIDIRECTIONAL",
			Category:      "Inter-gNB Control",
		},
		{
			InterfaceType: "Xn-U",
			Description:   "User plane interface between gNBs",
			Protocol:      "GTP-U",
			Direction:     "BIDIRECTIONAL",
			Category:      "Inter-gNB User",
		},
		{
			InterfaceType: "NG-C",
			Description:   "Control plane interface between gNB and AMF",
			Protocol:      "NGAP",
			Direction:     "BIDIRECTIONAL",
			Category:      "Core Network Control",
		},
		{
			InterfaceType: "NG-U",
			Description:   "User plane interface between gNB and UPF",
			Protocol:      "GTP-U",
			Direction:     "BIDIRECTIONAL",
			Category:      "Core Network User",
		},
		{
			InterfaceType: "X2-C",
			Description:   "Control plane interface between eNBs (legacy)",
			Protocol:      "X2AP",
			Direction:     "BIDIRECTIONAL",
			Category:      "Legacy Inter-eNB Control",
		},
		{
			InterfaceType: "X2-U",
			Description:   "User plane interface between eNBs (legacy)",
			Protocol:      "GTP-U",
			Direction:     "BIDIRECTIONAL",
			Category:      "Legacy Inter-eNB User",
		},
	}
}

// GetSupportedProtocols returns supported protocols for interface analysis
func (api *E2SMNIApi) GetSupportedProtocols() []ProtocolDefinition {
	return []ProtocolDefinition{
		{
			Name:        "E1AP",
			Description: "E1 Application Protocol",
			Version:     "16.4.0",
			Standard:    "3GPP TS 38.463",
		},
		{
			Name:        "F1AP",
			Description: "F1 Application Protocol",
			Version:     "16.4.0",
			Standard:    "3GPP TS 38.473",
		},
		{
			Name:        "XnAP",
			Description: "Xn Application Protocol",
			Version:     "16.4.0",
			Standard:    "3GPP TS 38.423",
		},
		{
			Name:        "NGAP",
			Description: "NG Application Protocol",
			Version:     "16.4.0",
			Standard:    "3GPP TS 38.413",
		},
		{
			Name:        "X2AP",
			Description: "X2 Application Protocol",
			Version:     "16.4.0",
			Standard:    "3GPP TS 36.423",
		},
		{
			Name:        "GTP-U",
			Description: "GPRS Tunnelling Protocol User Plane",
			Version:     "16.4.0",
			Standard:    "3GPP TS 29.281",
		},
	}
}

// NIIndicationResponse represents the response from processing an NI indication
type NIIndicationResponse struct {
	Header         *E2SMNIIndicationHeader `json:"header"`
	Message        *E2SMNIIndicationMessage `json:"message"`
	ProcessedData  *NIProcessedData        `json:"processedData"`
	ProcessingTime time.Duration           `json:"processingTime"`
	Timestamp      time.Time               `json:"timestamp"`
}

// NIProcessedData represents processed NI indication data
type NIProcessedData struct {
	InterfaceType      string                 `json:"interfaceType"`
	InterfaceID        string                 `json:"interfaceId"`
	InterfaceDirection string                 `json:"interfaceDirection"`
	MessageType        string                 `json:"messageType"`
	ProcessedAt        time.Time              `json:"processedAt"`
	Statistics         map[string]interface{} `json:"statistics"`
	ProtocolInfo       map[string]interface{} `json:"protocolInfo"`
}

// InterfaceDefinition represents an interface definition
type InterfaceDefinition struct {
	InterfaceType string `json:"interfaceType"`
	Description   string `json:"description"`
	Protocol      string `json:"protocol"`
	Direction     string `json:"direction"`
	Category      string `json:"category"`
}

// ProtocolDefinition represents a protocol definition
type ProtocolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Standard    string `json:"standard"`
}