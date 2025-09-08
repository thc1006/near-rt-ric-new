#!/bin/bash
# Configuration Drift Detection Script

set -e

NAMESPACE="oran-config"
BASELINE_CONFIG="/opt/oran/baseline-config.json"
CURRENT_CONFIG="/tmp/current-config.json"

# Extract current configuration
kubectl get configmaps,secrets -n "${NAMESPACE}" -o json | \
  jq 'del(.items[].metadata.resourceVersion, 
          .items[].metadata.uid, 
          .items[].metadata.creationTimestamp)' > "${CURRENT_CONFIG}"

# Compare with baseline
if ! diff -q "${BASELINE_CONFIG}" "${CURRENT_CONFIG}"; then
  echo "CONFIGURATION DRIFT DETECTED!"
  diff "${BASELINE_CONFIG}" "${CURRENT_CONFIG}"
  
  # Optional: Trigger automatic remediation
  # kubectl apply -f /opt/oran/config-remediation.yaml
else
  echo "Configuration is consistent with baseline."
fi