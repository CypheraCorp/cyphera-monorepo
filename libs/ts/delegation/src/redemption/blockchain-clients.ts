import { 
  createPublicClient,
  custom,
  type Transport,
  type Chain,
  type PublicClient
} from 'viem';
import { 
  createBundlerClient, 
  createPaymasterClient,
  type BundlerClient
} from 'viem/account-abstraction';
import { createPimlicoClient, type PimlicoClient } from 'permissionless/clients/pimlico';
import * as allChains from 'viem/chains';
import { BlockchainClients, RedemptionError, RedemptionErrorType } from './types';
import { NetworkConfig } from '../types/delegation';

/**
 * Finds a viem Chain object by its chain ID
 * @param chainId The chain ID to look up
 * @returns The Chain object
 * @throws RedemptionError if chain is not supported
 */
export function getChainById(chainId: number): Chain {
  // First try to find in viem's predefined chains
  for (const chainKey in allChains) {
    const chain = allChains[chainKey as keyof typeof allChains];
    if (typeof chain === 'object' && chain !== null && 'id' in chain && chain.id === chainId) {
      return chain as Chain;
    }
  }
  
  // If not found in viem, create a minimal chain object
  // This allows us to support any chain from the database
  return {
    id: chainId,
    name: `Chain ${chainId}`,
    nativeCurrency: {
      name: 'ETH',
      symbol: 'ETH',
      decimals: 18
    },
    rpcUrls: {
      default: { http: [] },
      public: { http: [] }
    },
    blockExplorers: undefined
  } as Chain;
}

/**
 * Creates a custom viem transport using fetch
 * This is necessary for Node.js environments where the default transport may not work
 * @param url The RPC URL
 * @returns A custom Transport
 */
export function createFetchTransport(url: string | undefined): Transport {
  if (!url) {
    throw new RedemptionError(
      'URL is required for transport',
      RedemptionErrorType.NETWORK_ERROR
    );
  }

  return custom({
    async request({ method, params }) {
      // For browser environments, use native fetch
      const fetchImpl = typeof window !== 'undefined' && window.fetch 
        ? window.fetch 
        : (await import('node-fetch')).default;

      const response = await fetchImpl(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          jsonrpc: '2.0', 
          method, 
          params, 
          id: 1 
        }),
      });

      if (!response.ok) {
        const errorBody = await response.text();
        throw new RedemptionError(
          `HTTP error! status: ${response.status}`,
          RedemptionErrorType.NETWORK_ERROR,
          { url, status: response.status, body: errorBody }
        );
      }

      const data = await response.json() as any;
      if (data.error) {
        throw new RedemptionError(
          `RPC error: ${data.error.message}`,
          RedemptionErrorType.NETWORK_ERROR,
          { code: data.error.code, message: data.error.message }
        );
      }

      return data.result;
    }
  });
}

/**
 * Initializes all necessary blockchain clients for redemption operations
 * @param networkConfig The network configuration containing RPC and bundler URLs
 * @param chain The chain object from viem
 * @returns BlockchainClients object containing all initialized clients
 */
export async function initializeBlockchainClients(
  networkConfig: NetworkConfig,
  chain: Chain
): Promise<BlockchainClients> {
  try {
    // Create public client for reading blockchain state
    const publicClient = createPublicClient({
      chain,
      transport: createFetchTransport(networkConfig.rpcUrl)
    }) as PublicClient<Transport, Chain>;

    // Create paymaster client (required for bundler client)
    const paymasterClient = createPaymasterClient({
      transport: createFetchTransport(networkConfig.bundlerUrl)
    });

    // Create bundler client for sending UserOperations
    const bundlerClient = createBundlerClient({
      transport: createFetchTransport(networkConfig.bundlerUrl),
      chain,
      paymaster: paymasterClient,
    }) as any;

    // Create Pimlico client for gas estimation and utilities
    const pimlicoClient = createPimlicoClient({
      chain,
      transport: createFetchTransport(networkConfig.bundlerUrl),
    }) as any;

    return { publicClient, bundlerClient, pimlicoClient };
  } catch (error) {
    throw new RedemptionError(
      'Failed to initialize blockchain clients',
      RedemptionErrorType.NETWORK_ERROR,
      error
    );
  }
}

/**
 * Creates a minimal network config from network name and chain ID
 * This is a helper for cases where full NetworkConfig isn't available
 * @param networkName The network name
 * @param chainId The chain ID
 * @param rpcUrl The RPC URL
 * @param bundlerUrl The bundler URL
 * @returns NetworkConfig object
 */
export function createNetworkConfigFromUrls(
  networkName: string,
  chainId: number,
  rpcUrl: string,
  bundlerUrl: string
): NetworkConfig {
  const chain = getChainById(chainId);
  
  return {
    chainId,
    name: networkName,
    rpcUrl,
    bundlerUrl,
    nativeCurrency: chain.nativeCurrency,
    blockExplorer: chain.blockExplorers?.default?.url
  };
}