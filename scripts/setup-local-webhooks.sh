#!/bin/bash
set -eo pipefail

# ===============================================
# Local Webhook Development Setup Script
# ===============================================
# This script sets up a complete local webhook development environment
# that mirrors the production AWS Lambda + SQS architecture using Docker

echo "🚀 Setting up local webhook development environment..."

# ===============================================
# Configuration and Validation
# ===============================================

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Default values
FORCE_REBUILD=${FORCE_REBUILD:-false}
SKIP_TESTS=${SKIP_TESTS:-false}
VERBOSE=${VERBOSE:-false}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --force-rebuild)
      FORCE_REBUILD=true
      shift
      ;;
    --skip-tests)
      SKIP_TESTS=true
      shift
      ;;
    --verbose)
      VERBOSE=true
      shift
      ;;
    --help)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --force-rebuild    Force rebuild of all Docker images"
      echo "  --skip-tests       Skip running tests after setup"
      echo "  --verbose          Enable verbose logging"
      echo "  --help             Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Verbose logging function
log() {
  if [[ "$VERBOSE" == "true" ]]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
  else
    echo "$1"
  fi
}

# ===============================================
# Prerequisites Check
# ===============================================

log "🔍 Checking prerequisites..."

check_command() {
  if ! command -v "$1" &> /dev/null; then
    echo "❌ Error: $1 is not installed or not in PATH"
    echo "Please install $1 and try again"
    exit 1
  fi
}

# Check required tools
check_command docker
check_command docker-compose
check_command go
check_command make
check_command curl
check_command jq

# Check Docker is running
if ! docker info &> /dev/null; then
  echo "❌ Error: Docker daemon is not running"
  echo "Please start Docker and try again"
  exit 1
fi

