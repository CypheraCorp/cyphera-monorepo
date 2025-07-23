#!/bin/bash

# Exit on any error
set -e

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Print section header
section() {
  echo -e "\n${BOLD}${YELLOW}=== $1 ===${NC}\n"
}

# Print success message
success() {
  echo -e "${GREEN}✅ $1${NC}"
}

# Print error message and exit
error() {
  echo -e "${RED}❌ $1${NC}"
  exit 1
}

# Check if we're in the js-server directory
if [[ "$(basename "$(pwd)")" != "js-server" ]]; then
  error "This script must be run from the js-server directory"
fi

# Clean and build
section "Cleaning and installing dependencies"
./install.sh --clean || error "Failed to clean and install dependencies"
success "Dependencies installed"

# Build the project
section "Building the project"
npm run build || error "Failed to build the project"
success "Project built successfully"

# Update proto client
section "Updating gRPC client for Go"
./install-grpc-client.sh || error "Failed to update Go gRPC client"
success "Go gRPC client updated"

# Start server in background for testing
section "Starting server for testing"
echo "Starting gRPC server in the background..."
node dist/index.js &
SERVER_PID=$!

# Wait for server to start
echo "Waiting for server to start up..."
sleep 3

# Run tests
section "Running gRPC test"
echo "Testing connection to gRPC server..."
node scripts/test-grpc.js
TEST_EXIT_CODE=$?

# Kill server
echo "Shutting down gRPC server..."
kill $SERVER_PID

# Check test results
if [ $TEST_EXIT_CODE -eq 0 ]; then
  success "Tests completed successfully"
else
  error "Tests failed with exit code $TEST_EXIT_CODE"
fi

section "All done!"
echo "The delegation redemption gRPC server has been built and tested successfully."
echo "You can start the server with:"
echo "  npm start      # Production mode"
echo "  npm run dev    # Development mode"
echo "Or use the convenience script:"
echo "  ./run.sh" 