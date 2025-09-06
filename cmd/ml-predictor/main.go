package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
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

// MLPrediction represents a machine learning prediction for network optimization
type MLPrediction struct {
	Timestamp         time.Time `json:"timestamp"`
	SourceName        string    `json:"source_name"`
	CellID            string    `json:"cell_id"`
	PredictionType    string    `json:"prediction_type"`
	
	// Capacity predictions
	PredictedLoad     float64   `json:"predicted_load"`
	PredictedUsers    int       `json:"predicted_users"`
	LoadTrend         string    `json:"load_trend"` // increasing, decreasing, stable
	
	// Quality predictions
	PredictedRSRP     float64   `json:"predicted_rsrp"`
	PredictedRSRQ     float64   `json:"predicted_rsrq"`
	QualityTrend      string    `json:"quality_trend"`
	
	// Anomaly detection
	AnomalyScore      float64   `json:"anomaly_score"`
	IsAnomaly         bool      `json:"is_anomaly"`
	AnomalyType       string    `json:"anomaly_type,omitempty"`
	
	// Optimization recommendations
	Recommendations   []Recommendation `json:"recommendations"`
	
	// Model metrics
	ModelAccuracy     float64   `json:"model_accuracy"`
	ConfidenceLevel   float64   `json:"confidence_level"`
}

type Recommendation struct {
	Type        string  `json:"type"`        // "handover", "power_control", "resource_allocation"
	Priority    string  `json:"priority"`    // "high", "medium", "low"
	Description string  `json:"description"`
	Impact      float64 `json:"impact"`      // Expected improvement percentage
	Action      string  `json:"action"`      // Specific action to take
}

// TimeSeriesData represents historical data for ML training
type TimeSeriesData struct {
	Timestamp time.Time
	Values    map[string]float64
}

// SimpleMLModel implements basic ML algorithms for O-RAN predictions
type SimpleMLModel struct {
	Type        string                 `json:"type"`         // "linear_regression", "isolation_forest", "moving_average"
	Features    []string               `json:"features"`
	Parameters  map[string]interface{} `json:"parameters"`
	TrainedAt   time.Time              `json:"trained_at"`
	Accuracy    float64                `json:"accuracy"`
}

// MLPredictor manages ML-based predictions for O-RAN network optimization
type MLPredictor struct {
	kafkaReader   *kafka.Reader
	kafkaWriter   *kafka.Writer
	influxClient  influxdb2.Client
	queryAPI      api.QueryAPI
	writeAPI      api.WriteAPIBlocking
	models        map[string]*SimpleMLModel
	modelPath     string
	metrics       *MLMetrics
}

// MLMetrics defines Prometheus metrics for the ML predictor
type MLMetrics struct {
	PredictionsMade   prometheus.Counter
	PredictionErrors  prometheus.Counter
	ModelAccuracy     *prometheus.GaugeVec
	AnomaliesDetected prometheus.Counter
	TrainingTime      prometheus.Histogram
}

func NewMLMetrics() *MLMetrics {
	return &MLMetrics{
		PredictionsMade: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_ml_predictions_total",
				Help: "Total number of ML predictions made",
			},
		),
		PredictionErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_ml_prediction_errors_total",
				Help: "Total number of ML prediction errors",
			},
		),
		ModelAccuracy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "oran_ml_model_accuracy",
				Help: "ML model accuracy scores",
			},
			[]string{"model_type", "source_name"},
		),
		AnomaliesDetected: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "oran_ml_anomalies_detected_total",
				Help: "Total number of anomalies detected",
			},
		),
		TrainingTime: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "oran_ml_training_duration_seconds",
				Help:    "Time spent training ML models",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
			},
		),
	}
}

func (m *MLMetrics) Register() {
	prometheus.MustRegister(m.PredictionsMade)
	prometheus.MustRegister(m.PredictionErrors)
	prometheus.MustRegister(m.ModelAccuracy)
	prometheus.MustRegister(m.AnomaliesDetected)
	prometheus.MustRegister(m.TrainingTime)
}

