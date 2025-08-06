/**
 * Adapter to bridge MetaMask Delegation Toolkit with Pimlico bundler
 * Handles format conversion between ERC-7677 and ERC-4337
 */

import type { Address } from 'viem';
import { logger } from '@/lib/core/logger/logger-utils';

interface ERC7677UserOperation {
  sender: Address;
  nonce: bigint;
  factory?: Address;
  factoryData?: `0x${string}`;
  callData: `0x${string}`;
  callGasLimit: bigint;
  verificationGasLimit: bigint;
  preVerificationGas: bigint;
  maxFeePerGas: bigint;
  maxPriorityFeePerGas: bigint;
  paymaster?: Address;
  paymasterVerificationGasLimit?: bigint;
  paymasterPostOpGasLimit?: bigint;
  paymasterData?: `0x${string}`;
  signature: `0x${string}`;
}

interface ERC4337UserOperation {
  sender: Address;
  nonce: `0x${string}`;
  initCode: `0x${string}`;
  callData: `0x${string}`;
  callGasLimit: `0x${string}`;
  verificationGasLimit: `0x${string}`;
  preVerificationGas: `0x${string}`;
  maxFeePerGas: `0x${string}`;
  maxPriorityFeePerGas: `0x${string}`;
  paymasterAndData: `0x${string}`;
  signature: `0x${string}`;
}

/**
 * Convert ERC-7677 format to ERC-4337 format
 */
export function convertERC7677ToERC4337(
  userOp: ERC7677UserOperation
): ERC4337UserOperation {
  logger.log('🔄 Converting ERC-7677 to ERC-4337 format...');
  
  // Combine factory and factoryData into initCode
  let initCode: `0x${string}` = '0x';
  if (userOp.factory && userOp.factoryData) {
    // Remove '0x' prefix from factoryData if present
    const factoryData = userOp.factoryData.startsWith('0x') 
      ? userOp.factoryData.slice(2) 
      : userOp.factoryData;
    initCode = `${userOp.factory}${factoryData}` as `0x${string}`;
  }
  
  // Combine paymaster fields into paymasterAndData
  let paymasterAndData: `0x${string}` = '0x';
  if (userOp.paymaster) {
    // Start with paymaster address
    let combined = userOp.paymaster;
    
    // Add verification gas limit (padded to 32 bytes)
    if (userOp.paymasterVerificationGasLimit !== undefined) {
      const verificationGas = userOp.paymasterVerificationGasLimit.toString(16).padStart(64, '0');
      combined += verificationGas;
    }
    
    // Add post-op gas limit (padded to 32 bytes)
    if (userOp.paymasterPostOpGasLimit !== undefined) {
      const postOpGas = userOp.paymasterPostOpGasLimit.toString(16).padStart(64, '0');
      combined += postOpGas;
    }
    
    // Add paymaster data if present
    if (userOp.paymasterData) {
      const data = userOp.paymasterData.startsWith('0x') 
        ? userOp.paymasterData.slice(2) 
        : userOp.paymasterData;
      combined += data;
    }
    
    paymasterAndData = combined as `0x${string}`;
  }
  
  const converted: ERC4337UserOperation = {
    sender: userOp.sender,
    nonce: `0x${userOp.nonce.toString(16)}`,
    initCode,
    callData: userOp.callData,
    callGasLimit: `0x${userOp.callGasLimit.toString(16)}`,
    verificationGasLimit: `0x${userOp.verificationGasLimit.toString(16)}`,
    preVerificationGas: `0x${userOp.preVerificationGas.toString(16)}`,
    maxFeePerGas: `0x${userOp.maxFeePerGas.toString(16)}`,
    maxPriorityFeePerGas: `0x${userOp.maxPriorityFeePerGas.toString(16)}`,
    paymasterAndData,
    signature: userOp.signature,
  };
  
  logger.log('✅ Conversion complete:', {
    hadFactory: !!userOp.factory,
    hadPaymaster: !!userOp.paymaster,
    initCodeLength: initCode.length,
    paymasterAndDataLength: paymasterAndData.length,
  });
  
  return converted;
}

/**
 * Create a custom bundler client that wraps the MetaMask smart account
 * and converts operations to ERC-4337 format
 */
