#!/bin/bash
# security-monitoring.sh - Security monitoring and compliance validation

set -euo pipefail

# Configuration
NAMESPACE="oran-ric"
SECURITY_LOG="/var/log/oran-ric-security.log"
COMPLIANCE_LOG="/var/log/oran-ric-compliance.log"
ALERT_WEBHOOK="https://hooks.slack.com/services/YOUR/SECURITY/WEBHOOK"

# Logging function
log_security() {
    echo "$(date -Iseconds) [SECURITY] $1" | tee -a "$SECURITY_LOG"
}

log_compliance() {
    echo "$(date -Iseconds) [COMPLIANCE] $1" | tee -a "$COMPLIANCE_LOG"
}

# Send security alert
send_security_alert() {
    local severity=$1
    local message=$2
    local component=${3:-"security"}
    
    log_security "$severity: $message"
    
    # Send to security team
    curl -X POST "$ALERT_WEBHOOK" \
        -H 'Content-type: application/json' \
        --data "{
            \"text\": \"🔒 O-RAN RIC Security Alert\",
            \"attachments\": [{
                \"color\": \"danger\",
                \"fields\": [
                    {\"title\": \"Severity\", \"value\": \"$severity\", \"short\": true},
                    {\"title\": \"Component\", \"value\": \"$component\", \"short\": true},
                    {\"title\": \"Message\", \"value\": \"$message\", \"short\": false},
                    {\"title\": \"Timestamp\", \"value\": \"$(date)\", \"short\": true}
                ]
            }]
        }" 2>/dev/null || log_security "Failed to send security alert"
}

# Check authentication failures
monitor_authentication() {
    log_security "Monitoring authentication events"
    
    # Check for failed login attempts in the last 5 minutes
    local failed_logins=$(kubectl logs -n "$NAMESPACE" --selector=app=dashboard --since=5m 2>/dev/null | \
        grep -c "authentication failed\|invalid credentials\|login failed" || echo "0")
    
    if [ "$failed_logins" -gt 10 ]; then
        send_security_alert "HIGH" "Multiple authentication failures detected: $failed_logins in last 5 minutes" "authentication"
    fi
    
    # Check for suspicious user agents
    kubectl logs -n "$NAMESPACE" --selector=app=dashboard --since=5m 2>/dev/null | \
        grep -i "user-agent" | \
        grep -E "(bot|crawler|scanner|exploit)" | \
        while read line; do
            send_security_alert "MEDIUM" "Suspicious user agent detected: $line" "authentication"
        done
    
    # Monitor JWT token anomalies
    local jwt_errors=$(kubectl logs -n "$NAMESPACE" --selector=app=dashboard --since=5m 2>/dev/null | \
        grep -c "jwt.*error\|token.*invalid\|signature.*verification.*failed" || echo "0")
    
    if [ "$jwt_errors" -gt 5 ]; then
        send_security_alert "MEDIUM" "JWT token errors detected: $jwt_errors in last 5 minutes" "authentication"
    fi
}

# Monitor network security
monitor_network_security() {
    log_security "Monitoring network security"
    
    # Check for unusual connection patterns
    kubectl get pods -n "$NAMESPACE" -o name | \
        while read pod; do
            # Check for connections to suspicious IPs
            kubectl exec "$pod" -n "$NAMESPACE" -- netstat -tn 2>/dev/null | \
                awk '/ESTABLISHED/ {print $5}' | \
                cut -d: -f1 | \
                sort -u | \
                while read ip; do
                    # Check if IP is in private ranges
                    if ! echo "$ip" | grep -E "^(10\.|172\.(1[6-9]|2[0-9]|3[01])\.|192\.168\.|127\.)" >/dev/null; then
                        send_security_alert "MEDIUM" "External connection detected from $pod to $ip" "network"
                    fi
                done
        done
    
    # Monitor for port scanning attempts
    local port_scan_attempts=$(kubectl logs -n "$NAMESPACE" --all-containers --since=5m 2>/dev/null | \
        grep -c "connection refused\|connection timeout" || echo "0")
    
    if [ "$port_scan_attempts" -gt 50 ]; then
        send_security_alert "HIGH" "Potential port scanning detected: $port_scan_attempts connection attempts" "network"
    fi
    
    # Check for TLS/SSL issues
    local tls_errors=$(kubectl logs -n "$NAMESPACE" --all-containers --since=5m 2>/dev/null | \
        grep -c "tls.*error\|ssl.*error\|certificate.*error" || echo "0")
    
    if [ "$tls_errors" -gt 0 ]; then
        send_security_alert "MEDIUM" "TLS/SSL errors detected: $tls_errors errors" "network"
    fi
}

