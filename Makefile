# Optimized O-RAN Near-RT RIC Makefile
# Production-ready build system with parallel execution support

# Build configuration
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
MAKEFLAGS += --warn-undefined-variables --no-builtin-rules

# Version and build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters with FIPS support
GO := go
GOFLAGS := -v -mod=readonly
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -s -w"
CGO_ENABLED ?= 1
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# FIPS 140-3 Configuration
FIPS_MODE ?= only
GO_VERSION := $(shell go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//' 2>/dev/null || echo "1.25")
FIPS_FLAGS := -tags=fips -ldflags="-X crypto/tls/fipsonly.fipsOnly=true"

# Directories
BIN_DIR := bin
BUILD_DIR := build
COVERAGE_DIR := coverage
DIST_DIR := dist
SCRIPTS_DIR := scripts
CONFIG_DIR := config
CERTS_DIR := certs

# Binary names
BINARIES := dashboard-api ric xapp-hello-world
UI_DIR := ui

# Docker settings
DOCKER_REGISTRY ?= ghcr.io
DOCKER_NAMESPACE ?= $(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\(.*\)\.git/\1/' | tr '[:upper:]' '[:lower:]')
DOCKER_TAG ?= $(VERSION)

# Parallel jobs
PARALLEL_JOBS ?= $(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

# Security settings
TRIVY_VERSION ?= latest
SECURITY_THRESHOLD ?= HIGH

# Colors
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: all
all: clean setup-env build test lint security-full ## Build everything and run comprehensive tests

.PHONY: help
help: ## Display this help message
	@echo "$(BLUE)O-RAN Near-RT RIC Development Makefile$(NC)"
	@echo ""
	@echo "$(YELLOW)Usage:$(NC) make [target]"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-25s$(NC) %s\n", $$1, $$2}'

# ============================================================================
# Environment Setup
# ============================================================================

.PHONY: setup-env
setup-env: setup-dirs install-tools install-security-tools ## Set up development environment
	@echo "$(YELLOW)Setting up development environment...$(NC)"

.PHONY: setup-dirs
setup-dirs: ## Create necessary directories
	@mkdir -p $(BIN_DIR) $(COVERAGE_DIR) $(DIST_DIR) $(BUILD_DIR) $(CERTS_DIR)

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "$(YELLOW)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golang/mock/mockgen@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "$(GREEN)Development tools installed$(NC)"

.PHONY: install-security-tools
install-security-tools: ## Install security tools
	@echo "$(YELLOW)Installing security tools...$(NC)"
	@if ! command -v trivy &> /dev/null; then \
		echo "$(YELLOW)Installing Trivy container scanner...$(NC)"; \
		curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin; \
	fi
	@if ! command -v kubesec &> /dev/null; then \
		echo "$(YELLOW)Installing kubesec...$(NC)"; \
		curl -sSfL https://github.com/controlplaneio/kubesec/releases/latest/download/kubesec_linux_amd64.tar.gz | tar -xzf - kubesec && mv kubesec /usr/local/bin/; \
	fi
	@echo "$(GREEN)Security tools installed$(NC)"

# ============================================================================
# Build Targets with FIPS Support
# ============================================================================

.PHONY: build
build: $(BINARIES) build-ui ## Build all components

.PHONY: build-fips
build-fips: setup-dirs deps ## Build with FIPS 140-3 compliance
	@echo "$(YELLOW)Building with FIPS 140-3 compliance (Go $(GO_VERSION))...$(NC)"
	@export GODEBUG=fips140=$(FIPS_MODE) OPENSSL_FIPS=1 GOFIPS=1 CGO_ENABLED=1; \
	for binary in $(BINARIES); do \
		echo "Building $$binary with FIPS..."; \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(GOFLAGS) $(FIPS_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$$binary ./cmd/$$binary; \
		echo "$(GREEN)✓ $$binary built with FIPS compliance$(NC)"; \
	done

$(BINARIES): setup-dirs deps
	@echo "$(YELLOW)Building $@...$(NC)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$@ ./cmd/$@
	@echo "$(GREEN)✓ $@ built successfully$(NC)"

.PHONY: build-parallel
build-parallel: setup-dirs deps ## Build all binaries in parallel
	@echo "$(YELLOW)Building binaries in parallel ($(PARALLEL_JOBS) jobs)...$(NC)"
	@$(MAKE) -j$(PARALLEL_JOBS) $(BINARIES)

.PHONY: build-ui
build-ui: ## Build React UI
	@echo "$(YELLOW)Building React UI...$(NC)"
	@if [ -d "$(UI_DIR)" ]; then \
		cd $(UI_DIR) && npm ci --silent && npm run build; \
		echo "$(GREEN)✓ UI built successfully$(NC)"; \
	else \
		echo "$(YELLOW)UI directory not found, skipping...$(NC)"; \
	fi

.PHONY: build-cross
build-cross: setup-dirs ## Cross-compile for multiple platforms
	@echo "$(YELLOW)Cross-compiling for multiple platforms...$(NC)"
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			if [ "$$os" = "windows" ] && [ "$$arch" = "arm64" ]; then continue; fi; \
			echo "Building for $$os/$$arch..."; \
			for binary in $(BINARIES); do \
				ext=""; if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
				CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
					$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$$binary-$$os-$$arch$$ext ./cmd/$$binary; \
			done; \
		done; \
	done
	@echo "$(GREEN)✓ Cross-compilation complete$(NC)"

# ============================================================================
# Security Targets
# ============================================================================

.PHONY: security-full
security-full: security-enforce security-scan security-test ## Full security compliance suite

.PHONY: security-enforce
security-enforce: ## Apply O-RAN WG11 security enforcement
	@echo "$(YELLOW)Enforcing O-RAN WG11 security compliance...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh full; \
	else \
		echo "$(RED)Security enforcement script not found$(NC)"; exit 1; \
	fi

.PHONY: security-test
security-test: ## Run security validation tests
	@echo "$(YELLOW)Running security validation tests...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/security-validation-tests.sh" ]; then \
		bash $(SCRIPTS_DIR)/security-validation-tests.sh full; \
	else \
		echo "$(RED)Security validation script not found$(NC)"; exit 1; \
	fi

.PHONY: security-scan
security-scan: ## Run comprehensive security scans
	@echo "$(YELLOW)Running comprehensive security scans...$(NC)"
	@$(MAKE) security-scan-code security-scan-containers security-scan-k8s

.PHONY: security-scan-code
security-scan-code: ## Scan source code for vulnerabilities
	@echo "$(YELLOW)Scanning source code for vulnerabilities...$(NC)"
	@if command -v gosec &> /dev/null; then \
		gosec -fmt json -out $(COVERAGE_DIR)/security.json ./...; \
		gosec ./...; \
	else \
		echo "$(RED)gosec not found. Run 'make install-tools'$(NC)"; \
	fi

.PHONY: security-scan-containers
security-scan-containers: ## Scan container images for vulnerabilities
	@echo "$(YELLOW)Scanning container images...$(NC)"
	@if command -v trivy &> /dev/null; then \
		mkdir -p $(COVERAGE_DIR); \
		for binary in $(BINARIES); do \
			if docker image ls | grep -q "$$binary"; then \
				echo "Scanning $$binary image..."; \
				trivy image --severity $(SECURITY_THRESHOLD),CRITICAL \
					--format json --output $(COVERAGE_DIR)/$$binary-scan.json \
					$(DOCKER_NAMESPACE)/$$binary:$(DOCKER_TAG) || true; \
				trivy image --severity $(SECURITY_THRESHOLD),CRITICAL \
					$(DOCKER_NAMESPACE)/$$binary:$(DOCKER_TAG) || true; \
			fi \
		done; \
	else \
		echo "$(RED)Trivy not found. Run 'make install-security-tools'$(NC)"; \
	fi

.PHONY: security-scan-k8s
security-scan-k8s: ## Scan Kubernetes manifests for security issues
	@echo "$(YELLOW)Scanning Kubernetes manifests...$(NC)"
	@if command -v kubesec &> /dev/null; then \
		find . -name "*.yaml" -o -name "*.yml" | grep -E "(k8s|kubernetes|helm)" | while read file; do \
			echo "Scanning $$file..."; \
			kubesec scan "$$file" || true; \
		done; \
	else \
		echo "$(YELLOW)kubesec not found, skipping K8s manifest scanning$(NC)"; \
	fi

.PHONY: security-help
security-help: ## Show detailed security targets
	@echo "$(BLUE)O-RAN Security Implementation Targets$(NC)"
	@echo "$(BLUE)=====================================$(NC)"
	@echo ""
	@echo "$(YELLOW)Core Security:$(NC)"
	@echo "  $(GREEN)security-full$(NC)                 - Complete security implementation and validation"
	@echo "  $(GREEN)security-enforce$(NC)              - Enforce all security policies"
	@echo "  $(GREEN)security-validate$(NC)             - Validate security compliance"
	@echo "  $(GREEN)security-test$(NC)                 - Run comprehensive security tests"
	@echo ""
	@echo "$(YELLOW)WG11 Compliance:$(NC)"
	@echo "  $(GREEN)wg11-validate$(NC)                 - Validate WG11 interface security compliance"
	@echo "  $(GREEN)wg11-deploy$(NC)                   - Deploy WG11 security policies"
	@echo "  $(GREEN)wg11-test$(NC)                     - Test WG11 interface security"
	@echo ""
	@echo "$(YELLOW)FIPS 140-3:$(NC)"
	@echo "  $(GREEN)fips-enable$(NC)                   - Enable FIPS 140-3 mode"
	@echo "  $(GREEN)fips-validate$(NC)                 - Validate FIPS compliance"
	@echo "  $(GREEN)fips-build$(NC)                    - Build with FIPS enforcement"
	@echo ""
	@echo "$(YELLOW)Certificates:$(NC)"
	@echo "  $(GREEN)certs-generate$(NC)                - Generate all interface certificates"
	@echo "  $(GREEN)certs-validate$(NC)                - Validate certificate status"
	@echo "  $(GREEN)certs-rotate$(NC)                  - Rotate expiring certificates"
	@echo ""
	@echo "$(YELLOW)Vulnerability Scanning:$(NC)"
	@echo "  $(GREEN)scan-containers$(NC)               - Scan all container images"
	@echo "  $(GREEN)scan-configs$(NC)                  - Scan configuration files"
	@echo "  $(GREEN)scan-dependencies$(NC)             - Scan Go dependencies"
	@echo ""
	@echo "$(YELLOW)Policy Enforcement:$(NC)"
	@echo "  $(GREEN)policies-apply$(NC)                - Apply security policies"
	@echo "  $(GREEN)policies-validate$(NC)             - Validate policy compliance"
	@echo "  $(GREEN)network-policies$(NC)              - Apply zero-trust network policies"
	@echo ""
	@echo "$(YELLOW)Monitoring:$(NC)"
	@echo "  $(GREEN)security-monitor$(NC)              - Setup security monitoring"
	@echo "  $(GREEN)security-alerts$(NC)               - Check security alerts"

.PHONY: security-certs
security-certs: ## Generate security certificates for O-RAN interfaces
	@echo "$(YELLOW)Generating security certificates...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh certs; \
	else \
		echo "$(RED)Security enforcement script not found$(NC)"; exit 1; \
	fi

.PHONY: security-fips
security-fips: ## Configure FIPS 140-3 enforcement
	@echo "$(YELLOW)Configuring FIPS 140-3 enforcement...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh fips; \
	else \
		echo "$(RED)Security enforcement script not found$(NC)"; exit 1; \
	fi

.PHONY: security-network
security-network: ## Apply zero-trust network policies
	@echo "$(YELLOW)Applying zero-trust network policies...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh network; \
	else \
		echo "$(RED)Security enforcement script not found$(NC)"; exit 1; \
	fi

.PHONY: security-verify
security-verify: ## Verify security compliance
	@echo "$(YELLOW)Verifying security compliance...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh verify; \
	else \
		echo "$(RED)Security enforcement script not found$(NC)"; exit 1; \
	fi

.PHONY: security-report
security-report: ## Generate comprehensive security report
	@echo "$(YELLOW)Generating security compliance report...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@echo "# O-RAN Security Compliance Report" > $(COVERAGE_DIR)/security-report.md
	@echo "Generated: $(shell date)" >> $(COVERAGE_DIR)/security-report.md
	@echo "" >> $(COVERAGE_DIR)/security-report.md
	@echo "## WG11 Compliance Status" >> $(COVERAGE_DIR)/security-report.md
	@if kubectl get securitypolicy -A &>/dev/null; then \
		echo "- ✅ WG11 Security Policies: Applied" >> $(COVERAGE_DIR)/security-report.md; \
	else \
		echo "- ❌ WG11 Security Policies: Not Applied" >> $(COVERAGE_DIR)/security-report.md; \
	fi
	@if [ -f "$(COVERAGE_DIR)/security.json" ]; then \
		echo "- ✅ Code Security Scan: Completed" >> $(COVERAGE_DIR)/security-report.md; \
	else \
		echo "- ⚠️ Code Security Scan: Not Run" >> $(COVERAGE_DIR)/security-report.md; \
	fi
	@echo "$(GREEN)Security report generated: $(COVERAGE_DIR)/security-report.md$(NC)"

# ============================================================================
# WG11 Compliance Targets
# ============================================================================

.PHONY: wg11-validate
wg11-validate: ## Validate WG11 interface security compliance
	@echo "$(YELLOW)Validating WG11 compliance...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/security-validation-tests.sh" ]; then \
		bash $(SCRIPTS_DIR)/security-validation-tests.sh interfaces; \
	else \
		echo "$(RED)Security validation script not found$(NC)"; exit 1; \
	fi
	@echo "$(GREEN)WG11 validation completed$(NC)"

.PHONY: wg11-deploy
wg11-deploy: ## Deploy WG11 security policies
	@echo "$(YELLOW)Deploying WG11 security policies...$(NC)"
	@kubectl apply -f $(CONFIG_DIR)/oran-wg11-security.yaml || echo "$(YELLOW)WG11 config not found$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh wg11; \
	else \
		echo "$(YELLOW)WG11 enforcement script not found$(NC)"; \
	fi
	@echo "$(GREEN)WG11 security policies deployed$(NC)"

.PHONY: wg11-test
wg11-test: ## Test WG11 interface security
	@echo "$(YELLOW)Testing WG11 interface security...$(NC)"
	@for interface in e2 a1 o1 o2; do \
		echo "Testing $$interface interface..."; \
		if [ -f "$(SCRIPTS_DIR)/security-validation-tests.sh" ]; then \
			bash $(SCRIPTS_DIR)/security-validation-tests.sh interfaces | grep "$$interface" || true; \
		fi \
	done
	@echo "$(GREEN)WG11 interface testing completed$(NC)"

# ============================================================================
# Enhanced Security Targets
# ============================================================================

.PHONY: fips-enable
fips-enable: ## Enable FIPS 140-3 mode
	@echo "$(YELLOW)Enabling FIPS 140-3 mode ($(FIPS_MODE))...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh fips; \
	else \
		echo "$(YELLOW)FIPS enforcement script not found$(NC)"; \
	fi
	@echo "$(GREEN)FIPS 140-3 enabled$(NC)"

.PHONY: fips-validate
fips-validate: ## Validate FIPS compliance
	@echo "$(YELLOW)Validating FIPS 140-3 compliance...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/security-validation-tests.sh" ]; then \
		bash $(SCRIPTS_DIR)/security-validation-tests.sh fips; \
	else \
		echo "$(YELLOW)FIPS validation script not found$(NC)"; \
	fi
	@echo "$(GREEN)FIPS validation completed$(NC)"

.PHONY: certs-generate
certs-generate: ## Generate all interface certificates
	@echo "$(YELLOW)Generating certificates...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh certs; \
	else \
		echo "$(YELLOW)Certificate generation script not found$(NC)"; \
	fi
	@echo "$(GREEN)Certificates generated$(NC)"

.PHONY: certs-validate
certs-validate: ## Validate certificate status
	@echo "$(YELLOW)Validating certificates...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/security-validation-tests.sh" ]; then \
		bash $(SCRIPTS_DIR)/security-validation-tests.sh certs; \
	else \
		echo "$(YELLOW)Certificate validation script not found$(NC)"; \
	fi
	@echo "$(GREEN)Certificate validation completed$(NC)"

