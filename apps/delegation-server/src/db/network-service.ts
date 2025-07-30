import { query } from './database';
import { logger } from '../utils/utils';

/**
 * Network data from database
 */
export interface NetworkData {
  id: string;
  name: string;
  type: string;
  network_type: string;
  rpc_id: string;
  block_explorer_url: string | null;
  chain_id: number;
  is_testnet: boolean;
  active: boolean;
  display_name: string | null;
  chain_namespace: string;
  deleted_at: string | null;
}

/**
 * Token data from database
 */
export interface TokenData {
  id: string;
  network_id: string;
  gas_token: boolean;
  name: string;
  symbol: string;
  contract_address: string;
  active: boolean;
  decimals: number;
  deleted_at: string | null;
}

/**
 * Get network by chain ID
 */
export async function getNetworkByChainId(chainId: number): Promise<NetworkData | null> {
  try {
    const result = await query<NetworkData>(
      `SELECT * FROM networks 
       WHERE chain_id = $1 
       AND active = true 
       AND deleted_at IS NULL 
       LIMIT 1`,
      [chainId]
    );

    if (result.rows.length === 0) {
      logger.warn(`Network not found for chain ID: ${chainId}`);
      return null;
    }

    return result.rows[0];
  } catch (error) {
    logger.error('Error fetching network by chain ID:', { error, chainId });
    throw new Error(`Failed to fetch network for chain ID ${chainId}`);
  }
}

/**
 * Get network by name
 */
export async function getNetworkByName(name: string): Promise<NetworkData | null> {
  try {
    const result = await query<NetworkData>(
      `SELECT * FROM networks 
       WHERE (LOWER(name) = LOWER($1) OR LOWER(display_name) = LOWER($1))
       AND active = true 
       AND deleted_at IS NULL 
       LIMIT 1`,
      [name]
    );

    if (result.rows.length === 0) {
      logger.warn(`Network not found for name: ${name}`);
      return null;
    }

    return result.rows[0];
  } catch (error) {
    logger.error('Error fetching network by name:', { error, name });
    throw new Error(`Failed to fetch network for name ${name}`);
  }
}

/**
 * Get all active networks
 */
export async function getAllNetworks(): Promise<NetworkData[]> {
  try {
    const result = await query<NetworkData>(
      `SELECT * FROM networks 
       WHERE active = true 
       AND deleted_at IS NULL 
       ORDER BY is_testnet ASC, display_name ASC`
    );

    return result.rows;
  } catch (error) {
    logger.error('Error fetching all networks:', { error });
    throw new Error('Failed to fetch networks');
  }
}

/**
 * Get tokens for a network
 */
export async function getTokensForNetwork(networkId: string): Promise<TokenData[]> {
  try {
    const result = await query<TokenData>(
      `SELECT * FROM tokens 
       WHERE network_id = $1 
       AND active = true 
       AND deleted_at IS NULL 
       ORDER BY gas_token DESC, symbol ASC`,
      [networkId]
    );

    return result.rows;
  } catch (error) {
    logger.error('Error fetching tokens for network:', { error, networkId });
    throw new Error(`Failed to fetch tokens for network ${networkId}`);
  }
}

/**
 * Get specific token by address on a network
 */
export async function getTokenByAddress(
  networkId: string, 
  contractAddress: string
): Promise<TokenData | null> {
  try {
    const result = await query<TokenData>(
      `SELECT * FROM tokens 
       WHERE network_id = $1 
       AND LOWER(contract_address) = LOWER($2)
       AND active = true 
       AND deleted_at IS NULL 
       LIMIT 1`,
      [networkId, contractAddress]
    );

    if (result.rows.length === 0) {
      return null;
    }

    return result.rows[0];
  } catch (error) {
    logger.error('Error fetching token by address:', { error, networkId, contractAddress });
    throw new Error(`Failed to fetch token ${contractAddress} on network ${networkId}`);
  }
}

/**
 * Validate if a token is supported on a network by chain ID and token address
 */
export async function validateTokenSupport(
  chainId: number,
  tokenAddress: string
): Promise<{ valid: boolean; token?: TokenData; error?: string }> {
  try {
    // First get the network
    const network = await getNetworkByChainId(chainId);
    if (!network) {
      return { 
        valid: false, 
        error: `Network with chain ID ${chainId} not found or not active` 
      };
    }

    // Then check if the token exists on this network
    const token = await getTokenByAddress(network.id, tokenAddress);
    if (!token) {
      return { 
        valid: false, 
        error: `Token ${tokenAddress} not found or not active on network ${network.display_name || network.name}` 
      };
    }

    return { valid: true, token };
  } catch (error) {
    return { 
      valid: false, 
      error: `Failed to validate token support: ${error}` 
    };
  }
}