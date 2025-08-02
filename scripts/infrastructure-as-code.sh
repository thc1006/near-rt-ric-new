#!/bin/bash
# infrastructure-as-code.sh - Infrastructure as Code management for O-RAN RIC

set -euo pipefail

# Configuration
NAMESPACE="oran-ric"
TERRAFORM_DIR="./terraform"
ANSIBLE_DIR="./ansible"
KUSTOMIZE_DIR="./kustomize"
STATE_BUCKET="oran-ric-terraform-state"
REGION="us-west-2"

# Logging function
log_iac() {
    echo "$(date -Iseconds) [IAC] $1"
}

# Initialize Terraform backend
init_terraform() {
    log_iac "Initializing Terraform backend"
    
    cd "$TERRAFORM_DIR"
    
    # Create backend configuration
    cat > backend.tf << EOF
terraform {
  backend "s3" {
    bucket = "$STATE_BUCKET"
    key    = "oran-ric/terraform.tfstate"
    region = "$REGION"
    
    # Enable state locking
    dynamodb_table = "terraform-state-lock"
    encrypt        = true
  }
}
EOF
    
    # Initialize Terraform
    terraform init
    
    cd - > /dev/null
}

# Plan infrastructure changes
plan_infrastructure() {
    local environment=${1:-"production"}
    
    log_iac "Planning infrastructure changes for $environment"
    
    cd "$TERRAFORM_DIR"
    
    # Select workspace
    terraform workspace select "$environment" || terraform workspace new "$environment"
    
    # Plan changes
    terraform plan \
        -var-file="environments/$environment.tfvars" \
        -out="$environment.tfplan"
    
    cd - > /dev/null
}

# Apply infrastructure changes
apply_infrastructure() {
    local environment=${1:-"production"}
    local auto_approve=${2:-false}
    
    log_iac "Applying infrastructure changes for $environment"
    
    cd "$TERRAFORM_DIR"
    
    # Apply changes
    if [ "$auto_approve" = "true" ]; then
        terraform apply -auto-approve "$environment.tfplan"
    else
        terraform apply "$environment.tfplan"
    fi
    
    # Clean up plan file
    rm -f "$environment.tfplan"
    
    cd - > /dev/null
}

# Destroy infrastructure
destroy_infrastructure() {
    local environment=${1:-""}
    
    if [ -z "$environment" ]; then
        echo "Environment required for destroy operation"
        exit 1
    fi
    
    if [ "$environment" = "production" ]; then
        echo "Production environment destruction requires manual confirmation"
        read -p "Type 'destroy-production' to confirm: " confirmation
        if [ "$confirmation" != "destroy-production" ]; then
            echo "Destruction cancelled"
            exit 1
        fi
    fi
    
    log_iac "Destroying infrastructure for $environment"
    
    cd "$TERRAFORM_DIR"
    
    terraform workspace select "$environment"
    terraform destroy -var-file="environments/$environment.tfvars"
    
    cd - > /dev/null
}

# Generate Kubernetes manifests with Kustomize
generate_manifests() {
    local environment=${1:-"production"}
    local output_dir=${2:-"./generated-manifests"}
    
    log_iac "Generating Kubernetes manifests for $environment"
    
    mkdir -p "$output_dir"
    
    # Generate manifests for each component
    local components=("base" "core" "interfaces" "observability" "security")
    
    for component in "${components[@]}"; do
        log_iac "Generating manifests for $component"
        
        kustomize build "$KUSTOMIZE_DIR/overlays/$environment/$component" \
            > "$output_dir/$component-$environment.yaml"
    done
    
    # Generate complete manifest
    kustomize build "$KUSTOMIZE_DIR/overlays/$environment" \
        > "$output_dir/complete-$environment.yaml"
    
    log_iac "Manifests generated in $output_dir"
}

# Deploy with Ansible
deploy_with_ansible() {
    local environment=${1:-"production"}
    local playbook=${2:-"site.yml"}
    
    log_iac "Deploying with Ansible for $environment"
    
    cd "$ANSIBLE_DIR"
    
    # Run Ansible playbook
    ansible-playbook \
        -i "inventories/$environment/hosts.yml" \
        -e "environment=$environment" \
        "$playbook"
    
    cd - > /dev/null
}

