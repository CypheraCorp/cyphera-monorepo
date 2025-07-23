import { NextRequest, NextResponse } from 'next/server';
import { logger } from '@/lib/core/logger/logger-utils';
import crypto from 'crypto';

// CSRF secret from environment variable
const CSRF_SECRET = process.env.CSRF_SECRET || 'default-secret-change-in-production';

// CSRF token header name
export const CSRF_TOKEN_HEADER = 'x-csrf-token';

// Routes that should be excluded from CSRF protection
const CSRF_EXCLUDED_ROUTES = [
  '/api/health',
  '/api/auth/callback', // External callbacks
  '/api/auth/signin', // Sign-in endpoints
  '/api/auth/customer/signin', // Customer sign-in
  '/api/auth/me', // Session check endpoints
  '/api/auth/customer/me', // Customer session check
  '/api/webhooks', // Webhook endpoints
  '/api/public', // Public endpoints
];

// Simple CSRF implementation
export const csrf = {
  create: (request: NextRequest): string => {
    // Generate a random token
    const token = crypto.randomBytes(32).toString('hex');
    
    // In a real implementation, you'd store this token server-side
    // For now, we'll use a signed token approach
    const signature = crypto
      .createHmac('sha256', CSRF_SECRET)
      .update(token)
      .digest('hex');
    
    return `${token}.${signature}`;
  },
  
  verify: (token: string): boolean => {
    if (!token || typeof token !== 'string') return false;
    
    const parts = token.split('.');
    if (parts.length !== 2) return false;
    
    const [tokenPart, signature] = parts;
    const expectedSignature = crypto
      .createHmac('sha256', CSRF_SECRET)
      .update(tokenPart)
      .digest('hex');
    
    return crypto.timingSafeEqual(
      Buffer.from(signature),
      Buffer.from(expectedSignature)
    );
  }
};

/**
 * Check if a route should be excluded from CSRF protection
 */
export function shouldExcludeFromCSRF(pathname: string): boolean {
  return CSRF_EXCLUDED_ROUTES.some((route) => pathname.startsWith(route));
}

/**
 * Apply CSRF protection to API routes
 */
export async function withCSRFProtection(
  request: NextRequest,
  handler: () => Promise<NextResponse>
): Promise<NextResponse> {
  const pathname = request.nextUrl.pathname;

  // Skip CSRF for excluded routes
  if (shouldExcludeFromCSRF(pathname)) {
    return handler();
  }

  // Skip CSRF for safe methods
  if (['GET', 'HEAD', 'OPTIONS'].includes(request.method)) {
    return handler();
  }

  try {
    // Verify CSRF token
    const token = request.headers.get(CSRF_TOKEN_HEADER) || 
                  request.cookies.get('cyphera-csrf')?.value;

    if (!token) {
      logger.warn('CSRF token missing', { pathname, method: request.method });
      return new NextResponse('CSRF token required', { status: 403 });
    }

    // Verify the token
    if (!csrf.verify(token)) {
      logger.warn('CSRF token invalid', { pathname, method: request.method });
      return new NextResponse('CSRF token invalid', { status: 403 });
    }

    return handler();
  } catch (error) {
    logger.error('CSRF validation failed', { error, pathname });
    return new NextResponse('CSRF validation failed', { status: 403 });
  }
}

/**
 * Get CSRF token for client-side usage
 */
export async function getCSRFToken(): Promise<string | null> {
  try {
    const response = await fetch('/api/auth/csrf', {
      method: 'GET',
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to fetch CSRF token');
    }

    const data = await response.json();
    return data.token;
  } catch (error) {
    logger.error('Failed to get CSRF token', error);
    return null;
  }
}