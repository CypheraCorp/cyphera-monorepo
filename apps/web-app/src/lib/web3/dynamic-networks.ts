import { type Chain } from 'viem';
import {
  mainnet,
  sepolia,
  base,
  baseSepolia,
  polygon,
  polygonAmoy,
  arbitrum,
  arbitrumSepolia,
  optimism,
  optimismSepolia,
} from 'viem/chains';
import type { NetworkWithTokensResponse, GasConfig } from '@/types/network';
import type { Address } from 'viem';
import { logger } from '@/lib/core/logger/logger-utils';
// Map of chain IDs to viem chain objects
// This is the only place where we reference specific chains
const VIEM_CHAINS: Record<number, Chain> = {
  1: mainnet,
  11155111: sepolia,
  8453: base,
  84532: baseSepolia,
  137: polygon,
  80002: polygonAmoy,
  42161: arbitrum,
  421614: arbitrumSepolia,
  10: optimism,
  11155420: optimismSepolia,
};

export interface DynamicNetworkConfig {
  chain: Chain;
  rpcUrl: string;
  fallbackRpcUrls?: string[];
  circleBridgeContractAddress?: Address;
  circleTokenMessengerAddress?: Address;
  circleMessageTransmitterAddress?: Address;
  circleNetworkType: string;
  isPimlicoSupported: boolean;
  isCircleSupported: boolean;
  tokens: {
    address: Address;
    symbol: string;
    name: string;
    decimals: number;
    isGasToken: boolean;
  }[];
  // New fields from backend
  logoUrl?: string;
  displayName?: string;
  chainNamespace?: string;
  gasConfig?: GasConfig;
}

// Cache for network configurations
let networkConfigCache: Map<number, DynamicNetworkConfig> | null = null;
let networkDataCache: NetworkWithTokensResponse[] | null = null;
let cacheTimestamp: number = 0;
const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes

/**
 * Fetch networks from the backend API
 */
export async function fetchNetworks(forceRefresh = false): Promise<NetworkWithTokensResponse[]> {
  const now = Date.now();

  // Return cached data if valid and not forcing refresh
  if (!forceRefresh && networkDataCache && now - cacheTimestamp < CACHE_DURATION) {
    return networkDataCache;
  }

  try {
    const response = await fetch('/api/networks?active=true');
    if (!response.ok) {
      throw new Error(`Failed to fetch networks: ${response.statusText}`);
    }

    const data = await response.json();
    const networks = data.data || [];
    networkDataCache = networks;
    cacheTimestamp = now;

    // Clear config cache when data is refreshed
    networkConfigCache = null;

    return networks;
  } catch (error) {
    logger.error('Error fetching networks:', error);
    // Return cached data if available, even if expired
    if (networkDataCache) {
      return networkDataCache;
    }
    throw error;
  }
}

/**
 * Transform backend network data to frontend configuration
 */
export function transformNetworkToConfig(
  network: NetworkWithTokensResponse,
  infuraApiKey?: string
): DynamicNetworkConfig | null {
  const chainId = network.network.chain_id;

  // Get the viem chain object
  const viemChain = VIEM_CHAINS[chainId];
  if (!viemChain) {
    logger.warn(`No viem chain configuration for chain ID ${chainId}`);
    // Create a custom chain if viem doesn't have it
    const customChain: Chain = {
      id: chainId,
      name: network.network.display_name || network.network.name,
      nativeCurrency: {
        name: 'Ether', // Default, should come from backend
        symbol: 'ETH', // Default, should come from backend
        decimals: 18, // Default, should come from backend
      },
      rpcUrls: {
        default: {
          http: [buildRpcUrl(network, infuraApiKey)],
        },
      },
      blockExplorers: network.network.block_explorer_url
        ? {
            default: {
              name: 'Explorer',
              url: network.network.block_explorer_url,
            },
          }
        : undefined,
      testnet: network.network.is_testnet,
    };

    return {
      chain: customChain,
      rpcUrl: buildRpcUrl(network, infuraApiKey),
      circleNetworkType: network.network.circle_network_type || '',
      isPimlicoSupported: isPimlicoSupported(chainId),
      isCircleSupported: isCircleSupported(network.network.circle_network_type),
      tokens: network.tokens.map((token) => ({
        address: token.contract_address as Address,
        symbol: token.symbol,
        name: token.name,
        decimals: token.decimals,
        isGasToken: token.gas_token,
      })),
      logoUrl: network.network.logo_url,
      displayName: network.network.display_name,
      chainNamespace: network.network.chain_namespace,
      gasConfig: network.network.gas_config,
    };
  }

  return {
    chain: viemChain,
    rpcUrl: buildRpcUrl(network, infuraApiKey),
    circleNetworkType: network.network.circle_network_type || '',
    isPimlicoSupported: isPimlicoSupported(chainId),
    isCircleSupported: isCircleSupported(network.network.circle_network_type),
    tokens: network.tokens.map((token) => ({
      address: token.contract_address as Address,
      symbol: token.symbol,
      name: token.name,
      decimals: token.decimals,
      isGasToken: token.gas_token,
    })),
    logoUrl: network.network.logo_url,
    displayName: network.network.display_name,
    chainNamespace: network.network.chain_namespace,
    gasConfig: network.network.gas_config,
  };
}

