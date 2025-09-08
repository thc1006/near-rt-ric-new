#!/bin/bash

# Enhanced O-RAN Analytics Framework Deployment Script
# This script deploys the complete telemetry data processing and analytics pipeline
# for O-RAN L Release with AI/ML capabilities and advanced optimization

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NAMESPACE="oran-analytics"
VERSION="${VERSION:-latest}"
WAIT_TIMEOUT=600 # 10 minutes

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "\n${PURPLE}================================${NC}"
    echo -e "${PURPLE}$1${NC}"
    echo -e "${PURPLE}================================${NC}\n"
}

print_step() {
    echo -e "${BLUE}▶ $1${NC}"
}

# Function to check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check required tools
    for cmd in kubectl docker helm; do
        if ! command -v $cmd &> /dev/null; then
            print_error "$cmd is required but not installed"
            exit 1
        else
            print_status "$cmd is available"
        fi
    done
    
    # Check Kubernetes cluster
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Kubernetes cluster is not accessible"
        exit 1
    else
        print_status "Kubernetes cluster is accessible"
    fi
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        exit 1
    else
        print_status "Docker daemon is running"
    fi
    
    # Check available resources
    local nodes=$(kubectl get nodes --no-headers | wc -l)
    if [ $nodes -lt 3 ]; then
        print_warning "Recommended minimum 3 nodes, found $nodes"
    else
        print_status "$nodes Kubernetes nodes available"
    fi
}

