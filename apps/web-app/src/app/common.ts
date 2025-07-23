import { NetworkWithTokensResponse } from '@/types/network';
import { NetworksAPI } from '@/services/cyphera-api/networks';
import { logger } from '@/lib/core/logger/logger';

/**
 * Fetches all active networks (including testnets)
 */
export async function getNetworksWithTokens(): Promise<NetworkWithTokensResponse[]> {
  try {
    // Instantiate the NetworksAPI service
    const networksApi = new NetworksAPI();

    // Call the public API method
    // The API route defaults to active: true and handles testnet param if present.
    // For a common server-side utility, fetching all active=true networks is a sensible default.
    const networks = await networksApi.getNetworksWithTokens({ active: true });

    return networks || []; // Ensure it returns an empty array on null/undefined
  } catch (error) {
    logger.error('Failed to fetch networks directly from service', {
      error: error instanceof Error ? error.message : error,
    });
    return [];
  }
}
