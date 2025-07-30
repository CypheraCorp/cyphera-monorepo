import { type Address, type Hex, type PublicClient, type Chain, type Transport } from 'viem';
import { type Implementation } from '@metamask/delegation-toolkit';

/**
 * Parameters required for redeeming a delegation
 */
export interface RedemptionParams {
  /** The serialized delegation data */
  delegationData: Uint8Array;
  /** The address of the merchant to receive tokens */
  merchantAddress: Address;
  /** The address of the token contract */
  tokenContractAddress: Address;
  /** The amount of tokens to redeem (in smallest unit) */
  tokenAmount: number | bigint;
  /** The number of decimals for the token */
  tokenDecimals: number;
  /** The blockchain chain ID */
  chainId: number;
  /** The network name (e.g., "Base Sepolia") */
  networkName: string;
}

/**
 * Result of a successful redemption
 */
export interface RedemptionResult {
  /** The transaction hash of the redemption */
  transactionHash: Hex;
  /** Whether the smart account was deployed as part of this transaction */
  deployedSmartAccount?: boolean;
  /** Gas costs information */
  gasCosts?: {
    maxFeePerGas: bigint;
    maxPriorityFeePerGas: bigint;
  };
  /** Timestamp of the redemption */
  timestamp: number;
}

/**
 * Inputs for redemption validation
 */
export interface RedemptionValidationInputs {
  delegationData: Uint8Array;
  merchantAddress: string;
  tokenContractAddress: string;
  tokenAmount: number | bigint;
  tokenDecimals: number;
  chainId: number;
  networkName: string;
}

/**
 * Blockchain clients required for redemption
 */
export interface BlockchainClients {
  /** Public client for reading blockchain state */
  publicClient: PublicClient<Transport, Chain>;
  /** Bundler client for sending UserOperations */
  bundlerClient: unknown; // BundlerClient with version compatibility issues
  /** Pimlico client for gas estimation and utilities */
  pimlicoClient: unknown; // PimlicoClient with version compatibility issues
}

/**
 * Configuration for the redeemer smart account
 */
export interface RedeemerConfig {
  /** Private key for the EOA that controls the smart account */
  privateKey: string;
  /** Optional salt for deterministic deployment */
  deploySalt?: Hex;
  /** Implementation type for the smart account */
  implementation?: Implementation;
}

/**
 * Error types specific to redemption operations
 */
export enum RedemptionErrorType {
  VALIDATION_ERROR = 'VALIDATION_ERROR',
  NETWORK_ERROR = 'NETWORK_ERROR',
  SMART_ACCOUNT_ERROR = 'SMART_ACCOUNT_ERROR',
  USER_OPERATION_ERROR = 'USER_OPERATION_ERROR',
  DELEGATION_ERROR = 'DELEGATION_ERROR',
  UNKNOWN_ERROR = 'UNKNOWN_ERROR'
}

/**
 * Custom error class for redemption operations
 */
export class RedemptionError extends Error {
  constructor(
    message: string,
    public readonly type: RedemptionErrorType,
    public readonly details?: unknown
  ) {
    super(message);
    this.name = 'RedemptionError';
  }
}

/**
 * Gas price information from Pimlico
 */
export interface GasPrices {
  maxFeePerGas: bigint;
  maxPriorityFeePerGas: bigint;
}

/**
 * User operation receipt information
 */
export interface UserOperationReceiptInfo {
  success: boolean;
  receipt: {
    transactionHash: Hex;
    blockNumber: bigint;
    gasUsed: bigint;
  };
}