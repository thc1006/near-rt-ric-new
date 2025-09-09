#!/bin/bash

# O-RAN SC L Release Near-RT RIC Deployment Script
# Following official O-RAN SC deployment guide and standards
# Production-ready deployment with validation and monitoring

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOYMENT_MODE="docker"  # docker | kubernetes | helm
ORAN_RELEASE="l-release"
NAMESPACE="ricplt"

# O-RAN SC official container registry
ORAN_REGISTRY="nexus3.o-ran-sc.org:10002"

# Deployment files
DOCKER_COMPOSE_FILE="$PROJECT_ROOT/docker-compose.oran-l-release.yml"
K8S_MANIFEST_FILE="$PROJECT_ROOT/deployments/oran-sc-l-release.yaml"
HELM_CHART_DIR="$PROJECT_ROOT/helm/ric-platform"

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

# Help function
show_help() {
    cat << EOF
O-RAN SC L Release Near-RT RIC Deployment Script

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -m, --mode MODE         Deployment mode: docker|kubernetes|helm (default: docker)
    -n, --namespace NS      Kubernetes namespace (default: ricplt)
    --pre-check            Run pre-deployment checks only
    --post-check           Run post-deployment validation
    --cleanup              Clean up existing deployment
    --logs                 Show component logs
    --status               Show deployment status
    --scale COMPONENT N    Scale component to N replicas (k8s/helm only)

DEPLOYMENT MODES:
    docker                 Deploy using Docker Compose (development/testing)
    kubernetes            Deploy using Kubernetes manifests (production)
    helm                  Deploy using Helm charts (production with customization)

EXAMPLES:
    # Deploy using Docker Compose (default)
    $0

    # Deploy to Kubernetes
    $0 --mode kubernetes

    # Deploy using Helm with custom namespace
    $0 --mode helm --namespace my-ric

    # Check deployment status
    $0 --status

    # Run pre-deployment checks
    $0 --pre-check

    # Clean up deployment
    $0 --cleanup

EOF
}

# Parse command line arguments
parse_args() {
    PRE_CHECK=false
    POST_CHECK=false
    CLEANUP=false
    SHOW_LOGS=false
    SHOW_STATUS=false
    SCALE_COMPONENT=""
    SCALE_REPLICAS=""

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -m|--mode)
                DEPLOYMENT_MODE="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            --pre-check)
                PRE_CHECK=true
                shift
                ;;
            --post-check)
                POST_CHECK=true
                shift
                ;;
            --cleanup)
                CLEANUP=true
                shift
                ;;
            --logs)
                SHOW_LOGS=true
                shift
                ;;
            --status)
                SHOW_STATUS=true
                shift
                ;;
            --scale)
                SCALE_COMPONENT="$2"
                SCALE_REPLICAS="$3"
                shift 3
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Validate deployment mode
    case $DEPLOYMENT_MODE in
        docker|kubernetes|helm)
            ;;
        *)
            log_error "Invalid deployment mode: $DEPLOYMENT_MODE"
            show_help
            exit 1
            ;;
    esac
}

