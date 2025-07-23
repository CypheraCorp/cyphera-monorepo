import { PaginatedResponse, PaginationParams } from '@/types/common';
import { CypheraAPI, UserRequestContext } from './api';
import type { SubscriptionResponse } from '@/types/subscription';
import { logger } from '@/lib/core/logger/logger-utils';
/**
 * Subscriptions API class for handling subscription-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class SubscriptionsAPI extends CypheraAPI {
  /**
   * Gets subscriptions for the current workspace with pagination
   * @param context - The user request context (token, IDs)
   * @param params - Pagination parameters (page and limit)
   * @returns Promise with the subscriptions response and pagination metadata
   * @throws Error if the request fails
   */
  async getSubscriptions(
    context: UserRequestContext,
    params?: PaginationParams
  ): Promise<PaginatedResponse<SubscriptionResponse>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    const url = `${this.baseUrl}/subscriptions?${queryParams.toString()}`;

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });

      const data = await this.handleResponse<PaginatedResponse<SubscriptionResponse>>(response);

      return data;
    } catch (error) {
      logger.error('Subscriptions fetch failed:', error);
      throw error;
    }
  }

  /**
   * Gets a single subscription by ID
   * @param context - The user request context (token, IDs)
   * @param subscriptionId - The ID of the subscription to fetch
   * @returns Promise with the subscription response
   * @throws Error if the request fails
   */
  async getSubscriptionById(
    context: UserRequestContext,
    subscriptionId: string
  ): Promise<SubscriptionResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/subscriptions/${subscriptionId}`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
      return await this.handleResponse<SubscriptionResponse>(response);
    } catch (error) {
      logger.error('Subscription fetch failed:', error);
      throw error;
    }
  }
}
