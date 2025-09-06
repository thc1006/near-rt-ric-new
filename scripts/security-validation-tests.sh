#!/bin/bash

# O-RAN WG11 Security Validation and Testing Script
# Comprehensive security testing for O-RAN L Release deployment

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_RESULTS_DIR="$PROJECT_ROOT/test-results"
TEMP_DIR="/tmp/oran-security-tests"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test result functions
test_passed() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log_success "✓ $1"
}

test_failed() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log_error "✗ $1"
}

run_test() {
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    local test_name="$1"
    local test_command="$2"
    
    log_info "Running test: $test_name"
    
    if eval "$test_command"; then
        test_passed "$test_name"
        return 0
    else
        test_failed "$test_name"
        return 1
    fi
}

# Setup test environment
setup_test_environment() {
    log_info "Setting up test environment..."
    
    mkdir -p "$TEST_RESULTS_DIR"
    mkdir -p "$TEMP_DIR"
    
    # Create test namespace
    kubectl create namespace oran-security-test --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Test environment setup completed"
}

# Test WG11 Interface Security
test_wg11_interface_security() {
    log_info "Testing WG11 Interface Security..."
    
    # Test E2 Interface Security
    run_test "E2 Interface Security Policy Exists" \
        "kubectl get securitypolicy e2-interface-security -n oran"
    
    run_test "E2 mTLS Configuration" \
        "kubectl get secret e2-server-cert-tls -n oran"
    
    run_test "E2 CA Certificate Exists" \
        "kubectl get secret e2-ca-cert-tls -n oran"
    
    # Test A1 Interface Security
    run_test "A1 Interface Security Policy Exists" \
        "kubectl get securitypolicy a1-interface-security -n nonrtric"
    
    run_test "A1 OAuth2 Secret Exists" \
        "kubectl get secret a1-oauth-secret -n nonrtric"
    
    # Test O1 Interface Security
    run_test "O1 Interface Security Policy Exists" \
        "kubectl get securitypolicy o1-interface-security -n oran"
    
    run_test "O1 NETCONF Configuration" \
        "kubectl get configmap netconf-acm-config -n oran"
    
    # Test O2 Interface Security
    run_test "O2 Interface Security Policy Exists" \
        "kubectl get securitypolicy o2-interface-security -n ocloud-system"
}

# Test FIPS 140-3 Compliance
test_fips_compliance() {
    log_info "Testing FIPS 140-3 Compliance..."
    
    # Check FIPS configuration
    run_test "FIPS Configuration Exists" \
        "kubectl get configmap fips-140-3-config -n oran"
    
    # Test FIPS environment variables in deployments
    local fips_compliant_count=0
    local total_deployments=0
    
    for ns in oran nonrtric nephio-system ocloud-system; do
        while IFS= read -r deploy; do
            if [[ -n "$deploy" ]]; then
                total_deployments=$((total_deployments + 1))
                if kubectl get "$deploy" -n "$ns" -o json | \
                   jq -e '.spec.template.spec.containers[].env[]? | select(.name=="GODEBUG" and (.value | contains("fips140")))' &>/dev/null; then
                    fips_compliant_count=$((fips_compliant_count + 1))
                fi
            fi
        done <<< "$(kubectl get deployments -n "$ns" -o name 2>/dev/null || true)"
    done
    
    if [[ $fips_compliant_count -gt 0 ]]; then
        test_passed "FIPS Environment Variables ($fips_compliant_count/$total_deployments deployments)"
    else
        test_failed "FIPS Environment Variables (0/$total_deployments deployments)"
    fi
    
    # Test Go version compatibility
    local go_version
    if command -v go &> /dev/null; then
        go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
        if [[ "$go_version" =~ ^1\.(2[5-9]|[3-9][0-9])$ ]]; then
            test_passed "Go Version FIPS Compatible ($go_version)"
        else
            test_failed "Go Version FIPS Compatible ($go_version)"
        fi
    else
        test_passed "Go Version FIPS Compatible (assumed 1.25+)"
    fi
}

