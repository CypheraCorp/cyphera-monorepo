import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import logger from '@/lib/core/logger/logger';
import { withValidation } from '@/lib/validation/validate';
import { transactionQuerySchema } from '@/lib/validation/schemas/transaction';

/**
 * GET /api/transactions
 * Gets Cyphera transactions (subscription events) for the current account with pagination
 */
export const GET = withValidation(
  { querySchema: transactionQuerySchema },
  async (request, { query }) => {
    try {
      await requireAuth();

      const { api, userContext } = await getAPIContextFromSession(request);

      // Build params from validated query
      const params = {
        ...(query?.page && { page: query.page }),
        ...(query?.limit && { limit: query.limit }),
        ...(query?.type && { type: query.type }),
        ...(query?.status && { status: query.status }),
        ...(query?.customer && { customer: query.customer }),
        ...(query?.walletId && { walletId: query.walletId }),
        ...(query?.startDate && { startDate: query.startDate }),
        ...(query?.endDate && { endDate: query.endDate }),
        ...(query?.minAmount && { minAmount: query.minAmount }),
        ...(query?.maxAmount && { maxAmount: query.maxAmount }),
      };

      // Fetch fresh data without caching
      const result = await api.transactions.getTransactions(userContext, params);

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
);
