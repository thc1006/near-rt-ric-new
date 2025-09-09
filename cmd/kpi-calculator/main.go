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

	"github.com/redis/go-redis/v9"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// ORanKPI represents calculated O-RAN Key Performance Indicators
type ORanKPI struct {
	Timestamp         time.Time `json:"timestamp"`
	SourceName        string    `json:"source_name"`
	CellID            string    `json:"cell_id"`
	
	// Radio Resource Management KPIs
	PRBUtilizationDL  float64 `json:"prb_utilization_dl"`
	PRBUtilizationUL  float64 `json:"prb_utilization_ul"`
	
	// Throughput KPIs
	ThroughputDL      float64 `json:"throughput_dl_mbps"`
	ThroughputUL      float64 `json:"throughput_ul_mbps"`
	
	// Quality KPIs
	CQI               float64 `json:"cqi"`
	RSRP              float64 `json:"rsrp"`
	RSRQ              float64 `json:"rsrq"`
	SINR              float64 `json:"sinr"`
	
	// Efficiency KPIs
	SpectralEfficiency float64 `json:"spectral_efficiency"`
	EnergyEfficiency   float64 `json:"energy_efficiency"`
	
	// Latency KPIs
	LatencyE2E        float64 `json:"latency_e2e_ms"`
	LatencyRAN        float64 `json:"latency_ran_ms"`
	
	// Connection KPIs
	ActiveUsers       int     `json:"active_users"`
	HandoverRate      float64 `json:"handover_rate"`
	CallDropRate      float64 `json:"call_drop_rate"`
	
	// Network Slicing KPIs
	SliceID           string  `json:"slice_id,omitempty"`
	SliceThroughput   float64 `json:"slice_throughput_mbps,omitempty"`
	SliceLatency      float64 `json:"slice_latency_ms,omitempty"`
}

