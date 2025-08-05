import { CypheraAPI, UserRequestContext } from './api';
import type { PaginatedResponse, PaginationParams } from '@/types/common';
import type { PaymentResponse } from '@/types/payment';
import { logger } from '@/lib/core/logger/logger-utils';

/**
 * Payments API class for handling payment-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class PaymentsAPI extends CypheraAPI {
  /**
   * Gets payments for the current workspace with pagination
   * @param context - The user request context (token, IDs)
   * @param params - Pagination and filter parameters
   * @returns Promise with the payments response and pagination metadata
   * @throws Error if the request fails
   */
  async getPayments(
    context: UserRequestContext,
    params?: PaginationParams & {
      status?: string;
      customer_id?: string;
      payment_method?: string;
      start_date?: string;
      end_date?: string;
    }
  ): Promise<PaginatedResponse<PaymentResponse>> {
    const queryParams = new URLSearchParams();
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.status) queryParams.append('status', params.status);
    if (params?.customer_id) queryParams.append('customer_id', params.customer_id);
    if (params?.payment_method) queryParams.append('payment_method', params.payment_method);
    if (params?.start_date) queryParams.append('start_date', params.start_date);
    if (params?.end_date) queryParams.append('end_date', params.end_date);
    
    const url = `${this.baseUrl}/payments?${queryParams.toString()}`;

    try {
      logger.info('Fetching payments from:', { url });
      return await this.fetchWithRateLimit<PaginatedResponse<PaymentResponse>>(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Payments fetch failed:', error);
      throw error;
    }
  }

  /**
   * Gets a single payment by ID
   * @param context - The user request context (token, IDs)
   * @param paymentId - The ID of the payment to fetch
   * @returns Promise with the payment response
   * @throws Error if the request fails
   */
  async getPaymentById(
    context: UserRequestContext,
    paymentId: string
  ): Promise<PaymentResponse> {
    try {
      return await this.fetchWithRateLimit<PaymentResponse>(`${this.baseUrl}/payments/${paymentId}`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Payment fetch failed:', error);
      throw error;
    }
  }
}