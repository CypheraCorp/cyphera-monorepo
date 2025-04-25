# Cyphera API

Cyphera API is a comprehensive backend system that enables blockchain-based subscription and payment processing. It consists of multiple integrated components that work together to provide a complete solution for managing cryptocurrency-based transactions, delegations, and subscriptions.

## Documentation

Detailed documentation on the architecture, workflows, and components of the Cyphera API can be found in the [docs/README.md](docs/README.md) file. The documentation includes:

- System architecture diagrams
- Component flow explanations
- Database schema overview
- Request processing workflows
- Subscription and payment processes
- Delegation system details
- API reference information

## System Overview

The Cyphera API ecosystem consists of four main components:

1. **Main API** (`/cmd/api/main`)
   - Core API service written in Go
   - Operates as an AWS Lambda function in production
   - Can run as a standalone HTTP server locally
   - Handles authentication, database operations, and business logic

2. **Delegation Server** (`/delegation-server`)
   - Node.js gRPC server for MetaMask delegation operations
   - Handles blockchain interactions for delegation redemption
   - Can operate in mock mode for local testing
   - Communicates with the main API via gRPC

3. **Subscription Processor** (`/cmd/subscription-processor`)
   - Background service that processes recurring subscription payments
   - Identifies subscriptions due for renewal
   - Uses stored delegation credentials for payment processing
   - Updates subscription records and logs events

4. **PostgreSQL Database**
   - Stores all application data including user accounts, workspaces, products, and subscriptions
   - Manages delegation data and transaction records
   - Runs locally via Docker for development
   - Used by all three other components

5. **Circle API Integration**
   - Provides seamless integration with Circle's programmable wallets
   - Automatically synchronizes Circle data with our database
   - Maintains a caching layer for improved performance
   - Supports multiple blockchain networks via Circle's infrastructure

```
                ┌───────────────────┐         ┌────────────────────┐
                │                   │         │                    │
 ┌─────────┐    │    Go API         │ gRPC    │   Node.js Server   │
 │         │    │  ┌─────────────┐  │ call    │  ┌──────────────┐  │
 │  User   ├────┼──┤ HTTP Routes ├──┼─────────┼──┤ gRPC Service │  │
 │         │    │  └─────┬───────┘  │         │  └──────┬───────┘  │
 └─────────┘    │        │          │         │         │          │
                │        ▼          │         │         ▼          │
                │  ┌─────────────┐  │         │  ┌──────────────┐  │
                │  │ Delegation  │  │         │  │  Blockchain  │  │
                │  │   Client    │  │         │  │  Operations  │  │         ┌─────────────┐
                │  └─────────────┘  │         │  └──────┬───────┘  │         │             │
                │                   │         │         │          │         │ Blockchain  │
                └───────────────────┘         │         ▼          │         │             │
                                              │  ┌──────────────┐  │         │  ┌───────┐  │
                ┌───────────────────┐         │  │Smart Account │  │user op  │  │       │  │
                │                   │         │  │  Creation    ├──┼─────────┼─►│ ERC20 │  │
                │  Subscription     │         │  └──────────────┘  │         │  │       │  │
                │   Processor       │         │                    │         │  └───────┘  │
                │                   │         └────────────────────┘         │             │
                └───────────────────┘                                        └─────────────┘
                          │
                          │
                          ▼
                ┌───────────────────┐
                │                   │
                │    PostgreSQL     │
                │    Database      │
                │                   │
                └───────────────────┘
```

## Installation and Setup

### Prerequisites

- Go 1.21 or later
- Node.js 18 or later
- Docker and Docker Compose
- PostgreSQL 14 or later (Docker setup provided)

### Environment Setup

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd cyphera-api
   ```

2. **Configure Local Development Environment (`.env`):**
   Copy the template file:
   ```bash
   cp .env.template .env
   ```
   *   Edit the `.env` file.
   *   Set `DATABASE_URL` to point to your local PostgreSQL instance (e.g., the one started by `docker-compose`). The default `postgres://apiuser:apipassword@localhost:5432/cyphera?sslmode=disable` often works if using the provided `docker-compose.yml`.
   *   Update `SUPABASE_URL`, `SUPABASE_JWT_SECRET`, `CYPHERA_SMART_WALLET_ADDRESS`, `CIRCLE_API_KEY`, and other necessary variables for your local setup.
   *   **Important:** The `DATABASE_URL` in this file is **ONLY** used for local development runs (like `make dev` or `make dev-all`). Deployed environments (dev/prod in AWS Lambda) fetch credentials securely from AWS Secrets Manager.

