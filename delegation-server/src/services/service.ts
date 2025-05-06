import { ServerUnaryCall, sendUnaryData } from '@grpc/grpc-js'
import { logger } from '../utils/utils'
import config from '../config'

// Conditionally import real or mock blockchain service based on MOCK_MODE
let redeemDelegation: (
  delegationData: Uint8Array, 
  merchantAddress: string, 
  tokenContractAddress: string, 
  price: string,
  chainId: number,
  networkName: string
) => Promise<string>;

logger.info('===== SERVICE.TS INITIALIZATION =====');
logger.info(`MOCK_MODE from environment: "${process.env.MOCK_MODE || 'not set'}"`);
logger.info(`MOCK_MODE from config: ${config.mockMode}`);

if (config.mockMode) {
  logger.info('SERVICE.TS: Running in MOCK MODE - using mock blockchain service')
  import('./mock-redeem-delegation').then(module => {
    redeemDelegation = module.redeemDelegation;
    logger.info('SERVICE.TS: Successfully loaded MOCK blockchain service')
  }).catch(error => {
    logger.error('SERVICE.TS: Failed to load mock blockchain service:', error);
  });
} else {
  logger.info('SERVICE.TS: Running in REAL MODE - using real blockchain service')
  import('./redeem-delegation').then(module => {
    redeemDelegation = module.redeemDelegation;
    logger.info('SERVICE.TS: Successfully loaded REAL blockchain service')
  }).catch(error => {
    logger.error('SERVICE.TS: Failed to load real blockchain service:', error);
  });
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
  async redeemDelegation(call: ServerUnaryCall<any, any>, callback: sendUnaryData<any>) {
    try {
      logger.info('Received RedeemDelegation request');
      
      // Check if the implementation was loaded
      if (!redeemDelegation) {
        logger.error('Blockchain service implementation not loaded yet');
        callback(null, {
          transaction_hash: "",
          transactionHash: "",
          success: false,
          error_message: "Service not ready yet, try again later",
          errorMessage: "Service not ready yet, try again later"
        });
        return;
      }

      // Extract request parameters
      const signature = call.request.signature;
      const merchantAddress = call.request.merchant_address || call.request.merchantAddress;
      const tokenContractAddress = call.request.token_contract_address || call.request.tokenContractAddress;
      const price = call.request.price;
      const chainId = call.request.chain_id;
      const networkName = call.request.network_name;

      // Basic validation for new parameters
      if (chainId === undefined || chainId === null || chainId <= 0) {
        throw new Error('Missing or invalid chain_id in request');
      }
      if (!networkName) {
        throw new Error('Missing network_name in request');
      }

      logger.info('Request parameters:', {
        signatureLength: signature ? signature.length : 0,
        merchantAddress,
        tokenContractAddress,
        price,
        chainId,
        networkName
      });

      // Call the implementation with new parameters
      const transactionHash = await redeemDelegation(
        signature,
        merchantAddress,
        tokenContractAddress,
        price,
        chainId,
        networkName
      );

      logger.info(`Redemption successful, transaction hash: ${transactionHash}`);

      // Send success response with both snake_case and camelCase fields for compatibility
      callback(null, {
        transaction_hash: transactionHash,
        transactionHash: transactionHash,
        success: true,
        error_message: "",
        errorMessage: ""
      });
    } catch (error) {
      // Handle errors
      const errorMessage = error instanceof Error ? error.message : String(error);
      logger.error('Error in redeemDelegation:', errorMessage);
      
      // Send error response with both snake_case and camelCase fields for compatibility
      callback(null, {
        transaction_hash: "",
        transactionHash: "",
        success: false,
        error_message: errorMessage,
        errorMessage: errorMessage
      });
    }
  }
};
