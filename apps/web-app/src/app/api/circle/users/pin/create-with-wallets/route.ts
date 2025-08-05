import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { z } from 'zod';

// Schema for creating PIN with wallets
const createPinWithWalletsSchema = z.object({
  blockchains: z.array(z.string()).optional(),
  account_type: z.enum(['SCA', 'EOA']).default('SCA'),
});

/**
 * POST /api/circle/users/pin/create-with-wallets
 * Creates a PIN challenge with wallet creation in a single operation.
 * This is for users with UNSET PIN status.
 */
export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: createPinWithWalletsSchema },
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
          // Call createUserPinWithWallets service method with validated body
          const challengeResponse = await api.circle.createUserPinWithWallets(workspaceId, body);
          return NextResponse.json(challengeResponse);
        } catch (error) {
          logger.error('Error creating PIN with wallets via API service', { error });
          throw error;
        }
      } catch (error) {
        if (error instanceof Error && error.message.includes('session')) {
          return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
        }
        logger.error('Error processing PIN with wallets request', { error });
        const message = error instanceof Error ? error.message : 'Failed to create PIN with wallets';
        return NextResponse.json({ error: message }, { status: 500 });
      }
    }
  )
);