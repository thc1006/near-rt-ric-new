#!/bin/bash
set -e

# Configuration Backup Script for O-RAN Components

BACKUP_DIR="/opt/oran/config-backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
NAMESPACE="oran-config"

# Create backup directory if not exists
mkdir -p "${BACKUP_DIR}/${TIMESTAMP}"

# Backup ConfigMaps
kubectl get configmaps -n "${NAMESPACE}" -o yaml > "${BACKUP_DIR}/${TIMESTAMP}/configmaps.yaml"

# Backup Secrets (exclude sensitive data)
kubectl get secrets -n "${NAMESPACE}" -o json | \
  jq 'del(.items[].data)' > "${BACKUP_DIR}/${TIMESTAMP}/secrets.json"

# Create tarball
tar -czvf "${BACKUP_DIR}/oran-config-${TIMESTAMP}.tar.gz" "${BACKUP_DIR}/${TIMESTAMP}"

# Optional: Prune old backups (keep last 5)
ls -dt "${BACKUP_DIR}"/* | tail -n +6 | xargs rm -rf

echo "Configuration backup completed: ${BACKUP_DIR}/oran-config-${TIMESTAMP}.tar.gz"