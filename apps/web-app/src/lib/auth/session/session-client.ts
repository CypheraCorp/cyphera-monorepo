// Client-side session management functions
// These work in browser environments without Next.js server dependencies

import type { CypheraUser, CypheraSession } from './session';
import { clientLogger } from '@/lib/core/logger/logger-client';

/**
 * Client-side function to get the current session from cookies
 * Works in browser environments
 */
export function getSessionFromCookie(): CypheraSession | null {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    // Parse cookies from document.cookie
    const cookies = document.cookie.split(';').reduce(
      (acc, cookie) => {
        const [key, value] = cookie.trim().split('=');
        acc[key] = value;
        return acc;
      },
      {} as Record<string, string>
    );

    // Try merchant session first, then customer session
    const sessionCookie = cookies['cyphera-session'] || cookies['cyphera-customer-session'];

    if (!sessionCookie) {
      return null;
    }

    // Decode URI component and parse JSON
    const sessionData = JSON.parse(decodeURIComponent(sessionCookie));

    // Validate session hasn't expired
    if (sessionData.expires_at && sessionData.expires_at < Date.now() / 1000) {
      clientLogger.debug('Session expired, returning null');
      return null;
    }

    // Validate required fields
    if (!sessionData.user || !sessionData.access_token) {
      clientLogger.warn('Invalid session data structure');
      return null;
    }

    return sessionData as CypheraSession;
  } catch (error) {
    clientLogger.error('Error getting session from cookie', {
      error: error instanceof Error ? error.message : error,
    });
    return null;
  }
}

/**
 * Client-side function to get the current user from session
 */
export async function getUser(): Promise<CypheraUser | null> {
  const session = getSessionFromCookie();
  return session?.user || null;
}

/**
 * Client-side function to check if user is authenticated
 */
export function isAuthenticated(): boolean {
  const session = getSessionFromCookie();
  return !!(session && session.user && session.access_token);
}

/**
 * Client-side function to clear the session cookie (logout)
 */
export function clearClientSession(sessionType: 'merchant' | 'customer' = 'merchant'): void {
  const cookieName = sessionType === 'merchant' ? 'cyphera-session' : 'cyphera-customer-session';

  // Clear the cookie
  document.cookie = `${cookieName}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; samesite=strict; secure`;
  clientLogger.info(`Cleared ${sessionType} session cookie`);

  // Redirect to appropriate signin page
  if (typeof window !== 'undefined') {
    window.location.href = sessionType === 'merchant' ? '/merchants/signin' : '/customers/signin';
  }
}
