# Higher-Order Components (HOCs) and Wrappers

## Overview

This document describes the reusable Higher-Order Components (HOCs) available in the Cyphera Web application. These HOCs provide consistent patterns for authentication, error handling, loading states, and suspense boundaries.

## Available HOCs

### 1. withAuth

Protects components with authentication requirements.

```tsx
import { withAuth, withMerchantAuth, withCustomerAuth } from '@/lib/hocs';

// Basic usage
export default withAuth(ProtectedComponent);

// Merchant-only page
export default withMerchantAuth(MerchantDashboard);

// Customer-only page with onboarding check
export default withCustomerAuth(CustomerProfile, {
  checkOnboarding: true,
  redirectTo: '/customers/onboarding',
});

// Allow both user types
export default withAuth(SharedComponent, {
  allowBothTypes: true,
});

// Custom error handling
export default withAuth(ProtectedComponent, {
  userType: 'merchant',
  onAuthFailure: () => {
    toast.error('Please sign in to continue');
  },
});
```

#### Props Injected

```tsx
interface WithAuthProps {
  session: Session; // The authenticated user's session
}
```

#### Options

```tsx
interface WithAuthOptions {
  userType?: 'merchant' | 'customer';
  redirectTo?: string;
  allowBothTypes?: boolean;
  onAuthFailure?: () => void;
  LoadingComponent?: ComponentType;
  checkOnboarding?: boolean;
}
```

### 2. withErrorBoundary

Wraps components with error boundaries for graceful error handling.

```tsx
import { withErrorBoundary, withPageErrorBoundary } from '@/lib/hocs';

// Basic usage
export default withErrorBoundary(MyComponent);

// With custom fallback
export default withErrorBoundary(MyComponent, {
  fallback: <CustomErrorPage />,
  onError: (error, errorInfo) => {
    Sentry.captureException(error);
  },
});

// Page-level error boundary
export default withPageErrorBoundary(PageComponent);

// Reset on prop changes
export default withErrorBoundary(DataComponent, {
  resetKeys: ['userId', 'dataId'],
  resetOnPropsChange: true,
});
```

#### Options

```tsx
interface WithErrorBoundaryOptions {
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  resetKeys?: Array<string | number>;
  resetOnPropsChange?: boolean;
  isolate?: boolean;
  showErrorDetails?: boolean;
  enableRecovery?: boolean;
  logErrors?: boolean;
  errorMessageOverride?: string;
}
```

### 3. withLoading

Adds loading state management to components.

```tsx
import { withLoading, withPageLoading } from '@/lib/hocs';

// Basic usage
const MyComponent = withLoading(({ setLoading, loadingState }) => {
  const fetchData = async () => {
    setLoading(true);
    try {
      const data = await api.getData();
    } finally {
      setLoading(false);
    }
  };

  // Or use loadingState.execute
  const handleSubmit = () => {
    loadingState.execute(api.submitData());
  };

  return <div>...</div>;
});

// Page-level loading
export default withPageLoading(PageComponent, {
  loadingMessage: 'Loading dashboard...',
});

// Delayed loading (prevents flash for quick operations)
export default withDelayedLoading(QuickComponent);

// Transparent overlay
export default withTransparentLoading(FormComponent);
```

#### Props Injected

```tsx
interface WithLoadingProps {
  isLoading?: boolean;
  setLoading?: (loading: boolean) => void;
  loadingState?: {
    isLoading: boolean;
    error: string | null;
    execute: <T>(promise: Promise<T>) => Promise<T>;
  };
}
```

#### Options

```tsx
interface WithLoadingOptions {
  showGlobalProgress?: boolean;
  showLocalSpinner?: boolean;
  delay?: number;
  minDuration?: number;
  fallback?: ReactNode;
  errorFallback?: ReactNode;
  LoadingComponent?: ComponentType;
  loadingMessage?: string;
  transparentOverlay?: boolean;
  fullScreen?: boolean;
}
```

### 4. withSuspense

Wraps components with Suspense and Error boundaries for async components.

```tsx
import { withSuspense, withPageSuspense } from '@/lib/hocs';

// Basic usage with React Query suspense
const AsyncComponent = withSuspense(() => {
  const { data } = useQuery({
    queryKey: ['data'],
    queryFn: fetchData,
    suspense: true,
  });

  return <div>{data}</div>;
});

// Page-level suspense
export default withPageSuspense(AsyncPage, {
  loadingMessage: 'Loading page data...',
});

// Custom fallbacks
export default withSuspense(AsyncComponent, {
  fallback: <CustomSkeleton />,
  errorFallback: <ErrorMessage />,
  onError: (error) => console.error(error),
});
```

#### Options

```tsx
interface WithSuspenseOptions {
  fallback?: ReactNode;
  errorFallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  loadingMessage?: string;
  showPageSkeleton?: boolean;
  resetKeys?: Array<string | number>;
  isolate?: boolean;
}
```

### 5. withFeatures (Composite HOC)

Combines multiple HOCs with a single configuration.

```tsx
import { withFeatures, withMerchantPage, withCustomerPage } from '@/lib/hocs';

// Enable all features
export default withFeatures(MyComponent, {
  auth: { userType: 'merchant' },
  errorBoundary: true,
  loading: { showGlobalProgress: true },
  suspense: true,
});

// Merchant page preset
export default withMerchantPage(MerchantDashboard);

// Customer page preset
export default withCustomerPage(CustomerProfile, {
  auth: { checkOnboarding: true },
});

// Public page with safety features
export default withPublicPage(LandingPage);

// Async component with all protections
export default withAsyncSafeComponent(DataTable);
```