3. **Configure Delegation Server Local Environment:**
   ```bash
   cp delegation-server/.env.example delegation-server/.env
   ```
   *   Edit `delegation-server/.env` and set necessary variables, including `NPM_TOKEN` if required for private dependencies.

4. **Configure Deployment Secrets (GitHub Actions):**
   For deployments via GitHub Actions, configure the following secrets in your GitHub repository settings (`Settings > Secrets and variables > Actions`):
   *   `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`: For deploying to AWS.
   *   `SUPABASE_URL`, `SUPABASE_JWT_SECRET`: For Supabase integration.
   *   `CYPHERA_SMART_WALLET_ADDRESS`: Smart contract address.
   *   `CIRCLE_API_KEY`: Circle API credentials.
   *   `CORS_ALLOWED_ORIGINS`, `CORS_ALLOWED_METHODS`, etc.: CORS settings for deployed environments.
   *   `NPM_TOKEN` (if needed by delegation server build/setup).
   *   **Note:** Database credentials (`DATABASE_URL` with password) are **NOT** stored here. They are managed by AWS Secrets Manager via Terraform.

5. **Install dependencies:**
   ```bash
   make install
   ```
   This command will install Go dependencies and set up the delegation server.

## Running the Application

### Running All Components At Once

To run all components (API, delegation server, and subscription processor) together:

```bash
make dev-all
```

This command:
- Starts the PostgreSQL database if not already running
- Launches the delegation server
- Starts the main API server
- Runs the subscription processor

### Starting Just The Database

If you just want to run the PostgreSQL database by itself then run the following:

```bash
docker-compose up postgres
```

This will start a PostgreSQL instance with the necessary schema loaded.

### Starting The Delegation Server

If you want to just run the delegation server individually, then run the following:

```bash
make delegation-server
```

### Starting the Cyphera API

If you want to just run the cyphera api individually you first need to make sure that the delegation-server is running, then run the following:

```bash
make dev
```
update the parameters int he make file to change the time interval

### Verifying the Cyphera API Setup

Once all components are running, you can verify the API is working by making a request to the health endpoint:

```bash
curl http://localhost:8000/health
```

If the api is set up correctly, you'll receive a successful response.

### Starting the Subscription Processor

If you want to just run the subscription processor individually, then run the following:

```bash
make subscription-processor
```
update the parameters int he make file to change the time interval


## Developing with Cyphera API

### Development Lifecycle

1. **Making code changes:**
   - The project uses hot reloading during development with Air for Go code
   - API changes will automatically restart the server
   - Delegation server uses nodemon to watch for changes

2. **Testing changes:**
   - Unit tests: `make test`
   - Integration tests: `make test-integration`
   - Full test suite: `make test-all`

3. **Committing changes:**
   - The project uses Git hooks for linting and formatting
   - Run `make lint` to check for issues before committing

### API Documentation with Swagger

The API is documented using Swagger/OpenAPI. To generate or update the Swagger documentation:

```bash
make swagger
```

Once the API is running, you can access the Swagger UI at:
```
http://localhost:8000/swagger/index.html
```

Swagger annotations are added to handlers in the `internal/handlers` directory using comments starting with `// @`.

### Authentication and Authorization

The API uses two authentication methods:

1. **JWT Token Authentication**
   - Used for user-based authentication
   - Tokens are validated in the `auth.EnsureValidAPIKeyOrToken` middleware
   - User roles and permissions are enforced in handlers

2. **API Key Authentication**
   - Used for service-to-service and programmatic access
   - Keys can have different access levels (read, write, admin)
   - Each API key is associated with a specific workspace

The authentication middleware is defined in `internal/auth/middleware.go` and is applied to routes that require authentication.

### Database Operations with SQLC

