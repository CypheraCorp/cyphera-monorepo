import { type Address, type PublicClient, parseEther, formatEther } from 'viem';
import {
  createBundlerClientForChain,
  isPimlicoSupported,
  isPimlicoConfigured,
} from '@/lib/web3/config/pimlico';
import {
  getNetworkConfig,
  getGasForPriority,
  getDeploymentGasLimit,
} from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';

// Type for Web3 provider with request method
interface Web3Provider {
  request(args: { method: string; params?: unknown[] }): Promise<unknown>;
}

// Type for Smart Account
interface SmartAccount {
  address: Address;
  isDeployed?: () => Promise<boolean>;
  client?: {
    account?: unknown;
    sendTransaction?: (args: unknown) => Promise<string>;
  };
}

/**
 * Gas configuration options for smart account deployment
 */
interface DeploymentGasConfig {
  gasLimit?: bigint;
  gasPrice?: bigint;
  maxFeePerGas?: bigint;
  maxPriorityFeePerGas?: bigint;
}

/**
 * Check if a smart account is deployed on the specified chain
 * @param smartAccountAddress - The smart account address to check
 * @param publicClient - The public client for the chain
 * @returns Promise<boolean> - True if deployed, false otherwise
 */
export async function isSmartAccountDeployed(
  smartAccountAddress: Address,
  publicClient: PublicClient
): Promise<boolean> {
  try {
    const bytecode = await publicClient.getBytecode({
      address: smartAccountAddress,
    });

    const isDeployed = bytecode !== undefined && bytecode !== '0x';
    logger.log(
      `📋 Smart account ${smartAccountAddress} deployment status: ${isDeployed ? 'DEPLOYED' : 'NOT DEPLOYED'}`
    );
    return isDeployed;
  } catch (error) {
    logger.error_sync('Error checking smart account deployment:', error);
    return false;
  }
}

/**
 * Get network-optimized gas prices for smart account deployments
 * Now uses gas configuration from backend
 */
async function getOptimizedGasPrices(chainId: number): Promise<DeploymentGasConfig> {
  // Always use 'standard' priority from backend
  const gasSettings = await getGasForPriority(chainId, 'standard');

  if (gasSettings) {
    return {
      maxFeePerGas: gasSettings.maxFeePerGas,
      maxPriorityFeePerGas: gasSettings.maxPriorityFeePerGas,
    };
  }

  // Fallback to hardcoded values if backend doesn't provide gas config
  const networkConfig = await getNetworkConfig(chainId);
  const isMainnet = networkConfig && !networkConfig.chain.testnet;

  if (isMainnet) {
    return {
      maxFeePerGas: parseEther('0.00000005'), // 50 gwei
      maxPriorityFeePerGas: parseEther('0.000000002'), // 2 gwei
    };
  } else {
    // Testnet - higher gas prices to ensure inclusion
    return {
      maxFeePerGas: parseEther('0.0000001'), // 100 gwei
      maxPriorityFeePerGas: parseEther('0.000000005'), // 5 gwei
    };
  }
}

/**
 * Estimate gas for smart account deployment
 */
export async function estimateSmartAccountDeploymentGas(
  smartAccountAddress: Address,
  publicClient: PublicClient,
  chainId: number
): Promise<DeploymentGasConfig> {
  try {
    // Get optimized gas prices from centralized config
    const gasPrices = await getOptimizedGasPrices(chainId);

    // Estimate gas for a simple deployment transaction
    const gasLimit = await publicClient.estimateGas({
      to: smartAccountAddress,
      value: BigInt(0),
      data: '0x',
    });

    // Add 20% buffer for gas limit
    const bufferedGasLimit = (gasLimit * BigInt(120)) / BigInt(100);

    return {
      gasLimit: bufferedGasLimit,
      maxFeePerGas: gasPrices.maxFeePerGas,
      maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas,
    };
  } catch (_error) {
    logger.warn_sync('Failed to estimate deployment gas, using defaults:');

    // Fallback to default values from centralized config
    const gasPrices = await getOptimizedGasPrices(chainId);
    const deploymentGasLimit = await getDeploymentGasLimit(chainId);
    return {
      gasLimit: deploymentGasLimit || BigInt(100000), // Use backend value or default
      maxFeePerGas: gasPrices.maxFeePerGas,
      maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas,
    };
  }
}

