# Local Webhook Development Guide

**Version:** 1.0  
**Date:** January 2025  
**Purpose:** Complete guide for local webhook development and testing using Docker

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Architecture](#architecture)
5. [Environment Setup](#environment-setup)
6. [Local Services](#local-services)
7. [Webhook Testing](#webhook-testing)
8. [Debugging](#debugging)
9. [Troubleshooting](#troubleshooting)

## Overview

This guide walks you through setting up a complete local development environment for webhook processing that mimics the AWS Lambda-based production architecture using Docker containers.

### What You'll Get

- **Local API Server** - Your main Cyphera API running in Docker
- **Webhook Receiver** - HTTP endpoint that receives webhooks locally
- **Webhook Processor** - Background service that processes queued webhooks
- **Local SQS** - Message queue simulation using LocalStack
- **PostgreSQL Database** - Local database with test data
- **Stripe CLI** - For testing Stripe webhooks locally

## Prerequisites

### Required Software

```bash
# Docker and Docker Compose
docker --version  # >= 20.0
docker-compose --version  # >= 2.0

# Go (for building binaries)
go version  # >= 1.22

# Make (for build automation)
make --version

# Optional: Stripe CLI for webhook testing
stripe version
```

### Required Accounts

- **Stripe Test Account** - For webhook testing
- **Ngrok Account** (optional) - For external webhook testing

## Quick Start

### 1. One-Command Setup

```bash
# Clone, build, and start everything
make local-webhooks-up
```

This single command will:
- Build all webhook components
- Start Docker services
- Initialize the database
- Set up test workspaces
- Display connection URLs and test commands

### 2. Verify Setup

```bash
# Check all services are running
docker-compose -f docker-compose.webhooks.yml ps

# Test API health
curl http://localhost:8080/health

# Test webhook receiver health
curl http://localhost:3001/health

# View logs
make logs-webhooks
```

## Architecture

### Local Development Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Stripe CLI    │───▶│ Webhook         │───▶│    LocalStack   │
│   (Test Events) │    │ Receiver        │    │   (SQS Queue)   │
└─────────────────┘    │ :3001           │    └─────────────────┘
                       └─────────────────┘             │
                                                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Your App      │◀───│ Cyphera API     │◀───│ Webhook         │
│   Frontend      │    │ :8080           │    │ Processor       │
└─────────────────┘    └─────────────────┘    │ (Background)    │
                                │              └─────────────────┘
                                ▼
                       ┌─────────────────┐
                       │   PostgreSQL    │
                       │   :5432         │
                       └─────────────────┘
```

### Component Mapping to Production

| Local Component | Production Equivalent | Purpose |
|----------------|----------------------|---------|
| Webhook Receiver (:3001) | Lambda + API Gateway | HTTP webhook endpoint |
| LocalStack SQS | AWS SQS | Message queuing |
| Webhook Processor | Lambda + SQS trigger | Background processing |
| PostgreSQL | RDS PostgreSQL | Data persistence |
| Stripe CLI | Stripe Webhooks | Event simulation |

## Environment Setup

### 1. Environment Files

Create `.env.local` for local webhook development:

```bash
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
```

### 2. Docker Environment Variables

Create `.env.docker` for Docker Compose:

```bash
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
```

## Local Services

### 1. Extended Docker Compose

The webhook development uses an extended Docker Compose configuration:

```yaml
# docker-compose.webhooks.yml
version: '3.8'

services:
  # Extend the existing API service
  cyphera-api:
    extends:
      file: docker-compose.yml
      service: cyphera-api
    environment:
      - WEBHOOK_RECEIVER_URL=http://webhook-receiver:3001
    depends_on:
      - postgres
      - localstack
      - webhook-receiver

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
        condition: service_started
    networks:
      - cyphera-network
    volumes:
      - ./logs:/app/logs

  # Webhook Processor Service
  webhook-processor:
    build:
      context: .
      dockerfile: Dockerfile.webhook-processor
    container_name: cyphera-webhook-processor
    environment:
      - STAGE=local
      - DATABASE_URL=postgresql://apiuser:apipassword@postgres:5432/cyphera
      - SQS_QUEUE_URL=http://localstack:4566/000000000000/webhook-queue
      - PAYMENT_SYNC_ENCRYPTION_KEY=local_development_key_32_characters
      - LOCALSTACK_ENDPOINT=http://localstack:4566
    depends_on:
      postgres:
        condition: service_healthy
      localstack:
        condition: service_started
    networks:
      - cyphera-network
    volumes:
      - ./logs:/app/logs
    restart: unless-stopped

networks:
  cyphera-network:
    external: true

volumes:
  postgres_data:
    external: true
```

### 2. Webhook-Specific Dockerfiles

#### Dockerfile.webhook-receiver

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o webhook-receiver ./cmd/webhook-receiver

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/webhook-receiver .
EXPOSE 3001
CMD ["./webhook-receiver"]
```

#### Dockerfile.webhook-processor

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o webhook-processor ./cmd/webhook-processor

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/webhook-processor .
CMD ["./webhook-processor"]
```

### 3. LocalStack Initialization

Create `scripts/setup-localstack.sh`:

```bash
#!/bin/bash
set -e

echo "🔧 Setting up LocalStack resources..."

# Wait for LocalStack to be ready
echo "⏳ Waiting for LocalStack to start..."
until curl -s http://localhost:4566/_localstack/health | grep -q "\"sqs\": \"available\""; do
  sleep 2
done

# Create SQS queues
echo "📦 Creating SQS queues..."

# Main webhook queue
aws --endpoint-url=http://localhost:4566 sqs create-queue \
  --queue-name webhook-queue \
  --region us-east-1

# Dead letter queue
aws --endpoint-url=http://localhost:4566 sqs create-queue \
  --queue-name webhook-dlq \
  --region us-east-1

# Create secrets for testing
echo "🔐 Creating test secrets..."

aws --endpoint-url=http://localhost:4566 secretsmanager create-secret \
  --name payment-sync-encryption-key \
  --secret-string "local_development_key_32_characters" \
  --region us-east-1

echo "✅ LocalStack setup complete!"
```

## Webhook Testing

### 1. Basic Webhook Flow Test

```bash
# 1. Start the local environment
make local-webhooks-up

# 2. Create a test workspace and payment configuration
curl -X POST http://localhost:8080/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Workspace",
    "account_id": "test-account-id"
  }'

# 3. Configure Stripe for the workspace
WORKSPACE_ID="your-workspace-id-from-step-2"
curl -X POST http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID/payment-configurations \
  -H "Content-Type: application/json" \
  -d '{
    "provider_name": "stripe",
    "is_active": true,
    "is_test_mode": true,
    "configuration": {
      "api_key": "sk_test_your_stripe_key",
      "webhook_secret": "whsec_test_local_secret",
      "environment": "test"
    }
  }'

# 4. Test webhook endpoint
curl -X POST http://localhost:3001/webhooks/stripe/$WORKSPACE_ID \
  -H "Content-Type: application/json" \
  -H "Stripe-Signature: t=1234567890,v1=test_signature" \
  -d '{
    "id": "evt_test_webhook",
    "object": "event",
    "type": "customer.created",
    "data": {
      "object": {
        "id": "cus_test123",
        "object": "customer",
        "email": "test@example.com"
      }
    }
  }'
```

### 2. Stripe CLI Integration

#### Setup Stripe CLI

```bash
# Install Stripe CLI (macOS)
brew install stripe/stripe-cli/stripe

# Login to your Stripe account
stripe login

# Listen for webhooks locally
stripe listen --forward-to localhost:3001/webhooks/stripe/$WORKSPACE_ID
```

#### Generate Test Events

```bash
# Trigger customer creation event
stripe trigger customer.created

# Trigger payment success event
stripe trigger payment_intent.succeeded

# Trigger subscription event
stripe trigger customer.subscription.created

# Send custom events
stripe events resend evt_1234567890
```

### 3. Multi-Workspace Testing

Create test script `scripts/test-multi-workspace.sh`:

```bash
#!/bin/bash
set -e

echo "🧪 Testing multi-workspace webhook isolation..."

# Create workspace A
WORKSPACE_A=$(curl -s -X POST http://localhost:8080/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{"name": "Workspace A", "account_id": "account-a"}' | jq -r '.id')

# Create workspace B  
WORKSPACE_B=$(curl -s -X POST http://localhost:8080/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{"name": "Workspace B", "account_id": "account-b"}' | jq -r '.id')

echo "Created workspaces: A=$WORKSPACE_A, B=$WORKSPACE_B"

# Configure different Stripe accounts for each workspace
configure_workspace() {
  local workspace_id=$1
  local stripe_key=$2
  local webhook_secret=$3
  
  curl -X POST http://localhost:8080/api/v1/workspaces/$workspace_id/payment-configurations \
    -H "Content-Type: application/json" \
    -d "{
      \"provider_name\": \"stripe\",
      \"is_active\": true,
      \"is_test_mode\": true,
      \"configuration\": {
        \"api_key\": \"$stripe_key\",
        \"webhook_secret\": \"$webhook_secret\",
        \"environment\": \"test\"
      }
    }"
}

configure_workspace $WORKSPACE_A "sk_test_workspace_a_key" "whsec_workspace_a_secret"
configure_workspace $WORKSPACE_B "sk_test_workspace_b_key" "whsec_workspace_b_secret"

# Send test webhooks to each workspace
send_test_webhook() {
  local workspace_id=$1
  local customer_id=$2
  
  curl -X POST http://localhost:3001/webhooks/stripe/$workspace_id \
    -H "Content-Type: application/json" \
    -H "Stripe-Signature: t=1234567890,v1=test_signature" \
    -d "{
      \"id\": \"evt_test_$customer_id\",
      \"type\": \"customer.created\",
      \"data\": {
        \"object\": {
          \"id\": \"$customer_id\",
          \"email\": \"test-$customer_id@example.com\"
        }
      }
    }"
}

