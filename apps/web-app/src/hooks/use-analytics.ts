import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/store/auth';
import { useDefaultCurrency } from '@/hooks/use-currency';
import { analyticsService } from '@/services/cyphera-api/analytics';
import type {
  DashboardSummary,
  ChartData,
  PieChartData,
  PaymentMetrics,
  NetworkBreakdown,
  HourlyMetrics,
} from '@/types/analytics';
import type { UserRequestContext } from '@/services/cyphera-api/api';
import { useEffect } from 'react';

interface UseAnalyticsOptions {
  currency?: string;
  period?: 'hourly' | 'daily' | 'weekly' | 'monthly';
  days?: number;
  metric?: string;
  months?: number;
  date?: string;
}

function useSetupAnalyticsService() {
  const { accessToken, account, workspace } = useAuthStore();
  
  useEffect(() => {
    if (accessToken && account && workspace) {
      const context: UserRequestContext = {
        access_token: accessToken,
        account_id: account.id,
        workspace_id: workspace.id,
      };
      analyticsService.setUserContext(context);
    }
  }, [accessToken, account, workspace]);
  
  return !!accessToken && !!workspace;
}

export function useDashboardSummary(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;

  return useQuery<DashboardSummary>({
    queryKey: ['dashboard-summary', workspaceId, currency],
    queryFn: async () => {
      if (!workspaceId) throw new Error('No workspace ID');
      const data = await analyticsService.getDashboardSummary({ workspaceId, currency });
      return data;
    },
    enabled: isReady && !!workspaceId,
    refetchInterval: (query) => {
      // More relaxed polling intervals
      const data = query.state.data;
      if (data?.is_calculating || data?.is_stale) {
        return 30000; // Poll every 30 seconds when calculating or stale
      }
      return 5 * 60 * 1000; // Poll every 5 minutes when data exists
    },
    refetchIntervalInBackground: true,
    staleTime: 60 * 1000, // Consider data fresh for 1 minute
    retry: (failureCount, error: any) => {
      // Don't retry 404s
      if (error?.status === 404 || error?.response?.status === 404) {
        return false;
      }
      return failureCount < 3;
    },
  });
}

