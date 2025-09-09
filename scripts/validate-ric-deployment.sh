#!/bin/bash
set -euo pipefail

# O-RAN SC Near-RT RIC Deployment Validation Script

NAMESPACE="ricplt"

# Check Namespace
if ! kubectl get namespace "${NAMESPACE}" &> /dev/null; then
    echo "Error: RIC Platform namespace not found"
    exit 1
fi

# Validate Core Components
required_components=(
    "e2term"
    "e2mgr"
    "a1mediator"
    "submgr"
    "rtmgr"
    "dbaas"
)

for component in "${required_components[@]}"; do
    if ! kubectl get deployment "${component}" -n "${NAMESPACE}" &> /dev/null; then
        echo "Error: ${component} not deployed"
        exit 1
    fi
done

# Check Pod Status
pod_status=$(kubectl get pods -n "${NAMESPACE}" -o jsonpath='{.items[*].status.phase}')
if [[ "$pod_status" == *"Pending"* || "$pod_status" == *"Failed"* ]]; then
    echo "Warning: Some pods are not in Running state"
    kubectl get pods -n "${NAMESPACE}"
    exit 1
fi

# Interface Validation
interfaces=("E2" "A1" "O1")
for interface in "${interfaces[@]}"; do
    if ! kubectl describe configmap "${interface,,}-interface-config" -n "${NAMESPACE}" &> /dev/null; then
        echo "Warning: ${interface} interface configuration missing"
    fi
done

# Dashboard Accessibility Check
dashboard_service=$(kubectl get service -n "${NAMESPACE}" -l app=dashboard -o name)
if [ -z "$dashboard_service" ]; then
    echo "Error: Dashboard service not found"
    exit 1
fi

echo "✅ O-RAN SC Near-RT RIC Deployment Validated Successfully"
exit 0