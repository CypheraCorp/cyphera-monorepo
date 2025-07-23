import { CypheraAPI } from './api';
import type { NetworkResponse, NetworkWithTokensResponse } from '@/types/network';
import { logger } from '@/lib/core/logger/logger-utils'; /**
 * Networks API class for handling network-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class NetworksAPI extends CypheraAPI {
  /**
   * Gets all networks with their associated tokens
   * @param context - The user request context (token, IDs)
   * @returns Promise with an array of networks and their tokens
   * @throws Error if the request fails
   */
  async getNetworksWithTokens(params: {
    active: boolean;
    testnet?: boolean;
  }): Promise<NetworkWithTokensResponse[]> {
    try {
      let url = `${this.baseUrl}/networks?active=${params.active}`;

      if (typeof params.testnet !== 'undefined') {
        url += `&testnet=${params.testnet}`;
      }

      const response = await fetch(url, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });

      const result = await this.handleResponse<{ data: NetworkWithTokensResponse[] }>(response);
      return result.data;
    } catch (error) {
      logger.error('Networks with tokens fetch failed:', error);
      throw error;
    }
  }

  /**
   * Gets a specific network by ID
   * @param context - The user request context
   * @param networkId - The ID of the network to fetch
   */
  async getNetworkById(networkId: string): Promise<NetworkResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/networks/${networkId}`, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });

      const result = await this.handleResponse<{ data: NetworkResponse }>(response);
      return result.data;
    } catch (error) {
      logger.error(`Network fetch failed for ID ${networkId}:`, error);
      throw error;
    }
  }
}
