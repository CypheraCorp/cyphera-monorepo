#!/bin/bash

# Script to run the delegation integration test
# This script will:
# 1. Run unit tests for the integration client
# 2. Start the Node.js gRPC delegation server
# 3. Run the Go integration test client
# 4. Shut down the Node.js server

set -e # Exit on any error

# Parse arguments
MODE="cli"
MOCK="true"
SERVER_PORT="8000"
GRPC_PORT="50051"

print_usage() {
  echo "Usage: $0 [OPTIONS]"
  echo "Options:"
  echo "  --server            Run in HTTP server mode"
  echo "  --cli               Run in CLI mode (default)"
  echo "  --mock              Use mock server (default)"
  echo "  --live              Use live server"
  echo "  --port PORT         Specify HTTP server port (default: 8000)"
  echo "  --grpc-port PORT    Specify gRPC server port (default: 50051)"
  echo "  --help              Show this help message"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server)
      MODE="server"
      shift
      ;;
    --cli)
      MODE="cli"
      shift
      ;;
    --mock)
      MOCK="true" 
      shift
      ;;
    --live)
      MOCK="false"
      shift
      ;;
    --port)
      SERVER_PORT="$2"
      shift 2
      ;;
    --grpc-port)
      GRPC_PORT="$2"
      shift 2
      ;;
    --help)
      print_usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      print_usage
      exit 1
      ;;
  esac
done

# Root directory of the project
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
echo "Project root: $ROOT_DIR"

# Directory for Node.js delegation server
DELEGATION_SERVER_DIR="$ROOT_DIR/delegation-server"
echo "Delegation server directory: $DELEGATION_SERVER_DIR"

# Function to check if port is in use
check_port() {
  local port=$1
  if command -v lsof >/dev/null; then
    if lsof -i:"$port" >/dev/null 2>&1; then
      return 0  # port is in use
    fi
  elif command -v netstat >/dev/null; then
    if netstat -tuln | grep ":$port " >/dev/null 2>&1; then
      return 0  # port is in use
    fi
  fi
  return 1  # port is free
}

# Function to kill process using port
kill_process_on_port() {
  local port=$1
  echo "Port $port is in use. Attempting to free it..."
  
  if command -v lsof >/dev/null; then
    local pid=$(lsof -t -i:"$port")
    if [ -n "$pid" ]; then
      echo "Killing process $pid using port $port"
      kill -9 "$pid" 2>/dev/null || true
      sleep 1
    fi
  elif command -v netstat >/dev/null && command -v grep >/dev/null && command -v awk >/dev/null; then
    local pid=$(netstat -tuln | grep ":$port " | awk '{print $7}' | cut -d'/' -f1)
    if [ -n "$pid" ]; then
      echo "Killing process $pid using port $port"
      kill -9 "$pid" 2>/dev/null || true
      sleep 1
    fi
  else
    echo "Warning: Cannot determine process using port $port. Please free it manually."
    exit 1
  fi
  
  if check_port "$port"; then
    echo "Failed to free port $port. Please terminate the process manually."
    exit 1
  fi
  
  echo "Port $port is now free"
}

# Function to cleanup on exit
cleanup() {
  echo "Cleaning up..."
  if [ ! -z "$SERVER_PID" ]; then
    echo "Stopping Node.js server (PID: $SERVER_PID)..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
  fi
  
  if [ ! -z "$HTTP_SERVER_PID" ]; then
    echo "Stopping HTTP server (PID: $HTTP_SERVER_PID)..."
    kill $HTTP_SERVER_PID 2>/dev/null || true
    wait $HTTP_SERVER_PID 2>/dev/null || true
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

if [ "$MOCK" = "true" ]; then
  # Check if Node.js and npm are installed
  if ! command -v node >/dev/null || ! command -v npm >/dev/null; then
    echo "Error: Node.js and npm are required to run this test"
    exit 1
  fi

  # Check if the gRPC port is in use and free it if necessary
  if check_port "$GRPC_PORT"; then
    kill_process_on_port "$GRPC_PORT"
  fi

  # Start the Node.js gRPC server
  echo "Starting Node.js gRPC delegation server in mock mode..."
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

  # Install dependencies if needed
  if [ ! -d "node_modules" ]; then
    echo "Installing Node.js dependencies..."
    npm install
  fi

  # Start the server in mock mode (for testing)
  echo "Starting server in background..."
  npm run start:mock > server.log 2>&1 &
  SERVER_PID=$!

  # Wait for server to start
  echo "Waiting for server to start up (PID: $SERVER_PID)..."
  # Give the server a few seconds to fully bind the port
  echo "Adding a 5-second delay for server binding..."
  sleep 5

  # Verify server is running
  if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "Error: Server failed to start"
    cat server.log
    exit 1
  fi

  echo "Server is running!"

  # Print server log for debugging
  echo "--- Server Log Start ---"
  cat server.log || echo "server.log not found or empty"
  echo "--- Server Log End ---"

  # Check if the port is actually listening
  echo "Checking listener on port ${GRPC_PORT}..."
  if ss -tlpn | grep -q ":${GRPC_PORT}\s"; then
    echo "Port ${GRPC_PORT} is listening."
  else
    echo "ERROR: Port ${GRPC_PORT} is NOT listening."
    # Optionally exit here if listening is critical before proceeding
    # exit 1 
  fi

else
  echo "Using live gRPC delegation server..."
  # Set env variable for the client to connect to the correct server
  export DELEGATION_GRPC_ADDR=$(grep GRPC_HOST "$DELEGATION_SERVER_DIR/.env" | cut -d '=' -f2):$(grep GRPC_PORT "$DELEGATION_SERVER_DIR/.env" | cut -d '=' -f2)
  echo "Using server at $DELEGATION_GRPC_ADDR"
fi

# Check if the HTTP port is in use (for server mode) and free it if necessary
if [ "$MODE" = "server" ] && check_port "$SERVER_PORT"; then
  kill_process_on_port "$SERVER_PORT"
fi

# Run the test based on the selected mode
cd "$ROOT_DIR"

if [ "$MODE" = "server" ]; then
  echo "Running in HTTP server mode..."
  ./cmd/delegation-integration-test/integration-test -server -port "$SERVER_PORT" &
  HTTP_SERVER_PID=$!
  
  # Wait for server to start
  echo "Waiting for HTTP server to start (PID: $HTTP_SERVER_PID)..."
  sleep 2
  
  # Verify server is running
  if ! kill -0 $HTTP_SERVER_PID 2>/dev/null; then
    echo "Error: HTTP Server failed to start"
    exit 1
  fi
  
  echo "HTTP Server is running on port $SERVER_PORT"
  echo "You can now make HTTP requests to http://localhost:$SERVER_PORT/api/delegations/redeem"
  echo "Example: curl -X POST -H 'Content-Type: application/json' -d '{\"delegationData\": \"{...}\"}' http://localhost:$SERVER_PORT/api/delegations/redeem"
  echo ""
  echo "Press Ctrl+C to stop the servers"
  
  # Keep running until interrupted
  wait $HTTP_SERVER_PID
else
  echo "Running in CLI mode..."
  # Also set the DELEGATION_GRPC_ADDR for the client to use the correct port (forcing IPv4)
  export DELEGATION_GRPC_ADDR="127.0.0.1:${GRPC_PORT}"
  ./cmd/delegation-integration-test/integration-test -verbose
  
  # Print test result
  if [ $? -eq 0 ]; then
    echo "Integration test passed successfully!"
  else
    echo "Integration test failed"
    exit 1
  fi
fi

echo "All tests completed successfully!"
exit 0 