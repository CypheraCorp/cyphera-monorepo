import { NextRequest, NextResponse } from 'next/server';
import { apiCache } from '@/lib/cache/api-cache';
import { logger } from '@/lib/core/logger/logger';

/**
 * POST /api/cache/clear
 * Clear server-side API cache (typically called during logout)
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function POST(_request: NextRequest) {
  try {
    logger.info('Clearing server-side API cache');

    // Clear the entire API cache
    apiCache.clear();

    logger.info('API cache cleared successfully');

    return NextResponse.json({
      success: true,
      message: 'API cache cleared successfully',
    });
  } catch (error) {
    logger.error('Failed to clear API cache', {
      error: error instanceof Error ? error.message : error,
    });

    const message = error instanceof Error ? error.message : 'Failed to clear API cache';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
