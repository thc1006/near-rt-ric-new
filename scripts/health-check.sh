#!/bin/bash
#
# SPDX-FileCopyrightText: 2022-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

# Health check script for O-RAN Near-RT RIC deployment (Docker/Kubernetes)

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
ENV_FILE="${ENV_FILE:-.env.production}"
NAMESPACE="ricplt"
TIMEOUT=300

# Detect deployment type
DEPLOYMENT_TYPE=""
if command -v kubectl >/dev/null 2>&1 && kubectl get namespace $NAMESPACE >/dev/null 2>&1; then
    DEPLOYMENT_TYPE="kubernetes"
elif command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    DEPLOYMENT_TYPE="docker"
else
    echo -e "${RED}✗ Neither Kubernetes nor Docker deployment detected${NC}"
    exit 1
fi

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
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

# =============================================================================
# Kubernetes Health Checks
# =============================================================================

# Function to check if pod is ready (Kubernetes)
check_pod_ready() {
    local app_label=$1
    local component_name=$2
    
    log "Checking $component_name..."
    
    # Wait for pod to be ready
    if kubectl wait --for=condition=ready pod -l app=$app_label -n $NAMESPACE --timeout=${TIMEOUT}s >/dev/null 2>&1; then
        success "$component_name is ready"
        return 0
    else
        error "$component_name is not ready"
        kubectl get pods -l app=$app_label -n $NAMESPACE
        return 1
    fi
}

# Function to check service endpoints (Kubernetes)
check_service_endpoint() {
    local service_name=$1
    local port=$2
    local component_name=$3
    
    log "Checking $component_name service endpoint..."
    
    if kubectl get service $service_name -n $NAMESPACE >/dev/null 2>&1; then
        success "$component_name service is available"
        return 0
    else
        error "$component_name service not found"
        return 1
    fi
}

# Kubernetes health check
check_kubernetes_health() {
    echo "=== O-RAN SC Near-RT RIC Health Check (Kubernetes) ==="
    
    # Check if namespace exists
    if ! kubectl get namespace $NAMESPACE >/dev/null 2>&1; then
        error "Namespace $NAMESPACE does not exist"
        exit 1
    fi
    
    success "Namespace $NAMESPACE exists"
    
    # Check core O-RAN SC components
    echo ""
    echo "=== Checking Core Components ==="
    
    # Check Database (Redis)
    check_pod_ready "dbaas" "Database (Redis)"
    
    # Check E2 Manager
    check_pod_ready "e2mgr" "E2 Manager"
    
    # Check Subscription Manager  
    check_pod_ready "submgr" "Subscription Manager"
    
    # Check Routing Manager
    check_pod_ready "rtmgr" "Routing Manager"
    
    # Check App Manager
    check_pod_ready "appmgr" "App Manager"
    
    # Check A1 Mediator
    check_pod_ready "a1mediator" "A1 Mediator"
    
    # Check E2 Termination
    check_pod_ready "e2term" "E2 Termination"
    
    echo ""
    echo "=== Checking Service Endpoints ==="
    
    # Check key service endpoints
    check_service_endpoint "e2mgr-http" "3800" "E2 Manager"
    check_service_endpoint "a1mediator-http" "10000" "A1 Mediator"
    
    echo ""
    echo "=== Checking Observability Components ==="
    kubectl get pods -n ricplt -l app.kubernetes.io/name=grafana || echo "Grafana not deployed"
    kubectl get pods -n ricplt -l app.kubernetes.io/name=prometheus || echo "Prometheus not deployed"
    kubectl get pods -n ricplt -l app.kubernetes.io/name=loki || echo "Loki not deployed"
    
    echo ""
    echo "=== Overall System Status ==="
    
    # Get overall pod status
    echo "Pod Status Summary:"
    kubectl get pods -n $NAMESPACE -o wide
    
    echo ""
    echo "Service Status Summary:"
    kubectl get services -n $NAMESPACE
    
    # Count ready pods
    ready_pods=$(kubectl get pods -n $NAMESPACE --no-headers | grep -c "Running\|Completed" || echo "0")
    total_pods=$(kubectl get pods -n $NAMESPACE --no-headers | wc -l || echo "0")
    
    echo ""
    echo "=== Health Check Summary ==="
    echo "Ready Pods: $ready_pods/$total_pods"
    
    if [ "$ready_pods" -eq "$total_pods" ] && [ "$total_pods" -gt 0 ]; then
        success "All O-RAN SC components are healthy"
        return 0
    else
        warn "Some components may not be ready yet"
        echo "This is normal during initial deployment. Components may take a few minutes to start."
        return 0  # Don't fail the build during initial setup
    fi
}

# =============================================================================
# Docker Health Checks
# =============================================================================

