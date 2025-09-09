#!/bin/bash

# Comprehensive O-RAN L Release and Nephio R5 Test Execution Script
# This script orchestrates the complete testing suite with proper environment setup

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_CONFIG_DIR="$PROJECT_ROOT/test-config"
RESULTS_DIR="$PROJECT_ROOT/test-results"
LOGS_DIR="$RESULTS_DIR/logs"

# Default configuration
TEST_ENVIRONMENT="${TEST_ENVIRONMENT:-local}"
TEST_SUITES="${TEST_SUITES:-all}"
PARALLEL_EXECUTION="${PARALLEL_EXECUTION:-true}"
CONTINUE_ON_ERROR="${CONTINUE_ON_ERROR:-true}"
TEST_TIMEOUT="${TEST_TIMEOUT:-2h}"
MIN_COVERAGE="${MIN_COVERAGE:-80.0}"
REPORT_FORMAT="${REPORT_FORMAT:-json,html}"
LOG_LEVEL="${LOG_LEVEL:-info}"
DRY_RUN="${DRY_RUN:-false}"
CLEANUP="${CLEANUP:-true}"

# Color output functions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_debug() {
    if [[ "${LOG_LEVEL}" == "debug" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1" >&2
    fi
}

# Help function
show_help() {
    cat << EOF
Comprehensive O-RAN L Release and Nephio R5 Test Suite

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -e, --environment       Test environment (local, k8s, cloud) [default: local]
    -s, --test-suites       Test suites to run (e2e,load,nephio,compliance,all) [default: all]
    -p, --parallel          Run test suites in parallel (true/false) [default: true]
    -c, --continue-on-error Continue on test failures (true/false) [default: true]
    -t, --timeout           Maximum test execution timeout [default: 2h]
    -m, --min-coverage      Minimum required test coverage percentage [default: 80.0]
    -f, --report-format     Comma-separated list of report formats [default: json,html]
    -l, --log-level         Log level (debug,info,warn,error) [default: info]
    -d, --dry-run           Perform dry run without executing tests [default: false]
    -n, --no-cleanup        Skip cleanup after tests [default: false]
    -h, --help              Show this help message

ENVIRONMENT VARIABLES:
    TEST_ENVIRONMENT        Same as --environment
    TEST_SUITES            Same as --test-suites
    PARALLEL_EXECUTION     Same as --parallel
    CONTINUE_ON_ERROR      Same as --continue-on-error
    TEST_TIMEOUT           Same as --timeout
    MIN_COVERAGE           Same as --min-coverage
    REPORT_FORMAT          Same as --report-format
    LOG_LEVEL              Same as --log-level
    DRY_RUN                Same as --dry-run
    CLEANUP                Opposite of --no-cleanup

EXAMPLES:
    # Run all tests with default configuration
    $0

    # Run only E2E and load tests in development environment
    $0 --environment dev --test-suites e2e,load

    # Run tests with high verbosity and generate only JSON reports
    $0 --log-level debug --report-format json

    # Dry run to validate configuration
    $0 --dry-run

    # Run stress tests with extended timeout
    $0 --test-suites load --timeout 4h --parallel false

    # Run compliance tests for production readiness
    $0 --test-suites compliance --min-coverage 95.0

EOF
}

# Parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                TEST_ENVIRONMENT="$2"
                shift 2
                ;;
            -s|--test-suites)
                TEST_SUITES="$2"
                shift 2
                ;;
            -p|--parallel)
                PARALLEL_EXECUTION="$2"
                shift 2
                ;;
            -c|--continue-on-error)
                CONTINUE_ON_ERROR="$2"
                shift 2
                ;;
            -t|--timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            -m|--min-coverage)
                MIN_COVERAGE="$2"
                shift 2
                ;;
            -f|--report-format)
                REPORT_FORMAT="$2"
                shift 2
                ;;
            -l|--log-level)
                LOG_LEVEL="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -n|--no-cleanup)
                CLEANUP="false"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."

    # Check Go installation
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_GO_VERSION="1.24"
    if ! printf '%s\n%s\n' "$REQUIRED_GO_VERSION" "$GO_VERSION" | sort -V -C; then
        log_error "Go version $GO_VERSION is below required version $REQUIRED_GO_VERSION"
        exit 1
    fi
    log_info "Go version $GO_VERSION is compatible"

    # Check kubectl if Kubernetes tests are enabled
    if [[ "$TEST_ENVIRONMENT" == "k8s" || "$TEST_ENVIRONMENT" == "cloud" ]]; then
        if ! command -v kubectl &> /dev/null; then
            log_error "kubectl is required for Kubernetes tests but not found"
            exit 1
        fi

        # Test kubectl connectivity
        if ! kubectl cluster-info &> /dev/null; then
            log_error "kubectl cannot connect to Kubernetes cluster"
            exit 1
        fi
        log_info "Kubernetes cluster connectivity verified"
    fi

    # Check if Nephio tests are enabled
    if [[ "$TEST_SUITES" == "all" || "$TEST_SUITES" == *"nephio"* ]]; then
        if ! command -v kpt &> /dev/null; then
            log_warn "kpt is recommended for Nephio tests but not found"
        fi
    fi

    # Check Docker if container tests are needed
    if [[ "$TEST_ENVIRONMENT" != "local" ]]; then
        if ! command -v docker &> /dev/null; then
            log_warn "Docker is recommended for non-local environments but not found"
        fi
    fi

    log_info "Prerequisites validation completed"
}

