# O-RAN Near-RT RIC Development Workflow Guide

This comprehensive guide outlines the complete development workflow for the O-RAN Near-RT RIC project, including quick start procedures, development practices, testing strategies, and deployment processes.

## Table of Contents

- [Quick Start](#quick-start)
- [Development Environment Setup](#development-environment-setup)
- [Development Workflow](#development-workflow)
- [Testing Strategy](#testing-strategy)
- [Code Quality](#code-quality)
- [Build and Deploy](#build-and-deploy)
- [CI/CD Pipeline](#cicd-pipeline)
- [Monitoring and Observability](#monitoring-and-observability)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Quick Start

### Prerequisites Check

Ensure you have the following installed:
- **Docker** 20.10+
- **Docker Compose** 2.0+
- **Go** 1.21+
- **Node.js** 18+
- **kubectl** (for Kubernetes deployment)
- **Helm** 3.0+ (for Kubernetes deployment)

Run this script to verify your environment:
```bash
./scripts/check-prerequisites.sh
```

### Initial Setup

```bash
# Clone and setup
git clone <repository>
cd near-rt-ric-new

# Install dependencies
make setup

# Clean existing artifacts
make clean

# Build all components
make build

# Run tests
make test
```

### One-Command Development Environment

```bash
# Option 1: Full development stack
make dev-stack

# Option 2: Manual Docker Compose
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Option 3: Development with hot reload
make dev
```

### Access Development Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Dashboard API | http://localhost:8080 | - |
| UI Development | http://localhost:3001 | - |
| Grafana | http://localhost:3000 | admin/admin123 |
| Prometheus | http://localhost:9092 | - |
| Jaeger | http://localhost:16686 | - |
| API Documentation | http://localhost:8085 | - |

## Development Environment Setup

### Automated Setup

Use the automated setup script:

```bash
./scripts/dev-setup.sh
```

This script will:
- Check prerequisites
- Install Go development tools
- Setup Node.js environment
- Create necessary directories
- Configure development services
- Validate the environment

### Manual Setup

If you prefer manual setup:

```bash
# 1. Install development tools
make install-tools

# 2. Download dependencies
make deps

# 3. Create directories
make setup-dirs

# 4. Validate environment
make validate
```

### Environment Variables

Create a `.env` file in the project root:

```bash
# Application
LOG_LEVEL=debug
GIN_MODE=debug
HOT_RELOAD=true

# Database
DATABASE_URL="postgres://ricuser:ricpassword@localhost:5432/ricdb?sslmode=disable"
REDIS_URL="redis://:redispassword@localhost:6379/0"

# Monitoring
JAEGER_AGENT_HOST=localhost
JAEGER_AGENT_PORT=6831
PROMETHEUS_URL=http://localhost:9092

# Development
PORT="8080"
```

## Development Workflow

### Branch Strategy

We follow **GitFlow** with the following branches:
- `main` - Production-ready code
- `develop` - Integration branch for features
- `feature/*` - Feature development branches
- `hotfix/*` - Critical bug fixes
- `release/*` - Release preparation branches

### Daily Development

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

### Feature Development Process

#### 1. Create Feature Branch
```bash
# Create and checkout feature branch
git checkout -b feature/your-feature-name develop

# Start development environment
make dev-stack
```

#### 2. Development Loop
```bash
# Write code
vim ...

# Format code
make fmt

# Run linters
make lint

# Test continuously
make test-unit

# Check coverage
make test-coverage
open coverage.html
```

#### 3. Integration Testing
```bash
# Start dependencies
docker-compose up postgres redis -d

# Run integration tests
make test-integration
```

#### 4. Before Committing
```bash
# Format and lint
make fmt
make lint

# Run tests
make test-coverage

# Security scan
make security-scan

# Run all pre-commit checks
make pre-commit
```

#### 5. Submit Changes
```bash
# Add changes
git add .

# Commit with conventional commit format
git commit -m "feat: add new feature

- Implement new functionality
- Add comprehensive tests
- Update documentation"

# Push to remote
git push origin feature/your-feature-name

# Create PR
gh pr create --title "Add new feature" --body "..."
```

### Code Organization

```
near-rt-ric-new/
├── cmd/                    # Application entry points
│   ├── dashboard-api/      # Dashboard API Gateway
│   ├── ric/               # RIC Core
│   └── xapp-hello-world/  # Example xApp
├── pkg/                    # Shared libraries
│   ├── dashboard/         # Dashboard API logic
│   ├── e2/               # E2 interface
│   ├── models/           # Data models
│   └── utils/            # Utility functions
├── api/                   # Protocol definitions
│   └── proto/            # Protobuf definitions
├── ui/                    # React frontend
├── helm/                  # Helm charts
├── scripts/               # Development scripts
├── test/                  # Test suites
│   ├── integration/      # Integration tests
│   ├── load/             # Load tests
│   └── e2e/              # End-to-end tests
└── configs/               # Configuration files
```

## Testing Strategy

### Test Pyramid

```
    /\     
   /  \    E2E Tests (Few, High-level)
  /____\   
 /      \  Integration Tests (Some, API-level)  
/________\ Unit Tests (Many, Component-level)
```

### Unit Tests

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test -v -run TestSpecificFunction ./pkg/...

# Debug test
go test -v -count=1 ./...

# Run benchmarks
make benchmark
```

**Coverage Requirements:**
- Minimum: 70%
- Target: 85%
- Critical paths: 95%

### Integration Tests

```bash
# Start test databases
docker-compose up -d postgres redis

# Run integration tests
make test-integration
```

**Integration Test Scope:**
- Database operations
- External service integration
- gRPC/REST API contracts
- Message queue operations

### End-to-End Tests

```bash
# Run complete E2E test suite
make test-e2e

# Or run the script directly
./scripts/e2e-test.sh
```

**E2E Test Scenarios:**
- Complete user workflows
- Service-to-service communication
- Data flow validation
- Performance benchmarks

### Load Testing

```bash
# Run load tests with k6
k6 run test/load/load-test.js

# Or through Docker
docker run --rm -v $(pwd)/test/load:/scripts loadimpact/k6:latest run /scripts/load-test.js
```

## Code Quality

### Linting Configuration

```bash
# Run Go linters
make lint-go

# Run UI linters
make lint-ui

# Run Helm linters
make lint-helm

# Run all linters
make lint
```

### Security Scanning

```bash
# Run security scans
make security

# View security report
cat coverage/security.json
```

### Code Formatting

```bash
# Format all code
make fmt
```

**Formatting Rules:**
- Go: `gofmt` + `goimports`
- JavaScript/TypeScript: Prettier
- YAML: Prettier
- Markdown: Prettier

### Pre-commit Hooks

Install pre-commit hooks:

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

## Build and Deploy

### Local Development

```bash
# Build everything
make build

# Build Docker images
make docker-build

# Deploy locally
docker-compose up -d
# OR
make deploy-local

# Deploy to local Kubernetes
make deploy-local

# Undeploy
make undeploy-local
```

### Production Deployment

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

# Or using Helm
helm upgrade --install near-rt-ric ./helm/near-rt-ric \
  --namespace oran \
  --values helm/values-production.yaml
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

## Monitoring and Observability

### Metrics

**Application Metrics:**
- Business metrics (custom)
- Performance metrics (response time, throughput)
- Error rates and success rates
- Resource utilization

**Infrastructure Metrics:**
- CPU, Memory, Disk, Network
- Container metrics
- Database performance
- Message queue metrics

### Logging

**Log Levels:**
- `ERROR` - System errors, failures
- `WARN` - Warning conditions
- `INFO` - General information
- `DEBUG` - Detailed debugging info

**Log Format:**
```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "INFO",
  "service": "dashboard-api",
  "message": "Request processed",
  "request_id": "req-123",
  "user_id": "user-456",
  "duration_ms": 45
}
```

**Access Services:**
- Prometheus endpoint: `http://localhost:9090/metrics`
- Grafana dashboards: `http://localhost:3001`
- Jaeger UI: `http://localhost:16686`

Enable tracing with `ENABLE_TRACING=true`

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

#### 4. Services Not Starting
```bash
# Check service logs
docker-compose logs dashboard-api

# Check service health
make validate

# Restart services
docker-compose restart dashboard-api
```

#### 5. Database Connection Issues
```bash
# Check database status
docker exec postgres pg_isready -U oran

# View database logs
docker-compose logs postgres

# Reset database
docker-compose down postgres
docker volume rm near-rt-ric-new_postgres-data
docker-compose up -d postgres
```

#### 6. Kubernetes Deployment Issues
```bash
# Check pod status
kubectl get pods -n oran

# Check logs
kubectl logs -n oran deployment/dashboard-api

# Describe pod
kubectl describe pod -n oran <pod-name>
```

#### 7. Network Issues
```bash
# Check network configuration
docker network ls
docker network inspect near-rt-ric-new_oran-network

# Reset network
docker-compose down
docker network prune -f
docker-compose up -d
```

### Debug Mode

Enable debug mode for detailed logging:

```bash
# Set environment variable
export LOG_LEVEL=debug

# Or in docker-compose.yml
environment:
  - LOG_LEVEL=debug
  - GIN_MODE=debug
```

### Performance Issues

```bash
# Check resource usage
docker stats

# Profile Go applications
go tool pprof http://localhost:8080/debug/pprof/profile

# View metrics
curl http://localhost:8080/metrics
```

## Best Practices

### Development
- Write tests first (TDD)
- Use meaningful commit messages
- Keep branches small and focused
- Review your own code before requesting review
- Update documentation with code changes

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

## Workflow Automation

### Make Targets Summary

| Target | Description |
|--------|-------------|
| `make help` | Show all available targets |
| `make setup-env` | Setup development environment |
| `make build` | Build all components |
| `make test` | Run all tests |
| `make lint` | Run all linters |
| `make security` | Run security scans |
| `make docker-build` | Build Docker images |
| `make dev-stack` | Start development stack |
| `make deploy-local` | Deploy to local Kubernetes |
| `make clean` | Clean build artifacts |
| `make pre-commit` | Run pre-commit checks |
| `make ci` | Run full CI pipeline locally |

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

This comprehensive development workflow ensures consistent, high-quality development practices across the O-RAN Near-RT RIC project.