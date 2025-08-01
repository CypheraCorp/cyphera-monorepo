import { useState, useCallback, useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { CypheraAPIClient } from '@/services/cyphera-api';
import { useAuth } from '@/hooks/auth/use-auth-user';
import type { Invoice, InvoiceListParams, InvoiceStatus } from '@/types/invoice';
import type { UserRequestContext } from '@/services/cyphera-api/api';
import { logger } from '@/lib/core/logger/logger-utils';
import { useCorrelationId } from '@/hooks/utils/use-correlation-id';

const api = new CypheraAPIClient();

interface UseInvoicesDataParams {
  limit?: number;
  initialStatus?: InvoiceStatus;
  initialCustomerId?: string;
}

export function useInvoicesData({
  limit = 20,
  initialStatus,
  initialCustomerId,
}: UseInvoicesDataParams = {}) {
  const { user, workspace } = useAuth();
  const queryClient = useQueryClient();
  const correlationId = useCorrelationId();

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

  // Filter state
  const [statusFilter, setStatusFilter] = useState<InvoiceStatus | undefined>(initialStatus);
  const [customerIdFilter, setCustomerIdFilter] = useState<string | undefined>(initialCustomerId);

  const offset = (currentPage - 1) * limit;

  // Build query params
  const queryParams: InvoiceListParams = {
    limit,
    offset,
    ...(statusFilter && { status: statusFilter }),
    ...(customerIdFilter && { customer_id: customerIdFilter }),
  };

  // Query key that includes all params
  const queryKey = ['invoices', queryParams];

  // Fetch invoices query
  const {
    data,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey,
    queryFn: async () => {
      if (!user?.access_token || !workspace?.id) {
        throw new Error('No authentication session');
      }

      const context: UserRequestContext = {
        access_token: user.access_token,
        workspace_id: workspace.id,
        account_id: user.account_id,
        user_id: user.user_id,
      };

      logger.log('Fetching invoices with params:', queryParams);
      const response = await api.invoices.getInvoices(context, queryParams);
      
      // Update total pages based on response
      const totalPagesCount = Math.ceil(response.total / limit);
      setTotalPages(totalPagesCount);
      
      return response;
    },
    enabled: !!user?.access_token && !!workspace?.id,
    staleTime: 30000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
  });

  // Navigation functions
  const goToPage = useCallback((page: number) => {
    if (page >= 1 && page <= totalPages) {
      setCurrentPage(page);
    }
  }, [totalPages]);

  const goToNextPage = useCallback(() => {
    goToPage(currentPage + 1);
  }, [currentPage, goToPage]);

  const goToPreviousPage = useCallback(() => {
    goToPage(currentPage - 1);
  }, [currentPage, goToPage]);

  // Filter functions
  const updateStatusFilter = useCallback((status: InvoiceStatus | undefined) => {
    setStatusFilter(status);
    setCurrentPage(1); // Reset to first page when filter changes
  }, []);

  const updateCustomerIdFilter = useCallback((customerId: string | undefined) => {
    setCustomerIdFilter(customerId);
    setCurrentPage(1); // Reset to first page when filter changes
  }, []);

  const clearFilters = useCallback(() => {
    setStatusFilter(undefined);
    setCustomerIdFilter(undefined);
    setCurrentPage(1);
  }, []);

  // Invalidate and refetch
  const invalidateInvoices = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['invoices'] });
  }, [queryClient]);

  // Get invoice by ID from cache if available
  const getInvoiceFromCache = useCallback((invoiceId: string): Invoice | undefined => {
    const cachedData = queryClient.getQueryData<{ invoices: Invoice[] }>(queryKey);
    return cachedData?.invoices.find(inv => inv.id === invoiceId);
  }, [queryClient, queryKey]);

  return {
    // Data
    invoices: data?.invoices || [],
    total: data?.total || 0,
    
    // Loading states
    isLoading,
    error,
    
    // Pagination
    currentPage,
    totalPages,
    hasNextPage: currentPage < totalPages,
    hasPreviousPage: currentPage > 1,
    goToPage,
    goToNextPage,
    goToPreviousPage,
    
    // Filters
    statusFilter,
    customerIdFilter,
    updateStatusFilter,
    updateCustomerIdFilter,
    clearFilters,
    hasActiveFilters: !!statusFilter || !!customerIdFilter,
    
    // Actions
    refetch,
    invalidateInvoices,
    getInvoiceFromCache,
    
    // Computed
    isEmpty: !isLoading && (!data?.invoices || data.invoices.length === 0),
    pageInfo: {
      from: offset + 1,
      to: Math.min(offset + limit, data?.total || 0),
      total: data?.total || 0,
    },
  };
}

// Hook for fetching a single invoice
export function useInvoiceById(invoiceId: string | undefined) {
  const { user, workspace } = useAuth();
  const correlationId = useCorrelationId();

  return useQuery({
    queryKey: ['invoice', invoiceId],
    queryFn: async () => {
      if (!user?.access_token || !workspace?.id || !invoiceId) {
        throw new Error('Missing required parameters');
      }

      const context: UserRequestContext = {
        access_token: user.access_token,
        workspace_id: workspace.id,
        account_id: user.account_id,
        user_id: user.user_id,
      };

      return api.invoices.getInvoiceById(context, invoiceId);
    },
    enabled: !!user?.access_token && !!workspace?.id && !!invoiceId,
    staleTime: 30000, // 30 seconds
  });
}

// Hook for invoice actions
export function useInvoiceActions() {
  const { user, workspace } = useAuth();
  const queryClient = useQueryClient();
  const correlationId = useCorrelationId();

  const getContext = useCallback(() => {
    if (!user?.access_token || !workspace?.id) {
      throw new Error('No authentication session');
    }

    return {
      access_token: user.access_token,
      workspace_id: workspace.id,
      account_id: user.account_id,
      user_id: user.user_id,
    };
  }, [user, workspace, correlationId]);

  const voidInvoice = useCallback(async (invoiceId: string) => {
    const context = getContext();
    const result = await api.invoices.voidInvoice(context, invoiceId);
    
    // Invalidate queries
    queryClient.invalidateQueries({ queryKey: ['invoices'] });
    queryClient.invalidateQueries({ queryKey: ['invoice', invoiceId] });
    
    return result;
  }, [getContext, queryClient]);

  const markInvoicePaid = useCallback(async (invoiceId: string) => {
    const context = getContext();
    const result = await api.invoices.markInvoicePaid(context, invoiceId);
    
    // Invalidate queries
    queryClient.invalidateQueries({ queryKey: ['invoices'] });
    queryClient.invalidateQueries({ queryKey: ['invoice', invoiceId] });
    
    return result;
  }, [getContext, queryClient]);

  const markInvoiceUncollectible = useCallback(async (invoiceId: string) => {
    const context = getContext();
    const result = await api.invoices.markInvoiceUncollectible(context, invoiceId);
    
    // Invalidate queries
    queryClient.invalidateQueries({ queryKey: ['invoices'] });
    queryClient.invalidateQueries({ queryKey: ['invoice', invoiceId] });
    
    return result;
  }, [getContext, queryClient]);

  const duplicateInvoice = useCallback(async (invoiceId: string) => {
    const context = getContext();
    const result = await api.invoices.duplicateInvoice(context, invoiceId);
    
    // Invalidate invoices list
    queryClient.invalidateQueries({ queryKey: ['invoices'] });
    
    return result;
  }, [getContext, queryClient]);

  return {
    voidInvoice,
    markInvoicePaid,
    markInvoiceUncollectible,
    duplicateInvoice,
  };
}