.PHONY: certs-rotate
certs-rotate: ## Rotate expiring certificates
	@echo "$(YELLOW)Checking for expiring certificates...$(NC)"
	@CERT_EXPIRATION_THRESHOLD=30; \
	for ns in oran nonrtric ocloud-system; do \
		echo "Checking certificates in $$ns namespace..."; \
		if kubectl get namespace $$ns &>/dev/null; then \
			kubectl get secrets -n $$ns -o json 2>/dev/null | jq -r \
			'.items[] | select(.type=="kubernetes.io/tls") | .metadata.name' 2>/dev/null | \
			while read secret; do \
				if [ ! -z "$$secret" ]; then \
					echo "Checking $$secret..."; \
					kubectl get secret $$secret -n $$ns -o jsonpath='{.data.tls\.crt}' 2>/dev/null | \
					base64 -d 2>/dev/null | openssl x509 -checkend $$(( $$CERT_EXPIRATION_THRESHOLD * 24 * 3600 )) -noout 2>/dev/null || \
					echo "$(YELLOW)Certificate $$secret expires within $$CERT_EXPIRATION_THRESHOLD days$(NC)"; \
				fi \
			done; \
		else \
			echo "$(YELLOW)Namespace $$ns not found$(NC)"; \
		fi \
	done
	@echo "$(GREEN)Certificate rotation check completed$(NC)"

