# O-RAN Near-RT RIC Development Workflow

## Quick Start

### 1. Initial Setup
```bash
# Clone and setup
git clone <repository>
cd near-rt-ric-new

# Install dependencies
make setup

# Clean existing artifacts
make clean
```

### 2. Development Workflow

#### Daily Development
```bash
# Start your day
git pull origin develop
make clean
make lint
make test

# Run locally
make run-local
# OR use Docker Compose
docker-compose up -d
```

#### Before Committing
```bash
# Format and lint
make fmt
make lint

# Run tests
make test-coverage

# Security scan
make security-scan
```

### 3. Build and Deploy

#### Local Development
```bash
# Build everything
make build

# Build Docker images
make docker-build

# Deploy locally
docker-compose up -d
# OR
make deploy-local
```

#### Production Deployment
```bash
# Build for production
make clean
make test
make build
make docker-build

# Push to registry
make docker-push

# Deploy to production
make deploy-production
```

## Detailed Workflows

### Feature Development

1. **Create Feature Branch**
```bash
git checkout -b feature/your-feature-name
```

2. **Development Cycle**
```bash
# Write code
vim ...

# Test continuously
make test-unit

# Check coverage
make test-coverage
open coverage.html
```

3. **Integration Testing**
```bash
# Start dependencies
docker-compose up postgres redis -d

# Run integration tests
make test-integration
```

4. **Submit PR**
```bash
# Ensure all checks pass
make lint
make test
make security-scan

# Commit and push
git add .
git commit -m "feat: add new feature"
git push origin feature/your-feature-name
```

### Debugging Workflow

1. **Local Debugging**
```bash
# Run with debug logging
LOG_LEVEL=debug make run-dashboard

# Check logs
docker-compose logs -f dashboard-api
```

2. **Remote Debugging**
```bash
# Forward ports for debugging
kubectl port-forward deployment/dashboard-api 8080:8080 9090:9090

# Access metrics
curl http://localhost:9090/metrics
```

### Performance Testing

1. **Load Testing**
```bash
# Run benchmark tests
make benchmark

# Profile the application
go test -cpuprofile cpu.prof -memprofile mem.prof -bench=.
go tool pprof cpu.prof
```

2. **Monitor Performance**
```bash
# Start monitoring stack
docker-compose --profile monitoring up -d

# Access Grafana
open http://localhost:3001
# Default: admin/admin
```

## CI/CD Pipeline

### GitHub Actions Workflow

The CI/CD pipeline automatically runs on:
- Push to `main` or `develop`
- Pull requests
- Release creation

#### Pipeline Stages:

1. **Lint & Code Quality**
   - Go linting with golangci-lint
   - React linting with ESLint
   - Helm chart validation

2. **Testing**
   - Unit tests with coverage
   - Integration tests
   - Coverage threshold enforcement (>70%)

3. **Security Scanning**
   - Trivy vulnerability scanning
   - Gosec security analysis

4. **Build & Push**
   - Multi-arch Docker builds
   - Push to GitHub Container Registry

5. **Deployment**
   - Staging deployment (develop branch)
   - Production deployment (main branch)

### Manual Release Process

1. **Prepare Release**
```bash
# Update version
git checkout develop
git pull origin develop

# Create release branch
git checkout -b release/v1.2.3

# Update CHANGELOG.md
vim CHANGELOG.md

# Commit changes
git commit -am "chore: prepare release v1.2.3"
```

2. **Create Release**
```bash
# Merge to main
git checkout main
git merge --no-ff release/v1.2.3

# Tag release
git tag -a v1.2.3 -m "Release v1.2.3"

# Push
git push origin main --tags
```

3. **Post-Release**
```bash
# Merge back to develop
git checkout develop
git merge --no-ff main
git push origin develop
```

## Troubleshooting

### Common Issues

#### 1. Build Failures
```bash
# Clear Go cache
go clean -cache -modcache

# Rebuild
make clean
make build
```

#### 2. Test Failures
```bash
# Run specific test
go test -v -run TestSpecificFunction ./pkg/...

# Debug test
go test -v -count=1 ./...
```

#### 3. Docker Issues
```bash
# Clean Docker resources
docker system prune -a

# Rebuild without cache
docker-compose build --no-cache
```

#### 4. Kubernetes Deployment Issues
```bash
# Check pod status
kubectl get pods -n oran

# Check logs
kubectl logs -n oran deployment/dashboard-api

# Describe pod
kubectl describe pod -n oran <pod-name>
```

## Environment Variables

### Development
```bash
export DATABASE_URL="postgres://ricuser:ricpassword@localhost:5432/ricdb?sslmode=disable"
export REDIS_URL="redis://:redispassword@localhost:6379/0"
export LOG_LEVEL="debug"
export PORT="8080"
```

### Production
```bash
# Use secrets management
kubectl create secret generic oran-secrets \
  --from-literal=database-url="..." \
  --from-literal=redis-url="..." \
  -n oran
```

## Monitoring and Observability

### Metrics
- Prometheus endpoint: `http://localhost:9090/metrics`
- Grafana dashboards: `http://localhost:3001`

### Logging
- Structured JSON logging
- Log aggregation with ELK/Loki
- Correlation IDs for request tracing

### Tracing
- Jaeger UI: `http://localhost:16686`
- Enable with `ENABLE_TRACING=true`

## Best Practices

### Code Quality
1. Always run `make lint` before committing
2. Maintain >80% test coverage
3. Write integration tests for new features
4. Document API changes in OpenAPI spec

### Security
1. Never commit secrets
2. Run security scans regularly
3. Keep dependencies updated
4. Use least privilege principles

### Performance
1. Profile before optimizing
2. Use connection pooling
3. Implement caching strategically
4. Monitor resource usage

### Documentation
1. Update README for new features
2. Document configuration changes
3. Keep CHANGELOG.md current
4. Write clear commit messages

## Support

### Getting Help
- Check documentation in `/docs`
- Review examples in `/examples`
- Open an issue on GitHub
- Contact the team

### Contributing
1. Fork the repository
2. Create feature branch
3. Make changes with tests
4. Submit pull request
5. Wait for review

## Appendix

### Makefile Targets Reference
```bash
make help              # Show all available targets
make clean             # Clean build artifacts
make setup             # Install dependencies
make lint              # Run all linters
make test              # Run all tests
make build             # Build all binaries
make docker-build      # Build Docker images
make deploy-local      # Deploy to local K8s
make security-scan     # Run security scans
```

### Docker Compose Profiles
```bash
# Basic stack
docker-compose up -d

# With simulators
docker-compose --profile simulators up -d

# With tracing
docker-compose --profile tracing up -d

# Full stack
docker-compose --profile simulators --profile tracing --profile docs up -d
```

### Port Mappings
- Dashboard API: 8080
- Metrics: 9090
- UI: 3000
- PostgreSQL: 5432
- Redis: 6379
- Prometheus: 9091
- Grafana: 3001
- Jaeger: 16686
- Documentation: 8088