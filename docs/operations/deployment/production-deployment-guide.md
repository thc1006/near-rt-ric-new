# Production Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying the O-RAN Near-RT RIC platform in production environments with security hardening, resource optimization, and operational best practices.

## Prerequisites

### Infrastructure Requirements

#### Minimum Production Requirements
- **Kubernetes Version**: 1.25+
- **Nodes**: 3+ worker nodes
- **CPU**: 16 cores total
- **Memory**: 32 GB RAM total
- **Storage**: 500 GB SSD with high IOPS
- **Network**: 10 Gbps connectivity

#### Recommended Production Requirements
- **Kubernetes Version**: 1.28+
- **Nodes**: 5+ worker nodes (with node affinity)
- **CPU**: 32 cores total
- **Memory**: 64 GB RAM total
- **Storage**: 1 TB NVMe SSD
- **Network**: 25 Gbps connectivity

#### Required Kubernetes Features
- **Container Runtime**: containerd or CRI-O
- **CNI**: Calico, Cilium, or equivalent with NetworkPolicy support
- **CSI**: Storage driver with dynamic provisioning
- **Metrics Server**: For HPA and resource monitoring
- **Ingress Controller**: NGINX, Traefik, or equivalent

### Software Dependencies

#### Required Tools
```bash
# Helm 3.11+
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# kubectl (matching cluster version)
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# Optional: Argo Rollouts (for blue-green deployments)
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml
```

#### Container Registry Access
Ensure access to O-RAN SC container registry:
```bash
# Test registry access
docker pull nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-e2:4.4.6
```

## Pre-Deployment Checklist

### Security Preparation

#### 1. Certificate Management
```bash
# Generate CA certificate for the platform
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 365 -key ca-key.pem -out ca-cert.pem \
    -subj "/C=US/ST=CA/L=San Francisco/O=O-RAN SC/CN=oran-ric-ca"

# Create Kubernetes secret
kubectl create secret tls oran-sc-ca-cert \
    --cert=ca-cert.pem \
    --key=ca-key.pem \
    -n oran-ric
```

#### 2. JWT Keys for Authentication
```bash
# Generate JWT signing keys
openssl genrsa -out jwt-private.pem 2048
openssl rsa -in jwt-private.pem -pubout -out jwt-public.pem

# Create production secrets
kubectl create secret generic production-secrets \
    --from-file=jwt-private-key=jwt-private.pem \
    --from-file=jwt-public-key=jwt-public.pem \
    --from-literal=database-password="$(openssl rand -base64 32)" \
    --from-literal=monitoring-api-key="$(openssl rand -base64 32)" \
    --from-literal=encryption-key="$(openssl rand -base64 32)" \
    -n oran-ric
```

#### 3. Network Security
```bash
# Create namespace with security labels
kubectl create namespace oran-ric
kubectl label namespace oran-ric \
    pod-security.kubernetes.io/enforce=restricted \
    pod-security.kubernetes.io/audit=restricted \
    pod-security.kubernetes.io/warn=restricted
```

### Resource Planning

#### 1. Storage Classes
```yaml
# production-storage-class.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: oran-ric-ssd
provisioner: kubernetes.io/aws-ebs  # Adjust for your cloud provider
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
  encrypted: "true"
allowVolumeExpansion: true
reclaimPolicy: Retain
volumeBindingMode: WaitForFirstConsumer
```

#### 2. Node Labeling
```bash
# Label nodes for component placement
kubectl label nodes node-1 oran-ric/component=core
kubectl label nodes node-2 oran-ric/component=core
kubectl label nodes node-3 oran-ric/component=interface
kubectl label nodes node-4 oran-ric/component=observability
kubectl label nodes node-5 oran-ric/component=xapp
```

## Production Deployment

### Step 1: Prepare Production Values

Create a production values file:

