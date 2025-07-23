import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { CreateWalletsRequest } from '@/types/circle';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { createCircleWalletsSchema, circleWalletQuerySchema } from '@/lib/validation/schemas/circle';

/**
 * POST /api/circle/wallets
 * Creates Circle wallets.
 * Calls the backend service which uses Public Headers + User-Token header.
 */
export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: createCircleWalletsSchema },
    async (request, { body }) => {
      try {
        const { api, userContext } = await getAPIContextFromSession(request);
        const workspaceId = userContext?.workspace_id;
        if (!workspaceId) {
          return NextResponse.json({ error: 'Workspace ID is required in context' }, { status: 400 });
        }

        if (!body) {
          return NextResponse.json({ error: 'Request body is required' }, { status: 400 });
        }

        let createResponse;
        try {
          createResponse = await api.circle.createWallets(workspaceId, body);
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
  )
);

/**
 * GET /api/circle/wallets
 * Lists Circle wallets.
 * Note: This appears to be a standard route without dynamic segments
 */
export async function GET(request: NextRequest) {
  try {
    const { api, userContext } = await getAPIContextFromSession(request);
    
    // Get workspace ID from user context
    const workspaceId = userContext?.workspace_id;
    if (!workspaceId) {
      return NextResponse.json({ error: 'Workspace ID is required in context' }, { status: 400 });
    }

    const { searchParams } = new URL(request.url);
    const listParams = {
      blockchain: searchParams.get('blockchain') || undefined,
      state: searchParams.get('state') || undefined,
      pageSize: searchParams.get('pageSize') ? parseInt(searchParams.get('pageSize')!, 10) : undefined,
      pageBefore: searchParams.get('pageBefore') || undefined,
      pageAfter: searchParams.get('pageAfter') || undefined,
    };

    // Call listCircleWallets service method
    const listResponse = await api.circle.listCircleWallets(userContext, listParams);

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