# Pre-deployment checks
run_pre_checks() {
    log_info "Running pre-deployment checks for O-RAN SC L Release..."
    
    local errors=0
    
    # Check deployment mode specific requirements
    case $DEPLOYMENT_MODE in
        docker)
            if ! command -v docker &> /dev/null; then
                log_error "Docker is not installed"
                ((errors++))
            fi
            
            if ! command -v docker-compose &> /dev/null; then
                log_error "Docker Compose is not installed"
                ((errors++))
            fi
            
            # Check Docker daemon
            if ! docker info &> /dev/null; then
                log_error "Docker daemon is not running"
                ((errors++))
            fi
            
            # Check Docker Compose file
            if [[ ! -f "$DOCKER_COMPOSE_FILE" ]]; then
                log_error "Docker Compose file not found: $DOCKER_COMPOSE_FILE"
                ((errors++))
            fi
            ;;
            
        kubernetes|helm)
            if ! command -v kubectl &> /dev/null; then
                log_error "kubectl is not installed"
                ((errors++))
            fi
            
            # Check Kubernetes connectivity
            if ! kubectl cluster-info &> /dev/null; then
                log_error "Cannot connect to Kubernetes cluster"
                ((errors++))
            fi
            
            # Check Kubernetes version (minimum v1.20)
            K8S_VERSION=$(kubectl version --output=json | jq -r '.serverVersion.gitVersion' | tr -d 'v')
            MAJOR=$(echo "$K8S_VERSION" | cut -d. -f1)
            MINOR=$(echo "$K8S_VERSION" | cut -d. -f2)
            
            if [[ "$MAJOR" -lt 1 ]] || [[ "$MAJOR" -eq 1 && "$MINOR" -lt 20 ]]; then
                log_error "Kubernetes version $K8S_VERSION is not supported. Minimum required: v1.20"
                ((errors++))
            fi
            
            if [[ "$DEPLOYMENT_MODE" == "helm" ]]; then
                if ! command -v helm &> /dev/null; then
                    log_error "Helm is not installed"
                    ((errors++))
                fi
                
                # Check Helm chart
                if [[ ! -d "$HELM_CHART_DIR" ]]; then
                    log_error "Helm chart directory not found: $HELM_CHART_DIR"
                    ((errors++))
                fi
            fi
            
            # Check Kubernetes manifest
            if [[ "$DEPLOYMENT_MODE" == "kubernetes" && ! -f "$K8S_MANIFEST_FILE" ]]; then
                log_error "Kubernetes manifest file not found: $K8S_MANIFEST_FILE"
                ((errors++))
            fi
            ;;
    esac
    
    # Check system resources
    check_system_resources
    
    # Check container registry access
    check_registry_access
    
    # Check configuration files
    check_config_files
    
    if [[ $errors -gt 0 ]]; then
        log_error "Pre-deployment checks failed with $errors errors"
        return 1
    else
        log_success "Pre-deployment checks passed"
        return 0
    fi
}

# Check system resources
check_system_resources() {
    log_info "Checking system resources..."
    
    # Check memory (minimum 8GB recommended)
    MEMORY_GB=$(free -g | awk '/^Mem:/{print $2}')
    if [[ "$MEMORY_GB" -lt 8 ]]; then
        log_warning "System has ${MEMORY_GB}GB RAM. 8GB+ recommended for O-RAN SC"
    else
        log_success "Memory: ${MEMORY_GB}GB"
    fi
    
    # Check disk space
    DISK_AVAILABLE=$(df "$PROJECT_ROOT" | awk 'NR==2{print int($4/1024/1024)}')
    if [[ "$DISK_AVAILABLE" -lt 10 ]]; then
        log_warning "Available disk space: ${DISK_AVAILABLE}GB. 10GB+ recommended"
    else
        log_success "Available disk space: ${DISK_AVAILABLE}GB"
    fi
    
    # Check CPU cores
    CPU_CORES=$(nproc)
    if [[ "$CPU_CORES" -lt 4 ]]; then
        log_warning "System has ${CPU_CORES} CPU cores. 4+ recommended"
    else
        log_success "CPU cores: ${CPU_CORES}"
    fi
}

# Check container registry access
check_registry_access() {
    log_info "Checking O-RAN SC container registry access..."
    
    # Try to pull a lightweight image to test registry access
    if docker pull "${ORAN_REGISTRY}/o-ran-sc/ric-plt-rtmgr:0.8.0" &> /dev/null; then
        log_success "O-RAN SC registry accessible"
        docker rmi "${ORAN_REGISTRY}/o-ran-sc/ric-plt-rtmgr:0.8.0" &> /dev/null || true
    else
        log_warning "Cannot access O-RAN SC registry. Will use cached images if available."
    fi
}

# Check configuration files
check_config_files() {
    log_info "Checking configuration files..."
    
    local config_dirs=("configs" "deployments" "helm")
    local missing_configs=0
    
    for dir in "${config_dirs[@]}"; do
        if [[ ! -d "$PROJECT_ROOT/$dir" ]]; then
            log_warning "Configuration directory missing: $dir"
            ((missing_configs++))
        fi
    done
    
    if [[ $missing_configs -gt 0 ]]; then
        log_warning "$missing_configs configuration directories are missing. Creating defaults..."
        create_default_configs
    else
        log_success "Configuration files present"
    fi
}