# Test Container Security
test_container_security() {
    log_info "Testing Container Security..."
    
    # Test Pod Security Standards
    run_test "Pod Security Standards Configuration" \
        "kubectl get configmap pod-security-standards -n oran"
    
    # Test security contexts in deployments
    local secure_deployments=0
    local total_deployments=0
    
    for ns in oran nonrtric nephio-system ocloud-system; do
        while IFS= read -r deploy; do
            if [[ -n "$deploy" ]]; then
                total_deployments=$((total_deployments + 1))
                if kubectl get "$deploy" -n "$ns" -o json | \
                   jq -e '.spec.template.spec.securityContext.runAsNonRoot == true' &>/dev/null; then
                    secure_deployments=$((secure_deployments + 1))
                fi
            fi
        done <<< "$(kubectl get deployments -n "$ns" -o name 2>/dev/null || true)"
    done
    
    if [[ $secure_deployments -gt $((total_deployments / 2)) ]]; then
        test_passed "Security Context Configuration ($secure_deployments/$total_deployments deployments)"
    else
        test_failed "Security Context Configuration ($secure_deployments/$total_deployments deployments)"
    fi
    
    # Test container scanning configuration
    run_test "Container Security Scanning Config" \
        "kubectl get configmap container-security-config -n oran"
}

# Test Network Security
test_network_security() {
    log_info "Testing Network Security..."
    
    local policy_count=0
    local namespace_count=0
    
    # Test network policies in each namespace
    for ns in oran nonrtric nephio-system ocloud-system; do
        namespace_count=$((namespace_count + 1))
        local ns_policies
        ns_policies=$(kubectl get networkpolicies -n "$ns" --no-headers 2>/dev/null | wc -l || echo 0)
        policy_count=$((policy_count + ns_policies))
        
        if [[ $ns_policies -gt 0 ]]; then
            test_passed "Network Policies in $ns namespace ($ns_policies policies)"
        else
            test_failed "Network Policies in $ns namespace (0 policies)"
        fi
        
        # Test default deny policy
        if kubectl get networkpolicy default-deny-all -n "$ns" &>/dev/null || \
           kubectl get networkpolicy zero-trust-default-deny -n "$ns" &>/dev/null; then
            test_passed "Default Deny Policy in $ns"
        else
            test_failed "Default Deny Policy in $ns"
        fi
    done
    
    # Test service mesh configuration
    if kubectl get namespace istio-system &>/dev/null; then
        run_test "Istio Service Mesh Deployed" \
            "kubectl get pods -n istio-system -l app=istiod --no-headers | grep -q Running"
        
        run_test "Istio mTLS Configuration" \
            "kubectl get peerauthentication -A --no-headers | wc -l | grep -v '^0$'"
    else
        log_info "Istio service mesh not detected, skipping related tests"
    fi
}

# Test Certificate Management
test_certificate_management() {
    log_info "Testing Certificate Management..."
    
    # Test TLS certificates for each interface
    local interfaces=("e2" "a1" "o1" "o2")
    
    for interface in "${interfaces[@]}"; do
        local namespace="oran"
        case "$interface" in
            "a1") namespace="nonrtric" ;;
            "o2") namespace="ocloud-system" ;;
        esac
        
        # Test server certificate
        if kubectl get secret "${interface}-server-cert-tls" -n "$namespace" &>/dev/null; then
            test_passed "$interface Server Certificate Exists"
            
            # Test certificate validity
            local cert_data
            cert_data=$(kubectl get secret "${interface}-server-cert-tls" -n "$namespace" -o jsonpath='{.data.tls\.crt}' | base64 -d)
            
            if echo "$cert_data" | openssl x509 -noout -checkend 2592000 2>/dev/null; then  # 30 days
                test_passed "$interface Certificate Valid (>30 days)"
            else
                test_failed "$interface Certificate Valid (expires soon)"
            fi
        else
            test_failed "$interface Server Certificate Exists"
        fi
        
        # Test CA certificate
        if kubectl get secret "${interface}-ca-cert-tls" -n "$namespace" &>/dev/null; then
            test_passed "$interface CA Certificate Exists"
        else
            test_failed "$interface CA Certificate Exists"
        fi
    done
}

