/**
 * Utility functions for handling delegation data
 */
import { DelegationFramework } from '@metamask-private/delegator-core-viem'
import { logger } from './utils'
import { DelegationStruct } from '../types/delegation'

/**
 * Parse a delegation from either bytes or JSON format
 * @param delegationData The delegation data as either Uint8Array or Buffer
 * @returns The parsed delegation structure
 */
export function parseDelegation(delegationData: Uint8Array | Buffer): DelegationStruct {
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
      return parsedObject as DelegationStruct
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
 * Validates a delegation structure
 * @param delegation The delegation to validate
 * @returns true if valid, throws error if invalid
 */
export function validateDelegation(delegation: DelegationStruct): boolean {
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
  
  // Check if delegation is expired
  if (delegation.expiry && typeof delegation.expiry === 'bigint') {
    const now = BigInt(Math.floor(Date.now() / 1000))
    if (delegation.expiry > 0n && delegation.expiry < now) {
      throw new Error(`Delegation is expired (expiry: ${delegation.expiry}, now: ${now})`)
    }
  }
  
  return true
} 