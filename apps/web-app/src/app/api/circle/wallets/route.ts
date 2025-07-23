import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { CreateWalletsRequest } from '@/types/circle';
import logger from '@/lib/core/logger/logger';

/**
 * POST /api/circle/wallets
 * Creates Circle wallets.
 * Calls the backend service which uses Public Headers + User-Token header.
 */
export async function POST(request: NextRequest) {
  try {
    const body = (await request.json()) as CreateWalletsRequest; // Expect body without user_token

    if (!body.idempotency_key || !body.blockchains || body.blockchains.length === 0) {
      return NextResponse.json(
        { error: 'Invalid request. Required fields: idempotency_key, blockchains' },
        { status: 400 }
      );
    }

    const { api, userContext } = await getAPIContextFromSession(request);
    const workspaceId = userContext?.workspace_id;
    if (!workspaceId) {
      return NextResponse.json({ error: 'Workspace ID is required in context' }, { status: 400 });
    }

    let createResponse;
    try {
      createResponse = await api.circle.createWallets(workspaceId, body as CreateWalletsRequest);
    } catch (serviceError) {
      // Re-throw the error to be caught by the outer try/catch which returns 500
      throw serviceError;
    }

    return NextResponse.json(createResponse);
  } catch (error) {
    if (error instanceof Error && error.message.includes('session')) {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error creating Circle wallets', { error });
    const message = error instanceof Error ? error.message : 'Failed to create Circle wallets';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}

/**
 * GET /api/circle/wallets/{workspaceId}
 * Lists Circle wallets.
 * Calls the backend service which uses Public Headers + User-Token header.
 */
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ workspaceId: string }> }
) {
  try {
    const { workspaceId } = await params;
    if (!workspaceId) {
      return NextResponse.json({ error: 'Workspace ID is required in path' }, { status: 400 });
    }

    const { api, userContext } = await getAPIContextFromSession(request);

    const { searchParams } = new URL(request.url);
    const listParams = {
      address: searchParams.get('address') || undefined,
      blockchain: searchParams.get('blockchain') || undefined,
      pageSize: searchParams.get('page_size')
        ? parseInt(searchParams.get('page_size')!, 10)
        : undefined,
      pageBefore: searchParams.get('page_before') || undefined,
      pageAfter: searchParams.get('page_after') || undefined,
    };

    const contextForCall = { ...userContext, workspace_id: workspaceId };

    // Call listCircleWallets service method
    const listResponse = await api.circle.listCircleWallets(contextForCall, listParams);

    return NextResponse.json(listResponse);
  } catch (error) {
    if (error instanceof Error && error.message.includes('session')) {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error listing Circle wallets', { error });
    const message = error instanceof Error ? error.message : 'Failed to list Circle wallets';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
