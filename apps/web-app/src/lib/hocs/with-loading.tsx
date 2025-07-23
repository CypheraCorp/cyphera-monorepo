'use client';

import { ComponentType, ReactNode, useEffect, useState } from 'react';
import { LoadingSpinner, PageLoadingSkeleton } from '@/components/ui/loading-states';
import { useLoadingState } from '@/lib/utils/loading-manager';
import { startProgress, stopProgress } from '@/components/ui/nprogress';
import { logger } from '@/lib/core/logger/logger-utils';

export interface WithLoadingProps {
  isLoading?: boolean;
  setLoading?: (loading: boolean) => void;
  loadingState?: ReturnType<typeof useLoadingState>;
}

export interface WithLoadingOptions {
  showGlobalProgress?: boolean;
  showLocalSpinner?: boolean;
  delay?: number;
  minDuration?: number;
  fallback?: ReactNode;
  errorFallback?: ReactNode;
  LoadingComponent?: ComponentType<{ message?: string }>;
  loadingMessage?: string;
  transparentOverlay?: boolean;
  fullScreen?: boolean;
}

/**
 * Higher-Order Component that adds loading state management to a component
 *
 * @example
 * // Basic usage
 * export default withLoading(MyComponent);
 *
 * // With custom options
 * export default withLoading(MyComponent, {
 *   showGlobalProgress: true,
 *   delay: 200,
 *   loadingMessage: 'Loading data...'
 * });
 *
 * // Custom loading component
 * export default withLoading(MyComponent, {
 *   LoadingComponent: CustomSkeleton,
 *   fullScreen: true
 * });
 *
 * // Inside the component
 * function MyComponent({ setLoading, loadingState }: Props & WithLoadingProps) {
 *   const fetchData = async () => {
 *     setLoading(true);
 *     try {
 *       const data = await api.getData();
 *     } finally {
 *       setLoading(false);
 *     }
 *   };
 *
 *   // Or use loadingState.execute
 *   const handleSubmit = () => {
 *     loadingState.execute(api.submitData());
 *   };
 * }
 */
export function withLoading<P extends object>(
  Component: ComponentType<P & WithLoadingProps>,
  options: WithLoadingOptions = {}
): ComponentType<P> {
  const {
    showGlobalProgress = false,
    showLocalSpinner = true,
    delay = 0,
    minDuration = 0,
    fallback,
    errorFallback,
    LoadingComponent = LoadingSpinner,
    loadingMessage = 'Loading...',
    transparentOverlay = false,
    fullScreen = false,
  } = options;

  const WrappedComponent = (props: P) => {
    const loadingState = useLoadingState();
    const [localLoading, setLocalLoading] = useState(false);
    const [showLoading, setShowLoading] = useState(false);
    const [startTime, setStartTime] = useState<number | null>(null);

    // Combined loading state
    const isLoading = localLoading || loadingState.isLoading;

    // Handle loading state changes with delay and min duration
    useEffect(() => {
      let timeoutId: NodeJS.Timeout;

      if (isLoading) {
        // Start loading after delay
        if (delay > 0) {
          timeoutId = setTimeout(() => {
            setShowLoading(true);
            setStartTime(Date.now());
            if (showGlobalProgress) {
              startProgress();
            }
          }, delay);
        } else {
          setShowLoading(true);
          setStartTime(Date.now());
          if (showGlobalProgress) {
            startProgress();
          }
        }
      } else {
        // Stop loading with min duration
        if (showLoading && startTime && minDuration > 0) {
          const elapsed = Date.now() - startTime;
          const remaining = minDuration - elapsed;

          if (remaining > 0) {
            timeoutId = setTimeout(() => {
              setShowLoading(false);
              setStartTime(null);
              if (showGlobalProgress) {
                stopProgress();
              }
            }, remaining);
          } else {
            setShowLoading(false);
            setStartTime(null);
            if (showGlobalProgress) {
              stopProgress();
            }
          }
        } else {
          setShowLoading(false);
          setStartTime(null);
          if (showGlobalProgress) {
            stopProgress();
          }
        }
      }

      return () => {
        if (timeoutId) {
          clearTimeout(timeoutId);
        }
      };
    }, [isLoading, showLoading, startTime]); // eslint-disable-line react-hooks/exhaustive-deps

    // Handle errors
    if (loadingState.error) {
      logger.error('Loading error in withLoading HOC', {
        error: loadingState.error,
        component: Component.displayName || Component.name,
      });

      if (errorFallback) {
        return <>{errorFallback}</>;
      }

      return (
        <div className="flex flex-col items-center justify-center min-h-[400px] p-8">
          <div className="text-center max-w-md">
            <h3 className="text-lg font-semibold text-destructive mb-2">Error Loading Data</h3>
            <p className="text-muted-foreground">
              {String(loadingState.error) || 'An unexpected error occurred'}
            </p>
          </div>
        </div>
      );
    }

    // Show loading state
    if (showLoading && showLocalSpinner) {
      if (fallback) {
        return <>{fallback}</>;
      }

      if (fullScreen) {
        return <PageLoadingSkeleton title={loadingMessage} />;
      }

      if (transparentOverlay) {
        return (
          <div className="relative">
            <div className="opacity-50 pointer-events-none">
              <Component
                {...props}
                isLoading={true}
                setLoading={setLocalLoading}
                loadingState={loadingState}
              />
            </div>
            <div className="absolute inset-0 flex items-center justify-center bg-background/50">
              <LoadingComponent message={loadingMessage} />
            </div>
          </div>
        );
      }

      return (
        <div className="flex items-center justify-center min-h-[400px]">
          <LoadingComponent message={loadingMessage} />
        </div>
      );
    }

    // Render component with loading props
    return (
      <Component
        {...props}
        isLoading={isLoading}
        setLoading={setLocalLoading}
        loadingState={loadingState}
      />
    );
  };

  // Set display name for debugging
  WrappedComponent.displayName = `withLoading(${Component.displayName || Component.name || 'Component'})`;

  return WrappedComponent;
}