# Setup test environment
setup_environment() {
    log_info "Setting up test environment: $TEST_ENVIRONMENT"

    # Create necessary directories
    mkdir -p "$RESULTS_DIR" "$LOGS_DIR" "$TEST_CONFIG_DIR"

    # Set Go environment for FIPS compliance (Go 1.24+)
    export GODEBUG="fips140=on"
    log_info "Enabled FIPS 140 compliance for Go runtime"

    case "$TEST_ENVIRONMENT" in
        local)
            setup_local_environment
            ;;
        k8s)
            setup_kubernetes_environment
            ;;
        cloud)
            setup_cloud_environment
            ;;
        *)
            log_error "Unsupported test environment: $TEST_ENVIRONMENT"
            exit 1
            ;;
    esac
}

setup_local_environment() {
    log_info "Configuring local test environment"
    
    # Start local services if needed (mock services)
    if pgrep -f "mock-e2term" > /dev/null; then
        log_info "Mock E2 Term service already running"
    else
        log_info "Starting mock E2 Term service"
        nohup go run "$PROJECT_ROOT/test/mock-services/e2term/main.go" > "$LOGS_DIR/mock-e2term.log" 2>&1 &
        sleep 2
    fi
}

setup_kubernetes_environment() {
    log_info "Configuring Kubernetes test environment"
    
    # Apply test namespace and resources
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: oran-test
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: oran-test
data:
  environment: "k8s"
  test-mode: "integration"
EOF

    # Wait for namespace to be ready
    kubectl wait --for=condition=Active namespace/oran-test --timeout=30s

    log_info "Kubernetes test environment ready"
}

setup_cloud_environment() {
    log_info "Configuring cloud test environment"
    
    # Cloud-specific setup (AWS, GCP, Azure)
    if [[ -n "${AWS_REGION:-}" ]]; then
        log_info "Detected AWS environment"
        # AWS-specific setup
    elif [[ -n "${GOOGLE_CLOUD_PROJECT:-}" ]]; then
        log_info "Detected GCP environment"
        # GCP-specific setup
    elif [[ -n "${AZURE_SUBSCRIPTION_ID:-}" ]]; then
        log_info "Detected Azure environment"
        # Azure-specific setup
    else
        log_warn "Cloud provider not detected, using generic cloud setup"
    fi
}

