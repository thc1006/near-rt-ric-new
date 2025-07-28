# Product Overview

## O-RAN Interactive Operations Console

This project provides a fully interactive, web-based operations console for O-RAN (Open Radio Access Network) environments. It's designed to manage and monitor O-RAN network functions in production environments.

### Key Features

- **Dynamic Dashboard**: React-based UI that auto-discovers and displays deployed network functions (near-RT RIC, O-CU/DU simulators, xApps, SMO micro-services)
- **Real-Time Observability**: Integrated observability stack with Grafana, Prometheus, Loki, and Elasticsearch for KPIs, alarms, and logs
- **Production-Grade SMO**: Based on O-RAN SC SMO package derived from ONAP Frankfurt
- **End-to-End Automation**: One-command setup and deployment for micro-Kubernetes environments

### Target Environment

- Kubernetes-based deployments (KIND, K3s, Minikube)
- O-RAN Service Management and Orchestration (SMO) environments
- Near-RT RIC (Real-time Intelligent Controller) platforms
- xApp (applications running on the RIC platform) management

### Architecture

The system consists of:
- Go-based backend services for RIC management
- React frontend for interactive dashboard
- Helm charts for Kubernetes deployment
- Integrated observability pipeline
- Health monitoring and automation scripts