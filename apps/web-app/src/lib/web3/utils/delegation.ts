/**
 * Web3 delegation utilities
 * 
 * This file now imports from the shared delegation library to maintain
 * consistency across the platform while preserving existing functionality.
 */

// Import from shared delegation library
export {
  createSalt,
  createAndSignDelegation,
  formatDelegation,
  isValidEthereumAddress,
  getCypheraDelegateAddress,
  isValidDelegateAddress,
  switchToNetwork,
  getCurrentChainId,
  getChainIdFromNetworkName,
  ensureCorrectNetwork
} from '@cyphera/delegation';

export type { Web3Provider } from '@cyphera/delegation';

// Re-export types for backward compatibility
export type { Delegation } from '@metamask/delegation-toolkit';
