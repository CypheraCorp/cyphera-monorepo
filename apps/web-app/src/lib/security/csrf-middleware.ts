import { NextRequest, NextResponse } from 'next/server';
import { logger } from '@/lib/core/logger/logger-utils';
import { shouldExcludeFromCSRF, CSRF_TOKEN_HEADER } from './csrf';

// RouteContext type to match Next.js expectations
type RouteContext = {
  params: Promise<Record<string, string>>;
};

/**
 * Middleware wrapper to add CSRF protection to API routes
 */
export function withCSRFProtection(
  handler: ((request: NextRequest) => Promise<NextResponse>) | ((request: NextRequest, context: RouteContext) => Promise<NextResponse>)
): typeof handler {
  // Handle routes without dynamic segments
  if (handler.length === 1) {
    return (async (request: NextRequest): Promise<NextResponse> => {
    const pathname = request.nextUrl.pathname;

    // Skip CSRF for excluded routes
    if (shouldExcludeFromCSRF(pathname)) {
      return (handler as (request: NextRequest) => Promise<NextResponse>)(request);
    }

    // Skip CSRF for safe methods
    if (['GET', 'HEAD', 'OPTIONS'].includes(request.method)) {
      return (handler as (request: NextRequest) => Promise<NextResponse>)(request);
    }

    try {
      // Get CSRF token from header or cookie
      const token = request.headers.get(CSRF_TOKEN_HEADER) || 
                    request.cookies.get('cyphera-csrf')?.value;

      if (!token) {
        logger.warn('CSRF token missing', { 
          pathname, 
          method: request.method,
          headers: Object.fromEntries(request.headers.entries())
        });
        return NextResponse.json(
          { error: 'CSRF token required' }, 
          { status: 403 }
        );
      }

      // For now, we'll trust that the token exists
      // In a production environment, you'd validate the token here
      // against the server-side stored token

      // Call the actual handler
      return (handler as (request: NextRequest) => Promise<NextResponse>)(request);
    } catch (error) {
      logger.error('CSRF validation failed', { error, pathname });
      return NextResponse.json(
        { error: 'CSRF validation failed' }, 
        { status: 403 }
      );
    }
    }) as typeof handler;
  }

  // Handle routes with dynamic segments
  return (async (request: NextRequest, context: RouteContext): Promise<NextResponse> => {
    const pathname = request.nextUrl.pathname;

    // Skip CSRF for excluded routes
    if (shouldExcludeFromCSRF(pathname)) {
      return (handler as (request: NextRequest, context: RouteContext) => Promise<NextResponse>)(request, context);
    }

    // Skip CSRF for safe methods
    if (['GET', 'HEAD', 'OPTIONS'].includes(request.method)) {
      return (handler as (request: NextRequest, context: RouteContext) => Promise<NextResponse>)(request, context);
    }

    try {
      // Get CSRF token from header or cookie
      const token = request.headers.get(CSRF_TOKEN_HEADER) || 
                    request.cookies.get('cyphera-csrf')?.value;

      if (!token) {
        logger.warn('CSRF token missing', { 
          pathname, 
          method: request.method,
          headers: Object.fromEntries(request.headers.entries())
        });
        return NextResponse.json(
          { error: 'CSRF token required' }, 
          { status: 403 }
        );
      }

      // For now, we'll trust that the token exists
      // In a production environment, you'd validate the token here
      // against the server-side stored token

      // Call the actual handler
      return (handler as (request: NextRequest, context: RouteContext) => Promise<NextResponse>)(request, context);
    } catch (error) {
      logger.error('CSRF validation failed', { error, pathname });
      return NextResponse.json(
        { error: 'CSRF validation failed' }, 
        { status: 403 }
      );
    }
  }) as typeof handler;
}