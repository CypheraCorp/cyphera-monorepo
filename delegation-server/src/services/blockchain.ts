import { 
  SINGLE_DEFAULT_MODE,
  DelegationFramework,
  Implementation,
  toMetaMaskSmartAccount
} from "@metamask-private/delegator-core-viem"
import { 
  type Address, 
  type Hex,
  parseEther,
  isAddressEqual,
  createPublicClient,
  http 
} from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { sepolia } from "viem/chains"
import { config } from "../config/config"
import { logger, formatPrivateKey } from "../utils/utils"
import { parseDelegation, validateDelegation } from "../utils/delegation-helpers"
import { 
  type DelegationStruct,
  type ExecutionStruct,
  type Call
} from "../types/delegation"

// Try to use viem/account-abstraction first, or fall back to permissionless
let createBundlerClient, createPaymasterClient;
try {
  // Attempt to import from viem/account-abstraction (viem >= 2.x)
  const viemAA = require("viem/account-abstraction");
  createBundlerClient = viemAA.createBundlerClient;
  createPaymasterClient = viemAA.createPaymasterClient;
  logger.info("Using viem/account-abstraction for bundler client");
} catch (error) {
  // Fall back to permissionless if viem/account-abstraction doesn't exist
  logger.info("viem/account-abstraction not found, falling back to permissionless");
  const permissionless = require("permissionless/clients/bundler");
  createBundlerClient = permissionless.createBundlerClient;
  createPaymasterClient = permissionless.createPaymasterClient;
}

// Use a const for the EntryPoint address to avoid hardcoding in multiple places
const ENTRY_POINT_ADDRESS = '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789' // Ethereum EntryPoint v0.6

// Initialize clients
const chain = sepolia

// Create public client for reading blockchain state
export const publicClient = createPublicClient({
  chain,
  transport: http(config.blockchain.rpcUrl)
})

// Create bundler client for user operations
export const bundlerClient = createBundlerClient({
  chain,
  transport: http(config.blockchain.bundlerUrl),
  paymaster: config.blockchain.paymasterUrl 
    ? createPaymasterClient({
        transport: http(config.blockchain.paymasterUrl)
      })
    : undefined
})

/**
 * Creates a MetaMask smart account from a private key
 */
export const createMetaMaskAccount = async (privateKey: string) => {
  try {
    const formattedKey = formatPrivateKey(privateKey)
    const account = privateKeyToAccount(formattedKey as `0x${string}`)
    
    logger.info(`Creating MetaMask Smart Account for address: ${account.address}`)
    
    const smartAccount = await toMetaMaskSmartAccount({
      client: publicClient,
      implementation: Implementation.Hybrid,
      deployParams: [account.address, [], [], []],
      deploySalt: "0x" as `0x${string}`,
      signatory: { account }
    })
    
    logger.info(`Smart Account address: ${smartAccount.address}`)
    return smartAccount
  } catch (error) {
    logger.error(`Failed to create MetaMask account:`, error)
    throw error
  }
}

/**
 * Gets the fee per gas for a user operation
 */
export const getFeePerGas = async () => {
  try {
    // Simplified fee estimation - could be improved with actual gas estimation
    return {
      maxFeePerGas: parseEther("0.00000001"),
      maxPriorityFeePerGas: parseEther("0.000000001")
    }
  } catch (error) {
    logger.error(`Failed to get fee per gas:`, error)
    throw error
  }
}

/**
 * Redeems a delegation, executing actions on behalf of the delegator
 * @param delegationData The serialized delegation data
 * @returns The transaction hash
 */
export const redeemDelegation = async (
  delegationData: Uint8Array
): Promise<string> => {
  try {
    // Parse the delegation data using our helper
    const delegation = parseDelegation(delegationData)
    
    // Validate the delegation
    validateDelegation(delegation)
    
    logger.info("Redeeming delegation...")
    logger.debug("Delegation details:", {
      delegate: delegation.delegate,
      delegator: delegation.delegator,
      expiry: delegation.expiry?.toString()
    })
    
    // Create redeemer account from private key
    if (!config.blockchain.privateKey) {
      throw new Error('Private key is not configured')
    }
    
    const redeemer = await createMetaMaskAccount(config.blockchain.privateKey)
    
    // Verify redeemer address matches delegate in delegation
    if (!isAddressEqual(redeemer.address, delegation.delegate)) {
      throw new Error(
        `Redeemer account address does not match delegate in delegation. ` +
        `Redeemer: ${redeemer.address}, delegate: ${delegation.delegate}`
      )
    }
    
    // We need to treat the delegation as the required type for DelegationFramework
    // This casting is necessary because our types might differ slightly from the framework's types
    const delegationForFramework = delegation as any
    const delegationChain = [delegationForFramework]

    // The execution that will be performed on behalf of the delegator
    // In this case, sending a minimal amount of ETH from delegator to redeemer as proof
    const executions: ExecutionStruct[] = [
      {
        target: redeemer.address,
        value: parseEther("0.000001"), // Minimal value for proof of successful execution
        callData: "0x"
      }
    ]

    // Create the calldata for redeeming the delegation
    const redeemDelegationCalldata = DelegationFramework.encode.redeemDelegations(
      [delegationChain],
      [SINGLE_DEFAULT_MODE],
      [executions]
    )

    // The call to the delegation framework to redeem the delegation
    const calls: Call[] = [
      {
        to: redeemer.address,
        data: redeemDelegationCalldata
      }
    ]

    // Get fee per gas
    const feePerGas = await getFeePerGas()

    // Encode calldata based on account interface
    let callData: `0x${string}`;
    if ('encodeCallData' in redeemer && typeof redeemer.encodeCallData === 'function') {
      callData = redeemer.encodeCallData(calls);
    } else if ('encodeCalls' in redeemer && typeof (redeemer as any).encodeCalls === 'function') {
      callData = (redeemer as any).encodeCalls(calls);
    } else {
      throw new Error('Account does not have encodeCallData or encodeCalls method');
    }
    
    logger.info("Sending UserOperation...")
    const userOperationHash = await bundlerClient.sendUserOperation({
      account: redeemer,
      userOperation: {
        callData,
        maxFeePerGas: feePerGas.maxFeePerGas,
        maxPriorityFeePerGas: feePerGas.maxPriorityFeePerGas
      },
      entryPoint: ENTRY_POINT_ADDRESS
    })

    logger.info("UserOperation hash:", userOperationHash)
    
    // Wait for the user operation to be included in a transaction
    logger.info("Waiting for transaction receipt...")
    const receipt = await bundlerClient.waitForUserOperationReceipt({
      hash: userOperationHash,
      timeout: 60_000 // 60 second timeout
    })
    
    const transactionHash = receipt.receipt.transactionHash
    logger.info("Transaction confirmed:", transactionHash)
    
    return transactionHash
  } catch (error) {
    logger.error("Error redeeming delegation:", error)
    throw error
  }
} 