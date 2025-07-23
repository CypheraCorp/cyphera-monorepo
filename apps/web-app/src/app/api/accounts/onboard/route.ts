import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { AccountsAPI } from '@/services/cyphera-api/accounts';
import type { AccountOnboardingRequest } from '@/types/account';
import logger from '@/lib/core/logger/logger';

/**
 * POST /api/accounts/onboard
 * Onboard a user account with profile details
 */
export async function POST(request: NextRequest) {
  try {
    // Get API context from session
    const { userContext } = await getAPIContextFromSession(request);

    // Parse request body
    const accountData: Partial<AccountOnboardingRequest> = await request.json();

    // Use AccountsAPI from context
    const accountsAPI = new AccountsAPI();

    // Call onboardAccount
    const result = await accountsAPI.onboardAccount(userContext, accountData);

    return NextResponse.json(result);
  } catch (error) {
    logger.error('Onboarding API error', { error });
    const message = error instanceof Error ? error.message : 'Failed to onboard account';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
