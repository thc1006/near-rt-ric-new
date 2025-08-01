#!/bin/bash

# Script to generate Go protobuf stubs from .proto files
# This script should be run from the project root directory

set -e

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "protoc is not installed. Please install Protocol Buffers compiler."
    echo "On Ubuntu/Debian: sudo apt-get install protobuf-compiler"
    echo "On macOS: brew install protobuf"
    echo "On Windows: Download from https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Create output directories
mkdir -p api/proto/e2mgr
mkdir -p api/proto/submgr
mkdir -p api/proto/rtmgr

echo "Generating Go stubs for E2 Manager..."
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/proto/e2mgr/e2mgr.proto

echo "Generating Go stubs for Subscription Manager..."
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/proto/submgr/submgr.proto

echo "Generating Go stubs for Routing Manager..."
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/proto/rtmgr/rtmgr.proto

echo "Protobuf generation completed successfully!"