/**
 * Request payload for creating a customer
 */
export interface CreateCustomerRequest {
  external_id?: string;
  email: string; // required, must be valid email
  name?: string;
  phone?: string;
  description?: string;
  balance_in_pennies?: number; // int32
  currency?: string; // required if balance_in_pennies is set
  default_source_id?: string; // must be valid UUID if provided
  invoice_prefix?: string;
  next_invoice_sequence?: number; // int32
  tax_exempt?: boolean;
  tax_ids?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  livemode?: boolean;
  finished_onboarding?: boolean; // Add onboarding status field
}

/**
 * Request payload for customer signin/register endpoint
 */
export interface CustomerSignInRequest {
  email: string;
  name?: string;
  phone?: string;
  finished_onboarding?: boolean; // Add onboarding status field
  metadata: {
    web3auth_id: string;
    verifier?: string;
    verifier_id?: string;
    [key: string]: unknown;
  };
  wallet_data?: {
    wallet_address: string;
    network_type: 'evm' | 'solana' | 'cosmos' | 'bitcoin' | 'polkadot';
    nickname?: string;
    ens?: string;
    is_primary?: boolean;
    verified?: boolean;
    metadata?: Record<string, unknown>;
  };
}

/**
 * Customer wallet response structure
 */
export interface CustomerWalletResponse {
  id: string;
  object: 'customer_wallet';
  customer_id: string;
  wallet_address: string;
  network_type: string;
  nickname?: string;
  ens?: string;
  is_primary: boolean;
  verified: boolean;
  metadata?: Record<string, unknown>;
  created_at: number;
  updated_at: number;
}

/**
 * Customer signin/register response structure
 */
export interface CustomerSignInResponse {
  success: boolean;
  data: {
    customer: {
      id: string;
      object: 'customer';
      external_id?: string;
      email: string;
      name?: string;
      phone?: string;
      description?: string;
      finished_onboarding: boolean; // Add onboarding status field
      metadata?: Record<string, unknown>;
      created_at: number;
      updated_at: number;
    };
    wallet?: CustomerWalletResponse;
  };
}

/**
 * Request payload for updating a customer
 */
export interface UpdateCustomerRequest {
  external_id?: string;
  email?: string;
  name?: string;
  phone?: string;
  description?: string;
  balance_in_pennies?: number;
  currency?: string;
  default_source_id?: string;
  invoice_prefix?: string;
  next_invoice_sequence?: number;
  tax_exempt?: boolean;
  tax_ids?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  livemode?: boolean;
  finished_onboarding?: boolean; // Add onboarding status field
}

/**
 * Response structure for customer data
 */
export interface CustomerResponse {
  id: string;
  object: string;
  workspace_id: string;
  external_id?: string;
  email: string;
  name?: string;
  phone?: string;
  description?: string;
  finished_onboarding: boolean; // Add onboarding status field
  metadata?: Record<string, unknown>;
  balance_in_pennies: number;
  currency: string;
  default_source_id?: string;
  invoice_prefix?: string;
  next_invoice_number: number;
  tax_exempt: boolean;
  tax_ids?: Record<string, unknown>;
  livemode: boolean;
  total_revenue?: number; // Total revenue in cents from completed payments
  created_at: number;
  updated_at: number;
  workspace_name?: string;
  business_name?: string;
}

/**
 * Request payload for updating customer onboarding status
 */
export interface UpdateCustomerOnboardingRequest {
  finished_onboarding: boolean;
}
