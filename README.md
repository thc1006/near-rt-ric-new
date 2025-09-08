# O-RAN Near-RT RIC Platform
## Interactive Operations Console & Production-Ready Implementation

[![CI](https://github.com/your-repo/near-rt-ric-new/actions/workflows/ci.yaml/badge.svg)](https://github.com/your-repo/near-rt-ric-new/actions/workflows/ci.yaml)

This project provides a comprehensive, production-ready O-RAN Near-RT RIC implementation with a fully interactive web-based operations console. It includes complete O-RAN L Release compliance, advanced analytics, ML-based optimization, and enterprise-grade security features.

## Project Overview

The O-RAN Near-RT RIC implementation includes three main interfaces:
- **E2 Interface**: SCTP+ASN.1 PER for RAN node communication
- **A1 Interface**: REST+JWT+RBAC for policy management
- **O1 Interface**: NETCONF/YANG + FCAPS for configuration and fault management

## Key Features

### Core Capabilities
- **Dynamic Dashboard**: React-based UI with auto-discovery of deployed network functions
- **Dashboard API Gateway**: Go-based REST API service with gRPC abstraction and WebSocket support
- **O-RAN SC Integration**: Native integration with O-RAN Software Community components
- **Production-Grade SMO**: Complete Service Management and Orchestration stack

### Advanced Analytics
- **Real-Time Telemetry**: VES event processing with InfluxDB storage
- **ML-Based Optimization**: Load prediction, quality forecasting, and anomaly detection
- **KPI Calculation**: Comprehensive O-RAN performance indicators
- **Network Health Scoring**: Intelligent network health assessment

### Security & Compliance
- **O-RAN WG11 Compliance**: Full WG11 security specification implementation
- **FIPS 140-3 Enforcement**: Go 1.25+ with strict cryptographic compliance
- **Zero-Trust Architecture**: Container hardening and network segmentation
- **mTLS Everywhere**: End-to-end encryption for all communications

### Observability
- **Real-Time Monitoring**: Integrated Grafana, Prometheus, and Jaeger stack
- **Comprehensive Logging**: Structured JSON logging with audit capabilities
- **Performance Analytics**: Advanced metrics collection and visualization
- **Alert Management**: Intelligent alerting with automated remediation

## Prerequisites

### Development Environment
- **Go 1.21+**: For building from source (FIPS 140-3 compliance with Go 1.25+)
- **Node.js 18+**: For UI development
- **Docker 20.10+**: Container runtime
- **Docker Compose 2.0+**: Multi-container orchestration
- **Kubernetes cluster**: KIND, K3s, Minikube, or production cluster
- **kubectl**: Kubernetes CLI tool
- **Helm 3.0+**: Kubernetes package manager
- **make**: Build automation tool

### Development Tools
```bash
# Install required tools
make install-tools

# Verify prerequisites
./scripts/check-prerequisites.sh
```

## Quick Start

### One-Command Setup
```bash
# Clone and setup the entire environment
git clone https://github.com/your-repo/near-rt-ric-new.git
cd near-rt-ric-new

# Deploy complete platform
make dev-stack
```

### Manual Setup
```bash
# 1. Build all components
make build

# 2. Build Docker images
make docker-build

# 3. Deploy to local Kubernetes
make deploy-local

# 4. Access services
# - Dashboard: http://localhost:8080
# - Grafana: http://localhost:3000
# - Analytics API: http://localhost:8088
```

## Building

### Build All Components
```bash
# Build all Go binaries
make build

# Build with FIPS compliance
make build-fips

# Build Docker images
make docker-build

# Lint and validate
make lint
make helm-lint
```

### Development Build
```bash
# Build dashboard API
go build ./cmd/dashboard-api

# Run comprehensive tests
make test-coverage

# Security scanning
make security-scan
```

### Frontend Development
```bash
# Install dependencies and build UI
cd ui
npm install
npm run build

# Start development server with hot reload
npm start
```

## Architecture

The platform follows a microservices architecture with the following key components:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   O-RAN DU/CU   │    │  Near-RT RIC     │    │   SMO/Non-RT    │
│   Components    │◄──►│  (E2 Interface)  │◄──►│   RIC (A1)      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │  Analytics       │
                       │  Platform        │
                       │  (ML/AI)         │
                       └──────────────────┘
```

### Core Components
- **E2 Termination**: Handles E2 node connections and messages
- **Subscription Manager**: Manages E2 subscriptions and policies
- **xApp Framework**: Runtime environment for network optimization applications
- **Dashboard API**: REST API gateway with WebSocket support
- **Analytics Engine**: Real-time telemetry processing and ML-based optimization

## Documentation

Our documentation is organized into the following categories:

### Development Guides
- [Development Workflow Guide](docs/development/workflow-guide.md) - Complete development workflow and best practices
- [Development Plan & Roadmap](docs/development/development-plan.md) - Strategic implementation roadmap

### Implementation Guides  
- [Analytics Platform](docs/implementation/analytics-platform.md) - Telemetry processing and ML analytics
- [Security Implementation](docs/implementation/security-implementation.md) - WG11 compliance and FIPS 140-3
- [Production Hardening](docs/implementation/PRODUCTION-HARDENING-SUMMARY.md) - Production deployment guide
- [Performance Testing](docs/implementation/PERFORMANCE_TESTING_IMPLEMENTATION.md) - Performance validation
- [Policy Management Framework](docs/implementation/POLICY-MANAGEMENT-FRAMEWORK.md) - Policy engine implementation
- [Service Model API](docs/implementation/SERVICE_MODEL_API_IMPLEMENTATION.md) - E2 service models
- [CU-DU Implementation](docs/implementation/CU-DU-IMPLEMENTATION-SUMMARY.md) - CU/DU integration

### Operations Guides
- [Infrastructure Setup](docs/operations/INFRASTRUCTURE_SETUP_COMPLETE.md) - Infrastructure deployment
- [Optimization Guide](docs/operations/OPTIMIZATION_GUIDE.md) - Performance optimization
- [Performance Testing Validation](docs/operations/validate-performance-testing.md) - Testing procedures

### Reports
- [Dependency Resolution](docs/reports/dependency-resolution.md) - Complete dependency analysis
- [O-RAN Compliance Testing](docs/reports/O-RAN-COMPLIANCE-TESTING-IMPLEMENTATION.md) - Compliance validation
- [ORAN-SC Migration Status](docs/reports/ORAN-SC-MIGRATION-STATUS.md) - Migration progress
- [Comprehensive Test Validation](docs/reports/COMPREHENSIVE_TEST_VALIDATION_REPORT.md) - Testing results
- [Performance Optimization Summary](docs/reports/PERFORMANCE_OPTIMIZATION_FINAL_SUMMARY.md) - Optimization results
- [Cleanup Report](docs/reports/CLEANUP_REPORT.md) - Code cleanup activities

## Testing

### Automated Testing
```bash
# Run all tests with coverage
make test-coverage

# Run integration tests  
make test-integration

# Run E2E tests
make test-e2e

# Run security tests
make security-test

# Run performance tests
make benchmark
```

### Test Coverage Requirements
- **Minimum Coverage**: 70%
- **Target Coverage**: 85%
- **Critical Paths**: 95%

## Development Standards

### Coding Standards
- **Go 1.21+**: Modules mode with FIPS 140-3 compliance
- **golangci-lint**: Comprehensive linting with strict rules
- **RFC-style comments**: Required on every public function
- **Unix LF, UTF-8 encoding**: Consistent file formatting

### Build Standards
```bash
# Lint code
make lint

# Format code
make fmt

# Security scanning
make security-scan

# Generate coverage report
make test-coverage
```

## Production Deployment

### Security Requirements
- **FIPS 140-3**: Cryptographic compliance enforced
- **mTLS**: All communications encrypted
- **RBAC**: Role-based access control
- **Zero-Trust**: Network policies with default deny
- **Container Security**: Pod Security Standards enforced

### Performance Targets
- **E2 Setup Latency**: <10ms
- **Subscription Processing**: >10K/sec
- **VES Events**: 10,000 events/second
- **API Requests**: 1,000 requests/second

## Service Endpoints

| Service | Port | Endpoint | Purpose |
|---------|------|----------|---------|
| Dashboard API | 8080 | http://localhost:8080 | Main API gateway |
| Dashboard UI | 3000 | http://localhost:3000 | Web interface |
| Telemetry Collector | 8085 | http://localhost:8085/api/v1/ves | VES events |
| Analytics API | 8088 | http://localhost:8088/api/v1 | Analytics data |
| Grafana | 3000 | http://localhost:3000 | Monitoring dashboards |
| Prometheus | 9092 | http://localhost:9092 | Metrics collection |

## Troubleshooting

### Common Issues
```bash
# Check service health
make validate

# View service logs
docker-compose logs -f

# Check Kubernetes pods
kubectl get pods -n oran

# Debug specific pod
kubectl describe pod -n oran <pod-name>
```

### Getting Help
- **Documentation**: Check the comprehensive guides in `/docs`
- **Issues**: Open an issue on GitHub with detailed logs
- **Development**: See [Development Workflow Guide](docs/development/workflow-guide.md)

## Contributing

1. **Read the Development Guide**: [docs/development/workflow-guide.md](docs/development/workflow-guide.md)
2. **Follow Coding Standards**: Use `make lint` and `make fmt`
3. **Write Tests**: Maintain >80% coverage
4. **Security First**: Run `make security-scan` before commits
5. **Documentation**: Update relevant documentation

For detailed contribution guidelines, see our [Development Workflow Guide](docs/development/workflow-guide.md).

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Support

For support and questions:
- **Documentation**: Comprehensive guides in `/docs`
- **Issues**: GitHub issue tracker
- **Development**: Follow the development workflow guide
