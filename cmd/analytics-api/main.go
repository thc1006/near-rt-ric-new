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
	"github.com/influxdata/influxdb-client-go/v2/api/query"
)

// AnalyticsAPI provides REST API access to O-RAN telemetry data and ML predictions
type AnalyticsAPI struct {
	influxClient influxdb2.Client
	queryAPI     api.QueryAPI
	kafkaWriter  *kafka.Writer
	metrics      *APIMetrics
}

// KPISummary represents aggregated KPI data
type KPISummary struct {
	TimeRange         string    `json:"time_range"`
	SourceName        string    `json:"source_name"`
	CellID            string    `json:"cell_id,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
	
	// Resource utilization
	AvgPRBUtilizationDL float64 `json:"avg_prb_utilization_dl"`
	MaxPRBUtilizationDL float64 `json:"max_prb_utilization_dl"`
	MinPRBUtilizationDL float64 `json:"min_prb_utilization_dl"`
	
	// Throughput metrics
	AvgThroughputDL     float64 `json:"avg_throughput_dl_mbps"`
	MaxThroughputDL     float64 `json:"max_throughput_dl_mbps"`
	PeakThroughputTime  string  `json:"peak_throughput_time"`
	
	// Quality metrics
	AvgRSRP            float64 `json:"avg_rsrp"`
	AvgRSRQ            float64 `json:"avg_rsrq"`
	AvgCQI             float64 `json:"avg_cqi"`
	
	// Efficiency metrics
	AvgEnergyEfficiency float64 `json:"avg_energy_efficiency"`
	AvgSpectralEfficiency float64 `json:"avg_spectral_efficiency"`
	
	// Connection metrics
	AvgActiveUsers     int     `json:"avg_active_users"`
	MaxActiveUsers     int     `json:"max_active_users"`
	AvgHandoverRate    float64 `json:"avg_handover_rate"`
	AvgCallDropRate    float64 `json:"avg_call_drop_rate"`
	
	// Anomaly information
	AnomalyCount       int     `json:"anomaly_count"`
	CriticalAlerts     int     `json:"critical_alerts"`
	SampleCount        int     `json:"sample_count"`
}

// PredictionSummary represents ML prediction results
type PredictionSummary struct {
	TimeRange          string    `json:"time_range"`
	SourceName         string    `json:"source_name"`
	Timestamp          time.Time `json:"timestamp"`
	
	LoadPredictions    []LoadForecast    `json:"load_predictions"`
	QualityPredictions []QualityForecast `json:"quality_predictions"`
	AnomalyAlerts      []AnomalyAlert    `json:"anomaly_alerts"`
	Recommendations    []string          `json:"recommendations"`
	
	ModelAccuracy      float64   `json:"model_accuracy"`
	ConfidenceLevel    float64   `json:"confidence_level"`
}

type LoadForecast struct {
	Time            time.Time `json:"time"`
	PredictedLoad   float64   `json:"predicted_load"`
	PredictedUsers  int       `json:"predicted_users"`
	Trend           string    `json:"trend"`
}

type QualityForecast struct {
	Time            time.Time `json:"time"`
	PredictedRSRP   float64   `json:"predicted_rsrp"`
	PredictedRSRQ   float64   `json:"predicted_rsrq"`
	Trend           string    `json:"trend"`
}

type AnomalyAlert struct {
	Time            time.Time `json:"time"`
	AnomalyType     string    `json:"anomaly_type"`
	Severity        string    `json:"severity"`
	AnomalyScore    float64   `json:"anomaly_score"`
	Description     string    `json:"description"`
}

// NetworkInsights provides comprehensive network analysis
type NetworkInsights struct {
	Timestamp           time.Time `json:"timestamp"`
	AnalysisPeriod      string    `json:"analysis_period"`
	
	// Overall network health
	OverallHealth       string    `json:"overall_health"`       // "excellent", "good", "fair", "poor"
	HealthScore         float64   `json:"health_score"`         // 0-100
	
	// Performance insights
	TopPerformingSites  []string  `json:"top_performing_sites"`
	UnderperformingSites []string `json:"underperforming_sites"`
	
	// Capacity insights
	HighLoadSites       []string  `json:"high_load_sites"`
	CapacityUtilization float64   `json:"capacity_utilization"`
	PeakHourAnalysis    string    `json:"peak_hour_analysis"`
	
	// Quality insights
	QualityTrends       string    `json:"quality_trends"`
	CoverageGaps        []string  `json:"coverage_gaps"`
	
	// Efficiency insights
	EnergyEfficiencyScore float64 `json:"energy_efficiency_score"`
	SpectrumEfficiency    float64 `json:"spectrum_efficiency"`
	
	// Predictive insights
	UpcomingIssues      []string  `json:"upcoming_issues"`
	MaintenanceNeeded   []string  `json:"maintenance_needed"`
	OptimizationOpportunities []string `json:"optimization_opportunities"`
}

// APIMetrics defines Prometheus metrics for the analytics API
type APIMetrics struct {
	RequestsTotal     *prometheus.CounterVec
	RequestDuration   *prometheus.HistogramVec
	ActiveQueries     prometheus.Gauge
	DataPointsQueried prometheus.Counter
}

func NewAPIMetrics() *APIMetrics {
	return &APIMetrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oran_analytics_api_requests_total",
				Help: "Total number of API requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "oran_analytics_api_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
			},
			[]string{"method", "endpoint"},
		),
		ActiveQueries: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "oran_analytics_api_active_queries",
				Help: "Number of active database queries",
			},
		),
		DataPointsQueried: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_analytics_api_datapoints_queried_total",
				Help: "Total number of data points queried",
			},
		),
	}
}

func (m *APIMetrics) Register() {
	prometheus.MustRegister(m.RequestsTotal)
	prometheus.MustRegister(m.RequestDuration)
	prometheus.MustRegister(m.ActiveQueries)
	prometheus.MustRegister(m.DataPointsQueried)
}

func NewAnalyticsAPI() *AnalyticsAPI {
	// InfluxDB configuration
	influxURL := getEnv("INFLUXDB_URL", "http://influxdb:8086")
	influxToken := getEnv("INFLUXDB_TOKEN", "oran-super-secret-token")
	influxOrg := getEnv("INFLUXDB_ORG", "oran")

	influxClient := influxdb2.NewClient(influxURL, influxToken)
	queryAPI := influxClient.QueryAPI(influxOrg)

	// Kafka configuration for real-time notifications
	kafkaBrokers := getEnv("KAFKA_BROKERS", "kafka:29092")
	brokersList := strings.Split(kafkaBrokers, ",")

	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokersList...),
		Topic:    "analytics-requests",
		Balancer: &kafka.LeastBytes{},
	}

	metrics := NewAPIMetrics()
	metrics.Register()

	return &AnalyticsAPI{
		influxClient: influxClient,
		queryAPI:     queryAPI,
		kafkaWriter:  kafkaWriter,
		metrics:      metrics,
	}
}

func (api *AnalyticsAPI) middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Content-Type", "application/json")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		endpoint := getEndpointName(r.URL.Path)
		
		api.metrics.RequestsTotal.WithLabelValues(
			r.Method, endpoint, strconv.Itoa(wrapped.statusCode),
		).Inc()
		
		api.metrics.RequestDuration.WithLabelValues(
			r.Method, endpoint,
		).Observe(duration.Seconds())
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// API Handlers

func (api *AnalyticsAPI) getKPISummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	api.metrics.ActiveQueries.Inc()
	defer api.metrics.ActiveQueries.Dec()

	// Parse query parameters
	params := r.URL.Query()
	timeRange := params.Get("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}
	
	sourceName := params.Get("source_name")
	cellID := params.Get("cell_id")

	summary, err := api.calculateKPISummary(ctx, timeRange, sourceName, cellID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error calculating KPI summary: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(summary)
}

func (api *AnalyticsAPI) getPredictions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	api.metrics.ActiveQueries.Inc()
	defer api.metrics.ActiveQueries.Dec()

	params := r.URL.Query()
	timeRange := params.Get("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}
	
	sourceName := params.Get("source_name")

	predictions, err := api.getPredictionSummary(ctx, timeRange, sourceName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting predictions: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(predictions)
}

func (api *AnalyticsAPI) getNetworkInsights(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	api.metrics.ActiveQueries.Inc()
	defer api.metrics.ActiveQueries.Dec()

	params := r.URL.Query()
	analysisPeriod := params.Get("period")
	if analysisPeriod == "" {
		analysisPeriod = "24h"
	}

	insights, err := api.generateNetworkInsights(ctx, analysisPeriod)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating insights: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(insights)
}

func (api *AnalyticsAPI) getTimeSeriesData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	api.metrics.ActiveQueries.Inc()
	defer api.metrics.ActiveQueries.Dec()

	params := r.URL.Query()
	measurement := params.Get("measurement")
	field := params.Get("field")
	timeRange := params.Get("time_range")
	sourceName := params.Get("source_name")
	
	if measurement == "" || field == "" {
		http.Error(w, "measurement and field parameters are required", http.StatusBadRequest)
		return
	}
	
	if timeRange == "" {
		timeRange = "1h"
	}

	data, err := api.queryTimeSeriesData(ctx, measurement, field, timeRange, sourceName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying time series data: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(data)
}

func (api *AnalyticsAPI) getRealTimeMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	params := r.URL.Query()
	sourceName := params.Get("source_name")

	metrics, err := api.getCurrentMetrics(ctx, sourceName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting real-time metrics: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(metrics)
}

// Core logic methods

func (api *AnalyticsAPI) calculateKPISummary(ctx context.Context, timeRange, sourceName, cellID string) (*KPISummary, error) {
	var sourceFilter string
	if sourceName != "" {
		sourceFilter = fmt.Sprintf(`|> filter(fn: (r) => r.source_name == "%s")`, sourceName)
	}
	if cellID != "" {
		sourceFilter += fmt.Sprintf(`|> filter(fn: (r) => r.cell_id == "%s")`, cellID)
	}

	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		%s
		|> group(columns: ["source_name", "cell_id"])
		|> aggregateWindow(every: 5m, fn: mean)
	`, timeRange, sourceFilter)

	result, err := api.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	summary := &KPISummary{
		TimeRange: timeRange,
		Timestamp: time.Now(),
	}

	var prbValues, throughputValues, rsrpValues, rsrqValues, cqiValues []float64
	var userValues []int
	var anomalyCount, sampleCount int

	for result.Next() {
		record := result.Record()
		api.metrics.DataPointsQueried.Inc()
		sampleCount++

		if summary.SourceName == "" {
			summary.SourceName = getStringFromRecord(record, "source_name")
		}
		if summary.CellID == "" {
			summary.CellID = getStringFromRecord(record, "cell_id")
		}

		switch record.Field() {
		case "prb_utilization_dl":
			prbValues = append(prbValues, getFloatFromRecord(record))
		case "throughput_dl_mbps":
			throughputValues = append(throughputValues, getFloatFromRecord(record))
		case "rsrp":
			rsrpValues = append(rsrpValues, getFloatFromRecord(record))
		case "rsrq":
			rsrqValues = append(rsrqValues, getFloatFromRecord(record))
		case "cqi":
			cqiValues = append(cqiValues, getFloatFromRecord(record))
		case "active_users":
			userValues = append(userValues, int(getFloatFromRecord(record)))
		}
	}

	// Calculate statistics
	if len(prbValues) > 0 {
		summary.AvgPRBUtilizationDL = calculateMean(prbValues)
		summary.MaxPRBUtilizationDL = calculateMax(prbValues)
		summary.MinPRBUtilizationDL = calculateMin(prbValues)
	}

	if len(throughputValues) > 0 {
		summary.AvgThroughputDL = calculateMean(throughputValues)
		summary.MaxThroughputDL = calculateMax(throughputValues)
	}

	if len(rsrpValues) > 0 {
		summary.AvgRSRP = calculateMean(rsrpValues)
	}
	if len(rsrqValues) > 0 {
		summary.AvgRSRQ = calculateMean(rsrqValues)
	}
	if len(cqiValues) > 0 {
		summary.AvgCQI = calculateMean(cqiValues)
	}

	if len(userValues) > 0 {
		summary.AvgActiveUsers = calculateMeanInt(userValues)
		summary.MaxActiveUsers = calculateMaxInt(userValues)
	}

	summary.AnomalyCount = anomalyCount
	summary.SampleCount = sampleCount

	return summary, nil
}

