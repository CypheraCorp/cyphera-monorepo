# Make Commands for Cyphera API

This document describes all the Make commands available for testing and development.

## 🧪 **Testing Commands (GitHub Actions Compatible)**

### Primary Commands

```bash
# Run EXACT same tests as GitHub Actions
make test-github-actions
```
**What it does:**
- Unit tests (handler tests): `cd apps/api && go test ./handlers/... -v -race -timeout=30s`
- Integration tests: `go test -tags=integration ./tests/integration/... -v -timeout=30m`
- Delegation server tests: `make delegation-server-test`
- Build verification: `make test-builds`
- Code formatting check: `make test-format`

```bash
# Complete test suite (includes everything)
make test-all
```
**What it does:**
- All the above plus service tests and mock generation

```bash
# Quick test suite (no database, no integration)
make test-quick
```
**What it does:**
- Handler tests + service tests (fast, no database required)

### Specific Test Categories

```bash
# API handler tests (same as GitHub Actions unit tests)
make test-handlers

# Service layer tests
make test-services

# Integration tests with database
make test-integration

# Verify all components build
make test-builds

# Check code formatting
make test-format
```

## 🌐 **Delegation Server Commands**

```bash
# Install dependencies
make delegation-server-setup

# Run TypeScript tests
make delegation-server-test

# Run TypeScript linting
make delegation-server-lint

# Build TypeScript
make delegation-server-build
```

## 🔧 **Development Commands**

```bash
# Generate SQLC database code
make gen

# Generate mocks for all interfaces
make generate-mocks

# Generate all protobuf code
make proto-gen

# Generate Swagger/OpenAPI docs
make swagger-gen
```

## 🐳 **Infrastructure Commands**

```bash
# Reset database to clean state
make db-reset

# Run development environment in Docker
make docker-dev

# Build AWS SAM applications
make sam-build
```

## 💡 **Recommended Workflow**

Before pushing to GitHub:
```bash
make test-github-actions
```

For quick development testing:
```bash
make test-quick
```

For comprehensive local testing:
```bash
make test-all
```

## 🎯 **Command Equivalence to GitHub Actions**

| GitHub Actions Job | Make Command | Description |
|-------------------|--------------|-------------|
| Unit Tests | `make test-handlers` | Handler tests with race detection |
| Integration Tests | `make test-integration` | Tests with PostgreSQL database |
| Delegation Server Tests | `make delegation-server-test` | TypeScript Jest tests |
| Build Verification | `make test-builds` | All components build check |
| Code Quality | `make test-format` | Formatting and linting |
| **Complete Suite** | `make test-github-actions` | **All of the above** |

## 🚀 **Usage Examples**

```bash
# Before pushing changes
make test-github-actions

# Quick development cycle
make test-quick

# Full comprehensive testing
make test-all

# Just test your handler changes
make test-handlers

# Just test service changes
make test-services

# Test specific integration scenarios
make test-integration

# Verify your code builds everywhere
make test-builds

# Check if code is properly formatted
make test-format
```

## ⚠️ **Prerequisites**

Make sure you have:
- Go 1.23+ installed
- Node.js 20+ installed
- Docker running (for integration tests)
- All dependencies installed (`go mod download`, `npm ci`)

## 🔍 **Troubleshooting**

If tests fail:
1. Check if dependencies are installed: `go mod download`
2. Regenerate mocks if needed: `make generate-mocks`
3. Ensure Docker is running for integration tests
4. Check if delegation server dependencies are installed: `make delegation-server-setup`

The commands are designed to give you the exact same feedback that GitHub Actions will provide!