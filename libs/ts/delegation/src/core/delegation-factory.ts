import { type Hex, toHex, type Address } from 'viem';
import {
  createDelegation,
  MetaMaskSmartAccount,
  type Delegation,
} from '@metamask/delegation-toolkit';

// Type for Window with ethereum property
interface WindowWithEthereum extends Window {
  ethereum?: {
    request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
  };
}

/**
 * Generates a random salt value
 * PRESERVED from apps/web-app/src/lib/web3/utils/delegation.ts
 * @returns A hex string
 */
export function createSalt(): Hex {
  // Check if window is defined (browser environment)
  if (typeof window !== 'undefined' && window.crypto) {
    // Use browser's native crypto API
    const array = new Uint8Array(8);
    window.crypto.getRandomValues(array);
    return toHex(array);
  } else {
    // Fallback for non-browser environments or when crypto API is unavailable
    // Generate a deterministic but random-looking value based on current time
    const timestamp = Date.now().toString();
    // Create a simple hash of the timestamp
    let hash = 0;
    for (let i = 0; i < timestamp.length; i++) {
      hash = (hash << 5) - hash + timestamp.charCodeAt(i);
      hash = hash & hash; // Convert to 32bit integer
    }
    // Convert to hex string of appropriate length
    const hexHash = Math.abs(hash).toString(16).padStart(16, '0');
    return `0x${hexHash}` as Hex;
  }
}

/**
 * Creates and signs a delegation from a user's wallet or smart account
 * PRESERVED from apps/web-app/src/lib/web3/utils/delegation.ts
 * @param smartAccount - The MetaMask smart account to use for delegation
 * @param targetAddress - The address to delegate to (merchant wallet)
 * @returns A promise that resolves to the signed delegation
 */
export async function createAndSignDelegation(
  smartAccount: MetaMaskSmartAccount,
  targetAddress: Address
): Promise<Delegation> {
  try {
    // Make sure the smart account is provided
    if (!smartAccount) {
      throw new Error('Smart account is required for delegation');
    }

    // Make sure the target address is provided
    if (!targetAddress) {
      throw new Error('Target address is required for delegation');
    }

    // Create a delegation with no caveats using the MetaMask delegation toolkit
    const delegation = createDelegation({
      from: smartAccount.address, // The address to delegate from (subscriber's smart account address)
      to: targetAddress, // The address to delegate to (Cyphera admin wallet address)
      caveats: [], // No caveats for this demo
    });

    // Ensure MetaMask is ready by forcing a small transaction first
    if (typeof window !== 'undefined') {
      const windowWithEth = window as WindowWithEthereum;
      if (windowWithEth.ethereum) {
        try {
          // First ensure we have access to accounts
          await windowWithEth.ethereum.request({ method: 'eth_requestAccounts' });

          // Now check the chain ID to make sure we're on the right network
          await windowWithEth.ethereum.request({ method: 'eth_chainId' });
        } catch {
          // Silent catch - no need to log
        }
      }
    }

    // Use the smart account's signDelegation method directly
    const signature = (await smartAccount.signDelegation({
      delegation,
    })) as Hex;

    // Create the final signed delegation in standard MetaMask format
    const signedDelegation = {
      ...delegation,
      signature,
    };

    // Return the signed delegation as-is
    return signedDelegation;
  } catch (error) {
    throw new Error(error instanceof Error ? error.message : 'Failed to create delegation');
  }
}

/**
 * Formats a delegation for display
 * PRESERVED from apps/web-app/src/lib/web3/utils/delegation.ts
 * @param delegation - The delegation to format
 * @returns A formatted string representation of the delegation
 */
export function formatDelegation(delegation: Delegation): string {
  // Custom replacer function to handle BigInt serialization
  const replacer = (key: string, value: unknown): string | unknown => {
    if (typeof value === 'bigint') {
      return value.toString();
    }
    return value;
  };

  return JSON.stringify(delegation, replacer, 2);
}