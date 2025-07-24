# CLAUDE.md - Cyphera API Configuration

This document provides the necessary context for Claude Code when working on the Cyphera API codebase. It follows best practices for AI-assisted development with hierarchical context management and clear workflow definitions.

## How to Use This Document

This is your primary context file for understanding the Cyphera codebase. It uses hierarchical context similar to object-oriented programming - information is inherited and structured for optimal AI comprehension. Start by understanding the project overview, then dive into specific sections as needed.

## 1. Project Overview & Goals

**Primary Goal:** Cyphera is a crypto-based subscription platform that enables businesses to create and manage subscription products using blockchain technology, integrating Circle's Programmable Wallet and MetaMask's Delegation Toolkit.

**Core Technologies:** 
- Backend: Go, Gin Framework, PostgreSQL, SQLC, gRPC
- Frontend: Next.js 15, TypeScript, React, TailwindCSS
- Blockchain: MetaMask Delegation Toolkit, Circle API
- Infrastructure: AWS Lambda, Docker, AWS Secrets Manager

**Key Concepts:**
- **Multi-tenant workspace architecture** - Each merchant operates within isolated workspaces
- **Delegation Server** - Separate Node.js service handling all blockchain operations via gRPC
- **Smart Account Management** - Uses MetaMask's delegation for gasless transactions
- **Subscription Processing** - Background job system for recurring crypto payments

## 2. Codebase Structure & Key Files

### Backend Structure (Go Monorepo)
```
/apps/
├── api/                      # Main API service (Go/Gin)
│   ├── cmd/main/            # Entry point
│   ├── handlers/            # HTTP request handlers
│   └── server/              # Server configuration and routes
├── delegation-server/        # Blockchain operations (Node.js)
│   ├── src/                 # TypeScript source
│   └── proto/               # gRPC definitions
└── subscription-processor/   # Background job processor (Go)

/libs/go/
├── db/                      # SQLC generated database code
│   ├── schema.sql          # Database schema
│   └── queries/            # SQL queries
├── middleware/             # Shared middleware
│   ├── validation.go       # Input validation
│   ├── ratelimit.go       # Rate limiting
│   ├── correlation.go     # Request tracing
│   └── logging.go         # Enhanced logging
├── auth/                   # Authentication logic
├── client/                 # External service clients
└── helpers/               # Shared utilities

/docs/                     # Documentation
├── feature-development-workflow.md
├── transaction-refactor-example.md
└── correlation-ids.md
```

### Frontend Structure (Next.js)
```
/apps/web-app/
├── src/
│   ├── app/              # Next.js 15 app directory
│   ├── components/       # React components
│   ├── services/         # API clients
│   ├── hooks/           # Custom React hooks
│   ├── lib/             # Utilities and helpers
│   └── types/           # TypeScript definitions
└── tests/e2e/           # Playwright E2E tests
```

## 3. Development Workflow & Best Practices

**Core Task Execution Process: Brainstorm → Plan → Execute → Test → Document**

### 3.1 Task Approach
1. **Brainstorm & Plan:** When given a task, first analyze requirements and create a plan using the TodoWrite tool. Outline files to modify and the approach. Wait for approval before proceeding with implementation.
2. **Security First:** Consider security implications - authentication, authorization, validation, rate limiting
3. **Backend → Frontend:** Implement backend first, then frontend, then E2E tests
4. **Test Everything:** Write tests for new features. Use existing test patterns.
5. **Document Changes:** Update relevant documentation and inline comments
6. **Ask Questions:** If a request is ambiguous or you need more context, ask for clarification before proceeding

### 3.2 Using Context Effectively
- Read existing code patterns before implementing new features
- Use the Task tool for complex searches across the codebase
- Check `/docs/` for architectural decisions and patterns
- Follow conventions in neighboring files

### 3.3 Common Workflows

#### Starting Development
```bash
# Backend
docker-compose up postgres   # Start database
make dev                    # Run API with hot reload
make delegation-server      # Run delegation server

# Frontend (in apps/web-app/)
npm run dev                # Start Next.js dev server
```

