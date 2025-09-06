package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// VESEvent represents a VES (Virtual Event Streaming) event from O-RAN components
type VESEvent struct {
	Event struct {
		CommonEventHeader struct {
			Domain               string    `json:"domain"`
			EventID              string    `json:"eventId"`
			EventName            string    `json:"eventName"`
			EventType            string    `json:"eventType"`
			InternalHeaderFields struct{} `json:"internalHeaderFields"`
			LastEpochMicrosec    int64     `json:"lastEpochMicrosec"`
			NfNamingCode         string    `json:"nfNamingCode"`
			NfcNamingCode        string    `json:"nfcNamingCode"`
			Priority             string    `json:"priority"`
			ReportingEntityID    string    `json:"reportingEntityId"`
			ReportingEntityName  string    `json:"reportingEntityName"`
			Sequence             int       `json:"sequence"`
			SourceID             string    `json:"sourceId"`
			SourceName           string    `json:"sourceName"`
			StartEpochMicrosec   int64     `json:"startEpochMicrosec"`
			TimeZoneOffset       string    `json:"timeZoneOffset"`
			Version              string    `json:"version"`
			VesEventListenerVer  string    `json:"vesEventListenerVersion"`
		} `json:"commonEventHeader"`
		MeasurementFields *struct {
			MeasurementInterval     int                    `json:"measurementInterval"`
			MeasurementFieldsVer    string                 `json:"measurementFieldsVersion"`
			AdditionalMeasurements  []AdditionalMeasurement `json:"additionalMeasurements,omitempty"`
			CpuUsageArray          []CPUUsage             `json:"cpuUsageArray,omitempty"`
			DiskUsageArray         []DiskUsage            `json:"diskUsageArray,omitempty"`
			FilesystemUsageArray   []FilesystemUsage      `json:"filesystemUsageArray,omitempty"`
			MemoryUsageArray       []MemoryUsage          `json:"memoryUsageArray,omitempty"`
			NicPerformanceArray    []NICPerformance       `json:"nicPerformanceArray,omitempty"`
		} `json:"measurementFields,omitempty"`
		FaultFields *struct {
			FaultFieldsVersion   string `json:"faultFieldsVersion"`
			EventSeverity        string `json:"eventSeverity"`
			EventSourceType      string `json:"eventSourceType"`
			AlarmCondition       string `json:"alarmCondition"`
			SpecificProblem      string `json:"specificProblem"`
			VfStatus             string `json:"vfStatus"`
			AlarmInterfaceA      string `json:"alarmInterfaceA,omitempty"`
			AlarmAdditionalInfo  []AlarmAdditionalInfo `json:"alarmAdditionalInformation,omitempty"`
		} `json:"faultFields,omitempty"`
	} `json:"event"`
}

type AdditionalMeasurement struct {
	Name       string            `json:"name"`
	Hashmap    map[string]string `json:"hashMap"`
}

type CPUUsage struct {
	CPUIdentifier       string  `json:"cpuIdentifier"`
	PercentUsage        float64 `json:"percentUsage"`
	CPUIdle             float64 `json:"cpuIdle,omitempty"`
	CPUUsageInterrupt   float64 `json:"cpuUsageInterrupt,omitempty"`
	CPUUsageNice        float64 `json:"cpuUsageNice,omitempty"`
	CPUUsageSoftIrq     float64 `json:"cpuUsageSoftIrq,omitempty"`
	CPUUsageSteal       float64 `json:"cpuUsageSteal,omitempty"`
	CPUUsageSystem      float64 `json:"cpuUsageSystem,omitempty"`
	CPUUsageUser        float64 `json:"cpuUsageUser,omitempty"`
	CPUWait             float64 `json:"cpuWait,omitempty"`
}

