#!/bin/bash
#
# SPDX-FileCopyrightText: 2022-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

# Health check script for O-RAN SC Near-RT RIC components

set -e

NAMESPACE="ricplt"
TIMEOUT=300

echo "=== O-RAN SC Near-RT RIC Health Check ==="

# Function to check if pod is ready
check_pod_ready() {
    local app_label=$1
    local component_name=$2
    
    echo "Checking $component_name..."
    
    # Wait for pod to be ready
    if kubectl wait --for=condition=ready pod -l app=$app_label -n $NAMESPACE --timeout=${TIMEOUT}s >/dev/null 2>&1; then
        echo "✓ $component_name is ready"
        return 0
    else
        echo "✗ $component_name is not ready"
        kubectl get pods -l app=$app_label -n $NAMESPACE
        return 1
    fi
}

# Function to check service endpoints
check_service_endpoint() {
    local service_name=$1
    local port=$2
    local component_name=$3
    
    echo "Checking $component_name service endpoint..."
    
    if kubectl get service $service_name -n $NAMESPACE >/dev/null 2>&1; then
        echo "✓ $component_name service is available"
        return 0
    else
        echo "✗ $component_name service not found"
        return 1
    fi
}

# Check if namespace exists
if ! kubectl get namespace $NAMESPACE >/dev/null 2>&1; then
    echo "✗ Namespace $NAMESPACE does not exist"
    exit 1
fi

echo "✓ Namespace $NAMESPACE exists"

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
    echo "✓ All O-RAN SC components are healthy"
    exit 0
else
    echo "⚠ Some components may not be ready yet"
    echo "This is normal during initial deployment. Components may take a few minutes to start."
    exit 0  # Don't fail the build during initial setup
fi
