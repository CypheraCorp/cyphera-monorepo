/**
 * Mock blockchain service for testing purposes
 * This file provides mock implementations of the blockchain service methods
 * to enable testing without actual blockchain interactions.
 */
import { logger } from '../utils/utils'
import { parseDelegation, validateDelegation } from '../utils/delegation-helpers'

/**
 * Mock implementation of the redeemDelegation function
 * @param delegationData The serialized delegation data (signature)
 * @param merchantAddress The address of the merchant
 * @param tokenContractAddress The address of the token contract
 * @param tokenAmount The amount of tokens to redeem
 * @param tokenDecimals The number of decimals of the token
 * @returns A mock transaction hash
 */
export const redeemDelegation = async (
  delegationData: Uint8Array,
  merchantAddress: string,
  tokenContractAddress: string,
  tokenAmount: number,
  tokenDecimals: number
): Promise<string> => {
  try {
    // Validate inputs first to avoid undefined errors
    if (!delegationData || delegationData.length === 0) {
      throw new Error("Delegation data is required");
    }
    
    if (!merchantAddress) {
      throw new Error("Merchant address is required");
    }
    
    if (!tokenContractAddress) {
      throw new Error("Token contract address is required");
    }
    
    if (!tokenAmount) {
      throw new Error("Token amount is required");
    }

    if (!tokenDecimals) {
      throw new Error("Token decimals are required");
    }
    
    // Parse and validate the delegation - this is real code that will actually check
    // the delegation format, so our test is still meaningful
    const delegation = parseDelegation(delegationData)
    validateDelegation(delegation)
    
    logger.info("[MOCK] Redeeming delegation...")
    logger.debug("[MOCK] Delegation details:", {
      delegate: delegation.delegate,
      delegator: delegation.delegator,
      merchantAddress,
      tokenContractAddress,
      tokenAmount,
      tokenDecimals
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