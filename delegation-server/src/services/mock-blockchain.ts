/**
 * Mock blockchain service for testing purposes
 * This file provides mock implementations of the blockchain service methods
 * to enable testing without actual blockchain interactions.
 */
import { logger } from '../utils/utils'
import { parseDelegation, validateDelegation } from '../utils/delegation-helpers'

/**
 * Mock implementation of the redeemDelegation function
 * @param delegationData The serialized delegation data
 * @returns A mock transaction hash
 */
export const redeemDelegation = async (
  delegationData: Uint8Array
): Promise<string> => {
  try {
    // Parse and validate the delegation - this is real code that will actually check
    // the delegation format, so our test is still meaningful
    const delegation = parseDelegation(delegationData)
    validateDelegation(delegation)
    
    logger.info("[MOCK] Redeeming delegation...")
    logger.debug("[MOCK] Delegation details:", {
      delegate: delegation.delegate,
      delegator: delegation.delegator,
      expiry: delegation.expiry?.toString()
    })
    
    // Simulate processing time to make the test more realistic
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // Generate a mock transaction hash
    const mockTxHash = '0x' + [...Array(64)].map(() => Math.floor(Math.random() * 16).toString(16)).join('')
    
    logger.info("[MOCK] Transaction confirmed:", mockTxHash)
    
    return mockTxHash
  } catch (error) {
    logger.error("[MOCK] Error redeeming delegation:", error)
    throw error
  }
} 