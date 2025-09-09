package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// E2IndicationProcessor handles real-time E2 indication processing for O-RAN L Release
type E2IndicationProcessor struct {
	kafkaReader     *kafka.Reader
	kafkaWriter     *kafka.Writer
	influxClient    influxdb2.Client
	writeAPI        api.WriteAPIBlocking
	correlationDB   *CorrelationDatabase
	metrics         *E2ProcessorMetrics
	processors      map[string]IndicationProcessor
	mu              sync.RWMutex
}

// E2Indication represents structured E2 indication data from RAN nodes
type E2Indication struct {
	RequestID       string                 `json:"request_id"`
	RanFunctionID   int                    `json:"ran_function_id"`
	ActionID        int                    `json:"action_id"`
	IndicationType  string                 `json:"indication_type"`    // "report" or "insert"
	Timestamp       time.Time              `json:"timestamp"`
	Source          E2NodeIdentity         `json:"source"`
	IndicationHeader E2IndicationHeader    `json:"indication_header"`
	IndicationMessage E2IndicationMessage  `json:"indication_message"`
	Measurements    map[string]interface{} `json:"measurements"`
}

type E2NodeIdentity struct {
	GlobalE2NodeID  string `json:"global_e2_node_id"`
	ENBName         string `json:"enb_name,omitempty"`
	GNBName         string `json:"gnb_name,omitempty"`
	PLMNIdentity    string `json:"plmn_identity"`
	CellIdentity    string `json:"cell_identity"`
}

type E2IndicationHeader struct {
	InterfaceID     string    `json:"interface_id"`
	Timestamp       time.Time `json:"timestamp"`
	EventTrigger    string    `json:"event_trigger"`
	ReportingPeriod int       `json:"reporting_period_ms"`
}

type E2IndicationMessage struct {
	MessageType     string                 `json:"message_type"`
	PayloadFormat   string                 `json:"payload_format"`   // "json", "asn1", "protobuf"
	PayloadData     map[string]interface{} `json:"payload_data"`
	MeasurementData []MeasurementRecord    `json:"measurement_data,omitempty"`
}

type MeasurementRecord struct {
	MeasurementID   string                 `json:"measurement_id"`
	MeasurementName string                 `json:"measurement_name"`
	Values          map[string]interface{} `json:"values"`
	Granularity     string                 `json:"granularity"`  // "cell", "ue", "slice", "drb"
	Labels          map[string]string      `json:"labels"`
}

// CorrelationDatabase manages cross-function data correlation
type CorrelationDatabase struct {
	sessionData   map[string]*SessionContext
	cellData      map[string]*CellContext
	ueData        map[string]*UEContext
	sliceData     map[string]*SliceContext
	mu            sync.RWMutex
}

type SessionContext struct {
	SessionID       string                 `json:"session_id"`
	UEIdentity      string                 `json:"ue_identity"`
	CellID          string                 `json:"cell_id"`
	SliceID         string                 `json:"slice_id"`
	StartTime       time.Time              `json:"start_time"`
	LastUpdate      time.Time              `json:"last_update"`
	KPIHistory      []KPISnapshot          `json:"kpi_history"`
	QoSParameters   map[string]interface{} `json:"qos_parameters"`
}

type CellContext struct {
	CellID          string                    `json:"cell_id"`
	PLMNIdentity    string                   `json:"plmn_identity"`
	ActiveUEs       map[string]*UEContext    `json:"active_ues"`
	ActiveSlices    map[string]*SliceContext `json:"active_slices"`
	ResourceStatus  ResourceStatus           `json:"resource_status"`
	PerformanceData PerformanceMetrics       `json:"performance_data"`
	LastUpdate      time.Time                `json:"last_update"`
}

type UEContext struct {
	UEIdentity      string                 `json:"ue_identity"`
	CellID          string                 `json:"cell_id"`
	SliceIDs        []string              `json:"slice_ids"`
	ConnectionState string                `json:"connection_state"`
	QoSFlows        map[string]QoSFlow    `json:"qos_flows"`
	Measurements    map[string]interface{} `json:"measurements"`
	LastSeen        time.Time             `json:"last_seen"`
}

