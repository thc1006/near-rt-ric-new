# O-RAN Interactive Operations Console

This project provides a fully interactive, web-based operations console for O-RAN environments. It includes a dynamic UI, a production-grade SMO stack based on O-RAN SC components, and an integrated observability pipeline.

![Architecture Diagram](https://i.imgur.com/YOUR_DIAGRAM_HERE.png) <!-- Replace with a real diagram -->

## Features

- **Dynamic Dashboard:** A React-based UI that auto-discovers and displays all deployed network functions (near-RT RIC, O-CU/DU simulators, xApps, SMO micro-services).
- **Real-Time Observability:** Exposes real-time KPIs, alarms, and logs through an integrated observability stack, including Grafana, Prometheus, Loki, and Elasticsearch.
- **Production-Grade SMO:** Based on the O-RAN SC SMO package derived from ONAP Frankfurt, deployed via Helm.
- **End-to-End Automation:** Includes scripts for one-command setup and deployment in a micro-Kubernetes environment (KIND, K3s, etc.).

## Prerequisites

- Docker
- a Kubernetes cluster (e.g., KIND, K3s, Minikube)
- `kubectl`
- `make`

## One-Command Setup

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/your-repo/near-rt-ric-new.git
    cd near-rt-ric-new
    ```

2.  **Run the end-to-end test:**
    This command will deploy the entire stack and launch the UI.
    ```bash
    make e2e
    ```

## CI/CD

[![CI](https://github.com/your-repo/near-rt-ric-new/actions/workflows/ci.yaml/badge.svg)](https://github.com/your-repo/near-rt-ric-new/actions/workflows/ci.yaml)

## Security Notes

- The default deployment uses self-signed certificates. For production environments, it is recommended to use a proper certificate manager like cert-manager.
- Default passwords and tokens are used for services like Grafana and InfluxDB. These should be changed for production deployments.

## Troubleshooting

- If you encounter issues with the deployment, you can use the following commands to debug:
  ```bash
  # Check the status of all pods
  kubectl get pods -n ricplt

  # View the logs of a specific pod
  kubectl logs -n ricplt <pod-name>
  ```

## Dashboard and Grafana Views

**Interactive Dashboard:**
![Dashboard Screenshot](https://i.imgur.com/YOUR_DASHBOARD_SCREENSHOT.png) <!-- Replace with a real screenshot -->

**Grafana Dashboards:**
![Grafana Screenshot](https://i.imgur.com/YOUR_GRAFANA_SCREENSHOT.png) <!-- Replace with a real screenshot -->
