import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import logger from '@/lib/core/logger/logger';

/**
 * POST /api/circle/users
 * Creates a new Circle user associated with the workspace.
 * Calls the backend service which uses an Admin API Key.
 */
export async function POST(request: NextRequest) {
  try {
    const { external_user_id } = await request.json();
    if (!external_user_id) {
      return NextResponse.json(
        { error: 'External user ID is required in the request body' },
        { status: 400 }
      );
    }

    // Get the API context from session
    const { api, userContext } = await getAPIContextFromSession(request);

    const workspaceId = userContext?.workspace_id;
    if (!workspaceId) {
      return NextResponse.json({ error: 'Workspace ID is required in context' }, { status: 400 });
    }

    // Call createUser service method, passing both workspaceId and external_user_id
    const userResponse = await api.circle.createUser(workspaceId, {
      external_user_id: external_user_id,
    });

    return NextResponse.json(userResponse);
  } catch (error) {
    if (error instanceof Error && error.message.includes('session')) {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error creating Circle user', { error });
    const message = error instanceof Error ? error.message : 'Failed to create Circle user';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
