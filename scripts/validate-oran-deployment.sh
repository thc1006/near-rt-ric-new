#!/bin/bash

# O-RAN SC Deployment Validation Script
# Comprehensive validation for O-RAN SC L Release deployment
# Validates deployment across Docker, Kubernetes, and Helm modes

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NAMESPACE="ricplt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validation summary
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_CHECKS=0

# Track validation result
track_result() {
    local result=$1
    local message="$2"
    
    ((TOTAL_CHECKS++))
    
    case $result in
        "PASS")
            log_success "$message"
            ((PASSED_CHECKS++))
            ;;
        "FAIL")
            log_error "$message"
            ((FAILED_CHECKS++))
            ;;
        "WARN")
            log_warning "$message"
            ((WARNING_CHECKS++))
            ;;
    esac
}

# Detect deployment mode
detect_deployment_mode() {
    log_info "Detecting deployment mode..."
    
    # Check for Docker Compose
    if docker-compose -f "$PROJECT_ROOT/docker-compose.oran-l-release.yml" ps &> /dev/null; then
        DEPLOYMENT_MODE="docker"
        log_info "Detected Docker Compose deployment"
        return 0
    fi
    
    # Check for Kubernetes
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        # Check if it's Helm deployment
        if helm list -n "$NAMESPACE" | grep -q "oran-sc-platform"; then
            DEPLOYMENT_MODE="helm"
            log_info "Detected Helm deployment"
        else
            DEPLOYMENT_MODE="kubernetes"
            log_info "Detected Kubernetes deployment"
        fi
        return 0
    fi
    
    log_error "No O-RAN SC deployment detected"
    return 1
}

# Validate Docker deployment
validate_docker_deployment() {
    log_info "Validating Docker Compose deployment..."
    
    local compose_file="$PROJECT_ROOT/docker-compose.oran-l-release.yml"
    
    # Check if compose file exists
    if [[ -f "$compose_file" ]]; then
        track_result "PASS" "Docker Compose file exists"
    else
        track_result "FAIL" "Docker Compose file not found"
        return 1
    fi
    
    # Check container status
    local containers=("ricplt-dbaas" "ricplt-e2term" "ricplt-e2mgr" "ricplt-submgr" "ricplt-rtmgr" "ricplt-a1mediator")
    
    for container in "${containers[@]}"; do
        if docker ps --format "table {{.Names}}" | grep -q "^$container$"; then
            # Check if container is healthy
            local health_status=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "no-health-check")
            
            if [[ "$health_status" == "healthy" ]] || [[ "$health_status" == "no-health-check" ]]; then
                track_result "PASS" "Container $container is running"
            else
                track_result "FAIL" "Container $container is unhealthy (status: $health_status)"
            fi
        else
            track_result "FAIL" "Container $container is not running"
        fi
    done
    
    # Test service connectivity
    validate_docker_connectivity
    
    # Validate volumes
    validate_docker_volumes
    
    # Validate networks
    validate_docker_networks
}

# Validate Docker connectivity
validate_docker_connectivity() {
    log_info "Validating Docker service connectivity..."
    
    # Test Redis connectivity
    if docker exec ricplt-dbaas redis-cli ping &> /dev/null; then
        track_result "PASS" "Redis database connectivity"
    else
        track_result "FAIL" "Redis database connectivity"
    fi
    
    # Test E2 Manager API
    if curl -f http://localhost:3800/health &> /dev/null; then
        track_result "PASS" "E2 Manager API accessibility"
    else
        track_result "WARN" "E2 Manager API not accessible (may still be starting)"
    fi
    
    # Test A1 Mediator API
    if curl -f http://localhost:10000/a1-p/healthcheck &> /dev/null; then
        track_result "PASS" "A1 Mediator API accessibility"
    else
        track_result "WARN" "A1 Mediator API not accessible (may still be starting)"
    fi
    
    # Test Prometheus metrics
    if curl -f http://localhost:9090/-/healthy &> /dev/null; then
        track_result "PASS" "Prometheus metrics endpoint"
    else
        track_result "WARN" "Prometheus metrics endpoint not accessible"
    fi
    
    # Test Grafana
    if curl -f http://localhost:3000/api/health &> /dev/null; then
        track_result "PASS" "Grafana dashboard accessibility"
    else
        track_result "WARN" "Grafana dashboard not accessible"
    fi
}

