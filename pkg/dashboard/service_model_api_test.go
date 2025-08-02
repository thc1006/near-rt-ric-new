/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package dashboard

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestServiceModelAPIManager(t *testing.T) {
	registry := NewServiceModelRegistry()
	manager := NewServiceModelAPIManager(registry)

	// Test API registration
	apis := manager.GetSupportedOperations()
	if len(apis) != 3 {
		t.Errorf("Expected 3 APIs, got %d", len(apis))
	}

	// Test KPM API
	kmpAPI, err := manager.GetAPI(ServiceModelTypeKPM)
	if err != nil {
		t.Fatalf("Failed to get KPM API: %v", err)
	}

	if kmpAPI.GetServiceModelType() != ServiceModelTypeKPM {
		t.Errorf("Expected KPM service model type, got %s", kmpAPI.GetServiceModelType())
	}

	// Test RC API
	rcAPI, err := manager.GetAPI(ServiceModelTypeRC)
	if err != nil {
		t.Fatalf("Failed to get RC API: %v", err)
	}

	if rcAPI.GetServiceModelType() != ServiceModelTypeRC {
		t.Errorf("Expected RC service model type, got %s", rcAPI.GetServiceModelType())
	}

	// Test NI API
	niAPI, err := manager.GetAPI(ServiceModelTypeNI)
	if err != nil {
		t.Fatalf("Failed to get NI API: %v", err)
	}

	if niAPI.GetServiceModelType() != ServiceModelTypeNI {
		t.Errorf("Expected NI service model type, got %s", niAPI.GetServiceModelType())
	}
}

func TestE2SMKPMApi(t *testing.T) {
	registry := NewServiceModelRegistry()
	api := NewE2SMKPMApi(registry)

	// Test service model type
	if api.GetServiceModelType() != ServiceModelTypeKPM {
		t.Errorf("Expected KPM service model type, got %s", api.GetServiceModelType())
	}

	// Test supported operations
	operations := api.GetSupportedOperations()
	if len(operations) == 0 {
		t.Error("Expected supported operations, got none")
	}

	// Test KPI definitions
	kpiDefs := api.GetKPIDefinitions()
	if len(kpiDefs) == 0 {
		t.Error("Expected KPI definitions, got none")
	}

	// Test message validation
	validHeader := map[string]interface{}{
		"collectionStartTime": time.Now().Format(time.RFC3339),
		"fileFormatVersion":   "1.0",
		"senderName":          "test-sender",
		"senderType":          "gNB",
	}
	headerBytes, _ := json.Marshal(validHeader)

	err := api.ValidateMessage("kmp-indication-header", headerBytes)
	if err != nil {
		t.Errorf("Valid header should pass validation: %v", err)
	}

	// Test invalid message validation
	invalidHeader := map[string]interface{}{
		"invalidField": "invalid",
	}
	invalidHeaderBytes, _ := json.Marshal(invalidHeader)

	err = api.ValidateMessage("kmp-indication-header", invalidHeaderBytes)
	if err == nil {
		t.Error("Invalid header should fail validation")
	}

	// Test indication processing
	validMessage := map[string]interface{}{
		"measurementData": []map[string]interface{}{
			{
				"measurementName":  "DL_PRB_Usage",
				"measurementType":  "percentage",
				"measurementValue": 75.5,
				"measurementUnit":  "%",
				"timestamp":        time.Now().Format(time.RFC3339),
				"cellId":           "cell-001",
			},
		},
		"granularityPeriod": 1000,
		"measurementInfoList": []map[string]interface{}{
			{
				"measurementTypeId":   1,
				"measurementTypeName": "DL_PRB_Usage",
			},
		},
	}
	messageBytes, _ := json.Marshal(validMessage)

	ctx := context.Background()
	result, err := api.ProcessIndication(ctx, headerBytes, messageBytes)
	if err != nil {
		t.Errorf("Failed to process valid indication: %v", err)
	}

	response, ok := result.(*KPMIndicationResponse)
	if !ok {
		t.Error("Expected KPMIndicationResponse")
	}

	if len(response.ProcessedMetrics) == 0 {
		t.Error("Expected processed metrics")
	}

	// Test control processing (should fail for KPM)
	_, err = api.ProcessControl(ctx, headerBytes, messageBytes)
	if err == nil {
		t.Error("Control processing should fail for KPM")
	}
}

func TestE2SMRCApi(t *testing.T) {
	registry := NewServiceModelRegistry()
	api := NewE2SMRCApi(registry)

	// Test service model type
	if api.GetServiceModelType() != ServiceModelTypeRC {
		t.Errorf("Expected RC service model type, got %s", api.GetServiceModelType())
	}

	// Test supported operations
	operations := api.GetSupportedOperations()
	if len(operations) == 0 {
		t.Error("Expected supported operations, got none")
	}

	// Test control action definitions
	controlDefs := api.GetControlActionDefinitions()
	if len(controlDefs) == 0 {
		t.Error("Expected control action definitions, got none")
	}

	// Test control message validation and processing
	validControlHeader := map[string]interface{}{
		"ricControlHeaderFormat": 1,
		"ueIdentity":            "ue-001",
		"ranParameterId":        123,
		"ranParameterName":      "QoS_Parameter",
	}
	headerBytes, _ := json.Marshal(validControlHeader)

	validControlMessage := map[string]interface{}{
		"ricControlMessageFormat": 1,
		"ranParameters": []map[string]interface{}{
			{
				"id":    123,
				"name":  "QoS_Parameter",
				"value": 5,
				"type":  "integer",
			},
		},
		"controlAction": "QOS_CONTROL",
	}
	messageBytes, _ := json.Marshal(validControlMessage)

	ctx := context.Background()
	result, err := api.ProcessControl(ctx, headerBytes, messageBytes)
	if err != nil {
		t.Errorf("Failed to process valid control: %v", err)
	}

	response, ok := result.(*RCControlResponse)
	if !ok {
		t.Error("Expected RCControlResponse")
	}

	if response.ControlResult.Status != "SUCCESS" {
		t.Errorf("Expected SUCCESS status, got %s", response.ControlResult.Status)
	}
}

