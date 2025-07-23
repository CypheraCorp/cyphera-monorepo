import { NextRequest } from 'next/server';
import { CypheraAPIClient } from '@/services/cyphera-api';
import { logger } from '@/lib/core/logger/logger';

export interface UserRequestContext {
  access_token: string;
  account_id?: string;
  user_id?: string;
  workspace_id?: string;
}

/**
 * Extract API context from cookie-based session (for API routes)
 * This replaces the previous middleware-based injection approach
 */
export async function getAPIContextFromSession(request: NextRequest) {
  // Get session from cookie
  const sessionCookie = request.cookies.get('cyphera-session');

  logger.debug(
    'All cookies',
    request.cookies.getAll().map((c) => ({ name: c.name, value: c.value.substring(0, 20) + '...' }))
  );

  if (!sessionCookie) {
    logger.debug('No session cookie found');
    throw new Error('No session cookie found');
  }

  logger.debug('Found session cookie', { preview: sessionCookie.value.substring(0, 20) + '...' });

  // Define session data type
  interface SessionData {
    access_token: string;
    account_id?: string;
    user_id?: string;
    workspace_id?: string;
    email: string;
    expires_at?: number;
  }

  // Decode session data from cookie
  let sessionData: SessionData;
  try {
    const decodedSession = Buffer.from(sessionCookie.value, 'base64').toString('utf-8');
    sessionData = JSON.parse(decodedSession) as SessionData;

    // Check if session is expired
    if (sessionData.expires_at && sessionData.expires_at < Date.now() / 1000) {
      logger.debug('Session expired');
      throw new Error('Session expired');
    }
  } catch (error) {
    logger.error('Failed to decode session', {
      error: error instanceof Error ? error.message : error,
    });
    throw new Error('Invalid session format');
  }

  logger.debug('Found session for user', { email: sessionData.email });

  // Create user context from session data
  const userContext: UserRequestContext = {
    access_token: sessionData.access_token,
    account_id: sessionData.account_id,
    user_id: sessionData.user_id,
    workspace_id: sessionData.workspace_id,
  };

  // Create API client instance
  const api = new CypheraAPIClient();

  return { api, userContext };
}

/**
 * LEGACY: Extract API context from request headers (for middleware-injected requests)
 * This is kept for backward compatibility but should be replaced with getAPIContextFromSession
 */
export async function getAPIContext(request: NextRequest) {
  // Get headers from the request
  const headers = request.headers;

  // Extract authorization header
  const authHeader = headers.get('authorization');

  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    throw new Error('No valid authorization header found');
  }

  // Extract JWT token from Authorization header
  const accessToken = authHeader.replace('Bearer ', '');

  // Extract context headers that were injected by middleware
  const accountId = headers.get('x-account-id');
  const userId = headers.get('x-user-id');
  const workspaceId = headers.get('x-workspace-id');

  // Create user context
  const userContext: UserRequestContext = {
    access_token: accessToken,
    account_id: accountId || undefined,
    user_id: userId || undefined,
    workspace_id: workspaceId || undefined,
  };

  // Validate that we have the required access token
  if (!userContext.access_token) {
    throw new Error('Missing access token in context');
  }

  // Create API client instance
  const api = new CypheraAPIClient();

  return { api, userContext };
}
