import { cookies } from 'next/headers';
import { NextRequest } from 'next/server';
import { logger } from '@/lib/core/logger/logger-utils';

// Session Types
export type UserType = 'merchant' | 'customer';

export interface BaseSession {
  access_token: string;
  user_type: UserType;
  expires_at: number;
  created_at: number;
}

export interface MerchantSession extends BaseSession {
  user_type: 'merchant';
  user_id?: string;
  account_id?: string;
  workspace_id?: string;
  email?: string;
}

export interface CustomerSession extends BaseSession {
  user_type: 'customer';
  customer_id: string;
  customer_email: string;
  customer_name?: string;
  wallet_address?: string;
  wallet_id?: string;
  finished_onboarding?: boolean;
}

export type Session = MerchantSession | CustomerSession;

// Cookie Configuration
const COOKIE_CONFIG = {
  merchant: 'cyphera-session',
  customer: 'cyphera-customer-session',
} as const;

const COOKIE_OPTIONS = {
  httpOnly: true,
  secure: process.env.NODE_ENV === 'production',
  sameSite: 'lax' as const,
  path: '/',
  maxAge: 60 * 60 * 24 * 7, // 7 days
};

/**
 * Unified session management service
 */
export class UnifiedSessionService {
  /**
   * Create a new session
   */
  static async create<T extends UserType>(
    data: T extends 'merchant'
      ? Omit<MerchantSession, 'expires_at' | 'created_at'>
      : Omit<CustomerSession, 'expires_at' | 'created_at'>
  ): Promise<T extends 'merchant' ? MerchantSession : CustomerSession> {
    const now = Date.now();
    const session: Session = {
      ...data,
      created_at: now,
      expires_at: now + COOKIE_OPTIONS.maxAge * 1000,
    } as Session;

    const cookieName = COOKIE_CONFIG[session.user_type];
    const cookieValue = this.encode(session);

    (await cookies()).set(cookieName, cookieValue, COOKIE_OPTIONS);

    logger.info(`Session created for ${session.user_type}`, {
      userType: session.user_type,
      userId: isMerchantSession(session) ? session.user_id : session.customer_id,
    });

    return session as T extends 'merchant' ? MerchantSession : CustomerSession;
  }

  /**
   * Get current session (checks both merchant and customer)
   */
  static async get(): Promise<Session | null> {
    const cookieStore = await cookies();

    // Check merchant session first
    const merchantCookie = cookieStore.get(COOKIE_CONFIG.merchant);
    if (merchantCookie?.value) {
      const session = this.decode(merchantCookie.value);
      if (session && this.isValid(session)) {
        return session as MerchantSession;
      }
    }

    // Check customer session
    const customerCookie = cookieStore.get(COOKIE_CONFIG.customer);
    if (customerCookie?.value) {
      const session = this.decode(customerCookie.value);
      if (session && this.isValid(session)) {
        return session as CustomerSession;
      }
    }

    return null;
  }

  /**
   * Get session by type
   */
  static async getByType<T extends UserType>(
    userType: T
  ): Promise<(T extends 'merchant' ? MerchantSession : CustomerSession) | null> {
    const cookieStore = await cookies();
    const cookieName = COOKIE_CONFIG[userType];
    const cookie = cookieStore.get(cookieName);

    if (!cookie?.value) {
      return null;
    }

    const session = this.decode(cookie.value);
    if (!session || !this.isValid(session)) {
      return null;
    }

    return session as T extends 'merchant' ? MerchantSession : CustomerSession;
  }

  /**
   * Get session from request (for middleware)
   */
  static async getFromRequest(request: NextRequest): Promise<Session | null> {
    // Check merchant session
    const merchantCookie = request.cookies.get(COOKIE_CONFIG.merchant);
    if (merchantCookie?.value) {
      const session = this.decode(merchantCookie.value);
      if (session && this.isValid(session)) {
        return session as MerchantSession;
      }
    }

    // Check customer session
    const customerCookie = request.cookies.get(COOKIE_CONFIG.customer);
    if (customerCookie?.value) {
      const session = this.decode(customerCookie.value);
      if (session && this.isValid(session)) {
        return session as CustomerSession;
      }
    }

    return null;
  }

