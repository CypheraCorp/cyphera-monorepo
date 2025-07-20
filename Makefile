.PHONY: all build install test clean lint run swag deploy test-all test-integration stop-integration-server ensure-executable proto-build-go proto-build-js proto-build-all subscription-processor delegation-server delegation-server-setup delegation-server-build delegation-server-start delegation-server-mock delegation-server-test delegation-server-lint build-webhook-receiver build-webhook-processor build-webhook-receiver-sam-local build-webhook-processor-sam-local local-webhooks-up local-webhooks-down local-webhooks-logs local-webhooks-reset local-webhooks-test local-webhooks-health

# Go parameters
BINARY_NAME=cyphera-api
MAIN_PACKAGE=./cmd/api/main
LOCAL_PACKAGE=./cmd/api/local
GO=go

all: lint test build

build:
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

build-local:
	$(GO) build -o bin/$(BINARY_NAME)-local $(LOCAL_PACKAGE)

install: delegation-server-setup
	go mod tidy

# Run all tests including integration tests
test-all: test test-integration delegation-server-test

test:
	$(GO) test -v ./...
	@echo "Note: Run 'make delegation-server-test' to test the delegation server"

delegation-server-test:
	cd delegation-server && npm run test

# Run integration tests with the delegation server
test-integration: ensure-executable
	@echo "Running delegation integration tests with mock server..."
	DELEGATION_LOCAL_MODE=true ./scripts/integration-test.sh --cli

# Interval for the local subscription processor loop (e.g., 60s, 1m, 5m)
SUB_INTERVAL ?= 10s

# Run the subscription processor
subscription-processor:
	@echo "Starting subscription processor loop (runs every $(SUB_INTERVAL))... Press Ctrl+C to stop."
	@while true; do \
		echo "Running subscription processor at $$(date)..."; \
		$(GO) run ./cmd/subscription-processor/main.go; \
		echo "Subscription processor finished. Sleeping for $(SUB_INTERVAL)..."; \
		sleep $(SUB_INTERVAL); \
	done

dev: ensure-executable
	air

dev-all: ensure-executable
	@echo "Starting development environment with hot reloading for the API server..."
	./scripts/start-dev-all.sh

# Ensure scripts are executable
ensure-executable:
	@chmod +x scripts/integration-test.sh
	@chmod +x scripts/start-dev-all.sh

# Stop integration server processes if they're still running
stop-integration-server:
	@echo "Stopping integration test server processes..."
	-@pkill -f "npm run start:mock" 2>/dev/null || true
	@echo "Checking for processes using port 50051 (gRPC)..."
	-@lsof -ti:50051 | xargs kill -9 2>/dev/null || true
	@echo "Servers stopped"

clean:
	$(GO) clean
	rm -f bin/$(BINARY_NAME)
	rm -f bin/cyphera-api-dev
	rm -rf bin/air*

lint:
	$(GO) vet ./...
	golangci-lint run ./...
	gofmt -l .

run:
	$(GO) run $(LOCAL_PACKAGE)

run-lambda:
	$(GO) run $(MAIN_PACKAGE)

swag:
	swag init --dir ./internal/handlers --generalInfo ../../cmd/api/main/main.go --output ./docs/gitbook/api --tags='!exclude'
	npx swagger2openapi docs/gitbook/api/swagger.json --yaml > docs/gitbook/api/openapi.yaml
	rm -f docs/gitbook/api/swagger.json docs/gitbook/api/swagger.yaml docs/gitbook/api/docs.go docs/gitbook/api/openapi.json
	@echo "✅ Generated OpenAPI 3.0 spec: docs/gitbook/api/openapi.yaml"

gitbook-sync: swag
	@echo "🚀 Publishing OpenAPI spec to GitBook..."
	@if [ -z "$(GITBOOK_API_KEY)" ]; then \
		echo "❌ Error: GITBOOK_API_KEY environment variable is required"; \
		echo "💡 Set it with: export GITBOOK_API_KEY=your_api_key"; \
		exit 1; \
	fi
	gitbook openapi publish --spec cyphera-api --organization 1AowlQrHqnVzYns51v22 docs/gitbook/api/openapi.yaml
	@echo "✅ Successfully published to GitBook!"

deploy:
	# Add deployment steps here

# ===============================================
# Local Webhook Development Commands
# ===============================================

