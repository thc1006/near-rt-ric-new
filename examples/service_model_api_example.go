//go:build service_model_api_example

/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0

Example demonstrating the Service Model API Development implementation.
This example shows how to use the E2SM-KPM, E2SM-RC, and E2SM-NI APIs
for performance monitoring, RAN control, and network interface management.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/oran/near-rt-ric-new/pkg/dashboard"
)

func main() {
	fmt.Println("=== Service Model API Development Example ===")

	// Initialize service model registry and API manager
	registry := dashboard.NewServiceModelRegistry()
	apiManager := dashboard.NewServiceModelAPIManager(registry)

	// Demonstrate E2SM-KPM API usage
	demonstrateKPMAPI(apiManager)

	// Demonstrate E2SM-RC API usage
	demonstrateRCAPI(apiManager)

	// Demonstrate E2SM-NI API usage
	demonstrateNIAPI(apiManager)

	// Demonstrate generic service model operations
	demonstrateGenericOperations(apiManager)

	fmt.Println("\n=== Service Model API Example Complete ===")
}

func demonstrateKPMAPI(apiManager *dashboard.ServiceModelAPIManager) {
	fmt.Println("\n--- E2SM-KPM API Demonstration ---")

	// Get KPM API
	kmpAPI, err := apiManager.GetAPI(dashboard.ServiceModelTypeKPM)
	if err != nil {
		log.Fatalf("Failed to get KPM API: %v", err)
	}

	// Show supported operations
	operations := kmpAPI.GetSupportedOperations()
	fmt.Printf("KPM Supported Operations: %v\n", operations)

	// Get KPI definitions
	if kmpAPITyped, ok := kmpAPI.(*dashboard.E2SMKPMApi); ok {
		kpiDefs := kmpAPITyped.GetKPIDefinitions()
		fmt.Printf("Available KPIs: %d\n", len(kpiDefs))
		for _, kpi := range kpiDefs[:3] { // Show first 3
			fmt.Printf("  - %s: %s (%s)\n", kpi.Name, kpi.Description, kpi.Unit)
		}
	}

	// Create sample KPM indication header
	kmpHeader := map[string]interface{}{
		"collectionStartTime": time.Now().Format(time.RFC3339),
		"fileFormatVersion":   "1.0",
		"senderName":          "gNB-001",
		"senderType":          "gNB",
		"vendorName":          "Example Vendor",
	}
	headerBytes, _ := json.Marshal(kmpHeader)

	// Create sample KPM indication message with multiple measurements
	kmpMessage := map[string]interface{}{
		"measurementData": []map[string]interface{}{
			{
				"measurementName":  "DL_PRB_Usage",
				"measurementType":  "percentage",
				"measurementValue": 75.5,
				"measurementUnit":  "%",
				"timestamp":        time.Now().Format(time.RFC3339),
				"cellId":           "cell-001",
			},
			{
				"measurementName":  "UL_PRB_Usage",
				"measurementType":  "percentage",
				"measurementValue": 68.2,
				"measurementUnit":  "%",
				"timestamp":        time.Now().Format(time.RFC3339),
				"cellId":           "cell-001",
			},
			{
				"measurementName":  "Active_UE_Count",
				"measurementType":  "count",
				"measurementValue": 42,
				"measurementUnit":  "count",
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
			{
				"measurementTypeId":   2,
				"measurementTypeName": "UL_PRB_Usage",
			},
			{
				"measurementTypeId":   3,
				"measurementTypeName": "Active_UE_Count",
			},
		},
	}
	messageBytes, _ := json.Marshal(kmpMessage)

	// Validate the message
	err = kmpAPI.ValidateMessage("kmp-indication-header", headerBytes)
	if err != nil {
		fmt.Printf("Header validation failed: %v\n", err)
	} else {
		fmt.Println("Header validation: PASSED")
	}

	err = kmpAPI.ValidateMessage("kmp-indication-message", messageBytes)
	if err != nil {
		fmt.Printf("Message validation failed: %v\n", err)
	} else {
		fmt.Println("Message validation: PASSED")
	}

	// Process the indication
	ctx := context.Background()
	result, err := kmpAPI.ProcessIndication(ctx, headerBytes, messageBytes)
	if err != nil {
		fmt.Printf("Failed to process KPM indication: %v\n", err)
		return
	}

	if response, ok := result.(*dashboard.KPMIndicationResponse); ok {
		fmt.Printf("KPM Processing Results:\n")
		fmt.Printf("  - Processing Time: %v\n", response.ProcessingTime)
		fmt.Printf("  - Raw Measurements: %d\n", len(response.Message.MeasurementData))
		fmt.Printf("  - Processed Metrics: %d\n", len(response.ProcessedMetrics))
		
		// Show some processed metrics
		for i, metric := range response.ProcessedMetrics[:6] { // Show first 6
			fmt.Printf("    %d. %s (%s): %.2f %s [%d samples]\n", 
				i+1, metric.MeasurementName, metric.MetricType, 
				metric.Value, metric.Unit, metric.SampleCount)
		}
	}
}

func demonstrateRCAPI(apiManager *dashboard.ServiceModelAPIManager) {
	fmt.Println("\n--- E2SM-RC API Demonstration ---")

	// Get RC API
	rcAPI, err := apiManager.GetAPI(dashboard.ServiceModelTypeRC)
	if err != nil {
		log.Fatalf("Failed to get RC API: %v", err)
	}

	// Show supported operations
	operations := rcAPI.GetSupportedOperations()
	fmt.Printf("RC Supported Operations: %v\n", operations)

	// Get control action definitions
	if rcAPITyped, ok := rcAPI.(*dashboard.E2SMRCApi); ok {
		controlDefs := rcAPITyped.GetControlActionDefinitions()
		fmt.Printf("Available Control Actions: %d\n", len(controlDefs))
		for _, action := range controlDefs[:3] { // Show first 3
			fmt.Printf("  - %s: %s (Category: %s)\n", 
				action.Action, action.Description, action.Category)
		}
	}

	// Create sample RC control header
	rcHeader := map[string]interface{}{
		"ricControlHeaderFormat": 1,
		"ueIdentity":            "ue-12345",
		"ranParameterId":        100,
		"ranParameterName":      "QoS_Control_Parameter",
	}
	headerBytes, _ := json.Marshal(rcHeader)

	// Create sample RC control message
	rcMessage := map[string]interface{}{
		"ricControlMessageFormat": 1,
		"ranParameters": []map[string]interface{}{
			{
				"id":    100,
				"name":  "QCI",
				"value": 7,
				"type":  "integer",
			},
			{
				"id":    101,
				"name":  "GBR",
				"value": 1000000, // 1 Mbps
				"type":  "integer",
			},
			{
				"id":    102,
				"name":  "MBR",
				"value": 10000000, // 10 Mbps
				"type":  "integer",
			},
		},
		"controlAction":  "QOS_CONTROL",
		"controlOutcome": "APPLY_QOS_PARAMETERS",
	}
	messageBytes, _ := json.Marshal(rcMessage)

	// Validate the message
	err = rcAPI.ValidateMessage("rc-control-header", headerBytes)
	if err != nil {
		fmt.Printf("Header validation failed: %v\n", err)
	} else {
		fmt.Println("Header validation: PASSED")
	}

	err = rcAPI.ValidateMessage("rc-control-message", messageBytes)
	if err != nil {
		fmt.Printf("Message validation failed: %v\n", err)
	} else {
		fmt.Println("Message validation: PASSED")
	}

	// Process the control message
	ctx := context.Background()
	result, err := rcAPI.ProcessControl(ctx, headerBytes, messageBytes)
	if err != nil {
		fmt.Printf("Failed to process RC control: %v\n", err)
		return
	}

	if response, ok := result.(*dashboard.RCControlResponse); ok {
		fmt.Printf("RC Control Results:\n")
		fmt.Printf("  - Processing Time: %v\n", response.ProcessingTime)
		fmt.Printf("  - Control Action: %s\n", response.ControlResult.ControlAction)
		fmt.Printf("  - Status: %s\n", response.ControlResult.Status)
		fmt.Printf("  - Executed Actions: %v\n", response.ControlResult.ExecutedActions)
		
		// Show control results
		fmt.Printf("  - Control Results:\n")
		for key, value := range response.ControlResult.Results {
			fmt.Printf("    %s: %v\n", key, value)
		}
	}
}

func demonstrateNIAPI(apiManager *dashboard.ServiceModelAPIManager) {
	fmt.Println("\n--- E2SM-NI API Demonstration ---")

	// Get NI API
	niAPI, err := apiManager.GetAPI(dashboard.ServiceModelTypeNI)
	if err != nil {
		log.Fatalf("Failed to get NI API: %v", err)
	}

	// Show supported operations
	operations := niAPI.GetSupportedOperations()
	fmt.Printf("NI Supported Operations: %v\n", operations)

	// Get interface and protocol definitions
	if niAPITyped, ok := niAPI.(*dashboard.E2SMNIApi); ok {
		interfaceDefs := niAPITyped.GetInterfaceDefinitions()
		protocolDefs := niAPITyped.GetSupportedProtocols()
		
		fmt.Printf("Supported Interfaces: %d\n", len(interfaceDefs))
		for _, iface := range interfaceDefs[:3] { // Show first 3
			fmt.Printf("  - %s: %s (Protocol: %s)\n", 
				iface.InterfaceType, iface.Description, iface.Protocol)
		}
		
		fmt.Printf("Supported Protocols: %d\n", len(protocolDefs))
		for _, protocol := range protocolDefs[:3] { // Show first 3
			fmt.Printf("  - %s v%s: %s\n", 
				protocol.Name, protocol.Version, protocol.Description)
		}
	}

	// Create sample NI indication header
	niHeader := map[string]interface{}{
		"interfaceType":      "F1-C",
		"interfaceId":        "f1-interface-gNB001-DU001",
		"interfaceDirection": "INGRESS",
		"timestamp":          time.Now().Format(time.RFC3339),
	}
	headerBytes, _ := json.Marshal(niHeader)

	// Create sample NI indication message
	niMessage := map[string]interface{}{
		"interfaceMessage": "MDEwMTAwMTEwMTEwMTAwMQ==", // Sample base64 encoded message
		"messageType":      "F1AP_SETUP_REQUEST",
		"protocolIEs": []map[string]interface{}{
			{
				"id":          1,
				"criticality": "reject",
				"value":       12345,
				"typeName":    "GlobalGNB-ID",
			},
			{
				"id":          2,
				"criticality": "ignore",
				"value":       67890,
				"typeName":    "GNB-DU-ID",
			},
			{
				"id":          3,
				"criticality": "notify",
				"value":       11111,
				"typeName":    "GNB-DU-Name",
			},
		},
		"additionalInfo": map[string]interface{}{
			"messageSize": 256,
			"source":      "gNB-DU-001",
		},
	}
	messageBytes, _ := json.Marshal(niMessage)

	// Validate the message
	err = niAPI.ValidateMessage("ni-indication-header", headerBytes)
	if err != nil {
		fmt.Printf("Header validation failed: %v\n", err)
	} else {
		fmt.Println("Header validation: PASSED")
	}

	err = niAPI.ValidateMessage("ni-indication-message", messageBytes)
	if err != nil {
		fmt.Printf("Message validation failed: %v\n", err)
	} else {
		fmt.Println("Message validation: PASSED")
	}

	// Process the indication
	ctx := context.Background()
	result, err := niAPI.ProcessIndication(ctx, headerBytes, messageBytes)
	if err != nil {
		fmt.Printf("Failed to process NI indication: %v\n", err)
		return
	}

	if response, ok := result.(*dashboard.NIIndicationResponse); ok {
		fmt.Printf("NI Processing Results:\n")
		fmt.Printf("  - Processing Time: %v\n", response.ProcessingTime)
		fmt.Printf("  - Interface: %s (%s)\n", 
			response.ProcessedData.InterfaceType, response.ProcessedData.InterfaceDirection)
		fmt.Printf("  - Message Type: %s\n", response.ProcessedData.MessageType)
		
		// Show statistics
		fmt.Printf("  - Statistics:\n")
		for key, value := range response.ProcessedData.Statistics {
			fmt.Printf("    %s: %v\n", key, value)
		}
		
		// Show protocol info
		fmt.Printf("  - Protocol Info:\n")
		for key, value := range response.ProcessedData.ProtocolInfo {
			if key == "ie_count" || key == "criticality_distribution" {
				fmt.Printf("    %s: %v\n", key, value)
			}
		}
	}
}

func demonstrateGenericOperations(apiManager *dashboard.ServiceModelAPIManager) {
	fmt.Println("\n--- Generic Service Model Operations ---")

	// Get all supported operations
	allOperations := apiManager.GetSupportedOperations()
	fmt.Printf("All Supported Operations by Service Model:\n")
	for modelType, operations := range allOperations {
		fmt.Printf("  %s: %d operations\n", modelType, len(operations))
	}

	// Demonstrate message schema retrieval
	fmt.Println("\nMessage Schemas:")
	
	// Get KPM indication header schema
	schema, err := apiManager.GetMessageSchema(dashboard.ServiceModelTypeKPM, "kmp-indication-header")
	if err != nil {
		fmt.Printf("Failed to get KPM header schema: %v\n", err)
	} else {
		fmt.Printf("  KPM Indication Header Schema: %d properties\n", 
			len(schema["properties"].(map[string]interface{})))
	}

	// Get RC control message schema
	schema, err = apiManager.GetMessageSchema(dashboard.ServiceModelTypeRC, "rc-control-message")
	if err != nil {
		fmt.Printf("Failed to get RC control schema: %v\n", err)
	} else {
		fmt.Printf("  RC Control Message Schema: %d properties\n", 
			len(schema["properties"].(map[string]interface{})))
	}

	// Get NI indication message schema
	schema, err = apiManager.GetMessageSchema(dashboard.ServiceModelTypeNI, "ni-indication-message")
	if err != nil {
		fmt.Printf("Failed to get NI indication schema: %v\n", err)
	} else {
		fmt.Printf("  NI Indication Message Schema: %d properties\n", 
			len(schema["properties"].(map[string]interface{})))
	}

	// Demonstrate validation with invalid data
	fmt.Println("\nValidation Examples:")
	
	invalidMessage := []byte(`{"invalid": "data"}`)
	
	err = apiManager.ValidateMessage(dashboard.ServiceModelTypeKPM, "kmp-indication-header", invalidMessage)
	if err != nil {
		fmt.Printf("  Invalid KPM header validation: FAILED (as expected)\n")
	}

	err = apiManager.ValidateMessage(dashboard.ServiceModelTypeRC, "rc-control-message", invalidMessage)
	if err != nil {
		fmt.Printf("  Invalid RC control validation: FAILED (as expected)\n")
	}

	err = apiManager.ValidateMessage(dashboard.ServiceModelTypeNI, "ni-indication-message", invalidMessage)
	if err != nil {
		fmt.Printf("  Invalid NI indication validation: FAILED (as expected)\n")
	}

	fmt.Println("\nService Model API framework provides:")
	fmt.Println("  ✓ Type-safe service model operations")
	fmt.Println("  ✓ Comprehensive message validation")
	fmt.Println("  ✓ Extensible API framework")
	fmt.Println("  ✓ Standards-compliant implementations")
	fmt.Println("  ✓ Performance monitoring capabilities")
	fmt.Println("  ✓ RAN control operations")
	fmt.Println("  ✓ Network interface analysis")
}