func (api *AnalyticsAPI) getPredictionSummary(ctx context.Context, timeRange, sourceName string) (*PredictionSummary, error) {
	var sourceFilter string
	if sourceName != "" {
		sourceFilter = fmt.Sprintf(`|> filter(fn: (r) => r.source_name == "%s")`, sourceName)
	}

	query := fmt.Sprintf(`
		from(bucket: "oran-predictions")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_predictions")
		%s
		|> sort(columns: ["_time"])
	`, timeRange, sourceFilter)

	result, err := api.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	summary := &PredictionSummary{
		TimeRange: timeRange,
		Timestamp: time.Now(),
		LoadPredictions: make([]LoadForecast, 0),
		QualityPredictions: make([]QualityForecast, 0),
		AnomalyAlerts: make([]AnomalyAlert, 0),
		Recommendations: make([]string, 0),
	}

	for result.Next() {
		record := result.Record()
		
		if summary.SourceName == "" {
			summary.SourceName = getStringFromRecord(record, "source_name")
		}

		timestamp := record.Time()

		switch record.Field() {
		case "predicted_load":
			forecast := LoadForecast{
				Time:          timestamp,
				PredictedLoad: getFloatFromRecord(record),
				Trend:         getStringFromRecord(record, "load_trend"),
			}
			summary.LoadPredictions = append(summary.LoadPredictions, forecast)

		case "predicted_rsrp":
			forecast := QualityForecast{
				Time:          timestamp,
				PredictedRSRP: getFloatFromRecord(record),
				Trend:         getStringFromRecord(record, "quality_trend"),
			}
			summary.QualityPredictions = append(summary.QualityPredictions, forecast)

		case "is_anomaly":
			if getBoolFromRecord(record) {
				alert := AnomalyAlert{
					Time:         timestamp,
					AnomalyType:  getStringFromRecord(record, "anomaly_type"),
					Severity:     "medium", // Default severity
					AnomalyScore: getFloatFromRecord(record),
					Description:  fmt.Sprintf("Anomaly detected: %s", getStringFromRecord(record, "anomaly_type")),
				}
				summary.AnomalyAlerts = append(summary.AnomalyAlerts, alert)
			}
		}
	}

	return summary, nil
}

