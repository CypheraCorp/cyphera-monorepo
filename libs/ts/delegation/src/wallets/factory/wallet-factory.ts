import { WalletProvider, WalletProviderType, WalletProviderConfig } from '../interfaces/wallet-provider';

/**
 * Factory for creating wallet provider instances
 * This factory pattern allows for easy extension to support new wallet providers
 * while maintaining a consistent interface.
 */
export class WalletFactory {
  private static providers: Map<WalletProviderType, new (config: WalletProviderConfig) => WalletProvider> = new Map();

  /**
   * Register a wallet provider implementation
   * @param type The wallet provider type
   * @param implementation The provider implementation class
   */
  static register(
    type: WalletProviderType, 
    implementation: new (config: WalletProviderConfig) => WalletProvider
  ): void {
    this.providers.set(type, implementation);
  }

  /**
   * Create a wallet provider instance
   * @param config The wallet provider configuration
   * @returns A wallet provider instance
   */
  static create(config: WalletProviderConfig): WalletProvider {
    const ProviderClass = this.providers.get(config.type);
    
    if (!ProviderClass) {
      throw new Error(`Wallet provider type '${config.type}' is not registered. Available types: ${Array.from(this.providers.keys()).join(', ')}`);
    }

    return new ProviderClass(config);
  }

  /**
   * Get all registered provider types
   * @returns Array of registered provider types
   */
  static getRegisteredTypes(): WalletProviderType[] {
    return Array.from(this.providers.keys());
  }

  /**
   * Check if a provider type is registered
   * @param type The provider type to check
   * @returns True if the provider type is registered
   */
  static isRegistered(type: WalletProviderType): boolean {
    return this.providers.has(type);
  }
}

/**
 * Utility function to create a wallet provider with validation
 * @param config The wallet provider configuration
 * @returns A wallet provider instance
 */
export function createWalletProvider(config: WalletProviderConfig): WalletProvider {
  // Validate configuration
  if (!config.type) {
    throw new Error('Wallet provider type is required');
  }

  // Create and return the provider
  return WalletFactory.create(config);
}