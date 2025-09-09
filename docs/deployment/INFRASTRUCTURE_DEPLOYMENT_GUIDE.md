# O-RAN SC Near-RT RIC Infrastructure Deployment Guide

This guide provides comprehensive instructions for deploying O-RAN SC Near-RT RIC following official L Release standards.

## Overview

This deployment supports multiple modes:
- **Docker Compose**: Development and testing
- **Kubernetes**: Production deployment 
- **Helm**: Production with advanced configuration management

## Quick Start

### 1. Infrastructure Preparation

```bash
# Check prerequisites
./scripts/prepare-oran-infrastructure.sh --check-only

# Setup complete infrastructure (recommended)
./scripts/prepare-oran-infrastructure.sh --all

# Or setup individual components
./scripts/prepare-oran-infrastructure.sh --docker-setup
./scripts/prepare-oran-infrastructure.sh --k8s-setup
./scripts/prepare-oran-infrastructure.sh --helm-setup
```

### 2. Deploy O-RAN SC Platform

```bash
# Docker Compose deployment (development)
./scripts/deploy-oran-l-release.sh --mode docker

# Kubernetes deployment (production)
./scripts/deploy-oran-l-release.sh --mode kubernetes

# Helm deployment (production with customization)
./scripts/deploy-oran-l-release.sh --mode helm
```

### 3. Validate Deployment

```bash
# Comprehensive validation
./scripts/validate-oran-deployment.sh

# Check deployment status
./scripts/deploy-oran-l-release.sh --status
```

## Deployment Modes

### Docker Compose Mode

**Best for**: Development, testing, single-node deployments

**Features**:
- Quick setup and teardown
- Complete O-RAN SC platform with monitoring
- Official O-RAN SC container images
- Persistent data storage
- Network isolation for different interfaces

**Usage**:
```bash
# Deploy
./scripts/deploy-oran-l-release.sh --mode docker

# Check status
docker-compose -f docker-compose.oran-l-release.yml ps

# View logs
docker-compose -f docker-compose.oran-l-release.yml logs -f

# Cleanup
./scripts/deploy-oran-l-release.sh --mode docker --cleanup
```

**Access Points**:
- E2 Interface (SCTP): `localhost:36421`
- A1 Interface (HTTP): `http://localhost:10000`
- O1 Interface (NETCONF): `localhost:830`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (admin/admin123)
- Jaeger: `http://localhost:16686`

### Kubernetes Mode

**Best for**: Production deployments, multi-node clusters, scalability

**Features**:
- Production-ready with security policies
- Horizontal pod autoscaling
- Persistent volume management
- Network policies for interface isolation
- Service mesh ready
- RBAC and Pod Security Standards

**Prerequisites**:
```bash
# Ensure Kubernetes cluster is available
kubectl cluster-info

# Verify nodes are ready
kubectl get nodes
```

**Usage**:
```bash
# Deploy
./scripts/deploy-oran-l-release.sh --mode kubernetes --namespace ricplt

# Check status
kubectl get all -n ricplt

# Scale components
kubectl scale deployment ricplt-e2mgr --replicas=2 -n ricplt

# Access services via port-forward
kubectl port-forward -n ricplt svc/service-ricplt-a1mediator-http 10000:10000
```

**Networking**:
- Multiple network attachment definitions for O-RAN interfaces
- Separate networks for E2, A1, O1, F1, N2, N3 interfaces
- SCTP support for E2 interface
- LoadBalancer services for external access

### Helm Mode

**Best for**: Production deployments with customization, GitOps workflows

**Features**:
- Parameterized deployment
- Environment-specific configurations
- Upgrade and rollback capabilities
- Dependency management
- Release management

**Usage**:
```bash
# Deploy with default values
./scripts/deploy-oran-l-release.sh --mode helm

# Deploy with custom values
helm upgrade --install oran-sc-platform ./helm/ric-platform \
  --namespace ricplt \
  --values custom-values.yaml

# Check release status
helm status oran-sc-platform -n ricplt

# Upgrade release
helm upgrade oran-sc-platform ./helm/ric-platform \
  --namespace ricplt \
  --reuse-values

# Rollback
helm rollback oran-sc-platform 1 -n ricplt
```