/**
 * Deploy a smart account using standard transaction (requires gas fees)
 */
export async function deploySmartAccountWithGas(
  smartAccount: SmartAccount,
  chainId: number,
  publicClient: PublicClient
): Promise<void> {
  try {
    logger.log(`Deploying smart account ${smartAccount.address} on chain ${chainId}...`);

    // Check if already deployed
    const isDeployed = await isSmartAccountDeployed(smartAccount.address, publicClient);
    if (isDeployed) {
      logger.log('Smart account is already deployed');
      return;
    }

    // Get gas configuration from centralized config
    const gasConfig = await getOptimizedGasPrices(chainId);

    // Get the wallet client from the smart account
    const walletClient = smartAccount.client;
    if (!walletClient || !walletClient.sendTransaction) {
      throw new Error(
        'Smart account must have a wallet client with sendTransaction for deployment'
      );
    }

    // Send deployment transaction
    const deploymentHash = await walletClient.sendTransaction({
      to: smartAccount.address,
      value: BigInt(0),
      data: '0x',
      maxFeePerGas: gasConfig.maxFeePerGas,
      maxPriorityFeePerGas: gasConfig.maxPriorityFeePerGas,
    });

    logger.log(`Deployment transaction sent: ${deploymentHash}`);

    // Wait for confirmation
    const receipt = await publicClient.waitForTransactionReceipt({
      hash: deploymentHash as `0x${string}`,
      timeout: 120000, // 2 minutes
    });

    if (receipt.status === 'success') {
      logger.log(`Smart account deployed successfully: ${receipt.transactionHash}`);
    } else {
      throw new Error('Deployment transaction failed');
    }
  } catch (error) {
    logger.error_sync('Error deploying smart account:', error);
    throw error;
  }
}

/**
 * Deploys a smart account using Pimlico bundler and paymaster
 * This method uses UserOperations instead of direct deployment transactions
 * @param smartAccount - The MetaMask smart account instance
 * @param chainId - The chain ID for the deployment
 * @returns Promise<void>
 *
 * NOTE: This function requires a proper smart account implementation that matches
 * the viem account abstraction types. The current SmartAccount interface is simplified
 * and doesn't include all required methods.
 */
