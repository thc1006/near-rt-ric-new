# Capacity Planning and Scaling Procedures

## Capacity Planning Overview

### Performance Baselines

#### Component Resource Requirements
| Component | CPU (cores) | Memory (GB) | Storage (GB) | Network (Mbps) |
|-----------|-------------|-------------|--------------|----------------|
| E2 Termination | 2-4 | 4-8 | 10 | 1000 |
| E2 Manager | 1-2 | 2-4 | 5 | 100 |
| Subscription Manager | 2-4 | 4-8 | 5 | 500 |
| A1 Mediator | 1-2 | 2-4 | 5 | 100 |
| O1 Mediator | 1-2 | 2-4 | 5 | 100 |
| Database (Redis) | 2-4 | 8-16 | 50 | 200 |
| xApp Framework | 1-2 | 2-4 | 5 | 100 |

#### Scaling Thresholds
- **CPU Usage**: Scale up when >70% for 5 minutes
- **Memory Usage**: Scale up when >80% for 5 minutes
- **E2 Connections**: Scale E2T when >50 connections per instance
- **Subscriptions**: Scale SubMgr when >500 active subscriptions per instance
- **Request Rate**: Scale when >1000 requests/second per instance

### Capacity Monitoring

#### Key Metrics to Monitor
```bash
# CPU utilization
rate(container_cpu_usage_seconds_total[5m]) * 100

# Memory utilization
(container_memory_usage_bytes / container_spec_memory_limit_bytes) * 100

# E2 node connections
e2mgr_connected_nodes

# Active subscriptions
submgr_active_subscriptions

# Request rate
rate(http_requests_total[5m])

# Queue depth
rmr_queue_depth

# Database connections
redis_connected_clients
```

#### Automated Monitoring Script
```bash
#!/bin/bash
# capacity-monitor.sh - Automated capacity monitoring

NAMESPACE="oran-ric"
PROMETHEUS_URL="http://prometheus.ric.local:9090"
ALERT_THRESHOLD_CPU=70
ALERT_THRESHOLD_MEM=80

echo "=== Capacity Monitoring Report ==="
echo "Timestamp: $(date)"
echo

# Function to query Prometheus
query_prometheus() {
    local query="$1"
    curl -s -G "${PROMETHEUS_URL}/api/v1/query" --data-urlencode "query=${query}" | \
        jq -r '.data.result[] | "\(.metric.pod // .metric.instance): \(.value[1])"'
}

# CPU Usage
echo "1. CPU Usage (%):"
query_prometheus "rate(container_cpu_usage_seconds_total{namespace=\"${NAMESPACE}\"}[5m]) * 100" | \
    while read line; do
        pod=$(echo $line | cut -d: -f1)
        usage=$(echo $line | cut -d: -f2 | cut -d. -f1)
        if [ "$usage" -gt "$ALERT_THRESHOLD_CPU" ]; then
            echo "  ⚠️  $pod: ${usage}% (HIGH)"
        else
            echo "  ✓  $pod: ${usage}%"
        fi
    done
echo

# Memory Usage
echo "2. Memory Usage (%):"
query_prometheus "(container_memory_usage_bytes{namespace=\"${NAMESPACE}\"} / container_spec_memory_limit_bytes{namespace=\"${NAMESPACE}\"}) * 100" | \
    while read line; do
        pod=$(echo $line | cut -d: -f1)
        usage=$(echo $line | cut -d: -f2 | cut -d. -f1)
        if [ "$usage" -gt "$ALERT_THRESHOLD_MEM" ]; then
            echo "  ⚠️  $pod: ${usage}% (HIGH)"
        else
            echo "  ✓  $pod: ${usage}%"
        fi
    done
echo

# E2 Connections
echo "3. E2 Node Connections:"
connections=$(query_prometheus "e2mgr_connected_nodes" | cut -d: -f2)
echo "  Connected nodes: $connections"
if [ "$connections" -gt 50 ]; then
    echo "  ⚠️  High connection count - consider scaling E2T"
fi
echo

# Active Subscriptions
echo "4. Active Subscriptions:"
subscriptions=$(query_prometheus "submgr_active_subscriptions" | cut -d: -f2)
echo "  Active subscriptions: $subscriptions"
if [ "$subscriptions" -gt 500 ]; then
    echo "  ⚠️  High subscription count - consider scaling SubMgr"
fi
echo

echo "=== Monitoring Complete ==="
```

## Horizontal Scaling Procedures

### Automatic Scaling with HPA

