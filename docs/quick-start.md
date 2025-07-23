# Quick Start Guide

> **Navigation:** [← Architecture](architecture.md) | [↑ README](../README.md) | [API Reference →](api-reference.md)

Get Cyphera running locally in under 10 minutes.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Environment Setup](#environment-setup)  
- [Database Setup](#database-setup)
- [Running Services](#running-services)
- [Verification](#verification)
- [Next Steps](#next-steps)

## Prerequisites

Ensure you have the following installed:

- **Node.js** ≥ 18.0.0 ([Download](https://nodejs.org/))
- **Go** ≥ 1.21 ([Download](https://golang.org/dl/))
- **Docker** ([Download](https://docs.docker.com/get-docker/))
- **Git** ([Download](https://git-scm.com/downloads))

Verify installations:
```bash
node --version  # Should be ≥ 18.0.0
go version      # Should be ≥ 1.21  
docker --version
```

## Installation

### 1. Clone Repository
```bash
git clone https://github.com/your-org/cyphera-api.git
cd cyphera-api
```

### 2. Install Dependencies
```bash
# Install all dependencies (Go + Node.js)
npm run install:all

# Alternative: Install separately
npm run install:go   # Go workspace sync
npm run install:ts   # TypeScript apps with legacy peer deps
```

### 3. Generate Code
```bash
# Generate database code and gRPC protobuf files
npm run generate:all

# Or individually:
make gen           # Database code (SQLC)
make proto-build-all  # gRPC code
```

## Environment Setup

### 1. Root Environment
Copy the template and configure:
```bash
cp .env.example .env
```

Edit `.env` with your settings:
```bash
# Database
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/cyphera_dev"

# Web3Auth (get from https://web3auth.io/)
WEB3AUTH_CLIENT_ID="your_client_id"
WEB3AUTH_CLIENT_SECRET="your_client_secret"

# Circle API (get from https://console.circle.com/)
CIRCLE_API_KEY="your_circle_api_key"

# Development settings
NODE_ENV="development"
LOG_LEVEL="debug"
```

### 2. Web App Environment  
```bash
cp apps/web-app/.env.example apps/web-app/.env.local
```

Configure web app settings:
```bash
# Web3Auth
NEXT_PUBLIC_WEB3AUTH_CLIENT_ID="your_client_id"

# API endpoints
NEXT_PUBLIC_API_URL="http://localhost:8080"
NEXT_PUBLIC_DELEGATION_SERVER_URL="http://localhost:50051"

# Development
NEXT_PUBLIC_NODE_ENV="development"
```

### 3. Delegation Server Environment
```bash
cp apps/delegation-server/.env.example apps/delegation-server/.env
```

Configure delegation server:
```bash
# Blockchain RPC URLs
ETHEREUM_RPC_URL="https://eth-sepolia.g.alchemy.com/v2/your_key"
POLYGON_RPC_URL="https://polygon-mumbai.g.alchemy.com/v2/your_key"

# Private keys (use test keys only!)
DELEGATION_PRIVATE_KEY="0x1234...your_test_private_key"

# gRPC Settings
GRPC_PORT="50051"
```

## Database Setup

### 1. Start PostgreSQL
```bash
# Using Docker Compose
docker-compose up postgres -d

# Verify database is running
docker-compose ps
```

### 2. Run Migrations
```bash
# Apply database schema
make db-migrate

# Verify tables were created
psql $DATABASE_URL -c "\dt"
```

### 3. Optional: Seed Data
```bash
# Add sample data for development
make db-seed
```

## Running Services

### Option 1: All Services Together
```bash
# Start all services with one command
npm run dev:all
```

This starts:
- **Main API** on `http://localhost:8080`
- **Web App** on `http://localhost:3000`  
- **Delegation Server** on `http://localhost:50051`

### Option 2: Individual Services

#### Main API Server
```bash
npm run dev:api
# Runs on http://localhost:8080
```

#### Web Application
```bash
npm run dev:web
# Runs on http://localhost:3000
```

#### Delegation Server
```bash
npm run dev:delegation
# Runs on http://localhost:50051 (gRPC)
```

#### Subscription Processor
```bash
npm run dev:subscription
# Background service, no web interface
```

## Verification

### 1. Check Service Health
```bash
# API Server
curl http://localhost:8080/health

# Expected response:
# {"status":"ok","timestamp":"2024-01-01T00:00:00Z"}
```

### 2. Access Web Interface
Open your browser to:
- **Main App:** http://localhost:3000
- **Merchant Dashboard:** http://localhost:3000/merchants
- **Customer Portal:** http://localhost:3000/customers

### 3. Test Authentication
1. Go to http://localhost:3000/merchants/signin
2. Click "Login with Web3Auth"
3. Use a test email (Google/Discord/etc.)
4. Verify you reach the merchant dashboard

### 4. Check Delegation Server
```bash
# Test gRPC health (requires grpcurl)
grpcurl -plaintext localhost:50051 list

# Or check logs for "gRPC server listening on :50051"
```

## Next Steps

### Development Workflow
1. **Make Changes:** Edit code in your preferred editor
2. **Hot Reload:** Services automatically reload on changes
3. **Test:** Run `npm run test:all` before committing
4. **Lint:** Run `npm run lint` to check code quality

### Key Development Commands
```bash
# Testing
npm run test:all          # Run all tests
npm run test:api          # API tests only  
npm run test:web          # Web app tests only

# Code Quality
npm run lint              # Lint all TypeScript
npm run format            # Format code with Prettier
npm run typecheck         # TypeScript type checking

# Database
npm run db:reset          # Reset database
npm run db:migrate        # Apply new migrations
make gen                  # Regenerate SQLC code

# Building
npm run build:all         # Build all services
npm run build:web         # Build web app only
npm run build:api         # Build API only
```

### Recommended Development Setup
1. **IDE:** VS Code with Go and TypeScript extensions
2. **API Testing:** Postman or Insomnia with OpenAPI spec
3. **Database:** pgAdmin or TablePlus for database management
4. **Blockchain:** MetaMask with test networks configured

### Troubleshooting Common Issues

**Port Already in Use:**
```bash
# Find process using port 3000
lsof -ti:3000
# Kill the process
kill -9 <PID>
```

**Database Connection Issues:**
```bash
# Check if PostgreSQL is running
docker-compose ps postgres
# View logs
docker-compose logs postgres
```

**Web3Auth Issues:**
- Verify client ID is correct in both `.env` files
- Check browser console for authentication errors
- Ensure you're using the correct redirect URLs

**Dependencies Issues:**
```bash
# Clean and reinstall
npm run clean:deps
npm run install:all
```

---

## What's Next?

- **[Architecture Overview](architecture.md)** - Understand the system design
- **[API Reference](api-reference.md)** - Explore available endpoints
- **[Web App Guide](../apps/web-app/README.md)** - Frontend development
- **[Deployment Guide](deployment.md)** - Production deployment

## Need Help?

- **[Troubleshooting Guide](troubleshooting.md)** - Common issues and solutions
- **[Contributing Guide](contributing.md)** - How to contribute to the project
- **GitHub Issues** - Report bugs or request features

---

*Last updated: $(date '+%Y-%m-%d')*