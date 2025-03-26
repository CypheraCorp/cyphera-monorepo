import { DelegationStruct } from '../types/delegation';
/**
 * Validates that the input is a valid Ethereum address
 * @param address The address to validate
 * @returns true if valid, throws error if invalid
 */
export declare function isValidEthereumAddress(address: string): boolean;
/**
 * Parse a delegation from either bytes or JSON format
 * @param delegationData The delegation data as either Uint8Array or Buffer
 * @returns The parsed delegation structure
 */
export declare function parseDelegation(delegationData: Uint8Array | Buffer): DelegationStruct;
/**
 * Validates a delegation structure
 * @param delegation The delegation to validate
 * @returns true if valid, throws error if invalid
 */
export declare function validateDelegation(delegation: DelegationStruct): boolean;
