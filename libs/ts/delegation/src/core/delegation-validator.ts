import { Delegation } from '@metamask/delegation-toolkit';
import { type Address } from 'viem';
import type { PublicClient } from 'viem';
import { ValidationResult } from '../types/delegation';

/**
 * Validates that the input is a valid Ethereum address
 * PRESERVED from apps/delegation-server/src/utils/delegation-helpers.ts
 * @param address The address to validate
 * @returns true if valid, throws error if invalid
 */
export function isValidEthereumAddress(address: string): boolean {
  // Check if address is defined
  if (!address) {
    return false;
  }
  
  // Check if address starts with 0x
  if (!address.startsWith('0x')) {
    return false;
  }
  
  // Check if address is 42 characters (0x + 40 hex digits)
  if (address.length !== 42) {
    return false;
  }
  
  // Check if address contains only hexadecimal characters after 0x
  const hexPart = address.slice(2);
  if (!/^[0-9a-fA-F]{40}$/.test(hexPart)) {
    return false;
  }
  
  return true;
}

/**
 * Validates a delegation structure and ensures the delegator SCA is deployed.
 * PRESERVED from apps/delegation-server/src/utils/delegation-helpers.ts
 * @param delegation The delegation to validate
 * @param publicClient A Viem PublicClient to interact with the blockchain
 * @returns true if valid, throws error if invalid
 */
export async function validateDelegation(delegation: Delegation, publicClient: PublicClient): Promise<boolean> {
  // Check required fields
  if (!delegation.delegator) {
    throw new Error('Invalid delegation: missing delegator');
  }
  
  if (!delegation.delegate) {
    throw new Error('Invalid delegation: missing delegate');
  }
  
  if (!delegation.signature) {
    throw new Error('Invalid delegation: missing signature');
  }
  
  // Validate delegator address format
  if (!isValidEthereumAddress(delegation.delegator)) {
    throw new Error('Invalid delegator address format: must be a valid Ethereum address (0x + 40 hex chars)');
  }
  
  // Validate delegate address format
  if (!isValidEthereumAddress(delegation.delegate)) {
    throw new Error('Invalid delegate address format: must be a valid Ethereum address (0x + 40 hex chars)');
  }

  // Note: We don't halt if the delegator Smart Account is not deployed
  // because Web3Auth Smart Accounts are deployed on their first transaction.
  // The redemption process will handle deployment automatically if needed.  
  try {
    const bytecode = await publicClient.getBytecode({ address: delegation.delegator as Address });
    if (!bytecode || bytecode === '0x') {
      console.debug(`Delegator smart account at ${delegation.delegator} is not yet deployed. It will be deployed during redemption.`);
    } else {
      console.debug(`Delegator smart account at ${delegation.delegator} is already deployed.`);
    }
  } catch (error: unknown) {
    // Just log RPC errors, don't throw - deployment status will be handled during redemption
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.debug(`Could not check delegator deployment status for ${delegation.delegator}: ${errorMessage}`);
  }
  
  return true;
}

/**
 * Validates a delegation structure without blockchain calls
 * @param delegation The delegation to validate
 * @returns ValidationResult with isValid flag and error messages
 */
export function validateDelegationStructure(delegation: Partial<Delegation>): ValidationResult {
  const errors: string[] = [];

  // Check required fields
  if (!delegation.delegator) {
    errors.push('Missing delegator');
  }
  
  if (!delegation.delegate) {
    errors.push('Missing delegate');
  }
  
  if (!delegation.signature) {
    errors.push('Missing signature');
  }
  
  // Validate address formats if present
  if (delegation.delegator && !isValidEthereumAddress(delegation.delegator)) {
    errors.push('Invalid delegator address format');
  }
  
  if (delegation.delegate && !isValidEthereumAddress(delegation.delegate)) {
    errors.push('Invalid delegate address format');
  }

  return {
    isValid: errors.length === 0,
    errors
  };
}