/**
 * Common constants and ABIs for delegation redemption
 */

/**
 * Standard ERC20 ABI (only the functions we need)
 */
export const erc20Abi = [
  {
    inputs: [
      { name: 'to', type: 'address' },
      { name: 'amount', type: 'uint256' }
    ],
    name: 'transfer',
    outputs: [{ name: '', type: 'bool' }],
    stateMutability: 'nonpayable',
    type: 'function'
  },
  {
    inputs: [
      { name: 'owner', type: 'address' },
      { name: 'spender', type: 'address' }
    ],
    name: 'allowance',
    outputs: [{ name: '', type: 'uint256' }],
    stateMutability: 'view',
    type: 'function'
  },
  {
    inputs: [{ name: 'account', type: 'address' }],
    name: 'balanceOf',
    outputs: [{ name: '', type: 'uint256' }],
    stateMutability: 'view',
    type: 'function'
  },
  {
    inputs: [],
    name: 'decimals',
    outputs: [{ name: '', type: 'uint8' }],
    stateMutability: 'view',
    type: 'function'
  },
  {
    inputs: [],
    name: 'symbol',
    outputs: [{ name: '', type: 'string' }],
    stateMutability: 'view',
    type: 'function'
  }
] as const;

/**
 * Default timeout for UserOperation confirmation (2 minutes)
 */
export const DEFAULT_USER_OPERATION_TIMEOUT = 120_000;

/**
 * Default gas multiplier for UserOperations
 */
export const DEFAULT_GAS_MULTIPLIER = 1.2;

/**
 * Maximum retries for UserOperation submission
 */
export const MAX_USER_OPERATION_RETRIES = 3;

/**
 * Delay between retries in milliseconds
 */
export const RETRY_DELAY_MS = 2000;