#### CPU-based Horizontal Pod Autoscaler
```yaml
# hpa-cpu.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: submgr-hpa
  namespace: oran-ric
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: submgr
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

#### Custom Metrics HPA
```yaml
# hpa-custom.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: e2term-hpa
  namespace: oran-ric
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: e2term
  minReplicas: 1
  maxReplicas: 5
  metrics:
  - type: Pods
    pods:
      metric:
        name: e2_connections_per_pod
      target:
        type: AverageValue
        averageValue: "50"
  - type: Pods
    pods:
      metric:
        name: e2_messages_per_second
      target:
        type: AverageValue
        averageValue: "1000"
```

### Manual Scaling Procedures

#### Scale Up Procedure
```bash
#!/bin/bash
# scale-up.sh - Manual scale up procedure

COMPONENT=$1
REPLICAS=$2
NAMESPACE="oran-ric"

if [ -z "$COMPONENT" ] || [ -z "$REPLICAS" ]; then
    echo "Usage: $0 <component> <replicas>"
    echo "Components: e2term, e2mgr, submgr, a1mediator, o1mediator"
    exit 1
fi

echo "Scaling up $COMPONENT to $REPLICAS replicas..."

# Pre-scaling checks
echo "1. Checking current status..."
kubectl get deployment $COMPONENT -n $NAMESPACE

echo "2. Checking resource availability..."
kubectl describe nodes | grep -A 5 "Allocated resources"

echo "3. Scaling deployment..."
kubectl scale deployment $COMPONENT -n $NAMESPACE --replicas=$REPLICAS

echo "4. Waiting for rollout to complete..."
kubectl rollout status deployment/$COMPONENT -n $NAMESPACE --timeout=300s

echo "5. Verifying scaled pods..."
kubectl get pods -n $NAMESPACE -l app=$COMPONENT

echo "6. Checking health endpoints..."
sleep 30
kubectl get pods -n $NAMESPACE -l app=$COMPONENT -o json | \
    jq -r '.items[] | select(.status.phase=="Running") | .metadata.name' | \
    while read pod; do
        echo "Checking health of $pod..."
        kubectl exec -n $NAMESPACE $pod -- curl -f localhost:8080/health || echo "Health check failed for $pod"
    done

echo "Scale up complete!"
```

#### Scale Down Procedure
```bash
#!/bin/bash
# scale-down.sh - Manual scale down procedure

COMPONENT=$1
REPLICAS=$2
NAMESPACE="oran-ric"

if [ -z "$COMPONENT" ] || [ -z "$REPLICAS" ]; then
    echo "Usage: $0 <component> <replicas>"
    exit 1
fi

echo "Scaling down $COMPONENT to $REPLICAS replicas..."

# Pre-scaling checks
echo "1. Checking current load..."
kubectl top pods -n $NAMESPACE -l app=$COMPONENT

echo "2. Draining connections (if applicable)..."
case $COMPONENT in
    "submgr")
        echo "Allowing subscriptions to complete..."
        sleep 60
        ;;
    "e2term")
        echo "Allowing E2 connections to stabilize..."
        sleep 30
        ;;
esac

echo "3. Scaling deployment..."
kubectl scale deployment $COMPONENT -n $NAMESPACE --replicas=$REPLICAS

echo "4. Waiting for scale down..."
kubectl rollout status deployment/$COMPONENT -n $NAMESPACE --timeout=300s

echo "5. Verifying remaining pods..."
kubectl get pods -n $NAMESPACE -l app=$COMPONENT

echo "Scale down complete!"
```

## Vertical Scaling Procedures

### Resource Limit Adjustment
```bash
#!/bin/bash
# adjust-resources.sh - Adjust resource limits

COMPONENT=$1
CPU_LIMIT=$2
MEMORY_LIMIT=$3
NAMESPACE="oran-ric"

if [ -z "$COMPONENT" ] || [ -z "$CPU_LIMIT" ] || [ -z "$MEMORY_LIMIT" ]; then
    echo "Usage: $0 <component> <cpu_limit> <memory_limit>"
    echo "Example: $0 submgr 2000m 4Gi"
    exit 1
fi

echo "Adjusting resources for $COMPONENT..."
echo "CPU: $CPU_LIMIT, Memory: $MEMORY_LIMIT"

# Create patch
PATCH=$(cat <<EOF
{
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "$COMPONENT",
            "resources": {
              "limits": {
                "cpu": "$CPU_LIMIT",
                "memory": "$MEMORY_LIMIT"
              },
              "requests": {
                "cpu": "$(echo $CPU_LIMIT | sed 's/000m/00m/')",
                "memory": "$(echo $MEMORY_LIMIT | sed 's/Gi/00Mi/' | sed 's/4/2/')"
              }
            }
          }
        ]
      }
    }
  }
}
EOF
)

echo "Applying resource changes..."
kubectl patch deployment $COMPONENT -n $NAMESPACE -p "$PATCH"

echo "Waiting for rollout..."
kubectl rollout status deployment/$COMPONENT -n $NAMESPACE --timeout=300s

