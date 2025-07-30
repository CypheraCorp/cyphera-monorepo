# Database-Driven Network Configuration

## Overview

The delegation server now uses PostgreSQL database for all network and token configurations. This provides a fully dynamic system where networks and tokens can be added, updated, or removed without any code changes or deployments.

## Architecture

### Database Schema

#### Networks Table
```sql
CREATE TABLE networks (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    network_type network_type NOT NULL,
    rpc_id TEXT NOT NULL,              -- Used for Infura URL construction
    block_explorer_url TEXT,
    chain_id INTEGER NOT NULL UNIQUE,
    is_testnet BOOLEAN NOT NULL,
    active BOOLEAN NOT NULL,
    display_name TEXT,
    chain_namespace TEXT DEFAULT 'eip155',
    -- Additional fields for gas configuration, etc.
);
```

#### Tokens Table
```sql
CREATE TABLE tokens (
    id UUID PRIMARY KEY,
    network_id UUID REFERENCES networks(id),
    gas_token BOOLEAN NOT NULL,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    contract_address TEXT NOT NULL,
    active BOOLEAN NOT NULL,
    decimals INTEGER NOT NULL,
    UNIQUE(network_id, contract_address)
);
```

### Network Service (`apps/delegation-server/src/db/network-service.ts`)

The network service provides database access functions:

```typescript
// Get network by chain ID
getNetworkByChainId(chainId: number): Promise<NetworkData | null>

// Get network by name
getNetworkByName(name: string): Promise<NetworkData | null>

// Get all active networks
getAllNetworks(): Promise<NetworkData[]>

// Get tokens for a network
getTokensForNetwork(networkId: string): Promise<TokenData[]>

// Get specific token by address
getTokenByAddress(networkId: string, contractAddress: string): Promise<TokenData | null>

// Validate token support
validateTokenSupport(chainId: number, tokenAddress: string): Promise<ValidationResult>
```

## Usage

### In the Delegation Server

The delegation server fetches network configuration from the database:

```typescript
import { getNetworkConfig } from './config/config';
import { validateTokenSupport } from './db/network-service';

// Get network configuration
const config = await getNetworkConfig(networkName, chainId);
// Returns: { rpcUrl, bundlerUrl }
// URLs are constructed using:
// - rpc_id from database + Infura API key
// - chain_id + Pimlico API key

// Validate token support
const validation = await validateTokenSupport(chainId, tokenAddress);
// Returns: { valid: boolean, token?: TokenData, error?: string }
```

### Adding New Networks

To add a new network:

1. Insert the network into the database:

```sql
INSERT INTO networks (
  name, 
  display_name,
  type, 
  network_type,
  rpc_id,           -- e.g., 'linea-mainnet' for Infura
  chain_id,
  is_testnet,
  active,
  block_explorer_url
) VALUES (
  'linea-mainnet',
  'Linea',
  'ethereum',
  'ETHEREUM',
  'linea-mainnet',
  59144,
  false,
  true,
  'https://lineascan.build'
);
```

2. Add supported tokens:

```sql
INSERT INTO tokens (
  network_id,
  name,
  symbol,
  contract_address,
  decimals,
  active,
  gas_token
) VALUES (
  (SELECT id FROM networks WHERE chain_id = 59144),
  'USD Coin',
  'USDC',
  '0x176211869cA2b568f2A7D4EE941E073a821EE1ff',
  6,
  true,
  false
);
```

3. The network is immediately available - no code changes or deployments needed!

### Network Resolution

The system resolves networks in the following order:

1. **By Chain ID** (most reliable): Queries the `chain_id` column
2. **By Name**: Searches both `name` and `display_name` columns (case-insensitive)

Example:
- Chain ID `1` → Ethereum Mainnet
- Name "ethereum mainnet" → matches `display_name`
- Name "mainnet" → matches `name`

## Database Configuration

### Environment Variables

The delegation server requires a database connection:

```bash
# .env file
DATABASE_URL=postgresql://username:password@localhost:5432/cyphera_db

# API keys (fetched from AWS Secrets Manager in production)
INFURA_API_KEY_ARN=arn:aws:secretsmanager:...
PIMLICO_API_KEY_ARN=arn:aws:secretsmanager:...
```

### URL Construction

The system constructs provider URLs dynamically:

```typescript
// RPC URL
const rpcUrl = `https://${network.rpc_id}.infura.io/v3/${infuraApiKey}`;

// Bundler URL
const bundlerUrl = `https://api.pimlico.io/v2/${network.chain_id}/rpc?apikey=${pimlicoApiKey}`;
```

The `rpc_id` field in the database must match Infura's subdomain format:
- `mainnet` for Ethereum Mainnet
- `sepolia` for Ethereum Sepolia
- `polygon-mainnet` for Polygon
- `base-sepolia` for Base Sepolia
- etc.

## Best Practices

1. **Database First**: All network/token configuration comes from the database
2. **No Hardcoding**: Never hardcode network IDs, token addresses, or decimals
3. **Active Flag**: Use the `active` flag to enable/disable networks without deletion
4. **Soft Deletes**: Use `deleted_at` timestamps instead of hard deletes
5. **Chain ID Uniqueness**: The `chain_id` column has a unique constraint
6. **Token Uniqueness**: The combination of `network_id` and `contract_address` is unique

## Security Considerations

1. **Database Access**: The delegation server needs read-only access to networks/tokens tables
2. **API Keys**: Infura and Pimlico keys are fetched from AWS Secrets Manager
3. **Input Validation**: Always validate chain IDs and addresses from user input
4. **Network Verification**: Verify the chain ID matches the expected network name

## Migration from Static Configuration

The system has been migrated from static TypeScript configuration to database-driven:

1. **Removed**: `libs/ts/delegation/src/config/networks.ts` (static registry)
2. **Removed**: `libs/ts/delegation/src/config/dynamic-config.ts` (static helper)
3. **Added**: `apps/delegation-server/src/db/network-service.ts` (database queries)
4. **Updated**: `getNetworkConfig()` now queries the database instead of static data

## Error Handling

```typescript
try {
  const network = await getNetworkByChainId(chainId);
  if (!network) {
    throw new Error(`Network with chain ID ${chainId} not found`);
  }
  // Use network...
} catch (error) {
  logger.error('Failed to fetch network configuration', { error, chainId });
  // Handle gracefully...
}
```