#!/bin/bash
# Load O-RAN SC Images for Local Kubernetes Development
# This script pulls official O-RAN SC images and loads them into Kind cluster

set -e

echo "🚀 Loading O-RAN SC Images for Near-RT RIC..."

# Define O-RAN SC images to pull
IMAGES=(
    "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-dashboard:2.1.0"
    "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-e2:6.0.4"
    "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-e2mgr:5.4.2"
    "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-a1:2.5.1"
    "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-rtmgr:4.1.0"
    "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-submgr:4.1.1"
    "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-dbaas:4.1.0"
    "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-appmgr:4.1.0"
)

# Backup images for fallback
BACKUP_IMAGES=(
    "redis:7-alpine"
    "nginx:alpine"
    "node:18-alpine"
)

echo "📥 Pulling O-RAN SC images..."

for image in "${IMAGES[@]}"; do
    echo "Pulling $image..."
    if docker pull "$image" 2>/dev/null; then
        echo "✅ Successfully pulled $image"
    else
        echo "⚠️  Failed to pull $image, will try backup"
    fi
done

echo "📥 Pulling backup images..."

for image in "${BACKUP_IMAGES[@]}"; do
    echo "Pulling $image..."
    if docker pull "$image" 2>/dev/null; then
        echo "✅ Successfully pulled $image"
    else
        echo "❌ Failed to pull $image"
    fi
done

# Function to load images into Kind cluster
load_into_kind() {
    local cluster_name=${1:-desktop}
    
    echo "🔄 Loading images into Kind cluster: $cluster_name..."
    
    # Check if Kind cluster exists
    if ! kind get clusters | grep -q "^${cluster_name}$"; then
        echo "❌ Kind cluster '$cluster_name' not found. Available clusters:"
        kind get clusters
        return 1
    fi
    
    # Load O-RAN SC images
    for image in "${IMAGES[@]}"; do
        echo "Loading $image into Kind..."
        if docker images --format "table {{.Repository}}:{{.Tag}}" | grep -q "${image}"; then
            # Try different approaches for loading images
            if kind load docker-image "$image" --name "$cluster_name" 2>/dev/null; then
                echo "✅ Loaded $image"
            else
                echo "⚠️  Failed to load $image with kind, trying alternative..."
                # Alternative: save and load through docker
                image_file="/tmp/$(basename "$image" | tr '/' '-' | tr ':' '-').tar"
                docker save "$image" -o "$image_file"
                docker exec "${cluster_name}-control-plane" ctr -n k8s.io images import "$image_file" || echo "❌ Failed alternative load for $image"
                rm -f "$image_file"
            fi
        else
            echo "❌ Image $image not found locally"
        fi
    done
    
    # Load backup images
    for image in "${BACKUP_IMAGES[@]}"; do
        echo "Loading backup $image into Kind..."
        if docker images --format "table {{.Repository}}:{{.Tag}}" | grep -q "${image}"; then
            kind load docker-image "$image" --name "$cluster_name" 2>/dev/null || echo "⚠️  Failed to load backup $image"
        fi
    done
}

# Function to configure Kubernetes for registry access
configure_k8s_registry() {
    echo "🔧 Configuring Kubernetes for O-RAN SC registry access..."
    
    # Create namespace if not exists
    kubectl create namespace ricplt --dry-run=client -o yaml | kubectl apply -f -
    
    # Create registry secret (anonymous access)
    kubectl create secret docker-registry oran-sc-registry-secret \
        --docker-server=nexus3.o-ran-sc.org:10002 \
        --docker-username=docker \
        --docker-password=docker \
        --namespace=ricplt \
        --dry-run=client -o yaml | kubectl apply -f -
    
    echo "✅ Registry secret created in ricplt namespace"
    
    # Patch default service account to use the secret
    kubectl patch serviceaccount default -n ricplt \
        -p '{"imagePullSecrets": [{"name": "oran-sc-registry-secret"}]}' || echo "⚠️  Failed to patch default service account"
    
    # Create service account for O-RAN SC platform
    kubectl create serviceaccount oran-sc-platform -n ricplt --dry-run=client -o yaml | kubectl apply -f -
    kubectl patch serviceaccount oran-sc-platform -n ricplt \
        -p '{"imagePullSecrets": [{"name": "oran-sc-registry-secret"}]}' || echo "⚠️  Failed to patch oran-sc-platform service account"
}

# Function to test registry connectivity
test_registry_connectivity() {
    echo "🔍 Testing O-RAN SC registry connectivity..."
    
    if curl -s --connect-timeout 10 "https://nexus3.o-ran-sc.org:10002/v2/" > /dev/null; then
        echo "✅ O-RAN SC registry is accessible"
        
        # Test specific repository access
        if curl -s "https://nexus3.o-ran-sc.org:10002/v2/o-ran-sc/ric-dashboard/tags/list" | grep -q "tags"; then
            echo "✅ Repository access confirmed"
        else
            echo "⚠️  Repository access may be limited"
        fi
    else
        echo "❌ O-RAN SC registry is not accessible"
        echo "   Please check your internet connection and firewall settings"
        return 1
    fi
}

# Function to apply network policies for improved connectivity
apply_network_fixes() {
    echo "🔧 Applying network fixes for container registry access..."
    
    # Create network policy to allow egress to registry
    cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-registry-access
  namespace: ricplt
spec:
  podSelector: {}
  policyTypes:
  - Egress
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 80
    - protocol: TCP
      port: 10002
EOF
    
    echo "✅ Network policy applied"
}

# Main execution
main() {
    echo "🏁 Starting O-RAN SC image loading process..."
    
    # Test registry connectivity first
    test_registry_connectivity || {
        echo "❌ Registry connectivity failed, continuing with available images..."
    }
    
    # Configure Kubernetes registry access
    configure_k8s_registry
    
    # Apply network fixes
    apply_network_fixes
    
    # Load images into Kind if available
    if command -v kind &> /dev/null; then
        load_into_kind
    else
        echo "⚠️  Kind not found, skipping image loading into cluster"
    fi
    
    echo "🎉 O-RAN SC image loading completed!"
    echo ""
    echo "📋 Next steps:"
    echo "   1. Deploy O-RAN SC components: kubectl apply -f deployments/oran-sc-fixed.yaml"
    echo "   2. Check pod status: kubectl get pods -n ricplt"
    echo "   3. Access dashboard: kubectl port-forward -n ricplt svc/ric-dashboard-api 8080:8080"
    echo ""
    echo "🔍 Troubleshooting:"
    echo "   - If images still fail to pull, try: docker system prune -a"
    echo "   - Check pod logs: kubectl logs -n ricplt <pod-name>"
    echo "   - Verify registry secret: kubectl get secret oran-sc-registry-secret -n ricplt -o yaml"
}

# Run main function
main "$@"