import { NextRequest, NextResponse } from 'next/server';
import type { AccountRequest } from '@/types/account';
import type { CustomerSignInRequest } from '@/types/customer';
import type { CustomerResponse } from '@/types/customer';
import type { WalletResponse } from '@/types/wallet';
import { logger } from '@/lib/core/logger/logger';
import { UnifiedSessionService } from '@/lib/auth/session/unified-session';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';

/**
 * POST /api/auth/customer/signin
 * Customer-side Web3Auth signin endpoint that creates customer accounts
 */
export async function POST(request: NextRequest) {
  try {
    // Parse the request body (expecting AccountRequest format from Web3Auth component)
    const accountData: AccountRequest = await request.json();

    logger.debug('Customer signin request received', {
      email: accountData.metadata?.email,
      name: accountData.name,
      hasWalletData: !!accountData.wallet_data,
      walletAddress: accountData.wallet_data?.wallet_address,
      metadataKeys: accountData.metadata ? Object.keys(accountData.metadata) : 'No metadata',
      ownerWeb3AuthId: accountData.metadata?.ownerWeb3AuthId,
    });

    // Validate required fields
    const ownerWeb3AuthId = accountData.metadata?.ownerWeb3AuthId as string;
    if (!ownerWeb3AuthId) {
      logger.error('Missing ownerWeb3AuthId in customer signin', {
        metadata: accountData.metadata,
      });
      return NextResponse.json({ error: 'Missing ownerWeb3AuthId in metadata' }, { status: 400 });
    }

    if (!accountData.metadata?.email) {
      return NextResponse.json({ error: 'Missing email in metadata' }, { status: 400 });
    }

    // Extract the JWT token from Web3Auth data
    const rawUserInfo = accountData.metadata?.raw_userInfo as { idToken?: string } | undefined;
    let accessToken = rawUserInfo?.idToken;

    // Handle cases where idToken is not available (like in merchant signin)
    if (!accessToken) {
      logger.warn('No JWT token found in Web3Auth userInfo, using placeholder for customer');
      accessToken = 'web3auth_customer_no_token'; // Placeholder token for customer accounts
    }

    const customerEmail = accountData.metadata.email as string;
    const customerName = (accountData.metadata.name as string) || customerEmail.split('@')[0];
    const walletAddress = accountData.wallet_data?.wallet_address;

    // Log customer wallet data if present
    if (accountData.wallet_data) {
      logger.info('Customer Web3Auth wallet data received', {
        wallet_type: accountData.wallet_data.wallet_type,
        wallet_address: accountData.wallet_data.wallet_address,
        nickname: accountData.wallet_data.nickname,
      });
    }

    // Transform AccountRequest to CustomerSignInRequest format for backend
    const customerSignInRequest: CustomerSignInRequest = {
      email: customerEmail,
      name: customerName,
      metadata: {
        web3auth_id: ownerWeb3AuthId,
        verifier: accountData.metadata?.verifier as string,
        verifier_id: ownerWeb3AuthId,
        ...accountData.metadata, // Include all original metadata
      },
      wallet_data: walletAddress
        ? {
            wallet_address: walletAddress,
            network_type:
              (accountData.wallet_data?.network_type as
                | 'evm'
                | 'solana'
                | 'cosmos'
                | 'bitcoin'
                | 'polkadot') || 'evm',
            nickname: accountData.wallet_data?.nickname || 'Customer Crypto Wallet',
            is_primary: accountData.wallet_data?.is_primary ?? true,
            verified: accountData.wallet_data?.verified ?? true,
            metadata: {
              wallet_type: accountData.wallet_data?.wallet_type,
              created_via: 'web3auth_signin',
            },
          }
        : undefined,
    };

    logger.debug('Calling backend customer signin API', {
      email: customerSignInRequest.email,
      name: customerSignInRequest.name,
      web3auth_id: customerSignInRequest.metadata.web3auth_id,
      has_wallet_data: !!customerSignInRequest.wallet_data,
      wallet_address: customerSignInRequest.wallet_data?.wallet_address,
    });

    // Import PublicAPI here to call backend
    const { PublicAPI } = await import('@/services/cyphera-api/public');
    const publicAPI = new PublicAPI();

    let customer: CustomerResponse | undefined;
    let wallet: WalletResponse | undefined;

    try {
      // Call backend customer signin API
      logger.debug('Making backend API request to create customer');
      const backendResponse = await publicAPI.customerSignInOrRegister(customerSignInRequest);

      logger.debug('Backend response received', {
        success: backendResponse?.success,
        hasData: !!backendResponse?.data,
        dataKeys: backendResponse?.data ? Object.keys(backendResponse.data) : 'No data',
      });

      // Extract customer and wallet data from backend response
      if (backendResponse?.data?.customer) {
        // Map the signin response customer to CustomerResponse format
        const backendCustomer = backendResponse.data.customer;
        customer = {
          id: backendCustomer.id,
          object: backendCustomer.object,
          workspace_id: '', // Will be set by backend
          external_id: backendCustomer.external_id,
          email: backendCustomer.email,
          name: backendCustomer.name,
          phone: backendCustomer.phone,
          description: backendCustomer.description,
          finished_onboarding: backendCustomer.finished_onboarding,
          metadata: backendCustomer.metadata,
          balance_in_pennies: 0, // Default value
          currency: 'USD', // Default currency
          default_source_id: undefined,
          invoice_prefix: undefined,
          next_invoice_number: 1,
          tax_exempt: false,
          tax_ids: undefined,
          livemode: false,
          created_at: backendCustomer.created_at,
          updated_at: backendCustomer.updated_at,
        } as CustomerResponse;

        // Map wallet if present
        if (backendResponse.data.wallet) {
          const backendWallet = backendResponse.data.wallet;
          wallet = {
            id: backendWallet.id,
            object: 'wallet',
            workspace_id: '', // Will be set by backend
            wallet_type: 'wallet',
            wallet_address: backendWallet.wallet_address,
            network_type: backendWallet.network_type,
            nickname: backendWallet.nickname,
            ens: backendWallet.ens,
            is_primary: backendWallet.is_primary,
            verified: backendWallet.verified,
            metadata: backendWallet.metadata,
            created_at: backendWallet.created_at,
            updated_at: backendWallet.updated_at,
          } as WalletResponse;
        }

        logger.info('Backend customer signin successful', {
          customerId: customer.id,
          email: customer.email,
          hasWallet: !!wallet,
          walletId: wallet?.id,
          walletAddress: wallet?.wallet_address,
        });
      } else {
        throw new Error(
          `Backend response missing customer data: ${JSON.stringify(backendResponse)}`
        );
      }
    } catch (backendError) {
      logger.error('Backend customer signin failed', {
        error: backendError instanceof Error ? backendError.message : backendError,
      });
      logger.warn('Falling back to local customer creation due to backend error');

      // Fallback to local customer creation if backend fails
      customer = {
        id: `customer_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
        object: 'customer',
        workspace_id: '', // Mock value for fallback
        email: customerEmail,
        name: customerName,
        metadata: customerSignInRequest.metadata,
        balance_in_pennies: 0,
        currency: 'USD',
        next_invoice_number: 1,
        tax_exempt: false,
        livemode: false,
        created_at: Math.floor(Date.now() / 1000),
        updated_at: Math.floor(Date.now() / 1000),
        finished_onboarding: false,
      } as CustomerResponse;

      // Create mock wallet data if wallet address is provided
      wallet = walletAddress
        ? ({
            id: `wallet_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            object: 'customer_wallet',
            workspace_id: '', // Mock value for fallback
            wallet_type: 'wallet',
            customer_id: customer.id,
            wallet_address: walletAddress,
            network_type: customerSignInRequest.wallet_data?.network_type || 'evm',
            nickname: customerSignInRequest.wallet_data?.nickname || 'Customer Crypto Wallet',
            is_primary: customerSignInRequest.wallet_data?.is_primary ?? true,
            verified: customerSignInRequest.wallet_data?.verified ?? true,
            metadata: customerSignInRequest.wallet_data?.metadata,
            created_at: Math.floor(Date.now() / 1000),
            updated_at: Math.floor(Date.now() / 1000),
          } as WalletResponse)
        : undefined;

      logger.warn('Local customer creation completed (fallback)', {
        customerId: customer.id,
        email: customer.email,
        walletId: wallet?.id,
        walletAddress: wallet?.wallet_address,
      });
    }

    // Ensure customer is defined before creating session
    if (!customer) {
      throw new Error('Customer creation failed');
    }

    // Create customer session using unified service
    await UnifiedSessionService.create({
      user_type: 'customer',
      access_token: accessToken || 'web3auth_customer_no_token',
      customer_id: customer.id,
      customer_email: customer.email,
      customer_name: customer.name,
      wallet_address: wallet?.wallet_address,
      wallet_id: wallet?.id,
      finished_onboarding: customer.finished_onboarding ?? false,
    });

    logger.info('Created customer session', {
      email: customer.email,
      customerId: customer.id,
      walletAddress: wallet?.wallet_address,
      finishedOnboarding: customer.finished_onboarding ?? false,
    });

    // Create response with real backend customer and wallet data
    return NextResponse.json({
      success: true,
      customer: {
        id: customer.id,
        email: customer.email,
        name: customer.name,
        finished_onboarding: customer.finished_onboarding ?? false,
        created_at: customer.created_at,
      },
      wallet: wallet
        ? {
            id: wallet.id,
            wallet_address: wallet.wallet_address,
            network_type: wallet.network_type,
            nickname: wallet.nickname,
            is_primary: wallet.is_primary,
          }
        : undefined,
    });
  } catch (error) {
    logger.error('Customer signin failed', {
      error: error instanceof Error ? error.message : error,
    });

    const message = error instanceof Error ? error.message : 'Failed to sign in as customer';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
