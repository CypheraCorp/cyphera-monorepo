/**
 * Development Authentication Check API
 * 
 * This endpoint allows automated tools to check if development authentication
 * bypass is available and optionally create development sessions.
 * 
 * ⚠️ DEVELOPMENT ONLY - This should never be enabled in production!
 */

import { NextRequest, NextResponse } from 'next/server';
import { 
  isDevAuthBypassAvailable, 
  createDevSessionToken, 
  DEV_TEST_USERS,
  logDevBypassWarning 
} from '@/lib/auth/dev-auth-bypass';
import { PublicAPI } from '@/services/cyphera-api/public';
import { UnifiedSessionService } from '@/lib/auth/session/unified-session';
import type { AccountRequest } from '@/types/account';
import { AccountType } from '@/types/account';
import type { CustomerSignInRequest } from '@/types/customer';
import { logger } from '@/lib/core/logger/logger';

export async function POST(request: NextRequest) {
  // Log warning about development bypass usage
  logDevBypassWarning('POST /api/debug/dev-auth-check');

  // Check if development bypass is available
  if (!isDevAuthBypassAvailable()) {
    return NextResponse.json(
      { 
        available: false, 
        message: 'Development auth bypass is not enabled' 
      },
      { status: 403 }
    );
  }

  try {
    const body = await request.json();
    const { userType, action } = body;

    if (!userType || !['merchant', 'customer'].includes(userType)) {
      return NextResponse.json(
        { error: 'Invalid user type. Must be "merchant" or "customer"' },
        { status: 400 }
      );
    }

    // Just check availability
    if (!action || action === 'check') {
      return NextResponse.json({
        available: true,
        userType,
        testUser: DEV_TEST_USERS[userType as keyof typeof DEV_TEST_USERS],
      });
    }

    // Create session token and ensure backend user exists
    if (action === 'create-session') {
      const testUser = DEV_TEST_USERS[userType as keyof typeof DEV_TEST_USERS];
      const token = createDevSessionToken(userType as 'merchant' | 'customer');
      
      try {
        // For merchants, ensure user exists in backend
        if (userType === 'merchant') {
          // Prepare account request data matching the expected format
          const accountRequest: AccountRequest = {
            name: testUser.name,
            account_type: AccountType.MERCHANT,
            business_name: 'Dev Test Business',
            support_email: testUser.email,
            wallet_data: {
              wallet_type: 'web3auth',
              wallet_address: testUser.smartAccountAddress,
              nickname: 'Dev Test Wallet',
              network_type: 'evm',
              is_primary: true,
              verified: true,
            },
            metadata: {
              ownerWeb3AuthId: testUser.id,
              verifier: 'google-oauth2', // Mock verifier for dev
              verifierId: testUser.id,
              email: testUser.email,
              name: testUser.name,
              profileImage: 'https://example.com/avatar.png',
              raw_userInfo: {
                email: testUser.email,
                name: testUser.name,
                verifierId: testUser.id,
                verifier: 'google-oauth2',
                idToken: token, // Use dev token
              },
            },
          };

          // Create PublicAPI instance and call backend
          const publicAPI = new PublicAPI();
          logger.info('Dev auth bypass: Creating/retrieving user in backend', {
            email: testUser.email,
            userId: testUser.id,
          });

          const backendResponse = await publicAPI.signInOrRegister(accountRequest);
          
          // Create proper session with backend data
          const workspaceId = backendResponse.account?.workspaces?.[0]?.id;
          
          await UnifiedSessionService.create({
            user_type: 'merchant',
            access_token: token,
            account_id: backendResponse.account?.id || testUser.id,
            user_id: backendResponse.user?.id || testUser.id,
            workspace_id: workspaceId,
            email: testUser.email,
          });

          logger.info('Dev auth bypass: Backend user created/retrieved successfully', {
            accountId: backendResponse.account?.id,
            userId: backendResponse.user?.id,
            workspaceId,
            finishedOnboarding: backendResponse.account?.finished_onboarding,
          });

          return NextResponse.json({
            available: true,
            token,
            user: testUser,
            backendUser: {
              accountId: backendResponse.account?.id,
              userId: backendResponse.user?.id,
              workspaceId,
              finishedOnboarding: backendResponse.account?.finished_onboarding,
            },
            message: 'Development session created with backend user',
          });
        }

        // For customers, ensure user exists in backend
        if (userType === 'customer') {
          // Prepare customer signin request data
          const customerRequest: CustomerSignInRequest = {
            email: testUser.email,
            name: testUser.name,
            metadata: {
              web3auth_id: testUser.id,
              verifier: 'google-oauth2',
              verifier_id: testUser.id,
              ownerWeb3AuthId: testUser.id,
              email: testUser.email,
              name: testUser.name,
              profileImage: 'https://example.com/customer-avatar.png',
              raw_userInfo: {
                email: testUser.email,
                name: testUser.name,
                verifierId: testUser.id,
                verifier: 'google-oauth2',
                idToken: token,
              },
            },
            wallet_data: {
              wallet_address: testUser.smartAccountAddress,
              network_type: 'evm',
              nickname: 'Dev Test Customer Wallet',
              is_primary: true,
              verified: true,
              metadata: {
                wallet_type: 'web3auth',
                created_via: 'dev_auth_bypass',
              },
            },
          };

          // Create PublicAPI instance and call backend
          const publicAPI = new PublicAPI();
          logger.info('Dev auth bypass: Creating/retrieving customer in backend', {
            email: testUser.email,
            customerId: testUser.id,
          });

          const backendResponse = await publicAPI.customerSignInOrRegister(customerRequest);
          
          // Extract customer data from response
          const customer = backendResponse.data?.customer;
          const wallet = backendResponse.data?.wallet;
          
          await UnifiedSessionService.create({
            user_type: 'customer',
            access_token: token,
            customer_id: customer?.id || testUser.id,
            customer_email: testUser.email,
            customer_name: testUser.name,
            wallet_address: wallet?.wallet_address || testUser.smartAccountAddress,
            wallet_id: wallet?.id,
            finished_onboarding: customer?.finished_onboarding || false,
          });

          logger.info('Dev auth bypass: Backend customer created/retrieved successfully', {
            customerId: customer?.id,
            email: customer?.email,
            walletId: wallet?.id,
            finishedOnboarding: customer?.finished_onboarding,
          });

          return NextResponse.json({
            available: true,
            token,
            user: testUser,
            backendUser: {
              customerId: customer?.id,
              email: customer?.email,
              walletId: wallet?.id,
              finishedOnboarding: customer?.finished_onboarding,
            },
            message: 'Development session created with backend customer',
          });
        }

        // This should never be reached due to earlier validation
        throw new Error('Invalid user type');

      } catch (backendError) {
        logger.error('Dev auth bypass: Failed to create backend user', {
          error: backendError instanceof Error ? backendError.message : backendError,
          userType,
          email: testUser.email,
        });

        // Still return success but warn about backend
        return NextResponse.json({
          available: true,
          token,
          user: testUser,
          warning: 'Session created but backend user creation failed',
          backendError: backendError instanceof Error ? backendError.message : 'Unknown error',
          message: 'Development session created (local only)',
        });
      }
    }

    return NextResponse.json(
      { error: 'Invalid action' },
      { status: 400 }
    );

  } catch (error) {
    console.error('Development auth check error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function GET(_request: NextRequest) {
  // Log warning about development bypass usage
  logDevBypassWarning('GET /api/debug/dev-auth-check');

  // Simple availability check
  if (!isDevAuthBypassAvailable()) {
    return NextResponse.json(
      { 
        available: false, 
        message: 'Development auth bypass is not enabled' 
      },
      { status: 403 }
    );
  }

  return NextResponse.json({
    available: true,
    message: 'Development auth bypass is available',
    testUsers: {
      merchant: DEV_TEST_USERS.merchant.email,
      customer: DEV_TEST_USERS.customer.email,
    },
  });
}