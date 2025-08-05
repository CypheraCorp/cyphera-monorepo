import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import { logger } from '@/lib/core/logger/logger';

interface RouteParams {
  params: Promise<{ subscriptionId: string }>;
}

export async function GET(
  request: NextRequest,
  { params }: RouteParams
) {
  const { subscriptionId } = await params;
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);
    
    const subscription = await api.subscriptions.getSubscriptionById(
      userContext,
      subscriptionId
    );
    
    return NextResponse.json(subscription);
  } catch (error) {
    logger.error('Error fetching subscription:', error);
    return NextResponse.json(
      { error: 'Failed to fetch subscription' },
      { status: 500 }
    );
  }
}