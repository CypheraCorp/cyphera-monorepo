import { 
  SINGLE_DEFAULT_MODE,
  DelegationFramework,
  Implementation,
  toMetaMaskSmartAccount,
  ExecutionStruct,
  Call,
  type MetaMaskSmartAccount,
} from "@metamask/delegation-toolkit"
import { 
  type Address, 
  isAddressEqual,
  createPublicClient,
  encodeFunctionData,
  parseUnits,
  type Transport,
  type Chain,
  custom,
  type PublicClient,
  parseEther,
  formatEther
} from "viem"
import { privateKeyToAccount, type LocalAccount } from "viem/accounts"
import { config, getNetworkConfig } from "../config/config"
import { logger, formatPrivateKey } from "../utils/utils"
import { parseDelegation, validateDelegation } from "../utils/delegation-helpers"
import { erc20Abi } from "../abis/erc20"

// Import account abstraction types and bundler clients
import { 
  createBundlerClient, 
  createPaymasterClient, 
  type UserOperationReceipt,
  UserOperationExecutionError
} from "viem/account-abstraction"
import { createPimlicoClient } from "permissionless/clients/pimlico"
import fetch from 'node-fetch'
import * as allChains from "viem/chains"
import { getSecretValue } from "../utils/secrets_manager"

/**
 * Finds a viem Chain object by its chain ID.
 */
function getChainById(chainId: number): Chain {
  for (const chainKey in allChains) {
    const chain = allChains[chainKey as keyof typeof allChains];
    if (typeof chain === 'object' && chain !== null && 'id' in chain && chain.id === chainId) {
      return chain as Chain;
    }
  }
  throw new Error(`Unsupported chainId: ${chainId}. Chain not found in viem/chains.`);
}

/**
 * Create a custom viem transport using node-fetch.
 * Renamed from createTransport to createFetchTransport for clarity.
 */