func NewMLPredictor() *MLPredictor {
	// Kafka configuration
	kafkaBrokers := getEnv("KAFKA_BROKERS", "kafka:29092")
	brokersList := strings.Split(kafkaBrokers, ",")

	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokersList,
		Topic:   "oran-kpis",
		GroupID: "ml-predictor",
	})

	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokersList...),
		Topic:        "oran-predictions",
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    50,
	}

	// InfluxDB configuration
	influxURL := getEnv("INFLUXDB_URL", "http://influxdb:8086")
	influxToken := getEnv("INFLUXDB_TOKEN", "oran-super-secret-token")
	influxOrg := getEnv("INFLUXDB_ORG", "oran")

	influxClient := influxdb2.NewClient(influxURL, influxToken)
	queryAPI := influxClient.QueryAPI(influxOrg)
	writeAPI := influxClient.WriteAPIBlocking(influxOrg, "oran-predictions")

	modelPath := getEnv("MODEL_PATH", "/models")

	metrics := NewMLMetrics()
	metrics.Register()

	return &MLPredictor{
		kafkaReader:  kafkaReader,
		kafkaWriter:  kafkaWriter,
		influxClient: influxClient,
		queryAPI:     queryAPI,
		writeAPI:     writeAPI,
		models:       make(map[string]*SimpleMLModel),
		modelPath:    modelPath,
		metrics:      metrics,
	}
}

func (mlp *MLPredictor) Start(ctx context.Context) error {
	log.Println("Starting ML Predictor...")

	// Initialize models
	if err := mlp.initializeModels(); err != nil {
		log.Printf("Error initializing models: %v", err)
	}

	// Start model training worker
	go mlp.modelTrainingWorker(ctx)

	// Start prediction worker
	go mlp.predictionWorker(ctx)

	// Process Kafka messages for real-time predictions
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := mlp.kafkaReader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading Kafka message: %v", err)
				continue
			}

			go mlp.processKPIMessage(ctx, message)
		}
	}
}

func (mlp *MLPredictor) initializeModels() error {
	// Initialize different model types for various prediction tasks

	// Load prediction model
	loadModel := &SimpleMLModel{
		Type:     "moving_average",
		Features: []string{"prb_utilization_dl", "prb_utilization_ul", "active_users"},
		Parameters: map[string]interface{}{
			"window_size": 10,
			"trend_threshold": 0.1,
		},
		TrainedAt: time.Now(),
		Accuracy:  0.85,
	}
	mlp.models["load_prediction"] = loadModel

	// Quality prediction model
	qualityModel := &SimpleMLModel{
		Type:     "linear_regression",
		Features: []string{"rsrp", "rsrq", "sinr", "cqi"},
		Parameters: map[string]interface{}{
			"coefficients": map[string]float64{
				"rsrp": 0.3,
				"rsrq": 0.25,
				"sinr": 0.35,
				"cqi":  0.1,
			},
		},
		TrainedAt: time.Now(),
		Accuracy:  0.78,
	}
	mlp.models["quality_prediction"] = qualityModel

	// Anomaly detection model
	anomalyModel := &SimpleMLModel{
		Type:     "isolation_forest",
		Features: []string{"prb_utilization_dl", "throughput_dl_mbps", "energy_efficiency", "latency_e2e_ms"},
		Parameters: map[string]interface{}{
			"contamination": 0.1,
			"threshold":     0.6,
		},
		TrainedAt: time.Now(),
		Accuracy:  0.92,
	}
	mlp.models["anomaly_detection"] = anomalyModel

	return nil
}

func (mlp *MLPredictor) processKPIMessage(ctx context.Context, message kafka.Message) {
	startTime := time.Now()
	defer mlp.metrics.TrainingTime.Observe(time.Since(startTime).Seconds())

	var kpi map[string]interface{}
	if err := json.Unmarshal(message.Value, &kpi); err != nil {
		log.Printf("Error unmarshaling KPI data: %v", err)
		mlp.metrics.PredictionErrors.Inc()
		return
	}

	prediction, err := mlp.generatePredictions(ctx, kpi)
	if err != nil {
		log.Printf("Error generating predictions: %v", err)
		mlp.metrics.PredictionErrors.Inc()
		return
	}

	// Store predictions
	if err := mlp.storePredictions(ctx, prediction); err != nil {
		log.Printf("Error storing predictions: %v", err)
		mlp.metrics.PredictionErrors.Inc()
		return
	}

	// Publish predictions
	if err := mlp.publishPredictions(ctx, prediction); err != nil {
		log.Printf("Error publishing predictions: %v", err)
		mlp.metrics.PredictionErrors.Inc()
		return
	}

	mlp.metrics.PredictionsMade.Inc()
	if prediction.IsAnomaly {
		mlp.metrics.AnomaliesDetected.Inc()
	}
}

