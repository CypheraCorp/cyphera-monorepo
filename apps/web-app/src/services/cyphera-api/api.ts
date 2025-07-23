// No longer need CypheraSupabaseUser here as setRequestData is removed
import { clientLogger } from '@/lib/core/logger/logger-client';

// Define a type for the user context data needed for authenticated requests
export interface UserRequestContext {
  access_token?: string; // Make token non-optional in context if always required
  account_id?: string;
  workspace_id?: string;
  user_id?: string;
}

/**
 * Base API class for handling Cyphera API requests
 * Contains core functionality for authentication and request handling (STATELESS regarding user)
 */
export class CypheraAPI {
  protected baseUrl: string;
  protected apiKey: string;

  constructor() {
    const baseUrl = process.env.CYPHERA_API_BASE_URL || 'http://localhost:8000';
    if (baseUrl.endsWith('/api/v1')) {
      this.baseUrl = baseUrl;
    } else if (baseUrl.endsWith('/')) {
      this.baseUrl = `${baseUrl}api/v1`;
    } else {
      this.baseUrl = `${baseUrl}/api/v1`;
    }
    this.apiKey = process.env.CYPHERA_API_KEY || '';

    clientLogger.info('CypheraAPI initialized', { baseUrl: this.baseUrl });

    if (!this.apiKey) {
      // Only warn if intending to use public headers
      clientLogger.warn('CYPHERA_API_KEY environment variable is not set');
    }
  }

  /**
   * Returns the headers needed for authenticated API requests.
   * Accepts user context as a parameter.
   */
  protected getHeaders(context: UserRequestContext, csrfToken?: string): Record<string, string> {
    if (!context?.access_token) {
      // Add a check here, as the token is crucial for authenticated requests
      throw new Error(
        'UserRequestContext must include an access_token for authenticated requests.'
      );
    }
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      Authorization: `Bearer ${context.access_token}`, // Use token from context
    };

    // Add CSRF token if provided
    if (csrfToken) {
      headers['X-CSRF-Token'] = csrfToken;
    }

    if (context.account_id) {
      headers['X-Account-ID'] = context.account_id;
    }
    if (context.workspace_id) {
      headers['X-Workspace-ID'] = context.workspace_id;
    }
    if (context.user_id) {
      headers['X-User-ID'] = context.user_id;
    }

    return headers;
  }

  /**
   * Returns the headers for public API requests (using API key).
   */
  protected getPublicHeaders(csrfToken?: string): Record<string, string> {
    if (!this.apiKey) {
      throw new Error('Cannot make public API call: CYPHERA_API_KEY is not configured.');
    }
    const headers: Record<string, string> = {
      'X-API-Key': this.apiKey,
      'Content-Type': 'application/json',
      Accept: 'application/json',
    };

    // Add CSRF token if provided
    if (csrfToken) {
      headers['X-CSRF-Token'] = csrfToken;
    }

    return headers;
  }

  // Removed setRequestData method
  // setRequestData(user: CypheraSupabaseUser) { ... }

  /**
   * Helper method to handle API responses
   * Improved error handling
   */
  protected async handleResponse<T>(response: Response): Promise<T> {
    if (response.status === 204) {
      // Handle No Content responses gracefully
      // Return an empty object cast to T, assuming consumers handle it
      return {} as T;
    }

    const text = await response.text();
    let data: unknown;

    try {
      data = JSON.parse(text);
    } catch {
      // If parsing fails, check if it was an error response with non-JSON body
      if (!response.ok) {
        clientLogger.error(`API Error Response (non-JSON ${response.status})`, {
          text: text.substring(0, 100),
        });
        throw new Error(`API Error: ${response.status} - ${text.substring(0, 100)}`);
      }
      // If it was a success response but not JSON
      clientLogger.error('Failed to parse successful API response (non-JSON)', {
        text: text.substring(0, 100),
      });
      throw new Error(`Invalid API response format: ${text.substring(0, 100)}`);
    }

    // If parsing succeeded, check response status
    if (!response.ok) {
      clientLogger.error(`API Error Response (JSON ${response.status})`, { data });
      interface ErrorResponse {
        error?: string | { message?: string };
        message?: string;
      }
      const errorData = data as ErrorResponse;
      const message =
        (typeof errorData?.error === 'object' ? errorData.error.message : errorData?.error) ||
        errorData?.message ||
        `API Error: ${response.status}`;
      throw new Error(message);
    }

    // Return parsed data for successful responses
    return data as T;
  }
}
