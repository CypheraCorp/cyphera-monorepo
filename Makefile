# Cyphera Monorepo Makefile
# This Makefile handles complex orchestration tasks that are better suited for Make
# For standard development commands, use npm scripts

.PHONY: help
help:
	@echo "Cyphera Monorepo - Make Commands"
	@echo ""
	@echo "For standard development, use npm commands:"
	@echo "  npm run dev:all     - Start all services"
	@echo "  npm run build:all   - Build all projects"
	@echo "  npm run test:all    - Run all tests"
	@echo ""
	@echo "Make is used for:"
	@echo "  Protocol generation, database operations, Docker management"
	@echo ""
	@echo "Available make targets:"
	@echo "  make gen            - Generate SQLC database code"
	@echo "  make proto-gen      - Generate all protobuf code"
	@echo "  make swagger-gen    - Generate Swagger/OpenAPI docs"
	@echo "  make db-reset       - Reset database to clean state"
	@echo "  make docker-dev     - Run development environment in Docker"
	@echo "  make sam-build      - Build AWS SAM applications"
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
		--generalInfo ../../apps/api/cmd/main/main.go \
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
# Testing Infrastructure
# ==============================================================================

.PHONY: test test-unit test-mock test-integration test-coverage test-coverage-html test-db-up test-db-down

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

generate-mocks:
	@echo "🔧 Generating Go mocks..."
	@./scripts/generate-mocks.sh

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