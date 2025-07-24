import { useState, useCallback } from 'react';
import { RateLimitError } from '@/lib/api/rate-limit-handler';
import { useToast } from '@/components/ui/use-toast';

/**
 * Hook to handle rate limit errors in API calls
 * Provides automatic retry and user notification
 */
export function useRateLimitHandler() {
  const [isRetrying, setIsRetrying] = useState(false);
  const { toast } = useToast();

  const handleApiCall = useCallback(
    async <T,>(
      apiCall: () => Promise<T>,
      options?: {
        onSuccess?: (data: T) => void;
        onError?: (error: Error) => void;
        showToast?: boolean;
      }
    ): Promise<T | null> => {
      try {
        setIsRetrying(false);
        const result = await apiCall();
        options?.onSuccess?.(result);
        return result;
      } catch (error) {
        if (error instanceof RateLimitError) {
          setIsRetrying(true);
          
          if (options?.showToast !== false) {
            toast({
              title: 'Rate limit exceeded',
              description: `Please wait ${error.retryAfter} seconds before trying again.`,
              variant: 'destructive',
            });
          }
        }
        
        options?.onError?.(error as Error);
        
        // Re-throw non-rate-limit errors
        if (!(error instanceof RateLimitError)) {
          throw error;
        }
        
        return null;
      } finally {
        setIsRetrying(false);
      }
    },
    [toast]
  );

  return {
    handleApiCall,
    isRetrying,
  };
}

/**
 * Example usage in a component:
 * 
 * const MyComponent = () => {
 *   const { handleApiCall, isRetrying } = useRateLimitHandler();
 *   const [data, setData] = useState(null);
 *   
 *   const fetchData = async () => {
 *     const result = await handleApiCall(
 *       () => api.getData(),
 *       {
 *         onSuccess: (data) => setData(data),
 *         onError: (error) => console.error('Failed to fetch:', error),
 *       }
 *     );
 *   };
 *   
 *   return (
 *     <Button onClick={fetchData} disabled={isRetrying}>
 *       {isRetrying ? 'Retrying...' : 'Fetch Data'}
 *     </Button>
 *   );
 * };
 */