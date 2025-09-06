#!/bin/bash
# O-RAN RIC L Release Deployment Script
# Version: 1.0.0
# Deployment Mode: Development/Testing

set -euo pipefail

# Logging and Error Handling
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

error_handler() {
    log "ERROR: Deployment failed at line $1"
    exit 1
}

trap 'error_handler $LINENO' ERR

# Pre-deployment Validation
validate_environment() {
    log "Validating deployment environment..."
    
    # Check required tools
    for tool in kubectl helm kpt; do
        if ! command -v $tool &> /dev/null; then
            log "❌ Required tool $tool not found"
            return 1
        fi
    done
    
    # Verify Kubernetes cluster
    if ! kubectl cluster-info &> /dev/null; then
        log "❌ Kubernetes cluster not accessible"
        return 1
    fi
    
    # Check cluster resources
    local available_cpu=$(kubectl describe nodes | grep "cpu:" | awk '{print $2}' | head -1)
    local available_memory=$(kubectl describe nodes | grep "memory:" | awk '{print $2}' | head -1)
    
    log "Cluster Resources: CPU: $available_cpu, Memory: $available_memory"
    
    # Minimum resource check for RIC deployment
    if (( $(echo "$available_cpu < 8" | bc -l) )) || (( $(echo "$available_memory < 32Gi" | bc -l) )); then
        log "⚠️ Low cluster resources might impact RIC deployment"
    fi
    
    return 0
}

# Deploy Near-RT RIC Platform
deploy_near_rt_ric() {
    local namespace="ricplt"
    
    log "Deploying Near-RT RIC Platform..."
    
    # Create namespace with proper labels
    kubectl create namespace $namespace --dry-run=client -o yaml | \
        kubectl label --local -f - oran.io/component=near-rt-ric -o yaml | \
        kubectl apply -f -
    
    # Helm deployment with enhanced configuration
    helm upgrade --install ric-platform o-ran-sc/ric-platform \
        --namespace $namespace \
        --version 3.0.0 \
        --create-namespace \
        --values - <<EOF
global:
  image:
    registry: nexus3.o-ran-sc.org:10002
    pullPolicy: IfNotPresent

e2term:
  enabled: true
  replicaCount: 2
  service:
    ports:
      sctp: 36421  # Specific port as requested

e2mgr:
  enabled: true
  config:
    ranFunctions:
      - KPM
      - RC
      - CCC

a1mediator:
  enabled: true
  policies:
    defaultAdmissionDelay: 2

submgr:
  enabled: true

rtmgr:
  enabled: true

dbaas:
  enabled: true
  persistence:
    enabled: true
    size: 10Gi
EOF

    # Wait for deployment
    kubectl wait --for=condition=Ready pods --all -n $namespace --timeout=300s
    
    log "Near-RT RIC Platform deployed successfully"
}

# Deploy Non-RT RIC (SMO)
deploy_non_rt_ric() {
    local namespace="nonrtric"
    
    log "Deploying Non-RT RIC (SMO)..."
    
    kubectl create namespace $namespace --dry-run=client -o yaml | kubectl apply -f -
    
    helm upgrade --install nonrtric o-ran-sc/nonrtric \
        --namespace $namespace \
        --version 2.5.0 \
        --values - <<EOF
policymanagementservice:
  enabled: true
  replicaCount: 2

rappcatalogue:
  enabled: true
  
nonrtricgateway:
  enabled: true

controlpanel:
  enabled: true
  ingress:
    enabled: false
EOF

    log "Non-RT RIC (SMO) deployed successfully"
}

# Configure A1 Policies
configure_a1_policies() {
    log "Configuring A1 Policies..."
    
    # Create policy types for development environment
    for policy_type in QoS TrafficSteering Slicing; do
        curl -X PUT http://a1mediator.ricplt:8080/A1-P/v2/policytypes/${policy_type} \
            -H "Content-Type: application/json" \
            -d "{
                \"name\": \"${policy_type}Policy\",
                \"description\": \"Development Policy for ${policy_type}\",
                \"policy_type_id\": \"${policy_type}\",
                \"create_schema\": {}
            }"
    done
    
    log "A1 Policies configured"
}

# Setup xApp Development Environment
setup_xapp_dev_env() {
    log "Setting up xApp Development Environment..."
    
    # Create xApp development namespace
    kubectl create namespace ricxapp --dry-run=client -o yaml | kubectl apply -f -
    
    # Create template for xApp development
    mkdir -p /tmp/xapp-template
    cat > /tmp/xapp-template/xapp-template.yaml <<'YAML'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: xapp-template
  namespace: ricxapp
spec:
  replicas: 1
  selector:
    matchLabels:
      xapp: template
  template:
    metadata:
      labels:
        xapp: template
    spec:
      containers:
      - name: xapp
        image: nexus3.o-ran-sc.org:10002/o-ran-sc/ric-app-template:1.0.0
        env:
        - name: RMR_SERVICE_NAME
          value: "ric-e2term.ricplt"
YAML
    
    log "xApp development template created"
}

# Validate Deployment
validate_deployment() {
    log "Validating RIC Platform Deployment..."
    
    # Check Near-RT RIC components
    kubectl get pods -n ricplt
    
    # Check Non-RT RIC components
    kubectl get pods -n nonrtric
    
    # Verify E2 and A1 interfaces
    kubectl exec -n ricplt deployment/e2mgr -- e2mgr-cli get nodes || true
    curl -s http://a1mediator.ricplt:8080/A1-P/v2/policytypes || true
    
    log "Deployment Validation Complete"
}

# Main Deployment Workflow
main() {
    validate_environment
    deploy_near_rt_ric
    deploy_non_rt_ric
    configure_a1_policies
    setup_xapp_dev_env
    validate_deployment
    
    log "O-RAN RIC L Release Development Environment Deployed Successfully!"
}

# Execute Main Workflow
main