# Validate Docker volumes
validate_docker_volumes() {
    log_info "Validating Docker volumes..."
    
    local expected_volumes=("dbaas-data" "prometheus-data" "grafana-data")
    
    for volume in "${expected_volumes[@]}"; do
        if docker volume ls | grep -q "${PROJECT_NAME:-oran-sc-l-release}_$volume"; then
            track_result "PASS" "Volume $volume exists"
        else
            track_result "WARN" "Volume $volume not found"
        fi
    done
}

# Validate Docker networks
validate_docker_networks() {
    log_info "Validating Docker networks..."
    
    local expected_networks=("ricplt-network" "e2-network" "a1-network" "monitoring-network")
    
    for network in "${expected_networks[@]}"; do
        if docker network ls | grep -q "${PROJECT_NAME:-oran-sc-l-release}_$network"; then
            track_result "PASS" "Network $network exists"
        else
            track_result "WARN" "Network $network not found"
        fi
    done
}

# Validate Kubernetes deployment
validate_kubernetes_deployment() {
    log_info "Validating Kubernetes deployment..."
    
    # Check namespace
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        track_result "PASS" "Namespace $NAMESPACE exists"
    else
        track_result "FAIL" "Namespace $NAMESPACE does not exist"
        return 1
    fi
    
    # Check deployments
    local deployments=("ricplt-e2mgr" "ricplt-submgr" "ricplt-rtmgr" "ricplt-a1mediator" "ricplt-e2term")
    
    for deployment in "${deployments[@]}"; do
        if kubectl get deployment "$deployment" -n "$NAMESPACE" &> /dev/null; then
            local ready_replicas=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
            local desired_replicas=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
            
            if [[ "$ready_replicas" == "$desired_replicas" ]]; then
                track_result "PASS" "Deployment $deployment is ready ($ready_replicas/$desired_replicas)"
            else
                track_result "FAIL" "Deployment $deployment is not ready ($ready_replicas/$desired_replicas)"
            fi
        else
            track_result "FAIL" "Deployment $deployment does not exist"
        fi
    done
    
    # Check StatefulSets
    if kubectl get statefulset ricplt-dbaas -n "$NAMESPACE" &> /dev/null; then
        local ready_replicas=$(kubectl get statefulset ricplt-dbaas -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
        local desired_replicas=$(kubectl get statefulset ricplt-dbaas -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
        
        if [[ "$ready_replicas" == "$desired_replicas" ]]; then
            track_result "PASS" "StatefulSet ricplt-dbaas is ready ($ready_replicas/$desired_replicas)"
        else
            track_result "FAIL" "StatefulSet ricplt-dbaas is not ready ($ready_replicas/$desired_replicas)"
        fi
    else
        track_result "FAIL" "StatefulSet ricplt-dbaas does not exist"
    fi
    
    # Validate services
    validate_kubernetes_services
    
    # Validate pods
    validate_kubernetes_pods
    
    # Validate persistent volumes
    validate_kubernetes_storage
    
    # Test connectivity
    validate_kubernetes_connectivity
}

# Validate Kubernetes services
validate_kubernetes_services() {
    log_info "Validating Kubernetes services..."
    
    local services=("service-ricplt-dbaas-tcp" "service-ricplt-e2mgr-http" "service-ricplt-submgr-http" 
                   "service-ricplt-rtmgr-http" "service-ricplt-a1mediator-http" "service-ricplt-e2term-sctp-alpha")
    
    for service in "${services[@]}"; do
        if kubectl get service "$service" -n "$NAMESPACE" &> /dev/null; then
            local endpoints=$(kubectl get endpoints "$service" -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' | wc -w)
            
            if [[ "$endpoints" -gt 0 ]]; then
                track_result "PASS" "Service $service has endpoints"
            else
                track_result "FAIL" "Service $service has no endpoints"
            fi
        else
            track_result "FAIL" "Service $service does not exist"
        fi
    done
}

# Validate Kubernetes pods
validate_kubernetes_pods() {
    log_info "Validating Kubernetes pods..."
    
    # Check for running pods
    local running_pods=$(kubectl get pods -n "$NAMESPACE" --no-headers | grep Running | wc -l)
    local total_pods=$(kubectl get pods -n "$NAMESPACE" --no-headers | wc -l)
    
    if [[ "$running_pods" -eq "$total_pods" && "$total_pods" -gt 0 ]]; then
        track_result "PASS" "All pods are running ($running_pods/$total_pods)"
    else
        track_result "FAIL" "Some pods are not running ($running_pods/$total_pods)"
        
        # List problematic pods
        kubectl get pods -n "$NAMESPACE" --no-headers | grep -v Running | while read -r line; do
            local pod_name=$(echo "$line" | awk '{print $1}')
            local pod_status=$(echo "$line" | awk '{print $3}')
            track_result "FAIL" "Pod $pod_name is in $pod_status state"
        done
    fi
    
    # Check resource usage
    validate_pod_resources
}

# Validate pod resources
validate_pod_resources() {
    log_info "Validating pod resource usage..."
    
    # Get resource usage for each pod
    local pods=$(kubectl get pods -n "$NAMESPACE" --no-headers | awk '{print $1}')
    
    for pod in $pods; do
        local cpu_usage=$(kubectl top pod "$pod" -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print $2}' || echo "unknown")
        local memory_usage=$(kubectl top pod "$pod" -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print $3}' || echo "unknown")
        
        if [[ "$cpu_usage" != "unknown" && "$memory_usage" != "unknown" ]]; then
            track_result "PASS" "Pod $pod resource usage: CPU=$cpu_usage, Memory=$memory_usage"
        else
            track_result "WARN" "Pod $pod resource usage not available"
        fi
    done
}

# Validate Kubernetes storage
validate_kubernetes_storage() {
    log_info "Validating Kubernetes storage..."
    
    # Check PVCs
    local pvcs=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    
    if [[ "$pvcs" -gt 0 ]]; then
        local bound_pvcs=$(kubectl get pvc -n "$NAMESPACE" --no-headers | grep Bound | wc -l)
        
        if [[ "$bound_pvcs" -eq "$pvcs" ]]; then
            track_result "PASS" "All PVCs are bound ($bound_pvcs/$pvcs)"
        else
            track_result "FAIL" "Some PVCs are not bound ($bound_pvcs/$pvcs)"
        fi
    else
        track_result "WARN" "No PVCs found (using ephemeral storage)"
    fi
    
    # Check storage classes
    if kubectl get storageclass local-path &> /dev/null; then
        track_result "PASS" "Default storage class available"
    else
        track_result "WARN" "Default storage class not found"
    fi
}

# Validate Kubernetes connectivity
validate_kubernetes_connectivity() {
    log_info "Validating Kubernetes service connectivity..."
    
    # Test internal connectivity using port-forward
    local services=("service-ricplt-dbaas-tcp:6379" "service-ricplt-e2mgr-http:3800" "service-ricplt-a1mediator-http:10000")
    
    for service_port in "${services[@]}"; do
        local service=$(echo "$service_port" | cut -d: -f1)
        local port=$(echo "$service_port" | cut -d: -f2)
        local local_port=$((8000 + RANDOM % 1000))
        
        # Start port-forward in background
        kubectl port-forward -n "$NAMESPACE" "svc/$service" "$local_port:$port" &> /dev/null &
        local pf_pid=$!
        
        # Wait a moment for port-forward to establish
        sleep 2
        
        # Test connectivity
        case $port in
            6379)
                if echo "PING" | nc localhost "$local_port" &> /dev/null; then
                    track_result "PASS" "Service $service connectivity"
                else
                    track_result "FAIL" "Service $service connectivity"
                fi
                ;;
            *)
                if curl -f "http://localhost:$local_port/health" &> /dev/null 2>&1; then
                    track_result "PASS" "Service $service connectivity"
                else
                    track_result "WARN" "Service $service connectivity (health endpoint may not be available)"
                fi
                ;;
        esac
        
        # Kill port-forward
        kill $pf_pid 2>/dev/null || true
        wait $pf_pid 2>/dev/null || true
    done
}

# Validate Helm deployment
validate_helm_deployment() {
    log_info "Validating Helm deployment..."
    
    # Check Helm release
    if helm list -n "$NAMESPACE" | grep -q "oran-sc-platform"; then
        local release_status=$(helm status oran-sc-platform -n "$NAMESPACE" -o json | jq -r '.info.status')
        
        if [[ "$release_status" == "deployed" ]]; then
            track_result "PASS" "Helm release status: $release_status"
        else
            track_result "FAIL" "Helm release status: $release_status"
        fi
        
        # Get release info
        local chart_version=$(helm list -n "$NAMESPACE" -o json | jq -r '.[] | select(.name=="oran-sc-platform") | .chart')
        track_result "PASS" "Helm chart version: $chart_version"
    else
        track_result "FAIL" "Helm release 'oran-sc-platform' not found"
        return 1
    fi
    
    # Validate Kubernetes resources (same as kubernetes deployment)
    validate_kubernetes_deployment
}

# Validate O-RAN SC specific functionality
validate_oran_functionality() {
    log_info "Validating O-RAN SC specific functionality..."
    
    case $DEPLOYMENT_MODE in
        docker)
            validate_oran_interfaces_docker
            ;;
        kubernetes|helm)
            validate_oran_interfaces_kubernetes
            ;;
    esac
    
    # Validate message routing
    validate_rmr_routing
    
    # Validate database operations
    validate_database_operations
}

