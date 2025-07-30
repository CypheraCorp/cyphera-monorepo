import { type Address } from 'viem';
import { type Delegation } from '@metamask/delegation-toolkit';
import { WalletProvider, WalletProviderType, WalletProviderConfig, WalletCapabilities, DelegationParams } from '../interfaces/wallet-provider';

/**
 * Abstract base class for wallet provider implementations
 * This provides common functionality and enforces the interface contract
 */
export abstract class BaseWalletProvider implements WalletProvider {
  protected config: WalletProviderConfig;
  protected _isConnected: boolean = false;
  protected _address: Address | null = null;

  constructor(config: WalletProviderConfig) {
    this.config = config;
  }

  get type(): WalletProviderType {
    return this.config.type;
  }

  get isConnected(): boolean {
    return this._isConnected;
  }

  get address(): Address | null {
    return this._address;
  }

  /**
   * Get the capabilities of this wallet provider
   * Override in implementations to specify actual capabilities
   */
  getCapabilities(): WalletCapabilities {
    return {
      supportsDelegation: false,
      supportsNetworkSwitching: false,
      supportsSmartAccounts: false,
      supportsGaslessTransactions: false,
    };
  }

  // Abstract methods to be implemented by concrete providers
  abstract connect(): Promise<void>;
  abstract disconnect(): Promise<void>;
  abstract getAddress(): Promise<Address | null>;
  abstract switchNetwork(chainId: number): Promise<void>;
  abstract signDelegation(delegationParams: DelegationParams): Promise<Delegation>;

  /**
   * Utility method to validate connection state
   */
  protected ensureConnected(): void {
    if (!this._isConnected) {
      throw new Error(`${this.type} wallet is not connected. Please connect first.`);
    }
  }

  /**
   * Utility method to validate address
   */
  protected ensureAddress(): Address {
    this.ensureConnected();
    
    if (!this._address) {
      throw new Error(`${this.type} wallet address is not available.`);
    }
    
    return this._address;
  }
}