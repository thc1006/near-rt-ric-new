# Makefile for the O-RAN Near-RT RIC project

# Go parameters
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
GO_PACKAGES := $(shell go list ./...)

.PHONY: all build clean test test-coverage fmt lint helm-lint docker-build-xapp-hello-world helm-lint-xapp-hello-world setup-dev-env build-ric-charts deploy-interactive-dashboard

all: build

# Build the application
build:
	@echo "Building binaries..."
	go build -v ./cmd/...

# Build the xapp-hello-world docker image
docker-build-xapp-hello-world:
	@echo "Building xapp-hello-world docker image..."
	docker build -f build/xapp-hello-world/Dockerfile -t oran/xapp-hello-world:latest .

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests and check coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	@echo "Checking coverage..."
	@go tool cover -func=coverage.out | grep total: | awk '{print $3}'

# Format the code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint the code
lint:
	@echo "Linting code..."
	# Assuming golangci-lint is installed
	golangci-lint run

# Lint Helm charts
helm-lint:
	@echo "Linting all Helm charts..."
	helm lint helm/*

# Lint the xapp-hello-world Helm chart
helm-lint-xapp-hello-world:
	@echo "Linting xapp-hello-world Helm chart..."
	helm lint helm/xapp-hello-world

# Setup development environment
setup-dev-env:
	@echo "Setting up development environment..."
	mkdir -p temp
	cd temp && \
	wget -q --show-progress https://get.helm.sh/helm-v3.11.2-linux-amd64.tar.gz && \
	wget -q --show-progress https://get.helm.sh/chartmuseum-v0.13.1-linux-amd64.tar.gz && \
	tar -xzf helm-v3.11.2-linux-amd64.tar.gz && \
	tar -xzf chartmuseum-v0.13.1-linux-amd64.tar.gz && \
	mv linux-amd64/helm /usr/local/bin/helm && \
	mv linux-amd64/chartmuseum /usr/local/bin/chartmuseum && \
	helm plugin install https://github.com/chartmuseum/helm-push && \
	cd .. && rm -rf temp
	@echo "Starting chartmuseum..."
	mkdir -p helm/chartmuseum
	chartmuseum --debug --port=6873 --storage local --storage-local-rootdir=helm/chartmuseum &

# Build RIC charts
build-ric-charts:
	@echo "Building and pushing RIC charts..."
	helm repo add local http://localhost:6873 || true
	cd ric-dep/new-installer/helm/charts && make nearrtric

# Deploy interactive dashboard
deploy-interactive-dashboard:
	@echo "Deploying interactive dashboard..."
	helm install nearrtric -n ricplt local/nearrtric -f ric-dep/new-installer/helm-overrides/nearrtric/interactive-dashboard.yaml

# End-to-end test
e2e: deploy-interactive-dashboard
	@echo "Running end-to-end test..."
	./scripts/health-check.sh
	cd ui && npm install && npm start

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	go clean
	rm -f coverage.out
	killall chartmuseum
	helm uninstall nearrtric -n ricplt
