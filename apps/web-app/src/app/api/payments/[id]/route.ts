import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import logger from '@/lib/core/logger/logger';
import { withValidation } from '@/lib/validation/validate';
import { z } from 'zod';

// Schema for payment ID parameter
const paymentIdSchema = z.object({
  id: z.string().uuid(),
});

/**
 * GET /api/payments/[id]
 * Gets a single payment by ID for the current account
 */
export const GET = withValidation(
  { paramsSchema: paymentIdSchema },
  async (request, { params }) => {
    try {
      await requireAuth();

      const { api, userContext } = await getAPIContextFromSession(request);
      
      if (!params?.id) {
        return NextResponse.json({ error: 'Payment ID is required' }, { status: 400 });
      }

      // Fetch payment data
      const result = await api.payments.getPaymentById(userContext, params.id);

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
      logger.error('Error getting payment', { error });
      const message = error instanceof Error ? error.message : 'Failed to get payment';
      return NextResponse.json({ error: message }, { status: 500 });
    }
  }
);