type DiskUsage struct {
	DiskIdentifier  string  `json:"diskIdentifier"`
	DiskIopsRead    float64 `json:"diskIopsRead"`
	DiskIopsWrite   float64 `json:"diskIopsWrite"`
	DiskMergedRead  float64 `json:"diskMergedRead"`
	DiskMergedWrite float64 `json:"diskMergedWrite"`
	DiskOctetsRead  float64 `json:"diskOctetsRead"`
	DiskOctetsWrite float64 `json:"diskOctetsWrite"`
	DiskTime        float64 `json:"diskTime"`
}

type FilesystemUsage struct {
	FilesystemName      string  `json:"filesystemName"`
	BlockConfigured     float64 `json:"blockConfigured"`
	BlockInode          float64 `json:"blockInode"`
	BlockUsed           float64 `json:"blockUsed"`
}

type MemoryUsage struct {
	VmIdentifier        string  `json:"vmIdentifier"`
	MemoryBuffered      float64 `json:"memoryBuffered,omitempty"`
	MemoryCached        float64 `json:"memoryCached,omitempty"`
	MemoryConfigured    float64 `json:"memoryConfigured,omitempty"`
	MemoryFree          float64 `json:"memoryFree,omitempty"`
	MemorySlabRecl      float64 `json:"memorySlabRecl,omitempty"`
	MemorySlabUnrecl    float64 `json:"memorySlabUnrecl,omitempty"`
	MemoryUsed          float64 `json:"memoryUsed,omitempty"`
}

type NICPerformance struct {
	NicIdentifier               string  `json:"nicIdentifier"`
	ReceivedBroadcastPackets    float64 `json:"receivedBroadcastPackets,omitempty"`
	ReceivedDiscardedPackets    float64 `json:"receivedDiscardedPackets,omitempty"`
	ReceivedErrorPackets        float64 `json:"receivedErrorPackets,omitempty"`
	ReceivedMulticastPackets    float64 `json:"receivedMulticastPackets,omitempty"`
	ReceivedOctets              float64 `json:"receivedOctets,omitempty"`
	ReceivedTotalPackets        float64 `json:"receivedTotalPackets,omitempty"`
	ReceivedUnicastPackets      float64 `json:"receivedUnicastPackets,omitempty"`
	TransmittedBroadcastPackets float64 `json:"transmittedBroadcastPackets,omitempty"`
	TransmittedDiscardedPackets float64 `json:"transmittedDiscardedPackets,omitempty"`
	TransmittedErrorPackets     float64 `json:"transmittedErrorPackets,omitempty"`
	TransmittedMulticastPackets float64 `json:"transmittedMulticastPackets,omitempty"`
	TransmittedOctets           float64 `json:"transmittedOctets,omitempty"`
	TransmittedTotalPackets     float64 `json:"transmittedTotalPackets,omitempty"`
	TransmittedUnicastPackets   float64 `json:"transmittedUnicastPackets,omitempty"`
}

type AlarmAdditionalInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TelemetryCollector manages the collection and processing of O-RAN telemetry data
type TelemetryCollector struct {
	kafkaWriter   *kafka.Writer
	influxClient  influxdb2.Client
	writeAPI      api.WriteAPIBlocking
	metrics       *CollectorMetrics
}

// CollectorMetrics defines Prometheus metrics for the telemetry collector
type CollectorMetrics struct {
	EventsReceived    prometheus.Counter
	EventsProcessed   prometheus.Counter
	EventsDropped     prometheus.Counter
	ProcessingLatency prometheus.Histogram
	ActiveSources     prometheus.Gauge
}

func NewCollectorMetrics() *CollectorMetrics {
	return &CollectorMetrics{
		EventsReceived: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_telemetry_events_received_total",
				Help: "Total number of VES events received",
			},
		),
		EventsProcessed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_telemetry_events_processed_total",
				Help: "Total number of VES events successfully processed",
			},
		),
		EventsDropped: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_telemetry_events_dropped_total",
				Help: "Total number of VES events dropped due to errors",
			},
		),
		ProcessingLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "oran_telemetry_processing_latency_seconds",
				Help:    "Processing latency for VES events",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			},
		),
		ActiveSources: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "oran_telemetry_active_sources",
				Help: "Number of active telemetry sources",
			},
		),
	}
}

