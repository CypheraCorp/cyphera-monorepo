# MetaMask Delegation Redemption gRPC Server

A Node.js gRPC server that provides delegation redemption services for the Cyphera API. This server enables blockchain transactions through MetaMask's delegation framework without requiring users to sign individual transactions.

## Overview

The delegation server is a critical component of the Cyphera platform that:

1. Receives delegation requests from the Go API server
2. Processes these delegations (in production or mock mode)
3. Executes blockchain transactions on behalf of users
4. Returns transaction results back to the API

```
┌─────────────┐         ┌──────────────┐         ┌────────────────┐         ┌────────────┐
│             │  HTTP   │              │  gRPC   │                │  RPC    │            │
│   Client    ├────────►│   Go API     ├────────►│  Delegation    ├────────►│ Blockchain │
│             │         │              │         │  Server (Node) │         │            │
└─────────────┘         └──────────────┘         └────────────────┘         └────────────┘
```

## Project Structure

```
delegation-server/
├── src/                      # Source code
│   ├── config.ts             # Configuration settings and env var loading
│   ├── index.ts              # Main entry point, starts gRPC server
│   ├── proto/                # Protocol buffer definitions and generated code
│   │   ├── delegation.proto  # gRPC service definition
│   │   ├── delegation_pb.js  # Generated JavaScript definitions
│   │   └── delegation_grpc_pb.js # Generated gRPC client/server code
│   └── services/             # Core business logic for handling delegations
├── scripts/                  # Utility scripts
├── test/                     # Test files
├── .env                      # Environment configuration (not committed)
├── .env.example              # Example environment variables
├── package.json              # Dependencies and scripts
├── tsconfig.json             # TypeScript configuration
└── README.md                 # This documentation
```

## Key Components

### 1. gRPC Service Implementation (`src/index.ts`)

The core service implementation that:
- Exposes the `redeemDelegation` RPC endpoint
- Receives delegation data from the Go API
- Validates and processes the delegation
- Returns transaction hashes or error responses

### 2. Configuration (`src/config.ts`)

Manages all environment variables and configuration settings:
- Server address and port configuration
- Mock mode settings
- Connection details for blockchain providers

### 3. MockBlockchainService

In mock mode, this service simulates blockchain interactions:
- Generates fake transaction hashes
- Allows testing without actual blockchain calls
- Useful for local development and testing

## Environment Variables

Create a `.env` file with these variables:

```
# Server Configuration
GRPC_PORT=50051          # Port for the gRPC server
GRPC_HOST=0.0.0.0        # Host address to bind

# Mode Configuration
MOCK_MODE=true           # Set to false for real blockchain transactions

# Blockchain Configuration (for production)
RPC_URL=                 # Blockchain RPC endpoint (e.g., Infura)
CHAIN_ID=11155111        # Chain ID (e.g., 11155111 for Sepolia testnet)
PRIVATE_KEY=             # Private key for transaction signing
```

## Running the Server

### Prerequisites

- Node.js 18 or higher
- npm

### Installation

```bash
# Install dependencies
npm install

# If you have access issues with private packages, configure npm
echo "@metamask-private:registry=https://npm.pkg.github.com/" > .npmrc
echo "//npm.pkg.github.com/:_authToken=YOUR_GITHUB_TOKEN" >> .npmrc
```

### Running in Development Mode

```bash
# Start in mock mode (simulated blockchain interactions)
npm run start:mock

# Start with actual blockchain interactions
npm run start
```

### Running in Production Mode

```bash
# Build the TypeScript files
npm run build

# Start the server
node dist/index.js
```

## Integration with Go API

The Go API communicates with this server via gRPC. The main integration points are:

1. **Delegation Client** (`internal/client/delegation_client.go`)
   - Creates a gRPC connection to this server
   - Sends delegation data for redemption
   - Handles responses and errors

2. **Environment Configuration**
   - `DELEGATION_GRPC_ADDR`: Address of this gRPC server (e.g., `localhost:50051`)
   - `DELEGATION_LOCAL_MODE`: Set to `true` for local development

3. **Health Checking**
   - The Go API periodically checks if this server is available
   - Implements circuit breaking for fault tolerance

## Testing

### Manual Testing

```bash
# Check if server is running
curl -v http://localhost:50051
# Expected: Error (normal since this is a gRPC server, not HTTP)
```

### Automated Testing

```bash
# Run unit tests
npm test

# Run integration tests (requires .env configuration)
npm run test:integration
```

### Testing with the Go API

The API includes integration tests that verify communication with this server:

```bash
# From the main project directory
make test-integration
```

## Mock Mode vs. Production Mode

### Mock Mode

In mock mode (set `MOCK_MODE=true` in `.env`):
- The server generates random transaction hashes
- No real blockchain interactions occur
- Suitable for local development and testing

### Production Mode

In production mode (set `MOCK_MODE=false` in `.env`):
- Actual blockchain interactions through MetaMask's delegation framework
- Requires valid private keys and RPC endpoints
- Creates smart accounts and submits user operations to bundlers

## Troubleshooting

### Common Issues

1. **Connection Refused**
   - Ensure the server is running on the configured port
   - Check if any firewall is blocking connections

2. **Authentication Errors**
   - Verify access to private npm packages
   - Check GitHub tokens if using private repositories

3. **Failed Delegations**
   - Examine server logs for error details
   - Verify delegation data format from the Go API

### Logs

The server outputs logs to help diagnose issues:
- Standard output for general info and errors
- Detailed error messages when processing delegations fails

## Technical Details for LLM Context

For LLMs analyzing this project, here are key implementation details:

1. **DelegationServiceImpl**: The primary class implementing the gRPC service defined in `delegation.proto`. It accepts delegation data and processes it.

2. **Mock Implementation**: In mock mode, it returns fake transaction hashes instead of making real blockchain calls.

3. **Error Handling**: Uses gRPC error codes to communicate different failure types:
   - `INVALID_ARGUMENT` for bad input data
   - `INTERNAL` for server-side processing errors

4. **Data Flow**:
   - Go API sends serialized delegation data
   - Server deserializes and validates the data
   - Server processes delegation (mock or real)
   - Transaction hash is returned to the Go API

5. **Integration Pattern**: Uses a service-to-service gRPC communication pattern with:
   - Strong typing through Protocol Buffers
   - Binary efficient data transfer
   - Proper error propagation

This server is designed to be a stateless microservice that can be horizontally scaled as needed.

## Protocol Buffers (gRPC)

The service uses Protocol Buffers and gRPC for communication. The protocol definition is maintained in:

- `src/proto/delegation.proto`: Main proto definition file

When updating the proto definition:

1. Modify `src/proto/delegation.proto`
2. Regenerate the TypeScript/JavaScript files:

```bash
npx grpc_tools_node_protoc --js_out=import_style=commonjs,binary:./src/proto --grpc_out=grpc_js:./src/proto --ts_out=grpc_js:./src/proto -I ./src/proto src/proto/delegation.proto
```

This will update the generated files in `src/proto/` directory. 