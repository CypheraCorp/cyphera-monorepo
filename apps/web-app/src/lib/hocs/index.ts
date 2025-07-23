// Auth HOC
export {
  withAuth,
  withMerchantAuth,
  withCustomerAuth,
  RequireAuth,
  type WithAuthProps,
  type WithAuthOptions,
} from './with-auth';

// Error Boundary HOC
export {
  withErrorBoundary,
  withAsyncBoundary,
  withPageErrorBoundary,
  withComponentErrorBoundary,
  ErrorBoundaryWrapper,
  type WithErrorBoundaryOptions,
} from './with-error-boundary';

// Loading HOC
export {
  withLoading,
  withPageLoading,
  withTransparentLoading,
  withDelayedLoading,
  LoadingBoundary,
  type WithLoadingProps,
  type WithLoadingOptions,
} from './with-loading';

// Suspense HOC
export {
  withSuspense,
  withPageSuspense,
  withAsyncComponent,
  SuspenseBoundary,
  createLazyComponent,
  lazy,
  type WithSuspenseOptions,
} from './with-suspense';

// Composite HOCs
export {
  withFeatures,
  withProtectedPage,
  withMerchantPage,
  withCustomerPage,
  withPublicPage,
  withAsyncSafeComponent,
  withDataComponent,
  type WithFeaturesOptions,
  type WithFeaturesProps,
} from './with-features';
