import dotenv from 'dotenv'
import { resolve } from 'path'
import { logger } from '../utils/utils'
import { getSecretValue } from '../utils/secrets_manager'
import { getNetworkByChainId, getNetworkByName } from '../db/network-service'

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
 * from the database based on the provided network name and chain ID.
 * 
 * @param networkName The network name (e.g., "Ethereum Mainnet", "Base Sepolia")
 * @param chainId The EVM chain ID
 * @returns Promise<NetworkConfig> containing rpcUrl and bundlerUrl
 * @throws Error if network is not found in database or API keys are not found
 */
export async function getNetworkConfig(networkName: string, chainId: number): Promise<NetworkConfig> {
  logger.info("Getting network configuration from database", { networkName, chainId });
  
  try {
    // Fetch network from database by chain ID (most reliable)
    let network = await getNetworkByChainId(chainId);
    
    // If not found by chain ID, try by name
    if (!network && networkName) {
      logger.warn(`Network not found by chain ID ${chainId}, trying by name: ${networkName}`);
      network = await getNetworkByName(networkName);
    }
    
    if (!network) {
      throw new Error(`Network not found in database for chain ID ${chainId} or name "${networkName}"`);
    }
    
    // Validate that the network matches the expected chain ID
    if (network.chain_id !== chainId) {
      throw new Error(`Chain ID mismatch: expected ${chainId}, but network "${network.name}" has chain ID ${network.chain_id}`);
    }
    
    // Get API keys from secrets
    const infuraApiKey = await getSecretValue('INFURA_API_KEY_ARN', 'INFURA_API_KEY');
    const pimlicoApiKey = await getSecretValue('PIMLICO_API_KEY_ARN', 'PIMLICO_API_KEY');
    
    // Build URLs using the rpc_id from database
    const rpcUrl = `https://${network.rpc_id}.infura.io/v3/${infuraApiKey}`;
    const bundlerUrl = `https://api.pimlico.io/v2/${network.chain_id}/rpc?apikey=${pimlicoApiKey}`;
    
    logger.debug(`Using network configuration from database:`, {
      networkId: network.id,
      chainId: network.chain_id,
      name: network.name,
      displayName: network.display_name,
      rpcId: network.rpc_id,
      isTestnet: network.is_testnet,
      rpcUrl: rpcUrl.replace(/\/[^\/]+$/, '/***'), // Hide API key
      bundlerUrl: bundlerUrl.replace(/apikey=.*$/, 'apikey=***') // Hide API key
    });
    
    return { rpcUrl, bundlerUrl };
  } catch (error) {
    logger.error('Failed to get network configuration from database', { error, networkName, chainId });
    throw error;
  }
}