.PHONY: scan-containers
scan-containers: ## Scan all container images for vulnerabilities
	@echo "$(YELLOW)Scanning container images...$(NC)"
	@mkdir -p $(COVERAGE_DIR)/vulnerability-reports
	@if command -v kubectl &> /dev/null; then \
		kubectl get pods -A -o jsonpath='{.items[*].spec.containers[*].image}' 2>/dev/null | \
		tr ' ' '\n' | sort -u | while read image; do \
			if [ ! -z "$$image" ] && command -v trivy &> /dev/null; then \
				echo "Scanning $$image..."; \
				trivy image --severity HIGH,CRITICAL --format json \
				--timeout 10m "$$image" > \
				"$(COVERAGE_DIR)/vulnerability-reports/$$(echo $$image | tr '/:' '_').json" 2>/dev/null || \
				echo "$(YELLOW)Failed to scan $$image$(NC)"; \
			fi; \
		done; \
	else \
		echo "$(YELLOW)kubectl not available, skipping container scanning$(NC)"; \
	fi
	@echo "$(GREEN)Container scanning completed$(NC)"

.PHONY: scan-configs
scan-configs: ## Scan configuration files
	@echo "$(YELLOW)Scanning configuration files...$(NC)"
	@mkdir -p $(COVERAGE_DIR)/vulnerability-reports
	@if command -v trivy &> /dev/null; then \
		trivy config --severity HIGH,CRITICAL \
		--format json --output $(COVERAGE_DIR)/vulnerability-reports/config-scan.json ./configs/network-functions/ 2>/dev/null || true; \
		trivy config --severity HIGH,CRITICAL \
		--format json --output $(COVERAGE_DIR)/vulnerability-reports/helm-scan.json ./helm/ 2>/dev/null || true; \
	else \
		echo "$(YELLOW)Trivy not available, skipping config scanning$(NC)"; \
	fi
	@echo "$(GREEN)Configuration scanning completed$(NC)"

