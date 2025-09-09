#!/bin/bash

# O-RAN SC Near-RT RIC Infrastructure Preparation Script
# Follows O-RAN SC L Release standards for production deployment
# Author: Infrastructure Setup for O-RAN SC L Release

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NAMESPACE="ricplt"
XAPP_NAMESPACE="ricxapp"
INFRA_NAMESPACE="ricinfra"

# O-RAN SC L Release specific versions
ORAN_SC_RELEASE="l-release"
KUBERNETES_VERSION="v1.28.0"
HELM_VERSION="v3.12.0"
DOCKER_VERSION="24.0.0"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Help function
show_help() {
    cat << EOF
O-RAN SC Near-RT RIC Infrastructure Preparation Script

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    --check-only           Only check prerequisites without setup
    --docker-setup         Setup Docker environment
    --k8s-setup           Setup Kubernetes cluster
    --helm-setup          Setup Helm repositories
    --security-setup      Setup security policies
    --network-setup       Setup networking components
    --storage-setup       Setup persistent storage
    --monitoring-setup    Setup monitoring stack
    --all                 Setup all components

EXAMPLES:
    # Check prerequisites only
    $0 --check-only

    # Setup Docker environment
    $0 --docker-setup

    # Setup complete infrastructure
    $0 --all

EOF
}

# Parse command line arguments
parse_args() {
    CHECK_ONLY=false
    DOCKER_SETUP=false
    K8S_SETUP=false
    HELM_SETUP=false
    SECURITY_SETUP=false
    NETWORK_SETUP=false
    STORAGE_SETUP=false
    MONITORING_SETUP=false
    ALL_SETUP=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            --check-only)
                CHECK_ONLY=true
                shift
                ;;
            --docker-setup)
                DOCKER_SETUP=true
                shift
                ;;
            --k8s-setup)
                K8S_SETUP=true
                shift
                ;;
            --helm-setup)
                HELM_SETUP=true
                shift
                ;;
            --security-setup)
                SECURITY_SETUP=true
                shift
                ;;
            --network-setup)
                NETWORK_SETUP=true
                shift
                ;;
            --storage-setup)
                STORAGE_SETUP=true
                shift
                ;;
            --monitoring-setup)
                MONITORING_SETUP=true
                shift
                ;;
            --all)
                ALL_SETUP=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Set all flags if --all is specified
    if [[ "$ALL_SETUP" == true ]]; then
        DOCKER_SETUP=true
        K8S_SETUP=true
        HELM_SETUP=true
        SECURITY_SETUP=true
        NETWORK_SETUP=true
        STORAGE_SETUP=true
        MONITORING_SETUP=true
    fi
}

# Check system prerequisites
check_prerequisites() {
    log_info "Checking system prerequisites for O-RAN SC L Release..."
    
    local errors=0
    
    # Check OS compatibility
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        log_success "Linux OS detected"
        
        # Check specific distributions
        if [[ -f /etc/os-release ]]; then
            . /etc/os-release
            log_info "OS: $NAME $VERSION"
            
            # Recommended distributions for O-RAN SC
            case $ID in
                ubuntu)
                    if [[ "$VERSION_ID" < "20.04" ]]; then
                        log_warning "Ubuntu 20.04 LTS or later recommended"
                    fi
                    ;;
                centos|rhel)
                    if [[ "${VERSION_ID%%.*}" -lt 8 ]]; then
                        log_warning "CentOS/RHEL 8 or later recommended"
                    fi
                    ;;
            esac
        fi
    else
        log_error "Non-Linux OS detected. O-RAN SC requires Linux environment"
        ((errors++))
    fi
    
    # Check CPU architecture
    ARCH=$(uname -m)
    if [[ "$ARCH" != "x86_64" ]] && [[ "$ARCH" != "aarch64" ]]; then
        log_error "Unsupported architecture: $ARCH. O-RAN SC requires x86_64 or ARM64"
        ((errors++))
    else
        log_success "Architecture: $ARCH"
    fi
    
    # Check memory (minimum 8GB recommended for O-RAN SC)
    MEMORY_GB=$(free -g | awk '/^Mem:/{print $2}')
    if [[ "$MEMORY_GB" -lt 8 ]]; then
        log_warning "System has ${MEMORY_GB}GB RAM. 8GB+ recommended for O-RAN SC"
    else
        log_success "Memory: ${MEMORY_GB}GB"
    fi
    
    # Check disk space (minimum 50GB recommended)
    DISK_AVAILABLE=$(df / | awk 'NR==2{print int($4/1024/1024)}')
    if [[ "$DISK_AVAILABLE" -lt 50 ]]; then
        log_warning "Available disk space: ${DISK_AVAILABLE}GB. 50GB+ recommended"
    else
        log_success "Available disk space: ${DISK_AVAILABLE}GB"
    fi
    
    # Check required tools
    local required_tools=("curl" "wget" "git" "tar" "gzip" "systemctl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool not found: $tool"
            ((errors++))
        else
            log_success "Found: $tool"
        fi
    done
    
    # Check network connectivity
    if ! ping -c 1 8.8.8.8 &> /dev/null; then
        log_error "No internet connectivity detected"
        ((errors++))
    else
        log_success "Internet connectivity verified"
    fi
    
    # Check for existing conflicting services
    local conflicting_services=("docker" "containerd" "k3s" "microk8s")
    for service in "${conflicting_services[@]}"; do
        if systemctl is-active --quiet "$service" 2>/dev/null; then
            log_warning "Service $service is already running - may cause conflicts"
        fi
    done
    
    if [[ $errors -gt 0 ]]; then
        log_error "Prerequisites check failed with $errors errors"
        return 1
    else
        log_success "Prerequisites check passed"
        return 0
    fi
}

