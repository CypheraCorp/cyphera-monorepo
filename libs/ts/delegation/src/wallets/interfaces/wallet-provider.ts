import { type Address, type Hex } from 'viem';
import { type Delegation, type Caveat } from '@metamask/delegation-toolkit';

/**
 * Generic wallet provider interface
 * This interface abstracts different embedded wallet solutions (Web3Auth, Circle, WalletConnect, etc.)
 * to provide a consistent API for delegation operations.
 */
export interface WalletProvider {
  /**
   * The type of wallet provider (e.g., 'web3auth', 'circle', 'metamask', 'walletconnect')
   */
  readonly type: WalletProviderType;
  
  /**
   * Whether the wallet is currently connected and ready for operations
   */
  readonly isConnected: boolean;
  
  /**
   * The current wallet address, if connected
   */
  readonly address: Address | null;
  
  /**
   * Connect to the wallet provider
   * @returns Promise that resolves when connection is established
   */
  connect(): Promise<void>;
  
  /**
   * Disconnect from the wallet provider
   * @returns Promise that resolves when disconnection is complete
   */
  disconnect(): Promise<void>;
  
  /**
   * Get the current wallet address
   * @returns The wallet address or null if not connected
   */
  getAddress(): Promise<Address | null>;
  
  /**
   * Switch to a specific network
   * @param chainId The target chain ID
   * @returns Promise that resolves when network switch is complete
   */
  switchNetwork(chainId: number): Promise<void>;
  
  /**
   * Sign a delegation using this wallet
   * @param delegationParams Parameters for creating the delegation
   * @returns Promise that resolves to the signed delegation
   */
  signDelegation(delegationParams: DelegationParams): Promise<Delegation>;
}

/**
 * Supported wallet provider types
 */
export type WalletProviderType = 
  | 'web3auth'
  | 'circle'
  | 'metamask'
  | 'walletconnect'
  | 'coinbase'
  | 'custom';

/**
 * Parameters for creating a delegation
 */
export interface DelegationParams {
  /** The address to delegate to */
  targetAddress: Address;
  /** Optional caveats to apply to the delegation */
  caveats?: Caveat[];
  /** Optional salt for the delegation */
  salt?: Hex;
}

/**
 * Wallet provider configuration
 */
export interface WalletProviderConfig {
  /** The type of provider */
  type: WalletProviderType;
  /** Provider-specific configuration options */
  options?: Record<string, unknown>;
  /** Network configurations */
  networks?: {
    chainId: number;
    rpcUrl: string;
    name: string;
  }[];
}

/**
 * Wallet provider capabilities
 */
export interface WalletCapabilities {
  /** Whether the provider supports delegation signing */
  supportsDelegation: boolean;
  /** Whether the provider supports network switching */
  supportsNetworkSwitching: boolean;
  /** Whether the provider supports smart accounts */
  supportsSmartAccounts: boolean;
  /** Whether the provider supports gasless transactions */
  supportsGaslessTransactions: boolean;
}