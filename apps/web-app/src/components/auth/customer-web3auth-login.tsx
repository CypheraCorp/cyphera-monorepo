'use client';

import {
  useWeb3AuthConnect,
  useWeb3AuthDisconnect,
  useWeb3AuthUser,
  useWeb3Auth,
} from '@web3auth/modal/react';
import { useAccount } from 'wagmi';
import { useEffect, useState, useCallback, useRef } from 'react';
import Image from 'next/image';
import { AccountRequest, AccountType } from '@/types/account';
import { CreateWalletRequest } from '@/types/wallet';
import { useRouter } from 'next/navigation';
import { logger } from '@/lib/core/logger/logger-utils';

interface CustomerWeb3AuthLoginProps {
  redirectTo?: string | null;
  autoConnect?: boolean;
  onSuccess?: (customerData: unknown) => void;
}

// Global flag to prevent auto-connect immediately after logout (resets on page reload)
let justLoggedOut = false;

// Function to check if user just logged out from localStorage
function checkLogoutFlag(): boolean {
  if (typeof window !== 'undefined') {
    const logoutFlag = window.localStorage.getItem('web3auth-customer-logout');
    return logoutFlag === 'true';
  }
  return false;
}

// Function to clear logout flag from localStorage
function clearLogoutFlag(): void {
  if (typeof window !== 'undefined') {
    window.localStorage.removeItem('web3auth-customer-logout');
  }
}

