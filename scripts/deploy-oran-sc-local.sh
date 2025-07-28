#!/bin/bash

# Deploy O-RAN SC Near-RT RIC for Local Development
# This script deploys O-RAN SC components in local Kubernetes environments

set -e

NAMESPACE="ricplt"
HELM_REPO_NAME="oran-sc"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=== O-RAN SC Near-RT RIC Local Deployment ==="

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
echo "Checking prerequisites..."
if ! command_exists kubectl; then
    echo "Error: kubectl is not installed"
    exit 1
fi

if ! command_exists helm; then
    echo "Error: helm is not installed"
    exit 1
fi

# Check Kubernetes cluster connectivity
if ! kubectl cluster-info >/dev/null 2>&1; then
    echo "Error: Cannot connect to Kubernetes cluster"
    echo "Please ensure your Kubernetes cluster (KIND/K3s/Minikube) is running"
    exit 1
fi

echo "✓ Prerequisites check passed"

# Create namespace
echo "Creating namespace: $NAMESPACE"
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Add O-RAN SC Helm repository (if not already added)
echo "Setting up Helm repositories..."
if ! helm repo list | grep -q "$HELM_REPO_NAME"; then
    echo "Adding O-RAN SC Helm repository..."
    # Note: Using local ric-dep charts for now
    echo "Using local O-RAN SC charts from ric-dep/"
else
    echo "✓ O-RAN SC Helm repository already configured"
fi

# Update Helm repositories
helm repo update || true

# Deploy infrastructure components first
echo "Deploying infrastructure components..."

# Deploy Redis (Database as a Service)
echo "Deploying Redis (DBAAS)..."
helm upgrade --install dbaas \
    "$PROJECT_ROOT/ric-dep/helm/dbaas" \
    --namespace $NAMESPACE \
    --set image.registry="docker.io" \
    --set image.name="redis" \
    --set image.tag="6.0.8-alpine" \
    --set persistence.enabled=false \
    --wait --timeout=300s

# Deploy E2 Manager
echo "Deploying E2 Manager..."
helm upgrade --install e2mgr \
    "$PROJECT_ROOT/ric-dep/helm/e2mgr" \
    --namespace $NAMESPACE \
    --set e2mgr.image.registry="nexus3.o-ran-sc.org:10002/o-ran-sc" \
    --set e2mgr.globalRicId.plmnId=131014 \
    --set e2mgr.globalRicId.ricNearRtId=556670 \
    --wait --timeout=300s

# Deploy Subscription Manager
echo "Deploying Subscription Manager..."
helm upgrade --install submgr \
    "$PROJECT_ROOT/ric-dep/helm/submgr" \
    --namespace $NAMESPACE \
    --set submgr.image.registry="nexus3.o-ran-sc.org:10002/o-ran-sc" \
    --wait --timeout=300s

# Deploy Routing Manager
echo "Deploying Routing Manager..."
helm upgrade --install rtmgr \
    "$PROJECT_ROOT/ric-dep/helm/rtmgr" \
    --namespace $NAMESPACE \
    --set rtmgr.image.registry="nexus3.o-ran-sc.org:10002/o-ran-sc" \
    --wait --timeout=300s

# Deploy App Manager
echo "Deploying App Manager..."
helm upgrade --install appmgr \
    "$PROJECT_ROOT/ric-dep/helm/appmgr" \
    --namespace $NAMESPACE \
    --set appmgr.image.appmgr.registry="nexus3.o-ran-sc.org:10002/o-ran-sc" \
    --set appmgr.image.init.registry="nexus3.o-ran-sc.org:10002/o-ran-sc" \
    --wait --timeout=300s

# Deploy A1 Mediator
echo "Deploying A1 Mediator..."
helm upgrade --install a1mediator \
    "$PROJECT_ROOT/ric-dep/helm/a1mediator" \
    --namespace $NAMESPACE \
    --set a1mediator.image.registry="nexus3.o-ran-sc.org:10002/o-ran-sc" \
    --wait --timeout=300s

# Deploy E2 Termination
echo "Deploying E2 Termination..."
helm upgrade --install e2term \
    "$PROJECT_ROOT/ric-dep/helm/e2term" \
    --namespace $NAMESPACE \
    --set e2term.image.registry="nexus3.o-ran-sc.org:10002/o-ran-sc" \
    --wait --timeout=300s

echo "=== Deployment Status ==="
kubectl get pods -n $NAMESPACE

echo ""
echo "=== Service Status ==="
kubectl get services -n $NAMESPACE

echo ""
echo "=== O-RAN SC Near-RT RIC Deployment Complete ==="
echo "Namespace: $NAMESPACE"
echo ""
echo "To check component status:"
echo "  kubectl get pods -n $NAMESPACE"
echo ""
echo "To access logs:"
echo "  kubectl logs -n $NAMESPACE <pod-name>"
echo ""
echo "To port-forward services for local access:"
echo "  kubectl port-forward -n $NAMESPACE service/e2mgr-http 3800:3800"
echo "  kubectl port-forward -n $NAMESPACE service/a1mediator-http 10000:10000"