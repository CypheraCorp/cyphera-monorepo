import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import logger from '@/lib/core/logger/logger';
import { withValidation } from '@/lib/validation/validate';
import { z } from 'zod';

// Schema for payment query parameters
const paymentQuerySchema = z.object({
  page: z.coerce.number().min(1).default(1).optional(),
  limit: z.coerce.number().min(1).max(100).default(20).optional(),
  status: z.enum(['pending', 'completed', 'failed', 'processing']).optional(),
  customer_id: z.string().uuid().optional(),
  payment_method: z.enum(['crypto', 'card', 'bank']).optional(),
  start_date: z.string().optional(),
  end_date: z.string().optional(),
});

/**
 * GET /api/payments
 * Gets payments for the current account with pagination and filtering
 */
export const GET = withValidation(
  { querySchema: paymentQuerySchema },
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
        ...(query?.payment_method && { payment_method: query.payment_method }),
        ...(query?.start_date && { start_date: query.start_date }),
        ...(query?.end_date && { end_date: query.end_date }),
      };

      // Fetch fresh data without caching
      const result = await api.payments.getPayments(userContext, params);

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
      logger.error('Error getting payments', { error });
      const message = error instanceof Error ? error.message : 'Failed to get payments';
      return NextResponse.json({ error: message }, { status: 500 });
    }
  }
);