# Validate O-RAN interfaces (Docker)
validate_oran_interfaces_docker() {
    log_info "Validating O-RAN interfaces (Docker)..."
    
    # Test E2 interface (SCTP)
    if nc -z localhost 36421 &> /dev/null; then
        track_result "PASS" "E2 interface (SCTP) port 36421 is accessible"
    else
        track_result "FAIL" "E2 interface (SCTP) port 36421 is not accessible"
    fi
    
    # Test A1 interface (HTTP)
    if curl -f http://localhost:10000/a1-p/healthcheck &> /dev/null; then
        track_result "PASS" "A1 interface HTTP endpoint is accessible"
    else
        track_result "WARN" "A1 interface HTTP endpoint not accessible"
    fi
    
    # Test O1 interface (NETCONF)
    if nc -z localhost 830 &> /dev/null; then
        track_result "PASS" "O1 interface (NETCONF) port 830 is accessible"
    else
        track_result "WARN" "O1 interface (NETCONF) port 830 not accessible"
    fi
}

# Validate O-RAN interfaces (Kubernetes)
validate_oran_interfaces_kubernetes() {
    log_info "Validating O-RAN interfaces (Kubernetes)..."
    
    # Check E2 NodePort service
    if kubectl get service service-ricplt-e2term-sctp-nodeport -n "$NAMESPACE" &> /dev/null; then
        local nodeport=$(kubectl get service service-ricplt-e2term-sctp-nodeport -n "$NAMESPACE" -o jsonpath='{.spec.ports[0].nodePort}')
        track_result "PASS" "E2 interface exposed on NodePort $nodeport"
    else
        track_result "WARN" "E2 interface NodePort service not found"
    fi
    
    # Check A1 service
    if kubectl get service service-ricplt-a1mediator-http -n "$NAMESPACE" &> /dev/null; then
        track_result "PASS" "A1 Mediator service exists"
    else
        track_result "FAIL" "A1 Mediator service not found"
    fi
}

