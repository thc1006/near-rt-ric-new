#!/bin/bash
# O-RAN Near-RT RIC Development Environment Setup
# This script sets up the complete development environment

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="O-RAN Near-RT RIC"
REQUIRED_TOOLS=("docker" "docker-compose" "kubectl" "helm" "go" "node" "npm")
GO_MIN_VERSION="1.21"
NODE_MIN_VERSION="16"

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check version
check_version() {
    local tool=$1
    local required=$2
    local current=""
    
    case $tool in
        "go")
            current=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
            ;;
        "node")
            current=$(node --version | sed 's/v//')
            ;;
        *)
            return 0
            ;;
    esac
    
    if ! printf '%s\n%s\n' "$required" "$current" | sort -V -C; then
        error "$tool version $current is lower than required $required"
    fi
    
    log "$tool version $current meets requirement (>= $required)"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites for $PROJECT_NAME development environment..."
    
    for tool in "${REQUIRED_TOOLS[@]}"; do
        if ! command_exists "$tool"; then
            error "$tool is not installed. Please install $tool and try again."
        fi
        log "✓ $tool is installed"
    done
    
    # Check specific version requirements
    check_version "go" "$GO_MIN_VERSION"
    check_version "node" "$NODE_MIN_VERSION"
    
    # Check Docker daemon
    if ! docker info >/dev/null 2>&1; then
        error "Docker daemon is not running. Please start Docker and try again."
    fi
    log "✓ Docker daemon is running"
    
    # Check Kubernetes context (optional)
    if command_exists kubectl; then
        if ! kubectl cluster-info >/dev/null 2>&1; then
            warn "No Kubernetes cluster found. Local deployment will be unavailable."
        else
            log "✓ Kubernetes cluster is accessible"
        fi
    fi
}

# Setup Go development environment
setup_go_environment() {
    log "Setting up Go development environment..."
    
    # Install Go development tools
    log "Installing Go development tools..."
    make install-tools || {
        warn "Failed to install tools via Makefile, trying manual installation..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        go install github.com/cosmtrek/air@latest
        go install github.com/golang/mock/mockgen@latest
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    }
    
    # Download Go dependencies
    log "Downloading Go dependencies..."
    go mod download
    go mod verify
    
    log "✓ Go environment setup complete"
}

# Setup Node.js environment
setup_node_environment() {
    if [ ! -d "ui" ]; then
        warn "UI directory not found, skipping Node.js setup"
        return 0
    fi
    
    log "Setting up Node.js development environment..."
    
    cd ui
    
    # Install npm dependencies
    log "Installing Node.js dependencies..."
    npm ci
    
    # Install global development tools
    if ! command_exists eslint; then
        npm install -g eslint
    fi
    
    cd ..
    log "✓ Node.js environment setup complete"
}

# Create necessary directories
setup_directories() {
    log "Creating project directories..."
    
    directories=(
        "bin"
        "build"
        "coverage"
        "dist"
        "logs"
        "configs/nginx"
        "configs/grafana/provisioning/dashboards"
        "configs/grafana/provisioning/datasources"
        "configs/prometheus/rules"
        "test/load"
        "docs/api"
    )
    
    for dir in "${directories[@]}"; do
        mkdir -p "$dir"
        log "✓ Created directory: $dir"
    done
}

