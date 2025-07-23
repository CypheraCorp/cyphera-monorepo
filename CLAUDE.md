# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Essential Commands
```bash
# Setup and code generation
make install                    # Install all dependencies (Go + Node.js)
make gen                       # Generate SQLC database code
make proto-build-all          # Generate gRPC code

# Development servers
make dev                      # Run API server with hot reload (requires delegation server running)
make dev-all                  # Run all services (API, delegation server, subscription processor)
make delegation-server        # Run delegation server only
make subscription-processor   # Run subscription processor

# Testing
make test                     # Run unit tests
make test-all                # Run all tests including integration
make test-integration        # Run integration tests
make delegation-server-test  # Test delegation server

# Building
make build                   # Build main API binary
make build-lambda-all       # Build all Lambda functions
make build-subprocessor     # Build subscription processor

# Code quality
make lint                   # Run linters
make swag                   # Generate/update Swagger documentation

# Database
docker-compose up postgres   # Start PostgreSQL database

# Delegation server (from delegation-server/)
npm run build               # Build TypeScript
npm run dev                # Run with auto-reload
npm run test               # Run tests
npm run proto:build        # Generate gRPC code
```

## High-Level Architecture

### System Components

1. **Main API (Go)** - Core business logic in `/cmd/api/main`
   - AWS Lambda deployment with Gin framework
   - Handles authentication, CRUD operations, and orchestrates services
   - Entry point: `/cmd/api/main/main.go`

2. **Delegation Server (Node.js)** - Blockchain operations in `/delegation-server`
   - gRPC server for MetaMask delegation operations
   - Manages smart account creation and transaction signing
   - Communicates with main API via gRPC

3. **Subscription Processor (Go)** - Background job in `/cmd/subscription-processor`
   - Processes recurring subscription payments
   - Uses stored delegation credentials
   - Runs periodically to check and process due subscriptions

4. **Database Layer** - PostgreSQL with SQLC
   - Type-safe SQL through SQLC code generation
   - Schema in `/internal/db/schema.sql`
   - Queries in `/internal/db/queries/`

### Key Architectural Patterns

1. **Clean Architecture**
   - Database queries isolated in SQLC-generated code (`/internal/db`)
   - Business logic in handlers (`/internal/handlers`)
   - External services abstracted in client packages (`/internal/client`)

2. **Multi-Service Communication**
   - Main API ↔ Delegation Server: gRPC
   - Services → Database: Direct PostgreSQL connections
   - External APIs: REST clients (Circle, Stripe, etc.)

3. **Authentication Flow**
   - JWT tokens for user authentication (Web3Auth integration)
   - API keys for service-to-service communication
   - Middleware in `internal/auth/middleware.go`
   - Workspace-based multi-tenancy

4. **Blockchain Integration**
   - Delegation Server handles all blockchain operations
   - Uses MetaMask Delegation Toolkit for smart account management
   - Supports multiple blockchain networks via Circle API

### Important Conventions

1. **Database**
   - UUID primary keys for all tables
   - Soft deletes with `deleted_at` timestamps
   - JSONB columns for flexible metadata storage
   - Foreign key constraints enforced

2. **API Design**
   - RESTful endpoints with consistent error handling
   - Swagger documentation via annotations
   - Request/response logging middleware
   - CORS configuration for cross-origin requests

3. **Error Handling**
   - Structured error responses with error codes
   - Logging with contextual information
   - Dead Letter Queue for failed webhook processing

4. **Security**
   - Secrets managed via AWS Secrets Manager in production
   - Environment variables for local development
   - Encrypted storage for sensitive delegation data

### Development Workflow

1. **Initial Setup**
   - Copy `.env.template` to `.env` and configure
   - Copy `delegation-server/.env.example` to `delegation-server/.env`
   - Run `make install` to set up dependencies
   - Start PostgreSQL with `docker-compose up postgres`

2. **Making Changes**
   - Database changes: Update schema/queries, then run `make gen`
   - gRPC changes: Update proto files, then run `make proto-build-all`
   - API changes: Update handlers and run `make swag` for docs
   - Hot reload active during development

3. **Testing**
   - Write tests alongside code changes
   - Run `make test` before committing
   - Integration tests require all services running

4. **Code Quality**
   - Run `make lint` to check for issues
   - Ensure Swagger docs are updated
   - Follow existing code patterns and conventions