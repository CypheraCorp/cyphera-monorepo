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

# Function to check if a port is listening (portable across OS)
check_port_listening() {
  local port=$1
  
  # Try different commands based on what's available
  if command -v lsof >/dev/null; then
    # macOS and some Linux systems
    lsof -i ":$port" >/dev/null 2>&1
  elif command -v ss >/dev/null; then
    # Modern Linux systems
    ss -tlpn | grep -q ":$port\s"
  elif command -v netstat >/dev/null; then
    # Fallback for older systems
    netstat -tlpn 2>/dev/null | grep -q ":$port\s" || netstat -an 2>/dev/null | grep -q ":$port"
  else
    echo "Warning: No suitable command found to check port status"
    return 1
  fi
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
  
  if check_port_listening "$port"; then
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
  if check_port_listening "$GRPC_PORT"; then
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

  # Wait actively for server to start listening
  echo "Waiting for server (PID: $SERVER_PID) to listen on port ${GRPC_PORT}..."
  MAX_WAIT=30 # Maximum wait time in seconds
  WAIT_INTERVAL=1 # Check interval in seconds
  ELAPSED_WAIT=0
  LISTENING=false
  while [ $ELAPSED_WAIT -lt $MAX_WAIT ]; do
    if check_port_listening "${GRPC_PORT}"; then
      echo "Server is listening on port ${GRPC_PORT} after ${ELAPSED_WAIT} seconds."
      LISTENING=true
      break
    fi
    sleep $WAIT_INTERVAL
    ELAPSED_WAIT=$((ELAPSED_WAIT + WAIT_INTERVAL))
    echo "Waited ${ELAPSED_WAIT}s..."
  done

  if [ "$LISTENING" = false ]; then
    echo "ERROR: Server did not start listening on port ${GRPC_PORT} within ${MAX_WAIT} seconds."
    echo "--- Last known server log --- "
    cat server.log || echo "server.log not found or empty"
    echo "--- End of log --- "
    exit 1
  fi

  # Verify server process still exists (sanity check)
  if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "Error: Server process disappeared after starting to listen."
    exit 1
  fi

  echo "Server is running!"

  # Final check if the port is actually listening (redundant after loop, but safe)
  echo "Final check listener on port ${GRPC_PORT}..."
  if check_port_listening "${GRPC_PORT}"; then
    echo "Port ${GRPC_PORT} is confirmed listening."
  else
    # This should ideally not be reached if the loop worked
    echo "ERROR: Port ${GRPC_PORT} is NOT listening despite earlier check."
    exit 1 
  fi

else
  echo "Using live gRPC delegation server..."
  # Set env variable for the client to connect to the correct server
  export DELEGATION_GRPC_ADDR=$(grep GRPC_HOST "$DELEGATION_SERVER_DIR/.env" | cut -d '=' -f2):$(grep GRPC_PORT "$DELEGATION_SERVER_DIR/.env" | cut -d '=' -f2)
  echo "Using server at $DELEGATION_GRPC_ADDR"
fi

# Check if the HTTP port is in use (for server mode) and free it if necessary
if [ "$MODE" = "server" ] && check_port_listening "$SERVER_PORT"; then
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