## Component Usage Examples

### Protected Dashboard Page

```tsx
import { withMerchantPage } from '@/lib/hocs';

function MerchantDashboard({ session, loadingState }) {
  const { data, refetch } = useQuery({
    queryKey: ['dashboard', session.workspace_id],
    queryFn: () => fetchDashboardData(session.workspace_id),
  });

  const handleRefresh = () => {
    loadingState.execute(refetch());
  };

  return (
    <div>
      <h1>Welcome, {session.email}</h1>
      <button onClick={handleRefresh}>Refresh Data</button>
      {/* Dashboard content */}
    </div>
  );
}

export default withMerchantPage(MerchantDashboard);
```

### Async Data Component

```tsx
import { withAsyncSafeComponent } from '@/lib/hocs';

function UserList({ setLoading }) {
  const [users, setUsers] = useState([]);

  const loadUsers = async () => {
    setLoading(true);
    try {
      const data = await api.getUsers();
      setUsers(data);
    } finally {
      setLoading(false);
    }
  };

  return <DataTable data={users} onRefresh={loadUsers} />;
}

export default withAsyncSafeComponent(UserList, {
  loading: {
    transparentOverlay: true,
    delay: 200,
  },
});
```

### Inline Usage

For cases where you need auth checks without HOCs:

```tsx
import { RequireAuth, LoadingBoundary, ErrorBoundaryWrapper } from '@/lib/hocs';

function MyComponent() {
  const [isLoading, setIsLoading] = useState(false);

  return (
    <div>
      <RequireAuth userType="merchant">
        {(session) => <div>Merchant ID: {session.account_id}</div>}
      </RequireAuth>

      <LoadingBoundary isLoading={isLoading} delay={200}>
        <DataContent />
      </LoadingBoundary>

      <ErrorBoundaryWrapper fallback={<ErrorCard />}>
        <RiskyComponent />
      </ErrorBoundaryWrapper>
    </div>
  );
}
```

## Presets and Patterns

### Page-Level Components

```tsx
// Merchant pages
export default withMerchantPage(Component);

// Customer pages
export default withCustomerPage(Component);

// Public pages
export default withPublicPage(Component);
```

### Data Components

```tsx
// Tables and lists
export default withDataComponent(DataTable);

// Forms with loading
export default withTransparentLoading(FormComponent);
```

### Async Components

```tsx
// With suspense and error handling
export default withAsyncSafeComponent(AsyncComponent);

// Lazy loaded
const LazyComponent = createLazyComponent(() => import('./HeavyComponent'), {
  showPageSkeleton: true,
});
```

## Best Practices

1. **Order Matters**: When using multiple HOCs separately, apply them in this order:

   ```tsx
   withErrorBoundary(withSuspense(withAuth(withLoading(Component))));
   ```

2. **Use Presets**: Prefer preset HOCs like `withMerchantPage` over manual composition

3. **Type Safety**: Always import prop types when using injected props:

   ```tsx
   import type { WithAuthProps, WithLoadingProps } from '@/lib/hocs';

   interface Props extends WithAuthProps, WithLoadingProps {
     // your props
   }
   ```

4. **Error Handling**: Always provide error handlers for production:

   ```tsx
   export default withFeatures(Component, {
     errorBoundary: {
       onError: (error) => {
         logErrorToService(error);
         showErrorToast();
       },
     },
   });
   ```

5. **Loading States**: Use appropriate loading patterns:
   - `delay` for preventing flash on quick operations
   - `transparentOverlay` for forms and interactive components
   - `fullScreen` for page-level loading
   - `showGlobalProgress` for navigation

6. **Reset Keys**: Use reset keys for components that should remount on prop changes:
   ```tsx
   withErrorBoundary(Component, {
     resetKeys: ['userId', 'productId'],
     resetOnPropsChange: true,
   });
   ```

## Migration Guide

### From Direct Auth Checks

```tsx
// Before
function Dashboard() {
  const { user } = useAuth();
  if (!user) return <Redirect to="/login" />;
  return <div>...</div>;
}

// After
export default withMerchantAuth(function Dashboard({ session }) {
  return <div>...</div>;
});
```

### From Manual Error Handling

```tsx
// Before
function Component() {
  try {
    return <RiskyOperation />;
  } catch (error) {
    return <ErrorPage />;
  }
}

// After
export default withErrorBoundary(function Component() {
  return <RiskyOperation />;
});
```

### From Manual Loading States

```tsx
// Before
function Component() {
  const [loading, setLoading] = useState(false);

  if (loading) return <Spinner />;
  return <div>...</div>;
}

// After
export default withLoading(function Component({ setLoading }) {
  return <div>...</div>;
});
```

## Testing

When testing components wrapped with HOCs:

```tsx
import { render } from '@testing-library/react';
import { withAuth } from '@/lib/hocs';

// Mock the session
jest.mock('@/lib/auth/session/unified-session-client', () => ({
  UnifiedSessionClient: {
    getByType: jest.fn(() => Promise.resolve(mockSession)),
  },
}));

// Test the wrapped component
const WrappedComponent = withAuth(MyComponent);
render(<WrappedComponent />);
```

## Performance Considerations

1. **Lazy Loading**: Use `createLazyComponent` for code splitting
2. **Memoization**: HOCs preserve React.memo wrapping
3. **Reset Keys**: Minimize unnecessary remounts
4. **Loading Delays**: Prevent loading flash with appropriate delays
5. **Error Boundaries**: Isolate errors to prevent full page crashes
