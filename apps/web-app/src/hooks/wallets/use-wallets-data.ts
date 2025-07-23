import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/hooks/auth/use-auth-user';
import { useWalletUIStore } from '@/store/wallet-ui';
import { useToast } from '@/components/ui/use-toast';

interface Wallet {
  id: string;
  address: string;
  blockchain: string;
  wallet_state: string;
  created_at: string;
  updated_at: string;
}

/**
 * Hook for fetching wallet list - always returns fresh data
 */
export function useWallets() {
  const { workspace } = useAuth();
  
  return useQuery({
    queryKey: ['wallets', workspace?.id],
    queryFn: async () => {
      const response = await fetch('/api/wallets', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch wallets');
      }
      
      return response.json() as Promise<Wallet[]>;
    },
    enabled: !!workspace?.id,
    staleTime: 0, // Always fetch fresh data
    gcTime: 5 * 60 * 1000, // Cache for 5 minutes
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    retry: 3,
  });
}

/**
 * Hook for fetching a single wallet
 */
export function useWallet(walletId: string | null) {
  const { workspace } = useAuth();
  
  return useQuery({
    queryKey: ['wallets', workspace?.id, walletId],
    queryFn: async () => {
      if (!walletId) throw new Error('No wallet ID provided');
      
      const response = await fetch(`/api/wallets/${walletId}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch wallet');
      }
      
      return response.json() as Promise<Wallet>;
    },
    enabled: !!workspace?.id && !!walletId,
    staleTime: 0,
    gcTime: 5 * 60 * 1000,
    refetchOnMount: 'always',
  });
}

/**
 * Hook for creating a new wallet
 */
export function useCreateWallet() {
  const queryClient = useQueryClient();
  const { setCreateModalOpen } = useWalletUIStore();
  const { toast } = useToast();
  const { workspace } = useAuth();
  
  return useMutation({
    mutationFn: async (data: { blockchain: string; idempotencyKey: string }) => {
      const response = await fetch('/api/wallets', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || 'Failed to create wallet');
      }
      
      return response.json();
    },
    onSuccess: (data) => {
      // Invalidate and refetch wallet list
      queryClient.invalidateQueries({ queryKey: ['wallets', workspace?.id] });
      
      // Close modal
      setCreateModalOpen(false);
      
      // Show success toast
      toast({
        title: 'Wallet Created',
        description: `Successfully created wallet ${data.address}`,
      });
    },
    onError: (error) => {
      toast({
        title: 'Error',
        description: error.message,
        variant: 'destructive',
      });
    },
  });
}

/**
 * Hook for deleting a wallet
 */
export function useDeleteWallet() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { workspace } = useAuth();
  
  return useMutation({
    mutationFn: async (walletId: string) => {
      const response = await fetch(`/api/wallets/${walletId}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to delete wallet');
      }
      
      return response.json();
    },
    onSuccess: () => {
      // Invalidate and refetch
      queryClient.invalidateQueries({ queryKey: ['wallets', workspace?.id] });
      
      toast({
        title: 'Wallet Deleted',
        description: 'The wallet has been successfully deleted.',
      });
    },
    onError: (error) => {
      toast({
        title: 'Error',
        description: error.message,
        variant: 'destructive',
      });
    },
  });
}

/**
 * Combined hook that provides wallet data and UI state
 * This is the main hook components should use
 */
export function useWalletsWithUI() {
  const walletQuery = useWallets();
  const uiStore = useWalletUIStore();
  const selectedWallet = useWallet(uiStore.selectedWalletId);
  
  // Filter wallets based on UI filters
  const filteredWallets = walletQuery.data?.filter((wallet) => {
    if (uiStore.filters.network && wallet.blockchain !== uiStore.filters.network) {
      return false;
    }
    if (uiStore.filters.search) {
      const search = uiStore.filters.search.toLowerCase();
      return wallet.address.toLowerCase().includes(search) ||
             wallet.blockchain.toLowerCase().includes(search);
    }
    return true;
  }) || [];
  
  return {
    // Data
    wallets: filteredWallets,
    selectedWallet: selectedWallet.data,
    
    // Loading states
    isLoading: walletQuery.isLoading,
    isRefetching: walletQuery.isRefetching,
    isLoadingSelected: selectedWallet.isLoading,
    
    // Error states
    error: walletQuery.error,
    
    // UI State
    ...uiStore,
    
    // Actions
    refetch: walletQuery.refetch,
  };
}