import { NetworkConfig } from '../types/delegation';

/**
 * Network configuration utilities
 * Based on apps/delegation-server/src/config/config.ts
 */

/**
 * Formats network name for Infura URL construction
 * PRESERVED from delegation-server config logic
 * @param networkName The network name (e.g., "Ethereum Mainnet", "Base Sepolia")
 * @returns Formatted network name for Infura subdomain
 */
export function formatNetworkNameForInfura(networkName: string): string {
  if (!networkName) {
    throw new Error('networkName parameter is required to construct Infura RPC URL');
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
    case 'ethereum holesky':
      formattedNetworkName = 'holesky';
      break;
    default:
      // General rule: lowercase, replace spaces with hyphens
      formattedNetworkName = networkName.toLowerCase().replace(/\s+/g, '-');
  }

  // Basic validation for common network name patterns (after transformation)
  if (!/^[a-z0-9-]+$/.test(formattedNetworkName)) {
    console.warn(`Potential issue: Formatted network name "${formattedNetworkName}" (from "${networkName}") contains unexpected characters. Ensure it matches Infura subdomain format.`);
  }

  return formattedNetworkName;
}

/**
 * Constructs RPC URL for a given network
 * @param networkName The network name
 * @param infuraApiKey The Infura API key
 * @returns The constructed RPC URL
 */
export function constructRpcUrl(networkName: string, infuraApiKey: string): string {
  const formattedNetworkName = formatNetworkNameForInfura(networkName);
  return `https://${formattedNetworkName}.infura.io/v3/${infuraApiKey}`;
}

/**
 * Constructs bundler URL for a given chain ID
 * @param chainId The EVM chain ID
 * @param pimlicoApiKey The Pimlico API key
 * @returns The constructed bundler URL
 */
export function constructBundlerUrl(chainId: number, pimlicoApiKey: string): string {
  const bundlerBaseUrl = "https://api.pimlico.io/v2/";
  return `${bundlerBaseUrl}${chainId}/rpc?apikey=${pimlicoApiKey}`;
}

/**
 * Creates a network configuration object
 * @param chainId The chain ID
 * @param name The network name
 * @param infuraApiKey The Infura API key
 * @param pimlicoApiKey The Pimlico API key
 * @returns NetworkConfig object
 */
export function createNetworkConfig(
  chainId: number, 
  name: string, 
  infuraApiKey: string, 
  pimlicoApiKey?: string
): NetworkConfig {
  return {
    chainId,
    name,
    rpcUrl: constructRpcUrl(name, infuraApiKey),
    bundlerUrl: pimlicoApiKey ? constructBundlerUrl(chainId, pimlicoApiKey) : undefined,
    pimlicoApiKey,
    nativeCurrency: {
      name: 'Ether', // Default, can be overridden
      symbol: 'ETH',
      decimals: 18
    }
  };
}

/**
 * Network configuration manager
 */
export class NetworkManager {
  private static configs: Map<number, NetworkConfig> = new Map();
  
  /**
   * Register a network configuration
   * @param config The network configuration
   */
  static register(config: NetworkConfig): void {
    this.configs.set(config.chainId, config);
  }
  
  /**
   * Get network configuration by chain ID
   * @param chainId The chain ID
   * @returns NetworkConfig or undefined if not found
   */
  static get(chainId: number): NetworkConfig | undefined {
    return this.configs.get(chainId);
  }
  
  /**
   * Get network configuration by name
   * @param name The network name
   * @returns NetworkConfig or undefined if not found
   */
  static getByName(name: string): NetworkConfig | undefined {
    for (const config of this.configs.values()) {
      if (config.name.toLowerCase() === name.toLowerCase()) {
        return config;
      }
    }
    return undefined;
  }
  
  /**
   * Get all registered network configurations
   * @returns Array of NetworkConfig
   */
  static getAll(): NetworkConfig[] {
    return Array.from(this.configs.values());
  }
  
  /**
   * Get RPC URL for a chain ID
   * @param chainId The chain ID
   * @returns RPC URL or undefined if not found
   */
  static getRpcUrl(chainId: number): string | undefined {
    return this.configs.get(chainId)?.rpcUrl;
  }
  
  /**
   * Get bundler URL for a chain ID
   * @param chainId The chain ID
   * @returns Bundler URL or undefined if not found
   */
  static getBundlerUrl(chainId: number): string | undefined {
    return this.configs.get(chainId)?.bundlerUrl;
  }
}