# O-RAN Near-RT RIC Production Deployment Guide

This guide provides complete instructions for deploying the O-RAN Near-RT RIC project in production environments with browser-accessible dashboard interface.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start (Docker)](#quick-start-docker)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Configuration](#configuration)
- [Security](#security)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Maintenance](#maintenance)

## Prerequisites

### System Requirements

- **Operating System**: Linux (Ubuntu 20.04+), macOS, or Windows 10/11 with WSL2
- **CPU**: Minimum 4 cores, Recommended 8+ cores
- **Memory**: Minimum 8GB RAM, Recommended 16GB+ RAM
- **Storage**: Minimum 50GB free space, Recommended 100GB+ SSD
- **Network**: Stable internet connection for downloads

### Software Requirements

#### For Docker Deployment
- **Docker**: Version 20.10+
- **Docker Compose**: Version 2.0+
- **curl**: For health checks
- **Git**: For cloning the repository

#### For Kubernetes Deployment
- **kubectl**: Version 1.24+
- **Kubernetes cluster**: Version 1.24+ (minikube, k3s, or cloud provider)
- **Helm**: Version 3.8+ (optional)

### Port Requirements

Ensure the following ports are available:

| Service | Port | Protocol | Description |
|---------|------|----------|-------------|
| Web Dashboard | 3000 | HTTP | Main dashboard interface |
| Dashboard API | 8080 | HTTP | REST API endpoint |
| RIC Core | 8081 | HTTP | RIC management API |
| xApp Manager | 8082 | HTTP | xApp management |
| Nginx (Load Balancer) | 80, 443 | HTTP/HTTPS | Reverse proxy |
| Grafana | 3001 | HTTP | Monitoring dashboards |
| Prometheus | 9090 | HTTP | Metrics collection |
| Jaeger | 16686 | HTTP | Distributed tracing |
| PostgreSQL | 5432 | TCP | Database |
| Redis | 6379 | TCP | Cache |
| Elasticsearch | 9200 | HTTP | Log storage |

## Quick Start (Docker)

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/near-rt-ric-new.git
cd near-rt-ric-new
```

### 2. Run Production Deployment

#### On Linux/macOS:
```bash
./scripts/deploy-production.sh
```

#### On Windows:
```cmd
scripts\deploy-production.bat
```

### 3. Access the Dashboard

Once deployment completes, access the dashboard at:
- **Main Dashboard**: http://localhost:3000
- **Grafana Monitoring**: http://localhost:3001 (admin/admin123)
- **Prometheus Metrics**: http://localhost:9090
- **Jaeger Tracing**: http://localhost:16686

### 4. Verify Deployment

```bash
./scripts/health-check.sh
```

## Kubernetes Deployment

### 1. Prepare Kubernetes Cluster

Ensure you have a running Kubernetes cluster:

```bash
# For minikube
minikube start --cpus=4 --memory=8192 --disk-size=50gb

# For k3s
curl -sfL https://get.k3s.io | sh -

# Verify cluster
kubectl cluster-info
```

### 2. Deploy to Kubernetes

```bash
# Apply the deployment
kubectl apply -f deployments/kubernetes/oran-ric-deployment.yaml

# Check deployment status
kubectl get pods -n oran-ric -w
```

### 3. Access via Port Forwarding

```bash
# Forward dashboard port
kubectl port-forward -n oran-ric service/web-dashboard 3000:80

# Forward API port
kubectl port-forward -n oran-ric service/dashboard-api 8080:8080
```

### 4. Access via Ingress (Recommended)

```bash
# Install NGINX Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml

# Add to /etc/hosts
echo "127.0.0.1 oran-ric.local" | sudo tee -a /etc/hosts

# Access dashboard
open http://oran-ric.local
```

## Configuration

### Environment Configuration

Create and customize your environment file:

```bash
cp .env.example .env.production
```

**Critical settings to update:**

```env
# Security - MUST CHANGE IN PRODUCTION
DB_PASSWORD=your_secure_db_password_here
REDIS_PASSWORD=your_secure_redis_password_here
JWT_SECRET=your_jwt_secret_key_here
GRAFANA_PASSWORD=your_grafana_password_here

# Application Settings
VERSION=v1.0.0
LOG_LEVEL=info
NODE_ENV=production
GO_ENV=production

# Database Settings
DB_HOST=postgres
DB_PORT=5432
DB_NAME=oran_prod
DB_USER=oran_prod
DB_SSL_MODE=require

# Redis Settings
REDIS_HOST=redis
REDIS_PORT=6379

# Monitoring
JAEGER_AGENT_HOST=jaeger
JAEGER_AGENT_PORT=6831
```

### Network Configuration

#### Docker Network
The production deployment creates an isolated Docker network:
- **Network**: `oran-production` (172.30.0.0/16)
- **Gateway**: 172.30.0.1

#### Service Discovery
Services communicate using Docker's internal DNS:
- `dashboard-api:8080`
- `ric-core:8081`
- `postgres:5432`
- `redis:6379`

### Volume Configuration

Persistent data is stored in Docker volumes:
- `postgres-data`: Database files
- `redis-data`: Cache data
- `prometheus-data`: Metrics history
- `grafana-data`: Dashboard configurations
- `elasticsearch-data`: Log storage

## Security

### SSL/TLS Configuration

#### Generate Certificates

The deployment script automatically generates self-signed certificates:

```bash
# Manual certificate generation
mkdir -p certs
openssl genrsa -out certs/ca.key 4096
openssl req -new -x509 -key certs/ca.key -sha256 -subj "/C=US/ST=CA/O=ORAN/CN=ORAN-CA" -days 3650 -out certs/ca.crt
```

#### Production Certificates

For production, use certificates from a trusted CA:

```bash
# Place your certificates in the certs directory
cp your-domain.crt certs/server.crt
cp your-domain.key certs/server.key
cp ca-bundle.crt certs/ca.crt
```

### Authentication

#### Default Credentials

**Change these immediately in production:**

| Service | Username | Password |
|---------|----------|----------|
| Grafana | admin | admin123 |
| PostgreSQL | oran_prod | (from .env file) |
| Redis | - | (from .env file) |

#### JWT Configuration

Generate a secure JWT secret:

```bash
openssl rand -base64 64
```

### Network Security

#### Firewall Configuration

Configure your firewall to only allow necessary ports:

```bash
# Ubuntu/Debian
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 3000/tcp
sudo ufw deny 5432/tcp  # Block direct database access
sudo ufw deny 6379/tcp  # Block direct Redis access
```

#### Security Headers

The Nginx configuration includes security headers:
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy`

## Monitoring

### Grafana Dashboards

Access Grafana at http://localhost:3001

**Pre-configured dashboards:**
- O-RAN RIC Overview
- API Performance Metrics
- Database Performance
- Container Resource Usage
- Network Performance

### Prometheus Metrics

Access Prometheus at http://localhost:9090

**Key metrics collected:**
- HTTP request rates and latencies
- Database connection pools
- Memory and CPU usage
- Custom O-RAN metrics

### Distributed Tracing

Access Jaeger at http://localhost:16686

**Trace collection:**
- API request flows
- Database query traces
- Inter-service communication
- Error propagation

### Log Aggregation

Logs are collected by Elasticsearch and viewable through:
- Docker Compose logs: `docker-compose logs -f`
- Centralized logging (if configured)

## Troubleshooting

### Common Issues

#### Port Conflicts

```bash
# Check port usage
netstat -tulpn | grep :3000

# Kill process using port
sudo lsof -ti:3000 | xargs kill -9
```

#### Container Health Issues

```bash
# Check container status
docker-compose ps

# View container logs
docker-compose logs <service-name>

# Restart specific service
docker-compose restart <service-name>
```

#### Database Connection Issues

```bash
# Test PostgreSQL connection
docker-compose exec postgres pg_isready -U oran_prod

# Test Redis connection
docker-compose exec redis redis-cli ping

# Check database logs
docker-compose logs postgres
```

#### Memory Issues

```bash
# Check container memory usage
docker stats

# Increase Docker memory limits (Docker Desktop)
# Settings > Resources > Advanced > Memory: 8GB+
```

### Health Checks

#### Automated Health Check

```bash
./scripts/health-check.sh
```

#### Manual Health Checks

```bash
# API health
curl http://localhost:8080/health

# Dashboard health
curl http://localhost:3000/health

# Database health
curl http://localhost:9090/-/healthy
```

### Debug Mode

Enable debug logging:

```bash
# Update .env.production
LOG_LEVEL=debug

# Restart services
docker-compose restart dashboard-api ric-core
```

## Maintenance

### Backup

#### Database Backup

```bash
# Create backup
docker-compose exec postgres pg_dump -U oran_prod oran_prod > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore backup
docker-compose exec -T postgres psql -U oran_prod oran_prod < backup_20231201_120000.sql
```

#### Volume Backup

```bash
# Backup all volumes
docker run --rm -v $(pwd):/backup -v oran-near-rt-ric_postgres-data:/data alpine tar czf /backup/postgres-backup.tar.gz -C /data .
```

### Updates

#### Application Updates

```bash
# Pull latest images
docker-compose pull

# Restart with new images
docker-compose up -d

# Verify deployment
./scripts/health-check.sh
```

#### Security Updates

```bash
# Update base images
docker-compose build --pull --no-cache

# Update dependencies
docker-compose down
docker-compose up -d
```

### Monitoring Maintenance

#### Log Rotation

Configure log rotation to prevent disk space issues:

```bash
# Add to crontab
0 2 * * * docker system prune -f --volumes --filter "until=24h"
```

#### Metrics Retention

Configure Prometheus retention in `configs/prometheus/prometheus-production.yml`:

```yaml
command:
  - '--storage.tsdb.retention.time=30d'
  - '--storage.tsdb.retention.size=10GB'
```

### Scaling

#### Horizontal Scaling

Scale specific services:

```bash
# Scale API instances
docker-compose up -d --scale dashboard-api=3

# Scale web instances
docker-compose up -d --scale web-dashboard=2
```

#### Resource Limits

Update resource limits in `docker-compose.production.yml`:

```yaml
deploy:
  resources:
    limits:
      cpus: '4.0'
      memory: 4G
    reservations:
      cpus: '1.0'
      memory: 1G
```

## Support

### Getting Help

1. **Check the logs**: `docker-compose logs -f`
2. **Run health checks**: `./scripts/health-check.sh`
3. **Review configuration**: Verify environment variables
4. **Check resources**: Ensure adequate CPU/memory
5. **Network connectivity**: Verify port accessibility

### Reporting Issues

When reporting issues, include:
- Docker/Kubernetes version
- Operating system
- Error logs
- Configuration files (redacted)
- Health check output

---

**© 2024 O-RAN Near-RT RIC Project. Licensed under Apache 2.0.**