func (api *AnalyticsAPI) generateNetworkInsights(ctx context.Context, period string) (*NetworkInsights, error) {
	insights := &NetworkInsights{
		Timestamp:                 time.Now(),
		AnalysisPeriod:           period,
		TopPerformingSites:       make([]string, 0),
		UnderperformingSites:     make([]string, 0),
		HighLoadSites:           make([]string, 0),
		CoverageGaps:            make([]string, 0),
		UpcomingIssues:          make([]string, 0),
		MaintenanceNeeded:       make([]string, 0),
		OptimizationOpportunities: make([]string, 0),
	}

	// Calculate overall network health
	healthScore, err := api.calculateNetworkHealthScore(ctx, period)
	if err != nil {
		return nil, err
	}

	insights.HealthScore = healthScore
	insights.OverallHealth = api.getHealthStatus(healthScore)

	// Get site performance analysis
	topSites, err := api.getTopPerformingSites(ctx, period, 5)
	if err == nil {
		insights.TopPerformingSites = topSites
	}

	underSites, err := api.getUnderperformingSites(ctx, period, 5)
	if err == nil {
		insights.UnderperformingSites = underSites
	}

	// Get capacity analysis
	highLoadSites, err := api.getHighLoadSites(ctx, period)
	if err == nil {
		insights.HighLoadSites = highLoadSites
	}

	// Calculate efficiency metrics
	energyScore, err := api.calculateEnergyEfficiencyScore(ctx, period)
	if err == nil {
		insights.EnergyEfficiencyScore = energyScore
	}

	spectrumEff, err := api.calculateSpectrumEfficiency(ctx, period)
	if err == nil {
		insights.SpectrumEfficiency = spectrumEff
	}

	// Generate predictive insights
	insights.UpcomingIssues = api.predictUpcomingIssues(ctx, period)
	insights.OptimizationOpportunities = api.identifyOptimizationOpportunities(insights)

	return insights, nil
}

