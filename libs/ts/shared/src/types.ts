// Shared TypeScript types that can be used across apps

export interface BaseResponse<T = any> {
  data: T;
  error?: string;
  success: boolean;
}

export interface PaginatedResponse<T> extends BaseResponse<T[]> {
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}