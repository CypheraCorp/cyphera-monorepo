import type { UserResponse } from './user';
import type { WorkspaceResponse } from './workspace';
import type { CreateWalletRequest } from './wallet';

/**
 * Account types as defined in the API
 */
export enum AccountType {
  ADMIN = 'admin',
  MERCHANT = 'merchant',
}

/**
 * Request payload for creating an account
 */
export interface AccountRequest {
  name: string;
  account_type: AccountType;
  description?: string;
  business_name?: string;
  business_type?: string;
  website_url?: string;
  support_email?: string;
  support_phone?: string;
  finished_onboarding?: boolean;
  metadata?: Record<string, unknown>;
  // Web3Auth embedded wallet data to be created during registration
  wallet_data?: CreateWalletRequest;
}

/**
 * Request payload for onboarding a new user
 */
export interface AccountOnboardingRequest {
  address_line1?: string;
  address_line2?: string;
  city?: string;
  state?: string;
  postal_code?: string;
  country?: string;
  first_name?: string;
  last_name?: string;
  wallet_address?: string;
  finished_onboarding?: boolean;
}

/**
 * Response structure for account data (matches Go backend format)
 */
export interface AccountResponse {
  id: string;
  object: string;
  name: string;
  account_type: string;
  business_name?: string;
  business_type?: string;
  website_url?: string;
  support_email?: string;
  support_phone?: string;
  finished_onboarding: boolean;
  metadata?: Record<string, unknown>;
  created_at: number;
  updated_at: number;
  workspaces: WorkspaceResponse[];
}

/**
 * Combined response structure for account initialization
 */
export interface AccountDetailsResponse {
  account: AccountResponse;
  user: UserResponse;
}

/**
 * Request payload for updating an account
 */
export interface UpdateAccountRequest {
  name?: string;
  description?: string;
  business_name?: string;
  business_type?: string;
  website_url?: string;
  support_email?: string;
  support_phone?: string;
  account_type?: AccountType;
  finished_onboarding?: boolean;
  metadata?: Record<string, unknown>;
}

/**
 * Generic account response
 */
export interface AccountMessageResponse {
  message: string;
  finished_onboarding: boolean;
}

/**
 * Response structure for account access
 */
export interface AccountAccessResponse {
  account: AccountResponse;
  user: UserResponse;
}
