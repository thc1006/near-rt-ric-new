#!/bin/bash
# production-monitoring.sh - Comprehensive production monitoring system

set -euo pipefail

# Configuration
NAMESPACE="oran-ric"
PROMETHEUS_URL="http://prometheus.ric.local:9090"
GRAFANA_URL="http://grafana.ric.local:3000"
ALERT_WEBHOOK="https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
LOG_FILE="/var/log/oran-ric-monitoring.log"
METRICS_FILE="/var/log/oran-ric-metrics.json"

# Logging function
log() {
    echo "$(date -Iseconds) [$1] $2" | tee -a "$LOG_FILE"
}

# Send alert function
send_alert() {
    local severity=$1
    local message=$2
    local component=${3:-"platform"}
    
    # Log alert
    log "ALERT" "$severity: $message"
    
    # Send to Slack
    curl -X POST "$ALERT_WEBHOOK" \
        -H 'Content-type: application/json' \
        --data "{
            \"text\": \"🚨 O-RAN RIC Alert\",
            \"attachments\": [{
                \"color\": \"$([ "$severity" = "CRITICAL" ] && echo "danger" || echo "warning")\",
                \"fields\": [
                    {\"title\": \"Severity\", \"value\": \"$severity\", \"short\": true},
                    {\"title\": \"Component\", \"value\": \"$component\", \"short\": true},
                    {\"title\": \"Message\", \"value\": \"$message\", \"short\": false},
                    {\"title\": \"Timestamp\", \"value\": \"$(date)\", \"short\": true}
                ]
            }]
        }" 2>/dev/null || log "ERROR" "Failed to send alert to Slack"
}

# Query Prometheus metrics
query_prometheus() {
    local query="$1"
    curl -s -G "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=${query}" | \
        jq -r '.data.result[0].value[1] // "0"' 2>/dev/null || echo "0"
}

