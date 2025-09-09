#!/bin/bash
# Production deployment script for O-RAN Near-RT RIC

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.production.yml"
PROJECT_NAME="oran-near-rt-ric"
VERSION="${VERSION:-v1.0.0}"
ENV_FILE="${ENV_FILE:-.env.production}"

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    
    # Check if Docker Compose is available
    if ! command -v docker-compose >/dev/null 2>&1 && ! docker compose version >/dev/null 2>&1; then
        error "Docker Compose is not available. Please install Docker Compose."
        exit 1
    fi
    
    # Set Docker Compose command
    if docker compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker compose"
    else
        DOCKER_COMPOSE="docker-compose"
    fi
    
    # Check if compose file exists
    if [[ ! -f "$COMPOSE_FILE" ]]; then
        error "Compose file $COMPOSE_FILE not found!"
        exit 1
    fi
    
    success "Prerequisites check passed"
}

# Create environment file if it doesn't exist
setup_environment() {
    log "Setting up environment..."
    
    if [[ ! -f "$ENV_FILE" ]]; then
        log "Creating environment file: $ENV_FILE"
        cat > "$ENV_FILE" << EOF
# O-RAN Near-RT RIC Production Environment

# Application Version
VERSION=$VERSION

# Database Configuration
DB_PASSWORD=$(openssl rand -base64 32)
POSTGRES_PASSWORD=\$DB_PASSWORD

# Redis Configuration
REDIS_PASSWORD=$(openssl rand -base64 32)

# Security
JWT_SECRET=$(openssl rand -base64 64)
GRAFANA_SECRET=$(openssl rand -base64 32)
GRAFANA_PASSWORD=admin123

# Monitoring
JAEGER_AGENT_HOST=jaeger
JAEGER_AGENT_PORT=6831

# Environment
NODE_ENV=production
GO_ENV=production
LOG_LEVEL=info
EOF
        success "Environment file created: $ENV_FILE"
        warn "Please review and update $ENV_FILE with your specific configuration"
    else
        log "Using existing environment file: $ENV_FILE"
    fi
}

# Generate SSL certificates
setup_certificates() {
    log "Setting up SSL certificates..."
    
    CERT_DIR="./certs"
    mkdir -p "$CERT_DIR"
    
    if [[ ! -f "$CERT_DIR/ca.crt" ]]; then
        log "Generating CA certificate..."
        openssl genrsa -out "$CERT_DIR/ca.key" 4096
        openssl req -new -x509 -key "$CERT_DIR/ca.key" -sha256 -subj "/C=US/ST=CA/O=ORAN/CN=ORAN-CA" -days 3650 -out "$CERT_DIR/ca.crt"
    fi
    
    if [[ ! -f "$CERT_DIR/server.crt" ]]; then
        log "Generating server certificate..."
        openssl genrsa -out "$CERT_DIR/server.key" 4096
        openssl req -subj "/C=US/ST=CA/O=ORAN/CN=localhost" -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr"
        openssl x509 -req -in "$CERT_DIR/server.csr" -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial -out "$CERT_DIR/server.crt" -days 365 -sha256
    fi
    
    if [[ ! -f "$CERT_DIR/ric.crt" ]]; then
        log "Generating RIC certificate..."
        openssl genrsa -out "$CERT_DIR/ric.key" 4096
        openssl req -subj "/C=US/ST=CA/O=ORAN/CN=ric-core" -new -key "$CERT_DIR/ric.key" -out "$CERT_DIR/ric.csr"
        openssl x509 -req -in "$CERT_DIR/ric.csr" -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial -out "$CERT_DIR/ric.crt" -days 365 -sha256
    fi
    
    success "SSL certificates ready"
}

# Build images
build_images() {
    log "Building Docker images..."
    
    export VERSION
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" build --parallel
    
    success "Docker images built successfully"
}

# Start services
start_services() {
    log "Starting services..."
    
    # Start infrastructure services first
    log "Starting infrastructure services..."
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" up -d postgres redis elasticsearch
    
    # Wait for databases to be ready
    log "Waiting for databases to be ready..."
    sleep 30
    
    # Start monitoring services
    log "Starting monitoring services..."
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" up -d prometheus grafana jaeger
    
    # Start application services
    log "Starting application services..."
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" up -d dashboard-api ric-core xapp-manager
    
    # Start web services
    log "Starting web services..."
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" up -d web-dashboard nginx
    
    success "All services started"
}

# Wait for services to be healthy
wait_for_services() {
    log "Waiting for services to become healthy..."
    
    local max_attempts=60
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -f http://localhost:80/health >/dev/null 2>&1; then
            success "All services are healthy"
            return 0
        fi
        
        attempt=$((attempt + 1))
        log "Attempt $attempt/$max_attempts - waiting for services..."
        sleep 10
    done
    
    error "Services did not become healthy within expected time"
    return 1
}

# Show deployment status
show_status() {
    log "Deployment Status:"
    echo "==================="
    
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" ps
    
    echo ""
    echo "Service URLs:"
    echo "==================="
    echo "🌐 O-RAN Dashboard:    http://localhost:3000"
    echo "📊 Grafana:           http://localhost:3001"
    echo "🔍 Prometheus:        http://localhost:9090"
    echo "🕵️  Jaeger:            http://localhost:16686"
    echo "🔌 API Endpoint:      http://localhost:8080"
    echo "🏥 Health Check:      http://localhost:80/health"
    echo ""
    echo "Default Credentials:"
    echo "==================="
    echo "Grafana: admin / admin123"
    echo ""
}

# Main deployment function
deploy() {
    log "Starting O-RAN Near-RT RIC production deployment..."
    
    check_prerequisites
    setup_environment
    setup_certificates
    build_images
    start_services
    wait_for_services
    show_status
    
    success "🎉 O-RAN Near-RT RIC deployed successfully!"
    log "Access the dashboard at: http://localhost:3000"
}

# Cleanup function
cleanup() {
    log "Stopping and removing services..."
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" down -v
    success "Cleanup completed"
}

# Update function
update() {
    log "Updating O-RAN Near-RT RIC deployment..."
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" pull
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" up -d
    success "Update completed"
}

# Show logs
logs() {
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" logs -f "${@}"
}

# Show help
show_help() {
    echo "O-RAN Near-RT RIC Production Deployment Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  deploy    Deploy the complete O-RAN Near-RT RIC stack"
    echo "  cleanup   Stop and remove all services"
    echo "  update    Update all services"
    echo "  status    Show deployment status"
    echo "  logs      Show logs (optionally for specific service)"
    echo "  help      Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  VERSION          Application version (default: v1.0.0)"
    echo "  ENV_FILE         Environment file path (default: .env.production)"
    echo ""
    echo "Examples:"
    echo "  $0 deploy                 # Deploy everything"
    echo "  $0 logs dashboard-api     # Show dashboard API logs"
    echo "  VERSION=v1.1.0 $0 deploy  # Deploy specific version"
}

# Main script logic
case "${1:-deploy}" in
    "deploy")
        deploy
        ;;
    "cleanup")
        cleanup
        ;;
    "update")
        update
        ;;
    "status")
        show_status
        ;;
    "logs")
        shift
        logs "$@"
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac