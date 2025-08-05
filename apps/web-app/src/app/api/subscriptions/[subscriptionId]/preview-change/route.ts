import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import { logger } from '@/lib/core/logger/logger';
import type { PreviewChangeRequest } from '@/types/subscription';

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
    const body = await request.json() as PreviewChangeRequest;
    
    const preview = await api.subscriptions.previewChange(
      userContext,
      subscriptionId,
      body
    );
    
    return NextResponse.json(preview);
  } catch (error) {
    logger.error('Error previewing subscription change:', error);
    return NextResponse.json(
      { error: 'Failed to preview subscription change' },
      { status: 500 }
    );
  }
}