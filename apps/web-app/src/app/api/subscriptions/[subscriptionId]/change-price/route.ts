import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import { logger } from '@/lib/core/logger/logger';

interface RouteParams {
  params: Promise<{ subscriptionId: string }>;
}

interface ChangePriceRequest {
  new_price_cents: number;
}

export async function POST(
  request: NextRequest,
  { params }: RouteParams
) {
  const { subscriptionId } = await params;
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);
    const body = await request.json() as ChangePriceRequest;
    
    const result = await api.subscriptions.changePrice(
      userContext,
      subscriptionId,
      body.new_price_cents
    );
    
    return NextResponse.json(result);
  } catch (error) {
    logger.error('Error changing subscription price:', error);
    return NextResponse.json(
      { error: 'Failed to change subscription price' },
      { status: 500 }
    );
  }
}