# Function to create namespace and basic resources
setup_namespace() {
    print_header "Setting up Namespace and Resources"
    
    print_step "Creating namespace '$NAMESPACE'"
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    print_step "Setting up RBAC permissions"
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: oran-analytics
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: oran-analytics
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["monitoring.coreos.com"]
  resources: ["servicemonitors", "prometheusrules"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: oran-analytics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: oran-analytics
subjects:
- kind: ServiceAccount
  name: oran-analytics
  namespace: $NAMESPACE
EOF
    
    print_step "Creating persistent volume claims"
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: kafka-data
  namespace: $NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
  storageClassName: ${STORAGE_CLASS:-default}
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: influxdb-data
  namespace: $NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: ${STORAGE_CLASS:-default}
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ml-models
  namespace: $NAMESPACE
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 10Gi
  storageClassName: ${SHARED_STORAGE_CLASS:-default}
EOF
}

# Function to build Docker images
build_images() {
    print_header "Building Docker Images"
    
    # Build E2 Telemetry Processor
    print_step "Building E2 Telemetry Processor image"
    cat > $PROJECT_ROOT/build/e2-telemetry-processor/Dockerfile <<EOF
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o e2-telemetry-processor cmd/e2-telemetry-processor/main.go

FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/e2-telemetry-processor .
EXPOSE 8089
CMD ["./e2-telemetry-processor"]
EOF
    docker build -f $PROJECT_ROOT/build/e2-telemetry-processor/Dockerfile -t oran/e2-telemetry-processor:$VERSION $PROJECT_ROOT
    
    # Build Performance Analytics Engine
    print_step "Building Performance Analytics Engine image"
    cat > $PROJECT_ROOT/build/performance-analytics/Dockerfile <<EOF
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o performance-analytics cmd/performance-analytics/main.go

FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/performance-analytics .
EXPOSE 8090
CMD ["./performance-analytics"]
EOF
    docker build -f $PROJECT_ROOT/build/performance-analytics/Dockerfile -t oran/performance-analytics:$VERSION $PROJECT_ROOT
    
    # Build Time Series Optimizer
    print_step "Building Time Series Optimizer image"
    cat > $PROJECT_ROOT/build/timeseries-optimizer/Dockerfile <<EOF
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o timeseries-optimizer cmd/timeseries-optimizer/main.go

FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/timeseries-optimizer .
EXPOSE 8091
CMD ["./timeseries-optimizer"]
EOF
    docker build -f $PROJECT_ROOT/build/timeseries-optimizer/Dockerfile -t oran/timeseries-optimizer:$VERSION $PROJECT_ROOT
    
    print_status "All Docker images built successfully"
}

# Function to deploy infrastructure components
deploy_infrastructure() {
    print_header "Deploying Infrastructure Components"
    
    # Deploy Kafka with KRaft mode
    print_step "Deploying Kafka cluster"
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm repo update
    
    cat > /tmp/kafka-values.yaml <<EOF
fullnameOverride: kafka
auth:
  clientProtocol: plaintext
kraft:
  enabled: true
listeners:
  client:
    containerPort: 9092
    protocol: PLAINTEXT
    name: CLIENT
  controller:
    name: CONTROLLER
    containerPort: 9093
    protocol: PLAINTEXT
  interbroker:
    containerPort: 9094
    protocol: PLAINTEXT
    name: INTERNAL
  external:
    containerPort: 9095
    protocol: PLAINTEXT
    name: EXTERNAL
service:
  ports:
    client: 9092
    external: 9095
persistence:
  enabled: true
  size: 50Gi
volumePermissions:
  enabled: true
metrics:
  kafka:
    enabled: true
  jmx:
    enabled: true
logPersistence:
  enabled: true
  size: 10Gi
EOF
    
    helm upgrade --install kafka bitnami/kafka \
        --namespace $NAMESPACE \
        --values /tmp/kafka-values.yaml \
        --wait --timeout=${WAIT_TIMEOUT}s
    
    # Deploy InfluxDB
    print_step "Deploying InfluxDB"
    helm repo add influxdata https://helm.influxdata.com/
    helm repo update
    
    # Create InfluxDB secrets
    kubectl create secret generic influxdb-auth \
        --from-literal=admin-password=oran123456 \
        --from-literal=admin-token=oran-super-secret-token-$(openssl rand -hex 16) \
        --namespace $NAMESPACE \
        --dry-run=client -o yaml | kubectl apply -f -
    
    cat > /tmp/influxdb-values.yaml <<EOF
adminUser:
  organization: "oran"
  bucket: "oran-metrics"
  user: "admin"
  retention_policy: "0s"
  existingSecret: influxdb-auth
image:
  tag: "2.7-alpine"
persistence:
  enabled: true
  size: 100Gi
resources:
  requests:
    memory: 2Gi
    cpu: 1
  limits:
    memory: 4Gi
    cpu: 2
service:
  type: ClusterIP
  port: 8086
EOF
    
    helm upgrade --install influxdb influxdata/influxdb2 \
        --namespace $NAMESPACE \
        --values /tmp/influxdb-values.yaml \
        --wait --timeout=${WAIT_TIMEOUT}s
    
    # Deploy Redis for caching
    print_step "Deploying Redis"
    helm repo add bitnami https://charts.bitnami.com/bitnami
    
    cat > /tmp/redis-values.yaml <<EOF
architecture: standalone
auth:
  enabled: false
master:
  persistence:
    enabled: true
    size: 8Gi
metrics:
  enabled: true
EOF
    
    helm upgrade --install redis bitnami/redis \
        --namespace $NAMESPACE \
        --values /tmp/redis-values.yaml \
        --wait --timeout=${WAIT_TIMEOUT}s
}

# Function to deploy analytics services
deploy_analytics_services() {
    print_header "Deploying Analytics Services"
    
    # Deploy E2 Telemetry Processor
    print_step "Deploying E2 Telemetry Processor"
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: e2-telemetry-processor
  namespace: $NAMESPACE
  labels:
    app: e2-telemetry-processor
    version: $VERSION
spec:
  replicas: 2
  selector:
    matchLabels:
      app: e2-telemetry-processor
  template:
    metadata:
      labels:
        app: e2-telemetry-processor
        version: $VERSION
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9098"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: oran-analytics
      containers:
      - name: e2-telemetry-processor
        image: oran/e2-telemetry-processor:$VERSION
        ports:
        - containerPort: 8089
          name: http
        - containerPort: 9098
          name: metrics
        env:
        - name: LOG_LEVEL
          value: "info"
        - name: KAFKA_BROKERS
          value: "kafka:9092"
        - name: INFLUXDB_URL
          value: "http://influxdb:8086"
        - name: INFLUXDB_TOKEN
          valueFrom:
            secretKeyRef:
              name: influxdb-auth
              key: admin-token
        - name: INFLUXDB_ORG
          value: "oran"
        resources:
          requests:
            memory: 512Mi
            cpu: 250m
          limits:
            memory: 1Gi
            cpu: 500m
        livenessProbe:
          httpGet:
            path: /health
            port: 8089
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8089
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: e2-telemetry-processor
  namespace: $NAMESPACE
  labels:
    app: e2-telemetry-processor
spec:
  ports:
  - port: 8089
    targetPort: 8089
    name: http
  - port: 9098
    targetPort: 9098
    name: metrics
  selector:
    app: e2-telemetry-processor
EOF
    
    # Deploy Performance Analytics Engine
    print_step "Deploying Performance Analytics Engine"
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: performance-analytics
  namespace: $NAMESPACE
  labels:
    app: performance-analytics
    version: $VERSION
spec:
  replicas: 2
  selector:
    matchLabels:
      app: performance-analytics
  template:
    metadata:
      labels:
        app: performance-analytics
        version: $VERSION
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9099"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: oran-analytics
      containers:
      - name: performance-analytics
        image: oran/performance-analytics:$VERSION
        ports:
        - containerPort: 8090
          name: http
        - containerPort: 9099
          name: metrics
        env:
        - name: LOG_LEVEL
          value: "info"
        - name: KAFKA_BROKERS
          value: "kafka:9092"
        - name: INFLUXDB_URL
          value: "http://influxdb:8086"
        - name: INFLUXDB_TOKEN
          valueFrom:
            secretKeyRef:
              name: influxdb-auth
              key: admin-token
        - name: INFLUXDB_ORG
          value: "oran"
        resources:
          requests:
            memory: 1Gi
            cpu: 500m
          limits:
            memory: 2Gi
            cpu: 1
        livenessProbe:
          httpGet:
            path: /health
            port: 8090
          initialDelaySeconds: 60
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8090
          initialDelaySeconds: 30
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: performance-analytics
  namespace: $NAMESPACE
  labels:
    app: performance-analytics
spec:
  ports:
  - port: 8090
    targetPort: 8090
    name: http
  - port: 9099
    targetPort: 9099
    name: metrics
  selector:
    app: performance-analytics
EOF
    
    # Deploy Time Series Optimizer
    print_step "Deploying Time Series Optimizer"
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: timeseries-optimizer
  namespace: $NAMESPACE
  labels:
    app: timeseries-optimizer
    version: $VERSION
spec:
  replicas: 1
  selector:
    matchLabels:
      app: timeseries-optimizer
  template:
    metadata:
      labels:
        app: timeseries-optimizer
        version: $VERSION
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9100"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: oran-analytics
      containers:
      - name: timeseries-optimizer
        image: oran/timeseries-optimizer:$VERSION
        ports:
        - containerPort: 8091
          name: http
        - containerPort: 9100
          name: metrics
        env:
        - name: LOG_LEVEL
          value: "info"
        - name: INFLUXDB_URL
          value: "http://influxdb:8086"
        - name: INFLUXDB_TOKEN
          valueFrom:
            secretKeyRef:
              name: influxdb-auth
              key: admin-token
        - name: INFLUXDB_ORG
          value: "oran"
        resources:
          requests:
            memory: 1Gi
            cpu: 500m
          limits:
            memory: 2Gi
            cpu: 1
        volumeMounts:
        - name: ml-models
          mountPath: /models
        livenessProbe:
          httpGet:
            path: /health
            port: 8091
          initialDelaySeconds: 60
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8091
          initialDelaySeconds: 30
          periodSeconds: 10
      volumes:
      - name: ml-models
        persistentVolumeClaim:
          claimName: ml-models
---
apiVersion: v1
kind: Service
metadata:
  name: timeseries-optimizer
  namespace: $NAMESPACE
  labels:
    app: timeseries-optimizer
spec:
  ports:
  - port: 8091
    targetPort: 8091
    name: http
  - port: 9100
    targetPort: 9100
    name: metrics
  selector:
    app: timeseries-optimizer
EOF
}

# Function to setup monitoring
setup_monitoring() {
    print_header "Setting up Monitoring and Alerting"
    
    # Deploy ServiceMonitors for Prometheus
    print_step "Creating ServiceMonitors"
    cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: oran-analytics-services
  namespace: $NAMESPACE
  labels:
    app: oran-analytics
spec:
  selector:
    matchLabels:
      app: e2-telemetry-processor
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: oran-performance-analytics
  namespace: $NAMESPACE
  labels:
    app: oran-analytics
spec:
  selector:
    matchLabels:
      app: performance-analytics
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: oran-timeseries-optimizer
  namespace: $NAMESPACE
  labels:
    app: oran-analytics
spec:
  selector:
    matchLabels:
      app: timeseries-optimizer
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
EOF
    
    # Deploy Prometheus rules
    print_step "Applying Prometheus alerting rules"
    kubectl apply -f $PROJECT_ROOT/monitoring/oran-analytics-alerts.yaml -n $NAMESPACE
    
    print_status "Monitoring setup completed"
}

# Function to setup visualization
setup_visualization() {
    print_header "Setting up Visualization"
    
    # Deploy Grafana if not exists
    if ! helm list -n monitoring | grep -q grafana; then
        print_step "Deploying Grafana"
        helm repo add grafana https://grafana.github.io/helm-charts
        helm repo update
        
        cat > /tmp/grafana-values.yaml <<EOF
adminPassword: admin123
persistence:
  enabled: true
  size: 10Gi
service:
  type: ClusterIP
  port: 3000
datasources:
  datasources.yaml:
    apiVersion: 1
    datasources:
    - name: Prometheus
      type: prometheus
      url: http://prometheus:9090
      access: proxy
      isDefault: true
    - name: InfluxDB
      type: influxdb
      url: http://influxdb.${NAMESPACE}:8086
      access: proxy
      database: oran-metrics
      user: admin
      secureJsonData:
        token: \$INFLUXDB_TOKEN
dashboardProviders:
  dashboardproviders.yaml:
    apiVersion: 1
    providers:
    - name: 'oran-dashboards'
      orgId: 1
      folder: 'O-RAN Analytics'
      type: file
      disableDeletion: false
      updateIntervalSeconds: 10
      allowUiUpdates: true
      options:
        path: /var/lib/grafana/dashboards/oran-analytics
dashboards:
  oran-analytics:
    oran-advanced-analytics:
      file: dashboards/oran-advanced-analytics.json
sidecar:
  dashboards:
    enabled: true
    label: grafana_dashboard
    folder: /tmp/dashboards
  datasources:
    enabled: true
    label: grafana_datasource
EOF
        
        helm upgrade --install grafana grafana/grafana \
            --namespace monitoring \
            --create-namespace \
            --values /tmp/grafana-values.yaml \
            --wait --timeout=${WAIT_TIMEOUT}s
    else
        print_status "Grafana already deployed"
    fi
    
    # Copy dashboard configuration
    print_step "Configuring Grafana dashboards"
    kubectl create configmap oran-advanced-dashboard \
        --from-file=$PROJECT_ROOT/monitoring/dashboards/oran-advanced-analytics.json \
        --namespace monitoring \
        --dry-run=client -o yaml | kubectl apply -f -
    
    kubectl label configmap oran-advanced-dashboard grafana_dashboard=1 --namespace monitoring --overwrite
}

# Function to initialize data and configurations
initialize_data() {
    print_header "Initializing Data and Configurations"
    
    print_step "Waiting for services to be ready"
    kubectl wait --for=condition=available --timeout=${WAIT_TIMEOUT}s deployment/e2-telemetry-processor -n $NAMESPACE
    kubectl wait --for=condition=available --timeout=${WAIT_TIMEOUT}s deployment/performance-analytics -n $NAMESPACE
    kubectl wait --for=condition=available --timeout=${WAIT_TIMEOUT}s deployment/timeseries-optimizer -n $NAMESPACE
    
    print_step "Creating Kafka topics"
    kubectl run kafka-client --image=confluentinc/cp-kafka:latest --rm -i --restart=Never -n $NAMESPACE -- bash -c "
        kafka-topics --bootstrap-server kafka:9092 --create --if-not-exists --topic e2-indications --partitions 10 --replication-factor 1
        kafka-topics --bootstrap-server kafka:9092 --create --if-not-exists --topic processed-e2-data --partitions 6 --replication-factor 1
        kafka-topics --bootstrap-server kafka:9092 --create --if-not-exists --topic performance-insights --partitions 4 --replication-factor 1
        kafka-topics --bootstrap-server kafka:9092 --create --if-not-exists --topic analytics-requests --partitions 2 --replication-factor 1
    " || true
    
    print_step "Initializing InfluxDB buckets"
    local influxdb_token=$(kubectl get secret influxdb-auth -n $NAMESPACE -o jsonpath="{.data.admin-token}" | base64 -d)
    kubectl run influxdb-init --image=influxdb:2.7-alpine --rm -i --restart=Never -n $NAMESPACE -- bash -c "
        influx bucket create --host http://influxdb:8086 --token $influxdb_token --org oran --name oran-e2-data --retention 7d || true
        influx bucket create --host http://influxdb:8086 --token $influxdb_token --org oran --name oran-kpis --retention 30d || true
        influx bucket create --host http://influxdb:8086 --token $influxdb_token --org oran --name oran-predictions --retention 7d || true
        influx bucket create --host http://influxdb:8086 --token $influxdb_token --org oran --name oran-anomalies --retention 30d || true
        influx bucket create --host http://influxdb:8086 --token $influxdb_token --org oran --name oran-analytics --retention 90d || true
    " || true
    
    print_status "Data initialization completed"
}

# Function to run validation tests
run_validation() {
    print_header "Running Validation Tests"
    
    print_step "Testing service endpoints"
    
    # Port forward services for testing
    kubectl port-forward service/e2-telemetry-processor 8089:8089 -n $NAMESPACE &
    local e2_pid=$!
    kubectl port-forward service/performance-analytics 8090:8090 -n $NAMESPACE &
    local perf_pid=$!
    kubectl port-forward service/timeseries-optimizer 8091:8091 -n $NAMESPACE &
    local ts_pid=$!
    
    sleep 10
    
    # Test health endpoints
    if curl -s http://localhost:8089/health | grep -q "healthy"; then
        print_status "E2 Telemetry Processor health check: PASS"
    else
        print_error "E2 Telemetry Processor health check: FAIL"
    fi
    
    if curl -s http://localhost:8090/health | grep -q "healthy"; then
        print_status "Performance Analytics Engine health check: PASS"
    else
        print_error "Performance Analytics Engine health check: FAIL"
    fi
    
    if curl -s http://localhost:8091/health | grep -q "healthy"; then
        print_status "Time Series Optimizer health check: PASS"
    else
        print_error "Time Series Optimizer health check: FAIL"
    fi
    
    # Clean up port forwards
    kill $e2_pid $perf_pid $ts_pid 2>/dev/null || true
    
    print_step "Testing Kafka connectivity"
    kubectl run kafka-test --image=confluentinc/cp-kafka:latest --rm -i --restart=Never -n $NAMESPACE -- bash -c "
        kafka-topics --bootstrap-server kafka:9092 --list | grep e2-indications
    " && print_status "Kafka connectivity: PASS" || print_error "Kafka connectivity: FAIL"
    
    print_step "Testing InfluxDB connectivity"
    local influxdb_token=$(kubectl get secret influxdb-auth -n $NAMESPACE -o jsonpath="{.data.admin-token}" | base64 -d)
    kubectl run influxdb-test --image=influxdb:2.7-alpine --rm -i --restart=Never -n $NAMESPACE -- bash -c "
        influx ping --host http://influxdb:8086 --token $influxdb_token
    " && print_status "InfluxDB connectivity: PASS" || print_error "InfluxDB connectivity: FAIL"
}

# Function to display access information
show_access_info() {
    print_header "Access Information"
    
    echo -e "${CYAN}🎯 Service Endpoints:${NC}"
    echo -e "  E2 Telemetry Processor: kubectl port-forward service/e2-telemetry-processor 8089:8089 -n $NAMESPACE"
    echo -e "  Performance Analytics:  kubectl port-forward service/performance-analytics 8090:8090 -n $NAMESPACE"
    echo -e "  Time Series Optimizer:  kubectl port-forward service/timeseries-optimizer 8091:8091 -n $NAMESPACE"
    
    echo -e "\n${CYAN}📊 Infrastructure Access:${NC}"
    echo -e "  InfluxDB:    kubectl port-forward service/influxdb 8086:8086 -n $NAMESPACE"
    echo -e "  Kafka UI:    kubectl port-forward service/kafka 9092:9092 -n $NAMESPACE"
    echo -e "  Redis:       kubectl port-forward service/redis-master 6379:6379 -n $NAMESPACE"
    
    echo -e "\n${CYAN}🔍 Monitoring Access:${NC}"
    echo -e "  Grafana:     kubectl port-forward service/grafana 3000:3000 -n monitoring"
    echo -e "  Prometheus:  kubectl port-forward service/prometheus 9090:9090 -n monitoring"
    
    echo -e "\n${CYAN}🔑 Credentials:${NC}"
    echo -e "  InfluxDB Token: kubectl get secret influxdb-auth -n $NAMESPACE -o jsonpath='{.data.admin-token}' | base64 -d"
    echo -e "  Grafana Admin:  admin / admin123"
    
    echo -e "\n${CYAN}📋 Useful Commands:${NC}"
    echo -e "  View logs:      kubectl logs -f deployment/e2-telemetry-processor -n $NAMESPACE"
    echo -e "  Scale services: kubectl scale deployment/performance-analytics --replicas=3 -n $NAMESPACE"
    echo -e "  Check status:   kubectl get pods,svc,pvc -n $NAMESPACE"
}

# Function to clean up deployment
cleanup() {
    print_header "Cleaning Up Enhanced Analytics Deployment"
    
    print_step "Removing Kubernetes resources"
    kubectl delete namespace $NAMESPACE --ignore-not-found=true
    
    print_step "Removing Docker images"
    docker rmi -f oran/e2-telemetry-processor:$VERSION 2>/dev/null || true
    docker rmi -f oran/performance-analytics:$VERSION 2>/dev/null || true
    docker rmi -f oran/timeseries-optimizer:$VERSION 2>/dev/null || true
    
    print_step "Removing Helm releases"
    helm uninstall kafka -n $NAMESPACE 2>/dev/null || true
    helm uninstall influxdb -n $NAMESPACE 2>/dev/null || true
    helm uninstall redis -n $NAMESPACE 2>/dev/null || true
    
    print_status "Cleanup completed"
}

# Main function
main() {
    case "${1:-deploy}" in
        "deploy")
            check_prerequisites
            setup_namespace
            build_images
            deploy_infrastructure
            deploy_analytics_services
            setup_monitoring
            setup_visualization
            initialize_data
            run_validation
            show_access_info
            ;;
        "verify")
            run_validation
            show_access_info
            ;;
        "cleanup")
            cleanup
            ;;
        "build")
            build_images
            ;;
        "monitor")
            setup_monitoring
            setup_visualization
            ;;
        *)
            echo "Usage: $0 {deploy|verify|cleanup|build|monitor}"
            echo ""
            echo "Commands:"
            echo "  deploy  - Deploy complete enhanced analytics framework"
            echo "  verify  - Run validation tests and show access info"
            echo "  cleanup - Remove all deployed resources"
            echo "  build   - Build Docker images only"
            echo "  monitor - Setup monitoring and visualization only"
            exit 1
            ;;
    esac
}

# Error handling
trap 'print_error "Script failed at line $LINENO"' ERR

# Execute main function
main "$@"