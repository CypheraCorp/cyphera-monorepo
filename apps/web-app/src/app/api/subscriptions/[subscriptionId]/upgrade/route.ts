import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import { logger } from '@/lib/core/logger/logger';
import type { UpgradeSubscriptionRequest } from '@/types/subscription';

interface RouteParams {
  params: Promise<{ subscriptionId: string }>;
}

export async function POST(
  request: NextRequest,
  { params }: RouteParams
) {
  const { subscriptionId } = await params;
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);
    const body = await request.json() as UpgradeSubscriptionRequest;
    
    const result = await api.subscriptions.upgradeSubscription(
      userContext,
      subscriptionId,
      body
    );
    
    return NextResponse.json(result);
  } catch (error) {
    logger.error('Error upgrading subscription:', error);
    return NextResponse.json(
      { error: 'Failed to upgrade subscription' },
      { status: 500 }
    );
  }
}