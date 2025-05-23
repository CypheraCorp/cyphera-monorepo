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

// --- Helper Functions ---

interface RedemptionValidationInputs {
  delegationData: Uint8Array;
  merchantAddress: string;
  tokenContractAddress: string;
  tokenAmount: number;
  tokenDecimals: number;
  chainId: number;
  networkName: string;
}

/**
 * Validates the inputs for the redeemDelegation function.
 */
function _validateRedemptionInputs(inputs: RedemptionValidationInputs): void {
  if (!inputs.delegationData || inputs.delegationData.length === 0) throw new Error('Delegation data is required');
  if (!inputs.merchantAddress || inputs.merchantAddress === '0x0000000000000000000000000000000000000000') throw new Error('Valid merchant address is required');
  if (!inputs.tokenContractAddress || inputs.tokenContractAddress === '0x0000000000000000000000000000000000000000') throw new Error('Valid token contract address is required');
  if (!inputs.tokenAmount || inputs.tokenAmount === 0) throw new Error('Valid token amount is required');
  if (!inputs.tokenDecimals || inputs.tokenDecimals <= 0) throw new Error('Valid token decimals are required');
  if (!inputs.chainId || inputs.chainId <= 0) throw new Error('Valid chainId is required');
  if (!inputs.networkName) throw new Error('Valid networkName is required');
}

interface BlockchainClients {
  publicClient: PublicClient;
  bundlerClient: ReturnType<typeof createBundlerClient>;
  pimlicoClient: ReturnType<typeof createPimlicoClient>;
  // paymasterClient is created but not always directly returned if not used explicitly later by main flow
}

/**
 * Initializes the necessary blockchain clients.
 */
async function _initializeBlockchainClients(networkName: string, chainId: number, chain: Chain): Promise<BlockchainClients> {
  const { rpcUrl, bundlerUrl } = await getNetworkConfig(networkName, chainId);

  const publicClient = createPublicClient({
    chain,
    transport: createTransport(rpcUrl)
  });

  const paymasterClient = createPaymasterClient({ // Needed for bundler client, even if not sponsoring
      transport: createTransport(bundlerUrl)
  });

  const bundlerClient = createBundlerClient({
      transport: createTransport(bundlerUrl),
      chain,
      paymaster: paymasterClient,
  });

  const pimlicoClient = createPimlicoClient({
      chain,
      transport: createTransport(bundlerUrl),
  });

  return { publicClient, bundlerClient, pimlicoClient };
}

/**
 * Creates or gets the redeemer's MetaMask Smart Account.
 */
async function _getOrCreateRedeemerAccount(
  publicClient: PublicClient,
  chainId: number
): Promise<MetaMaskSmartAccount<Implementation.Hybrid>> {
  const privateKey = await getSecretValue('PRIVATE_KEY_ARN', "PRIVATE_KEY");
  const formattedKey = formatPrivateKey(privateKey);
  const account = privateKeyToAccount(formattedKey as `0x${string}`);

  logger.info(`Creating MetaMask Smart Account for EOA: ${account.address} on chain ${chainId}`);
  const redeemer = await toMetaMaskSmartAccount({
    client: publicClient,
    implementation: Implementation.Hybrid,
    deployParams: [account.address, [], [], []],
    deploySalt: "0x" as `0x${string}`,
    signatory: { account }
  });
  logger.info(`Target Smart Account address: ${redeemer.address}`);
  return redeemer;
}

/**
 * Prepares the payload (calls array) for the redeemDelegations UserOperation.
 */
