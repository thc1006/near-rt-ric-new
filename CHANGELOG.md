# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2025-07-25

### Added

- **Interactive Dashboard:** A new React-based UI for real-time monitoring and operations.
- **O-RAN SC SMO:** A production-grade Service Management and Orchestration (SMO) stack based on ONAP Frankfurt.
- **Observability Pipeline:** An integrated observability stack with Grafana, Prometheus, Loki, and Elasticsearch.
- **End-to-End Automation:** New `make` targets for one-command setup and deployment.

### Changed

- **Architecture:** The project has been completely refactored to use a modern, microservices-based architecture aligned with O-RAN SC standards.
- **Deployment:** The deployment process is now managed by Helm, using the official O-RAN SC charts.

### Removed

- **Legacy Code:** All previous placeholder code and non-compliant components have been removed.
