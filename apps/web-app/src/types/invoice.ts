// Invoice type definitions

export type InvoiceStatus = 'draft' | 'open' | 'paid' | 'void' | 'uncollectible';

export interface InvoiceMetadata {
  generated_from?: string;
  transaction_hash?: string;
  payment_id?: string;
  payer_wallet?: string;
  receiver_wallet?: string;
  [key: string]: any;
}

export interface InvoiceLineItem {
  id: string;
  invoice_id: string;
  description: string;
  quantity: number;
  unit_amount_in_cents: number;
  amount_in_cents: number;
  fiat_currency: string;
  line_item_type?: string;
  subscription_line_item_id?: string;
  gas_fee_payment_id?: string;
  is_gas_sponsored?: boolean;
  gas_sponsor_type?: string;
  gas_sponsor_name?: string;
  tax_rate?: number;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface TaxDetail {
  jurisdiction_id: string;
  jurisdiction_name: string;
  tax_rate: number;
  tax_amount_cents: number;
  tax_type: string;
}

export interface Invoice {
  id: string;
  workspace_id: string;
  customer_id: string;
  subscription_id?: string;
  invoice_number: string;
  status: InvoiceStatus;
  currency: string;
  due_date?: string;
  product_subtotal: number;
  gas_fees_subtotal: number;
  sponsored_gas_fees: number;
  tax_amount: number;
  discount_amount: number;
  total_amount: number;
  customer_total: number;
  line_items: InvoiceLineItem[];
  tax_details: TaxDetail[];
  payment_link_id?: string;
  payment_link_url?: string;
  period_start?: string;
  period_end?: string;
  reminder_sent_at?: string;
  reminder_count: number;
  notes?: string;
  terms?: string;
  footer?: string;
  metadata?: InvoiceMetadata;
  created_at: string;
  updated_at: string;
}

export interface InvoiceActivity {
  id: string;
  invoice_id: string;
  workspace_id: string;
  activity_type: string;
  from_status?: string;
  to_status?: string;
  performed_by?: string;
  description?: string;
  metadata?: Record<string, any>;
  created_at: string;
}

// API response types
export interface InvoiceListParams {
  limit?: number;
  offset?: number;
  status?: InvoiceStatus;
  customer_id?: string;
}

export interface InvoiceListResponse {
  invoices: Invoice[];
  total: number;
  limit: number;
  offset: number;
}

export interface BulkInvoiceError {
  subscription_id: string;
  customer_id: string;
  error: string;
}

export interface BulkInvoiceGenerationResult {
  success: Invoice[];
  failed: BulkInvoiceError[];
  total_processed: number;
  success_count: number;
  failed_count: number;
}

export interface InvoiceStatsResponse {
  draft_count: number;
  open_count: number;
  paid_count: number;
  void_count: number;
  uncollectible_count: number;
  total_count: number;
  total_outstanding_cents: number;
  total_paid_cents: number;
  total_uncollectible_cents: number;
  currency: string;
  period_start: string;
  period_end: string;
}