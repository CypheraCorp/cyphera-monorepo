#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "Starting development environment..."

# Check if .env file exists
if [ ! -f .env ]; then
  echo "Error: .env file not found. Please create one with the required environment variables."
  exit 1
fi

# Load environment variables from .env
set -a # automatically export all variables
source .env
set +a

# Function to kill background processes on exit
cleanup() {
  echo "Stopping servers..."
  # Kill any processes using ports 8080 (API) and 50051 (delegation)
  lsof -ti:8080 | xargs kill -9 2>/dev/null || true
  lsof -ti:50051 | xargs kill -9 2>/dev/null || true
  echo "Servers stopped"
}

# Set up the trap to call cleanup on script exit
trap cleanup EXIT

# Check if the delegation server is already running
if lsof -ti:50051 > /dev/null; then
  echo "Delegation server is already running on port 50051. Stopping it first..."
  lsof -ti:50051 | xargs kill -9
  # Wait a moment for the port to be freed
  sleep 1
fi

# Change to the project root directory
cd "$(dirname "$0")/.."

# Start the delegation server in the background
echo "Starting delegation server..."
cd delegation-server
npm run start:mock &
cd ..

# Wait a moment for the delegation server to start
sleep 2

# Start the API server
echo "Starting API server..."
go run ./cmd/api/main/main.go &

# Wait for both servers to be ready
echo "Development environment is running"
echo "API server: http://localhost:${PORT:-8080}"
echo "Delegation server: gRPC at localhost:${GRPC_PORT:-50051}"
echo "Press Ctrl+C to stop all servers and exit"

# Keep the script running until user interrupts
wait 