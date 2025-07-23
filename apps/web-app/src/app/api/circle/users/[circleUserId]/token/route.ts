import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import logger from '@/lib/core/logger/logger';

/**
 * POST /api/circle/users/{circleUserId}/token
 * Creates a Circle user token.
 * Calls the backend service which uses an Admin API Key.
 */
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ circleUserId: string }> }
) {
  try {
    const { circleUserId } = await params;
    if (!circleUserId) {
      return NextResponse.json({ error: 'User ID is required in path' }, { status: 400 });
    }

    const { api, userContext } = await getAPIContextFromSession(request);
    const workspaceId = userContext?.workspace_id;
    if (!workspaceId) {
      return NextResponse.json({ error: 'Workspace ID is required in context' }, { status: 400 });
    }

    // Call createUserToken service method
    const tokenResponse = await api.circle.createUserToken(workspaceId, {
      external_user_id: circleUserId,
    });

    return NextResponse.json(tokenResponse);
  } catch (error) {
    if (error instanceof Error && error.message.includes('session')) {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error creating Circle user token', { error });
    const message = error instanceof Error ? error.message : 'Failed to create Circle user token';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