```yaml
# production-values.yaml
global:
  namespace: oran-ric

# Enable production mode
production:
  enabled: true
  
  logging:
    level: "warn"  # Reduce log verbosity in production
  
  security:
    certRotationDays: 30
    jwtExpiry: "1h"
    jwtRefreshExpiry: "24h"
    rateLimitRpm: 5000  # Higher limits for production
    rateLimitBurst: 500
  
  performance:
    dbMaxConnections: 200
    dbIdleTimeout: "10m"
    dbMaxLifetime: "2h"
    httpMaxIdleConns: 200
    httpIdleTimeout: "120s"
    httpResponseTimeout: "30s"
    cacheTtl: "10m"
    cacheMaxSize: "500MB"
    metricsInterval: "15s"
    detailedMetrics: true
  
  monitoring:
    healthCheckInterval: "15s"
    healthCheckTimeout: "5s"
    healthCheckFailureThreshold: 3
    alertWebhook: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
    tracingEnabled: true
    tracingSamplingRate: 0.05  # 5% sampling for production
  
  resources:
    defaultCpuLimit: "2000m"
    defaultMemoryLimit: "4Gi"
    defaultCpuRequest: "500m"
    defaultMemoryRequest: "1Gi"
    maxCpuLimit: "8000m"
    maxMemoryLimit: "16Gi"
  
  resourceQuota:
    requestsCpu: "20"
    requestsMemory: "40Gi"
    limitsCpu: "40"
    limitsMemory: "80Gi"
    pvcCount: "20"
    requestsStorage: "500Gi"

# Component-specific production settings
e2term:
  replicaCount: 2
  resources:
    limits:
      cpu: "4000m"
      memory: "8Gi"
    requests:
      cpu: "1000m"
      memory: "2Gi"
  nodeSelector:
    oran-ric/component: core
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - e2term
        topologyKey: kubernetes.io/hostname

e2mgr:
  replicaCount: 2
  resources:
    limits:
      cpu: "2000m"
      memory: "4Gi"
    requests:
      cpu: "500m"
      memory: "1Gi"
  nodeSelector:
    oran-ric/component: core

submgr:
  replicaCount: 3
  resources:
    limits:
      cpu: "2000m"
      memory: "4Gi"
    requests:
      cpu: "500m"
      memory: "1Gi"
  nodeSelector:
    oran-ric/component: core

a1mediator:
  replicaCount: 2
  resources:
    limits:
      cpu: "1000m"
      memory: "2Gi"
    requests:
      cpu: "250m"
      memory: "512Mi"
  nodeSelector:
    oran-ric/component: interface

o1mediator:
  replicaCount: 2
  resources:
    limits:
      cpu: "1000m"
      memory: "2Gi"
    requests:
      cpu: "250m"
      memory: "512Mi"
  nodeSelector:
    oran-ric/component: interface

dbaas:
  replicaCount: 1  # Redis single instance with persistence
  resources:
    limits:
      cpu: "2000m"
      memory: "8Gi"
    requests:
      cpu: "500m"
      memory: "2Gi"
  persistence:
    enabled: true
    size: "100Gi"
    storageClass: "oran-ric-ssd"
  nodeSelector:
    oran-ric/component: core

# Enable backup and disaster recovery
backup:
  enabled: true
  schedule: "0 2 * * *"  # Daily at 2 AM
  storage:
    size: "200Gi"
    storageClass: "oran-ric-ssd"
  s3Bucket: "oran-ric-backups-prod"
  s3Region: "us-west-2"

# Enable deployment validation
validation:
  enabled: true
  integrationTests: true
  performanceTests: true
  securityTests: true

# Observability for production
observability:
  prometheus:
    enabled: true
    retention: "30d"
    storage: "200Gi"
    resources:
      limits:
        cpu: "4000m"
        memory: "8Gi"
      requests:
        cpu: "1000m"
        memory: "4Gi"
  
  grafana:
    enabled: true
    persistence:
      enabled: true
      size: "20Gi"
    resources:
      limits:
        cpu: "1000m"
        memory: "2Gi"
      requests:
        cpu: "250m"
        memory: "512Mi"
  
  loki:
    enabled: true
    retention: "30d"
    storage: "100Gi"
    resources:
      limits:
        cpu: "2000m"
        memory: "4Gi"
      requests:
        cpu: "500m"
        memory: "2Gi"

# Security hardening
security:
  tls:
    enabled: true
  rbac:
    enabled: true

networking:
  networkPolicies:
    enabled: true
    defaultDeny: true
  ingress:
    enabled: true
    className: "nginx"
    annotations:
      nginx.ingress.kubernetes.io/ssl-redirect: "true"
      nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
    hosts:
    - host: oran-ric.yourdomain.com
      paths:
      - path: /
        pathType: Prefix
    tls:
    - secretName: oran-ric-tls
      hosts:
      - oran-ric.yourdomain.com
```

### Step 2: Deploy the Platform

```bash
# Add the O-RAN SC Helm repository
helm repo add oran-sc https://charts.o-ran-sc.org
helm repo update

# Deploy with production values
helm install oran-ric-platform oran-sc/oran-sc-platform \
    --namespace oran-ric \
    --create-namespace \
    --values production-values.yaml \
    --timeout 20m \
    --wait

# Verify deployment
kubectl get pods -n oran-ric
kubectl get services -n oran-ric
kubectl get ingress -n oran-ric
```

