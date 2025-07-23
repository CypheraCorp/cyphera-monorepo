import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { NextRequest, NextResponse } from 'next/server';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import logger from '@/lib/core/logger/logger';

/**
 * GET /api/wallets
 * Gets all wallets for the current account
 */
export async function GET(request: NextRequest) {
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);

    // Fetch fresh data without caching
    const wallets = await api.wallets.listWallets(userContext);

    // Return response with no-cache headers
    const response = NextResponse.json(wallets);
    response.headers.set('Cache-Control', 'no-store, no-cache, must-revalidate');
    response.headers.set('Pragma', 'no-cache');
    response.headers.set('Expires', '0');
    return response;
  } catch (error) {
    logger.error('Error fetching wallets', { error });
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'An unknown error occurred' },
      { status: 500 }
    );
  }
}

/**
 * POST /api/wallets
 * Creates a new wallet
 */
export async function POST(request: NextRequest) {
  try {
    await requireAuth();
    const data = await request.json();

    if (!data.nickname) {
      return NextResponse.json({ error: 'Wallet nickname is required' }, { status: 400 });
    }

    const { api, userContext } = await getAPIContextFromSession(request);
    const wallet = await api.wallets.createWallet(userContext, data);

    return NextResponse.json(wallet);
  } catch (error) {
    logger.error('Error creating wallet', { error });
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'An unknown error occurred' },
      { status: 500 }
    );
  }
}
