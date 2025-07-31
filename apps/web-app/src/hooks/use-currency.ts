import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/store/auth';
import { currencyService } from '@/services/cyphera-api/currency';
import type { Currency } from '@/types/analytics';
import type { UserRequestContext } from '@/services/cyphera-api/api';
import { useEffect } from 'react';

function useSetupCurrencyService() {
  const { accessToken, account, workspace, user } = useAuthStore();
  
  useEffect(() => {
    if (accessToken && account && workspace && user) {
      const context: UserRequestContext = {
        access_token: accessToken,
        account_id: account.id,
        workspace_id: workspace.id,
        user_id: user.id,
      };
      currencyService.setUserContext(context);
    }
  }, [accessToken, account, workspace, user]);
  
  return !!accessToken && !!workspace;
}

export function useCurrencies() {
  const { workspace } = useAuthStore();
  const isReady = useSetupCurrencyService();
  const workspaceId = workspace?.id;

  return useQuery<Currency[]>({
    queryKey: ['currencies', workspaceId],
    queryFn: () => {
      if (!workspaceId) throw new Error('No workspace ID');
      return currencyService.getCurrencies(workspaceId);
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 60, // 1 hour
  });
}

export function useDefaultCurrency() {
  const { workspace } = useAuthStore();
  const isReady = useSetupCurrencyService();
  const workspaceId = workspace?.id;

  return useQuery<Currency | null>({
    queryKey: ['default-currency', workspaceId],
    queryFn: () => {
      if (!workspaceId) throw new Error('No workspace ID');
      return currencyService.getDefaultCurrency(workspaceId);
    },
    enabled: isReady && !!workspaceId,
    staleTime: 1000 * 60 * 60, // 1 hour
  });
}

export function useSetDefaultCurrency() {
  const { workspace } = useAuthStore();
  const queryClient = useQueryClient();
  const isReady = useSetupCurrencyService();
  const workspaceId = workspace?.id;

  return useMutation({
    mutationFn: (currencyCode: string) => {
      if (!workspaceId) throw new Error('No workspace ID');
      if (!isReady) throw new Error('Service not ready');
      return currencyService.setDefaultCurrency(workspaceId, currencyCode);
    },
    onSuccess: () => {
      // Invalidate currency queries to refetch
      queryClient.invalidateQueries({ queryKey: ['currencies', workspaceId] });
      queryClient.invalidateQueries({ queryKey: ['default-currency', workspaceId] });
      // Also invalidate analytics data that depends on currency
      queryClient.invalidateQueries({ queryKey: ['dashboard-summary', workspaceId] });
      queryClient.invalidateQueries({ queryKey: ['analytics'] });
    },
  });
}

export function useCurrency() {
  const { data: currencies = [], isLoading: currenciesLoading } = useCurrencies();
  const { data: defaultCurrency, isLoading: defaultLoading } = useDefaultCurrency();
  const setDefaultCurrency = useSetDefaultCurrency();

  return {
    currencies,
    defaultCurrency,
    isLoading: currenciesLoading || defaultLoading,
    setDefaultCurrency: setDefaultCurrency.mutate,
    isSettingDefault: setDefaultCurrency.isPending,
  };
}