echo "Verifying new resource limits..."
kubectl describe deployment $COMPONENT -n $NAMESPACE | grep -A 10 "Limits:"

echo "Resource adjustment complete!"
```

## Database Scaling

### Redis Scaling Procedures
```bash
#!/bin/bash
# scale-redis.sh - Redis scaling procedures

ACTION=$1  # scale-up, scale-out, optimize

case $ACTION in
    "scale-up")
        echo "Scaling up Redis resources..."
        kubectl patch deployment dbaas -n oran-ric -p '{
            "spec": {
                "template": {
                    "spec": {
                        "containers": [
                            {
                                "name": "redis",
                                "resources": {
                                    "limits": {
                                        "cpu": "4000m",
                                        "memory": "16Gi"
                                    }
                                }
                            }
                        ]
                    }
                }
            }
        }'
        ;;
    "scale-out")
        echo "Implementing Redis cluster..."
        # Deploy Redis Cluster
        helm install redis-cluster bitnami/redis-cluster \
            --namespace oran-ric \
            --set cluster.nodes=6 \
            --set cluster.replicas=1 \
            --set persistence.enabled=true \
            --set persistence.size=50Gi
        ;;
    "optimize")
        echo "Optimizing Redis configuration..."
        kubectl exec -n oran-ric deployment/dbaas -- redis-cli CONFIG SET maxmemory-policy allkeys-lru
        kubectl exec -n oran-ric deployment/dbaas -- redis-cli CONFIG SET save "900 1 300 10 60 10000"
        ;;
    *)
        echo "Usage: $0 {scale-up|scale-out|optimize}"
        exit 1
        ;;
esac
```

## Load Testing and Validation

### Capacity Validation Script
```bash
#!/bin/bash
# capacity-validation.sh - Validate scaling effectiveness

COMPONENT=$1
TARGET_LOAD=$2

echo "Validating capacity for $COMPONENT with target load $TARGET_LOAD..."

# Generate load
case $COMPONENT in
    "submgr")
        echo "Generating subscription load..."
        for i in $(seq 1 $TARGET_LOAD); do
            curl -X POST http://dashboard.ric.local/api/subscriptions \
                -H "Content-Type: application/json" \
                -d "{\"e2NodeId\":\"test-node-$i\",\"ranFunctionId\":1}" &
        done
        ;;
    "a1mediator")
        echo "Generating policy load..."
        for i in $(seq 1 $TARGET_LOAD); do
            curl -X PUT http://a1mediator.ric.local/a1-p/policytypes/1001/policies/policy-$i \
                -H "Content-Type: application/json" \
                -d "{\"qci\":7,\"priority\":$i}" &
        done
        ;;
esac

wait

# Monitor performance
echo "Monitoring performance for 5 minutes..."
for i in {1..30}; do
    echo "Sample $i:"
    kubectl top pods -n oran-ric -l app=$COMPONENT
    sleep 10
done

echo "Capacity validation complete!"
```

## Scaling Decision Matrix

### When to Scale Up
| Metric | Threshold | Action |
|--------|-----------|--------|
| CPU > 70% | 5 minutes | Add 1 replica |
| Memory > 80% | 5 minutes | Add 1 replica |
| E2 Connections > 50/pod | Immediate | Scale E2T |
| Subscriptions > 500/pod | Immediate | Scale SubMgr |
| Response time > 100ms | 2 minutes | Scale component |
| Queue depth > 1000 | Immediate | Scale component |

### When to Scale Down
| Metric | Threshold | Action |
|--------|-----------|--------|
| CPU < 30% | 15 minutes | Remove 1 replica |
| Memory < 40% | 15 minutes | Remove 1 replica |
| E2 Connections < 20/pod | 10 minutes | Scale down E2T |
| Subscriptions < 200/pod | 10 minutes | Scale down SubMgr |
| All pods < 50% utilization | 20 minutes | Scale down |

### Emergency Scaling
```bash
#!/bin/bash
# emergency-scale.sh - Emergency scaling procedures

EMERGENCY_TYPE=$1

case $EMERGENCY_TYPE in
    "high-load")
        echo "EMERGENCY: High load detected - scaling all components"
        kubectl scale deployment e2term submgr a1mediator -n oran-ric --replicas=5
        kubectl scale deployment e2mgr o1mediator -n oran-ric --replicas=3
        ;;
    "resource-exhaustion")
        echo "EMERGENCY: Resource exhaustion - emergency scale down"
        kubectl scale deployment --all -n oran-ric --replicas=1
        ;;
    "cascade-failure")
        echo "EMERGENCY: Cascade failure - restart all components"
        kubectl rollout restart deployment --all -n oran-ric
        ;;
    *)
        echo "Usage: $0 {high-load|resource-exhaustion|cascade-failure}"
        exit 1
        ;;
esac
```