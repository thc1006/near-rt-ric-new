#!/bin/bash
set -euo pipefail

# O-RAN SC Near-RT RIC L Release Deployment Script
# September 2025 Deployment Standards

# Prerequisite Checks
command -v kubectl >/dev/null 2>&1 || { echo "kubectl is not installed"; exit 1; }
command -v helm >/dev/null 2>&1 || { echo "helm is not installed"; exit 1; }

# Configuration Variables
NAMESPACE="ricplt"
RIC_PLATFORM_VERSION="3.0.0"
DASHBOARD_VERSION="1.2.0"

# Create Namespace
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | \
    kubectl apply -f -

# Deploy RIC Platform Core Components
helm upgrade --install ric-platform ./helm/ric-platform \
    --namespace "${NAMESPACE}" \
    --version "${RIC_PLATFORM_VERSION}" \
    --values ./configs/component-configs.yaml \
    --set global.image.pullPolicy=Always \
    --wait \
    --timeout 15m

# Deploy Dashboard
helm upgrade --install ric-dashboard ./helm/dashboard-api \
    --namespace "${NAMESPACE}" \
    --version "${DASHBOARD_VERSION}" \
    --set ingress.enabled=true \
    --set service.type=LoadBalancer

# Configure Interfaces
kubectl apply -f ./configs/e2-service-models-enhanced.yaml -n "${NAMESPACE}"
kubectl apply -f ./configs/rmr_routes.yaml -n "${NAMESPACE}"

# Verify Deployments
kubectl get pods -n "${NAMESPACE}"
kubectl get services -n "${NAMESPACE}"

# Expose Dashboard
dashboard_port=$(kubectl get service -n "${NAMESPACE}" ric-dashboard -o jsonpath='{.spec.ports[0].nodePort}')
echo "RIC Dashboard accessible at: http://localhost:${dashboard_port}"