### Step 3: Post-Deployment Validation

```bash
# Run deployment validation
kubectl logs -n oran-ric job/deployment-validation -f

# Check component health
kubectl exec -n oran-ric deployment/e2mgr -- curl -f localhost:8080/health
kubectl exec -n oran-ric deployment/submgr -- curl -f localhost:8080/health
kubectl exec -n oran-ric deployment/a1mediator -- curl -f localhost:8080/a1-p/healthcheck

# Verify database connectivity
kubectl exec -n oran-ric deployment/dbaas -- redis-cli ping

# Check metrics collection
kubectl port-forward -n oran-ric svc/prometheus 9090:9090 &
curl http://localhost:9090/api/v1/query?query=up
```

## Blue-Green Deployment (Optional)

For zero-downtime deployments, enable blue-green deployment:

```yaml
# blue-green-values.yaml
blueGreen:
  enabled: true
  autoPromotion: false  # Manual promotion for production
  scaleDownDelay: 300   # 5 minutes
  
  analysis:
    interval: "60s"
    count: 10
    successRate: 0.99
    maxResponseTime: 0.2
    maxErrorRate: 0.01
    failureLimit: 2
```

Deploy with blue-green:
```bash
helm upgrade oran-ric-platform oran-sc/oran-sc-platform \
    --namespace oran-ric \
    --values production-values.yaml \
    --values blue-green-values.yaml \
    --timeout 30m
```

## Monitoring and Alerting Setup

### Configure Prometheus Alerts

```yaml
# production-alerts.yaml
groups:
- name: oran-ric-production
  rules:
  - alert: ComponentDown
    expr: up{job=~"oran-ric.*"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "O-RAN RIC component {{ $labels.instance }} is down"
      description: "Component {{ $labels.instance }} has been down for more than 1 minute"

  - alert: HighCPUUsage
    expr: rate(container_cpu_usage_seconds_total{namespace="oran-ric"}[5m]) * 100 > 80
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High CPU usage on {{ $labels.pod }}"
      description: "CPU usage is {{ $value }}% on pod {{ $labels.pod }}"

  - alert: HighMemoryUsage
    expr: (container_memory_usage_bytes{namespace="oran-ric"} / container_spec_memory_limit_bytes{namespace="oran-ric"}) * 100 > 85
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage on {{ $labels.pod }}"
      description: "Memory usage is {{ $value }}% on pod {{ $labels.pod }}"

  - alert: DatabaseConnectionFailure
    expr: redis_connected_clients{namespace="oran-ric"} == 0
    for: 30s
    labels:
      severity: critical
    annotations:
      summary: "Database connection failure"
      description: "No clients connected to Redis database"

  - alert: E2NodeDisconnection
    expr: e2mgr_connected_nodes < 1
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "All E2 nodes disconnected"
      description: "No E2 nodes are currently connected to the platform"
```

### Set Up Grafana Dashboards

Access Grafana and import production dashboards:
```bash
# Port forward to Grafana
kubectl port-forward -n oran-ric svc/grafana 3000:3000

# Access at http://localhost:3000
# Default credentials: admin/admin123
```

## Backup and Disaster Recovery

### Configure Automated Backups

```bash
# Create S3 bucket for backups (AWS example)
aws s3 mb s3://oran-ric-backups-prod
aws s3api put-bucket-versioning \
    --bucket oran-ric-backups-prod \
    --versioning-configuration Status=Enabled

# Create IAM user and policy for backup access
aws iam create-user --user-name oran-ric-backup
aws iam create-access-key --user-name oran-ric-backup

# Create Kubernetes secret with AWS credentials
kubectl create secret generic backup-secrets \
    --from-literal=aws-access-key-id=YOUR_ACCESS_KEY \
    --from-literal=aws-secret-access-key=YOUR_SECRET_KEY \
    -n oran-ric
```

### Test Disaster Recovery

```bash
# Test backup creation
kubectl create job --from=cronjob/backup-cronjob manual-backup -n oran-ric

# Test restore procedure
kubectl exec -n oran-ric deployment/backup -- /scripts/disaster-recovery.sh database-failure
```

## Security Hardening

### Enable Pod Security Standards

```bash
# Apply pod security standards
kubectl label namespace oran-ric \
    pod-security.kubernetes.io/enforce=restricted \
    pod-security.kubernetes.io/audit=restricted \
    pod-security.kubernetes.io/warn=restricted
```

