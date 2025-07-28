# Technology Stack

## Backend

- **Language**: Go 1.21
- **Framework**: Standard Go with gRPC for service communication
- **Key Dependencies**:
  - `github.com/onosproject/onos-ric-sdk-go` - O-RAN RIC SDK
  - `github.com/onosproject/onos-lib-go` - ONOS library utilities
  - `google.golang.org/grpc` - gRPC communication

## Frontend

- **Framework**: React 19.1.0
- **Build Tool**: Create React App (react-scripts 5.0.1)
- **Testing**: Jest with React Testing Library
- **Key Dependencies**:
  - Standard React ecosystem
  - Web Vitals for performance monitoring

## Infrastructure & Deployment

- **Containerization**: Docker
- **Orchestration**: Kubernetes (KIND, K3s, Minikube supported)
- **Package Management**: Helm 3.11.2+
- **Chart Repository**: ChartMuseum for local development

## Observability Stack

- **Metrics**: Prometheus
- **Visualization**: Grafana
- **Logging**: Loki, Elasticsearch
- **Health Monitoring**: Custom health check scripts

## Build System

The project uses Make for build automation. Key commands:

### Development Commands
```bash
# Build Go binaries
make build

# Run tests with coverage
make test-coverage

# Format and lint code
make fmt
make lint

# Setup development environment (installs Helm, ChartMuseum)
make setup-dev-env
```

### Docker Commands
```bash
# Build xApp Docker image
make docker-build-xapp-hello-world
```

### Helm Commands
```bash
# Lint all Helm charts
make helm-lint

# Build and push RIC charts
make build-ric-charts

# Deploy interactive dashboard
make deploy-interactive-dashboard
```

### End-to-End Testing
```bash
# Complete deployment and testing
make e2e
```

### Frontend Commands
```bash
# Navigate to UI directory first
cd ui

# Install dependencies
npm install

# Start development server
npm start

# Run tests
npm test

# Build for production
npm build
```

## Development Workflow

1. Use `make setup-dev-env` for initial setup
2. Use `make build` to compile Go services
3. Use `make test-coverage` to run tests
4. Use `make e2e` for full integration testing
5. Frontend development uses standard React workflow in `ui/` directory