# Main command: Set up complete local webhook development environment
local-webhooks-up: ensure-executable
	@echo "🚀 Setting up local webhook development environment..."
	@chmod +x scripts/setup-local-webhooks.sh
	./scripts/setup-local-webhooks.sh

# Alternative with options
local-webhooks-up-force: ensure-executable
	@echo "🚀 Setting up local webhook environment (force rebuild)..."
	@chmod +x scripts/setup-local-webhooks.sh
	FORCE_REBUILD=true ./scripts/setup-local-webhooks.sh

local-webhooks-up-skip-tests: ensure-executable
	@echo "🚀 Setting up local webhook environment (skip tests)..."
	@chmod +x scripts/setup-local-webhooks.sh
	SKIP_TESTS=true ./scripts/setup-local-webhooks.sh

# Stop local webhook services
local-webhooks-down:
	@echo "🛑 Stopping local webhook services..."
	docker-compose -f docker-compose.webhooks.yml down

# View logs from all webhook services
local-webhooks-logs:
	@echo "📋 Viewing webhook service logs (Ctrl+C to exit)..."
	docker-compose -f docker-compose.webhooks.yml logs -f

# View logs from specific service
local-webhooks-logs-receiver:
	docker-compose -f docker-compose.webhooks.yml logs -f webhook-receiver

local-webhooks-logs-processor:
	docker-compose -f docker-compose.webhooks.yml logs -f webhook-processor

local-webhooks-logs-api:
	docker-compose -f docker-compose.webhooks.yml logs -f cyphera-api

local-webhooks-logs-db:
	docker-compose -f docker-compose.webhooks.yml logs -f postgres

local-webhooks-logs-localstack:
	docker-compose -f docker-compose.webhooks.yml logs -f localstack

