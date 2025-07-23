/**
 * Circle utility functions
 */

/**
 * Format a wallet address for display (shortening with ellipsis)
 * @param address - The wallet address to format
 * @param prefixLength - Number of characters to show at the start
 * @param suffixLength - Number of characters to show at the end
 * @returns The formatted address string, or an empty string if the address is null/undefined.
 */
export function formatAddress(
  address: string | undefined | null, // Allow undefined/null
  prefixLength: number = 6,
  suffixLength: number = 4
): string {
  if (!address) return ''; // Return empty string for null/undefined
  if (address.length <= prefixLength + suffixLength) return address;

  return `${address.substring(0, prefixLength)}...${address.substring(address.length - suffixLength)}`;
}

/**
 * Extract blockchain identifier (e.g., ETH-SEPOLIA) from a Circle wallet ID.
 * Circle wallet IDs typically follow the format "BLOCKCHAIN_ID:WALLET_UUID".
 * @param walletId - The full Circle wallet ID string.
 * @returns The blockchain identifier part of the ID, or an empty string if the format is incorrect or the ID is invalid.
 */
export function getBlockchainFromCircleWalletId(walletId: string | undefined | null): string {
  if (!walletId || !walletId.includes(':')) return '';
  return walletId.split(':')[0];
}