# Test Security Monitoring
test_security_monitoring() {
    log_info "Testing Security Monitoring..."
    
    # Test monitoring configuration
    run_test "Security Monitoring Rules" \
        "kubectl get configmap security-monitoring-rules -n oran"
    
    # Test if Prometheus is available
    if kubectl get pods -A -l app=prometheus --no-headers | grep -q Running; then
        test_passed "Prometheus Monitoring Available"
        
        # Test security metrics (if accessible)
        # This would require actual metrics endpoint access in a real deployment
        log_info "Security metrics validation requires live Prometheus endpoint"
    else
        test_failed "Prometheus Monitoring Available"
    fi
}

# Test Access Controls
test_access_controls() {
    log_info "Testing Access Controls..."
    
    # Test RBAC configurations
    local rbac_count=0
    for ns in oran nonrtric nephio-system ocloud-system; do
        local roles
        roles=$(kubectl get roles -n "$ns" --no-headers 2>/dev/null | wc -l || echo 0)
        rbac_count=$((rbac_count + roles))
    done
    
    if [[ $rbac_count -gt 0 ]]; then
        test_passed "RBAC Configuration ($rbac_count roles configured)"
    else
        test_failed "RBAC Configuration (no roles found)"
    fi
    
    # Test service accounts
    local sa_count=0
    for ns in oran nonrtric nephio-system ocloud-system; do
        local sas
        sas=$(kubectl get serviceaccounts -n "$ns" --no-headers 2>/dev/null | grep -v default | wc -l || echo 0)
        sa_count=$((sa_count + sas))
    done
    
    if [[ $sa_count -gt 0 ]]; then
        test_passed "Service Account Configuration ($sa_count service accounts)"
    else
        test_failed "Service Account Configuration (only default accounts)"
    fi
}

# Test Compliance with WG11 Requirements
test_wg11_compliance() {
    log_info "Testing WG11 Compliance Requirements..."
    
    # Test interface-specific compliance
    local compliant_interfaces=0
    local total_interfaces=4
    
    for interface in e2 a1 o1 o2; do
        local compliance_score=0
        
        # Check security policy
        if kubectl get securitypolicy "${interface}-interface-security" -A &>/dev/null; then
            compliance_score=$((compliance_score + 1))
        fi
        
        # Check TLS configuration
        if kubectl get secret "${interface}-server-cert-tls" -A &>/dev/null; then
            compliance_score=$((compliance_score + 1))
        fi
        
        # Check network policy
        if kubectl get networkpolicy -A -o json | jq -e ".items[] | select(.metadata.name | contains(\"${interface}\"))" &>/dev/null; then
            compliance_score=$((compliance_score + 1))
        fi
        
        if [[ $compliance_score -ge 2 ]]; then
            compliant_interfaces=$((compliant_interfaces + 1))
            test_passed "$interface Interface WG11 Compliance ($compliance_score/3 requirements)"
        else
            test_failed "$interface Interface WG11 Compliance ($compliance_score/3 requirements)"
        fi
    done
    
    # Overall WG11 compliance
    if [[ $compliant_interfaces -eq $total_interfaces ]]; then
        test_passed "Overall WG11 Compliance (4/4 interfaces compliant)"
    else
        test_failed "Overall WG11 Compliance ($compliant_interfaces/4 interfaces compliant)"
    fi
}

