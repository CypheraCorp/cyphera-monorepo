#!/bin/bash

# Nx-based development environment startup script
# This script starts all services using Nx commands

set -e

echo "🚀 Starting Cyphera Development Environment with Nx..."
echo ""

# Change to the project root directory
cd "$(dirname "$0")/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to wait for a port to be available
wait_for_port() {
    local port=$1
    local service=$2
    local max_attempts=30
    local attempt=0
    
    echo -e "${YELLOW}Waiting for $service on port $port...${NC}"
    while ! nc -z localhost $port >/dev/null 2>&1; do
        attempt=$((attempt + 1))
        if [ $attempt -eq $max_attempts ]; then
            echo -e "${RED}Error: $service failed to start on port $port${NC}"
            return 1
        fi
        sleep 1
    done
    echo -e "${GREEN}✓ $service is ready on port $port${NC}"
}

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command_exists docker; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

if ! command_exists npx; then
    echo -e "${RED}Error: npx is not installed${NC}"
    exit 1
fi

if ! command_exists go; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ All prerequisites satisfied${NC}"
echo ""

# Install dependencies if needed
echo "📦 Checking Go dependencies..."
if [ ! -d "apps/api/vendor" ] && [ ! -f "apps/api/go.sum" ]; then
    echo -e "${YELLOW}Installing Go dependencies...${NC}"
    npx nx run-many --target=install --projects=api,go-shared,subscription-processor,webhook-receiver,webhook-processor,dlq-processor --parallel=false
    go work sync
    echo -e "${GREEN}✓ Go dependencies installed${NC}"
fi

# Check environment files
echo "📄 Checking environment files..."

if [ ! -f .env ]; then
    echo -e "${YELLOW}Warning: .env file not found. Copying from template...${NC}"
    cp .env.template .env
    echo -e "${YELLOW}Please update .env with your configuration${NC}"
fi

if [ ! -f apps/delegation-server/.env ]; then
    echo -e "${YELLOW}Warning: apps/delegation-server/.env not found. Copying from template...${NC}"
    if [ -f apps/delegation-server/.env.example ]; then
        cp apps/delegation-server/.env.example apps/delegation-server/.env
        echo -e "${YELLOW}Please update apps/delegation-server/.env with your configuration${NC}"
    fi
fi

if [ ! -f apps/web-app/.env.local ]; then
    echo -e "${YELLOW}Warning: apps/web-app/.env.local not found.${NC}"
    echo -e "${YELLOW}The web app may use default values.${NC}"
fi

echo ""

# Start PostgreSQL
echo "🐘 Starting PostgreSQL..."
if docker-compose ps | grep -q "postgres.*Up"; then
    echo -e "${GREEN}✓ PostgreSQL is already running${NC}"
else
    docker-compose up -d postgres
    wait_for_port 5432 "PostgreSQL"
fi
echo ""

# Clean up any existing processes
echo "🧹 Cleaning up existing processes..."
# Kill any existing processes on our ports
lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
lsof -ti:3000 | xargs -r kill -9 2>/dev/null || true
lsof -ti:50051 | xargs -r kill -9 2>/dev/null || true
echo -e "${GREEN}✓ Cleanup complete${NC}"
echo ""

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Shutting down services..."
    
    # Kill the background processes
    [ ! -z "$DELEGATION_PID" ] && kill $DELEGATION_PID 2>/dev/null || true
    [ ! -z "$API_PID" ] && kill $API_PID 2>/dev/null || true
    [ ! -z "$WEB_PID" ] && kill $WEB_PID 2>/dev/null || true
    
    # Give them time to shutdown gracefully
    sleep 2
    
    # Force kill if still running
    lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
    lsof -ti:3000 | xargs -r kill -9 2>/dev/null || true
    lsof -ti:50051 | xargs -r kill -9 2>/dev/null || true
    
    echo -e "${GREEN}✓ All services stopped${NC}"
    exit 0
}

# Set up trap for cleanup on script exit
trap cleanup INT TERM EXIT

# Start services
echo "🚀 Starting services..."
echo ""

# Start Delegation Server
echo -e "${YELLOW}Starting Delegation Server (gRPC on port 50051)...${NC}"
npx nx serve delegation-server &
DELEGATION_PID=$!
sleep 2

# Start API Server
echo -e "${YELLOW}Starting API Server (HTTP on port 8080)...${NC}"
npx nx serve api &
API_PID=$!
sleep 2

# Start Web App
echo -e "${YELLOW}Starting Web App (HTTP on port 3000)...${NC}"
npx nx serve web-app &
WEB_PID=$!

# Wait for all services to be ready
echo ""
echo "⏳ Waiting for all services to start..."
wait_for_port 50051 "Delegation Server"
wait_for_port 8080 "API Server"
wait_for_port 3000 "Web App"

echo ""
echo "✅ All services are running!"
echo ""
echo "📍 Service URLs:"
echo "   - Web App:        http://localhost:3000"
echo "   - API Server:     http://localhost:8080"
echo "   - API Docs:       http://localhost:8080/swagger/index.html"
echo "   - gRPC Server:    localhost:50051"
echo "   - PostgreSQL:     localhost:5432"
echo ""
echo "📊 View logs in real-time in this terminal"
echo "🛑 Press Ctrl+C to stop all services"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Keep the script running and show logs
wait