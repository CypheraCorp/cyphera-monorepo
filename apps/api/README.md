# Main API Server

> **Navigation:** [← Root README](../../README.md) | [API Reference →](../../docs/api-reference.md) | [Architecture →](../../docs/architecture.md)

The main API server is the core business logic service of the Cyphera platform, built with Go and the Gin web framework.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Development](#development)
- [API Documentation](#api-documentation)
- [Database Integration](#database-integration)
- [Authentication](#authentication)
- [External Integrations](#external-integrations)
- [Deployment](#deployment)

## Overview

The main API server provides RESTful endpoints for all platform operations including user management, subscription processing, wallet operations, and external service integrations.

### Key Features
- **RESTful API Design** with proper HTTP status codes
- **Multi-tenant Architecture** with workspace isolation
- **JWT & API Key Authentication** with Web3Auth integration
- **Database Operations** via SQLC type-safe queries
- **External Integrations** (Circle API, Delegation Server, Payment Sync)
- **AWS Lambda Deployment** with local development support
- **Comprehensive Logging** with structured output
- **Swagger Documentation** with OpenAPI specs

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │    │   Mobile App    │    │  External API   │
│                 │    │                 │    │   Integration   │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │     Gin HTTP Server     │
                    │   (Authentication)      │
                    └────────────┬────────────┘
                                 │
            ┌────────────────────┼────────────────────┐
            │                    │                    │
   ┌────────▼─────────┐ ┌────────▼─────────┐ ┌───────▼────────┐
   │    Handlers      │ │   Middleware     │ │   Database     │
   │   (Business      │ │  (Auth, CORS,    │ │   (SQLC)       │
   │    Logic)        │ │   Logging)       │ │                │
   └────────┬─────────┘ └──────────────────┘ └────────────────┘
            │
   ┌────────▼─────────┐
   │  External APIs   │
   │ (Circle, Stripe, │
   │ Delegation gRPC) │
   └──────────────────┘
```

### Directory Structure

```
apps/api/
├── cmd/
│   ├── local/          # Local development entry point
│   │   └── main.go
│   └── main/           # Production Lambda entry point
│       └── main.go
├── handlers/           # HTTP request handlers
│   ├── account_handlers.go
│   ├── apikey_handlers.go
│   ├── circle_handlers.go
│   ├── customer_handlers.go
│   ├── product_handlers.go
│   ├── subscription_handlers.go
│   ├── wallet_handlers.go
│   └── workspace_handlers.go
├── server/            # Server configuration
│   └── server.go
├── services/          # Business logic services
│   └── error_recovery_service.go
├── go.mod            # Go module dependencies
└── README.md         # This file
```

## Development

### Prerequisites
- Go 1.21 or later
- PostgreSQL database (via Docker)
- Environment variables configured

### Running Locally

#### Start Database
```bash
# From project root
docker-compose up postgres -d
```

#### Start API Server
```bash
# From project root
npm run dev:api

# Or directly with Go
cd apps/api
go run cmd/local/main.go
```

The server will start on `http://localhost:8080` with hot reload via Air.

#### Environment Variables
Ensure your `.env` file contains:
```bash
# Database
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/cyphera_dev"

# Web3Auth
WEB3AUTH_CLIENT_ID="your_client_id"
WEB3AUTH_CLIENT_SECRET="your_client_secret"

# Circle API
CIRCLE_API_KEY="your_circle_api_key"

# gRPC Services
DELEGATION_GRPC_ADDR="localhost:50051"

# Development
NODE_ENV="development"
LOG_LEVEL="debug"
```

### Testing

#### Unit Tests
```bash
# Run all tests
npm run test:api

# Or directly with Go
cd apps/api
go test -v ./...
```

#### Integration Tests
```bash
# From project root
npm run test:integration
```

#### API Testing
Access Swagger UI at: `http://localhost:8080/swagger/index.html`

## API Documentation

### Swagger/OpenAPI
The API is documented using Swagger annotations in handler files:

```go
// @Summary Get product by ID
// @Description Retrieves a product by its ID
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {object} ProductResponse
// @Failure 404 {object} ErrorResponse
// @Router /products/{product_id} [get]
func GetProduct(c *gin.Context) {
    // Handler implementation
}
```

#### Generate Documentation
```bash
# From project root
npm run generate:swagger

# Or with make
make swag
```

### Key Endpoints

#### Authentication
- `POST /admin/accounts/signin` - Account sign-in/registration
- `GET /health` - Health check endpoint

#### Resources
- `GET /products` - List products
- `POST /products` - Create product
- `GET /customers` - List customers
- `POST /customers` - Create customer
- `GET /subscriptions` - List subscriptions
- `GET /wallets` - List wallets

For complete API documentation, see: [API Reference →](../../docs/api-reference.md)

## Database Integration

### SQLC Integration
The API uses [SQLC](https://sqlc.dev/) for type-safe database operations:

#### Database Schema
Located at: `libs/go/db/schema.sql`

#### Query Files
SQL queries are defined in: `libs/go/db/queries/`

#### Generated Code
Type-safe Go code is generated in: `libs/go/db/`

#### Working with SQLC

1. **Add New Query:**
   ```sql
   -- name: GetProductsByWorkspace :many
   SELECT * FROM products 
   WHERE workspace_id = $1 AND deleted_at IS NULL
   ORDER BY created_at DESC;
   ```

2. **Generate Code:**
   ```bash
   # From project root
   make gen
   ```

3. **Use in Handler:**
   ```go
   products, err := queries.GetProductsByWorkspace(ctx, workspaceID)
   if err != nil {
       c.JSON(500, gin.H{"error": "Failed to fetch products"})
       return
   }
   ```

### Database Migrations
Migrations are managed via the init script: `libs/go/db/init-scripts/01-init.sql`

## Authentication

### Web3Auth JWT
Primary authentication method using Web3Auth's JWKS:

```go
// Middleware validates JWT tokens
func EnsureValidAPIKeyOrToken() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Validate JWT or API key
        // Set user context
    }
}
```

### API Keys
Service-to-service authentication:

```go
// API key validation
apiKey := c.GetHeader("Authorization")
if isValidAPIKey(apiKey) {
    // Set API key context
}
```

### Multi-tenancy
All operations are scoped to workspaces:

```go
workspaceID := c.GetHeader("X-Workspace-ID")
if workspaceID == "" {
    c.JSON(400, gin.H{"error": "Workspace ID required"})
    return
}
```

## External Integrations

### Circle API
Programmable wallet integration:

```go
// Circle client initialization
circleClient := circle.NewClient(apiKey)

// Create wallet
wallet, err := circleClient.CreateWallet(ctx, params)
```

### Delegation Server (gRPC)
Blockchain operations:

```go
// gRPC client
delegationClient := delegation.NewDelegationServiceClient(conn)

// Redeem delegation
response, err := delegationClient.RedeemDelegation(ctx, request)
```

### Payment Sync
Stripe and other payment provider integration:

```go
// Payment sync client
syncClient := paymentsync.NewClient(config)

// Sync customer data
err := syncClient.SyncCustomer(customerID)
```

## Deployment

### AWS Lambda
The API deploys as AWS Lambda functions:

#### Production Entry Point
`cmd/main/main.go` - Lambda handler

#### Build for Lambda
```bash
# From project root
npm run build:api

# Or with make
make build
```

#### Environment Variables
Production environment variables are managed via AWS Secrets Manager.

### Local Development
Use `cmd/local/main.go` for local development with hot reload.

#### Development Server
```bash
# Start with Air hot reload
npm run dev:api

# Direct Go run
cd apps/api && go run cmd/local/main.go
```

## Logging

### Structured Logging
The API uses structured logging with configurable levels:

```go
log.WithFields(log.Fields{
    "user_id": userID,
    "action": "create_product",
    "workspace_id": workspaceID,
}).Info("Product created successfully")
```

### Log Levels
- `DEBUG` - Development debugging
- `INFO` - General information
- `WARN` - Warning conditions
- `ERROR` - Error conditions

## Error Handling

### Standard Error Responses
```go
// Database errors
func handleDatabaseError(c *gin.Context, err error, resource string) {
    if errors.Is(err, pgx.ErrNoRows) {
        c.JSON(404, gin.H{"error": fmt.Sprintf("%s not found", resource)})
        return
    }
    c.JSON(500, gin.H{"error": "Internal server error"})
}

// Validation errors
func handleValidationError(c *gin.Context, err error) {
    c.JSON(400, gin.H{"error": err.Error()})
}
```

### Error Context
All errors are logged with context:

```go
log.WithFields(log.Fields{
    "error": err.Error(),
    "request_id": requestID,
    "user_id": userID,
}).Error("Failed to process request")
```

## Performance Considerations

### Database Connection Pooling
SQLC uses connection pooling for optimal performance:

```go
// Connection pool configuration
config.MaxConns = 25
config.MinConns = 5
config.MaxConnLifetime = time.Hour
```

### Request Caching
Consider implementing Redis caching for frequently accessed data.

### Response Compression
Gin middleware for response compression:

```go
r.Use(gzip.Gzip(gzip.DefaultCompression))
```

---

## Related Documentation

- **[API Reference](../../docs/api-reference.md)** - Complete endpoint documentation
- **[Architecture Guide](../../docs/architecture.md)** - System overview
- **[Database Documentation](../../libs/go/db/README.md)** - Database schema and queries
- **[Deployment Guide](../../docs/deployment.md)** - Production deployment

## Need Help?

- **[Troubleshooting](../../docs/troubleshooting.md)** - Common issues
- **[Contributing](../../docs/contributing.md)** - Development workflow
- **GitHub Issues** - Bug reports and feature requests

---

*Last updated: $(date '+%Y-%m-%d')*
*Service Version: 2.0.0*