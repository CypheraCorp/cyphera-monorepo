'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import { CSRF_TOKEN_HEADER } from '@/lib/security/csrf';
import { logger } from '@/lib/core/logger/logger-utils';

interface CSRFContextType {
  csrfToken: string | null;
  isLoading: boolean;
  refreshToken: () => Promise<void>;
  getHeaders: (headers?: HeadersInit) => HeadersInit;
}

const CSRFContext = createContext<CSRFContextType | undefined>(undefined);

export function CSRFProvider({ children }: { children: React.ReactNode }) {
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
      logger.info('CSRF token fetched successfully');
    } catch (error) {
      logger.error('Failed to fetch CSRF token', error);
    } finally {
      setIsLoading(false);
    }
  };

  const refreshToken = async () => {
    setIsLoading(true);
    await fetchCSRFToken();
  };

  const getHeaders = (headers: HeadersInit = {}): HeadersInit => {
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

  return (
    <CSRFContext.Provider value={{ csrfToken, isLoading, refreshToken, getHeaders }}>
      {children}
    </CSRFContext.Provider>
  );
}

export function useCSRF() {
  const context = useContext(CSRFContext);
  if (context === undefined) {
    throw new Error('useCSRF must be used within a CSRFProvider');
  }
  return context;
}