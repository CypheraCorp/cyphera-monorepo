/**
 * Utility functions for handling delegation data
 */
import { Delegation } from '@metamask/delegation-toolkit'
import { logger } from './utils'
import { type Address, isAddress } from 'viem'
import type { PublicClient } from 'viem'


/**
 * Validates that the input is a valid Ethereum address
 * @param address The address to validate
 * @returns true if valid, throws error if invalid
 */
export function isValidEthereumAddress(address: string): boolean {
  // Check if address is defined
  if (!address) {
    return false
  }
  
  // Check if address starts with 0x
  if (!address.startsWith('0x')) {
    return false
  }
  
  // Check if address is 42 characters (0x + 40 hex digits)
  if (address.length !== 42) {
    return false
  }
  
  // Check if address contains only hexadecimal characters after 0x
  const hexPart = address.slice(2)
  if (!/^[0-9a-fA-F]{40}$/.test(hexPart)) {
    return false
  }
  
  return true
}

/**
 * Parse a delegation from either bytes or JSON format
 * @param delegationData The delegation data as either Uint8Array or Buffer
 * @returns The parsed delegation structure
 */
export function parseDelegation(delegationData: Uint8Array | Buffer): Delegation {
  try {
    try {
      // Try to parse as JSON first (most common case in our system)
      const jsonString = Buffer.from(delegationData).toString('utf-8')
      const parsedObject = JSON.parse(jsonString)
      
      // Validate the parsed object has the required fields
      if (!parsedObject.delegator || !parsedObject.delegate) {
        throw new Error('Invalid delegation format: missing required fields')
      }
      
      // If parsed successfully, return the object
      return parsedObject as Delegation
    } catch (jsonError) {
      logger.debug('Failed to parse delegation as JSON, trying alternative format', jsonError)
      
      // If JSON parsing fails, the delegation might be in a different format
      // In a real implementation, we would attempt other parsing methods here
      // However, since we don't have direct access to DelegationFramework.decode,
      // we'll throw a more descriptive error
      
      throw new Error('Failed to parse delegation: Binary format not supported in this version')
    }
  } catch (error) {
    logger.error('Failed to parse delegation data', error)
    throw new Error(`Failed to parse delegation: ${error instanceof Error ? error.message : String(error)}`)
  }
}

/**
 * Validates a delegation structure and ensures the delegator SCA is deployed.
 * @param delegation The delegation to validate
 * @param publicClient A Viem PublicClient to interact with the blockchain
 * @returns true if valid, throws error if invalid
 */
export async function validateDelegation(delegation: Delegation, publicClient: PublicClient): Promise<boolean> {
  // Check required fields
  if (!delegation.delegator) {
    throw new Error('Invalid delegation: missing delegator')
  }
  
  if (!delegation.delegate) {
    throw new Error('Invalid delegation: missing delegate')
  }
  
  if (!delegation.signature) {
    throw new Error('Invalid delegation: missing signature')
  }
  
  // Validate delegator address format
  if (!isValidEthereumAddress(delegation.delegator)) {
    throw new Error('Invalid delegator address format: must be a valid Ethereum address (0x + 40 hex chars)')
  }
  
  // Validate delegate address format
  if (!isValidEthereumAddress(delegation.delegate)) {
    throw new Error('Invalid delegate address format: must be a valid Ethereum address (0x + 40 hex chars)')
  }

  // Check if the delegator Smart Account (customer's account) is deployed
  try {
    const bytecode = await publicClient.getBytecode({ address: delegation.delegator as Address });
    if (!bytecode || bytecode === '0x') {
      throw new Error(`Invalid delegation: Delegator smart account at ${delegation.delegator} is not deployed. The customer must deploy their account first.`);
    }
  } catch (error: any) {
    // Catch errors from getBytecode (e.g., RPC issues) and re-throw appropriately
    if (error.message.includes('Delegator smart account') && error.message.includes('is not deployed')) {
        throw error; // Re-throw our specific error
    }
    throw new Error(`Failed to verify delegator deployment status for ${delegation.delegator}: ${error.message}`);
  }
  
  return true
} 