export async function deploySmartAccountWithPimlico(
  smartAccount: SmartAccount,
  chainId: number
): Promise<void> {
  try {
    if (!smartAccount) {
      throw new Error('Smart account instance is required for Pimlico deployment');
    }

    if (!isPimlicoSupported(chainId)) {
      throw new Error(`Pimlico is not supported on chain ${chainId}`);
    }

    if (!(await isPimlicoConfigured())) {
      throw new Error('Pimlico API key is not configured');
    }

    // Check if already deployed
    if (smartAccount.isDeployed) {
      const isDeployed = await smartAccount.isDeployed();
      if (isDeployed) {
        logger.log('Smart account is already deployed');
        return;
      }
    }

    logger.log('Deploying smart account using Pimlico bundler...');

    // Get the wallet client from the smart account
    if (!smartAccount.client || !smartAccount.client.account) {
      throw new Error('Smart account must have a wallet client for Pimlico deployment');
    }

    const walletClient = smartAccount.client;

    // Step 1: Fund the smart account address so it can pay for its own UserOperation
    const fundingAmount = parseEther('0.01'); // 0.01 ETH should be enough for deployment gas

    logger.log(
      `Step 1: Funding smart account ${smartAccount.address} with ${formatEther(fundingAmount)} ETH for gas...`
    );

    if (!walletClient.sendTransaction) {
      throw new Error('Wallet client does not support sendTransaction');
    }

    const fundingTxHash = await walletClient.sendTransaction({
      to: smartAccount.address,
      value: fundingAmount,
    });

    logger.log(`Funding transaction sent: ${fundingTxHash}`);
    logger.log('Waiting for funding transaction to be confirmed...');

    // Wait a bit for the funding transaction to be mined
    await new Promise((resolve) => setTimeout(resolve, 5000));

    // Step 2: Send a UserOperation to deploy the smart account
    logger.log('Step 2: Sending UserOperation to deploy smart account...');

    // Create bundler client using centralized config
    const bundlerClient = await createBundlerClientForChain(chainId);

    // Get optimized gas prices from centralized config
    const gasPrices = await getOptimizedGasPrices(chainId);
    const maxFeePerGas = gasPrices.maxFeePerGas;
    const maxPriorityFeePerGas = gasPrices.maxPriorityFeePerGas;

    logger.log(
      `Using network gas prices: maxFeePerGas: ${maxFeePerGas}, maxPriorityFeePerGas: ${maxPriorityFeePerGas}`
    );

    // The dummy receiver for the deployment transaction
    const DUMMY_RECEIVER_ADDRESS = '0x819f58bEf39B809B34e9Cce706CEf95476035136';
    const ZERO_VALUE = BigInt(0);

    logger.log(
      `Deploying smart account ${smartAccount.address} with zero-value transaction to ${DUMMY_RECEIVER_ADDRESS}`
    );

    // Send UserOperation for deployment
    const userOperationHash = await bundlerClient.sendUserOperation({
      account: smartAccount as Parameters<typeof bundlerClient.sendUserOperation>[0]['account'],
      calls: [
        {
          to: DUMMY_RECEIVER_ADDRESS,
          value: ZERO_VALUE,
          data: '0x',
        },
      ],
      maxFeePerGas,
      maxPriorityFeePerGas,
    });

    logger.log('UserOperation sent. Hash:', userOperationHash);
    logger.log('Waiting for UserOperation receipt...');

    // Wait for the UserOperation to be processed
    const receipt = await bundlerClient.waitForUserOperationReceipt({
      hash: userOperationHash,
      timeout: 120000, // Increased timeout to 120 seconds
    });

    logger.log('UserOperation successful! Transaction Hash:', receipt.receipt.transactionHash);
    logger.log(`Smart Account ${smartAccount.address} has been deployed using Pimlico.`);
  } catch (error) {
    logger.error_sync('Error deploying smart account with Pimlico:', error);

    // Enhanced error handling for specific AA errors
    let enhancedErrorMessage = 'Failed to deploy smart account with Pimlico';

    if (error instanceof Error) {
      const errorMessage = error.message.toLowerCase();

      if (errorMessage.includes('insufficient funds')) {
        enhancedErrorMessage =
          'Insufficient funds for smart account deployment. Please ensure the account has enough ETH for gas fees.';
      } else if (errorMessage.includes('user operation reverted')) {
        enhancedErrorMessage =
          'Smart account deployment transaction reverted. This may be due to network congestion or invalid parameters.';
      } else if (errorMessage.includes('timeout')) {
        enhancedErrorMessage =
          'Smart account deployment timed out. The transaction may still be pending on the network.';
      } else if (errorMessage.includes('bundler')) {
        enhancedErrorMessage =
          'Pimlico bundler error during deployment. Please check your API key and try again.';
      } else if (errorMessage.includes('paymaster')) {
        enhancedErrorMessage =
          'Paymaster error during deployment. Gas sponsorship may be unavailable.';
      } else if (errorMessage.includes('already deployed')) {
        enhancedErrorMessage = 'Smart account is already deployed.';
      } else {
        enhancedErrorMessage = `Smart account deployment failed: ${error.message}`;
      }
    }

    throw new Error(enhancedErrorMessage);
  }
}

/**
 * Deploy smart account by sending a minimal UserOperation with sponsored gas
 * This follows the backend pattern where the bundler automatically includes deployment
 * when sending a UserOperation to an undeployed smart account
 */
