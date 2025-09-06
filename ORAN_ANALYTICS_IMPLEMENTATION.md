# O-RAN L Release Analytics Implementation

## Overview

This document describes the comprehensive implementation of telemetry data processing, real-time analytics, and ML-based predictions for O-RAN L Release network optimization. The implementation provides a complete analytics pipeline from VES event collection to intelligent network optimization recommendations.

## Architecture

### System Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   O-RAN DU/CU   │    │  Telemetry       │    │     Kafka       │
│   Components    │───▶│  Collector       │───▶│   (Streaming)   │
│                 │    │  (VES Events)    │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                         │
                                                         ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   InfluxDB      │◀───│  KPI Calculator  │◀───│  Stream         │
│ (Time Series)   │    │  (Real-time)     │    │  Processing     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  ML Predictor   │    │  Analytics API   │    │   Grafana       │
│ (Optimization)  │    │  (REST/Query)    │    │ (Dashboards)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Data Flow

1. **VES Event Collection**: O-RAN components send VES events to Telemetry Collector
2. **Stream Processing**: Events are published to Kafka for real-time processing
3. **Time Series Storage**: Raw metrics stored in InfluxDB for historical analysis
4. **KPI Calculation**: Real-time calculation of O-RAN specific KPIs
5. **ML Predictions**: Machine learning models generate network optimization predictions
6. **Analytics API**: REST API provides unified access to all analytics data
7. **Visualization**: Grafana dashboards display real-time network health and predictions

## Services

### 1. Telemetry Collector (`telemetry-collector`)

**Purpose**: Collects VES events from O-RAN components and processes them for analytics

**Key Features**:
- VES 4.0/7.2 compliant event processing
- Real-time event validation and transformation
- Dual-path storage (Kafka + InfluxDB)
- Prometheus metrics for monitoring
- Configurable event routing by domain

**Endpoints**:
- `POST /api/v1/ves` - VES event ingestion
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

**Configuration**: `configs/telemetry/collector.yaml`

### 2. KPI Calculator (`kpi-calculator`)

**Purpose**: Calculates O-RAN specific KPIs from telemetry data in real-time

**Calculated KPIs**:
- **Resource Utilization**: PRB utilization (DL/UL)
- **Throughput**: DL/UL throughput in Mbps
- **Quality Metrics**: RSRP, RSRQ, CQI, SINR
- **Efficiency**: Energy efficiency, Spectral efficiency
- **Connection Metrics**: Active users, Handover rate, Call drop rate
- **Latency**: E2E latency, RAN latency

**Features**:
- Real-time KPI computation
- Time-window aggregations (1h, 24h, 7d)
- Redis caching for fast access
- Configurable thresholds and alerts

**Configuration**: `configs/kpi/calculator.yaml`

### 3. ML Predictor (`ml-predictor`)

**Purpose**: Provides ML-based predictions for network optimization

**ML Models**:
1. **Load Prediction**: Moving average model for capacity forecasting
2. **Quality Prediction**: Linear regression for signal quality trends
3. **Anomaly Detection**: Isolation Forest for anomaly identification

**Predictions**:
- Load forecasting (15m, 1h, 4h horizons)
- Quality degradation trends
- Anomaly detection and classification
- Optimization recommendations

**Features**:
- Real-time and batch predictions
- Model retraining every 6 hours
- Confidence scoring
- Automated recommendation generation

**Configuration**: `configs/ml/predictor.yaml`

### 4. Analytics API (`analytics-api`)

**Purpose**: Unified REST API for accessing all analytics data

**Endpoints**:
- `GET /api/v1/kpi-summary` - Aggregated KPI data
- `GET /api/v1/predictions` - ML predictions and forecasts  
- `GET /api/v1/insights` - Network health insights
- `GET /api/v1/timeseries` - Historical time series data
- `GET /api/v1/realtime` - Current real-time metrics

**Features**:
- Query optimization for large datasets
- CORS support for web dashboards
- Rate limiting and caching
- Comprehensive network health scoring

**Configuration**: `configs/analytics/api.yaml`

## KPI Definitions

### Resource Management KPIs

| KPI | Description | Formula | Unit |
|-----|-------------|---------|------|
| PRB Utilization DL | Downlink Physical Resource Block utilization | (PRB_Used_DL / PRB_Available_DL) × 100 | % |
| PRB Utilization UL | Uplink Physical Resource Block utilization | (PRB_Used_UL / PRB_Available_UL) × 100 | % |
| Throughput DL | Downlink throughput | (MAC_Volume_DL_Bytes × 8) / Measurement_Interval | Mbps |
| Throughput UL | Uplink throughput | (MAC_Volume_UL_Bytes × 8) / Measurement_Interval | Mbps |

### Quality KPIs

| KPI | Description | Target Range | Unit |
|-----|-------------|--------------|------|
| RSRP | Reference Signal Received Power | -70 to -90 dBm | dBm |
| RSRQ | Reference Signal Received Quality | -10 to -15 dB | dB |
| CQI | Channel Quality Indicator | 10-15 | - |
| SINR | Signal-to-Interference-plus-Noise Ratio | 15-25 dB | dB |

