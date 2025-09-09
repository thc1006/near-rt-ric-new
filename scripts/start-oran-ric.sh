#!/bin/bash
# Quick start script for O-RAN Near-RT RIC

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

echo "========================================"
echo "🚀 O-RAN Near-RT RIC Quick Start"
echo "========================================"
echo ""

# Check if Docker is running
log "Checking Docker status..."
if ! docker info >/dev/null 2>&1; then
    error "Docker is not running!"
    echo ""
    echo "Please start Docker and try again:"
    echo "  - On Linux: sudo systemctl start docker"
    echo "  - On macOS/Windows: Start Docker Desktop"
    exit 1
fi
success "Docker is running"

# Check if Docker Compose is available
if docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
else
    error "Docker Compose not found!"
    echo "Please install Docker Compose and try again."
    exit 1
fi
success "Docker Compose is available"

# Check available disk space
log "Checking disk space..."
available_space=$(df . | tail -1 | awk '{print $4}')
if [ "$available_space" -lt 5000000 ]; then  # 5GB in KB
    warn "Low disk space detected. Ensure you have at least 5GB free."
else
    success "Sufficient disk space available"
fi

# Check if project files exist
if [ ! -f "docker-compose.production.yml" ]; then
    error "Production compose file not found!"
    echo "Please run this script from the project root directory."
    exit 1
fi
success "Project files found"

echo ""
log "Starting O-RAN Near-RT RIC deployment..."

# Run the production deployment
if [ -f "scripts/deploy-production.sh" ]; then
    chmod +x scripts/deploy-production.sh
    ./scripts/deploy-production.sh
else
    error "Deployment script not found!"
    exit 1
fi

echo ""
echo "========================================"
echo "🎉 Deployment Complete!"
echo "========================================"
echo ""
echo "Access your O-RAN Dashboard at:"
echo "  📊 Main Dashboard: http://localhost:3000"
echo "  📈 Grafana:       http://localhost:3001"
echo "  🔍 Prometheus:    http://localhost:9090"
echo "  🕵️  Jaeger:        http://localhost:16686"
echo ""
echo "Useful commands:"
echo "  ./scripts/health-check.sh     - Check system health"
echo "  docker-compose logs -f        - View logs"
echo "  docker-compose ps             - View container status"
echo ""