func (m *CollectorMetrics) Register() {
	prometheus.MustRegister(m.EventsReceived)
	prometheus.MustRegister(m.EventsProcessed)
	prometheus.MustRegister(m.EventsDropped)
	prometheus.MustRegister(m.ProcessingLatency)
	prometheus.MustRegister(m.ActiveSources)
}

func NewTelemetryCollector() *TelemetryCollector {
	// Kafka configuration
	kafkaBrokers := getEnv("KAFKA_BROKERS", "kafka:29092")
	brokersList := strings.Split(kafkaBrokers, ",")

	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokersList...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
	}

	// InfluxDB configuration
	influxURL := getEnv("INFLUXDB_URL", "http://influxdb:8086")
	influxToken := getEnv("INFLUXDB_TOKEN", "oran-super-secret-token")
	influxOrg := getEnv("INFLUXDB_ORG", "oran")

	influxClient := influxdb2.NewClient(influxURL, influxToken)
	writeAPI := influxClient.WriteAPIBlocking(influxOrg, "oran-metrics")

	metrics := NewCollectorMetrics()
	metrics.Register()

	return &TelemetryCollector{
		kafkaWriter:   kafkaWriter,
		influxClient:  influxClient,
		writeAPI:      writeAPI,
		metrics:       metrics,
	}
}