### Efficiency KPIs

| KPI | Description | Formula | Unit |
|-----|-------------|---------|------|
| Energy Efficiency | Data rate per unit power | Throughput_Total / Power_Consumption | Mbps/W |
| Spectral Efficiency | Data rate per unit bandwidth | Throughput_Total / Bandwidth | bps/Hz |

## ML Model Specifications

### Load Prediction Model

**Type**: Moving Average with Trend Analysis  
**Input Features**: PRB utilization, Active users, Historical load  
**Output**: Predicted load, User count, Trend direction  
**Update Frequency**: Every 30 seconds  
**Training Window**: Last 10 measurement periods  

**Thresholds**:
- Trend detection: ±10% change
- High load warning: >80% PRB utilization
- Critical load: >90% PRB utilization

### Quality Prediction Model

**Type**: Linear Regression  
**Input Features**: RSRP, RSRQ, SINR, CQI  
**Output**: Predicted signal quality, Trend analysis  
**Coefficients**: RSRP(0.3), RSRQ(0.25), SINR(0.35), CQI(0.1)  
**Update Frequency**: Every 5 minutes  

### Anomaly Detection Model

**Type**: Isolation Forest  
**Input Features**: PRB utilization, Throughput, Energy efficiency, Latency  
**Parameters**: 
- Contamination rate: 10%
- Anomaly threshold: 0.6
- Number of estimators: 100

**Anomaly Types**:
- High Load: PRB utilization >90%
- Low Throughput: <1 Mbps
- High Latency: >50ms E2E
- Quality Degradation: RSRP <-110dBm

## Network Health Scoring

The system calculates an overall network health score (0-100) based on:

```python
health_score = 100 - penalties

Penalties:
- PRB utilization >90%: -20 points
- PRB utilization <20%: -10 points  
- RSRP <-100dBm: -25 points
- RSRP <-90dBm: -10 points
- Call drop rate >5%: -30 points
- Call drop rate >2%: -15 points
```

**Health Categories**:
- Excellent: 90-100 points
- Good: 75-89 points  
- Fair: 50-74 points
- Poor: <50 points

## Optimization Recommendations

The ML Predictor generates automated recommendations based on predictions and anomalies:

### Load-Based Recommendations
- **High Load Predicted**: Trigger load balancing, Consider capacity expansion
- **Increasing Trend**: Prepare for handover optimization
- **Capacity Shortage**: Scale resources, Optimize scheduling

### Quality-Based Recommendations  
- **Signal Degradation**: Adjust transmission power, Check antenna alignment
- **Interference Issues**: Optimize frequency allocation, Reduce power
- **Coverage Gaps**: Deploy small cells, Adjust antenna tilt

### Anomaly-Based Recommendations
- **High Load Anomaly**: Immediate handover, Emergency load balancing
- **Low Throughput**: Check interference, Verify backhaul capacity
- **High Latency**: Optimize routing, Check core network
- **General Anomaly**: Investigate root cause, Monitor trends

## Deployment

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 8GB RAM minimum
- 20GB disk space

### Quick Start

1. **Deploy Analytics Platform**:
```bash
chmod +x scripts/deploy-analytics.sh
./scripts/deploy-analytics.sh
```

2. **Verify Deployment**:
```bash
./scripts/deploy-analytics.sh verify
```

3. **Send Test Data**:
```bash
curl -X POST http://localhost:8085/api/v1/ves \
  -H 'Content-Type: application/json' \
  -d @examples/test-ves-event.json
```

### Service Endpoints

| Service | Port | Endpoint | Purpose |
|---------|------|----------|---------|
| Telemetry Collector | 8085 | http://localhost:8085/api/v1/ves | VES event ingestion |
| KPI Calculator | 8086 | http://localhost:8086/health | KPI processing |
| ML Predictor | 8087 | http://localhost:8087/api/v1/models | ML predictions |
| Analytics API | 8088 | http://localhost:8088/api/v1/docs | Analytics queries |
| Kafka UI | 8087 | http://localhost:8087 | Message stream monitoring |
| InfluxDB | 8086 | http://localhost:8086 | Time series database |
| Grafana | 3000 | http://localhost:3000 | Visualization dashboards |
| Prometheus | 9092 | http://localhost:9092 | Metrics monitoring |

### Configuration

Each service has its own configuration file in the `configs/` directory:

- `configs/telemetry/collector.yaml` - Telemetry collector settings
- `configs/kpi/calculator.yaml` - KPI calculation parameters  
- `configs/ml/predictor.yaml` - ML model configuration
- `configs/analytics/api.yaml` - API service settings

### Environment Variables

Key environment variables in `.env`:

```bash
# InfluxDB
INFLUXDB_ADMIN_PASSWORD=oran123456
INFLUXDB_TOKEN=oran-super-secret-token-xxx

# Security
JWT_SECRET=your-jwt-secret-key

# Monitoring
GRAFANA_ADMIN_PASSWORD=admin123
```

## API Usage Examples

### Get KPI Summary