# Validate RMR routing
validate_rmr_routing() {
    log_info "Validating RMR routing..."
    
    case $DEPLOYMENT_MODE in
        docker)
            # Check if routing manager is running and accessible
            if curl -f http://localhost:3802/health &> /dev/null; then
                track_result "PASS" "Routing Manager is accessible"
            else
                track_result "WARN" "Routing Manager not accessible"
            fi
            ;;
        kubernetes|helm)
            # Check routing manager service
            if kubectl get service service-ricplt-rtmgr-http -n "$NAMESPACE" &> /dev/null; then
                track_result "PASS" "Routing Manager service exists"
            else
                track_result "FAIL" "Routing Manager service not found"
            fi
            ;;
    esac
}

# Validate database operations
validate_database_operations() {
    log_info "Validating database operations..."
    
    case $DEPLOYMENT_MODE in
        docker)
            # Test basic Redis operations
            if docker exec ricplt-dbaas redis-cli set test-key test-value &> /dev/null; then
                if [[ "$(docker exec ricplt-dbaas redis-cli get test-key)" == "test-value" ]]; then
                    track_result "PASS" "Database read/write operations"
                    docker exec ricplt-dbaas redis-cli del test-key &> /dev/null
                else
                    track_result "FAIL" "Database read operation failed"
                fi
            else
                track_result "FAIL" "Database write operation failed"
            fi
            ;;
        kubernetes|helm)
            # Test database through port-forward
            kubectl port-forward -n "$NAMESPACE" svc/service-ricplt-dbaas-tcp 16379:6379 &> /dev/null &
            local pf_pid=$!
            sleep 2
            
            if echo -e "SET test-key test-value\r\nGET test-key\r\nDEL test-key\r\n" | nc localhost 16379 | grep -q "test-value"; then
                track_result "PASS" "Database read/write operations"
            else
                track_result "FAIL" "Database operations failed"
            fi
            
            kill $pf_pid 2>/dev/null || true
            ;;
    esac
}