type SliceContext struct {
	SliceID         string                 `json:"slice_id"`
	ServiceType     string                 `json:"service_type"`  // "eMBB", "URLLC", "mMTC"
	ActiveUEs       []string              `json:"active_ues"`
	ResourceAlloc   ResourceAllocation    `json:"resource_allocation"`
	SLARequirements SLARequirements       `json:"sla_requirements"`
	CurrentKPIs     map[string]float64    `json:"current_kpis"`
	LastUpdate      time.Time             `json:"last_update"`
}

type QoSFlow struct {
	QFI             int                    `json:"qfi"`
	FiveQI          int                    `json:"five_qi"`
	ARP             int                    `json:"arp"`
	GBRParameters   *GBRParameters         `json:"gbr_parameters,omitempty"`
	Measurements    map[string]interface{} `json:"measurements"`
}

type GBRParameters struct {
	UplinkGBR       int64 `json:"uplink_gbr_kbps"`
	DownlinkGBR     int64 `json:"downlink_gbr_kbps"`
	UplinkMBR       int64 `json:"uplink_mbr_kbps"`
	DownlinkMBR     int64 `json:"downlink_mbr_kbps"`
}

type ResourceStatus struct {
	PRBUtilizationDL float64 `json:"prb_utilization_dl"`
	PRBUtilizationUL float64 `json:"prb_utilization_ul"`
	AvailablePRBs    int     `json:"available_prbs"`
	TotalPRBs        int     `json:"total_prbs"`
	PowerConsumption float64 `json:"power_consumption_watts"`
}

type ResourceAllocation struct {
	AllocatedPRBs    int                    `json:"allocated_prbs"`
	GuaranteedBitRate int64                 `json:"guaranteed_bit_rate_kbps"`
	MaximumBitRate    int64                 `json:"maximum_bit_rate_kbps"`
	Latency          int                    `json:"latency_ms"`
	PacketLossRate   float64               `json:"packet_loss_rate"`
	CustomParams     map[string]interface{} `json:"custom_params"`
}

type SLARequirements struct {
	Throughput      int64   `json:"throughput_kbps"`
	Latency         int     `json:"latency_ms"`
	Reliability     float64 `json:"reliability_percentage"`
	Availability    float64 `json:"availability_percentage"`
}

type PerformanceMetrics struct {
	ThroughputDL       float64 `json:"throughput_dl_mbps"`
	ThroughputUL       float64 `json:"throughput_ul_mbps"`
	LatencyE2E         float64 `json:"latency_e2e_ms"`
	LatencyRAN         float64 `json:"latency_ran_ms"`
	PacketLossRate     float64 `json:"packet_loss_rate"`
	HandoverRate       float64 `json:"handover_rate"`
	CallDropRate       float64 `json:"call_drop_rate"`
	EnergyEfficiency   float64 `json:"energy_efficiency_mbps_per_watt"`
	SpectralEfficiency float64 `json:"spectral_efficiency_bps_per_hz"`
}

type KPISnapshot struct {
	Timestamp       time.Time              `json:"timestamp"`
	KPIValues       map[string]float64     `json:"kpi_values"`
	QualityMetrics  map[string]float64     `json:"quality_metrics"`
	ResourceMetrics map[string]interface{} `json:"resource_metrics"`
}

// IndicationProcessor defines interface for processing different indication types
type IndicationProcessor interface {
	ProcessIndication(indication *E2Indication) error
	GetSupportedMeasurements() []string
}

// E2ProcessorMetrics defines Prometheus metrics
type E2ProcessorMetrics struct {
	IndicationsProcessed   *prometheus.CounterVec
	ProcessingLatency      *prometheus.HistogramVec
	CorrelationMatches     prometheus.Counter
	ActiveSessions         prometheus.Gauge
	ActiveCells            prometheus.Gauge
	ActiveUEs              prometheus.Gauge
	KPICalculations        prometheus.Counter
	DataCorrelationErrors  prometheus.Counter
}

