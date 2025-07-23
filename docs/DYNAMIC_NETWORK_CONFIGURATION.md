# Dynamic Network Configuration

## Overview

The Cyphera API now supports dynamic network configuration with comprehensive gas optimization settings, allowing frontend applications to retrieve all network information from the backend API rather than hardcoding it. This ensures network data remains consistent across all clients and simplifies the process of adding new networks.

## API Changes

### Network Response Structure

The GET `/networks` endpoint now returns enhanced network information with gas configuration:

```json
{
  "object": "list",
  "data": [
    {
      "network": {
        "id": "uuid",
        "object": "network",
        "name": "Ethereum Sepolia",
        "type": "Sepolia",
        "chain_id": 11155111,
        "network_type": "evm",
        "circle_network_type": "ETH-SEPOLIA",
        "block_explorer_url": "https://sepolia.etherscan.io",
        "is_testnet": true,
        "active": true,
        "logo_url": "https://cryptologos.cc/logos/ethereum-eth-logo.png",
        "display_name": "Ethereum Sepolia",
        "chain_namespace": "eip155",
        "created_at": 1234567890,
        "updated_at": 1234567890,
        "gas_config": {
          "base_fee_multiplier": 1.2,
          "priority_fee_multiplier": 1.1,
          "deployment_gas_limit": "500000",
          "token_transfer_gas_limit": "100000",
          "supports_eip1559": true,
          "gas_oracle_url": "",
          "gas_refresh_interval_ms": 30000,
          "gas_priority_levels": {
            "slow": {
              "max_fee_per_gas": "1000000000",
              "max_priority_fee_per_gas": "100000000"
            },
            "standard": {
              "max_fee_per_gas": "2000000000",
              "max_priority_fee_per_gas": "200000000"
            },
            "fast": {
              "max_fee_per_gas": "5000000000",
              "max_priority_fee_per_gas": "500000000"
            }
          },
          "average_block_time_ms": 12000,
          "peak_hours_multiplier": 1.5
        }
      },
      "tokens": [
        {
          "id": "uuid",
          "object": "token",
          "name": "USD Coin",
          "symbol": "USDC",
          "contract_address": "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238",
          "gas_token": false,
          "active": true,
          "decimals": 6
        }
      ]
    }
  ]
}
```

### New Network Fields

The following fields have been added to the network model:

#### Display Information
- **`logo_url`** (string, optional): URL to the network's logo image for UI display
- **`display_name`** (string, optional): User-friendly name for the network (e.g., "Ethereum" instead of "Ethereum Mainnet")
- **`chain_namespace`** (string, optional): Chain type identifier (defaults to "eip155" for EVM chains)

#### Gas Configuration (`gas_config` object)
- **`base_fee_multiplier`** (float): Multiplier for base fee calculations (e.g., 1.2 = 20% buffer)
- **`priority_fee_multiplier`** (float): Multiplier for priority fee calculations
- **`deployment_gas_limit`** (string): Gas limit for smart account deployment
- **`token_transfer_gas_limit`** (string): Gas limit for ERC20 token transfers
- **`supports_eip1559`** (boolean): Whether the network supports EIP-1559 gas pricing
- **`gas_oracle_url`** (string, optional): External gas oracle endpoint for dynamic pricing
- **`gas_refresh_interval_ms`** (integer): How often to refresh gas prices (milliseconds)
- **`gas_priority_levels`** (object): Predefined gas settings for different transaction speeds
  - `slow`: Lower gas prices for non-urgent transactions
  - `standard`: Default gas prices for normal transactions
  - `fast`: Higher gas prices for urgent transactions
- **`average_block_time_ms`** (integer): Average block time in milliseconds for transaction time estimates
- **`peak_hours_multiplier`** (float): Additional multiplier for peak usage times

### Supported Networks

The API currently includes the following networks:

| Network | Chain ID | Type | Active |
|---------|----------|------|---------|
| Ethereum Sepolia | 11155111 | Testnet | ✅ |
| Ethereum Mainnet | 1 | Mainnet | ❌ |
| Base Sepolia | 84532 | Testnet | ✅ |
| Base Mainnet | 8453 | Mainnet | ✅ |
| Polygon Amoy | 80002 | Testnet | ❌ |
| Polygon Mainnet | 137 | Mainnet | ❌ |
| Arbitrum Sepolia | 421614 | Testnet | ❌ |
| Arbitrum One | 42161 | Mainnet | ❌ |
| Optimism Sepolia | 11155420 | Testnet | ❌ |
| Optimism Mainnet | 10 | Mainnet | ❌ |
| Unichain Sepolia | 1301 | Testnet | ❌ |
| Unichain Mainnet | 130 | Mainnet | ❌ |

## Frontend Integration

### Fetching Network Data

Frontend applications should fetch network data on initialization:

```typescript
// Example TypeScript implementation
interface GasPriorityLevel {
  max_fee_per_gas: string;
  max_priority_fee_per_gas: string;
}

interface GasConfig {
  base_fee_multiplier: number;
  priority_fee_multiplier: number;
  deployment_gas_limit: string;
  token_transfer_gas_limit: string;
  supports_eip1559: boolean;
  gas_oracle_url?: string;
  gas_refresh_interval_ms: number;
  gas_priority_levels: {
    slow: GasPriorityLevel;
    standard: GasPriorityLevel;
    fast: GasPriorityLevel;
  };
  average_block_time_ms: number;
  peak_hours_multiplier: number;
}

interface Network {
  id: string;
  name: string;
  display_name?: string;
  chain_id: number;
  network_type: string;
  circle_network_type: string;
  block_explorer_url?: string;
  is_testnet: boolean;
  active: boolean;
  logo_url?: string;
  chain_namespace?: string;
  gas_config?: GasConfig;
}

interface Token {
  id: string;
  name: string;
  symbol: string;
  contract_address: string;
  gas_token: boolean;
  decimals: number;
}

interface NetworkWithTokens {
  network: Network;
  tokens: Token[];
}

async function fetchNetworks(): Promise<NetworkWithTokens[]> {
  const response = await fetch('/api/networks?active=true');
  const data = await response.json();
  return data.data;
}
```