// Client-side only component that uses Web3Auth hooks
function CustomerWeb3AuthLoginClient({
  redirectTo,
  autoConnect = false,
  onSuccess,
}: CustomerWeb3AuthLoginProps) {
  const router = useRouter();
  const [shouldShowModal, setShouldShowModal] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [web3AuthAddress, setWeb3AuthAddress] = useState<string | null>(null);
  const [hasValidSession, setHasValidSession] = useState<boolean | null>(null);
  const [signinAttempted, setSigninAttempted] = useState(false);

  // Ref to prevent multiple simultaneous signin attempts
  const signinInProgress = useRef(false);

  // Initialize logout flag from localStorage on component mount
  useEffect(() => {
    if (checkLogoutFlag()) {
      justLoggedOut = true;
      logger.log('🔍 Customer logout flag detected from localStorage');
    }
  }, []);

  // Web3Auth hooks
  const { connect } = useWeb3AuthConnect();
  const { disconnect } = useWeb3AuthDisconnect();
  const { userInfo } = useWeb3AuthUser();
  const { web3Auth, isConnected } = useWeb3Auth();

  // Check if currently connecting (derived from isConnected state changes)
  const [isConnecting] = useState(false);

  // Wagmi hook for account info
  const { address: wagmiAddress } = useAccount();

  // Check for existing valid customer session
  useEffect(() => {
    async function checkCustomerSession() {
      try {
        logger.log('🔍 Checking customer session...');
        const response = await fetch('/api/auth/customer/me', {
          method: 'GET',
          credentials: 'include',
        });

        if (response.ok) {
          const sessionData = await response.json();
          const hasSession = !!sessionData?.customer;
          setHasValidSession(hasSession);
          logger.log('✅ Customer session check result:', hasSession, sessionData);
        } else {
          logger.log('❌ No valid customer session found, status:', response.status);
          setHasValidSession(false);
        }
      } catch (error) {
        logger.warn('⚠️ Customer session check failed:', { error });
        setHasValidSession(false);
      }
    }

    // Add a small delay to ensure Web3Auth is initialized
    const timeoutId = setTimeout(checkCustomerSession, 100);
    return () => clearTimeout(timeoutId);
  }, []);

  // Get wallet address directly from Web3Auth provider when connected
  useEffect(() => {
    async function getWeb3AuthAddress() {
      if (isConnected && web3Auth?.provider) {
        try {
          logger.log('🔍 Fetching wallet address from Web3Auth provider...');

          // Get accounts directly from Web3Auth provider
          const accounts = (await web3Auth.provider.request({
            method: 'eth_accounts',
          })) as string[];

          if (accounts && Array.isArray(accounts) && accounts.length > 0) {
            logger.log('🔗 Customer got wallet address from Web3Auth:', accounts[0]);
            setWeb3AuthAddress(accounts[0]);
          } else {
            logger.warn('⚠️ No accounts found in Web3Auth provider, retrying in 1 second...');

            // Retry after a short delay
            setTimeout(async () => {
              try {
                if (web3Auth?.provider) {
                  const retryAccounts = (await web3Auth.provider.request({
                    method: 'eth_accounts',
                  })) as string[];

                  if (retryAccounts && Array.isArray(retryAccounts) && retryAccounts.length > 0) {
                    logger.log(
                      '🔗 Customer got wallet address from Web3Auth (retry):',
                      retryAccounts[0]
                    );
                    setWeb3AuthAddress(retryAccounts[0]);
                  } else {
                    logger.warn('⚠️ Still no accounts found after retry');
                  }
                } else {
                  logger.warn('⚠️ Web3Auth provider not available on retry');
                }
              } catch (retryError) {
                logger.error('❌ Failed to get address on retry:', retryError);
              }
            }, 1000);
          }
        } catch (error) {
          logger.error('❌ Failed to get address from Web3Auth provider:', error);
        }
      } else if (!isConnected) {
        setWeb3AuthAddress(null);
      }
    }

    // Add a small delay to ensure Web3Auth provider is ready
    const timeoutId = setTimeout(getWeb3AuthAddress, 100);
    return () => clearTimeout(timeoutId);
  }, [isConnected, web3Auth]);

  // Use Web3Auth address as primary, fallback to Wagmi address
  const walletAddress = web3AuthAddress || wagmiAddress;

  const handleAutoSignIn = useCallback(async () => {
    logger.log('🚀 Customer handleAutoSignIn called - FUNCTION EXECUTED!');

    // Prevent multiple simultaneous signin attempts
    if (signinInProgress.current || isProcessing || signinAttempted) {
      logger.log('🔄 Customer signin already in progress or attempted, skipping...', {
        signinInProgress: signinInProgress.current,
        isProcessing,
        signinAttempted,
      });
      return;
    }

    try {
      logger.log('✅ Starting customer signin process');
      signinInProgress.current = true;
      setIsProcessing(true);

      // Ensure we have a valid wallet address
      if (!walletAddress) {
        logger.warn('⚠️ Customer wallet address is not available:', { walletAddress });
        return;
      }

      logger.log('🔍 Customer signin data:', {
        walletAddress,
        userEmail: userInfo?.email,
        userName: userInfo?.name,
      });

      // Create wallet data for customer - Web3Auth automatically provides smart accounts
      // when accountAbstractionConfig is configured in the Web3AuthProvider
      const walletData: CreateWalletRequest = {
        wallet_address: walletAddress,
        wallet_type: 'web3auth_smart_account', // Indicate this is a smart account
        network_type: 'evm',
        nickname: 'Customer Smart Account Wallet',
        is_primary: true,
        verified: true,
      };

      logger.log('🔐 Customer signing in with Web3Auth Smart Account:', {
        address: walletAddress,
        wallet_type: walletData.wallet_type,
        is_smart_account: true,
      });

      // Create account request for customer (using same approach as merchant login)
      const ownerWeb3AuthId = userInfo?.email || 'unknown';

      const accountRequest: AccountRequest = {
        name: userInfo?.name || userInfo?.email || 'Customer Account',
        account_type: AccountType.MERCHANT, // Use merchant type for customer accounts
        metadata: {
          ownerWeb3AuthId: ownerWeb3AuthId,
          verifier: (userInfo as { verifier?: string })?.verifier || 'google',
          verifierId: ownerWeb3AuthId,
          name: userInfo?.name,
          email: userInfo?.email,
          profileImage: userInfo?.profileImage,
          raw_userInfo: userInfo as Record<string, unknown>,
        },
        wallet_data: walletData,
      };

      logger.log('🔍 Customer account request metadata:', {
        ownerWeb3AuthId,
        email: userInfo?.email,
        name: userInfo?.name,
      });

      // Debug: Log the full request being sent
      logger.log('🔍 Customer full account request:', {
        fullRequest: accountRequest,
        metadata: accountRequest.metadata,
        metadataKeys: Object.keys(accountRequest.metadata || {}),
      });

      // Call customer signin endpoint
      logger.log('📡 Making customer signin API request...');
      const response = await fetch('/api/auth/customer/signin', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(accountRequest),
      });

      logger.log('📡 Customer signin API response status:', response.status);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        logger.error('❌ Customer signin API error:', errorData);
        throw new Error(errorData.error || `Customer signin failed: ${response.statusText}`);
      }

      const customerData = await response.json();
      logger.log('✅ Customer signin successful:', customerData);

      // Mark signin as successful
      setSigninAttempted(true);
      setHasValidSession(true);

      // Call success callback if provided
      if (onSuccess) {
        onSuccess(customerData);
      }

      // Redirect to dashboard or specified redirect
      if (redirectTo) {
        logger.log('🔄 Customer redirecting to:', redirectTo);
        router.push(redirectTo);
      } else {
        logger.log('🔄 Customer redirecting to dashboard...');
        router.push('/customers/dashboard');
      }
    } catch (error) {
      logger.error('❌ Customer auto-signin failed:', error);
      setSigninAttempted(true); // Mark as attempted even on failure to prevent loops
      setHasValidSession(false);
    } finally {
      signinInProgress.current = false;
      setIsProcessing(false);
    }
  }, [walletAddress, userInfo, isProcessing, signinAttempted, redirectTo, router, onSuccess]);

  // Auto-signin when connected
  useEffect(() => {
    // More flexible condition - allow auto-signin if no valid session is confirmed OR if session is explicitly false
    const shouldAttemptSignin =
      isConnected &&
      userInfo &&
      walletAddress &&
      !justLoggedOut &&
      !isProcessing &&
      !signinAttempted &&
      (hasValidSession === false || hasValidSession === null) && // Allow signin if session is false or still checking
      !signinInProgress.current;

    logger.log('🔍 Customer auto-signin check:', {
      isConnected,
      hasUserInfo: !!userInfo,
      hasWalletAddress: !!walletAddress,
      justLoggedOut,
      isProcessing,
      signinAttempted,
      hasValidSession,
      signinInProgress: signinInProgress.current,
      shouldAttemptSignin,
    });

    // Extra debugging for the exact values
    logger.log('🔍 Customer auto-signin DETAILED VALUES:', {
      isConnected: isConnected,
      userInfo: userInfo,
      walletAddress: walletAddress,
      justLoggedOut: justLoggedOut,
      isProcessing: isProcessing,
      signinAttempted: signinAttempted,
      hasValidSession: hasValidSession,
      'signinInProgress.current': signinInProgress.current,
      sessionCheckCondition: hasValidSession === false || hasValidSession === null,
      'ALL CONDITIONS RESULT':
        isConnected &&
        userInfo &&
        walletAddress &&
        !justLoggedOut &&
        !isProcessing &&
        !signinAttempted &&
        (hasValidSession === false || hasValidSession === null) &&
        !signinInProgress.current,
      shouldAttemptSignin: shouldAttemptSignin,
    });

    if (shouldAttemptSignin) {
      logger.log(
        '🔄 Customer auto-signin triggered - conditions met, calling handleAutoSignIn in 200ms'
      );
      logger.log('🔍 handleAutoSignIn function check:', {
        'function exists': typeof handleAutoSignIn === 'function',
        'function string': handleAutoSignIn.toString().substring(0, 100) + '...',
      });

      // Shorter delay for more responsive signin
      const timeoutId = setTimeout(() => {
        logger.log('⏰ Customer auto-signin timeout executing - calling handleAutoSignIn now!');
        logger.log('🔍 About to call handleAutoSignIn function...');
        try {
          handleAutoSignIn();
          logger.log('✅ Customer handleAutoSignIn function called successfully');
        } catch (error) {
          logger.error('❌ Error calling handleAutoSignIn:', error);
        }
      }, 200); // Reduced from 500ms to 200ms

      return () => {
        logger.log('🔄 Customer auto-signin timeout cleared');
        clearTimeout(timeoutId);
      };
    } else if (hasValidSession === true) {
      logger.log('✅ Valid customer session exists, skipping auto-signin');
      // Call success callback with existing session if provided
      if (onSuccess) {
        onSuccess({ customer: { message: 'Already authenticated' } });
      }
      // Redirect to appropriate page if we have a valid session
      if (redirectTo) {
        const timeoutId = setTimeout(() => {
          router.push(redirectTo);
        }, 100);

        return () => clearTimeout(timeoutId);
      }
    } else {
      logger.log('❌ Customer auto-signin conditions not met - detailed breakdown:', {
        missingConnection: !isConnected,
        missingUserInfo: !userInfo,
        missingWalletAddress: !walletAddress,
        justLoggedOut,
        currentlyProcessing: isProcessing,
        alreadyAttempted: signinAttempted,
        sessionStatus: hasValidSession,
        signinInProgress: signinInProgress.current,
        '--- SPECIFIC FAILING CONDITIONS ---': '👆 Check these values:',
        'hasValidSession === false || null': hasValidSession === false || hasValidSession === null,
        'hasValidSession actual value': hasValidSession,
        '!signinAttempted': !signinAttempted,
        'signinAttempted actual value': signinAttempted,
        '!signinInProgress.current': !signinInProgress.current,
        'signinInProgress.current actual value': signinInProgress.current,
      });
    }
  }, [
    isConnected,
    userInfo,
    walletAddress,
    handleAutoSignIn,
    isProcessing,
    signinAttempted,
    hasValidSession,
    redirectTo,
    router,
    onSuccess,
  ]);

  const handleConnect = useCallback(async () => {
    try {
      justLoggedOut = false; // Reset logout flag when user manually connects
      clearLogoutFlag(); // Clear logout flag from localStorage
      setSigninAttempted(false); // Reset signin attempt flag
      setHasValidSession(null); // Reset session state
      await connect();
    } catch (error) {
      logger.error('❌ Customer Web3Auth connection failed:', error);
    }
  }, [connect, setSigninAttempted, setHasValidSession]);

  // Auto-connect logic
  useEffect(() => {
    if (autoConnect && !isConnected && !isConnecting && !justLoggedOut && !shouldShowModal) {
      logger.log('🔄 Customer auto-connecting to Web3Auth...');
      setShouldShowModal(true);
    }
  }, [autoConnect, isConnected, isConnecting, shouldShowModal]);

  // Handle showing modal
  useEffect(() => {
    if (shouldShowModal && !isConnected && !isConnecting) {
      handleConnect();
      setShouldShowModal(false);
    }
  }, [shouldShowModal, isConnected, isConnecting, handleConnect]);

  const handleDisconnect = async () => {
    try {
      // Set logout flag to prevent immediate auto-connect
      justLoggedOut = true;
      if (typeof window !== 'undefined') {
        window.localStorage.setItem('web3auth-customer-logout', 'true');
      }

      // Call customer logout endpoint
      await fetch('/api/auth/customer/logout', {
        method: 'POST',
      });

      await disconnect();
      setWeb3AuthAddress(null);
      setIsProcessing(false);
      setSigninAttempted(false);
      setHasValidSession(false);

      logger.log('✅ Customer disconnected successfully');
    } catch (error) {
      logger.error('❌ Customer Web3Auth disconnection failed:', error);
    }
  };

  // Show loading state while checking session
  if (hasValidSession === null) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
          <p className="text-gray-600">Checking customer authentication...</p>
        </div>
      </div>
    );
  }

  // Show loading state while processing
  if (isProcessing || isConnecting) {
    return (
      <div className="text-center py-12">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
          <p className="text-gray-600">
            {isConnecting ? 'Connecting your wallet...' : 'Creating your account...'}
          </p>
        </div>
      </div>
    );
  }

  // Show connected state with manual signin option
  if (isConnected && userInfo) {
    return (
      <div className="text-center py-8">
        <div className="flex flex-col items-center space-y-4">
          {userInfo.profileImage && (
            <Image
              src={userInfo.profileImage}
              alt="Profile"
              width={64}
              height={64}
              className="w-16 h-16 rounded-full"
            />
          )}
          <div>
            <p className="text-lg font-semibold">{userInfo.name || userInfo.email}</p>
            <p className="text-sm text-gray-600">{userInfo.email}</p>
            {(web3AuthAddress || wagmiAddress) && (
              <p className="text-xs text-gray-500 font-mono mt-1">
                {(web3AuthAddress || wagmiAddress)?.slice(0, 6)}...
                {(web3AuthAddress || wagmiAddress)?.slice(-4)}
              </p>
            )}
          </div>

          {/* Manual signin button for testing */}
          {hasValidSession === false && !isProcessing && (
            <button
              onClick={() => {
                logger.log('🔴 Manual signin button clicked');
                handleAutoSignIn();
              }}
              className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
            >
              Complete Signin
            </button>
          )}

          <button
            onClick={handleDisconnect}
            className="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors"
          >
            Sign Out
          </button>
        </div>
      </div>
    );
  }

  // Show connect button
  return (
    <div className="text-center py-8">
      <button
        onClick={handleConnect}
        disabled={isConnecting}
        className="px-6 py-3 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50"
      >
        {isConnecting ? 'Connecting...' : 'Sign In with Web3Auth'}
      </button>
    </div>
  );
}

export function CustomerWeb3AuthLogin({
  redirectTo,
  autoConnect = false,
  onSuccess,
}: CustomerWeb3AuthLoginProps) {
  const [isClient, setIsClient] = useState(false);

  // Client-side rendering check
  useEffect(() => {
    setIsClient(true);
  }, []);

  // Don't render on server-side
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

  return (
    <CustomerWeb3AuthLoginClient
      redirectTo={redirectTo}
      autoConnect={autoConnect}
      onSuccess={onSuccess}
    />
  );
}