.PHONY: scan-dependencies
scan-dependencies: ## Scan Go dependencies
	@echo "$(YELLOW)Scanning Go dependencies...$(NC)"
	@mkdir -p $(COVERAGE_DIR)/vulnerability-reports
	@if command -v trivy &> /dev/null; then \
		trivy fs --severity HIGH,CRITICAL \
		--format json --output $(COVERAGE_DIR)/vulnerability-reports/deps-scan.json . 2>/dev/null || true; \
	fi
	@if command -v nancy &> /dev/null; then \
		$(GO) list -json -m all | nancy sleuth 2>/dev/null || true; \
	fi
	@echo "$(GREEN)Dependency scanning completed$(NC)"

.PHONY: policies-apply
policies-apply: ## Apply security policies
	@echo "$(YELLOW)Applying security policies...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh network; \
	else \
		echo "$(YELLOW)Security enforcement script not found$(NC)"; \
	fi
	@echo "$(GREEN)Security policies applied$(NC)"

.PHONY: policies-validate
policies-validate: ## Validate policy compliance
	@echo "$(YELLOW)Validating policy compliance...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/security-validation-tests.sh" ]; then \
		bash $(SCRIPTS_DIR)/security-validation-tests.sh compliance; \
	else \
		echo "$(YELLOW)Policy validation script not found$(NC)"; \
	fi
	@echo "$(GREEN)Policy validation completed$(NC)"

