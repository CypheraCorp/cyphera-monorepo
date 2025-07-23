/**
 * Response structure for user data (matches Go backend format)
 */
export interface UserResponse {
  id: string;
  object: string;
  web3auth_id?: string;
  verifier?: string;
  verifier_id?: string;
  email: string;
  first_name?: string;
  last_name?: string;
  address_line_1?: string;
  address_line_2?: string;
  city?: string;
  state_region?: string;
  postal_code?: string;
  country?: string;
  display_name?: string;
  picture_url?: string;
  phone?: string;
  timezone?: string;
  locale?: string;
  email_verified: boolean;
  two_factor_enabled: boolean;
  finished_onboarding: boolean;
  status: string;
  metadata?: Record<string, unknown>;
  created_at: number;
  updated_at: number;
}
