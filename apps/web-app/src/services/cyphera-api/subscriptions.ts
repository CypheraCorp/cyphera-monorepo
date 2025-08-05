import { PaginatedResponse, PaginationParams } from '@/types/common';
import { CypheraAPI, UserRequestContext } from './api';
import type { 
  SubscriptionResponse, 
  UpgradeSubscriptionRequest,
  DowngradeSubscriptionRequest,
  CancelSubscriptionRequest,
  PauseSubscriptionRequest,
  PreviewChangeRequest,
  ChangePreview,
  SubscriptionStateHistory
} from '@/types/subscription';
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
      return await this.fetchWithRateLimit<PaginatedResponse<SubscriptionResponse>>(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
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
      return await this.fetchWithRateLimit<SubscriptionResponse>(`${this.baseUrl}/subscriptions/${subscriptionId}`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Subscription fetch failed:', error);
      throw error;
    }
  }

  /**
   * Preview a subscription change (upgrade or downgrade)
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @param request - The preview change request
   * @returns Promise with the change preview including proration details
   */
  async previewChange(
    context: UserRequestContext,
    subscriptionId: string,
    request: PreviewChangeRequest
  ): Promise<ChangePreview> {
    try {
      return await this.fetchWithRateLimit<ChangePreview>(
        `${this.baseUrl}/subscriptions/${subscriptionId}/preview-change`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
          body: JSON.stringify(request),
        }
      );
    } catch (error) {
      logger.error('Subscription change preview failed:', error);
      throw error;
    }
  }

  /**
   * Upgrade a subscription immediately with proration
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @param request - The upgrade request with new line items
   * @returns Promise with the success response
   */
  async upgradeSubscription(
    context: UserRequestContext,
    subscriptionId: string,
    request: UpgradeSubscriptionRequest
  ): Promise<{ message: string; subscription_id: string; status: string }> {
    try {
      return await this.fetchWithRateLimit(
        `${this.baseUrl}/subscriptions/${subscriptionId}/upgrade`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
          body: JSON.stringify(request),
        }
      );
    } catch (error) {
      logger.error('Subscription upgrade failed:', error);
      throw error;
    }
  }

  /**
   * Schedule a subscription downgrade for end of billing period
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @param request - The downgrade request with new line items
   * @returns Promise with the success response
   */
  async downgradeSubscription(
    context: UserRequestContext,
    subscriptionId: string,
    request: DowngradeSubscriptionRequest
  ): Promise<{ message: string; subscription_id: string; status: string }> {
    try {
      return await this.fetchWithRateLimit(
        `${this.baseUrl}/subscriptions/${subscriptionId}/downgrade`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
          body: JSON.stringify(request),
        }
      );
    } catch (error) {
      logger.error('Subscription downgrade failed:', error);
      throw error;
    }
  }

  /**
   * Cancel a subscription at the end of the billing period
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @param request - The cancellation request with reason and feedback
   * @returns Promise with the success response
   */
  async cancelSubscription(
    context: UserRequestContext,
    subscriptionId: string,
    request: CancelSubscriptionRequest
  ): Promise<{ message: string; subscription_id: string; status: string }> {
    try {
      return await this.fetchWithRateLimit(
        `${this.baseUrl}/subscriptions/${subscriptionId}/cancel`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
          body: JSON.stringify(request),
        }
      );
    } catch (error) {
      logger.error('Subscription cancellation failed:', error);
      throw error;
    }
  }

  /**
   * Reactivate a subscription that was scheduled for cancellation
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @returns Promise with the success response
   */
  async reactivateSubscription(
    context: UserRequestContext,
    subscriptionId: string
  ): Promise<{ message: string; subscription_id: string; status: string }> {
    try {
      return await this.fetchWithRateLimit(
        `${this.baseUrl}/subscriptions/${subscriptionId}/reactivate`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
        }
      );
    } catch (error) {
      logger.error('Subscription reactivation failed:', error);
      throw error;
    }
  }

  /**
   * Pause a subscription immediately or until a specific date
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @param request - The pause request with optional end date
   * @returns Promise with the success response
   */
  async pauseSubscription(
    context: UserRequestContext,
    subscriptionId: string,
    request: PauseSubscriptionRequest
  ): Promise<{ message: string; subscription_id: string; status: string; pause_until?: string }> {
    try {
      return await this.fetchWithRateLimit(
        `${this.baseUrl}/subscriptions/${subscriptionId}/pause`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
          body: JSON.stringify(request),
        }
      );
    } catch (error) {
      logger.error('Subscription pause failed:', error);
      throw error;
    }
  }

  /**
   * Resume a paused subscription
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @returns Promise with the success response
   */
  async resumeSubscription(
    context: UserRequestContext,
    subscriptionId: string
  ): Promise<{ message: string; subscription_id: string; status: string }> {
    try {
      return await this.fetchWithRateLimit(
        `${this.baseUrl}/subscriptions/${subscriptionId}/resume`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
        }
      );
    } catch (error) {
      logger.error('Subscription resume failed:', error);
      throw error;
    }
  }

  /**
   * Get subscription state change history
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @param limit - Number of history entries to return (default: 50)
   * @returns Promise with the history entries
   */
  async getSubscriptionHistory(
    context: UserRequestContext,
    subscriptionId: string,
    limit: number = 50
  ): Promise<{ history: SubscriptionStateHistory[]; count: number }> {
    try {
      return await this.fetchWithRateLimit(
        `${this.baseUrl}/subscriptions/${subscriptionId}/history?limit=${limit}`,
        {
          method: 'GET',
          headers: this.getHeaders(context),
        }
      );
    } catch (error) {
      logger.error('Subscription history fetch failed:', error);
      throw error;
    }
  }

  /**
   * Change subscription price - automatically handles upgrade/downgrade logic
   * @param context - The user request context
   * @param subscriptionId - The ID of the subscription
   * @param newPriceCents - The new price in cents
   * @returns Promise with the success response
   */
  async changePrice(
    context: UserRequestContext,
    subscriptionId: string,
    newPriceCents: number
  ): Promise<{ message: string; subscription_id: string }> {
    try {
      return await this.fetchWithRateLimit(
        `${this.baseUrl}/subscriptions/${subscriptionId}/change-price`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
          body: JSON.stringify({ new_price_cents: newPriceCents }),
        }
      );
    } catch (error) {
      logger.error('Subscription price change failed:', error);
      throw error;
    }
  }
}
