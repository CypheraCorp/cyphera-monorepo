import { NextResponse } from 'next/server';
import { apiCache } from '@/lib/cache/api-cache';
import { logger } from '@/lib/core/logger/logger';
import { UnifiedSessionService } from '@/lib/auth/session/unified-session';

/**
 * POST /api/auth/customer/logout
 * Customer logout endpoint that clears customer sessions
 */
export async function POST() {
  try {
    // Clear customer session using unified service
    await UnifiedSessionService.clearByType('customer');

    // Clear API cache for this customer
    apiCache.clear();
    logger.info('Customer logout successful, API cache cleared');

    return NextResponse.json({ success: true });
  } catch (error) {
    logger.error('Customer logout failed', {
      error: error instanceof Error ? error.message : error,
    });

    const message = error instanceof Error ? error.message : 'Failed to logout customer';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
