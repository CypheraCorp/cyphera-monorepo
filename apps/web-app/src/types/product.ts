// import { IntervalType, ProductType } from '@/lib/constants/products' // Likely unused now

/**
 * Product token response structure (existing, assumed to be aligned or separately handled)
 */
export interface ProductTokenResponse {
  id: string;
  product_id: string;
  network_id: string;
  token_id: string;
  token_name?: string;
  token_symbol?: string;
  token_decimals?: number;
  contract_address?: string;
  gas_token?: boolean;
  chain_id?: number;
  network_name?: string;
  network_type?: string;
  active: boolean;
  created_at: number;
  updated_at: number;
}

/**
 * Request payload for creating a product token (existing, assumed to be aligned or separately handled)
 */
export interface CreateProductTokenRequest {
  product_id: string;
  network_id: string;
  token_id: string;
  active: boolean;
}

/**
 * Request payload for product tokens when creating a new product
 * product_id is omitted as it will be set by the backend
 */
export interface CreateProductTokenWithoutIdRequest {
  network_id: string;
  token_id: string;
  active?: boolean;
}

/**
 * Request payload for updating a product token
 */
export interface UpdateProductTokenRequest {
  active: boolean;
}

/**
 * Response structure for product data
 */
export interface ProductResponse {
  id: string;
  object: string;
  workspace_id: string;
  wallet_id: string;
  name: string;
  description?: string;
  image_url?: string;
  url?: string;
  active: boolean;
  metadata?: Record<string, unknown> | null; // json.RawMessage can be null
  created_at: number;
  updated_at: number;
  product_tokens?: ProductTokenResponse[];
  // Embedded price fields
  product_type?: string;
  product_group?: string;
  price_type?: string;
  currency?: string;
  unit_amount_in_pennies?: number;
  interval_type?: string;
  term_length?: number;
  available_addons?: ProductAddonResponse[];
}

/**
 * Request payload for creating a product
 */
export interface CreateProductRequest {
  // Renamed from ProductRequest
  name: string;
  wallet_id: string;
  description?: string; // Was required, now optional to match Go (empty string vs omitempty)
  image_url?: string;
  url?: string;
  active: boolean;
  metadata?: Record<string, unknown> | null; // json.RawMessage can be null
  product_tokens?: CreateProductTokenWithoutIdRequest[];
  // Embedded price fields (now part of product)
  product_type?: string;
  product_group?: string;
  price_type: string;
  currency: string;
  unit_amount_in_pennies: number;
  interval_type?: string;
  term_length?: number;
  price_nickname?: string;
  price_external_id?: string;
  payment_provider?: string;
}

/**
 * Request payload for updating a product
 */
export interface UpdateProductRequest {
  name?: string;
  wallet_id?: string;
  description?: string;
  image_url?: string;
  url?: string;
  active?: boolean;
  metadata?: Record<string, unknown> | null; // json.RawMessage can be null
  product_tokens?: CreateProductTokenRequest[];
  // Note: Prices are not part of UpdateProductRequest in the provided Go struct
}

/**
 * Public Product Token Response from the API - MUST match backend ProductTokenResponse exactly
 */
export interface PublicProductTokenResponse {
  id: string;
  object: string;
  product_id: string;
  product_token_id: string;
  network_id: string;
  token_id: string;
  token_name?: string;
  token_symbol?: string;
  token_address?: string;
  contract_address?: string;
  token_decimals?: number;
  gas_token?: boolean;
  chain_id?: number;
  network_name?: string;
  network_type?: string;
  active: boolean;
  metadata?: Record<string, unknown>;
  created_at: number;
  updated_at: number;
}

/**
 * Parameters for getting a public product by price ID
 */
export interface GetPublicProductByPriceIdParams {
  priceId: string;
}

/**
 * Public Product Response from the API with embedded pricing
 */
export interface PublicProductResponse {
  id: string;
  account_id: string;
  workspace_id: string;
  wallet_address: string;
  name: string;
  description: string;
  image_url?: string;
  url?: string;
  product_tokens?: PublicProductTokenResponse[];
  smart_account_address?: string;
  smart_account_explorer_url?: string;
  smart_account_network?: string;
  // Embedded price fields
  product_type?: string;
  product_group?: string;
  price_type: string;
  currency: string;
  unit_amount_in_pennies: number;
  interval_type?: string;
  term_length?: number;
  available_addons?: ProductAddonResponse[];
}

/**
 * Publish product response
 */
export interface PublishProductResponse {
  message: string;
  cyphera_product_id: string;
  cyphera_product_token_id: string;
}

/**
 * Product addon relationship response
 */
export interface ProductAddonRelationshipResponse {
  id: string;
  object: string;
  base_product_id: string;
  addon_product_id: string;
  is_required: boolean;
  max_quantity?: number | null;
  min_quantity: number;
  display_order: number;
  metadata?: Record<string, unknown> | null;
  created_at: number;
  updated_at: number;
}

/**
 * Product addon response with full product details
 */
export interface ProductAddonResponse extends ProductAddonRelationshipResponse {
  addon_product: ProductResponse;
}

/**
 * Product with available addons
 */
export interface ProductWithAddonsResponse extends ProductResponse {
  available_addons?: ProductAddonResponse[];
}

/**
 * Request payload for creating a product addon relationship
 */
export interface CreateProductAddonRelationshipRequest {
  addon_product_id: string;
  is_required?: boolean;
  max_quantity?: number | null;
  min_quantity?: number;
  display_order?: number;
  metadata?: Record<string, unknown> | null;
}

/**
 * Request payload for updating a product addon relationship
 */
export interface UpdateProductAddonRelationshipRequest {
  is_required?: boolean;
  max_quantity?: number | null;
  min_quantity?: number;
  display_order?: number;
  metadata?: Record<string, unknown> | null;
}

/**
 * Request payload for bulk setting product addons
 */
export interface BulkSetProductAddonsRequest {
  addons: CreateProductAddonRelationshipRequest[];
}