The project uses [sqlc](https://sqlc.dev/) to generate type-safe Go code from SQL queries.

#### Database Schema

The database schema is defined in `internal/db/init-scripts/01-init.sql`, which includes tables for:
- Users and accounts
- Workspaces
- Products and tokens
- Subscriptions and subscription events
- Customer and wallet information
- API keys and delegation data

#### Working with SQLC

1. **Writing queries:**
   - Create or modify SQL query files in `internal/db/queries/`
   - Follow the existing pattern for query naming and structure

2. **Generating Go code:**
   ```bash
   make gen
   ```
   This command runs sqlc to generate Go files based on your SQL queries.

3. **Using generated code:**
   - Import the db package: `import "cyphera-api/internal/db"`
   - Create a database connection
   - Use the generated methods to interact with the database

Example of using the generated code in a handler:

```go
// Get a user by ID
user, err := queries.GetUser(ctx, userID)
if err != nil {
    // Handle error
}

// Create a new user
newUser, err := queries.CreateUser(ctx, db.CreateUserParams{
    Name:  "John Doe",
    Email: "john@example.com",
    Role:  db.RoleUser,
})
```

### Server Handlers

The API routes and handlers are defined in the `internal/handlers` directory, organized by resource type. Each handler file contains functions that handle specific API endpoints.

Handlers follow a consistent pattern:
1. Extract parameters from the request
2. Validate input data
3. Perform database operations or call other services
4. Return an appropriate response

Example handler structure:

```go
// @Summary Get product by ID
// @Description Retrieves a product by its ID
// @Tags products
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {object} ProductResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /products/{product_id} [get]
func GetProduct(c *gin.Context) {
    // Extract product ID from request
    productID := c.Param("product_id")
    
    // Convert to UUID
    id, err := uuid.Parse(productID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID format"})
        return
    }
    
    // Query the database
    product, err := queries.GetProduct(c.Request.Context(), id)
    if err != nil {
        // Handle error (not found, server error, etc.)
        handleDatabaseError(c, err, "product")
        return
    }
    
    // Return the product
    c.JSON(http.StatusOK, formatProductResponse(product))
}
```

## Component Details

### Main API

The main API is a Go application built with the Gin web framework. It provides HTTP endpoints for all operations and internally communicates with the delegation server and database.

Key features:
- RESTful API design with proper status codes and error handling
- JWT and API key authentication
- Rate limiting and request validation
- Swagger documentation
- Structured logging

### Delegation Server

The delegation server is a Node.js application that handles blockchain-related operations for delegations:

- Implements a gRPC service defined in protocol buffers
- Handles delegation redemption and validation
- Interacts with blockchain networks through Web3 providers
- Supports both production and mock modes for testing

### Subscription Processor

The subscription processor is a Go application that runs either on a schedule or continuously to process subscription renewals:

- Identifies subscriptions that need processing
- Processes payments using stored delegation credentials
- Records subscription events and handles failures
- Provides detailed logs for troubleshooting

### Circle API Integration

The Circle API integration provides programmable wallet functionality through Circle's infrastructure:

- **Automatic Caching**: All Circle users, wallets, and balances are automatically stored in the database, creating a caching layer
- **Network Support**: Compatible with multiple blockchain networks supported by Circle (Ethereum, Polygon, Arbitrum, Base, Solana)
- **Synchronization**: Database records are kept in sync with Circle's data whenever API calls are made
- **Performance**: Reduces redundant API calls to Circle by storing wallet and user data locally
- **Transparent Proxy**: Acts as a transparent proxy to Circle's API while adding persistence and caching

Key features:
- User management with automatic database synchronization
- Wallet creation and management across multiple blockchains
- Balance retrieval with local caching
- PIN management for secure operations
- Challenge handling for secure wallet operations

### Database Structure

The PostgreSQL database is structured with the following main tables:

- `accounts` - Organization accounts and settings
- `users` - User information and authentication
- `workspaces` - Organizational units within accounts
- `products` - Subscription products offered by workspaces
- `subscriptions` - Active and historical subscriptions
- `tokens` - Supported cryptocurrency tokens
- `networks` - Blockchain networks and connection details
- `wallets` - Cryptocurrency wallet information
- `delegation_data` - Stored delegation credentials
- `api_keys` - API keys for programmatic access
- `circle_users` - Circle user information and tokens
- `circle_wallets` - Circle wallet details linked to local wallets

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Ensure PostgreSQL is running: `docker ps | grep postgres`
   - Check your `DATABASE_URL` in the `.env` file
   - Verify the database schema has been initialized

2. **Delegation Server Connection Issues**
   - Verify the delegation server is running: `make delegation-server`
   - Check `DELEGATION_GRPC_ADDR` in your `.env` file
   - Look for gRPC connection errors in the logs

3. **NPM Token Issues**
   - If delegation server setup fails with npm errors, check the NPM_TOKEN in delegation-server/.env
   - Ensure you have the correct permissions to access private packages

4. **API Return Errors**
   - Check the server logs for detailed error information
   - Verify authentication credentials are correct
   - Ensure the requested resource exists

For more detailed component-specific troubleshooting, refer to the documentation in the `docs/` directory.