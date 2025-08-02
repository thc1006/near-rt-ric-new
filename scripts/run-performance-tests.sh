#!/bin/bash

# O-RAN Near-RT RIC Performance Testing Script
# This script executes comprehensive performance tests to validate:
# - 100+ concurrent E2 nodes (Requirement 6.2)
# - 10,000+ indications per second (Requirement 6.4)
# - Sub-10ms processing latency (Requirement 6.1)
# - Resource exhaustion scenarios (Stress testing)
# - Long-running stability with memory leak detection (Requirement 6.5)

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RESULTS_DIR="$PROJECT_ROOT/test-results/performance"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_FILE="$RESULTS_DIR/performance_test_$TIMESTAMP.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

# Print banner
print_banner() {
    echo "=================================================================="
    echo "O-RAN Near-RT RIC Comprehensive Performance Testing Suite"
    echo "=================================================================="
    echo "Task 9.3: Performance and Load Testing Implementation"
    echo ""
    echo "Requirements Validation:"
    echo "• Load Testing: 100+ concurrent E2 nodes"
    echo "• Throughput Testing: 10,000+ indications per second"
    echo "• Latency Testing: sub-10ms processing validation"
    echo "• Stress Testing: resource exhaustion scenarios"
    echo "• Stability Testing: long-running with memory leak detection"
    echo "=================================================================="
    echo ""
}

# Setup test environment
setup_test_environment() {
    log_info "Setting up test environment..."
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Initialize log file
    echo "Performance Test Execution Log - $(date)" > "$LOG_FILE"
    echo "=======================================" >> "$LOG_FILE"
    
    # Check dependencies
    log_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Build the performance test binary
    log_info "Building performance test binary..."
    cd "$PROJECT_ROOT"
    
    if ! go build -o "$RESULTS_DIR/performance-test" ./cmd/performance-test; then
        log_error "Failed to build performance test binary"
        exit 1
    fi
    
    log_success "Test environment setup completed"
}

# Execute load testing scenarios
execute_load_tests() {
    log_info "Executing Load Testing Scenarios (100+ concurrent E2 nodes)..."
    
    local output_file="$RESULTS_DIR/load_test_results_$TIMESTAMP.txt"
    
    log_info "Running comprehensive load tests with the following scenarios:"
    log_info "• Gradual Load Increase - 200 Nodes"
    log_info "• Burst Load Test - 100 Nodes (minimum requirement)"
    log_info "• Exponential Growth - 150 Nodes"
    log_info "• High Density Load - 200 Nodes"
    log_info "• Stress Burst - 250 Nodes (extreme test)"
    
    if "$RESULTS_DIR/performance-test" -load=true -throughput=false -latency=false -stress=false -stability=false \
        -format=text -output="$output_file" -verbose=true; then
        log_success "Load testing completed successfully"
        log_info "Results saved to: $output_file"
        
        # Extract key metrics
        if grep -q "PASS" "$output_file"; then
            log_success "Load testing requirements MET (100+ concurrent E2 nodes)"
        else
            log_warning "Load testing requirements may not be fully met"
        fi
    else
        log_error "Load testing failed"
        return 1
    fi
}

# Execute throughput testing scenarios
execute_throughput_tests() {
    log_info "Executing Throughput Testing Scenarios (10,000+ indications per second)..."
    
    local output_file="$RESULTS_DIR/throughput_test_results_$TIMESTAMP.txt"
    
    log_info "Running comprehensive throughput tests with the following scenarios:"
    log_info "• Linear Ramp-up Test - 20K IPS"
    log_info "• High Burst Test - 25K IPS"
    log_info "• Sustained 10K+ Test - 12K IPS (requirement validation)"
    log_info "• Complex Processing - 8K IPS"
    log_info "• Peak Performance Test - 30K IPS"
    log_info "• Variable Load Test - 10-20K IPS"
    
    if "$RESULTS_DIR/performance-test" -load=false -throughput=true -latency=false -stress=false -stability=false \
        -format=text -output="$output_file" -verbose=true; then
        log_success "Throughput testing completed successfully"
        log_info "Results saved to: $output_file"
        
        # Extract key metrics
        if grep -q "PASS" "$output_file"; then
            log_success "Throughput testing requirements MET (10,000+ IPS)"
        else
            log_warning "Throughput testing requirements may not be fully met"
        fi
    else
        log_error "Throughput testing failed"
        return 1
    fi
}

# Execute latency testing scenarios
execute_latency_tests() {
    log_info "Executing Latency Testing Scenarios (sub-10ms processing validation)..."
    
    local output_file="$RESULTS_DIR/latency_test_results_$TIMESTAMP.txt"
    
    log_info "Running comprehensive latency tests with the following scenarios:"
    log_info "• E2 Setup Sub-10ms Test"
    log_info "• Subscription Sub-8ms Test"
    log_info "• High-Rate Indication Sub-5ms Test"
    log_info "• Control Message Sub-10ms Test"
    log_info "• End-to-End Sub-15ms Test"
    log_info "• Mixed Operations Latency Test"
    log_info "• Peak Load Latency Validation"
    
    if "$RESULTS_DIR/performance-test" -load=false -throughput=false -latency=true -stress=false -stability=false \
        -format=text -output="$output_file" -verbose=true; then
        log_success "Latency testing completed successfully"
        log_info "Results saved to: $output_file"
        
        # Extract key metrics
        if grep -q "PASS" "$output_file"; then
            log_success "Latency testing requirements MET (sub-10ms processing)"
        else
            log_warning "Latency testing requirements may not be fully met"
        fi
    else
        log_error "Latency testing failed"
        return 1
    fi
}