# Create default configuration files
create_default_configs() {
    log_info "Creating default configuration files..."
    
    # Create configs directory structure
    mkdir -p "$PROJECT_ROOT/configs"/{prometheus,grafana/dashboards,grafana/datasources}
    mkdir -p "$PROJECT_ROOT/configs"/{e2term,e2mgr,submgr,rtmgr,a1mediator,o1mediator}
    mkdir -p "$PROJECT_ROOT/configs"/{cu-cp,cu-up,du}
    mkdir -p "$PROJECT_ROOT/configs"/xapp/hello-world
    
    # Create basic Prometheus configuration
    cat > "$PROJECT_ROOT/configs/prometheus/prometheus.yml" << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  # "first_rules.yml"
  # "second_rules.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'ric-platform'
    static_configs:
      - targets: ['ricplt-e2mgr:8080', 'ricplt-submgr:8080', 'ricplt-rtmgr:8080', 'ricplt-a1mediator:8080']
    metrics_path: /metrics
    scrape_interval: 30s

  - job_name: 'xapps'
    static_configs:
      - targets: ['ricxapp-hello-world:8080']
    metrics_path: /metrics
    scrape_interval: 30s
EOF

    # Create basic Grafana datasource
    cat > "$PROJECT_ROOT/configs/grafana/datasources/prometheus.yml" << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
EOF

    log_success "Default configuration files created"
}

# Deploy using Docker Compose
deploy_docker() {
    log_info "Deploying O-RAN SC L Release using Docker Compose..."
    
    # Set environment variables
    export VERSION="$ORAN_RELEASE"
    export COMPOSE_PROJECT_NAME="oran-sc-l-release"
    
    # Pull latest images
    log_info "Pulling container images..."
    docker-compose -f "$DOCKER_COMPOSE_FILE" pull
    
    # Start services
    log_info "Starting O-RAN SC services..."
    docker-compose -f "$DOCKER_COMPOSE_FILE" up -d
    
    # Wait for services to be ready
    wait_for_services_docker
    
    log_success "O-RAN SC L Release deployed successfully with Docker Compose"
}

# Deploy using Kubernetes
deploy_kubernetes() {
    log_info "Deploying O-RAN SC L Release using Kubernetes..."
    
    # Create namespace
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply manifests
    kubectl apply -f "$K8S_MANIFEST_FILE"
    
    # Wait for deployments
    wait_for_services_kubernetes
    
    log_success "O-RAN SC L Release deployed successfully with Kubernetes"
}

# Deploy using Helm
deploy_helm() {
    log_info "Deploying O-RAN SC L Release using Helm..."
    
    # Update Helm dependencies
    helm dependency update "$HELM_CHART_DIR"
    
    # Install/upgrade release
    helm upgrade --install oran-sc-platform "$HELM_CHART_DIR" \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --values "$HELM_CHART_DIR/values.yaml" \
        --wait \
        --timeout 600s
    
    log_success "O-RAN SC L Release deployed successfully with Helm"
}

# Wait for Docker services
wait_for_services_docker() {
    log_info "Waiting for services to be ready..."
    
    local services=("ricplt-dbaas" "ricplt-e2term" "ricplt-e2mgr" "ricplt-submgr" "ricplt-rtmgr" "ricplt-a1mediator")
    local max_wait=300
    local wait_time=0
    
    for service in "${services[@]}"; do
        log_info "Waiting for $service..."
        while [[ $wait_time -lt $max_wait ]]; do
            if docker-compose -f "$DOCKER_COMPOSE_FILE" ps "$service" | grep -q "Up"; then
                if docker-compose -f "$DOCKER_COMPOSE_FILE" exec -T "$service" curl -f http://localhost:8080/health &> /dev/null 2>&1 || \
                   docker-compose -f "$DOCKER_COMPOSE_FILE" exec -T "$service" redis-cli ping &> /dev/null 2>&1; then
                    log_success "$service is ready"
                    break
                fi
            fi
            sleep 10
            ((wait_time+=10))
        done
        
        if [[ $wait_time -ge $max_wait ]]; then
            log_warning "$service not ready within timeout"
        fi
        wait_time=0
    done
}

# Wait for Kubernetes services
wait_for_services_kubernetes() {
    log_info "Waiting for services to be ready..."
    
    # Wait for deployments
    kubectl wait --for=condition=available --timeout=300s deployment --all -n "$NAMESPACE"
    
    # Wait for statefulsets
    kubectl wait --for=condition=ready --timeout=300s pod -l app=ricplt-dbaas -n "$NAMESPACE"
    
    log_success "All services are ready"
}

