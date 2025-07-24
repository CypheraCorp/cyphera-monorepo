import { NextRequest, NextResponse } from 'next/server';
import { PublicAPI } from '@/services/cyphera-api/public';
import type { AccountRequest, AccountAccessResponse } from '@/types/account';
import { logger } from '@/lib/core/logger/logger';
import { UnifiedSessionService } from '@/lib/auth/session/unified-session';

/**
 * POST /api/auth/signin
 * Server-side Web3Auth signin endpoint that can access the CYPHERA_API_KEY
 */
export async function POST(request: NextRequest) {
  try {
    // Parse the request body
    const accountData: AccountRequest = await request.json();

    // Validate required fields
    if (!accountData.metadata?.ownerWeb3AuthId) {
      return NextResponse.json({ error: 'Missing ownerWeb3AuthId in metadata' }, { status: 400 });
    }

    if (!accountData.metadata?.verifier) {
      return NextResponse.json({ error: 'Missing verifier in metadata' }, { status: 400 });
    }

    // Extract the JWT token from Web3Auth data
    const rawUserInfo = accountData.metadata?.raw_userInfo as Record<string, unknown>;
    let accessToken = rawUserInfo?.idToken as string;

    // Handle the case where frontend sends placeholder token or no token
    if (!accessToken || accessToken === 'web3auth_no_token' || accessToken === '') {
      // Log what we received for debugging
      logger.info('No JWT token from Web3Auth', {
        accountId: accountData.metadata?.ownerWeb3AuthId,
        hasRawUserInfo: !!rawUserInfo,
        rawUserInfoKeys: rawUserInfo ? Object.keys(rawUserInfo) : [],
      });
      
      // For now, we'll use a placeholder that the backend can recognize
      // The backend should validate using Web3Auth metadata instead of JWT
      accessToken = 'no_jwt_token_available';
    }

    // Log wallet data if present
    if (accountData.wallet_data) {
      logger.info('Web3Auth wallet data received', {
        wallet_type: accountData.wallet_data.wallet_type,
        wallet_address: accountData.wallet_data.wallet_address,
        nickname: accountData.wallet_data.nickname,
      });
    }

    // Create PublicAPI instance (this will have access to server-side env vars)
    const publicAPI = new PublicAPI();

    // Call the backend signin API with wallet data
    const response: AccountAccessResponse = await publicAPI.signInOrRegister(accountData);

    // Check if this is a new user (first time signup)
    const isNewUser = !response.account?.finished_onboarding;
    const workspaceId = response.account?.workspaces?.[0]?.id;
    const userEmail = (accountData.metadata?.email as string) || '';

    // Create merchant session using unified service
    await UnifiedSessionService.create({
      user_type: 'merchant',
      access_token: accessToken,
      account_id: response.account?.id,
      user_id: response.user?.id,
      workspace_id: workspaceId,
      email: userEmail,
    });

    logger.info('Created merchant session', {
      email: userEmail,
      accountId: response.account?.id,
      userId: response.user?.id,
      workspaceId: workspaceId,
      backendResponse: {
        hasAccount: !!response.account,
        hasUser: !!response.user,
        finishedOnboarding: response.account?.finished_onboarding,
      },
    });

    // Log wallet creation status for new users
    if (isNewUser && accountData.wallet_data) {
      logger.info('Web3Auth wallet data sent to backend for processing during signin', {
        accountId: response.account?.id,
      });
    }

    return NextResponse.json(response);
  } catch (error) {
    logger.error('Server-side signin failed', {
      error: error instanceof Error ? error.message : error,
    });

    const message = error instanceof Error ? error.message : 'Failed to sign in';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