const createTransport = (url: string | undefined): Transport => {
  if (!url) {
    throw new Error('URL is required for transport');
  }
  return custom({
    async request({ method, params }) {
      const response = await fetch(url, {
        method: 'POST', headers: { 'Content-Type': 'application/json', },
        body: JSON.stringify({ jsonrpc: '2.0', method, params, id: 1, }),
      });
      if (!response.ok) {
        const errorBody = await response.text();
        logger.error(`HTTP error! status: ${response.status}, url: ${url}, body: ${errorBody}`);
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      if (data.error) {
        logger.error('RPC Error:', data.error);
        throw new Error(`RPC error: ${data.error.message} (code: ${data.error.code})`);
      }
      return data.result;
    }
  });
};

/**
 * Redeems a delegation, executing actions on behalf of the delegator
 * 
 * @param delegationData - The serialized delegation data
 * @param merchantAddress - The address of the merchant
 * @param tokenContractAddress - The address of the token contract
 * @param tokenAmount - The amount of tokens to redeem
 * @param tokenDecimals - The number of decimals of the token
 * @returns The transaction hash of the redemption
 */
export const redeemDelegation = async (
  delegationData: Uint8Array,
  merchantAddress: string,
  tokenContractAddress: string,
  tokenAmount: number,
  tokenDecimals: number,
  chainId: number,
  networkName: string
): Promise<string> => {
  try {
    // --- 1. Validation & Config ---
    if (!delegationData || delegationData.length === 0) throw new Error('Delegation data is required');
    if (!merchantAddress || merchantAddress === '0x0000000000000000000000000000000000000000') throw new Error('Valid merchant address is required');
    if (!tokenContractAddress || tokenContractAddress === '0x0000000000000000000000000000000000000000') throw new Error('Valid token contract address is required');
    if (!tokenAmount || tokenAmount === 0) throw new Error('Valid token amount is required');
    if (!tokenDecimals || tokenDecimals <= 0) throw new Error('Valid token decimals are required');
    if (!chainId || chainId <= 0) throw new Error('Valid chainId is required');
    if (!networkName) throw new Error('Valid networkName is required');

    logger.info(`Starting delegation redemption for chainId: ${chainId}, network: ${networkName}`);

    // Get dynamic network configuration
    const { rpcUrl, bundlerUrl } = await getNetworkConfig(networkName, chainId);
    const chain: Chain = getChainById(chainId);

    // --- 2. Initialize Clients Dynamically ---
    const publicClient: PublicClient = createPublicClient({
      chain,
      transport: createTransport(rpcUrl)
    });

    // Initialize Paymaster client (needed for bundler client setup, even if not sponsoring)
    const paymasterClient = createPaymasterClient({
        transport: createTransport(bundlerUrl)
    });

    // Initialize Bundler client
    const bundlerClient = createBundlerClient({
        transport: createTransport(bundlerUrl),
        chain,
        // entryPoint // Add if needed
        paymaster: paymasterClient, // Add if sponsoring
    });

    // Initialize Pimlico client (for gas price fetching)
    const pimlicoClient = createPimlicoClient({
        chain,
        transport: createTransport(bundlerUrl),
    });

    // --- Remaining logic (to be refactored step-by-step) ---

    // Parse the delegation data using our helper
    const delegation = parseDelegation(delegationData);
    
    // Validate the delegation
    validateDelegation(delegation);
    
    logger.info("Redeeming delegation...");
    logger.debug("Delegation details:", {
      delegate: delegation.delegate,
      delegator: delegation.delegator,
      merchantAddress,
      tokenContractAddress,
      tokenAmount,
      tokenDecimals,
      chainId,
      networkName
    });

    // Get private key from AWS Secrets Manager
    const privateKey = await getSecretValue('PRIVATE_KEY_ARN', "PRIVATE_KEY");
    
    // Create redeemer account from private key (Inline logic from old createMetaMaskAccount)
    const formattedKey = formatPrivateKey(privateKey);

    const account = privateKeyToAccount(formattedKey as `0x${string}`);
    logger.info(`Creating MetaMask Smart Account for address: ${account.address} on chain ${chainId}`);
    // Use the local publicClient instance here
    const redeemer: MetaMaskSmartAccount<Implementation.Hybrid> = await toMetaMaskSmartAccount({
      client: publicClient,
      implementation: Implementation.Hybrid,
      deployParams: [account.address, [], [], []],
      deploySalt: "0x" as `0x${string}`,
      signatory: { account }
    });
    logger.info(`Target Smart Account address: ${redeemer.address}`);

    if (!isAddressEqual(redeemer.address, delegation.delegate)) {
      throw new Error(
        `Redeemer SA address (${redeemer.address}) does not match delegate (${delegation.delegate}) in delegation on chain ${chainId}.`
      );
    }

    // --- 5. Ensure Smart Account is Deployed ---
    let isDeployed = await redeemer.isDeployed();
    logger.info(`Is Smart Account ${redeemer.address} deployed? ${isDeployed}`);

    if (!isDeployed) {
      logger.info(`Smart Account ${redeemer.address} is not deployed. Sending UserOperation to trigger deployment...`);
      const { fast: deploymentGasPrices } = await pimlicoClient.getUserOperationGasPrice();

      if (!deploymentGasPrices || !deploymentGasPrices.maxFeePerGas || !deploymentGasPrices.maxPriorityFeePerGas) {
        throw new Error("Could not fetch gas prices from Pimlico for SA deployment.");
      }
      logger.info(`Using gas prices for SA deployment: maxFeePerGas: ${formatEther(deploymentGasPrices.maxFeePerGas, "gwei")} gwei, maxPriorityFeePerGas: ${formatEther(deploymentGasPrices.maxPriorityFeePerGas, "gwei")} gwei`);
      
      const DUMMY_DEPLOY_RECEIVER_ADDRESS = redeemer.address; 
      const DUMMY_DEPLOY_VALUE = parseEther("0"); 
      let deployUserOpHash: Address | undefined;

      try {
        deployUserOpHash = await bundlerClient.sendUserOperation({
          account: redeemer,
          calls: [{ to: DUMMY_DEPLOY_RECEIVER_ADDRESS, value: DUMMY_DEPLOY_VALUE }],
          maxFeePerGas: deploymentGasPrices.maxFeePerGas,
          maxPriorityFeePerGas: deploymentGasPrices.maxPriorityFeePerGas,
          verificationGasLimit: 150000n,
        });
        logger.info(`Deployment UserOperation sent. Hash: ${deployUserOpHash}`);
        logger.info("Waiting for deployment UserOperation receipt...");
        const deployReceipt = await bundlerClient.waitForUserOperationReceipt({ hash: deployUserOpHash, timeout: 120_000 });
        
        if (deployReceipt.success) {
          logger.info(`Deployment UserOperation successful! Transaction Hash: ${deployReceipt.receipt.transactionHash}`);
          isDeployed = await redeemer.isDeployed();
          if (!isDeployed) {
              const code = await publicClient.getBytecode({address: redeemer.address});
              if (!code || code === "0x") {
                   throw new Error(`SA deployment failed: No bytecode found at ${redeemer.address} after deployment UserOp ${deployReceipt.receipt.transactionHash}.`);
              }
              logger.info(`Bytecode found at ${redeemer.address}. Assuming deployed.`);
              isDeployed = true; 
          }
        } else {
          // This case should ideally be caught by the catch block if waitForUserOperationReceipt throws on failure
          throw new Error(`SA Deployment UserOperation did not succeed. Receipt: ${JSON.stringify(deployReceipt)}`);
        }
      } catch (e: any) {
        let errMsg = `Error during SA deployment UserOperation (hash: ${deployUserOpHash || 'N/A'}): ${e.message}`;
        if (e instanceof UserOperationExecutionError) {
            errMsg += ` Reason: ${e.cause?.details || e.cause?.message || 'N/A'}`;
        }
        logger.error(errMsg, e);
        throw new Error(errMsg);
      }
    }
    
    if (!isDeployed) {
        throw new Error(`Smart Account ${redeemer.address} could not be confirmed as deployed.`);
    }

    // --- 6. Prepare and Send RedeemDelegations UserOperation ---
    logger.info(`Smart Account ${redeemer.address} is deployed. Proceeding with redeemDelegations.`);
    // Convert tokenAmount (e.g., 499992) and tokenDecimals (e.g., 6)
    // to a human-readable string representation (e.g., "0.499992")
    // before parsing to BigInt for the contract call.
    if (tokenDecimals <= 0) {
      // Or handle as an error, depending on expected constraints
      throw new Error('Token decimals must be positive for this parsing logic.');
    }
    const humanReadableAmountString = (tokenAmount / Math.pow(10, tokenDecimals)).toFixed(tokenDecimals);

    const tokenAmountBigInt = parseUnits(humanReadableAmountString, tokenDecimals);

    const transferCalldata = encodeFunctionData({
      abi: erc20Abi,
      functionName: 'transfer',
      args: [merchantAddress as Address, tokenAmountBigInt]
    });
    
    const executions: ExecutionStruct[] = [{
      target: tokenContractAddress as Address,
      value: 0n,
      callData: transferCalldata
    }];

    const delegationForFramework = delegation as any;
    const delegationChain = [delegationForFramework];

    const redeemDelegationCalldata = DelegationFramework.encode.redeemDelegations({
      delegations: [delegationChain],
      modes: [SINGLE_DEFAULT_MODE],
      executions: [executions]
    });

    const callsForRedemption: Call[] = [{
      to: redeemer.address, // The SA calls itself to invoke DelegationFramework
      data: redeemDelegationCalldata,
      // value: 0n // No native value needed for the framework call itself
    }];
    
    const { fast: redemptionGasPrices } = await pimlicoClient.getUserOperationGasPrice();
    if (!redemptionGasPrices || !redemptionGasPrices.maxFeePerGas || !redemptionGasPrices.maxPriorityFeePerGas) {
      throw new Error("Could not fetch gas prices for redemption UserOp.");
    }
    logger.info(`Using gas prices for redemption: maxFeePerGas: ${formatEther(redemptionGasPrices.maxFeePerGas, "gwei")} gwei, maxPriorityFeePerGas: ${formatEther(redemptionGasPrices.maxPriorityFeePerGas, "gwei")} gwei`);

    logger.info("Sending redeemDelegations UserOperation...");
    const overallStartTime = Date.now();
    let redeemUserOpHash: Address | undefined;

    try {
      redeemUserOpHash = await bundlerClient.sendUserOperation({
        account: redeemer, 
        calls: callsForRedemption,
        maxFeePerGas: redemptionGasPrices.maxFeePerGas,
        maxPriorityFeePerGas: redemptionGasPrices.maxPriorityFeePerGas,
        // Optional: Increase gas limits if needed
        // callGasLimit: 10000000n,
        // preVerificationGas: 5000000n,
        // verificationGasLimit: 15000000n,
      });
      logger.info(`Redemption UserOperation hash (sent in ${(Date.now() - overallStartTime) / 1000}s): ${redeemUserOpHash}`);
      logger.info("Waiting for redemption transaction receipt...");
      const redeemReceipt = await bundlerClient.waitForUserOperationReceipt({ hash: redeemUserOpHash, timeout: 120_000 }) as UserOperationReceipt;

      if (!redeemReceipt.success) {
        // This case should ideally be caught by the catch block
        throw new Error(`Redemption UserOperation did not succeed. Receipt: ${JSON.stringify(redeemReceipt)}`);
      }
      const transactionHash = redeemReceipt.receipt.transactionHash;
      logger.info(`Redemption Transaction confirmed in ${(Date.now() - overallStartTime) / 1000}s total: ${transactionHash}`);
      return transactionHash;
    } catch (e: any) {
        let errMsg = `Error during redeemDelegations UserOperation (hash: ${redeemUserOpHash || 'N/A'}): ${e.message}`;
        if (e instanceof UserOperationExecutionError) {
             errMsg += ` Reason: ${e.cause?.details || e.cause?.message || 'N/A'}`;
        }
        logger.error(errMsg, e);
        throw new Error(errMsg);
    }

  } catch (error) {
    logger.error("Error in redeemDelegation service:", { message: (error as Error)?.message, stack: (error as Error)?.stack, error });
    if (error instanceof Error) {
        throw new Error(`RedeemDelegation failed: ${error.message}`);
    }
    throw new Error(`RedeemDelegation failed with unknown error.`);
  }
}; 