'use client';

import { ComponentType } from 'react';
import { withAuth, WithAuthOptions, WithAuthProps } from './with-auth';
import { withErrorBoundary, WithErrorBoundaryOptions } from './with-error-boundary';
import { withLoading, WithLoadingOptions, WithLoadingProps } from './with-loading';
import { withSuspense, WithSuspenseOptions } from './with-suspense';
import type { UserType } from '@/lib/auth/session/unified-session';

export interface WithFeaturesOptions {
  auth?: WithAuthOptions | boolean;
  errorBoundary?: WithErrorBoundaryOptions | boolean;
  loading?: WithLoadingOptions | boolean;
  suspense?: WithSuspenseOptions | boolean;
}

export type WithFeaturesProps = WithAuthProps & WithLoadingProps;

/**
 * Composite HOC that combines multiple features
 *
 * @example
 * // Enable all features with defaults
 * export default withFeatures(MyComponent, {
 *   auth: true,
 *   errorBoundary: true,
 *   loading: true,
 *   suspense: true
 * });
 *
 * // Custom configuration
 * export default withFeatures(MyComponent, {
 *   auth: { userType: 'merchant', checkOnboarding: true },
 *   errorBoundary: { showErrorDetails: true },
 *   loading: { showGlobalProgress: true, delay: 200 },
 *   suspense: { showPageSkeleton: true }
 * });
 *
 * // Partial features
 * export default withFeatures(MyComponent, {
 *   auth: { userType: 'customer' },
 *   errorBoundary: true
 * });
 */
export function withFeatures<P extends object>(
  Component: ComponentType<P & Partial<WithFeaturesProps>>,
  options: WithFeaturesOptions = {}
): ComponentType<P> {
  let WrappedComponent = Component as ComponentType<P & Partial<WithFeaturesProps>>;

  // Apply HOCs in order (innermost to outermost)
  // Order: Component -> Loading -> Auth -> Suspense -> ErrorBoundary

  // Apply loading HOC
  if (options.loading) {
    const loadingOptions = options.loading === true ? {} : options.loading;
    WrappedComponent = withLoading(WrappedComponent, loadingOptions) as ComponentType<P & Partial<WithFeaturesProps>>;
  }

  // Apply auth HOC
  if (options.auth) {
    const authOptions = options.auth === true ? {} : options.auth;
    WrappedComponent = withAuth(WrappedComponent as any, authOptions) as ComponentType<P & Partial<WithFeaturesProps>>;
  }

  // Apply suspense HOC
  if (options.suspense) {
    const suspenseOptions = options.suspense === true ? {} : options.suspense;
    WrappedComponent = withSuspense(WrappedComponent, suspenseOptions) as ComponentType<P & Partial<WithFeaturesProps>>;
  }

  // Apply error boundary HOC (outermost)
  if (options.errorBoundary) {
    const errorOptions = options.errorBoundary === true ? {} : options.errorBoundary;
    WrappedComponent = withErrorBoundary(WrappedComponent, errorOptions) as ComponentType<P & Partial<WithFeaturesProps>>;
  }

  // Set display name
  WrappedComponent.displayName = `withFeatures(${Component.displayName || Component.name || 'Component'})`;

  return WrappedComponent as ComponentType<P>;
}

// Preset configurations for common use cases

/**
 * Protected page with all safety features
 */
export const withProtectedPage = <P extends object>(
  Component: ComponentType<P & Partial<WithFeaturesProps>>,
  userType: UserType,
  options?: Partial<WithFeaturesOptions>
) =>
  withFeatures(Component, {
    auth: { userType, ...((options?.auth as WithAuthOptions) || {}) },
    errorBoundary: {
      isolate: false,
      ...((options?.errorBoundary as WithErrorBoundaryOptions) || {}),
    },
    loading: {
      fullScreen: true,
      showGlobalProgress: true,
      ...((options?.loading as WithLoadingOptions) || {}),
    },
    suspense: { showPageSkeleton: true, ...((options?.suspense as WithSuspenseOptions) || {}) },
  });

/**
 * Merchant dashboard page
 */
export const withMerchantPage = <P extends object>(
  Component: ComponentType<P & Partial<WithFeaturesProps>>,
  options?: Partial<WithFeaturesOptions>
) => withProtectedPage(Component, 'merchant', options);

/**
 * Customer dashboard page
 */
export const withCustomerPage = <P extends object>(
  Component: ComponentType<P & Partial<WithFeaturesProps>>,
  options?: Partial<WithFeaturesOptions>
) => withProtectedPage(Component, 'customer', options);

/**
 * Public page with error handling and loading
 */
export const withPublicPage = <P extends object>(
  Component: ComponentType<P & Partial<WithFeaturesProps>>,
  options?: Partial<WithFeaturesOptions>
) =>
  withFeatures(Component, {
    errorBoundary: true,
    loading: { showGlobalProgress: true, ...((options?.loading as WithLoadingOptions) || {}) },
    suspense: options?.suspense,
  });

/**
 * Async component with all safety features
 */
export const withAsyncSafeComponent = <P extends object>(
  Component: ComponentType<P & Partial<WithFeaturesProps>>,
  options?: Partial<WithFeaturesOptions>
) =>
  withFeatures(Component, {
    errorBoundary: {
      isolate: true,
      ...((options?.errorBoundary as WithErrorBoundaryOptions) || {}),
    },
    loading: { delay: 200, minDuration: 500, ...((options?.loading as WithLoadingOptions) || {}) },
    suspense: true,
    ...options,
  });

/**
 * Data table or list component
 */
export const withDataComponent = <P extends object>(
  Component: ComponentType<P & Partial<WithFeaturesProps>>,
  options?: Partial<WithFeaturesOptions>
) =>
  withFeatures(Component, {
    errorBoundary: { isolate: true, enableRecovery: true },
    loading: {
      transparentOverlay: true,
      delay: 100,
      ...((options?.loading as WithLoadingOptions) || {}),
    },
    ...options,
  });
