import { CypheraAPI, UserRequestContext } from './api';
import type { CreateCustomerRequest, CustomerResponse } from '@/types/customer';
import type { PaginatedResponse } from '@/types/common';
import { logger } from '@/lib/core/logger/logger-utils';
/**
 * Pagination parameters for customer list requests
 */
interface CustomerPaginationParams {
  limit?: number;
  page?: number;
}

/**
 * Customers API class for handling customer-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class CustomersAPI extends CypheraAPI {
  /**
   * Gets customers for the current account and workspace with pagination
   * @param context - The user request context (token, IDs)
   * @param params - Pagination parameters
   * @returns Promise with the customers response and pagination metadata
   * @throws Error if the request fails
   */
  async getCustomers(
    context: UserRequestContext,
    params?: CustomerPaginationParams
  ): Promise<PaginatedResponse<CustomerResponse>> {
    const queryParams = new URLSearchParams();
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.page) queryParams.append('page', params.page.toString());
    const url = `${this.baseUrl}/customers?${queryParams.toString()}`;

    try {
      return await this.fetchWithRateLimit<PaginatedResponse<CustomerResponse>>(url, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Customers fetch failed:', error);
      throw error;
    }
  }

  /**
   * Gets a single customer by ID
   * @param context - The user request context (token, IDs)
   * @param customerId - The ID of the customer to fetch
   * @returns Promise with the customer response
   * @throws Error if the request fails
   */
  async getCustomerById(
    context: UserRequestContext,
    customerId: string
  ): Promise<CustomerResponse> {
    try {
      return await this.fetchWithRateLimit<CustomerResponse>(`${this.baseUrl}/customers/${customerId}`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Customer fetch failed:', error);
      throw error;
    }
  }

  /**
   * Creates a new customer
   * @param context - The user request context (token, IDs)
   * @param customerData - The customer data to create
   * @returns Promise with the created customer response
   * @throws Error if the request fails
   */
  async createCustomer(
    context: UserRequestContext,
    customerData: CreateCustomerRequest
  ): Promise<CustomerResponse> {
    try {
      return await this.fetchWithRateLimit<CustomerResponse>(`${this.baseUrl}/customers`, {
        method: 'POST',
        headers: this.getHeaders(context),
        body: JSON.stringify(customerData),
      });
    } catch (error) {
      logger.error('Customer creation failed:', error);
      throw error;
    }
  }

  /**
   * Updates a customer
   * @param context - The user request context (token, IDs)
   * @param customerId - The ID of the customer to update
   * @param customerData - The customer data to update
   * @returns Promise with the updated customer response
   * @throws Error if the request fails
   */
  async updateCustomer(
    context: UserRequestContext,
    customerId: string,
    customerData: Partial<CreateCustomerRequest>
  ): Promise<CustomerResponse> {
    try {
      return await this.fetchWithRateLimit<CustomerResponse>(`${this.baseUrl}/customers/${customerId}`, {
        method: 'PUT',
        headers: this.getHeaders(context),
        body: JSON.stringify(customerData),
      });
    } catch (error) {
      logger.error('Customer update failed:', error);
      throw error;
    }
  }

  /**
   * Deletes a customer
   * @param context - The user request context (token, IDs)
   * @param customerId - The ID of the customer to delete
   * @returns Promise<void>
   * @throws Error if the request fails
   */
  async deleteCustomer(context: UserRequestContext, customerId: string): Promise<void> {
    try {
      await this.fetchWithRateLimit<void>(`${this.baseUrl}/customers/${customerId}`, {
        method: 'DELETE',
        headers: this.getHeaders(context),
      });
    } catch (error) {
      logger.error('Customer deletion failed:', error);
      throw error;
    }
  }
}