func NewE2ProcessorMetrics() *E2ProcessorMetrics {
	return &E2ProcessorMetrics{
		IndicationsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oran_e2_indications_processed_total",
				Help: "Total number of E2 indications processed",
			},
			[]string{"ran_function", "indication_type", "source"},
		),
		ProcessingLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "oran_e2_processing_latency_seconds",
				Help:    "E2 indication processing latency",
				Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15),
			},
			[]string{"ran_function", "indication_type"},
		),
		CorrelationMatches: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_e2_correlation_matches_total",
				Help: "Total data correlation matches",
			},
		),
		ActiveSessions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "oran_e2_active_sessions",
				Help: "Number of active UE sessions",
			},
		),
		ActiveCells: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "oran_e2_active_cells",
				Help: "Number of active cells",
			},
		),
		ActiveUEs: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "oran_e2_active_ues",
				Help: "Number of active UEs",
			},
		),
		KPICalculations: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_e2_kpi_calculations_total",
				Help: "Total KPI calculations performed",
			},
		),
		DataCorrelationErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_e2_correlation_errors_total",
				Help: "Total data correlation errors",
			},
		),
	}
}

func (m *E2ProcessorMetrics) Register() {
	prometheus.MustRegister(m.IndicationsProcessed)
	prometheus.MustRegister(m.ProcessingLatency)
	prometheus.MustRegister(m.CorrelationMatches)
	prometheus.MustRegister(m.ActiveSessions)
	prometheus.MustRegister(m.ActiveCells)
	prometheus.MustRegister(m.ActiveUEs)
	prometheus.MustRegister(m.KPICalculations)
	prometheus.MustRegister(m.DataCorrelationErrors)
}

func NewE2IndicationProcessor() *E2IndicationProcessor {
	// Kafka configuration
	kafkaBrokers := getEnv("KAFKA_BROKERS", "kafka:29092")
	
	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{kafkaBrokers},
		Topic:       "e2-indications",
		GroupID:     "e2-telemetry-processor",
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     1 * time.Second,
	})

	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBrokers),
		Topic:        "processed-e2-data",
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
	}

	// InfluxDB configuration
	influxURL := getEnv("INFLUXDB_URL", "http://influxdb:8086")
	influxToken := getEnv("INFLUXDB_TOKEN", "oran-super-secret-token")
	influxOrg := getEnv("INFLUXDB_ORG", "oran")

	influxClient := influxdb2.NewClient(influxURL, influxToken)
	writeAPI := influxClient.WriteAPIBlocking(influxOrg, "oran-e2-data")

	// Initialize correlation database
	correlationDB := &CorrelationDatabase{
		sessionData: make(map[string]*SessionContext),
		cellData:    make(map[string]*CellContext),
		ueData:      make(map[string]*UEContext),
		sliceData:   make(map[string]*SliceContext),
	}

	metrics := NewE2ProcessorMetrics()
	metrics.Register()

	processor := &E2IndicationProcessor{
		kafkaReader:   kafkaReader,
		kafkaWriter:   kafkaWriter,
		influxClient:  influxClient,
		writeAPI:      writeAPI,
		correlationDB: correlationDB,
		metrics:       metrics,
		processors:    make(map[string]IndicationProcessor),
	}

	// Register indication processors
	processor.registerProcessors()
	
	return processor
}

func (ep *E2IndicationProcessor) registerProcessors() {
	// Register different RAN function processors
	ep.processors["1"] = NewRICIndicationProcessor()      // RC RAN Function
	ep.processors["2"] = NewKPMIndicationProcessor()      // KPM RAN Function  
	ep.processors["3"] = NewE2SMProcessor()               // E2SM Processor
}

