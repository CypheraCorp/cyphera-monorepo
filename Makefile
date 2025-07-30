# Cyphera Monorepo Makefile
# This Makefile handles complex orchestration tasks that are better suited for Make
# For standard development commands, use npm scripts

.PHONY: help
help:
	@echo "Cyphera Monorepo - Make Commands"
	@echo ""
	@echo "🧪 TESTING COMMANDS (GitHub Actions Compatible):"
	@echo "  make test-github-actions  - Run EXACT same tests as GitHub Actions"
	@echo "  make test-all            - Complete test suite (unit + integration + builds)"
	@echo "  make test-quick          - Fast tests only (no database/integration)"
	@echo "  make test-handlers       - API handler tests (same as GitHub Actions unit tests)"
	@echo "  make test-services       - Service layer tests"
	@echo "  make test-integration    - Integration tests with database"
	@echo "  make test-builds         - Verify all components build"
	@echo "  make test-format         - Check code formatting"
	@echo ""
	@echo "🔧 DEVELOPMENT COMMANDS:"
	@echo "  make gen                 - Generate SQLC database code"
	@echo "  make generate-mocks      - Generate mocks for all interfaces"
	@echo "  make proto-gen           - Generate all protobuf code"
	@echo "  make swagger-gen         - Generate Swagger/OpenAPI docs"
	@echo ""
	@echo "🐳 INFRASTRUCTURE:"
	@echo "  make db-reset            - Reset database to clean state"
	@echo "  make docker-dev          - Run development environment in Docker"
	@echo "  make sam-build           - Build AWS SAM applications"
	@echo ""
	@echo "🌐 DELEGATION SERVER:"
	@echo "  make delegation-server-setup  - Install dependencies"
	@echo "  make delegation-server-test   - Run TypeScript tests"
	@echo "  make delegation-server-lint   - Run TypeScript linting"
	@echo "  make delegation-server-build  - Build TypeScript"
	@echo ""
	@echo "💡 RECOMMENDED: Run 'make test-github-actions' before pushing to ensure CI passes!"
	@echo ""

# ==============================================================================
# Variables
# ==============================================================================

GO := go
PROTOC := protoc
DOCKER_COMPOSE := docker compose
NX := npx nx

# Load environment variables from .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# ==============================================================================
# Protocol Buffer Generation
# ==============================================================================

.PHONY: proto-gen proto-gen-go proto-gen-ts

proto-gen: proto-gen-go proto-gen-ts
	@echo "✅ All protocol buffers generated"

proto-gen-go:
	@echo "🔧 Generating Go protocol buffers..."
	@$(PROTOC) --go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		libs/go/proto/delegation.proto

proto-gen-ts:
	@echo "🔧 Generating TypeScript protocol buffers..."
	@cd apps/delegation-server && npm run proto:build

# ==============================================================================
# Database Operations
# ==============================================================================

.PHONY: db-reset db-migrate db-seed db-console gen

db-reset:
	@echo "🗄️  Resetting database..."
	@./scripts/reset_db.sh

db-migrate:
	@echo "🗄️  Running database migrations..."
	@$(NX) run go-shared:migrate

db-seed:
	@echo "🗄️  Seeding database..."
	@$(NX) run go-shared:seed

db-console:
	@echo "🗄️  Opening database console..."
	@psql "$(DATABASE_URL)"

# Generate SQLC code
gen:
	@echo "🔧 Generating SQLC code..."
	@cd libs/go/db && sqlc generate
	@echo "✅ SQLC code generated"

# ==============================================================================
# Mock Generation
# ==============================================================================

.PHONY: generate-mocks mock-gen

# Generate all mocks
generate-mocks: mock-gen
	@echo "✅ All mocks generated"

mock-gen:
	@echo "🔧 Generating mocks..."
	@echo "  → Generating mock for db.Querier..."
	@mockgen -source=libs/go/db/querier.go -destination=libs/go/mocks/mock_querier.go -package=mocks
	@echo "  → Generating mocks for service interfaces..."
	@mockgen -source=libs/go/interfaces/services.go -destination=libs/go/mocks/mock_services.go -package=mocks
	@echo "  → Generating mocks for client interfaces..."
	@mockgen -source=libs/go/interfaces/clients.go -destination=libs/go/mocks/mock_clients.go -package=mocks
	@echo "✅ Mock generation complete"

# ==============================================================================
# Docker Development Environment
# ==============================================================================

.PHONY: docker-dev docker-down docker-logs docker-clean

docker-dev:
	@echo "🐳 Starting Docker development environment..."
	@$(DOCKER_COMPOSE) -f docker-compose.dev.yml up -d

docker-down:
	@echo "🐳 Stopping Docker environment..."
	@$(DOCKER_COMPOSE) -f docker-compose.dev.yml down

docker-logs:
	@echo "📋 Viewing Docker logs..."
	@$(DOCKER_COMPOSE) -f docker-compose.dev.yml logs -f

docker-clean:
	@echo "🧹 Cleaning Docker environment..."
	@$(DOCKER_COMPOSE) -f docker-compose.dev.yml down -v
	@docker system prune -f

