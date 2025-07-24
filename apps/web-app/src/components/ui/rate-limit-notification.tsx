'use client';

import { useEffect, useState } from 'react';
import { AlertCircle, Clock } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Progress } from '@/components/ui/progress';
import { RateLimitError } from '@/lib/api/rate-limit-handler';

interface RateLimitNotificationProps {
  error: RateLimitError | null;
  onRetry?: () => void;
}

export function RateLimitNotification({ error, onRetry }: RateLimitNotificationProps) {
  const [timeRemaining, setTimeRemaining] = useState(0);

  useEffect(() => {
    if (!error) return;

    // Set initial time remaining
    const initialTime = error.retryAfter;
    setTimeRemaining(initialTime);

    // Update countdown every second
    const interval = setInterval(() => {
      setTimeRemaining((prev) => {
        if (prev <= 1) {
          clearInterval(interval);
          // Auto-retry if handler provided
          if (onRetry) {
            onRetry();
          }
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [error, onRetry]);

  if (!error) return null;

  const progress = error.retryAfter > 0 
    ? ((error.retryAfter - timeRemaining) / error.retryAfter) * 100
    : 0;

  return (
    <Alert className="mb-4">
      <AlertCircle className="h-4 w-4" />
      <AlertTitle>Rate Limit Exceeded</AlertTitle>
      <AlertDescription className="space-y-2">
        <p>
          You've made too many requests. Please wait before trying again.
        </p>
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>Retrying in {timeRemaining} seconds...</span>
        </div>
        <Progress value={progress} className="h-2" />
        <div className="text-xs text-muted-foreground">
          Limit: {error.limit} requests | Remaining: {error.remaining}
        </div>
      </AlertDescription>
    </Alert>
  );
}

/**
 * Hook to manage rate limit notifications
 */
export function useRateLimitNotification() {
  const [rateLimitError, setRateLimitError] = useState<RateLimitError | null>(null);

  const handleError = (error: unknown) => {
    if (error instanceof RateLimitError) {
      setRateLimitError(error);
      return true;
    }
    return false;
  };

  const clearError = () => {
    setRateLimitError(null);
  };

  return {
    rateLimitError,
    handleError,
    clearError,
    RateLimitNotification: (props: Omit<RateLimitNotificationProps, 'error'>) => (
      <RateLimitNotification error={rateLimitError} {...props} />
    ),
  };
}