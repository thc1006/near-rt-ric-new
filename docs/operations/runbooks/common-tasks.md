# Common Operational Tasks

## Platform Health Monitoring

### Check Overall Platform Status
```bash
# Check all pods status
kubectl get pods -n oran-ric

# Check service endpoints
kubectl get svc -n oran-ric

# Check ingress status
kubectl get ingress -n oran-ric
```

### Verify Component Health
```bash
# E2 Manager health
curl -k https://e2mgr.ric.local/health

# Subscription Manager health
curl -k https://submgr.ric.local/health

# A1 Mediator health
curl -k https://a1mediator.ric.local/a1-p/healthcheck

# Dashboard API health
curl -k https://dashboard.ric.local/health
```

## E2 Node Management

### List Connected E2 Nodes
```bash
# Via kubectl
kubectl exec -n oran-ric deployment/e2mgr -- curl localhost:3800/v1/nodeb/states

# Via API
curl -k https://dashboard.ric.local/api/e2/nodes
```

### Check E2 Node Status
```bash
# Get specific node details
curl -k https://dashboard.ric.local/api/e2/nodes/{nodeId}

# Check node subscriptions
curl -k https://dashboard.ric.local/api/e2/subscriptions?nodeId={nodeId}
```

### Restart E2 Connection
```bash
# Reset E2 node connection
kubectl exec -n oran-ric deployment/e2mgr -- \
  curl -X POST localhost:3800/v1/nodeb/{nodeId}/reset
```

## Subscription Management

### List Active Subscriptions
```bash
# All subscriptions
curl -k https://dashboard.ric.local/api/subscriptions

# Subscriptions by xApp
curl -k https://dashboard.ric.local/api/subscriptions?xappId={xappId}
```

### Create Test Subscription
```bash
curl -k -X POST https://dashboard.ric.local/api/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "e2NodeId": "test-node-1",
    "ranFunctionId": 1,
    "eventTrigger": {
      "type": "periodic",
      "period": "1000ms"
    },
    "actions": [{
      "id": 1,
      "type": "report"
    }]
  }'
```

### Delete Subscription
```bash
curl -k -X DELETE https://dashboard.ric.local/api/subscriptions/{subscriptionId}
```

## Policy Management

### List Policy Types
```bash
curl -k https://a1mediator.ric.local/a1-p/policytypes
```

### Create Policy Type
```bash
curl -k -X POST https://a1mediator.ric.local/a1-p/policytypes/1001 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "QoS Policy",
    "description": "Quality of Service policy",
    "policy_type_schema": {
      "type": "object",
      "properties": {
        "qci": {"type": "integer"},
        "priority": {"type": "integer"}
      }
    }
  }'
```

### Deploy Policy Instance
```bash
curl -k -X PUT https://a1mediator.ric.local/a1-p/policytypes/1001/policies/policy-1 \
  -H "Content-Type: application/json" \
  -d '{
    "qci": 7,
    "priority": 10
  }'
```

## xApp Management

### List Deployed xApps
```bash
kubectl get pods -n oran-ric -l app.kubernetes.io/component=xapp
```

### Deploy xApp
```bash
helm install hello-world-xapp helm/xapp-hello-world \
  --namespace oran-ric \
  --set image.tag=latest
```

### Scale xApp
```bash
kubectl scale deployment hello-world-xapp -n oran-ric --replicas=3
```

### View xApp Logs
```bash
kubectl logs -n oran-ric deployment/hello-world-xapp -f
```

## Configuration Management

### Update Component Configuration
```bash
# Update E2 Manager config
kubectl patch configmap e2mgr-config -n oran-ric --patch '{"data":{"config.yaml":"new-config"}}'

# Restart component to pick up changes
kubectl rollout restart deployment/e2mgr -n oran-ric
```

### Backup Configuration
```bash
# Export all configmaps
kubectl get configmaps -n oran-ric -o yaml > ric-configs-backup.yaml

# Export secrets (be careful with sensitive data)
kubectl get secrets -n oran-ric -o yaml > ric-secrets-backup.yaml
```

## Log Management

### View Component Logs
```bash
# E2 Manager logs
kubectl logs -n oran-ric deployment/e2mgr -f

# Subscription Manager logs
kubectl logs -n oran-ric deployment/submgr -f

# A1 Mediator logs
kubectl logs -n oran-ric deployment/a1mediator -f
```

### Search Logs with Loki
```bash
# Query logs via LogQL
curl -G -s "http://loki.ric.local:3100/loki/api/v1/query" \
  --data-urlencode 'query={namespace="oran-ric"} |= "ERROR"'
```

## Performance Monitoring

### Check Resource Usage
```bash
# Pod resource usage
kubectl top pods -n oran-ric

# Node resource usage
kubectl top nodes
```

### Query Metrics
```bash
# CPU usage
curl 'http://prometheus.ric.local:9090/api/v1/query?query=rate(container_cpu_usage_seconds_total[5m])'

# Memory usage
curl 'http://prometheus.ric.local:9090/api/v1/query?query=container_memory_usage_bytes'
```

## Database Operations

### Check Redis/SDL Status
```bash
# Connect to Redis
kubectl exec -n oran-ric deployment/dbaas -- redis-cli ping

# Check database size
kubectl exec -n oran-ric deployment/dbaas -- redis-cli info memory
```

### Backup Database
```bash
# Create Redis backup
kubectl exec -n oran-ric deployment/dbaas -- redis-cli bgsave

# Copy backup file
kubectl cp oran-ric/dbaas-pod:/data/dump.rdb ./redis-backup-$(date +%Y%m%d).rdb
```

## Certificate Management

### Check Certificate Expiry
```bash
# Check TLS certificates
kubectl get secrets -n oran-ric -o json | \
  jq -r '.items[] | select(.type=="kubernetes.io/tls") | .metadata.name' | \
  while read cert; do
    echo "Certificate: $cert"
    kubectl get secret $cert -n oran-ric -o json | \
      jq -r '.data."tls.crt"' | base64 -d | \
      openssl x509 -noout -dates
  done
```

### Renew Certificates
```bash
# Trigger cert-manager renewal
kubectl annotate secret tls-secret -n oran-ric cert-manager.io/issue-temporary-certificate=""
```

## Troubleshooting Quick Commands

### Network Connectivity
```bash
# Test internal connectivity
kubectl run test-pod --image=busybox -it --rm -- /bin/sh

# DNS resolution test
nslookup e2mgr.oran-ric.svc.cluster.local

# Port connectivity test
telnet e2mgr.oran-ric.svc.cluster.local 3800
```

### Resource Constraints
```bash
# Check for resource limits
kubectl describe pods -n oran-ric | grep -A 5 "Limits:"

# Check for pending pods
kubectl get pods -n oran-ric --field-selector=status.phase=Pending
```

### Event Monitoring
```bash
# Recent events
kubectl get events -n oran-ric --sort-by='.lastTimestamp'

# Watch for new events
kubectl get events -n oran-ric --watch
```