# Setup development configurations
setup_dev_configs() {
    log "Setting up development configurations..."
    
    # Create Grafana datasources configuration
    cat > configs/grafana/provisioning/datasources/datasources.yml << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    
  - name: Jaeger
    type: jaeger
    access: proxy
    url: http://jaeger:16686
EOF

    # Create Grafana dashboards configuration
    cat > configs/grafana/provisioning/dashboards/dashboards.yml << 'EOF'
apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
EOF

    # Create Prometheus alerting rules
    cat > configs/prometheus/rules/oran.yml << 'EOF'
groups:
  - name: oran-alerts
    rules:
      - alert: HighCPUUsage
        expr: rate(process_cpu_seconds_total[5m]) > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage detected"
          description: "CPU usage is above 80% for more than 5 minutes"
          
      - alert: ServiceDown
        expr: up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Service {{ $labels.job }} is down"
          description: "Service {{ $labels.job }} has been down for more than 1 minute"
          
      - alert: HighMemoryUsage
        expr: (process_resident_memory_bytes / 1024 / 1024) > 500
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is above 500MB for more than 5 minutes"
EOF

    # Create nginx development configuration
    cat > configs/nginx/nginx.conf << 'EOF'
events {
    worker_connections 1024;
}

http {
    upstream dashboard_api {
        server dashboard-api:8080;
    }
    
    upstream ui_dev {
        server ui-dev:3000;
    }
    
    server {
        listen 80;
        server_name localhost;
        
        # API routes
        location /api/ {
            proxy_pass http://dashboard_api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
        
        # WebSocket support
        location /ws {
            proxy_pass http://dashboard_api;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
        }
        
        # UI routes
        location / {
            proxy_pass http://ui_dev;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
EOF

    log "✓ Development configurations created"
}

# Create sample load test
create_load_test() {
    log "Creating sample load test..."
    
    cat > test/load/load-test.js << 'EOF'
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

export let options = {
  stages: [
    { duration: '2m', target: 10 }, // Ramp up to 10 users
    { duration: '5m', target: 10 }, // Stay at 10 users
    { duration: '2m', target: 20 }, // Ramp up to 20 users
    { duration: '5m', target: 20 }, // Stay at 20 users
    { duration: '2m', target: 0 },  // Ramp down to 0 users
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500'], // 95% of requests should be below 500ms
    'errors': ['rate<0.1'], // Error rate should be less than 10%
  },
};

export default function() {
  // Test dashboard API health endpoint
  const healthRes = http.get('http://dashboard-api:8080/health');
  check(healthRes, {
    'health status is 200': (r) => r.status === 200,
    'health response time < 200ms': (r) => r.timings.duration < 200,
  }) || errorRate.add(1);

  // Test API endpoints
  const apiRes = http.get('http://dashboard-api:8080/api/v1/status');
  check(apiRes, {
    'api status is 200': (r) => r.status === 200,
    'api response time < 500ms': (r) => r.timings.duration < 500,
  }) || errorRate.add(1);

  sleep(1);
}
EOF

    log "✓ Load test created"
}

# Setup development database
setup_dev_database() {
    log "Setting up development database..."
    
    # Create database initialization scripts
    mkdir -p scripts/db
    
    cat > scripts/db/init-dev.sql << 'EOF'
-- Development database initialization
CREATE DATABASE IF NOT EXISTS oran_test;
CREATE DATABASE IF NOT EXISTS sonar;

-- Create sonar user for SonarQube
CREATE USER IF NOT EXISTS 'sonar'@'%' IDENTIFIED BY 'sonar123';
GRANT ALL PRIVILEGES ON sonar.* TO 'sonar'@'%';

-- Create test data tables
USE oran;

-- Example: xApps table
CREATE TABLE IF NOT EXISTS xapps (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    version VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'stopped',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Example: E2 nodes table
CREATE TABLE IF NOT EXISTS e2_nodes (
    id SERIAL PRIMARY KEY,
    node_id VARCHAR(255) NOT NULL UNIQUE,
    plmn_id VARCHAR(10) NOT NULL,
    nb_id VARCHAR(20) NOT NULL,
    ran_name VARCHAR(255),
    connection_status VARCHAR(50) DEFAULT 'disconnected',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

FLUSH PRIVILEGES;
EOF

    cat > scripts/db/test-data.sql << 'EOF'
-- Test data for development
USE oran;

-- Insert sample xApps
INSERT INTO xapps (name, version, status) VALUES
    ('hello-world', '1.0.0', 'running'),
    ('kpimon', '1.2.0', 'stopped'),
    ('traffic-steering', '2.0.0', 'running')
ON DUPLICATE KEY UPDATE version = VALUES(version);

-- Insert sample E2 nodes
INSERT INTO e2_nodes (node_id, plmn_id, nb_id, ran_name, connection_status) VALUES
    ('gnb_001', '310410', '1', 'gNodeB-001', 'connected'),
    ('gnb_002', '310410', '2', 'gNodeB-002', 'connected'),
    ('enb_001', '310410', '3', 'eNodeB-001', 'disconnected')
ON DUPLICATE KEY UPDATE connection_status = VALUES(connection_status);
EOF

    log "✓ Development database scripts created"
}

# Validate environment
validate_environment() {
    log "Validating development environment..."
    
    # Check if project structure is valid
    make validate || error "Project validation failed"
    
    # Test Docker Compose configuration
    docker-compose -f docker-compose.yml -f docker-compose.dev.yml config --quiet || {
        error "Docker Compose configuration is invalid"
    }
    
    log "✓ Environment validation passed"
}

# Main setup function
main() {
    echo -e "${BLUE}"
    echo "========================================"
    echo "$PROJECT_NAME Development Setup"
    echo "========================================"
    echo -e "${NC}"
    
    check_prerequisites
    setup_directories
    setup_go_environment
    setup_node_environment
    setup_dev_configs
    create_load_test
    setup_dev_database
    validate_environment
    
    echo -e "${GREEN}"
    echo "========================================"
    echo "✅ Development environment setup complete!"
    echo "========================================"
    echo -e "${NC}"
    
    echo ""
    echo "🚀 Quick start commands:"
    echo "  make help                    - Show all available commands"
    echo "  make dev-stack               - Start full development stack"
    echo "  make build                   - Build all components"
    echo "  make test                    - Run tests"
    echo "  docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d"
    echo ""
    echo "🌐 Development URLs:"
    echo "  Dashboard API:    http://localhost:8080"
    echo "  UI Dev Server:    http://localhost:3001"
    echo "  Grafana:         http://localhost:3000 (admin/admin123)"
    echo "  Prometheus:      http://localhost:9092"
    echo "  Jaeger:          http://localhost:16686"
    echo "  SonarQube:       http://localhost:9000 (admin/admin)"
    echo ""
    echo "📚 Documentation:"
    echo "  API Documentation: http://localhost:8085"
    echo "  Load Testing:      k6 run test/load/load-test.js"
    echo ""
}

# Run main function
main "$@"