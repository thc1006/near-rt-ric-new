#!/bin/bash
# deployment-validation.sh - Comprehensive deployment validation and smoke testing

set -euo pipefail

# Configuration
NAMESPACE="oran-ric"
TIMEOUT=600
VALIDATION_LOG="/var/log/oran-ric-validation.log"

# Logging function
log_validation() {
    echo "$(date -Iseconds) [VALIDATION] $1" | tee -a "$VALIDATION_LOG"
}

# Wait for deployment to be ready
wait_for_deployment() {
    local deployment=$1
    local timeout=${2:-$TIMEOUT}
    
    log_validation "Waiting for deployment $deployment to be ready (timeout: ${timeout}s)"
    
    if kubectl wait --for=condition=available deployment/"$deployment" -n "$NAMESPACE" --timeout="${timeout}s"; then
        log_validation "✓ Deployment $deployment is ready"
        return 0
    else
        log_validation "✗ Deployment $deployment failed to become ready within ${timeout}s"
        return 1
    fi
}

# Check pod health
check_pod_health() {
    local component=$1
    local expected_replicas=${2:-1}
    
    log_validation "Checking pod health for $component"
    
    # Check if pods are running
    local running_pods=$(kubectl get pods -n "$NAMESPACE" -l app="$component" --field-selector=status.phase=Running --no-headers | wc -l)
    
    if [ "$running_pods" -ge "$expected_replicas" ]; then
        log_validation "✓ $component has $running_pods/$expected_replicas pods running"
    else
        log_validation "✗ $component has only $running_pods/$expected_replicas pods running"
        
        # Show pod details for debugging
        kubectl get pods -n "$NAMESPACE" -l app="$component"
        kubectl describe pods -n "$NAMESPACE" -l app="$component"
        return 1
    fi
    
    # Check pod readiness
    local ready_pods=$(kubectl get pods -n "$NAMESPACE" -l app="$component" -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -o True | wc -l)
    
    if [ "$ready_pods" -ge "$expected_replicas" ]; then
        log_validation "✓ $component has $ready_pods/$expected_replicas pods ready"
    else
        log_validation "✗ $component has only $ready_pods/$expected_replicas pods ready"
        return 1
    fi
    
    return 0
}

