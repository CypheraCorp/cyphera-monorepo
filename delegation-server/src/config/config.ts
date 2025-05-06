import dotenv from 'dotenv'
import { resolve } from 'path'
import { logger } from '../utils/utils'

// Load environment variables from .env file
dotenv.config({ path: resolve(__dirname, '../../.env') })

// Basic configuration not dependent on chainId or networkName
export const config = {
  grpc: {
    port: parseInt(process.env.GRPC_PORT || '50051', 10),
    host: process.env.GRPC_HOST || '0.0.0.0'
  },
  blockchain: {
    privateKey: process.env.PRIVATE_KEY
  },
  logging: {
    level: process.env.LOG_LEVEL || 'info'
  },
  mockMode: process.env.MOCK_MODE === 'true'
}

interface NetworkConfig {
  rpcUrl: string
  bundlerUrl: string
}

/**
 * Retrieves network-specific configuration (RPC URL, Bundler URL)
 * based on the provided network name (for Infura RPC) and chain ID (for Pimlico Bundler).
 * Requires INFURA_API_KEY, PIMLICO_API_KEY, and base BUNDLER_URL to be set.
 * 
 * @param networkName The network name (e.g., "Ethereum Mainnet", "Base Sepolia")
 * @param chainId The EVM chain ID (used for Bundler URL construction)
 * @returns NetworkConfig containing rpcUrl and bundlerUrl
 * @throws Error if required URLs or API keys are not found
 */
export function getNetworkConfig(networkName: string, chainId: number): NetworkConfig {
  // --- RPC URL (Infura) ---
  const infuraApiKey = process.env.INFURA_API_KEY
  if (!infuraApiKey) {
    throw new Error('INFURA_API_KEY environment variable is not set')
  }
  if (!networkName) {
    throw new Error('networkName parameter is required to construct Infura RPC URL')
  }
  
  let formattedNetworkName: string;

  // Specific handling for Ethereum networks
  switch (networkName.toLowerCase()) {
    case 'ethereum mainnet':
      formattedNetworkName = 'mainnet';
      break;
    case 'ethereum sepolia':
      formattedNetworkName = 'sepolia';
      break;
    case 'ethereum holesky': // Assuming "Hoodi" meant Holesky
      formattedNetworkName = 'holesky';
      break;
    default:
      // General rule: lowercase, replace spaces with hyphens
      formattedNetworkName = networkName.toLowerCase().replace(/\s+/g, '-');
  }

  // Basic validation for common network name patterns (after transformation)
  if (!/^[a-z0-9-]+$/.test(formattedNetworkName)) {
    logger.warn(`Potential issue: Formatted network name "${formattedNetworkName}" (from "${networkName}") contains unexpected characters. Ensure it matches Infura subdomain format.`)
    // Allow potentially custom names, but log a warning
  }
  const rpcUrl = `https://${formattedNetworkName}.infura.io/v3/${infuraApiKey}`

  // --- Bundler URL (Pimlico V2 Format) ---
  const pimlicoApiKey = process.env.PIMLICO_API_KEY;
  const baseBundlerUrl = process.env.BUNDLER_URL; // Expects base like https://api.pimlico.io/v2/

  if (!pimlicoApiKey) {
    throw new Error('PIMLICO_API_KEY environment variable is not set');
  }
  if (!baseBundlerUrl) {
    throw new Error('Base BUNDLER_URL environment variable is not set (e.g., https://api.pimlico.io/v2/)');
  }

  // Ensure base URL ends with a slash
  const sanitizedBaseBundlerUrl = baseBundlerUrl.endsWith('/') ? baseBundlerUrl : `${baseBundlerUrl}/`;

  const bundlerUrl = `${sanitizedBaseBundlerUrl}${chainId}/rpc?apikey=${pimlicoApiKey}`;

  logger.debug(`Using configuration for chainId ${chainId} / network ${networkName}:`, {
    rpcUrlSource: `Infura (network: ${formattedNetworkName})`,
    bundlerUrlSource: `Pimlico (base: ${sanitizedBaseBundlerUrl}, chainId: ${chainId})`,
  })

  return { rpcUrl, bundlerUrl }
}

// Validate required base configuration
export function validateConfig(): void {
  const requiredVars = [
    // Check for INFURA_API_KEY and default BUNDLER_URL as baseline requirements
    { key: 'INFURA_API_KEY', value: process.env.INFURA_API_KEY },
    { key: 'PIMLICO_API_KEY', value: process.env.PIMLICO_API_KEY },
    { key: 'BUNDLER_URL (base)', value: process.env.BUNDLER_URL },
    { key: 'blockchain.privateKey', value: config.blockchain.privateKey },
  ]
  
  const missingVars = requiredVars.filter(v => !v.value)
  
  if (missingVars.length > 0) {
    const missingKeys = missingVars.map(v => v.key).join(', ')
    throw new Error(`Missing required environment variables: ${missingKeys}`)
  }
  
  // Validate private key format
  if (config.blockchain.privateKey) {
    const pkRegex = /^0x[0-9a-fA-F]{64}$/
    if (!pkRegex.test(config.blockchain.privateKey)) {
      throw new Error('PRIVATE_KEY must be a valid 32-byte hex string with 0x prefix (66 characters total)')
    }
  }
  // Basic check for Infura key format (doesn't validate the key itself)
  if (process.env.INFURA_API_KEY && !/^[a-zA-Z0-9]{32,}$/.test(process.env.INFURA_API_KEY)) {
    logger.warn('INFURA_API_KEY format looks unusual. Ensure it is correct.')
  }
  // Optional: Add basic check for Pimlico key format
  if (process.env.PIMLICO_API_KEY && !/^pim_[a-zA-Z0-9]+$/.test(process.env.PIMLICO_API_KEY)) {
    logger.warn('PIMLICO_API_KEY format looks unusual. Ensure it starts with \'pim_\'.');
  }
} 