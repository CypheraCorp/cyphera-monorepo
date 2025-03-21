import { logger } from '../utils/utils'

// Conditionally import real or mock blockchain service based on MOCK_MODE
let redeemDelegation: (delegationData: Uint8Array) => Promise<string>;

if (process.env.MOCK_MODE === 'true') {
  logger.info('Running in MOCK MODE - using mock blockchain service')
  import('./mock-blockchain').then(module => {
    redeemDelegation = module.redeemDelegation;
  });
} else {
  import('./blockchain').then(module => {
    redeemDelegation = module.redeemDelegation;
  });
}

// Define types for gRPC service
interface RedeemDelegationRequest {
  request: {
    delegationData: Buffer;
  }
}

interface RedeemDelegationCallback {
  (error: Error | null, response: {
    transactionHash: string;
    success: boolean;
    errorMessage: string;
  }): void;
}

/**
 * Implementation of the DelegationService gRPC service
 */
export const delegationService = {
  /**
   * Redeems a delegation by processing the delegation data and executing on-chain transactions
   * 
   * @param call - The gRPC call containing the delegation data
   * @param callback - The gRPC callback to return the result
   */
  async redeemDelegation(call: RedeemDelegationRequest, callback: RedeemDelegationCallback) {
    const startTime = Date.now()
    logger.info("Received delegation redemption request")
    
    try {
      // Wait for the dynamic import to complete
      if (!redeemDelegation) {
        // Import is still in progress, wait for it to complete
        if (process.env.MOCK_MODE === 'true') {
          const module = await import('./mock-blockchain');
          redeemDelegation = module.redeemDelegation;
        } else {
          const module = await import('./blockchain');
          redeemDelegation = module.redeemDelegation;
        }
      }
      
      // Extract the delegation data from the request
      const delegationData = call.request.delegationData
      
      if (!delegationData || delegationData.length === 0) {
        throw new Error("Delegation data is empty or invalid")
      }
      
      logger.debug(`Received delegation data of size: ${delegationData.length} bytes`)
      
      // Redeem the delegation using the blockchain service
      const transactionHash = await redeemDelegation(new Uint8Array(delegationData))
      
      const elapsedTime = (Date.now() - startTime) / 1000
      logger.info(`Delegation redeemed successfully in ${elapsedTime.toFixed(2)}s, transaction hash: ${transactionHash}`)
      
      // Return success response
      callback(null, {
        transactionHash,
        success: true,
        errorMessage: ''
      })
    } catch (error) {
      const elapsedTime = (Date.now() - startTime) / 1000
      logger.error(`Error redeeming delegation after ${elapsedTime.toFixed(2)}s:`, error)
      
      // Return error response
      callback(null, {
        transactionHash: '',
        success: false,
        errorMessage: error instanceof Error ? error.message : 'Unknown error'
      })
    }
  }
} 