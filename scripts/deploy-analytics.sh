#!/bin/bash

# O-RAN Analytics Platform Deployment Script
set -e

echo "🚀 Deploying O-RAN Analytics Platform..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILES="-f docker-compose.yml -f docker-compose.analytics.yml"
SERVICES_TO_WAIT_FOR="kafka influxdb redis postgres"
ANALYTICS_SERVICES="telemetry-collector kpi-calculator ml-predictor analytics-api"

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅${NC} $1"
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️${NC} $1"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌${NC} $1"
}

# Check if Docker and Docker Compose are available
check_dependencies() {
    log "Checking dependencies..."
    
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
    
    success "Dependencies check passed"
}

# Create necessary directories
create_directories() {
    log "Creating necessary directories..."
    
    mkdir -p configs/{telemetry,kpi,ml,analytics,influxdb}
    mkdir -p data/{kafka,influxdb,prometheus,grafana,ml-models}
    mkdir -p logs
    
    success "Directories created"
}

# Generate secrets and environment variables
setup_environment() {
    log "Setting up environment variables..."
    
    # Create .env file if it doesn't exist
    if [ ! -f .env ]; then
        cat > .env << EOF
# O-RAN Analytics Environment Configuration
VERSION=latest

# InfluxDB Configuration
INFLUXDB_ADMIN_PASSWORD=oran123456
INFLUXDB_TOKEN=oran-super-secret-token-$(openssl rand -hex 16)

# Kafka Configuration
KAFKA_CLUSTER_ID=oran-analytics-cluster

# Security (Enable in production)
JWT_SECRET=$(openssl rand -hex 32)

# Monitoring
GRAFANA_ADMIN_PASSWORD=admin123
PROMETHEUS_RETENTION=30d
EOF
        success "Environment file created"
    else
        warning "Environment file already exists, skipping..."
    fi
}

# Setup Kafka topics
setup_kafka_topics() {
    log "Setting up Kafka topics..."
    
    # Wait for Kafka to be ready
    log "Waiting for Kafka to be ready..."
    timeout=300
    while ! docker-compose $COMPOSE_FILES exec -T kafka kafka-topics --bootstrap-server localhost:9093 --list &> /dev/null; do
        sleep 2
        timeout=$((timeout - 2))
        if [ $timeout -le 0 ]; then
            error "Timeout waiting for Kafka to be ready"
            return 1
        fi
    done
    
    # Create topics
    topics=(
        "ves-measurement:10:3"
        "ves-fault:5:3"
        "ves-notification:5:3"
        "ves-registration:3:3"
        "ves-other:3:3"
        "oran-kpis:10:3"
        "oran-predictions:5:3"
        "analytics-requests:3:3"
    )
    
    for topic_config in "${topics[@]}"; do
        IFS=':' read -r topic partitions replication <<< "$topic_config"
        
        if ! docker-compose $COMPOSE_FILES exec -T kafka kafka-topics --bootstrap-server localhost:9093 --describe --topic "$topic" &> /dev/null; then
            log "Creating topic: $topic"
            docker-compose $COMPOSE_FILES exec -T kafka kafka-topics \
                --bootstrap-server localhost:9093 \
                --create \
                --topic "$topic" \
                --partitions "$partitions" \
                --replication-factor "$replication" \
                --config retention.ms=604800000 \
                --config compression.type=lz4
            success "Topic $topic created"
        else
            warning "Topic $topic already exists"
        fi
    done
}

# Setup InfluxDB buckets
setup_influxdb() {
    log "Setting up InfluxDB buckets..."
    
    # Wait for InfluxDB to be ready
    log "Waiting for InfluxDB to be ready..."
    timeout=300
    while ! docker-compose $COMPOSE_FILES exec -T influxdb influx ping &> /dev/null; do
        sleep 2
        timeout=$((timeout - 2))
        if [ $timeout -le 0 ]; then
            error "Timeout waiting for InfluxDB to be ready"
            return 1
        fi
    done
    
    # Create additional buckets
    buckets=(
        "oran-kpis:30d"
        "oran-predictions:7d"
        "oran-alerts:90d"
    )
    
    for bucket_config in "${buckets[@]}"; do
        IFS=':' read -r bucket retention <<< "$bucket_config"
        
        if ! docker-compose $COMPOSE_FILES exec -T influxdb influx bucket list --org oran | grep -q "$bucket"; then
            log "Creating bucket: $bucket"
            docker-compose $COMPOSE_FILES exec -T influxdb influx bucket create \
                --name "$bucket" \
                --org oran \
                --retention "$retention"
            success "Bucket $bucket created"
        else
            warning "Bucket $bucket already exists"
        fi
    done
}

# Deploy core services first
deploy_core_services() {
    log "Deploying core services..."
    
    # Start infrastructure services
    docker-compose $COMPOSE_FILES up -d postgres redis
    
    # Wait for database to be ready
    log "Waiting for PostgreSQL to be ready..."
    timeout=120
    while ! docker-compose $COMPOSE_FILES exec -T postgres pg_isready -U oran &> /dev/null; do
        sleep 2
        timeout=$((timeout - 2))
        if [ $timeout -le 0 ]; then
            error "Timeout waiting for PostgreSQL"
            return 1
        fi
    done
    
    # Start monitoring services
    docker-compose $COMPOSE_FILES up -d prometheus grafana jaeger
    
    success "Core services deployed"
}

