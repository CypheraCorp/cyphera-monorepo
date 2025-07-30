import { 
  type PublicClient,
  type Address,
  type Hex,
  isAddressEqual
} from 'viem';
import { privateKeyToAccount, type Account } from 'viem/accounts';
import { 
  toMetaMaskSmartAccount,
  Implementation,
  type MetaMaskSmartAccount
} from '@metamask/delegation-toolkit';
import { RedeemerConfig, RedemptionError, RedemptionErrorType } from './types';

/**
 * Creates or retrieves a MetaMask Smart Account for the redeemer
 * @param publicClient The public client for blockchain interaction
 * @param config The redeemer configuration
 * @returns The MetaMask smart account instance
 */
export async function getOrCreateRedeemerAccount(
  publicClient: PublicClient,
  config: RedeemerConfig
): Promise<MetaMaskSmartAccount<Implementation>> {
  try {
    // Format and validate the private key
    const formattedKey = formatPrivateKey(config.privateKey);
    const account = privateKeyToAccount(formattedKey);

    // Determine implementation type (default to Hybrid)
    const implementation = config.implementation || Implementation.Hybrid;
    
    // Create the smart account
    const smartAccount = await toMetaMaskSmartAccount({
      client: publicClient,
      implementation,
      deployParams: [account.address, [], [], []],
      deploySalt: config.deploySalt || '0x' as Hex,
      signatory: { account }
    });

    return smartAccount;
  } catch (error) {
    throw new RedemptionError(
      'Failed to create redeemer smart account',
      RedemptionErrorType.SMART_ACCOUNT_ERROR,
      error
    );
  }
}

/**
 * Formats a private key to ensure it has the correct 0x prefix
 * @param privateKey The private key to format
 * @returns The formatted private key
 */
export function formatPrivateKey(privateKey: string): Hex {
  if (!privateKey) {
    throw new RedemptionError(
      'Private key is required',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  // Remove any whitespace
  const trimmedKey = privateKey.trim();

  // Check if it's already properly formatted
  if (trimmedKey.startsWith('0x') && trimmedKey.length === 66) {
    return trimmedKey as Hex;
  }

  // Add 0x prefix if missing
  if (!trimmedKey.startsWith('0x') && trimmedKey.length === 64) {
    return `0x${trimmedKey}` as Hex;
  }

  throw new RedemptionError(
    'Invalid private key format',
    RedemptionErrorType.VALIDATION_ERROR
  );
}

/**
 * Validates that the redeemer address matches the delegate address
 * @param redeemerAddress The redeemer smart account address
 * @param delegateAddress The delegate address from the delegation
 * @throws RedemptionError if addresses don't match
 */
export function validateDelegateMatch(
  redeemerAddress: Address,
  delegateAddress: Address
): void {
  if (!isAddressEqual(redeemerAddress, delegateAddress)) {
    throw new RedemptionError(
      `Redeemer SA address (${redeemerAddress}) does not match delegate (${delegateAddress}) in delegation`,
      RedemptionErrorType.DELEGATION_ERROR,
      { redeemerAddress, delegateAddress }
    );
  }
}

/**
 * Checks if a smart account is deployed on-chain
 * @param publicClient The public client
 * @param smartAccountAddress The smart account address to check
 * @returns Whether the account is deployed
 */
export async function checkSmartAccountDeployment(
  publicClient: PublicClient,
  smartAccountAddress: Address
): Promise<boolean> {
  try {
    const code = await publicClient.getBytecode({ address: smartAccountAddress });
    return code !== undefined && code !== '0x';
  } catch (error) {
    throw new RedemptionError(
      'Failed to check smart account deployment',
      RedemptionErrorType.SMART_ACCOUNT_ERROR,
      error
    );
  }
}

/**
 * Configuration for deterministic smart account deployment
 */
export interface DeterministicDeploymentConfig {
  /** The EOA address that will control the smart account */
  eoaAddress: Address;
  /** Optional salt for deterministic deployment */
  salt?: Hex;
  /** Implementation type */
  implementation?: Implementation;
}

/**
 * Calculates the deterministic address for a smart account
 * This can be used to predict the address before deployment
 * @param config The deployment configuration
 * @returns The predicted smart account address
 */
export async function calculateSmartAccountAddress(
  config: DeterministicDeploymentConfig
): Promise<Address> {
  // This would require the MetaMask delegation toolkit to expose
  // address calculation functionality. For now, we'll throw an error
  // indicating this is a future enhancement
  throw new RedemptionError(
    'Deterministic address calculation not yet implemented',
    RedemptionErrorType.SMART_ACCOUNT_ERROR,
    { feature: 'calculateSmartAccountAddress' }
  );
}

/**
 * Gets deployment information for a smart account
 */
export interface SmartAccountInfo {
  address: Address;
  isDeployed: boolean;
  implementation: Implementation;
  eoaController: Address;
}

/**
 * Gets comprehensive information about a smart account
 * @param smartAccount The smart account instance
 * @param publicClient The public client
 * @param implementation The implementation type used
 * @param eoaAddress The EOA controller address
 * @returns SmartAccountInfo object
 */
export async function getSmartAccountInfo(
  smartAccount: MetaMaskSmartAccount<Implementation>,
  publicClient: PublicClient,
  implementation: Implementation,
  eoaAddress: Address
): Promise<SmartAccountInfo> {
  const isDeployed = await checkSmartAccountDeployment(publicClient, smartAccount.address);
  
  return {
    address: smartAccount.address,
    isDeployed,
    implementation,
    eoaController: eoaAddress
  };
}