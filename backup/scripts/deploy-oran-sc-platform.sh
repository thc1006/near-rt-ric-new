#!/bin/bash

# O-RAN SC Platform Foundation Deployment Script
# This script deploys the core O-RAN SC components with proper configuration

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NAMESPACE="ricplt"
XAPP_NAMESPACE="ricxapp"
RELEASE_NAME="oran-sc-platform"
CHART_PATH="$PROJECT_ROOT/helm/oran-sc-platform"
VALUES_FILE="$PROJECT_ROOT/helm/oran-sc-platform/values.yaml"
OVERRIDE_FILE=""

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
O-RAN SC Platform Foundation Deployment Script

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -n, --namespace         Kubernetes namespace (default: ricplt)
    -r, --release           Helm release name (default: oran-sc-platform)
    -f, --values-file       Custom values file path
    -o, --override-file     Additional override file
    --dry-run              Perform a dry run without actual deployment
    --debug                Enable debug mode
    --skip-deps            Skip dependency update
    --uninstall            Uninstall the platform

EXAMPLES:
    # Deploy with default configuration
    $0

    # Deploy with custom values file
    $0 -f custom-values.yaml

    # Deploy with override file
    $0 -o production-overrides.yaml

    # Dry run deployment
    $0 --dry-run

    # Uninstall platform
    $0 --uninstall

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -r|--release)
                RELEASE_NAME="$2"
                shift 2
                ;;
            -f|--values-file)
                VALUES_FILE="$2"
                shift 2
                ;;
            -o|--override-file)
                OVERRIDE_FILE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            --debug)
                DEBUG="true"
                shift
                ;;
            --skip-deps)
                SKIP_DEPS="true"
                shift
                ;;
            --uninstall)
                UNINSTALL="true"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if helm is available
    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed or not in PATH"
        exit 1
    fi
    
    # Check Kubernetes connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check Helm version
    HELM_VERSION=$(helm version --short --client | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+')
    log_info "Using Helm version: $HELM_VERSION"
    
    # Check Kubernetes version
    K8S_VERSION=$(kubectl version --short --client | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+')
    log_info "Using Kubernetes client version: $K8S_VERSION"
    
    log_success "Prerequisites check completed"
}

# Create namespaces
create_namespaces() {
    log_info "Creating namespaces..."
    
    # Create platform namespace
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        kubectl create namespace "$NAMESPACE"
        log_success "Created namespace: $NAMESPACE"
    else
        log_info "Namespace $NAMESPACE already exists"
    fi
    
    # Create xApp namespace
    if ! kubectl get namespace "$XAPP_NAMESPACE" &> /dev/null; then
        kubectl create namespace "$XAPP_NAMESPACE"
        log_success "Created namespace: $XAPP_NAMESPACE"
    else
        log_info "Namespace $XAPP_NAMESPACE already exists"
    fi
    
    # Label namespaces
    kubectl label namespace "$NAMESPACE" name="$NAMESPACE" --overwrite
    kubectl label namespace "$XAPP_NAMESPACE" name="$XAPP_NAMESPACE" --overwrite
}

# Update Helm dependencies
update_dependencies() {
    if [[ "${SKIP_DEPS:-false}" == "true" ]]; then
        log_info "Skipping dependency update"
        return
    fi
    
    log_info "Updating Helm chart dependencies..."
    cd "$CHART_PATH"
    
    # Update dependencies
    helm dependency update
    
    log_success "Dependencies updated successfully"
    cd "$PROJECT_ROOT"
}

# Validate Helm chart
validate_chart() {
    log_info "Validating Helm chart..."
    
    # Lint the chart
    if ! helm lint "$CHART_PATH"; then
        log_error "Helm chart validation failed"
        exit 1
    fi
    
    log_success "Helm chart validation passed"
}

# Deploy platform
deploy_platform() {
    log_info "Deploying O-RAN SC Platform Foundation..."
    
    # Prepare Helm command
    HELM_CMD="helm upgrade --install $RELEASE_NAME $CHART_PATH"
    HELM_CMD="$HELM_CMD --namespace $NAMESPACE"
    HELM_CMD="$HELM_CMD --create-namespace"
    HELM_CMD="$HELM_CMD --values $VALUES_FILE"
    
    # Add override file if specified
    if [[ -n "$OVERRIDE_FILE" ]]; then
        HELM_CMD="$HELM_CMD --values $OVERRIDE_FILE"
    fi
    
    # Add debug flag if specified
    if [[ "${DEBUG:-false}" == "true" ]]; then
        HELM_CMD="$HELM_CMD --debug"
    fi
    
    # Add dry-run flag if specified
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        HELM_CMD="$HELM_CMD --dry-run"
        log_info "Performing dry run deployment..."
    fi
    
    # Execute deployment
    log_info "Executing: $HELM_CMD"
    if eval "$HELM_CMD"; then
        if [[ "${DRY_RUN:-false}" != "true" ]]; then
            log_success "O-RAN SC Platform deployed successfully"
        else
            log_success "Dry run completed successfully"
        fi
    else
        log_error "Deployment failed"
        exit 1
    fi
}

# Wait for deployment
wait_for_deployment() {
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        return
    fi
    
    log_info "Waiting for deployment to be ready..."
    
    # Core components to wait for
    COMPONENTS=("dbaas" "e2term" "e2mgr" "submgr" "rtmgr")
    
    for component in "${COMPONENTS[@]}"; do
        log_info "Waiting for $component to be ready..."
        if ! kubectl wait --for=condition=available --timeout=300s deployment -l app="$component" -n "$NAMESPACE"; then
            log_warning "$component deployment not ready within timeout"
        else
            log_success "$component is ready"
        fi
    done
    
    # Wait for StatefulSets (like dbaas)
    log_info "Waiting for StatefulSets to be ready..."
    if ! kubectl wait --for=condition=ready --timeout=300s pod -l app=dbaas -n "$NAMESPACE"; then
        log_warning "StatefulSet pods not ready within timeout"
    else
        log_success "StatefulSet pods are ready"
    fi
}

# Verify deployment
verify_deployment() {
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        return
    fi
    
    log_info "Verifying deployment..."
    
    # Check pod status
    log_info "Pod status:"
    kubectl get pods -n "$NAMESPACE" -o wide
    
    # Check service status
    log_info "Service status:"
    kubectl get services -n "$NAMESPACE"
    
    # Check for any failed pods
    FAILED_PODS=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Failed --no-headers | wc -l)
    if [[ "$FAILED_PODS" -gt 0 ]]; then
        log_warning "Found $FAILED_PODS failed pods"
        kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Failed
    fi
    
    # Check for pending pods
    PENDING_PODS=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Pending --no-headers | wc -l)
    if [[ "$PENDING_PODS" -gt 0 ]]; then
        log_warning "Found $PENDING_PODS pending pods"
        kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Pending
    fi
    
    # Check Helm release status
    RELEASE_STATUS=$(helm status "$RELEASE_NAME" -n "$NAMESPACE" -o json | jq -r '.info.status')
    log_info "Helm release status: $RELEASE_STATUS"
    
    if [[ "$RELEASE_STATUS" == "deployed" ]]; then
        log_success "Deployment verification completed successfully"
    else
        log_warning "Deployment may have issues. Release status: $RELEASE_STATUS"
    fi
}

