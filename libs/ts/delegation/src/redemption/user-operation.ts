import { 
  type Address,
  type Hex,
  type PublicClient,
  formatEther
} from 'viem';
import { 
  type UserOperationReceipt,
  UserOperationExecutionError
} from 'viem/account-abstraction';
import { type MetaMaskSmartAccount, type Implementation, type Call } from '@metamask/delegation-toolkit';
import { 
  RedemptionError, 
  RedemptionErrorType, 
  GasPrices,
  UserOperationReceiptInfo 
} from './types';
import { 
  DEFAULT_USER_OPERATION_TIMEOUT,
  MAX_USER_OPERATION_RETRIES,
  RETRY_DELAY_MS
} from './constants';

/**
 * Fetches current gas prices from Pimlico
 * @param pimlicoClient The Pimlico client
 * @returns Gas prices for the UserOperation
 */
export async function fetchGasPrices(pimlicoClient: unknown): Promise<GasPrices> {
  try {
    const { fast: gasPrices } = await (pimlicoClient as { getUserOperationGasPrice: () => Promise<{ fast: { maxFeePerGas?: bigint; maxPriorityFeePerGas?: bigint } }> }).getUserOperationGasPrice();
    
    if (!gasPrices || !gasPrices.maxFeePerGas || !gasPrices.maxPriorityFeePerGas) {
      throw new RedemptionError(
        'Could not fetch gas prices for the UserOperation',
        RedemptionErrorType.USER_OPERATION_ERROR
      );
    }

    return {
      maxFeePerGas: gasPrices.maxFeePerGas,
      maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas
    };
  } catch (error) {
    throw new RedemptionError(
      'Failed to fetch gas prices',
      RedemptionErrorType.NETWORK_ERROR,
      error
    );
  }
}

/**
 * Sends a UserOperation and waits for its confirmation
 * @param bundlerClient The bundler client
 * @param pimlicoClient The Pimlico client for gas prices
 * @param redeemer The smart account performing the redemption
 * @param calls The calls to execute
 * @param publicClient The public client for additional checks
 * @param options Additional options for the operation
 * @returns The transaction hash
 */
export interface SendUserOperationOptions {
  /** Custom timeout for waiting (defaults to 2 minutes) */
  timeout?: number;
  /** Whether to retry on failure */
  retryOnFailure?: boolean;
  /** Custom gas prices (if not provided, will fetch from Pimlico) */
  gasPrices?: GasPrices;
  /** Callback for status updates */
  onStatusUpdate?: (status: string) => void;
}

export async function sendAndConfirmUserOperation(
  bundlerClient: unknown,
  pimlicoClient: unknown,
  redeemer: MetaMaskSmartAccount<Implementation>,
  calls: Call[],
  publicClient: PublicClient,
  options: SendUserOperationOptions = {}
): Promise<string> {
  const {
    timeout = DEFAULT_USER_OPERATION_TIMEOUT,
    retryOnFailure = true,
    onStatusUpdate = () => { /* Empty callback */ }
  } = options;

  let userOpHash: Address | undefined;
  let retries = 0;

  while (retries <= (retryOnFailure ? MAX_USER_OPERATION_RETRIES : 0)) {
    try {
      // Fetch gas prices if not provided
      const gasPrices = options.gasPrices || await fetchGasPrices(pimlicoClient);
      
      onStatusUpdate(`Using gas prices: maxFeePerGas: ${formatEther(gasPrices.maxFeePerGas)} gwei, maxPriorityFeePerGas: ${formatEther(gasPrices.maxPriorityFeePerGas)} gwei`);

      // Send the UserOperation
      onStatusUpdate('Sending UserOperation...');
      const startTime = Date.now();
      
      userOpHash = await (bundlerClient as { sendUserOperation: (params: { account: MetaMaskSmartAccount<Implementation>; calls: Call[]; maxFeePerGas: bigint; maxPriorityFeePerGas: bigint }) => Promise<Address> }).sendUserOperation({
        account: redeemer,
        calls,
        maxFeePerGas: gasPrices.maxFeePerGas,
        maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas,
      });

      onStatusUpdate(`UserOperation sent: ${userOpHash}`);

      // Wait for confirmation
      onStatusUpdate('Waiting for UserOperation confirmation...');
      const receipt = await waitForUserOperationReceipt(
        bundlerClient,
        userOpHash,
        timeout
      );

      if (!receipt.success) {
        throw new RedemptionError(
          'UserOperation did not succeed',
          RedemptionErrorType.USER_OPERATION_ERROR,
          { receipt }
        );
      }

      // Verify smart account deployment if needed
      await verifySmartAccountDeployment(redeemer, publicClient, onStatusUpdate);

      const transactionHash = receipt.receipt.transactionHash;
      const totalTime = (Date.now() - startTime) / 1000;
      onStatusUpdate(`Transaction confirmed in ${totalTime}s: ${transactionHash}`);

      return transactionHash;

    } catch (error) {
      const errorMessage = formatUserOperationError(error, userOpHash);
      
      if (retryOnFailure && retries < MAX_USER_OPERATION_RETRIES) {
        retries++;
        onStatusUpdate(`Retry ${retries}/${MAX_USER_OPERATION_RETRIES} after error: ${errorMessage}`);
        await delay(RETRY_DELAY_MS);
        continue;
      }

      // Check deployment status on error
      try {
        const isDeployed = await redeemer.isDeployed();
        onStatusUpdate(`Smart account deployment status on error: ${isDeployed}`);
      } catch {
        // Ignore deployment check errors
      }

      throw new RedemptionError(
        errorMessage,
        RedemptionErrorType.USER_OPERATION_ERROR,
        { originalError: error, userOpHash }
      );
    }
  }

  throw new RedemptionError(
    'Failed to send UserOperation after all retries',
    RedemptionErrorType.USER_OPERATION_ERROR
  );
}

