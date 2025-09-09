#!/bin/bash

# O-RAN SC Deployment Preparation Script
# Orchestrator Agent - Deployment Coordination

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m'

echo -e "${MAGENTA}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║     O-RAN SC Official Deployment Preparation          ║${NC}"
echo -e "${MAGENTA}╚════════════════════════════════════════════════════════╝${NC}"
echo

# Configuration
DEPLOYMENT_DIR="deployments/oran-sc"
ARTIFACTS_DIR="build/artifacts"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-ghcr.io/o-ran-sc}"
VERSION="${VERSION:-latest}"

# Function to check build status
check_build_status() {
    echo -e "${BLUE}Checking build status...${NC}"
    
    local all_built=true
    local components=(
        "dashboard-api"
        "analytics-api"
        "e2-telemetry-processor"
        "kpi-calculator"
        "ml-predictor"
        "performance-analytics"
        "performance-optimizer"
        "telemetry-collector"
        "xapp-hello-world"
    )
    
    for component in "${components[@]}"; do
        if [ -f "bin/$component" ] || [ -f "bin/${component}.exe" ]; then
            echo -e "  ${GREEN}✓${NC} $component"
        else
            echo -e "  ${RED}✗${NC} $component - NOT BUILT"
            all_built=false
        fi
    done
    
    if [ "$all_built" = false ]; then
        echo -e "\n${RED}ERROR: Not all components are built. Please fix build errors first.${NC}"
        echo "Run: ./scripts/build-verification.sh"
        exit 1
    fi
    
    echo -e "${GREEN}All components built successfully!${NC}\n"
}

