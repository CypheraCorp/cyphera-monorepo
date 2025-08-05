import { PaginatedResponse, PaginationParams } from '@/types/common';
import { CypheraAPI, UserRequestContext } from './api';
import type { 
  Invoice, 
  InvoiceListParams, 
  InvoiceListResponse,
  InvoiceActivity,
  BulkInvoiceGenerationResult,
  InvoiceStatsResponse,
  InvoiceStatus
} from '@/types/invoice';
import { logger } from '@/lib/core/logger/logger-utils';

/**
 * Invoices API class for handling invoice-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class InvoicesAPI extends CypheraAPI {
  /**
   * Gets invoices for the current workspace with pagination and optional filters
   * @param context - The user request context (token, IDs)
   * @param params - Query parameters including pagination and filters
   * @returns Promise with the invoices response
   * @throws Error if the request fails
   */
  async getInvoices(
    context: UserRequestContext,
    params?: InvoiceListParams
  ): Promise<InvoiceListResponse> {
    const queryParams = new URLSearchParams();
    
    // Add pagination params
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.offset) queryParams.append('offset', params.offset.toString());
    
    // Add filter params
    if (params?.status) queryParams.append('status', params.status);
    if (params?.customer_id) queryParams.append('customer_id', params.customer_id);
    
    const url = `${this.baseUrl}/invoices?${queryParams.toString()}`;

    try {
      const response = await this.fetchWithRateLimit<{ invoices: Invoice[]; total: number }>(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });

      return {
        invoices: response.invoices || [],
        total: response.total || 0,
        limit: params?.limit || 20,
        offset: params?.offset || 0,
      };
    } catch (error) {
      logger.error('Invoices fetch failed:', error);
      throw error;
    }
  }

  /**
   * Gets a single invoice by ID
   * @param context - The user request context (token, IDs)
   * @param invoiceId - The ID of the invoice to fetch
   * @returns Promise with the invoice response
   * @throws Error if the request fails
   */
  async getInvoiceById(
    context: UserRequestContext,
    invoiceId: string
  ): Promise<Invoice> {
    try {
      return await this.fetchWithRateLimit<Invoice>(`${this.baseUrl}/invoices/${invoiceId}`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Invoice fetch failed:', error);
      throw error;
    }
  }

  /**
   * Gets invoice activity history
   * @param context - The user request context (token, IDs)
   * @param invoiceId - The ID of the invoice
   * @param params - Pagination parameters
   * @returns Promise with the activities response
   * @throws Error if the request fails
   */
  async getInvoiceActivity(
    context: UserRequestContext,
    invoiceId: string,
    params?: { limit?: number; offset?: number }
  ): Promise<{ activities: InvoiceActivity[]; limit: number; offset: number }> {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.offset) queryParams.append('offset', params.offset.toString());
    
    const url = `${this.baseUrl}/invoices/${invoiceId}/activity?${queryParams.toString()}`;

    try {
      return await this.fetchWithRateLimit<{ activities: InvoiceActivity[]; limit: number; offset: number }>(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Invoice activity fetch failed:', error);
      throw error;
    }
  }

  /**
   * Voids an invoice
   * @param context - The user request context (token, IDs)
   * @param invoiceId - The ID of the invoice to void
   * @returns Promise with the updated invoice
   * @throws Error if the request fails
   */
  async voidInvoice(
    context: UserRequestContext,
    invoiceId: string
  ): Promise<Invoice> {
    try {
      return await this.fetchWithRateLimit<Invoice>(`${this.baseUrl}/invoices/${invoiceId}/void`, {
        method: 'POST',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Void invoice failed:', error);
      throw error;
    }
  }

  /**
   * Marks an invoice as paid
   * @param context - The user request context (token, IDs)
   * @param invoiceId - The ID of the invoice to mark as paid
   * @returns Promise with the updated invoice
   * @throws Error if the request fails
   */
  async markInvoicePaid(
    context: UserRequestContext,
    invoiceId: string
  ): Promise<Invoice> {
    try {
      return await this.fetchWithRateLimit<Invoice>(`${this.baseUrl}/invoices/${invoiceId}/mark-paid`, {
        method: 'POST',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Mark invoice paid failed:', error);
      throw error;
    }
  }

  /**
   * Marks an invoice as uncollectible
   * @param context - The user request context (token, IDs)
   * @param invoiceId - The ID of the invoice to mark as uncollectible
   * @returns Promise with the updated invoice
   * @throws Error if the request fails
   */
  async markInvoiceUncollectible(
    context: UserRequestContext,
    invoiceId: string
  ): Promise<Invoice> {
    try {
      return await this.fetchWithRateLimit<Invoice>(`${this.baseUrl}/invoices/${invoiceId}/mark-uncollectible`, {
        method: 'POST',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Mark invoice uncollectible failed:', error);
      throw error;
    }
  }

  /**
   * Duplicates an invoice
   * @param context - The user request context (token, IDs)
   * @param invoiceId - The ID of the invoice to duplicate
   * @returns Promise with the new invoice
   * @throws Error if the request fails
   */
  async duplicateInvoice(
    context: UserRequestContext,
    invoiceId: string
  ): Promise<Invoice> {
    try {
      return await this.fetchWithRateLimit<Invoice>(`${this.baseUrl}/invoices/${invoiceId}/duplicate`, {
        method: 'POST',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Duplicate invoice failed:', error);
      throw error;
    }
  }

  /**
   * Bulk generates invoices for due subscriptions
   * @param context - The user request context (token, IDs)
   * @param endDate - End date for invoice generation
   * @param maxInvoices - Maximum number of invoices to generate
   * @returns Promise with the bulk generation result
   * @throws Error if the request fails
   */
  async bulkGenerateInvoices(
    context: UserRequestContext,
    endDate: string,
    maxInvoices?: number
  ): Promise<BulkInvoiceGenerationResult> {
    try {
      return await this.fetchWithRateLimit<BulkInvoiceGenerationResult>(`${this.baseUrl}/invoices/bulk-generate`, {
        method: 'POST',
        headers: this.getHeaders(context),
        body: JSON.stringify({
          end_date: endDate,
          max_invoices: maxInvoices || 100,
        }),
      });
    } catch (error) {
      logger.error('Bulk generate invoices failed:', error);
      throw error;
    }
  }

  /**
   * Gets invoice statistics
   * @param context - The user request context (token, IDs)
   * @param startDate - Start date for stats
   * @param endDate - End date for stats
   * @returns Promise with the invoice stats
   * @throws Error if the request fails
   */
  async getInvoiceStats(
    context: UserRequestContext,
    startDate?: string,
    endDate?: string
  ): Promise<InvoiceStatsResponse> {
    const queryParams = new URLSearchParams();
    if (startDate) queryParams.append('start_date', startDate);
    if (endDate) queryParams.append('end_date', endDate);
    
    const url = `${this.baseUrl}/invoices/stats?${queryParams.toString()}`;

    try {
      return await this.fetchWithRateLimit<InvoiceStatsResponse>(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Invoice stats fetch failed:', error);
      throw error;
    }
  }
}