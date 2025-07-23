'use client';

import { useWeb3AuthConnect, useWeb3AuthUser, useWeb3Auth } from '@web3auth/modal/react';
import { useAccount } from 'wagmi';
import { useEffect, useState, useCallback, useRef } from 'react';
import { AccountRequest, AccountType } from '@/types/account';
import { useRouter } from 'next/navigation';
import { logger } from '@/lib/core/logger/logger-utils';

interface Web3AuthLoginProps {
  redirectTo?: string | null;
  autoConnect?: boolean;
}

// Global flag to prevent auto-connect immediately after logout (resets on page reload)
let justLoggedOut = false;

// Client-side only component that uses Web3Auth hooks
function Web3AuthLoginClient({ redirectTo, autoConnect = false }: Web3AuthLoginProps) {
  const router = useRouter();
  const [shouldShowModal, setShouldShowModal] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [web3AuthAddress, setWeb3AuthAddress] = useState<string | null>(null);
  const [hasValidSession, setHasValidSession] = useState<boolean | null>(null);
  const [signinAttempted, setSigninAttempted] = useState(false);
  const [sessionCheckAttempted, setSessionCheckAttempted] = useState(false);

  // Ref to prevent multiple simultaneous signin attempts
  const signinInProgress = useRef(false);

  // Always call hooks in the same order - safe to call here since this component only renders client-side
  const { connect, isConnected } = useWeb3AuthConnect();
  const { userInfo } = useWeb3AuthUser();
  const { web3Auth } = useWeb3Auth(); // Get the Web3Auth instance
  const { address: wagmiAddress } = useAccount(); // Get wallet address from wagmi (might be null)

  // Check for existing session on mount
  useEffect(() => {
    // Prevent multiple session checks
    if (sessionCheckAttempted) return;
    setSessionCheckAttempted(true);

    async function checkSession() {
      try {
        // Add a small delay to ensure auth context is ready
        await new Promise((resolve) => setTimeout(resolve, 200));

        const response = await fetch('/api/auth/me', {
          credentials: 'include',
          headers: {
            'Cache-Control': 'no-cache', // Don't cache auth checks
          },
        });

        if (response.ok) {
          const data = await response.json();
          logger.log('✅ Found valid session:', data.user?.email);
          setHasValidSession(true);
        } else {
          logger.log('❌ No valid session found, status:', response.status);
          setHasValidSession(false);
        }
      } catch (error) {
        logger.error('❌ Session check failed:', error);
        setHasValidSession(false);
      }
    }

    // Small delay to prevent race condition with Web3Auth initialization
    const timeoutId = setTimeout(checkSession, 300);
    return () => clearTimeout(timeoutId);
  }, [sessionCheckAttempted]);

  // Auto-fallback mechanism - if authentication takes too long, show manual options
  useEffect(() => {
    if (hasValidSession === null && !isProcessing && !signinAttempted) {
      // After 10 seconds of being stuck, show manual login option
      const fallbackTimeoutId = setTimeout(() => {
        logger.log('⏰ Authentication timeout - showing manual login option');
        setShouldShowModal(true);
        setHasValidSession(false);
      }, 10000); // 10 seconds

      return () => clearTimeout(fallbackTimeoutId);
    }
  }, [hasValidSession, isProcessing, signinAttempted]);

  // Get wallet address directly from Web3Auth provider when connected
  useEffect(() => {
    async function getWeb3AuthAddress() {
      if (isConnected && web3Auth?.provider) {
        try {
          const accounts = (await web3Auth.provider.request({
            method: 'eth_accounts',
          })) as string[];

          if (accounts && Array.isArray(accounts) && accounts.length > 0) {
            const address = accounts[0];
            setWeb3AuthAddress(address);
            logger.log('🔑 Web3Auth address retrieved:', address);
          }
        } catch (error) {
          logger.error('❌ Failed to get address from Web3Auth provider:', error);
        }
      } else if (!isConnected) {
        setWeb3AuthAddress(null);
      }
    }

    getWeb3AuthAddress();
  }, [isConnected, web3Auth?.provider]);

  // Manual session validation function
  const validateSession = useCallback(async () => {
    try {
      logger.log('🔍 Manual session validation triggered');
      const response = await fetch('/api/auth/me', {
        credentials: 'include',
        headers: {
          'Cache-Control': 'no-cache',
        },
      });

      logger.log('📡 Session validation response:', response.status);

      if (response.ok) {
        const data = await response.json();
        logger.log('✅ Valid session found:', data);
        setHasValidSession(true);
        return true;
      } else {
        logger.log('❌ No valid session, status:', response.status);
        setHasValidSession(false);
        return false;
      }
    } catch (error) {
      logger.error('❌ Session validation error:', error);
      setHasValidSession(false);
      return false;
    }
  }, []);

  // Use Web3Auth address as primary, fallback to Wagmi address
  const walletAddress = web3AuthAddress || wagmiAddress;

  // Auto-connect logic with more robust connection handling
  useEffect(() => {
    if (autoConnect && !isConnected && !justLoggedOut && !isProcessing) {
      logger.log('🔄 Attempting Web3Auth auto-connect...', {
        autoConnect,
        isConnected,
        justLoggedOut,
        isProcessing,
        hasWeb3Auth: !!web3Auth,
      });

      // Try to connect and handle both success and failure
      connect()
        .then(() => {
          logger.log('✅ Web3Auth auto-connect successful');
        })
        .catch((error) => {
          logger.error('❌ Auto-connect failed:', error);
          // If auto-connect fails, show manual login option after a brief delay
          setTimeout(() => {
            logger.log('🔧 Showing manual login option after auto-connect failure');
            setShouldShowModal(true);
          }, 2000); // Give 2 seconds before showing manual option
        });
    } else if (autoConnect && !isConnected && justLoggedOut) {
      // If we just logged out, show a manual login option instead
      logger.log('🔄 Just logged out, showing manual login option');
      setShouldShowModal(true);
    } else if (autoConnect && isConnected) {
      logger.log('✅ Web3Auth already connected');
    } else {
      logger.log('🔍 Auto-connect not triggered:', {
        autoConnect,
        isConnected,
        justLoggedOut,
        isProcessing,
      });
    }
  }, [autoConnect, isConnected, connect, isProcessing, web3Auth]);

  const handleSignin = useCallback(async () => {
    // Prevent multiple simultaneous signin attempts
    if (signinInProgress.current || isProcessing) {
      logger.log('🔄 Signin already in progress, skipping...');
      return;
    }

    try {
      signinInProgress.current = true;
      setIsProcessing(true);

      // Get the JWT token from Web3Auth
      let accessToken: string | null = null;

      // Try multiple methods to get the JWT token
      if (web3Auth?.provider) {
        try {
          // Method 1: Try to get user info with idToken from Web3Auth
          const web3AuthUserInfo = await web3Auth.getUserInfo();
          interface Web3AuthUserInfo {
            idToken?: string;
            [key: string]: unknown;
          }
          const userInfoTyped = web3AuthUserInfo as Web3AuthUserInfo;
          accessToken = userInfoTyped?.idToken || null;
          logger.log('🔍 Web3Auth getUserInfo result:', { hasIdToken: !!accessToken });
        } catch (error) {
          logger.warn('⚠️ Could not get idToken from Web3Auth getUserInfo:', { error });
        }
      }

      // Method 2: Fallback to userInfo from hook
      if (!accessToken) {
        const rawUserInfo = userInfo as Record<string, unknown>;
        accessToken = (rawUserInfo?.idToken as string) || null;
        logger.log('🔍 Fallback userInfo result:', { hasIdToken: !!accessToken });

        // Debug: Log all available keys in userInfo
        logger.log('🔍 Available userInfo keys:', Object.keys(rawUserInfo || {}));
        logger.log('🔍 Full userInfo object:', rawUserInfo);
      }

      // For now, we'll proceed without JWT token if it's not available
      // The backend can handle merchant creation using Web3Auth metadata
      if (!accessToken) {
        logger.warn('⚠️ No JWT token found, proceeding with Web3Auth metadata only');
        accessToken = 'web3auth_no_token'; // Placeholder token
      }

      // Get wallet address - try multiple sources
      let currentWalletAddress = walletAddress;

      // If walletAddress is not available, try to get it directly from Web3Auth
      if (!currentWalletAddress && web3Auth?.provider) {
        try {
          logger.log('🔍 Wallet address not available, trying to get it directly from Web3Auth...');
          const accounts = (await web3Auth.provider.request({
            method: 'eth_accounts',
          })) as string[];

          if (accounts && Array.isArray(accounts) && accounts.length > 0) {
            currentWalletAddress = accounts[0];
            logger.log(
              '✅ Got wallet address directly from Web3Auth provider:',
              currentWalletAddress
            );
          }
        } catch (error) {
          logger.error('❌ Failed to get address directly from Web3Auth provider:', error);
        }
      }

      if (!currentWalletAddress) {
        logger.error('❌ No wallet address available from any source:', {
          walletAddress,
          web3AuthAddress,
          wagmiAddress,
          hasProvider: !!web3Auth?.provider,
          isConnected,
        });
        throw new Error('No wallet address available from Web3Auth or Wagmi');
      }

      const ownerWeb3AuthId = userInfo?.email || 'unknown';

      // Prepare wallet data matching CreateWalletRequest structure
      // Web3Auth automatically provides smart accounts when accountAbstractionConfig is configured
      const walletData = {
        wallet_type: 'web3auth_smart_account', // Indicate this is a smart account
        wallet_address: currentWalletAddress,
        network_type: 'evm',
        nickname: 'Cyphera Smart Account Wallet',
        is_primary: true,
        verified: true,
        metadata: {
          created_via: 'web3auth',
          connector_name: 'web3auth',
          user_email: userInfo?.email,
          is_smart_account: true,
        },
      };

      logger.log('🔐 Signing in with Web3Auth Smart Account:', {
        address: currentWalletAddress,
        wallet_type: walletData.wallet_type,
        is_smart_account: true,
      });

      const accountData: AccountRequest = {
        name: userInfo?.name || 'Unknown User',
        account_type: AccountType.MERCHANT,
        business_name: userInfo?.name || userInfo?.email?.split('@')[0] || 'Unknown Business',
        website_url: (userInfo as Record<string, unknown>)?.profileImage as string,
        support_email: userInfo?.email,
        wallet_data: walletData, // Include wallet data in the request
        metadata: {
          ownerWeb3AuthId: ownerWeb3AuthId,
          email: userInfo?.email,
          verifier: 'google',
          verifierId: ownerWeb3AuthId,
          raw_userInfo: userInfo,
        },
      };

      const response = await fetch('/api/auth/signin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(accountData),
      });

      logger.log('📡 Signin API response status:', response.status);

      if (!response.ok) {
        const errorData = await response.json();
        logger.error('❌ Signin API error:', errorData);
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      const signinResponse = await response.json();
      logger.log('✅ Signin API success:', {
        hasAccount: !!signinResponse?.account,
        accountId: signinResponse?.account?.id,
        finishedOnboarding: signinResponse?.account?.finished_onboarding,
      });

      const hasFinishedOnboarding = signinResponse?.account?.finished_onboarding;

      // Mark that signin was attempted and successful
      setSigninAttempted(true);
      setHasValidSession(true);

      // Add a small delay to ensure session is set before redirect
      await new Promise((resolve) => setTimeout(resolve, 200));

      if (hasFinishedOnboarding === true) {
        logger.log('🔄 Redirecting to dashboard...');
        router.push(redirectTo || '/merchants/dashboard');
      } else {
        logger.log('🔄 Redirecting to onboarding...');
        router.push('/merchants/onboarding');
      }
    } catch (error) {
      logger.error('❌ Signin failed:', error);
      setSigninAttempted(true); // Mark as attempted even on failure to prevent loops
      setHasValidSession(false);
    } finally {
      signinInProgress.current = false;
      setIsProcessing(false);
    }
  }, [
    walletAddress,
    web3Auth,
    userInfo,
    router,
    redirectTo,
    web3AuthAddress,
    wagmiAddress,
    isConnected,
    isProcessing,
  ]);

  // Auto-signin when connected (but not if we just logged out or already processing)
  useEffect(() => {
    const shouldAttemptSignin =
      isConnected &&
      userInfo &&
      walletAddress &&
      !justLoggedOut &&
      !isProcessing &&
      !signinAttempted &&
      hasValidSession === false && // Only signin if we confirmed no valid session
      !signinInProgress.current;

    if (shouldAttemptSignin) {
      logger.log('🔄 Auto-signin triggered - conditions met', {
        isConnected,
        hasUserInfo: !!userInfo,
        hasWalletAddress: !!walletAddress,
        justLoggedOut,
        isProcessing,
        signinAttempted,
        hasValidSession,
        signinInProgress: signinInProgress.current,
      });
      // Add a small delay to prevent rapid firing
      const timeoutId = setTimeout(() => {
        handleSignin();
      }, 100); // Reduced delay for faster response

      return () => clearTimeout(timeoutId);
    } else if (isConnected && userInfo && !walletAddress && !justLoggedOut && !isProcessing) {
      // Wallet is connected but address not available yet - wait a bit longer
      logger.log('⏳ Waiting for wallet address to be available...');
    } else if (hasValidSession === true) {
      logger.log('✅ Valid session exists, skipping auto-signin');
      // Redirect to appropriate page if we have a valid session
      const timeoutId = setTimeout(() => {
        router.push(redirectTo || '/merchants/dashboard');
      }, 100);

      return () => clearTimeout(timeoutId);
    } else {
      logger.log('🔍 Auto-signin conditions not met:', {
        isConnected,
        hasUserInfo: !!userInfo,
        hasWalletAddress: !!walletAddress,
        justLoggedOut,
        isProcessing,
        signinAttempted,
        hasValidSession,
        signinInProgress: signinInProgress.current,
      });
    }
  }, [
    isConnected,
    userInfo,
    walletAddress,
    handleSignin,
    isProcessing,
    signinAttempted,
    hasValidSession,
    redirectTo,
    router,
  ]);

  const handleManualConnect = async () => {
    // Clear the logout flag and connect
    justLoggedOut = false;
    setShouldShowModal(false);
    setSigninAttempted(false); // Reset signin attempt flag
    setHasValidSession(null); // Reset session state
    try {
      await connect();
    } catch (error) {
      logger.error('❌ Manual connect failed:', error);
    }
  };

  // Show manual login button if we just logged out
  if (shouldShowModal) {
    return (
      <div className="text-center py-12">
        <button
          onClick={handleManualConnect}
          className="px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded-lg font-medium hover:from-blue-700 hover:to-purple-700 transition-all duration-200 shadow-lg hover:shadow-xl"
        >
          Sign In with Web3Auth
        </button>
      </div>
    );
  }

  // Show loading state when processing signin
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

  // Show loading state when connected but waiting for address
  if (isConnected && userInfo && !walletAddress) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          <p className="text-gray-600">Connecting your wallet...</p>
        </div>
      </div>
    );
  }

  // Show loading state while checking session
  if (hasValidSession === null) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
          <p className="text-gray-600">Checking authentication...</p>
          {/* Add manual trigger and fallback options */}
          <div className="mt-4 space-y-2">
            <button
              onClick={async () => {
                logger.log('🔧 Manual session recheck triggered');
                setSessionCheckAttempted(false);
                setHasValidSession(null);
                await validateSession();
              }}
              className="text-sm text-blue-600 hover:text-blue-700 underline block"
            >
              Taking too long? Click to retry
            </button>
            <button
              onClick={async () => {
                logger.log('🔧 Manual Web3Auth connection triggered');
                justLoggedOut = false;
                setShouldShowModal(false);
                setSigninAttempted(false);
                setHasValidSession(null);
                setSessionCheckAttempted(false);
                try {
                  await connect();
                } catch (error) {
                  logger.error('❌ Manual connect failed:', error);
                  setShouldShowModal(true);
                }
              }}
              className="text-sm text-purple-600 hover:text-purple-700 underline block"
            >
              Or try reconnecting Web3Auth
            </button>
          </div>
        </div>
      </div>
    );
  }

  // Show signin button if we have Web3Auth connection but no valid session and auto-signin hasn't worked
  if (
    isConnected &&
    userInfo &&
    walletAddress &&
    hasValidSession === false &&
    signinAttempted &&
    !isProcessing
  ) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <p className="text-gray-600">Connected to Web3Auth but authentication incomplete</p>
          <button
            onClick={() => {
              logger.log('🔧 Manual signin triggered');
              setSigninAttempted(false);
              setIsProcessing(false);
              signinInProgress.current = false;
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

  // Return nothing - let Web3Auth modal handle the UI
  return null;
}

// Main component that handles SSR safety
export function Web3AuthLogin({ redirectTo, autoConnect = false }: Web3AuthLoginProps) {
  const [isClient, setIsClient] = useState(false);

  // Client-side check to prevent SSR issues
  useEffect(() => {
    setIsClient(true);
  }, []);

  // Don't render anything on server-side
  if (!isClient) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
          <p className="text-gray-600">Loading Web3Auth...</p>
        </div>
      </div>
    );
  }

  // Render the client-side component that can safely use Web3Auth hooks
  return <Web3AuthLoginClient redirectTo={redirectTo} autoConnect={autoConnect} />;
}

export function getAuthState() {
  return {
    justLoggedOut,
  };
}

export function setLogoutFlag() {
  justLoggedOut = true;
}