func (mlp *MLPredictor) generatePredictions(ctx context.Context, kpi map[string]interface{}) (*MLPrediction, error) {
	sourceName := getString(kpi, "source_name")
	cellID := getString(kpi, "cell_id")

	prediction := &MLPrediction{
		Timestamp:      time.Now(),
		SourceName:     sourceName,
		CellID:         cellID,
		PredictionType: "comprehensive",
		Recommendations: make([]Recommendation, 0),
	}

	// Generate load predictions
	if err := mlp.predictLoad(ctx, kpi, prediction); err != nil {
		log.Printf("Error predicting load: %v", err)
	}

	// Generate quality predictions
	if err := mlp.predictQuality(ctx, kpi, prediction); err != nil {
		log.Printf("Error predicting quality: %v", err)
	}

	// Detect anomalies
	if err := mlp.detectAnomalies(ctx, kpi, prediction); err != nil {
		log.Printf("Error detecting anomalies: %v", err)
	}

	// Generate recommendations
	mlp.generateRecommendations(prediction)

	return prediction, nil
}

func (mlp *MLPredictor) predictLoad(ctx context.Context, kpi map[string]interface{}, prediction *MLPrediction) error {
	model := mlp.models["load_prediction"]
	if model == nil {
		return fmt.Errorf("load prediction model not found")
	}

	// Get historical data for trend analysis
	historicalData, err := mlp.getHistoricalData(ctx, prediction.SourceName, "1h", model.Features)
	if err != nil {
		return err
	}

	if len(historicalData) < 3 {
		// Not enough data for prediction
		prediction.PredictedLoad = getFloat(kpi["prb_utilization_dl"])
		prediction.LoadTrend = "stable"
		return nil
	}

	// Simple moving average prediction
	windowSize := int(model.Parameters["window_size"].(float64))
	if windowSize > len(historicalData) {
		windowSize = len(historicalData)
	}

	recent := historicalData[len(historicalData)-windowSize:]
	
	// Calculate trend
	var loadSum float64
	var userSum float64
	for _, data := range recent {
		loadSum += data.Values["prb_utilization_dl"]
		userSum += data.Values["active_users"]
	}

	prediction.PredictedLoad = loadSum / float64(len(recent))
	prediction.PredictedUsers = int(userSum / float64(len(recent)))

	// Determine trend
	if len(recent) >= 2 {
		oldAvg := (recent[0].Values["prb_utilization_dl"] + recent[1].Values["prb_utilization_dl"]) / 2
		newAvg := (recent[len(recent)-2].Values["prb_utilization_dl"] + recent[len(recent)-1].Values["prb_utilization_dl"]) / 2
		
		trendThreshold := model.Parameters["trend_threshold"].(float64)
		if newAvg > oldAvg+trendThreshold {
			prediction.LoadTrend = "increasing"
		} else if newAvg < oldAvg-trendThreshold {
			prediction.LoadTrend = "decreasing"
		} else {
			prediction.LoadTrend = "stable"
		}
	}

	prediction.ModelAccuracy = model.Accuracy
	prediction.ConfidenceLevel = calculateConfidence(len(historicalData), model.Accuracy)

	return nil
}

