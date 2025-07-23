import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { createCircleUserSchema } from '@/lib/validation/schemas/circle';

/**
 * POST /api/circle/users
 * Creates a new Circle user associated with the workspace.
 * Calls the backend service which uses an Admin API Key.
 */
export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: createCircleUserSchema },
    async (request, { body }) => {
      try {
        // Get the API context from session
        const { api, userContext } = await getAPIContextFromSession(request);

        const workspaceId = userContext?.workspace_id;
        if (!workspaceId) {
          return NextResponse.json({ error: 'Workspace ID is required in context' }, { status: 400 });
        }

        if (!body) {
          return NextResponse.json({ error: 'Request body is required' }, { status: 400 });
        }

        // Call createUser service method with validated body
        const userResponse = await api.circle.createUser(workspaceId, body);

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
  )
);
