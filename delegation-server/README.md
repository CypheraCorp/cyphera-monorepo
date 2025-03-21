# Delegation Redemption gRPC Server

A Node.js server providing a gRPC interface for redeeming MetaMask delegations and executing blockchain transactions.

## Project Structure

```
├── src/                  # Source code
│   ├── config/           # Configuration and environment setup
│   ├── services/         # Core business logic services
│   ├── types/            # TypeScript type definitions
│   ├── utils/            # Utility functions
│   ├── proto/            # Protocol Buffers definitions
│   └── index.ts          # Main application entry point
├── scripts/              # Helper scripts
│   ├── install.sh        # Installation script
│   ├── run.sh            # Server startup script
│   ├── build-and-test.sh # Build and test script
│   ├── test-grpc.js      # gRPC test script
│   └── install-grpc-client.sh # Go client code generation script
├── .env                  # Environment variables (not committed)
├── .env.example          # Example environment variables
├── package.json          # Node.js dependencies and scripts
└── tsconfig.json         # TypeScript configuration
```

## Prerequisites

- Node.js 18 or higher
- npm
- Access to a blockchain RPC endpoint (e.g., Infura)
- Access to a bundler service (e.g., Pimlico)
- Private key with ETH for gas fees
- Access to the private MetaMask package (`@metamask-private/delegator-core-viem`)

## Quick Start

1. **Clone the repository**

2. **Install dependencies**:
   ```bash
   chmod +x scripts/install.sh
   ./scripts/install.sh
   ```

3. **Configure environment variables**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run the server**:
   ```bash
   npm run dev    # Development mode
   # OR
   npm start      # Production mode
   ```

5. **Or use the convenience script**:
   ```bash
   chmod +x scripts/run.sh
   ./scripts/run.sh
   ```

## Environment Variables

Create a `.env` file with the following variables:

```
# gRPC Server Configuration
GRPC_PORT=50051
GRPC_HOST=0.0.0.0

# Blockchain Configuration
RPC_URL=https://sepolia.infura.io/v3/your-api-key
BUNDLER_URL=https://api.pimlico.io/v1/sepolia/rpc?apikey=your-api-key
PAYMASTER_URL=https://api.pimlico.io/v2/sepolia/rpc?apikey=your-api-key
CHAIN_ID=11155111

# Private Key (replace with your actual private key)
PRIVATE_KEY=your-private-key-here
```

## Integration with Go Backend

To generate the Go client code for your backend:

```bash
chmod +x scripts/install-grpc-client.sh
./scripts/install-grpc-client.sh
```

This will create the necessary Go files in `internal/proto/` directory.

## Advanced Configuration

### Private Package Access

The server depends on a private MetaMask package. You need access to this package before proceeding.

If you have access to the private package registry, add your npm authentication token to the `.npmrc` file.

### Fallback Mechanism

The server has a dynamic import system that tries to use:
1. `viem/account-abstraction` (for viem ≥ 2.x)
2. Falls back to `permissionless/clients/bundler` if not available

## Development

### Building the project
```bash
npm run build
```

### Linting
```bash
npm run lint
```

### Clean build
```bash
npm run clean
```

### Running Tests

For quick testing of the server connectivity without making blockchain calls:

```bash
node scripts/test-grpc.js
```

For comprehensive testing including building and running the server:

```bash
chmod +x scripts/build-and-test.sh
./scripts/build-and-test.sh
```

## Troubleshooting

- **Dependency Issues**: Use `./scripts/install.sh --clean` to perform a clean installation
- **Connection Problems**: Ensure your RPC and bundler endpoints are accessible
- **Private Key Errors**: Make sure your private key is properly formatted (66 characters including 0x prefix)
- **Protocol Buffers**: If changing the proto definition, regenerate both TypeScript and Go code

## Security Considerations

- Store your private key securely
- Use TLS for production gRPC connections
- Implement proper authentication between services

## Implementation Details

### Key Components

1. **gRPC Service Layer** (`src/services/service.ts`)
   - Handles incoming requests from the Go backend
   - Validates and processes delegation data
   - Returns transaction status and hash

2. **Blockchain Interaction** (`src/services/blockchain.ts`)
   - Creates and manages MetaMask smart accounts
   - Redeems delegations using the MetaMask delegation framework
   - Handles transaction submission and monitoring

3. **Configuration Management** (`src/config/config.ts`)
   - Loads and validates environment variables
   - Provides structured access to configuration

4. **Utility Functions** (`src/utils/`)
   - Logging utilities
   - Data formatting and conversion
   - Delegation parsing and validation

### Data Flow

1. Go backend calls the `RedeemDelegation` gRPC method
2. Server receives the serialized delegation data
3. Delegation is parsed and validated
4. Server creates a smart account using the provided private key
5. The delegation is redeemed on-chain via a user operation
6. Transaction hash is returned to the Go backend

### Running Tests

For quick testing of the server connectivity without making blockchain calls:

```bash
node scripts/test-grpc.js
```

For comprehensive testing including building and running the server:

```bash
chmod +x scripts/build-and-test.sh
./scripts/build-and-test.sh
``` 