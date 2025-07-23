import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import type { UpdateWalletRequest } from '@/types/wallet';
import logger from '@/lib/core/logger/logger';

interface RouteParams {
  params: Promise<{
    walletId: string;
  }>;
}

/**
 * PUT /api/wallets/[walletId]
 * Updates a specific wallet by ID
 */
export async function PUT(request: NextRequest, { params }: RouteParams) {
  try {
    await requireAuth();
    const body = (await request.json()) as UpdateWalletRequest;

    const { api, userContext } = await getAPIContextFromSession(request);
    const { walletId } = await params;
    const wallet = await api.wallets.updateWallet(userContext, walletId, body);

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

/**
 * DELETE /api/wallets/[walletId]
 * Deletes a specific wallet by ID
 */
export async function DELETE(request: NextRequest, { params }: RouteParams) {
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);
    const { walletId } = await params;
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