# Show deployment status
show_deployment_status() {
    log_info "Deployment Status for O-RAN SC L Release"
    echo "=========================================="
    
    case $DEPLOYMENT_MODE in
        docker)
            echo "Docker Compose Status:"
            docker-compose -f "$DOCKER_COMPOSE_FILE" ps
            echo ""
            echo "Container Health:"
            docker-compose -f "$DOCKER_COMPOSE_FILE" exec ricplt-dbaas redis-cli ping 2>/dev/null && echo "✓ Database (Redis) - Healthy" || echo "✗ Database (Redis) - Unhealthy"
            ;;
        kubernetes|helm)
            echo "Namespace: $NAMESPACE"
            echo ""
            echo "Pod Status:"
            kubectl get pods -n "$NAMESPACE" -o wide
            echo ""
            echo "Service Status:"
            kubectl get services -n "$NAMESPACE"
            echo ""
            echo "Deployment Status:"
            kubectl get deployments -n "$NAMESPACE"
            ;;
    esac
    
    echo ""
    show_access_info
}

# Show access information
show_access_info() {
    log_info "Access Information"
    echo "=================="
    
    case $DEPLOYMENT_MODE in
        docker)
            echo "O-RAN SC Interfaces:"
            echo "• E2 Interface (SCTP): localhost:36421"
            echo "• A1 Interface (HTTP): http://localhost:10000"
            echo "• O1 Interface (NETCONF): localhost:830"
            echo ""
            echo "Management Interfaces:"
            echo "• E2 Manager API: http://localhost:3800"
            echo "• Subscription Manager API: http://localhost:3801"
            echo "• Routing Manager API: http://localhost:3802"
            echo ""
            echo "Monitoring:"
            echo "• Prometheus: http://localhost:9090"
            echo "• Grafana: http://localhost:3000 (admin/admin123)"
            echo "• Jaeger: http://localhost:16686"
            ;;
        kubernetes|helm)
            echo "Use kubectl port-forward to access services:"
            echo "• A1 Interface: kubectl port-forward -n $NAMESPACE svc/service-ricplt-a1mediator-http 10000:10000"
            echo "• E2 Manager: kubectl port-forward -n $NAMESPACE svc/service-ricplt-e2mgr-http 3800:3800"
            echo "• Subscription Manager: kubectl port-forward -n $NAMESPACE svc/service-ricplt-submgr-http 3800:3800"
            echo "• Routing Manager: kubectl port-forward -n $NAMESPACE svc/service-ricplt-rtmgr-http 3800:3800"
            ;;
    esac
}

# Show logs
show_deployment_logs() {
    log_info "Showing deployment logs..."
    
    case $DEPLOYMENT_MODE in
        docker)
            echo "Recent logs from all services:"
            docker-compose -f "$DOCKER_COMPOSE_FILE" logs --tail=50
            ;;
        kubernetes|helm)
            echo "Recent logs from RIC Platform components:"
            kubectl logs -n "$NAMESPACE" -l app=ricplt-e2mgr --tail=20
            kubectl logs -n "$NAMESPACE" -l app=ricplt-submgr --tail=20
            kubectl logs -n "$NAMESPACE" -l app=ricplt-rtmgr --tail=20
            ;;
    esac
}

# Clean up deployment
cleanup_deployment() {
    log_warning "Cleaning up O-RAN SC deployment..."
    
    read -p "Are you sure you want to remove the deployment? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Cleanup cancelled"
        return 0
    fi
    
    case $DEPLOYMENT_MODE in
        docker)
            docker-compose -f "$DOCKER_COMPOSE_FILE" down -v
            # Remove images (optional)
            read -p "Remove pulled images as well? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                docker-compose -f "$DOCKER_COMPOSE_FILE" down --rmi all -v
            fi
            ;;
        kubernetes)
            kubectl delete -f "$K8S_MANIFEST_FILE" --ignore-not-found
            kubectl delete namespace "$NAMESPACE" --ignore-not-found
            ;;
        helm)
            helm uninstall oran-sc-platform -n "$NAMESPACE" --ignore-not-found
            kubectl delete namespace "$NAMESPACE" --ignore-not-found
            ;;
    esac
    
    log_success "Cleanup completed"
}

# Scale components (Kubernetes/Helm only)
scale_component() {
    if [[ "$DEPLOYMENT_MODE" == "docker" ]]; then
        log_error "Scaling is not supported in Docker mode"
        return 1
    fi
    
    log_info "Scaling $SCALE_COMPONENT to $SCALE_REPLICAS replicas..."
    
    case $DEPLOYMENT_MODE in
        kubernetes)
            kubectl scale deployment "$SCALE_COMPONENT" --replicas="$SCALE_REPLICAS" -n "$NAMESPACE"
            ;;
        helm)
            # For Helm, we need to update values and upgrade
            log_warning "Helm scaling requires values file update and upgrade"
            ;;
    esac
    
    log_success "Scaling initiated"
}

