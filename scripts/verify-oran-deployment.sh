#!/bin/bash
# Verify O-RAN SC Near-RT RIC Deployment
# This script checks the status of O-RAN SC components and provides troubleshooting info

set -e

echo "🔍 Verifying O-RAN SC Near-RT RIC Deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "OK")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "WARN")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "ERROR")
            echo -e "${RED}❌ $message${NC}"
            ;;
        *)
            echo -e "ℹ️  $message"
            ;;
    esac
}

# Function to check namespace
check_namespace() {
    echo ""
    echo "📁 Checking namespace..."
    
    if kubectl get namespace ricplt > /dev/null 2>&1; then
        print_status "OK" "Namespace 'ricplt' exists"
    else
        print_status "ERROR" "Namespace 'ricplt' does not exist"
        echo "   Run: kubectl create namespace ricplt"
        return 1
    fi
}

# Function to check registry secret
check_registry_secret() {
    echo ""
    echo "🔐 Checking registry secret..."
    
    if kubectl get secret oran-sc-registry-secret -n ricplt > /dev/null 2>&1; then
        print_status "OK" "Registry secret exists"
        
        # Check if secret is properly configured
        local auth_data=$(kubectl get secret oran-sc-registry-secret -n ricplt -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d)
        if echo "$auth_data" | grep -q "nexus3.o-ran-sc.org:10002"; then
            print_status "OK" "Registry secret properly configured for O-RAN SC"
        else
            print_status "WARN" "Registry secret may not be configured correctly"
        fi
    else
        print_status "ERROR" "Registry secret not found"
        echo "   Run: ./scripts/load-oran-images.sh"
        return 1
    fi
}

# Function to check pod status
check_pods() {
    echo ""
    echo "🚀 Checking pod status..."
    
    local pods=(
        "ricplt-dbaas"
        "ricplt-e2term" 
        "ricplt-e2mgr"
        "ricplt-rtmgr"
        "ricplt-a1mediator"
        "ricplt-submgr"
        "ric-dashboard-api"
    )
    
    local all_running=true
    
    for pod in "${pods[@]}"; do
        local status=$(kubectl get pods -n ricplt -l app=$pod -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "NotFound")
        local ready=$(kubectl get pods -n ricplt -l app=$pod -o jsonpath='{.items[0].status.containerStatuses[0].ready}' 2>/dev/null || echo "false")
        
        case $status in
            "Running")
                if [ "$ready" = "true" ]; then
                    print_status "OK" "$pod is running and ready"
                else
                    print_status "WARN" "$pod is running but not ready"
                    all_running=false
                fi
                ;;
            "Pending")
                print_status "WARN" "$pod is pending"
                all_running=false
                # Show more details for pending pods
                local reason=$(kubectl get pods -n ricplt -l app=$pod -o jsonpath='{.items[0].status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || echo "Unknown")
                echo "   Reason: $reason"
                ;;
            "Failed"|"Error")
                print_status "ERROR" "$pod has failed"
                all_running=false
                ;;
            "NotFound")
                print_status "ERROR" "$pod not found"
                all_running=false
                ;;
            *)
                print_status "WARN" "$pod status: $status"
                all_running=false
                ;;
        esac
    done
    
    return $([ "$all_running" = true ] && echo 0 || echo 1)
}