function _prepareRedemptionUserOperationPayload(
  delegation: any, // Type it more strictly if possible, from parseDelegation
  merchantAddress: string,
  tokenContractAddress: string,
  tokenAmount: number,
  tokenDecimals: number,
  redeemerAddress: Address
): Call[] {
  if (tokenDecimals <= 0) {
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

  const delegationForFramework = delegation; // Potentially add 'as const' or a more specific type
  const delegationChain = [delegationForFramework];

  const redeemDelegationCalldata = DelegationFramework.encode.redeemDelegations({
    delegations: [delegationChain],
    modes: [SINGLE_DEFAULT_MODE],
    executions: [executions]
  });

  return [{
    to: redeemerAddress,
    data: redeemDelegationCalldata,
  }];
}

/**
 * Sends the UserOperation, waits for its receipt, and handles confirmation/errors.
 */
async function _sendAndConfirmUserOperation(
  bundlerClient: ReturnType<typeof createBundlerClient>,
  pimlicoClient: ReturnType<typeof createPimlicoClient>,
  redeemer: MetaMaskSmartAccount<Implementation.Hybrid>,
  calls: Call[],
  publicClient: PublicClient // Added for post-confirmation check
): Promise<string> {
  const { fast: gasPrices } = await pimlicoClient.getUserOperationGasPrice();
  if (!gasPrices || !gasPrices.maxFeePerGas || !gasPrices.maxPriorityFeePerGas) {
    throw new Error("Could not fetch gas prices for the UserOperation.");
  }
  logger.info(`Using gas prices: maxFeePerGas: ${formatEther(gasPrices.maxFeePerGas, "gwei")} gwei, maxPriorityFeePerGas: ${formatEther(gasPrices.maxPriorityFeePerGas, "gwei")} gwei`);

  logger.info("Sending UserOperation (will include deployment if SA is not on-chain)...");
  const overallStartTime = Date.now();
  let userOpHash: Address | undefined;

  try {
    userOpHash = await bundlerClient.sendUserOperation({
      account: redeemer, 
      calls: calls,
      maxFeePerGas: gasPrices.maxFeePerGas,
      maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas,
    });
    logger.info(`UserOperation hash (sent in ${(Date.now() - overallStartTime) / 1000}s): ${userOpHash}`);
    logger.info("Waiting for UserOperation receipt (includes potential deployment and redemption)...");
    const receipt = await bundlerClient.waitForUserOperationReceipt({ hash: userOpHash, timeout: 120_000 }) as UserOperationReceipt;

    if (!receipt.success) {
      throw new Error(`UserOperation did not succeed. Receipt: ${JSON.stringify(receipt)}`);
    }

    const isNowDeployed = await redeemer.isDeployed();
    if (isNowDeployed) {
      logger.info(`Smart Account ${redeemer.address} is confirmed deployed.`);
    } else {
      const code = await publicClient.getBytecode({address: redeemer.address});
      if (!code || code === "0x") {
           logger.warn(`Warning: SA ${redeemer.address} bytecode not found after UserOp ${receipt.receipt.transactionHash}, but UserOp reported success.`);
      } else {
          logger.info(`Bytecode found at ${redeemer.address}. Assuming deployed despite isDeployed() returning false.`);
      }
    }

    const transactionHash = receipt.receipt.transactionHash;
    logger.info(`Redemption Transaction confirmed in ${(Date.now() - overallStartTime) / 1000}s total: ${transactionHash}`);
    return transactionHash;
  } catch (e: any) {
      let errMsg = `Error during UserOperation (hash: ${userOpHash || 'N/A'}): ${e.message}`;
      if (e instanceof UserOperationExecutionError) {
           errMsg += ` Reason: ${e.cause?.details || e.cause?.message || 'N/A'}`;
      }
      logger.error(errMsg, e);
      try {
          const deployedStatusOnError = await redeemer.isDeployed();
          logger.error(`SA ${redeemer.address} deployment status on error: ${deployedStatusOnError}`);
      } catch (deployCheckError) {
          logger.error(`Could not check SA deployment status on error: ${deployCheckError}`);
      }
      throw new Error(errMsg);
  }
}

// --- Main Exported Function ---

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
    // 1. Validate all incoming parameters to ensure data integrity early.
    _validateRedemptionInputs({ delegationData, merchantAddress, tokenContractAddress, tokenAmount, tokenDecimals, chainId, networkName });
    logger.info(`Starting delegation redemption for chainId: ${chainId}, network: ${networkName}`);

    // 2. Determine the correct blockchain and initialize necessary clients.
    const chain: Chain = getChainById(chainId);
    const { publicClient, bundlerClient, pimlicoClient } = await _initializeBlockchainClients(networkName, chainId, chain);
    
    // 3. Parse and validate the provided delegation data.
    // validateDelegation will ensure the delegator (customer's) smart account is deployed.
    const delegation = parseDelegation(delegationData);
    await validateDelegation(delegation, publicClient); 
    
    logger.info("Redeeming delegation...");
    logger.debug("Delegation details for redemption:", {
      delegate: delegation.delegate,
      delegator: delegation.delegator,
      merchantAddress,
      tokenContractAddress,
      tokenAmount,
      tokenDecimals,
      chainId,
      networkName
    });

    // 4. Get or create the smart account that will execute the redemption (the 'redeemer').
    // This account is derived from a private key stored securely.
    const redeemer = await _getOrCreateRedeemerAccount(publicClient, chainId);

    // 5. Crucial check: Ensure the on-chain delegate matches the derived redeemer smart account address.
    if (!isAddressEqual(redeemer.address, delegation.delegate)) {
      throw new Error(
        `Redeemer SA address (${redeemer.address}) does not match delegate (${delegation.delegate}) in delegation on chain ${chainId}. Mismatched delegate.`
      );
    }

    // 6. Prepare the UserOperation payload.
    // This includes encoding the ERC20 transfer and the delegation redemption calls.
    const callsForRedemption = _prepareRedemptionUserOperationPayload(
      delegation,
      merchantAddress,
      tokenContractAddress,
      tokenAmount,
      tokenDecimals,
      redeemer.address
    );
    
    // 7. Send the UserOperation to the bundler and wait for confirmation.
    // This step will also handle the smart account deployment if it's not yet on-chain.
    return await _sendAndConfirmUserOperation(
      bundlerClient,
      pimlicoClient,
      redeemer,
      callsForRedemption,
      publicClient
    );

  } catch (error) {
    // Centralized error handling for the entire redemption process.
    logger.error("Critical error in redeemDelegation service:", { 
        message: (error as Error)?.message, 
        stack: (error as Error)?.stack, 
        // Include additional context if available, e.g., parts of the input if safe
        errorObject: error 
    });
    if (error instanceof Error) {
        // Re-throw with a more specific prefix for easier upstream identification.
        throw new Error(`RedeemDelegation failed: ${error.message}`);
    }
    throw new Error(`RedeemDelegation failed due to an unknown error.`);
  }
}; 