## Architecture

### Core Components

1. **Database (Redis)**: Persistent storage for RIC data
2. **E2 Termination**: Entry point for E2 interface connections
3. **E2 Manager**: Manages E2 node connections and configurations
4. **Subscription Manager**: Handles RIC subscriptions
5. **Routing Manager**: Manages RMR message routing
6. **A1 Mediator**: Implements A1 interface for policy management
7. **O1 Mediator**: Implements O1 interface for configuration management

### Network Functions (Optional)

1. **CU-CP**: 5G Centralized Unit - Control Plane
2. **CU-UP**: 5G Centralized Unit - User Plane  
3. **DU**: 5G Distributed Unit

### Monitoring Stack

1. **Prometheus**: Metrics collection and alerting
2. **Grafana**: Visualization and dashboards
3. **Jaeger**: Distributed tracing

## Configuration

### Environment Variables

Key environment variables for customization:

```bash
# Release version
export VERSION="l-release"

# Namespace (Kubernetes/Helm)
export NAMESPACE="ricplt"

# Log levels
export LOG_LEVEL="INFO"

# Database configuration
export DBAAS_SERVICE_HOST="ricplt-dbaas"
export DBAAS_SERVICE_PORT="6379"

# RMR configuration
export RMR_RTG_SVC="4561"
```

### Configuration Files

Configuration files are located in `configs/` directory:

```
configs/
├── e2term/           # E2 Termination configuration
├── e2mgr/            # E2 Manager configuration
├── submgr/           # Subscription Manager configuration
├── rtmgr/            # Routing Manager configuration
├── a1mediator/       # A1 Mediator configuration
├── o1mediator/       # O1 Mediator configuration
├── prometheus/       # Prometheus configuration
├── grafana/          # Grafana dashboards and datasources
└── xapp/            # xApp configurations
```

### Network Configuration

O-RAN interfaces are isolated using dedicated networks:

- **E2 Network**: `192.168.10.0/24` - E2 interface traffic
- **A1 Network**: `192.168.11.0/24` - A1 interface traffic
- **O1 Network**: `192.168.12.0/24` - O1 interface traffic
- **F1 Network**: `192.168.20.0/24` - F1 interface traffic
- **N2 Network**: `192.168.30.0/24` - N2 interface traffic

## Security

### Pod Security Standards

- **Platform Namespace**: Restricted security context
- **xApp Namespace**: Restricted security context
- **Infrastructure Namespace**: Privileged (for system components)

### Network Policies

- Default deny-all policy
- Allow traffic between RIC components
- Allow external access to O-RAN interfaces
- DNS and HTTPS egress allowed

### RBAC

- Service accounts for each component
- Least privilege principle
- Cluster-level permissions for platform management

## Monitoring and Observability

### Metrics

Prometheus collects metrics from:
- All RIC platform components
- xApps
- Kubernetes cluster (when using k8s mode)
- Infrastructure components

### Dashboards

Pre-configured Grafana dashboards for:
- O-RAN SC Platform Overview
- E2 Interface Monitoring
- A1 Policy Management
- xApp Performance
- Resource Utilization

### Tracing

Jaeger provides distributed tracing for:
- RMR message routing
- API call flows
- Inter-component communication

### Logging

Structured logging with:
- Component identification
- Correlation IDs
- Log aggregation ready
- ELK stack compatible

## Troubleshooting

### Common Issues

1. **Container Registry Access**:
   ```bash
   # Check registry connectivity
   docker pull nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-rtmgr:0.8.0
   ```

2. **Resource Constraints**:
   ```bash
   # Check resource usage
   kubectl top nodes
   kubectl top pods -n ricplt
   ```

3. **Network Connectivity**:
   ```bash
   # Test service connectivity
   ./scripts/validate-oran-deployment.sh
   ```

4. **Configuration Issues**:
   ```bash
   # Validate configurations
   ./scripts/deploy-oran-l-release.sh --pre-check
   ```