func (api *AnalyticsAPI) queryTimeSeriesData(ctx context.Context, measurement, field, timeRange, sourceName string) (map[string]interface{}, error) {
	var sourceFilter string
	if sourceName != "" {
		sourceFilter = fmt.Sprintf(`|> filter(fn: (r) => r.source_name == "%s")`, sourceName)
	}

	query := fmt.Sprintf(`
		from(bucket: "oran-metrics")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "%s")
		|> filter(fn: (r) => r._field == "%s")
		%s
		|> sort(columns: ["_time"])
	`, timeRange, measurement, field, sourceFilter)

	result, err := api.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"measurement": measurement,
		"field":       field,
		"time_range":  timeRange,
		"data_points": make([]map[string]interface{}, 0),
	}

	for result.Next() {
		record := result.Record()
		api.metrics.DataPointsQueried.Inc()

		point := map[string]interface{}{
			"time":        record.Time(),
			"value":       record.Value(),
			"source_name": getStringFromRecord(record, "source_name"),
		}

		if cellID := getStringFromRecord(record, "cell_id"); cellID != "" {
			point["cell_id"] = cellID
		}

		data["data_points"] = append(data["data_points"].([]map[string]interface{}), point)
	}

	return data, nil
}

func (api *AnalyticsAPI) getCurrentMetrics(ctx context.Context, sourceName string) (map[string]interface{}, error) {
	var sourceFilter string
	if sourceName != "" {
		sourceFilter = fmt.Sprintf(`|> filter(fn: (r) => r.source_name == "%s")`, sourceName)
	}

	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -5m)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		%s
		|> last()
		|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
	`, sourceFilter)

	result, err := api.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]interface{})
	
	for result.Next() {
		record := result.Record()
		metrics["timestamp"] = record.Time()
		metrics["source_name"] = getStringFromRecord(record, "source_name")
		metrics["cell_id"] = getStringFromRecord(record, "cell_id")
		metrics["prb_utilization_dl"] = getFloatFromRecord(record, "prb_utilization_dl")
		metrics["throughput_dl_mbps"] = getFloatFromRecord(record, "throughput_dl_mbps")
		metrics["rsrp"] = getFloatFromRecord(record, "rsrp")
		metrics["rsrq"] = getFloatFromRecord(record, "rsrq")
		metrics["active_users"] = getFloatFromRecord(record, "active_users")
		break // Only need the first record
	}

	return metrics, nil
}

// Helper methods

func (api *AnalyticsAPI) calculateNetworkHealthScore(ctx context.Context, period string) (float64, error) {
	// Simplified health score calculation
	// In practice, this would involve complex analysis of multiple KPIs
	
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> filter(fn: (r) => r._field == "prb_utilization_dl" or r._field == "call_drop_rate" or r._field == "rsrp")
		|> group(columns: ["_field"])
		|> mean()
	`, period)

	result, err := api.queryAPI.Query(ctx, query)
	if err != nil {
		return 0, err
	}

	var prbUtil, callDrop, rsrp float64
	var hasData bool

	for result.Next() {
		record := result.Record()
		hasData = true
		
		switch record.Field() {
		case "prb_utilization_dl":
			prbUtil = getFloatFromRecord(record)
		case "call_drop_rate":
			callDrop = getFloatFromRecord(record)
		case "rsrp":
			rsrp = getFloatFromRecord(record)
		}
	}

	if !hasData {
		return 50.0, nil // Default score if no data
	}

	// Simple health score calculation (0-100)
	// Good PRB utilization: 50-80%, Good RSRP: > -90dBm, Low call drop: < 2%
	score := 100.0
	
	if prbUtil > 90 {
		score -= 20 // High utilization penalty
	} else if prbUtil < 20 {
		score -= 10 // Low utilization penalty
	}
	
	if rsrp < -100 {
		score -= 25 // Poor signal quality
	} else if rsrp < -90 {
		score -= 10 // Fair signal quality
	}
	
	if callDrop > 5 {
		score -= 30 // High call drop rate
	} else if callDrop > 2 {
		score -= 15 // Moderate call drop rate
	}

	if score < 0 {
		score = 0
	}

	return score, nil
}

