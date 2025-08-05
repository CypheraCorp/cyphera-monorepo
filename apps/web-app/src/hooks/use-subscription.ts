import { useQuery } from '@tanstack/react-query';
import type { SubscriptionResponse } from '@/types/subscription';

export function useSubscription(subscriptionId: string) {
  return useQuery<SubscriptionResponse>({
    queryKey: ['subscription', subscriptionId],
    queryFn: async () => {
      const response = await fetch(`/api/subscriptions/${subscriptionId}`);
      if (!response.ok) {
        throw new Error('Failed to fetch subscription');
      }
      return response.json();
    },
    enabled: !!subscriptionId,
    staleTime: 1000 * 60, // 1 minute
    refetchOnWindowFocus: true,
  });
}