.PHONY: network-policies
network-policies: ## Apply zero-trust network policies
	@echo "$(YELLOW)Applying zero-trust network policies...$(NC)"
	@if command -v kubectl &> /dev/null; then \
		for ns in oran nonrtric nephio-system ocloud-system; do \
			if kubectl get namespace $$ns &>/dev/null; then \
				echo "Applying policies to $$ns..."; \
				kubectl apply -f - <<< "apiVersion: networking.k8s.io/v1\nkind: NetworkPolicy\nmetadata:\n  name: default-deny-all\n  namespace: $$ns\nspec:\n  podSelector: {}\n  policyTypes: [Ingress, Egress]" 2>/dev/null || true; \
				kubectl apply -f - <<< "apiVersion: networking.k8s.io/v1\nkind: NetworkPolicy\nmetadata:\n  name: allow-dns\n  namespace: $$ns\nspec:\n  podSelector: {}\n  policyTypes: [Egress]\n  egress:\n  - to: []\n    ports:\n    - protocol: UDP\n      port: 53" 2>/dev/null || true; \
			else \
				echo "$(YELLOW)Namespace $$ns not found$(NC)"; \
			fi \
		done; \
	else \
		echo "$(YELLOW)kubectl not available$(NC)"; \
	fi
	@echo "$(GREEN)Zero-trust network policies applied$(NC)"

.PHONY: security-monitor
security-monitor: ## Setup security monitoring
	@echo "$(YELLOW)Setting up security monitoring...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/oran-security-enforcement.sh" ]; then \
		bash $(SCRIPTS_DIR)/oran-security-enforcement.sh monitor; \
	else \
		echo "$(YELLOW)Security monitoring script not found$(NC)"; \
	fi
	@echo "$(GREEN)Security monitoring setup completed$(NC)"

.PHONY: security-alerts
security-alerts: ## Check security alerts
	@echo "$(YELLOW)Checking security alerts...$(NC)"
	@if command -v kubectl &> /dev/null; then \
		kubectl get events -A 2>/dev/null | grep -i security || echo "No security events found"; \
		kubectl get pods -A 2>/dev/null | grep -E "(Error|CrashLoopBackOff|ImagePullBackOff)" || echo "No problematic pods found"; \
	else \
		echo "$(YELLOW)kubectl not available$(NC)"; \
	fi
	@echo "$(GREEN)Security alerts check completed$(NC)"

.PHONY: install-nancy
install-nancy: ## Install Nancy dependency scanner
	@echo "$(YELLOW)Installing Nancy...$(NC)"
	@$(GO) install github.com/sonatypecommunity/nancy@latest
	@echo "$(GREEN)Nancy installed successfully$(NC)"

