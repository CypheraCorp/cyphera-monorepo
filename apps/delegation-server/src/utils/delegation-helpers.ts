/**
 * Delegation helper utilities for delegation server
 * 
 * This file now imports from the shared delegation library to maintain
 * consistency across the platform while preserving existing functionality.
 */

// Import from shared delegation library
export {
  isValidEthereumAddress,
  parseDelegation,
  validateDelegation
} from '@cyphera/delegation';

// Import for local logger compatibility
import { logger } from './utils';

/**
 * Wrapper function to adapt shared library logger to local logger
 * This ensures the delegation server continues to use its existing logger
 */
export async function validateDelegationWithLogger(delegation: any, publicClient: any): Promise<boolean> {
  // The shared library uses console.debug, but delegation server uses a custom logger
  // We can use the shared function directly since it handles the core logic
  const { validateDelegation } = await import('@cyphera/delegation');
  return validateDelegation(delegation, publicClient);
} 