# Run post-deployment validation
run_post_checks() {
    log_info "Running post-deployment validation..."
    
    case $DEPLOYMENT_MODE in
        docker)
            # Check if all containers are running
            RUNNING_CONTAINERS=$(docker-compose -f "$DOCKER_COMPOSE_FILE" ps -q | wc -l)
            TOTAL_CONTAINERS=$(docker-compose -f "$DOCKER_COMPOSE_FILE" config --services | wc -l)
            
            if [[ "$RUNNING_CONTAINERS" -eq "$TOTAL_CONTAINERS" ]]; then
                log_success "All containers are running ($RUNNING_CONTAINERS/$TOTAL_CONTAINERS)"
            else
                log_error "Some containers are not running ($RUNNING_CONTAINERS/$TOTAL_CONTAINERS)"
                return 1
            fi
            ;;
        kubernetes|helm)
            # Check if all pods are ready
            NOT_READY_PODS=$(kubectl get pods -n "$NAMESPACE" --no-headers | grep -v Running | wc -l)
            if [[ "$NOT_READY_PODS" -eq 0 ]]; then
                log_success "All pods are running"
            else
                log_error "$NOT_READY_PODS pods are not ready"
                kubectl get pods -n "$NAMESPACE"
                return 1
            fi
            ;;
    esac
    
    # Test basic connectivity
    test_component_connectivity
    
    log_success "Post-deployment validation passed"
}

# Test component connectivity
test_component_connectivity() {
    log_info "Testing component connectivity..."
    
    case $DEPLOYMENT_MODE in
        docker)
            # Test Redis connectivity
            if docker-compose -f "$DOCKER_COMPOSE_FILE" exec -T ricplt-dbaas redis-cli ping &> /dev/null; then
                log_success "Database connectivity OK"
            else
                log_error "Database connectivity failed"
            fi
            
            # Test A1 Mediator
            if curl -f http://localhost:10000/a1-p/healthcheck &> /dev/null; then
                log_success "A1 Mediator connectivity OK"
            else
                log_warning "A1 Mediator not accessible (may still be starting)"
            fi
            ;;
        kubernetes|helm)
            # Test using kubectl port-forward in background
            kubectl port-forward -n "$NAMESPACE" svc/service-ricplt-dbaas-tcp 6379:6379 &> /dev/null &
            PF_PID=$!
            sleep 2
            
            if redis-cli -h localhost -p 6379 ping &> /dev/null; then
                log_success "Database connectivity OK"
            else
                log_warning "Database connectivity test failed"
            fi
            
            kill $PF_PID 2>/dev/null || true
            ;;
    esac
}

# Main function
main() {
    log_info "O-RAN SC L Release Near-RT RIC Deployment"
    log_info "=========================================="
    
    # Parse arguments
    parse_args "$@"
    
    # Handle specific actions
    if [[ "$PRE_CHECK" == true ]]; then
        run_pre_checks
        exit $?
    fi
    
    if [[ "$POST_CHECK" == true ]]; then
        run_post_checks
        exit $?
    fi
    
    if [[ "$CLEANUP" == true ]]; then
        cleanup_deployment
        exit 0
    fi
    
    if [[ "$SHOW_LOGS" == true ]]; then
        show_deployment_logs
        exit 0
    fi
    
    if [[ "$SHOW_STATUS" == true ]]; then
        show_deployment_status
        exit 0
    fi
    
    if [[ -n "$SCALE_COMPONENT" ]]; then
        scale_component
        exit 0
    fi
    
    # Run pre-deployment checks
    if ! run_pre_checks; then
        log_error "Pre-deployment checks failed. Use --pre-check for details."
        exit 1
    fi
    
    # Deploy based on mode
    case $DEPLOYMENT_MODE in
        docker)
            deploy_docker
            ;;
        kubernetes)
            deploy_kubernetes
            ;;
        helm)
            deploy_helm
            ;;
    esac
    
    # Run post-deployment validation
    run_post_checks
    
    # Show deployment info
    show_deployment_status
    
    log_success "O-RAN SC L Release deployment completed successfully!"
    log_info "Use '$0 --status' to check deployment status anytime"
    log_info "Use '$0 --logs' to view component logs"
}

# Execute main function with all arguments
main "$@"