# Reset the entire local webhook environment
local-webhooks-reset: ensure-executable
	@echo "🔄 Resetting local webhook environment..."
	@if [ -f "scripts/reset-local-webhooks.sh" ]; then \
		chmod +x scripts/reset-local-webhooks.sh; \
		./scripts/reset-local-webhooks.sh; \
	else \
		echo "Reset script not found. Running manual reset..."; \
		docker-compose -f docker-compose.webhooks.yml down -v; \
		docker system prune -f; \
		rm -rf /tmp/localstack/* 2>/dev/null || true; \
		make local-webhooks-up; \
	fi

# Run health checks on local webhook services
local-webhooks-health: ensure-executable
	@echo "🏥 Running webhook service health checks..."
	@if [ -f "scripts/health-check-local.sh" ]; then \
		chmod +x scripts/health-check-local.sh; \
		./scripts/health-check-local.sh; \
	else \
		echo "Health check script not found. Install with 'make local-webhooks-up' first."; \
	fi

# Run webhook tests
local-webhooks-test: ensure-executable
	@echo "🧪 Running local webhook tests..."
	@if [ -f "scripts/test-multi-workspace.sh" ]; then \
		chmod +x scripts/test-multi-workspace.sh; \
		./scripts/test-multi-workspace.sh; \
	else \
		echo "Test script not found. Setting up test environment..."; \
		make local-webhooks-up; \
	fi

# Quick webhook test with curl
local-webhooks-test-quick:
	@echo "🧪 Running quick webhook test..."
	@WORKSPACE_ID=$$(cat .local-workspace-id 2>/dev/null || echo "test-workspace-id"); \
	echo "Testing with workspace ID: $$WORKSPACE_ID"; \
	curl -X POST http://localhost:3001/webhooks/stripe/$$WORKSPACE_ID \
		-H "Content-Type: application/json" \
		-H "Stripe-Signature: t=$$(date +%s),v1=test_signature" \
		-d '{"id":"evt_quick_test","type":"customer.created","data":{"object":{"id":"cus_quick_test","email":"test@example.com"}}}' \
		-w "\nHTTP Status: %{http_code}\n"

# Check SQS queue status
local-webhooks-sqs-status:
	@echo "📊 Checking SQS queue status..."
	@AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 sqs get-queue-attributes \
		--queue-url http://localhost:4566/000000000000/webhook-queue \
		--attribute-names ApproximateNumberOfMessages,ApproximateNumberOfMessagesNotVisible \
		--region us-east-1 \
		--output table || echo "❌ Failed to connect to LocalStack SQS"

# Purge SQS queue (clear all messages)
local-webhooks-sqs-purge:
	@echo "🧹 Purging SQS queue..."
	@AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 sqs purge-queue \
		--queue-url http://localhost:4566/000000000000/webhook-queue \
		--region us-east-1 && echo "✅ Queue purged successfully"

# Connect to local database
local-webhooks-db:
	@echo "🗄️ Connecting to local database..."
	psql postgresql://apiuser:apipassword@localhost:5432/cyphera

# Show webhook service status
local-webhooks-status:
	@echo "📊 Webhook services status:"
	@echo "=================================="
	docker-compose -f docker-compose.webhooks.yml ps

# Restart specific service
local-webhooks-restart-receiver:
	@echo "🔄 Restarting webhook receiver..."
	docker-compose -f docker-compose.webhooks.yml restart webhook-receiver

local-webhooks-restart-processor:
	@echo "🔄 Restarting webhook processor..."
	docker-compose -f docker-compose.webhooks.yml restart webhook-processor

local-webhooks-restart-api:
	@echo "🔄 Restarting API service..."
	docker-compose -f docker-compose.webhooks.yml restart cyphera-api

# ===============================================
# End Local Webhook Development Commands
# ===============================================

# Build proto definitions for Go
proto-build-go:
	@echo "Generating Go gRPC code from proto definitions..."
	protoc --go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		internal/proto/delegation.proto
	@echo "✅ Go gRPC code successfully generated in internal/proto/"

# Build proto definitions for Node.js delegation server
proto-build-js:
	@echo "Generating Node.js gRPC code from proto definitions..."
	cd delegation-server && npm run proto:build
	@echo "✅ Node.js gRPC code successfully generated in delegation-server/src/proto/"

# Build all proto definitions
proto-build-all: proto-build-go proto-build-js
	@echo "✅ All gRPC code successfully generated"

# Individual server commands (used directly by the start-dev.sh script)
api-server:
	$(GO) run $(MAIN_PACKAGE)

# --- Subscription Processor --- #
SUBSCRIPTION_PROCESSOR_PACKAGE=./cmd/subscription-processor
SUBSCRIPTION_PROCESSOR_BINARY_NAME=subprocessor

# Build target for Subscription Processor (general)
build-subprocessor:
	@echo "Building subscription processor binary..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/$(SUBSCRIPTION_PROCESSOR_BINARY_NAME) $(SUBSCRIPTION_PROCESSOR_PACKAGE)
	@echo "Subscription processor binary built at bin/$(SUBSCRIPTION_PROCESSOR_BINARY_NAME)"

# Target for building Subscription Processor specifically for SAM local build
.PHONY: build-subprocessor-sam-local
build-subprocessor-sam-local:
	@echo "Building subscription processor for SAM local..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/bootstrap-subprocessor $(SUBSCRIPTION_PROCESSOR_PACKAGE)
	@echo "Subscription processor SAM local bootstrap built at bin/bootstrap-subprocessor"

# Target for SAM Build Process (SubscriptionProcessorFunction matches template Logical ID)
# This target is called by 'sam build' when BuildMethod: makefile is specified.
# It compiles the code directly into the SAM artifacts directory.
build-SubscriptionProcessorFunction:
	@echo "Building subscription processor directly into SAM artifacts dir: $(ARTIFACTS_DIR)"
	mkdir -p $(ARTIFACTS_DIR) # Ensure the directory exists
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(ARTIFACTS_DIR)/bootstrap $(SUBSCRIPTION_PROCESSOR_PACKAGE)
	@echo "Subscription processor bootstrap built successfully in $(ARTIFACTS_DIR)"

api-server-air:
	air
	
# Delegation server commands
delegation-server:
	cd delegation-server && npm run dev

delegation-server-setup:
	cd delegation-server && npm run setup

delegation-server-build:
	cd delegation-server && npm run build

delegation-server-start:
	cd delegation-server && npm run start

delegation-server-mock:
	cd delegation-server && npm run start:mock

delegation-server-lint:
	cd delegation-server && npm run lint

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Core Development Targets:"
	@echo "  make all            - Run default targets: lint, test, and build"
	@echo "  make build          - Build the binary"
	@echo "  make test           - Run all unit tests"
	@echo "  make test-all       - Run all tests, including integration tests"
	@echo "  make test-integration - Run integration tests with mock server"
	@echo "  make clean          - Clean build files"
	@echo "  make lint           - Run linter"
	@echo "  make run            - Run the application"
	@echo "  make dev            - Run the application in development mode (loads .env)"
	@echo ""
	@echo "Local Webhook Development:"
	@echo "  make local-webhooks-up - 🚀 Set up complete local webhook environment (MAIN COMMAND)"
	@echo "  make local-webhooks-up-force - Set up with force rebuild"
	@echo "  make local-webhooks-up-skip-tests - Set up skipping tests"
	@echo "  make local-webhooks-down - Stop all webhook services"
	@echo "  make local-webhooks-logs - View all service logs"
	@echo "  make local-webhooks-health - Run health checks"
	@echo "  make local-webhooks-test - Run multi-workspace tests"
	@echo "  make local-webhooks-test-quick - Quick webhook test with curl"
	@echo "  make local-webhooks-reset - Reset entire environment"
	@echo "  make local-webhooks-status - Show service status"
	@echo ""
	@echo "Webhook Service Management:"
	@echo "  make local-webhooks-logs-receiver - Webhook receiver logs"
	@echo "  make local-webhooks-logs-processor - Webhook processor logs"
	@echo "  make local-webhooks-logs-api - API service logs"
	@echo "  make local-webhooks-logs-db - Database logs"
	@echo "  make local-webhooks-logs-localstack - LocalStack logs"
	@echo "  make local-webhooks-restart-receiver - Restart webhook receiver"
	@echo "  make local-webhooks-restart-processor - Restart webhook processor"
	@echo "  make local-webhooks-restart-api - Restart API service"
	@echo ""
	@echo "Webhook Utilities:"
	@echo "  make local-webhooks-sqs-status - Check SQS queue status"
	@echo "  make local-webhooks-sqs-purge - Clear SQS queue"
	@echo "  make local-webhooks-db - Connect to local database"
	@echo ""
	@echo "Build Targets:"
	@echo "  make build-webhook-receiver - Build webhook receiver binary"
	@echo "  make build-webhook-processor - Build webhook processor binary"
	@echo "  make build-dlq-processor - Build DLQ processor binary"
	@echo "  make build-lambda-all - Build all Lambda functions"
	@echo "  make build-subprocessor - Build subscription processor binary"
	@echo ""
	@echo "SAM Deployment:"
	@echo "  make build-webhook-receiver-sam-local - Build bootstrap for SAM local testing (webhook receiver)"
	@echo "  make build-webhook-processor-sam-local - Build bootstrap for SAM local testing (webhook processor)"
	@echo "  make build-subprocessor-sam-local - Build bootstrap for SAM local testing (subprocessor)"
	@echo ""
	@echo "Other Services:"
	@echo "  make api-server     - Run the API server without live reload"
	@echo "  make api-server-air - Run the API server with air for live reload"
	@echo "  make delegation-server - Run the delegation server" 
	@echo "  make subscription-processor - Run the subscription processor"
	@echo "  make stop-integration-server - Stop integration server processes"
	@echo ""
	@echo "Code Generation:"
	@echo "  make gen            - Generate SQLC code"
	@echo "  make swag           - Generate Swagger documentation"
	@echo "  make proto-build-go - Generate Go gRPC code from proto definitions"
	@echo "  make proto-build-js - Generate Node.js gRPC code from proto definitions"
	@echo "  make proto-build-all - Generate both Go and Node.js gRPC code"
	@echo ""
	@echo "🚀 Quick Start for Webhook Development:"
	@echo "  1. make local-webhooks-up    # Set up everything"
	@echo "  2. make local-webhooks-test-quick  # Test it works"
	@echo "  3. make local-webhooks-logs  # Monitor logs"
	@echo ""
	@echo "📚 See docs/local_webhook_development_guide.md for detailed instructions"

gen:
	sqlc generate

# Makefile for AWS SAM build with provided.al2 runtime

# The SAM build process for a function with logical ID 'MainFunction'
# will look for a target named 'build-MainFunction'.
# This target should place the built artifact (our 'bootstrap' binary)
# into the directory specified by the ARTIFACTS_DIR environment variable provided by SAM.

# --- Target for SAM Build (Main API) ---
# This target is called by 'sam build' when BuildMethod: makefile is specified.
# It compiles the main API code directly into the SAM artifacts directory.
build-MainFunction:
	@echo "Building main API directly into SAM artifacts dir: $(ARTIFACTS_DIR)"
	mkdir -p $(ARTIFACTS_DIR) # Ensure the directory exists
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(ARTIFACTS_DIR)/bootstrap $(MAIN_PACKAGE)
	@echo "Main API bootstrap built successfully in $(ARTIFACTS_DIR)"

# --- Target for Local SAM Builds ---
# Use this target *before* running 'sam build' locally.
.PHONY: build-sam-local
build-sam-local:
	@echo "DEPRECATED: Use specific targets like build-api-sam-local or build-subprocessor-sam-local"
	@echo "Executing build script (scripts/build.sh) to create bootstrap binary..."
	chmod +x scripts/build.sh # Ensure script is executable
	./scripts/build.sh
	@echo "Build script finished. Bootstrap binary should be ready."

# Target for building main API specifically for SAM local build
.PHONY: build-api-sam-local
build-api-sam-local:
	@echo "Building main API for SAM local..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/bootstrap-api $(MAIN_PACKAGE)
	@echo "Main API SAM local bootstrap built at bin/bootstrap-api"

# --- Webhook Functions --- #
WEBHOOK_RECEIVER_PACKAGE=./cmd/webhook-receiver
WEBHOOK_PROCESSOR_PACKAGE=./cmd/webhook-processor

# Build target for Webhook Receiver (general)
build-webhook-receiver:
	@echo "Building webhook receiver binary..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/webhook-receiver $(WEBHOOK_RECEIVER_PACKAGE)
	@echo "Webhook receiver binary built at bin/webhook-receiver"

# Build target for Webhook Processor (general)
build-webhook-processor:
	@echo "Building webhook processor binary..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o bin/webhook-processor $(WEBHOOK_PROCESSOR_PACKAGE)
	@echo "Webhook processor binary built at bin/webhook-processor"

# Target for SAM Build Process (WebhookReceiverFunction matches template Logical ID)
build-WebhookReceiverFunction:
	@echo "Building webhook receiver directly into SAM artifacts dir: $(ARTIFACTS_DIR)"
	mkdir -p $(ARTIFACTS_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(ARTIFACTS_DIR)/bootstrap $(WEBHOOK_RECEIVER_PACKAGE)
	@echo "Webhook receiver bootstrap built successfully in $(ARTIFACTS_DIR)"

# Target for SAM Build Process (WebhookProcessorFunction matches template Logical ID)
build-WebhookProcessorFunction:
	@echo "Building webhook processor directly into SAM artifacts dir: $(ARTIFACTS_DIR)"
	mkdir -p $(ARTIFACTS_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(ARTIFACTS_DIR)/bootstrap $(WEBHOOK_PROCESSOR_PACKAGE)
	@echo "Webhook processor bootstrap built successfully in $(ARTIFACTS_DIR)"

# Target for SAM Build Process (DLQProcessorFunction matches template Logical ID)
build-DLQProcessorFunction:
	@echo "Building DLQ processor directly into SAM artifacts dir: $(ARTIFACTS_DIR)"
	mkdir -p $(ARTIFACTS_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/dlq-processor
	@echo "DLQ processor bootstrap built successfully in $(ARTIFACTS_DIR)"

# Build target for Webhook Processor SAM deployment (creates bootstrap)
build-webhook-processor-sam-local:
	@echo "Building webhook processor bootstrap for SAM local..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(ARTIFACTS_DIR)/bootstrap $(WEBHOOK_PROCESSOR_PACKAGE)
	@echo "Webhook processor bootstrap built successfully in $(ARTIFACTS_DIR)"

# Build target for Webhook Receiver SAM deployment (creates bootstrap)  
build-webhook-receiver-sam-local:
	@echo "Building webhook receiver bootstrap for SAM local..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(ARTIFACTS_DIR)/bootstrap $(WEBHOOK_RECEIVER_PACKAGE)
	@echo "Webhook receiver bootstrap built successfully in $(ARTIFACTS_DIR)"

# Build DLQ processor binary for Lambda deployment
build-dlq-processor:
	@echo "Building DLQ processor binary..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/dlq-processor ./cmd/dlq-processor
	@echo "DLQ processor binary built at bin/dlq-processor"

# Build all Lambda functions
build-lambda-all: build-webhook-receiver build-webhook-processor build-dlq-processor
	@echo "All Lambda functions built successfully"