# Generate validation report
generate_report() {
    echo ""
    echo "=================================================="
    echo "O-RAN SC Deployment Validation Report"
    echo "=================================================="
    echo "Deployment Mode: $DEPLOYMENT_MODE"
    echo "Namespace: $NAMESPACE"
    echo "Validation Time: $(date)"
    echo ""
    echo "Summary:"
    echo "  Total Checks: $TOTAL_CHECKS"
    echo "  Passed: $PASSED_CHECKS"
    echo "  Failed: $FAILED_CHECKS"
    echo "  Warnings: $WARNING_CHECKS"
    echo ""
    
    local pass_rate=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
    
    if [[ $FAILED_CHECKS -eq 0 ]]; then
        if [[ $WARNING_CHECKS -eq 0 ]]; then
            echo -e "${GREEN}✓ Deployment is fully validated and operational${NC}"
        else
            echo -e "${YELLOW}⚠ Deployment is operational with $WARNING_CHECKS warnings${NC}"
        fi
    else
        echo -e "${RED}✗ Deployment has $FAILED_CHECKS critical issues${NC}"
    fi
    
    echo "Pass Rate: ${pass_rate}%"
    echo ""
    
    if [[ $FAILED_CHECKS -gt 0 ]]; then
        echo "Next Steps:"
        echo "1. Review failed checks above"
        echo "2. Check component logs for errors"
        echo "3. Verify configuration files"
        echo "4. Restart failed components if necessary"
        return 1
    else
        echo "Deployment Status: HEALTHY"
        return 0
    fi
}

# Main function
main() {
    log_info "O-RAN SC Deployment Validation"
    log_info "==============================="
    
    # Detect deployment mode
    if ! detect_deployment_mode; then
        exit 1
    fi
    
    # Run validation based on deployment mode
    case $DEPLOYMENT_MODE in
        docker)
            validate_docker_deployment
            ;;
        kubernetes)
            validate_kubernetes_deployment
            ;;
        helm)
            validate_helm_deployment
            ;;
    esac
    
    # Validate O-RAN SC specific functionality
    validate_oran_functionality
    
    # Generate and show report
    generate_report
}

# Execute main function
main "$@"