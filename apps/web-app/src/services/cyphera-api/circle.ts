import {
  CircleChallengeResponse,
  CircleCreateChallengeResponse,
  CircleCreateWalletsResponse,
  CircleRequestWithIdempotencyKeyAndToken,
  CircleTransaction,
  CircleTransactionListResponse,
  CircleUserData,
  CircleUserInitResponse,
  CircleUserResponse,
  CircleUserTokenResponse,
  CircleWalletBalanceResponse,
  CircleWalletListResponse,
  CircleWalletResponse,
  CreateWalletsRequest,
  InitializeUserRequest,
} from '@/types/circle';
import { CreateUserWithPinAuthRequest } from '@/types/circle';
import { CypheraAPI, UserRequestContext } from './api';
import { logger } from '@/lib/core/logger/logger-utils';
/**
 * CircleAPI class for handling Circle wallet-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 * Uses Public Headers (Admin API Key) for backend communication.
 */
export class CircleAPI extends CypheraAPI {
  // User management

  /**
   * Creates a Circle user via the backend proxy.
   * Requires workspaceId in the path.
   */
  async createUser(
    workspaceId: string,
    request: CreateUserWithPinAuthRequest
  ): Promise<CircleUserData> {
    if (!workspaceId) throw new Error('Workspace ID is required');
    try {
      const headers = this.getPublicHeaders();
      const url = `${this.baseUrl}/admin/circle/users/${workspaceId}`;

      // Use workspaceId in path, use public headers for admin API
      // Assuming backend returns CircleUserData directly or within a data object
      // Adjust this based on actual backend response structure if necessary
      const result = await this.fetchWithRateLimit<{ data: CircleUserData } | CircleUserData>(url, {
        method: 'POST',
        headers,
        body: JSON.stringify(request),
      });
      return 'data' in result ? result.data : result;
    } catch (error) {
      logger.error('Failed to create Circle user:', error);
      throw error;
    }
  }

  /**
   * Gets the user's details from Circle via backend user token endpoint.
   * Requires workspaceId in the path. Uses public headers for the call to *our* backend.
   */
  async getUserByToken(context: UserRequestContext): Promise<CircleUserResponse> {
    if (!context.workspace_id) throw new Error('Workspace ID is required in context');
    try {
      // Use workspaceId in path, use public headers for admin API
      return await this.fetchWithRateLimit<CircleUserResponse>(
        `${this.baseUrl}/admin/circle/users/${context.workspace_id}/token`,
        {
          method: 'GET',
          headers: this.getPublicHeaders(),
        }
      );
    } catch (error) {
      logger.error('Failed to get user by token:', error);
      throw error;
    }
  }