func (api *AnalyticsAPI) getHealthStatus(score float64) string {
	if score >= 90 {
		return "excellent"
	} else if score >= 75 {
		return "good"
	} else if score >= 50 {
		return "fair"
	}
	return "poor"
}

func (api *AnalyticsAPI) getTopPerformingSites(ctx context.Context, period string, limit int) ([]string, error) {
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> filter(fn: (r) => r._field == "throughput_dl_mbps")
		|> group(columns: ["source_name"])
		|> mean()
		|> sort(columns: ["_value"], desc: true)
		|> limit(n: %d)
	`, period, limit)

	return api.executeSourceNameQuery(ctx, query)
}

func (api *AnalyticsAPI) getUnderperformingSites(ctx context.Context, period string, limit int) ([]string, error) {
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> filter(fn: (r) => r._field == "throughput_dl_mbps")
		|> group(columns: ["source_name"])
		|> mean()
		|> sort(columns: ["_value"])
		|> limit(n: %d)
	`, period, limit)

	return api.executeSourceNameQuery(ctx, query)
}

func (api *AnalyticsAPI) getHighLoadSites(ctx context.Context, period string) ([]string, error) {
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> filter(fn: (r) => r._field == "prb_utilization_dl")
		|> group(columns: ["source_name"])
		|> mean()
		|> filter(fn: (r) => r._value > 80.0)
	`, period)

	return api.executeSourceNameQuery(ctx, query)
}

func (api *AnalyticsAPI) executeSourceNameQuery(ctx context.Context, query string) ([]string, error) {
	result, err := api.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var sites []string
	for result.Next() {
		record := result.Record()
		if sourceName := getStringFromRecord(record, "source_name"); sourceName != "" {
			sites = append(sites, sourceName)
		}
	}

	return sites, nil
}

func (api *AnalyticsAPI) calculateEnergyEfficiencyScore(ctx context.Context, period string) (float64, error) {
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> filter(fn: (r) => r._field == "energy_efficiency")
		|> mean()
	`, period)

	result, err := api.queryAPI.Query(ctx, query)
	if err != nil {
		return 0, err
	}

	for result.Next() {
		record := result.Record()
		return getFloatFromRecord(record), nil
	}

	return 0, nil
}