# Test service endpoints
test_service_endpoint() {
    local service=$1
    local port=$2
    local path=${3:-"/health"}
    local expected_status=${4:-200}
    
    log_validation "Testing service endpoint: $service:$port$path"
    
    # Test from within the cluster
    local test_pod=$(kubectl get pods -n "$NAMESPACE" -l app="$service" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    if [ -z "$test_pod" ]; then
        log_validation "✗ No pods found for service $service"
        return 1
    fi
    
    local response_code=$(kubectl exec -n "$NAMESPACE" "$test_pod" -- \
        curl -s -o /dev/null -w "%{http_code}" "http://localhost:$port$path" 2>/dev/null || echo "000")
    
    if [ "$response_code" = "$expected_status" ]; then
        log_validation "✓ Service $service endpoint returned $response_code"
        return 0
    else
        log_validation "✗ Service $service endpoint returned $response_code (expected $expected_status)"
        return 1
    fi
}

# Test database connectivity
test_database_connectivity() {
    log_validation "Testing database connectivity"
    
    # Test Redis connectivity
    if kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli ping 2>/dev/null | grep -q PONG; then
        log_validation "✓ Database (Redis) is responding"
    else
        log_validation "✗ Database (Redis) is not responding"
        return 1
    fi
    
    # Test database operations
    local test_key="validation-test-$(date +%s)"
    local test_value="test-value-$(date +%s)"
    
    # Write test
    if kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
       redis-cli set "$test_key" "$test_value" 2>/dev/null | grep -q OK; then
        log_validation "✓ Database write operation successful"
    else
        log_validation "✗ Database write operation failed"
        return 1
    fi
    
    # Read test
    local retrieved_value=$(kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
        redis-cli get "$test_key" 2>/dev/null || echo "")
    
    if [ "$retrieved_value" = "$test_value" ]; then
        log_validation "✓ Database read operation successful"
    else
        log_validation "✗ Database read operation failed (got: '$retrieved_value', expected: '$test_value')"
        return 1
    fi
    
    # Cleanup test
    kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
        redis-cli del "$test_key" >/dev/null 2>&1 || true
    
    return 0
}

# Test inter-component communication
test_inter_component_communication() {
    log_validation "Testing inter-component communication"
    
    local components=("e2mgr" "submgr" "a1mediator" "o1mediator")
    local communication_tests=(
        "e2mgr:submgr:8080"
        "submgr:e2term:8080"
        "e2mgr:dbaas:6379"
        "submgr:dbaas:6379"
    )
    
    for test in "${communication_tests[@]}"; do
        local source=$(echo "$test" | cut -d: -f1)
        local target=$(echo "$test" | cut -d: -f2)
        local port=$(echo "$test" | cut -d: -f3)
        
        log_validation "Testing communication: $source -> $target:$port"
        
        if kubectl exec -n "$NAMESPACE" deployment/"$source" -- \
           nc -z "$target.$NAMESPACE.svc.cluster.local" "$port" 2>/dev/null; then
            log_validation "✓ Communication successful: $source -> $target:$port"
        else
            log_validation "✗ Communication failed: $source -> $target:$port"
            return 1
        fi
    done
    
    return 0
}

# Test API functionality
test_api_functionality() {
    log_validation "Testing API functionality"
    
    # Test A1 API
    log_validation "Testing A1 API endpoints"
    
    # Health check
    if test_service_endpoint "a1mediator" "8080" "/a1-p/healthcheck" "200"; then
        log_validation "✓ A1 health endpoint working"
    else
        log_validation "✗ A1 health endpoint failed"
        return 1
    fi
    
    # Policy types endpoint
    local policy_response=$(kubectl exec -n "$NAMESPACE" deployment/a1mediator -- \
        curl -s "http://localhost:8080/a1-p/policytypes" 2>/dev/null || echo "")
    
    if [ -n "$policy_response" ]; then
        log_validation "✓ A1 policy types endpoint responding"
    else
        log_validation "✗ A1 policy types endpoint not responding"
        return 1
    fi
    
    # Test E2 Manager API
    log_validation "Testing E2 Manager API endpoints"
    
    if test_service_endpoint "e2mgr" "3800" "/v1/nodeb/states" "200"; then
        log_validation "✓ E2 Manager API working"
    else
        log_validation "✗ E2 Manager API failed"
        return 1
    fi
    
    # Test Subscription Manager API
    log_validation "Testing Subscription Manager API endpoints"
    
    if test_service_endpoint "submgr" "8080" "/ric/v1/health" "200"; then
        log_validation "✓ Subscription Manager API working"
    else
        log_validation "✗ Subscription Manager API failed"
        return 1
    fi
    
    return 0
}

# Test resource limits and requests
test_resource_configuration() {
    log_validation "Testing resource configuration"
    
    local components=("e2term" "e2mgr" "submgr" "a1mediator" "o1mediator" "dbaas")
    
    for component in "${components[@]}"; do
        log_validation "Checking resource configuration for $component"
        
        # Check if resource limits are set
        local cpu_limit=$(kubectl get deployment "$component" -n "$NAMESPACE" \
            -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}' 2>/dev/null || echo "")
        local memory_limit=$(kubectl get deployment "$component" -n "$NAMESPACE" \
            -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}' 2>/dev/null || echo "")
        
        if [ -n "$cpu_limit" ] && [ -n "$memory_limit" ]; then
            log_validation "✓ $component has resource limits: CPU=$cpu_limit, Memory=$memory_limit"
        else
            log_validation "⚠ $component missing resource limits"
        fi
        
        # Check if resource requests are set
        local cpu_request=$(kubectl get deployment "$component" -n "$NAMESPACE" \
            -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}' 2>/dev/null || echo "")
        local memory_request=$(kubectl get deployment "$component" -n "$NAMESPACE" \
            -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}' 2>/dev/null || echo "")
        
        if [ -n "$cpu_request" ] && [ -n "$memory_request" ]; then
            log_validation "✓ $component has resource requests: CPU=$cpu_request, Memory=$memory_request"
        else
            log_validation "⚠ $component missing resource requests"
        fi
    done
    
    return 0
}

# Test security configuration
test_security_configuration() {
    log_validation "Testing security configuration"
    
    local components=("e2term" "e2mgr" "submgr" "a1mediator" "o1mediator")
    
    for component in "${components[@]}"; do
        log_validation "Checking security configuration for $component"
        
        # Check if running as non-root
        local run_as_user=$(kubectl get deployment "$component" -n "$NAMESPACE" \
            -o jsonpath='{.spec.template.spec.securityContext.runAsUser}' 2>/dev/null || echo "")
        
        if [ -n "$run_as_user" ] && [ "$run_as_user" != "0" ]; then
            log_validation "✓ $component running as non-root user ($run_as_user)"
        else
            log_validation "⚠ $component security context not properly configured"
        fi
        
        # Check for read-only root filesystem
        local read_only_fs=$(kubectl get deployment "$component" -n "$NAMESPACE" \
            -o jsonpath='{.spec.template.spec.containers[0].securityContext.readOnlyRootFilesystem}' 2>/dev/null || echo "false")
        
        if [ "$read_only_fs" = "true" ]; then
            log_validation "✓ $component has read-only root filesystem"
        else
            log_validation "⚠ $component does not have read-only root filesystem"
        fi
    done
    
    # Check for network policies
    local network_policies=$(kubectl get networkpolicies -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    
    if [ "$network_policies" -gt 0 ]; then
        log_validation "✓ Network policies configured ($network_policies policies)"
    else
        log_validation "⚠ No network policies found"
    fi
    
    return 0
}

# Test observability stack
test_observability() {
    log_validation "Testing observability stack"
    
    # Test Prometheus
    if kubectl get deployment prometheus -n "$NAMESPACE" >/dev/null 2>&1; then
        if test_service_endpoint "prometheus" "9090" "/api/v1/query?query=up" "200"; then
            log_validation "✓ Prometheus is working"
        else
            log_validation "✗ Prometheus is not responding"
            return 1
        fi
    else
        log_validation "⚠ Prometheus not deployed"
    fi
    
    # Test Grafana
    if kubectl get deployment grafana -n "$NAMESPACE" >/dev/null 2>&1; then
        if test_service_endpoint "grafana" "3000" "/api/health" "200"; then
            log_validation "✓ Grafana is working"
        else
            log_validation "✗ Grafana is not responding"
            return 1
        fi
    else
        log_validation "⚠ Grafana not deployed"
    fi
    
    # Test metrics collection
    local components=("e2mgr" "submgr" "a1mediator")
    
    for component in "${components[@]}"; do
        if test_service_endpoint "$component" "9090" "/metrics" "200"; then
            log_validation "✓ $component metrics endpoint working"
        else
            log_validation "⚠ $component metrics endpoint not working"
        fi
    done
    
    return 0
}

# Performance validation
test_performance() {
    log_validation "Testing performance characteristics"
    
    # Test API response times
    local components=("e2mgr" "submgr" "a1mediator")
    
    for component in "${components[@]}"; do
        log_validation "Testing response time for $component"
        
        local start_time=$(date +%s%N)
        if kubectl exec -n "$NAMESPACE" deployment/"$component" -- \
           curl -sf localhost:8080/health >/dev/null 2>&1; then
            local end_time=$(date +%s%N)
            local response_time=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
            
            if [ "$response_time" -lt 100 ]; then
                log_validation "✓ $component response time: ${response_time}ms (Good)"
            elif [ "$response_time" -lt 500 ]; then
                log_validation "⚠ $component response time: ${response_time}ms (Acceptable)"
            else
                log_validation "✗ $component response time: ${response_time}ms (Too slow)"
                return 1
            fi
        else
            log_validation "✗ $component health check failed"
            return 1
        fi
    done
    
    # Test database performance
    log_validation "Testing database performance"
    
    local start_time=$(date +%s)
    for i in {1..100}; do
        kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
            redis-cli set "perf-test-$i" "value-$i" >/dev/null 2>&1
        kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
            redis-cli get "perf-test-$i" >/dev/null 2>&1
    done
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Cleanup
    kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
        redis-cli eval "for i=1,100 do redis.call('del', 'perf-test-'..i) end" 0 >/dev/null 2>&1
    
    if [ "$duration" -lt 5 ]; then
        log_validation "✓ Database performance: ${duration}s for 100 operations (Good)"
    elif [ "$duration" -lt 15 ]; then
        log_validation "⚠ Database performance: ${duration}s for 100 operations (Acceptable)"
    else
        log_validation "✗ Database performance: ${duration}s for 100 operations (Too slow)"
        return 1
    fi
    
    return 0
}

# Load testing
run_load_test() {
    local duration=${1:-60}
    local concurrent_users=${2:-10}
    
    log_validation "Running load test (duration: ${duration}s, concurrent users: $concurrent_users)"
    
    # Simple load test using kubectl exec and curl
    local pids=()
    
    for i in $(seq 1 "$concurrent_users"); do
        (
            local requests=0
            local errors=0
            local start_time=$(date +%s)
            local end_time=$((start_time + duration))
            
            while [ $(date +%s) -lt $end_time ]; do
                if kubectl exec -n "$NAMESPACE" deployment/e2mgr -- \
                   curl -sf localhost:8080/health >/dev/null 2>&1; then
                    requests=$((requests + 1))
                else
                    errors=$((errors + 1))
                fi
                sleep 0.1
            done
            
            echo "User $i: $requests requests, $errors errors"
        ) &
        pids+=($!)
    done
    
    # Wait for all background processes
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    log_validation "Load test completed"
    return 0
}

# Generate validation report
generate_validation_report() {
    local report_file="/tmp/validation-report-$(date +%Y%m%d-%H%M%S).json"
    
    log_validation "Generating validation report: $report_file"
    
    # Collect validation data
    local pod_count=$(kubectl get pods -n "$NAMESPACE" --no-headers | wc -l)
    local running_pods=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Running --no-headers | wc -l)
    local ready_pods=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -o True | wc -l)
    
    local service_count=$(kubectl get services -n "$NAMESPACE" --no-headers | wc -l)
    local deployment_count=$(kubectl get deployments -n "$NAMESPACE" --no-headers | wc -l)
    local available_deployments=$(kubectl get deployments -n "$NAMESPACE" -o jsonpath='{.items[*].status.conditions[?(@.type=="Available")].status}' | grep -o True | wc -l)
    
    cat > "$report_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "namespace": "$NAMESPACE",
    "validation_summary": {
        "total_pods": $pod_count,
        "running_pods": $running_pods,
        "ready_pods": $ready_pods,
        "total_services": $service_count,
        "total_deployments": $deployment_count,
        "available_deployments": $available_deployments
    },
    "component_status": {
        "e2term": "$(kubectl get deployment e2term -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")",
        "e2mgr": "$(kubectl get deployment e2mgr -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")",
        "submgr": "$(kubectl get deployment submgr -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")",
        "a1mediator": "$(kubectl get deployment a1mediator -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")",
        "o1mediator": "$(kubectl get deployment o1mediator -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")",
        "dbaas": "$(kubectl get deployment dbaas -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "NotFound")"
    },
    "validation_tests": {
        "pod_health": "$(check_pod_health e2mgr 1 >/dev/null 2>&1 && echo "PASS" || echo "FAIL")",
        "database_connectivity": "$(test_database_connectivity >/dev/null 2>&1 && echo "PASS" || echo "FAIL")",
        "api_functionality": "$(test_api_functionality >/dev/null 2>&1 && echo "PASS" || echo "FAIL")",
        "inter_component_communication": "$(test_inter_component_communication >/dev/null 2>&1 && echo "PASS" || echo "FAIL")",
        "security_configuration": "$(test_security_configuration >/dev/null 2>&1 && echo "PASS" || echo "FAIL")"
    }
}
EOF
    
    echo "$report_file"
}