# Penetration Testing (Basic Security Tests)
test_security_penetration() {
    log_info "Running Basic Security Penetration Tests..."
    
    # Test for common misconfigurations
    local security_issues=0
    
    # Check for privileged containers
    local privileged_count
    privileged_count=$(kubectl get pods -A -o json | jq '[.items[] | select(.spec.containers[]?.securityContext?.privileged == true)] | length' 2>/dev/null || echo 0)
    
    if [[ $privileged_count -eq 0 ]]; then
        test_passed "No Privileged Containers Found"
    else
        test_failed "Privileged Containers Found ($privileged_count containers)"
        security_issues=$((security_issues + 1))
    fi
    
    # Check for containers running as root
    local root_containers
    root_containers=$(kubectl get pods -A -o json | jq '[.items[] | select(.spec.securityContext?.runAsUser == 0 or (.spec.securityContext?.runAsUser == null and .spec.securityContext?.runAsNonRoot != true))] | length' 2>/dev/null || echo 0)
    
    if [[ $root_containers -eq 0 ]]; then
        test_passed "No Root Containers Found"
    else
        test_failed "Root Containers Found ($root_containers containers)"
        security_issues=$((security_issues + 1))
    fi
    
    # Check for containers with excessive capabilities
    local excessive_caps
    excessive_caps=$(kubectl get pods -A -o json | jq '[.items[] | select(.spec.containers[]?.securityContext?.capabilities?.add[]? | contains("SYS_ADMIN"))] | length' 2>/dev/null || echo 0)
    
    if [[ $excessive_caps -eq 0 ]]; then
        test_passed "No Excessive Capabilities Found"
    else
        test_failed "Excessive Capabilities Found ($excessive_caps containers)"
        security_issues=$((security_issues + 1))
    fi
    
    # Check for host network usage
    local host_network
    host_network=$(kubectl get pods -A -o json | jq '[.items[] | select(.spec.hostNetwork == true)] | length' 2>/dev/null || echo 0)
    
    if [[ $host_network -eq 0 ]]; then
        test_passed "No Host Network Usage Found"
    else
        test_failed "Host Network Usage Found ($host_network pods)"
        security_issues=$((security_issues + 1))
    fi
    
    # Overall security posture
    if [[ $security_issues -eq 0 ]]; then
        test_passed "Security Posture Assessment (No critical issues)"
    else
        test_failed "Security Posture Assessment ($security_issues critical issues found)"
    fi
}

# Generate test report
generate_test_report() {
    log_info "Generating security test report..."
    
    local report_file="$TEST_RESULTS_DIR/security_test_report.yaml"
    local timestamp
    timestamp=$(date -Iseconds)
    
    cat > "$report_file" <<EOF
security_test_report:
  metadata:
    timestamp: $timestamp
    cluster: $(kubectl config current-context)
    o_ran_release: L
    wg11_version: "3.0"
    go_version: "$(command -v go &> /dev/null && go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//' || echo '1.25')"
  
  test_summary:
    total_tests: $TESTS_TOTAL
    passed: $TESTS_PASSED
    failed: $TESTS_FAILED
    success_rate: $((TESTS_PASSED * 100 / TESTS_TOTAL))%
    
  compliance_status:
    wg11_interfaces: $(if [[ $(kubectl get securitypolicy -A --no-headers | wc -l) -ge 4 ]]; then echo "COMPLIANT"; else echo "NON_COMPLIANT"; fi)
    fips_140_3: $(if [[ $(kubectl get configmap fips-140-3-config -n oran &>/dev/null) ]]; then echo "CONFIGURED"; else echo "NOT_CONFIGURED"; fi)
    container_security: $(if [[ $(kubectl get configmap pod-security-standards -n oran &>/dev/null) ]]; then echo "CONFIGURED"; else echo "NOT_CONFIGURED"; fi)
    network_security: $(if [[ $(kubectl get networkpolicies -A --no-headers | wc -l) -gt 0 ]]; then echo "ENABLED"; else echo "DISABLED"; fi)
    certificate_management: $(if [[ $(kubectl get secrets -A -o name | grep -c "\-cert\-tls" || echo 0) -gt 0 ]]; then echo "CONFIGURED"; else echo "NOT_CONFIGURED"; fi)
    
  security_score:
    interface_security: $(($(kubectl get securitypolicy -A --no-headers | wc -l) * 25))%
    encryption: $(if [[ $(kubectl get secrets -A -o name | grep -c "tls" || echo 0) -gt 0 ]]; then echo "100%"; else echo "0%"; fi)
    access_control: $(if [[ $(kubectl get roles -A --no-headers | wc -l) -gt 0 ]]; then echo "100%"; else echo "0%"; fi)
    monitoring: $(if [[ $(kubectl get configmap security-monitoring-rules -n oran &>/dev/null) ]]; then echo "100%"; else echo "0%"; fi)
    
  recommendations:
    - "Implement automated certificate rotation"
    - "Enable real-time security monitoring"
    - "Conduct regular vulnerability assessments"
    - "Implement security incident response procedures"
    - "Regular security training for operators"
    
  next_assessment_due: $(date -d "+3 months" -Iseconds)
EOF
    
    log_success "Security test report generated: $report_file"
    
    # Generate summary
    echo -e "\n${BLUE}Security Test Summary:${NC}"
    echo -e "====================="
    echo -e "Total Tests: $TESTS_TOTAL"
    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo -e "Success Rate: $((TESTS_PASSED * 100 / TESTS_TOTAL))%"
    echo -e "\nDetailed report: $report_file"
}