#### Making Database Changes
```bash
# 1. Update schema
vi libs/go/db/schema.sql

# 2. Add/update queries
vi libs/go/db/queries/*.sql

# 3. Generate code
make gen

# 4. Run migrations (if needed)
make migrate
```

## 4. Coding Conventions & Style Guide

### Go Backend
- **Style:** Follow standard Go conventions
- **Error Handling:** Always handle errors explicitly
- **Logging:** Use structured logging with zap
- **Testing:** Table-driven tests preferred
- **Linting:** Must pass `golangci-lint run`

### TypeScript Frontend  
- **Language:** TypeScript with strict mode enabled
- **Components:** Functional components with hooks
- **State:** Zustand for global state
- **Styling:** TailwindCSS utilities
- **Linting:** Must pass `npm run lint`

### Universal Rules
- No commented-out code
- Meaningful variable/function names
- Add comments only for complex logic
- Use correlation IDs for tracing
- Hash sensitive data before storage

## 5. Development Commands

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

# Delegation server (from apps/delegation-server/)
npm run build               # Build TypeScript
npm run dev                # Run with auto-reload
npm run test               # Run tests
npm run proto:build        # Generate gRPC code
```

## High-Level Architecture

### System Components

1. **Main API (Go)** - Core business logic in `/apps/api/cmd/main`
   - AWS Lambda deployment with Gin framework
   - Handles authentication, CRUD operations, and orchestrates services
   - Entry point: `/apps/api/cmd/main/main.go`

2. **Delegation Server (Node.js)** - Blockchain operations in `/apps/delegation-server`
   - gRPC server for MetaMask delegation operations
   - Manages smart account creation and transaction signing
   - Communicates with main API via gRPC

3. **Subscription Processor (Go)** - Background job in `/apps/subscription-processor/cmd`
   - Processes recurring subscription payments
   - Uses stored delegation credentials
   - Runs periodically to check and process due subscriptions

4. **Database Layer** - PostgreSQL with SQLC
   - Type-safe SQL through SQLC code generation
   - Schema in `/libs/go/db/schema.sql`
   - Queries in `/libs/go/db/queries/`

### Key Architectural Patterns

1. **Clean Architecture**
   - Database queries isolated in SQLC-generated code (`/libs/go/db`)
   - Business logic in handlers (`/apps/api/handlers`)
   - External services abstracted in client packages (`/libs/go/client`)

2. **Multi-Service Communication**
   - Main API ↔ Delegation Server: gRPC
   - Services → Database: Direct PostgreSQL connections
   - External APIs: REST clients (Circle, Stripe, etc.)

3. **Authentication Flow**
   - JWT tokens for user authentication (Web3Auth integration)
   - API keys for service-to-service communication
   - Middleware in `libs/go/auth/middleware.go`
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
   - Copy `apps/delegation-server/.env.example` to `apps/delegation-server/.env`
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

## Feature Development Process

When implementing new features, follow the standardized workflow documented in `/docs/feature-development-workflow.md`. The process ensures:

1. **Backend First**: Implement and test API endpoints
2. **Frontend Integration**: Build UI components with proper error handling
3. **E2E Testing**: Verify complete user flows with Playwright
4. **Documentation**: Update all relevant documentation

Key principles:
- Always implement security measures (validation, rate limiting, authentication)
- Use correlation IDs for request tracing
- Include comprehensive error handling
- Write E2E tests for critical user flows
- No feature is complete without passing all quality checks

### Task Completion Criteria

Every task must meet these criteria before being marked complete:
1. No breaking changes to existing functionality
2. No new bugs introduced
3. API builds successfully (`go build`)
4. Frontend builds successfully (`npm run build`)
5. No linting issues (`make lint` and `npm run lint`)
6. Backwards compatible with existing clients
7. E2E tests pass for the feature
8. Documentation is updated

### Common Development Tasks

#### Adding a New API Endpoint
1. Define validation rules in `/libs/go/middleware/validation.go`
2. Create handler in `/apps/api/handlers/`
3. Add route in `/apps/api/server/server.go`
4. Apply appropriate middleware (auth, rate limiting, validation)
5. Update Swagger documentation
6. Test with curl or Postman

#### Creating a Frontend Feature
1. Create API route handlers in `/apps/web-app/src/app/api/`
2. Build components in `/apps/web-app/src/components/`
3. Add types in `/apps/web-app/src/types/`
4. Implement error handling with correlation IDs
5. Create E2E test in `/apps/web-app/tests/e2e/`
6. Run `npm run test:e2e` to verify

#### Security Considerations
- Hash sensitive data (API keys, passwords) with bcrypt
- Validate all inputs on both frontend and backend
- Apply rate limiting to sensitive endpoints
- Use proper authentication middleware
- Log security events with correlation IDs
- Never expose secrets in logs or responses

## 6. GitHub Workflow & Integration

### Creating Issues and PRs
- Use the `gh` command via Bash tool for GitHub operations
- When creating issues, provide clear titles and detailed descriptions
- Link PRs to relevant issues
- Follow commit message conventions with descriptive messages

### Pull Request Process
1. Create feature branch from main
2. Implement changes following the workflow above
3. Ensure all tests pass
4. Update documentation
5. Create PR with comprehensive description
6. Request review when ready

### GitHub Actions Integration
- Automated testing runs on PR creation
- Code coverage reports generated automatically
- Use GitHub Actions for CI/CD workflows

## 7. Debugging & Testing Strategy

### Backend Debugging
- Use structured logging with correlation IDs
- Check logs with: `docker logs <container-name>`
- Use delve debugger for Go: `dlv debug`
- Analyze database queries with pgAdmin or psql

### Frontend Debugging
- Use browser DevTools for debugging
- Check correlation IDs in network tab
- Use React Developer Tools for component inspection
- Analyze bundle size with `npm run build:analyze`

### E2E Testing with Playwright
```bash
# Visual debugging
npm run test:e2e:debug

