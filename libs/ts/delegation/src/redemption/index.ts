/**
 * Redemption module exports
 * 
 * This module provides reusable functionality for redeeming delegations
 * across the Cyphera platform.
 */

// Types
export * from './types';

// Validation utilities
export {
  validateRedemptionInputs,
  validateAddress,
  validateTokenAmount,
  validateTokenDecimals,
  validateChainId,
  validateDelegateMatch
} from './validation';

// Blockchain client utilities
export {
  getChainById,
  createFetchTransport,
  initializeBlockchainClients,
  createNetworkConfigFromUrls
} from './blockchain-clients';

// Payload building utilities
export {
  prepareRedemptionUserOperationPayload,
  prepareTokenAmount,
  encodeERC20Transfer,
  buildExecutionStruct,
  encodeDelegationRedemption,
  prepareBatchRedemptionPayload,
  type BatchRedemptionDetails
} from './payload-builder';

// Smart account management
export {
  getOrCreateRedeemerAccount,
  formatPrivateKey,
  validateDelegateMatch as validateRedeemerDelegateMatch,
  checkSmartAccountDeployment,
  calculateSmartAccountAddress,
  getSmartAccountInfo,
  type DeterministicDeploymentConfig,
  type SmartAccountInfo
} from './smart-account';

// UserOperation handling
export {
  fetchGasPrices,
  sendAndConfirmUserOperation,
  estimateUserOperationGasCost,
  type SendUserOperationOptions
} from './user-operation';

// Constants
export * from './constants';