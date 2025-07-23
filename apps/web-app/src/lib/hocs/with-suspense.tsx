'use client';

import { ComponentType, ReactNode, Suspense, lazy } from 'react';
import { ErrorBoundary } from '@/components/ui/error-boundary';
import { LoadingSpinner, PageLoadingSkeleton } from '@/components/ui/loading-states';
import { logger } from '@/lib/core/logger/logger-utils';

export interface WithSuspenseOptions {
  fallback?: ReactNode;
  errorFallback?: ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  loadingMessage?: string;
  showPageSkeleton?: boolean;
  resetKeys?: Array<string | number>;
  isolate?: boolean;
}

/**
 * Higher-Order Component that wraps a component with Suspense and Error Boundaries
 * Perfect for components that use React's suspend mechanism (like React Query with suspense: true)
 *
 * @example
 * // Basic usage
 * export default withSuspense(MyAsyncComponent);
 *
 * // With custom fallback
 * export default withSuspense(MyAsyncComponent, {
 *   fallback: <CustomLoadingSkeleton />,
 *   loadingMessage: 'Loading user data...'
 * });
 *
 * // With error handling
 * export default withSuspense(MyAsyncComponent, {
 *   errorFallback: <CustomErrorComponent />,
 *   onError: (error) => console.error('Async component failed:', error)
 * });
 *
 * // Page-level suspense
 * export default withSuspense(MyPage, {
 *   showPageSkeleton: true,
 *   loadingMessage: 'Loading page content...'
 * });
 */
export function withSuspense<P extends object>(
  Component: ComponentType<P>,
  options: WithSuspenseOptions = {}
): ComponentType<P> {
  const {
    fallback,
    errorFallback,
    onError,
    loadingMessage = 'Loading...',
    showPageSkeleton = false,
    resetKeys = [],
    isolate: _isolate = false,
  } = options;

  const WrappedComponent = (props: P) => {
    // Create a key based on resetKeys for resetting the boundaries
    const resetKey = resetKeys
      .map((key) => {
        const value = (props as Record<string, unknown>)[key];
        return value !== undefined ? String(value) : '';
      })
      .join('-');

    // Determine loading fallback
    const loadingFallback =
      fallback ||
      (showPageSkeleton ? (
        <PageLoadingSkeleton title={loadingMessage} />
      ) : (
        <div className="flex items-center justify-center min-h-[400px]">
          <LoadingSpinner message={loadingMessage} />
        </div>
      ));

    // Error handler
    const handleError = (error: Error, errorInfo: React.ErrorInfo) => {
      logger.error('Suspense boundary error', {
        error: error.message,
        stack: error.stack,
        componentStack: errorInfo.componentStack,
        component: Component.displayName || Component.name,
      });

      if (onError) {
        onError(error, errorInfo);
      }
    };

    // Default error fallback
    const defaultErrorFallback = (
      <div className="flex flex-col items-center justify-center min-h-[400px] p-8">
        <div className="text-center max-w-md">
          <h2 className="text-2xl font-semibold text-destructive mb-4">Failed to Load</h2>
          <p className="text-muted-foreground mb-6">
            We couldn&apos;t load this content. Please try refreshing the page.
          </p>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
          >
            Refresh Page
          </button>
        </div>
      </div>
    );

    // Wrap with error boundary and suspense
    return (
      <ErrorBoundary
        key={resetKey}
        fallback={errorFallback || defaultErrorFallback}
        onError={handleError}
      >
        <Suspense fallback={loadingFallback}>
          <Component {...props} />
        </Suspense>
      </ErrorBoundary>
    );
  };

  // Set display name for debugging
  WrappedComponent.displayName = `withSuspense(${Component.displayName || Component.name || 'Component'})`;

  return WrappedComponent;
}

// Convenience functions for common patterns
export const withPageSuspense = <P extends object>(
  Component: ComponentType<P>,
  options?: Omit<WithSuspenseOptions, 'showPageSkeleton'>
) => withSuspense(Component, { ...options, showPageSkeleton: true });

export const withAsyncComponent = <P extends object>(
  Component: ComponentType<P>,
  options?: WithSuspenseOptions
) =>
  withSuspense(Component, {
    ...options,
    isolate: true,
    fallback: options?.fallback || <LoadingSpinner message="Loading component..." />,
  });

/**
 * Suspense wrapper component for inline usage
 */
export function SuspenseBoundary({
  children,
  fallback,
  errorFallback,
  onError,
  loadingMessage = 'Loading...',
  resetKey,
}: {
  children: ReactNode;
  fallback?: ReactNode;
  errorFallback?: ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  loadingMessage?: string;
  resetKey?: string | number;
}) {
  const defaultFallback = (
    <div className="flex items-center justify-center p-4">
      <LoadingSpinner message={loadingMessage} />
    </div>
  );

  const defaultErrorFallback = (
    <div className="text-center p-4">
      <p className="text-destructive">Failed to load content</p>
    </div>
  );

  return (
    <ErrorBoundary
      key={resetKey}
      fallback={errorFallback || defaultErrorFallback}
      onError={(error, errorInfo) => {
        logger.error('Inline suspense boundary error', {
          error: error.message,
          errorInfo: errorInfo.componentStack,
        });
        onError?.(error, errorInfo);
      }}
    >
      <Suspense fallback={fallback || defaultFallback}>{children}</Suspense>
    </ErrorBoundary>
  );
}

/**
 * Utility to create lazy-loaded components with built-in suspense
 */
export function createLazyComponent<P extends object>(
  importFn: () => Promise<{ default: ComponentType<P> }>,
  options?: WithSuspenseOptions
): ComponentType<P> {
  const LazyComponent = lazy(importFn);
  return withSuspense(LazyComponent as any, options) as ComponentType<P>;
}

// Re-export React.lazy for convenience
export { lazy };