func (mlp *MLPredictor) predictQuality(ctx context.Context, kpi map[string]interface{}, prediction *MLPrediction) error {
	model := mlp.models["quality_prediction"]
	if model == nil {
		return fmt.Errorf("quality prediction model not found")
	}

	// Simple linear regression prediction
	var predictedRSRP, predictedRSRQ float64
	currentRSRP := getFloat(kpi["rsrp"])
	currentRSRQ := getFloat(kpi["rsrq"])
	
	// Apply simple trend analysis
	if currentRSRP != 0 && currentRSRQ != 0 {
		// Get historical trend
		historicalData, err := mlp.getHistoricalData(ctx, prediction.SourceName, "30m", []string{"rsrp", "rsrq"})
		if err == nil && len(historicalData) >= 2 {
			oldRSRP := historicalData[0].Values["rsrp"]
			oldRSRQ := historicalData[0].Values["rsrq"]
			
			trend := (currentRSRP - oldRSRP) / float64(len(historicalData))
			predictedRSRP = currentRSRP + trend
			
			trend = (currentRSRQ - oldRSRQ) / float64(len(historicalData))
			predictedRSRQ = currentRSRQ + trend
		} else {
			predictedRSRP = currentRSRP
			predictedRSRQ = currentRSRQ
		}
	}

	prediction.PredictedRSRP = predictedRSRP
	prediction.PredictedRSRQ = predictedRSRQ

	// Determine quality trend
	if predictedRSRP > currentRSRP+1 {
		prediction.QualityTrend = "improving"
	} else if predictedRSRP < currentRSRP-1 {
		prediction.QualityTrend = "degrading"
	} else {
		prediction.QualityTrend = "stable"
	}

	return nil
}

func (mlp *MLPredictor) detectAnomalies(ctx context.Context, kpi map[string]interface{}, prediction *MLPrediction) error {
	model := mlp.models["anomaly_detection"]
	if model == nil {
		return fmt.Errorf("anomaly detection model not found")
	}

	// Simple anomaly detection based on statistical thresholds
	features := []float64{
		getFloat(kpi["prb_utilization_dl"]),
		getFloat(kpi["throughput_dl_mbps"]),
		getFloat(kpi["energy_efficiency"]),
		getFloat(kpi["latency_e2e_ms"]),
	}

	// Calculate Z-score based anomaly detection
	historicalData, err := mlp.getHistoricalData(ctx, prediction.SourceName, "2h", model.Features)
	if err != nil || len(historicalData) < 10 {
		// Not enough data for anomaly detection
		prediction.AnomalyScore = 0.0
		prediction.IsAnomaly = false
		return nil
	}

	// Calculate mean and standard deviation for each feature
	means := make([]float64, len(model.Features))
	stds := make([]float64, len(model.Features))

	for i, feature := range model.Features {
		var sum, sumSq float64
		for _, data := range historicalData {
			value := data.Values[feature]
			sum += value
			sumSq += value * value
		}
		
		means[i] = sum / float64(len(historicalData))
		variance := (sumSq / float64(len(historicalData))) - (means[i] * means[i])
		stds[i] = math.Sqrt(variance)
	}

	// Calculate anomaly score
	var anomalyScore float64
	for i, feature := range features {
		if stds[i] > 0 {
			zScore := math.Abs((feature - means[i]) / stds[i])
			anomalyScore += zScore
		}
	}
	
	anomalyScore = anomalyScore / float64(len(features))
	threshold := model.Parameters["threshold"].(float64)

	prediction.AnomalyScore = anomalyScore
	prediction.IsAnomaly = anomalyScore > threshold

	// Determine anomaly type
	if prediction.IsAnomaly {
		if features[0] > 90 { // High PRB utilization
			prediction.AnomalyType = "high_load"
		} else if features[1] < 1 { // Low throughput
			prediction.AnomalyType = "low_throughput"
		} else if features[3] > 50 { // High latency
			prediction.AnomalyType = "high_latency"
		} else {
			prediction.AnomalyType = "general"
		}
	}

	return nil
}

func (mlp *MLPredictor) generateRecommendations(prediction *MLPrediction) {
	// Generate recommendations based on predictions and anomalies
	
	// Load-based recommendations
	if prediction.LoadTrend == "increasing" && prediction.PredictedLoad > 80 {
		recommendation := Recommendation{
			Type:        "resource_allocation",
			Priority:    "high",
			Description: "High load predicted. Consider load balancing or capacity expansion.",
			Impact:      15.0,
			Action:      "trigger_load_balancing",
		}
		prediction.Recommendations = append(prediction.Recommendations, recommendation)
	}

	// Quality-based recommendations
	if prediction.QualityTrend == "degrading" {
		recommendation := Recommendation{
			Type:        "power_control",
			Priority:    "medium",
			Description: "Signal quality degrading. Adjust transmission power.",
			Impact:      10.0,
			Action:      "adjust_power_control",
		}
		prediction.Recommendations = append(prediction.Recommendations, recommendation)
	}

	// Anomaly-based recommendations
	if prediction.IsAnomaly {
		var priority, action string
		impact := 20.0

		switch prediction.AnomalyType {
		case "high_load":
			priority = "high"
			action = "initiate_handover"
		case "low_throughput":
			priority = "medium"
			action = "check_interference"
		case "high_latency":
			priority = "high"
			action = "optimize_routing"
		default:
			priority = "low"
			action = "investigate_issue"
		}

		recommendation := Recommendation{
			Type:        "anomaly_response",
			Priority:    priority,
			Description: fmt.Sprintf("Anomaly detected: %s. Immediate action required.", prediction.AnomalyType),
			Impact:      impact,
			Action:      action,
		}
		prediction.Recommendations = append(prediction.Recommendations, recommendation)
	}

	// Sort recommendations by priority
	sort.Slice(prediction.Recommendations, func(i, j int) bool {
		priorityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
		return priorityOrder[prediction.Recommendations[i].Priority] > priorityOrder[prediction.Recommendations[j].Priority]
	})
}