# Generate test configuration
generate_test_config() {
    log_info "Generating test configuration for environment: $TEST_ENVIRONMENT"
    
    local config_file="$TEST_CONFIG_DIR/test-config.json"
    
    cat > "$config_file" << EOF
{
  "e2eConfig": {
    "e2TermEndpoint": "$(get_endpoint "e2term" "36421")",
    "e2MgrEndpoint": "$(get_endpoint "e2mgr" "3800")",
    "subMgrEndpoint": "$(get_endpoint "submgr" "3801")",
    "a1MediatorURL": "$(get_endpoint "a1mediator" "9001")",
    "o1MediatorURL": "$(get_endpoint "o1mediator" "8080")",
    "o2CloudAPI": "$(get_endpoint "ocloud" "8080")",
    "porchEndpoint": "$(get_endpoint "porch" "7007")",
    "maxConcurrentE2Nodes": 100,
    "testDuration": "30m",
    "coverageThreshold": $MIN_COVERAGE,
    "namespace": "$(get_test_namespace)",
    "reportOutputDir": "$RESULTS_DIR"
  },
  "loadTestConfig": {
    "maxConcurrentE2Nodes": 100,
    "testDuration": "15m",
    "rampUpDuration": "5m",
    "rampDownDuration": "2m",
    "requestsPerSecond": 1000,
    "maxBurstSize": 2000,
    "maxLatencyP99": "100ms",
    "maxLatencyP95": "50ms",
    "minThroughputMbps": 1000.0,
    "maxErrorRate": 1.0,
    "maxCpuUtilization": 80.0,
    "maxMemoryUtilization": 80.0,
    "e2TermEndpoint": "$(get_endpoint "e2term" "36421")",
    "e2MgrEndpoint": "$(get_endpoint "e2mgr" "3800")",
    "dashboardEndpoint": "$(get_endpoint "dashboard" "3000")",
    "reportPath": "$RESULTS_DIR/load-test-report.json"
  },
  "nephioR5Config": {
    "porchAPIEndpoint": "$(get_endpoint "porch-server" "7007")",
    "packageRegistryURL": "https://github.com/nephio-project/catalog",
    "gitOpsRepoURL": "https://github.com/nephio-project/nephio-test",
    "configSyncEndpoint": "$(get_endpoint "config-sync" "8080")",
    "targetNamespace": "$(get_test_namespace)",
    "workloadClusters": [
      {
        "name": "edge-cluster-1",
        "endpoint": "https://edge-1.example.com",
        "region": "us-west-1",
        "provider": "aws",
        "capabilities": ["5g-ran", "edge-compute"],
        "labels": {
          "cluster-type": "edge",
          "region": "us-west-1"
        }
      }
    ],
    "packageRepository": "oci://registry.nephio.org/packages",
    "testTimeout": "30m",
    "deploymentTimeout": "10m"
  },
  "complianceConfig": {
    "e2TermEndpoint": "$(get_endpoint "e2term" "36421")",
    "e2MgrEndpoint": "$(get_endpoint "e2mgr" "3800")",
    "subMgrEndpoint": "$(get_endpoint "submgr" "3801")",
    "a1MediatorURL": "$(get_endpoint "a1mediator" "9001")",
    "o1MediatorURL": "$(get_endpoint "o1mediator" "8080")",
    "timeout": "30s",
    "retryAttempts": 3,
    "testDataPath": "$PROJECT_ROOT/test-data",
    "reportOutputPath": "$RESULTS_DIR/compliance-report.json"
  },
  "parallelExecution": $PARALLEL_EXECUTION,
  "continueOnFailure": $CONTINUE_ON_ERROR,
  "maxRetries": 3,
  "testTimeout": "$TEST_TIMEOUT",
  "minCoveragePercent": $MIN_COVERAGE,
  "maxFailureRate": 5.0,
  "qualityGates": [
    {
      "name": "Test Pass Rate",
      "type": "threshold",
      "metric": "overall_pass_rate",
      "threshold": 95.0,
      "operator": ">=",
      "severity": "critical",
      "failureAction": "fail_build"
    },
    {
      "name": "Test Coverage",
      "type": "threshold",
      "metric": "coverage_percent",
      "threshold": $MIN_COVERAGE,
      "operator": ">=",
      "severity": "high",
      "failureAction": "warning"
    }
  ],
  "outputDirectory": "$RESULTS_DIR",
  "reportFormats": ["$(echo "$REPORT_FORMAT" | sed 's/,/","/g')"],
  "environmentType": "$TEST_ENVIRONMENT",
  "testLabels": {
    "release": "o-ran-l",
    "nephio": "r5",
    "environment": "$TEST_ENVIRONMENT"
  }
}
EOF

    log_info "Test configuration generated: $config_file"
}

