import { useState, useEffect } from 'react';
import { useWeb3AuthUser, useWeb3Auth } from '@web3auth/modal/react';
import { logger } from '@/lib/core/logger/logger-utils';
/**
 * Custom hook to track Web3Auth initialization state
 * This prevents the flash of unauthenticated content (FOUC) during initialization
 */
export function useWeb3AuthInitialization() {
  const [isInitializing, setIsInitializing] = useState(true);
  const [hasInitialized, setHasInitialized] = useState(false);

  // Web3Auth hooks - must be called unconditionally at the top level
  const userResult = useWeb3AuthUser();
  const authResult = useWeb3Auth();

  // Extract values with proper error handling
  const userInfo = userResult?.userInfo || null;
  const isConnected = authResult?.isConnected || false;
  const status = authResult?.status || 'not_ready';
  const web3Auth = authResult?.web3Auth || null;

  useEffect(() => {
    let timeoutId: NodeJS.Timeout;

    logger.log('🔍 Web3Auth Status:', {
      status,
      isConnected,
      hasUserInfo: !!userInfo,
      hasWeb3Auth: !!web3Auth,
    });

    // Web3Auth is considered initialized when:
    // 1. The status is ready (not 'not_ready' or 'initializing')
    // 2. We have a definitive authentication state
    const checkInitialization = () => {
      if (status === 'ready') {
        logger.log('✅ Web3Auth ready, initialization complete');
        setIsInitializing(false);
        setHasInitialized(true);
        return;
      }

      if (status === 'connected' || status === 'disconnected') {
        logger.log('✅ Web3Auth status definitive, initialization complete:', status);
        setIsInitializing(false);
        setHasInitialized(true);
        return;
      }

      // Give Web3Auth more time to initialize (increased from 3s to 8s)
      if (!hasInitialized) {
        timeoutId = setTimeout(() => {
          logger.log(
            '🕐 Web3Auth initialization timeout reached after 8s, proceeding with status:',
            status
          );
          setIsInitializing(false);
          setHasInitialized(true);
        }, 8000);
      }
    };

    checkInitialization();

    return () => {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [status, hasInitialized, isConnected, userInfo, web3Auth]);

  const isAuthenticated = isConnected && !!userInfo;

  logger.log('🔍 useWeb3AuthInitialization result:', {
    isInitializing,
    isAuthenticated,
    hasUserInfo: !!userInfo,
    isConnected,
    status,
  });

  return {
    isInitializing,
    isAuthenticated,
    userInfo,
    isConnected,
    status,
  };
}

/**
 * Safe version of the Web3Auth hooks that handles context availability
 * @deprecated Use useWeb3AuthInitialization instead for better initialization tracking
 */
export function useSafeCustomerAuth() {
  // Web3Auth hooks - must be called unconditionally at the top level
  const userResult = useWeb3AuthUser();
  const authResult = useWeb3Auth();

  // Extract values with proper error handling
  const userInfo = userResult?.userInfo || null;
  const isConnected = authResult?.isConnected || false;

  return { userInfo, isConnected };
}