### Debug Commands

```bash
# Docker mode
docker-compose -f docker-compose.oran-l-release.yml logs <service>
docker exec -it <container> /bin/sh

# Kubernetes mode
kubectl logs -f deployment/<deployment-name> -n ricplt
kubectl describe pod <pod-name> -n ricplt
kubectl exec -it <pod-name> -n ricplt -- /bin/sh

# Helm mode
helm status oran-sc-platform -n ricplt
helm get values oran-sc-platform -n ricplt
```

### Log Locations

- **Docker**: Container logs via `docker logs`
- **Kubernetes**: Pod logs via `kubectl logs`
- **Host logs**: `/var/log/` (component-specific)

## Performance Tuning

### Resource Allocation

Recommended resources per component:

| Component | CPU Request | CPU Limit | Memory Request | Memory Limit |
|-----------|-------------|-----------|----------------|--------------|
| E2 Term   | 500m        | 1000m     | 512Mi          | 1Gi          |
| E2 Manager| 200m        | 500m      | 256Mi          | 512Mi        |
| Sub Manager| 200m       | 500m      | 256Mi          | 512Mi        |
| Routing Manager| 200m    | 500m      | 256Mi          | 512Mi        |
| A1 Mediator| 200m       | 500m      | 256Mi          | 512Mi        |
| Database  | 100m        | 500m      | 256Mi          | 512Mi        |

### Scaling

```bash
# Horizontal scaling (Kubernetes/Helm)
kubectl scale deployment ricplt-e2mgr --replicas=3 -n ricplt

# Vertical scaling (update resource limits)
kubectl patch deployment ricplt-e2mgr -n ricplt -p '{"spec":{"template":{"spec":{"containers":[{"name":"e2mgr","resources":{"limits":{"cpu":"2000m","memory":"2Gi"}}}]}}}}'
```

## Backup and Recovery

### Database Backup

```bash
# Docker mode
docker exec ricplt-dbaas redis-cli BGSAVE
docker cp ricplt-dbaas:/data/dump.rdb ./backup/

# Kubernetes mode
kubectl exec -n ricplt ricplt-dbaas-0 -- redis-cli BGSAVE
kubectl cp ricplt/ricplt-dbaas-0:/data/dump.rdb ./backup/
```

### Configuration Backup

```bash
# Backup all configurations
tar -czf oran-config-backup-$(date +%Y%m%d).tar.gz configs/

# Kubernetes resources backup
kubectl get all -n ricplt -o yaml > ricplt-backup-$(date +%Y%m%d).yaml
```

## Upgrade Procedures

### Docker Compose

```bash
# Pull new images
docker-compose -f docker-compose.oran-l-release.yml pull

# Recreate containers
docker-compose -f docker-compose.oran-l-release.yml up -d --force-recreate
```

### Kubernetes

```bash
# Apply updated manifests
kubectl apply -f deployments/oran-sc-l-release.yaml

# Rolling update
kubectl rollout restart deployment/ricplt-e2mgr -n ricplt
```

### Helm

```bash
# Upgrade release
helm upgrade oran-sc-platform ./helm/ric-platform \
  --namespace ricplt \
  --reuse-values
```

## Support

### Documentation

- [O-RAN SC Documentation](https://docs.o-ran-sc.org/)
- [O-RAN SC GitHub](https://github.com/o-ran-sc)
- [Release Notes](https://docs.o-ran-sc.org/en/latest/release-notes.html)

### Community

- [O-RAN SC Wiki](https://wiki.o-ran-sc.org/)
- [Mailing Lists](https://lists.o-ran-sc.org/)
- [Slack Workspace](https://o-ran-sc-community.slack.com/)

### Issue Reporting

1. Check existing issues in respective component repositories
2. Use component-specific JIRA projects
3. Provide detailed logs and reproduction steps
4. Include deployment mode and configuration details

---

**Note**: This deployment follows O-RAN SC L Release standards and is production-ready when deployed using Kubernetes or Helm modes. Docker Compose mode is recommended for development and testing only.