# Helper functions for configuration generation
get_endpoint() {
    local service="$1"
    local port="$2"
    
    case "$TEST_ENVIRONMENT" in
        local)
            echo "http://localhost:$port"
            ;;
        k8s)
            echo "http://$service.oran-test:$port"
            ;;
        cloud)
            # For cloud environments, endpoints might be external
            echo "https://$service.example.com"
            ;;
    esac
}

get_test_namespace() {
    case "$TEST_ENVIRONMENT" in
        local)
            echo "default"
            ;;
        k8s|cloud)
            echo "oran-test"
            ;;
    esac
}

# Build test orchestrator
build_test_orchestrator() {
    log_info "Building test orchestrator..."
    
    cd "$PROJECT_ROOT"
    
    # Build with proper flags for production readiness
    local build_flags="-ldflags=-s -w"
    if [[ "$TEST_ENVIRONMENT" != "local" ]]; then
        build_flags="$build_flags -tags netgo -installsuffix netgo"
    fi
    
    go build $build_flags -o "$RESULTS_DIR/test-orchestrator" ./cmd/test-orchestrator/
    
    if [[ ! -x "$RESULTS_DIR/test-orchestrator" ]]; then
        log_error "Failed to build test orchestrator"
        exit 1
    fi
    
    log_info "Test orchestrator built successfully"
}

# Execute tests
execute_tests() {
    log_info "Starting comprehensive test execution..."
    
    local config_file="$TEST_CONFIG_DIR/test-config.json"
    local orchestrator_bin="$RESULTS_DIR/test-orchestrator"
    local log_file="$LOGS_DIR/test-execution.log"
    
    # Prepare orchestrator arguments
    local args=(
        "--config" "$config_file"
        "--output-dir" "$RESULTS_DIR"
        "--log-level" "$LOG_LEVEL"
        "--report-format" "$REPORT_FORMAT"
        "--timeout" "$TEST_TIMEOUT"
        "--min-coverage" "$MIN_COVERAGE"
        "--test-suites" "$TEST_SUITES"
    )
    
    if [[ "$PARALLEL_EXECUTION" == "true" ]]; then
        args+=("--parallel")
    fi
    
    if [[ "$CONTINUE_ON_ERROR" == "true" ]]; then
        args+=("--continue-on-error")
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        args+=("--dry-run")
    fi
    
    if [[ "$LOG_LEVEL" == "debug" ]]; then
        args+=("--verbose")
    fi
    
    log_info "Executing: $orchestrator_bin ${args[*]}"
    
    # Execute with timeout and logging
    local exit_code=0
    timeout "$TEST_TIMEOUT" "$orchestrator_bin" "${args[@]}" 2>&1 | tee "$log_file" || exit_code=$?
    
    if [[ $exit_code -eq 124 ]]; then
        log_error "Test execution timed out after $TEST_TIMEOUT"
        return 1
    elif [[ $exit_code -ne 0 ]]; then
        log_error "Test execution failed with exit code: $exit_code"
        return $exit_code
    fi
    
    log_info "Test execution completed successfully"
    return 0
}

