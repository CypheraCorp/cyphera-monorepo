import { 
  SINGLE_DEFAULT_MODE,
  DelegationFramework,
  Implementation,
  toMetaMaskSmartAccount,
  MetaMaskSmartAccount,
  ExecutionStruct,
  Call,
} from "@metamask-private/delegator-core-viem"
import { 
  type Address, 
  isAddressEqual,
  createPublicClient,
  http,
  encodeFunctionData,
  parseUnits,
  parseEther
} from "viem"
import { privateKeyToAccount } from "viem/accounts"
import { sepolia } from "viem/chains"
import { config } from "../config/config"
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

// TODO: support multiple chains and paymasters/bundlers

// Initialize clients
const chain = sepolia

/**
 * Create public client for reading blockchain state
 */
export const publicClient = createPublicClient({
  chain,
  transport: http(config.blockchain.rpcUrl)
})

/**
 * Create bundler client for user operations
 */
export const bundlerClient = getBundlerClient()

/**
 * Creates a bundler client based on configuration settings
 */
export function getBundlerClient() {
  if (!config.blockchain.bundlerUrl) {
    throw new Error('Bundler URL is not configured')
  }

  const paymasterClient = createPaymasterClient({
    transport: http(config.blockchain.bundlerUrl)
  })
  const bundlerClient = createBundlerClient({
    transport: http(config.blockchain.bundlerUrl),
    chain,
    paymaster: paymasterClient,
  })
  return bundlerClient
}

/**
 * Creates a MetaMask smart account from a private key
 * 
 * @param privateKey - The private key to create the account from
 * @returns A MetaMask smart account instance
 */
export const createMetaMaskAccount = async (privateKey: string): Promise<MetaMaskSmartAccount<Implementation>> => {
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
 * 
 * @returns Gas fee parameters (maxFeePerGas and maxPriorityFeePerGas)
 */
export const getFeePerGas = async () => {
  // The method for determining fee per gas is dependent on the bundler
  // implementation. For this reason, this is centralized here.
  const pimlicoClient = createPimlicoClient({
    chain,
    transport: http(config.blockchain.bundlerUrl),
  })

  const { fast } = await pimlicoClient.getUserOperationGasPrice()
  return fast
}

/**
 * Redeems a delegation, executing actions on behalf of the delegator
 * 
 * @param delegationData - The serialized delegation data
 * @param merchantAddress - The address of the merchant
 * @param tokenContractAddress - The address of the token contract
 * @param price - The price of the token
 * @returns The transaction hash of the redemption
 */
export const redeemDelegation = async (
  delegationData: Uint8Array,
  merchantAddress: string,
  tokenContractAddress: string,
  price: string
): Promise<string> => {
  try {
    // Validate required parameters
    if (!delegationData || delegationData.length === 0) {
      throw new Error('Delegation data is required')
    }
    
    if (!merchantAddress || merchantAddress === '0x0000000000000000000000000000000000000000') {
      throw new Error('Valid merchant address is required')
    }
    
    if (!tokenContractAddress || tokenContractAddress === '0x0000000000000000000000000000000000000000') {
      throw new Error('Valid token contract address is required')
    }
    
    if (!price || price === '0') {
      throw new Error('Valid price is required')
    }
    
    // Parse the delegation data using our helper
    const delegation = parseDelegation(delegationData)
    
    // Validate the delegation
    validateDelegation(delegation)
    
    logger.info("Redeeming delegation...")
    logger.debug("Delegation details:", {
      delegate: delegation.delegate,
      delegator: delegation.delegator,
      merchantAddress,
      tokenContractAddress,
      price
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

    // Create ERC20 transfer calldata
    const transferCalldata = encodeFunctionData({
      abi: erc20Abi, // ABI for the ERC20 contract
      functionName: 'transfer', // Name of the function to call
      args: [merchantAddress as Address, parseUnits(price, 6)] // Assuming USDC with 6 decimals
    })
    
    // The execution that will be performed on behalf of the delegator
    // target is the address of the merchant (the recipient of the ERC20 transfer)
    // value is 0 because we are not sending any ETH with the transaction
    // callData is the calldata for the ERC20 transfer
    const executions: ExecutionStruct[] = [
      {
        target: tokenContractAddress as Address, // Address of the ERC20 contract
        value: 0n, // No ETH value for ERC20 transfers
        callData: transferCalldata // Calldata for the ERC20 transfer
      }
    ]

    // Format the delegation for the framework
    const delegationForFramework = delegation as any
    const delegationChain = [delegationForFramework]

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

    // Get fee per gas for the transaction
    const feePerGas = await getFeePerGas()

    logger.info("Sending UserOperation...")
    
    // Start timer for overall transaction operation
    const overallStartTime = Date.now()
    
    // Properly type our account for the bundler client
    // Note: This assertion is necessary because the MetaMask smart account
    // implementation doesn't exactly match what the bundler expects
    const sendOpStartTime = Date.now()
    const userOperationHash = await bundlerClient.sendUserOperation({
      account: redeemer as any,
      calls,
      ...feePerGas
    })
    const sendOpTime = (Date.now() - sendOpStartTime) / 1000

    logger.info(`UserOperation hash (sent in ${sendOpTime.toFixed(2)}s):`, userOperationHash)
    
    // Wait for the user operation to be included in a transaction
    logger.info("Waiting for transaction receipt...")
    const receiptStartTime = Date.now()
    const receipt = await bundlerClient.waitForUserOperationReceipt({
      hash: userOperationHash,
      timeout: 60_000 // 60 second timeout
    }) as UserOperationReceipt
    const receiptWaitTime = (Date.now() - receiptStartTime) / 1000

    const transactionHash = receipt.receipt.transactionHash
    
    // Calculate and log elapsed time
    const totalElapsedTimeSeconds = (Date.now() - overallStartTime) / 1000
    logger.info(`Transaction confirmed in ${totalElapsedTimeSeconds.toFixed(2)}s total (${sendOpTime.toFixed(2)}s to send, ${receiptWaitTime.toFixed(2)}s to confirm):`, transactionHash)
    
    return transactionHash
  } catch (error) {
    logger.error("Error redeeming delegation:", error)
    throw error
  }
} 