export function createMetaMaskPimlicoBundlerClient(
  smartAccount: any,
  bundlerUrl: string,
  entryPointAddress: Address = '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'
) {
  return {
    async sendUserOperation(args: {
      account: any;
      calls: Array<{ to: Address; value: bigint; data?: `0x${string}` }>;
      maxFeePerGas?: bigint;
      maxPriorityFeePerGas?: bigint;
      paymasterAndData?: `0x${string}`;
    }): Promise<`0x${string}`> {
      logger.log('🚀 Sending UserOperation through MetaMask-Pimlico adapter...');
      
      try {
        // Check if smart account has prepareUserOperation method
        if (!smartAccount.prepareUserOperation) {
          logger.log('⚠️ Smart account does not have prepareUserOperation, using minimal UserOp');
          
          // Create a minimal UserOperation for deployment
          const minimalUserOp: ERC4337UserOperation = {
            sender: smartAccount.address,
            nonce: '0x0',
            initCode: '0x', // Bundler will fill this if needed
            callData: '0x',
            callGasLimit: '0x5208',
            verificationGasLimit: '0x20000',
            preVerificationGas: '0x5208',
            maxFeePerGas: args.maxFeePerGas ? `0x${args.maxFeePerGas.toString(16)}` : '0x1',
            maxPriorityFeePerGas: args.maxPriorityFeePerGas ? `0x${args.maxPriorityFeePerGas.toString(16)}` : '0x1',
            paymasterAndData: args.paymasterAndData || '0x',
            signature: '0x',
          };
          
          // Send directly to bundler
          const response = await fetch(bundlerUrl, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              jsonrpc: '2.0',
              method: 'eth_sendUserOperation',
              params: [minimalUserOp, entryPointAddress],
              id: Date.now(),
            }),
          });
          
          const result = await response.json();
          
          if (result.error) {
            logger.error('❌ Bundler rejected minimal UserOperation:', result.error);
            throw new Error(result.error.message || 'Failed to send UserOperation');
          }
          
          logger.log('✅ Minimal UserOperation accepted by bundler:', result.result);
          return result.result as `0x${string}`;
        }
        
        // First, try to get the user operation from the smart account
        // This will be in ERC-7677 format
        const userOp = await smartAccount.prepareUserOperation({
          calls: args.calls,
          maxFeePerGas: args.maxFeePerGas,
          maxPriorityFeePerGas: args.maxPriorityFeePerGas,
        });
        
        logger.log('📝 Prepared UserOperation from MetaMask smart account:', {
          hasFactory: !!userOp.factory,
          hasPaymaster: !!userOp.paymaster,
          format: 'ERC-7677',
        });
        
        // Convert to ERC-4337 format
        const convertedUserOp = convertERC7677ToERC4337(userOp);
        
        // Override paymasterAndData if provided
        if (args.paymasterAndData) {
          convertedUserOp.paymasterAndData = args.paymasterAndData;
        }
        
        // Send to Pimlico bundler
        const response = await fetch(bundlerUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            jsonrpc: '2.0',
            method: 'eth_sendUserOperation',
            params: [convertedUserOp, entryPointAddress],
            id: Date.now(),
          }),
        });
        
        const result = await response.json();
        
        if (result.error) {
          logger.error('❌ Bundler rejected UserOperation:', result.error);
          throw new Error(result.error.message || 'Failed to send UserOperation');
        }
        
        logger.log('✅ UserOperation accepted by bundler:', result.result);
        return result.result as `0x${string}`;
      } catch (error) {
        logger.error('❌ Failed to send UserOperation through adapter:', error);
        throw error;
      }
    },
    
    async waitForUserOperationReceipt(args: {
      hash: `0x${string}`;
      timeout?: number;
    }) {
      const startTime = Date.now();
      const timeout = args.timeout || 60000;
      
      logger.log(`⏳ Waiting for UserOperation receipt: ${args.hash}`);
      
      while (Date.now() - startTime < timeout) {
        try {
          const response = await fetch(bundlerUrl, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              jsonrpc: '2.0',
              method: 'eth_getUserOperationReceipt',
              params: [args.hash],
              id: Date.now(),
            }),
          });
          
          const result = await response.json();
          
          if (result.result) {
            logger.log('✅ UserOperation receipt received:', result.result);
            return {
              success: result.result.success,
              receipt: {
                transactionHash: result.result.receipt.transactionHash,
                blockNumber: result.result.receipt.blockNumber,
              },
            };
          }
        } catch (error) {
          logger.warn('⚠️ Error checking receipt, will retry:', { error: error instanceof Error ? error.message : 'Unknown error' });
        }
        
        // Wait 2 seconds before retrying
        await new Promise(resolve => setTimeout(resolve, 2000));
      }
      
      throw new Error(`Timeout waiting for UserOperation receipt: ${args.hash}`);
    },
  };
}