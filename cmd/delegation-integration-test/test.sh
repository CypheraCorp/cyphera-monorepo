#!/bin/bash

# Script to run the delegation integration test
# This script will:
# 1. Run unit tests for the integration client
# 2. Start the Node.js gRPC delegation server
# 3. Run the Go integration test client
# 4. Shut down the Node.js server

set -e # Exit on any error

# Root directory of the project
ROOT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
echo "Project root: $ROOT_DIR"

# Directory for Node.js delegation server
DELEGATION_SERVER_DIR="$ROOT_DIR/delegation-server"
echo "Delegation server directory: $DELEGATION_SERVER_DIR"

# Function to cleanup on exit
cleanup() {
  echo "Cleaning up..."
  if [ ! -z "$SERVER_PID" ]; then
    echo "Stopping Node.js server (PID: $SERVER_PID)..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
  fi
}

# Register cleanup function
trap cleanup EXIT INT TERM

# Check if the Node.js server exists
if [ ! -d "$DELEGATION_SERVER_DIR" ]; then
  echo "Error: Delegation server directory not found at $DELEGATION_SERVER_DIR"
  exit 1
fi

# Run unit tests first
echo "Running unit tests..."
cd "$ROOT_DIR/cmd/delegation-integration-test"
go test -v

# Build the Go integration test client
echo "Building integration test client..."
cd "$ROOT_DIR"
go build -o cmd/delegation-integration-test/integration-test cmd/delegation-integration-test/main.go

# Check if Node.js and npm are installed
if ! command -v node >/dev/null || ! command -v npm >/dev/null; then
  echo "Error: Node.js and npm are required to run this test"
  exit 1
fi

# Start the Node.js gRPC server
echo "Starting Node.js gRPC delegation server..."
cd "$DELEGATION_SERVER_DIR"

# Check if .env file exists, if not create it from .env.example
if [ ! -f ".env" ]; then
  echo "Creating .env file from .env.example..."
  if [ -f ".env.example" ]; then
    cp .env.example .env
  else
    echo "Error: .env.example not found. Please create a .env file manually."
    exit 1
  fi
fi

# Mock configuration for testing
echo "Adding mock configuration to .env file..."
cat > .env <<EOL
GRPC_PORT=50051
GRPC_HOST=0.0.0.0
RPC_URL=https://sepolia.infura.io/v3/your-infura-key
BUNDLER_URL=https://sepolia.infura.io/v3/your-infura-key
PAYMASTER_URL=
CHAIN_ID=11155111
# This is a dummy private key for testing only, do not use in production
PRIVATE_KEY=0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
LOG_LEVEL=debug
EOL

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
  echo "Installing Node.js dependencies..."
  npm install
fi

# Start the server in mock mode (for testing)
echo "Starting server in mock mode..."
npm run start:mock > server.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
echo "Waiting for server to start up (PID: $SERVER_PID)..."
sleep 5

# Verify server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
  echo "Error: Server failed to start"
  cat server.log
  exit 1
fi

echo "Server is running!"

# Run the integration test
echo "Running integration test with actual gRPC server..."
cd "$ROOT_DIR"
./cmd/delegation-integration-test/integration-test -verbose

# Print test result
if [ $? -eq 0 ]; then
  echo "Integration test passed successfully!"
else
  echo "Integration test failed"
  exit 1
fi

echo "All tests completed successfully!"
exit 0 