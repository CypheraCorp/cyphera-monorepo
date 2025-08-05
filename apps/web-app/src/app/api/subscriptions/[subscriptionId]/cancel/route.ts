import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import { logger } from '@/lib/core/logger/logger';
import type { CancelSubscriptionRequest } from '@/types/subscription';

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
    const body = await request.json() as CancelSubscriptionRequest;
    
    const result = await api.subscriptions.cancelSubscription(
      userContext,
      subscriptionId,
      body
    );
    
    return NextResponse.json(result);
  } catch (error) {
    logger.error('Error cancelling subscription:', error);
    return NextResponse.json(
      { error: 'Failed to cancel subscription' },
      { status: 500 }
    );
  }
}