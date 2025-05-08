import dotenv from 'dotenv'
import { resolve } from 'path'
import { logger } from '../utils/utils'
import { getSecretValue } from '../utils/secrets_manager'

// Load environment variables from .env file
dotenv.config({ path: resolve(__dirname, '../../.env') })

// Basic configuration
export const config = {
  grpc: {
    port: parseInt(process.env.GRPC_PORT || '50051', 10),
    host: process.env.GRPC_HOST || '0.0.0.0'
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
 * Requires INFURA_API_KEY_ARN, PIMLICO_API_KEY_ARN to be set for ARN-based fetching,
 * or INFURA_API_KEY, PIMLICO_API_KEY for direct/fallback fetching.
 * 
 * @param networkName The network name (e.g., "Ethereum Mainnet", "Base Sepolia")
 * @param chainId The EVM chain ID (used for Bundler URL construction)
 * @returns Promise<NetworkConfig> containing rpcUrl and bundlerUrl
 * @throws Error if required URLs or API keys are not found
 */
export async function getNetworkConfig(networkName: string, chainId: number): Promise<NetworkConfig> {
  logger.info("Infura API Key ARN (env): ", process.env.INFURA_API_KEY_ARN);
  logger.info("Pimlico API Key ARN (env): ", process.env.PIMLICO_API_KEY_ARN);
  logger.info("Private Key ARN (env): ", process.env.PRIVATE_KEY_ARN);

  // --- RPC URL (Infura) ---
  const infuraApiKey = await getSecretValue('INFURA_API_KEY_ARN', 'INFURA_API_KEY')

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
  const pimlicoApiKey = await getSecretValue('PIMLICO_API_KEY_ARN', 'PIMLICO_API_KEY')

  const bundlerBaseUrl = "https://api.pimlico.io/v2/";

  const bundlerUrl = `${bundlerBaseUrl}${chainId}/rpc?apikey=${pimlicoApiKey}`;
  logger.debug(`Using configuration for chainId ${chainId} / network ${networkName}:`, {
    rpcUrlSource: `Infura (network: ${formattedNetworkName})`,
    bundlerUrlSource: `Pimlico (base: ${bundlerBaseUrl}, chainId: ${chainId})`,
  })

  return { rpcUrl, bundlerUrl }
}