### Filtering Networks

The API supports query parameters for filtering:

- `?active=true` - Only return active networks
- `?testnet=true` - Only return testnet networks
- `?testnet=false` - Only return mainnet networks

### Using Network Data

Frontend applications can use the network data to:

1. **Display Network Selection**: Use `display_name` and `logo_url` for UI elements
2. **Configure Web3 Providers**: Use `chain_id` and `network_type` for blockchain connections
3. **Show Block Explorers**: Use `block_explorer_url` for transaction links
4. **Filter by Environment**: Use `is_testnet` to separate test/production networks
5. **Optimize Gas Usage**: Use `gas_config` for smart gas pricing and limits

### Gas Configuration Usage

```typescript
// Example: Using gas configuration for smart account deployment
async function deploySmartAccount(network: Network) {
  const gasConfig = network.gas_config;
  
  // Get current gas prices
  const gasPrice = await getGasPrice(network);
  
  // Apply multipliers for safety margin
  const adjustedGasPrice = {
    maxFeePerGas: BigInt(gasPrice.maxFeePerGas) * BigInt(gasConfig.base_fee_multiplier * 100) / 100n,
    maxPriorityFeePerGas: BigInt(gasPrice.maxPriorityFeePerGas) * BigInt(gasConfig.priority_fee_multiplier * 100) / 100n
  };
  
  // Use network-specific gas limits
  const gasLimit = BigInt(gasConfig.deployment_gas_limit);
  
  // Deploy with optimized gas settings
  return deployAccount({
    gasLimit,
    ...adjustedGasPrice
  });
}

// Example: Selecting gas priority level
function getGasForPriority(network: Network, priority: 'slow' | 'standard' | 'fast') {
  const gasLevel = network.gas_config.gas_priority_levels[priority];
  
  return {
    maxFeePerGas: BigInt(gasLevel.max_fee_per_gas),
    maxPriorityFeePerGas: BigInt(gasLevel.max_priority_fee_per_gas)
  };
}

// Example: Using gas oracle if available
async function getDynamicGasPrice(network: Network) {
  if (network.gas_config.gas_oracle_url) {
    try {
      const response = await fetch(network.gas_config.gas_oracle_url);
      const oracleData = await response.json();
      // Process oracle data based on provider format
      return processOracleData(oracleData);
    } catch (error) {
      // Fallback to default gas levels
      return network.gas_config.gas_priority_levels.standard;
    }
  }
  
  return network.gas_config.gas_priority_levels.standard;
}
```

## Adding New Networks

To add a new network:

1. Update the database by adding an INSERT statement in `01-init.sql`:
```sql
INSERT INTO networks (
  name, type, network_type, circle_network_type, chain_id, 
  is_testnet, active, block_explorer_url, logo_url, display_name, 
  chain_namespace, base_fee_multiplier, priority_fee_multiplier,
  deployment_gas_limit, token_transfer_gas_limit, supports_eip1559,
  average_block_time_ms, gas_priority_levels
)
VALUES (
  'Network Name', 'Type', 'evm', 'CIRCLE-TYPE', chain_id, 
  false, true, 'https://explorer.url', 'https://logo.url', 'Display Name', 
  'eip155', 1.2, 1.1,
  '500000', '100000', true,
  2000, '{"slow":{"max_fee_per_gas":"1000000000","max_priority_fee_per_gas":"100000000"},"standard":{"max_fee_per_gas":"2000000000","max_priority_fee_per_gas":"200000000"},"fast":{"max_fee_per_gas":"5000000000","max_priority_fee_per_gas":"500000000"}}'
);
```

2. Add corresponding tokens for the network:
```sql
INSERT INTO tokens (network_id, name, symbol, contract_address, gas_token, active, decimals)
VALUES ((SELECT id FROM networks WHERE chain_id = your_chain_id), 'Token Name', 'SYMBOL', '0xaddress', false, true, 18);
```

3. Restart the API server to load the new network data

### Gas Configuration Guidelines

When adding a new network, consider these gas configuration values:

- **Layer 1 Networks** (Ethereum, etc.):
  - `base_fee_multiplier`: 1.2-1.3 (20-30% buffer)
  - `average_block_time_ms`: 12000-15000
  - Higher gas prices in priority levels

- **Layer 2 Networks** (Arbitrum, Optimism, Base):
  - `base_fee_multiplier`: 1.1-1.2 (10-20% buffer)
  - `average_block_time_ms`: 250-2000
  - Lower gas prices due to optimized architecture

- **Sidechains** (Polygon):
  - `base_fee_multiplier`: 1.3-1.5 (30-50% buffer due to volatility)
  - `average_block_time_ms`: 2000-3000
  - Moderate gas prices

## Benefits

1. **Single Source of Truth**: All network configuration is managed in the backend
2. **Easy Updates**: Adding new networks requires no frontend changes
3. **Consistency**: All clients use the same network data
4. **Feature Flags**: Networks can be activated/deactivated without code changes
5. **Rich Metadata**: Additional network information can be added without breaking existing clients

## Migration Notes

Frontend applications migrating from hardcoded networks should:

1. Remove all hardcoded network configuration
2. Implement network data fetching on app initialization
3. Cache network data appropriately (consider 5-minute cache)
4. Handle network data loading states in the UI
5. Implement fallback behavior for network fetch failures