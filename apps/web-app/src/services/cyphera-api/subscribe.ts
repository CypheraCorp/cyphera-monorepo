import { DelegationStruct as MetaMaskDelegationStruct } from '@metamask/delegation-toolkit';
import { type SubscribeRequest, type SubscriptionResponse } from '@/types/subscription';
// No UserRequestContext needed here as this uses public headers
import { CypheraAPI } from './api';
import { logger } from '@/lib/core/logger/logger-utils';
/**
 * Products API class for handling product-related API requests
 * Extends the base CypheraAPI class
 */
export class SubscribeAPI extends CypheraAPI {
  /**
   * Transform a ProductResponse into a Product with UI-specific fields
   */

  /**
   * Custom serializer for JSON.stringify that handles BigInt conversion
   * Converts BigInt values to strings with format "BigInt(value)"
   */
  private replaceBigInt(key: string, value: unknown): string | unknown {
    if (typeof value === 'bigint') {
      return value.toString();
    }
    return value;
  }

  /**
   * Submits a subscription request to the Golang backend
   * @param productId - The ID of the product being subscribed to
   * @param productTokenId - The specific token ID for the product/network combination
   * @param delegation - The signed delegation from the smart account
   * @param smartAccountAddress - The subscriber's smart account address
   * @returns Promise with the result of the subscription request
   */
  async submitSubscription(
    priceId: string,
    productTokenId: string,
    tokenAmount: string,
    delegation: MetaMaskDelegationStruct,
    smartAccountAddress: string
  ): Promise<{ success: boolean; message: string; data?: SubscriptionResponse }> {
    // Validate input (keep existing checks)
    if (!delegation) throw new Error('Delegation is required for subscription');
    if (!smartAccountAddress) throw new Error('Smart account address is required for subscription');
    if (!priceId) throw new Error('Price ID is required for subscription');
    if (!productTokenId) throw new Error('Product Token ID is required for subscription');

    try {
      const serializedDelegation = JSON.parse(JSON.stringify(delegation, this.replaceBigInt));
      const subscribeRequest: SubscribeRequest = {
        subscriber_address: smartAccountAddress,
        product_token_id: productTokenId,
        price_id: priceId,
        delegation: serializedDelegation,
        token_amount: tokenAmount,
      };

      const apiEndpoint = `/admin/prices/${priceId}/subscribe`;
      const url = `${this.baseUrl}${apiEndpoint}`;

      const responseData = await this.fetchWithRateLimit<SubscriptionResponse>(url, {
        method: 'POST',
        headers: this.getPublicHeaders(), // Use getPublicHeaders
        body: JSON.stringify(subscribeRequest),
      });

      return {
        success: true,
        message: 'Subscription created successfully',
        data: responseData, // Return the full subscription data from Go backend
      };
    } catch (error) {
      // Log the error and return a structured failure response
      logger.error(`Subscription submission failed for price ${priceId}:`, error);
      return {
        success: false,
        message: error instanceof Error ? error.message : 'Failed to subscribe to price',
      };
    }
  }
}