### Configure Network Policies

```yaml
# strict-network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: oran-ric
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-oran-ric-internal
  namespace: oran-ric
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/part-of: oran-ric
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app.kubernetes.io/part-of: oran-ric
  egress:
  - to:
    - podSelector:
        matchLabels:
          app.kubernetes.io/part-of: oran-ric
  - to: []  # Allow DNS
    ports:
    - protocol: UDP
      port: 53
```

### Enable Audit Logging

```yaml
# audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
  namespaces: ["oran-ric"]
  resources:
  - group: ""
    resources: ["pods", "services", "secrets", "configmaps"]
  - group: "apps"
    resources: ["deployments", "replicasets"]
```

## Performance Optimization

### Configure Resource Quotas

```yaml
# resource-quota.yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: oran-ric-quota
  namespace: oran-ric
spec:
  hard:
    requests.cpu: "20"
    requests.memory: "40Gi"
    limits.cpu: "40"
    limits.memory: "80Gi"
    persistentvolumeclaims: "20"
    requests.storage: "500Gi"
```

### Enable Horizontal Pod Autoscaling

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: submgr-hpa
  namespace: oran-ric
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: submgr
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Operational Procedures

### Daily Operations

```bash
# Check platform health
./scripts/production-monitoring.sh monitor

# Review alerts
kubectl logs -n oran-ric deployment/alertmanager

# Check resource usage
kubectl top nodes
kubectl top pods -n oran-ric
```

### Weekly Operations

```bash
# Generate health report
./scripts/production-monitoring.sh report

# Review backup status
kubectl get cronjobs -n oran-ric
kubectl get jobs -n oran-ric | grep backup

# Security scan
./scripts/security-monitoring.sh full
```

### Monthly Operations

```bash
# Update platform components
./scripts/automated-updates.sh check
./scripts/automated-updates.sh update

# Certificate rotation check
./scripts/security-monitoring.sh compliance

# Capacity planning review
./scripts/capacity-monitor.sh
```

## Troubleshooting

### Common Issues

#### Pod Startup Issues
```bash
# Check pod events
kubectl describe pod <pod-name> -n oran-ric

# Check resource constraints
kubectl get events -n oran-ric --sort-by='.lastTimestamp'

# Check node resources
kubectl describe nodes
```

#### Network Connectivity Issues
```bash
# Test inter-pod connectivity
kubectl exec -n oran-ric deployment/e2mgr -- nc -z submgr 8080

# Check network policies
kubectl get networkpolicies -n oran-ric

# Test external connectivity
kubectl exec -n oran-ric deployment/e2term -- nc -z external-e2-node 36422
```

#### Performance Issues
```bash
# Check resource usage
kubectl top pods -n oran-ric --sort-by=cpu
kubectl top pods -n oran-ric --sort-by=memory

# Check database performance
kubectl exec -n oran-ric deployment/dbaas -- redis-cli info stats

# Review metrics
kubectl port-forward -n oran-ric svc/prometheus 9090:9090
```

### Emergency Procedures

#### Platform Restart
```bash
# Graceful restart of all components
kubectl rollout restart deployment --all -n oran-ric

# Wait for all pods to be ready
kubectl wait --for=condition=ready pods --all -n oran-ric --timeout=600s
```

#### Emergency Backup
```bash
# Create immediate backup
kubectl create job --from=cronjob/backup-cronjob emergency-backup -n oran-ric

# Monitor backup progress
kubectl logs -n oran-ric job/emergency-backup -f
```

#### Disaster Recovery
```bash
# Complete platform recovery
./scripts/disaster-recovery.sh complete-failure

# Database-only recovery
./scripts/disaster-recovery.sh database-failure
```

## Maintenance Windows

### Planned Maintenance

1. **Schedule**: Monthly, during low-traffic hours
2. **Duration**: 2-4 hours
3. **Notification**: 48 hours advance notice
4. **Rollback Plan**: Always prepared

### Maintenance Checklist

```bash
# Pre-maintenance
- [ ] Create backup
- [ ] Verify rollback procedures
- [ ] Notify stakeholders
- [ ] Scale down non-critical workloads

# During maintenance
- [ ] Apply updates
- [ ] Run validation tests
- [ ] Monitor system health
- [ ] Document changes

# Post-maintenance
- [ ] Verify all services
- [ ] Run integration tests
- [ ] Monitor for 24 hours
- [ ] Update documentation
```

This production deployment guide provides a comprehensive approach to deploying and maintaining the O-RAN RIC platform in production environments with enterprise-grade reliability, security, and performance.