# Function to check services
check_services() {
    echo ""
    echo "🌐 Checking services..."
    
    local services=(
        "ricplt-dbaas:6379"
        "ricplt-e2term:38000"
        "ricplt-e2mgr:3800"
        "ricplt-rtmgr:4560"
        "ricplt-a1mediator:10000"
        "ricplt-submgr:8080"
        "ric-dashboard-api:8080"
    )
    
    for service in "${services[@]}"; do
        local svc_name=${service%%:*}
        local svc_port=${service##*:}
        
        if kubectl get service $svc_name -n ricplt > /dev/null 2>&1; then
            local endpoints=$(kubectl get endpoints $svc_name -n ricplt -o jsonpath='{.subsets[0].addresses[0].ip}' 2>/dev/null || echo "")
            if [ -n "$endpoints" ]; then
                print_status "OK" "Service $svc_name has endpoints"
            else
                print_status "WARN" "Service $svc_name has no endpoints"
            fi
        else
            print_status "ERROR" "Service $svc_name not found"
        fi
    done
}

# Function to check image pull status
check_image_pulls() {
    echo ""
    echo "🖼️  Checking image pull status..."
    
    # Get all pods and check for ImagePullBackOff
    local failed_pods=$(kubectl get pods -n ricplt -o jsonpath='{range .items[*]}{.metadata.name}:{.status.containerStatuses[0].state.waiting.reason}{"\n"}{end}' 2>/dev/null | grep -E "(ImagePullBackOff|ErrImagePull)" || true)
    
    if [ -z "$failed_pods" ]; then
        print_status "OK" "No image pull failures detected"
    else
        print_status "ERROR" "Image pull failures detected:"
        echo "$failed_pods"
        echo ""
        echo "🔧 Troubleshooting steps:"
        echo "   1. Check registry connectivity: curl -I https://nexus3.o-ran-sc.org:10002/v2/"
        echo "   2. Verify registry secret: kubectl get secret oran-sc-registry-secret -n ricplt"
        echo "   3. Re-run image loading: ./scripts/load-oran-images.sh"
        echo "   4. Check specific pod logs: kubectl describe pod <pod-name> -n ricplt"
    fi
}

# Function to test dashboard accessibility
test_dashboard() {
    echo ""
    echo "🎛️  Testing dashboard accessibility..."
    
    # Check if dashboard service exists and has endpoints
    if kubectl get service ric-dashboard-api -n ricplt > /dev/null 2>&1; then
        local service_type=$(kubectl get service ric-dashboard-api -n ricplt -o jsonpath='{.spec.type}')
        print_status "OK" "Dashboard service exists (type: $service_type)"
        
        # Try to access dashboard internally
        local dashboard_pod=$(kubectl get pods -n ricplt -l app=ric-dashboard-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
        if [ -n "$dashboard_pod" ]; then
            if kubectl exec $dashboard_pod -n ricplt -- curl -s -f http://localhost:8080/api/health/alive > /dev/null 2>&1; then
                print_status "OK" "Dashboard is responding to health checks"
            else
                print_status "WARN" "Dashboard may not be fully ready"
            fi
        fi
        
        echo ""
        echo "🔗 Dashboard access options:"
        echo "   1. Port forward: kubectl port-forward -n ricplt svc/ric-dashboard-api 8080:8080"
        echo "   2. Access at: http://localhost:8080"
    else
        print_status "ERROR" "Dashboard service not found"
    fi
}

# Function to show troubleshooting information
show_troubleshooting() {
    echo ""
    echo "🔧 Troubleshooting Information"
    echo "================================"
    
    # Show cluster info
    echo ""
    echo "Cluster Information:"
    kubectl cluster-info --context=$(kubectl config current-context) 2>/dev/null || echo "Unable to get cluster info"
    
    # Show node status
    echo ""
    echo "Node Status:"
    kubectl get nodes -o wide
    
    # Show ricplt namespace resources
    echo ""
    echo "RIC Platform Resources:"
    kubectl get all -n ricplt 2>/dev/null || echo "No resources found in ricplt namespace"
    
    # Show recent events
    echo ""
    echo "Recent Events:"
    kubectl get events -n ricplt --sort-by=.metadata.creationTimestamp | tail -10
}

# Function to provide next steps
show_next_steps() {
    echo ""
    echo "📋 Next Steps"
    echo "=============="
    echo ""
    echo "If components are not running:"
    echo "  1. Apply deployment: kubectl apply -f deployments/oran-sc-fixed.yaml"
    echo "  2. Load images: ./scripts/load-oran-images.sh"
    echo "  3. Check logs: kubectl logs -n ricplt <pod-name>"
    echo ""
    echo "If images are failing to pull:"
    echo "  1. Test registry: curl -I https://nexus3.o-ran-sc.org:10002/v2/"
    echo "  2. Check network: ping nexus3.o-ran-sc.org"
    echo "  3. Clean Docker: docker system prune -a"
    echo ""
    echo "For development:"
    echo "  1. Access dashboard: kubectl port-forward -n ricplt svc/ric-dashboard-api 8080:8080"
    echo "  2. Check E2 connections: kubectl logs -n ricplt <e2term-pod> -f"
    echo "  3. Monitor A1 policies: kubectl logs -n ricplt <a1mediator-pod> -f"
}

# Main execution
main() {
    echo "🏁 Starting O-RAN SC deployment verification..."
    
    local exit_code=0
    
    # Run all checks
    check_namespace || exit_code=1
    check_registry_secret || exit_code=1
    check_pods || exit_code=1
    check_services || exit_code=1
    check_image_pulls || exit_code=1
    test_dashboard || exit_code=1
    
    echo ""
    if [ $exit_code -eq 0 ]; then
        print_status "OK" "All checks passed! O-RAN SC deployment is healthy."
    else
        print_status "WARN" "Some issues detected. Check the details above."
        show_troubleshooting
    fi
    
    show_next_steps
    
    return $exit_code
}

# Handle command line arguments
case "${1:-}" in
    --troubleshoot)
        show_troubleshooting
        ;;
    --help|-h)
        echo "Usage: $0 [--troubleshoot] [--help]"
        echo ""
        echo "Options:"
        echo "  --troubleshoot  Show detailed troubleshooting information"
        echo "  --help         Show this help message"
        ;;
    *)
        main "$@"
        ;;
esac