'use client';

import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { CACHE_DURATIONS } from '@/lib/query/query-client';
import { useCSRF } from '@/hooks/security/use-csrf';
import type { PaginatedResponse } from '@/types/common';
import type { ProductResponse, CreateProductRequest, UpdateProductRequest } from '@/types/product';
import type { WalletResponse } from '@/types/wallet';
import type { NetworkWithTokensResponse } from '@/types/network';
import type { CustomerResponse } from '@/types/customer';
import type { SubscriptionResponse } from '@/types/subscription';
import type { SubscriptionEventFullResponse } from '@/types/subscription-event';

// Query keys for consistent caching
export const queryKeys = {
  products: (page?: number, limit?: number) => ['products', { page, limit }] as const,
  wallets: () => ['wallets'] as const,
  networks: (activeOnly?: boolean) => ['networks', { activeOnly }] as const,
  customers: (page?: number, limit?: number) => ['customers', { page, limit }] as const,
  subscriptions: (page?: number, limit?: number) => ['subscriptions', { page, limit }] as const,
  transactions: (page?: number, limit?: number) => ['transactions', { page, limit }] as const,
};

// Products hooks
export function useProducts(page = 1, limit = 10) {
  return useQuery({
    queryKey: queryKeys.products(page, limit),
    queryFn: async (): Promise<PaginatedResponse<ProductResponse>> => {
      const response = await fetch(`/api/products?page=${page}&limit=${limit}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch products: ${response.statusText}`);
      }
      return response.json();
    },
    staleTime: CACHE_DURATIONS.products,
  });
}

// Wallets hooks
export function useWallets() {
  return useQuery({
    queryKey: queryKeys.wallets(),
    queryFn: async (): Promise<WalletResponse[]> => {
      const response = await fetch('/api/wallets');
      if (!response.ok) {
        throw new Error(`Failed to fetch wallets: ${response.statusText}`);
      }
      return response.json();
    },
    staleTime: CACHE_DURATIONS.wallets,
  });
}

