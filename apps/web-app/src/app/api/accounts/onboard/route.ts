import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { AccountsAPI } from '@/services/cyphera-api/accounts';
import type { AccountOnboardingRequest } from '@/types/account';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { accountOnboardingSchema } from '@/lib/validation/schemas/account';

/**
 * POST /api/accounts/onboard
 * Onboard a user account with profile details
 */
export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: accountOnboardingSchema },
    async (request, { body }) => {
      try {
        // Get API context from session
        const { userContext } = await getAPIContextFromSession(request);

        // Use AccountsAPI from context
        const accountsAPI = new AccountsAPI();

        // Call onboardAccount with validated body
        if (!body) {
          return NextResponse.json({ error: 'Request body is required' }, { status: 400 });
        }
        const result = await accountsAPI.onboardAccount(userContext, body);

        return NextResponse.json(result);
      } catch (error) {
        logger.error('Onboarding API error', { error });
        const message = error instanceof Error ? error.message : 'Failed to onboard account';
        return NextResponse.json({ error: message }, { status: 500 });
      }
    }
  )
);
