#!/bin/bash
# automated-updates.sh - Automated updates and patch management

set -euo pipefail

# Configuration
NAMESPACE="oran-ric"
UPDATE_LOG="/var/log/oran-ric-updates.log"
BACKUP_DIR="/var/backups/oran-ric"
HELM_REPO_URL="https://charts.o-ran-sc.org"
UPDATE_WINDOW_START="02:00"
UPDATE_WINDOW_END="04:00"

# Logging function
log_update() {
    echo "$(date -Iseconds) [UPDATE] $1" | tee -a "$UPDATE_LOG"
}

# Check if we're in maintenance window
in_maintenance_window() {
    local current_time=$(date +%H:%M)
    local start_time="$UPDATE_WINDOW_START"
    local end_time="$UPDATE_WINDOW_END"
    
    if [[ "$current_time" > "$start_time" && "$current_time" < "$end_time" ]]; then
        return 0
    else
        return 1
    fi
}

# Create backup before updates
create_backup() {
    local backup_name="backup-$(date +%Y%m%d-%H%M%S)"
    local backup_path="$BACKUP_DIR/$backup_name"
    
    log_update "Creating backup: $backup_name"
    
    mkdir -p "$backup_path"
    
    # Backup Kubernetes resources
    kubectl get all -n "$NAMESPACE" -o yaml > "$backup_path/resources.yaml"
    kubectl get configmaps -n "$NAMESPACE" -o yaml > "$backup_path/configmaps.yaml"
    kubectl get secrets -n "$NAMESPACE" -o yaml > "$backup_path/secrets.yaml"
    kubectl get pvc -n "$NAMESPACE" -o yaml > "$backup_path/pvcs.yaml" 2>/dev/null || true
    
    # Backup Helm releases
    helm list -n "$NAMESPACE" -o yaml > "$backup_path/helm-releases.yaml"
    
    # Backup database if accessible
    if kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli ping >/dev/null 2>&1; then
        kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli bgsave
        sleep 10
        kubectl cp "$NAMESPACE/$(kubectl get pods -n "$NAMESPACE" -l app=dbaas -o jsonpath='{.items[0].metadata.name}'):/data/dump.rdb" "$backup_path/redis-dump.rdb" 2>/dev/null || true
    fi
    
    # Compress backup
    tar -czf "$backup_path.tar.gz" -C "$BACKUP_DIR" "$backup_name"
    rm -rf "$backup_path"
    
    log_update "Backup created: $backup_path.tar.gz"
    echo "$backup_path.tar.gz"
}