func (tc *TelemetryCollector) processVESEvent(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		tc.metrics.ProcessingLatency.Observe(time.Since(startTime).Seconds())
	}()

	tc.metrics.EventsReceived.Inc()

	var vesEvent VESEvent
	if err := json.NewDecoder(r.Body).Decode(&vesEvent); err != nil {
		log.Printf("Error decoding VES event: %v", err)
		tc.metrics.EventsDropped.Inc()
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Process the event asynchronously
	go func() {
		if err := tc.handleVESEvent(vesEvent); err != nil {
			log.Printf("Error handling VES event: %v", err)
			tc.metrics.EventsDropped.Inc()
			return
		}
		tc.metrics.EventsProcessed.Inc()
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (tc *TelemetryCollector) handleVESEvent(event VESEvent) error {
	ctx := context.Background()

	// Send to Kafka for stream processing
	if err := tc.sendToKafka(ctx, event); err != nil {
		return fmt.Errorf("failed to send to Kafka: %w", err)
	}

	// Store in InfluxDB for time-series analysis
	if err := tc.storeInInfluxDB(ctx, event); err != nil {
		return fmt.Errorf("failed to store in InfluxDB: %w", err)
	}

	return nil
}

func (tc *TelemetryCollector) sendToKafka(ctx context.Context, event VESEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	topic := tc.getKafkaTopic(event)
	
	message := kafka.Message{
		Topic: topic,
		Key:   []byte(event.Event.CommonEventHeader.SourceName),
		Value: eventJSON,
		Time:  time.Now(),
	}

	return tc.kafkaWriter.WriteMessages(ctx, message)
}

func (tc *TelemetryCollector) getKafkaTopic(event VESEvent) string {
	domain := event.Event.CommonEventHeader.Domain
	switch domain {
	case "measurement":
		return "ves-measurement"
	case "fault":
		return "ves-fault"
	case "notification":
		return "ves-notification"
	case "pnfRegistration":
		return "ves-registration"
	default:
		return "ves-other"
	}
}

func (tc *TelemetryCollector) storeInInfluxDB(ctx context.Context, event VESEvent) error {
	commonHeader := event.Event.CommonEventHeader
	timestamp := time.Unix(0, commonHeader.LastEpochMicrosec*1000) // Convert microseconds to nanoseconds

	// Create base point with common tags
	tags := map[string]string{
		"source_name":     commonHeader.SourceName,
		"source_id":       commonHeader.SourceID,
		"domain":          commonHeader.Domain,
		"event_type":      commonHeader.EventType,
		"reporting_entity": commonHeader.ReportingEntityName,
	}

	// Process measurement fields if present
	if event.Event.MeasurementFields != nil {
		if err := tc.storeMeasurementData(ctx, *event.Event.MeasurementFields, tags, timestamp); err != nil {
			return err
		}
	}

	// Process fault fields if present
	if event.Event.FaultFields != nil {
		if err := tc.storeFaultData(ctx, *event.Event.FaultFields, tags, timestamp); err != nil {
			return err
		}
	}

	return nil
}

func (tc *TelemetryCollector) storeMeasurementData(ctx context.Context, mf interface{}, baseTags map[string]string, timestamp time.Time) error {
	// Type assertion for MeasurementFields
	measurementFields, ok := mf.(struct {
		MeasurementInterval     int                    `json:"measurementInterval"`
		MeasurementFieldsVer    string                 `json:"measurementFieldsVersion"`
		AdditionalMeasurements  []AdditionalMeasurement `json:"additionalMeasurements,omitempty"`
		CpuUsageArray          []CPUUsage             `json:"cpuUsageArray,omitempty"`
		DiskUsageArray         []DiskUsage            `json:"diskUsageArray,omitempty"`
		FilesystemUsageArray   []FilesystemUsage      `json:"filesystemUsageArray,omitempty"`
		MemoryUsageArray       []MemoryUsage          `json:"memoryUsageArray,omitempty"`
		NicPerformanceArray    []NICPerformance       `json:"nicPerformanceArray,omitempty"`
	})
	if !ok {
		return fmt.Errorf("invalid measurement fields type")
	}

	// Store CPU metrics
	for _, cpu := range measurementFields.CpuUsageArray {
		point := influxdb2.NewPoint("cpu_usage",
			mergeTags(baseTags, map[string]string{"cpu_id": cpu.CPUIdentifier}),
			map[string]interface{}{
				"percent_usage":       cpu.PercentUsage,
				"cpu_idle":           cpu.CPUIdle,
				"cpu_usage_interrupt": cpu.CPUUsageInterrupt,
				"cpu_usage_nice":     cpu.CPUUsageNice,
				"cpu_usage_system":   cpu.CPUUsageSystem,
				"cpu_usage_user":     cpu.CPUUsageUser,
				"cpu_wait":           cpu.CPUWait,
			},
			timestamp)
		if err := tc.writeAPI.WritePoint(ctx, point); err != nil {
			return err
		}
	}

	// Store Memory metrics
	for _, mem := range measurementFields.MemoryUsageArray {
		point := influxdb2.NewPoint("memory_usage",
			mergeTags(baseTags, map[string]string{"vm_id": mem.VmIdentifier}),
			map[string]interface{}{
				"memory_free":      mem.MemoryFree,
				"memory_used":      mem.MemoryUsed,
				"memory_buffered":  mem.MemoryBuffered,
				"memory_cached":    mem.MemoryCached,
				"memory_configured": mem.MemoryConfigured,
			},
			timestamp)
		if err := tc.writeAPI.WritePoint(ctx, point); err != nil {
			return err
		}
	}

	// Store NIC Performance metrics
	for _, nic := range measurementFields.NicPerformanceArray {
		point := influxdb2.NewPoint("nic_performance",
			mergeTags(baseTags, map[string]string{"nic_id": nic.NicIdentifier}),
			map[string]interface{}{
				"received_octets":              nic.ReceivedOctets,
				"received_total_packets":       nic.ReceivedTotalPackets,
				"received_error_packets":       nic.ReceivedErrorPackets,
				"transmitted_octets":           nic.TransmittedOctets,
				"transmitted_total_packets":    nic.TransmittedTotalPackets,
				"transmitted_error_packets":    nic.TransmittedErrorPackets,
			},
			timestamp)
		if err := tc.writeAPI.WritePoint(ctx, point); err != nil {
			return err
		}
	}

	// Store additional measurements (O-RAN specific KPIs)
	for _, additional := range measurementFields.AdditionalMeasurements {
		fields := make(map[string]interface{})
		for key, value := range additional.Hashmap {
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				fields[key] = floatVal
			} else {
				fields[key] = value
			}
		}

		point := influxdb2.NewPoint("oran_measurements",
			mergeTags(baseTags, map[string]string{"measurement_name": additional.Name}),
			fields,
			timestamp)
		if err := tc.writeAPI.WritePoint(ctx, point); err != nil {
			return err
		}
	}

	return nil
}

func (tc *TelemetryCollector) storeFaultData(ctx context.Context, ff interface{}, baseTags map[string]string, timestamp time.Time) error {
	// Type assertion for FaultFields
	faultFields, ok := ff.(struct {
		FaultFieldsVersion   string `json:"faultFieldsVersion"`
		EventSeverity        string `json:"eventSeverity"`
		EventSourceType      string `json:"eventSourceType"`
		AlarmCondition       string `json:"alarmCondition"`
		SpecificProblem      string `json:"specificProblem"`
		VfStatus             string `json:"vfStatus"`
		AlarmInterfaceA      string `json:"alarmInterfaceA,omitempty"`
		AlarmAdditionalInfo  []AlarmAdditionalInfo `json:"alarmAdditionalInformation,omitempty"`
	})
	if !ok {
		return fmt.Errorf("invalid fault fields type")
	}

	severityValue := tc.getSeverityValue(faultFields.EventSeverity)

	point := influxdb2.NewPoint("oran_faults",
		mergeTags(baseTags, map[string]string{
			"severity":        faultFields.EventSeverity,
			"source_type":     faultFields.EventSourceType,
			"alarm_condition": faultFields.AlarmCondition,
		}),
		map[string]interface{}{
			"severity_value":    severityValue,
			"specific_problem":  faultFields.SpecificProblem,
			"vf_status":        faultFields.VfStatus,
			"alarm_interface":  faultFields.AlarmInterfaceA,
		},
		timestamp)

	return tc.writeAPI.WritePoint(ctx, point)
}

func (tc *TelemetryCollector) getSeverityValue(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 5
	case "major":
		return 4
	case "minor":
		return 3
	case "warning":
		return 2
	case "normal":
		return 1
	default:
		return 0
	}
}