# Generate summary report
generate_summary() {
    log_info "Generating test execution summary..."
    
    local summary_file="$RESULTS_DIR/execution-summary.md"
    
    cat > "$summary_file" << EOF
# O-RAN L Release and Nephio R5 Test Execution Summary

**Execution Date:** $(date)
**Environment:** $TEST_ENVIRONMENT
**Test Suites:** $TEST_SUITES
**Parallel Execution:** $PARALLEL_EXECUTION

## Configuration
- **Test Timeout:** $TEST_TIMEOUT
- **Minimum Coverage:** $MIN_COVERAGE%
- **Report Formats:** $REPORT_FORMAT
- **Continue on Error:** $CONTINUE_ON_ERROR

## Results
$(if [[ -f "$RESULTS_DIR/test-report-"*".json" ]]; then
    echo "✅ Test reports generated successfully"
    echo ""
    echo "### Generated Reports"
    find "$RESULTS_DIR" -name "*.json" -o -name "*.html" -o -name "*.xml" | sed 's|^|- |'
else
    echo "❌ No test reports found"
fi)

## Logs
- **Execution Log:** $LOGS_DIR/test-execution.log
$(if [[ -f "$LOGS_DIR/mock-e2term.log" ]]; then
    echo "- **Mock E2 Term Log:** $LOGS_DIR/mock-e2term.log"
fi)

## Coverage Analysis
$(if [[ -f "$RESULTS_DIR/coverage.out" ]]; then
    echo "Coverage report available at: $RESULTS_DIR/coverage.out"
    go tool cover -func="$RESULTS_DIR/coverage.out" | tail -1
else
    echo "Coverage analysis will be available after test completion"
fi)

## Next Steps
1. Review test reports in \`$RESULTS_DIR\`
2. Address any failed test cases
3. Verify coverage meets minimum requirements
4. Update documentation based on findings

EOF

    log_info "Summary report generated: $summary_file"
}

# Cleanup function
cleanup_environment() {
    if [[ "$CLEANUP" != "true" ]]; then
        log_info "Skipping cleanup (disabled)"
        return 0
    fi
    
    log_info "Cleaning up test environment..."
    
    # Stop mock services
    if pgrep -f "mock-e2term" > /dev/null; then
        log_info "Stopping mock E2 Term service"
        pkill -f "mock-e2term" || true
    fi
    
    # Kubernetes cleanup
    if [[ "$TEST_ENVIRONMENT" == "k8s" ]]; then
        log_info "Cleaning up Kubernetes test resources"
        kubectl delete namespace oran-test --ignore-not-found=true || true
    fi
    
    # Clean up temporary files older than 7 days
    find "$RESULTS_DIR" -type f -mtime +7 -name "*.log" -delete 2>/dev/null || true
    
    log_info "Cleanup completed"
}

# Signal handlers
trap_exit() {
    local exit_code=$?
    log_info "Received exit signal, cleaning up..."
    cleanup_environment
    exit $exit_code
}

trap trap_exit EXIT
trap 'log_info "Received interrupt signal, shutting down gracefully..."; exit 130' INT TERM

# Main execution function
main() {
    local start_time=$(date +%s)
    
    log_info "Starting O-RAN L Release and Nephio R5 comprehensive test suite"
    log_info "Timestamp: $(date)"
    log_info "Environment: $TEST_ENVIRONMENT"
    log_info "Test Suites: $TEST_SUITES"
    
    # Execute test phases
    validate_prerequisites
    setup_environment
    generate_test_config
    build_test_orchestrator
    
    local test_exit_code=0
    execute_tests || test_exit_code=$?
    
    generate_summary
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log_info "Test execution completed in $(date -d@$duration -u +%H:%M:%S)"
    
    if [[ $test_exit_code -eq 0 ]]; then
        log_info "✅ All tests completed successfully!"
        echo ""
        echo "📊 Test results are available in: $RESULTS_DIR"
        echo "📝 Execution summary: $RESULTS_DIR/execution-summary.md"
        echo "📋 Logs directory: $LOGS_DIR"
    else
        log_error "❌ Tests completed with failures (exit code: $test_exit_code)"
        echo ""
        echo "Please review the test reports and logs for details."
    fi
    
    return $test_exit_code
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_arguments "$@"
    main
fi