# Check for available updates
check_for_updates() {
    log_update "Checking for available updates"
    
    # Update Helm repositories
    helm repo update
    
    # Check for chart updates
    local updates_available=()
    
    helm list -n "$NAMESPACE" -o json | jq -r '.[] | "\(.name) \(.chart)"' | \
        while read release_name chart_version; do
            local chart_name=$(echo "$chart_version" | cut -d- -f1)
            local current_version=$(echo "$chart_version" | cut -d- -f2)
            
            # Get latest version from repository
            local latest_version=$(helm search repo "$chart_name" -o json | jq -r '.[0].version' 2>/dev/null || echo "$current_version")
            
            if [ "$current_version" != "$latest_version" ]; then
                log_update "Update available: $release_name ($current_version -> $latest_version)"
                updates_available+=("$release_name:$current_version:$latest_version")
            fi
        done
    
    # Check for container image updates
    kubectl get deployments -n "$NAMESPACE" -o json | \
        jq -r '.items[] | "\(.metadata.name) \(.spec.template.spec.containers[0].image)"' | \
        while read deployment_name image; do
            local image_name=$(echo "$image" | cut -d: -f1)
            local current_tag=$(echo "$image" | cut -d: -f2)
            
            # Skip if using latest tag
            if [ "$current_tag" = "latest" ]; then
                continue
            fi
            
            # Check for newer tags (simplified - would need registry API in practice)
            log_update "Checking image updates for $deployment_name: $image"
        done
    
    if [ ${#updates_available[@]} -eq 0 ]; then
        log_update "No updates available"
        return 1
    else
        log_update "${#updates_available[@]} updates available"
        return 0
    fi
}

# Update Helm charts
update_helm_charts() {
    local release_name=$1
    local target_version=${2:-""}
    
    log_update "Updating Helm release: $release_name"
    
    # Get current values
    local values_file="/tmp/${release_name}-values.yaml"
    helm get values "$release_name" -n "$NAMESPACE" > "$values_file"
    
    # Perform update
    if [ -n "$target_version" ]; then
        helm upgrade "$release_name" "$release_name" \
            --namespace "$NAMESPACE" \
            --version "$target_version" \
            --values "$values_file" \
            --wait \
            --timeout=600s
    else
        helm upgrade "$release_name" "$release_name" \
            --namespace "$NAMESPACE" \
            --values "$values_file" \
            --wait \
            --timeout=600s
    fi
    
    if [ $? -eq 0 ]; then
        log_update "Successfully updated $release_name"
        rm -f "$values_file"
        return 0
    else
        log_update "Failed to update $release_name"
        rm -f "$values_file"
        return 1
    fi
}

# Update container images
update_container_images() {
    local deployment_name=$1
    local new_image=$2
    
    log_update "Updating container image for $deployment_name to $new_image"
    
    # Update deployment
    kubectl set image deployment/"$deployment_name" -n "$NAMESPACE" \
        "$deployment_name=$new_image"
    
    # Wait for rollout
    if kubectl rollout status deployment/"$deployment_name" -n "$NAMESPACE" --timeout=600s; then
        log_update "Successfully updated image for $deployment_name"
        return 0
    else
        log_update "Failed to update image for $deployment_name"
        return 1
    fi
}

# Rollback updates if needed
rollback_update() {
    local release_name=$1
    local backup_file=$2
    
    log_update "Rolling back update for $release_name"
    
    # Rollback Helm release
    helm rollback "$release_name" -n "$NAMESPACE"
    
    # If Helm rollback fails, restore from backup
    if [ $? -ne 0 ] && [ -n "$backup_file" ]; then
        log_update "Helm rollback failed, restoring from backup"
        restore_from_backup "$backup_file"
    fi
}

# Restore from backup
restore_from_backup() {
    local backup_file=$1
    
    log_update "Restoring from backup: $backup_file"
    
    if [ ! -f "$backup_file" ]; then
        log_update "Backup file not found: $backup_file"
        return 1
    fi
    
    # Extract backup
    local restore_dir="/tmp/restore-$(date +%s)"
    mkdir -p "$restore_dir"
    tar -xzf "$backup_file" -C "$restore_dir"
    
    # Find the backup directory
    local backup_dir=$(find "$restore_dir" -type d -name "backup-*" | head -1)
    
    if [ -z "$backup_dir" ]; then
        log_update "Invalid backup file structure"
        rm -rf "$restore_dir"
        return 1
    fi
    
    # Restore resources (be careful with this in production)
    log_update "WARNING: Restoring resources - this may cause service disruption"
    
    # Restore ConfigMaps and Secrets first
    kubectl apply -f "$backup_dir/configmaps.yaml" 2>/dev/null || true
    kubectl apply -f "$backup_dir/secrets.yaml" 2>/dev/null || true
    
    # Restore other resources
    kubectl apply -f "$backup_dir/resources.yaml" 2>/dev/null || true
    
    # Restore database if backup exists
    if [ -f "$backup_dir/redis-dump.rdb" ]; then
        kubectl cp "$backup_dir/redis-dump.rdb" \
            "$NAMESPACE/$(kubectl get pods -n "$NAMESPACE" -l app=dbaas -o jsonpath='{.items[0].metadata.name}'):/data/dump.rdb"
        kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli debug restart
    fi
    
    # Cleanup
    rm -rf "$restore_dir"
    
    log_update "Restore completed"
}

# Validate updates
validate_updates() {
    log_update "Validating updates"
    
    # Check pod status
    local unhealthy_pods=$(kubectl get pods -n "$NAMESPACE" --no-headers | grep -v Running | wc -l)
    
    if [ "$unhealthy_pods" -gt 0 ]; then
        log_update "Validation failed: $unhealthy_pods pods are not running"
        return 1
    fi
    
    # Check service endpoints
    local services=("e2mgr" "submgr" "a1mediator" "o1mediator")
    
    for service in "${services[@]}"; do
        if ! kubectl exec -n "$NAMESPACE" deployment/"$service" -- \
             curl -sf localhost:8080/health >/dev/null 2>&1; then
            log_update "Validation failed: $service health check failed"
            return 1
        fi
    done
    
    # Check database connectivity
    if ! kubectl exec -n "$NAMESPACE" deployment/dbaas -- redis-cli ping >/dev/null 2>&1; then
        log_update "Validation failed: database connectivity check failed"
        return 1
    fi
    
    log_update "Validation passed"
    return 0
}

# Security updates
apply_security_updates() {
    log_update "Applying security updates"
    
    # Update base images with security patches
    local deployments=$(kubectl get deployments -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')
    
    for deployment in $deployments; do
        local current_image=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
        local image_name=$(echo "$current_image" | cut -d: -f1)
        local current_tag=$(echo "$current_image" | cut -d: -f2)
        
        # Check for security updates (simplified - would need vulnerability scanner)
        log_update "Checking security updates for $deployment: $current_image"
        
        # Example: Update to security-patched version
        # This would typically involve checking a vulnerability database
        # and updating to the latest patched version
    done
    
    # Update system packages in containers (if possible)
    kubectl get pods -n "$NAMESPACE" -o name | \
        while read pod; do
            # Try to update packages (this may not work in all containers)
            kubectl exec "$pod" -n "$NAMESPACE" -- sh -c '
                if command -v apt-get >/dev/null; then
                    apt-get update && apt-get upgrade -y
                elif command -v yum >/dev/null; then
                    yum update -y
                elif command -v apk >/dev/null; then
                    apk update && apk upgrade
                fi
            ' 2>/dev/null || true
        done
}

# Cleanup old backups
cleanup_old_backups() {
    log_update "Cleaning up old backups"
    
    # Keep backups for 30 days
    find "$BACKUP_DIR" -name "backup-*.tar.gz" -mtime +30 -delete 2>/dev/null || true
    
    # Keep at least 5 most recent backups
    local backup_count=$(find "$BACKUP_DIR" -name "backup-*.tar.gz" | wc -l)
    
    if [ "$backup_count" -gt 5 ]; then
        find "$BACKUP_DIR" -name "backup-*.tar.gz" -printf '%T@ %p\n' | \
            sort -n | \
            head -n $((backup_count - 5)) | \
            cut -d' ' -f2- | \
            xargs rm -f
    fi
    
    log_update "Backup cleanup completed"
}

# Generate update report
generate_update_report() {
    local report_file="/tmp/update-report-$(date +%Y%m%d-%H%M%S).json"
    
    log_update "Generating update report: $report_file"
    
    # Collect update information
    local helm_releases=$(helm list -n "$NAMESPACE" -o json)
    local pod_images=$(kubectl get pods -n "$NAMESPACE" -o json | \
        jq '[.items[] | {name: .metadata.name, image: .spec.containers[0].image}]')
    
    cat > "$report_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "namespace": "$NAMESPACE",
    "helm_releases": $helm_releases,
    "container_images": $pod_images,
    "last_update": "$(tail -n 1 "$UPDATE_LOG" 2>/dev/null || echo "No updates logged")",
    "backup_count": $(find "$BACKUP_DIR" -name "backup-*.tar.gz" 2>/dev/null | wc -l),
    "system_status": {
        "pods_running": $(kubectl get pods -n "$NAMESPACE" --no-headers | grep Running | wc -l),
        "pods_total": $(kubectl get pods -n "$NAMESPACE" --no-headers | wc -l)
    }
}
EOF
    
    echo "$report_file"
}

# Main update function
main() {
    local action=${1:-"check"}
    
    case "$action" in
        "check")
            check_for_updates
            ;;
            
        "update")
            local component=${2:-"all"}
            
            if ! in_maintenance_window; then
                log_update "Not in maintenance window ($UPDATE_WINDOW_START - $UPDATE_WINDOW_END)"
                exit 1
            fi
            
            log_update "Starting update process for $component"
            
            # Create backup
            local backup_file=$(create_backup)
            
            # Check for updates
            if check_for_updates; then
                case "$component" in
                    "all")
                        # Update all components
                        helm list -n "$NAMESPACE" --short | \
                            while read release; do
                                if update_helm_charts "$release"; then
                                    log_update "Updated $release successfully"
                                else
                                    log_update "Failed to update $release - rolling back"
                                    rollback_update "$release" "$backup_file"
                                fi
                            done
                        ;;
                    *)
                        # Update specific component
                        if update_helm_charts "$component"; then
                            log_update "Updated $component successfully"
                        else
                            log_update "Failed to update $component - rolling back"
                            rollback_update "$component" "$backup_file"
                        fi
                        ;;
                esac
                
                # Validate updates
                if validate_updates; then
                    log_update "Update validation passed"
                else
                    log_update "Update validation failed - consider rollback"
                fi
            else
                log_update "No updates available"
            fi
            
            cleanup_old_backups
            ;;
            
        "security")
            log_update "Applying security updates"
            
            # Create backup
            local backup_file=$(create_backup)
            
            # Apply security updates
            apply_security_updates
            
            # Validate
            if validate_updates; then
                log_update "Security updates applied successfully"
            else
                log_update "Security update validation failed"
            fi
            ;;
            
        "rollback")
            local release_name=${2:-""}
            local backup_file=${3:-""}
            
            if [ -z "$release_name" ]; then
                echo "Usage: $0 rollback <release_name> [backup_file]"
                exit 1
            fi
            
            rollback_update "$release_name" "$backup_file"
            ;;
            
        "backup")
            backup_file=$(create_backup)
            echo "Backup created: $backup_file"
            ;;
            
        "restore")
            local backup_file=${2:-""}
            
            if [ -z "$backup_file" ]; then
                echo "Usage: $0 restore <backup_file>"
                exit 1
            fi
            
            restore_from_backup "$backup_file"
            ;;
            
        "report")
            report_file=$(generate_update_report)
            echo "Update report generated: $report_file"
            cat "$report_file"
            ;;
            
        "cleanup")
            cleanup_old_backups
            ;;
            
        *)
            echo "Usage: $0 {check|update|security|rollback|backup|restore|report|cleanup}"
            echo "  check     - Check for available updates"
            echo "  update    - Apply updates (all or specific component)"
            echo "  security  - Apply security updates"
            echo "  rollback  - Rollback updates"
            echo "  backup    - Create backup"
            echo "  restore   - Restore from backup"
            echo "  report    - Generate update report"
            echo "  cleanup   - Cleanup old backups"
            exit 1
            ;;
    esac
}

# Ensure backup directory exists
mkdir -p "$BACKUP_DIR"

# Run main function
main "$@"