func (mlp *MLPredictor) getHistoricalData(ctx context.Context, sourceName, timeRange string, features []string) ([]TimeSeriesData, error) {
	featureFilter := strings.Join(features, `" or r._field == "`)
	
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -%s)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> filter(fn: (r) => r.source_name == "%s")
		|> filter(fn: (r) => r._field == "%s")
		|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
	`, timeRange, sourceName, featureFilter)

	result, err := mlp.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var data []TimeSeriesData
	for result.Next() {
		record := result.Record()
		tsData := TimeSeriesData{
			Timestamp: record.Time(),
			Values:    make(map[string]float64),
		}

		for _, feature := range features {
			if val := record.ValueByKey(feature); val != nil {
				tsData.Values[feature] = getFloat(val)
			}
		}
		
		data = append(data, tsData)
	}

	return data, nil
}

func (mlp *MLPredictor) storePredictions(ctx context.Context, prediction *MLPrediction) error {
	point := influxdb2.NewPoint("oran_predictions",
		map[string]string{
			"source_name":     prediction.SourceName,
			"cell_id":         prediction.CellID,
			"prediction_type": prediction.PredictionType,
		},
		map[string]interface{}{
			"predicted_load":    prediction.PredictedLoad,
			"predicted_users":   prediction.PredictedUsers,
			"load_trend":        prediction.LoadTrend,
			"predicted_rsrp":    prediction.PredictedRSRP,
			"predicted_rsrq":    prediction.PredictedRSRQ,
			"quality_trend":     prediction.QualityTrend,
			"anomaly_score":     prediction.AnomalyScore,
			"is_anomaly":        prediction.IsAnomaly,
			"anomaly_type":      prediction.AnomalyType,
			"model_accuracy":    prediction.ModelAccuracy,
			"confidence_level":  prediction.ConfidenceLevel,
		},
		prediction.Timestamp)

	return mlp.writeAPI.WritePoint(ctx, point)
}

func (mlp *MLPredictor) publishPredictions(ctx context.Context, prediction *MLPrediction) error {
	predictionJSON, err := json.Marshal(prediction)
	if err != nil {
		return err
	}

	message := kafka.Message{
		Key:   []byte(prediction.SourceName),
		Value: predictionJSON,
		Time:  time.Now(),
	}

	return mlp.kafkaWriter.WriteMessages(ctx, message)
}

func (mlp *MLPredictor) modelTrainingWorker(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour) // Retrain models every 6 hours
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Println("Starting periodic model training...")
			if err := mlp.trainModels(ctx); err != nil {
				log.Printf("Error training models: %v", err)
			}
		}
	}
}

func (mlp *MLPredictor) predictionWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Generate periodic predictions
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Println("Generating periodic predictions...")
			if err := mlp.generatePeriodicPredictions(ctx); err != nil {
				log.Printf("Error generating periodic predictions: %v", err)
			}
		}
	}
}

func (mlp *MLPredictor) trainModels(ctx context.Context) error {
	startTime := time.Now()
	defer mlp.metrics.TrainingTime.Observe(time.Since(startTime).Seconds())

	// For simplicity, we'll just update model accuracy metrics
	// In a real implementation, this would involve actual ML training
	
	for modelType, model := range mlp.models {
		// Simulate training by calculating accuracy on recent data
		accuracy := mlp.evaluateModel(ctx, model)
		model.Accuracy = accuracy
		model.TrainedAt = time.Now()

		mlp.metrics.ModelAccuracy.WithLabelValues(modelType, "global").Set(accuracy)
	}

	return nil
}

func (mlp *MLPredictor) evaluateModel(ctx context.Context, model *SimpleMLModel) float64 {
	// Simulate model evaluation
	// In a real implementation, this would involve cross-validation
	baseAccuracy := model.Accuracy
	
	// Add some random variation to simulate real performance changes
	variation := (math.Sin(float64(time.Now().Unix())/3600) * 0.05) // +/- 5% variation
	return math.Max(0.5, math.Min(1.0, baseAccuracy+variation))
}

func (mlp *MLPredictor) generatePeriodicPredictions(ctx context.Context) error {
	// Get list of active sources
	query := `
		from(bucket: "oran-kpis")
		|> range(start: -1h)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> group(columns: ["source_name"])
		|> last()
	`

	result, err := mlp.queryAPI.Query(ctx, query)
	if err != nil {
		return err
	}

	// Generate predictions for each active source
	for result.Next() {
		record := result.Record()
		sourceName := record.ValueByKey("source_name").(string)
		
		// Get latest KPI data for this source
		latestKPI, err := mlp.getLatestKPI(ctx, sourceName)
		if err != nil {
			continue
		}

		// Generate prediction
		prediction, err := mlp.generatePredictions(ctx, latestKPI)
		if err != nil {
			continue
		}

		// Store and publish
		mlp.storePredictions(ctx, prediction)
		mlp.publishPredictions(ctx, prediction)
	}

	return nil
}

func (mlp *MLPredictor) getLatestKPI(ctx context.Context, sourceName string) (map[string]interface{}, error) {
	query := fmt.Sprintf(`
		from(bucket: "oran-kpis")
		|> range(start: -10m)
		|> filter(fn: (r) => r._measurement == "oran_kpis")
		|> filter(fn: (r) => r.source_name == "%s")
		|> last()
		|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
	`, sourceName)

	result, err := mlp.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	kpi := make(map[string]interface{})
	if result.Next() {
		record := result.Record()
		kpi["source_name"] = sourceName
		kpi["cell_id"] = record.ValueByKey("cell_id")
		kpi["prb_utilization_dl"] = record.ValueByKey("prb_utilization_dl")
		kpi["throughput_dl_mbps"] = record.ValueByKey("throughput_dl_mbps")
		kpi["energy_efficiency"] = record.ValueByKey("energy_efficiency")
		kpi["latency_e2e_ms"] = record.ValueByKey("latency_e2e_ms")
		kpi["rsrp"] = record.ValueByKey("rsrp")
		kpi["rsrq"] = record.ValueByKey("rsrq")
		kpi["active_users"] = record.ValueByKey("active_users")
	}

	return kpi, nil
}

func (mlp *MLPredictor) Close() {
	if mlp.kafkaReader != nil {
		mlp.kafkaReader.Close()
	}
	if mlp.kafkaWriter != nil {
		mlp.kafkaWriter.Close()
	}
	if mlp.influxClient != nil {
		mlp.influxClient.Close()
	}
}

func calculateConfidence(dataPoints int, modelAccuracy float64) float64 {
	// Simple confidence calculation based on data points and model accuracy
	dataConfidence := math.Min(1.0, float64(dataPoints)/100.0) // More data = more confidence
	return (modelAccuracy + dataConfidence) / 2.0
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
		// Note: In real implementation, use strconv.ParseFloat
		return 0.0
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
	log.Println("Starting O-RAN ML Predictor...")

	predictor := NewMLPredictor()
	defer predictor.Close()

	// Setup HTTP server
	router := mux.NewRouter()
	
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")
	
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	
	// API endpoints
	router.HandleFunc("/api/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(predictor.models)
	}).Methods("GET")

	server := &http.Server{
		Addr:    ":8087",
		Handler: router,
	}

	// Start HTTP server
	go func() {
		log.Printf("ML Predictor HTTP server starting on :8087")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start ML predictor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := predictor.Start(ctx); err != nil {
			log.Fatalf("ML Predictor failed: %v", err)
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

	log.Println("ML Predictor exited")
}