# Check certificate security
monitor_certificates() {
    log_security "Monitoring certificate security"
    
    # Check for weak certificates
    kubectl get secrets -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.type=="kubernetes.io/tls") | .metadata.name' | \
        while read cert_name; do
            local cert_data=$(kubectl get secret "$cert_name" -n "$NAMESPACE" -o json | \
                jq -r '.data."tls.crt"' | base64 -d 2>/dev/null || echo "")
            
            if [ -n "$cert_data" ]; then
                # Check key length
                local key_length=$(echo "$cert_data" | openssl x509 -noout -text 2>/dev/null | \
                    grep "Public-Key:" | sed 's/.*(\([0-9]*\) bit).*/\1/' || echo "0")
                
                if [ "$key_length" -lt 2048 ]; then
                    send_security_alert "HIGH" "Weak certificate detected: $cert_name has $key_length bit key" "certificates"
                fi
                
                # Check for self-signed certificates in production
                local issuer=$(echo "$cert_data" | openssl x509 -noout -issuer 2>/dev/null | \
                    grep -o "CN=[^,]*" | cut -d= -f2 || echo "")
                local subject=$(echo "$cert_data" | openssl x509 -noout -subject 2>/dev/null | \
                    grep -o "CN=[^,]*" | cut -d= -f2 || echo "")
                
                if [ "$issuer" = "$subject" ] && [ -n "$issuer" ]; then
                    send_security_alert "MEDIUM" "Self-signed certificate detected: $cert_name" "certificates"
                fi
                
                # Check expiry (already covered in health checks, but security perspective)
                local expiry_date=$(echo "$cert_data" | openssl x509 -noout -enddate 2>/dev/null | \
                    cut -d= -f2 || echo "")
                
                if [ -n "$expiry_date" ]; then
                    local expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null || echo "0")
                    local current_epoch=$(date +%s)
                    local days_until_expiry=$(( (expiry_epoch - current_epoch) / 86400 ))
                    
                    if [ "$days_until_expiry" -lt 7 ]; then
                        send_security_alert "HIGH" "Certificate expires soon: $cert_name expires in $days_until_expiry days" "certificates"
                    fi
                fi
            fi
        done
}

# Monitor container security
monitor_container_security() {
    log_security "Monitoring container security"
    
    # Check for containers running as root
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.securityContext.runAsUser == 0 or (.spec.securityContext.runAsUser == null and .spec.containers[].securityContext.runAsUser == null)) | .metadata.name' | \
        while read pod; do
            send_security_alert "MEDIUM" "Container running as root: $pod" "container"
        done
    
    # Check for privileged containers
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.containers[].securityContext.privileged == true) | .metadata.name' | \
        while read pod; do
            send_security_alert "HIGH" "Privileged container detected: $pod" "container"
        done
    
    # Check for containers with excessive capabilities
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.containers[].securityContext.capabilities.add[]? == "SYS_ADMIN") | .metadata.name' | \
        while read pod; do
            send_security_alert "HIGH" "Container with SYS_ADMIN capability: $pod" "container"
        done
    
    # Monitor for suspicious process activity
    kubectl get pods -n "$NAMESPACE" -o name | \
        while read pod; do
            # Check for unusual processes
            kubectl exec "$pod" -n "$NAMESPACE" -- ps aux 2>/dev/null | \
                grep -E "(nc|netcat|nmap|wget.*http|curl.*-X.*POST)" | \
                while read process; do
                    send_security_alert "MEDIUM" "Suspicious process in $pod: $process" "container"
                done
        done
}

# Check RBAC compliance
check_rbac_compliance() {
    log_compliance "Checking RBAC compliance"
    
    # Check for overly permissive cluster roles
    kubectl get clusterroles -o json | \
        jq -r '.items[] | select(.rules[]?.verbs[]? == "*" and .rules[]?.resources[]? == "*") | .metadata.name' | \
        while read role; do
            log_compliance "WARNING: Overly permissive cluster role: $role"
        done
    
    # Check for service accounts with cluster-admin
    kubectl get clusterrolebindings -o json | \
        jq -r '.items[] | select(.roleRef.name == "cluster-admin") | .subjects[]? | select(.kind == "ServiceAccount") | "\(.namespace)/\(.name)"' | \
        while read sa; do
            log_compliance "WARNING: Service account with cluster-admin: $sa"
        done
    
    # Check for default service account usage
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.serviceAccountName == "default" or .spec.serviceAccountName == null) | .metadata.name' | \
        while read pod; do
            log_compliance "WARNING: Pod using default service account: $pod"
        done
}

