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
	"sync"
	"time"
)

// ServiceModelInterface provides a generic interface for service model operations
// Renamed to avoid conflict with ServiceModelAPI struct in types.go
type ServiceModelInterface interface {
	// GetServiceModelType returns the service model type
	GetServiceModelType() ServiceModelType
	
	// ValidateMessage validates a service model message
	ValidateMessage(messageType string, data []byte) error
	
	// ProcessIndication processes an indication message
	ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error)
	
	// ProcessControl processes a control message (if supported)
	ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error)
	
	// GetSupportedOperations returns supported operations for this service model
	GetSupportedOperations() []string
	
	// GetMessageSchema returns the JSON schema for message validation
	GetMessageSchema(messageType string) (map[string]interface{}, error)
}

// ServiceModelAPIManager manages all service model APIs
type ServiceModelAPIManager struct {
	apis     map[ServiceModelType]ServiceModelInterface
	registry *ServiceModelRegistry
	mu       sync.RWMutex
}

// NewServiceModelAPIManager creates a new service model API manager
func NewServiceModelAPIManager(registry *ServiceModelRegistry) *ServiceModelAPIManager {
	manager := &ServiceModelAPIManager{
		apis:     make(map[ServiceModelType]ServiceModelAPI),
		registry: registry,
	}
	
	// Initialize standard service model APIs
	manager.initializeStandardAPIs()
	
	return manager
}

// initializeStandardAPIs initializes the standard O-RAN service model APIs
func (m *ServiceModelAPIManager) initializeStandardAPIs() {
	// Register E2SM-KPM API
	kmpAPI := NewE2SMKPMApi(m.registry)
	m.apis[ServiceModelTypeKPM] = kmpAPI
	
	// Register E2SM-RC API
	rcAPI := NewE2SMRCApi(m.registry)
	m.apis[ServiceModelTypeRC] = rcAPI
	
	// Register E2SM-NI API
	niAPI := NewE2SMNIApi(m.registry)
	m.apis[ServiceModelTypeNI] = niAPI
	
	log.Println("Initialized standard service model APIs")
}

// RegisterAPI registers a new service model API
func (m *ServiceModelAPIManager) RegisterAPI(api ServiceModelAPI) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	modelType := api.GetServiceModelType()
	if _, exists := m.apis[modelType]; exists {
		return fmt.Errorf("service model API already registered for type: %s", modelType)
	}
	
	m.apis[modelType] = api
	log.Printf("Registered service model API for type: %s", modelType)
	return nil
}

// GetAPI returns the API for a specific service model type
func (m *ServiceModelAPIManager) GetAPI(modelType ServiceModelType) (ServiceModelAPI, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	api, exists := m.apis[modelType]
	if !exists {
		return nil, fmt.Errorf("no API registered for service model type: %s", modelType)
	}
	
	return api, nil
}

// ProcessIndication processes an indication message using the appropriate API
func (m *ServiceModelAPIManager) ProcessIndication(ctx context.Context, modelType ServiceModelType, header []byte, message []byte) (interface{}, error) {
	api, err := m.GetAPI(modelType)
	if err != nil {
		return nil, err
	}
	
	return api.ProcessIndication(ctx, header, message)
}

// ProcessControl processes a control message using the appropriate API
func (m *ServiceModelAPIManager) ProcessControl(ctx context.Context, modelType ServiceModelType, header []byte, message []byte) (interface{}, error) {
	api, err := m.GetAPI(modelType)
	if err != nil {
		return nil, err
	}
	
	return api.ProcessControl(ctx, header, message)
}

// ValidateMessage validates a message using the appropriate API
func (m *ServiceModelAPIManager) ValidateMessage(modelType ServiceModelType, messageType string, data []byte) error {
	api, err := m.GetAPI(modelType)
	if err != nil {
		return err
	}
	
	return api.ValidateMessage(messageType, data)
}

// GetSupportedOperations returns all supported operations across all APIs
func (m *ServiceModelAPIManager) GetSupportedOperations() map[ServiceModelType][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	operations := make(map[ServiceModelType][]string)
	for modelType, api := range m.apis {
		operations[modelType] = api.GetSupportedOperations()
	}
	
	return operations
}

// GetMessageSchema returns the message schema for a specific service model and message type
func (m *ServiceModelAPIManager) GetMessageSchema(modelType ServiceModelType, messageType string) (map[string]interface{}, error) {
	api, err := m.GetAPI(modelType)
	if err != nil {
		return nil, err
	}
	
	return api.GetMessageSchema(messageType)
}

