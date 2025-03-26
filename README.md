# Cyphera API

Cyphera API is a Go-based API service designed to run as an AWS Lambda function in production and as a standalone server for local development. It works in conjunction with a dedicated Node.js delegation server that handles MetaMask delegation redemption operations.

## System Architecture

The project consists of two main components:

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
                                              │  │Smart Account │  │user op  │  │       │  │
                                              │  │  Creation    ├──┼─────────┼─►│ ERC20 │  │
                                              │  └──────────────┘  │         │  │       │  │
                                              │                    │         │  └───────┘  │
                                              └────────────────────┘         │             │
                                                                             └─────────────┘
```

## Environment Setup

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd cyphera-api
   ```

2. **Configure environment variables:**
   ```bash
   cp .env.template .env
   ```
   
3. **Edit the `.env` file with your configuration:**
   Required variables include:
   - `DATABASE_URL`: PostgreSQL connection string
   - `DELEGATION_SERVER_URL`: URL for the delegation server
   - `DELEGATION_LOCAL_MODE`: Set to "true" for local development
   - `DELEGATION_GRPC_ADDR`: gRPC address for the delegation server (e.g., "localhost:50051")
   - Additional variables for Auth0, CORS settings, etc.

## Running the Project

### Development Mode

To run both the API and delegation server locally:

```bash
make dev
```

This command:
- Loads environment variables from `.env`
- Starts the delegation server in mock mode
- Starts the API server
- Sets up appropriate error handling and shutdown processes

### Running Individual Components

**API Server only:**
```bash
make api-server
```

**Delegation Server only:**
```bash
make delegation-server
```

### Docker Compose (with PostgreSQL)

For a complete local environment with PostgreSQL:

```bash
docker compose up
```

## Testing

### Running Unit Tests

```bash
make test
```

### Running Integration Tests

```bash
make test-integration
```

This will:
- Start the delegation server in mock mode
- Run integration tests with the delegation system
- Clean up processes after tests complete

### Manual API Testing

Once the servers are running, you can test the API:

```bash
curl -X GET 'http://localhost:8000/health'
```

## API Documentation

Swagger documentation is available when the server is running:

```
http://localhost:8000/swagger/index.html
```

To update the Swagger documentation:

```bash
make swagger
```

## Project Structure

```
├── cmd/
│   └── api/
│       ├── local/      # Local server implementation
│       └── main/       # AWS Lambda implementation
├── delegation-server/  # Node.js gRPC server for delegations
├── docs/               # Documentation files
├── internal/           # Internal Go packages
│   ├── auth/           # Authentication logic
│   ├── client/         # External service clients
│   ├── db/             # Database models and queries
│   ├── handlers/       # API route handlers
│   ├── logger/         # Logging utilities
│   ├── server/         # HTTP server setup
│   └── proto/          # Protocol buffer definitions
├── scripts/            # Helper scripts
├── .env                # Environment variables (not committed)
├── .env.template       # Template for environment variables
├── docker-compose.yml  # Docker setup with PostgreSQL
├── Makefile            # Build and run commands
└── serverless.yml      # AWS Lambda configuration
```

## Build and Deployment

### Building the Binary

```bash
make build
```

### AWS Lambda Deployment

The API is designed to run as an AWS Lambda function in production:

```bash
make deploy
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Ensure PostgreSQL is running and accessible
   - Check your `DATABASE_URL` in the `.env` file

2. **Delegation Server Connection Issues**
   - Verify the delegation server is running (`make delegation-server`)
   - Check `DELEGATION_GRPC_ADDR` in your `.env` file

3. **AWS Lambda Environment Variables**
   - Ensure all required environment variables are set in both local `.env` and Lambda configuration

For detailed information about the delegation system, refer to `docs/DELEGATION_SYSTEM.md`.