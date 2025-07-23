# Delegation Server

> **Navigation:** [← Root README](../../README.md) | [Main API →](../api/README.md) | [Architecture →](../../docs/architecture.md)

The delegation server is a Node.js gRPC service that handles blockchain operations and MetaMask delegation management for the Cyphera platform.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Development](#development)
- [gRPC Interface](#grpc-interface)
- [Blockchain Integration](#blockchain-integration)
- [MetaMask Delegation](#metamask-delegation)
- [Testing](#testing)
- [Deployment](#deployment)

## Overview

The delegation server provides blockchain interaction capabilities through a gRPC interface, enabling secure transaction execution and smart account management across multiple networks.

### Key Features
- **gRPC Server** for high-performance blockchain operations
- **MetaMask Delegation Toolkit** integration for smart accounts
- **Multi-network Support** (Ethereum, Polygon, Arbitrum)
- **Transaction Signing** with hardware security modules
- **Smart Account Creation** and management
- **Mock Mode** for development and testing
- **Comprehensive Logging** with structured output

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Main API      │    │  Web Frontend   │    │   Background    │
│   (Go/gRPC)     │    │   (Direct)      │    │    Jobs         │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │ gRPC Calls
                    ┌────────────▼────────────┐
                    │   Delegation Server     │
                    │   (Node.js/gRPC)        │
                    └────────────┬────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌────────▼─────────┐ ┌───────────▼────────┐ ┌───────────▼────────┐
│   MetaMask       │ │   Blockchain       │ │   Smart Account    │
│   Delegation     │ │   Networks         │ │   Management       │
│   Toolkit        │ │   (RPC Clients)    │ │   (Creation/Ops)   │
└──────────────────┘ └────────────────────┘ └────────────────────┘
```

### Directory Structure

```
apps/delegation-server/
├── src/
│   ├── index.ts              # Server entry point
│   ├── config/
│   │   └── config.ts         # Configuration management
│   ├── proto/                # gRPC protocol definitions
│   │   ├── delegation.proto
│   │   ├── delegation_grpc_pb.js
│   │   └── delegation_pb.js
│   ├── services/             # Business logic services
│   │   ├── service.ts        # Main gRPC service implementation
│   │   ├── redeem-delegation.ts
│   │   └── mock-redeem-delegation.ts
│   ├── utils/                # Utility functions
│   │   ├── delegation-helpers.ts
│   │   ├── utils.ts
│   │   └── secrets_manager.ts
│   └── abis/                 # Smart contract ABIs
│       ├── erc20.ts
│       └── simpleFactory.ts
├── test/                     # Test files
│   ├── service.test.ts
│   ├── integration-test.ts
│   └── utils.test.ts
├── scripts/                  # Build and deployment scripts
├── package.json
├── tsconfig.json
└── README.md                 # This file
```

## Development

### Prerequisites
- Node.js 18 or later
- TypeScript
- Docker (for PostgreSQL)
- Environment variables configured

### Installation
```bash
# From project root
npm run install:ts

# Or directly in delegation server
cd apps/delegation-server
npm install
```

### Running Locally

#### Start Server
```bash
# From project root
npm run dev:delegation

# Or directly
cd apps/delegation-server
npm run dev
```

The server will start on port `50051` (gRPC) by default.

#### Environment Variables
Create `.env` file from template:
```bash
cd apps/delegation-server
cp .env.example .env
```

Configure the following variables:
```bash
# Server Configuration
GRPC_PORT=50051
NODE_ENV=development

# Blockchain RPC URLs
ETHEREUM_RPC_URL="https://eth-sepolia.g.alchemy.com/v2/your_key"
POLYGON_RPC_URL="https://polygon-mumbai.g.alchemy.com/v2/your_key"
ARBITRUM_RPC_URL="https://arb-sepolia.g.alchemy.com/v2/your_key"

# Security (Use test keys only in development!)
DELEGATION_PRIVATE_KEY="0x1234...your_test_private_key"
ENCRYPTION_SECRET="your_encryption_secret"

# Database
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/cyphera_dev"

# Circle API
CIRCLE_API_KEY="your_circle_api_key"

# Development
LOG_LEVEL="debug"
MOCK_MODE="false"
```

### Development Commands
```bash
# Development server with hot reload
npm run dev

# Build TypeScript
npm run build

# Run tests
npm run test

# Generate gRPC code
npm run proto:build

# Lint code
npm run lint
```

## gRPC Interface

### Protocol Definition
The gRPC service is defined in `src/proto/delegation.proto`:

```protobuf
syntax = "proto3";
package delegation;

service DelegationService {
  rpc RedeemDelegation(RedeemDelegationRequest) returns (RedeemDelegationResponse);
  rpc CreateSmartAccount(CreateSmartAccountRequest) returns (CreateSmartAccountResponse);
  rpc GetAccountBalance(GetAccountBalanceRequest) returns (GetAccountBalanceResponse);
  rpc ExecuteTransaction(ExecuteTransactionRequest) returns (ExecuteTransactionResponse);
}

message RedeemDelegationRequest {
  string delegation_hash = 1;
  string network_id = 2;
  string token_address = 3;
  string amount = 4;
  string recipient_address = 5;
}

message RedeemDelegationResponse {
  string transaction_hash = 1;
  string status = 2;
  string message = 3;
}
```

### Service Implementation
Main service logic in `src/services/service.ts`:

```typescript
class DelegationServiceImpl implements IDelegationService {
  async redeemDelegation(
    request: RedeemDelegationRequest
  ): Promise<RedeemDelegationResponse> {
    try {
      // Validate delegation
      const delegation = await this.validateDelegation(request.delegationHash);
      
      // Execute blockchain transaction
      const txHash = await this.executeTransaction(delegation, request);
      
      return {
        transactionHash: txHash,
        status: 'success',
        message: 'Delegation redeemed successfully'
      };
    } catch (error) {
      return {
        transactionHash: '',
        status: 'error',
        message: error.message
      };
    }
  }
}
```

### Client Usage
Example gRPC client usage from the main API:

```go
// Create gRPC client
conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := delegation.NewDelegationServiceClient(conn)

// Make request
response, err := client.RedeemDelegation(ctx, &delegation.RedeemDelegationRequest{
    DelegationHash: "0x...",
    NetworkId:      "ethereum",
    TokenAddress:   "0x...",
    Amount:         "100.00",
    RecipientAddress: "0x...",
})
```

## Blockchain Integration

### Supported Networks

| Network | Chain ID | RPC Configuration | Status |
|---------|----------|-------------------|--------|
| Ethereum Mainnet | 1 | `ETHEREUM_RPC_URL` | ✅ Production |
| Ethereum Sepolia | 11155111 | `ETHEREUM_SEPOLIA_RPC_URL` | ✅ Testnet |
| Polygon Mainnet | 137 | `POLYGON_RPC_URL` | ✅ Production |
| Polygon Mumbai | 80001 | `POLYGON_MUMBAI_RPC_URL` | ✅ Testnet |
| Arbitrum One | 42161 | `ARBITRUM_RPC_URL` | ✅ Production |
| Arbitrum Sepolia | 421614 | `ARBITRUM_SEPOLIA_RPC_URL` | ✅ Testnet |

### Network Configuration
Networks are configured in `src/config/config.ts`:

```typescript
export const NETWORK_CONFIG = {
  ethereum: {
    chainId: 1,
    rpcUrl: process.env.ETHEREUM_RPC_URL,
    name: 'Ethereum Mainnet',
    nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 }
  },
  polygon: {
    chainId: 137,
    rpcUrl: process.env.POLYGON_RPC_URL,
    name: 'Polygon Mainnet',
    nativeCurrency: { name: 'MATIC', symbol: 'MATIC', decimals: 18 }
  }
  // ... additional networks
};
```

### RPC Client Management
Blockchain clients are managed with connection pooling:

```typescript
class BlockchainService {
  private clients: Map<string, ethers.JsonRpcProvider> = new Map();

  getClient(networkId: string): ethers.JsonRpcProvider {
    if (!this.clients.has(networkId)) {
      const config = NETWORK_CONFIG[networkId];
      const client = new ethers.JsonRpcProvider(config.rpcUrl);
      this.clients.set(networkId, client);
    }
    return this.clients.get(networkId)!;
  }
}
```

## MetaMask Delegation

### Delegation Toolkit Integration
The server uses MetaMask's Delegation Toolkit for smart account operations:

```typescript
import { createDelegationExecutor } from '@metamask/delegation-toolkit';

class DelegationManager {
  private executor = createDelegationExecutor({
    privateKey: process.env.DELEGATION_PRIVATE_KEY,
    rpcUrl: this.getRpcUrl(networkId)
  });

  async executeDelegation(params: DelegationParams) {
    const result = await this.executor.executeDelegation({
      delegation: params.delegation,
      transaction: params.transaction
    });
    
    return result.transactionHash;
  }
}
```

### Smart Account Creation
Smart accounts are created using the delegation toolkit:

```typescript
async createSmartAccount(request: CreateSmartAccountRequest) {
  const factory = new ethers.Contract(
    SMART_ACCOUNT_FACTORY_ADDRESS,
    SmartAccountFactoryABI,
    this.getSigner(request.networkId)
  );

  const tx = await factory.createAccount(
    request.ownerAddress,
    request.salt || 0
  );

  const receipt = await tx.wait();
  const accountAddress = await this.getAccountAddress(receipt);
  
  return {
    accountAddress,
    transactionHash: receipt.transactionHash,
    status: 'success'
  };
}
```

### Delegation Validation
Delegations are validated before execution:

```typescript
async validateDelegation(delegationHash: string): Promise<Delegation> {
  // Check delegation exists in database
  const delegation = await this.db.getDelegation(delegationHash);
  if (!delegation) {
    throw new Error('Delegation not found');
  }

  // Verify delegation signature
  const isValid = await this.verifyDelegationSignature(delegation);
  if (!isValid) {
    throw new Error('Invalid delegation signature');
  }

  // Check delegation hasn't expired
  if (delegation.expiresAt < Date.now()) {
    throw new Error('Delegation has expired');
  }

  return delegation;
}
```

## Testing

### Unit Tests
Run unit tests for individual components:

```bash
npm run test

# Run specific test file
npm run test -- service.test.ts

# Run with coverage
npm run test:coverage
```

### Integration Tests
Test the full gRPC service:

```bash
# Start test database
docker-compose up postgres-test -d

# Run integration tests
npm run test:integration
```

### Mock Mode
For development without blockchain dependencies:

```bash
# Set in .env
MOCK_MODE=true

# Start server
npm run dev
```

Mock mode provides:
- Simulated transaction execution
- Fake transaction hashes
- Consistent test responses
- No actual blockchain interaction

### gRPC Testing
Test gRPC endpoints directly:

```bash
# Install grpcurl
brew install grpcurl

# Test service
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 delegation.DelegationService/RedeemDelegation
```

## Deployment

### Production Build
```bash
# Build TypeScript
npm run build

# Output in dist/ directory
```

### Docker Deployment
The service can be containerized:

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY dist/ ./dist/
CMD ["node", "dist/index.js"]
```

### Environment Configuration
Production environment variables should be managed via:
- AWS Secrets Manager (for cloud deployment)
- Kubernetes Secrets (for container deployment)
- Environment-specific configuration files

### Health Checks
The server provides health check endpoints:

```typescript
// Health check for container orchestration
app.get('/health', (req, res) => {
  res.json({ 
    status: 'healthy', 
    timestamp: new Date().toISOString(),
    version: process.env.npm_package_version 
  });
});
```

## Security Considerations

### Private Key Management
- **Development:** Use test private keys only
- **Production:** Use AWS KMS or Hardware Security Modules
- **Never:** Commit private keys to version control

### Network Security
- Use VPC with private subnets for production
- Implement IP whitelisting for gRPC endpoints
- Enable TLS for all gRPC communications

### Logging & Monitoring
- Structured logging with no sensitive data
- CloudWatch integration for AWS deployments
- Alert on failed transactions or unusual patterns

```typescript
// Safe logging example
logger.info('Delegation executed', {
  delegationId: delegation.id,
  networkId: request.networkId,
  // Never log private keys or sensitive data
});
```

## Performance Optimization

### Connection Pooling
- Reuse RPC connections across requests
- Implement connection health checks
- Configure appropriate timeouts

### Caching
- Cache network configurations
- Store frequently accessed delegation data
- Use Redis for distributed caching

### Resource Management
- Monitor memory usage with large transaction volumes
- Implement circuit breakers for RPC calls
- Use connection limits and rate limiting

---

## Related Documentation

- **[Architecture Guide](../../docs/architecture.md)** - System overview
- **[Main API Documentation](../api/README.md)** - API service integration
- **[Blockchain Networks](../../docs/networks.md)** - Network configuration
- **[Security Guide](../../docs/security.md)** - Security best practices

## Need Help?

- **[Troubleshooting](../../docs/troubleshooting.md)** - Common issues
- **[Contributing](../../docs/contributing.md)** - Development workflow
- **MetaMask Delegation Docs** - Official toolkit documentation
- **GitHub Issues** - Bug reports and feature requests

---

*Last updated: $(date '+%Y-%m-%d')*
*Service Version: 2.0.0*