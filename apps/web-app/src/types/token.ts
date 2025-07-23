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
 * Request payload for token price
 */
export interface TokenQuotePayload {
  fiat_symbol: string; // if provided, the conversion amount is in USD or EUR
  token_symbol: string; // The symbol of the token to convert TO (e.g. USDC, USDT, ETH, BTC, etc.)
}

/**
 * Response structure for token price
 */
export interface TokenQuoteResponse {
  fiat_symbol: string;
  token_symbol: string;
  token_amount_in_fiat: number;
}
