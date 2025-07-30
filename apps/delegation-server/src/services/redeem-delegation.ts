import { type Address } from "viem";
import { isAddressEqual } from "viem";
import { 
  parseDelegation, 
  validateDelegation,
  // Import from shared library
  validateRedemptionInputs,
  getChainById,
  initializeBlockchainClients,
  createNetworkConfigFromUrls,
  getOrCreateRedeemerAccount,
  validateDelegateMatch,
  prepareRedemptionUserOperationPayload,
  sendAndConfirmUserOperation,
  type RedemptionParams,
  type RedeemerConfig
} from "@cyphera/delegation";
import { getNetworkConfig } from "../config/config";
import { logger } from "../utils/utils";
import { getSecretValue } from "../utils/secrets_manager";

/**
 * Redeems a delegation, executing actions on behalf of the delegator
 * This implementation uses the shared delegation library for common functionality
 * 
 * @param delegationData - The serialized delegation data
 * @param merchantAddress - The address of the merchant
 * @param tokenContractAddress - The address of the token contract
 * @param tokenAmount - The amount of tokens to redeem
 * @param tokenDecimals - The number of decimals of the token
 * @param chainId - The blockchain chain ID
 * @param networkName - The network name
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
    // 1. Validate all incoming parameters using shared validation
    const validationInputs: RedemptionParams = {
      delegationData,
      merchantAddress: merchantAddress as Address,
      tokenContractAddress: tokenContractAddress as Address,
      tokenAmount,
      tokenDecimals,
      chainId,
      networkName
    };
    validateRedemptionInputs(validationInputs);
    
    logger.info(`Starting delegation redemption for chainId: ${chainId}, network: ${networkName}`);

    // 2. Get network configuration and create blockchain clients
    const { rpcUrl, bundlerUrl } = await getNetworkConfig(networkName, chainId);
    const chain = getChainById(chainId);
    const networkConfig = createNetworkConfigFromUrls(networkName, chainId, rpcUrl, bundlerUrl);
    const { publicClient, bundlerClient, pimlicoClient } = await initializeBlockchainClients(networkConfig, chain);
    
    // 3. Parse and validate the delegation
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

    // 4. Get the redeemer configuration and create smart account
    const privateKey = await getSecretValue('PRIVATE_KEY_ARN', "PRIVATE_KEY");
    const redeemerConfig: RedeemerConfig = {
      privateKey,
      deploySalt: "0x" as `0x${string}`
    };
    const redeemer = await getOrCreateRedeemerAccount(publicClient, redeemerConfig);
    logger.info(`Target Smart Account address: ${redeemer.address}`);

    // 5. Validate delegate matches redeemer
    validateDelegateMatch(redeemer.address, delegation.delegate);

    // 6. Prepare the redemption payload
    const callsForRedemption = prepareRedemptionUserOperationPayload(
      delegation,
      merchantAddress,
      tokenContractAddress,
      tokenAmount,
      tokenDecimals,
      redeemer.address
    );
    
    // 7. Send and confirm the UserOperation
    const transactionHash = await sendAndConfirmUserOperation(
      bundlerClient,
      pimlicoClient,
      redeemer,
      callsForRedemption,
      publicClient,
      {
        onStatusUpdate: (status) => logger.info(status)
      }
    );

    return transactionHash;

  } catch (error) {
    // Centralized error handling
    logger.error("Critical error in redeemDelegation service:", { 
      message: (error as Error)?.message, 
      stack: (error as Error)?.stack,
      errorObject: error 
    });
    
    if (error instanceof Error) {
      throw new Error(`RedeemDelegation failed: ${error.message}`);
    }
    throw new Error(`RedeemDelegation failed due to an unknown error.`);
  }
};