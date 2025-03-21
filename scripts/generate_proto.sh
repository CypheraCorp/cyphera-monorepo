#!/bin/bash

# Install Protocol Buffers compiler if not already installed
# On macOS: brew install protobuf
# On Ubuntu: apt-get install -y protobuf-compiler

# Install Go plugins for Protocol Buffers
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure the proto directory exists
mkdir -p internal/proto

# Generate Go code from proto definition
echo "Generating Go gRPC client code..."
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  internal/proto/delegation.proto

echo "Proto generation complete!" 