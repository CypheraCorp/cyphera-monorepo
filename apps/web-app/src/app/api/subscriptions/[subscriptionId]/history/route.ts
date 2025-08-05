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
    const { searchParams } = new URL(request.url);
    const limit = parseInt(searchParams.get('limit') || '50', 10);
    
    const history = await api.subscriptions.getSubscriptionHistory(
      userContext,
      subscriptionId,
      limit
    );
    
    return NextResponse.json(history);
  } catch (error) {
    logger.error('Error fetching subscription history:', error);
    return NextResponse.json(
      { error: 'Failed to fetch subscription history' },
      { status: 500 }
    );
  }
}