# ==============================================================================
# API Documentation
# ==============================================================================

.PHONY: swagger-gen swagger-serve

swagger-gen:
	@echo "📚 Generating Swagger documentation..."
	@swag init \
		--dir ./apps/api/handlers \
		--generalInfo ../cmd/main/main.go \
		--output ./docs/api \
		--tags='!exclude'
	@echo "✅ Swagger docs generated at docs/api/"

swagger-serve:
	@echo "🌐 Serving Swagger UI at http://localhost:8080"
	@docker run -p 8080:8080 -e SWAGGER_JSON=/api/swagger.json \
		-v $(PWD)/docs/api:/api swaggerapi/swagger-ui

# ==============================================================================
# AWS SAM Build Targets
# ==============================================================================

.PHONY: sam-build sam-build-api sam-build-webhooks sam-build-processor sam-build-dunning

sam-build: sam-build-api sam-build-webhooks sam-build-processor sam-build-dunning
	@echo "✅ All SAM applications built"

sam-build-api:
	@echo "🔨 Building API for SAM..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		$(GO) build -o ./apps/api/bootstrap ./apps/api/cmd/main/main.go

sam-build-webhooks:
	@echo "🔨 Building webhook functions for SAM..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		$(GO) build -o ./apps/webhook-receiver/bootstrap ./apps/webhook-receiver/cmd/main.go
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		$(GO) build -o ./apps/webhook-processor/bootstrap ./apps/webhook-processor/cmd/main.go

sam-build-processor:
	@echo "🔨 Building subscription processor for SAM..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		$(GO) build -o ./apps/subscription-processor/bootstrap ./apps/subscription-processor/cmd/main.go

sam-build-dunning:
	@echo "🔨 Building dunning processor for SAM..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		$(GO) build -o ./apps/dunning-processor/bin/bootstrap ./apps/dunning-processor/cmd/main.go

# ==============================================================================
# Local Development Utilities
# ==============================================================================

.PHONY: local-certs local-env-check

local-certs:
	@echo "🔐 Generating local SSL certificates..."
	@./scripts/generate-local-certs.sh

local-env-check:
	@echo "🔍 Checking local environment..."
	@./scripts/check-local-env.sh

# ==============================================================================
# CI/CD Helpers
# ==============================================================================

.PHONY: ci-setup ci-validate

ci-setup:
	@echo "🤖 Setting up CI environment..."
	@npm ci --silent
	@$(GO) mod download

ci-validate:
	@echo "✓ Validating CI configuration..."
	@$(NX) run-many --target=lint --all --quiet
	@$(NX) format:check --all

# ==============================================================================
# Delegation Server Commands
# ==============================================================================

.PHONY: delegation-server-setup delegation-server-lint delegation-server-test delegation-server-build

# Install delegation server dependencies
delegation-server-setup:
	@echo "📦 Setting up delegation server..."
	@echo "📚 Building delegation library first..."
	@cd libs/ts/delegation && npm install --legacy-peer-deps && npm run build
	@echo "📦 Installing delegation server dependencies..."
	@cd apps/delegation-server && npm ci

# Run delegation server linting
delegation-server-lint:
	@echo "🔍 Linting delegation server..."
	@echo "📦 Building delegation library first..."
	@cd libs/ts/delegation && npm run build
	@cd apps/delegation-server && npm run lint

# Run delegation server tests
delegation-server-test:
	@echo "🧪 Testing delegation server..."
	@cd apps/delegation-server && npm test

# Build delegation server
delegation-server-build:
	@echo "🔨 Building delegation server..."
	@cd apps/delegation-server && npm run build

# ==============================================================================
# Testing Infrastructure
# ==============================================================================

.PHONY: test test-unit test-mock test-integration test-coverage test-coverage-html test-db-up test-db-down
.PHONY: test-all test-github-actions test-handlers test-services test-quick test-ci

# Load test configuration
TEST_CONFIG := test.config.json
COVERAGE_THRESHOLD := $(shell jq -r '.coverage.threshold' $(TEST_CONFIG))
TEST_DB_NAME := $(shell jq -r '.database.test_db_name' $(TEST_CONFIG))

test: test-unit
	@echo "✅ All tests completed"

test-unit: test-mock
	@echo "🧪 Running unit tests (includes mocked tests)..."
	@$(GO) test -race -timeout=30s ./apps/api/... ./libs/go/... -v

test-mock:
	@echo "🎭 Running unit tests with database mocks (fast)..."
	@$(GO) test -race -timeout=30s \
		-run=".*Mock.*" \
		./apps/api/handlers/... -v

test-integration: test-db-up
	@echo "🧪 Running integration tests with real database..."
	@$(GO) test -race -timeout=30m -tags=integration ./tests/integration/... -v
	@$(MAKE) test-db-down

test-coverage:
	@echo "📊 Running tests with coverage..."
	@$(GO) test -race -timeout=30s -coverprofile=coverage.out \
		-coverpkg=./apps/api/...,./libs/go/... \
		./apps/api/... ./libs/go/...
	@$(GO) tool cover -func=coverage.out | grep total:
	@echo "Checking coverage threshold ($(COVERAGE_THRESHOLD)%)..."
	@./scripts/check-coverage.sh $(COVERAGE_THRESHOLD)

