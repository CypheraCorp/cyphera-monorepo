import { CypheraAPI } from './api';
import type { TokenQuoteResponse, TokenQuotePayload } from '@/types/token';
import { logger } from '@/lib/core/logger/logger-utils';
/**
 * Tokens API class for handling token-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class TokensAPI extends CypheraAPI {
  /**
   * Gets the price of a token in USD
   * @param context - The user request context (token, IDs)
   * @param payload - The payload for the token price
   * @returns Promise with the token price response
   * @throws Error if the request fails
   */
  async getTokenQuote(payload: TokenQuotePayload): Promise<TokenQuoteResponse> {
    try {
      if (!payload.token_symbol || !payload.fiat_symbol) {
        throw new Error('Missing required parameters');
      }

      const url = `${this.baseUrl}/tokens/quote`;
      const response = await fetch(url, {
        method: 'POST',
        headers: this.getPublicHeaders(),
        body: JSON.stringify(payload),
        // Removed cache: 'no-store' to allow HTTP caching
      });

      const data = await response.json();

      return data;
    } catch (error) {
      logger.error('Error getting token price:', error);
      throw error;
    }
  }
}
