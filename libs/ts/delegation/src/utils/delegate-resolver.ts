import { type Address } from 'viem';
import { isValidEthereumAddress } from '../core/delegation-validator';

/**
 * Resolves the Cyphera delegate address from environment config or API
 * Centralized utility to avoid code duplication across web app components
 * 
 * @param envDelegateAddress - Optional delegate address from environment config
 * @returns Promise resolving to a valid Ethereum address
 * @throws Error if address cannot be resolved or is invalid
 */
export async function getCypheraDelegateAddress(envDelegateAddress?: string): Promise<Address> {
  try {
    // First check if we have the address from our environment config
    if (envDelegateAddress?.startsWith('0x')) {
      if (!isValidEthereumAddress(envDelegateAddress)) {
        throw new Error('Invalid delegate address format in environment configuration');
      }
      return envDelegateAddress as Address;
    }

    // Fallback to API request if not available in config
    const response = await fetch('/api/config/delegate-address');
    const data = await response.json();
    if (!data.success || !data.address) throw new Error('Failed to get delegate address');
    
    if (!isValidEthereumAddress(data.address)) {
      throw new Error('Invalid delegate address format from API');
    }
    
    return data.address as Address;
  } catch (error) {
    console.error('Error getting delegate address:', { error });
    throw new Error('Cyphera delegate address is not configured');
  }
}

/**
 * Type guard to check if an address is a valid delegate address
 * @param address - Address to validate
 * @returns boolean indicating if address is valid
 */
export function isValidDelegateAddress(address: unknown): address is Address {
  return typeof address === 'string' && isValidEthereumAddress(address);
}