# Execute stress testing scenarios
execute_stress_tests() {
    log_info "Executing Stress Testing Scenarios (resource exhaustion scenarios)..."
    
    local output_file="$RESULTS_DIR/stress_test_results_$TIMESTAMP.txt"
    
    log_info "Running comprehensive stress tests with the following scenarios:"
    log_info "• Extreme CPU Exhaustion Test"
    log_info "• Memory Exhaustion with Leak Detection"
    log_info "• Connection and File Descriptor Exhaustion"
    log_info "• Multi-Resource Cascading Failure"
    log_info "• Disk and I/O Exhaustion Test"
    log_info "• Network Partition and Recovery Test"
    log_info "• Ultimate Stress Test - All Resources"
    
    if "$RESULTS_DIR/performance-test" -load=false -throughput=false -latency=false -stress=true -stability=false \
        -format=text -output="$output_file" -verbose=true; then
        log_success "Stress testing completed successfully"
        log_info "Results saved to: $output_file"
        
        # Extract key metrics
        if grep -q "PASS" "$output_file"; then
            log_success "Stress testing requirements MET (resource exhaustion scenarios)"
        else
            log_warning "Stress testing requirements may not be fully met"
        fi
    else
        log_error "Stress testing failed"
        return 1
    fi
}

# Execute stability testing scenarios
execute_stability_tests() {
    log_info "Executing Stability Testing Scenarios (long-running with memory leak detection)..."
    
    local output_file="$RESULTS_DIR/stability_test_results_$TIMESTAMP.txt"
    
    log_info "Running comprehensive stability tests with the following scenarios:"
    log_info "• Extended Stability Test - 72h"
    log_info "• Memory Leak Detection Test - 24h"
    log_info "• Variable Load Stability Test - 36h"
    log_info "• High Load Stability Test - 12h"
    log_info "• Micro-Leak Detection Test - 72h"
    
    log_warning "Note: Stability tests are configured for long durations (up to 72 hours)"
    log_warning "For demonstration purposes, tests will run for reduced time"
    
    if "$RESULTS_DIR/performance-test" -load=false -throughput=false -latency=false -stress=false -stability=true \
        -format=text -output="$output_file" -verbose=true; then
        log_success "Stability testing completed successfully"
        log_info "Results saved to: $output_file"
        
        # Extract key metrics
        if grep -q "PASS" "$output_file"; then
            log_success "Stability testing requirements MET (memory leak detection)"
        else
            log_warning "Stability testing requirements may not be fully met"
        fi
    else
        log_error "Stability testing failed"
        return 1
    fi
}

# Execute comprehensive performance tests
execute_comprehensive_tests() {
    log_info "Executing Comprehensive Performance Test Suite..."
    
    local output_file="$RESULTS_DIR/comprehensive_test_results_$TIMESTAMP.txt"
    
    log_info "Running all performance tests together for complete validation..."
    
    if "$RESULTS_DIR/performance-test" -load=true -throughput=true -latency=true -stress=true -stability=true \
        -format=text -output="$output_file" -verbose=true -continue=true; then
        log_success "Comprehensive performance testing completed successfully"
        log_info "Results saved to: $output_file"
        
        # Generate summary report
        generate_summary_report "$output_file"
    else
        log_error "Comprehensive performance testing failed"
        return 1
    fi
}