export function useRevenueChart(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;
  const period = options?.period || 'daily';
  const days = options?.days || 30;

  return useQuery<ChartData>({
    queryKey: ['revenue-chart', workspaceId, currency, period, days],
    queryFn: async () => {
      if (!workspaceId) throw new Error('No workspace ID');
      try {
        return await analyticsService.getRevenueChart({ workspaceId, currency, period, days });
      } catch (error: any) {
        // Return empty chart data on 404
        if (error?.status === 404 || error?.response?.status === 404) {
          return {
            chart_type: 'line',
            title: 'Revenue Over Time',
            data: [],
            period: period,
          };
        }
        throw error;
      }
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 15, // 15 minutes
    retry: (failureCount, error: any) => {
      if (error?.status === 404 || error?.response?.status === 404) return false;
      return failureCount < 3;
    },
  });
}

export function useCustomerChart(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;
  const period = options?.period || 'daily';
  const days = options?.days || 30;
  const metric = options?.metric || 'total';

  return useQuery<ChartData>({
    queryKey: ['customer-chart', workspaceId, currency, metric, period, days],
    queryFn: async () => {
      if (!workspaceId) throw new Error('No workspace ID');
      try {
        return await analyticsService.getCustomerChart({ 
          workspaceId, 
          currency, 
          metric: metric as any,
          period, 
          days 
        });
      } catch (error: any) {
        if (error?.status === 404 || error?.response?.status === 404) {
          return {
            chart_type: 'line',
            title: 'Customer Metrics',
            data: [],
            period: period,
          };
        }
        throw error;
      }
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 15, // 15 minutes
    retry: (failureCount, error: any) => {
      if (error?.status === 404 || error?.response?.status === 404) return false;
      return failureCount < 3;
    },
  });
}

export function useSubscriptionChart(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;
  const period = options?.period || 'daily';
  const days = options?.days || 30;
  const metric = options?.metric || 'active';

  return useQuery<ChartData>({
    queryKey: ['subscription-chart', workspaceId, currency, metric, period, days],
    queryFn: async () => {
      if (!workspaceId) throw new Error('No workspace ID');
      try {
        return await analyticsService.getSubscriptionChart({ 
          workspaceId, 
          currency, 
          metric: metric as any,
          period, 
          days 
        });
      } catch (error: any) {
        if (error?.status === 404 || error?.response?.status === 404) {
          return {
            chart_type: 'line',
            title: 'Subscription Metrics',
            data: [],
            period: period,
          };
        }
        throw error;
      }
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 15, // 15 minutes
    retry: (failureCount, error: any) => {
      if (error?.status === 404 || error?.response?.status === 404) return false;
      return failureCount < 3;
    },
  });
}

export function useMRRChart(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;
  const period = (options?.period === 'hourly' ? 'monthly' : options?.period) || 'monthly';
  const months = options?.months || 12;
  const metric = options?.metric || 'mrr';

  return useQuery<ChartData>({
    queryKey: ['mrr-chart', workspaceId, currency, metric, period, months],
    queryFn: async () => {
      if (!workspaceId) throw new Error('No workspace ID');
      try {
        return await analyticsService.getMRRChart({ 
          workspaceId, 
          currency, 
          metric: metric as any,
          period: period as 'daily' | 'weekly' | 'monthly', 
          months 
        });
      } catch (error: any) {
        if (error?.status === 404 || error?.response?.status === 404) {
          return {
            chart_type: 'line',
            title: metric === 'mrr' ? 'Monthly Recurring Revenue' : 'Annual Recurring Revenue',
            data: [],
            period: period,
          };
        }
        throw error;
      }
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 15, // 15 minutes
    retry: (failureCount, error: any) => {
      if (error?.status === 404 || error?.response?.status === 404) return false;
      return failureCount < 3;
    },
  });
}

export function usePaymentMetrics(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;
  const days = options?.days || 30;

  return useQuery<PaymentMetrics>({
    queryKey: ['payment-metrics', workspaceId, currency, days],
    queryFn: () => {
      if (!workspaceId) throw new Error('No workspace ID');
      return analyticsService.getPaymentMetrics({ workspaceId, currency, days });
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 15, // 15 minutes
  });
}

export function useGasFeePieChart(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;
  const days = options?.days || 30;

  return useQuery<PieChartData>({
    queryKey: ['gas-fee-chart', workspaceId, currency, days],
    queryFn: async () => {
      if (!workspaceId) throw new Error('No workspace ID');
      try {
        return await analyticsService.getGasFeePieChart({ workspaceId, currency, days });
      } catch (error: any) {
        if (error?.status === 404 || error?.response?.status === 404) {
          const currencyCode = currency || 'USD';
          return {
            chart_type: 'pie',
            title: 'Gas Fee Distribution',
            data: [
              { label: 'Merchant Sponsored', value: 0 },
              { label: 'Customer Paid', value: 0 },
            ],
            total: { amount_cents: 0, currency: currencyCode, formatted: '$0.00' },
          };
        }
        throw error;
      }
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 15, // 15 minutes
    retry: (failureCount, error: any) => {
      if (error?.status === 404 || error?.response?.status === 404) return false;
      return failureCount < 3;
    },
  });
}

export function useNetworkBreakdown(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;
  const date = options?.date || new Date().toISOString().split('T')[0];

  return useQuery<NetworkBreakdown>({
    queryKey: ['network-breakdown', workspaceId, currency, date],
    queryFn: () => {
      if (!workspaceId) throw new Error('No workspace ID');
      return analyticsService.getNetworkBreakdown({ workspaceId, currency, date });
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 15, // 15 minutes
  });
}

export function useHourlyMetrics(options?: UseAnalyticsOptions) {
  const { workspace } = useAuthStore();
  const { data: defaultCurrency } = useDefaultCurrency();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;
  const currency = options?.currency || defaultCurrency?.code;

  return useQuery<HourlyMetrics>({
    queryKey: ['hourly-metrics', workspaceId, currency],
    queryFn: () => {
      if (!workspaceId) throw new Error('No workspace ID');
      return analyticsService.getHourlyMetrics({ workspaceId, currency });
    },
    enabled: isReady && !!workspaceId,
    refetchInterval: 1000 * 60 * 5, // Refetch every 5 minutes
  });
}

export function useMetricsRefresh() {
  const { workspace } = useAuthStore();
  const queryClient = useQueryClient();
  const isReady = useSetupAnalyticsService();
  const workspaceId = workspace?.id;

  return useMutation({
    mutationFn: (date?: string) => {
      if (!workspaceId) throw new Error('No workspace ID');
      if (!isReady) throw new Error('Service not ready');
      return analyticsService.triggerMetricsRefresh({ workspaceId, date });
    },
    onSuccess: () => {
      // Set a slight delay before invalidating to give the backend time to process
      setTimeout(() => {
        // Invalidate all analytics queries to refetch fresh data
        queryClient.invalidateQueries({ queryKey: ['dashboard-summary'] });
        queryClient.invalidateQueries({ queryKey: ['revenue-chart'] });
        queryClient.invalidateQueries({ queryKey: ['customer-chart'] });
        queryClient.invalidateQueries({ queryKey: ['subscription-chart'] });
        queryClient.invalidateQueries({ queryKey: ['mrr-chart'] });
        queryClient.invalidateQueries({ queryKey: ['payment-metrics'] });
        queryClient.invalidateQueries({ queryKey: ['gas-fee-chart'] });
        queryClient.invalidateQueries({ queryKey: ['network-breakdown'] });
        queryClient.invalidateQueries({ queryKey: ['hourly-metrics'] });
      }, 2000); // 2 second delay to allow backend processing
    },
  });
}