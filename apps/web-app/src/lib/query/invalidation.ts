import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from './prefetch';
import { logger } from '@/lib/core/logger/logger-utils';

/**
 * Hook for smart query invalidation with relationship awareness
 */
export function useSmartInvalidation() {
  const queryClient = useQueryClient();

  return {
    // Invalidate product and related queries
    invalidateProduct: async (productId?: string) => {
      logger.debug('Invalidating product queries', { productId });

      const invalidations = [];

      // Invalidate specific product if ID provided
      if (productId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queryKeys.products.detail(productId),
          })
        );
      }

      // Always invalidate product lists
      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.products.all,
        })
      );

      // Invalidate subscriptions as they depend on products
      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.subscriptions.all,
        })
      );

      await Promise.all(invalidations);
    },

    // Invalidate wallet and related queries
    invalidateWallet: async (walletId?: string) => {
      logger.debug('Invalidating wallet queries', { walletId });

      const invalidations = [];

      // Invalidate specific wallet if ID provided
      if (walletId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queryKeys.wallets.detail(walletId),
          }),
          queryClient.invalidateQueries({
            queryKey: queryKeys.wallets.balances(walletId),
          })
        );
      }

      // Always invalidate wallet lists
      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.wallets.all,
        })
      );

      // Invalidate transactions as they relate to wallets
      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.transactions.all,
        })
      );

      await Promise.all(invalidations);
    },

    // Invalidate customer and related queries
    invalidateCustomer: async (customerId?: string) => {
      logger.debug('Invalidating customer queries', { customerId });

      const invalidations = [];

      // Invalidate specific customer if ID provided
      if (customerId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queryKeys.customers.detail(customerId),
          })
        );
      }

      // Always invalidate customer lists
      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.customers.all,
        })
      );

      // Invalidate subscriptions and transactions for this customer
      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.subscriptions.all,
        }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.transactions.all,
        })
      );

      await Promise.all(invalidations);
    },

    // Invalidate subscription and related queries
    invalidateSubscription: async (subscriptionId?: string) => {
      logger.debug('Invalidating subscription queries', { subscriptionId });

      const invalidations = [];

      // Invalidate specific subscription if ID provided
      if (subscriptionId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queryKeys.subscriptions.detail(subscriptionId),
          })
        );
      }

      // Always invalidate subscription lists
      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.subscriptions.all,
        })
      );

      // Invalidate customer and transaction data
      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.customers.all,
        }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.transactions.all,
        })
      );

      await Promise.all(invalidations);
    },

    // Invalidate all user-related data (useful after auth changes)
    invalidateUserData: async () => {
      logger.debug('Invalidating all user data');

      await queryClient.invalidateQueries({
        predicate: (query) => {
          const key = query.queryKey[0];
          return [
            'user',
            'products',
            'wallets',
            'customers',
            'subscriptions',
            'transactions',
          ].includes(key as string);
        },
      });
    },

    // Smart invalidation based on mutation type
    invalidateAfterMutation: async (
      entityType: 'product' | 'wallet' | 'customer' | 'subscription' | 'transaction',
      entityId?: string,
      action?: 'create' | 'update' | 'delete'
    ) => {
      logger.debug('Smart invalidation after mutation', { entityType, entityId, action });

      switch (entityType) {
        case 'product':
          await invalidations.invalidateProduct(entityId);
          break;
        case 'wallet':
          await invalidations.invalidateWallet(entityId);
          break;
        case 'customer':
          await invalidations.invalidateCustomer(entityId);
          break;
        case 'subscription':
          await invalidations.invalidateSubscription(entityId);
          break;
        case 'transaction':
          // Transactions affect multiple entities
          await Promise.all([
            queryClient.invalidateQueries({ queryKey: queryKeys.transactions.all }),
            queryClient.invalidateQueries({ queryKey: queryKeys.wallets.all }),
            queryClient.invalidateQueries({ queryKey: queryKeys.customers.all }),
          ]);
          break;
      }

      // For create/delete actions, also invalidate lists
      if (action === 'create' || action === 'delete') {
        await queryClient.invalidateQueries({
          queryKey: [entityType],
          exact: false,
        });
      }
    },
  };

  const invalidations = {
    invalidateProduct: async (productId?: string) => {
      logger.debug('Invalidating product queries', { productId });

      const invalidations = [];

      if (productId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queryKeys.products.detail(productId),
          })
        );
      }

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.products.all,
        })
      );

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.subscriptions.all,
        })
      );

      await Promise.all(invalidations);
    },

    invalidateWallet: async (walletId?: string) => {
      logger.debug('Invalidating wallet queries', { walletId });

      const invalidations = [];

      if (walletId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queryKeys.wallets.detail(walletId),
          }),
          queryClient.invalidateQueries({
            queryKey: queryKeys.wallets.balances(walletId),
          })
        );
      }

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.wallets.all,
        })
      );

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.transactions.all,
        })
      );

      await Promise.all(invalidations);
    },

    invalidateCustomer: async (customerId?: string) => {
      logger.debug('Invalidating customer queries', { customerId });

      const invalidations = [];

      if (customerId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queryKeys.customers.detail(customerId),
          })
        );
      }

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.customers.all,
        })
      );

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.subscriptions.all,
        }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.transactions.all,
        })
      );

      await Promise.all(invalidations);
    },

    invalidateSubscription: async (subscriptionId?: string) => {
      logger.debug('Invalidating subscription queries', { subscriptionId });

      const invalidations = [];

      if (subscriptionId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queryKeys.subscriptions.detail(subscriptionId),
          })
        );
      }

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.subscriptions.all,
        })
      );

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: queryKeys.customers.all,
        }),
        queryClient.invalidateQueries({
          queryKey: queryKeys.transactions.all,
        })
      );

      await Promise.all(invalidations);
    },
  };

  return invalidations;
}
