import { NextResponse } from 'next/server';
import { logger } from '@/lib/core/logger/logger';
import { UnifiedSessionService } from '@/lib/auth/session/unified-session';

/**
 * POST /api/auth/logout
 * Clears the session cookie
 */
export async function POST() {
  try {
    // Clear merchant session using unified service
    await UnifiedSessionService.clearByType('merchant');

    logger.info('Merchant logout successful');
    return NextResponse.json({ message: 'Logged out successfully' });
  } catch (error) {
    logger.error('Logout failed', { error: error instanceof Error ? error.message : error });
    return NextResponse.json({ error: 'Failed to logout' }, { status: 500 });
  }
}
