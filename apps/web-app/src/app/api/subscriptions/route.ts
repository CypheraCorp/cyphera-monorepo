import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import logger from '@/lib/core/logger/logger';
import { withValidation } from '@/lib/validation/validate';
import { subscriptionQuerySchema } from '@/lib/validation/schemas/subscription';

/**
 * GET /api/subscriptions
 * Gets subscriptions for the current account with pagination
 */
export const GET = withValidation(
  { querySchema: subscriptionQuerySchema },
  async (request, { query }) => {
    try {
      await requireAuth();

      const { api, userContext } = await getAPIContextFromSession(request);

      // Build params from validated query
      const params = {
        ...(query?.page && { page: query.page }),
        ...(query?.limit && { limit: query.limit }),
        ...(query?.status && { status: query.status }),
        ...(query?.customer_id && { customer_id: query.customer_id }),
        ...(query?.product_id && { product_id: query.product_id }),
      };

      // Fetch fresh data without caching
      const result = await api.subscriptions.getSubscriptions(userContext, params);

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
      logger.error('Error getting subscriptions', { error });
      const message = error instanceof Error ? error.message : 'Failed to get subscriptions';
      return NextResponse.json({ error: message }, { status: 500 });
    }
  }
);
