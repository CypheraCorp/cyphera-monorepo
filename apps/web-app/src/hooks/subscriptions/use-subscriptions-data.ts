import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/hooks/auth/use-auth-user';
import { useSubscriptionUIStore } from '@/store/subscription-ui';
import { useToast } from '@/components/ui/use-toast';

interface Subscription {
  id: string;
  product_id: string;
  customer_id: string;
  status: 'active' | 'canceled' | 'past_due' | 'trialing';
  current_period_start: string;
  current_period_end: string;
  cancel_at_period_end: boolean;
  canceled_at?: string;
  trial_end?: string;
  created_at: string;
  updated_at: string;
  amount: number;
  currency: string;
  interval: 'day' | 'week' | 'month' | 'year';
  interval_count: number;
}

/**
 * Hook for fetching subscriptions list - always returns fresh data
 */
export function useSubscriptions(filters?: {
  status?: string;
  customerId?: string;
  productId?: string;
}) {
  const { workspace } = useAuth();
  
  return useQuery({
    queryKey: ['subscriptions', workspace?.id, filters],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (filters?.status) params.append('status', filters.status);
      if (filters?.customerId) params.append('customer_id', filters.customerId);
      if (filters?.productId) params.append('product_id', filters.productId);
      
      const response = await fetch(`/api/subscriptions?${params}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch subscriptions');
      }
      
      return response.json() as Promise<Subscription[]>;
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
 * Hook for fetching a single subscription
 */
export function useSubscription(subscriptionId: string | null) {
  const { workspace } = useAuth();
  
  return useQuery({
    queryKey: ['subscriptions', workspace?.id, subscriptionId],
    queryFn: async () => {
      if (!subscriptionId) throw new Error('No subscription ID provided');
      
      const response = await fetch(`/api/subscriptions/${subscriptionId}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch subscription');
      }
      
      return response.json() as Promise<Subscription>;
    },
    enabled: !!workspace?.id && !!subscriptionId,
    staleTime: 0,
    gcTime: 5 * 60 * 1000,
    refetchOnMount: 'always',
  });
}

/**
 * Hook for canceling a subscription
 */
export function useCancelSubscription() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { workspace } = useAuth();
  const { closeCancelModal } = useSubscriptionUIStore();
  
  return useMutation({
    mutationFn: async ({ 
      subscriptionId, 
      cancelAtPeriodEnd = true 
    }: { 
      subscriptionId: string; 
      cancelAtPeriodEnd?: boolean;
    }) => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/cancel`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ cancel_at_period_end: cancelAtPeriodEnd }),
      });
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || 'Failed to cancel subscription');
      }
      
      return response.json();
    },
    onSuccess: (data, variables) => {
      // Invalidate queries to refetch
      queryClient.invalidateQueries({ queryKey: ['subscriptions', workspace?.id] });
      queryClient.invalidateQueries({ 
        queryKey: ['subscriptions', workspace?.id, variables.subscriptionId] 
      });
      
      // Close modal
      closeCancelModal();
      
      // Show success toast
      toast({
        title: 'Subscription Canceled',
        description: variables.cancelAtPeriodEnd 
          ? 'Your subscription will remain active until the end of the current period.'
          : 'Your subscription has been canceled immediately.',
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
 * Hook for reactivating a subscription
 */
export function useReactivateSubscription() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { workspace } = useAuth();
  
  return useMutation({
    mutationFn: async (subscriptionId: string) => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/reactivate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to reactivate subscription');
      }
      
      return response.json();
    },
    onSuccess: (data, subscriptionId) => {
      // Invalidate and refetch
      queryClient.invalidateQueries({ queryKey: ['subscriptions', workspace?.id] });
      queryClient.invalidateQueries({ 
        queryKey: ['subscriptions', workspace?.id, subscriptionId] 
      });
      
      toast({
        title: 'Subscription Reactivated',
        description: 'Your subscription has been reactivated successfully.',
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
 * Combined hook that provides subscription data and UI state
 * This is the main hook components should use
 */
export function useSubscriptionsWithUI() {
  const { filters, sortBy, sortOrder, ...uiStore } = useSubscriptionUIStore();
  const subscriptionQuery = useSubscriptions(filters);
  const selectedSubscription = useSubscription(uiStore.selectedSubscriptionId);
  
  // Sort subscriptions based on UI state
  const sortedSubscriptions = subscriptionQuery.data?.sort((a, b) => {
    let compareValue = 0;
    
    switch (sortBy) {
      case 'created_at':
        compareValue = new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
        break;
      case 'status':
        compareValue = a.status.localeCompare(b.status);
        break;
      case 'amount':
        compareValue = a.amount - b.amount;
        break;
      case 'next_billing_date':
        compareValue = new Date(a.current_period_end).getTime() - 
                      new Date(b.current_period_end).getTime();
        break;
    }
    
    return sortOrder === 'asc' ? compareValue : -compareValue;
  }) || [];
  
  return {
    // Data
    subscriptions: sortedSubscriptions,
    selectedSubscription: selectedSubscription.data,
    
    // Loading states
    isLoading: subscriptionQuery.isLoading,
    isRefetching: subscriptionQuery.isRefetching,
    isLoadingSelected: selectedSubscription.isLoading,
    
    // Error states
    error: subscriptionQuery.error,
    
    // UI State
    filters,
    sortBy,
    sortOrder,
    ...uiStore,
    
    // Actions
    refetch: subscriptionQuery.refetch,
  };
}