func (ep *E2IndicationProcessor) Start(ctx context.Context) error {
	log.Println("Starting E2 Indication Processor...")

	// Start correlation database cleanup routine
	go ep.correlationCleanupRoutine(ctx)

	// Start metrics update routine
	go ep.metricsUpdateRoutine(ctx)

	// Main processing loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := ep.processNextIndication(ctx); err != nil {
				log.Printf("Error processing indication: %v", err)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func (ep *E2IndicationProcessor) processNextIndication(ctx context.Context) error {
	message, err := ep.kafkaReader.ReadMessage(ctx)
	if err != nil {
		return err
	}

	startTime := time.Now()

	var indication E2Indication
	if err := json.Unmarshal(message.Value, &indication); err != nil {
		log.Printf("Error unmarshaling E2 indication: %v", err)
		return err
	}

	// Process indication based on RAN function
	ranFunctionKey := fmt.Sprintf("%d", indication.RanFunctionID)
	processor, exists := ep.processors[ranFunctionKey]
	if !exists {
		log.Printf("No processor found for RAN function %d", indication.RanFunctionID)
		return nil
	}

	// Process the indication
	if err := processor.ProcessIndication(&indication); err != nil {
		log.Printf("Error processing indication: %v", err)
		return err
	}

	// Update correlation database
	if err := ep.updateCorrelationData(&indication); err != nil {
		log.Printf("Error updating correlation data: %v", err)
		ep.metrics.DataCorrelationErrors.Inc()
	}

	// Calculate and store KPIs
	if err := ep.calculateAndStoreKPIs(&indication); err != nil {
		log.Printf("Error calculating KPIs: %v", err)
	}

	// Store processed data in InfluxDB
	if err := ep.storeProcessedData(&indication); err != nil {
		log.Printf("Error storing processed data: %v", err)
	}

	// Send correlated data to downstream systems
	if err := ep.publishCorrelatedData(&indication); err != nil {
		log.Printf("Error publishing correlated data: %v", err)
	}

	// Update metrics
	ep.metrics.IndicationsProcessed.WithLabelValues(
		ranFunctionKey,
		indication.IndicationType,
		indication.Source.GlobalE2NodeID,
	).Inc()

	ep.metrics.ProcessingLatency.WithLabelValues(
		ranFunctionKey,
		indication.IndicationType,
	).Observe(time.Since(startTime).Seconds())

	ep.metrics.KPICalculations.Inc()

	return nil
}

func (ep *E2IndicationProcessor) updateCorrelationData(indication *E2Indication) error {
	ep.correlationDB.mu.Lock()
	defer ep.correlationDB.mu.Unlock()

	// Update cell context
	cellID := indication.Source.CellIdentity
	if cellID != "" {
		cellCtx, exists := ep.correlationDB.cellData[cellID]
		if !exists {
			cellCtx = &CellContext{
				CellID:         cellID,
				PLMNIdentity:   indication.Source.PLMNIdentity,
				ActiveUEs:      make(map[string]*UEContext),
				ActiveSlices:   make(map[string]*SliceContext),
				LastUpdate:     time.Now(),
			}
			ep.correlationDB.cellData[cellID] = cellCtx
		}

		// Update performance data from measurements
		ep.updatePerformanceMetrics(cellCtx, indication.Measurements)
		cellCtx.LastUpdate = time.Now()
	}

	// Update UE contexts if present in measurements
	if ueID, exists := indication.Measurements["ue_identity"]; exists && ueID != nil {
		ueIDStr := fmt.Sprintf("%v", ueID)
		ueCtx, exists := ep.correlationDB.ueData[ueIDStr]
		if !exists {
			ueCtx = &UEContext{
				UEIdentity:      ueIDStr,
				CellID:          cellID,
				QoSFlows:        make(map[string]QoSFlow),
				Measurements:    make(map[string]interface{}),
				LastSeen:        time.Now(),
			}
			ep.correlationDB.ueData[ueIDStr] = ueCtx
		}

		// Update UE measurements
		for key, value := range indication.Measurements {
			ueCtx.Measurements[key] = value
		}
		ueCtx.LastSeen = time.Now()
	}

	// Update slice contexts if present
	if sliceID, exists := indication.Measurements["slice_id"]; exists && sliceID != nil {
		sliceIDStr := fmt.Sprintf("%v", sliceID)
		sliceCtx, exists := ep.correlationDB.sliceData[sliceIDStr]
		if !exists {
			sliceCtx = &SliceContext{
				SliceID:       sliceIDStr,
				CurrentKPIs:   make(map[string]float64),
				LastUpdate:    time.Now(),
			}
			ep.correlationDB.sliceData[sliceIDStr] = sliceCtx
		}

		// Update slice KPIs from measurements
		ep.updateSliceKPIs(sliceCtx, indication.Measurements)
		sliceCtx.LastUpdate = time.Now()
	}

	ep.metrics.CorrelationMatches.Inc()
	return nil
}

func (ep *E2IndicationProcessor) updatePerformanceMetrics(cellCtx *CellContext, measurements map[string]interface{}) {
	if throughputDL, exists := measurements["throughput_dl_mbps"]; exists {
		if val, ok := throughputDL.(float64); ok {
			cellCtx.PerformanceData.ThroughputDL = val
		}
	}

	if throughputUL, exists := measurements["throughput_ul_mbps"]; exists {
		if val, ok := throughputUL.(float64); ok {
			cellCtx.PerformanceData.ThroughputUL = val
		}
	}

	if latencyE2E, exists := measurements["latency_e2e_ms"]; exists {
		if val, ok := latencyE2E.(float64); ok {
			cellCtx.PerformanceData.LatencyE2E = val
		}
	}

	if prbUtilDL, exists := measurements["prb_utilization_dl"]; exists {
		if val, ok := prbUtilDL.(float64); ok {
			cellCtx.ResourceStatus.PRBUtilizationDL = val
		}
	}

	if prbUtilUL, exists := measurements["prb_utilization_ul"]; exists {
		if val, ok := prbUtilUL.(float64); ok {
			cellCtx.ResourceStatus.PRBUtilizationUL = val
		}
	}
}

func (ep *E2IndicationProcessor) updateSliceKPIs(sliceCtx *SliceContext, measurements map[string]interface{}) {
	for key, value := range measurements {
		if val, ok := value.(float64); ok {
			sliceCtx.CurrentKPIs[key] = val
		}
	}
}

func (ep *E2IndicationProcessor) calculateAndStoreKPIs(indication *E2Indication) error {
	// Calculate O-RAN specific KPIs based on indication data
	kpis := make(map[string]interface{})

	// Resource efficiency KPIs
	if prbDL, ok := indication.Measurements["prb_utilization_dl"].(float64); ok {
		if prbUL, ok := indication.Measurements["prb_utilization_ul"].(float64); ok {
			kpis["resource_efficiency"] = (prbDL + prbUL) / 2
		}
	}

	// Energy efficiency KPI
	if throughput, ok := indication.Measurements["throughput_total_mbps"].(float64); ok {
		if power, ok := indication.Measurements["power_consumption_watts"].(float64); ok && power > 0 {
			kpis["energy_efficiency"] = throughput / power
		}
	}

	// Spectral efficiency KPI
	if throughput, ok := indication.Measurements["throughput_total_mbps"].(float64); ok {
		bandwidth := 20.0 // MHz - should come from configuration
		if bw, ok := indication.Measurements["bandwidth_mhz"].(float64); ok {
			bandwidth = bw
		}
		if bandwidth > 0 {
			kpis["spectral_efficiency"] = (throughput * 1000000) / (bandwidth * 1000000) // bps/Hz
		}
	}

	// Quality of Experience KPI
	if rsrp, ok := indication.Measurements["rsrp"].(float64); ok {
		if rsrq, ok := indication.Measurements["rsrq"].(float64); ok {
			if cqi, ok := indication.Measurements["cqi"].(float64); ok {
				// Weighted QoE calculation
				qoe := (rsrp*0.4 + rsrq*0.3 + cqi*0.3) / 3
				kpis["quality_of_experience"] = qoe
			}
		}
	}

	// Latency performance KPI
	if latencyE2E, ok := indication.Measurements["latency_e2e_ms"].(float64); ok {
		if latencyRAN, ok := indication.Measurements["latency_ran_ms"].(float64); ok {
			kpis["latency_performance"] = 100 - (latencyE2E+latencyRAN)/2 // Inverse relationship
		}
	}

	// Connection reliability KPI
	if handoverRate, ok := indication.Measurements["handover_rate"].(float64); ok {
		if callDropRate, ok := indication.Measurements["call_drop_rate"].(float64); ok {
			reliability := 100 - (handoverRate*0.3 + callDropRate*0.7)
			if reliability < 0 {
				reliability = 0
			}
			kpis["connection_reliability"] = reliability
		}
	}

	// Store calculated KPIs in InfluxDB
	return ep.storeKPIs(indication, kpis)
}

func (ep *E2IndicationProcessor) storeKPIs(indication *E2Indication, kpis map[string]interface{}) error {
	ctx := context.Background()
	timestamp := indication.Timestamp

	tags := map[string]string{
		"source":         indication.Source.GlobalE2NodeID,
		"cell_id":        indication.Source.CellIdentity,
		"plmn_identity":  indication.Source.PLMNIdentity,
		"ran_function":   fmt.Sprintf("%d", indication.RanFunctionID),
	}

	point := influxdb2.NewPoint("oran_e2_kpis", tags, kpis, timestamp)
	return ep.writeAPI.WritePoint(ctx, point)
}

func (ep *E2IndicationProcessor) storeProcessedData(indication *E2Indication) error {
	ctx := context.Background()
	timestamp := indication.Timestamp

	tags := map[string]string{
		"source":           indication.Source.GlobalE2NodeID,
		"cell_id":          indication.Source.CellIdentity,
		"ran_function":     fmt.Sprintf("%d", indication.RanFunctionID),
		"indication_type":  indication.IndicationType,
	}

	// Store all measurements as fields
	fields := make(map[string]interface{})
	for key, value := range indication.Measurements {
		fields[key] = value
	}

	point := influxdb2.NewPoint("oran_e2_measurements", tags, fields, timestamp)
	return ep.writeAPI.WritePoint(ctx, point)
}

func (ep *E2IndicationProcessor) publishCorrelatedData(indication *E2Indication) error {
	// Create correlated data message
	correlatedData := map[string]interface{}{
		"indication":      indication,
		"cell_context":    ep.getCellContext(indication.Source.CellIdentity),
		"correlation_ts":  time.Now(),
	}

	// Add UE context if available
	if ueID, exists := indication.Measurements["ue_identity"]; exists {
		ueIDStr := fmt.Sprintf("%v", ueID)
		if ueCtx := ep.getUEContext(ueIDStr); ueCtx != nil {
			correlatedData["ue_context"] = ueCtx
		}
	}

	// Add slice context if available
	if sliceID, exists := indication.Measurements["slice_id"]; exists {
		sliceIDStr := fmt.Sprintf("%v", sliceID)
		if sliceCtx := ep.getSliceContext(sliceIDStr); sliceCtx != nil {
			correlatedData["slice_context"] = sliceCtx
		}
	}

	// Publish to Kafka
	msgData, err := json.Marshal(correlatedData)
	if err != nil {
		return err
	}

	message := kafka.Message{
		Key:   []byte(indication.Source.GlobalE2NodeID),
		Value: msgData,
		Time:  time.Now(),
	}

	return ep.kafkaWriter.WriteMessages(context.Background(), message)
}

func (ep *E2IndicationProcessor) getCellContext(cellID string) *CellContext {
	ep.correlationDB.mu.RLock()
	defer ep.correlationDB.mu.RUnlock()
	return ep.correlationDB.cellData[cellID]
}

func (ep *E2IndicationProcessor) getUEContext(ueID string) *UEContext {
	ep.correlationDB.mu.RLock()
	defer ep.correlationDB.mu.RUnlock()
	return ep.correlationDB.ueData[ueID]
}

func (ep *E2IndicationProcessor) getSliceContext(sliceID string) *SliceContext {
	ep.correlationDB.mu.RLock()
	defer ep.correlationDB.mu.RUnlock()
	return ep.correlationDB.sliceData[sliceID]
}

func (ep *E2IndicationProcessor) correlationCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ep.cleanupStaleData()
		}
	}
}