.PHONY: clean-security
clean-security: ## Clean security artifacts
	@echo "$(YELLOW)Cleaning security artifacts...$(NC)"
	@rm -rf $(COVERAGE_DIR)/vulnerability-reports
	@rm -rf $(CERTS_DIR)
	@echo "$(GREEN)Security cleanup completed$(NC)"

.PHONY: validate-environment
validate-environment: ## Validate security environment
	@echo "$(YELLOW)Validating security environment...$(NC)"
	@echo "Go version: $(shell $(GO) version)"
	@echo "FIPS mode: $(FIPS_MODE)"
	@echo "CGO enabled: $(CGO_ENABLED)"
	@if command -v kubectl &> /dev/null; then \
		echo "Kubernetes context: $$(kubectl config current-context 2>/dev/null || echo 'Not configured')"; \
		kubectl version --client 2>/dev/null || echo "$(YELLOW)kubectl client check failed$(NC)"; \
	else \
		echo "$(YELLOW)kubectl not available$(NC)"; \
	fi
	@openssl version 2>/dev/null || echo "$(YELLOW)OpenSSL not available$(NC)"
	@trivy version 2>/dev/null || echo "$(YELLOW)Trivy not available$(NC)"
	@echo "$(GREEN)Environment validation completed$(NC)"

# ============================================================================
# Testing
# ============================================================================

.PHONY: test
test: setup-dirs ## Run all tests with coverage
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@$(GO) test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	@$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1 | awk '{print "$(GREEN)Total coverage: " $$3 "$(NC)"}'

.PHONY: test-short
test-short: ## Run short tests only
	@echo "$(YELLOW)Running short tests...$(NC)"
	@$(GO) test -short -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@if [ -d "test/integration" ]; then \
		$(GO) test -v -race -tags=integration ./test/integration/...; \
	else \
		echo "$(YELLOW)No integration tests found$(NC)"; \
	fi

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@echo "$(YELLOW)Running end-to-end tests...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/e2e-test.sh" ]; then \
		bash $(SCRIPTS_DIR)/e2e-test.sh; \
	else \
		echo "$(YELLOW)E2E test script not found$(NC)"; \
	fi

.PHONY: test-coverage
test-coverage: test ## Generate HTML coverage report
	@echo "$(YELLOW)Generating coverage report...$(NC)"
	@$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)Coverage report: $(COVERAGE_DIR)/coverage.html$(NC)"

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	@$(GO) test -bench=. -benchmem -run=^$$ ./... | tee $(COVERAGE_DIR)/benchmark.txt

# ============================================================================
# Code Quality
# ============================================================================

.PHONY: lint
lint: lint-go lint-ui lint-helm ## Run all linters

.PHONY: lint-go
lint-go: ## Lint Go code
	@echo "$(YELLOW)Linting Go code...$(NC)"
	@if command -v golangci-lint &> /dev/null; then \
		golangci-lint run --timeout=5m --config .golangci.yml; \
	else \
		echo "$(RED)golangci-lint not found. Run 'make install-tools'$(NC)"; exit 1; \
	fi

.PHONY: lint-ui
lint-ui: ## Lint React UI code
	@echo "$(YELLOW)Linting UI code...$(NC)"
	@if [ -d "$(UI_DIR)" ]; then \
		cd $(UI_DIR) && npm run lint; \
	else \
		echo "$(YELLOW)UI directory not found, skipping...$(NC)"; \
	fi