# Check component health
check_component_health() {
    local component=$1
    local port=${2:-8080}
    local endpoint=${3:-"/health"}
    
    log "INFO" "Checking health of $component"
    
    if kubectl get deployment "$component" -n "$NAMESPACE" >/dev/null 2>&1; then
        local ready_replicas=$(kubectl get deployment "$component" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local desired_replicas=$(kubectl get deployment "$component" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
        
        if [ "$ready_replicas" -lt "$desired_replicas" ]; then
            send_alert "CRITICAL" "$component has $ready_replicas/$desired_replicas replicas ready" "$component"
            return 1
        fi
        
        # Check health endpoint
        if kubectl exec -n "$NAMESPACE" "deployment/$component" -- \
           curl -sf "localhost:$port$endpoint" >/dev/null 2>&1; then
            log "INFO" "$component health check passed"
            return 0
        else
            send_alert "CRITICAL" "$component health endpoint is not responding" "$component"
            return 1
        fi
    else
        send_alert "CRITICAL" "$component deployment not found" "$component"
        return 1
    fi
}

# Monitor resource usage
monitor_resources() {
    log "INFO" "Monitoring resource usage"
    
    # CPU usage
    local cpu_usage=$(query_prometheus 'avg(rate(container_cpu_usage_seconds_total{namespace="'$NAMESPACE'"}[5m])) * 100')
    local cpu_threshold=70
    
    if (( $(echo "$cpu_usage > $cpu_threshold" | bc -l) )); then
        send_alert "WARNING" "High CPU usage: ${cpu_usage}% (threshold: ${cpu_threshold}%)"
    fi
    
    # Memory usage
    local memory_usage=$(query_prometheus 'avg(container_memory_usage_bytes{namespace="'$NAMESPACE'"} / container_spec_memory_limit_bytes{namespace="'$NAMESPACE'"}) * 100')
    local memory_threshold=80
    
    if (( $(echo "$memory_usage > $memory_threshold" | bc -l) )); then
        send_alert "WARNING" "High memory usage: ${memory_usage}% (threshold: ${memory_threshold}%)"
    fi
    
    # Disk usage
    local disk_usage=$(query_prometheus 'avg((node_filesystem_size_bytes{fstype!="tmpfs"} - node_filesystem_avail_bytes{fstype!="tmpfs"}) / node_filesystem_size_bytes{fstype!="tmpfs"}) * 100')
    local disk_threshold=85
    
    if (( $(echo "$disk_usage > $disk_threshold" | bc -l) )); then
        send_alert "CRITICAL" "High disk usage: ${disk_usage}% (threshold: ${disk_threshold}%)"
    fi
    
    # Store metrics
    cat > "$METRICS_FILE" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "cpu_usage": $cpu_usage,
    "memory_usage": $memory_usage,
    "disk_usage": $disk_usage
}
EOF
}

# Monitor E2 connectivity
monitor_e2_connectivity() {
    log "INFO" "Monitoring E2 connectivity"
    
    local connected_nodes=$(query_prometheus 'e2mgr_connected_nodes')
    local min_nodes=1
    
    if (( $(echo "$connected_nodes < $min_nodes" | bc -l) )); then
        send_alert "CRITICAL" "Low E2 node connectivity: $connected_nodes nodes (minimum: $min_nodes)"
    fi
    
    # Check E2 message processing rate
    local e2_message_rate=$(query_prometheus 'rate(e2term_messages_total[5m])')
    local max_rate=10000
    
    if (( $(echo "$e2_message_rate > $max_rate" | bc -l) )); then
        send_alert "WARNING" "High E2 message rate: $e2_message_rate msg/s (max: $max_rate)"
    fi
}

# Monitor subscription health
monitor_subscriptions() {
    log "INFO" "Monitoring subscription health"
    
    local active_subscriptions=$(query_prometheus 'submgr_active_subscriptions')
    local failed_subscriptions=$(query_prometheus 'submgr_failed_subscriptions_total')
    
    # Check for subscription failures
    if (( $(echo "$failed_subscriptions > 0" | bc -l) )); then
        send_alert "WARNING" "Subscription failures detected: $failed_subscriptions failed"
    fi
    
    # Check subscription processing latency
    local sub_latency=$(query_prometheus 'histogram_quantile(0.95, rate(submgr_processing_duration_seconds_bucket[5m]))')
    local latency_threshold=0.1  # 100ms
    
    if (( $(echo "$sub_latency > $latency_threshold" | bc -l) )); then
        send_alert "WARNING" "High subscription processing latency: ${sub_latency}s (threshold: ${latency_threshold}s)"
    fi
}

# Monitor database health
monitor_database() {
    log "INFO" "Monitoring database health"
    
    # Check Redis connectivity
    if ! kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli ping >/dev/null 2>&1; then
        send_alert "CRITICAL" "Database (Redis) is not responding" "dbaas"
        return 1
    fi
    
    # Check memory usage
    local redis_memory=$(kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
        redis-cli info memory | grep used_memory_human | cut -d: -f2 | tr -d '\r')
    
    # Check connected clients
    local redis_clients=$(kubectl exec -n "$NAMESPACE" deployment/dbaas -- \
        redis-cli info clients | grep connected_clients | cut -d: -f2 | tr -d '\r')
    
    if [ "$redis_clients" -eq 0 ]; then
        send_alert "WARNING" "No clients connected to database" "dbaas"
    fi
    
    log "INFO" "Database health: Memory=$redis_memory, Clients=$redis_clients"
}

# Monitor security events
monitor_security() {
    log "INFO" "Monitoring security events"
    
    # Check for failed authentication attempts
    local auth_failures=$(query_prometheus 'increase(auth_failures_total[5m])')
    local auth_threshold=10
    
    if (( $(echo "$auth_failures > $auth_threshold" | bc -l) )); then
        send_alert "WARNING" "High authentication failures: $auth_failures in 5 minutes"
    fi
    
    # Check certificate expiry
    local cert_expiry_days=$(query_prometheus 'min(cert_expiry_days)')
    local cert_threshold=30
    
    if (( $(echo "$cert_expiry_days < $cert_threshold" | bc -l) )); then
        send_alert "WARNING" "Certificate expiring soon: $cert_expiry_days days remaining"
    fi
    
    # Check for suspicious network activity
    local network_errors=$(query_prometheus 'increase(network_errors_total[5m])')
    local network_threshold=100
    
    if (( $(echo "$network_errors > $network_threshold" | bc -l) )); then
        send_alert "WARNING" "High network errors: $network_errors in 5 minutes"
    fi
}

# Performance monitoring
monitor_performance() {
    log "INFO" "Monitoring performance metrics"
    
    # API response times
    local api_latency=$(query_prometheus 'histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))')
    local latency_threshold=1.0  # 1 second
    
    if (( $(echo "$api_latency > $latency_threshold" | bc -l) )); then
        send_alert "WARNING" "High API latency: ${api_latency}s (threshold: ${latency_threshold}s)"
    fi
    
    # Throughput monitoring
    local request_rate=$(query_prometheus 'rate(http_requests_total[5m])')
    local max_request_rate=1000
    
    if (( $(echo "$request_rate > $max_request_rate" | bc -l) )); then
        send_alert "WARNING" "High request rate: $request_rate req/s (max: $max_request_rate)"
    fi
    
    # Error rate monitoring
    local error_rate=$(query_prometheus 'rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100')
    local error_threshold=5  # 5%
    
    if (( $(echo "$error_rate > $error_threshold" | bc -l) )); then
        send_alert "CRITICAL" "High error rate: ${error_rate}% (threshold: ${error_threshold}%)"
    fi
}

# Self-healing actions
perform_self_healing() {
    local component=$1
    local issue=$2
    
    log "INFO" "Attempting self-healing for $component: $issue"
    
    case "$issue" in
        "pod_not_ready")
            log "INFO" "Restarting $component deployment"
            kubectl rollout restart deployment/"$component" -n "$NAMESPACE"
            
            # Wait for rollout to complete
            if kubectl rollout status deployment/"$component" -n "$NAMESPACE" --timeout=300s; then
                log "INFO" "Self-healing successful: $component restarted"
                send_alert "INFO" "Self-healing successful: $component restarted" "$component"
            else
                log "ERROR" "Self-healing failed: $component restart failed"
                send_alert "CRITICAL" "Self-healing failed: $component restart failed" "$component"
            fi
            ;;
        "high_memory")
            log "INFO" "Scaling up $component due to high memory usage"
            local current_replicas=$(kubectl get deployment "$component" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
            local new_replicas=$((current_replicas + 1))
            
            kubectl scale deployment "$component" -n "$NAMESPACE" --replicas="$new_replicas"
            log "INFO" "Scaled $component from $current_replicas to $new_replicas replicas"
            ;;
        "database_connection")
            log "INFO" "Restarting database connection pools"
            kubectl exec -n "$NAMESPACE" deployment/e2mgr -- pkill -f "redis-client" || true
            kubectl exec -n "$NAMESPACE" deployment/submgr -- pkill -f "redis-client" || true
            log "INFO" "Database connection pools restarted"
            ;;
    esac
}