# Check Go version
GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
REQUIRED_GO_VERSION="1.22"
if [[ "$(printf '%s\n' "$REQUIRED_GO_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_GO_VERSION" ]]; then
  echo "❌ Error: Go version $REQUIRED_GO_VERSION or higher required (found $GO_VERSION)"
  exit 1
fi

log "✅ Prerequisites check passed"

# ===============================================
# Environment Setup
# ===============================================

log "📝 Setting up environment files..."

# Create .env.local if it doesn't exist
if [[ ! -f ".env.local" ]]; then
  log "Creating .env.local..."
  cat > .env.local << 'EOF'
# .env.local - Local webhook development environment

# === Application Settings ===
STAGE=local
NODE_ENV=development
GIN_MODE=debug
PORT=8080

# === Database ===
DATABASE_URL=postgresql://apiuser:apipassword@localhost:5432/cyphera
DB_HOST=localhost
DB_NAME=cyphera
DB_SSLMODE=disable

# === Webhook Configuration ===
WEBHOOK_RECEIVER_PORT=3001
WEBHOOK_PROCESSOR_ENABLED=true
PAYMENT_SYNC_ENCRYPTION_KEY=local_development_key_32_characters

# === LocalStack (AWS Simulation) ===
LOCALSTACK_ENDPOINT=http://localhost:4566
SQS_QUEUE_URL=http://localhost:4566/000000000000/webhook-queue
SQS_DLQ_URL=http://localhost:4566/000000000000/webhook-dlq

# === Stripe Test Configuration ===
STRIPE_TEST_WEBHOOK_SECRET=whsec_test_local_secret
STRIPE_CLI_WEBHOOK_SECRET=whsec_stripe_cli_secret

# === CORS Settings ===
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Workspace-ID,Stripe-Signature
CORS_EXPOSED_HEADERS=
CORS_ALLOW_CREDENTIALS=true

# === Logging ===
LOG_LEVEL=debug
WEBHOOK_LOG_LEVEL=debug
EOF
else
  log ".env.local already exists, skipping creation"
fi

# Create .env.docker if it doesn't exist
if [[ ! -f ".env.docker" ]]; then
  log "Creating .env.docker..."
  cat > .env.docker << 'EOF'
# .env.docker - Docker-specific environment variables

# === Docker Network ===
COMPOSE_PROJECT_NAME=cyphera-webhooks

# === Service Ports ===
API_PORT=8080
WEBHOOK_PORT=3001
POSTGRES_PORT=5432
LOCALSTACK_PORT=4566

# === Database ===
POSTGRES_USER=apiuser
POSTGRES_PASSWORD=apipassword
POSTGRES_DB=cyphera

# === LocalStack ===
LOCALSTACK_SERVICES=sqs,secretsmanager
LOCALSTACK_DEBUG=1
EOF
else
  log ".env.docker already exists, skipping creation"
fi

# ===============================================
# Docker Configuration Files
# ===============================================

log "🐳 Creating Docker configuration files..."

# Create Dockerfile.webhook-receiver
if [[ ! -f "Dockerfile.webhook-receiver" ]]; then
  log "Creating Dockerfile.webhook-receiver..."
  cat > Dockerfile.webhook-receiver << 'EOF'
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o webhook-receiver ./apps/webhook-receiver/cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /app/webhook-receiver .

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:3001/health || exit 1

EXPOSE 3001
CMD ["./webhook-receiver"]
EOF
fi

# Create Dockerfile.webhook-processor
if [[ ! -f "Dockerfile.webhook-processor" ]]; then
  log "Creating Dockerfile.webhook-processor..."
  cat > Dockerfile.webhook-processor << 'EOF'
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o webhook-processor ./apps/webhook-processor/cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/webhook-processor .

EXPOSE 8080
CMD ["./webhook-processor"]
EOF
fi

# Create extended docker-compose.webhooks.yml
if [[ ! -f "docker-compose.webhooks.yml" ]]; then
  log "Creating docker-compose.webhooks.yml..."
  cat > docker-compose.webhooks.yml << 'EOF'
version: '3.8'

services:
  # Extend the existing API service
  cyphera-api:
    extends:
      file: docker-compose.yml
      service: cyphera-api
    environment:
      - WEBHOOK_RECEIVER_URL=http://webhook-receiver:3001
      - LOCALSTACK_ENDPOINT=http://localstack:4566
      - SQS_QUEUE_URL=http://localstack:4566/000000000000/webhook-queue
    depends_on:
      postgres:
        condition: service_healthy
      localstack:
        condition: service_started
      webhook-receiver:
        condition: service_healthy

  # PostgreSQL (from base docker-compose.yml)
  postgres:
    extends:
      file: docker-compose.yml
      service: postgres

  # LocalStack for AWS service simulation
  localstack:
    container_name: cyphera-localstack
    image: localstack/localstack:2.0
    ports:
      - "4566:4566"
    environment:
      - SERVICES=sqs,secretsmanager
      - DEBUG=1
      - DATA_DIR=/tmp/localstack/data
      - DOCKER_HOST=unix:///var/run/docker.sock
    volumes:
      - "/tmp/localstack:/tmp/localstack"
      - "/var/run/docker.sock:/var/run/docker.sock"
    networks:
      - cyphera-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4566/_localstack/health"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 10s

  # Webhook Receiver Service
  webhook-receiver:
    build:
      context: .
      dockerfile: Dockerfile.webhook-receiver
    container_name: cyphera-webhook-receiver
    ports:
      - "3001:3001"
    environment:
      - STAGE=local
      - PORT=3001
      - DATABASE_URL=postgresql://apiuser:apipassword@postgres:5432/cyphera
      - SQS_QUEUE_URL=http://localstack:4566/000000000000/webhook-queue
      - PAYMENT_SYNC_ENCRYPTION_KEY=local_development_key_32_characters
      - LOCALSTACK_ENDPOINT=http://localstack:4566
    depends_on:
      postgres:
        condition: service_healthy
      localstack:
        condition: service_healthy
    networks:
      - cyphera-network
    volumes:
      - ./logs:/app/logs
    restart: unless-stopped

  # Webhook Processor Service
  webhook-processor:
    build:
      context: .
      dockerfile: Dockerfile.webhook-processor
    container_name: cyphera-webhook-processor
    environment:
      - STAGE=local
      - DATABASE_URL=postgresql://apiuser:apipassword@postgres:5432/cyphera
      - SQS_QUEUE_URL=http://localhost:4566/000000000000/webhook-queue
      - PAYMENT_SYNC_ENCRYPTION_KEY=local_development_key_32_characters
      - LOCALSTACK_ENDPOINT=http://localhost:4566
      - AWS_ACCESS_KEY_ID=test
      - AWS_SECRET_ACCESS_KEY=test
      - AWS_DEFAULT_REGION=us-east-1
    depends_on:
      postgres:
        condition: service_healthy
      localstack:
        condition: service_healthy
    networks:
      - cyphera-network
    volumes:
      - ./logs:/app/logs
    restart: unless-stopped

networks:
  cyphera-network:
    driver: bridge

volumes:
  postgres_data:
EOF
fi

# ===============================================
# Build Components
# ===============================================

log "🔨 Building webhook components..."

# Ensure logs directory exists
mkdir -p logs

# Build Go binaries first to catch compilation errors early
log "Building Go binaries..."
make build-webhook-receiver
make build-webhook-processor

if [[ "$FORCE_REBUILD" == "true" ]]; then
  log "Force rebuilding Docker images..."
  docker-compose -f docker-compose.webhooks.yml build --no-cache
else
  log "Building Docker images..."
  docker-compose -f docker-compose.webhooks.yml build
fi

# ===============================================
# Start Services
# ===============================================

log "🚀 Starting services..."

# Create network if it doesn't exist
docker network create cyphera-network 2>/dev/null || true

# Create volumes if they don't exist
docker volume create postgres_data 2>/dev/null || true

# Stop any existing services
docker-compose -f docker-compose.webhooks.yml down 2>/dev/null || true

# Start services
docker-compose -f docker-compose.webhooks.yml up -d

# ===============================================
# Wait for Services to be Ready
# ===============================================

log "⏳ Waiting for services to be ready..."

# Function to wait for a service to be healthy
wait_for_service() {
  local service_name="$1"
  local max_attempts=30
  local attempt=1
  
  log "Waiting for $service_name to be ready..."
  
  while [[ $attempt -le $max_attempts ]]; do
    if docker-compose -f docker-compose.webhooks.yml ps "$service_name" | grep -q "healthy\|Up"; then
      log "✅ $service_name is ready"
      return 0
    fi
    
    if [[ $((attempt % 5)) -eq 0 ]]; then
      log "Still waiting for $service_name... (attempt $attempt/$max_attempts)"
    fi
    
    sleep 2
    ((attempt++))
  done
  
  log "❌ $service_name failed to become ready after $max_attempts attempts"
  return 1
}

# Wait for core services
wait_for_service "postgres"
wait_for_service "localstack"

# ===============================================
# Initialize LocalStack Resources
# ===============================================

log "🔧 Setting up LocalStack resources..."

# Create setup script if it doesn't exist
if [[ ! -f "scripts/setup-localstack.sh" ]]; then
  mkdir -p scripts
  cat > scripts/setup-localstack.sh << 'EOF'
#!/bin/bash
set -e

echo "🔧 Setting up LocalStack resources..."

# Wait for LocalStack to be ready
echo "⏳ Waiting for LocalStack to start..."
until curl -s http://localhost:4566/_localstack/health | grep -q '"sqs": "available"'; do
  sleep 2
done

# Create SQS queues
echo "📦 Creating SQS queues..."

# Main webhook queue
aws --endpoint-url=http://localhost:4566 sqs create-queue \
  --queue-name webhook-queue \
  --region us-east-1 || echo "Queue webhook-queue may already exist"

# Dead letter queue
aws --endpoint-url=http://localhost:4566 sqs create-queue \
  --queue-name webhook-dlq \
  --region us-east-1 || echo "Queue webhook-dlq may already exist"

# Create secrets for testing
echo "🔐 Creating test secrets..."

aws --endpoint-url=http://localhost:4566 secretsmanager create-secret \
  --name payment-sync-encryption-key \
  --secret-string "local_development_key_32_characters" \
  --region us-east-1 || echo "Secret may already exist"

echo "✅ LocalStack setup complete!"
EOF
  chmod +x scripts/setup-localstack.sh
fi

# Run LocalStack setup
./scripts/setup-localstack.sh

# ===============================================
# Database Migration
# ===============================================

log "🗄️ Running database migrations..."

# Wait a bit more for database to be fully ready
sleep 5

# Check if we can connect to the database
if ! docker-compose -f docker-compose.webhooks.yml exec -T postgres pg_isready -U apiuser -d cyphera; then
  log "❌ Database is not ready"
  exit 1
fi

# Run migrations if make target exists
if make -n db-migrate &>/dev/null; then
  make db-migrate
else
  log "⚠️ No db-migrate target found, skipping migrations"
fi

# ===============================================
# Wait for Webhook Services
# ===============================================

wait_for_service "webhook-receiver"

# Give webhook processor a moment to start
sleep 5

# ===============================================
# Health Checks
# ===============================================

log "🏥 Running health checks..."

# Create health check script
if [[ ! -f "scripts/health-check-local.sh" ]]; then
  cat > scripts/health-check-local.sh << 'EOF'
#!/bin/bash
set -e

echo "🔍 Checking local webhook system health..."

# Check API
echo -n "API (port 8080): "
if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check webhook receiver
echo -n "Webhook Receiver (port 3001): "
if curl -sf http://localhost:3001/health > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check database
echo -n "Database (port 5432): "
if pg_isready -h localhost -p 5432 -U apiuser > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check LocalStack
echo -n "LocalStack (port 4566): "
if curl -sf http://localhost:4566/_localstack/health > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check SQS queue
echo -n "SQS Queue: "
if AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 sqs get-queue-url --queue-name webhook-queue --region us-east-1 > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check webhook processor
echo -n "Webhook Processor: "
if docker-compose -f docker-compose.webhooks.yml ps webhook-processor | grep -q Up; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

echo "🏁 Health check complete!"
EOF
  chmod +x scripts/health-check-local.sh
fi

# Run health checks
./scripts/health-check-local.sh

# ===============================================
# Create Test Workspace
# ===============================================

log "🧪 Setting up test workspace..."

# Wait for API to be fully ready
sleep 10

# Create test workspace
log "Creating test workspace..."
WORKSPACE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Local Test Workspace",
    "account_id": "local-test-account"
  }' || echo '{"error": "failed"}')

if echo "$WORKSPACE_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
  WORKSPACE_ID=$(echo "$WORKSPACE_RESPONSE" | jq -r '.id')
  log "✅ Test workspace created: $WORKSPACE_ID"
  
  # Save workspace ID for easy reference
  echo "$WORKSPACE_ID" > .local-workspace-id
  
  # Configure test Stripe settings
  log "Configuring test payment provider..."
  curl -s -X POST "http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID/payment-configurations" \
    -H "Content-Type: application/json" \
    -d '{
      "provider_name": "stripe",
      "is_active": true,
      "is_test_mode": true,
      "configuration": {
        "api_key": "sk_test_local_key_for_testing",
        "webhook_secret": "whsec_test_local_secret",
        "environment": "test"
      }
    }' > /dev/null || log "⚠️ Failed to configure payment provider (this is normal if endpoints don't exist yet)"
