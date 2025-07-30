import { type Hex, type Address } from 'viem';

/**
 * Authority structure for delegation
 * Based on the working integration test format
 */
export interface AuthorityStruct {
  scheme: string;
  signature: string;
  signer: string;
}

/**
 * Caveat structure for delegation restrictions
 * Based on MetaMask delegation toolkit: https://docs.metamask.io/delegation-toolkit/concepts/caveat-enforcers/
 */
export interface CaveatStruct {
  enforcer: Address; // Address of the caveat enforcer contract
  terms: Hex;        // Encoded parameters defining the specific restrictions
}

/**
 * Complete delegation structure
 * This matches the format used by both Go backend and TypeScript frontend
 * Compatible with MetaMask delegation toolkit
 */
export interface DelegationStruct {
  delegate: Address;
  delegator: Address;
  authority: AuthorityStruct;
  caveats: CaveatStruct[];
  salt: string;
  signature: Hex;
}

/**
 * MetaMask delegation toolkit types re-export
 * For direct compatibility with existing code
 */
export type { Delegation, Caveat } from '@metamask/delegation-toolkit';

/**
 * Network configuration interface
 */
export interface NetworkConfig {
  chainId: number;
  name: string;
  rpcUrl: string;
  bundlerUrl?: string;
  pimlicoApiKey?: string;
  blockExplorer?: string;
  nativeCurrency: {
    name: string;
    symbol: string;
    decimals: number;
  };
}

/**
 * Validation result for delegation validation
 */
export interface ValidationResult {
  isValid: boolean;
  errors: string[];
}