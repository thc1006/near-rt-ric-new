# O-RAN CU/DU Network Functions Deployment Guide

This guide provides comprehensive instructions for deploying O-RAN Central Unit (CU) and Distributed Unit (DU) network functions with proper F1/E1 interface configuration in the Near-RT RIC environment.

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Prerequisites](#prerequisites)
3. [Network Function Components](#network-function-components)
4. [Interface Configuration](#interface-configuration)
5. [Deployment Procedures](#deployment-procedures)
6. [Performance Optimization](#performance-optimization)
7. [Monitoring and Troubleshooting](#monitoring-and-troubleshooting)
8. [Operational Procedures](#operational-procedures)

## Architecture Overview

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│    5GC      │    │   Near-RT   │    │   E2 Node   │
│   (AMF)     │◄──►│    RIC      │◄──►│   (gNB)     │
└─────────────┘    └─────────────┘    └─────────────┘
       │N2                                   │E2
       ▼                                     ▼
┌─────────────┐                     ┌─────────────┐
│   CU-CP     │◄────────────────────┤    DU-1     │
│ (Control    │         F1-C        │(Distributed │
│  Plane)     │◄────────────────────┤   Unit)     │
└─────────────┘         F1-U        └─────────────┘
       │E1                                 │
       ▼                                   │
┌─────────────┐                           │
│   CU-UP     │                           │
│ (User Plane)│                           │
└─────────────┘                           │
       │N3                                │
       ▼                                  ▼
┌─────────────┐                 ┌─────────────┐
│    UPF      │                 │    DU-2     │
│ (User Plane │                 │(Distributed │
│  Function)  │                 │   Unit)     │
└─────────────┘                 └─────────────┘
```

### Key Interfaces:
- **F1-C**: Control plane interface between CU-CP and DU
- **F1-U**: User plane interface between CU-UP and DU
- **E1**: Interface between CU-CP and CU-UP
- **E2**: Interface between gNB components and Near-RT RIC
- **N2**: Interface between CU-CP and AMF
- **N3**: Interface between CU-UP and UPF

## Prerequisites

### System Requirements
- Kubernetes cluster v1.21+
- Helm v3.8+
- SR-IOV capable hardware (recommended for production)
- DPDK support for high-performance packet processing
- Intel VT-x or AMD-V virtualization support

### Required Tools
```bash
# Install required tools
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
curl https://get.helm.sh/helm-v3.8.0-linux-amd64.tar.gz | tar xz
sudo apt-get install jq yq
```

### Node Preparation
```bash
# Label nodes for CU workloads
kubectl label node <cu-node> node-role.kubernetes.io/cu=true

# Label nodes for DU workloads
kubectl label node <du-node> node-role.kubernetes.io/du=true

# Label SR-IOV capable nodes
kubectl label node <sriov-node> intel.feature.node.kubernetes.io/network-sriov=true
```

### Network Configuration
```bash
# Configure SR-IOV (if available)
echo 8 > /sys/class/net/enp1s0f0/device/sriov_numvfs
echo 8 > /sys/class/net/enp1s0f1/device/sriov_numvfs

# Configure hugepages
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
echo 512 > /sys/kernel/mm/hugepages/hugepages-1048576kB/nr_hugepages

# Mount hugepages
mkdir -p /dev/hugepages
mount -t hugetlbfs nodev /dev/hugepages
```

## Network Function Components

### 1. CU-CP (Central Unit - Control Plane)

**Purpose**: Handles RRC and PDCP-C functions, F1-C termination, and E1 interface with CU-UP.

**Configuration Files**:
- `configs/cu-cp-enhanced.yaml` - Main deployment configuration
- `configs/f1-interface-config.yaml` - F1 interface parameters
- `configs/e1-interface-config.yaml` - E1 interface parameters

**Key Features**:
- RRC connection management
- PDCP control plane processing
- F1-C message handling
- E1 bearer context management
- E2 service model support (KPM, RC)

### 2. CU-UP (Central Unit - User Plane)

**Purpose**: Handles PDCP-U and SDAP functions, F1-U termination, and N3 interface with UPF.

**Configuration Files**:
- `configs/cu-up-enhanced.yaml` - Main deployment configuration
- `configs/e1-interface-config.yaml` - E1 interface parameters

**Key Features**:
- PDCP user plane processing
- SDAP header compression/decompression
- GTP-U tunneling
- QoS flow management
- Network slicing support

### 3. DU (Distributed Unit)

**Purpose**: Handles RLC, MAC, and lower PHY functions, F1 termination, and radio interface.

**Configuration Files**:
- `configs/du-enhanced.yaml` - Main deployment configuration
- `configs/fronthaul-radio-config.yaml` - Radio and fronthaul parameters

**Key Features**:
- RLC acknowledged/unacknowledged mode
- MAC scheduling and HARQ
- PHY layer processing
- Fronthaul interface (eCPRI/CPRI)
- E2 measurement reporting

## Interface Configuration

### F1 Interface Setup

The F1 interface connects the CU with DU components:

```yaml
# F1-C (Control Plane) Configuration
f1_c_interface:
  protocol: "F1AP"
  transport: "SCTP"
  port: 38472
  
  # Timer configuration
  timer_values:
    t_wait: 3000
    t_ng_setup: 3000
    t_ue_context_setup: 5000

# F1-U (User Plane) Configuration  
f1_u_interface:
  protocol: "GTP-U"
  transport: "UDP"
  port: 2152
```

**Key Procedures**:
1. F1 Setup Request/Response
2. UE Context Setup/Release/Modification
3. DL/UL RRC Message Transfer
4. Initial UL RRC Message Transfer

### E1 Interface Setup

The E1 interface connects CU-CP with CU-UP:

```yaml
# E1 Interface Configuration
e1_interface:
  protocol: "E1AP"
  transport: "SCTP"
  port: 38462
  
  # Bearer context management
  bearer_context:
    max_bearers_per_ue: 16
    qos_flow_id_range: "1-63"
```

**Key Procedures**:
1. E1 Setup Request/Response
2. Bearer Context Setup/Modification/Release
3. gNB-CU-UP Status Indication

### E2 Interface Setup

The E2 interface connects gNB components with Near-RT RIC:

```yaml
# E2 Agent Configuration
e2_agent:
  near_ric_ip_addr: "e2term-service.oran-system.svc.cluster.local"
  near_ric_port: 36421
  
  # Service models
  service_models:
    - name: "ORAN-E2SM-KPM"
      version: "3.0.0"
      oid: "1.3.6.1.4.1.53148.1.2.2.2"
    - name: "ORAN-E2SM-RC"
      version: "1.0.3"
      oid: "1.3.6.1.4.1.53148.1.1.2.3"
```

## Deployment Procedures

### Quick Start Deployment

1. **Prepare the environment**:
```bash
# Clone the repository
git clone <repository-url>
cd near-rt-ric-new

# Make setup script executable
chmod +x scripts/setup-cu-du.sh
```

2. **Deploy CU/DU network functions**:
```bash
# Full deployment
./scripts/setup-cu-du.sh

# Dry run to validate configuration
DRY_RUN=true ./scripts/setup-cu-du.sh
```

3. **Verify deployment**:
```bash
./scripts/setup-cu-du.sh verify
```

### Manual Deployment Steps

#### Step 1: Create Namespace and Resources
```bash
kubectl create namespace oran-network-functions
kubectl apply -f configs/fronthaul-radio-config.yaml
kubectl apply -f configs/f1-interface-config.yaml
kubectl apply -f configs/e1-interface-config.yaml
```

#### Step 2: Deploy E2 Service Models
```bash
kubectl apply -f configs/e2-service-models-enhanced.yaml
```

#### Step 3: Deploy Network Functions
```bash
# Deploy CU-CP
kubectl apply -f configs/cu-cp-enhanced.yaml

# Deploy CU-UP
kubectl apply -f configs/cu-up-enhanced.yaml

# Deploy DU instances
kubectl apply -f configs/du-enhanced.yaml
```

#### Step 4: Configure Performance Optimization
```bash
kubectl apply -f configs/performance-monitoring.yaml
```

### Helm Chart Deployment

For production environments, use the Helm chart:

```bash
# Add dependencies
helm dependency update helm/cu-du-network-functions/

# Deploy with custom values
helm install cu-du-nf helm/cu-du-network-functions/ \
  --namespace oran-network-functions \
  --create-namespace \
  --values custom-values.yaml
```

## Performance Optimization

### CPU Optimization

```bash
# Isolate CPUs for real-time processing
echo "2-3" > /sys/fs/cgroup/cpuset/cpuset.cpus  # CU-CP
echo "4-7" > /sys/fs/cgroup/cpuset/cpuset.cpus  # CU-UP
echo "8-15" > /sys/fs/cgroup/cpuset/cpuset.cpus # DU

# Set CPU governor to performance
echo performance > /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Set real-time scheduling priority
chrt -f 99 $(pidof nr-softmodem)
```

### Memory Optimization

```bash
# Configure hugepages for DPDK
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# Configure NUMA policy
echo 1 > /proc/sys/kernel/numa_balancing

# Reduce swappiness
echo 10 > /proc/sys/vm/swappiness
```

### Network Optimization

```bash
# Increase network buffer sizes
echo 16777216 > /proc/sys/net/core/rmem_max
echo 16777216 > /proc/sys/net/core/wmem_max

# Configure interface ring buffers
ethtool -G eth0 rx 4096 tx 4096
ethtool -C eth0 rx-usecs 50 tx-usecs 50
```

## Monitoring and Troubleshooting

### Health Checks

```bash
# Check pod status
kubectl get pods -n oran-network-functions

# Check service endpoints
kubectl get services -n oran-network-functions

# Check logs
kubectl logs -f deployment/cu-cp -n oran-network-functions
kubectl logs -f deployment/cu-up -n oran-network-functions
kubectl logs -f deployment/du-1 -n oran-network-functions
```

### Interface Connectivity Tests

```bash
# Test F1 interface
kubectl exec -n oran-network-functions deployment/cu-cp -- netstat -tuln | grep :38472

# Test E1 interface
kubectl exec -n oran-network-functions deployment/cu-cp -- netstat -tuln | grep :38462

# Test E2 interface
kubectl exec -n oran-network-functions deployment/du-1 -- netstat -tuln | grep :36421
```

### Performance Monitoring

Access Grafana dashboard:
```bash
kubectl port-forward -n oran-network-functions svc/grafana 3000:3000
```

View metrics in browser: http://localhost:3000

### Common Issues and Solutions

#### 1. F1 Setup Timeout
**Symptoms**: CU-CP and DU cannot establish F1 connection
**Solution**: 
- Check network connectivity between pods
- Verify SCTP port 38472 is accessible
- Check F1AP configuration parameters

#### 2. E1 Setup Failure
**Symptoms**: CU-CP and CU-UP cannot establish E1 connection
**Solution**:
- Verify CU-UP service endpoint is accessible
- Check E1AP protocol configuration
- Review bearer context setup parameters

#### 3. High Latency
**Symptoms**: High message processing latency
**Solution**:
- Verify CPU isolation settings
- Check real-time scheduling priority
- Review network interface configuration

#### 4. Memory Issues
**Symptoms**: Out of memory errors
**Solution**:
- Verify hugepage configuration
- Check resource limits and requests
- Review memory allocation parameters

## Operational Procedures

### Scaling Operations

```bash
# Scale DU replicas
kubectl scale deployment du --replicas=4 -n oran-network-functions

# Rolling update
kubectl set image deployment/cu-cp cu-cp=oaisoftwarealliance/oai-gnb:v2.1.0 -n oran-network-functions
```

### Backup and Recovery

```bash
# Backup configurations
kubectl get all -n oran-network-functions -o yaml > cu-du-backup.yaml

# Backup persistent data
kubectl exec -n oran-network-functions deployment/cu-cp -- tar czf /backup/cu-cp-data.tar.gz /var/lib/cu-cp/
```

### Maintenance Procedures

```bash
# Graceful shutdown
kubectl delete deployment cu-cp --timeout=60s -n oran-network-functions

# Update configuration
kubectl apply -f updated-config.yaml

# Restart services
kubectl rollout restart deployment/cu-cp -n oran-network-functions
```

### Security Considerations

1. **Network Policies**: Restrict inter-pod communication
2. **RBAC**: Implement role-based access control
3. **Pod Security**: Use security contexts and policies
4. **Secrets Management**: Store sensitive data in Kubernetes secrets

## Conclusion

This deployment guide provides comprehensive instructions for setting up O-RAN CU/DU network functions with proper interface configuration. For production deployments, ensure proper hardware sizing, network configuration, and security measures are in place.

For additional support or questions, refer to the troubleshooting section or contact the O-RAN integration team.