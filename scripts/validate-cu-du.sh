#!/bin/bash

# O-RAN CU/DU Network Functions Validation Script
# This script validates the CU/DU deployment and interface configurations

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-oran-network-functions}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validation results
VALIDATION_RESULTS=()
TOTAL_TESTS=0
PASSED_TESTS=0

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

# Test result functions
test_pass() {
    PASSED_TESTS=$((PASSED_TESTS + 1))
    VALIDATION_RESULTS+=("PASS: $1")
    echo -e "${GREEN}✓${NC} $1"
}

test_fail() {
    VALIDATION_RESULTS+=("FAIL: $1")
    echo -e "${RED}✗${NC} $1"
}

test_skip() {
    VALIDATION_RESULTS+=("SKIP: $1")
    echo -e "${YELLOW}⊝${NC} $1 (SKIPPED)"
}

run_test() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log_test "$1"
}

# Validate basic deployment
validate_deployment() {
    log_info "Validating basic deployment..."
    
    # Check namespace exists
    run_test "Namespace existence"
    if kubectl get namespace $NAMESPACE &>/dev/null; then
        test_pass "Namespace $NAMESPACE exists"
    else
        test_fail "Namespace $NAMESPACE does not exist"
        return 1
    fi
    
    # Check deployments
    local deployments=("cu-cp" "cu-up" "du-1" "du-2")
    for deployment in "${deployments[@]}"; do
        run_test "Deployment $deployment status"
        if kubectl get deployment $deployment -n $NAMESPACE &>/dev/null; then
            local ready_replicas=$(kubectl get deployment $deployment -n $NAMESPACE -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
            local desired_replicas=$(kubectl get deployment $deployment -n $NAMESPACE -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
            
            if [ "$ready_replicas" = "$desired_replicas" ] && [ "$ready_replicas" != "0" ]; then
                test_pass "Deployment $deployment is ready ($ready_replicas/$desired_replicas)"
            else
                test_fail "Deployment $deployment is not ready ($ready_replicas/$desired_replicas)"
            fi
        else
            test_fail "Deployment $deployment does not exist"
        fi
    done
    
    # Check services
    local services=("cu-cp-service" "cu-up-service" "du-service")
    for service in "${services[@]}"; do
        run_test "Service $service existence"
        if kubectl get service $service -n $NAMESPACE &>/dev/null; then
            test_pass "Service $service exists"
        else
            test_fail "Service $service does not exist"
        fi
    done
}

# Validate F1 interface
validate_f1_interface() {
    log_info "Validating F1 interface configuration..."
    
    # Check F1-C port on CU-CP
    run_test "F1-C interface port (38472) on CU-CP"
    local cu_cp_pod=$(kubectl get pods -n $NAMESPACE -l app=cu-cp -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$cu_cp_pod" ]; then
        if kubectl exec -n $NAMESPACE $cu_cp_pod -- netstat -tuln 2>/dev/null | grep -q :38472; then
            test_pass "F1-C port 38472 is listening on CU-CP"
        else
            test_fail "F1-C port 38472 is not listening on CU-CP"
        fi
    else
        test_fail "CU-CP pod not found"
    fi
    
    # Check F1-U port on DU
    run_test "F1-U interface port (2152) on DU"
    local du_pods=$(kubectl get pods -n $NAMESPACE -l app=du -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    local f1u_found=false
    for pod in $du_pods; do
        if kubectl exec -n $NAMESPACE $pod -- netstat -tuln 2>/dev/null | grep -q :2152; then
            test_pass "F1-U port 2152 is listening on $pod"
            f1u_found=true
            break
        fi
    done
    if [ "$f1u_found" = false ]; then
        test_fail "F1-U port 2152 is not listening on any DU pod"
    fi
    
    # Check F1 configuration
    run_test "F1 interface configuration"
    if kubectl get configmap f1-interface-config -n $NAMESPACE &>/dev/null; then
        test_pass "F1 interface configuration exists"
    else
        test_fail "F1 interface configuration missing"
    fi
}

# Validate E1 interface
validate_e1_interface() {
    log_info "Validating E1 interface configuration..."
    
    # Check E1 port on CU-CP
    run_test "E1 interface port (38462) on CU-CP"
    local cu_cp_pod=$(kubectl get pods -n $NAMESPACE -l app=cu-cp -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$cu_cp_pod" ]; then
        if kubectl exec -n $NAMESPACE $cu_cp_pod -- netstat -tuln 2>/dev/null | grep -q :38462; then
            test_pass "E1 port 38462 is listening on CU-CP"
        else
            test_fail "E1 port 38462 is not listening on CU-CP"
        fi
    else
        test_fail "CU-CP pod not found for E1 validation"
    fi
    
    # Check E1 port on CU-UP
    run_test "E1 interface port (38462) on CU-UP"
    local cu_up_pod=$(kubectl get pods -n $NAMESPACE -l app=cu-up -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$cu_up_pod" ]; then
        if kubectl exec -n $NAMESPACE $cu_up_pod -- netstat -tuln 2>/dev/null | grep -q :38462; then
            test_pass "E1 port 38462 is listening on CU-UP"
        else
            test_fail "E1 port 38462 is not listening on CU-UP"
        fi
    else
        test_fail "CU-UP pod not found for E1 validation"
    fi
    
    # Check E1 configuration
    run_test "E1 interface configuration"
    if kubectl get configmap e1-interface-config -n $NAMESPACE &>/dev/null; then
        test_pass "E1 interface configuration exists"
    else
        test_fail "E1 interface configuration missing"
    fi
}

# Validate E2 interface
validate_e2_interface() {
    log_info "Validating E2 interface configuration..."
    
    # Check E2 port on network functions
    run_test "E2 interface port (36421) on network functions"
    local e2_found=false
    local pods=$(kubectl get pods -n $NAMESPACE -l 'app in (cu-cp,du)' -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for pod in $pods; do
        if kubectl exec -n $NAMESPACE $pod -- netstat -tuln 2>/dev/null | grep -q :36421; then
            test_pass "E2 port 36421 is listening on $pod"
            e2_found=true
        fi
    done
    if [ "$e2_found" = false ]; then
        test_fail "E2 port 36421 is not listening on any network function"
    fi
    
    # Check E2 service models configuration
    run_test "E2 service models configuration"
    if kubectl get configmap e2-service-models-enhanced -n $NAMESPACE &>/dev/null; then
        test_pass "E2 service models configuration exists"
    else
        test_fail "E2 service models configuration missing"
    fi
    
    # Validate service model contents
    run_test "E2SM-KPM service model"
    if kubectl get configmap e2-service-models-enhanced -n $NAMESPACE -o yaml | grep -q "E2SM-KPM"; then
        test_pass "E2SM-KPM service model is configured"
    else
        test_fail "E2SM-KPM service model is missing"
    fi
    
    run_test "E2SM-RC service model"
    if kubectl get configmap e2-service-models-enhanced -n $NAMESPACE -o yaml | grep -q "E2SM-RC"; then
        test_pass "E2SM-RC service model is configured"
    else
        test_fail "E2SM-RC service model is missing"
    fi
}

# Validate radio configuration
validate_radio_config() {
    log_info "Validating radio configuration..."
    
    # Check radio configuration
    run_test "Radio configuration"
    if kubectl get configmap radio-config -n $NAMESPACE &>/dev/null; then
        test_pass "Radio configuration exists"
    else
        test_fail "Radio configuration missing"
    fi
    
    # Check fronthaul configuration
    run_test "Fronthaul configuration"
    if kubectl get configmap fronthaul-radio-config -n $NAMESPACE &>/dev/null; then
        test_pass "Fronthaul configuration exists"
    else
        test_fail "Fronthaul configuration missing"
    fi
    
    # Check NetworkAttachmentDefinition (if SR-IOV is available)
    run_test "Fronthaul NetworkAttachmentDefinition"
    if kubectl get crd network-attachment-definitions.k8s.cni.cncf.io &>/dev/null; then
        if kubectl get networkattachmentdefinition fronthaul-net -n $NAMESPACE &>/dev/null; then
            test_pass "Fronthaul NetworkAttachmentDefinition exists"
        else
            test_fail "Fronthaul NetworkAttachmentDefinition missing"
        fi
    else
        test_skip "SR-IOV NetworkAttachmentDefinition CRD not available"
    fi
}

# Validate resource allocation
validate_resources() {
    log_info "Validating resource allocation..."
    
    # Check CPU requests and limits
    local pods=$(kubectl get pods -n $NAMESPACE -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for pod in $pods; do
        run_test "CPU resources for $pod"
        local cpu_requests=$(kubectl get pod $pod -n $NAMESPACE -o jsonpath='{.spec.containers[0].resources.requests.cpu}' 2>/dev/null || echo "")
        local cpu_limits=$(kubectl get pod $pod -n $NAMESPACE -o jsonpath='{.spec.containers[0].resources.limits.cpu}' 2>/dev/null || echo "")
        
        if [ -n "$cpu_requests" ] && [ -n "$cpu_limits" ]; then
            test_pass "CPU resources configured for $pod (requests: $cpu_requests, limits: $cpu_limits)"
        else
            test_fail "CPU resources not properly configured for $pod"
        fi
    done
    
    # Check memory resources
    for pod in $pods; do
        run_test "Memory resources for $pod"
        local mem_requests=$(kubectl get pod $pod -n $NAMESPACE -o jsonpath='{.spec.containers[0].resources.requests.memory}' 2>/dev/null || echo "")
        local mem_limits=$(kubectl get pod $pod -n $NAMESPACE -o jsonpath='{.spec.containers[0].resources.limits.memory}' 2>/dev/null || echo "")
        
        if [ -n "$mem_requests" ] && [ -n "$mem_limits" ]; then
            test_pass "Memory resources configured for $pod (requests: $mem_requests, limits: $mem_limits)"
        else
            test_fail "Memory resources not properly configured for $pod"
        fi
    done
    
    # Check hugepages (if configured)
    run_test "Hugepages configuration"
    local hugepages_found=false
    for pod in $pods; do
        local hugepages=$(kubectl get pod $pod -n $NAMESPACE -o jsonpath='{.spec.containers[0].resources.requests.hugepages-1Gi}' 2>/dev/null || echo "")
        if [ -n "$hugepages" ]; then
            test_pass "Hugepages configured for $pod ($hugepages)"
            hugepages_found=true
        fi
    done
    if [ "$hugepages_found" = false ]; then
        test_skip "No hugepages configuration found (optional)"
    fi
}

# Validate performance settings
validate_performance() {
    log_info "Validating performance settings..."
    
    # Check performance tuning configuration
    run_test "Performance tuning configuration"
    if kubectl get configmap performance-tuning-config -n $NAMESPACE &>/dev/null; then
        test_pass "Performance tuning configuration exists"
    else
        test_fail "Performance tuning configuration missing"
    fi
    
    # Check security context for capabilities
    local pods=$(kubectl get pods -n $NAMESPACE -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for pod in $pods; do
        run_test "Security context capabilities for $pod"
        local capabilities=$(kubectl get pod $pod -n $NAMESPACE -o jsonpath='{.spec.containers[0].securityContext.capabilities.add}' 2>/dev/null || echo "")
        
        if echo "$capabilities" | grep -q "IPC_LOCK\|NET_ADMIN"; then
            test_pass "Required capabilities configured for $pod"
        else
            test_fail "Required capabilities missing for $pod"
        fi
    done
    
    # Check node affinity/anti-affinity
    run_test "Node affinity configuration"
    local affinity_configured=false
    for pod in $pods; do
        local node_affinity=$(kubectl get pod $pod -n $NAMESPACE -o jsonpath='{.spec.affinity.nodeAffinity}' 2>/dev/null || echo "")
        if [ -n "$node_affinity" ]; then
            test_pass "Node affinity configured for $pod"
            affinity_configured=true
        fi
    done
    if [ "$affinity_configured" = false ]; then
        test_skip "No node affinity configuration found (optional)"
    fi
}

# Validate monitoring
validate_monitoring() {
    log_info "Validating monitoring configuration..."
    
    # Check monitoring configuration
    run_test "Monitoring configuration"
    if kubectl get configmap monitoring-config -n $NAMESPACE &>/dev/null; then
        test_pass "Monitoring configuration exists"
    else
        test_fail "Monitoring configuration missing"
    fi
    
    # Check Prometheus rules
    run_test "Prometheus rules configuration"
    if kubectl get configmap monitoring-config -n $NAMESPACE -o yaml | grep -q "prometheus-rules"; then
        test_pass "Prometheus rules are configured"
    else
        test_fail "Prometheus rules are missing"
    fi
    
    # Check metrics endpoints
    local pods=$(kubectl get pods -n $NAMESPACE -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    for pod in $pods; do
        run_test "Metrics endpoint for $pod"
        # Check if pod has metrics port configured
        local metrics_port=$(kubectl get pod $pod -n $NAMESPACE -o jsonpath='{.spec.containers[0].ports[?(@.name=="metrics")].containerPort}' 2>/dev/null || echo "")
        
        if [ -n "$metrics_port" ]; then
            test_pass "Metrics port $metrics_port configured for $pod"
        else
            test_skip "No metrics port configured for $pod (optional)"
        fi
    done
}

# Validate connectivity
validate_connectivity() {
    log_info "Validating network connectivity..."
    
    # Test internal service connectivity
    local services=("cu-cp-service" "cu-up-service" "du-service")
    for service in "${services[@]}"; do
        run_test "Service $service connectivity"
        if kubectl get endpoints $service -n $NAMESPACE &>/dev/null; then
            local endpoints=$(kubectl get endpoints $service -n $NAMESPACE -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)
            if [ -n "$endpoints" ]; then
                test_pass "Service $service has endpoints: $endpoints"
            else
                test_fail "Service $service has no endpoints"
            fi
        else
            test_fail "Service $service endpoints not found"
        fi
    done
    
    # Test DNS resolution
    run_test "DNS resolution"
    local cu_cp_pod=$(kubectl get pods -n $NAMESPACE -l app=cu-cp -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$cu_cp_pod" ]; then
        if kubectl exec -n $NAMESPACE $cu_cp_pod -- nslookup cu-up-service &>/dev/null; then
            test_pass "DNS resolution working from CU-CP to CU-UP"
        else
            test_fail "DNS resolution failed from CU-CP to CU-UP"
        fi
    else
        test_skip "CU-CP pod not available for DNS test"
    fi
}

# Generate comprehensive report
generate_report() {
    log_info "Generating validation report..."
    
    echo ""
    echo "================================================="
    echo "           CU/DU VALIDATION REPORT"
    echo "================================================="
    echo "Total Tests: $TOTAL_TESTS"
    echo "Passed Tests: $PASSED_TESTS"
    echo "Failed Tests: $((TOTAL_TESTS - PASSED_TESTS))"
    echo "Success Rate: $(( PASSED_TESTS * 100 / TOTAL_TESTS ))%"
    echo ""
    echo "Detailed Results:"
    echo "=================="
    
    for result in "${VALIDATION_RESULTS[@]}"; do
        case "$result" in
            PASS:*)
                echo -e "${GREEN}✓${NC} ${result#PASS: }"
                ;;
            FAIL:*)
                echo -e "${RED}✗${NC} ${result#FAIL: }"
                ;;
            SKIP:*)
                echo -e "${YELLOW}⊝${NC} ${result#SKIP: }"
                ;;
        esac
    done
    
    echo ""
    if [ "$PASSED_TESTS" -eq "$TOTAL_TESTS" ]; then
        echo -e "${GREEN}🎉 ALL VALIDATIONS PASSED!${NC}"
        echo "Your CU/DU deployment is ready for operation."
        return 0
    else
        echo -e "${RED}❌ SOME VALIDATIONS FAILED${NC}"
        echo "Please review the failed tests and fix the issues."
        return 1
    fi
}

# Main validation function
main() {
    log_info "Starting O-RAN CU/DU Network Functions validation..."
    echo "Target namespace: $NAMESPACE"
    echo ""
    
    # Run all validation tests
    validate_deployment
    validate_f1_interface
    validate_e1_interface
    validate_e2_interface
    validate_radio_config
    validate_resources
    validate_performance
    validate_monitoring
    validate_connectivity
    
    # Generate final report
    generate_report
}

# Execute main function
main "$@"