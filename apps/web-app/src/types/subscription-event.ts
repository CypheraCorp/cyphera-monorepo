import { CustomerResponse } from './customer';
import { NetworkResponse } from './network';
import { ProductTokenResponse } from './product';
import type { SubscriptionStatus } from './subscription';

/**
 * Subscription event/transaction types
 */
export type SubscriptionEventType = 'redeemed' | 'failed' | 'failed_redemption';

/**
 * Contains essential price information relevant to a subscription event.
 * Simplified compared to a full PriceResponse for embedding.
 */
export interface SubscriptionEventPriceInfo {
  id: string; // uuid.UUID
  type: string; // db.PriceType (e.g., recurring, one_off)
  currency: string; // db.Currency
  unit_amount_in_pennies: number; // int32
  interval_type?: string; // db.NullIntervalType (optional)
  interval_count?: number; // pgtype.Int4 (optional)
  term_length?: number; // pgtype.Int4 (optional)
}

/**
 * Represents a detailed view of a subscription event.
 * Includes denormalized information from related entities like subscription, product, and price.
 */
export interface SubscriptionEventFullResponse {
  id: string; // uuid.UUID
  subscription_id: string; // uuid.UUID
  event_type: SubscriptionEventType; // db.SubscriptionEventType
  transaction_hash?: string; // pgtype.Text (optional)
  event_amount_in_cents: number; // int32
  event_occurred_at: string; // pgtype.Timestamptz
  error_message?: string; // pgtype.Text (optional)
  event_metadata?: Record<string, unknown> | null; // json.RawMessage (optional)
  event_created_at: string; // pgtype.Timestamptz
  customer_id: string; // uuid.UUID
  subscription_status: SubscriptionStatus; // db.SubscriptionStatus
  product_id: string; // uuid.UUID
  product_name: string;
  price_info: SubscriptionEventPriceInfo;
  product_token: ProductTokenResponse;
  network: NetworkResponse;
  customer: CustomerResponse;
}