# Check network policy compliance
check_network_policy_compliance() {
    log_compliance "Checking network policy compliance"
    
    # Check if network policies exist
    local network_policies=$(kubectl get networkpolicies -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    
    if [ "$network_policies" -eq 0 ]; then
        log_compliance "WARNING: No network policies found in namespace $NAMESPACE"
    else
        log_compliance "INFO: $network_policies network policies found"
    fi
    
    # Check for default deny policy
    if ! kubectl get networkpolicy default-deny -n "$NAMESPACE" >/dev/null 2>&1; then
        log_compliance "WARNING: No default deny network policy found"
    fi
}

# Check pod security standards compliance
check_pod_security_compliance() {
    log_compliance "Checking pod security standards compliance"
    
    # Check for missing security contexts
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.securityContext == null) | .metadata.name' | \
        while read pod; do
            log_compliance "WARNING: Pod missing security context: $pod"
        done
    
    # Check for missing resource limits
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.containers[].resources.limits == null) | .metadata.name' | \
        while read pod; do
            log_compliance "WARNING: Pod missing resource limits: $pod"
        done
    
    # Check for missing readiness/liveness probes
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.containers[].readinessProbe == null or .spec.containers[].livenessProbe == null) | .metadata.name' | \
        while read pod; do
            log_compliance "WARNING: Pod missing health probes: $pod"
        done
}

# Check secret management compliance
check_secret_compliance() {
    log_compliance "Checking secret management compliance"
    
    # Check for secrets in environment variables
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.containers[].env[]?.value? | test("password|secret|key|token"; "i")) | .metadata.name' | \
        while read pod; do
            log_compliance "WARNING: Potential secret in environment variable: $pod"
        done
    
    # Check for unencrypted secrets (this would require etcd access in real scenario)
    local secret_count=$(kubectl get secrets -n "$NAMESPACE" --no-headers | wc -l)
    log_compliance "INFO: $secret_count secrets found in namespace"
    
    # Check for default tokens
    kubectl get secrets -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.type == "kubernetes.io/service-account-token" and .metadata.name | startswith("default-token")) | .metadata.name' | \
        while read secret; do
            log_compliance "WARNING: Default service account token found: $secret"
        done
}

# Vulnerability scanning simulation
perform_vulnerability_scan() {
    log_security "Performing vulnerability scan"
    
    # Check for known vulnerable images (simplified check)
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | .spec.containers[] | "\(.image)"' | \
        sort -u | \
        while read image; do
            # Check for latest tag (not recommended for production)
            if echo "$image" | grep -q ":latest"; then
                send_security_alert "MEDIUM" "Container using 'latest' tag: $image" "vulnerability"
            fi
            
            # Check for very old base images (simplified)
            if echo "$image" | grep -E "(ubuntu:14|centos:6|alpine:3\.[0-5])"; then
                send_security_alert "HIGH" "Container using outdated base image: $image" "vulnerability"
            fi
        done
    
    # Check for exposed sensitive ports
    kubectl get services -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.ports[]?.port == 22 or .spec.ports[]?.port == 3389 or .spec.ports[]?.port == 5432) | .metadata.name' | \
        while read service; do
            send_security_alert "HIGH" "Service exposing sensitive port: $service" "vulnerability"
        done
}