# Setup Docker environment
setup_docker() {
    log_info "Setting up Docker environment for O-RAN SC..."
    
    # Remove old Docker versions
    if command -v docker &> /dev/null; then
        CURRENT_DOCKER_VERSION=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
        log_info "Current Docker version: $CURRENT_DOCKER_VERSION"
        
        # Check if we need to upgrade
        if [[ "$CURRENT_DOCKER_VERSION" < "${DOCKER_VERSION#v}" ]]; then
            log_info "Upgrading Docker to version $DOCKER_VERSION..."
            sudo apt-get remove -y docker docker-engine docker.io containerd runc || true
        else
            log_info "Docker version is sufficient"
            return 0
        fi
    fi
    
    # Install Docker CE
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    
    # Configure Docker for O-RAN SC requirements
    sudo mkdir -p /etc/docker
    cat <<EOF | sudo tee /etc/docker/daemon.json
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "10"
  },
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ],
  "insecure-registries": [
    "localhost:5000",
    "registry.o-ran-sc.org"
  ],
  "registry-mirrors": [
    "https://registry.o-ran-sc.org"
  ]
}
EOF
    
    # Start and enable Docker
    sudo systemctl daemon-reload
    sudo systemctl enable docker
    sudo systemctl start docker
    
    # Add user to docker group
    sudo usermod -aG docker "$USER"
    
    # Test Docker installation
    if sudo docker run --rm hello-world &> /dev/null; then
        log_success "Docker installation verified"
    else
        log_error "Docker installation failed"
        return 1
    fi
    
    # Install Docker Compose if not present
    if ! command -v docker-compose &> /dev/null; then
        sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        sudo chmod +x /usr/local/bin/docker-compose
        log_success "Docker Compose installed"
    fi
}

# Setup Kubernetes cluster
setup_kubernetes() {
    log_info "Setting up Kubernetes cluster for O-RAN SC..."
    
    # Install kubectl
    if ! command -v kubectl &> /dev/null; then
        curl -LO "https://dl.k8s.io/release/$KUBERNETES_VERSION/bin/linux/amd64/kubectl"
        sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
        rm kubectl
        log_success "kubectl installed"
    fi
    
    # Install kind for local development cluster
    if ! command -v kind &> /dev/null; then
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
        chmod +x ./kind
        sudo mv ./kind /usr/local/bin/kind
        log_success "kind installed"
    fi
    
    # Create kind cluster configuration for O-RAN SC
    cat <<EOF > "$PROJECT_ROOT/kind-oran-config.yaml"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: oran-near-rt-ric
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
        authorization-mode: "AlwaysAllow"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  - containerPort: 36421
    hostPort: 36421
    protocol: SCTP  # E2 Interface
  - containerPort: 38412
    hostPort: 38412
    protocol: SCTP  # N2 Interface
  - containerPort: 38422
    hostPort: 38422
    protocol: UDP   # F1-C Interface
  - containerPort: 2152
    hostPort: 2152
    protocol: UDP   # GTP-U
- role: worker
  labels:
    node-type: ric-platform
- role: worker
  labels:
    node-type: xapp-runtime
- role: worker
  labels:
    node-type: network-functions
networking:
  disableDefaultCNI: false
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/12"
kubeadmConfigPatches:
- |
  kind: ClusterConfiguration
  networking:
    podSubnet: "10.244.0.0/16"
    serviceSubnet: "10.96.0.0/12"
  etcd:
    local:
      dataDir: "/var/lib/etcd"
EOF
    
    # Create cluster if not exists
    if ! kind get clusters | grep -q "oran-near-rt-ric"; then
        log_info "Creating O-RAN SC Kubernetes cluster..."
        kind create cluster --config "$PROJECT_ROOT/kind-oran-config.yaml" --wait 300s
        log_success "Kubernetes cluster created"
    else
        log_info "O-RAN SC cluster already exists"
    fi
    
    # Verify cluster access
    if kubectl cluster-info &> /dev/null; then
        log_success "Kubernetes cluster accessible"
    else
        log_error "Cannot access Kubernetes cluster"
        return 1
    fi
}

