'use client';

import { MainSidebar } from '@/components/layout/main-sidebar';
import { Button } from '@/components/ui/button';
import { AlertCircle } from 'lucide-react';
import { useEffect } from 'react';
import { logger } from '@/lib/core/logger/logger-utils';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Log the error to an error reporting service
    logger.error('Customer page error', error);
  }, [error]);

  return (
    <div className="flex h-screen bg-white dark:bg-neutral-900">
      <MainSidebar />
      <main className="flex-1 overflow-y-auto">
        <div className="container mx-auto p-8">
          <div className="flex flex-col items-center justify-center min-h-[60vh] text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mb-4" />
            <h2 className="text-2xl font-bold mb-2">Something went wrong!</h2>
            <p className="text-muted-foreground mb-6 max-w-md">
              {error.message ||
                'An error occurred while loading the customers page. Please try again.'}
            </p>
            <Button onClick={reset}>Try again</Button>
          </div>
        </div>
      </main>
    </div>
  );
}