export async function deploySmartAccountWithUserOperation(
  smartAccountAddress: Address,
  chainId: number,
  web3AuthProvider: Web3Provider
): Promise<string> {
  const networkConfig = await getNetworkConfig(chainId);

  if (!networkConfig || !networkConfig.isPimlicoSupported) {
    throw new Error(`Pimlico is not supported for chain ID: ${chainId}`);
  }

  if (!(await isPimlicoConfigured())) {
    throw new Error('Pimlico API key is not configured');
  }

  logger.log('🚀 Deploying smart account using UserOperation pattern...');
  logger.log(`📍 Smart account address: ${smartAccountAddress}`);
  logger.log(`⛓️ Chain ID: ${chainId}`);
  const gasPrices = await getOptimizedGasPrices(chainId);
  logger.log(`⛽ Using network gas config:`, gasPrices);

  try {
    // Create bundler client for the specific chain
    const bundlerClient = await createBundlerClientForChain(chainId);

    // Use network config gas prices as fallback for sponsored transactions
    const sponsoredGasPrices = {
      maxFeePerGas: gasPrices.maxFeePerGas || parseEther('0.0000001'),
      maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas || parseEther('0.000000005'),
    };
    logger.log('💰 Using network gas config:', sponsoredGasPrices);

    // Create a minimal UserOperation - a self-transfer of 0 ETH
    // This will trigger deployment if the smart account isn't deployed
    // Note: The calls array is not used directly in eth_sendUserOperation

    logger.log('📤 Sending UserOperation (will include deployment if needed)...');

    // Use Web3Auth's sendUserOperation method
    // This should work with Web3Auth's account abstraction provider
    const userOpHash = (await web3AuthProvider.request({
      method: 'eth_sendUserOperation',
      params: [
        {
          sender: smartAccountAddress,
          nonce: '0x0', // First operation
          initCode: '0x', // Let bundler handle init code
          callData: '0x', // Minimal call data for self-transfer
          callGasLimit: '0x5208', // 21000 gas for basic transfer
          verificationGasLimit: '0x1388', // 5000 gas for verification
          preVerificationGas: '0x5208', // 21000 gas for pre-verification
          maxFeePerGas: `0x${sponsoredGasPrices.maxFeePerGas.toString(16)}`,
          maxPriorityFeePerGas: `0x${sponsoredGasPrices.maxPriorityFeePerGas.toString(16)}`,
          paymasterAndData: '0x', // Let Pimlico handle paymaster
          signature: '0x', // Will be filled by Web3Auth
        },
      ],
    })) as `0x${string}`;

    logger.log(`✅ UserOperation sent: ${userOpHash}`);
    logger.log('⏳ Waiting for UserOperation receipt...');

    // Wait for the UserOperation to be mined
    const receipt = await bundlerClient.waitForUserOperationReceipt({
      hash: userOpHash,
      timeout: 120_000, // 2 minutes timeout
    });

    if (!receipt.success) {
      throw new Error(`UserOperation failed. Receipt: ${JSON.stringify(receipt)}`);
    }

    const transactionHash = receipt.receipt.transactionHash;
    logger.log(`🎉 Smart account deployment completed! Transaction: ${transactionHash}`);

    return transactionHash;
  } catch (error) {
    const err = error as Error & { code?: string };
    let errorMessage = `Error deploying smart account: ${err.message}`;

    if (err.code === 'NETWORK_ERROR') {
      errorMessage = 'Network error during deployment. Please check your connection and try again.';
    } else if (err.code === 'INSUFFICIENT_FUNDS') {
      errorMessage = 'Insufficient funds for deployment. Please contact support.';
    } else if (err.message?.includes('User rejected')) {
      errorMessage = 'Deployment cancelled by user';
    }

    logger.error_sync('❌ Smart account deployment failed:', error);
    throw new Error(errorMessage);
  }
}

/**
 * Legacy function name for backward compatibility
 * @deprecated Use deploySmartAccountWithUserOperation instead
 */
export async function deploySmartAccountWithSponsoredGas(
  _smartAccount: unknown,
  _chainId: number
): Promise<void> {
  throw new Error(
    'This function should not be called directly. Use the UserOperation pattern through the API endpoint.'
  );
}

/**
 * Estimate gas for smart account deployment
 */