# Validate infrastructure
validate_infrastructure() {
    local environment=${1:-"production"}
    
    log_iac "Validating infrastructure for $environment"
    
    # Terraform validation
    cd "$TERRAFORM_DIR"
    terraform workspace select "$environment"
    terraform validate
    terraform plan -detailed-exitcode -var-file="environments/$environment.tfvars" > /dev/null
    local tf_exit_code=$?
    cd - > /dev/null
    
    if [ $tf_exit_code -eq 0 ]; then
        log_iac "✓ Terraform validation passed - no changes needed"
    elif [ $tf_exit_code -eq 2 ]; then
        log_iac "⚠ Terraform validation passed - changes detected"
    else
        log_iac "✗ Terraform validation failed"
        return 1
    fi
    
    # Kubernetes validation
    if kubectl cluster-info >/dev/null 2>&1; then
        log_iac "✓ Kubernetes cluster accessible"
        
        # Check namespace
        if kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
            log_iac "✓ Namespace $NAMESPACE exists"
        else
            log_iac "⚠ Namespace $NAMESPACE does not exist"
        fi
        
        # Check critical resources
        local resources=("deployment" "service" "configmap" "secret")
        for resource in "${resources[@]}"; do
            local count=$(kubectl get "$resource" -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
            log_iac "✓ Found $count $resource resources"
        done
    else
        log_iac "✗ Kubernetes cluster not accessible"
        return 1
    fi
    
    # Ansible validation
    if [ -d "$ANSIBLE_DIR" ]; then
        cd "$ANSIBLE_DIR"
        ansible-playbook --syntax-check "site.yml" >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            log_iac "✓ Ansible playbook syntax valid"
        else
            log_iac "✗ Ansible playbook syntax invalid"
            return 1
        fi
        cd - > /dev/null
    fi
    
    log_iac "Infrastructure validation completed"
}

# Drift detection
detect_drift() {
    local environment=${1:-"production"}
    
    log_iac "Detecting infrastructure drift for $environment"
    
    cd "$TERRAFORM_DIR"
    
    terraform workspace select "$environment"
    
    # Run plan to detect drift
    terraform plan \
        -var-file="environments/$environment.tfvars" \
        -detailed-exitcode \
        -out="drift-check.tfplan"
    
    local exit_code=$?
    
    case $exit_code in
        0)
            log_iac "✓ No drift detected - infrastructure matches configuration"
            ;;
        1)
            log_iac "✗ Error occurred during drift detection"
            return 1
            ;;
        2)
            log_iac "⚠ Drift detected - infrastructure differs from configuration"
            
            # Show drift details
            terraform show drift-check.tfplan
            
            read -p "Apply changes to fix drift? (y/N): " fix_drift
            if [ "$fix_drift" = "y" ]; then
                terraform apply drift-check.tfplan
                log_iac "✓ Drift corrected"
            fi
            ;;
    esac
    
    rm -f drift-check.tfplan
    cd - > /dev/null
}