# Main validation function
main() {
    local test_type=${1:-"full"}
    
    log_validation "Starting deployment validation: $test_type"
    
    case "$test_type" in
        "smoke")
            log_validation "Running smoke tests"
            
            # Basic smoke tests
            local components=("e2term" "e2mgr" "submgr" "a1mediator" "o1mediator" "dbaas")
            local failed_tests=0
            
            for component in "${components[@]}"; do
                if ! check_pod_health "$component"; then
                    failed_tests=$((failed_tests + 1))
                fi
            done
            
            if ! test_database_connectivity; then
                failed_tests=$((failed_tests + 1))
            fi
            
            if [ $failed_tests -eq 0 ]; then
                log_validation "✓ All smoke tests passed"
                exit 0
            else
                log_validation "✗ $failed_tests smoke tests failed"
                exit 1
            fi
            ;;
            
        "integration")
            log_validation "Running integration tests"
            
            local failed_tests=0
            
            if ! test_inter_component_communication; then
                failed_tests=$((failed_tests + 1))
            fi
            
            if ! test_api_functionality; then
                failed_tests=$((failed_tests + 1))
            fi
            
            if [ $failed_tests -eq 0 ]; then
                log_validation "✓ All integration tests passed"
                exit 0
            else
                log_validation "✗ $failed_tests integration tests failed"
                exit 1
            fi
            ;;
            
        "performance")
            log_validation "Running performance tests"
            
            if test_performance && run_load_test 30 5; then
                log_validation "✓ All performance tests passed"
                exit 0
            else
                log_validation "✗ Performance tests failed"
                exit 1
            fi
            ;;
            
        "security")
            log_validation "Running security tests"
            
            if test_security_configuration; then
                log_validation "✓ All security tests passed"
                exit 0
            else
                log_validation "✗ Security tests failed"
                exit 1
            fi
            ;;
            
        "full")
            log_validation "Running full validation suite"
            
            local failed_tests=0
            
            # Component health
            local components=("e2term" "e2mgr" "submgr" "a1mediator" "o1mediator" "dbaas")
            for component in "${components[@]}"; do
                if ! check_pod_health "$component"; then
                    failed_tests=$((failed_tests + 1))
                fi
            done
            
            # Database connectivity
            if ! test_database_connectivity; then
                failed_tests=$((failed_tests + 1))
            fi
            
            # Inter-component communication
            if ! test_inter_component_communication; then
                failed_tests=$((failed_tests + 1))
            fi
            
            # API functionality
            if ! test_api_functionality; then
                failed_tests=$((failed_tests + 1))
            fi
            
            # Resource configuration
            test_resource_configuration || true  # Non-critical
            
            # Security configuration
            test_security_configuration || true  # Non-critical
            
            # Observability
            test_observability || true  # Non-critical
            
            # Performance
            if ! test_performance; then
                failed_tests=$((failed_tests + 1))
            fi
            
            # Generate report
            local report_file=$(generate_validation_report)
            log_validation "Validation report generated: $report_file"
            
            if [ $failed_tests -eq 0 ]; then
                log_validation "✓ All validation tests passed"
                exit 0
            else
                log_validation "✗ $failed_tests validation tests failed"
                exit 1
            fi
            ;;
            
        "report")
            report_file=$(generate_validation_report)
            echo "Validation report: $report_file"
            cat "$report_file"
            ;;
            
        *)
            echo "Usage: $0 {smoke|integration|performance|security|full|report}"
            echo "  smoke       - Basic smoke tests"
            echo "  integration - Integration tests"
            echo "  performance - Performance tests"
            echo "  security    - Security tests"
            echo "  full        - Complete validation suite"
            echo "  report      - Generate validation report"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"