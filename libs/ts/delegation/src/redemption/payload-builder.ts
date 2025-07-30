import { 
  type Address, 
  type Hex,
  encodeFunctionData,
  parseUnits
} from 'viem';
import { 
  type ExecutionStruct,
  type Call,
  type Delegation,
  type ExecutionMode,
  DelegationFramework,
  SINGLE_DEFAULT_MODE
} from '@metamask/delegation-toolkit';
import { RedemptionError, RedemptionErrorType } from './types';
import { erc20Abi } from './constants';

/**
 * Prepares the payload (calls array) for the redeemDelegations UserOperation
 * @param delegation The parsed delegation object
 * @param merchantAddress The merchant address to receive tokens
 * @param tokenContractAddress The token contract address
 * @param tokenAmount The amount of tokens to transfer
 * @param tokenDecimals The number of decimals for the token
 * @param redeemerAddress The address of the redeemer smart account
 * @returns Array of calls for the UserOperation
 */
export function prepareRedemptionUserOperationPayload(
  delegation: Delegation,
  merchantAddress: string,
  tokenContractAddress: string,
  tokenAmount: number | bigint,
  tokenDecimals: number,
  redeemerAddress: Address
): Call[] {
  try {
    // Convert token amount to bigint if needed
    const tokenAmountBigInt = prepareTokenAmount(tokenAmount, tokenDecimals);

    // Encode the ERC20 transfer that will be executed by the delegator
    const transferCalldata = encodeERC20Transfer(
      merchantAddress as Address,
      tokenAmountBigInt
    );
    
    // Build execution struct for the token transfer
    const executions: ExecutionStruct[] = [{
      target: tokenContractAddress as Address,
      value: 0n,
      callData: transferCalldata
    }];

    // Create delegation chain (array of delegations)
    const delegationChain = [delegation];

    // Use DelegationFramework to encode the redemption
    // This will execute the transfer from the delegator's account
    const redeemDelegationCalldata = DelegationFramework.encode.redeemDelegations({
      delegations: [delegationChain],
      modes: [SINGLE_DEFAULT_MODE],
      executions: [executions]
    });

    // Return the call to the redeemer's account with the redemption data
    return [{
      to: redeemerAddress,
      data: redeemDelegationCalldata,
    }];
  } catch (error) {
    throw new RedemptionError(
      'Failed to prepare redemption payload',
      RedemptionErrorType.USER_OPERATION_ERROR,
      error
    );
  }
}

/**
 * Prepares the token amount for use in the transfer
 * Handles conversion from number to bigint with proper decimal handling
 * @param tokenAmount The token amount (can be number or bigint)
 * @param tokenDecimals The number of decimals
 * @returns The token amount as bigint
 */
export function prepareTokenAmount(
  tokenAmount: number | bigint,
  tokenDecimals: number
): bigint {
  if (tokenDecimals <= 0) {
    throw new RedemptionError(
      'Token decimals must be positive for parsing',
      RedemptionErrorType.VALIDATION_ERROR
    );
  }

  // If already bigint, return as is
  if (typeof tokenAmount === 'bigint') {
    return tokenAmount;
  }

  // Convert number to string with proper decimal places
  const humanReadableAmountString = (tokenAmount / Math.pow(10, tokenDecimals))
    .toFixed(tokenDecimals);
  
  // Parse using viem's parseUnits
  return parseUnits(humanReadableAmountString, tokenDecimals);
}

/**
 * Encodes an ERC20 transfer function call
 * @param recipient The recipient address
 * @param amount The amount to transfer
 * @returns The encoded function data
 */
export function encodeERC20Transfer(
  recipient: Address,
  amount: bigint
): Hex {
  return encodeFunctionData({
    abi: erc20Abi,
    functionName: 'transfer',
    args: [recipient, amount]
  });
}

/**
 * Builds an execution struct for the MetaMask delegation toolkit
 * @param target The target contract address
 * @param callData The encoded function call data
 * @param value Optional ETH value to send (defaults to 0)
 * @returns ExecutionStruct object
 */
export function buildExecutionStruct(
  target: Address,
  callData: Hex,
  value: bigint = 0n
): ExecutionStruct {
  return {
    target,
    value,
    callData
  };
}

/**
 * Encodes a delegation redemption call using the MetaMask DelegationFramework
 * @param delegations Array of delegations (usually just one)
 * @param executions Array of execution structs
 * @param modes Optional modes array (defaults to SINGLE_DEFAULT_MODE)
 * @returns The encoded redemption call data
 */
export function encodeDelegationRedemption(
  delegations: Delegation[],
  executions: ExecutionStruct[],
  modes?: ExecutionMode[]
): Hex {
  // Create delegation chains (each delegation is wrapped in an array)
  const delegationChains = delegations.map(d => [d]);
  
  // Use DelegationFramework to properly encode the redemption
  return DelegationFramework.encode.redeemDelegations({
    delegations: delegationChains,
    modes: modes || [SINGLE_DEFAULT_MODE],
    executions: [executions]
  });
}

/**
 * Creates a batch redemption payload for multiple delegations
 * This is an advanced use case for redeeming multiple delegations in one transaction
 * @param redemptions Array of redemption details
 * @param redeemerAddress The redeemer smart account address
 * @returns Array of calls for batch redemption
 */
export interface BatchRedemptionDetails {
  delegation: Delegation;
  merchantAddress: Address;
  tokenContractAddress: Address;
  tokenAmount: bigint;
  tokenDecimals: number;
}

export function prepareBatchRedemptionPayload(
  redemptions: BatchRedemptionDetails[],
  _redeemerAddress: Address
): Call[] {
  // Handle empty batch
  if (redemptions.length === 0) {
    // Return a placeholder call for empty batch
    return [{
      to: _redeemerAddress,
      data: '0x' as Hex
    }];
  }

  // For batch redemptions, we'll return individual calls for each redemption
  const calls: Call[] = [];
  
  for (const redemption of redemptions) {
    const tokenAmountBigInt = typeof redemption.tokenAmount === 'bigint' 
      ? redemption.tokenAmount 
      : BigInt(redemption.tokenAmount);
      
    const transferCalldata = encodeERC20Transfer(
      redemption.merchantAddress,
      tokenAmountBigInt
    );
    
    calls.push({
      to: redemption.tokenContractAddress,
      data: transferCalldata
    });
  }

  return calls;
}