// ServiceModelMessage represents a generic service model message
type ServiceModelMessage struct {
	ServiceModelOID string                 `json:"serviceModelOid"`
	MessageType     string                 `json:"messageType"`
	Header          json.RawMessage        `json:"header"`
	Message         json.RawMessage        `json:"message"`
	Timestamp       time.Time              `json:"timestamp"`
	NodeID          string                 `json:"nodeId,omitempty"`
	SubscriptionID  string                 `json:"subscriptionId,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ServiceModelResponse represents a generic service model response
type ServiceModelResponse struct {
	ServiceModelOID string                 `json:"serviceModelOid"`
	MessageType     string                 `json:"messageType"`
	Status          string                 `json:"status"`
	Result          interface{}            `json:"result,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	ProcessingTime  time.Duration          `json:"processingTime"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// MessageValidator provides validation capabilities for service model messages
type MessageValidator struct {
	schemas map[string]map[string]interface{}
	mu      sync.RWMutex
}

// NewMessageValidator creates a new message validator
func NewMessageValidator() *MessageValidator {
	return &MessageValidator{
		schemas: make(map[string]map[string]interface{}),
	}
}

// RegisterSchema registers a JSON schema for message validation
func (v *MessageValidator) RegisterSchema(messageType string, schema map[string]interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	v.schemas[messageType] = schema
	log.Printf("Registered schema for message type: %s", messageType)
}

// ValidateMessage validates a message against its registered schema
func (v *MessageValidator) ValidateMessage(messageType string, data []byte) error {
	v.mu.RLock()
	schema, exists := v.schemas[messageType]
	v.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("no schema registered for message type: %s", messageType)
	}
	
	// Parse the message data
	var messageData interface{}
	if err := json.Unmarshal(data, &messageData); err != nil {
		return fmt.Errorf("invalid JSON in message: %w", err)
	}
	
	// Perform basic schema validation
	if err := v.validateAgainstSchema(messageData, schema); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	
	return nil
}

// validateAgainstSchema performs basic JSON schema validation
func (v *MessageValidator) validateAgainstSchema(data interface{}, schema map[string]interface{}) error {
	// This is a simplified schema validation
	// In production, you would use a proper JSON schema validator like github.com/xeipuuv/gojsonschema
	
	schemaType, ok := schema["type"].(string)
	if !ok {
		return fmt.Errorf("schema must have a type field")
	}
	
	switch schemaType {
	case "object":
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected object, got %T", data)
		}
		
		// Check required fields
		if required, exists := schema["required"].([]interface{}); exists {
			for _, field := range required {
				fieldName := field.(string)
				if _, fieldExists := dataMap[fieldName]; !fieldExists {
					return fmt.Errorf("required field missing: %s", fieldName)
				}
			}
		}
		
		// Validate properties
		if properties, exists := schema["properties"].(map[string]interface{}); exists {
			for fieldName, fieldData := range dataMap {
				if fieldSchema, fieldExists := properties[fieldName]; fieldExists {
					if err := v.validateAgainstSchema(fieldData, fieldSchema.(map[string]interface{})); err != nil {
						return fmt.Errorf("field %s: %w", fieldName, err)
					}
				}
			}
		}
		
	case "array":
		dataArray, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("expected array, got %T", data)
		}
		
		// Validate array items
		if items, exists := schema["items"].(map[string]interface{}); exists {
			for i, item := range dataArray {
				if err := v.validateAgainstSchema(item, items); err != nil {
					return fmt.Errorf("array item %d: %w", i, err)
				}
			}
		}
		
	case "string":
		if _, ok := data.(string); !ok {
			return fmt.Errorf("expected string, got %T", data)
		}
		
	case "number":
		switch data.(type) {
		case float64, int, int32, int64:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", data)
		}
		
	case "boolean":
		if _, ok := data.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", data)
		}
		
	default:
		return fmt.Errorf("unsupported schema type: %s", schemaType)
	}
	
	return nil
}

// GetSchema returns the schema for a message type
func (v *MessageValidator) GetSchema(messageType string) (map[string]interface{}, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	schema, exists := v.schemas[messageType]
	return schema, exists
}

// ListSchemas returns all registered schemas
func (v *MessageValidator) ListSchemas() map[string]map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]map[string]interface{})
	for messageType, schema := range v.schemas {
		result[messageType] = schema
	}
	
	return result
}