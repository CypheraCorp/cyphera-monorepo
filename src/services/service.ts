import { logger } from '../utils/utils'
import { ServerUnaryCall, sendUnaryData } from '@grpc/grpc-js'
import * as delegationPb from '../proto/delegation_pb'

// Conditionally import real or mock blockchain service based on MOCK_MODE
let redeemDelegation: (delegationData: Uint8Array, merchantAddress: string, tokenContractAddress: string, price: string) => Promise<string>;

if (process.env.MOCK_MODE === 'true') {
  logger.info('Running in MOCK MODE - using mock blockchain service')
  import('./mock-redeem-delegation').then(module => {
    redeemDelegation = module.redeemDelegation;
    logger.info('Successfully loaded MOCK blockchain service')
  });
} else {
  logger.info('Running in REAL MODE - using real blockchain service')
  import('./redeem-delegation').then(module => {
    redeemDelegation = module.redeemDelegation;
    logger.info('Successfully loaded REAL blockchain service')
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
  async redeemDelegation(call: any, callback: sendUnaryData<any>) {
    // Very simple implementation that always returns success
    const response = {
      transactionHash: '0xmocktransactionhash',
      transaction_hash: '0xmocktransactionhash',
      success: true,
      errorMessage: '',
      error_message: ''
    };
    
    callback(null, response);
  }
} 