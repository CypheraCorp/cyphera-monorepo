import type { TokenResponse } from './token';

// Gas priority level configuration
export interface GasPriorityLevel {
  max_fee_per_gas: string;
  max_priority_fee_per_gas: string;
}

// Gas configuration for a network
export interface GasConfig {
  base_fee_multiplier: number;
  priority_fee_multiplier: number;
  deployment_gas_limit: string;
  token_transfer_gas_limit: string;
  supports_eip1559: boolean;
  gas_oracle_url?: string;
  gas_refresh_interval_ms: number;
  gas_priority_levels: {
    slow: GasPriorityLevel;
    standard: GasPriorityLevel;
    fast: GasPriorityLevel;
  };
  average_block_time_ms: number;
  peak_hours_multiplier: number;
}

const NetworkType = {
  EVM: 'evm',
  SOLANA: 'solana',
} as const;
export type NetworkType = (typeof NetworkType)[keyof typeof NetworkType];

// NetworkResponse represents the standardized API response for network operations
export interface NetworkResponse {
  id: string;
  object: string;
  name: string;
  type: string;
  chain_id: number; // int32
  network_type: string;
  circle_network_type: string;
  block_explorer_url?: string;
  is_testnet: boolean;
  active: boolean;
  created_at: number; // int64
  updated_at: number; // int64
  tokens?: TokenResponse[];
  // New fields from backend
  logo_url?: string;
  display_name?: string;
  chain_namespace?: string;
  gas_config?: GasConfig;
}

// CreateNetworkRequest represents the request body for creating a network
export interface CreateNetworkRequest {
  name: string;
  type: string;
  network_type: string;
  circle_network_type: string;
  block_explorer_url?: string;
  chain_id: number; // int32
  is_testnet: boolean;
  active: boolean;
}

// UpdateNetworkRequest represents the request body for updating a network
export interface UpdateNetworkRequest {
  name?: string;
  type?: string;
  network_type?: string;
  circle_network_type?: string;
  block_explorer_url?: string;
  chain_id?: number; // int32
  is_testnet?: boolean;
  active?: boolean;
}

// ListNetworksResponse represents the paginated response for network list operations
export interface ListNetworksResponse {
  object: string;
  data: NetworkResponse[];
}

// NetworkWithTokensResponse represents a network with its associated tokens
export interface NetworkWithTokensResponse {
  network: NetworkResponse;
  tokens: TokenResponse[];
}

// ListNetworksWithTokensResponse represents the response for listing networks with tokens
export interface ListNetworksWithTokensResponse {
  object: string;
  data: NetworkWithTokensResponse[];
}