// KPIAggregation represents aggregated KPI data over time windows
type KPIAggregation struct {
	TimeWindow      string                 `json:"time_window"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	KPIs            map[string]interface{} `json:"kpis"`
	SampleCount     int                    `json:"sample_count"`
	AggregationType string                 `json:"aggregation_type"`
}

// KPICalculator processes telemetry data and calculates O-RAN KPIs
type KPICalculator struct {
	kafkaReader   *kafka.Reader
	kafkaWriter   *kafka.Writer
	influxClient  influxdb2.Client
	queryAPI      api.QueryAPI
	writeAPI      api.WriteAPIBlocking
	redisClient   *redis.Client
	metrics       *KPIMetrics
}

// KPIMetrics defines Prometheus metrics for the KPI calculator
type KPIMetrics struct {
	KPIsCalculated    prometheus.Counter
	KPIErrors         prometheus.Counter
	CalculationTime   prometheus.Histogram
	ActiveSources     prometheus.Gauge
	KPIValues         *prometheus.GaugeVec
}

func NewKPIMetrics() *KPIMetrics {
	return &KPIMetrics{
		KPIsCalculated: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_kpis_calculated_total",
				Help: "Total number of KPIs calculated",
			},
		),
		KPIErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_kpi_errors_total",
				Help: "Total number of KPI calculation errors",
			},
		),
		CalculationTime: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "oran_kpi_calculation_duration_seconds",
				Help:    "Time spent calculating KPIs",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			},
		),
		ActiveSources: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "oran_kpi_active_sources",
				Help: "Number of active KPI sources",
			},
		),
		KPIValues: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "oran_kpi_value",
				Help: "Current KPI values",
			},
			[]string{"kpi_name", "source_name", "cell_id"},
		),
	}
}

func (m *KPIMetrics) Register() {
	prometheus.MustRegister(m.KPIsCalculated)
	prometheus.MustRegister(m.KPIErrors)
	prometheus.MustRegister(m.CalculationTime)
	prometheus.MustRegister(m.ActiveSources)
	prometheus.MustRegister(m.KPIValues)
}

func NewKPICalculator() *KPICalculator {
	// Kafka configuration
	kafkaBrokers := getEnv("KAFKA_BROKERS", "kafka:29092")
	brokersList := strings.Split(kafkaBrokers, ",")

	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokersList,
		Topic:   "ves-measurement",
		GroupID: "kpi-calculator",
	})

	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokersList...),
		Topic:        "oran-kpis",
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
	}

	// InfluxDB configuration
	influxURL := getEnv("INFLUXDB_URL", "http://influxdb:8086")
	influxToken := getEnv("INFLUXDB_TOKEN", "oran-super-secret-token")
	influxOrg := getEnv("INFLUXDB_ORG", "oran")

	influxClient := influxdb2.NewClient(influxURL, influxToken)
	queryAPI := influxClient.QueryAPI(influxOrg)
	writeAPI := influxClient.WriteAPIBlocking(influxOrg, "oran-kpis")

	// Redis configuration
	redisHost := getEnv("REDIS_HOST", "redis")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
		DB:   0,
	})

	metrics := NewKPIMetrics()
	metrics.Register()

	return &KPICalculator{
		kafkaReader:  kafkaReader,
		kafkaWriter:  kafkaWriter,
		influxClient: influxClient,
		queryAPI:     queryAPI,
		writeAPI:     writeAPI,
		redisClient:  redisClient,
		metrics:      metrics,
	}
}

func (kc *KPICalculator) Start(ctx context.Context) error {
	log.Println("Starting KPI Calculator...")

	// Start KPI calculation workers
	for i := 0; i < 3; i++ {
		go kc.kpiCalculationWorker(ctx)
	}

	// Start aggregation worker
	go kc.aggregationWorker(ctx)

	// Process Kafka messages
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := kc.kafkaReader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading Kafka message: %v", err)
				continue
			}

			go kc.processMessage(ctx, message)
		}
	}
}

func (kc *KPICalculator) processMessage(ctx context.Context, message kafka.Message) {
	startTime := time.Now()
	defer func() {
		kc.metrics.CalculationTime.Observe(time.Since(startTime).Seconds())
	}()

	var vesEvent map[string]interface{}
	if err := json.Unmarshal(message.Value, &vesEvent); err != nil {
		log.Printf("Error unmarshaling VES event: %v", err)
		kc.metrics.KPIErrors.Inc()
		return
	}

	kpi, err := kc.calculateKPIs(ctx, vesEvent)
	if err != nil {
		log.Printf("Error calculating KPIs: %v", err)
		kc.metrics.KPIErrors.Inc()
		return
	}

	// Store KPIs
	if err := kc.storeKPIs(ctx, kpi); err != nil {
		log.Printf("Error storing KPIs: %v", err)
		kc.metrics.KPIErrors.Inc()
		return
	}

	// Publish KPIs to Kafka
	if err := kc.publishKPIs(ctx, kpi); err != nil {
		log.Printf("Error publishing KPIs: %v", err)
		kc.metrics.KPIErrors.Inc()
		return
	}

	// Update metrics
	kc.updatePrometheusMetrics(kpi)
	kc.metrics.KPIsCalculated.Inc()
}

func (kc *KPICalculator) calculateKPIs(ctx context.Context, vesEvent map[string]interface{}) (*ORanKPI, error) {
	event, ok := vesEvent["event"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid VES event format")
	}

	commonHeader, ok := event["commonEventHeader"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing common event header")
	}

	measurementFields, ok := event["measurementFields"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing measurement fields")
	}

	kpi := &ORanKPI{
		Timestamp:  time.Now(),
		SourceName: getString(commonHeader, "sourceName"),
		CellID:     getString(commonHeader, "sourceId"),
	}

	// Calculate PRB Utilization
	if err := kc.calculatePRBUtilization(ctx, measurementFields, kpi); err != nil {
		log.Printf("Error calculating PRB utilization: %v", err)
	}

	// Calculate Throughput KPIs
	if err := kc.calculateThroughput(ctx, measurementFields, kpi); err != nil {
		log.Printf("Error calculating throughput: %v", err)
	}

	// Calculate Quality KPIs
	if err := kc.calculateQualityKPIs(ctx, measurementFields, kpi); err != nil {
		log.Printf("Error calculating quality KPIs: %v", err)
	}

	// Calculate Efficiency KPIs
	if err := kc.calculateEfficiencyKPIs(ctx, kpi); err != nil {
		log.Printf("Error calculating efficiency KPIs: %v", err)
	}

	// Calculate Connection KPIs
	if err := kc.calculateConnectionKPIs(ctx, measurementFields, kpi); err != nil {
		log.Printf("Error calculating connection KPIs: %v", err)
	}

	return kpi, nil
}

func (kc *KPICalculator) calculatePRBUtilization(ctx context.Context, measurementFields map[string]interface{}, kpi *ORanKPI) error {
	additionalMeasurements, ok := measurementFields["additionalMeasurements"].([]interface{})
	if !ok {
		return nil
	}

	var prbUsedDL, prbUsedUL, prbAvailableDL, prbAvailableUL float64

	for _, measurement := range additionalMeasurements {
		m, ok := measurement.(map[string]interface{})
		if !ok {
			continue
		}

		hashMap, ok := m["hashMap"].(map[string]interface{})
		if !ok {
			continue
		}

		// Extract PRB metrics from hashMap
		for key, value := range hashMap {
			switch key {
			case "prb.used.dl":
				prbUsedDL = getFloat(value)
			case "prb.used.ul":
				prbUsedUL = getFloat(value)
			case "prb.available.dl":
				prbAvailableDL = getFloat(value)
			case "prb.available.ul":
				prbAvailableUL = getFloat(value)
			}
		}
	}

	if prbAvailableDL > 0 {
		kpi.PRBUtilizationDL = (prbUsedDL / prbAvailableDL) * 100
	}
	if prbAvailableUL > 0 {
		kpi.PRBUtilizationUL = (prbUsedUL / prbAvailableUL) * 100
	}

	return nil
}

func (kc *KPICalculator) calculateThroughput(ctx context.Context, measurementFields map[string]interface{}, kpi *ORanKPI) error {
	additionalMeasurements, ok := measurementFields["additionalMeasurements"].([]interface{})
	if !ok {
		return nil
	}

	var dlBytes, ulBytes float64
	measurementInterval := getFloat(measurementFields["measurementInterval"])

	for _, measurement := range additionalMeasurements {
		m, ok := measurement.(map[string]interface{})
		if !ok {
			continue
		}

		hashMap, ok := m["hashMap"].(map[string]interface{})
		if !ok {
			continue
		}

		for key, value := range hashMap {
			switch key {
			case "mac.volume.dl.bytes":
				dlBytes = getFloat(value)
			case "mac.volume.ul.bytes":
				ulBytes = getFloat(value)
			}
		}
	}

	if measurementInterval > 0 {
		// Convert bytes to Mbps
		kpi.ThroughputDL = (dlBytes * 8) / (measurementInterval * 1_000_000)
		kpi.ThroughputUL = (ulBytes * 8) / (measurementInterval * 1_000_000)
	}

	return nil
}

func (kc *KPICalculator) calculateQualityKPIs(ctx context.Context, measurementFields map[string]interface{}, kpi *ORanKPI) error {
	additionalMeasurements, ok := measurementFields["additionalMeasurements"].([]interface{})
	if !ok {
		return nil
	}

	for _, measurement := range additionalMeasurements {
		m, ok := measurement.(map[string]interface{})
		if !ok {
			continue
		}

		hashMap, ok := m["hashMap"].(map[string]interface{})
		if !ok {
			continue
		}

		for key, value := range hashMap {
			switch key {
			case "cqi.avg":
				kpi.CQI = getFloat(value)
			case "rsrp.avg":
				kpi.RSRP = getFloat(value)
			case "rsrq.avg":
				kpi.RSRQ = getFloat(value)
			case "sinr.avg":
				kpi.SINR = getFloat(value)
			}
		}
	}

	return nil
}

func (kc *KPICalculator) calculateEfficiencyKPIs(ctx context.Context, kpi *ORanKPI) error {
	// Query historical data for power consumption
	query := fmt.Sprintf(`
		from(bucket: "oran-metrics")
		|> range(start: -5m)
		|> filter(fn: (r) => r._measurement == "oran_measurements")
		|> filter(fn: (r) => r.source_name == "%s")
		|> filter(fn: (r) => r._field == "power.consumption.watts")
		|> last()
	`, kpi.SourceName)

	result, err := kc.queryAPI.Query(ctx, query)
	if err != nil {
		return err
	}

	var powerConsumption float64 = 100 // Default value
	for result.Next() {
		if result.Record().Value() != nil {
			powerConsumption = getFloat(result.Record().Value())
		}
	}

	// Calculate Energy Efficiency (Mbps/Watt)
	totalThroughput := kpi.ThroughputDL + kpi.ThroughputUL
	if powerConsumption > 0 {
		kpi.EnergyEfficiency = totalThroughput / powerConsumption
	}

	// Calculate Spectral Efficiency (bps/Hz/cell)
	// Assuming 20 MHz bandwidth for LTE
	bandwidth := 20.0
	if bandwidth > 0 && totalThroughput > 0 {
		kpi.SpectralEfficiency = (totalThroughput * 1_000_000) / (bandwidth * 1_000_000)
	}

	return nil
}

func (kc *KPICalculator) calculateConnectionKPIs(ctx context.Context, measurementFields map[string]interface{}, kpi *ORanKPI) error {
	additionalMeasurements, ok := measurementFields["additionalMeasurements"].([]interface{})
	if !ok {
		return nil
	}

	for _, measurement := range additionalMeasurements {
		m, ok := measurement.(map[string]interface{})
		if !ok {
			continue
		}

		hashMap, ok := m["hashMap"].(map[string]interface{})
		if !ok {
			continue
		}

		for key, value := range hashMap {
			switch key {
			case "rrc.connected.users":
				kpi.ActiveUsers = int(getFloat(value))
			case "handover.success.rate":
				kpi.HandoverRate = getFloat(value)
			case "call.drop.rate":
				kpi.CallDropRate = getFloat(value)
			case "latency.e2e.ms":
				kpi.LatencyE2E = getFloat(value)
			case "latency.ran.ms":
				kpi.LatencyRAN = getFloat(value)
			}
		}
	}

	return nil
}

func (kc *KPICalculator) storeKPIs(ctx context.Context, kpi *ORanKPI) error {
	// Store in InfluxDB
	point := influxdb2.NewPoint("oran_kpis",
		map[string]string{
			"source_name": kpi.SourceName,
			"cell_id":     kpi.CellID,
		},
		map[string]interface{}{
			"prb_utilization_dl":  kpi.PRBUtilizationDL,
			"prb_utilization_ul":  kpi.PRBUtilizationUL,
			"throughput_dl_mbps":  kpi.ThroughputDL,
			"throughput_ul_mbps":  kpi.ThroughputUL,
			"cqi":                kpi.CQI,
			"rsrp":               kpi.RSRP,
			"rsrq":               kpi.RSRQ,
			"sinr":               kpi.SINR,
			"spectral_efficiency": kpi.SpectralEfficiency,
			"energy_efficiency":   kpi.EnergyEfficiency,
			"latency_e2e_ms":     kpi.LatencyE2E,
			"latency_ran_ms":     kpi.LatencyRAN,
			"active_users":       kpi.ActiveUsers,
			"handover_rate":      kpi.HandoverRate,
			"call_drop_rate":     kpi.CallDropRate,
		},
		kpi.Timestamp)

	if err := kc.writeAPI.WritePoint(ctx, point); err != nil {
		return err
	}

	// Store in Redis for fast access
	kpiJSON, err := json.Marshal(kpi)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("kpi:%s:%s", kpi.SourceName, kpi.CellID)
	if err := kc.redisClient.Set(ctx, key, kpiJSON, time.Hour).Err(); err != nil {
		return err
	}

	return nil
}

func (kc *KPICalculator) publishKPIs(ctx context.Context, kpi *ORanKPI) error {
	kpiJSON, err := json.Marshal(kpi)
	if err != nil {
		return err
	}

	message := kafka.Message{
		Key:   []byte(kpi.SourceName),
		Value: kpiJSON,
		Time:  time.Now(),
	}

	return kc.kafkaWriter.WriteMessages(ctx, message)
}

func (kc *KPICalculator) updatePrometheusMetrics(kpi *ORanKPI) {
	kc.metrics.KPIValues.With(prometheus.Labels{
		"kpi_name":    "prb_utilization_dl",
		"source_name": kpi.SourceName,
		"cell_id":     kpi.CellID,
	}).Set(kpi.PRBUtilizationDL)

	kc.metrics.KPIValues.With(prometheus.Labels{
		"kpi_name":    "throughput_dl_mbps",
		"source_name": kpi.SourceName,
		"cell_id":     kpi.CellID,
	}).Set(kpi.ThroughputDL)

	kc.metrics.KPIValues.With(prometheus.Labels{
		"kpi_name":    "energy_efficiency",
		"source_name": kpi.SourceName,
		"cell_id":     kpi.CellID,
	}).Set(kpi.EnergyEfficiency)
}

func (kc *KPICalculator) kpiCalculationWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Periodic KPI calculations and aggregations
			if err := kc.performPeriodicCalculations(ctx); err != nil {
				log.Printf("Error in periodic calculations: %v", err)
			}
		}
	}
}

func (kc *KPICalculator) aggregationWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Perform KPI aggregations
			if err := kc.performAggregations(ctx); err != nil {
				log.Printf("Error in aggregations: %v", err)
			}
		}
	}
}

func (kc *KPICalculator) performPeriodicCalculations(ctx context.Context) error {
	// Implement periodic trend analysis and anomaly detection
	log.Println("Performing periodic KPI calculations...")
	return nil
}

func (kc *KPICalculator) performAggregations(ctx context.Context) error {
	// Calculate hourly, daily aggregations
	timeWindows := []string{"1h", "24h"}
	
	for _, window := range timeWindows {
		if err := kc.calculateAggregations(ctx, window); err != nil {
			return err
		}
	}

	return nil
}

func (kc *KPICalculator) calculateAggregations(ctx context.Context, timeWindow string) error {
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> group(columns: ["source_name", "cell_id"])
		|> aggregateWindow(every: %s, fn: mean)
	`, timeWindow, timeWindow)

	result, err := kc.queryAPI.Query(ctx, query)
	if err != nil {
		return err
	}

	// Process aggregation results
	for result.Next() {
		record := result.Record()
		// Store aggregated KPIs back to InfluxDB with different measurement name
		log.Printf("Aggregated KPI: %s = %v for %s", 
			record.Field(), record.Value(), record.ValueByKey("source_name"))
	}

	return nil
}

func (kc *KPICalculator) Close() {
	if kc.kafkaReader != nil {
		kc.kafkaReader.Close()
	}
	if kc.kafkaWriter != nil {
		kc.kafkaWriter.Close()
	}
	if kc.influxClient != nil {
		kc.influxClient.Close()
	}
	if kc.redisClient != nil {
		kc.redisClient.Close()
	}
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		// Try to parse string as float
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0.0
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	log.Println("Starting O-RAN KPI Calculator...")

	calculator := NewKPICalculator()
	defer calculator.Close()

	// Setup HTTP server for health checks and metrics
	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	server := &http.Server{
		Addr:    ":8086",
		Handler: router,
	}

	// Start HTTP server
	go func() {
		log.Printf("KPI Calculator HTTP server starting on :8086")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start KPI calculation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := calculator.Start(ctx); err != nil {
			log.Fatalf("KPI Calculator failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("KPI Calculator exited")
}