# Uninstall platform
uninstall_platform() {
    log_info "Uninstalling O-RAN SC Platform..."
    
    if helm list -n "$NAMESPACE" | grep -q "$RELEASE_NAME"; then
        helm uninstall "$RELEASE_NAME" -n "$NAMESPACE"
        log_success "Platform uninstalled successfully"
    else
        log_info "Platform is not installed"
    fi
    
    # Optionally delete namespaces
    read -p "Do you want to delete the namespaces? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kubectl delete namespace "$NAMESPACE" --ignore-not-found
        kubectl delete namespace "$XAPP_NAMESPACE" --ignore-not-found
        log_success "Namespaces deleted"
    fi
}

# Show deployment info
show_deployment_info() {
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        return
    fi
    
    log_info "Deployment Information:"
    echo "=========================="
    echo "Release Name: $RELEASE_NAME"
    echo "Namespace: $NAMESPACE"
    echo "Chart Path: $CHART_PATH"
    echo "Values File: $VALUES_FILE"
    if [[ -n "$OVERRIDE_FILE" ]]; then
        echo "Override File: $OVERRIDE_FILE"
    fi
    echo ""
    
    log_info "Access Information:"
    echo "==================="
    
    # E2 Interface
    E2_NODEPORT=$(kubectl get service -n "$NAMESPACE" -l app=e2term -o jsonpath='{.items[0].spec.ports[?(@.name=="sctp")].nodePort}')
    if [[ -n "$E2_NODEPORT" ]]; then
        echo "E2 Interface (SCTP): NodePort $E2_NODEPORT"
    fi
    
    # A1 Interface
    A1_PORT=$(kubectl get service -n "$NAMESPACE" -l app=a1mediator -o jsonpath='{.items[0].spec.ports[?(@.name=="http")].port}')
    if [[ -n "$A1_PORT" ]]; then
        echo "A1 Interface (HTTP): Port $A1_PORT"
        echo "  Access via: kubectl port-forward -n $NAMESPACE svc/service-ricplt-a1mediator-http $A1_PORT:$A1_PORT"
    fi
    
    # O1 Interface
    O1_SSH_PORT=$(kubectl get service -n "$NAMESPACE" -l app=o1mediator -o jsonpath='{.items[0].spec.ports[?(@.name=="netconf-ssh")].port}')
    if [[ -n "$O1_SSH_PORT" ]]; then
        echo "O1 Interface (NETCONF/SSH): Port $O1_SSH_PORT"
        echo "  Access via: kubectl port-forward -n $NAMESPACE svc/service-ricplt-o1mediator $O1_SSH_PORT:$O1_SSH_PORT"
    fi
    
    echo ""
    log_info "Useful Commands:"
    echo "================"
    echo "View pods: kubectl get pods -n $NAMESPACE"
    echo "View services: kubectl get services -n $NAMESPACE"
    echo "View logs: kubectl logs -n $NAMESPACE -l app=<component-name>"
    echo "Port forward A1: kubectl port-forward -n $NAMESPACE svc/service-ricplt-a1mediator-http 10000:10000"
    echo "Helm status: helm status $RELEASE_NAME -n $NAMESPACE"
}

# Main function
main() {
    log_info "O-RAN SC Platform Foundation Deployment"
    log_info "========================================"
    
    # Parse arguments
    parse_args "$@"
    
    # Handle uninstall
    if [[ "${UNINSTALL:-false}" == "true" ]]; then
        uninstall_platform
        exit 0
    fi
    
    # Execute deployment steps
    check_prerequisites
    create_namespaces
    update_dependencies
    validate_chart
    deploy_platform
    wait_for_deployment
    verify_deployment
    show_deployment_info
    
    log_success "O-RAN SC Platform Foundation deployment completed!"
}

# Execute main function with all arguments
main "$@"