# Function to prepare deployment artifacts
prepare_artifacts() {
    echo -e "${BLUE}Preparing deployment artifacts...${NC}"
    
    mkdir -p "$ARTIFACTS_DIR"
    mkdir -p "$DEPLOYMENT_DIR"
    
    # Copy binaries
    echo "  Copying binaries..."
    cp -r bin/* "$ARTIFACTS_DIR/" 2>/dev/null || true
    
    # Copy configurations
    echo "  Copying configurations..."
    cp -r configs/* "$DEPLOYMENT_DIR/" 2>/dev/null || true
    
    # Copy Helm charts
    echo "  Copying Helm charts..."
    cp -r helm/* "$DEPLOYMENT_DIR/" 2>/dev/null || true
    
    echo -e "${GREEN}Artifacts prepared${NC}\n"
}

# Function to validate configurations
validate_configs() {
    echo -e "${BLUE}Validating configurations...${NC}"
    
    # Check for required config files
    local required_configs=(
        "configs/rmr_routes.yaml"
        "configs/service-models.yaml"
        "configs/component-configs.yaml"
    )
    
    for config in "${required_configs[@]}"; do
        if [ -f "$config" ]; then
            echo -e "  ${GREEN}✓${NC} $config"
        else
            echo -e "  ${YELLOW}⚠${NC} $config - Missing (using defaults)"
        fi
    done
    
    echo
}

# Function to build Docker images
build_docker_images() {
    echo -e "${BLUE}Building Docker images for O-RAN SC...${NC}"
    
    if ! command -v docker &> /dev/null; then
        echo -e "${YELLOW}Docker not available. Skipping image build.${NC}\n"
        return
    fi
    
    # Build main RIC image
    echo "  Building near-rt-ric image..."
    docker build -t "${DOCKER_REGISTRY}/near-rt-ric:${VERSION}" \
        -f Dockerfile \
        --build-arg VERSION="${VERSION}" \
        --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        . || echo -e "${YELLOW}Warning: Docker build failed${NC}"
    
    # Build xApp image
    echo "  Building xapp-hello-world image..."
    docker build -t "${DOCKER_REGISTRY}/xapp-hello-world:${VERSION}" \
        -f deployments/docker/Dockerfile.xapp \
        . 2>/dev/null || echo -e "${YELLOW}Warning: xApp Docker build failed${NC}"
    
    echo -e "${GREEN}Docker images built${NC}\n"
}

# Function to generate deployment manifests
generate_manifests() {
    echo -e "${BLUE}Generating O-RAN SC deployment manifests...${NC}"
    
    cat > "$DEPLOYMENT_DIR/oran-sc-deployment.yaml" <<'EOF'
apiVersion: v1
kind: Namespace
metadata:
  name: oran-sc
  labels:
    name: oran-sc
    oran.sc/component: near-rt-ric
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: near-rt-ric
  namespace: oran-sc
  labels:
    app: near-rt-ric
    version: v1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: near-rt-ric
  template:
    metadata:
      labels:
        app: near-rt-ric
        version: v1
    spec:
      containers:
      - name: dashboard-api
        image: ghcr.io/o-ran-sc/near-rt-ric:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 4560
          name: rmr-data
        - containerPort: 4561
          name: rmr-route
        env:
        - name: RMR_SEED_RT
          value: "/etc/rmr/routes.yaml"
        volumeMounts:
        - name: rmr-config
          mountPath: /etc/rmr
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: rmr-config
        configMap:
          name: rmr-routes
---
apiVersion: v1
kind: Service
metadata:
  name: near-rt-ric
  namespace: oran-sc
spec:
  type: ClusterIP
  ports:
  - port: 8080
    targetPort: 8080
    name: http
  - port: 4560
    targetPort: 4560
    name: rmr-data
  selector:
    app: near-rt-ric
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: rmr-routes
  namespace: oran-sc
data:
  routes.yaml: |
    newrt|start
    mse|10060|10061|dashboard-api:4560
    mse|10062|10063|xapp-hello-world:4560
    newrt|end
EOF
    
    echo -e "${GREEN}Manifests generated${NC}\n"
}

# Function to create deployment package
create_deployment_package() {
    echo -e "${BLUE}Creating O-RAN SC deployment package...${NC}"
    
    local package_name="oran-sc-near-rt-ric-${VERSION}.tar.gz"
    
    tar -czf "$package_name" \
        -C . \
        bin/ \
        configs/ \
        deployments/ \
        helm/ \
        scripts/deploy-to-oran-sc.sh \
        README.md \
        2>/dev/null || echo -e "${YELLOW}Warning: Some files not found${NC}"
    
    echo -e "${GREEN}Deployment package created: $package_name${NC}\n"
}

# Function to generate deployment instructions
generate_instructions() {
    echo -e "${BLUE}Generating deployment instructions...${NC}"
    
    cat > "$DEPLOYMENT_DIR/DEPLOYMENT_INSTRUCTIONS.md" <<'EOF'
# O-RAN SC Near-RT RIC Deployment Instructions

## Prerequisites
- Kubernetes cluster (1.24+) with O-RAN SC components
- kubectl configured
- Helm 3.x installed
- Access to O-RAN SC container registry

## Deployment Steps

### 1. Deploy Using Kubernetes Manifests
```bash
kubectl apply -f deployments/oran-sc/oran-sc-deployment.yaml
```

### 2. Deploy Using Helm
```bash
helm install near-rt-ric ./helm/ric-platform \
  --namespace oran-sc \
  --create-namespace \
  --values helm/ric-platform/values.yaml
```

### 3. Verify Deployment
```bash
# Check pods
kubectl get pods -n oran-sc

# Check services
kubectl get svc -n oran-sc

# Check logs
kubectl logs -n oran-sc deployment/near-rt-ric
```

### 4. Access Dashboard
```bash
# Port forward
kubectl port-forward -n oran-sc svc/near-rt-ric 8080:8080

# Access at http://localhost:8080
```

## Configuration

### RMR Routes
Edit `configs/rmr_routes.yaml` to configure message routing

### Service Models
Configure E2 service models in `configs/service-models.yaml`

## Troubleshooting

### Pod not starting
- Check logs: `kubectl logs -n oran-sc <pod-name>`
- Check events: `kubectl describe pod -n oran-sc <pod-name>`

### RMR connectivity issues
- Verify RMR routes configuration
- Check network policies

## Integration with O-RAN SC Components

### A1 Mediator
Connect to A1 Mediator at: `http://a1-mediator:8080`

### E2 Manager
E2 Manager endpoint: `http://e2-manager:3800`

### SMO Integration
Configure SMO endpoint in environment variables

## Support
For issues, refer to O-RAN SC documentation or file an issue in the repository.
EOF
    
    echo -e "${GREEN}Instructions generated${NC}\n"
}

# Function to run final validation
final_validation() {
    echo -e "${BLUE}Running final validation...${NC}"
    
    local ready=true
    
    # Check binaries
    echo -n "  Checking binaries... "
    if [ -d "bin" ] && [ "$(ls -A bin)" ]; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}FAILED${NC}"
        ready=false
    fi
    
    # Check configs
    echo -n "  Checking configurations... "
    if [ -d "configs" ] && [ "$(ls -A configs)" ]; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${YELLOW}WARNING${NC}"
    fi
    
    # Check deployments
    echo -n "  Checking deployment files... "
    if [ -f "$DEPLOYMENT_DIR/oran-sc-deployment.yaml" ]; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}FAILED${NC}"
        ready=false
    fi
    
    echo
    
    if [ "$ready" = true ]; then
        echo -e "${GREEN}✅ READY FOR O-RAN SC DEPLOYMENT${NC}"
        return 0
    else
        echo -e "${RED}❌ NOT READY FOR DEPLOYMENT${NC}"
        return 1
    fi
}

# Main execution
main() {
    echo -e "${YELLOW}Starting deployment preparation...${NC}\n"
    
    # Step 1: Check build status
    check_build_status
    
    # Step 2: Prepare artifacts
    prepare_artifacts
    
    # Step 3: Validate configurations
    validate_configs
    
    # Step 4: Build Docker images
    build_docker_images
    
    # Step 5: Generate manifests
    generate_manifests
    
    # Step 6: Create deployment package
    create_deployment_package
    
    # Step 7: Generate instructions
    generate_instructions
    
    # Step 8: Final validation
    if final_validation; then
        echo -e "${MAGENTA}╔════════════════════════════════════════════════════════╗${NC}"
        echo -e "${MAGENTA}║                 DEPLOYMENT READY                      ║${NC}"
        echo -e "${MAGENTA}╚════════════════════════════════════════════════════════╝${NC}"
        echo
        echo -e "${GREEN}Next steps:${NC}"
        echo "1. Review deployment instructions: $DEPLOYMENT_DIR/DEPLOYMENT_INSTRUCTIONS.md"
        echo "2. Deploy to O-RAN SC cluster: kubectl apply -f $DEPLOYMENT_DIR/oran-sc-deployment.yaml"
        echo "3. Monitor deployment: kubectl get pods -n oran-sc -w"
        echo
        echo -e "${BLUE}Deployment package: oran-sc-near-rt-ric-${VERSION}.tar.gz${NC}"
    else
        echo -e "${RED}Deployment preparation incomplete. Please fix issues and try again.${NC}"
        exit 1
    fi
}

# Run main function
main "$@"