func mergeTags(tags1, tags2 map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range tags1 {
		result[k] = v
	}
	for k, v := range tags2 {
		result[k] = v
	}
	return result
}

func (tc *TelemetryCollector) healthCheck(w http.ResponseWriter, r *http.Request) {
	// Check InfluxDB connection
	health, err := tc.influxClient.Health(context.Background())
	if err != nil || health.Status != "pass" {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"reason": "InfluxDB connection failed",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (tc *TelemetryCollector) Close() {
	if tc.kafkaWriter != nil {
		tc.kafkaWriter.Close()
	}
	if tc.influxClient != nil {
		tc.influxClient.Close()
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	log.Println("Starting O-RAN Telemetry Collector...")

	collector := NewTelemetryCollector()
	defer collector.Close()

	router := mux.NewRouter()

	// VES event endpoint
	vesEndpoint := getEnv("VES_ENDPOINT", "/api/v1/ves")
	router.HandleFunc(vesEndpoint, collector.processVESEvent).Methods("POST")
	
	// Health check endpoint
	router.HandleFunc("/health", collector.healthCheck).Methods("GET")
	
	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// API info endpoint
	router.HandleFunc("/api/v1/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "O-RAN Telemetry Collector",
			"version": "1.0.0",
			"endpoints": map[string]string{
				"ves":     vesEndpoint,
				"health":  "/health",
				"metrics": "/metrics",
			},
		})
	}).Methods("GET")

	server := &http.Server{
		Addr:         ":8085",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Telemetry Collector server starting on :8085")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}