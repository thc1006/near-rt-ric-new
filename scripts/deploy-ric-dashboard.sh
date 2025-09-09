#!/bin/bash
set -euo pipefail

# Create namespace for RIC Dashboard
kubectl create namespace ricplt --dry-run=client -o yaml | kubectl apply -f -

# Create Docker registry secret 
# Note: Replace with actual O-RAN SC registry credentials
kubectl create secret docker-registry o-ran-sc-registry-secret \
  --docker-server=nexus3.o-ran-sc.org:10002 \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  --namespace=ricplt

# Deploy RIC Dashboard using Helm
helm upgrade --install ric-dashboard \
  /c/Users/thc1006/Desktop/1/near-rt-ric-new/helm/ric-platform \
  --namespace ricplt \
  --values /c/Users/thc1006/Desktop/1/near-rt-ric-new/helm/ric-platform/values.yaml

# Wait for deployment to be ready
kubectl rollout status deployment/ric-dashboard -n ricplt

# Port forward for local access (optional)
kubectl port-forward svc/ric-dashboard -n ricplt 4200:4200 8080:8080 &

echo "RIC Dashboard deployed successfully!"
echo "Access Frontend: http://localhost:4200"
echo "Access Backend: http://localhost:8080"