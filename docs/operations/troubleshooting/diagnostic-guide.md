# Diagnostic and Troubleshooting Guide

## Platform Health Diagnostics

### Automated Health Check Script
```bash
#!/bin/bash
# Platform health diagnostic script

echo "=== O-RAN RIC Platform Health Check ==="
echo "Timestamp: $(date)"
echo

# Check Kubernetes cluster
echo "1. Kubernetes Cluster Status:"
kubectl cluster-info
echo

# Check namespace
echo "2. RIC Namespace Status:"
kubectl get ns oran-ric
echo

# Check all pods
echo "3. Pod Status:"
kubectl get pods -n oran-ric -o wide
echo

# Check services
echo "4. Service Status:"
kubectl get svc -n oran-ric
echo

# Check persistent volumes
echo "5. Storage Status:"
kubectl get pv,pvc -n oran-ric
echo

# Check component health endpoints
echo "6. Component Health Checks:"
components=("e2mgr:3800/health" "submgr:8080/health" "a1mediator:8080/a1-p/healthcheck")
for component in "${components[@]}"; do
    name=$(echo $component | cut -d: -f1)
    endpoint=$(echo $component | cut -d: -f2-)
    echo -n "  $name: "
    if kubectl exec -n oran-ric deployment/$name -- curl -s -f localhost:$endpoint > /dev/null 2>&1; then
        echo "✓ Healthy"
    else
        echo "✗ Unhealthy"
    fi
done
echo

# Check resource usage
echo "7. Resource Usage:"
kubectl top pods -n oran-ric 2>/dev/null || echo "  Metrics server not available"
echo

echo "=== Health Check Complete ==="
```

### Component-Specific Diagnostics

#### E2 Manager Diagnostics
```bash
# Check E2 Manager logs for errors
kubectl logs -n oran-ric deployment/e2mgr --tail=100 | grep -i error

# Check E2 node connections
kubectl exec -n oran-ric deployment/e2mgr -- \
  curl -s localhost:3800/v1/nodeb/states | jq '.'

# Verify E2 Manager configuration
kubectl get configmap e2mgr-config -n oran-ric -o yaml

# Check E2 Manager metrics
curl -s http://prometheus.ric.local:9090/api/v1/query?query=e2mgr_connected_nodes
```

#### Subscription Manager Diagnostics
```bash
# Check subscription manager logs
kubectl logs -n oran-ric deployment/submgr --tail=100 | grep -i error

# List active subscriptions
kubectl exec -n oran-ric deployment/submgr -- \
  curl -s localhost:8080/ric/v1/subscriptions

# Check subscription metrics
curl -s http://prometheus.ric.local:9090/api/v1/query?query=submgr_active_subscriptions
```

#### A1 Mediator Diagnostics
```bash
# Check A1 mediator logs
kubectl logs -n oran-ric deployment/a1mediator --tail=100 | grep -i error

# Test A1 API endpoints
kubectl exec -n oran-ric deployment/a1mediator -- \
  curl -s localhost:8080/a1-p/healthcheck

# Check policy types
kubectl exec -n oran-ric deployment/a1mediator -- \
  curl -s localhost:8080/a1-p/policytypes
```

## Common Issues and Solutions

### Issue: Pods Stuck in Pending State

**Symptoms:**
- Pods remain in Pending status
- Events show scheduling failures

**Diagnosis:**
```bash
# Check pod events
kubectl describe pod <pod-name> -n oran-ric

# Check node resources
kubectl describe nodes

# Check resource requests vs available
kubectl top nodes
```

**Solutions:**
1. **Insufficient Resources:**
   ```bash
   # Scale down non-critical workloads
   kubectl scale deployment <deployment> --replicas=1 -n oran-ric
   
   # Add more nodes to cluster
   # (cluster-specific commands)
   ```

2. **Node Selector Issues:**
   ```bash
   # Check node labels
   kubectl get nodes --show-labels
   
   # Update deployment node selector
   kubectl patch deployment <deployment> -n oran-ric -p '{"spec":{"template":{"spec":{"nodeSelector":null}}}}'
   ```

### Issue: E2 Nodes Not Connecting

**Symptoms:**
- E2 nodes show as disconnected
- SCTP connection failures in logs

**Diagnosis:**
```bash
# Check E2T logs for SCTP errors
kubectl logs -n oran-ric deployment/e2term | grep -i sctp

# Check network connectivity
kubectl exec -n oran-ric deployment/e2term -- netstat -ln | grep 36422

# Test port accessibility
kubectl run test-pod --image=busybox -it --rm -- telnet e2term.oran-ric.svc.cluster.local 36422
```

**Solutions:**
1. **Network Configuration:**
   ```bash
   # Check service configuration
   kubectl get svc e2term -n oran-ric -o yaml
   
   # Verify ingress/load balancer
   kubectl get ingress -n oran-ric
   ```

2. **Certificate Issues:**
   ```bash
   # Check TLS certificates
   kubectl get secrets -n oran-ric | grep tls
   
   # Verify certificate validity
   kubectl get secret e2term-tls -n oran-ric -o json | \
     jq -r '.data."tls.crt"' | base64 -d | openssl x509 -noout -dates
   ```