# Setup Helm repositories
setup_helm() {
    log_info "Setting up Helm for O-RAN SC..."
    
    # Install Helm
    if ! command -v helm &> /dev/null; then
        curl https://baltocdn.com/helm/signing.asc | gpg --dearmor | sudo tee /usr/share/keyrings/helm.gpg > /dev/null
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list
        sudo apt-get update
        sudo apt-get install -y helm
        log_success "Helm installed"
    fi
    
    # Add O-RAN SC Helm repositories
    helm repo add oran-sc-platform https://gerrit.o-ran-sc.org/r/it/dep/ric-platform/helm-manager/helm-charts
    helm repo add stable https://charts.helm.sh/stable
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo add grafana https://grafana.github.io/helm-charts
    helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
    
    # Update repositories
    helm repo update
    log_success "Helm repositories configured"
}

# Setup security policies
setup_security() {
    log_info "Setting up security policies for O-RAN SC..."
    
    # Create security namespace
    kubectl create namespace "$INFRA_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Install Pod Security Standards
    kubectl label namespace "$NAMESPACE" pod-security.kubernetes.io/enforce=restricted --overwrite
    kubectl label namespace "$XAPP_NAMESPACE" pod-security.kubernetes.io/enforce=restricted --overwrite
    kubectl label namespace "$INFRA_NAMESPACE" pod-security.kubernetes.io/enforce=privileged --overwrite
    
    # Create RBAC for O-RAN SC components
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: oran-sc-platform
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: oran-sc-platform-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints", "persistentvolumeclaims", "events", "configmaps", "secrets"]
  verbs: ["*"]
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets", "replicasets", "statefulsets"]
  verbs: ["*"]
- apiGroups: ["monitoring.coreos.com"]
  resources: ["servicemonitors"]
  verbs: ["get", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: oran-sc-platform-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: oran-sc-platform-role
subjects:
- kind: ServiceAccount
  name: oran-sc-platform
  namespace: $NAMESPACE
EOF
    
    # Install Network Policies for O-RAN SC security
    cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: oran-sc-network-policy
  namespace: $NAMESPACE
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: $NAMESPACE
    - namespaceSelector:
        matchLabels:
          name: $XAPP_NAMESPACE
    - namespaceSelector:
        matchLabels:
          name: $INFRA_NAMESPACE
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: $NAMESPACE
    - namespaceSelector:
        matchLabels:
          name: $XAPP_NAMESPACE
    - namespaceSelector:
        matchLabels:
          name: $INFRA_NAMESPACE
  - to: []
    ports:
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
  - to: []
    ports:
    - protocol: TCP
      port: 443
EOF
    
    log_success "Security policies configured"
}

# Setup networking components
setup_networking() {
    log_info "Setting up networking for O-RAN SC..."
    
    # Install CNI plugins for advanced networking
    kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/v3.9.3/deployments/multus-daemonset-thick.yml
    
    # Install Whereabouts IPAM for secondary networks
    kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/v0.6.1/doc/crds/daemonset-install.yaml
    kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/v0.6.1/doc/crds/whereabouts.cni.cncf.io_ippools.yaml
    
    # Create network attachment definitions for O-RAN interfaces
    cat <<EOF | kubectl apply -f -
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: e2-network
  namespace: $NAMESPACE
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "name": "e2-network",
      "type": "bridge",
      "bridge": "e2br0",
      "isGateway": true,
      "ipam": {
        "type": "whereabouts",
        "range": "192.168.10.0/24"
      }
    }
---
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: f1-network
  namespace: $NAMESPACE
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "name": "f1-network",
      "type": "bridge",
      "bridge": "f1br0",
      "isGateway": true,
      "ipam": {
        "type": "whereabouts",
        "range": "192.168.20.0/24"
      }
    }
