# Local Development Guide - Nx Monorepo

This guide walks you through running all Cyphera services locally using Nx commands.

## Prerequisites

### 1. Environment Setup
```bash
# Copy environment templates
cp .env.template .env
cp delegation-server/.env.example delegation-server/.env
cp apps/web-app/.env.example apps/web-app/.env.local  # If exists
```

### 2. Install Dependencies
```bash
# Install all dependencies (Go, Node.js, etc.)
make install

# Or manually:
go mod tidy
go work sync
cd apps/web-app && npm install --legacy-peer-deps
cd delegation-server && npm install
```

### 3. Database Setup
```bash
# Start PostgreSQL using Docker
docker-compose up -d postgres

# Wait for it to be ready
docker-compose ps

# Run database migrations (if needed)
make gen  # Generate SQLC code
```

## Running Services with Nx

### Individual Services

#### 1. API Server (Go) - Port 8080
```bash
# Using Nx
npx nx serve api

# Using Make
make dev

# What it does:
# - Starts the Go API server with hot reload (using Air)
# - Connects to PostgreSQL database
# - Serves on http://localhost:8080
```

#### 2. Web App (Next.js) - Port 3000
```bash
# Using Nx
npx nx serve web-app

# Using Make
make web-dev

# What it does:
# - Starts Next.js dev server with hot reload
# - Connects to API at localhost:8080
# - Serves on http://localhost:3000
```

#### 3. Delegation Server (Node.js) - Port 50051 (gRPC)
```bash
# Using Nx
npx nx serve delegation-server

# Using Make
make delegation-server

# What it does:
# - Starts gRPC server for blockchain operations
# - Handles MetaMask delegation toolkit
# - Listens on localhost:50051
```

### Running All Services Together

#### Option 1: Sequential Start (Recommended for first time)
```bash
# 1. Start database
docker-compose up -d postgres

# 2. Start delegation server (in terminal 1)
npx nx serve delegation-server

# 3. Start API server (in terminal 2)
npx nx serve api

# 4. Start web app (in terminal 3)
npx nx serve web-app
```

#### Option 2: Parallel Execution with Nx
```bash
# Run all services at once
npx nx run-many --target=serve --projects=api,web-app,delegation-server --parallel=3

# Or use the Make command
make dev-all
```

#### Option 3: Using PM2 (Process Manager)
```bash
# Install PM2 globally
npm install -g pm2

# Start all services
pm2 start ecosystem.config.js

# View logs
pm2 logs

# Stop all
pm2 stop all
```

## Service Dependencies

```mermaid
graph LR
    Web[Web App :3000] --> API[API Server :8080]
    API --> DB[(PostgreSQL :5432)]
    API --> DS[Delegation Server :50051]
    Web --> DS
```

## Checking Service Health

### 1. API Server
```bash
# Health check
curl http://localhost:8080/health

# Swagger docs
open http://localhost:8080/swagger/index.html
```

### 2. Web App
```bash
# Open in browser
open http://localhost:3000

# Check Next.js build
npx nx build web-app
```

### 3. Delegation Server
```bash
# Check gRPC service (requires grpcurl)
grpcurl -plaintext localhost:50051 list

# Check logs
npx nx run delegation-server:logs
```

### 4. Database
```bash
# Check PostgreSQL
docker-compose exec postgres psql -U postgres -c "\l"

# View tables
docker-compose exec postgres psql -U postgres -d cyphera -c "\dt"
```

## Common Nx Commands

### Building
```bash
# Build all projects
npx nx run-many --target=build --all

# Build specific project
npx nx build api
npx nx build web-app

# Build only affected projects
npx nx affected --target=build
```

### Testing
```bash
# Test all projects
npx nx run-many --target=test --all

# Test specific project
npx nx test api
npx nx test web-app

# Test only affected
npx nx affected --target=test
```

### Linting
```bash
# Lint all
npx nx run-many --target=lint --all

# Lint specific
npx nx lint web-app
```

### Other Useful Commands
```bash
# View dependency graph
npx nx graph

# Reset Nx cache
npx nx reset

# Show all projects
npx nx show projects

# Run SQLC generation
npx nx run go-shared:generate
```

## Troubleshooting

### Port Already in Use
```bash
# Find process using port 8080 (API)
lsof -i :8080
kill -9 <PID>

# Find process using port 3000 (Web)
lsof -i :3000
kill -9 <PID>

# Find process using port 50051 (gRPC)
lsof -i :50051
kill -9 <PID>
```

### Database Connection Issues
```bash
# Check if PostgreSQL is running
docker-compose ps

# Restart PostgreSQL
docker-compose restart postgres

# View PostgreSQL logs
docker-compose logs postgres
```

### Module Not Found Errors
```bash
# For Go modules
go work sync
go mod tidy

# For Node modules
cd apps/web-app && npm install --legacy-peer-deps
cd delegation-server && npm install
```

### Nx Cache Issues
```bash
# Clear Nx cache
npx nx reset

# Clean and rebuild
npx nx run-many --target=clean --all
npx nx run-many --target=build --all
```

## Environment Variables

### API Server (.env)
```bash
DATABASE_URL=postgres://username:password@localhost:5432/database_name?sslmode=disable
CIRCLE_API_KEY=your_circle_api_key
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

### Web App (apps/web-app/.env.local)
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_CIRCLE_APP_ID=your_app_id
# Add other web-specific env vars
```

### Delegation Server (delegation-server/.env)
```bash
NODE_ENV=development
GRPC_PORT=50051
# Add delegation-specific env vars
```

## Development Workflow

1. **Start Services**: Use either individual commands or `make dev-all`
2. **Make Changes**: Edit code - hot reload will pick up changes
3. **Test Changes**: Run tests with `nx test <project>`
4. **Build**: Verify builds work with `nx build <project>`
5. **Commit**: Follow conventional commits

## VS Code Integration

### Recommended Extensions
- Nx Console
- Go
- ESLint
- Prettier
- Thunder Client (API testing)

### Launch Configurations
Add to `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "API Server",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/apps/api/cmd/local/main.go"
    },
    {
      "name": "Next.js: debug client-side",
      "type": "chrome",
      "request": "launch",
      "url": "http://localhost:3000"
    }
  ]
}
```

## Quick Start Summary

```bash
# 1. Setup (first time only)
make install
docker-compose up -d postgres

# 2. Run everything
make dev-all

# 3. Access services
# - Web App: http://localhost:3000
# - API: http://localhost:8080
# - API Docs: http://localhost:8080/swagger/index.html

# 4. Stop everything
# Ctrl+C in each terminal or:
docker-compose down
```