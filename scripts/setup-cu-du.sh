#!/bin/bash

# O-RAN CU/DU Network Functions Setup Script
# This script sets up the complete CU/DU deployment with F1/E1 interfaces

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-oran-network-functions}"
HELM_RELEASE_NAME="${HELM_RELEASE_NAME:-cu-du-nf}"
VALUES_FILE="${VALUES_FILE:-values.yaml}"
DRY_RUN="${DRY_RUN:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check required tools
    local required_tools=("kubectl" "helm" "jq" "yq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    # Check Kubernetes connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check SR-IOV support (optional)
    if kubectl get crd network-attachment-definitions.k8s.cni.cncf.io &> /dev/null; then
        log_info "SR-IOV NetworkAttachmentDefinition CRD found"
    else
        log_warn "SR-IOV NetworkAttachmentDefinition CRD not found (optional for production)"
    fi
    
    log_info "Prerequisites check completed successfully"
}

# Validate CU/DU environment
validate_environment() {
    log_info "Validating CU/DU deployment environment..."
    
    # Check node labels for CU/DU scheduling
    local cu_nodes=$(kubectl get nodes -l node-role.kubernetes.io/cu=true --no-headers 2>/dev/null | wc -l)
    local du_nodes=$(kubectl get nodes -l node-role.kubernetes.io/du=true --no-headers 2>/dev/null | wc -l)
    
    if [ "$cu_nodes" -eq 0 ]; then
        log_warn "No nodes labeled for CU workloads (node-role.kubernetes.io/cu=true)"
    else
        log_info "Found $cu_nodes node(s) for CU workloads"
    fi
    
    if [ "$du_nodes" -eq 0 ]; then
        log_warn "No nodes labeled for DU workloads (node-role.kubernetes.io/du=true)"
    else
        log_info "Found $du_nodes node(s) for DU workloads"
    fi
    
    # Check for Intel SR-IOV nodes
    local sriov_nodes=$(kubectl get nodes -l intel.feature.node.kubernetes.io/network-sriov=true --no-headers 2>/dev/null | wc -l)
    if [ "$sriov_nodes" -eq 0 ]; then
        log_warn "No SR-IOV capable nodes found (recommended for production)"
    else
        log_info "Found $sriov_nodes SR-IOV capable node(s)"
    fi
    
    log_info "Environment validation completed"
}

# Create namespace and resources
create_namespace() {
    log_info "Creating namespace and basic resources..."
    
    # Create namespace
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Label namespace
    kubectl label namespace "$NAMESPACE" name="$NAMESPACE" --overwrite
    
    # Create service accounts
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cu-cp-sa
  namespace: $NAMESPACE
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cu-up-sa
  namespace: $NAMESPACE
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: du-sa
  namespace: $NAMESPACE
EOF

    log_info "Namespace and basic resources created"
}

# Setup network configurations
setup_network_config() {
    log_info "Setting up network configurations..."
    
    # Apply fronthaul network configuration
    if [ -f "../configs/fronthaul-radio-config.yaml" ]; then
        kubectl apply -f ../configs/fronthaul-radio-config.yaml
        log_info "Fronthaul network configuration applied"
    else
        log_warn "Fronthaul configuration file not found"
    fi
    
    # Apply F1 interface configuration
    if [ -f "../configs/f1-interface-config.yaml" ]; then
        kubectl apply -f ../configs/f1-interface-config.yaml
        log_info "F1 interface configuration applied"
    else
        log_warn "F1 interface configuration file not found"
    fi
    
    # Apply E1 interface configuration
    if [ -f "../configs/e1-interface-config.yaml" ]; then
        kubectl apply -f ../configs/e1-interface-config.yaml
        log_info "E1 interface configuration applied"
    else
        log_warn "E1 interface configuration file not found"
    fi
    
    log_info "Network configurations setup completed"
}