### Issue: High Memory Usage

**Symptoms:**
- Pods being OOMKilled
- High memory usage alerts

**Diagnosis:**
```bash
# Check memory usage
kubectl top pods -n oran-ric --sort-by=memory

# Check memory limits
kubectl describe pods -n oran-ric | grep -A 5 "Limits:"

# Check for memory leaks
kubectl exec -n oran-ric deployment/<pod> -- ps aux --sort=-%mem
```

**Solutions:**
1. **Increase Memory Limits:**
   ```bash
   kubectl patch deployment <deployment> -n oran-ric -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container>","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
   ```

2. **Optimize Application:**
   ```bash
   # Enable memory profiling
   kubectl port-forward -n oran-ric deployment/<deployment> 6060:6060
   go tool pprof http://localhost:6060/debug/pprof/heap
   ```

### Issue: Database Connection Failures

**Symptoms:**
- Components unable to connect to Redis/SDL
- Database connection errors in logs

**Diagnosis:**
```bash
# Check database pod status
kubectl get pods -n oran-ric -l app=dbaas

# Test database connectivity
kubectl exec -n oran-ric deployment/dbaas -- redis-cli ping

# Check database logs
kubectl logs -n oran-ric deployment/dbaas
```

**Solutions:**
1. **Restart Database:**
   ```bash
   kubectl rollout restart deployment/dbaas -n oran-ric
   ```

2. **Check Configuration:**
   ```bash
   # Verify database service
   kubectl get svc dbaas -n oran-ric
   
   # Check connection strings in components
   kubectl get configmaps -n oran-ric -o yaml | grep -i redis
   ```

## Performance Troubleshooting

### High Latency Issues

**Diagnosis:**
```bash
# Check processing latency metrics
curl -s 'http://prometheus.ric.local:9090/api/v1/query?query=histogram_quantile(0.95,rate(http_request_duration_seconds_bucket[5m]))'

# Check queue depths
curl -s 'http://prometheus.ric.local:9090/api/v1/query?query=rmr_queue_depth'
```

**Solutions:**
1. **Scale Components:**
   ```bash
   kubectl scale deployment submgr -n oran-ric --replicas=3
   ```

2. **Optimize Configuration:**
   ```bash
   # Increase worker threads
   kubectl patch configmap submgr-config -n oran-ric --patch '{"data":{"workers":"10"}}'
   ```

### High CPU Usage

**Diagnosis:**
```bash
# Check CPU usage
kubectl top pods -n oran-ric --sort-by=cpu

# Profile CPU usage
kubectl exec -n oran-ric deployment/<pod> -- top -p 1
```

**Solutions:**
1. **Increase CPU Limits:**
   ```bash
   kubectl patch deployment <deployment> -n oran-ric -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container>","resources":{"limits":{"cpu":"2000m"}}}]}}}}'
   ```

2. **Optimize Algorithms:**
   - Review code for inefficient loops
   - Implement caching where appropriate
   - Use connection pooling

## Log Analysis

### Centralized Log Analysis
```bash
# Query logs with LogQL
curl -G -s "http://loki.ric.local:3100/loki/api/v1/query_range" \
  --data-urlencode 'query={namespace="oran-ric"} |= "ERROR"' \
  --data-urlencode 'start=2024-01-01T00:00:00Z' \
  --data-urlencode 'end=2024-01-01T23:59:59Z'

# Common error patterns
curl -G -s "http://loki.ric.local:3100/loki/api/v1/query" \
  --data-urlencode 'query={namespace="oran-ric"} |~ "connection.*failed|timeout|error"'
```

### Log Correlation
```bash
# Find related logs by correlation ID
kubectl logs -n oran-ric --selector=app=e2mgr | grep "correlation-id-12345"

# Trace request flow
kubectl logs -n oran-ric --selector=app=submgr | grep "request-id-67890"
```

## Emergency Procedures

### Platform Emergency Shutdown
```bash
#!/bin/bash
echo "EMERGENCY: Shutting down O-RAN RIC Platform"

# Scale down all deployments
kubectl scale deployment --all -n oran-ric --replicas=0

# Wait for pods to terminate
kubectl wait --for=delete pods --all -n oran-ric --timeout=300s

echo "Platform shutdown complete"
```

### Emergency Recovery
```bash
#!/bin/bash
echo "RECOVERY: Starting O-RAN RIC Platform"

# Start core infrastructure first
kubectl scale deployment dbaas -n oran-ric --replicas=1
kubectl wait --for=condition=available deployment/dbaas -n oran-ric

# Start core platform components
kubectl scale deployment e2term e2mgr submgr -n oran-ric --replicas=1
kubectl wait --for=condition=available deployment/e2term deployment/e2mgr deployment/submgr -n oran-ric

# Start interface components
kubectl scale deployment a1mediator o1mediator -n oran-ric --replicas=1
kubectl wait --for=condition=available deployment/a1mediator deployment/o1mediator -n oran-ric

# Start observability stack
kubectl scale deployment prometheus grafana loki -n oran-ric --replicas=1

echo "Platform recovery complete"
```