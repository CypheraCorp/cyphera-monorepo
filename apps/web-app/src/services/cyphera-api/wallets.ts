import { CypheraAPI, UserRequestContext } from './api';
import type {
  WalletResponse,
  WalletListResponse,
  CreateWalletRequest,
  UpdateWalletRequest,
} from '@/types/wallet';
import { logger } from '@/lib/core/logger/logger-utils';

/**
 * Wallets API class for handling wallet-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class WalletsAPI extends CypheraAPI {
  /**
   * Gets all wallets for the authenticated account
   * including Circle wallets with their specific data
   * @param context - The user request context (token, IDs)
   * @returns Promise with an array of wallets
   * @throws Error if the request fails
   */
  async listWallets(context: UserRequestContext): Promise<WalletResponse[]> {
    try {
      const result = await this.fetchWithRateLimit<WalletListResponse>(`${this.baseUrl}/wallets?include_circle_data=true`, {
        method: 'GET',
        headers: this.getHeaders(context),
        // Removed cache: 'no-store' to allow HTTP caching
      });
      return result.data;
    } catch (error) {
      logger.error_sync('Wallets fetch failed:', error);
      throw error;
    }
  }

  /**
   * Gets a wallet by ID, including Circle wallet data if available
   * @param context - The user request context (token, IDs)
   * @param walletId - The ID of the wallet to retrieve
   * @returns Promise with the wallet response
   * @throws Error if the request fails
   */
  async getWallet(context: UserRequestContext, walletId: string): Promise<WalletResponse> {
    try {
      return await this.fetchWithRateLimit<WalletResponse>(`${this.baseUrl}/wallets/${walletId}?include_circle_data=true`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error_sync('Wallet fetch failed:', error);
      throw error;
    }
  }

  /**
   * Creates a new wallet
   * @param context - The user request context (token, IDs)
   * @param walletData - The wallet data to create
   * @returns Promise with the created wallet response
   * @throws Error if the request fails
   */
  async createWallet(
    context: UserRequestContext,
    walletData: CreateWalletRequest
  ): Promise<WalletResponse> {
    try {
      return await this.fetchWithRateLimit<WalletResponse>(`${this.baseUrl}/wallets`, {
        method: 'POST',
        headers: this.getHeaders(context),
        body: JSON.stringify(walletData),
      });
    } catch (error) {
      logger.error_sync('Wallet creation failed:', error);
      throw error;
    }
  }

  /**
   * Updates an existing wallet
   * @param context - The user request context (token, IDs)
   * @param walletId - The ID of the wallet to update
   * @param walletData - The updated wallet data
   * @returns Promise with the updated wallet response
   * @throws Error if the request fails
   */
  async updateWallet(
    context: UserRequestContext,
    walletId: string,
    walletData: UpdateWalletRequest
  ): Promise<WalletResponse> {
    try {
      return await this.fetchWithRateLimit<WalletResponse>(`${this.baseUrl}/wallets/${walletId}`, {
        method: 'PUT',
        headers: this.getHeaders(context),
        body: JSON.stringify(walletData),
      });
    } catch (error) {
      logger.error_sync('Wallet update failed:', error);
      throw error;
    }
  }

  /**
   * Deletes a wallet
   * @param context - The user request context (token, IDs)
   * @param walletId - The ID of the wallet to delete
   * @returns Promise with success message
   * @throws Error if the request fails
   */
  async deleteWallet(context: UserRequestContext, walletId: string): Promise<void> {
    try {
      await this.fetchWithRateLimit<void>(`${this.baseUrl}/wallets/${walletId}`, {
        method: 'DELETE',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error_sync('Wallet deletion failed:', error);
      throw error;
    }
  }
}
