#!/bin/bash
# automated-health-checks.sh - Automated health check and self-healing system

set -euo pipefail

# Configuration
NAMESPACE="oran-ric"
CHECK_INTERVAL=60  # seconds
MAX_RETRIES=3
HEALTH_LOG="/var/log/oran-ric-health.log"

# Logging function
log_health() {
    echo "$(date -Iseconds) [HEALTH] $1" | tee -a "$HEALTH_LOG"
}

# Component health check with retry logic
check_component_with_retry() {
    local component=$1
    local port=${2:-8080}
    local endpoint=${3:-"/health"}
    local retries=0
    
    while [ $retries -lt $MAX_RETRIES ]; do
        if kubectl exec -n "$NAMESPACE" "deployment/$component" -- \
           curl -sf "localhost:$port$endpoint" >/dev/null 2>&1; then
            log_health "$component is healthy"
            return 0
        fi
        
        retries=$((retries + 1))
        log_health "$component health check failed (attempt $retries/$MAX_RETRIES)"
        sleep 5
    done
    
    return 1
}

# Automated pod restart
restart_unhealthy_pod() {
    local component=$1
    
    log_health "Attempting to restart unhealthy component: $component"
    
    # Get current replica count
    local current_replicas=$(kubectl get deployment "$component" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
    
    # Perform rolling restart
    kubectl rollout restart deployment/"$component" -n "$NAMESPACE"
    
    # Wait for rollout with timeout
    if kubectl rollout status deployment/"$component" -n "$NAMESPACE" --timeout=300s; then
        log_health "Successfully restarted $component"
        
        # Verify health after restart
        sleep 30
        if check_component_with_retry "$component"; then
            log_health "Health check passed after restart for $component"
            return 0
        else
            log_health "Health check still failing after restart for $component"
            return 1
        fi
    else
        log_health "Failed to restart $component - rollout timeout"
        return 1
    fi
}

# Database health check and recovery
check_database_health() {
    log_health "Checking database health"
    
    # Check if Redis is responding
    if ! kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli ping >/dev/null 2>&1; then
        log_health "Database not responding - attempting recovery"
        
        # Restart Redis
        kubectl rollout restart deployment/dbaas -n "$NAMESPACE"
        
        # Wait for restart
        kubectl rollout status deployment/dbaas -n "$NAMESPACE" --timeout=300s
        
        # Verify recovery
        sleep 30
        if kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli ping >/dev/null 2>&1; then
            log_health "Database recovery successful"
            return 0
        else
            log_health "Database recovery failed"
            return 1
        fi
    fi
    
    # Check memory usage
    local memory_info=$(kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
        redis-cli info memory | grep used_memory_human)
    log_health "Database memory usage: $memory_info"
    
    return 0
}

# Network connectivity check
check_network_connectivity() {
    log_health "Checking network connectivity"
    
    # Check inter-pod connectivity
    local components=("e2mgr" "submgr" "a1mediator")
    
    for component in "${components[@]}"; do
        if ! kubectl exec -n "$NAMESPACE" deployment/e2term -- \
             nc -z "$component.$NAMESPACE.svc.cluster.local" 8080 2>/dev/null; then
            log_health "Network connectivity issue: e2term -> $component"
            return 1
        fi
    done
    
    log_health "Network connectivity checks passed"
    return 0
}

# Resource usage monitoring and auto-scaling
monitor_and_scale() {
    local component=$1
    local cpu_threshold=70
    local memory_threshold=80
    
    # Get current resource usage
    local pod_name=$(kubectl get pods -n "$NAMESPACE" -l app="$component" -o jsonpath='{.items[0].metadata.name}')
    
    if [ -n "$pod_name" ]; then
        # Check CPU usage (requires metrics-server)
        local cpu_usage=$(kubectl top pod "$pod_name" -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print $2}' | sed 's/m//' || echo "0")
        local memory_usage=$(kubectl top pod "$pod_name" -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print $3}' | sed 's/Mi//' || echo "0")
        
        # Convert CPU to percentage (assuming 1000m = 100%)
        local cpu_percent=$((cpu_usage / 10))
        
        if [ "$cpu_percent" -gt "$cpu_threshold" ] || [ "$memory_usage" -gt "$memory_threshold" ]; then
            log_health "High resource usage detected for $component (CPU: ${cpu_percent}%, Memory: ${memory_usage}Mi)"
            
            # Auto-scale if HPA is not managing this deployment
            local current_replicas=$(kubectl get deployment "$component" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
            local max_replicas=5
            
            if [ "$current_replicas" -lt "$max_replicas" ]; then
                local new_replicas=$((current_replicas + 1))
                kubectl scale deployment "$component" -n "$NAMESPACE" --replicas="$new_replicas"
                log_health "Auto-scaled $component from $current_replicas to $new_replicas replicas"
            else
                log_health "Cannot auto-scale $component - already at maximum replicas ($max_replicas)"
            fi
        fi
    fi
}

# Certificate expiry check
check_certificates() {
    log_health "Checking certificate expiry"
    
    # Get all TLS secrets
    kubectl get secrets -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.type=="kubernetes.io/tls") | .metadata.name' | \
        while read cert_name; do
            # Extract certificate and check expiry
            local expiry_date=$(kubectl get secret "$cert_name" -n "$NAMESPACE" -o json | \
                jq -r '.data."tls.crt"' | base64 -d | \
                openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2 || echo "")
            
            if [ -n "$expiry_date" ]; then
                local expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null || echo "0")
                local current_epoch=$(date +%s)
                local days_until_expiry=$(( (expiry_epoch - current_epoch) / 86400 ))
                
                if [ "$days_until_expiry" -lt 30 ]; then
                    log_health "Certificate $cert_name expires in $days_until_expiry days"
                    
                    # Trigger certificate renewal if cert-manager is available
                    if kubectl get crd certificates.cert-manager.io >/dev/null 2>&1; then
                        kubectl annotate secret "$cert_name" -n "$NAMESPACE" \
                            cert-manager.io/issue-temporary-certificate="" --overwrite
                        log_health "Triggered renewal for certificate $cert_name"
                    fi
                fi
            fi
        done
}

