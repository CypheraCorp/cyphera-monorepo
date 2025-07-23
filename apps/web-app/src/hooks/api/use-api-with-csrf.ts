import { useCallback } from 'react';
import { useCSRF } from '@/hooks/security/use-csrf';
import { UserRequestContext } from '@/services/cyphera-api/api';
import { ProductsAPI } from '@/services/cyphera-api/products';
import { CustomersAPI } from '@/services/cyphera-api/customers';
import { SubscriptionsAPI } from '@/services/cyphera-api/subscriptions';
import { TransactionsAPI } from '@/services/cyphera-api/transactions';
import { WalletsAPI } from '@/services/cyphera-api/wallets';

/**
 * Hook that provides API instances with CSRF protection
 * Automatically includes CSRF tokens in all requests
 */
export function useAPIWithCSRF() {
  const { csrfToken } = useCSRF();

  // Create API instances
  const productsAPI = new ProductsAPI();
  const customersAPI = new CustomersAPI();
  const subscriptionsAPI = new SubscriptionsAPI();
  const transactionsAPI = new TransactionsAPI();
  const walletsAPI = new WalletsAPI();

  /**
   * Wrap API method to include CSRF token
   */
  const wrapAPIMethod = useCallback(
    <T extends any[], R>(
      apiMethod: (context: UserRequestContext, ...args: T) => Promise<R>,
      apiInstance: any
    ) => {
      return async (context: UserRequestContext, ...args: T): Promise<R> => {
        // Create a new context with CSRF token
        const contextWithCSRF = {
          ...context,
          csrfToken: csrfToken || undefined,
        };
        return apiMethod.call(apiInstance, contextWithCSRF, ...args);
      };
    },
    [csrfToken]
  );

  /**
   * Create wrapped API instance with CSRF support
   */
  const createWrappedAPI = useCallback(
    <T extends object>(api: T): T => {
      const wrapped = {} as T;
      
      Object.keys(api).forEach((key) => {
        const prop = api[key as keyof T];
        if (typeof prop === 'function') {
          // Wrap the method to include CSRF token
          (wrapped as any)[key] = wrapAPIMethod(prop as any, api);
        } else {
          // Copy non-function properties
          (wrapped as any)[key] = prop;
        }
      });

      return wrapped;
    },
    [wrapAPIMethod]
  );

  return {
    products: createWrappedAPI(productsAPI),
    customers: createWrappedAPI(customersAPI),
    subscriptions: createWrappedAPI(subscriptionsAPI),
    transactions: createWrappedAPI(transactionsAPI),
    wallets: createWrappedAPI(walletsAPI),
    csrfToken,
  };
}