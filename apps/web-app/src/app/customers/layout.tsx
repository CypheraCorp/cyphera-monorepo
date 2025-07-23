'use client';

import { useState, useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { CustomerSidebar } from '@/components/public/customer-sidebar';
import { CustomerHeader } from '@/components/public/customer-header';
import { AuthProvider } from '@/contexts/auth-context';
import { QueryProvider } from '@/lib/query/query-client';
import { logger } from '@/lib/core/logger/logger-utils';
import { AuthSyncProvider } from '@/components/providers/auth-sync-provider';

// Customer authentication wrapper component
function CustomerAuthWrapper({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Get page title based on pathname
  const getPageTitle = () => {
    switch (pathname) {
      case '/customers/dashboard':
        return { title: 'Dashboard', subtitle: 'Overview of your account and subscriptions' };
      case '/customers/marketplace':
        return { title: 'Marketplace', subtitle: 'Discover and subscribe to services' };
      case '/customers/wallet':
        return { title: 'My Wallet', subtitle: 'Manage your crypto wallet and transactions' };
      case '/customers/subscriptions':
        return { title: 'Subscriptions', subtitle: 'Manage your active subscriptions' };
      case '/customers/settings':
        return { title: 'Settings', subtitle: 'Update your account preferences' };
      default:
        return { title: 'Customer Portal', subtitle: 'Welcome to your customer portal' };
    }
  };

  // Skip authentication check for public customer routes
  const isPublicRoute = pathname === '/customers/signin';

  useEffect(() => {
    if (isPublicRoute) {
      setIsAuthenticated(true);
      setIsLoading(false);
      return;
    }

    async function checkAuth() {
      try {
        const response = await fetch('/api/auth/customer/me', {
          credentials: 'include',
        });

        if (response.ok) {
          const data = await response.json();
          const hasValidSession = !!data.customer;
          setIsAuthenticated(hasValidSession);
        } else {
          logger.info('No valid customer session, redirecting to signin');
          setIsAuthenticated(false);
          router.push('/customers/signin');
        }
      } catch (error) {
        logger.error('Auth check failed', error);
        setIsAuthenticated(false);
        router.push('/customers/signin');
      } finally {
        setIsLoading(false);
      }
    }

    checkAuth();
  }, [pathname, router, isPublicRoute]);

  // Show loading state
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-neutral-50 dark:bg-neutral-900">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600 mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading customer portal...</p>
        </div>
      </div>
    );
  }

  // Show signin page for public routes
  if (isPublicRoute) {
    return <div className="min-h-screen bg-neutral-50 dark:bg-neutral-900">{children}</div>;
  }

  // Show authenticated layout
  if (isAuthenticated) {
    const { title, subtitle } = getPageTitle();

    return (
      <div className="min-h-screen bg-neutral-50 dark:bg-neutral-900 flex">
        <CustomerSidebar />
        <div className="flex-1 ml-0 lg:ml-16">
          <CustomerHeader title={title} subtitle={subtitle} />
          <main className="pt-0">
            <div className="p-4 lg:p-6 max-w-7xl mx-auto">
              <div className="mb-6">
                <div className="text-center">
                  <h1 className="text-2xl lg:text-3xl font-bold text-gray-900 dark:text-white">
                    {title}
                  </h1>
                  <p className="text-gray-600 dark:text-gray-400 mt-1">{subtitle}</p>
                </div>
              </div>
              <div className="w-full">{children}</div>
            </div>
          </main>
        </div>
      </div>
    );
  }

  // This should not be reached due to redirects above
  return null;
}

export default function CustomersLayout({ children }: { children: React.ReactNode }) {
  return (
    <QueryProvider>
      <AuthProvider>
        <AuthSyncProvider>
          <CustomerAuthWrapper>{children}</CustomerAuthWrapper>
        </AuthSyncProvider>
      </AuthProvider>
    </QueryProvider>
  );
}
