// Middleware disabled - using direct JWT authentication instead
// export { default } from 'next-auth/middleware'

import { NextRequest, NextResponse } from 'next/server';
import { logger } from '@/lib/core/logger/edge-logger';
import {
  UnifiedSessionService,
  isMerchantSession,
  isCustomerSession,
} from '@/lib/auth/session/unified-session';

// Routes that don't require authentication
const publicRoutes = [
  '/',
  '/api/auth/signin',
  '/api/auth/me',
  '/api/auth/callback',
  '/api/auth/customer/signin',
  '/api/auth/customer/me',
  '/api/auth/customer/logout',
  '/verify-email',
  '/merchants/signin',
  '/customers/signin',
];

// Routes that require merchant authentication
const merchantProtectedRoutes = [
  '/merchants/dashboard',
  '/merchants/onboarding',
  '/merchants/products',
  '/merchants/subscriptions',
  '/merchants/transactions',
  '/merchants/wallets',
  '/merchants/settings',
  '/merchants/customers',
];

// Routes that require customer authentication
const customerProtectedRoutes = [
  '/customers/dashboard',
  '/customers/subscriptions',
  '/customers/wallet',
  '/customers/settings',
];

// API routes that need JWT injection
const protectedAPIRoutes = [
  '/api/customers',
  '/api/products',
  '/api/subscriptions',
  '/api/transactions',
  '/api/wallets',
  '/api/accounts/onboard',
];

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Skip middleware for static files and Next.js internals
  if (
    pathname.startsWith('/_next') ||
    pathname.startsWith('/favicon') ||
    pathname.startsWith('/icon') ||
    pathname.includes('.')
  ) {
    return NextResponse.next();
  }

  // Get session using unified service
  const session = await UnifiedSessionService.getFromRequest(request);
  const hasMerchantSession = session && isMerchantSession(session);
  const hasCustomerSession = session && isCustomerSession(session);

  // Check route types
  const isPublicRoute = publicRoutes.includes(pathname) || pathname.startsWith('/public/prices') || pathname.startsWith('/pay/');
  const isMerchantProtectedRoute = merchantProtectedRoutes.some((route) =>
    pathname.startsWith(route)
  );
  const isCustomerProtectedRoute = customerProtectedRoutes.some((route) =>
    pathname.startsWith(route)
  );
  const isProtectedAPI = protectedAPIRoutes.some((route) => pathname.startsWith(route));

  // Debug logging for auth routes
  if (pathname.includes('signin') || pathname.includes('auth')) {
    logger.debug('Middleware debug', {
      pathname,
      hasMerchantSession,
      hasCustomerSession,
      sessionType: session?.user_type,
      isPublicRoute,
      isMerchantProtectedRoute,
      isCustomerProtectedRoute,
    });
  }

  // Allow public routes without authentication
  if (isPublicRoute) {
    return NextResponse.next();
  }

  // For merchant protected routes, check merchant session
  if (isMerchantProtectedRoute && !hasMerchantSession) {
    logger.debug('Redirecting to merchant signin', { pathname });
    const redirectUrl = new URL('/merchants/signin', request.url);
    redirectUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(redirectUrl);
  }

  // For customer protected routes, check customer session
  if (isCustomerProtectedRoute && !hasCustomerSession) {
    logger.debug('Redirecting to customer signin', { pathname });
    const redirectUrl = new URL('/customers/signin', request.url);
    return NextResponse.redirect(redirectUrl);
  }

  // For protected API routes, validate session exists and inject headers
  if (isProtectedAPI) {
    if (!session) {
      logger.warn('API request without session, returning 401', { pathname });
      return new NextResponse('Unauthorized', { status: 401 });
    }

    // Session is already validated by UnifiedSessionService
    const response = NextResponse.next();
    response.headers.set('authorization', `Bearer ${session.access_token}`);

    // Add additional headers based on session type
    if (isCustomerSession(session)) {
      response.headers.set('x-customer-id', session.customer_id);
      if (session.wallet_id) {
        response.headers.set('x-wallet-id', session.wallet_id);
      }
    } else if (isMerchantSession(session)) {
      if (session.account_id) {
        response.headers.set('x-account-id', session.account_id);
      }
      if (session.workspace_id) {
        response.headers.set('x-workspace-id', session.workspace_id);
      }
      // if (session.user_id) {
      //   response.headers.set('x-user-id', session.user_id);
      // }
    }

    logger.debug('Session validated and headers injected for API route', {
      pathname,
      userType: session.user_type,
    });
    return response;
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    '/((?!_next/static|_next/image|favicon.ico).*)',
  ],
};
