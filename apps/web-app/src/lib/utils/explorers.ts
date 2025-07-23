// No longer needs 'use client' as it's not a hook
import type { NetworkWithTokensResponse } from '@/types/network';
/**
 * Generates a block explorer URL for a given chain ID, type, and value (hash/address).
 *
 * @param networks The list of all available network configurations.
 * @param chainId The chain ID of the network.
 * @param type The type of link ('tx' for transaction, 'address' for address).
 * @param hashOrAddress The transaction hash or wallet address.
 * @returns The full explorer URL string, or null if the network or explorer URL is not found.
 */
export function generateExplorerLink(
  networks: NetworkWithTokensResponse[] | undefined,
  chainId: number | undefined,
  type: 'tx' | 'address',
  hashOrAddress: string | null | undefined
): string | null {
  if (!networks || !chainId || !hashOrAddress) {
    return null;
  }

  // Find the network
  const network = networks.find((n) => n.network.chain_id === chainId);

  // Use block_explorer_url directly from the network object
  const baseUrl = network?.network.block_explorer_url;

  if (!baseUrl) {
    // Cannot generate link without base URL
    return null;
  }

  // Ensure baseUrl doesn't end with a slash
  const cleanBaseUrl = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;

  // Construct the full URL
  return `${cleanBaseUrl}/${type}/${hashOrAddress}`;
}

/**
 * Example Usage (Server Component):
 *
 * async function MyServerComponent() {
 *   const networks = await fetchNetworks(); // Fetch network data
 *
 *   function TransactionRow({ tx }) {
 *     const txLink = generateExplorerLink(networks, tx.chainId, 'tx', tx.hash);
 *     return (
 *       <div>
 *         {txLink && <a href={txLink} target="_blank">View Transaction</a>}
 *       </div>
 *     );
 *   }
 *   // ... render TransactionRow for each transaction
 * }
 */