func (ep *E2IndicationProcessor) cleanupStaleData() {
	ep.correlationDB.mu.Lock()
	defer ep.correlationDB.mu.Unlock()

	now := time.Now()
	staleThreshold := 10 * time.Minute

	// Clean up stale UE contexts
	for ueID, ueCtx := range ep.correlationDB.ueData {
		if now.Sub(ueCtx.LastSeen) > staleThreshold {
			delete(ep.correlationDB.ueData, ueID)
		}
	}

	// Clean up stale cell contexts
	for cellID, cellCtx := range ep.correlationDB.cellData {
		if now.Sub(cellCtx.LastUpdate) > staleThreshold {
			delete(ep.correlationDB.cellData, cellID)
		}
	}

	// Clean up stale slice contexts
	for sliceID, sliceCtx := range ep.correlationDB.sliceData {
		if now.Sub(sliceCtx.LastUpdate) > staleThreshold {
			delete(ep.correlationDB.sliceData, sliceID)
		}
	}
}

func (ep *E2IndicationProcessor) metricsUpdateRoutine(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ep.updateMetrics()
		}
	}
}

func (ep *E2IndicationProcessor) updateMetrics() {
	ep.correlationDB.mu.RLock()
	defer ep.correlationDB.mu.RUnlock()

	ep.metrics.ActiveCells.Set(float64(len(ep.correlationDB.cellData)))
	ep.metrics.ActiveUEs.Set(float64(len(ep.correlationDB.ueData)))

	// Count active sessions
	activeSessions := 0
	for _, ueCtx := range ep.correlationDB.ueData {
		if ueCtx.ConnectionState == "connected" {
			activeSessions++
		}
	}
	ep.metrics.ActiveSessions.Set(float64(activeSessions))
}

