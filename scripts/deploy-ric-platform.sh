#!/bin/bash

# O-RAN Near-RT RIC Complete Deployment Script
# This script orchestrates the full deployment of the RIC platform

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOYMENT_DIR="$PROJECT_ROOT/deployments"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}==========================================${NC}"
echo -e "${GREEN}O-RAN Near-RT RIC Deployment Orchestrator${NC}"
echo -e "${GREEN}==========================================${NC}"

# Function to print status
print_status() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}kubectl not found! Please install kubectl.${NC}"
        exit 1
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}Cannot connect to Kubernetes cluster!${NC}"
        exit 1
    fi
    
    print_status "Prerequisites check passed"
}

# Function to install NGINX Ingress Controller
install_ingress_controller() {
    print_status "Installing NGINX Ingress Controller..."
    
    # Check if ingress-nginx namespace exists
    if kubectl get namespace ingress-nginx &> /dev/null; then
        print_status "NGINX Ingress Controller already installed"
    else
        kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
        print_status "Waiting for Ingress Controller to be ready..."
        kubectl wait --namespace ingress-nginx \
            --for=condition=ready pod \
            --selector=app.kubernetes.io/component=controller \
            --timeout=120s || true
    fi
}

# Function to clean up existing deployment
cleanup_existing() {
    print_status "Cleaning up existing deployments..."
    
    # Delete namespaces if they exist
    for ns in ricplt ricxapp monitoring; do
        if kubectl get namespace $ns &> /dev/null; then
            print_status "Deleting namespace $ns..."
            kubectl delete namespace $ns --wait=false
        fi
    done
    
    # Wait for namespaces to be deleted
    print_status "Waiting for cleanup to complete..."
    for ns in ricplt ricxapp monitoring; do
        kubectl wait --for=delete namespace/$ns --timeout=60s 2>/dev/null || true
    done
}

# Function to deploy RIC platform
deploy_ric_platform() {
    print_status "Deploying O-RAN Near-RT RIC Platform..."
    
    # Apply the complete deployment
    kubectl apply -f "$DEPLOYMENT_DIR/complete-ric-deployment.yaml"
    
    print_status "Waiting for deployments to be ready..."
    
    # Wait for ricplt namespace to be created
    kubectl wait --for=condition=established namespace/ricplt --timeout=30s
    kubectl wait --for=condition=established namespace/ricxapp --timeout=30s
    
    # Wait for all deployments in ricplt namespace
    kubectl wait --namespace ricplt --for=condition=available \
        deployment --all --timeout=300s || true
    
    # Wait for StatefulSet
    kubectl wait --namespace ricplt --for=condition=ready \
        pod -l app=ricplt-dbaas --timeout=120s || true
}

# Function to verify deployment
verify_deployment() {
    print_status "Verifying deployment..."
    
    echo -e "\n${GREEN}=== Namespace Status ===${NC}"
    kubectl get namespaces | grep -E "ricplt|ricxapp|monitoring"
    
    echo -e "\n${GREEN}=== RIC Platform Components ===${NC}"
    kubectl get pods -n ricplt
    
    echo -e "\n${GREEN}=== xApps ===${NC}"
    kubectl get pods -n ricxapp
    
    echo -e "\n${GREEN}=== Services ===${NC}"
    kubectl get services -n ricplt
    
    echo -e "\n${GREEN}=== Ingress ===${NC}"
    kubectl get ingress -n ricplt
}

# Function to get access information
get_access_info() {
    print_status "Getting access information..."
    
    # Get NodePort
    NODEPORT=$(kubectl get service ric-dashboard-nodeport -n ricplt -o jsonpath='{.spec.ports[0].nodePort}')
    
    # Get Node IP
    NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
    if [ -z "$NODE_IP" ]; then
        NODE_IP="localhost"
    fi
    
    echo -e "\n${GREEN}==========================================${NC}"
    echo -e "${GREEN}Deployment Complete!${NC}"
    echo -e "${GREEN}==========================================${NC}"
    echo -e "\n${YELLOW}Access the RIC Dashboard:${NC}"
    echo -e "  Browser URL: ${GREEN}http://${NODE_IP}:${NODEPORT}${NC}"
    echo -e "  Alternative: ${GREEN}http://localhost:${NODEPORT}${NC}"
    echo -e "\n${YELLOW}API Endpoints:${NC}"
    echo -e "  Status API: ${GREEN}http://${NODE_IP}:${NODEPORT}/api/status${NC}"
    echo -e "  Health Check: ${GREEN}http://${NODE_IP}:${NODEPORT}/health${NC}"
    echo -e "\n${YELLOW}Port Forwarding (if needed):${NC}"
    echo -e "  kubectl port-forward -n ricplt svc/ric-dashboard-api 8080:8080"
    echo -e "\n${YELLOW}Component Access:${NC}"
    echo -e "  E2 Manager: kubectl port-forward -n ricplt svc/service-ricplt-e2mgr-http 3800:3800"
    echo -e "  A1 Mediator: kubectl port-forward -n ricplt svc/service-ricplt-a1mediator-http 10000:10000"
}

# Function to setup port forwarding
setup_port_forwarding() {
    print_status "Setting up port forwarding for easy access..."
    
    # Kill any existing port-forward processes
    pkill -f "kubectl port-forward" || true
    
    # Start port forwarding in background
    kubectl port-forward -n ricplt svc/ric-dashboard-api 8080:8080 &
    PF_PID=$!
    
    echo -e "\n${GREEN}Port forwarding established!${NC}"
    echo -e "Dashboard accessible at: ${GREEN}http://localhost:8080${NC}"
    echo -e "Port forward PID: $PF_PID"
    echo -e "\nTo stop port forwarding: kill $PF_PID"
}

# Main deployment flow
main() {
    echo -e "${YELLOW}Starting O-RAN Near-RT RIC deployment...${NC}\n"
    
    # Check prerequisites
    check_prerequisites
    
    # Ask user for clean deployment
    read -p "Do you want to clean up existing deployments? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup_existing
    fi
    
    # Install ingress controller
    install_ingress_controller
    
    # Deploy RIC platform
    deploy_ric_platform
    
    # Verify deployment
    verify_deployment
    
    # Get access information
    get_access_info
    
    # Ask if user wants port forwarding
    read -p "Do you want to setup port forwarding for easy access? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        setup_port_forwarding
    fi
    
    echo -e "\n${GREEN}Deployment orchestration complete!${NC}"
}

# Run main function
main "$@"