---
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: n2-network
  namespace: $NAMESPACE
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "name": "n2-network",
      "type": "bridge",
      "bridge": "n2br0",
      "isGateway": true,
      "ipam": {
        "type": "whereabouts",
        "range": "192.168.30.0/24"
      }
    }
EOF
    
    log_success "Networking components configured"
}

# Setup persistent storage
setup_storage() {
    log_info "Setting up persistent storage for O-RAN SC..."
    
    # Install local path provisioner for development
    kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.24/deploy/local-path-storage.yaml
    
    # Set as default storage class
    kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
    
    # Create storage classes for different O-RAN SC requirements
    cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: oran-fast-ssd
  annotations:
    storageclass.kubernetes.io/is-default-class: "false"
provisioner: rancher.io/local-path
parameters:
  nodePath: /opt/local-path-provisioner/fast-ssd
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: oran-database
  annotations:
    storageclass.kubernetes.io/is-default-class: "false"
provisioner: rancher.io/local-path
parameters:
  nodePath: /opt/local-path-provisioner/database
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Retain
EOF
    
    log_success "Storage configured"
}

# Setup monitoring stack
setup_monitoring() {
    log_info "Setting up monitoring stack for O-RAN SC..."
    
    # Create monitoring namespace
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
    
    # Install Prometheus Operator
    helm upgrade --install prometheus-operator prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --set prometheus.service.type=ClusterIP \
        --set grafana.service.type=ClusterIP \
        --set alertmanager.service.type=ClusterIP \
        --wait
    
    # Install Jaeger for distributed tracing
    helm upgrade --install jaeger jaegertracing/jaeger \
        --namespace monitoring \
        --set provisionDataStore.cassandra=false \
        --set allInOne.enabled=true \
        --set storage.type=memory \
        --wait
    
    # Create ServiceMonitor for O-RAN SC components
    cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: oran-sc-metrics
  namespace: monitoring
  labels:
    app: oran-sc-platform
spec:
  selector:
    matchLabels:
      app: ric-platform
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
  namespaceSelector:
    matchNames:
    - $NAMESPACE
    - $XAPP_NAMESPACE
EOF
    
    log_success "Monitoring stack configured"
}

# Create namespaces with proper labels
create_namespaces() {
    log_info "Creating O-RAN SC namespaces..."
    
    # Create RIC Platform namespace
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | \
    kubectl apply -f -
    kubectl label namespace "$NAMESPACE" name="$NAMESPACE" --overwrite
    kubectl label namespace "$NAMESPACE" oran.org/component=platform --overwrite
    
    # Create xApp namespace
    kubectl create namespace "$XAPP_NAMESPACE" --dry-run=client -o yaml | \
    kubectl apply -f -
    kubectl label namespace "$XAPP_NAMESPACE" name="$XAPP_NAMESPACE" --overwrite
    kubectl label namespace "$XAPP_NAMESPACE" oran.org/component=xapp --overwrite
    
    # Create Infrastructure namespace
    kubectl create namespace "$INFRA_NAMESPACE" --dry-run=client -o yaml | \
    kubectl apply -f -
    kubectl label namespace "$INFRA_NAMESPACE" name="$INFRA_NAMESPACE" --overwrite
    kubectl label namespace "$INFRA_NAMESPACE" oran.org/component=infrastructure --overwrite
    
    log_success "Namespaces created and labeled"
}

