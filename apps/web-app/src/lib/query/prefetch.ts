import { QueryClient } from '@tanstack/react-query';
import { CACHE_DURATIONS } from './query-client';

// Query key factory for consistent key generation
export const queryKeys = {
  // Products
  products: {
    all: ['products'] as const,
    list: (params?: { page?: number; limit?: number }) => ['products', 'list', params] as const,
    detail: (id: string) => ['products', 'detail', id] as const,
  },

  // Wallets
  wallets: {
    all: ['wallets'] as const,
    list: () => ['wallets', 'list'] as const,
    detail: (id: string) => ['wallets', 'detail', id] as const,
    balances: (walletId: string) => ['wallets', 'balances', walletId] as const,
  },

  // Customers
  customers: {
    all: ['customers'] as const,
    list: (params?: { page?: number; limit?: number }) => ['customers', 'list', params] as const,
    detail: (id: string) => ['customers', 'detail', id] as const,
  },

  // Subscriptions
  subscriptions: {
    all: ['subscriptions'] as const,
    list: (params?: { page?: number; limit?: number; status?: string }) =>
      ['subscriptions', 'list', params] as const,
    detail: (id: string) => ['subscriptions', 'detail', id] as const,
  },

  // Transactions
  transactions: {
    all: ['transactions'] as const,
    list: (params?: { page?: number; limit?: number }) => ['transactions', 'list', params] as const,
    detail: (id: string) => ['transactions', 'detail', id] as const,
  },

  // Networks
  networks: {
    all: ['networks'] as const,
    list: () => ['networks', 'list'] as const,
  },

  // User
  user: {
    current: ['user', 'current'] as const,
    profile: ['user', 'profile'] as const,
  },
};

// Prefetch functions for common navigation patterns
export const prefetchUtils = {
  // Prefetch merchant dashboard data
  async prefetchMerchantDashboard(queryClient: QueryClient) {
    await Promise.all([
      queryClient.prefetchQuery({
        queryKey: queryKeys.products.list({ page: 1, limit: 10 }),
        queryFn: () => fetch('/api/products?page=1&limit=10').then((res) => res.json()),
        staleTime: CACHE_DURATIONS.products,
      }),
      queryClient.prefetchQuery({
        queryKey: queryKeys.wallets.list(),
        queryFn: () => fetch('/api/wallets').then((res) => res.json()),
        staleTime: CACHE_DURATIONS.wallets,
      }),
      queryClient.prefetchQuery({
        queryKey: queryKeys.networks.list(),
        queryFn: () => fetch('/api/networks').then((res) => res.json()),
        staleTime: CACHE_DURATIONS.networks,
      }),
    ]);
  },

  // Prefetch product details page
  async prefetchProductDetails(queryClient: QueryClient, productId: string) {
    await queryClient.prefetchQuery({
      queryKey: queryKeys.products.detail(productId),
      queryFn: () => fetch(`/api/products/${productId}`).then((res) => res.json()),
      staleTime: CACHE_DURATIONS.products,
    });
  },

  // Prefetch customer data
  async prefetchCustomerData(queryClient: QueryClient) {
    await Promise.all([
      queryClient.prefetchQuery({
        queryKey: queryKeys.customers.list({ page: 1, limit: 20 }),
        queryFn: () => fetch('/api/customers?page=1&limit=20').then((res) => res.json()),
        staleTime: CACHE_DURATIONS.customers,
      }),
      queryClient.prefetchQuery({
        queryKey: queryKeys.subscriptions.list({ page: 1, limit: 20 }),
        queryFn: () => fetch('/api/subscriptions?page=1&limit=20').then((res) => res.json()),
        staleTime: CACHE_DURATIONS.subscriptions,
      }),
    ]);
  },

  // Prefetch on hover (for instant navigation)
  prefetchOnHover(queryClient: QueryClient, prefetchFn: () => Promise<void>) {
    let timeoutId: NodeJS.Timeout;

    return {
      onMouseEnter: () => {
        // Delay prefetch by 100ms to avoid prefetching on accidental hovers
        timeoutId = setTimeout(() => {
          prefetchFn();
        }, 100);
      },
      onMouseLeave: () => {
        clearTimeout(timeoutId);
      },
    };
  },
};

// Optimistic update utilities
export const optimisticUpdates = {
  // Update product in cache
  updateProduct: (queryClient: QueryClient, productId: string, updates: Partial<unknown>) => {
    queryClient.setQueryData(queryKeys.products.detail(productId), (old: unknown) => ({
      ...(old as object),
      ...updates,
    }));

    // Also update in list
    queryClient.setQueriesData(
      { queryKey: queryKeys.products.all, exact: false },
      (old: unknown) => {
        if (!Array.isArray(old)) return old;
        return old.map((item: Record<string, unknown> & { id: string }) =>
          item.id === productId ? { ...item, ...updates } : item
        );
      }
    );
  },

  // Add item to list
  addToList: <T extends { id: string }>(
    queryClient: QueryClient,
    queryKey: readonly unknown[],
    newItem: T
  ) => {
    queryClient.setQueryData(queryKey, (old: T[] = []) => [newItem, ...old]);
  },

  // Remove item from list
  removeFromList: <T extends { id: string }>(
    queryClient: QueryClient,
    queryKey: readonly unknown[],
    itemId: string
  ) => {
    queryClient.setQueryData(queryKey, (old: T[] = []) => old.filter((item) => item.id !== itemId));
  },
};
