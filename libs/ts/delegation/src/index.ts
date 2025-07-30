/**
 * @cyphera/delegation - Shared delegation utilities for Cyphera platform
 * 
 * This library consolidates delegation logic from both web-app and delegation-server
 * to provide a consistent, reusable foundation for delegation operations.
 */

// Core delegation functionality
export {
  createSalt,
  createAndSignDelegation,
  formatDelegation
} from './core/delegation-factory';

export {
  parseDelegation
} from './core/delegation-parser';

export {
  isValidEthereumAddress,
  validateDelegation,
  validateDelegationStructure
} from './core/delegation-validator';

// Types
export type {
  AuthorityStruct,
  CaveatStruct,
  DelegationStruct,
  NetworkConfig,
  ValidationResult
} from './types/delegation';

// Re-export MetaMask delegation toolkit types for compatibility
export type { Delegation, Caveat } from './types/delegation';

// Network utilities
export {
  formatNetworkNameForInfura,
  constructRpcUrl,
  constructBundlerUrl,
  createNetworkConfig,
  NetworkManager
} from './utils/network-config';

// Crypto utilities
export {
  generateSecureSalt,
  isValidHex,
  isValidAddress
} from './utils/crypto';

// Delegate resolver utilities
export {
  getCypheraDelegateAddress,
  isValidDelegateAddress
} from './utils/delegate-resolver';

// Network switching utilities
export {
  switchToNetwork,
  getCurrentChainId,
  getChainIdFromNetworkName,
  ensureCorrectNetwork
} from './utils/network-switching';
export type { Web3Provider } from './utils/network-switching';

// Wallet abstraction (for future extensibility)
export type {
  WalletProvider,
  WalletProviderType,
  WalletProviderConfig,
  WalletCapabilities,
  DelegationParams
} from './wallets/interfaces/wallet-provider';

export {
  WalletFactory,
  createWalletProvider
} from './wallets/factory/wallet-factory';

export {
  BaseWalletProvider
} from './wallets/providers/base-wallet-provider';

// Redemption utilities
export * from './redemption';

// Note: Network configuration is now managed in the database
// The static network registry has been removed

// Version
export const VERSION = '1.1.0';