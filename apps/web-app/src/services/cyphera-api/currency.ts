import { CypheraAPI, UserRequestContext } from './api';
import type { Currency } from '@/types/analytics';

interface CurrencyListResponse {
  currencies: Currency[];
}

class CurrencyService extends CypheraAPI {
  private userContext: UserRequestContext | null = null;

  /**
   * Set user context for authenticated requests
   */
  setUserContext(context: UserRequestContext) {
    this.userContext = context;
  }

  /**
   * Get list of supported currencies
   */
  async getCurrencies(workspaceId: string): Promise<Currency[]> {
    if (!this.userContext) throw new Error('User context not set');

    const url = `${this.baseUrl}/currencies`;
    console.log('DEBUG: Currencies URL:', url);
    const response = await this.fetchWithRateLimit<CurrencyListResponse>(
      url,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response.currencies;
  }

  /**
   * Get default currency for workspace
   */
  async getDefaultCurrency(workspaceId: string): Promise<Currency | null> {
    const currencies = await this.getCurrencies(workspaceId);
    return currencies.find(c => c.is_default) || null;
  }

  /**
   * Update default currency for workspace
   */
  async setDefaultCurrency(workspaceId: string, currencyCode: string): Promise<void> {
    if (!this.userContext) throw new Error('User context not set');

    await this.fetchWithRateLimit<void>(
      `${this.baseUrl}/workspaces/current/currency-settings`,
      {
        method: 'PUT',
        headers: this.getHeaders(this.userContext),
        body: JSON.stringify({
          currency_code: currencyCode,
        }),
      }
    );
  }
}

export const currencyService = new CurrencyService();