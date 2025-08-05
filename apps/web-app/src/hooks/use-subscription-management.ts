import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type {
  UpgradeSubscriptionRequest,
  DowngradeSubscriptionRequest,
  CancelSubscriptionRequest,
  PauseSubscriptionRequest,
  PreviewChangeRequest,
  ChangePreview,
  SubscriptionResponse,
  SubscriptionStateHistory,
} from '@/types/subscription';
import { toast } from 'sonner';

/**
 * Hook for previewing subscription changes
 */
export function usePreviewSubscriptionChange(subscriptionId: string) {
  return useMutation({
    mutationFn: async (request: PreviewChangeRequest) => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/preview-change`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });
      if (!response.ok) {
        throw new Error('Failed to preview changes');
      }
      return response.json() as Promise<ChangePreview>;
    },
    onError: (error) => {
      toast.error('Failed to preview changes');
      console.error('Preview change error:', error);
    },
  });
}

/**
 * Hook for upgrading a subscription
 */
export function useUpgradeSubscription(subscriptionId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: UpgradeSubscriptionRequest) => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/upgrade`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });
      if (!response.ok) {
        throw new Error('Failed to upgrade subscription');
      }
      return response.json();
    },
    onSuccess: (data) => {
      // Invalidate subscription queries to refetch updated data
      queryClient.invalidateQueries({ queryKey: ['subscription', subscriptionId] });
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] });
      toast.success(data.message || 'Subscription upgraded successfully');
    },
    onError: (error) => {
      toast.error('Failed to upgrade subscription');
      console.error('Upgrade error:', error);
    },
  });
}

/**
 * Hook for downgrading a subscription
 */
export function useDowngradeSubscription(subscriptionId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: DowngradeSubscriptionRequest) => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/downgrade`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });
      if (!response.ok) {
        throw new Error('Failed to schedule downgrade');
      }
      return response.json();
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['subscription', subscriptionId] });
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] });
      toast.success(data.message || 'Downgrade scheduled successfully');
    },
    onError: (error) => {
      toast.error('Failed to schedule downgrade');
      console.error('Downgrade error:', error);
    },
  });
}

/**
 * Hook for cancelling a subscription
 */
export function useCancelSubscription(subscriptionId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CancelSubscriptionRequest) => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/cancel`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });
      if (!response.ok) {
        throw new Error('Failed to cancel subscription');
      }
      return response.json();
    },
    onSuccess: (data) => {
      // Optimistically update the subscription status
      queryClient.setQueryData<SubscriptionResponse>(
        ['subscription', subscriptionId],
        (old) => {
          if (!old) return old;
          return {
            ...old,
            cancel_at: new Date().toISOString(), // This would be the actual end date
            cancellation_reason: 'User requested',
          };
        }
      );
      queryClient.invalidateQueries({ queryKey: ['subscription', subscriptionId] });
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] });
      toast.success(data.message || 'Subscription cancellation scheduled');
    },
    onError: (error) => {
      toast.error('Failed to cancel subscription');
      console.error('Cancel error:', error);
    },
  });
}

/**
 * Hook for reactivating a cancelled subscription
 */
export function useReactivateSubscription(subscriptionId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/reactivate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (!response.ok) {
        throw new Error('Failed to reactivate subscription');
      }
      return response.json();
    },
    onSuccess: (data) => {
      // Optimistically update the subscription
      queryClient.setQueryData<SubscriptionResponse>(
        ['subscription', subscriptionId],
        (old) => {
          if (!old) return old;
          return {
            ...old,
            cancel_at: undefined,
            cancellation_reason: undefined,
          };
        }
      );
      queryClient.invalidateQueries({ queryKey: ['subscription', subscriptionId] });
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] });
      toast.success(data.message || 'Subscription reactivated');
    },
    onError: (error) => {
      toast.error('Failed to reactivate subscription');
      console.error('Reactivate error:', error);
    },
  });
}

/**
 * Hook for pausing a subscription
 */
export function usePauseSubscription(subscriptionId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: PauseSubscriptionRequest) => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/pause`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });
      if (!response.ok) {
        throw new Error('Failed to pause subscription');
      }
      return response.json();
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['subscription', subscriptionId] });
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] });
      toast.success(data.message || 'Subscription paused');
    },
    onError: (error) => {
      toast.error('Failed to pause subscription');
      console.error('Pause error:', error);
    },
  });
}

/**
 * Hook for resuming a paused subscription
 */
export function useResumeSubscription(subscriptionId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/resume`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (!response.ok) {
        throw new Error('Failed to resume subscription');
      }
      return response.json();
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['subscription', subscriptionId] });
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] });
      toast.success(data.message || 'Subscription resumed');
    },
    onError: (error) => {
      toast.error('Failed to resume subscription');
      console.error('Resume error:', error);
    },
  });
}

/**
 * Hook for fetching subscription history
 */
export function useSubscriptionHistory(subscriptionId: string, limit: number = 50) {
  return useQuery({
    queryKey: ['subscription-history', subscriptionId, limit],
    queryFn: async () => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}/history?limit=${limit}`);
      if (!response.ok) {
        throw new Error('Failed to fetch subscription history');
      }
      return response.json() as Promise<{ history: SubscriptionStateHistory[]; count: number }>;
    },
    enabled: !!subscriptionId,
  });
}

/**
 * Helper hook to get a single subscription with React Query
 */
export function useSubscription(subscriptionId: string) {
  return useQuery({
    queryKey: ['subscription', subscriptionId],
    queryFn: async () => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}`);
      if (!response.ok) {
        throw new Error('Failed to fetch subscription');
      }
      return response.json() as Promise<SubscriptionResponse>;
    },
    enabled: !!subscriptionId,
  });
}