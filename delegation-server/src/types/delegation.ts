/**
 * TypeScript models for delegation data structure
 * These types are compatible with the MetaMask delegation framework
 */
import { Address, Hex } from 'viem'

/**
 * A structure representing a delegation invocation
 */
export interface DelegationInvocation {
  /** The contract to which the delegation has access */
  to: Address
  /** The function selectors that can be called */
  selectors: readonly Hex[]
  /** The max native token value that can be used in a transaction */
  value: bigint
  /** The specific contract values that can be accessed (optional) */
  valueLimit?: Hex
}

/**
 * A structure representing an authority for the delegation
 */
export interface DelegationAuthority {
  /** The signature scheme used */
  scheme: Hex
  /** The signature value */
  signature: Hex
  /** Additional data for verification */
  signer?: Address
}

/**
 * A structure representing a delegation caveat
 */
export interface DelegationCaveat {
  /** The enforcer contract address */
  enforcer: Address
  /** Terms for the caveat */
  terms: Hex
}

/**
 * A structure representing the core delegation data
 */
export interface DelegationCore {
  /** The smart contract account that acts as the delegator */
  delegator: Address
  /** The address that can act on behalf of the delegator */
  delegate: Address
  /** When the delegation expires (0 means no expiry) */
  expiry: bigint
  /** The actions that can be performed as part of the delegation */
  invocations: readonly DelegationInvocation[]
  /** The salt used in the delegation */
  salt: Hex
}

/**
 * A signed delegation with authentication
 */
export interface DelegationStruct extends DelegationCore {
  /** The authentication scheme used */
  scheme: Hex
  /** The signature of the delegation */
  signature: Hex
  /** Authority information for the delegation */
  authority: DelegationAuthority
  /** Additional restrictions on the delegation */
  caveats: readonly DelegationCaveat[]
}

/**
 * Execution data structure representing what actions to perform
 */
export interface ExecutionStruct {
  /** The target contract to interact with */
  target: Address
  /** The native token value to include in the transaction */
  value: bigint
  /** The calldata for the transaction */
  callData: Hex
}

/**
 * A contract call to be executed
 */
export interface Call {
  /** The target contract address */
  to: Address
  /** The calldata to use */
  data: Hex
  /** The value to send (optional) */
  value?: bigint
} 