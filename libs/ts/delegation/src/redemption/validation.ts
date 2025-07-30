import { RedemptionValidationInputs, RedemptionError, RedemptionErrorType } from './types';
import { isAddress } from 'viem';

/**
 * Validates all inputs required for delegation redemption
 * @param inputs The redemption inputs to validate
 * @throws RedemptionError if validation fails
 */
export function validateRedemptionInputs(inputs: RedemptionValidationInputs): void {
  // Validate delegation data
  if (!inputs.delegationData || inputs.delegationData.length === 0) {
    throw new RedemptionError(
      'Delegation data is required',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  // Validate merchant address
  validateAddress(inputs.merchantAddress, 'merchant');

  // Validate token contract address
  validateAddress(inputs.tokenContractAddress, 'token contract');

  // Validate token amount
  validateTokenAmount(inputs.tokenAmount);

  // Validate token decimals
  validateTokenDecimals(inputs.tokenDecimals);

  // Validate chain ID
  validateChainId(inputs.chainId);

  // Validate network name
  if (!inputs.networkName || inputs.networkName.trim() === '') {
    throw new RedemptionError(
      'Valid network name is required',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }
}

/**
 * Validates an Ethereum address
 * @param address The address to validate
 * @param addressType The type of address for error messaging
 * @throws RedemptionError if address is invalid
 */
export function validateAddress(address: string, addressType: string): void {
  if (!address || address === '0x0000000000000000000000000000000000000000') {
    throw new RedemptionError(
      `Valid ${addressType} address is required`,
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  if (!isAddress(address)) {
    throw new RedemptionError(
      `Invalid ${addressType} address format: ${address}`,
      RedemptionErrorType.VALIDATION_ERROR
    );
  }
}

/**
 * Validates token amount
 * @param tokenAmount The token amount to validate
 * @throws RedemptionError if amount is invalid
 */
export function validateTokenAmount(tokenAmount: number | bigint): void {
  if (tokenAmount === undefined || tokenAmount === null) {
    throw new RedemptionError(
      'Token amount is required',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  const amount = typeof tokenAmount === 'bigint' ? tokenAmount : BigInt(tokenAmount);
  
  if (amount <= 0n) {
    throw new RedemptionError(
      'Token amount must be greater than zero',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }
}

/**
 * Validates token decimals
 * @param tokenDecimals The number of decimals to validate
 * @throws RedemptionError if decimals are invalid
 */
export function validateTokenDecimals(tokenDecimals: number): void {
  if (!tokenDecimals || tokenDecimals <= 0) {
    throw new RedemptionError(
      'Valid token decimals are required (must be positive)',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  if (tokenDecimals > 18) {
    throw new RedemptionError(
      'Token decimals cannot exceed 18',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  if (!Number.isInteger(tokenDecimals)) {
    throw new RedemptionError(
      'Token decimals must be an integer',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }
}

/**
 * Validates chain ID
 * @param chainId The chain ID to validate
 * @throws RedemptionError if chain ID is invalid
 */
export function validateChainId(chainId: number): void {
  if (!chainId || chainId <= 0) {
    throw new RedemptionError(
      'Valid chain ID is required (must be positive)',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  if (!Number.isInteger(chainId)) {
    throw new RedemptionError(
      'Chain ID must be an integer',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }
}

/**
 * Validates that the delegate address matches the redeemer address
 * @param delegateAddress The delegate address from the delegation
 * @param redeemerAddress The redeemer smart account address
 * @throws RedemptionError if addresses don't match
 */
export function validateDelegateMatch(delegateAddress: string, redeemerAddress: string): void {
  if (!delegateAddress || !redeemerAddress) {
    throw new RedemptionError(
      'Both delegate and redeemer addresses are required for validation',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  // Case-insensitive comparison for Ethereum addresses
  if (delegateAddress.toLowerCase() !== redeemerAddress.toLowerCase()) {
    throw new RedemptionError(
      `Redeemer address (${redeemerAddress}) does not match delegate (${delegateAddress}) in delegation`,
      RedemptionErrorType.DELEGATION_ERROR,
      { delegateAddress, redeemerAddress }
    );
  }
}