# Generate health report
generate_health_report() {
    local report_file="/tmp/health-report-$(date +%Y%m%d-%H%M%S).json"
    
    log "INFO" "Generating health report: $report_file"
    
    cat > "$report_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "platform_status": "$(kubectl get pods -n "$NAMESPACE" --no-headers | grep -v Running | wc -l | xargs -I {} echo "{} unhealthy pods")",
    "resource_usage": {
        "cpu": $(query_prometheus 'avg(rate(container_cpu_usage_seconds_total{namespace="'$NAMESPACE'"}[5m])) * 100'),
        "memory": $(query_prometheus 'avg(container_memory_usage_bytes{namespace="'$NAMESPACE'"} / container_spec_memory_limit_bytes{namespace="'$NAMESPACE'"}) * 100'),
        "disk": $(query_prometheus 'avg((node_filesystem_size_bytes{fstype!="tmpfs"} - node_filesystem_avail_bytes{fstype!="tmpfs"}) / node_filesystem_size_bytes{fstype!="tmpfs"}) * 100')
    },
    "connectivity": {
        "e2_nodes": $(query_prometheus 'e2mgr_connected_nodes'),
        "active_subscriptions": $(query_prometheus 'submgr_active_subscriptions')
    },
    "performance": {
        "api_latency_p95": $(query_prometheus 'histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))'),
        "request_rate": $(query_prometheus 'rate(http_requests_total[5m])'),
        "error_rate": $(query_prometheus 'rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100')
    }
}
EOF
    
    echo "$report_file"
}

# Main monitoring loop
main() {
    local action=${1:-"monitor"}
    
    case "$action" in
        "monitor")
            log "INFO" "Starting production monitoring cycle"
            
            # Core component health checks
            local components=("e2term" "e2mgr" "submgr" "a1mediator" "o1mediator" "dbaas")
            local unhealthy_components=()
            
            for component in "${components[@]}"; do
                if ! check_component_health "$component"; then
                    unhealthy_components+=("$component")
                fi
            done
            
            # Resource monitoring
            monitor_resources
            
            # Service-specific monitoring
            monitor_e2_connectivity
            monitor_subscriptions
            monitor_database
            monitor_security
            monitor_performance
            
            # Self-healing for unhealthy components
            for component in "${unhealthy_components[@]}"; do
                perform_self_healing "$component" "pod_not_ready"
            done
            
            log "INFO" "Monitoring cycle completed"
            ;;
            
        "report")
            report_file=$(generate_health_report)
            echo "Health report generated: $report_file"
            cat "$report_file"
            ;;
            
        "alert-test")
            send_alert "INFO" "Test alert from monitoring system" "monitoring"
            ;;
            
        *)
            echo "Usage: $0 {monitor|report|alert-test}"
            echo "  monitor    - Run full monitoring cycle"
            echo "  report     - Generate health report"
            echo "  alert-test - Send test alert"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"