test-coverage-html: test-coverage
	@echo "📊 Generating HTML coverage report..."
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

test-db-up:
	@echo "🐳 Starting test database..."
	@docker run -d --name cyphera-test-db \
		-e POSTGRES_DB=$(TEST_DB_NAME) \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-p 5433:5432 \
		postgres:15-alpine || true
	@sleep 3
	@echo "✅ Test database ready"

test-db-down:
	@echo "🐳 Stopping test database..."
	@docker stop cyphera-test-db || true
	@docker rm cyphera-test-db || true

# ==============================================================================
# Comprehensive Test Commands (GitHub Actions Compatible)
# ==============================================================================

# Run all tests exactly like GitHub Actions
test-github-actions:
	@echo "🚀 Running GitHub Actions Test Suite Locally"
	@echo "============================================="
	@echo ""
	@echo "1️⃣ Running Unit Tests (Handler Tests)..."
	@cd apps/api && go test ./handlers/... -v -race -timeout=30s
	@echo ""
	@echo "2️⃣ Running Integration Tests..."
	@go test -tags=integration ./tests/integration/... -v -timeout=30m
	@echo ""
	@echo "3️⃣ Running Delegation Server Tests..."
	@$(MAKE) delegation-server-test
	@echo ""
	@echo "4️⃣ Verifying Builds..."
	@$(MAKE) test-builds
	@echo ""
	@echo "5️⃣ Checking Code Quality..."
	@$(MAKE) test-format
	@echo ""
	@echo "✅ All GitHub Actions tests completed successfully!"

# Run comprehensive test suite (includes everything)
test-all: generate-mocks
	@echo "🧪 Running Complete Test Suite"
	@echo "=============================="
	@$(MAKE) test-quick
	@$(MAKE) test-integration
	@$(MAKE) delegation-server-test
	@$(MAKE) test-builds
	@$(MAKE) test-format
	@echo "✅ All tests completed successfully!"

# Quick test suite (no database, no integration)
test-quick:
	@echo "⚡ Running Quick Test Suite"
	@echo "=========================="
	@echo "🧪 Handler tests..."
	@cd apps/api && go test ./handlers/... -v -race -timeout=30s
	@echo "🧪 Service tests..."
	@$(MAKE) test-services
	@echo "✅ Quick tests completed!"

# Test all services
test-services:
	@echo "🔧 Testing Go services..."
	@cd libs/go && go test ./services/... -v -race -timeout=30s

# Test only handlers (GitHub Actions unit test equivalent)
test-handlers:
	@echo "🎯 Testing API handlers (GitHub Actions equivalent)..."
	@cd apps/api && go test ./handlers/... -v -race -timeout=30s

# Test build commands
test-builds:
	@echo "🔨 Testing builds..."
	@echo "  → API build..."
	@cd apps/api && go build ./... > /dev/null
	@echo "  → Libraries build..."
	@cd libs/go && go build ./... > /dev/null
	@echo "  → Subscription processor build..."
	@cd apps/subscription-processor && go build ./... > /dev/null
	@echo "  → Delegation server build..."
	@$(MAKE) delegation-server-build > /dev/null
	@echo "✅ All builds successful!"

# Test code formatting
test-format:
	@echo "📝 Checking code formatting..."
	@echo "  → Go formatting..."
	@FORMAT_ISSUES=$$(gofmt -s -l libs/go/ apps/api/ | grep -v "libs/go/mocks/helpers.go" | wc -l); \
	if [ "$$FORMAT_ISSUES" -eq 0 ]; then \
		echo "✅ Go code is properly formatted"; \
	else \
		echo "❌ Found $$FORMAT_ISSUES formatting issues:"; \
		gofmt -s -l libs/go/ apps/api/ | grep -v "libs/go/mocks/helpers.go"; \
		exit 1; \
	fi

# CI-friendly test command (no colors, structured output)
test-ci:
	@echo "Running CI Test Suite..."
	@cd apps/api && go test ./handlers/... -race -timeout=30s
	@go test -tags=integration ./tests/integration/... -timeout=30m
	@cd apps/delegation-server && npm test
	@echo "CI tests completed"


# ==============================================================================
# Advanced Operations (Use with caution)
# ==============================================================================

.PHONY: deep-clean nuke

deep-clean:
	@echo "🧹 Deep cleaning build artifacts..."
	@$(NX) run-many --target=clean --all
	@find . -name "node_modules" -type d -prune -exec rm -rf {} +
	@find . -name "tmp" -type d -prune -exec rm -rf {} +
	@find . -name "dist" -type d -prune -exec rm -rf {} +
	@find . -name ".next" -type d -prune -exec rm -rf {} +
	@$(GO) clean -cache -testcache -modcache
	@rm -f coverage.out coverage.html

nuke: deep-clean
	@echo "☢️  Nuclear option - removing everything..."
	@git clean -fdx -e .env -e .env.local