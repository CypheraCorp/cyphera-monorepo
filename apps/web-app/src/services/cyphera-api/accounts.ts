import { CypheraAPI, UserRequestContext } from './api';
import type { AccountMessageResponse, AccountOnboardingRequest } from '@/types/account';
import { logger } from '@/lib/core/logger/logger-utils';
/**
 * Accounts API class for handling account-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class AccountsAPI extends CypheraAPI {
  /**
   * Onboards a new account
   * @param context - The user request context (token, IDs)
   * @param accountData - The account data to onboard
   * @returns Promise with the onboarded account response
   * @throws Error if the request fails
   */
  async onboardAccount(
    context: UserRequestContext,
    accountData: Partial<AccountOnboardingRequest>
  ): Promise<AccountMessageResponse> {
    try {
      const url = `${this.baseUrl}/accounts/onboard`;
      const headers = this.getHeaders(context);
      const body = JSON.stringify(accountData);

      return await this.fetchWithRateLimit<AccountMessageResponse>(url, {
        method: 'POST',
        headers: headers,
        body: body,
      });
    } catch (error) {
      logger.error('❌ AccountsAPI: Account onboarding failed:', error);
      throw error;
    }
  }
}