func TestE2SMNIApi(t *testing.T) {
	registry := NewServiceModelRegistry()
	api := NewE2SMNIApi(registry)

	// Test service model type
	if api.GetServiceModelType() != ServiceModelTypeNI {
		t.Errorf("Expected NI service model type, got %s", api.GetServiceModelType())
	}

	// Test supported operations
	operations := api.GetSupportedOperations()
	if len(operations) == 0 {
		t.Error("Expected supported operations, got none")
	}

	// Test interface definitions
	interfaceDefs := api.GetInterfaceDefinitions()
	if len(interfaceDefs) == 0 {
		t.Error("Expected interface definitions, got none")
	}

	// Test protocol definitions
	protocolDefs := api.GetSupportedProtocols()
	if len(protocolDefs) == 0 {
		t.Error("Expected protocol definitions, got none")
	}

	// Test indication processing
	validHeader := map[string]interface{}{
		"interfaceType":      "F1-C",
		"interfaceId":        "f1-interface-001",
		"interfaceDirection": "INGRESS",
		"timestamp":          time.Now().Format(time.RFC3339),
	}
	headerBytes, _ := json.Marshal(validHeader)

	validMessage := map[string]interface{}{
		"interfaceMessage": "dGVzdCBtZXNzYWdl", // base64 encoded "test message"
		"messageType":      "F1AP_SETUP_REQUEST",
		"protocolIEs": []map[string]interface{}{
			{
				"id":          1,
				"criticality": "reject",
				"value":       100,
				"typeName":    "GlobalGNB-ID",
			},
		},
	}
	messageBytes, _ := json.Marshal(validMessage)

	ctx := context.Background()
	result, err := api.ProcessIndication(ctx, headerBytes, messageBytes)
	if err != nil {
		t.Errorf("Failed to process valid indication: %v", err)
	}

	response, ok := result.(*NIIndicationResponse)
	if !ok {
		t.Error("Expected NIIndicationResponse")
	}

	if response.ProcessedData.InterfaceType != "F1-C" {
		t.Errorf("Expected F1-C interface type, got %s", response.ProcessedData.InterfaceType)
	}

	// Test control processing (should fail for NI)
	_, err = api.ProcessControl(ctx, headerBytes, messageBytes)
	if err == nil {
		t.Error("Control processing should fail for NI")
	}
}

func TestMessageValidator(t *testing.T) {
	validator := NewMessageValidator()

	// Test schema registration
	testSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"name", "value"},
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"value": map[string]interface{}{
				"type": "number",
			},
		},
	}

	validator.RegisterSchema("test-message", testSchema)

	// Test valid message
	validMessage := map[string]interface{}{
		"name":  "test",
		"value": 123,
	}
	validBytes, _ := json.Marshal(validMessage)

	err := validator.ValidateMessage("test-message", validBytes)
	if err != nil {
		t.Errorf("Valid message should pass validation: %v", err)
	}

	// Test invalid message (missing required field)
	invalidMessage := map[string]interface{}{
		"name": "test",
		// missing "value" field
	}
	invalidBytes, _ := json.Marshal(invalidMessage)

	err = validator.ValidateMessage("test-message", invalidBytes)
	if err == nil {
		t.Error("Invalid message should fail validation")
	}

	// Test invalid message (wrong type)
	wrongTypeMessage := map[string]interface{}{
		"name":  "test",
		"value": "not-a-number",
	}
	wrongTypeBytes, _ := json.Marshal(wrongTypeMessage)

	err = validator.ValidateMessage("test-message", wrongTypeBytes)
	if err == nil {
		t.Error("Wrong type message should fail validation")
	}

	// Test unknown message type
	err = validator.ValidateMessage("unknown-message", validBytes)
	if err == nil {
		t.Error("Unknown message type should fail validation")
	}
}

func TestServiceModelMessage(t *testing.T) {
	message := ServiceModelMessage{
		ServiceModelOID: "1.3.6.1.4.1.53148.1.2.2.2",
		MessageType:     "indication",
		Header:          json.RawMessage(`{"test": "header"}`),
		Message:         json.RawMessage(`{"test": "message"}`),
		Timestamp:       time.Now(),
		NodeID:          "node-001",
		SubscriptionID:  "sub-001",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(message)
	if err != nil {
		t.Errorf("Failed to marshal ServiceModelMessage: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled ServiceModelMessage
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ServiceModelMessage: %v", err)
	}

	if unmarshaled.ServiceModelOID != message.ServiceModelOID {
		t.Errorf("Expected OID %s, got %s", message.ServiceModelOID, unmarshaled.ServiceModelOID)
	}
}

func TestServiceModelResponse(t *testing.T) {
	response := ServiceModelResponse{
		ServiceModelOID: "1.3.6.1.4.1.53148.1.2.2.2",
		MessageType:     "indication",
		Status:          "success",
		Result:          map[string]interface{}{"processed": true},
		Timestamp:       time.Now(),
		ProcessingTime:  time.Millisecond * 100,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal ServiceModelResponse: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled ServiceModelResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ServiceModelResponse: %v", err)
	}

	if unmarshaled.Status != response.Status {
		t.Errorf("Expected status %s, got %s", response.Status, unmarshaled.Status)
	}
}