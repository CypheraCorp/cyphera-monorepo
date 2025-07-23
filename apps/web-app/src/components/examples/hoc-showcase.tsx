'use client';

import { useState } from 'react';
import {
  withAuth,
  withErrorBoundary,
  withLoading,
  withMerchantPage,
  RequireAuth,
  LoadingBoundary,
  ErrorBoundaryWrapper,
  type WithAuthProps,
  type WithLoadingProps,
} from '@/lib/hocs';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { toast } from '@/components/ui/use-toast';

// Example 1: Basic component with auth
function AuthProtectedComponent({ session }: WithAuthProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Auth Protected Component</CardTitle>
        <CardDescription>This component requires authentication</CardDescription>
      </CardHeader>
      <CardContent>
        <p>Welcome, {session.user_type === 'merchant' ? session.email : session.customer_email}!</p>
        <p>User Type: {session.user_type}</p>
        <p>Session ID: {session.access_token.slice(0, 10)}...</p>
      </CardContent>
    </Card>
  );
}

export const AuthExample = withAuth(AuthProtectedComponent, {
  userType: 'merchant',
  redirectTo: '/merchants/signin',
});

// Example 2: Component with error boundary
function ErrorProneComponent({ throwError }: { throwError?: boolean }) {
  if (throwError) {
    throw new Error('This is a test error!');
  }

  const [shouldThrow, setShouldThrow] = useState(false);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Error Boundary Example</CardTitle>
        <CardDescription>This component has error handling</CardDescription>
      </CardHeader>
      <CardContent>
        <Button variant="destructive" onClick={() => setShouldThrow(true)}>
          Trigger Error
        </Button>
        <ErrorProneComponent throwError={shouldThrow} />
      </CardContent>
    </Card>
  );
}

export const ErrorBoundaryExample = withErrorBoundary(ErrorProneComponent, {
  onError: (error) => {
    toast({
      title: 'Error Caught',
      description: error.message,
      variant: 'destructive',
    });
  },
  resetOnPropsChange: true,
});

