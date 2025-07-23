import type { Delegation } from '@metamask/delegation-toolkit';
import type { PriceResponse, ProductResponse, ProductTokenResponse } from './product';

/**
 * Subscription status types
 */
export type SubscriptionStatus = 'active' | 'canceled' | 'past_due' | 'expired';

/**
 * Request payload for creating a subscription (matches Go backend SubscribeRequest)
 */
export interface SubscribeRequest {
  subscriber_address: string;
  price_id: string;
  product_token_id: string;
  token_amount: string;
  delegation: Delegation;
}

/**
 * Represents a subscription along with its associated price and product details (new, matches Go backend SubscriptionResponse)
 */
export interface SubscriptionResponse {
  id: string;
  workspace_id: string;
  customer_id?: string;
  customer_name?: string;
  customer_email?: string;
  token_amount: string;
  total_amount_in_cents?: number;
  status: SubscriptionStatus;
  current_period_start: string;
  current_period_end: string;
  next_redemption_date?: string;
  trial_start?: string;
  trial_end?: string;
  canceled_at?: string;
  created_at: string;
  updated_at: string;
  initial_transaction_hash?: string;
  metadata?: {
    wallet_address?: string;
    [key: string]: unknown;
  };
  price: PriceResponse;
  product: ProductResponse;
  product_token: ProductTokenResponse;
}
