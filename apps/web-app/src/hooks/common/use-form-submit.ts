import { useState } from 'react';
import { useToast } from '@/components/ui/use-toast';
import { logger } from '@/lib/core/logger/logger-utils';
import { startProgress, stopProgress } from '@/components/ui/nprogress';

interface UseFormSubmitOptions {
  onSuccess?: (data?: unknown) => void;
  onError?: (error: Error) => void;
  successMessage?: string;
  errorMessage?: string;
  showProgress?: boolean;
}

export function useFormSubmit(options?: UseFormSubmitOptions) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { toast } = useToast();

  const handleSubmit = async (submitFn: () => Promise<unknown>) => {
    if (isSubmitting) return;

    setIsSubmitting(true);

    if (options?.showProgress) {
      startProgress();
    }

    try {
      const result = await submitFn();

      if (options?.successMessage) {
        toast({
          title: 'Success',
          description: options.successMessage,
        });
      }

      options?.onSuccess?.(result);
      return result;
    } catch (error) {
      const err = error instanceof Error ? error : new Error('Submission failed');

      logger.error('Form submission failed', { error: err });

      toast({
        title: 'Error',
        description: options?.errorMessage || err.message,
        variant: 'destructive',
      });

      options?.onError?.(err);
      throw err;
    } finally {
      setIsSubmitting(false);

      if (options?.showProgress) {
        stopProgress();
      }
    }
  };

  return {
    handleSubmit,
    isSubmitting,
  };
}