// Example 3: Component with loading states
function LoadingComponent({ setLoading, loadingState }: WithLoadingProps) {
  const simulateAsync = async (duration: number) => {
    setLoading?.(true);
    await new Promise((resolve) => setTimeout(resolve, duration));
    setLoading?.(false);
  };

  const simulateWithExecute = () => {
    loadingState?.execute(
      new Promise((resolve) => {
        setTimeout(() => {
          toast({ title: 'Operation completed!' });
          resolve(true);
        }, 2000);
      })
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Loading States Example</CardTitle>
        <CardDescription>Different loading patterns</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-x-2">
          <Button onClick={() => simulateAsync(500)}>Quick Load (500ms)</Button>
          <Button onClick={() => simulateAsync(2000)}>Slow Load (2s)</Button>
          <Button onClick={simulateWithExecute}>Load with Execute</Button>
        </div>
        <p>Loading: {loadingState?.isLoading ? 'Yes' : 'No'}</p>
      </CardContent>
    </Card>
  );
}

export const LoadingExample = withLoading(LoadingComponent, {
  delay: 200,
  minDuration: 500,
  showGlobalProgress: true,
  loadingMessage: 'Processing your request...',
});

// Example 4: All features combined
interface DashboardData {
  stats: { users: number; revenue: number };
  recentActivity: string[];
}

function FullFeaturedDashboard({ session, loadingState }: WithAuthProps & WithLoadingProps) {
  const [data, setData] = useState<DashboardData | null>(null);
  const [shouldError, setShouldError] = useState(false);

  const fetchDashboardData = async () => {
    if (shouldError) {
      throw new Error('Failed to fetch dashboard data');
    }

    await loadingState?.execute(
      new Promise<void>((resolve) => {
        setTimeout(() => {
          setData({
            stats: { users: 150, revenue: 25000 },
            recentActivity: ['User signed up', 'Payment received', 'Product created'],
          });
          resolve();
        }, 1500);
      })
    );
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Merchant Dashboard</CardTitle>
          <CardDescription>
            Welcome back, {session.user_type === 'merchant' ? session.email : 'User'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="flex space-x-2">
              <Button onClick={fetchDashboardData}>Load Dashboard Data</Button>
              <Button variant="outline" onClick={() => setShouldError(!shouldError)}>
                Toggle Error Mode: {shouldError ? 'ON' : 'OFF'}
              </Button>
            </div>

            {data && (
              <div className="grid grid-cols-2 gap-4">
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-lg">Total Users</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-2xl font-bold">{data.stats.users}</p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-lg">Revenue</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-2xl font-bold">${data.stats.revenue}</p>
                  </CardContent>
                </Card>
              </div>
            )}

            {data?.recentActivity && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Recent Activity</CardTitle>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-1">
                    {data.recentActivity.map((activity, i) => (
                      <li key={i} className="text-sm text-muted-foreground">
                        • {activity}
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Using the preset for merchant pages
export const MerchantDashboardExample = withMerchantPage(FullFeaturedDashboard);

// Example 5: Inline usage without HOCs
export function InlineExamples() {
  const [isLoading, setIsLoading] = useState(false);
  const [showError, setShowError] = useState(false);

  const simulateLoad = () => {
    setIsLoading(true);
    setTimeout(() => setIsLoading(false), 1500);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Inline HOC Usage</CardTitle>
          <CardDescription>Using HOC utilities without wrapping the component</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Inline auth check */}
          <div>
            <h3 className="font-medium mb-2">RequireAuth Component:</h3>
            <RequireAuth userType="merchant" fallback={<p>Not authenticated</p>}>
              {(session) => (
                <p className="text-sm text-muted-foreground">
                  Authenticated as:{' '}
                  {session.user_type === 'merchant'
                    ? (session as unknown as Record<string, unknown>).email as string
                    : (session as unknown as Record<string, unknown>).customer_email as string}
                </p>
              )}
            </RequireAuth>
          </div>

          {/* Inline loading boundary */}
          <div>
            <h3 className="font-medium mb-2">LoadingBoundary Component:</h3>
            <Button onClick={simulateLoad} className="mb-2">
              Trigger Loading
            </Button>
            <LoadingBoundary isLoading={isLoading} delay={200} fallback={<p>Loading content...</p>}>
              <p className="text-sm text-muted-foreground">
                This content is shown when not loading
              </p>
            </LoadingBoundary>
          </div>

          {/* Inline error boundary */}
          <div>
            <h3 className="font-medium mb-2">ErrorBoundaryWrapper Component:</h3>
            <Button onClick={() => setShowError(!showError)} variant="outline" className="mb-2">
              Toggle Error Component
            </Button>
            <ErrorBoundaryWrapper
              fallback={<p className="text-destructive">Component failed to load</p>}
              onError={(error) => console.error('Inline error:', error)}
            >
              {showError && <ErrorProneComponent throwError={true} />}
              {!showError && <p className="text-sm text-muted-foreground">No errors!</p>}
            </ErrorBoundaryWrapper>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Main showcase component
export default function HOCShowcase() {
  return (
    <div className="space-y-8 p-8">
      <h1 className="text-3xl font-bold">HOC Showcase</h1>

      <section>
        <h2 className="text-2xl font-semibold mb-4">Authentication HOC</h2>
        <AuthExample />
      </section>

      <section>
        <h2 className="text-2xl font-semibold mb-4">Error Boundary HOC</h2>
        <ErrorBoundaryExample />
      </section>

      <section>
        <h2 className="text-2xl font-semibold mb-4">Loading HOC</h2>
        <LoadingExample />
      </section>

      {/* Full Featured Example - requires authentication to work */}
      {/* <section>
        <h2 className="text-2xl font-semibold mb-4">Full Featured Example</h2>
        <MerchantDashboardExample />
      </section> */}

      <section>
        <h2 className="text-2xl font-semibold mb-4">Inline Usage</h2>
        <InlineExamples />
      </section>
    </div>
  );
}
