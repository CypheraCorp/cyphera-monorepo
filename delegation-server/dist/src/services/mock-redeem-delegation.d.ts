/**
 * Mock implementation of the redeemDelegation function
 * @param delegationData The serialized delegation data (signature)
 * @param merchantAddress The address of the merchant
 * @param tokenContractAddress The address of the token contract
 * @param price The price of the token
 * @returns A mock transaction hash
 */
export declare const redeemDelegation: (delegationData: Uint8Array, merchantAddress: string, tokenContractAddress: string, price: string) => Promise<string>;
