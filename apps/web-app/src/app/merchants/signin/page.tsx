'use client';

import { useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  AlertCircle,
  Building2,
  Loader2,
  ArrowLeft,
  HelpCircle,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import Link from 'next/link';
import { useWeb3AuthConnect, useWeb3AuthUser, useWeb3Auth } from '@web3auth/modal/react';
import { useAccount } from 'wagmi';
import { AccountRequest, AccountType } from '@/types/account';
import { CreateWalletRequest } from '@/types/wallet';
import { useAuthStore } from '@/store/auth';
import { logger } from '@/lib/core/logger/logger-utils';

// Status steps for better system visibility
const SIGNIN_STEPS = [
  { id: 'connect', label: 'Signing In', description: 'Connecting to Web3Auth' },
  { id: 'authenticate', label: 'Authenticate', description: 'Verifying your identity' },
  { id: 'setup', label: 'Setup Account', description: 'Setting up your merchant account' },
  { id: 'complete', label: 'Complete', description: 'Redirecting to dashboard' },
];

type SignInStep = 'idle' | 'connect' | 'authenticate' | 'setup' | 'complete';

export default function MerchantSignInPage() {
  const router = useRouter();
  const [isProcessing, setIsProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentStep, setCurrentStep] = useState<SignInStep>('idle');
  const [debugInfo, setDebugInfo] = useState<string>('');
  const [showHelp, setShowHelp] = useState(false);

  // Web3Auth hooks
  const { connect } = useWeb3AuthConnect();
  const { userInfo } = useWeb3AuthUser();
  const { web3Auth, isConnected } = useWeb3Auth();
  const { address } = useAccount();

  // Auth store actions
  const { setTokens } = useAuthStore();

  // Debug logging
  const addDebugInfo = useCallback((info: string) => {
    logger.debug(`[MerchantSignin] ${info}`);
    setDebugInfo((prev) => prev + '\n' + info);
  }, []);

  // Clear error when user attempts to retry
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  // Auto-signin when Web3Auth is connected and we have user data
  interface UserInfo {
    email?: string;
    name?: string;
    profileImage?: string;
    verifierId?: string;
    [key: string]: unknown;
  }

  const handleAutoSignIn = useCallback(
    async (providedUserInfo?: UserInfo) => {
      try {
        setCurrentStep('authenticate');

        // Use provided userInfo or fall back to hook userInfo
        const currentUserInfo = providedUserInfo || userInfo;

        if (!currentUserInfo) {
          addDebugInfo('No user info available for signin');
          return;
        }

        addDebugInfo('Creating wallet data for signin...');

        // Get wallet address
        let walletAddress: string;
        try {
          if (web3Auth?.provider) {
            const accounts = (await web3Auth.provider.request({
              method: 'eth_accounts',
            })) as string[];
            walletAddress = accounts[0];
            addDebugInfo(`Got wallet address from web3Auth: ${walletAddress}`);
          } else {
            // Fallback to wagmi
            walletAddress = address || '';
            addDebugInfo(`Got wallet address from wagmi: ${walletAddress}`);
          }
        } catch (err) {
          addDebugInfo(`Error getting wallet address: ${err}`);
          throw new Error('Failed to get wallet address');
        }

        if (!walletAddress) {
          throw new Error('No wallet address available');
        }

        // Get user email
        const userEmail = currentUserInfo.email;
        if (!userEmail) {
          throw new Error('No email found in user info');
        }

        setCurrentStep('setup');
        addDebugInfo(`Creating wallet data for: ${userEmail}`);

        const walletData: CreateWalletRequest = {
          wallet_address: walletAddress,
          wallet_type: 'web3auth',
          network_type: 'evm',
          is_primary: true,
          verified: true,
          metadata: {
            connector_name: 'web3auth',
            created_via: 'web3auth',
            is_smart_account: true,
            smart_account_type: 'web3auth_smart_account',
            user_email: userEmail,
            web3auth_verifier: (currentUserInfo as Record<string, unknown>).verifier as string || 'unknown',
            web3auth_verifier_id: (currentUserInfo as Record<string, unknown>).verifierId as string || userEmail,
          },
        };

        // Create account request with proper structure
        const accountRequest: AccountRequest = {
          name: currentUserInfo.name || currentUserInfo.email || 'Merchant Account',
          account_type: AccountType.MERCHANT,
          business_name: currentUserInfo.name || currentUserInfo.email?.split('@')[0] || 'Business',
          support_email: userEmail,
          wallet_data: walletData,
          metadata: {
            ownerWeb3AuthId: (currentUserInfo as Record<string, unknown>).verifierId as string || userEmail,
            verifier: (currentUserInfo as Record<string, unknown>).verifier as string || 'unknown',
            verifierId: (currentUserInfo as Record<string, unknown>).verifierId as string || userEmail,
            email: userEmail,
            name: currentUserInfo.name,
            profileImage: currentUserInfo.profileImage,
            raw_userInfo: currentUserInfo,
          },
        };

        addDebugInfo('Sending signin request to backend...');

        const response = await fetch('/api/auth/signin', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(accountRequest),
        });

        if (!response.ok) {
          const errorData = await response.text();
          addDebugInfo(`Backend signin failed: ${response.status} - ${errorData}`);
          throw new Error(`Authentication failed. Please try again.`);
        }

        await response.json();
        addDebugInfo('Backend signin successful!');

        // Auth state will automatically update via React Query
        addDebugInfo('Authentication state updated');

        setCurrentStep('complete');

        // Small delay to show completion step
        setTimeout(() => {
          addDebugInfo('Redirecting to dashboard...');
          router.push('/merchants/dashboard');
        }, 1000);
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : 'Authentication failed. Please try again.';
        logger.error('[MerchantSignin] Signin Error', error);
        setError(errorMessage);
        addDebugInfo(`Signin Error: ${errorMessage}`);
        setIsProcessing(false);
        setCurrentStep('idle');
      }
    },
    [userInfo, web3Auth, address, router, addDebugInfo]
  );

  const handleSignIn = async () => {
    try {
      setIsProcessing(true);
      setError(null);
      setCurrentStep('connect');
      addDebugInfo('Starting Web3Auth connection...');

      // Check if already connected
      if (isConnected && userInfo) {
        addDebugInfo('Already connected, proceeding with signin...');
        await handleAutoSignIn();
        return;
      }

      // Connect to Web3Auth
      addDebugInfo('Calling Web3Auth connect()...');
      await connect();
      addDebugInfo('Web3Auth connect() completed, waiting for connection state...');

      // Poll for connection status with detailed logging
      const maxWait = 15000; // 15 seconds
      const startTime = Date.now();
      let pollCount = 0;

      while (Date.now() - startTime < maxWait) {
        pollCount++;

        // Try to get user info directly from web3Auth if hook isn't working
        let directUserInfo = null;
        try {
          if (web3Auth?.connected) {
            directUserInfo = await web3Auth.getUserInfo();
          }
        } catch (err) {
          addDebugInfo(`Failed to get direct user info: ${err}`);
        }

        addDebugInfo(
          `Poll ${pollCount}: isConnected=${isConnected}, userInfo=${!!userInfo}, web3Auth.connected=${web3Auth?.connected}, directUserInfo=${!!directUserInfo}`
        );

        // Check if we have what we need (prefer hook userInfo, fallback to direct)
        const availableUserInfo = userInfo || directUserInfo;

        if ((isConnected || web3Auth?.connected) && availableUserInfo) {
          addDebugInfo('Connection successful! Proceeding with signin...');

          // Temporarily set userInfo if we got it directly
          if (!userInfo && directUserInfo) {
            addDebugInfo('Using direct user info from web3Auth.getUserInfo()');
          }

          await handleAutoSignIn(availableUserInfo);
          return;
        }

        // Wait before next check
        await new Promise((resolve) => setTimeout(resolve, 500));
      }

      // If we get here, connection timed out
      addDebugInfo('Connection timeout - checking final states...');
      addDebugInfo(
        `Final state: isConnected=${isConnected}, userInfo=${!!userInfo}, web3Auth.connected=${web3Auth?.connected}`
      );

      // Try one more time with current state, including direct user info
      let finalUserInfo = null;
      try {
        if (web3Auth?.connected) {
          finalUserInfo = await web3Auth.getUserInfo();
        }
      } catch (err) {
        addDebugInfo(`Failed to get final user info: ${err}`);
      }

      const availableFinalUserInfo = userInfo || finalUserInfo;

      if ((isConnected || web3Auth?.connected) && availableFinalUserInfo) {
        addDebugInfo('Found connection on timeout check, proceeding...');
        await handleAutoSignIn(availableFinalUserInfo);
        return;
      }

      throw new Error('Connection timed out. Please check your internet connection and try again.');
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'An unexpected error occurred. Please try again.';
      logger.error('[MerchantSignin] Connection Error', error);

      // Check if user cancelled the Web3Auth modal
      if (
        error instanceof Error &&
        (error.message.includes('User cancelled') ||
          error.message.includes('User closed') ||
          error.message.includes('User denied') ||
          error.message.includes('Modal closed') ||
          error.message.includes('cancelled') ||
          error.message.includes('aborted'))
      ) {
        // User cancelled - same logic as cancel button
        addDebugInfo('User cancelled Web3Auth modal');
        handleCancel();
        return;
      }

      setError(errorMessage);
      addDebugInfo(`Connection Error: ${errorMessage}`);
      setIsProcessing(false);
      setCurrentStep('idle');
    }
  };

  // Cancel/retry actions for user control
  const handleCancel = () => {
    setIsProcessing(false);
    setCurrentStep('idle');
    setError(null);
  };

  // Get current step info for progress display
  const getCurrentStepInfo = () => {
    return SIGNIN_STEPS.find((step) => step.id === currentStep) || SIGNIN_STEPS[0];
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Header with clear branding and navigation */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center mb-4">
            <Building2 className="h-8 w-8 text-indigo-600 mr-2" />
            <span className="text-2xl font-bold text-gray-900">Cyphera</span>
          </div>
          <div className="flex items-center justify-center space-x-4 text-sm">
            <Link
              href="/customers/signin"
              className="text-indigo-600 hover:text-indigo-500 transition-colors flex items-center"
            >
              <ArrowLeft className="h-4 w-4 mr-1" />
              Customer Portal
            </Link>
            <span className="text-gray-300">|</span>
            <button
              onClick={() => setShowHelp(!showHelp)}
              className="text-gray-600 hover:text-gray-500 transition-colors flex items-center"
            >
              <HelpCircle className="h-4 w-4 mr-1" />
              Help
            </button>
          </div>
        </div>

        {/* Help section - contextual help when needed */}
        {showHelp && (
          <Card className="mb-6 shadow-lg border-blue-200">
            <CardContent className="p-4">
              <h3 className="font-semibold text-gray-900 mb-2">How to sign in</h3>
              <div className="space-y-2 text-sm text-gray-600">
                <p>• Click &quot;Sign In with Web3Auth&quot; to connect your wallet</p>
                <p>• Choose your preferred sign-in method (Google, Twitter, etc.)</p>
                <p>• Your merchant account will be created automatically</p>
                <p>• New users will be guided through account setup</p>
              </div>
              <div className="mt-3 pt-3 border-t border-gray-200">
                <p className="text-xs text-gray-500">
                  Having trouble? Contact support at support@cyphera.com
                </p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Sign In Card */}
        <Card className="shadow-xl border-0 bg-white/80 backdrop-blur-sm">
          <CardHeader className="text-center space-y-2">
            <CardTitle className="text-2xl font-bold text-gray-900">Merchant Portal</CardTitle>
            <CardDescription className="text-gray-600">
              Sign in to manage your business, products, and subscriptions
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-6">
            {/* Progress indicator when processing */}
            {isProcessing && (
              <div className="space-y-3">
                <div className="flex items-center justify-end">
                  {currentStep !== 'complete' && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleCancel}
                      className="text-gray-500 hover:text-gray-700"
                    >
                      Cancel
                    </Button>
                  )}
                </div>
              </div>
            )}

            {/* Error handling with clear actions */}
            {error && (
              <Alert className="border-red-200 bg-red-50">
                <AlertCircle className="h-4 w-4 text-red-500" />
                <AlertDescription className="text-red-700">
                  <div className="flex items-center justify-between">
                    <span>{error}</span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={clearError}
                      className="ml-2 text-red-600 hover:text-red-700"
                    >
                      <XCircle className="h-3 w-3 mr-1" />
                      Dismiss
                    </Button>
                  </div>
                  <div className="mt-2 text-sm">
                    <button
                      onClick={handleSignIn}
                      className="text-red-600 hover:text-red-700 underline"
                    >
                      Try again
                    </button>
                    {' or '}
                    <button
                      onClick={() => setShowHelp(true)}
                      className="text-red-600 hover:text-red-700 underline"
                    >
                      get help
                    </button>
                  </div>
                </AlertDescription>
              </Alert>
            )}

            {/* Primary action button */}
            <Button
              onClick={handleSignIn}
              disabled={isProcessing}
              className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:bg-indigo-400 text-white font-medium py-3 rounded-lg transition-all duration-200 transform hover:scale-[1.02] active:scale-[0.98]"
              size="lg"
            >
              {isProcessing ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  {getCurrentStepInfo().label}...
                </>
              ) : (
                'Sign in'
              )}
            </Button>

            {/* Success state */}
            {currentStep === 'complete' && (
              <div className="text-center space-y-2">
                <CheckCircle className="h-8 w-8 text-green-500 mx-auto" />
                <p className="text-sm text-green-600 font-medium">
                  Welcome to Cyphera! Redirecting to your dashboard...
                </p>
              </div>
            )}

            {/* Informational text */}
            <div className="text-center space-y-2">
              <p className="text-xs text-gray-500">
                New to Cyphera? Your merchant account will be created automatically
              </p>
              <p className="text-xs text-gray-400">Secure • Decentralized • Easy to use</p>
            </div>

            {/* Debug Info (only show in development) */}
            {process.env.NODE_ENV === 'development' && debugInfo && (
              <details className="mt-4">
                <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-700">
                  Debug Information
                </summary>
                <pre className="text-xs text-gray-400 mt-2 p-2 bg-gray-50 rounded whitespace-pre-wrap max-h-40 overflow-y-auto font-mono">
                  {debugInfo}
                </pre>
              </details>
            )}
          </CardContent>
        </Card>

        {/* Footer */}
        <div className="text-center mt-8 space-y-2">
          <p className="text-xs text-gray-500">
            © 2024 Cyphera. Secure Web3 subscription platform.
          </p>
          <div className="flex justify-center space-x-4 text-xs text-gray-400">
            <Link href="/terms" className="hover:text-gray-600 transition-colors">
              Terms
            </Link>
            <Link href="/privacy" className="hover:text-gray-600 transition-colors">
              Privacy
            </Link>
            <Link href="/support" className="hover:text-gray-600 transition-colors">
              Support
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
