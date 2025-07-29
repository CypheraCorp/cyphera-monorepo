/**
 * Response structure for token data
 */
export interface TokenResponse {
  id: string;
  object: string;
  network_id: string;
  gas_token: boolean;
  name: string;
  symbol: string;
  contract_address: string;
  decimals: number; // Added
  active: boolean;
  created_at: number;
  updated_at: number;
  deleted_at?: number; // Added, optional
}

/**
 * Request payload for token price - MUST match backend GetTokenQuoteRequest
 */
export interface TokenQuotePayload {
  token_id: string;    // UUID of the token
  network_id: string;  // UUID of the network
  amount_wei: string;  // Amount in wei (smallest unit)
  to_currency: string; // Target currency (e.g., "USD", "EUR")
}

/**
 * Response structure for token price
 */
export interface TokenQuoteResponse {
  fiat_symbol: string;
  token_symbol: string;
  token_amount_in_fiat: number;
}