export async function estimateDeploymentGas(
  _smartAccountAddress: Address,
  chainId: number
): Promise<{
  gasLimit: bigint;
  maxFeePerGas: bigint;
  maxPriorityFeePerGas: bigint;
  estimatedCost: string;
}> {
  if (!isPimlicoSupported(chainId)) {
    throw new Error(`Gas estimation not supported for chain ID: ${chainId}`);
  }

  try {
    // Get optimized gas prices
    const gasPrices = await getOptimizedGasPrices(chainId);

    // Get deployment gas limit from backend or use conservative default
    const deploymentGasLimit = await getDeploymentGasLimit(chainId);
    const gasLimit = deploymentGasLimit || BigInt(500000); // Use backend value or conservative default
    const verificationGas = BigInt(50000);
    const totalGas = gasLimit + verificationGas;
    const maxFee = gasPrices.maxFeePerGas || parseEther('0.0000001');
    const maxPriorityFee = gasPrices.maxPriorityFeePerGas || parseEther('0.000000005');
    const estimatedCost = formatEther(totalGas * maxFee);

    return {
      gasLimit,
      maxFeePerGas: maxFee,
      maxPriorityFeePerGas: maxPriorityFee,
      estimatedCost: `${estimatedCost} ETH`,
    };
  } catch (error) {
    logger.error_sync('Error estimating deployment gas:', error);
    throw new Error('Failed to estimate deployment gas');
  }
}

/**
 * Deploy smart account using Web3Auth provider with standard transaction methods
 * This approach uses regular transactions instead of UserOperations
 */
export async function deploySmartAccountWithWeb3Auth(
  smartAccountAddress: Address,
  chainId: number,
  web3AuthProvider: Web3Provider
): Promise<string> {
  const gasPrices = await getOptimizedGasPrices(chainId);

  logger.log('🚀 Deploying smart account using Web3Auth transaction...');
  logger.log(`📍 Smart account address: ${smartAccountAddress}`);
  logger.log(`⛓️ Chain ID: ${chainId}`);

  try {
    // Get the current account (should be the smart account address)
    const accounts = (await web3AuthProvider.request({
      method: 'eth_accounts',
    })) as string[];

    if (!accounts || accounts.length === 0) {
      throw new Error('No accounts available from Web3Auth');
    }

    const currentAccount = accounts[0];
    logger.log('🔑 Current account:', currentAccount);

    // Check if the account matches the smart account address
    if (currentAccount.toLowerCase() !== smartAccountAddress.toLowerCase()) {
      logger.warn('⚠️ Account mismatch, but proceeding with deployment...');
    }

    // Send a minimal transaction to trigger deployment
    // This will be a self-transfer of 0 ETH which should trigger smart account deployment
    const transactionParams = {
      from: currentAccount,
      to: smartAccountAddress,
      value: '0x0', // 0 ETH
      data: '0x', // No data
      gas: '0x5208', // 21000 gas (standard transfer)
      gasPrice: `0x${(gasPrices.maxFeePerGas || parseEther('0.0000001')).toString(16)}`,
    };

    logger.log('📤 Sending deployment transaction:', transactionParams);

    const transactionHash = (await web3AuthProvider.request({
      method: 'eth_sendTransaction',
      params: [transactionParams],
    })) as string;

    logger.log('✅ Transaction sent:', transactionHash);

    // Wait for transaction confirmation
    let receipt = null;
    let attempts = 0;
    const maxAttempts = 60; // Wait up to 60 seconds

    while (!receipt && attempts < maxAttempts) {
      try {
        receipt = (await web3AuthProvider.request({
          method: 'eth_getTransactionReceipt',
          params: [transactionHash],
        })) as { status: string } | null;

        if (receipt) {
          logger.log('📥 Transaction confirmed:', receipt);
          break;
        }
      } catch {
        // Transaction might still be pending
      }

      attempts++;
      await new Promise((resolve) => setTimeout(resolve, 1000)); // Wait 1 second
    }

    if (!receipt) {
      throw new Error('Transaction confirmation timeout');
    }

    if (receipt.status === '0x0') {
      throw new Error('Transaction failed');
    }

    logger.log('🎉 Smart account deployed successfully with Web3Auth transaction!');
    return transactionHash;
  } catch (error) {
    logger.error_sync('❌ Web3Auth deployment failed:', error);

    let errorMessage = 'Smart account deployment failed';
    const err = error as Error;

    if (err.message?.includes('User rejected')) {
      errorMessage = 'Deployment cancelled by user';
    } else if (err.message?.includes('insufficient funds')) {
      errorMessage = 'Insufficient funds for deployment transaction';
    } else if (err.message?.includes('timeout')) {
      errorMessage = 'Deployment transaction timed out';
    } else if (err.message) {
      errorMessage = `Deployment failed: ${err.message}`;
    }

    throw new Error(errorMessage);
  }
}