// Convenience functions for common loading patterns
export const withPageLoading = <P extends object>(
  Component: ComponentType<P & WithLoadingProps>,
  options?: Omit<WithLoadingOptions, 'fullScreen'>
) => withLoading(Component, { ...options, fullScreen: true, showGlobalProgress: true });

export const withTransparentLoading = <P extends object>(
  Component: ComponentType<P & WithLoadingProps>,
  options?: Omit<WithLoadingOptions, 'transparentOverlay'>
) => withLoading(Component, { ...options, transparentOverlay: true });

export const withDelayedLoading = <P extends object>(
  Component: ComponentType<P & WithLoadingProps>,
  options?: Omit<WithLoadingOptions, 'delay' | 'minDuration'>
) =>
  withLoading(Component, {
    ...options,
    delay: 200, // Don't show loading for quick operations
    minDuration: 500, // Show loading for at least 500ms to prevent flashing
  });

/**
 * Component for conditional loading states
 */
export function LoadingBoundary({
  children,
  isLoading,
  fallback,
  delay = 0,
  minDuration = 0,
  showProgress = false,
  message = 'Loading...',
}: {
  children: ReactNode;
  isLoading: boolean;
  fallback?: ReactNode;
  delay?: number;
  minDuration?: number;
  showProgress?: boolean;
  message?: string;
}) {
  const [showLoading, setShowLoading] = useState(false);
  const [startTime, setStartTime] = useState<number | null>(null);

  useEffect(() => {
    let timeoutId: NodeJS.Timeout;

    if (isLoading) {
      if (delay > 0) {
        timeoutId = setTimeout(() => {
          setShowLoading(true);
          setStartTime(Date.now());
          if (showProgress) {
            startProgress();
          }
        }, delay);
      } else {
        setShowLoading(true);
        setStartTime(Date.now());
        if (showProgress) {
          startProgress();
        }
      }
    } else {
      if (showLoading && startTime && minDuration > 0) {
        const elapsed = Date.now() - startTime;
        const remaining = minDuration - elapsed;

        if (remaining > 0) {
          timeoutId = setTimeout(() => {
            setShowLoading(false);
            setStartTime(null);
            if (showProgress) {
              stopProgress();
            }
          }, remaining);
        } else {
          setShowLoading(false);
          setStartTime(null);
          if (showProgress) {
            stopProgress();
          }
        }
      } else {
        setShowLoading(false);
        setStartTime(null);
        if (showProgress) {
          stopProgress();
        }
      }
    }

    return () => {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [isLoading, delay, minDuration, showProgress, showLoading, startTime]);

  if (showLoading) {
    return <>{fallback || <LoadingSpinner message={message} />}</>;
  }

  return <>{children}</>;
}
