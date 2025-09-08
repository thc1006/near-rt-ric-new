# O-RAN L Release and Nephio R5 Testing Guide

This guide provides comprehensive information for testing O-RAN L Release and Nephio R5 deployments with end-to-end validation, performance testing, and compliance verification.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Test Architecture](#test-architecture)
- [Test Suites](#test-suites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Running Tests](#running-tests)
- [Test Results](#test-results)
- [Troubleshooting](#troubleshooting)
- [Advanced Usage](#advanced-usage)

## Overview

The comprehensive testing framework provides:

- **E2E Integration Testing**: Complete workflow testing for O-RAN interfaces
- **Load and Performance Testing**: Scale testing with 100+ concurrent E2 nodes
- **Nephio R5 Integration**: Package management and multi-cluster deployment testing
- **Compliance Testing**: O-RAN.WG3.E2AP-R003 and O-RAN.WG2.A1 specification compliance
- **Security Testing**: Authentication, authorization, and encryption validation
- **Interoperability Testing**: Third-party component integration testing

### Key Features

- **Phase 9 Integration Testing**: Complete integration test coverage as per O-RAN requirements
- **>80% Test Coverage**: Automated code coverage measurement and reporting
- **Go 1.24.6 Support**: FIPS 140 compliance and latest Go features
- **Multi-Environment**: Local, Kubernetes, and cloud deployment testing
- **Comprehensive Reporting**: JSON, HTML, XML, and JUnit report formats

## Prerequisites

### System Requirements

- **Go**: Version 1.24.6 or later with FIPS 140 support
- **Kubernetes**: Version 1.26+ for K8s testing
- **Docker**: For containerized testing (optional)
- **Git**: For GitOps integration testing
- **kubectl**: For Kubernetes operations
- **kpt**: For Nephio package management (recommended)

### Environment Setup

```bash
# Install Go 1.24.6 (with FIPS support)
curl -LO https://golang.org/dl/go1.24.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Enable FIPS 140 mode
export GODEBUG=fips140=on

# Verify installation
go version
```

### Network Requirements

The testing framework requires access to:
- O-RAN component endpoints (E2 Term, E2 Manager, etc.)
- Kubernetes API server
- Package repositories (for Nephio testing)
- GitOps repositories

## Test Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                Test Orchestrator                           │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│ │    E2E      │ │    Load     │ │   Nephio    │ │ Compliance  │ │
│ │    Tests    │ │    Tests    │ │  R5 Tests   │ │    Tests    │ │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                     Test Clients                           │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│ │E2 Term  │ │E2 Mgr   │ │Sub Mgr  │ │A1 Med   │ │O1 Med   │   │
│ │Client   │ │Client   │ │Client   │ │Client   │ │Client   │   │
│ └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘   │
├─────────────────────────────────────────────────────────────┤
│                O-RAN Components                             │
└─────────────────────────────────────────────────────────────┘
```

## Test Suites

### 1. E2E Integration Tests

Tests complete workflows including:
- E2 Node onboarding and connection establishment
- E2 subscription lifecycle management
- A1 policy creation, distribution, and enforcement
- xApp deployment and operational testing
- SMO integration and coordination

### 2. Load and Performance Tests

- **Concurrent E2 Nodes**: Up to 100+ simultaneous connections
- **Throughput Testing**: Message processing performance
- **Latency Testing**: P99 and P95 latency measurements
- **Resource Utilization**: CPU, memory, and network monitoring
- **Stress Testing**: System behavior under extreme load

### 3. Nephio R5 Integration Tests

- **Package Management**: Porch package operations
- **GitOps Integration**: Configuration synchronization
- **Multi-Cluster Deployment**: Cross-cluster package deployment
- **Package Variants**: Configuration customization testing
- **Workflow Testing**: Complete CI/CD pipeline validation

### 4. Compliance Tests

- **E2AP Compliance**: O-RAN.WG3.E2AP-R003 specification testing
- **A1 Interface**: O-RAN.WG2.A1 specification compliance
- **O1 Management**: YANG model validation
- **Security Compliance**: Authentication and encryption testing

## Quick Start

### 1. Clone and Setup

```bash
# Clone repository
git clone <repository-url>
cd near-rt-ric-new

# Build test orchestrator
go build -o test-orchestrator ./cmd/test-orchestrator/
```

### 2. Run Basic Tests

```bash
# Run all tests with default configuration
./scripts/run-comprehensive-tests.sh

# Run specific test suites
./scripts/run-comprehensive-tests.sh --test-suites e2e,load

# Run in Kubernetes environment
./scripts/run-comprehensive-tests.sh --environment k8s

# Dry run to validate configuration
./scripts/run-comprehensive-tests.sh --dry-run
```

### 3. View Results

```bash
# Results are saved in test-results directory
ls -la test-results/

# View HTML report
open test-results/test-report-*.html

# View JSON results
cat test-results/test-report-*.json | jq '.'
```

## Configuration

### Configuration File Structure

The test configuration is defined in JSON format:

```json
{
  "e2eConfig": {
    "e2TermEndpoint": "http://e2term:36421",
    "maxConcurrentE2Nodes": 100,
    "testDuration": "30m",
    "coverageThreshold": 80.0
  },
  "loadTestConfig": {
    "requestsPerSecond": 1000,
    "maxLatencyP99": "100ms",
    "testDuration": "15m"
  },
  "nephioR5Config": {
    "porchAPIEndpoint": "http://porch-server:7007",
    "workloadClusters": [...]
  },
  "qualityGates": [
    {
      "name": "Test Pass Rate",
      "metric": "overall_pass_rate",
      "threshold": 95.0,
      "operator": ">=",
      "severity": "critical"
    }
  ]
}
```

### Environment-Specific Configuration

#### Local Development
```json
{
  "e2eConfig": {
    "e2TermEndpoint": "http://localhost:36421",
    "namespace": "default"
  }
}
```

#### Kubernetes Deployment
```json
{
  "e2eConfig": {
    "e2TermEndpoint": "http://e2term.oran:36421",
    "namespace": "oran",
    "kubernetesConfig": "/etc/kubeconfig"
  }
}
```

#### Production Environment
```json
{
  "qualityGates": [
    {
      "name": "Coverage Gate",
      "metric": "coverage_percent",
      "threshold": 90.0,
      "severity": "critical"
    }
  ],
  "minCoveragePercent": 90.0,
  "maxFailureRate": 1.0
}
```

## Running Tests

### Command Line Interface

```bash
# Basic execution
./test-orchestrator --config test-config.json

# Parallel execution
./test-orchestrator --parallel --test-suites all

# High coverage requirements
./test-orchestrator --min-coverage 90.0 --report-format json,html

# Extended timeout for load tests
./test-orchestrator --timeout 4h --test-suites load

# Development mode with verbose logging
./test-orchestrator --log-level debug --verbose
```

### Script-Based Execution

```bash
# Standard test execution
./scripts/run-comprehensive-tests.sh

# Custom environment and suites
TEST_ENVIRONMENT=k8s TEST_SUITES=e2e,compliance ./scripts/run-comprehensive-tests.sh

# Production readiness testing
./scripts/run-comprehensive-tests.sh \
  --min-coverage 95.0 \
  --continue-on-error false \
  --report-format json,html,junit
```

### CI/CD Integration

#### Jenkins Pipeline
```groovy
pipeline {
    agent any
    stages {
        stage('O-RAN Tests') {
            steps {
                sh '''
                    export GODEBUG=fips140=on
                    ./scripts/run-comprehensive-tests.sh \
                      --environment k8s \
                      --report-format junit,html \
                      --output-dir ${WORKSPACE}/test-results
                '''
            }
            post {
                always {
                    publishTestResults testResultsPattern: 'test-results/*.xml'
                    publishHTML([
                        allowMissing: false,
                        alwaysLinkToLastBuild: true,
                        keepAll: true,
                        reportDir: 'test-results',
                        reportFiles: '*.html',
                        reportName: 'O-RAN Test Report'
                    ])
                }
            }
        }
    }
}
```

#### GitHub Actions
```yaml
name: O-RAN Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24.6'
      
      - name: Run Tests
        env:
          GODEBUG: fips140=on
        run: |
          ./scripts/run-comprehensive-tests.sh \
            --test-suites e2e,compliance \
            --report-format json,junit
      
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: test-results/
```

## Test Results

### Result Structure

```
test-results/
├── test-report-20231215-143022.json    # Main test report
├── test-report-20231215-143022.html    # HTML report
├── load-test-report.json               # Load test details
├── nephio-test-report.json             # Nephio R5 results
├── compliance-report.json              # Compliance results
├── coverage.out                        # Go coverage data
├── execution-summary.md                # Summary report
└── logs/
    ├── test-execution.log              # Main execution log
    └── mock-services.log               # Mock service logs
```

### Key Metrics

#### Overall Test Metrics
- **Total Tests**: Number of test cases executed
- **Pass Rate**: Percentage of successful tests
- **Coverage**: Code coverage percentage
- **Execution Time**: Total test duration

#### Performance Metrics
- **Latency**: P99, P95, and average response times
- **Throughput**: Requests per second
- **Resource Utilization**: CPU, memory, network usage
- **Error Rate**: Percentage of failed requests

#### O-RAN Specific Metrics
- **E2 Nodes Connected**: Concurrent E2 node connections
- **Active Subscriptions**: Number of active E2 subscriptions
- **Policy Enforcements**: A1 policy enforcement count
- **xApps Deployed**: Successfully deployed xApplications

### Quality Gates

Quality gates ensure test quality and production readiness:

```json
{
  "qualityGates": [
    {
      "name": "Test Pass Rate",
      "threshold": 95.0,
      "operator": ">=",
      "severity": "critical"
    },
    {
      "name": "Test Coverage",
      "threshold": 85.0,
      "operator": ">=",
      "severity": "critical"
    },
    {
      "name": "P99 Latency",
      "threshold": 100.0,
      "operator": "<=",
      "severity": "high"
    }
  ]
}
```

## Troubleshooting

### Common Issues

#### Test Execution Failures

**Issue**: Tests fail with connection errors
```
Error: failed to connect to E2 Term endpoint
```

**Solution**:
1. Verify service endpoints are accessible
2. Check network connectivity
3. Validate service health status
4. Review service logs

```bash
# Check service status
kubectl get pods -n oran
kubectl logs deployment/e2term -n oran

# Test connectivity
curl http://e2term.oran:36421/health
```

#### Coverage Issues

**Issue**: Coverage below threshold
```
Coverage 75.2% below minimum requirement 80.0%
```

**Solution**:
1. Add missing unit tests
2. Remove untestable code
3. Update coverage thresholds

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# View uncovered code
go tool cover -func=coverage.out | grep -v 100.0%
```

#### Load Test Failures

**Issue**: High latency or error rates
```
P99 latency 250ms exceeds threshold 100ms
```

**Solution**:
1. Check system resources
2. Review database performance
3. Analyze network latency
4. Scale services if needed

```bash
# Check resource usage
kubectl top pods -n oran
kubectl describe hpa -n oran

# Review service metrics
kubectl port-forward svc/prometheus 9090:9090
```

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
# Enable debug logging
./test-orchestrator --log-level debug --verbose

# Use debug script
LOG_LEVEL=debug ./scripts/run-comprehensive-tests.sh
```

### Log Analysis

Important log locations:
- **Main execution log**: `test-results/logs/test-execution.log`
- **Individual test logs**: `test-results/logs/<test-suite>.log`
- **Service logs**: Available through kubectl logs

```bash
# View recent test execution
tail -f test-results/logs/test-execution.log

# Search for errors
grep -i error test-results/logs/*.log

# Analyze performance issues
grep -i latency test-results/logs/load-test.log
```

## Advanced Usage

### Custom Test Scenarios

Create custom test scenarios by extending the configuration:

```json
{
  "loadTestConfig": {
    "testScenarios": [
      {
        "name": "Custom E2 Test",
        "description": "Custom test scenario",
        "weight": 0.4,
        "requestTemplate": {
          "method": "POST",
          "endpoint": "/api/v1/custom",
          "body": {
            "customParameter": "value"
          }
        }
      }
    ]
  }
}
```

### Multi-Environment Testing

Test across multiple environments:

```bash
# Test sequence across environments
for env in local k8s cloud; do
  echo "Testing environment: $env"
  TEST_ENVIRONMENT=$env ./scripts/run-comprehensive-tests.sh
done
```

### Performance Benchmarking

Run performance benchmarks:

```bash
# Extended load test
./test-orchestrator \
  --test-suites load \
  --timeout 8h \
  --config performance-config.json

# Stress testing
TEST_SUITES=load \
PARALLEL_EXECUTION=false \
./scripts/run-comprehensive-tests.sh
```

### Compliance Automation

Automate compliance checking:

```bash
# Daily compliance check
crontab -e
0 2 * * * /path/to/run-comprehensive-tests.sh --test-suites compliance --report-format json
```

### Integration with Monitoring

Connect test results to monitoring systems:

```bash
# Send metrics to Prometheus
curl -X POST http://prometheus-pushgateway:9091/metrics/job/oran-tests \
  --data-binary @test-results/metrics.txt

# Create Grafana dashboard
# Use test-results/grafana-dashboard.json
```

## Best Practices

### Test Organization

- **Modular Tests**: Keep tests focused and independent
- **Environment Isolation**: Use separate namespaces for different test environments
- **Resource Management**: Clean up resources after tests
- **Data Management**: Use realistic but anonymized test data

### Performance Optimization

- **Parallel Execution**: Enable parallel testing where possible
- **Resource Allocation**: Ensure adequate resources for load testing
- **Test Duration**: Balance thoroughness with execution time
- **Caching**: Use caching for repeated operations

### CI/CD Integration

- **Quality Gates**: Define clear quality thresholds
- **Failure Handling**: Implement appropriate failure responses
- **Artifact Management**: Store and version test artifacts
- **Notification**: Set up alerts for test failures

### Security Considerations

- **Credentials Management**: Use secure credential storage
- **Network Security**: Implement proper network policies
- **Access Control**: Apply least privilege principles
- **Data Protection**: Protect sensitive test data

## Contributing

To contribute to the testing framework:

1. **Fork the repository**
2. **Create a feature branch**
3. **Add tests for new functionality**
4. **Ensure coverage requirements are met**
5. **Submit a pull request**

### Development Guidelines

- Follow Go coding standards
- Maintain test coverage above 85%
- Add documentation for new features
- Include integration tests for new components

## Support

For support and questions:

- **Documentation**: Check this guide and inline code documentation
- **Issues**: Create GitHub issues for bugs and feature requests
- **Discussions**: Use GitHub discussions for general questions
- **Community**: Join the O-RAN testing community forums

---

**Last Updated**: December 2024  
**Version**: O-RAN L Release / Nephio R5  
**Go Version**: 1.24.6 with FIPS 140 support