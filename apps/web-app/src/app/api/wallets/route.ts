import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { NextRequest, NextResponse } from 'next/server';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { createWalletSchema, walletQuerySchema } from '@/lib/validation/schemas/wallet';

/**
 * GET /api/wallets
 * Gets all wallets for the current account
 */
export const GET = withValidation(
  { querySchema: walletQuerySchema },
  async (request, { query }) => {
    try {
      await requireAuth();
      const { api, userContext } = await getAPIContextFromSession(request);

      // Fetch fresh data without caching
      // Note: listWallets doesn't support query params in this implementation
      // TODO: If filtering is needed, it should be implemented in the WalletsAPI service
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
);

/**
 * POST /api/wallets
 * Creates a new wallet
 */
export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: createWalletSchema },
    async (request, { body }) => {
      try {
        await requireAuth();
        
        const { api, userContext } = await getAPIContextFromSession(request);
        
        if (!body) {
          return NextResponse.json({ error: 'Request body is required' }, { status: 400 });
        }
        
        const wallet = await api.wallets.createWallet(userContext, body);

        return NextResponse.json(wallet);
      } catch (error) {
        logger.error('Error creating wallet', { error });
        return NextResponse.json(
          { error: error instanceof Error ? error.message : 'An unknown error occurred' },
          { status: 500 }
        );
      }
    }
  )
);
