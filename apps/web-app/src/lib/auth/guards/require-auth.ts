import { redirect } from 'next/navigation';
import { getSession, isSessionValid } from '../session/session';
import { logger } from '@/lib/core/logger/logger';

/**
 * Server-side authentication check using Web3Auth sessions
 * Validates JWT token and redirects to login if not authenticated
 */
export async function requireAuth() {
  try {
    const session = await getSession();

    if (!session) {
      logger.debug('No session found, redirecting to signin');
      redirect('/merchants/signin');
    }

    if (!isSessionValid(session)) {
      logger.debug('Invalid session, redirecting to signin');
      redirect('/merchants/signin');
    }

    logger.debug('Authentication successful', { email: session.user.email });
    return session.user;
  } catch (error) {
    logger.error('Auth error', { error: error instanceof Error ? error.message : error });
    redirect('/merchants/signin');
  }
}

/**
 * Customer-specific authentication check
 * Validates customer session and redirects to customer login if not authenticated
 */
export async function requireCustomerAuth() {
  try {
    const session = await getSession();

    if (!session) {
      logger.debug('No customer session found, redirecting to signin');
      redirect('/customers/signin');
    }

    if (!isSessionValid(session)) {
      logger.debug('Invalid customer session, redirecting to signin');
      redirect('/customers/signin');
    }

    logger.debug('Customer authentication successful', { email: session.user.email });
    return session.user;
  } catch (error) {
    logger.error('Customer auth error', { error: error instanceof Error ? error.message : error });
    redirect('/customers/signin');
  }
}
