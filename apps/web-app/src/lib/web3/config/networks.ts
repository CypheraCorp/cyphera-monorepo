/**
 * @deprecated This file is being replaced by dynamic network configuration
 * Use src/lib/web3/dynamic-networks.ts instead
 *
 * This file is kept temporarily for backward compatibility during migration
 */

import {
  getAllNetworkConfigs,
  getNetworkConfig,
  type DynamicNetworkConfig,
} from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';

// Re-export the main interface for backward compatibility
export type NetworkConfig = DynamicNetworkConfig;

// Legacy functions that now use dynamic configuration

/**
 * @deprecated Use getAllNetworkConfigs() from dynamic-networks.ts
 */
export async function getNetworkConfigs(): Promise<NetworkConfig[]> {
  const configMap = await getAllNetworkConfigs();
  return Array.from(configMap.values());
}

/**
 * @deprecated Use getNetworkConfig() from dynamic-networks.ts
 */
export async function getNetworkConfigByChainId(
  chainId: number
): Promise<NetworkConfig | undefined> {
  const config = await getNetworkConfig(chainId);
  return config || undefined;
}

/**
 * @deprecated Use getNetworkConfig() with chain.name from dynamic-networks.ts
 */
export async function getNetworkConfigByName(name: string): Promise<NetworkConfig | undefined> {
  const configs = await getAllNetworkConfigs();

  for (const config of configs.values()) {
    if (config.chain.name.toLowerCase() === name.toLowerCase()) {
      return config;
    }
  }

  return undefined;
}

/**
 * @deprecated Use isPimlicoSupported from network config
 */
export function isPimlicoSupportedForChain(chainId: number): boolean {
  // Import the function from dynamic-networks
  // Note: This is a sync function that needs to access potentially async data
  // It will use cached data if available, otherwise fall back to hardcoded list
  // This is a temporary solution - the function should be made async
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { isPimlicoSupported } = require('@/lib/web3/dynamic-networks');
  const result = isPimlicoSupported(chainId);

  logger.log('🔍 [isPimlicoSupportedForChain] Checking chain support:', {
    chainId,
    supported: result,
  });

  return result;
}

/**
 * @deprecated Use dynamic network configuration
 */
export async function getFallbackNetwork(): Promise<NetworkConfig> {
  // Get all available networks
  const configs = await getAllNetworkConfigs();

  // Try to find a testnet first (testnets are commonly used for development)
  for (const config of configs.values()) {
    if (config.chain.testnet) {
      return config;
    }
  }

  // Fallback to any available network
  const firstConfig = configs.values().next().value;

  if (!firstConfig) {
    throw new Error('No networks available from backend');
  }

  return firstConfig;
}

// Re-export utility functions
export { getPimlicoUrls } from '@/lib/web3/dynamic-networks';
