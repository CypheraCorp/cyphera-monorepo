import type { PublicProductResponse, PriceResponse } from '@/types/product';
import { CypheraAPI } from './api';
import type { AccountAccessResponse, AccountRequest } from '@/types/account';
import type { CustomerSignInRequest, CustomerSignInResponse } from '@/types/customer';
import { clientLogger } from '@/lib/core/logger/logger-client';

/**
 * Public API class for handling public API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class PublicAPI extends CypheraAPI {
  /**
   * Gets public price information by price ID
   * @param priceId - The ID of the price to fetch
   * @returns Promise with the price response
   * @throws Error if the request fails
   */
  async getPublicPrice(priceId: string): Promise<PriceResponse> {
    try {
      return await this.fetchWithRateLimit<PriceResponse>(`${this.baseUrl}/public/prices/${priceId}`, {
        method: 'GET',
        headers: this.getPublicHeaders(),
        // Removed cache: 'no-store' to allow HTTP caching
      });
    } catch (error) {
      clientLogger.error('Public price fetch failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Get a public product by Price ID
   * Uses the public API key.
   * @param priceId - The ID of the price to get
   * @returns Promise<PublicProductResponse>
   * @throws Error if the API call fails
   */
  async getPublicProductByPriceId(priceId: string): Promise<PublicProductResponse> {
    try {
      return await this.fetchWithRateLimit<PublicProductResponse>(`${this.baseUrl}/admin/prices/${priceId}`, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });
    } catch (error) {
      clientLogger.error('Public product fetch failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Creates/registers an account in the Cyphera system.
   * Requires system admin API Key access.
   * @param accountData - The account creation data (including optional wallet data)
   * @returns Promise<AccountAccessResponse>
   * @throws Error if the API call fails
   */
  async signInOrRegister(accountData: AccountRequest): Promise<AccountAccessResponse> {
    try {
      // Log wallet data being sent to backend if present
      if (accountData.wallet_data) {
        clientLogger.info('Sending Web3Auth wallet data to backend', {
          wallet_type: accountData.wallet_data.wallet_type,
          wallet_address: accountData.wallet_data.wallet_address,
          network_type: accountData.wallet_data.network_type,
          nickname: accountData.wallet_data.nickname,
        });
      }

      return await this.fetchWithRateLimit<AccountAccessResponse>(`${this.baseUrl}/admin/accounts/signin`, {
        method: 'POST',
        headers: this.getPublicHeaders(),
        body: JSON.stringify(accountData),
      });
    } catch (error) {
      clientLogger.error('Account signin failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }

  /**
   * Customer signin/register endpoint for Web3Auth integration
   * @param customerData - The customer signin data
   * @param workspaceId - Optional workspace ID to associate the customer with
   * @returns Promise<CustomerSignInResponse>
   * @throws Error if the API call fails
   */
  async customerSignInOrRegister(
    customerData: CustomerSignInRequest,
    workspaceId?: string
  ): Promise<CustomerSignInResponse> {
    try {
      // Log customer data being sent to backend
      clientLogger.debug('Sending customer signin data to backend', {
        email: customerData.email,
        name: customerData.name,
        web3auth_id: customerData.metadata.web3auth_id,
        wallet_address: customerData.wallet_data?.wallet_address,
        network_type: customerData.wallet_data?.network_type,
      });

      const headers = this.getPublicHeaders();

      // Add workspace ID header if provided
      if (workspaceId) {
        headers['X-Workspace-ID'] = workspaceId;
      }

      return await this.fetchWithRateLimit<CustomerSignInResponse>(`${this.baseUrl}/admin/customers/signin`, {
        method: 'POST',
        headers,
        body: JSON.stringify(customerData),
      });
    } catch (error) {
      clientLogger.error('Customer signin failed', {
        error: error instanceof Error ? error.message : error,
      });
      throw error;
    }
  }
}