```bash
# Get last hour KPI summary for all sources
curl "http://localhost:8088/api/v1/kpi-summary?time_range=1h"

# Get KPI summary for specific source
curl "http://localhost:8088/api/v1/kpi-summary?time_range=24h&source_name=O-RAN-DU-001"
```

### Get ML Predictions

```bash
# Get predictions for all sources
curl "http://localhost:8088/api/v1/predictions?time_range=1h"

# Get predictions for specific source
curl "http://localhost:8088/api/v1/predictions?time_range=1h&source_name=O-RAN-DU-001"
```

### Get Network Insights

```bash
# Get 24-hour network analysis
curl "http://localhost:8088/api/v1/insights?period=24h"
```

### Get Time Series Data

```bash
# Get PRB utilization time series
curl "http://localhost:8088/api/v1/timeseries?measurement=oran_kpis&field=prb_utilization_dl&time_range=1h"
```

## Monitoring and Alerting

### Prometheus Metrics

Each service exposes Prometheus metrics at `/metrics`:

**Telemetry Collector**:
- `oran_telemetry_events_received_total` - Total VES events received
- `oran_telemetry_events_processed_total` - Total events processed
- `oran_telemetry_processing_latency_seconds` - Processing latency

**KPI Calculator**:
- `oran_kpis_calculated_total` - Total KPIs calculated
- `oran_kpi_calculation_duration_seconds` - Calculation time
- `oran_kpi_value` - Current KPI values

**ML Predictor**:
- `oran_ml_predictions_total` - Total predictions made
- `oran_ml_model_accuracy` - Model accuracy scores
- `oran_ml_anomalies_detected_total` - Anomalies detected

**Analytics API**:
- `oran_analytics_api_requests_total` - API requests
- `oran_analytics_api_request_duration_seconds` - Request latency

### Grafana Dashboards

Pre-configured dashboards available at http://localhost:3000:

1. **O-RAN Network Overview** - High-level network health
2. **KPI Monitoring** - Real-time KPI trends  
3. **ML Predictions** - Forecasts and anomalies
4. **Service Health** - Analytics platform monitoring

### Log Aggregation

All services log in JSON format to stdout. Use centralized logging:

```bash
# View real-time logs
docker-compose -f docker-compose.yml -f docker-compose.analytics.yml logs -f

# View specific service logs
docker-compose -f docker-compose.yml -f docker-compose.analytics.yml logs telemetry-collector
```

## Performance Considerations

### Throughput Targets

- **VES Events**: 10,000 events/second
- **KPI Calculations**: 1,000 KPIs/second  
- **ML Predictions**: 100 predictions/second
- **API Queries**: 1,000 requests/second

### Resource Requirements

**Production Deployment**:
- CPU: 8 cores minimum
- RAM: 16GB minimum
- Disk: 100GB for 30 days retention
- Network: 1Gbps minimum

**Scaling Recommendations**:
- Kafka: 3+ brokers for high availability
- InfluxDB: Clustering for >100K writes/second
- Analytics services: Horizontal scaling with load balancer

## Security

### Current Implementation
- Basic CORS protection
- Input validation
- Health checks

### Production Security (TODO)
- JWT authentication
- TLS encryption
- Network segmentation  
- Secrets management
- Rate limiting
- Audit logging

## Troubleshooting

### Common Issues

1. **Kafka Connection Failed**
   - Verify Kafka is running: `docker-compose ps kafka`
   - Check network connectivity: `docker-compose exec telemetry-collector nc -zv kafka 29092`

2. **InfluxDB Write Errors**
   - Check InfluxDB health: `curl http://localhost:8086/health`
   - Verify token: Check `.env` file for `INFLUXDB_TOKEN`

3. **No KPIs Being Calculated**
   - Verify VES events are being received: Check telemetry-collector logs
   - Check Kafka topics: `docker-compose exec kafka kafka-topics --bootstrap-server localhost:9093 --list`

4. **ML Predictions Not Generated**
   - Ensure KPIs are being calculated first
   - Check model training: View ml-predictor logs
   - Verify sufficient historical data

### Health Checks

```bash
# Check all service health
curl http://localhost:8085/health  # Telemetry Collector
curl http://localhost:8086/health  # KPI Calculator  
curl http://localhost:8087/health  # ML Predictor
curl http://localhost:8088/health  # Analytics API
```

### Log Analysis

```bash
# Check error rates
docker-compose logs | grep ERROR

# Monitor processing rates
docker-compose logs telemetry-collector | grep "processed"
docker-compose logs kpi-calculator | grep "calculated"
```

## Future Enhancements

### Phase 2 Improvements
1. **Advanced ML Models**: Deep learning for complex predictions
2. **5G SA Features**: Network slicing analytics, MEC optimization
3. **xApp Integration**: Direct optimization actions
4. **Edge Computing**: Distributed analytics processing

### Integration Roadmap
1. **O1 Interface**: NETCONF/YANG model integration
2. **A1 Policy**: ML-driven policy recommendations
3. **E2 Interface**: Real-time control loop optimization
4. **SMO Integration**: Service Management and Orchestration

This implementation provides a comprehensive analytics foundation for O-RAN L Release, enabling intelligent network optimization through real-time telemetry processing, KPI calculation, and ML-based predictions.