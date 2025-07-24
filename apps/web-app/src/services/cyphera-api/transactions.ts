import { CypheraAPI, UserRequestContext } from './api';
import type { PaginatedResponse, PaginationParams } from '@/types/common';
import type { SubscriptionEventResponse } from '@/types/subscription-event';
import { logger } from '@/lib/core/logger/logger-utils'; /**
 * Transactions API class for handling transaction-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class TransactionsAPI extends CypheraAPI {
  /**
   * Gets Cyphera (subscription) transactions for the current workspace with pagination
   * @param context - The user request context (token, IDs)
   * @param params - Pagination parameters
   * @returns Promise with the transactions response and pagination metadata
   * @throws Error if the request fails
   */
  async getTransactions(
    context: UserRequestContext,
    params?: PaginationParams
  ): Promise<PaginatedResponse<SubscriptionEventResponse>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    const url = `${this.baseUrl}/subscription-events/transactions?${queryParams.toString()}`;

    try {
      logger.info('Fetching transactions from:', { url });
      return await this.fetchWithRateLimit<PaginatedResponse<SubscriptionEventResponse>>(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Cyphera Transactions fetch failed:', error);
      throw error;
    }
  }

  /**
   * Gets a single Cyphera (subscription) transaction by ID
   * @param context - The user request context (token, IDs)
   * @param transactionId - The ID of the transaction to fetch
   * @returns Promise with the transaction response
   * @throws Error if the request fails
   */
  async getTransactionById(
    context: UserRequestContext,
    transactionId: string
  ): Promise<SubscriptionEventResponse> {
    try {
      return await this.fetchWithRateLimit<SubscriptionEventResponse>(`${this.baseUrl}/subscription-events/${transactionId}`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Cyphera Transaction fetch failed:', error);
      throw error;
    }
  }
}
