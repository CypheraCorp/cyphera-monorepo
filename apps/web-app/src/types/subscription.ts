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
  addons?: SubscriptionAddonRequest[];
}

/**
 * Subscription addon request
 */
export interface SubscriptionAddonRequest {
  product_id: string;
  quantity: number;
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
 * Subscription line item response
 */
export interface SubscriptionLineItemResponse {
  id: string;
  subscription_id: string;
  product_id: string;
  line_item_type: 'base' | 'addon';
  quantity: number;
  unit_amount_in_pennies: number;
  currency: string;
  price_type: string;
  interval_type?: string;
  total_amount_in_pennies: number;
  is_active: boolean;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  product?: ProductResponse;
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
  status: SubscriptionStatus;
  current_period_start: string;
  current_period_end: string;
  next_redemption_date?: string;
  total_redemptions: number;
  total_amount_in_cents?: number;
  token_amount: number;
  delegation_id: string;
  customer_wallet_id?: string;
  external_id?: string;
  payment_sync_status?: string;
  payment_synced_at?: string;
  payment_sync_version?: number;
  payment_provider?: string;
  initial_transaction_hash?: string;
  metadata?: {
    wallet_address?: string;
    [key: string]: unknown;
  };
  created_at: string;
  updated_at: string;
  trial_start?: string;
  trial_end?: string;
  canceled_at?: string;
  cancel_at?: string; // Scheduled cancellation date
  cancellation_reason?: string;
  paused_at?: string;
  pause_ends_at?: string;
  product: ProductResponse;
  product_token: ProductTokenResponse;
  line_items?: SubscriptionLineItemResponse[];
  scheduled_changes?: ScheduledChange[];
}

/**
 * Line item update for subscription changes
 */
export interface LineItemUpdate {
  action: 'add' | 'update' | 'remove';
  line_item_id?: string;
  product_id?: string;
  price_id?: string;
  product_token_id?: string;
  quantity: number;
  unit_amount?: number;
}

/**
 * Request types for subscription management
 */
export interface UpgradeSubscriptionRequest {
  line_items: LineItemUpdate[];
  reason?: string;
}

export interface DowngradeSubscriptionRequest {
  line_items: LineItemUpdate[];
  reason?: string;
}

export interface CancelSubscriptionRequest {
  reason: string;
  feedback?: string;
}

export interface PauseSubscriptionRequest {
  pause_until?: string; // ISO 8601 timestamp
  reason?: string;
}

export interface PreviewChangeRequest {
  change_type: 'upgrade' | 'downgrade';
  line_items: LineItemUpdate[];
}

/**
 * Proration calculation result
 */
export interface ProrationResult {
  credit_amount: number;
  charge_amount: number;
  net_amount: number;
  days_total: number;
  days_used: number;
  days_remaining: number;
  old_daily_rate: number;
  new_daily_rate: number;
  calculation: Record<string, any>;
}

/**
 * Change preview response
 */
export interface ChangePreview {
  current_amount: number;
  new_amount: number;
  proration_credit?: number;
  immediate_charge?: number;
  effective_date: string;
  proration_details?: ProrationResult;
  message?: string;
}

/**
 * Scheduled change information
 */
export interface ScheduledChange {
  id: string;
  subscription_id: string;
  change_type: 'upgrade' | 'downgrade' | 'cancel' | 'pause' | 'resume';
  scheduled_for: string;
  from_line_items?: any;
  to_line_items?: any;
  proration_amount_cents?: number;
  proration_calculation?: any;
  status: 'scheduled' | 'processing' | 'completed' | 'cancelled' | 'failed';
  reason?: string;
  initiated_by?: string;
  processed_at?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Subscription state history entry
 */
export interface SubscriptionStateHistory {
  id: string;
  subscription_id: string;
  from_status?: SubscriptionStatus;
  to_status: SubscriptionStatus;
  from_amount_cents?: number;
  to_amount_cents?: number;
  line_items_snapshot?: any;
  change_reason?: string;
  schedule_change_id?: string;
  occurred_at: string;
}