/**
 * Build RPC URL based on network type and available API keys
 */
function buildRpcUrl(network: NetworkWithTokensResponse, infuraApiKey?: string): string {
  const chainId = network.network.chain_id;
  // const networkName = network.network.name.toLowerCase(); // Unused variable

  // Use Infura for supported networks if API key is available
  if (infuraApiKey) {
    const infuraEndpoints: Record<number, string> = {
      1: 'mainnet',
      11155111: 'sepolia',
      137: 'polygon-mainnet',
      80002: 'polygon-amoy',
      42161: 'arbitrum-mainnet',
      421614: 'arbitrum-sepolia',
      10: 'optimism-mainnet',
      11155420: 'optimism-sepolia',
    };

    const endpoint = infuraEndpoints[chainId];
    if (endpoint) {
      return `https://${endpoint}.infura.io/v3/${infuraApiKey}`;
    }
  }

  // Fallback to public RPC endpoints
  const chain = VIEM_CHAINS[chainId];
  if (chain?.rpcUrls?.default?.http?.[0]) {
    return chain.rpcUrls.default.http[0];
  }

  // Last resort - construct from block explorer
  if (network.network.block_explorer_url) {
    const domain = new URL(network.network.block_explorer_url).hostname;
    return `https://rpc.${domain}`;
  }

  throw new Error(`No RPC URL available for network ${network.network.name}`);
}

/**
 * Check if Pimlico is supported for a chain
 * This should be determined by the backend's network configuration
 */
export function isPimlicoSupported(chainId: number): boolean {
  // Check if we have a cached network config
  if (networkConfigCache) {
    const config = networkConfigCache.get(chainId);
    if (config) {
      logger.log('🔍 [isPimlicoSupported] Using cached config:', {
        chainId,
        supported: config.isPimlicoSupported,
      });
      return config.isPimlicoSupported;
    }
  }

  // Fallback to hardcoded list if cache not available
  // This list should eventually be removed once backend fully controls this
  const supportedChains = [
    1, // Ethereum Mainnet
    11155111, // Sepolia
    8453, // Base
    84532, // Base Sepolia
    137, // Polygon
    80002, // Polygon Amoy
    42161, // Arbitrum One
    421614, // Arbitrum Sepolia
    10, // Optimism
    11155420, // Optimism Sepolia
  ];

  const fallbackResult = supportedChains.includes(chainId);

  logger.log('🔍 [isPimlicoSupported] Using fallback hardcoded list:', {
    chainId,
    hasNetworkCache: !!networkConfigCache,
    supportedChains,
    fallbackResult,
  });

  return fallbackResult;
}

/**
 * Check if Circle is supported based on circle network type
 */
function isCircleSupported(circleNetworkType?: string): boolean {
  if (!circleNetworkType) return false;

  const supportedTypes = [
    'Ethereum',
    'ETH-SEPOLIA',
    'BASE-SEPOLIA',
    'AVAX-FUJI',
    'MATIC-AMOY',
    'ARB-SEPOLIA',
  ];

  return supportedTypes.includes(circleNetworkType);
}

/**
 * Get all network configurations
 */
export async function getAllNetworkConfigs(): Promise<Map<number, DynamicNetworkConfig>> {
  if (networkConfigCache) {
    return networkConfigCache;
  }

  const networks = await fetchNetworks();
  const infuraApiKey = process.env.NEXT_PUBLIC_INFURA_API_KEY;

  const configMap = new Map<number, DynamicNetworkConfig>();

  for (const network of networks) {
    const config = transformNetworkToConfig(network, infuraApiKey);
    if (config) {
      configMap.set(network.network.chain_id, config);
    }
  }

  networkConfigCache = configMap;
  return configMap;
}

/**
 * Get network configuration by chain ID
 */
export async function getNetworkConfig(chainId: number): Promise<DynamicNetworkConfig | null> {
  const configs = await getAllNetworkConfigs();
  return configs.get(chainId) || null;
}