/**
 * Waits for a UserOperation receipt
 * @param bundlerClient The bundler client
 * @param userOpHash The UserOperation hash
 * @param timeout Timeout in milliseconds
 * @returns The UserOperation receipt
 */
async function waitForUserOperationReceipt(
  bundlerClient: unknown,
  userOpHash: Address,
  timeout: number
): Promise<UserOperationReceiptInfo> {
  const receipt = await (bundlerClient as { waitForUserOperationReceipt: (params: { hash: Address; timeout: number }) => Promise<UserOperationReceipt> }).waitForUserOperationReceipt({ 
    hash: userOpHash, 
    timeout 
  }) as UserOperationReceipt;

  return {
    success: receipt.success,
    receipt: {
      transactionHash: receipt.receipt.transactionHash as Hex,
      blockNumber: receipt.receipt.blockNumber,
      gasUsed: receipt.receipt.gasUsed || 0n
    }
  };
}

/**
 * Verifies that a smart account is deployed after a UserOperation
 * @param smartAccount The smart account
 * @param publicClient The public client
 * @param onStatusUpdate Status update callback
 */
async function verifySmartAccountDeployment(
  smartAccount: MetaMaskSmartAccount<Implementation>,
  publicClient: PublicClient,
  onStatusUpdate: (status: string) => void
): Promise<void> {
  const isNowDeployed = await smartAccount.isDeployed();
  
  if (isNowDeployed) {
    onStatusUpdate(`Smart Account ${smartAccount.address} is confirmed deployed.`);
  } else {
    // Double-check with bytecode
    const code = await publicClient.getBytecode({ address: smartAccount.address });
    if (!code || code === '0x') {
      onStatusUpdate(`Warning: SA ${smartAccount.address} bytecode not found after UserOp, but UserOp reported success.`);
    } else {
      onStatusUpdate(`Bytecode found at ${smartAccount.address}. Assuming deployed despite isDeployed() returning false.`);
    }
  }
}

/**
 * Formats UserOperation errors for better readability
 * @param error The error object
 * @param userOpHash The UserOperation hash if available
 * @returns Formatted error message
 */
function formatUserOperationError(error: unknown, userOpHash?: Address): string {
  let errorMsg = `Error during UserOperation (hash: ${userOpHash || 'N/A'})`;

  if (error instanceof UserOperationExecutionError) {
    errorMsg += `: ${error.message}`;
    if (error.cause?.details || error.cause?.message) {
      errorMsg += ` Reason: ${error.cause.details || error.cause.message}`;
    }
  } else if (error instanceof Error) {
    errorMsg += `: ${error.message}`;
  } else {
    errorMsg += ': Unknown error';
  }

  return errorMsg;
}

/**
 * Simple delay utility
 * @param ms Milliseconds to delay
 */
function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Estimates the gas cost for a UserOperation
 * @param pimlicoClient The Pimlico client
 * @param userOp The UserOperation to estimate
 * @returns Estimated gas cost in wei
 */
export async function estimateUserOperationGasCost(
  pimlicoClient: unknown,
  userOp: unknown
): Promise<bigint> {
  try {
    const estimation = await (pimlicoClient as { estimateUserOperationGas: (userOp: unknown) => Promise<{ callGasLimit?: bigint; verificationGasLimit?: bigint; preVerificationGas?: bigint }> }).estimateUserOperationGas(
      userOp
    );

    const gasPrices = await fetchGasPrices(pimlicoClient);
    
    // Calculate total estimated cost
    const totalGas = (estimation.callGasLimit || 0n) + 
                    (estimation.verificationGasLimit || 0n) + 
                    (estimation.preVerificationGas || 0n);
    
    return totalGas * gasPrices.maxFeePerGas;
  } catch (error) {
    throw new RedemptionError(
      'Failed to estimate UserOperation gas cost',
      RedemptionErrorType.USER_OPERATION_ERROR,
      error
    );
  }
}