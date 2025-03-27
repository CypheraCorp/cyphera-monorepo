#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "Starting development environment..."

# Change to the project root directory
cd "$(dirname "$0")/.."

# Check if root .env file exists for the API server
if [ ! -f .env ]; then
  echo "Warning: Root .env file not found. The API server may not function correctly."
else
  echo "Root .env file found for API server."
fi

# Function to kill background processes on exit
cleanup() {
  echo "Stopping servers..."
  # Kill any processes using ports 8000 (API) and 50051 (delegation)
  lsof -ti:8000 | xargs kill -9 2>/dev/null || true
  lsof -ti:50051 | xargs kill -9 2>/dev/null || true
  # Kill the subscription processor
  pkill -f "subscription-processor" 2>/dev/null || true
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

# Start the delegation server in the background with environment variables
echo "Starting delegation server..."
cd delegation-server

# Check if delegation server .env file exists
if [ ! -f .env ]; then
  echo "Warning: delegation-server/.env file not found. The delegation server may not function correctly."
else 
  echo "Delegation server .env file found."
fi

# Create a temporary env script to ensure all variables are passed
cat > run_with_env.sh << 'EOF'
#!/bin/bash
# Load variables from delegation-server/.env
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi
# Log environment variables for debugging (with sensitive info redacted)
echo "Environment variables being used by delegation server:"
echo "MOCK_MODE=${MOCK_MODE}"
echo "GRPC_HOST=${GRPC_HOST}"
echo "GRPC_PORT=${GRPC_PORT}"
echo "CHAIN_ID=${CHAIN_ID}"
[ -n "${PRIVATE_KEY}" ] && echo "PRIVATE_KEY=[REDACTED]" || echo "PRIVATE_KEY=not set"
echo "LOG_LEVEL=${LOG_LEVEL}"
npm run dev
EOF

chmod +x run_with_env.sh

# Run with environment variables
./run_with_env.sh &

# Clean up the temporary script (but not immediately, to ensure it's available for the background process)
sleep 1

cd ..

# Wait a moment for the delegation server to start
sleep 2

# Start the subscription processor in the background
echo "Starting subscription processor with 1-minute interval..."
go run ./cmd/subscription-processor/main.go --interval=1m &

# Start the API server with hot reloading via air
echo "Starting API server with hot reloading via air..." 
# Load root .env variables
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

# Check if air is installed
if ! command -v air &> /dev/null; then
  echo "Error: 'air' is not installed. Installing it now..."
  go install github.com/air-verse/air@latest
  if [ $? -ne 0 ]; then
    echo "Failed to install air. Falling back to standard Go run."
    go run ./cmd/api/local/main.go &
  else
    air &
  fi
else
  air &
fi

# Wait for all servers to be ready
echo "Development environment is running"
echo "API server: http://localhost:${PORT:-8000} (with hot reloading enabled)"
echo "Delegation server: gRPC at localhost:${GRPC_PORT:-50051}"
echo "Subscription processor: Running with 1-minute interval"
echo "Press Ctrl+C to stop all servers and exit"

# Keep the script running until user interrupts
wait 