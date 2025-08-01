import type { Delegation } from '@metamask/delegation-toolkit';
import type { ProductResponse, ProductTokenResponse } from './product';

/**
 * Subscription status types
 */
export type SubscriptionStatus = 'active' | 'canceled' | 'past_due' | 'expired';

/**
 * Request payload for creating a subscription (matches Go backend SubscribeRequest)
 */
export interface SubscribeRequest {
  subscriber_address: string;
  product_id: string;
  product_token_id: string;
  token_amount: string;
  delegation: Delegation;
}

/**
 * Customer information within a subscription
 */
export interface SubscriptionCustomer {
  id: string;
  num_id: number;
  name?: string;
  email: string;
  phone?: string;
  description?: string;
  finished_onboarding: boolean;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

/**
 * Represents a subscription along with its associated price and product details (new, matches Go backend SubscriptionResponse)
 */
export interface SubscriptionResponse {
  id: string;
  num_id: number;
  workspace_id: string;
  customer_id?: string;
  customer_name?: string;
  customer_email?: string;
  customer?: SubscriptionCustomer;
  token_amount: number;
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
  product: ProductResponse;
  product_token: ProductTokenResponse;
}