func (ep *E2IndicationProcessor) Close() {
	if ep.kafkaReader != nil {
		ep.kafkaReader.Close()
	}
	if ep.kafkaWriter != nil {
		ep.kafkaWriter.Close()
	}
	if ep.influxClient != nil {
		ep.influxClient.Close()
	}
}

// Health check endpoint
func (ep *E2IndicationProcessor) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":        "healthy",
		"active_cells":  len(ep.correlationDB.cellData),
		"active_ues":    len(ep.correlationDB.ueData),
		"active_slices": len(ep.correlationDB.sliceData),
		"timestamp":     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	log.Println("Starting O-RAN E2 Telemetry Processor...")

	processor := NewE2IndicationProcessor()
	defer processor.Close()

	// Setup HTTP server for health checks and metrics
	router := mux.NewRouter()
	router.HandleFunc("/health", processor.healthCheck).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	server := &http.Server{
		Addr:    ":8089",
		Handler: router,
	}

	// Start HTTP server
	go func() {
		log.Printf("HTTP server starting on :8089")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start main processing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := processor.Start(ctx); err != nil {
			log.Printf("Processor error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down E2 Telemetry Processor...")
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("E2 Telemetry Processor exited")
}

// Placeholder processor implementations
type RICIndicationProcessor struct{}

func NewRICIndicationProcessor() *RICIndicationProcessor {
	return &RICIndicationProcessor{}
}

func (p *RICIndicationProcessor) ProcessIndication(indication *E2Indication) error {
	// Process RIC Control indications
	log.Printf("Processing RIC indication from %s", indication.Source.GlobalE2NodeID)
	return nil
}

func (p *RICIndicationProcessor) GetSupportedMeasurements() []string {
	return []string{"ric_control_outcome", "ric_control_ack", "ric_control_failure"}
}

type KPMIndicationProcessor struct{}

func NewKPMIndicationProcessor() *KPMIndicationProcessor {
	return &KPMIndicationProcessor{}
}

func (p *KPMIndicationProcessor) ProcessIndication(indication *E2Indication) error {
	// Process KPM (Key Performance Measurement) indications
	log.Printf("Processing KPM indication from %s", indication.Source.GlobalE2NodeID)
	return nil
}

func (p *KPMIndicationProcessor) GetSupportedMeasurements() []string {
	return []string{"dl_prb_usage", "ul_prb_usage", "dl_total_prb", "ul_total_prb"}
}

type E2SMProcessor struct{}

func NewE2SMProcessor() *E2SMProcessor {
	return &E2SMProcessor{}
}

func (p *E2SMProcessor) ProcessIndication(indication *E2Indication) error {
	// Process E2SM specific indications
	log.Printf("Processing E2SM indication from %s", indication.Source.GlobalE2NodeID)
	return nil
}

func (p *E2SMProcessor) GetSupportedMeasurements() []string {
	return []string{"e2sm_rc", "e2sm_kpm", "e2sm_ni"}
}