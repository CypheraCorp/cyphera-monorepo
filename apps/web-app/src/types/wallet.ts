/**
 * Circle-specific wallet data
 */
export interface CircleWalletData {
  circle_wallet_id: string;
  circle_user_id: string;
  chain_id: number;
  state: string;
}

/**
 * Response structure for wallet data
 */
export interface WalletResponse {
  id: string;
  object: string;
  workspace_id: string;
  wallet_type: string; // 'wallet' or 'circle_wallet'
  wallet_address: string;
  network_type: string;
  network_id?: string;
  nickname?: string;
  ens?: string;
  is_primary: boolean;
  verified: boolean;
  last_used_at?: number;
  metadata?: Record<string, unknown>;
  circle_data?: CircleWalletData; // Only present for circle wallets
  created_at: number;
  updated_at: number;
}

/**
 * Response structure for wallet list operations
 */
export interface WalletListResponse {
  object: string;
  data: WalletResponse[];
}

/**
 * Request payload for creating a wallet
 */
export interface CreateWalletRequest {
  wallet_type: string; // 'wallet' or 'circle_wallet' or 'web3auth'
  wallet_address: string;
  network_type: string;
  nickname?: string;
  ens?: string;
  is_primary: boolean;
  verified: boolean;
  metadata?: Record<string, unknown>;
  // Circle wallet specific fields
  circle_user_id?: string;
  circle_wallet_id?: string;
  chain_id?: number;
  state?: string;
}

/**
 * Request payload for updating a wallet
 */
export interface UpdateWalletRequest {
  nickname?: string;
  ens?: string;
  is_primary?: boolean;
  verified?: boolean;
  metadata?: Record<string, unknown>;
  // Circle wallet specific fields
  state?: string;
}