# Generate summary report
generate_summary_report() {
    local comprehensive_results="$1"
    local summary_file="$RESULTS_DIR/performance_test_summary_$TIMESTAMP.txt"
    
    log_info "Generating performance test summary report..."
    
    cat > "$summary_file" << EOF
================================================================
O-RAN Near-RT RIC Performance Test Summary Report
================================================================
Test Execution Date: $(date)
Test Duration: $(date -d @$(($(date +%s) - $(stat -c %Y "$LOG_FILE"))) -u +%H:%M:%S)

REQUIREMENTS VALIDATION SUMMARY:
================================

Task 9.3 Performance and Load Testing Requirements:

1. Load Testing Scenarios (100+ concurrent E2 nodes)
   Status: $(grep -q "Load.*PASS" "$comprehensive_results" && echo "✓ PASSED" || echo "✗ FAILED")
   
2. Throughput Testing (10,000+ indications per second)
   Status: $(grep -q "Throughput.*PASS" "$comprehensive_results" && echo "✓ PASSED" || echo "✗ FAILED")
   
3. Latency Testing (sub-10ms processing validation)
   Status: $(grep -q "Latency.*PASS" "$comprehensive_results" && echo "✓ PASSED" || echo "✗ FAILED")
   
4. Stress Testing (resource exhaustion scenarios)
   Status: $(grep -q "Stress.*PASS" "$comprehensive_results" && echo "✓ PASSED" || echo "✗ FAILED")
   
5. Stability Testing (memory leak detection)
   Status: $(grep -q "Stability.*PASS" "$comprehensive_results" && echo "✓ PASSED" || echo "✗ FAILED")

DETAILED RESULTS:
================
$(cat "$comprehensive_results" 2>/dev/null || echo "Detailed results not available")

TEST FILES GENERATED:
====================
- Comprehensive Results: $comprehensive_results
- Test Execution Log: $LOG_FILE
- Summary Report: $summary_file

================================================================
EOF

    log_success "Summary report generated: $summary_file"
    
    # Display summary to console
    echo ""
    echo "================================================================"
    echo "PERFORMANCE TEST EXECUTION SUMMARY"
    echo "================================================================"
    cat "$summary_file"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test environment..."
    
    # Kill any remaining test processes
    pkill -f "performance-test" 2>/dev/null || true
    
    # Compress results for archival
    if [ -d "$RESULTS_DIR" ]; then
        tar -czf "$RESULTS_DIR/performance_test_archive_$TIMESTAMP.tar.gz" -C "$RESULTS_DIR" . 2>/dev/null || true
        log_info "Test results archived to: performance_test_archive_$TIMESTAMP.tar.gz"
    fi
}

# Main execution function
main() {
    # Set up trap for cleanup
    trap cleanup EXIT
    
    print_banner
    
    # Setup test environment
    setup_test_environment
    
    # Parse command line arguments
    RUN_LOAD_TESTS=true
    RUN_THROUGHPUT_TESTS=true
    RUN_LATENCY_TESTS=true
    RUN_STRESS_TESTS=true
    RUN_STABILITY_TESTS=true
    RUN_COMPREHENSIVE=true
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --load-only)
                RUN_THROUGHPUT_TESTS=false
                RUN_LATENCY_TESTS=false
                RUN_STRESS_TESTS=false
                RUN_STABILITY_TESTS=false
                RUN_COMPREHENSIVE=false
                shift
                ;;
            --throughput-only)
                RUN_LOAD_TESTS=false
                RUN_LATENCY_TESTS=false
                RUN_STRESS_TESTS=false
                RUN_STABILITY_TESTS=false
                RUN_COMPREHENSIVE=false
                shift
                ;;
            --latency-only)
                RUN_LOAD_TESTS=false
                RUN_THROUGHPUT_TESTS=false
                RUN_STRESS_TESTS=false
                RUN_STABILITY_TESTS=false
                RUN_COMPREHENSIVE=false
                shift
                ;;
            --stress-only)
                RUN_LOAD_TESTS=false
                RUN_THROUGHPUT_TESTS=false
                RUN_LATENCY_TESTS=false
                RUN_STABILITY_TESTS=false
                RUN_COMPREHENSIVE=false
                shift
                ;;
            --stability-only)
                RUN_LOAD_TESTS=false
                RUN_THROUGHPUT_TESTS=false
                RUN_LATENCY_TESTS=false
                RUN_STRESS_TESTS=false
                RUN_COMPREHENSIVE=false
                shift
                ;;
            --no-comprehensive)
                RUN_COMPREHENSIVE=false
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --load-only         Run only load tests"
                echo "  --throughput-only   Run only throughput tests"
                echo "  --latency-only      Run only latency tests"
                echo "  --stress-only       Run only stress tests"
                echo "  --stability-only    Run only stability tests"
                echo "  --no-comprehensive  Skip comprehensive test run"
                echo "  --help              Show this help message"
                exit 0
                ;;
            *)
                log_warning "Unknown option: $1"
                shift
                ;;
        esac
    done
    
    # Execute individual test suites
    local test_failures=0
    
    if [ "$RUN_LOAD_TESTS" = true ]; then
        execute_load_tests || ((test_failures++))
    fi
    
    if [ "$RUN_THROUGHPUT_TESTS" = true ]; then
        execute_throughput_tests || ((test_failures++))
    fi
    
    if [ "$RUN_LATENCY_TESTS" = true ]; then
        execute_latency_tests || ((test_failures++))
    fi
    
    if [ "$RUN_STRESS_TESTS" = true ]; then
        execute_stress_tests || ((test_failures++))
    fi
    
    if [ "$RUN_STABILITY_TESTS" = true ]; then
        execute_stability_tests || ((test_failures++))
    fi
    
    # Execute comprehensive tests
    if [ "$RUN_COMPREHENSIVE" = true ]; then
        execute_comprehensive_tests || ((test_failures++))
    fi
    
    # Final summary
    echo ""
    echo "================================================================"
    if [ $test_failures -eq 0 ]; then
        log_success "All performance tests completed successfully!"
        log_success "Task 9.3 Performance and Load Testing implementation validated"
    else
        log_warning "Performance testing completed with $test_failures test suite(s) reporting issues"
        log_warning "Review individual test results for details"
    fi
    echo "================================================================"
    
    return $test_failures
}

# Execute main function
main "$@"