# Deploy E2 service models
deploy_e2_service_models() {
    log_info "Deploying E2 service models..."
    
    if [ -f "../configs/e2-service-models-enhanced.yaml" ]; then
        kubectl apply -f ../configs/e2-service-models-enhanced.yaml
        log_info "E2 service models deployed"
    else
        log_warn "E2 service models configuration file not found"
    fi
}

# Setup performance optimization
setup_performance_optimization() {
    log_info "Setting up performance optimization..."
    
    if [ -f "../configs/performance-monitoring.yaml" ]; then
        kubectl apply -f ../configs/performance-monitoring.yaml
        log_info "Performance monitoring configuration applied"
    else
        log_warn "Performance monitoring configuration file not found"
    fi
    
    # Create performance tuning daemon set
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: performance-tuning
  namespace: $NAMESPACE
spec:
  selector:
    matchLabels:
      name: performance-tuning
  template:
    metadata:
      labels:
        name: performance-tuning
    spec:
      hostPID: true
      hostIPC: true
      containers:
      - name: performance-tuning
        image: busybox
        command: 
        - /bin/sh
        - -c
        - |
          # Enable performance mode
          echo performance > /host/sys/devices/system/cpu/cpu*/cpufreq/scaling_governor 2>/dev/null || true
          # Configure hugepages
          echo 1024 > /host/sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages 2>/dev/null || true
          # Sleep to keep container running
          sleep infinity
        volumeMounts:
        - name: host-sys
          mountPath: /host/sys
        - name: host-proc
          mountPath: /host/proc
        securityContext:
          privileged: true
      volumes:
      - name: host-sys
        hostPath:
          path: /sys
      - name: host-proc
        hostPath:
          path: /proc
      tolerations:
      - operator: Exists
EOF

    log_info "Performance optimization setup completed"
}

# Deploy CU/DU network functions
deploy_cu_du() {
    log_info "Deploying CU/DU network functions..."
    
    # Check if we have Helm chart
    if [ -d "../helm/cu-du-network-functions" ]; then
        log_info "Using Helm chart for deployment..."
        
        # Add Helm dependencies
        helm dependency update ../helm/cu-du-network-functions/
        
        # Deploy using Helm
        local helm_cmd="helm upgrade --install $HELM_RELEASE_NAME ../helm/cu-du-network-functions/ \
            --namespace $NAMESPACE \
            --create-namespace"
            
        if [ -f "$VALUES_FILE" ]; then
            helm_cmd="$helm_cmd --values $VALUES_FILE"
        fi
        
        if [ "$DRY_RUN" = "true" ]; then
            helm_cmd="$helm_cmd --dry-run"
        fi
        
        eval $helm_cmd
        
    else
        log_info "Using individual YAML manifests..."
        
        # Deploy individual components
        for config_file in \
            "../configs/cu-up-enhanced.yaml" \
            "../configs/du-enhanced.yaml"; do
            
            if [ -f "$config_file" ]; then
                if [ "$DRY_RUN" = "true" ]; then
                    kubectl apply -f "$config_file" --dry-run=client
                else
                    kubectl apply -f "$config_file"
                fi
                log_info "Applied $(basename $config_file)"
            else
                log_warn "Configuration file $config_file not found"
            fi
        done
    fi
    
    log_info "CU/DU network functions deployment completed"
}

# Wait for deployments to be ready
wait_for_deployments() {
    log_info "Waiting for deployments to be ready..."
    
    local deployments=("cu-cp" "cu-up" "du-1" "du-2")
    
    for deployment in "${deployments[@]}"; do
        log_info "Waiting for deployment/$deployment..."
        if kubectl wait --for=condition=available --timeout=300s deployment/$deployment -n $NAMESPACE 2>/dev/null; then
            log_info "Deployment $deployment is ready"
        else
            log_warn "Deployment $deployment is not ready within timeout"
        fi
    done
}