// Networks hooks
export function useNetworks(activeOnly = true) {
  return useQuery({
    queryKey: queryKeys.networks(activeOnly),
    queryFn: async (): Promise<NetworkWithTokensResponse[]> => {
      const url = activeOnly ? '/api/networks?active=true' : '/api/networks';
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Failed to fetch networks: ${response.statusText}`);
      }
      return response.json();
    },
    staleTime: CACHE_DURATIONS.networks,
  });
}

// Customers hooks
export function useCustomers(page = 1, limit = 10) {
  return useQuery({
    queryKey: queryKeys.customers(page, limit),
    queryFn: async (): Promise<PaginatedResponse<CustomerResponse>> => {
      const response = await fetch(`/api/customers?page=${page}&limit=${limit}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch customers: ${response.statusText}`);
      }
      return response.json();
    },
    staleTime: CACHE_DURATIONS.customers,
  });
}

// Subscriptions hooks
export function useSubscriptions(page = 1, limit = 10) {
  return useQuery({
    queryKey: queryKeys.subscriptions(page, limit),
    queryFn: async (): Promise<PaginatedResponse<SubscriptionResponse>> => {
      const response = await fetch(`/api/subscriptions?page=${page}&limit=${limit}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch subscriptions: ${response.statusText}`);
      }
      return response.json();
    },
    staleTime: CACHE_DURATIONS.subscriptions,
  });
}

// Transactions hooks
export function useTransactions(page = 1, limit = 10) {
  return useQuery({
    queryKey: queryKeys.transactions(page, limit),
    queryFn: async (): Promise<PaginatedResponse<SubscriptionEventFullResponse>> => {
      const response = await fetch(`/api/transactions?page=${page}&limit=${limit}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch transactions: ${response.statusText}`);
      }
      return response.json();
    },
    staleTime: CACHE_DURATIONS.transactions,
  });
}

// Combined hook for pages that need multiple data sources
export function useProductsPageData(page = 1, limit = 10) {
  const productsQuery = useProducts(page, limit);
  const networksQuery = useNetworks(true);
  const walletsQuery = useWallets();

  return {
    products: productsQuery.data,
    networks: networksQuery.data,
    wallets: walletsQuery.data,
    isLoading: productsQuery.isLoading || networksQuery.isLoading || walletsQuery.isLoading,
    error: productsQuery.error || networksQuery.error || walletsQuery.error,
    refetch: () => {
      productsQuery.refetch();
      networksQuery.refetch();
      walletsQuery.refetch();
    },
  };
}

export function useWalletsPageData() {
  const walletsQuery = useWallets();
  const networksQuery = useNetworks(true);

  return {
    wallets: walletsQuery.data,
    networks: networksQuery.data,
    isLoading: walletsQuery.isLoading || networksQuery.isLoading,
    error: walletsQuery.error || networksQuery.error,
    refetch: () => {
      walletsQuery.refetch();
      networksQuery.refetch();
    },
  };
}

export function useTransactionsPageData(page = 1, limit = 10) {
  const transactionsQuery = useTransactions(page, limit);
  const networksQuery = useNetworks(true);

  return {
    transactions: transactionsQuery.data,
    networks: networksQuery.data,
    isLoading: transactionsQuery.isLoading || networksQuery.isLoading,
    error: transactionsQuery.error || networksQuery.error,
    refetch: () => {
      transactionsQuery.refetch();
      networksQuery.refetch();
    },
  };
}

// Mutation hooks for data updates
export function useInvalidateQueries() {
  const queryClient = useQueryClient();

  return {
    invalidateProducts: () => queryClient.invalidateQueries({ queryKey: ['products'] }),
    invalidateWallets: () => queryClient.invalidateQueries({ queryKey: ['wallets'] }),
    invalidateCustomers: () => queryClient.invalidateQueries({ queryKey: ['customers'] }),
    invalidateSubscriptions: () => queryClient.invalidateQueries({ queryKey: ['subscriptions'] }),
    invalidateTransactions: () => queryClient.invalidateQueries({ queryKey: ['transactions'] }),
    invalidateAll: () => queryClient.invalidateQueries(),
  };
}

// Product mutation hooks
export function useCreateProduct() {
  const queryClient = useQueryClient();
  const { addCSRFHeader } = useCSRF();

  return useMutation({
    mutationFn: async (productData: CreateProductRequest): Promise<ProductResponse> => {
      // Log the request data for debugging
      console.log('Creating product with data:', JSON.stringify(productData, null, 2));
      
      const response = await fetch('/api/products', {
        method: 'POST',
        headers: addCSRFHeader({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(productData),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        // Log validation details if available
        if (errorData.details) {
          console.error('Validation errors:', JSON.stringify(errorData.details, null, 2));
        }
        throw new Error(errorData.error || 'Failed to create product');
      }

      return response.json();
    },
    onSuccess: () => {
      // Invalidate all product queries to refresh the list
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
  });
}

export function useUpdateProduct() {
  const queryClient = useQueryClient();
  const { addCSRFHeader } = useCSRF();

  return useMutation({
    mutationFn: async ({
      id,
      data,
    }: {
      id: string;
      data: UpdateProductRequest;
    }): Promise<ProductResponse> => {
      const response = await fetch(`/api/products/${id}`, {
        method: 'PUT',
        headers: addCSRFHeader({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to update product');
      }

      return response.json();
    },
    onSuccess: () => {
      // Invalidate all product queries to refresh the list
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
  });
}

export function useDeleteProduct() {
  const queryClient = useQueryClient();
  const { addCSRFHeader } = useCSRF();

  return useMutation({
    mutationFn: async (productId: string): Promise<void> => {
      const response = await fetch(`/api/products/${productId}`, {
        method: 'DELETE',
        headers: addCSRFHeader({}),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to delete product');
      }
    },
    onSuccess: () => {
      // Invalidate all product queries to refresh the list
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
  });
}

// Wallet mutation hooks
export function useCreateWallet() {
  const queryClient = useQueryClient();
  const { addCSRFHeader } = useCSRF();

  return useMutation({
    mutationFn: async (walletData: {
      wallet_address: string;
      network_id: string;
      is_primary: boolean;
      verified: boolean;
    }): Promise<WalletResponse> => {
      const response = await fetch('/api/wallets', {
        method: 'POST',
        headers: addCSRFHeader({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(walletData),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to create wallet');
      }

      return response.json();
    },
    onSuccess: () => {
      // Invalidate wallet queries to refresh the list
      queryClient.invalidateQueries({ queryKey: ['wallets'] });
    },
  });
}
