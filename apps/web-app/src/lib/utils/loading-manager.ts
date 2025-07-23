import { startProgress, stopProgress } from '@/components/ui/nprogress';

// Loading state manager to handle multiple concurrent requests
class LoadingManager {
  private activeRequests = new Set<string>();
  private progressTimeout: NodeJS.Timeout | null = null;

  start(requestId: string) {
    this.activeRequests.add(requestId);

    // Clear any existing timeout
    if (this.progressTimeout) {
      clearTimeout(this.progressTimeout);
      this.progressTimeout = null;
    }

    // Start progress if this is the first request
    if (this.activeRequests.size === 1) {
      startProgress();
    }
  }

  stop(requestId: string) {
    this.activeRequests.delete(requestId);

    // Only stop progress if no more active requests
    if (this.activeRequests.size === 0) {
      // Delay stop to prevent flashing on quick requests
      this.progressTimeout = setTimeout(() => {
        stopProgress();
      }, 100);
    }
  }

  clear() {
    this.activeRequests.clear();
    if (this.progressTimeout) {
      clearTimeout(this.progressTimeout);
    }
    stopProgress();
  }
}

export const loadingManager = new LoadingManager();

// Wrapper for fetch with loading state
export async function fetchWithProgress<T>(url: string, options?: RequestInit): Promise<T> {
  const requestId = `${url}-${Date.now()}`;

  try {
    loadingManager.start(requestId);
    const response = await fetch(url, options);

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.json();
  } finally {
    loadingManager.stop(requestId);
  }
}

// Hook for managing loading states in components
import { useState, useCallback } from 'react';

export function useLoadingState() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const execute = useCallback(async <T>(promise: Promise<T>): Promise<T | undefined> => {
    setIsLoading(true);
    setError(null);

    const requestId = `component-${Date.now()}`;
    loadingManager.start(requestId);

    try {
      const result = await promise;
      return result;
    } catch (err) {
      setError(err instanceof Error ? err : new Error('An error occurred'));
      throw err;
    } finally {
      setIsLoading(false);
      loadingManager.stop(requestId);
    }
  }, []);

  return { isLoading, error, execute };
}