else
  log "⚠️ Failed to create test workspace (this is normal if endpoints don't exist yet)"
  WORKSPACE_ID="test-workspace-id"
  echo "$WORKSPACE_ID" > .local-workspace-id
fi

# ===============================================
# Run Tests (Optional)
# ===============================================

if [[ "$SKIP_TESTS" != "true" ]]; then
  log "🧪 Running basic webhook tests..."
  
  # Test webhook receiver endpoint
  log "Testing webhook receiver..."
  WEBHOOK_TEST_RESPONSE=$(curl -s -w "%{http_code}" -X POST "http://localhost:3001/webhooks/stripe/$WORKSPACE_ID" \
    -H "Content-Type: application/json" \
    -H "Stripe-Signature: t=$(date +%s),v1=test_signature" \
    -d '{
      "id": "evt_test_setup",
      "type": "customer.created",
      "data": {
        "object": {
          "id": "cus_test_setup",
          "email": "test@example.com"
        }
      }
    }' || echo "000")
  
  if [[ "${WEBHOOK_TEST_RESPONSE: -3}" == "200" ]]; then
    log "✅ Webhook receiver test passed"
  else
    log "⚠️ Webhook receiver test failed (HTTP ${WEBHOOK_TEST_RESPONSE: -3})"
  fi
