import { 
  SINGLE_DEFAULT_MODE,
  DelegationFramework,
  Implementation,
  toMetaMaskSmartAccount,
  ExecutionStruct,
  Call,
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
  type UserOperationReceipt
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
    const redeemer = await toMetaMaskSmartAccount({
      client: publicClient, 
      implementation: Implementation.Hybrid,
      deployParams: [account.address, [], [], []],
      deploySalt: "0x" as `0x${string}`,
      signatory: { account }
    });
    logger.info(`Smart Account address: ${redeemer.address}`);
        
    // Verify redeemer address matches delegate in delegation
    if (!isAddressEqual(redeemer.address, delegation.delegate)) {
      throw new Error(
        `Redeemer account address (${redeemer.address}) does not match delegate (${delegation.delegate}) in delegation on chain ${chainId}.`
      );
    }

    // Convert tokenAmount (e.g., 499992) and tokenDecimals (e.g., 6)
    // to a human-readable string representation (e.g., "0.499992")
    // before parsing to BigInt for the contract call.
    if (tokenDecimals <= 0) {
      // Or handle as an error, depending on expected constraints
      throw new Error('Token decimals must be positive for this parsing logic.');
    }
    const humanReadableAmountString = (tokenAmount / Math.pow(10, tokenDecimals)).toFixed(tokenDecimals);

    const tokenAmountBigInt = parseUnits(humanReadableAmountString, tokenDecimals);

    // Create ERC20 transfer calldata
    const transferCalldata = encodeFunctionData({
      abi: erc20Abi,
      functionName: 'transfer',
      args: [merchantAddress as Address, tokenAmountBigInt]
    });
    
    const executions: ExecutionStruct[] = [
      {
        target: tokenContractAddress as Address,
        value: 0n,
        callData: transferCalldata
      }
    ];

    const delegationForFramework = delegation as any;
    const delegationChain = [delegationForFramework];

    const redeemDelegationCalldata = DelegationFramework.encode.redeemDelegations({
      delegations: [delegationChain],
      modes: [SINGLE_DEFAULT_MODE],
      executions: [executions]
    });

    // The call to the delegation framework to redeem the delegation
    const calls: Call[] = [
      {
        to: redeemer.address,
        data: redeemDelegationCalldata
      }
    ]

    // Get fee per gas for the transaction (Inline logic from old getFeePerGas)
    // Use the local pimlicoClient instance here
    const feePerGas = (await pimlicoClient.getUserOperationGasPrice()).fast;

    console.log("feePerGas", feePerGas);

    logger.info("Sending UserOperation...");
    const overallStartTime = Date.now();
    
    const sendOpStartTime = Date.now();
    // Use the local bundlerClient instance here
    const userOperationHash = await bundlerClient.sendUserOperation({
      account: redeemer as any, // Keep existing logic/type assertion for now
      calls,
      ...feePerGas
    });
    const sendOpTime = (Date.now() - sendOpStartTime) / 1000;

    logger.info(`UserOperation hash (sent in ${sendOpTime.toFixed(2)}s):`, userOperationHash);
    
    logger.info("Waiting for transaction receipt...");
    const receiptStartTime = Date.now();
    // Use the local bundlerClient instance here
    const receipt = await bundlerClient.waitForUserOperationReceipt({
      hash: userOperationHash,
      timeout: 60_000 // 60 second timeout
    }) as UserOperationReceipt;
    const receiptWaitTime = (Date.now() - receiptStartTime) / 1000;

    const transactionHash = receipt.receipt.transactionHash;
    
    const totalElapsedTimeSeconds = (Date.now() - overallStartTime) / 1000;
    logger.info(`Transaction confirmed in ${totalElapsedTimeSeconds.toFixed(2)}s total (${sendOpTime.toFixed(2)}s to send, ${receiptWaitTime.toFixed(2)}s to confirm):`, transactionHash);
    
    return transactionHash;

  } catch (error) {
    logger.error("Error redeeming delegation:", error);
    throw error;
  }
}; 