# Generate new tests
npm run test:e2e:codegen

# Run specific test file
npx playwright test api-keys.spec.ts

# Use screenshots for UI verification
npx playwright test --screenshot=on
```

### Integration Testing
- Ensure all services are running
- Use test database for isolation
- Mock external services when needed
- Clean up test data after runs

## 8. Custom Tools & MCP Integration

### Available MCP Tools
- **Playwright MCP**: Use for visual iteration and UI testing
- **GitHub MCP**: Integrated GitHub operations
- **Task Tool**: Complex codebase searches and analysis

### Custom Scripts
```bash
# Reset database with test data
./scripts/reset-database.sh

# Generate API documentation
make swag

# Run all quality checks
make check-all  # Runs lint, test, and build

# Deploy to staging
make deploy-staging
```

### Leveraging Tools Effectively
1. **Codebase Q&A**: Start here when exploring new areas of the codebase
2. **Parallel Operations**: Run multiple Claude sessions for complex tasks
3. **Visual Debugging**: Use Playwright MCP for UI/UX iterations
4. **Automated Reviews**: Use GitHub Actions for code review assistance

## 9. AI-Ready Codebase Principles

This codebase is structured to be "AI-ready":

1. **Clear File Organization**: Components are logically grouped
2. **Consistent Naming**: Predictable file and function names
3. **Self-Documenting Code**: Clear variable names and structure
4. **Modular Design**: Small, focused components and functions
5. **Type Safety**: Full TypeScript/Go type coverage
6. **Test Coverage**: Examples of how each component should work

## 10. Quick Reference

### Most Important Files
- `/apps/api/server/server.go` - API route definitions
- `/libs/go/db/schema.sql` - Database schema
- `/apps/web-app/src/app/` - Frontend pages
- `/docs/feature-development-workflow.md` - Development process

### Common Issues & Solutions
- **Build fails**: Check environment variables in `.env`
- **Database connection**: Ensure PostgreSQL is running
- **gRPC errors**: Verify delegation server is running
- **Type errors**: Run `make gen` after schema changes

### Performance Considerations
- Use database indexes for frequently queried fields
- Implement pagination for list endpoints
- Cache expensive operations with Redis
- Monitor response times with correlation IDs