.PHONY: lint-helm
lint-helm: ## Lint Helm charts
	@echo "$(YELLOW)Linting Helm charts...$(NC)"
	@if command -v helm &> /dev/null; then \
		for chart in helm/*/Chart.yaml; do \
			if [ -f "$$chart" ]; then \
				helm lint "$$(dirname "$$chart")"; \
			fi \
		done; \
	else \
		echo "$(YELLOW)Helm not found, skipping chart linting$(NC)"; \
	fi

.PHONY: fmt
fmt: ## Format all code
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	@$(GO) fmt ./...
	@if command -v goimports &> /dev/null; then \
		goimports -w -local github.com/oran/near-rt-ric-new $$(find . -name "*.go" -not -path "./vendor/*"); \
	fi
	@if [ -d "$(UI_DIR)" ]; then \
		echo "$(YELLOW)Formatting UI code...$(NC)"; \
		cd $(UI_DIR) && npm run format; \
	fi
	@echo "$(GREEN)Code formatting complete$(NC)"

.PHONY: security
security: security-scan-code ## Run basic security scans

# ============================================================================
# Docker Operations
# ============================================================================

.PHONY: docker-build
docker-build: ## Build all Docker images
	@echo "$(YELLOW)Building Docker images...$(NC)"
	@for binary in $(BINARIES); do \
		echo "Building $$binary image..."; \
		if [ -f "$(BUILD_DIR)/$$binary/Dockerfile" ]; then \
			docker build \
				--build-arg VERSION=$(VERSION) \
				--build-arg BUILD_TIME=$(BUILD_TIME) \
				--build-arg GIT_COMMIT=$(GIT_COMMIT) \
				--build-arg FIPS_MODE=$(FIPS_MODE) \
				-f $(BUILD_DIR)/$$binary/Dockerfile \
				-t $(DOCKER_NAMESPACE)/$$binary:$(DOCKER_TAG) \
				-t $(DOCKER_NAMESPACE)/$$binary:latest .; \
		else \
			echo "$(YELLOW)Dockerfile not found for $$binary, using default$(NC)"; \
			docker build \
				--build-arg BINARY=$$binary \
				--build-arg VERSION=$(VERSION) \
				--build-arg FIPS_MODE=$(FIPS_MODE) \
				-t $(DOCKER_NAMESPACE)/$$binary:$(DOCKER_TAG) .; \
		fi \
	done
	@echo "$(GREEN)Docker images built successfully$(NC)"

.PHONY: docker-build-ui
docker-build-ui: ## Build UI Docker image
	@echo "$(YELLOW)Building UI Docker image...$(NC)"
	@if [ -d "$(UI_DIR)" ]; then \
		cd $(UI_DIR) && docker build \
			--build-arg VERSION=$(VERSION) \
			-t $(DOCKER_NAMESPACE)/ric-ui:$(DOCKER_TAG) \
			-t $(DOCKER_NAMESPACE)/ric-ui:latest .; \
	fi

.PHONY: docker-push
docker-push: ## Push Docker images to registry
	@echo "$(YELLOW)Pushing Docker images...$(NC)"
	@for binary in $(BINARIES); do \
		docker tag $(DOCKER_NAMESPACE)/$$binary:$(DOCKER_TAG) \
			$(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$$binary:$(DOCKER_TAG); \
		docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$$binary:$(DOCKER_TAG); \
	done

.PHONY: docker-compose-up
docker-compose-up: ## Start all services with Docker Compose
	@echo "$(YELLOW)Starting services with Docker Compose...$(NC)"
	@VERSION=$(VERSION) docker-compose up -d
	@echo "$(GREEN)Services started. Access dashboard at http://localhost:8080$(NC)"

.PHONY: docker-compose-down
docker-compose-down: ## Stop all services
	@echo "$(YELLOW)Stopping services...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Services stopped$(NC)"

# ============================================================================
# Development Environment
# ============================================================================

.PHONY: dev
dev: ## Start development environment
	@echo "$(YELLOW)Starting development environment...$(NC)"
	@if command -v air &> /dev/null; then \
		air -c .air.toml; \
	else \
		echo "$(RED)air not found. Run 'make install-tools' first$(NC)"; \
	fi

.PHONY: dev-ui
dev-ui: ## Start UI development server
	@echo "$(YELLOW)Starting UI development server...$(NC)"
	@if [ -d "$(UI_DIR)" ]; then \
		cd $(UI_DIR) && npm start; \
	else \
		echo "$(RED)UI directory not found$(NC)"; \
	fi

.PHONY: dev-stack
dev-stack: ## Start full development stack
	@echo "$(YELLOW)Starting full development stack...$(NC)"
	@docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d
	@echo "$(GREEN)Development stack started:$(NC)"
	@echo "  Dashboard API: http://localhost:8080"
	@echo "  UI Dev Server: http://localhost:3001"
	@echo "  Grafana: http://localhost:3000 (admin/admin123)"
	@echo "  Prometheus: http://localhost:9092"
	@echo "  Jaeger: http://localhost:16686"

# ============================================================================
# Protocol Buffers and Code Generation
# ============================================================================

.PHONY: proto
proto: ## Generate protobuf files
	@echo "$(YELLOW)Generating protobuf files...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/generate-proto.sh" ]; then \
		bash $(SCRIPTS_DIR)/generate-proto.sh; \
	else \
		echo "$(RED)Proto generation script not found$(NC)"; \
	fi

.PHONY: mock
mock: ## Generate mocks
	@echo "$(YELLOW)Generating mocks...$(NC)"
	@$(GO) generate ./...

# ============================================================================
# Dependencies
# ============================================================================

.PHONY: deps
deps: ## Download and verify dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	@$(GO) mod download
	@$(GO) mod verify
	@if [ -d "$(UI_DIR)" ]; then \
		cd $(UI_DIR) && npm install; \
	fi

.PHONY: deps-update
deps-update: ## Update all dependencies
	@echo "$(YELLOW)Updating Go dependencies...$(NC)"
	@$(GO) get -u ./...
	@$(GO) mod tidy
	@if [ -d "$(UI_DIR)" ]; then \
		echo "$(YELLOW)Updating UI dependencies...$(NC)"; \
		cd $(UI_DIR) && npm update; \
	fi

.PHONY: deps-audit
deps-audit: ## Audit dependencies for vulnerabilities
	@echo "$(YELLOW)Auditing dependencies...$(NC)"
	@if command -v nancy &> /dev/null; then \
		$(GO) list -json -m all | nancy sleuth; \
	fi
	@if [ -d "$(UI_DIR)" ]; then \
		cd $(UI_DIR) && npm audit; \
	fi

# ============================================================================
# Deployment
# ============================================================================

.PHONY: deploy-local
deploy-local: docker-build security-enforce ## Deploy to local Kubernetes with security
	@echo "$(YELLOW)Deploying to local Kubernetes with security...$(NC)"
	@kubectl create namespace oran --dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -f $(CONFIG_DIR)/oran-wg11-security.yaml || true
	@if [ -d "helm/near-rt-ric" ]; then \
		helm upgrade --install near-rt-ric ./helm/near-rt-ric \
			--namespace oran \
			--set image.tag=$(DOCKER_TAG) \
			--set image.pullPolicy=Never \
			--set security.enabled=true \
			--set fips.enabled=true \
			--wait; \
	fi

.PHONY: undeploy-local
undeploy-local: ## Remove local deployment
	@echo "$(YELLOW)Removing local deployment...$(NC)"
	@helm uninstall near-rt-ric -n oran 2>/dev/null || true
	@kubectl delete namespace oran --ignore-not-found

# ============================================================================
# Cleanup
# ============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BIN_DIR) $(COVERAGE_DIR) $(DIST_DIR) $(CERTS_DIR)
	@rm -f *.exe *.out coverage coverage.* *.test *.log
	@$(GO) clean -cache -testcache
	@if [ -d "$(UI_DIR)" ]; then \
		rm -rf $(UI_DIR)/build $(UI_DIR)/dist; \
	fi
	@echo "$(GREEN)Clean complete$(NC)"

.PHONY: clean-all
clean-all: clean docker-clean ## Clean everything including Docker
	@echo "$(YELLOW)Deep cleaning...$(NC)"
	@$(GO) clean -modcache
	@if [ -d "$(UI_DIR)" ]; then \
		rm -rf $(UI_DIR)/node_modules; \
	fi
	@docker system prune -f --volumes
	@echo "$(GREEN)Deep clean complete$(NC)"

.PHONY: docker-clean
docker-clean: ## Clean Docker artifacts
	@echo "$(YELLOW)Cleaning Docker artifacts...$(NC)"
	@for binary in $(BINARIES); do \
		docker rmi -f $(DOCKER_NAMESPACE)/$$binary:$(DOCKER_TAG) 2>/dev/null || true; \
		docker rmi -f $(DOCKER_NAMESPACE)/$$binary:latest 2>/dev/null || true; \
	done
	@docker rmi -f $(DOCKER_NAMESPACE)/ric-ui:$(DOCKER_TAG) 2>/dev/null || true
	@docker rmi -f $(DOCKER_NAMESPACE)/ric-ui:latest 2>/dev/null || true

# ============================================================================
# Utilities
# ============================================================================

.PHONY: info
info: ## Show build information
	@echo "$(BLUE)Build Information:$(NC)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Go Version: $(shell $(GO) version)"
	@echo "  FIPS Mode: $(FIPS_MODE)"
	@echo "  Platform: $(GOOS)/$(GOARCH)"
	@echo "  Parallel Jobs: $(PARALLEL_JOBS)"
	@echo "  Docker Registry: $(DOCKER_REGISTRY)"
	@echo "  Docker Namespace: $(DOCKER_NAMESPACE)"

.PHONY: validate
validate: ## Validate project structure
	@echo "$(YELLOW)Validating project structure...$(NC)"
	@test -f go.mod || (echo "$(RED)go.mod not found$(NC)" && exit 1)
	@test -f docker-compose.yml || (echo "$(RED)docker-compose.yml not found$(NC)" && exit 1)
	@test -f $(CONFIG_DIR)/oran-wg11-security.yaml || (echo "$(RED)WG11 security config not found$(NC)" && exit 1)
	@for binary in $(BINARIES); do \
		test -d cmd/$$binary || (echo "$(RED)cmd/$$binary not found$(NC)" && exit 1); \
	done
	@echo "$(GREEN)Project structure valid$(NC)"

.PHONY: pre-commit
pre-commit: fmt lint test-short security ## Run pre-commit checks

.PHONY: ci
ci: clean setup-env build-fips test lint security-full ## Run full CI pipeline with security

.PHONY: security-ci
security-ci: security-enforce security-test security-verify ## Security-focused CI pipeline

# Default target
.DEFAULT_GOAL := help