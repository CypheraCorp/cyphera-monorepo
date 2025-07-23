import { useState, useCallback } from 'react';
import { logger } from '@/lib/core/logger/logger-utils';

interface UseAsyncOptions {
  onSuccess?: (data: unknown) => void;
  onError?: (error: Error) => void;
}

export function useAsync<T = unknown>(options?: UseAsyncOptions) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [data, setData] = useState<T | null>(null);

  const execute = useCallback(
    async (asyncFunction: () => Promise<T>) => {
      setIsLoading(true);
      setError(null);

      try {
        const result = await asyncFunction();
        setData(result);
        options?.onSuccess?.(result);
        return result;
      } catch (err) {
        const error = err instanceof Error ? err : new Error('An error occurred');
        setError(error);
        logger.error('Async operation failed', error);
        options?.onError?.(error);
        throw error;
      } finally {
        setIsLoading(false);
      }
    },
    [options]
  );

  const reset = useCallback(() => {
    setIsLoading(false);
    setError(null);
    setData(null);
  }, []);

  return {
    execute,
    reset,
    isLoading,
    error,
    data,
  };
}