  /**
   * Gets a specific user by ID.
   * Uses public headers for admin API.
   */
  async getUserById(userId: string): Promise<CircleUserResponse> {
    try {
      return await this.fetchWithRateLimit<CircleUserResponse>(`${this.baseUrl}/admin/circle/users/${userId}`, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });
    } catch (error) {
      logger.error('Failed to get user by ID:', error);
      throw error;
    }
  }

  /**
   * Creates a token for a Circle user.
   * Requires workspaceId in the path. Uses public headers for admin API.
   */
  async createUserToken(
    workspaceId: string,
    request: CreateUserWithPinAuthRequest
  ): Promise<CircleUserTokenResponse> {
    if (!workspaceId) throw new Error('Workspace ID is required');
    try {
      return await this.fetchWithRateLimit<CircleUserTokenResponse>(`${this.baseUrl}/admin/circle/users/${workspaceId}/token`, {
        method: 'POST',
        headers: this.getPublicHeaders(),
        body: JSON.stringify(request),
      });
    } catch (error) {
      logger.error('Failed to create user token:', error);
      throw error;
    }
  }

  /**
   * Initializes a Circle user.
   * Requires workspaceId in the path. Uses public headers for the call to *our* backend.
   */
  async initializeUser(
    context: UserRequestContext,
    request: InitializeUserRequest
  ): Promise<CircleUserInitResponse> {
    if (!context.workspace_id) throw new Error('Workspace ID is required in context');
    try {
      // Use workspaceId in path, use public headers
      return await this.fetchWithRateLimit<CircleUserInitResponse>(
        `${this.baseUrl}/admin/circle/users/${context.workspace_id}/initialize`,
        {
          method: 'POST',
          headers: this.getPublicHeaders(),

          body: JSON.stringify(request),
        }
      );
    } catch (error) {
      logger.error('Failed to initialize user:', error);
      throw error;
    }
  }

  // Challenge management

  /**
   * Gets a Circle challenge by ID.
   * Requires workspaceId in the path. Uses public headers for the call to *our* backend.
   */
  async getChallenge(
    context: UserRequestContext,
    challengeId: string
  ): Promise<CircleChallengeResponse> {
    if (!context.workspace_id) throw new Error('Workspace ID is required in context');
    try {
      // Use workspaceId in path, use public headers
      return await this.fetchWithRateLimit<CircleChallengeResponse>(
        `${this.baseUrl}/circle/${context.workspace_id}/challenges/${challengeId}`,
        {
          method: 'GET',
          headers: this.getPublicHeaders(),
        }
      );
    } catch (error) {
      logger.error('Failed to get challenge:', error);
      throw error;
    }
  }

  /**
   * Creates a PIN challenge for setting up a new PIN.
   * Requires workspaceId in the path. Uses public headers for the call to *our* backend.
   */
  async createPinChallenge(
    workspaceId: string,
    request: CircleRequestWithIdempotencyKeyAndToken
  ): Promise<CircleCreateChallengeResponse> {
    if (!workspaceId) throw new Error('Workspace ID is required in context');
    try {
      // Use workspaceId in path, use public headers
      return await this.fetchWithRateLimit<CircleCreateChallengeResponse>(`${this.baseUrl}/admin/circle/users/${workspaceId}/pin/create`, {
        method: 'POST',
        // Backend route requires user_token header, which *should* be in getHeaders
        // BUT user asked for public headers only. This might fail if backend strictly enforces user_token header.
        headers: this.getPublicHeaders(),
        // Go type RequestWithIdempotencyKey includes user_token in body
        body: JSON.stringify(request),
      });
    } catch (error) {
      logger.error('Failed to create PIN challenge:', error);
      throw error;
    }
  }

  /**
   * Creates a PIN restore challenge.
   * Requires workspaceId in the path. Uses public headers for the call to *our* backend.
   */
  async createPinRestoreChallenge(
    context: UserRequestContext,
    idempotencyKey: string
  ): Promise<CircleChallengeResponse> {
    // Assuming this still returns full challenge
    if (!context.workspace_id) throw new Error('Workspace ID is required in context');
    try {
      // Use workspaceId in path, use public headers
      return await this.fetchWithRateLimit<CircleChallengeResponse>(
        `${this.baseUrl}/admin/circle/users/${context.workspace_id}/pin/restore`,
        {
          method: 'POST',
          // Backend route requires user_token header, which *should* be in getHeaders
          // BUT user asked for public headers only. This might fail if backend strictly enforces user_token header.
          headers: {
            ...this.getPublicHeaders(),
            'X-Idempotency-Key': idempotencyKey,
          },
          // Go type RequestWithIdempotencyKey includes user_token in body
          body: JSON.stringify({
            idempotency_key: idempotencyKey,
            workspace_id: context.workspace_id, // Use token from context here
          }),
        }
      );
    } catch (error) {
      logger.error('Failed to create PIN restore challenge:', error);
      throw error;
    }
  }

  /**
   * Updates a PIN via a challenge.
   * Requires workspaceId in the path. Uses public headers for the call to *our* backend.
   */
  async updatePinChallenge(
    context: UserRequestContext,
    idempotencyKey: string
  ): Promise<CircleChallengeResponse> {
    // Assuming this still returns full challenge
    if (!context.workspace_id) throw new Error('Workspace ID is required in context');
    try {
      // Use workspaceId in path, use public headers
      return await this.fetchWithRateLimit<CircleChallengeResponse>(
        `${this.baseUrl}/admin/circle/users/${context.workspace_id}/pin/update`,
        {
          method: 'PUT',
          // Backend route requires user_token header, which *should* be in getHeaders
          // BUT user asked for public headers only. This might fail if backend strictly enforces user_token header.
          headers: {
            ...this.getPublicHeaders(),
            'X-Idempotency-Key': idempotencyKey,
          },
          // Go type RequestWithIdempotencyKey includes user_token in body
          body: JSON.stringify({
            idempotency_key: idempotencyKey,
            workspace_id: context.workspace_id, // Use token from context here
          }),
        }
      );
    } catch (error) {
      logger.error('Failed to update PIN challenge:', error);
      throw error;
    }
  }

  // Wallet management

  /**
   * Creates Circle wallets.
   * Requires workspaceId in the path. Uses public headers for the call to *our* backend.
   */
  async createWallets(
    workspace_id: string,
    params: CreateWalletsRequest
  ): Promise<CircleCreateWalletsResponse> {
    if (!workspace_id) throw new Error('Workspace ID is required in context');
    try {
      // Use workspaceId in path, use public headers for admin API
      return await this.fetchWithRateLimit<CircleCreateWalletsResponse>(`${this.baseUrl}/admin/circle/wallets/${workspace_id}`, {
        method: 'POST',
        headers: this.getPublicHeaders(),
        body: JSON.stringify(params), // Send the params object
      });
    } catch (error) {
      logger.error('Failed to create wallets:', error);
      throw error;
    }
  }

  /**
   * Lists Circle wallets.
   * Requires workspaceId in the path. Uses public headers for the call to *our* backend.
   */
  async listCircleWallets(
    context: UserRequestContext,
    params?: {
      address?: string;
      blockchain?: string;
      pageSize?: number; // Mapped to page_size
      pageBefore?: string; // Mapped to page_before
      pageAfter?: string; // Mapped to page_after
    }
  ): Promise<CircleWalletListResponse> {
    if (!context.workspace_id) throw new Error('Workspace ID is required in context');
    try {
      const queryParams = new URLSearchParams();
      if (params?.address) queryParams.append('address', params.address);
      if (params?.blockchain) queryParams.append('blockchain', params.blockchain);
      // Map frontend names to backend names
      if (params?.pageSize) queryParams.append('page_size', params.pageSize.toString());
      if (params?.pageBefore) queryParams.append('page_before', params.pageBefore);
      if (params?.pageAfter) queryParams.append('page_after', params.pageAfter);

      const queryString = queryParams.toString();
      const url = `${this.baseUrl}/admin/circle/wallets/${context.workspace_id}${queryString ? `?${queryString}` : ''}`;

      // Use workspaceId in path, use public headers for admin API
      return await this.fetchWithRateLimit<CircleWalletListResponse>(url, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });
    } catch (error) {
      logger.error('Failed to list Circle wallets:', error);
      throw error;
    }
  }

  /**
   * Gets a specific Circle wallet.
   * Uses public headers for the call to *our* backend.
   */
  async getCircleWallet(
    context: UserRequestContext, // Keep context to potentially get user_token if needed by header
    walletId: string
  ): Promise<CircleWalletResponse> {
    try {
      return await this.fetchWithRateLimit<CircleWalletResponse>(`${this.baseUrl}/admin/circle/wallets/${walletId}`, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });
    } catch (error) {
      logger.error('Failed to get Circle wallet:', error);
      throw error;
    }
  }

  /**
   * Gets balances for a specific Circle wallet.
   * Uses public headers for the call to *our* backend.
   */
  async getWalletBalance(
    context: UserRequestContext, // Keep context to potentially get user_token if needed by header
    walletId: string,
    params?: {
      include_all?: boolean; // Use backend query param name
      name?: string;
      token_address?: string; // Use backend query param name
      standard?: string;
      page_size?: number; // Use backend query param name
      page_before?: string; // Use backend query param name
      page_after?: string; // Use backend query param name
    }
  ): Promise<CircleWalletBalanceResponse> {
    try {
      const queryParams = new URLSearchParams();
      // Use backend query param names
      if (params?.include_all !== undefined)
        queryParams.append('include_all', String(params.include_all));
      if (params?.name) queryParams.append('name', params.name);
      if (params?.token_address) queryParams.append('token_address', params.token_address);
      if (params?.standard) queryParams.append('standard', params.standard);
      if (params?.page_size) queryParams.append('page_size', params.page_size.toString());
      if (params?.page_before) queryParams.append('page_before', params.page_before);
      if (params?.page_after) queryParams.append('page_after', params.page_after);

      const queryString = queryParams.toString();
      const url = `${this.baseUrl}/admin/circle/wallets/${walletId}/balances${queryString ? `?${queryString}` : ''}`;

      return await this.fetchWithRateLimit<CircleWalletBalanceResponse>(url, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });
    } catch (error) {
      logger.error('Failed to get wallet balances:', error);
      throw error;
    }
  }

  /**
   * Lists transactions.
   * Uses public headers for the call to *our* backend.
   */
  async listTransactions(
    context: UserRequestContext, // Keep context to potentially get user_token if needed by header
    params?: {
      blockchain?: string;
      destination_address?: string; // Use backend query param name
      include_all?: boolean; // Use backend query param name
      operation?: string;
      state?: string;
      tx_hash?: string; // Use backend query param name
      tx_type?: string; // Use backend query param name
      user_id?: string; // Use backend query param name
      wallet_ids?: string; // Use backend query param name (comma separated)
      from?: string;
      to?: string;
      page_size?: number; // Use backend query param name
      page_before?: string; // Use backend query param name
      page_after?: string; // Use backend query param name
    }
  ): Promise<CircleTransactionListResponse> {
    try {
      const queryParams = new URLSearchParams();
      // Use backend query param names
      if (params?.blockchain) queryParams.append('blockchain', params.blockchain);
      if (params?.destination_address)
        queryParams.append('destination_address', params.destination_address);
      if (params?.include_all !== undefined)
        queryParams.append('include_all', String(params.include_all));
      if (params?.operation) queryParams.append('operation', params.operation);
      if (params?.state) queryParams.append('state', params.state);
      if (params?.tx_hash) queryParams.append('tx_hash', params.tx_hash);
      if (params?.tx_type) queryParams.append('tx_type', params.tx_type);
      if (params?.user_id) queryParams.append('user_id', params.user_id);
      if (params?.wallet_ids) queryParams.append('wallet_ids', params.wallet_ids);
      if (params?.from) queryParams.append('from', params.from);
      if (params?.to) queryParams.append('to', params.to);
      if (params?.page_size) queryParams.append('page_size', params.page_size.toString());
      if (params?.page_before) queryParams.append('page_before', params.page_before);
      if (params?.page_after) queryParams.append('page_after', params.page_after);

      const queryString = queryParams.toString();
      const url = `${this.baseUrl}/admin/circle/transactions${queryString ? `?${queryString}` : ''}`;

      return await this.fetchWithRateLimit<CircleTransactionListResponse>(url, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });
    } catch (error) {
      logger.error('Failed to list transactions:', error);
      throw error;
    }
  }

  /**
   * Gets a specific transaction.
   * Uses public headers for the call to *our* backend.
   */
  async getTransaction(
    context: UserRequestContext, // Keep context to potentially get user_token if needed by header
    transactionId: string
  ): Promise<{ data: { transaction: CircleTransaction } }> {
    try {
      return await this.fetchWithRateLimit<{ data: { transaction: CircleTransaction } }>(`${this.baseUrl}/admin/circle/transactions/${transactionId}`, {
        method: 'GET',
        headers: this.getPublicHeaders(),
      });
    } catch (error) {
      logger.error('Failed to get transaction:', error);
      throw error;
    }
  }
  /**
   * Estimates fees for a transfer transaction.
   * Uses public headers for the call to *our* backend.
   */
  async estimateFee(params: {
    wallet_id: string;
    destination_address: string;
    token_id?: string;
    amount: string;
    blockchain: string;
  }): Promise<{
    low: { amount: string; amountInUSD: string };
    medium: { amount: string; amountInUSD: string };
    high: { amount: string; amountInUSD: string };
  }> {
    try {
      // Adjust based on actual backend response structure
      type FeeEstimateResponse = {
        low: { amount: string; amountInUSD: string };
        medium: { amount: string; amountInUSD: string };
        high: { amount: string; amountInUSD: string };
      };
      const result = await this.fetchWithRateLimit<{ data: FeeEstimateResponse } | FeeEstimateResponse>(
        `${this.baseUrl}/admin/circle/transactions/transfer/estimate-fee`,
        {
          method: 'POST',
          headers: this.getPublicHeaders(), // Use public headers
          body: JSON.stringify(params),
        }
      );
      return 'data' in result ? result.data : result;
    } catch (error) {
      logger.error('Failed to estimate fee:', error);
      throw error;
    }
  }

  /**
   * Validates an address.
   * Uses public headers.
   */
  async validateAddress(address: string, blockchain: string): Promise<{ isValid: boolean }> {
    // Request body type if needed
    const requestBody = { address, blockchain };
    try {
      // Adjust based on actual backend response structure
      const result = await this.fetchWithRateLimit<
        { data: { isValid: boolean } } | { isValid: boolean }
      >(`${this.baseUrl}/admin/circle/transactions/validate-address`, {
        method: 'POST',
        headers: this.getPublicHeaders(),
        body: JSON.stringify(requestBody),
      });
      return 'data' in result ? result.data : result;
    } catch (error) {
      logger.error('Failed to validate address:', error);
      throw error;
    }
  }
}
