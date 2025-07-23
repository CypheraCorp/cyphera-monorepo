'use client';

import { useEffect, ReactNode } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { MainSidebar } from '@/components/layout/main-sidebar';
import { useAuth } from '@/hooks/auth/use-auth-user';
import { QueryProvider } from '@/lib/query/query-client';

// Merchant authentication wrapper component
function MerchantAuthWrapper({ children }: { children: ReactNode }) {
  // Using new auth hook with fresh data
  const { isAuthenticated, loading, hasHydrated } = useAuth();
  const pathname = usePathname();
  const router = useRouter();

  // Get page title based on pathname (preserve original function)
  const getPageTitle = () => {
    switch (pathname) {
      case '/merchants/dashboard':
        return { title: 'Dashboard', subtitle: 'Overview of your business and subscriptions' };
      case '/merchants/products':
        return { title: 'Products', subtitle: 'Manage your subscription products' };
      case '/merchants/customers':
        return { title: 'Customers', subtitle: 'Manage your customer base and relationships' };
      case '/merchants/wallets':
        return { title: 'Wallets', subtitle: 'Manage your crypto wallets and transactions' };
      case '/merchants/transactions':
        return { title: 'Transactions', subtitle: 'View and manage payment transactions' };
      case '/merchants/subscriptions':
        return { title: 'Subscriptions', subtitle: 'Monitor and manage active subscriptions' };
      case '/merchants/settings':
        return { title: 'Settings', subtitle: 'Configure your account and preferences' };
      default:
        return { title: 'Merchant Portal', subtitle: 'Welcome to your merchant portal' };
    }
  };

  useEffect(() => {
    // Wait for hydration to complete
    if (!hasHydrated || loading) return;

    if (PUBLIC_ROUTES.includes(pathname)) {
      // If user is authenticated and visits signin page, redirect to dashboard
      if (isAuthenticated && pathname === '/merchants/signin') {
        router.push('/merchants/dashboard');
      }
      return;
    }

    if (!isAuthenticated) {
      router.push('/merchants/signin');
    } else if (pathname === '/merchants') {
      router.push('/merchants/dashboard');
    }
  }, [hasHydrated, loading, isAuthenticated, pathname, router]);

  if (!hasHydrated || loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-neutral-50 dark:bg-neutral-900">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
        </div>
      </div>
    );
  }

  if (PUBLIC_ROUTES.includes(pathname)) {
    return <div className="min-h-screen bg-neutral-50 dark:bg-neutral-900">{children}</div>;
  }

  if (!isAuthenticated) {
    return null; // Redirect will handle
  }

  const { title, subtitle } = getPageTitle();

  return (
    <div className="min-h-screen bg-neutral-50 dark:bg-neutral-900 flex">
      <MainSidebar />
      <main className="flex-1 ml-0 lg:ml-16 pt-14 md:pt-0">
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
  );
}

// Add PUBLIC_ROUTES definition
const PUBLIC_ROUTES = ['/merchants/signin', '/merchants/verify-email', '/merchants/onboarding'];

export default function MerchantsLayout({ children }: { children: ReactNode }) {
  return (
    <QueryProvider>
      <MerchantAuthWrapper>{children}</MerchantAuthWrapper>
    </QueryProvider>
  );
}
