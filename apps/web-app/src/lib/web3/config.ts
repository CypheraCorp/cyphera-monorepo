'use client';

import { createConfig, http } from 'wagmi';
import type { Chain } from 'viem';
import { injected } from 'wagmi/connectors';
import type { NetworkWithTokensResponse } from '@/types/network';
import { TokenResponse } from '@/types/token';
import { getNetworkConfigByChainId, getNetworkConfigs } from '@/lib/web3/config/networks';
import { logger } from '@/lib/core/logger/logger-utils';
// Helper function to transform backend network data to Wagmi Chain format
async function transformNetworkToWagmiChain(
  network: NetworkWithTokensResponse
): Promise<Chain | null> {
  // Ensure network.tokens exists and is an array
  const tokens = network.tokens || [];
  let nativeCurrencyToken: TokenResponse | undefined = tokens.find(
    (token) => token.gas_token === true
  );

  // Fallback: Create default ETH token if no gas token found
  if (!nativeCurrencyToken) {
    // Create a default ETH token for the network
    nativeCurrencyToken = {
      id: `${network.network.id || network.network.chain_id}-eth-fallback`,
      object: 'token',
      network_id: network.network.id || network.network.chain_id.toString(),
      name: 'Ethereum',
      symbol: 'ETH',
      decimals: 18,
      contract_address: '', // Native token has no contract address
      gas_token: true,
      active: true,
      created_at: Date.now(),
      updated_at: Date.now(),
    };
  }

  // Get RPC URL from centralized config
  const networkConfig = await getNetworkConfigByChainId(network.network.chain_id);
  if (!networkConfig) {
    logger.warn(`Failed to get network config for chain ${network.network.chain_id}`);
    return null;
  }
  const rpcUrl = networkConfig.rpcUrl;

  const chainResult = {
    id: network.network.chain_id,
    name: network.network.name,
    nativeCurrency: {
      name: nativeCurrencyToken.name,
      symbol: nativeCurrencyToken.symbol,
      decimals: nativeCurrencyToken.decimals || 18,
    },
    // Use the RPC URL from centralized config
    rpcUrls: {
      default: { http: [rpcUrl] },
      public: { http: [rpcUrl] },
    },
    blockExplorers: {
      default: {
        name: network.network.block_explorer_url || 'Explorer',
        url: network.network.block_explorer_url || '',
      },
    },
    testnet: network.network.is_testnet,
  } as const;
  return chainResult;
}

// Helper to create chains and transports for Wagmi config
export async function createWagmiConfigFromNetworks(
  networks: NetworkWithTokensResponse[]
): Promise<{
  chains: readonly [Chain, ...Chain[]];
  transports: Record<number, ReturnType<typeof http>>;
}> {
  if (!networks || networks.length === 0) {
    throw new Error('No networks provided for Wagmi config');
  }

  const chainPromises = networks.map((n) =>
    transformNetworkToWagmiChain(n as NetworkWithTokensResponse)
  );
  const chainResults = await Promise.all(chainPromises);
  const chains = chainResults.filter((chain): chain is Chain => chain !== null);

  if (chains.length === 0) {
    throw new Error(
      'Failed to create any valid chains for Wagmi config. Check network data and API configuration.'
    );
  }

  const transports: Record<number, ReturnType<typeof http>> = {};
  for (const chain of chains) {
    // Get the RPC URL from centralized config
    try {
      const networkConfig = await getNetworkConfigByChainId(chain.id);
      if (networkConfig) {
        transports[chain.id] = http(networkConfig.rpcUrl);
      }
    } catch (error) {
      logger.warn(
        `[createWagmiConfigFromNetworks] Failed to get RPC URL for chain ${chain.name}, transport not created:`,
        { error }
      );
    }
  }

  const finalChains = chains as [Chain, ...Chain[]];

  return {
    chains: finalChains,
    transports,
  };
}

// Fallback Wagmi config using centralized network configuration
export async function createFallbackWagmiConfig() {
  const networkConfigs = await getNetworkConfigs();

  // Convert centralized config to the format expected by createWagmiConfigFromNetworks
  const fallbackNetworks: NetworkWithTokensResponse[] = networkConfigs.map((config) => ({
    network: {
      id: config.chain.id.toString(),
      object: 'network',
      name: config.chain.name,
      type: config.chain.testnet ? 'Testnet' : 'Mainnet',
      chain_id: config.chain.id,
      network_type: 'evm',
      circle_network_type: config.circleNetworkType,
      is_testnet: config.chain.testnet || false,
      block_explorer_url: config.chain.blockExplorers?.default?.url,
      active: true,
      created_at: Date.now(),
      updated_at: Date.now(),
    },
    tokens: config.tokens.map((token) => ({
      id: `${config.chain.id}-${token.symbol}`,
      object: 'token',
      network_id: config.chain.id.toString(),
      name: token.name,
      symbol: token.symbol,
      decimals: token.decimals,
      contract_address: token.address,
      gas_token: token.isGasToken,
      active: true,
      created_at: Date.now(),
      updated_at: Date.now(),
    })),
  }));

  return createWagmiConfigFromNetworks(fallbackNetworks);
}

// Default Wagmi config factory
export async function createDefaultWagmiConfig() {
  try {
    return createFallbackWagmiConfig();
  } catch (error) {
    logger.error('Failed to create Wagmi config:', error);
    throw new Error(
      'Unable to initialize Web3 configuration. Please check your environment setup.'
    );
  }
}

// Create a custom injected connector with specific options
const metaMaskConnector = injected({
  target: 'metaMask',
  shimDisconnect: true,
});

// Modified function to accept chains and transports
export const createBaseConfig = ({
  chains,
  transports,
}: {
  chains: readonly [Chain, ...Chain[]];
  transports: Record<number, ReturnType<typeof http>>;
}) => {
  return createConfig({
    chains,
    transports,
    connectors: [metaMaskConnector],
    // ssr: true, // Consider adding if passing config via RSC/SSR
  });
};

// Default config export is removed or conditional
// The config should now be created dynamically where network data is available
// Example (remove or adapt this): Place this in a Provider component
// export const config = createBaseConfig({ chains: [/* default chain? */], transports: {/* default transport? */} });
