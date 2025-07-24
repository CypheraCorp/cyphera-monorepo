// API Error Response type
export interface ApiErrorResponse {
  error: string;
  correlation_id?: string;
}

// API Success Response wrapper
export interface ApiResponse<T> {
  data?: T;
  error?: ApiErrorResponse;
}

// Request headers type with correlation ID support
export interface ApiRequestHeaders {
  'X-Correlation-ID'?: string;
  [key: string]: string | undefined;
}