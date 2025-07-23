import { http } from 'viem';
import { createBundlerClient, createPaymasterClient } from 'viem/account-abstraction';
import { isPimlicoSupportedForChain } from '@/lib/web3/config/networks';
import { getNetworkConfig, getPimlicoUrls } from '@/lib/web3/dynamic-networks';

/**
 * Get Pimlico configuration for a specific chain ID
 * Now uses centralized network configuration
 */
export async function getPimlicoConfig(chainId: number) {
  const networkConfig = await getNetworkConfig(chainId);
  if (!networkConfig) {
    throw new Error(`Network configuration not found for chain ID: ${chainId}`);
  }

  const pimlicoApiKey = process.env.NEXT_PUBLIC_PIMLICO_API_KEY;
  if (!pimlicoApiKey) {
    throw new Error(`Pimlico API key not configured`);
  }

  const pimlicoUrls = getPimlicoUrls(chainId, pimlicoApiKey);
  if (!pimlicoUrls) {
    throw new Error(`Pimlico is not supported for chain ID: ${chainId}`);
  }

  return {
    bundlerUrl: pimlicoUrls.bundlerUrl,
    paymasterUrl: pimlicoUrls.paymasterUrl,
    chain: networkConfig.chain,
  };
}

/**
 * Create a bundler client for the specified chain
 */
export async function createBundlerClientForChain(chainId: number) {
  const config = await getPimlicoConfig(chainId);

  return createBundlerClient({
    transport: http(config.bundlerUrl),
    chain: config.chain,
  });
}

/**
 * Check if Pimlico is supported for the given chain
 */
export function isPimlicoSupported(chainId: number): boolean {
  return isPimlicoSupportedForChain(chainId);
}

/**
 * Get all Pimlico supported chain IDs
 */
export async function getPimlicoSupportedChains(): Promise<number[]> {
  // Get from dynamic network configuration
  const { getAllNetworkConfigs } = await import('@/lib/web3/dynamic-networks');
  const configs = await getAllNetworkConfigs();

  const supportedChains: number[] = [];
  for (const [chainId, config] of configs) {
    if (config.isPimlicoSupported) {
      supportedChains.push(chainId);
    }
  }

  return supportedChains;
}

/**
 * Get human-readable network name for Pimlico supported chains
 */
export async function getPimlicoNetworkName(chainId: number): Promise<string> {
  try {
    const networkConfig = await getNetworkConfig(chainId);
    return networkConfig ? networkConfig.chain.name : `Unsupported Network (${chainId})`;
  } catch {
    return `Unsupported Network (${chainId})`;
  }
}

/**
 * Check if Pimlico is properly configured
 */
export async function isPimlicoConfigured(): Promise<boolean> {
  try {
    // Check if API key exists
    const pimlicoApiKey = process.env.NEXT_PUBLIC_PIMLICO_API_KEY;
    if (!pimlicoApiKey) {
      return false;
    }

    // Get any supported chain to test configuration
    const supportedChains = await getPimlicoSupportedChains();
    if (supportedChains.length === 0) {
      return false;
    }

    // Try to get config for the first supported chain
    const testConfig = await getPimlicoConfig(supportedChains[0]);
    return !!(testConfig.bundlerUrl && testConfig.paymasterUrl);
  } catch {
    return false;
  }
}

/**
 * Create a paymaster client for the specified chain
 */
export async function createPaymasterClientForChain(chainId: number) {
  const config = await getPimlicoConfig(chainId);

  return createPaymasterClient({
    transport: http(config.paymasterUrl),
  });
}

/**
 * Check if paymaster sponsorship is available for the given chain
 */
export async function isPaymasterSponsorshipAvailable(chainId: number): Promise<boolean> {
  try {
    const config = await getPimlicoConfig(chainId);
    return !!(config.bundlerUrl && config.paymasterUrl);
  } catch {
    return false;
  }
}
