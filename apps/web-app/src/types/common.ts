/**
 * Common pagination response interface
 */
export interface PaginatedResponse<T> {
  data: T[];
  object: string;
  has_more: boolean;
  pagination: Pagination;
}

export interface Pagination {
  current_page: number;
  per_page: number;
  total_items: number;
  total_pages: number;
}

/**
 * Pagination parameters for list requests
 */
export interface PaginationParams {
  page?: number;
  limit?: number;
}

/**
 * API Error response structure
 */
export interface APIError {
  error: string;
  code?: string;
  details?: Record<string, string>;
}
