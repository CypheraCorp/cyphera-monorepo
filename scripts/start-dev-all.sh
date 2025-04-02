#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "Starting development environment..."

# Change to the project root directory
cd "$(dirname "$0")/.."

# --- Load Root Environment Variables Early ---
echo "Checking for root .env file..."
if [ ! -f .env ]; then
  echo "Error: Root .env file not found. Cannot proceed without DATABASE_URL."
  exit 1
else
  echo "Loading root .env file..."
  set -a        # Automatically export all variables
  source .env
  set +a        # Stop automatically exporting variables
fi

# --- Check DATABASE_URL ---
if [ -z "$DATABASE_URL" ]; then
  echo "Error: DATABASE_URL is not set in the .env file."
  exit 1
fi
echo "DATABASE_URL found."

# --- Check Docker Availability ---
echo "Checking if Docker command is available..."
if ! command -v docker &> /dev/null; then
    echo "Error: 'docker' command not found. Docker is required to potentially start the database container."
    exit 1
fi
echo "Docker command found."

# --- Check PostgreSQL Client Availability ---
echo "Checking if pg_isready command is available..."
if ! command -v pg_isready &> /dev/null; then
    echo "Error: 'pg_isready' command not found. Please install PostgreSQL client utilities."
    exit 1
fi
echo "pg_isready command found."

# --- Check PostgreSQL Connection & Attempt Start ---
echo "Checking PostgreSQL connection (attempt 1)..."
if ! pg_isready -d "$DATABASE_URL" -t 5; then # Short timeout for first check
  echo "PostgreSQL connection failed (attempt 1). Attempting to start container via docker-compose..."
  
  # Check if docker-compose command exists
  if ! command -v docker-compose &> /dev/null; then
      echo "Error: 'docker-compose' command not found. Cannot start the database container."
      exit 1
  fi

  # Check if docker daemon is running
  if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker daemon is not running. Please start Docker."
    exit 1
  fi

  # Attempt to start the postgres service in detached mode
  if ! docker-compose up -d postgres; then
    echo "Error: 'docker-compose up -d postgres' failed. Please check docker-compose setup and logs."
    exit 1
  fi

  echo "'docker-compose up -d postgres' executed. Waiting for container to initialize..."
  sleep 5 # Wait a few seconds for PostgreSQL to start within the container

  echo "Checking PostgreSQL connection (attempt 2)..."
  if ! pg_isready -d "$DATABASE_URL" -t 10; then # Longer timeout for second check
    echo "Error: Failed to connect to PostgreSQL at $DATABASE_URL even after attempting to start the container."
    exit 1
  fi
fi
echo "PostgreSQL connection successful."

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

chmod +x scripts/run.sh

# Run with environment variables
./scripts/run.sh &

# Clean up the temporary script (but not immediately, to ensure it's available for the background process)
sleep 1

cd ..

# Wait a moment for the delegation server to start
sleep 2

# Start the API server with hot reloading via air
echo "Starting API server with hot reloading via air..." 
# Load root .env variables
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

# Start the subscription processor in the background
echo "Starting subscription processor with 1-minute interval..."
go run ./cmd/subscription-processor/main.go --interval=1m &

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