else
  log "⚠️ Skipping tests (--skip-tests flag provided)"
fi

# ===============================================
# Success Summary
# ===============================================

log ""
log "🎉 Local webhook development environment setup complete!"
log ""
log "📋 Summary:"
log "  • API Server: http://localhost:8080"
log "  • Webhook Receiver: http://localhost:3001"
log "  • Database: postgresql://apiuser:apipassword@localhost:5432/cyphera"
log "  • LocalStack: http://localhost:4566"
log "  • Test Workspace ID: $WORKSPACE_ID"
log ""
log "🔗 Quick test commands:"
log "  # Health check all services"
log "  ./scripts/health-check-local.sh"
log ""
log "  # View all logs"
log "  docker-compose -f docker-compose.webhooks.yml logs -f"
log ""
log "  # Test webhook endpoint"
log "  curl -X POST http://localhost:3001/webhooks/stripe/$WORKSPACE_ID \\"
log "    -H 'Content-Type: application/json' \\"
log "    -H 'Stripe-Signature: t=\$(date +%s),v1=test_signature' \\"
log "    -d '{\"id\":\"evt_test\",\"type\":\"customer.created\",\"data\":{\"object\":{\"id\":\"cus_test\"}}}'"
log ""
log "  # Check SQS messages"
log "  AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 sqs receive-message --queue-url http://localhost:4566/000000000000/webhook-queue --region us-east-1"
log ""
log "🔧 Additional setup scripts:"
log "  • ./scripts/test-multi-workspace.sh - Test workspace isolation"
log "  • ./scripts/debug-webhook-signatures.sh - Debug webhook signatures"
log "  • ./scripts/reset-local-webhooks.sh - Reset environment"
log ""
log "📚 See docs/local_webhook_development_guide.md for detailed usage instructions"
log ""
log "✅ Ready for webhook development!" 