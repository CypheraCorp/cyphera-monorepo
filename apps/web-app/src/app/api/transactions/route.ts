import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import logger from '@/lib/core/logger/logger';

/**
 * GET /api/transactions
 * Gets Cyphera transactions (subscription events) for the current account with pagination
 */
export async function GET(request: NextRequest) {
  try {
    await requireAuth();

    const { searchParams } = new URL(request.url);
    const page = searchParams.get('page') || undefined;
    const limit = searchParams.get('limit') || undefined;

    const { api, userContext } = await getAPIContextFromSession(request);

    // Fetch fresh data without caching
    const result = await api.transactions.getTransactions(userContext, {
      page: page ? Number(page) : undefined,
      limit: limit ? Number(limit) : undefined,
    });

    // Return response with no-cache headers
    const response = NextResponse.json(result);
    response.headers.set('Cache-Control', 'no-store, no-cache, must-revalidate');
    response.headers.set('Pragma', 'no-cache');
    response.headers.set('Expires', '0');
    return response;
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error getting transactions', { error });
    const message = error instanceof Error ? error.message : 'Failed to get transactions';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