send_test_webhook $WORKSPACE_A "cus_workspace_a_001"
send_test_webhook $WORKSPACE_B "cus_workspace_b_001"

# Verify isolation
echo "🔍 Verifying workspace isolation..."
sleep 2

CUSTOMERS_A=$(curl -s http://localhost:8080/api/v1/workspaces/$WORKSPACE_A/customers | jq length)
CUSTOMERS_B=$(curl -s http://localhost:8080/api/v1/workspaces/$WORKSPACE_B/customers | jq length)

echo "Workspace A customers: $CUSTOMERS_A"
echo "Workspace B customers: $CUSTOMERS_B"

if [ "$CUSTOMERS_A" -eq 1 ] && [ "$CUSTOMERS_B" -eq 1 ]; then
  echo "✅ Workspace isolation test passed!"
else
  echo "❌ Workspace isolation test failed!"
  exit 1
fi
```

### 4. Performance Testing

Create `scripts/test-webhook-performance.sh`:

```bash
#!/bin/bash
set -e

echo "🚀 Testing webhook performance..."

WORKSPACE_ID="test-workspace-id"
WEBHOOK_URL="http://localhost:3001/webhooks/stripe/$WORKSPACE_ID"

# Test concurrent webhooks
test_concurrent_webhooks() {
  local num_requests=$1
  local concurrency=$2
  
  echo "Testing $num_requests requests with $concurrency concurrent connections..."
  
  # Create test payload
  cat > test_payload.json << EOF
{
  "id": "evt_test_performance",
  "type": "customer.created",
  "data": {
    "object": {
      "id": "cus_perf_test",
      "email": "perf@test.com"
    }
  }
}
EOF

  # Use Apache Bench for performance testing
  ab -n $num_requests -c $concurrency -p test_payload.json -T "application/json" \
    -H "Stripe-Signature: t=1234567890,v1=test_signature" \
    $WEBHOOK_URL
  
  rm test_payload.json
}

# Run performance tests
test_concurrent_webhooks 100 10
test_concurrent_webhooks 500 25
test_concurrent_webhooks 1000 50

echo "✅ Performance testing complete!"
```

## Debugging

### 1. Log Monitoring

#### Real-time Log Viewing

```bash
# View all webhook-related logs
make logs-webhooks

# View specific service logs
docker-compose -f docker-compose.webhooks.yml logs -f webhook-receiver
docker-compose -f docker-compose.webhooks.yml logs -f webhook-processor
docker-compose -f docker-compose.webhooks.yml logs -f localstack

# View with timestamps
docker-compose -f docker-compose.webhooks.yml logs -f -t webhook-processor
```

#### Structured Log Analysis

```bash
# Filter logs by level
docker-compose -f docker-compose.webhooks.yml logs webhook-processor | grep ERROR

# Filter by workspace
docker-compose -f docker-compose.webhooks.yml logs webhook-processor | grep "workspace_id.*test-workspace"

# Monitor webhook processing time
docker-compose -f docker-compose.webhooks.yml logs webhook-processor | grep "processing_time"
```

### 2. Database Debugging

#### Monitor Database Activity

```bash
# Connect to local database
psql postgresql://apiuser:apipassword@localhost:5432/cyphera

# Monitor webhook events
SELECT 
  workspace_id,
  provider_name,
  event_type,
  status,
  occurred_at,
  processed_at,
  error_message
FROM payment_sync_events 
ORDER BY occurred_at DESC 
LIMIT 10;

# Monitor processing queue
SELECT 
  COUNT(*) as pending_events,
  provider_name,
  workspace_id
FROM payment_sync_events 
WHERE status = 'pending'
GROUP BY provider_name, workspace_id;

# Check workspace configurations
SELECT 
  w.id,
  w.name,
  wpc.provider_name,
  wpc.is_active,
  wpc.is_test_mode
FROM workspaces w
JOIN workspace_payment_configurations wpc ON w.id = wpc.workspace_id
ORDER BY w.name;
```

### 3. SQS Queue Debugging

#### Monitor LocalStack SQS

```bash
# List queues
aws --endpoint-url=http://localhost:4566 sqs list-queues --region us-east-1

# Get queue attributes
aws --endpoint-url=http://localhost:4566 sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/webhook-queue \
  --attribute-names All --region us-east-1

# Receive messages (for debugging)
aws --endpoint-url=http://localhost:4566 sqs receive-message \
  --queue-url http://localhost:4566/000000000000/webhook-queue \
  --region us-east-1

# Purge queue (clear all messages)
aws --endpoint-url=http://localhost:4566 sqs purge-queue \
  --queue-url http://localhost:4566/000000000000/webhook-queue \
  --region us-east-1
```

### 4. Webhook Signature Debugging

Create `scripts/debug-webhook-signatures.sh`:

```bash
#!/bin/bash

# Generate valid Stripe webhook signature for testing
generate_stripe_signature() {
  local payload="$1"
  local secret="$2"
  local timestamp=$(date +%s)
  
  # Create signature string
  local sig_string="${timestamp}.${payload}"
  
  # Generate HMAC signature
  local signature=$(echo -n "$sig_string" | openssl dgst -sha256 -hmac "$secret" -binary | base64)
  
  echo "t=${timestamp},v1=${signature}"
}

# Test payload
PAYLOAD='{"id":"evt_test","type":"customer.created","data":{"object":{"id":"cus_test"}}}'
SECRET="whsec_test_local_secret"

echo "Test payload: $PAYLOAD"
echo "Secret: $SECRET"
echo "Generated signature: $(generate_stripe_signature "$PAYLOAD" "$SECRET")"

# Test with webhook endpoint
WORKSPACE_ID="test-workspace-id"
SIGNATURE=$(generate_stripe_signature "$PAYLOAD" "$SECRET")

curl -X POST http://localhost:3001/webhooks/stripe/$WORKSPACE_ID \
  -H "Content-Type: application/json" \
  -H "Stripe-Signature: $SIGNATURE" \
  -d "$PAYLOAD" \
  -v
```

## Troubleshooting

### Common Issues and Solutions

#### Issue: Webhook receiver not responding

**Symptoms:**
- `curl: (7) Failed to connect to localhost:3001`
- Connection refused errors

**Solutions:**
```bash
# Check if service is running
docker-compose -f docker-compose.webhooks.yml ps webhook-receiver

# Check logs for startup errors
docker-compose -f docker-compose.webhooks.yml logs webhook-receiver

# Restart the service
docker-compose -f docker-compose.webhooks.yml restart webhook-receiver

# Check port binding
netstat -tlnp | grep 3001
```

#### Issue: Database connection failures

**Symptoms:**
- `failed to connect to database`
- `connection refused` in logs

**Solutions:**
```bash
# Verify database is running
docker-compose -f docker-compose.webhooks.yml ps postgres

# Check database connectivity
psql postgresql://apiuser:apipassword@localhost:5432/cyphera -c "SELECT 1;"

# Check database logs
docker-compose -f docker-compose.webhooks.yml logs postgres

# Reset database
make db-reset-local
```

#### Issue: LocalStack SQS not working

**Symptoms:**
- `AWS endpoint not found`
- SQS operation errors

**Solutions:**
```bash
# Check LocalStack status
curl http://localhost:4566/_localstack/health

# Restart LocalStack
docker-compose -f docker-compose.webhooks.yml restart localstack

# Recreate SQS resources
./scripts/setup-localstack.sh

# Check SQS queue exists
aws --endpoint-url=http://localhost:4566 sqs list-queues --region us-east-1
```

#### Issue: Webhook signature validation fails

**Symptoms:**
- `invalid signature` errors
- `400 Bad Request` responses

**Solutions:**
```bash
# Verify webhook secret configuration
curl http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID/payment-configurations

# Use signature debugging script
./scripts/debug-webhook-signatures.sh

# Check timestamp tolerance
# Ensure timestamp in signature is recent (within 5 minutes)

# Verify payload format
# Ensure Content-Type is application/json
# Ensure payload is valid JSON
```

#### Issue: Messages not being processed

**Symptoms:**
- Messages visible in SQS but not processed
- No processing logs

**Solutions:**
```bash
# Check webhook processor is running
docker-compose -f docker-compose.webhooks.yml ps webhook-processor

# Check processor logs
docker-compose -f docker-compose.webhooks.yml logs webhook-processor

# Manually check SQS message count
aws --endpoint-url=http://localhost:4566 sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/webhook-queue \
  --attribute-names ApproximateNumberOfMessages --region us-east-1

# Restart processor
docker-compose -f docker-compose.webhooks.yml restart webhook-processor
```

### Health Check Commands

Create `scripts/health-check-local.sh`:

```bash
#!/bin/bash
set -e

echo "🔍 Checking local webhook system health..."

# Check API
echo -n "API (port 8080): "
curl -sf http://localhost:8080/health > /dev/null && echo "✅ OK" || echo "❌ FAIL"

# Check webhook receiver
echo -n "Webhook Receiver (port 3001): "
curl -sf http://localhost:3001/health > /dev/null && echo "✅ OK" || echo "❌ FAIL"

# Check database
echo -n "Database (port 5432): "
pg_isready -h localhost -p 5432 -U apiuser > /dev/null && echo "✅ OK" || echo "❌ FAIL"

# Check LocalStack
echo -n "LocalStack (port 4566): "
curl -sf http://localhost:4566/_localstack/health > /dev/null && echo "✅ OK" || echo "❌ FAIL"

# Check SQS queue
echo -n "SQS Queue: "
aws --endpoint-url=http://localhost:4566 sqs get-queue-url \
  --queue-name webhook-queue --region us-east-1 > /dev/null 2>&1 && echo "✅ OK" || echo "❌ FAIL"

# Check webhook processor
echo -n "Webhook Processor: "
docker-compose -f docker-compose.webhooks.yml ps webhook-processor | grep -q Up && echo "✅ OK" || echo "❌ FAIL"

echo "🏁 Health check complete!"
```

### Reset Everything

Create `scripts/reset-local-webhooks.sh`:

```bash
#!/bin/bash
set -e

echo "🔄 Resetting local webhook environment..."

# Stop all services
docker-compose -f docker-compose.webhooks.yml down

# Remove volumes (optional - uncomment if you want to reset data)
# docker volume rm cyphera-webhooks_postgres_data 2>/dev/null || true

# Clean up LocalStack data
rm -rf /tmp/localstack/* 2>/dev/null || true

# Rebuild and start
docker-compose -f docker-compose.webhooks.yml build
docker-compose -f docker-compose.webhooks.yml up -d

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 10

# Setup LocalStack resources
./scripts/setup-localstack.sh

# Run database migrations
make db-migrate

echo "✅ Local webhook environment reset complete!"
```

---

## Next Steps

Once you have the local environment working:

1. **Test Integration** - Run the multi-workspace test
2. **Performance Testing** - Use the performance test scripts
3. **Add Custom Webhooks** - Extend for other payment providers
4. **Production Deployment** - Use this local setup to validate before deploying to AWS

## Support

If you encounter issues:

1. Check the troubleshooting section above
2. Run the health check script
3. Review service logs for specific error messages
4. Reset the environment if needed

Remember: This local setup closely mirrors production, so successful local testing means your webhooks should work in the cloud! 