# Resource management
manage_resources() {
    local action=${1:-""}
    local resource_type=${2:-""}
    local environment=${3:-"production"}
    
    case "$action" in
        "scale")
            log_iac "Scaling $resource_type in $environment"
            
            case "$resource_type" in
                "cluster")
                    # Scale Kubernetes cluster
                    cd "$TERRAFORM_DIR"
                    terraform workspace select "$environment"
                    
                    read -p "Enter new node count: " node_count
                    terraform apply -var="node_count=$node_count" -auto-approve
                    cd - > /dev/null
                    ;;
                "pods")
                    # Scale pod replicas
                    read -p "Enter deployment name: " deployment
                    read -p "Enter replica count: " replicas
                    
                    kubectl scale deployment "$deployment" -n "$NAMESPACE" --replicas="$replicas"
                    ;;
            esac
            ;;
            
        "backup")
            log_iac "Creating infrastructure backup for $environment"
            
            # Backup Terraform state
            cd "$TERRAFORM_DIR"
            terraform workspace select "$environment"
            terraform state pull > "backups/terraform-state-$environment-$(date +%Y%m%d-%H%M%S).json"
            cd - > /dev/null
            
            # Backup Kubernetes resources
            kubectl get all -n "$NAMESPACE" -o yaml > "backups/k8s-resources-$environment-$(date +%Y%m%d-%H%M%S).yaml"
            ;;
            
        "restore")
            log_iac "Restoring infrastructure for $environment"
            
            local backup_file=${resource_type:-""}
            if [ -z "$backup_file" ]; then
                echo "Backup file required for restore"
                exit 1
            fi
            
            if [[ "$backup_file" == *.json ]]; then
                # Restore Terraform state
                cd "$TERRAFORM_DIR"
                terraform workspace select "$environment"
                terraform state push "$backup_file"
                cd - > /dev/null
            elif [[ "$backup_file" == *.yaml ]]; then
                # Restore Kubernetes resources
                kubectl apply -f "$backup_file" -n "$NAMESPACE"
            fi
            ;;
    esac
}

# Cost optimization
optimize_costs() {
    local environment=${1:-"production"}
    
    log_iac "Analyzing costs for $environment"
    
    # Analyze Terraform resources
    cd "$TERRAFORM_DIR"
    terraform workspace select "$environment"
    
    # Generate cost estimate (requires Infracost)
    if command -v infracost >/dev/null 2>&1; then
        infracost breakdown --path . --terraform-var-file="environments/$environment.tfvars"
    else
        log_iac "Infracost not available - install for cost analysis"
    fi
    
    cd - > /dev/null
    
    # Analyze Kubernetes resource usage
    log_iac "Kubernetes resource utilization:"
    kubectl top nodes
    kubectl top pods -n "$NAMESPACE" --sort-by=cpu
    
    # Identify unused resources
    log_iac "Checking for unused resources..."
    
    # Find pods with low CPU usage
    kubectl top pods -n "$NAMESPACE" --no-headers | \
        awk '$2 ~ /[0-9]+m/ && $2+0 < 10 {print "Low CPU usage: " $1 " (" $2 ")"}' || true
    
    # Find unused ConfigMaps and Secrets
    local unused_cms=$(comm -23 \
        <(kubectl get configmaps -n "$NAMESPACE" -o name | sort) \
        <(kubectl get pods -n "$NAMESPACE" -o yaml | grep -o 'configMapRef:\|configMap:' | sort -u) 2>/dev/null || echo "")
    
    if [ -n "$unused_cms" ]; then
        log_iac "Potentially unused ConfigMaps: $unused_cms"
    fi
}

