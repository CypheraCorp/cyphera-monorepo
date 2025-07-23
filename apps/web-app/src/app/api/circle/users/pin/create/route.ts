import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { CircleRequestWithIdempotencyKeyAndToken } from '@/types/circle';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { createCirclePinSchema } from '@/lib/validation/schemas/circle';
/**
 * POST /api/circle/users/pin/create
 * Creates a PIN challenge for setting up a new PIN.
 * Calls the backend service which uses Public Headers + User-Token header.
 */
export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: createCirclePinSchema },
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

        try {
          // Call createPinChallenge service method with validated body
          const challengeResponse = await api.circle.createPinChallenge(workspaceId, body);
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
  )
);
