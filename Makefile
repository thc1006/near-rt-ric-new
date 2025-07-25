# Makefile for the O-RAN Near-RT RIC project

# Go parameters
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
GO_PACKAGES := $(shell go list ./...)

.PHONY: all build clean test test-coverage fmt lint helm-lint

all: build

# Build the application
build:
	@echo "Building binaries..."
	go build -v ./...

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests and check coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	@echo "Checking coverage..."
	@go tool cover -func=coverage.out | grep total: | awk '{print $$3}'

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
	@echo "Linting Helm charts..."
	# Assuming helm is installed
	helm lint helm/

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	go clean
	rm -f coverage.out
