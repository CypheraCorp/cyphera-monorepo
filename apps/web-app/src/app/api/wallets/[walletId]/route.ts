import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import type { UpdateWalletRequest } from '@/types/wallet';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { updateWalletSchema, walletIdParamSchema } from '@/lib/validation/schemas/wallet';

interface RouteParams {
  params: Promise<Record<string, string>>;
}

/**
 * PUT /api/wallets/[walletId]
 * Updates a specific wallet by ID
 */
export const PUT = withCSRFProtection(
  async (request: NextRequest, context: RouteParams) => {
    try {
      await requireAuth();
      
      // Validate params
      const { walletId } = await context.params;
      const paramsValidation = walletIdParamSchema.safeParse({ walletId });
      if (!paramsValidation.success) {
        return NextResponse.json(
          { error: 'Invalid wallet ID format' },
          { status: 400 }
        );
      }
      
      // Validate body
      const body = await request.json();
      const bodyValidation = updateWalletSchema.safeParse(body);
      if (!bodyValidation.success) {
        return NextResponse.json(
          { 
            error: 'Validation failed',
            details: bodyValidation.error.errors 
          },
          { status: 400 }
        );
      }

      const { api, userContext } = await getAPIContextFromSession(request);
      const wallet = await api.wallets.updateWallet(userContext, walletId, bodyValidation.data);

      return NextResponse.json(wallet);
    } catch (error) {
      if (error instanceof Error && error.message === 'Unauthorized') {
        return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
      }
      logger.error('Error updating wallet', { error });
      const message = error instanceof Error ? error.message : 'Failed to update wallet';
      return NextResponse.json({ error: message }, { status: 500 });
    }
  }
);

/**
 * DELETE /api/wallets/[walletId]
 * Deletes a specific wallet by ID
 */
export const DELETE = withCSRFProtection(
  async (request: NextRequest, context: RouteParams) => {
    try {
      await requireAuth();
      
      // Validate params
      const { walletId } = await context.params;
      const paramsValidation = walletIdParamSchema.safeParse({ walletId });
      if (!paramsValidation.success) {
        return NextResponse.json(
          { error: 'Invalid wallet ID format' },
          { status: 400 }
        );
      }
      
      const { api, userContext } = await getAPIContextFromSession(request);
      await api.wallets.deleteWallet(userContext, walletId);
      return new NextResponse(null, { status: 204 });
    } catch (error) {
      if (error instanceof Error && error.message === 'Unauthorized') {
        return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
      }
      logger.error('Error deleting wallet', { error });
      const message = error instanceof Error ? error.message : 'Failed to delete wallet';
      return NextResponse.json({ error: message }, { status: 500 });
    }
  }
);