  /**
   * Update existing session
   */
  static async update(updates: Partial<Session>): Promise<Session | null> {
    const current = await this.get();
    if (!current) {
      return null;
    }

    const updated = { ...current, ...updates } as Session;
    const cookieName = COOKIE_CONFIG[current.user_type];
    const cookieValue = this.encode(updated);

    (await cookies()).set(cookieName, cookieValue, COOKIE_OPTIONS);

    logger.info(`Session updated for ${current.user_type}`, {
      userType: current.user_type,
      updates: Object.keys(updates),
    });

    return updated;
  }

  /**
   * Clear session by type
   */
  static async clearByType(userType: UserType): Promise<void> {
    const cookieName = COOKIE_CONFIG[userType];
    (await cookies()).delete(cookieName);

    logger.info(`Session cleared for ${userType}`);
  }

  /**
   * Clear all sessions
   */
  static async clearAll(): Promise<void> {
    const cookieStore = await cookies();
    cookieStore.delete(COOKIE_CONFIG.merchant);
    cookieStore.delete(COOKIE_CONFIG.customer);

    logger.info('All sessions cleared');
  }

  /**
   * Switch between session types (useful for users with both accounts)
   */
  static async switchTo(userType: UserType): Promise<Session | null> {
    const session = await this.getByType(userType);
    if (!session) {
      logger.warn(`No ${userType} session found to switch to`);
      return null;
    }

    // Optionally update last accessed time
    const updated = await this.update({
      ...session,
      created_at: Date.now(),
    });

    logger.info(`Switched to ${userType} session`);
    return updated;
  }

  /**
   * Check if user has both session types
   */
  static async hasBothSessions(): Promise<boolean> {
    const merchant = await this.getByType('merchant');
    const customer = await this.getByType('customer');
    return !!(merchant && customer);
  }

  /**
   * Get all active sessions
   */
  static async getAllSessions(): Promise<{
    merchant?: MerchantSession;
    customer?: CustomerSession;
  }> {
    const merchant = await this.getByType('merchant');
    const customer = await this.getByType('customer');

    return {
      ...(merchant && { merchant }),
      ...(customer && { customer }),
    };
  }

  /**
   * Validate session expiry
   */
  private static isValid(session: Session): boolean {
    if (!session.expires_at) {
      return true; // No expiry set
    }
    return session.expires_at > Date.now();
  }

  /**
   * Encode session data
   */
  private static encode(session: Session): string {
    return Buffer.from(JSON.stringify(session)).toString('base64');
  }

  /**
   * Decode session data
   */
  private static decode(value: string): Session | null {
    try {
      const decoded = Buffer.from(value, 'base64').toString('utf-8');
      return JSON.parse(decoded) as Session;
    } catch (error) {
      logger.error('Failed to decode session', error);
      return null;
    }
  }
}

// Type guards
export function isMerchantSession(session: Session): session is MerchantSession {
  return session.user_type === 'merchant';
}

export function isCustomerSession(session: Session): session is CustomerSession {
  return session.user_type === 'customer';
}

// Helper functions for common operations
export async function requireMerchantSession(): Promise<MerchantSession> {
  const session = await UnifiedSessionService.getByType('merchant');
  if (!session) {
    throw new Error('Merchant session required');
  }
  return session;
}

export async function requireCustomerSession(): Promise<CustomerSession> {
  const session = await UnifiedSessionService.getByType('customer');
  if (!session) {
    throw new Error('Customer session required');
  }
  return session;
}

export async function requireSession(): Promise<Session> {
  const session = await UnifiedSessionService.get();
  if (!session) {
    throw new Error('Session required');
  }
  return session;
}
