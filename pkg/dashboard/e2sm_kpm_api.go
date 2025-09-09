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

// E2SMKPMApi implements the ServiceModelAPI interface for E2SM-KPM
type E2SMKPMApi struct {
	registry  *ServiceModelRegistry
	validator *MessageValidator
}

// NewE2SMKPMApi creates a new E2SM-KPM API instance
func NewE2SMKPMApi(registry *ServiceModelRegistry) *E2SMKPMApi {
	api := &E2SMKPMApi{
		registry:  registry,
		validator: NewMessageValidator(),
	}
	
	// Register message schemas
	api.registerSchemas()
	
	return api
}

// GetServiceModelType returns the service model type
func (api *E2SMKPMApi) GetServiceModelType() ServiceModelType {
	return ServiceModelTypeKPM
}

// ValidateMessage validates a KPM message
func (api *E2SMKPMApi) ValidateMessage(messageType string, data []byte) error {
	return api.validator.ValidateMessage(messageType, data)
}

// ProcessIndication processes a KPM indication message
func (api *E2SMKPMApi) ProcessIndication(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	startTime := time.Now()
	
	// Validate header
	if err := api.ValidateMessage("kmp-indication-header", header); err != nil {
		return nil, fmt.Errorf("invalid KPM indication header: %w", err)
	}
	
	// Validate message
	if err := api.ValidateMessage("kmp-indication-message", message); err != nil {
		return nil, fmt.Errorf("invalid KPM indication message: %w", err)
	}
	
	// Parse header and message
	indicationHeader, indicationMessage, err := api.registry.ProcessKPMIndication(header, message)
	if err != nil {
		return nil, fmt.Errorf("failed to process KPM indication: %w", err)
	}
	
	// Convert MeasurementData to E2SMKPMMetrics and process
	kmpMetrics := api.convertToKPMMetrics(indicationMessage.MeasurementData)
	processedMetrics := api.processMetrics(kmpMetrics)
	
	// Create response
	response := &KPMIndicationResponse{
		Header:           indicationHeader,
		Message:          indicationMessage,
		ProcessedMetrics: processedMetrics,
		ProcessingTime:   time.Since(startTime),
		Timestamp:        time.Now(),
	}
	
	log.Printf("Processed KPM indication with %d measurements in %v", 
		len(indicationMessage.MeasurementData), response.ProcessingTime)
	
	return response, nil
}

// ProcessControl processes a KPM control message (not supported for KPM)
func (api *E2SMKPMApi) ProcessControl(ctx context.Context, header []byte, message []byte) (interface{}, error) {
	return nil, fmt.Errorf("control operations not supported for E2SM-KPM service model")
}

// GetSupportedOperations returns supported operations for KPM
func (api *E2SMKPMApi) GetSupportedOperations() []string {
	return []string{
		"indication-processing",
		"measurement-collection",
		"kpi-calculation",
		"performance-monitoring",
		"cell-level-metrics",
		"ue-level-metrics",
		"periodic-reporting",
		"event-triggered-reporting",
	}
}

// GetMessageSchema returns the JSON schema for KPM message validation
func (api *E2SMKPMApi) GetMessageSchema(messageType string) (map[string]interface{}, error) {
	schema, exists := api.validator.GetSchema(messageType)
	if !exists {
		return nil, fmt.Errorf("no schema found for message type: %s", messageType)
	}
	
	return schema, nil
}

// registerSchemas registers JSON schemas for KPM message validation
func (api *E2SMKPMApi) registerSchemas() {
	// KPM Indication Header Schema
	kmpHeaderSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"collectionStartTime", "fileFormatVersion", "senderName", "senderType"},
		"properties": map[string]interface{}{
			"collectionStartTime": map[string]interface{}{
				"type": "string",
				"format": "date-time",
			},
			"fileFormatVersion": map[string]interface{}{
				"type": "string",
			},
			"senderName": map[string]interface{}{
				"type": "string",
			},
			"senderType": map[string]interface{}{
				"type": "string",
			},
			"vendorName": map[string]interface{}{
				"type": "string",
			},
		},
	}
	api.validator.RegisterSchema("kmp-indication-header", kmpHeaderSchema)
	
	// KPM Indication Message Schema
	kmpMessageSchema := map[string]interface{}{
		"type": "object",
		"required": []string{"measurementData", "granularityPeriod"},
		"properties": map[string]interface{}{
			"measurementData": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"required": []string{"measurementName", "measurementType", "measurementValue", "timestamp"},
					"properties": map[string]interface{}{
						"measurementName": map[string]interface{}{
							"type": "string",
						},
						"measurementType": map[string]interface{}{
							"type": "string",
						},
						"measurementValue": map[string]interface{}{
							"type": "number",
						},
						"measurementUnit": map[string]interface{}{
							"type": "string",
						},
						"timestamp": map[string]interface{}{
							"type": "string",
							"format": "date-time",
						},
						"cellId": map[string]interface{}{
							"type": "string",
						},
						"additionalInfo": map[string]interface{}{
							"type": "object",
						},
					},
				},
			},
			"granularityPeriod": map[string]interface{}{
				"type": "number",
			},
			"measurementInfoList": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"required": []string{"measurementTypeId", "measurementTypeName"},
					"properties": map[string]interface{}{
						"measurementTypeId": map[string]interface{}{
							"type": "number",
						},
						"measurementTypeName": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
	}
	api.validator.RegisterSchema("kmp-indication-message", kmpMessageSchema)
	
	log.Println("Registered E2SM-KPM message schemas")
}

