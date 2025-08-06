'use client';

import { usePrivy } from '@privy-io/react-auth';
import { useEffect, useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { AccountRequest, AccountType } from '@/types/account';
import { logger } from '@/lib/core/logger/logger-utils';
import { usePrivySmartAccount } from '@/hooks/privy/use-privy-smart-account';

interface PrivyLoginProps {
  redirectTo?: string | null;
  autoConnect?: boolean;
}

export function PrivyLogin({ redirectTo, autoConnect = false }: PrivyLoginProps) {
  const router = useRouter();
  const { ready, authenticated, user, login, logout } = usePrivy();
  const { smartAccountAddress, smartAccountReady } = usePrivySmartAccount();
  
  const [isProcessing, setIsProcessing] = useState(false);
  const [hasValidSession, setHasValidSession] = useState<boolean | null>(null);
  const [signinAttempted, setSigninAttempted] = useState(false);

  // Check for existing session on mount
  useEffect(() => {
    async function checkSession() {
      try {
        const response = await fetch('/api/auth/me', {
          credentials: 'include',
          headers: {
            'Cache-Control': 'no-cache',
          },
        });

        if (response.ok) {
          const data = await response.json();
          logger.log('✅ Found valid session:', data.user?.email);
          setHasValidSession(true);
        } else {
          logger.log('❌ No valid session found');
          setHasValidSession(false);
        }
      } catch (error) {
        logger.error('❌ Session check failed:', error);
        setHasValidSession(false);
      }
    }

    if (ready) {
      checkSession();
    }
  }, [ready]);

  // Handle signin with backend
  const handleSignin = useCallback(async () => {
    if (isProcessing || signinAttempted) return;

    try {
      setIsProcessing(true);

      if (!user || !smartAccountAddress) {
        logger.error('❌ Missing required data for signin:', {
          hasUser: !!user,
          hasSmartAccount: !!smartAccountAddress,
        });
        return;
      }

      logger.log('🔐 Signing in with Privy Smart Account:', {
        address: smartAccountAddress,
        email: user.email?.address,
      });

      // Prepare account data for backend
      const accountData: AccountRequest = {
        name: user.google?.name || user.email?.address?.split('@')[0] || 'Unknown User',
        account_type: AccountType.MERCHANT,
        business_name: user.google?.name || user.email?.address?.split('@')[0] || 'Unknown Business',
        website_url: undefined, // Remove invalid pictureUrl reference
        support_email: user.email?.address,
        wallet_data: {
          wallet_type: 'privy_smart_account',
          wallet_address: smartAccountAddress,
          network_type: 'evm',
          nickname: 'Privy Smart Account',
          is_primary: true,
          verified: true,
          metadata: {
            created_via: 'privy',
            user_email: user.email?.address,
            is_smart_account: true,
            privy_user_id: user.id,
          },
        },
        metadata: {
          privy_user_id: user.id,
          email: user.email?.address,
          created_at: user.createdAt,
          linked_accounts: user.linkedAccounts?.map(acc => acc.type),
        },
      };

      const response = await fetch('/api/auth/signin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(accountData),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      const signinResponse = await response.json();
      logger.log('✅ Signin successful:', {
        accountId: signinResponse?.account?.id,
        finishedOnboarding: signinResponse?.account?.finished_onboarding,
      });

      setSigninAttempted(true);
      setHasValidSession(true);

      // Redirect based on onboarding status
      if (signinResponse?.account?.finished_onboarding === true) {
        router.push(redirectTo || '/merchants/dashboard');
      } else {
        router.push('/merchants/onboarding');
      }
    } catch (error) {
      logger.error('❌ Signin failed:', error);
      setSigninAttempted(true);
      setHasValidSession(false);
    } finally {
      setIsProcessing(false);
    }
  }, [user, smartAccountAddress, router, redirectTo, isProcessing, signinAttempted]);

  // Auto-signin when authenticated and smart account is ready
  useEffect(() => {
    if (
      ready &&
      authenticated &&
      user &&
      smartAccountReady &&
      smartAccountAddress &&
      !isProcessing &&
      !signinAttempted &&
      hasValidSession === false
    ) {
      logger.log('🔄 Auto-signin triggered with Privy');
      handleSignin();
    }
  }, [
    ready,
    authenticated,
    user,
    smartAccountReady,
    smartAccountAddress,
    handleSignin,
    isProcessing,
    signinAttempted,
    hasValidSession,
  ]);

  // Redirect if already has valid session
  useEffect(() => {
    if (hasValidSession === true && !isProcessing) {
      router.push(redirectTo || '/merchants/dashboard');
    }
  }, [hasValidSession, isProcessing, router, redirectTo]);

  // Handle auto-connect
  useEffect(() => {
    if (ready && !authenticated && autoConnect && hasValidSession === false) {
      logger.log('🔄 Auto-connect triggered for Privy');
      login();
    }
  }, [ready, authenticated, autoConnect, hasValidSession, login]);

  // Loading state while checking session
  if (!ready || hasValidSession === null) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
          <p className="text-gray-600">Checking authentication...</p>
        </div>
      </div>
    );
  }

  // Processing state
  if (isProcessing) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
          <p className="text-gray-600">Setting up your Cyphera wallet...</p>
        </div>
      </div>
    );
  }

  // Waiting for smart account
  if (authenticated && !smartAccountReady) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          <p className="text-gray-600">Initializing smart account...</p>
        </div>
      </div>
    );
  }

  // Show login button if not authenticated
  if (!authenticated) {
    return (
      <div className="text-center py-12">
        <button
          onClick={login}
          className="px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg hover:shadow-xl"
        >
          Sign In with Privy
        </button>
      </div>
    );
  }

  // Show retry button if signin failed
  if (authenticated && signinAttempted && hasValidSession === false) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <p className="text-gray-600">Connected but authentication incomplete</p>
          <button
            onClick={() => {
              setSigninAttempted(false);
              setIsProcessing(false);
              handleSignin();
            }}
            className="px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg hover:shadow-xl"
          >
            Complete Sign In
          </button>
        </div>
      </div>
    );
  }

  // Return nothing if everything is processed
  return null;
}