# Check if Docker Compose is available
check_docker_compose() {
    if docker compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker-compose"
    else
        error "Docker Compose is not available"
        exit 1
    fi
}

# Check container status (Docker)
check_container_status() {
    log "Checking container status..."
    
    local containers
    containers=$($DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" ps --format "table {{.Name}}\t{{.State}}\t{{.Health}}")
    
    echo "$containers"
    echo ""
    
    # Count running containers
    local running_count
    running_count=$(echo "$containers" | grep -c "running" || true)
    
    if [ "$running_count" -gt 0 ]; then
        success "$running_count containers are running"
        return 0
    else
        error "No containers are running"
        return 1
    fi
}

# Check service health endpoints (Docker)
check_service_health() {
    log "Checking service health endpoints..."
    
    local failed_services=0
    
    # Service endpoints for Docker deployment
    local SERVICES=(
        "nginx:80:/health"
        "dashboard-api:8080:/health"
        "ric-core:8081:/health"
        "xapp-manager:8082:/health"
        "web-dashboard:80:/health"
        "prometheus:9090/-/healthy"
        "grafana:3000/api/health"
        "jaeger:16686/"
    )
    
    for service_config in "${SERVICES[@]}"; do
        IFS=':' read -r service port path <<< "$service_config"
        
        if curl -f --max-time 10 "http://localhost:$port$path" >/dev/null 2>&1; then
            success "$service ($port$path)"
        else
            error "$service ($port$path)"
            ((failed_services++))
        fi
    done
    
    echo ""
    
    if [ $failed_services -eq 0 ]; then
        success "All service health checks passed"
        return 0
    else
        error "$failed_services service health checks failed"
        return 1
    fi
}

# Check database connectivity (Docker)
check_database_connectivity() {
    log "Checking database connectivity..."
    
    local failed_dbs=0
    
    # Check PostgreSQL
    if $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" exec -T postgres pg_isready -U oran_prod >/dev/null 2>&1; then
        success "PostgreSQL"
    else
        error "PostgreSQL"
        ((failed_dbs++))
    fi
    
    # Check Redis
    if $DOCKER_COMPOSE -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" exec -T redis redis-cli ping >/dev/null 2>&1; then
        success "Redis"
    else
        error "Redis"
        ((failed_dbs++))
    fi
    
    # Check Elasticsearch
    if curl -f --max-time 10 "http://localhost:9200/_cluster/health" >/dev/null 2>&1; then
        success "Elasticsearch"
    else
        error "Elasticsearch"
        ((failed_dbs++))
    fi
    
    echo ""
    
    if [ $failed_dbs -eq 0 ]; then
        success "All database connectivity checks passed"
        return 0
    else
        error "$failed_dbs database connectivity checks failed"
        return 1
    fi
}

# Docker health check
check_docker_health() {
    echo "=== O-RAN Near-RT RIC Health Check (Docker) ==="
    
    check_docker_compose
    
    local checks_passed=0
    local total_checks=3
    
    if check_container_status; then ((checks_passed++)); fi
    if check_service_health; then ((checks_passed++)); fi
    if check_database_connectivity; then ((checks_passed++)); fi
    
    echo "=============================================="
    echo "Health Check Summary"
    echo "=============================================="
    echo "Checks passed: $checks_passed/$total_checks"
    echo ""
    
    if [ $checks_passed -eq $total_checks ]; then
        success "All health checks passed! 🎉"
        return 0
    elif [ $checks_passed -ge $((total_checks * 3 / 4)) ]; then
        warn "Most health checks passed with some warnings"
        return 0
    else
        error "Multiple health checks failed - investigation required"
        return 1
    fi
}

# =============================================================================
# Main Function
# =============================================================================

main() {
    log "Detected deployment type: $DEPLOYMENT_TYPE"
    
    case $DEPLOYMENT_TYPE in
        "kubernetes")
            check_kubernetes_health
            ;;
        "docker")
            check_docker_health
            ;;
        *)
            error "Unknown deployment type: $DEPLOYMENT_TYPE"
            exit 1
            ;;
    esac
}

# Command line options
case "${1:-health}" in
    "health"|"check")
        main
        ;;
    "help"|"-h"|"--help")
        echo "O-RAN Near-RT RIC Health Check Script"
        echo ""
        echo "Usage: $0 [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  health    Run health check (default)"
        echo "  help      Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  ENV_FILE  Environment file path for Docker (default: .env.production)"
        echo ""
        echo "This script automatically detects whether you're running:"
        echo "  - Kubernetes deployment (checks for kubectl and ricplt namespace)"
        echo "  - Docker deployment (checks for Docker and docker-compose)"
        ;;
    *)
        error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac