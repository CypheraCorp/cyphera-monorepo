'use client';

import { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { ComponentType } from 'react';
import { UnifiedSessionClient } from '@/lib/auth/session/unified-session-client';
import type { Session, UserType } from '@/lib/auth/session/unified-session';
import { LoadingSpinner } from '@/components/ui/loading-states';
import { logger } from '@/lib/core/logger/logger-utils';

export interface WithAuthProps {
  session: Session;
}

export interface WithAuthOptions {
  userType?: UserType;
  redirectTo?: string;
  allowBothTypes?: boolean;
  onAuthFailure?: () => void;
  LoadingComponent?: ComponentType;
  checkOnboarding?: boolean;
}

/**
 * Higher-Order Component for protecting routes with authentication
 *
 * @example
 * // Protect a merchant page
 * export default withAuth(MerchantDashboard, { userType: 'merchant' });
 *
 * // Protect a customer page
 * export default withAuth(CustomerDashboard, { userType: 'customer' });
 *
 * // Allow both user types
 * export default withAuth(SharedPage, { allowBothTypes: true });
 *
 * // Custom redirect
 * export default withAuth(ProtectedPage, {
 *   userType: 'merchant',
 *   redirectTo: '/merchants/signin?error=unauthorized'
 * });
 *
 * // Check onboarding status
 * export default withAuth(ProtectedPage, {
 *   userType: 'customer',
 *   checkOnboarding: true,
 *   redirectTo: '/customers/onboarding'
 * });
 */
export function withAuth<P extends object>(
  Component: ComponentType<P & WithAuthProps>,
  options: WithAuthOptions = {}
): ComponentType<P> {
  const {
    userType,
    redirectTo,
    allowBothTypes = false,
    onAuthFailure,
    LoadingComponent = DefaultLoadingComponent,
    checkOnboarding = false,
  } = options;

  const WrappedComponent = (props: P) => {
    const router = useRouter();
    const pathname = usePathname();
    const [session, setSession] = useState<Session | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
      const checkAuth = async () => {
        try {
          setIsLoading(true);
          setError(null);

          // Get session based on options
          let currentSession: Session | null = null;

          if (allowBothTypes) {
            // Get any available session
            currentSession = await UnifiedSessionClient.get();
          } else if (userType) {
            // Get specific user type session
            currentSession = await UnifiedSessionClient.getByType(userType);
          } else {
            // Default to any session
            currentSession = await UnifiedSessionClient.get();
          }

          if (!currentSession) {
            logger.warn('No session found, redirecting to login', {
              pathname,
              requiredUserType: userType,
              allowBothTypes,
            });

            if (onAuthFailure) {
              onAuthFailure();
            }

            // Determine redirect URL
            let redirectUrl = redirectTo;
            if (!redirectUrl) {
              if (userType === 'merchant') {
                redirectUrl = `/merchants/signin?redirect=${encodeURIComponent(pathname)}`;
              } else if (userType === 'customer') {
                redirectUrl = `/customers/signin?redirect=${encodeURIComponent(pathname)}`;
              } else {
                redirectUrl = '/';
              }
            }

            router.replace(redirectUrl);
            return;
          }

          // Validate user type if specified
          if (userType && currentSession.user_type !== userType && !allowBothTypes) {
            logger.warn('User type mismatch', {
              required: userType,
              actual: currentSession.user_type,
              pathname,
            });

            setError(`This page requires ${userType} access`);

            // Redirect to appropriate login
            const redirectUrl =
              redirectTo || `/${userType}s/signin?redirect=${encodeURIComponent(pathname)}`;
            router.replace(redirectUrl);
            return;
          }

          // Check onboarding status if required
          if (
            checkOnboarding &&
            userType === 'customer' &&
            currentSession.user_type === 'customer'
          ) {
            const needsOnboarding = !currentSession.finished_onboarding;
            if (needsOnboarding && !pathname.includes('onboarding')) {
              logger.info('Customer needs onboarding, redirecting', {
                customerId: currentSession.customer_id,
                pathname,
              });

              const onboardingUrl = redirectTo || '/customers/onboarding';
              router.replace(onboardingUrl);
              return;
            }
          }

          // Session is valid
          setSession(currentSession);
          logger.debug('Auth check passed', {
            userType: currentSession.user_type,
            pathname,
          });
        } catch (error) {
          logger.error('Auth check failed', error);
          setError('Failed to verify authentication');

          if (onAuthFailure) {
            onAuthFailure();
          }

          // Redirect to home on error
          router.replace(redirectTo || '/');
        } finally {
          setIsLoading(false);
        }
      };

      checkAuth();
    }, [router, pathname]); // eslint-disable-line react-hooks/exhaustive-deps

    // Show loading state
    if (isLoading) {
      return <LoadingComponent />;
    }

    // Show error state
    if (error) {
      return (
        <div className="flex min-h-screen items-center justify-center">
          <div className="text-center">
            <h2 className="text-2xl font-semibold text-destructive mb-2">Authentication Error</h2>
            <p className="text-muted-foreground">{error}</p>
          </div>
        </div>
      );
    }

    // Session not found (shouldn't reach here, but just in case)
    if (!session) {
      return null;
    }

    // Render the protected component with session
    return <Component {...props} session={session} />;
  };

  // Set display name for debugging
  WrappedComponent.displayName = `withAuth(${Component.displayName || Component.name || 'Component'})`;

  return WrappedComponent;
}

// Default loading component
function DefaultLoadingComponent() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <LoadingSpinner message="Verifying authentication..." />
    </div>
  );
}

// Convenience functions for specific user types
export const withMerchantAuth = <P extends object>(
  Component: ComponentType<P & WithAuthProps>,
  options?: Omit<WithAuthOptions, 'userType'>
) => withAuth(Component, { ...options, userType: 'merchant' });

export const withCustomerAuth = <P extends object>(
  Component: ComponentType<P & WithAuthProps>,
  options?: Omit<WithAuthOptions, 'userType'>
) => withAuth(Component, { ...options, userType: 'customer' });

// Type guard components for inline usage
export function RequireAuth({
  children,
  userType,
  fallback,
  ...options
}: {
  children: (session: Session) => React.ReactNode;
  fallback?: React.ReactNode;
} & WithAuthOptions) {
  const [session, setSession] = useState<Session | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const currentSession = userType
          ? await UnifiedSessionClient.getByType(userType)
          : await UnifiedSessionClient.get();

        if (!currentSession) {
          if (options.redirectTo) {
            router.replace(options.redirectTo);
          } else if (userType) {
            router.replace(`/${userType}s/signin?redirect=${encodeURIComponent(pathname)}`);
          }
          return;
        }

        setSession(currentSession);
      } catch (error) {
        logger.error('RequireAuth check failed', error);
        if (options.onAuthFailure) {
          options.onAuthFailure();
        }
      } finally {
        setIsLoading(false);
      }
    };

    checkAuth();
  }, [userType, router, pathname, options]);

  if (isLoading) {
    return <>{fallback || <LoadingSpinner />}</>;
  }

  if (!session) {
    return <>{fallback || null}</>;
  }

  return <>{children(session)}</>;
}