// convertToKPMMetrics converts MeasurementData to E2SMKPMMetrics
func (api *E2SMKPMApi) convertToKPMMetrics(measurementData []MeasurementData) []E2SMKPMMetrics {
	var kmpMetrics []E2SMKPMMetrics
	
	for _, data := range measurementData {
		kmpMetric := E2SMKPMMetrics{
			CellID:              fmt.Sprintf("cell_%d", data.MeasurementID), // 從 MeasurementID 生成 CellID
			MeasurementName:     data.MeasurementType,                       // 使用 MeasurementType 作為 MeasurementName
			MeasurementValue:    data.MeasurementValue,                      // 直接複製 MeasurementValue
			MeasurementUnit:     "count",                                    // 設定預設單位
			Timestamp:           time.Now(),                                 // 設定當前時間戳
			MeasurementData:     []MeasurementData{data},                   // 將原始數據包裝進去
		}
		
		// 從 labels 中提取額外信息
		if cellID, ok := data.Labels["cellId"]; ok {
			kmpMetric.CellID = cellID
		}
		if nodeid, ok := data.Labels["e2NodeId"]; ok {
			kmpMetric.E2NodeID = nodeid
		}
		
		kmpMetrics = append(kmpMetrics, kmpMetric)
	}
	
	return kmpMetrics
}

// processMetrics processes measurement data and calculates derived metrics
func (api *E2SMKPMApi) processMetrics(measurements []E2SMKPMMetrics) []ProcessedMetric {
	var processedMetrics []ProcessedMetric
	
	// Group measurements by cell ID and measurement type
	cellMetrics := make(map[string]map[string][]E2SMKPMMetrics)
	
	for _, measurement := range measurements {
		cellID := measurement.CellID
		if cellID == "" {
			cellID = "unknown"
		}
		
		if cellMetrics[cellID] == nil {
			cellMetrics[cellID] = make(map[string][]E2SMKPMMetrics)
		}
		
		cellMetrics[cellID][measurement.MeasurementName] = append(
			cellMetrics[cellID][measurement.MeasurementName], measurement)
	}
	
	// Process metrics for each cell
	for cellID, metrics := range cellMetrics {
		for measurementName, measurementList := range metrics {
			processed := api.calculateAggregatedMetrics(cellID, measurementName, measurementList)
			processedMetrics = append(processedMetrics, processed...)
		}
	}
	
	return processedMetrics
}

