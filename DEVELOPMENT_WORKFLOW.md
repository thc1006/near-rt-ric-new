# O-RAN Near-RT RIC Development Workflow

This document outlines the complete development workflow for the O-RAN Near-RT RIC project, including setup, development practices, testing procedures, and deployment processes.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Development Environment Setup](#development-environment-setup)
- [Development Workflow](#development-workflow)
- [Testing Strategy](#testing-strategy)
- [Code Quality](#code-quality)
- [Deployment Process](#deployment-process)
- [Monitoring and Observability](#monitoring-and-observability)
- [Troubleshooting](#troubleshooting)

## 🚀 Quick Start

### Prerequisites

Ensure you have the following installed:
- **Docker** 20.10+
- **Docker Compose** 2.0+
- **Go** 1.21+
- **Node.js** 18+
- **kubectl** (for Kubernetes deployment)
- **Helm** 3.0+ (for Kubernetes deployment)

### One-Command Setup

```bash
# Clone and setup the entire development environment
git clone <repository-url>
cd near-rt-ric-new
./scripts/dev-setup.sh
```

### Start Development Environment

```bash
# Option 1: Full development stack
make dev-stack

# Option 2: Manual Docker Compose
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Option 3: Development with hot reload
make dev
```

### Access Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Dashboard API | http://localhost:8080 | - |
| UI Development | http://localhost:3001 | - |
| Grafana | http://localhost:3000 | admin/admin123 |
| Prometheus | http://localhost:9092 | - |
| Jaeger | http://localhost:16686 | - |
| SonarQube | http://localhost:9000 | admin/admin |
| API Documentation | http://localhost:8085 | - |

## 🛠 Development Environment Setup

### Automated Setup

Use the automated setup script:

```bash
./scripts/dev-setup.sh
```

This script will:
- ✅ Check prerequisites
- ✅ Install Go development tools
- ✅ Setup Node.js environment
- ✅ Create necessary directories
- ✅ Configure development services
- ✅ Validate the environment

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
DB_HOST=localhost
DB_PORT=5432
DB_NAME=oran
DB_USER=oran
DB_PASSWORD=oran123

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Monitoring
JAEGER_AGENT_HOST=localhost
JAEGER_AGENT_PORT=6831
PROMETHEUS_URL=http://localhost:9092
```

## 🔄 Development Workflow

### Branch Strategy

We follow **GitFlow** with the following branches:

- `main` - Production-ready code
- `develop` - Integration branch for features
- `feature/*` - Feature development branches
- `hotfix/*` - Critical bug fixes
- `release/*` - Release preparation branches

### Development Process

#### 1. Start New Feature

```bash
# Create and checkout feature branch
git checkout -b feature/your-feature-name develop

# Start development environment
make dev-stack
```

#### 2. Development Loop

```bash
# Format code
make fmt

# Run linters
make lint

# Run tests
make test

# Build components
make build
```

#### 3. Continuous Testing

```bash
# Run tests in watch mode
make test-short

# Run integration tests
make test-integration

# Run E2E tests
make test-e2e
```

#### 4. Pre-commit Checks

```bash
# Run all pre-commit checks
make pre-commit

# Or run individual checks
make fmt lint test-short security
```

#### 5. Submit Changes

```bash
# Add changes
git add .

# Commit with conventional commit format
git commit -m "feat: add new xApp management API

- Implement xApp lifecycle management
- Add REST endpoints for xApp operations
- Include comprehensive unit tests
- Update API documentation"

# Push to remote
git push origin feature/your-feature-name
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

## 🧪 Testing Strategy

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

## 📊 Code Quality

### Linting Configuration

The project uses comprehensive linting with `.golangci.yml`:

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

## 🚀 Deployment Process

### Local Development Deployment

```bash
# Deploy to local Kubernetes
make deploy-local

# Undeploy
make undeploy-local
```

### Docker Deployment

```bash
# Build Docker images
make docker-build

# Start with Docker Compose
make docker-compose-up

# Stop services
make docker-compose-down
```

### Production Deployment

```bash
# Build and push images
make docker-build docker-push

# Deploy to production
helm upgrade --install near-rt-ric ./helm/near-rt-ric \
  --namespace oran \
  --values helm/values-production.yaml
```

### CI/CD Pipeline

The project includes GitHub Actions workflows:

- **CI Pipeline** (`.github/workflows/ci-cd.yml`):
  - Lint Go and UI code
  - Run unit and integration tests
  - Security scanning
  - Build Docker images
  - Deploy to staging/production

- **Release Pipeline**:
  - Automated versioning
  - Release notes generation
  - Docker image publishing
  - Helm chart publishing

## 📈 Monitoring and Observability

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

### Tracing

Distributed tracing with Jaeger:

- **Trace Context:** Propagated across service boundaries
- **Span Attributes:** Include relevant business context
- **Sampling:** Configurable sampling rates

### Alerting

**Alert Categories:**
- **Critical:** Service down, high error rate
- **Warning:** High latency, resource usage
- **Info:** Deployment events, configuration changes

**Alert Channels:**
- Email notifications
- Webhook integrations
- Dashboard alerts

## 🔧 Troubleshooting

### Common Issues

#### Services Not Starting

```bash
# Check service logs
docker-compose logs dashboard-api

# Check service health
make validate

# Restart services
docker-compose restart dashboard-api
```

#### Database Connection Issues

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

#### Build Failures

```bash
# Clean and rebuild
make clean
make build

# Check Go module issues
go mod tidy
go mod verify

# Clear Docker build cache
docker system prune -f
```

#### Network Issues

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

### Getting Help

1. **Check Documentation**: Review relevant sections above
2. **Search Issues**: Check existing GitHub issues
3. **Create Issue**: Use issue templates for bug reports
4. **Ask Questions**: Use discussion forums or Slack channels

## 📚 Additional Resources

- [O-RAN Architecture](docs/architecture.md)
- [API Documentation](docs/api/)
- [Deployment Guide](docs/deployment.md)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Security Guidelines](SECURITY.md)

## 🔄 Workflow Automation

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

### Git Hooks

Pre-commit hooks automatically run:
- Code formatting
- Linting
- Unit tests
- Security checks

This ensures code quality before commits reach the repository.

---

## 🎯 Best Practices

### Development
- ✅ Write tests first (TDD)
- ✅ Use meaningful commit messages
- ✅ Keep branches small and focused
- ✅ Review your own code before requesting review
- ✅ Update documentation with code changes

### Testing
- ✅ Test behavior, not implementation
- ✅ Use test doubles for external dependencies
- ✅ Keep tests fast and reliable
- ✅ Test error conditions
- ✅ Maintain test data independently

### Code Quality
- ✅ Follow established conventions
- ✅ Write self-documenting code
- ✅ Handle errors explicitly
- ✅ Use logging effectively
- ✅ Keep functions small and focused

This development workflow ensures consistent, high-quality development practices across the O-RAN Near-RT RIC project.