# Generate infrastructure documentation
generate_docs() {
    local environment=${1:-"production"}
    local output_dir=${2:-"./docs/infrastructure"}
    
    log_iac "Generating infrastructure documentation for $environment"
    
    mkdir -p "$output_dir"
    
    # Generate Terraform documentation
    if command -v terraform-docs >/dev/null 2>&1; then
        cd "$TERRAFORM_DIR"
        terraform-docs markdown table . > "$output_dir/terraform-$environment.md"
        cd - > /dev/null
    fi
    
    # Generate Kubernetes resource documentation
    cat > "$output_dir/kubernetes-$environment.md" << EOF
# Kubernetes Resources - $environment

## Deployments
$(kubectl get deployments -n "$NAMESPACE" -o wide)

## Services
$(kubectl get services -n "$NAMESPACE" -o wide)

## ConfigMaps
$(kubectl get configmaps -n "$NAMESPACE")

## Secrets
$(kubectl get secrets -n "$NAMESPACE")

## Persistent Volumes
$(kubectl get pvc -n "$NAMESPACE")

## Network Policies
$(kubectl get networkpolicies -n "$NAMESPACE" 2>/dev/null || echo "No network policies found")

## Resource Usage
$(kubectl top pods -n "$NAMESPACE" 2>/dev/null || echo "Metrics not available")
EOF
    
    # Generate architecture diagram (requires diagrams library)
    if command -v python3 >/dev/null 2>&1; then
        cat > "$output_dir/generate_diagram.py" << 'EOF'
#!/usr/bin/env python3
from diagrams import Diagram, Cluster, Edge
from diagrams.k8s.compute import Deployment, Pod
from diagrams.k8s.network import Service, Ingress
from diagrams.k8s.storage import PersistentVolume
from diagrams.onprem.database import Redis
from diagrams.onprem.monitoring import Prometheus, Grafana

with Diagram("O-RAN RIC Architecture", show=False, direction="TB"):
    ingress = Ingress("Ingress")
    
    with Cluster("Core Components"):
        e2term = Deployment("E2 Termination")
        e2mgr = Deployment("E2 Manager")
        submgr = Deployment("Subscription Manager")
        
    with Cluster("Interface Components"):
        a1med = Deployment("A1 Mediator")
        o1med = Deployment("O1 Mediator")
        
    with Cluster("Data Layer"):
        redis = Redis("Database")
        
    with Cluster("Observability"):
        prometheus = Prometheus("Prometheus")
        grafana = Grafana("Grafana")
    
    ingress >> [a1med, o1med]
    e2term >> e2mgr >> submgr
    [e2mgr, submgr, a1med, o1med] >> redis
    [e2term, e2mgr, submgr, a1med, o1med] >> prometheus
    prometheus >> grafana
EOF
        
        cd "$output_dir"
        python3 generate_diagram.py 2>/dev/null || log_iac "Could not generate architecture diagram"
        cd - > /dev/null
    fi
    
    log_iac "Documentation generated in $output_dir"
}

# Main function
main() {
    local action=${1:-""}
    
    case "$action" in
        "init")
            init_terraform
            ;;
        "plan")
            local environment=${2:-"production"}
            plan_infrastructure "$environment"
            ;;
        "apply")
            local environment=${2:-"production"}
            local auto_approve=${3:-false}
            apply_infrastructure "$environment" "$auto_approve"
            ;;
        "destroy")
            local environment=${2:-""}
            destroy_infrastructure "$environment"
            ;;
        "validate")
            local environment=${2:-"production"}
            validate_infrastructure "$environment"
            ;;
        "drift")
            local environment=${2:-"production"}
            detect_drift "$environment"
            ;;
        "generate")
            local environment=${2:-"production"}
            generate_manifests "$environment"
            ;;
        "deploy")
            local environment=${2:-"production"}
            deploy_with_ansible "$environment"
            ;;
        "manage")
            local sub_action=${2:-""}
            local resource_type=${3:-""}
            local environment=${4:-"production"}
            manage_resources "$sub_action" "$resource_type" "$environment"
            ;;
        "optimize")
            local environment=${2:-"production"}
            optimize_costs "$environment"
            ;;
        "docs")
            local environment=${2:-"production"}
            generate_docs "$environment"
            ;;
        *)
            echo "Usage: $0 {init|plan|apply|destroy|validate|drift|generate|deploy|manage|optimize|docs}"
            echo
            echo "Commands:"
            echo "  init                    - Initialize Terraform backend"
            echo "  plan <env>             - Plan infrastructure changes"
            echo "  apply <env> [auto]     - Apply infrastructure changes"
            echo "  destroy <env>          - Destroy infrastructure"
            echo "  validate <env>         - Validate infrastructure"
            echo "  drift <env>            - Detect configuration drift"
            echo "  generate <env>         - Generate Kubernetes manifests"
            echo "  deploy <env>           - Deploy with Ansible"
            echo "  manage <action> <type> - Manage resources (scale/backup/restore)"
            echo "  optimize <env>         - Analyze and optimize costs"
            echo "  docs <env>             - Generate documentation"
            echo
            echo "Environments: development, staging, production"
            exit 1
            ;;
    esac
}

# Create required directories
mkdir -p "$TERRAFORM_DIR/environments" "$ANSIBLE_DIR/inventories" "$KUSTOMIZE_DIR/overlays" "backups"

# Run main function
main "$@"