import { NextResponse } from 'next/server';
import { logger } from '@/lib/core/logger/logger';
import { UnifiedSessionService } from '@/lib/auth/session/unified-session';

// Type for cached auth response
interface CachedAuthData {
  data: {
    user: {
      id: string;
      email: string;
    };
    account: {
      id: string;
    };
    workspace: {
      id: string;
    };
  };
  timestamp: number;
}

// Simple cache for auth responses (short-lived)
const authCache = new Map<string, CachedAuthData>();
const AUTH_CACHE_DURATION = 30 * 1000; // 30 seconds

/**
 * GET /api/auth/me
 * Check if the current user has a valid session
 */
export async function GET() {
  try {
    // Get merchant session using unified service
    const session = await UnifiedSessionService.getByType('merchant');

    if (!session) {
      return NextResponse.json({ error: 'No session found' }, { status: 401 });
    }

    // Create cache key from session data
    const cacheKey = `${session.user_id}_${session.account_id}`;

    // Check cache first
    const cached = authCache.get(cacheKey);
    if (cached && Date.now() - cached.timestamp < AUTH_CACHE_DURATION) {
      return NextResponse.json({ ...cached.data, session });
    }

    // Prepare response data
    const responseData = {
      user: {
        id: session.user_id || '',
        email: session.email || '',
      },
      account: {
        id: session.account_id || '',
      },
      workspace: {
        id: session.workspace_id || '',
      },
    };

    // Cache the response
    authCache.set(cacheKey, { data: responseData, timestamp: Date.now() });

    return NextResponse.json({ ...responseData, session });
  } catch (error) {
    logger.error('Session validation failed', {
      error: error instanceof Error ? error.message : error,
    });

    return NextResponse.json({ error: 'Session validation failed' }, { status: 500 });
  }
}