# Generate security report
generate_security_report() {
    local report_file="/tmp/security-report-$(date +%Y%m%d-%H%M%S).json"
    
    log_security "Generating security report: $report_file"
    
    # Collect security metrics
    local failed_logins=$(kubectl logs -n "$NAMESPACE" --selector=app=dashboard --since=1h 2>/dev/null | \
        grep -c "authentication failed\|invalid credentials" || echo "0")
    
    local tls_errors=$(kubectl logs -n "$NAMESPACE" --all-containers --since=1h 2>/dev/null | \
        grep -c "tls.*error\|ssl.*error" || echo "0")
    
    local privileged_containers=$(kubectl get pods -n "$NAMESPACE" -o json | \
        jq '[.items[] | select(.spec.containers[].securityContext.privileged == true)] | length')
    
    local root_containers=$(kubectl get pods -n "$NAMESPACE" -o json | \
        jq '[.items[] | select(.spec.securityContext.runAsUser == 0 or (.spec.securityContext.runAsUser == null and .spec.containers[].securityContext.runAsUser == null))] | length')
    
    cat > "$report_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "security_metrics": {
        "failed_logins_1h": $failed_logins,
        "tls_errors_1h": $tls_errors,
        "privileged_containers": $privileged_containers,
        "root_containers": $root_containers
    },
    "compliance_status": {
        "network_policies": $(kubectl get networkpolicies -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l),
        "pod_security_policies": $(kubectl get podsecuritypolicies --no-headers 2>/dev/null | wc -l),
        "rbac_roles": $(kubectl get roles -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    },
    "certificate_status": {
        "total_certificates": $(kubectl get secrets -n "$NAMESPACE" -o json | jq '[.items[] | select(.type=="kubernetes.io/tls")] | length'),
        "expiring_soon": "$(kubectl get secrets -n "$NAMESPACE" -o json | jq -r '.items[] | select(.type=="kubernetes.io/tls") | .metadata.name' | wc -l)"
    }
}
EOF
    
    echo "$report_file"
}

# Automated security remediation
perform_security_remediation() {
    log_security "Performing automated security remediation"
    
    # Remove default service account tokens where possible
    kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.spec.serviceAccountName == "default") | .metadata.name' | \
        while read pod; do
            log_security "INFO: Pod $pod using default service account - consider creating dedicated SA"
        done
    
    # Update weak TLS configurations
    kubectl get configmaps -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.data | to_entries[] | .value | contains("TLSv1.0") or contains("TLSv1.1")) | .metadata.name' | \
        while read cm; do
            log_security "WARNING: ConfigMap $cm contains weak TLS configuration"
        done
    
    # Rotate service account tokens (if needed)
    kubectl get serviceaccounts -n "$NAMESPACE" --no-headers | \
        awk '{print $1}' | \
        while read sa; do
            # Check token age (simplified)
            local token_secret=$(kubectl get sa "$sa" -n "$NAMESPACE" -o jsonpath='{.secrets[0].name}' 2>/dev/null || echo "")
            if [ -n "$token_secret" ]; then
                local creation_time=$(kubectl get secret "$token_secret" -n "$NAMESPACE" -o jsonpath='{.metadata.creationTimestamp}' 2>/dev/null || echo "")
                if [ -n "$creation_time" ]; then
                    local creation_epoch=$(date -d "$creation_time" +%s 2>/dev/null || echo "0")
                    local current_epoch=$(date +%s)
                    local age_days=$(( (current_epoch - creation_epoch) / 86400 ))
                    
                    if [ "$age_days" -gt 90 ]; then
                        log_security "INFO: Service account token $token_secret is $age_days days old - consider rotation"
                    fi
                fi
            fi
        done
}

# Main security monitoring function
main() {
    local action=${1:-"monitor"}
    
    case "$action" in
        "monitor")
            log_security "Starting security monitoring cycle"
            
            monitor_authentication
            monitor_network_security
            monitor_certificates
            monitor_container_security
            
            log_security "Security monitoring cycle completed"
            ;;
            
        "compliance")
            log_compliance "Starting compliance check"
            
            check_rbac_compliance
            check_network_policy_compliance
            check_pod_security_compliance
            check_secret_compliance
            
            log_compliance "Compliance check completed"
            ;;
            
        "scan")
            perform_vulnerability_scan
            ;;
            
        "remediate")
            perform_security_remediation
            ;;
            
        "report")
            report_file=$(generate_security_report)
            echo "Security report generated: $report_file"
            cat "$report_file"
            ;;
            
        "full")
            log_security "Starting full security assessment"
            
            monitor_authentication
            monitor_network_security
            monitor_certificates
            monitor_container_security
            check_rbac_compliance
            check_network_policy_compliance
            check_pod_security_compliance
            check_secret_compliance
            perform_vulnerability_scan
            
            log_security "Full security assessment completed"
            ;;
            
        *)
            echo "Usage: $0 {monitor|compliance|scan|remediate|report|full}"
            echo "  monitor     - Monitor security events"
            echo "  compliance  - Check compliance status"
            echo "  scan        - Perform vulnerability scan"
            echo "  remediate   - Perform automated remediation"
            echo "  report      - Generate security report"
            echo "  full        - Run complete security assessment"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"