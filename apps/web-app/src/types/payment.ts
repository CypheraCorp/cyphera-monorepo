// Payment-related types

export interface PaymentResponse {
  id: string;
  workspace_id: string;
  invoice_id?: string;
  subscription_id?: string;
  subscription_event_id?: string;
  customer_id: string;
  customer?: CustomerBasic;
  amount_in_cents: number;
  formatted_amount: string;
  currency: string;
  status: 'pending' | 'completed' | 'failed' | 'processing';
  payment_method: 'crypto' | 'card' | 'bank';
  transaction_hash?: string;
  network_id?: string;
  network?: NetworkBasic;
  token_id?: string;
  token?: TokenBasic;
  crypto_amount?: string;
  exchange_rate?: string;
  has_gas_fee: boolean;
  gas_fee_usd_cents?: number;
  gas_sponsored: boolean;
  external_payment_id?: string;
  payment_provider?: string;
  product_amount_cents: number;
  tax_amount_cents: number;
  gas_amount_cents: number;
  discount_amount_cents: number;
  product_name?: string;
  product_id?: string;
  initiated_at: string;
  completed_at?: string;
  failed_at?: string;
  error_message?: string;
  metadata?: any;
  created_at: string;
  updated_at: string;
}

export interface CustomerBasic {
  id: string;
  name: string;
  email: string;
}

export interface NetworkBasic {
  id: string;
  name: string;
  chain_id: number;
  display_name: string;
}

export interface TokenBasic {
  id: string;
  symbol: string;
  name: string;
  contract_address: string;
  decimals: number;
}

export interface PaymentListResponse {
  data: PaymentResponse[];
  pagination: PaginationMeta;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
  has_prev: boolean;
  has_next: boolean;
}