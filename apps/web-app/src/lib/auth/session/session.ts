// Web3Auth session management implementation
// Proper session handling with Web3Auth integration

import { cookies } from 'next/headers';
import { logger } from '@/lib/core/logger/logger';

/**
 * User type for Web3Auth integration
 * Matches the original CypheraUser interface structure
 */
export interface CypheraUser {
  id: string;
  email: string;
  name?: string;
  // Custom metadata fields that were stored in Supabase user_metadata
  user_id?: string;
  account_id?: string;
  workspace_id?: string;
  finished_onboarding?: boolean;
  email_verified?: boolean;
  access_token?: string;
  // Additional fields to match Supabase User interface
  aud?: string;
  user_metadata?: Record<string, unknown>;
}

/**
 * Session type for Web3Auth integration
 */
export interface CypheraSession {
  user: CypheraUser;
  access_token: string;
  expires_at?: number; // Unix timestamp
  provider?: string; // Web3Auth provider (google, email, etc.)
}

/**
 * Gets the current session from cookies and validates it
 * Works with both merchant and customer sessions
 */
export async function getSession(): Promise<CypheraSession | null> {
  try {
    const cookieStore = await cookies();

    // Try merchant session first, then customer session
    const merchantSessionCookie = cookieStore.get('cyphera-session');
    const customerSessionCookie = cookieStore.get('cyphera-customer-session');

    const sessionCookie = merchantSessionCookie || customerSessionCookie;

    if (!sessionCookie) {
      return null;
    }

    // Decode session data from cookie
    const isCustomerSession = !!customerSessionCookie;

    // Define session data types
    interface MerchantSessionData {
      user_id: string;
      email: string;
      account_id?: string;
      workspace_id?: string;
      access_token: string;
      expires_at?: number;
    }

    interface CustomerSessionData {
      customer_id: string;
      customer_email: string;
      customer_name?: string;
      access_token: string;
      expires_at?: number;
      finished_onboarding?: boolean;
    }

    let sessionData: MerchantSessionData | CustomerSessionData;
    try {
      // Decode base64 encoded session data
      const decodedSession = Buffer.from(sessionCookie.value, 'base64').toString('utf-8');
      sessionData = JSON.parse(decodedSession);

      // Check if session is expired
      if (sessionData.expires_at && sessionData.expires_at < Date.now() / 1000) {
        logger.debug('Session expired');
        return null;
      }
    } catch (error) {
      logger.debug('Failed to decode session data', {
        error: error instanceof Error ? error.message : error,
      });
      return null;
    }

    // Convert session data to CypheraSession format
    let session: CypheraSession;

    if (isCustomerSession) {
      // Handle customer session format
      const customerData = sessionData as CustomerSessionData;
      session = {
        user: {
          id: customerData.customer_id || '',
          email: customerData.customer_email || '',
          name: customerData.customer_name,
          access_token: customerData.access_token,
          // Customer-specific fields
          finished_onboarding: customerData.finished_onboarding,
        },
        access_token: sessionData.access_token,
        expires_at: sessionData.expires_at,
        provider: 'customer',
      };
    } else {
      // Handle merchant session format
      const merchantData = sessionData as MerchantSessionData;
      session = {
        user: {
          id: merchantData.user_id || '',
          email: merchantData.email || '',
          user_id: merchantData.user_id,
          account_id: merchantData.account_id,
          workspace_id: merchantData.workspace_id,
          access_token: merchantData.access_token,
        },
        access_token: sessionData.access_token,
        expires_at: sessionData.expires_at,
      };
    }

    logger.debug('Valid session found', {
      userId: session.user.id,
      email: session.user.email,
      hasToken: !!session.access_token,
      type: isCustomerSession ? 'customer' : 'merchant',
    });

    return session;
  } catch (error) {
    logger.error('Error getting session', {
      error: error instanceof Error ? error.message : error,
    });
    return null;
  }
}

/**
 * Gets the current user from session
 */
export async function getUser(): Promise<CypheraUser | null> {
  const session = await getSession();
  return session?.user || null;
}

/**
 * User profile response type
 */
export interface UserProfileResponse {
  data: CypheraUser | null;
  error: string | null;
}

/**
 * Gets user profile by ID from backend API
 */
export async function getUserProfileById(userId: string): Promise<UserProfileResponse> {
  try {
    const session = await getSession();
    if (!session) {
      return { data: null, error: 'No active session' };
    }

    // Make API call to backend with JWT token
    const response = await fetch(`/api/users/${userId}`, {
      headers: {
        Authorization: `Bearer ${session.access_token}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      return { data: null, error: `Failed to fetch user profile: ${response.statusText}` };
    }

    const data = await response.json();
    return { data, error: null };
  } catch (error) {
    logger.error('Error fetching user profile', {
      error: error instanceof Error ? error.message : error,
    });
    return { data: null, error: error instanceof Error ? error.message : 'Unknown error' };
  }
}

/**
 * Gets current user profile from backend API
 */
export async function getCurrentUserProfile(): Promise<CypheraUser | null> {
  try {
    const user = await getUser();
    if (!user) {
      return null;
    }

    const result = await getUserProfileById(user.id);
    return result.data;
  } catch (error) {
    logger.error('Error getting current user profile', {
      error: error instanceof Error ? error.message : error,
    });
    return null;
  }
}

/**
 * Validates if a session token is still valid
 */
export function isSessionValid(session: CypheraSession | null): boolean {
  if (!session) return false;

  // Check if session has expired
  if (session.expires_at && session.expires_at < Date.now() / 1000) {
    return false;
  }

  // Check required fields
  return !!(session.user && session.access_token && session.user.id && session.user.email);
}

/**
 * Clears the session cookie (logout)
 */
export function clearSession(sessionType: 'merchant' | 'customer' = 'merchant'): void {
  const cookieName = sessionType === 'merchant' ? 'cyphera-session' : 'cyphera-customer-session';

  if (typeof window !== 'undefined') {
    // Client-side cookie clearing
    document.cookie = `${cookieName}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; samesite=strict; secure`;
    logger.info(`Cleared ${sessionType} session cookie`);
  }
}