# Verify deployment
verify_deployment() {
    log_info "Verifying CU/DU deployment..."
    
    # Check pod status
    log_info "Pod status:"
    kubectl get pods -n $NAMESPACE -o wide
    
    # Check services
    log_info "Services:"
    kubectl get services -n $NAMESPACE
    
    # Check F1 interface connectivity
    log_info "Checking F1 interface connectivity..."
    local cu_cp_pod=$(kubectl get pods -n $NAMESPACE -l app=cu-cp -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$cu_cp_pod" ]; then
        if kubectl exec -n $NAMESPACE $cu_cp_pod -- netstat -tuln | grep -q :38472; then
            log_info "F1 interface port is listening"
        else
            log_warn "F1 interface port is not listening"
        fi
    fi
    
    # Check E1 interface connectivity
    log_info "Checking E1 interface connectivity..."
    if [ -n "$cu_cp_pod" ]; then
        if kubectl exec -n $NAMESPACE $cu_cp_pod -- netstat -tuln | grep -q :38462; then
            log_info "E1 interface port is listening"
        else
            log_warn "E1 interface port is not listening"
        fi
    fi
    
    # Check E2 interface connectivity
    log_info "Checking E2 interface connectivity..."
    local du_pods=$(kubectl get pods -n $NAMESPACE -l app=du -o jsonpath='{.items[*].metadata.name}')
    for pod in $du_pods; do
        if kubectl exec -n $NAMESPACE $pod -- netstat -tuln | grep -q :36421; then
            log_info "E2 interface port is listening on $pod"
        else
            log_warn "E2 interface port is not listening on $pod"
        fi
    done
    
    log_info "Deployment verification completed"
}

# Display status
display_status() {
    log_info "CU/DU Deployment Status Summary"
    echo "================================="
    
    # Deployment status
    echo "Deployments:"
    kubectl get deployments -n $NAMESPACE -o custom-columns=NAME:.metadata.name,READY:.status.readyReplicas,UP-TO-DATE:.status.updatedReplicas,AVAILABLE:.status.availableReplicas
    
    # Service endpoints
    echo -e "\nService Endpoints:"
    kubectl get services -n $NAMESPACE -o custom-columns=NAME:.metadata.name,TYPE:.spec.type,CLUSTER-IP:.spec.clusterIP,PORTS:.spec.ports[*].port
    
    # Resource usage
    echo -e "\nResource Usage:"
    kubectl top pods -n $NAMESPACE --no-headers 2>/dev/null || echo "Metrics server not available"
}

# Cleanup function
cleanup() {
    if [ "${1:-}" = "--force" ]; then
        log_info "Performing cleanup..."
        kubectl delete namespace $NAMESPACE --ignore-not-found=true
        helm uninstall $HELM_RELEASE_NAME -n $NAMESPACE 2>/dev/null || true
        log_info "Cleanup completed"
    else
        echo "To cleanup the deployment, run: $0 cleanup --force"
    fi
}

# Main execution
main() {
    case "${1:-}" in
        "cleanup")
            cleanup "${2:-}"
            ;;
        "verify")
            verify_deployment
            display_status
            ;;
        "status")
            display_status
            ;;
        *)
            log_info "Starting O-RAN CU/DU Network Functions setup..."
            
            check_prerequisites
            validate_environment
            create_namespace
            setup_network_config
            deploy_e2_service_models
            setup_performance_optimization
            
            if [ "$DRY_RUN" != "true" ]; then
                deploy_cu_du
                wait_for_deployments
                verify_deployment
            else
                log_info "Dry run mode - skipping actual deployment"
                deploy_cu_du
            fi
            
            display_status
            
            log_info "CU/DU Network Functions setup completed successfully!"
            log_info "Use '$0 verify' to verify the deployment"
            log_info "Use '$0 status' to check current status"
            log_info "Use '$0 cleanup --force' to remove the deployment"
            ;;
    esac
}

# Execute main function with all arguments
main "$@"