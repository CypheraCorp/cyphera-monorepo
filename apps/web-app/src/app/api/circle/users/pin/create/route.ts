import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { CircleRequestWithIdempotencyKeyAndToken } from '@/types/circle';
import logger from '@/lib/core/logger/logger';
/**
 * POST /api/circle/users/pin/create
 * Creates a PIN challenge for setting up a new PIN.
 * Calls the backend service which uses Public Headers + User-Token header.
 */
export async function POST(request: NextRequest) {
  try {
    // Get idempotency_key from the request body
    const { idempotency_key, user_token } = await request.json();
    if (!idempotency_key) {
      return NextResponse.json(
        { error: 'Idempotency key is required in the request body' },
        { status: 400 }
      );
    }

    const { api, userContext } = await getAPIContextFromSession(request);
    const workspaceId = userContext?.workspace_id;
    if (!workspaceId) {
      return NextResponse.json({ error: 'Workspace ID is required in context' }, { status: 400 });
    }

    try {
      // Call createPinChallenge service method
      const challengeResponse = await api.circle.createPinChallenge(workspaceId, {
        idempotency_key,
        user_token,
      } as CircleRequestWithIdempotencyKeyAndToken);
      return NextResponse.json(challengeResponse);
    } catch (error) {
      logger.error('Error creating PIN challenge via API service', { error });
      throw error;
    }
  } catch (error) {
    if (error instanceof Error && error.message.includes('session')) {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error processing PIN challenge request', { error });
    const message = error instanceof Error ? error.message : 'Failed to create PIN challenge';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
