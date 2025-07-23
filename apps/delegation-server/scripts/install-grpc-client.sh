#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to check for protoc installation
check_protoc() {
  if ! command -v protoc &> /dev/null; then
    echo "Error: protoc (Protocol Buffers Compiler) is not installed."
    echo "Please install Protocol Buffers:"
    echo "  - On macOS: brew install protobuf"
    echo "  - On Ubuntu: apt-get install protobuf-compiler"
    exit 1
  fi
  
  echo "✓ Protocol Buffers compiler (protoc) is installed"
}

# Check for protoc installation
check_protoc

# Get the absolute path of the project root
# This script is now in js-server/scripts, so we need to go up two levels
PROJECT_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
echo "Project root: $PROJECT_ROOT"

# Check if we're running from the js-server/scripts directory
if [[ "$(basename "$(pwd)")" == "scripts" && "$(basename "$(dirname "$(pwd)")")" == "js-server" ]]; then
  echo "Running from js-server/scripts directory"
  cd ../..
elif [[ "$(basename "$(pwd)")" == "js-server" ]]; then
  echo "Running from js-server directory"
  cd ..
else
  echo "Running from project root directory"
fi

# Install required Go modules for gRPC and protobuf
echo "Installing required Go modules..."
go get -u google.golang.org/protobuf/cmd/protoc-gen-go
go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Create the internal/proto directory if it doesn't exist
echo "Creating internal/proto directory..."
mkdir -p internal/proto

# Copy the proto file from js-server
echo "Copying proto file from js-server/src/proto/delegation.proto to internal/proto/..."
cp js-server/src/proto/delegation.proto internal/proto/

# Generate Go gRPC client code from proto
echo "Generating Go gRPC client code..."
protoc --go_out=. \
       --go_opt=paths=source_relative \
       --go-grpc_out=. \
       --go-grpc_opt=paths=source_relative \
       internal/proto/delegation.proto

echo "✅ Go gRPC client code successfully generated!"
echo "Generated files:"
ls -la internal/proto/

echo ""
echo "To use the gRPC client in your Go code:"
echo "import \"cyphera-api/internal/proto\""
echo "" 