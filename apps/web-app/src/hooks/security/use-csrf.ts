import { useEffect, useState } from 'react';
import { CSRF_TOKEN_HEADER } from '@/lib/security/csrf';
import { logger } from '@/lib/core/logger/logger-utils';

/**
 * Hook to manage CSRF tokens
 */
export function useCSRF() {
  const [csrfToken, setCSRFToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchCSRFToken();
  }, []);

  const fetchCSRFToken = async () => {
    try {
      const response = await fetch('/api/auth/csrf', {
        method: 'GET',
        credentials: 'include',
      });

      if (!response.ok) {
        throw new Error('Failed to fetch CSRF token');
      }

      const data = await response.json();
      setCSRFToken(data.token);
    } catch (error) {
      logger.error('Failed to fetch CSRF token', error);
    } finally {
      setIsLoading(false);
    }
  };

  /**
   * Add CSRF token to request headers
   */
  const addCSRFHeader = (headers: HeadersInit = {}): HeadersInit => {
    if (!csrfToken) return headers;

    if (headers instanceof Headers) {
      headers.set(CSRF_TOKEN_HEADER, csrfToken);
      return headers;
    }

    return {
      ...headers,
      [CSRF_TOKEN_HEADER]: csrfToken,
    };
  };

  /**
   * Refresh CSRF token
   */
  const refreshToken = () => {
    fetchCSRFToken();
  };

  return {
    csrfToken,
    isLoading,
    addCSRFHeader,
    refreshToken,
  };
}