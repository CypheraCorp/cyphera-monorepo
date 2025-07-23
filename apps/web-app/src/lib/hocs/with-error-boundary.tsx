'use client';

import { ComponentType, ReactNode } from 'react';
import {
  ErrorBoundary,
  PageErrorBoundary,
  AsyncErrorBoundary,
} from '@/components/ui/error-boundary';
import { logger } from '@/lib/core/logger/logger-utils';

export interface WithErrorBoundaryOptions {
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  resetKeys?: Array<string | number>;
  resetOnPropsChange?: boolean;
  isolate?: boolean;
  showErrorDetails?: boolean;
  enableRecovery?: boolean;
  logErrors?: boolean;
  errorMessageOverride?: string;
}

/**
 * Higher-Order Component that wraps a component with an error boundary
 *
 * @example
 * // Basic usage
 * export default withErrorBoundary(MyComponent);
 *
 * // With custom fallback
 * export default withErrorBoundary(MyComponent, {
 *   fallback: <CustomErrorFallback />
 * });
 *
 * // With error handler
 * export default withErrorBoundary(MyComponent, {
 *   onError: (error, errorInfo) => {
 *     console.error('Component error:', error);
 *     // Send to error tracking service
 *   }
 * });
 *
 * // Reset on prop changes
 * export default withErrorBoundary(MyComponent, {
 *   resetKeys: ['userId', 'dataId'],
 *   resetOnPropsChange: true
 * });
 */
export function withErrorBoundary<P extends object>(
  Component: ComponentType<P>,
  options: WithErrorBoundaryOptions = {}
): ComponentType<P> {
  const {
    fallback,
    onError,
    resetKeys = [],
    resetOnPropsChange = false,
    isolate = false,
    showErrorDetails = process.env.NODE_ENV === 'development',
    enableRecovery = true,
    logErrors = true,
    errorMessageOverride,
  } = options;

  const WrappedComponent = (props: P) => {
    // Create a key based on resetKeys for resetting the error boundary
    const resetKey = resetKeys
      .map((key) => {
        const value = (props as Record<string, unknown>)[key];
        return value !== undefined ? String(value) : '';
      })
      .join('-');

    const handleError = (error: Error, errorInfo: React.ErrorInfo) => {
      if (logErrors) {
        logger.error('Error boundary caught error', {
          error: error.message,
          stack: error.stack,
          componentStack: errorInfo.componentStack,
          component: Component.displayName || Component.name,
        });
      }

      if (onError) {
        onError(error, errorInfo);
      }
    };

    // Custom fallback UI
    const errorFallback = fallback || (
      <div className="flex flex-col items-center justify-center min-h-[400px] p-8">
        <div className="text-center max-w-md">
          <h2 className="text-2xl font-semibold text-destructive mb-4">
            {errorMessageOverride || 'Something went wrong'}
          </h2>
          <p className="text-muted-foreground mb-6">
            {showErrorDetails
              ? 'An error occurred while rendering this component. Please try refreshing the page.'
              : 'We encountered an unexpected error. Please try again.'}
          </p>
          {enableRecovery && (
            <button
              onClick={() => window.location.reload()}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
            >
              Refresh Page
            </button>
          )}
        </div>
      </div>
    );

    // Use different error boundary based on options
    if (isolate) {
      // Use basic ErrorBoundary for isolated components
      return (
        <ErrorBoundary
          key={resetOnPropsChange ? resetKey : undefined}
          fallback={errorFallback}
          onError={handleError}
        >
          <Component {...props} />
        </ErrorBoundary>
      );
    }

    // Use PageErrorBoundary for full page protection
    return (
      <div key={resetOnPropsChange ? resetKey : undefined}>
        <PageErrorBoundary>
          <Component {...props} />
        </PageErrorBoundary>
      </div>
    );
  };

  // Set display name for debugging
  WrappedComponent.displayName = `withErrorBoundary(${Component.displayName || Component.name || 'Component'})`;

  return WrappedComponent;
}

/**
 * HOC for async components with Suspense + Error Boundary
 */
export function withAsyncBoundary<P extends object>(
  Component: ComponentType<P>,
  options: WithErrorBoundaryOptions & { suspenseFallback?: ReactNode } = {}
): ComponentType<P> {
  const { suspenseFallback: _suspenseFallback, ...errorBoundaryOptions } = options;

  const WrappedComponent = (props: P) => {
    return (
      <AsyncErrorBoundary fallback={errorBoundaryOptions.fallback}>
        <Component {...props} />
      </AsyncErrorBoundary>
    );
  };

  WrappedComponent.displayName = `withAsyncBoundary(${Component.displayName || Component.name || 'Component'})`;

  return WrappedComponent;
}

// Convenience function for page-level error boundaries
export const withPageErrorBoundary = <P extends object>(
  Component: ComponentType<P>,
  options?: Omit<WithErrorBoundaryOptions, 'isolate'>
) => withErrorBoundary(Component, { ...options, isolate: false });

// Convenience function for component-level error boundaries
export const withComponentErrorBoundary = <P extends object>(
  Component: ComponentType<P>,
  options?: Omit<WithErrorBoundaryOptions, 'isolate'>
) => withErrorBoundary(Component, { ...options, isolate: true });

/**
 * Hook-like component for inline error boundaries
 */
export function ErrorBoundaryWrapper({
  children,
  fallback,
  onError,
  resetKey,
}: {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  resetKey?: string | number;
}) {
  return (
    <ErrorBoundary
      key={resetKey}
      fallback={fallback}
      onError={(error, errorInfo) => {
        logger.error('Inline error boundary caught error', {
          error: error.message,
          errorInfo: errorInfo.componentStack,
        });
        onError?.(error, errorInfo);
      }}
    >
      {children}
    </ErrorBoundary>
  );
}