// calculateAggregatedMetrics calculates aggregated metrics from raw measurements
func (api *E2SMKPMApi) calculateAggregatedMetrics(cellID, measurementName string, measurements []E2SMKPMMetrics) []ProcessedMetric {
	var processedMetrics []ProcessedMetric
	
	if len(measurements) == 0 {
		return processedMetrics
	}
	
	// Calculate basic statistics
	var sum, min, max float64
	var count int
	
	for i, measurement := range measurements {
		value, ok := measurement.MeasurementValue.(float64)
		if !ok {
			// Try to convert from other numeric types
			switch v := measurement.MeasurementValue.(type) {
			case int:
				value = float64(v)
			case int32:
				value = float64(v)
			case int64:
				value = float64(v)
			default:
				continue // Skip non-numeric values
			}
		}
		
		sum += value
		count++
		
		if i == 0 {
			min = value
			max = value
		} else {
			if value < min {
				min = value
			}
			if value > max {
				max = value
			}
		}
	}
	
	if count == 0 {
		return processedMetrics
	}
	
	avg := sum / float64(count)
	
	// Create processed metrics
	processedMetrics = append(processedMetrics, ProcessedMetric{
		CellID:          cellID,
		MeasurementName: measurementName,
		MetricType:      "average",
		Value:           avg,
		Unit:            measurements[0].MeasurementUnit,
		Timestamp:       time.Now(),
		SampleCount:     count,
	})
	
	processedMetrics = append(processedMetrics, ProcessedMetric{
		CellID:          cellID,
		MeasurementName: measurementName,
		MetricType:      "minimum",
		Value:           min,
		Unit:            measurements[0].MeasurementUnit,
		Timestamp:       time.Now(),
		SampleCount:     count,
	})
	
	processedMetrics = append(processedMetrics, ProcessedMetric{
		CellID:          cellID,
		MeasurementName: measurementName,
		MetricType:      "maximum",
		Value:           max,
		Unit:            measurements[0].MeasurementUnit,
		Timestamp:       time.Now(),
		SampleCount:     count,
	})
	
	processedMetrics = append(processedMetrics, ProcessedMetric{
		CellID:          cellID,
		MeasurementName: measurementName,
		MetricType:      "sum",
		Value:           sum,
		Unit:            measurements[0].MeasurementUnit,
		Timestamp:       time.Now(),
		SampleCount:     count,
	})
	
	return processedMetrics
}

// KPMIndicationResponse represents the response from processing a KPM indication
type KPMIndicationResponse struct {
	Header           *E2SMKPMIndicationHeader `json:"header"`
	Message          *E2SMKPMIndicationMessage `json:"message"`
	ProcessedMetrics []ProcessedMetric         `json:"processedMetrics"`
	ProcessingTime   time.Duration             `json:"processingTime"`
	Timestamp        time.Time                 `json:"timestamp"`
}

// ProcessedMetric represents a processed measurement metric
type ProcessedMetric struct {
	CellID          string    `json:"cellId"`
	MeasurementName string    `json:"measurementName"`
	MetricType      string    `json:"metricType"` // average, minimum, maximum, sum, etc.
	Value           float64   `json:"value"`
	Unit            string    `json:"unit"`
	Timestamp       time.Time `json:"timestamp"`
	SampleCount     int       `json:"sampleCount"`
}

// GetKPIDefinitions returns standard KPI definitions for E2SM-KPM
func (api *E2SMKPMApi) GetKPIDefinitions() []KPIDefinition {
	return []KPIDefinition{
		{
			Name:        "DL_PRB_Usage",
			Description: "Downlink Physical Resource Block Usage",
			Unit:        "percentage",
			Category:    "Resource Utilization",
			Formula:     "Used_DL_PRBs / Total_DL_PRBs * 100",
		},
		{
			Name:        "UL_PRB_Usage",
			Description: "Uplink Physical Resource Block Usage",
			Unit:        "percentage",
			Category:    "Resource Utilization",
			Formula:     "Used_UL_PRBs / Total_UL_PRBs * 100",
		},
		{
			Name:        "DL_Throughput",
			Description: "Downlink Throughput",
			Unit:        "Mbps",
			Category:    "Performance",
			Formula:     "DL_Data_Volume / Measurement_Period",
		},
		{
			Name:        "UL_Throughput",
			Description: "Uplink Throughput",
			Unit:        "Mbps",
			Category:    "Performance",
			Formula:     "UL_Data_Volume / Measurement_Period",
		},
		{
			Name:        "Active_UE_Count",
			Description: "Number of Active User Equipment",
			Unit:        "count",
			Category:    "Capacity",
			Formula:     "Count of UEs with active sessions",
		},
		{
			Name:        "Packet_Loss_Rate",
			Description: "Packet Loss Rate",
			Unit:        "percentage",
			Category:    "Quality",
			Formula:     "Lost_Packets / Total_Packets * 100",
		},
		{
			Name:        "Latency",
			Description: "End-to-End Latency",
			Unit:        "ms",
			Category:    "Quality",
			Formula:     "Average packet transmission delay",
		},
		{
			Name:        "Handover_Success_Rate",
			Description: "Handover Success Rate",
			Unit:        "percentage",
			Category:    "Mobility",
			Formula:     "Successful_Handovers / Total_Handover_Attempts * 100",
		},
	}
}

// KPIDefinition represents a Key Performance Indicator definition
type KPIDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
	Category    string `json:"category"`
	Formula     string `json:"formula"`
}