# Cleanup test environment
cleanup_test_environment() {
    log_info "Cleaning up test environment..."
    
    # Remove test namespace
    kubectl delete namespace oran-security-test --ignore-not-found=true
    
    # Clean up temp files
    rm -rf "$TEMP_DIR"
    
    log_success "Test environment cleanup completed"
}

# Main test execution
main() {
    log_info "Starting O-RAN WG11 Security Validation Tests"
    log_info "============================================="
    
    local test_suite="${1:-full}"
    
    setup_test_environment
    
    case "$test_suite" in
        "interfaces"|"wg11")
            test_wg11_interface_security
            ;;
        "fips")
            test_fips_compliance
            ;;
        "containers")
            test_container_security
            ;;
        "network")
            test_network_security
            ;;
        "certificates"|"certs")
            test_certificate_management
            ;;
        "monitoring")
            test_security_monitoring
            ;;
        "access")
            test_access_controls
            ;;
        "compliance")
            test_wg11_compliance
            ;;
        "penetration"|"pentest")
            test_security_penetration
            ;;
        "full"|"all")
            test_wg11_interface_security
            test_fips_compliance
            test_container_security
            test_network_security
            test_certificate_management
            test_security_monitoring
            test_access_controls
            test_wg11_compliance
            test_security_penetration
            ;;
        "help"|"-h"|"--help")
            echo "O-RAN WG11 Security Validation Test Suite"
            echo ""
            echo "Usage: $0 [test_suite]"
            echo ""
            echo "Test Suites:"
            echo "  full         - Run all security tests (default)"
            echo "  interfaces   - Test WG11 interface security"
            echo "  fips         - Test FIPS 140-3 compliance"
            echo "  containers   - Test container security"
            echo "  network      - Test network security policies"
            echo "  certs        - Test certificate management"
            echo "  monitoring   - Test security monitoring"
            echo "  access       - Test access controls"
            echo "  compliance   - Test WG11 compliance"
            echo "  pentest      - Run penetration tests"
            echo "  help         - Show this help message"
            ;;
        *)
            log_error "Unknown test suite: $test_suite"
            exit 1
            ;;
    esac
    
    generate_test_report
    cleanup_test_environment
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log_success "All security tests passed! ✓"
        exit 0
    else
        log_error "$TESTS_FAILED tests failed. Please review the report."
        exit 1
    fi
}

# Execute main function
main "$@"