/**
 * Hardcoded USDC contract addresses for supported testnets
 */
const HARDCODED_USDC_ADDRESSES: Record<number, Address> = {
  11155111: '0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238' as Address, // Ethereum Sepolia
  84532: '0x036CbD53842c5426634e7929541eC2318f3dCF7e' as Address, // Base Sepolia
};

/**
 * Get USDC token address for a specific network
 */
export async function getUSDCAddress(chainId: number): Promise<Address | null> {
  // First try hardcoded addresses (reliable fallback)
  const hardcodedAddress = HARDCODED_USDC_ADDRESSES[chainId];
  if (hardcodedAddress) {
    logger.log('🔍 [getUSDCAddress] Using hardcoded USDC address:', {
      chainId,
      address: hardcodedAddress,
    });
    return hardcodedAddress;
  }

  // Fallback to dynamic network config
  try {
    const config = await getNetworkConfig(chainId);
    if (!config) {
      logger.log('🔍 [getUSDCAddress] No network config found for chain:', chainId);
      return null;
    }

    const usdcToken = config.tokens.find((token) => token.symbol.toUpperCase() === 'USDC');

    if (usdcToken?.address) {
      logger.log('🔍 [getUSDCAddress] Using dynamic USDC address:', {
        chainId,
        address: usdcToken.address,
      });
      return usdcToken.address;
    }

    logger.log('🔍 [getUSDCAddress] No USDC token found in config for chain:', chainId);
    return null;
  } catch (error) {
    logger.error('🔍 [getUSDCAddress] Error getting dynamic address:', error);
    return null;
  }
}

/**
 * Get token by symbol for a specific network
 */
export async function getTokenBySymbol(
  chainId: number,
  symbol: string
): Promise<{ address: Address; decimals: number; name: string } | null> {
  const config = await getNetworkConfig(chainId);
  if (!config) return null;

  const token = config.tokens.find((t) => t.symbol.toUpperCase() === symbol.toUpperCase());

  return token || null;
}

/**
 * Get Pimlico URLs for a network
 */
export function getPimlicoUrls(
  chainId: number,
  apiKey: string
): {
  bundlerUrl: string;
  paymasterUrl: string;
} | null {
  if (!isPimlicoSupported(chainId) || !apiKey) {
    return null;
  }

  const baseUrl = `https://api.pimlico.io/v2/${chainId}/rpc?apikey=${apiKey}`;

  return {
    bundlerUrl: baseUrl,
    paymasterUrl: baseUrl,
  };
}

/**
 * Get chains for wagmi configuration
 */
export async function getWagmiChains(): Promise<readonly [Chain, ...Chain[]]> {
  const configs = await getAllNetworkConfigs();
  const chains = Array.from(configs.values()).map((config) => config.chain);

  if (chains.length === 0) {
    logger.warn('No chains available from backend, falling back to mainnet');
    return [mainnet];
  }

  return [chains[0], ...chains.slice(1)] as readonly [Chain, ...Chain[]];
}

/**
 * Clear all caches
 */
export function clearNetworkCache(): void {
  networkConfigCache = null;
  networkDataCache = null;
  cacheTimestamp = 0;
}

/**
 * Get gas configuration for a specific network
 */
export async function getGasConfig(chainId: number): Promise<GasConfig | null> {
  const config = await getNetworkConfig(chainId);
  return config?.gasConfig || null;
}

/**
 * Get gas settings for a specific priority level
 */
export async function getGasForPriority(
  chainId: number,
  priority: 'slow' | 'standard' | 'fast' = 'standard'
): Promise<{ maxFeePerGas: bigint; maxPriorityFeePerGas: bigint } | null> {
  const gasConfig = await getGasConfig(chainId);
  if (!gasConfig) return null;

  const gasLevel = gasConfig.gas_priority_levels[priority];
  return {
    maxFeePerGas: BigInt(gasLevel.max_fee_per_gas),
    maxPriorityFeePerGas: BigInt(gasLevel.max_priority_fee_per_gas),
  };
}

/**
 * Get deployment gas limit for a network
 */
export async function getDeploymentGasLimit(chainId: number): Promise<bigint | null> {
  const gasConfig = await getGasConfig(chainId);
  return gasConfig ? BigInt(gasConfig.deployment_gas_limit) : null;
}

/**
 * Get token transfer gas limit for a network
 */
export async function getTokenTransferGasLimit(chainId: number): Promise<bigint | null> {
  const gasConfig = await getGasConfig(chainId);
  return gasConfig ? BigInt(gasConfig.token_transfer_gas_limit) : null;
}
