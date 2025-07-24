import { useCallback } from 'react';
import { generateCorrelationId } from '@/lib/utils/correlation';

/**
 * Hook to manage correlation IDs for request tracking
 */
export function useCorrelationId() {
  // Generate a new correlation ID for a request chain
  const createCorrelationId = useCallback(() => {
    return generateCorrelationId();
  }, []);

  // Extract correlation ID from an error response
  const getCorrelationIdFromError = useCallback((error: any): string | undefined => {
    if (error?.correlation_id) {
      return error.correlation_id;
    }
    return undefined;
  }, []);

  // Log an error with its correlation ID
  const logError = useCallback((message: string, error: any, additionalData?: Record<string, any>) => {
    const correlationId = getCorrelationIdFromError(error);
    
    console.error(message, {
      error: error?.error || error?.message || error,
      correlationId,
      timestamp: new Date().toISOString(),
      ...additionalData,
    });

    return correlationId;
  }, [getCorrelationIdFromError]);

  return {
    createCorrelationId,
    getCorrelationIdFromError,
    logError,
  };
}