func (api *AnalyticsAPI) calculateSpectrumEfficiency(ctx context.Context, period string) (float64, error) {
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> filter(fn: (r) => r._field == "spectral_efficiency")
		|> mean()
	`, period)

	result, err := api.queryAPI.Query(ctx, query)
	if err != nil {
		return 0, err
	}

	for result.Next() {
		record := result.Record()
		return getFloatFromRecord(record), nil
	}

	return 0, nil
}

func (api *AnalyticsAPI) predictUpcomingIssues(ctx context.Context, period string) []string {
	// Simplified prediction based on trends
	// In practice, this would use ML models and historical patterns
	return []string{
		"Capacity congestion expected in cell_001 within 2 hours",
		"Signal quality degradation trend detected in cell_003",
		"Energy efficiency declining in sector_north",
	}
}

func (api *AnalyticsAPI) identifyOptimizationOpportunities(insights *NetworkInsights) []string {
	opportunities := make([]string, 0)

	if insights.EnergyEfficiencyScore < 70 {
		opportunities = append(opportunities, "Implement power saving algorithms")
	}

	if len(insights.HighLoadSites) > 0 {
		opportunities = append(opportunities, "Deploy load balancing for high-traffic sites")
	}

	if insights.HealthScore < 80 {
		opportunities = append(opportunities, "Optimize antenna configurations")
	}

	return opportunities
}

func (api *AnalyticsAPI) Close() {
	if api.influxClient != nil {
		api.influxClient.Close()
	}
	if api.kafkaWriter != nil {
		api.kafkaWriter.Close()
	}
}

// Utility functions

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func calculateMeanInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum / len(values)
}

func calculateMaxInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

// Helper functions to safely extract values from InfluxDB records
func getStringFromRecord(record *query.FluxRecord, key string) string {
	if val := record.ValueByKey(key); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloatFromRecord(record *query.FluxRecord, keys ...string) float64 {
	var key string
	if len(keys) > 0 {
		key = keys[0]
	} else {
		// If no key specified, use the record value directly
		if val := record.Value(); val != nil {
			switch v := val.(type) {
			case float64:
				return v
			case int64:
				return float64(v)
			case int:
				return float64(v)
			}
		}
		return 0.0
	}

	if val := record.ValueByKey(key); val != nil {
		switch v := val.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return 0.0
}

func getBoolFromRecord(record *query.FluxRecord) bool {
	if val := record.Value(); val != nil {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getEndpointName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 && parts[1] == "api" && parts[2] == "v1" {
		if len(parts) >= 4 {
			return parts[3]
		}
	}
	return "unknown"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	log.Println("Starting O-RAN Analytics API...")

	api := NewAnalyticsAPI()
	defer api.Close()

	// Setup routes
	router := mux.NewRouter()

	// Apply middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			api.middleware(next.ServeHTTP)(w, r)
		})
	})

	// API routes
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/kpi-summary", api.getKPISummary).Methods("GET", "OPTIONS")
	v1.HandleFunc("/predictions", api.getPredictions).Methods("GET", "OPTIONS")
	v1.HandleFunc("/insights", api.getNetworkInsights).Methods("GET", "OPTIONS")
	v1.HandleFunc("/timeseries", api.getTimeSeriesData).Methods("GET", "OPTIONS")
	v1.HandleFunc("/realtime", api.getRealTimeMetrics).Methods("GET", "OPTIONS")

	// Health and metrics endpoints
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// API documentation endpoint
	router.HandleFunc("/api/v1/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		docs := map[string]interface{}{
			"service": "O-RAN Analytics API",
			"version": "1.0.0",
			"endpoints": map[string]string{
				"GET /api/v1/kpi-summary":  "Get KPI summary with optional filters",
				"GET /api/v1/predictions":  "Get ML predictions and forecasts",
				"GET /api/v1/insights":     "Get comprehensive network insights",
				"GET /api/v1/timeseries":   "Get time series data",
				"GET /api/v1/realtime":     "Get current real-time metrics",
			},
			"parameters": map[string]string{
				"time_range":  "Time range for queries (e.g., 1h, 24h, 7d)",
				"source_name": "Filter by specific source/cell",
				"cell_id":     "Filter by specific cell ID",
				"measurement": "InfluxDB measurement name",
				"field":       "Field name within measurement",
			},
		}
		json.NewEncoder(w).Encode(docs)
	}).Methods("GET")

	server := &http.Server{
		Addr:         ":8088",
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Analytics API server starting on :8088")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Analytics API exited")
}