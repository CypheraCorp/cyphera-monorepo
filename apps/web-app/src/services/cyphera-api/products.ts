import { PaginatedResponse, PaginationParams } from '@/types/common';
import { CypheraAPI, UserRequestContext } from './api';
import type {
  ProductResponse,
  UpdateProductRequest,
  UpdateProductTokenRequest,
  ProductTokenResponse,
  CreateProductRequest,
  ProductAddonResponse,
  CreateProductAddonRelationshipRequest,
  UpdateProductAddonRelationshipRequest,
  BulkSetProductAddonsRequest,
  ProductWithAddonsResponse,
} from '@/types/product';
import { clientLogger } from '@/lib/core/logger/logger-client';

/**
 * Products API class for handling product-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class ProductsAPI extends CypheraAPI {
  /**
   * Gets products for the current workspace with pagination
   * @param context - The user request context (token, IDs)
   * @param params - Pagination parameters
   * @returns Promise with the products response and pagination metadata
   * @throws Error if the request fails
   */
  async getProducts(
    context: UserRequestContext,
    params?: PaginationParams
  ): Promise<PaginatedResponse<ProductResponse>> {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.page) queryParams.append('page', params.page.toString());
    const url = `${this.baseUrl}/products?${queryParams.toString()}`;

    try {
      return await this.fetchWithRateLimit<PaginatedResponse<ProductResponse>>(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      clientLogger.error('Products fetch failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Gets a single product by ID
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the product to fetch
   * @returns Promise with the product response
   * @throws Error if the request fails
   */
  async getProductById(context: UserRequestContext, productId: string): Promise<ProductResponse> {
    try {
      return await this.fetchWithRateLimit<ProductResponse>(`${this.baseUrl}/products/${productId}`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      clientLogger.error('Product fetch failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Creates a new product
   * @param context - The user request context (token, IDs)
   * @param productData - The product data to create
   * @returns Promise with the created product response
   * @throws Error if the request fails
   */
  async createProduct(
    context: UserRequestContext,
    productData: CreateProductRequest
  ): Promise<ProductResponse> {
    if (!productData.wallet_id) {
      throw new Error('Wallet ID is required to create a product');
    }
    try {
      return await this.fetchWithRateLimit<ProductResponse>(`${this.baseUrl}/products`, {
        method: 'POST',
        headers: this.getHeaders(context),
        body: JSON.stringify(productData),
      });
    } catch (error) {
      clientLogger.error('Product creation failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Updates a product
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the product to update
   * @param productData - The product data to update
   * @returns Promise with the updated product response
   * @throws Error if the request fails
   */
  async updateProduct(
    context: UserRequestContext,
    productId: string,
    productData: UpdateProductRequest
  ): Promise<ProductResponse> {
    try {
      return await this.fetchWithRateLimit<ProductResponse>(`${this.baseUrl}/products/${productId}`, {
        method: 'PUT',
        headers: this.getHeaders(context),
        body: JSON.stringify(productData),
      });
    } catch (error) {
      clientLogger.error('Product update failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Updates a product token
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the product
   * @param networkId - The ID of the network
   * @param tokenId - The ID of the token
   * @param tokenData - The token data to update
   * @returns Promise with the updated product token response
   * @throws Error if the request fails
   */
  async updateProductToken(
    context: UserRequestContext,
    productId: string,
    networkId: string,
    tokenId: string,
    tokenData: UpdateProductTokenRequest
  ): Promise<ProductTokenResponse> {
    try {
      return await this.fetchWithRateLimit<ProductTokenResponse>(
        `${this.baseUrl}/products/${productId}/networks/${networkId}/tokens/${tokenId}`,
        {
          method: 'PUT',
          headers: this.getHeaders(context),
          body: JSON.stringify(tokenData),
        }
      );
    } catch (error) {
      clientLogger.error('Product token update failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Deletes a product
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the product to delete
   * @returns Promise<void>
   * @throws Error if the request fails
   */
  async deleteProduct(context: UserRequestContext, productId: string): Promise<void> {
    try {
      await this.fetchWithRateLimit<void>(`${this.baseUrl}/products/${productId}`, {
        method: 'DELETE',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      clientLogger.error('Product deletion failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Gets a product token by ID
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the product
   * @param networkId - The ID of the network
   * @param tokenId - The ID of the token
   * @returns Promise with the product token response
   * @throws Error if the request fails
   */
  async getProductTokenById(
    context: UserRequestContext,
    productId: string,
    networkId: string,
    tokenId: string
  ): Promise<ProductTokenResponse> {
    try {
      return await this.fetchWithRateLimit<ProductTokenResponse>(
        `${this.baseUrl}/products/${productId}/networks/${networkId}/tokens/${tokenId}`,
        {
          method: 'GET',
          headers: this.getHeaders(context),
        }
      );
    } catch (error) {
      clientLogger.error('Product token fetch failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Deletes a product token
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the product
   * @param networkId - The ID of the network
   * @param tokenId - The ID of the token
   * @returns Promise<void>
   * @throws Error if the request fails
   */
  async deleteProductToken(
    context: UserRequestContext,
    productId: string,
    networkId: string,
    tokenId: string
  ): Promise<void> {
    try {
      await this.fetchWithRateLimit<void>(
        `${this.baseUrl}/products/${productId}/networks/${networkId}/tokens/${tokenId}`,
        {
          method: 'DELETE',
          headers: this.getHeaders(context),
        }
      );
    } catch (error) {
      clientLogger.error('Product token deletion failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Gets a product with its available addons
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the product
   * @returns Promise with the product and its addons
   * @throws Error if the request fails
   */
  async getProductWithAddons(
    context: UserRequestContext,
    productId: string
  ): Promise<ProductWithAddonsResponse> {
    try {
      return await this.fetchWithRateLimit<ProductWithAddonsResponse>(
        `${this.baseUrl}/products/${productId}/with-addons`,
        {
          method: 'GET',
          headers: this.getHeaders(context),
        }
      );
    } catch (error) {
      clientLogger.error('Product with addons fetch failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Gets addons for a product
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the base product
   * @returns Promise with the product addons
   * @throws Error if the request fails
   */
  async getProductAddons(
    context: UserRequestContext,
    productId: string
  ): Promise<ProductAddonResponse[]> {
    try {
      const response = await this.fetchWithRateLimit<{ data: ProductAddonResponse[] }>(
        `${this.baseUrl}/products/${productId}/addons`,
        {
          method: 'GET',
          headers: this.getHeaders(context),
        }
      );
      return response.data;
    } catch (error) {
      clientLogger.error('Product addons fetch failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Creates a product addon relationship
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the base product
   * @param addonData - The addon relationship data
   * @returns Promise with the created addon relationship
   * @throws Error if the request fails
   */
  async createProductAddon(
    context: UserRequestContext,
    productId: string,
    addonData: CreateProductAddonRelationshipRequest
  ): Promise<ProductAddonResponse> {
    try {
      return await this.fetchWithRateLimit<ProductAddonResponse>(
        `${this.baseUrl}/products/${productId}/addons`,
        {
          method: 'POST',
          headers: this.getHeaders(context),
          body: JSON.stringify(addonData),
        }
      );
    } catch (error) {
      clientLogger.error('Product addon creation failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Updates a product addon relationship
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the base product
   * @param addonProductId - The ID of the addon product
   * @param addonData - The addon relationship data to update
   * @returns Promise with the updated addon relationship
   * @throws Error if the request fails
   */
  async updateProductAddon(
    context: UserRequestContext,
    productId: string,
    addonProductId: string,
    addonData: UpdateProductAddonRelationshipRequest
  ): Promise<ProductAddonResponse> {
    try {
      return await this.fetchWithRateLimit<ProductAddonResponse>(
        `${this.baseUrl}/products/${productId}/addons/${addonProductId}`,
        {
          method: 'PUT',
          headers: this.getHeaders(context),
          body: JSON.stringify(addonData),
        }
      );
    } catch (error) {
      clientLogger.error('Product addon update failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Deletes a product addon relationship
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the base product
   * @param addonProductId - The ID of the addon product
   * @returns Promise<void>
   * @throws Error if the request fails
   */
  async deleteProductAddon(
    context: UserRequestContext,
    productId: string,
    addonProductId: string
  ): Promise<void> {
    try {
      await this.fetchWithRateLimit<void>(
        `${this.baseUrl}/products/${productId}/addons/${addonProductId}`,
        {
          method: 'DELETE',
          headers: this.getHeaders(context),
        }
      );
    } catch (error) {
      clientLogger.error('Product addon deletion failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Bulk sets all addons for a product (replaces existing)
   * @param context - The user request context (token, IDs)
   * @param productId - The ID of the base product
   * @param addonsData - The addon relationships to set
   * @returns Promise<void>
   * @throws Error if the request fails
   */
  async bulkSetProductAddons(
    context: UserRequestContext,
    productId: string,
    addonsData: BulkSetProductAddonsRequest
  ): Promise<void> {
    try {
      await this.fetchWithRateLimit<void>(
        `${this.baseUrl}/products/${productId}/addons/bulk`,
        {
          method: 'PUT',
          headers: this.getHeaders(context),
          body: JSON.stringify(addonsData),
        }
      );
    } catch (error) {
      clientLogger.error('Product addons bulk set failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }
}