# Deploy analytics infrastructure
deploy_analytics_infrastructure() {
    log "Deploying analytics infrastructure..."
    
    # Start Kafka and InfluxDB
    docker-compose $COMPOSE_FILES up -d kafka influxdb kafka-ui
    
    # Wait for services to be ready
    for service in $SERVICES_TO_WAIT_FOR; do
        log "Waiting for $service to be ready..."
        timeout=180
        while ! docker-compose $COMPOSE_FILES ps $service | grep -q "Up"; do
            sleep 3
            timeout=$((timeout - 3))
            if [ $timeout -le 0 ]; then
                error "Timeout waiting for $service"
                return 1
            fi
        done
        success "$service is ready"
    done
    
    # Setup Kafka topics and InfluxDB buckets
    setup_kafka_topics
    setup_influxdb
    
    success "Analytics infrastructure deployed"
}

# Deploy analytics services
deploy_analytics_services() {
    log "Deploying analytics services..."
    
    # Build and start analytics services
    for service in $ANALYTICS_SERVICES; do
        log "Building and deploying $service..."
        docker-compose $COMPOSE_FILES build $service
        docker-compose $COMPOSE_FILES up -d $service
        
        # Wait for service to be healthy
        log "Waiting for $service to be healthy..."
        timeout=120
        while [ "$(docker-compose $COMPOSE_FILES ps -q $service | xargs docker inspect --format='{{.State.Health.Status}}')" != "healthy" ]; do
            sleep 3
            timeout=$((timeout - 3))
            if [ $timeout -le 0 ]; then
                warning "$service might not be healthy, but continuing..."
                break
            fi
        done
        
        success "$service deployed"
    done
}

# Deploy Flink for stream processing
deploy_flink() {
    log "Deploying Apache Flink..."
    
    docker-compose $COMPOSE_FILES up -d flink-jobmanager flink-taskmanager
    
    # Wait for Flink to be ready
    log "Waiting for Flink JobManager to be ready..."
    timeout=120
    while ! curl -f http://localhost:8081 &> /dev/null; do
        sleep 3
        timeout=$((timeout - 3))
        if [ $timeout -le 0 ]; then
            warning "Flink JobManager might not be ready"
            break
        fi
    done
    
    success "Flink deployed"
}

# Verify deployment
verify_deployment() {
    log "Verifying deployment..."
    
    # Check service health
    echo "Service Status:"
    docker-compose $COMPOSE_FILES ps
    
    # Test API endpoints
    endpoints=(
        "http://localhost:8085/health:Telemetry Collector"
        "http://localhost:8086/health:KPI Calculator"
        "http://localhost:8087/health:ML Predictor"
        "http://localhost:8088/health:Analytics API"
        "http://localhost:8086:InfluxDB"
        "http://localhost:8087:Kafka UI"
        "http://localhost:3000:Grafana"
        "http://localhost:16686:Jaeger"
    )
    
    echo -e "\n${BLUE}Service Endpoints:${NC}"
    for endpoint_config in "${endpoints[@]}"; do
        IFS=':' read -r url name <<< "$endpoint_config"
        if curl -f "$url" &> /dev/null; then
            echo -e "${GREEN}✅ $name: $url${NC}"
        else
            echo -e "${RED}❌ $name: $url (not ready)${NC}"
        fi
    done
    
    echo -e "\n${BLUE}Analytics Platform Deployed Successfully! 🎉${NC}"
    echo -e "\n${YELLOW}Next Steps:${NC}"
    echo "1. Access Grafana at http://localhost:3000 (admin/admin123)"
    echo "2. Access Kafka UI at http://localhost:8087"
    echo "3. Access InfluxDB at http://localhost:8086"
    echo "4. Access Analytics API docs at http://localhost:8088/api/v1/docs"
    echo "5. Send test VES events to http://localhost:8085/api/v1/ves"
    echo ""
    echo -e "${YELLOW}To send a test VES event:${NC}"
    echo "curl -X POST http://localhost:8085/api/v1/ves \\"
    echo "  -H 'Content-Type: application/json' \\"
    echo "  -d @examples/test-ves-event.json"
}

# Cleanup function
cleanup() {
    if [ "$1" == "clean" ]; then
        log "Cleaning up deployment..."
        docker-compose $COMPOSE_FILES down -v
        docker system prune -f
        success "Cleanup complete"
    fi
}

# Main deployment flow
main() {
    case "$1" in
        "clean")
            cleanup clean
            ;;
        "infrastructure")
            check_dependencies
            create_directories
            setup_environment
            deploy_core_services
            deploy_analytics_infrastructure
            ;;
        "services")
            deploy_analytics_services
            deploy_flink
            ;;
        "verify")
            verify_deployment
            ;;
        *)
            # Full deployment
            check_dependencies
            create_directories
            setup_environment
            deploy_core_services
            deploy_analytics_infrastructure
            deploy_analytics_services
            deploy_flink
            verify_deployment
            ;;
    esac
}

# Handle script interruption
trap 'error "Deployment interrupted"; exit 1' INT TERM

# Execute main function
main "$@"