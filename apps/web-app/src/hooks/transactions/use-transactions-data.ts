import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/hooks/auth/use-auth-user';
import { useTransactionUIStore } from '@/store/transaction-ui';
import { useToast } from '@/components/ui/use-toast';

interface Transaction {
  id: string;
  wallet_id: string;
  network_id: string;
  transaction_hash: string;
  status: 'pending' | 'completed' | 'failed' | 'processing';
  type: 'payment' | 'refund' | 'payout' | 'fee';
  amount: string;
  currency: string;
  gas_fee?: string;
  from_address: string;
  to_address: string;
  customer_id?: string;
  subscription_id?: string;
  product_id?: string;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

/**
 * Hook for fetching transactions list - always returns fresh data
 */
export function useTransactions(filters?: {
  status?: string;
  type?: string;
  walletId?: string;
  customerId?: string;
  dateRange?: { from: Date; to: Date };
  minAmount?: number;
  maxAmount?: number;
}) {
  const { workspace } = useAuth();
  
  return useQuery({
    queryKey: ['transactions', workspace?.id, filters],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (filters?.status) params.append('status', filters.status);
      if (filters?.type) params.append('type', filters.type);
      if (filters?.walletId) params.append('wallet_id', filters.walletId);
      if (filters?.customerId) params.append('customer_id', filters.customerId);
      if (filters?.dateRange) {
        params.append('from_date', filters.dateRange.from.toISOString());
        params.append('to_date', filters.dateRange.to.toISOString());
      }
      if (filters?.minAmount) params.append('min_amount', filters.minAmount.toString());
      if (filters?.maxAmount) params.append('max_amount', filters.maxAmount.toString());
      
      const response = await fetch(`/api/transactions?${params}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch transactions');
      }
      
      return response.json() as Promise<Transaction[]>;
    },
    enabled: !!workspace?.id,
    staleTime: 30 * 1000, // Consider stale after 30 seconds
    gcTime: 5 * 60 * 1000, // Cache for 5 minutes
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    retry: 3,
  });
}

/**
 * Hook for fetching a single transaction
 */
export function useTransaction(transactionId: string | null) {
  const { workspace } = useAuth();
  
  return useQuery({
    queryKey: ['transactions', workspace?.id, transactionId],
    queryFn: async () => {
      if (!transactionId) throw new Error('No transaction ID provided');
      
      const response = await fetch(`/api/transactions/${transactionId}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch transaction');
      }
      
      return response.json() as Promise<Transaction>;
    },
    enabled: !!workspace?.id && !!transactionId,
    staleTime: 30 * 1000,
    gcTime: 5 * 60 * 1000,
    refetchOnMount: 'always',
  });
}

/**
 * Hook for creating a refund
 */
export function useCreateRefund() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { workspace } = useAuth();
  const { closeRefundModal } = useTransactionUIStore();
  
  return useMutation({
    mutationFn: async ({ 
      transactionId, 
      amount,
      reason 
    }: { 
      transactionId: string; 
      amount?: string;
      reason?: string;
    }) => {
      const response = await fetch(`/api/transactions/${transactionId}/refund`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ amount, reason }),
      });
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || 'Failed to create refund');
      }
      
      return response.json();
    },
    onSuccess: () => {
      // Invalidate queries to refetch
      queryClient.invalidateQueries({ queryKey: ['transactions', workspace?.id] });
      
      // Close modal
      closeRefundModal();
      
      // Show success toast
      toast({
        title: 'Refund Initiated',
        description: 'The refund has been initiated successfully.',
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
 * Hook for exporting transactions
 */
export function useExportTransactions() {
  const { filters, exportFormat, setExporting } = useTransactionUIStore();
  const { toast } = useToast();
  
  return useMutation({
    mutationFn: async () => {
      setExporting(true);
      
      const params = new URLSearchParams();
      params.append('format', exportFormat);
      
      // Add filters to params
      if (filters.status) params.append('status', filters.status);
      if (filters.type) params.append('type', filters.type);
      if (filters.dateRange) {
        params.append('from_date', filters.dateRange.from.toISOString());
        params.append('to_date', filters.dateRange.to.toISOString());
      }
      
      const response = await fetch(`/api/transactions/export?${params}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to export transactions');
      }
      
      // Handle file download
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `transactions-${new Date().toISOString()}.${exportFormat}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    },
    onSuccess: () => {
      toast({
        title: 'Export Complete',
        description: 'Your transactions have been exported successfully.',
      });
    },
    onError: (error) => {
      toast({
        title: 'Export Failed',
        description: error.message,
        variant: 'destructive',
      });
    },
    onSettled: () => {
      setExporting(false);
    },
  });
}

/**
 * Combined hook that provides transaction data and UI state
 * This is the main hook components should use
 */
export function useTransactionsWithUI() {
  const { filters, sortBy, sortOrder, itemsPerPage, ...uiStore } = useTransactionUIStore();
  const transactionQuery = useTransactions(filters);
  const selectedTransaction = useTransaction(uiStore.selectedTransactionId);
  
  // Sort transactions based on UI state
  const sortedTransactions = transactionQuery.data?.sort((a, b) => {
    let compareValue = 0;
    
    switch (sortBy) {
      case 'created_at':
        compareValue = new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
        break;
      case 'amount':
        compareValue = parseFloat(a.amount) - parseFloat(b.amount);
        break;
      case 'status':
        compareValue = a.status.localeCompare(b.status);
        break;
      case 'type':
        compareValue = a.type.localeCompare(b.type);
        break;
    }
    
    return sortOrder === 'asc' ? compareValue : -compareValue;
  }) || [];
  
  return {
    // Data
    transactions: sortedTransactions,
    selectedTransaction: selectedTransaction.data,
    
    // Loading states
    isLoading: transactionQuery.isLoading,
    isRefetching: transactionQuery.isRefetching,
    isLoadingSelected: selectedTransaction.isLoading,
    
    // Error states
    error: transactionQuery.error,
    
    // UI State
    filters,
    sortBy,
    sortOrder,
    itemsPerPage,
    ...uiStore,
    
    // Actions
    refetch: transactionQuery.refetch,
  };
}