# Generate deployment validation script
create_validation_script() {
    log_info "Creating deployment validation script..."
    
    cat <<'EOF' > "$PROJECT_ROOT/scripts/validate-infrastructure.sh"
#!/bin/bash

# O-RAN SC Infrastructure Validation Script

set -euo pipefail

NAMESPACE="ricplt"
XAPP_NAMESPACE="ricxapp"
INFRA_NAMESPACE="ricinfra"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

validate_cluster() {
    log_info "Validating Kubernetes cluster..."
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot access Kubernetes cluster"
        return 1
    fi
    
    # Check node readiness
    NOT_READY_NODES=$(kubectl get nodes --no-headers | grep -v Ready | wc -l)
    if [[ "$NOT_READY_NODES" -gt 0 ]]; then
        log_error "$NOT_READY_NODES nodes are not ready"
        kubectl get nodes
        return 1
    fi
    
    log_success "Cluster validation passed"
}

validate_namespaces() {
    log_info "Validating namespaces..."
    
    local namespaces=("$NAMESPACE" "$XAPP_NAMESPACE" "$INFRA_NAMESPACE" "monitoring")
    for ns in "${namespaces[@]}"; do
        if ! kubectl get namespace "$ns" &> /dev/null; then
            log_error "Namespace $ns does not exist"
            return 1
        else
            log_success "Namespace $ns exists"
        fi
    done
}

validate_storage() {
    log_info "Validating storage..."
    
    if ! kubectl get storageclass local-path &> /dev/null; then
        log_error "Default storage class not found"
        return 1
    fi
    
    log_success "Storage validation passed"
}

validate_networking() {
    log_info "Validating networking..."
    
    # Check CNI
    if ! kubectl get pods -n kube-system -l app=multus | grep -q Running; then
        log_warning "Multus CNI not running"
    else
        log_success "Multus CNI running"
    fi
    
    # Check network attachment definitions
    local networks=("e2-network" "f1-network" "n2-network")
    for net in "${networks[@]}"; do
        if kubectl get network-attachment-definitions -n "$NAMESPACE" "$net" &> /dev/null; then
            log_success "Network $net configured"
        else
            log_warning "Network $net not configured"
        fi
    done
}

validate_monitoring() {
    log_info "Validating monitoring stack..."
    
    # Check Prometheus
    if kubectl get pods -n monitoring -l app.kubernetes.io/name=prometheus &> /dev/null; then
        log_success "Prometheus deployed"
    else
        log_warning "Prometheus not deployed"
    fi
    
    # Check Grafana
    if kubectl get pods -n monitoring -l app.kubernetes.io/name=grafana &> /dev/null; then
        log_success "Grafana deployed"
    else
        log_warning "Grafana not deployed"
    fi
}

main() {
    log_info "O-RAN SC Infrastructure Validation"
    log_info "=================================="
    
    validate_cluster
    validate_namespaces
    validate_storage
    validate_networking
    validate_monitoring
    
    log_success "Infrastructure validation completed!"
}

main "$@"
EOF
    
    chmod +x "$PROJECT_ROOT/scripts/validate-infrastructure.sh"
    log_success "Validation script created"
}

# Main function
main() {
    log_info "O-RAN SC Near-RT RIC Infrastructure Preparation"
    log_info "==============================================="
    
    # Parse arguments
    parse_args "$@"
    
    # Check prerequisites
    if ! check_prerequisites; then
        if [[ "$CHECK_ONLY" == true ]]; then
            exit 1
        else
            log_error "Prerequisites check failed. Use --check-only to see details."
            exit 1
        fi
    fi
    
    if [[ "$CHECK_ONLY" == true ]]; then
        log_success "Prerequisites check completed successfully!"
        exit 0
    fi
    
    # Execute setup steps based on flags
    if [[ "$DOCKER_SETUP" == true ]]; then
        setup_docker
    fi
    
    if [[ "$K8S_SETUP" == true ]]; then
        setup_kubernetes
        create_namespaces
    fi
    
    if [[ "$HELM_SETUP" == true ]]; then
        setup_helm
    fi
    
    if [[ "$SECURITY_SETUP" == true ]]; then
        setup_security
    fi
    
    if [[ "$NETWORK_SETUP" == true ]]; then
        setup_networking
    fi
    
    if [[ "$STORAGE_SETUP" == true ]]; then
        setup_storage
    fi
    
    if [[ "$MONITORING_SETUP" == true ]]; then
        setup_monitoring
    fi
    
    # Create validation script
    create_validation_script
    
    log_success "O-RAN SC infrastructure preparation completed!"
    log_info "Next steps:"
    echo "1. Run: ./scripts/validate-infrastructure.sh"
    echo "2. Deploy O-RAN SC platform: ./scripts/deploy-oran-sc-platform.sh"
    echo "3. Access monitoring: kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090"
    echo "4. Access Grafana: kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80"
}

# Execute main function with all arguments
main "$@"