# Log rotation and cleanup
perform_log_cleanup() {
    log_health "Performing log cleanup"
    
    # Rotate health log if it's too large (>100MB)
    if [ -f "$HEALTH_LOG" ] && [ $(stat -f%z "$HEALTH_LOG" 2>/dev/null || stat -c%s "$HEALTH_LOG" 2>/dev/null || echo 0) -gt 104857600 ]; then
        mv "$HEALTH_LOG" "${HEALTH_LOG}.$(date +%Y%m%d)"
        gzip "${HEALTH_LOG}.$(date +%Y%m%d)" 2>/dev/null || true
        log_health "Rotated health log"
    fi
    
    # Clean up old log files (keep last 7 days)
    find "$(dirname "$HEALTH_LOG")" -name "$(basename "$HEALTH_LOG").*.gz" -mtime +7 -delete 2>/dev/null || true
    
    # Clean up old pod logs via kubectl
    kubectl get pods -n "$NAMESPACE" --no-headers | \
        awk '{print $1}' | \
        while read pod; do
            # Truncate logs if they're too large
            kubectl exec -n "$NAMESPACE" "$pod" -- sh -c '
                for log in /var/log/*.log; do
                    if [ -f "$log" ] && [ $(stat -c%s "$log" 2>/dev/null || echo 0) -gt 52428800 ]; then
                        tail -n 1000 "$log" > "$log.tmp" && mv "$log.tmp" "$log"
                    fi
                done
            ' 2>/dev/null || true
        done
}

# Performance optimization
optimize_performance() {
    log_health "Running performance optimizations"
    
    # Database optimization
    kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli CONFIG SET maxmemory-policy allkeys-lru 2>/dev/null || true
    kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli CONFIG SET tcp-keepalive 60 2>/dev/null || true
    
    # Clear any stuck connections
    kubectl get pods -n "$NAMESPACE" -o name | \
        grep -E "(e2mgr|submgr)" | \
        while read pod; do
            kubectl exec "$pod" -n "$NAMESPACE" -- sh -c '
                # Clear any zombie processes
                ps aux | grep -E "(defunct|<zombie>)" | awk "{print \$2}" | xargs -r kill -9 2>/dev/null || true
                
                # Clear connection pools if they exist
                pkill -f "connection-pool" 2>/dev/null || true
            ' 2>/dev/null || true
        done
}

# Main health check loop
run_health_checks() {
    log_health "Starting automated health check cycle"
    
    local failed_components=()
    local components=("e2term" "e2mgr" "submgr" "a1mediator" "o1mediator")
    
    # Check each component
    for component in "${components[@]}"; do
        if ! check_component_with_retry "$component"; then
            failed_components+=("$component")
        fi
    done
    
    # Attempt self-healing for failed components
    for component in "${failed_components[@]}"; do
        if restart_unhealthy_pod "$component"; then
            log_health "Self-healing successful for $component"
        else
            log_health "Self-healing failed for $component - manual intervention required"
        fi
    done
    
    # Database health check
    check_database_health
    
    # Network connectivity check
    check_network_connectivity
    
    # Resource monitoring and auto-scaling
    for component in "${components[@]}"; do
        monitor_and_scale "$component"
    done
    
    # Certificate expiry check
    check_certificates
    
    # Performance optimization
    optimize_performance
    
    # Log cleanup
    perform_log_cleanup
    
    log_health "Health check cycle completed"
}

# Continuous monitoring mode
continuous_monitoring() {
    log_health "Starting continuous health monitoring (interval: ${CHECK_INTERVAL}s)"
    
    while true; do
        run_health_checks
        sleep "$CHECK_INTERVAL"
    done
}

# Generate health summary
generate_health_summary() {
    local summary_file="/tmp/health-summary-$(date +%Y%m%d-%H%M%S).txt"
    
    cat > "$summary_file" << EOF
O-RAN RIC Health Summary
========================
Generated: $(date)

Pod Status:
$(kubectl get pods -n "$NAMESPACE" --no-headers | awk '{print $1 ": " $3}')

Service Status:
$(kubectl get svc -n "$NAMESPACE" --no-headers | awk '{print $1 ": " $2}')

Recent Health Events:
$(tail -n 20 "$HEALTH_LOG" 2>/dev/null || echo "No health log available")

Resource Usage:
$(kubectl top pods -n "$NAMESPACE" 2>/dev/null || echo "Metrics not available")

Storage Usage:
$(kubectl get pvc -n "$NAMESPACE" 2>/dev/null || echo "No persistent volumes")
EOF
    
    echo "$summary_file"
}

# Main function
main() {
    local action=${1:-"check"}
    
    case "$action" in
        "check")
            run_health_checks
            ;;
        "monitor")
            continuous_monitoring
            ;;
        "summary")
            summary_file=$(generate_health_summary)
            echo "Health summary generated: $summary_file"
            cat "$summary_file"
            ;;
        "restart")
            local component=${2:-""}
            if [ -n "$component" ]; then
                restart_unhealthy_pod "$component"
            else
                echo "Usage: $0 restart <component>"
                exit 1
            fi
            ;;
        *)
            echo "Usage: $0 {check|monitor|summary|restart}"
            echo "  check    - Run single health check cycle"
            echo "  monitor  - Run continuous monitoring"
            echo "  summary  - Generate health summary"
            echo "  restart  - Restart specific component"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"