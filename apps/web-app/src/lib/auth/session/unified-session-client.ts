/**
 * Client-side session utilities
 * These functions make API calls to manage sessions from the browser
 */

import { logger } from '@/lib/core/logger/logger-utils';
import type { Session, UserType } from './unified-session';

export class UnifiedSessionClient {
  /**
   * Get current session from API
   */
  static async get(): Promise<Session | null> {
    try {
      // Try merchant session first
      const merchantResponse = await fetch('/api/auth/me');
      if (merchantResponse.ok) {
        const data = await merchantResponse.json();
        return data.session;
      }

      // Try customer session
      const customerResponse = await fetch('/api/auth/customer/me');
      if (customerResponse.ok) {
        const data = await customerResponse.json();
        return data.session;
      }

      return null;
    } catch (error) {
      logger.error('Failed to get session', error);
      return null;
    }
  }

  /**
   * Get session by type
   */
  static async getByType(userType: UserType): Promise<Session | null> {
    try {
      const endpoint = userType === 'merchant' ? '/api/auth/me' : '/api/auth/customer/me';
      const response = await fetch(endpoint);

      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data.session;
    } catch (error) {
      logger.error(`Failed to get ${userType} session`, error);
      return null;
    }
  }

  /**
   * Clear session by type
   */
  static async clearByType(userType: UserType): Promise<boolean> {
    try {
      const endpoint = userType === 'merchant' ? '/api/auth/logout' : '/api/auth/customer/logout';
      const response = await fetch(endpoint, { method: 'POST' });

      return response.ok;
    } catch (error) {
      logger.error(`Failed to clear ${userType} session`, error);
      return false;
    }
  }

  /**
   * Clear all sessions
   */
  static async clearAll(): Promise<boolean> {
    try {
      const [merchantResult, customerResult] = await Promise.all([
        this.clearByType('merchant'),
        this.clearByType('customer'),
      ]);

      return merchantResult || customerResult;
    } catch (error) {
      logger.error('Failed to clear all sessions', error);
      return false;
    }
  }

  /**
   * Check if user has both session types
   */
  static async hasBothSessions(): Promise<boolean> {
    try {
      const [merchant, customer] = await Promise.all([
        this.getByType('merchant'),
        this.getByType('customer'),
      ]);

      return !!(merchant && customer);
    } catch (error) {
      logger.error('Failed to check sessions', error);
      return false;
    }
  }

  /**
   * Get all active sessions
   */
  static async getAllSessions(): Promise<{ merchant?: Session; customer?: Session }> {
    try {
      const [merchant, customer] = await Promise.all([
        this.getByType('merchant'),
        this.getByType('customer'),
      ]);

      return {
        ...(merchant && { merchant }),
        ...(customer && { customer }),
      };
    } catch (error) {
      logger.error('Failed to get all sessions', error);
      return {};
    }
  }

  /**
   * Helper to determine if user needs onboarding
   */
  static async needsOnboarding(userType: UserType): Promise<boolean> {
    const session = await this.getByType(userType);

    if (!session) {
      return true; // No session means needs onboarding
    }

    if (userType === 'customer' && 'finished_onboarding' in session) {
      return !session.finished_onboarding;
    }

    // Add merchant onboarding check if needed
    return false;
  }

  /**
   * Refresh session
   */
  static async refresh(userType: UserType): Promise<Session | null> {
    try {
      const endpoint = userType === 'merchant' ? '/api/auth/refresh' : '/api/auth/customer/refresh';

      const response = await fetch(endpoint, { method: 'POST' });

      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data.session;
    } catch (error) {
      logger.error(`Failed to refresh ${userType} session`, error);
      return null;
    }
  }
}

// Export convenience functions
export const getSession = () => UnifiedSessionClient.get();
export const getMerchantSession = () => UnifiedSessionClient.getByType('merchant');
export const getCustomerSession = () => UnifiedSessionClient.getByType('customer');
export const clearMerchantSession = () => UnifiedSessionClient.clearByType('merchant');
export const clearCustomerSession = () => UnifiedSessionClient.clearByType('customer');
export const clearAllSessions = () => UnifiedSessionClient.clearAll();
export const hasBothSessions = () => UnifiedSessionClient.hasBothSessions();
export const getAllSessions = () => UnifiedSessionClient.getAllSessions();
