'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import { clientLogger } from '@/lib/core/logger/logger-client';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { useToast } from '@/components/ui/use-toast';
import { useCircleSDK } from '@/hooks/web3';
import { CircleAPI } from '@/services/cyphera-api/circle';
import { Loader2, ShieldCheck, AlertTriangle, RefreshCw } from 'lucide-react';
import { v4 as uuidv4 } from 'uuid';
import { SetupPinDialog } from './setup-pin-dialog';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';
import type { CircleUserData, InitializeUserRequest } from '@/types/circle';
import type { UserRequestContext } from '@/services/cyphera-api/api';
import type { NetworkWithTokensResponse } from '@/types/network';
import { Badge } from '@/components/ui/badge';

/** Type for the status update callback */
interface CircleInitializationStatus {
  isInitialized: boolean;
  isPinEnabled: boolean;
}

/**
 * Props for InitializeCircleUser component
 */
interface InitializeCircleUserProps {
  workspaceId: string;
  networks: NetworkWithTokensResponse[];
  /** Callback triggered when initialization status is checked or updated */
  onStatusUpdate?: (status: CircleInitializationStatus) => void;
}

/**
 * InitializeCircleUser component
 *
 * Checks if the user is initialized with Circle and guides them through
 * the initialization process if needed.
 */
export function InitializeCircleUser({
  workspaceId,
  networks = [],
  onStatusUpdate,
}: InitializeCircleUserProps) {
  const [isInitialized, setIsInitialized] = useState<boolean | null>(null);
  const [isChecking, setIsChecking] = useState(true);
  const [isInitializing, setIsInitializing] = useState(false);
  const [isPinSetupOpen, setIsPinSetupOpen] = useState(false);
  const [challengeId, setChallengeId] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);
  const [userData, setUserData] = useState<CircleUserData | null>(null);
  const { toast } = useToast();
  const { client, isInitialized: isSDKInitialized } = useCircleSDK();

  // Filter networks to get Circle-compatible testnets
  const circleTestnetNetworks = useMemo(() => {
    return networks.filter(
      (network) => network.network.is_testnet && network.network.circle_network_type
    );
  }, [networks]);

  // Helper to create UserRequestContext
  const getUserRequestContext = (): UserRequestContext | null => {
    if (!workspaceId) {
      clientLogger.error('Missing workspaceId for context');
      return null;
    }
    // Add account_id and user_id if available/needed by getAPIContext
    return {
      workspace_id: workspaceId,
    };
  };

  // Call the callback in useEffect and checkUserInitialization
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const checkUserInitializationCallback = useCallback(() => {
    checkUserInitialization();
  }, []);

  useEffect(() => {
    if (isSDKInitialized) {
      checkUserInitializationCallback();
    } else {
      // Report not initialized if SDK isn't ready
      onStatusUpdate?.({ isInitialized: false, isPinEnabled: false });
    }
  }, [isSDKInitialized, retryCount, checkUserInitializationCallback, onStatusUpdate]);

  const checkUserInitialization = async () => {
    let status: CircleInitializationStatus = { isInitialized: false, isPinEnabled: false };
    if (!isSDKInitialized) {
      onStatusUpdate?.(status);
      return;
    }
    try {
      const context = getUserRequestContext();
      if (!context) {
        setErrorMessage('Cannot check status: missing user token or workspace ID.');
        onStatusUpdate?.(status);
        return;
      }
      setIsChecking(true);
      setErrorMessage(null);
      const circleApi = new CircleAPI();
      try {
        const userResponse = await circleApi.getUserByToken(context);
        if (userResponse?.data?.user) {
          const user = userResponse.data.user;
          const pinEnabled = user.pinStatus === 'ENABLED';
          setIsInitialized(true);
          status = { isInitialized: true, isPinEnabled: pinEnabled };
          setUserData(user as unknown as CircleUserData);
        } else {
          setIsInitialized(false);
          status = { isInitialized: false, isPinEnabled: false };
        }
      } catch (error: unknown) {
        setIsInitialized(false);
        status = { isInitialized: false, isPinEnabled: false };

        // Type guard for API error responses
        interface APIError extends Error {
          status?: number;
        }

        const apiError = error as APIError;
        if (apiError?.status !== 404) {
          setErrorMessage('Failed to check user status. ' + (apiError.message || ''));
        }
      }
    } catch (error) {
      clientLogger.error('Error checking initialization', {
        error: error instanceof Error ? error.message : error,
      });
      setErrorMessage(
        error instanceof Error
          ? error.message
          : 'Failed to check user initialization status. Please try again.'
      );
      setIsInitialized(false);
      status = { isInitialized: false, isPinEnabled: false };
    } finally {
      setIsChecking(false);
      onStatusUpdate?.(status);
    }
  };

  // Handle retrying the check
  const handleRetryCheck = () => {
    setRetryCount((prev) => prev + 1);
  };

  // Handle initializing a user
  const handleInitializeUser = async () => {
    if (!isSDKInitialized || !client) {
      toast({
        title: 'Error',
        description: 'SDK not initialized. Please refresh the page and try again.',
        variant: 'destructive',
      });
      return;
    }

    // Use the filtered networks for the API call
    const circleNetworkIds = circleTestnetNetworks
      .map((n) => n.network.circle_network_type)
      .filter((id): id is string => !!id); // Filter out potential null/undefined

    if (circleNetworkIds.length === 0) {
      toast({
        title: 'Error',
        description: 'No compatible Circle testnet networks found in configuration.',
        variant: 'destructive',
      });
      return;
    }

    try {
      const context = getUserRequestContext();
      if (!context) {
        toast({
          title: 'Error',
          description: 'Missing user token or workspace ID for initialization.',
          variant: 'destructive',
        });
        return;
      }

      setIsInitializing(true);
      setErrorMessage(null);
      const circleApi = new CircleAPI();

      const idempotencyKey = uuidv4();

      // Prepare the request body with dynamically filtered blockchain IDs
      const initRequestBody: InitializeUserRequest = {
        idempotency_key: idempotencyKey,
        blockchains: circleNetworkIds,
        account_type: 'SCA',
      };

      const initResponse = await circleApi.initializeUser(context, initRequestBody);

      if (initResponse && initResponse.data && initResponse.data.challengeId) {
        // Open the PIN setup dialog with the challenge ID
        setChallengeId(initResponse.data.challengeId);

        // Initialize userData with response data if available, otherwise use defaults
        // This satisfies the type requirements while setting up PIN flow
        setUserData({
          id: '',
          createDate: new Date().toISOString(),
          pinStatus: 'UNSET',
          status: 'ACTIVE',
          securityQuestionStatus: 'UNSET',
          isPinSetUp: false,
          pinDetails: {
            failedAttempts: 0,
            lockedDate: '',
            lockedExpiryDate: '',
            lastLockOverrideDate: '',
          },
          securityQuestionDetails: {
            failedAttempts: 0,
            lockedDate: '',
            lockedExpiryDate: '',
            lastLockOverrideDate: '',
          },
        });

        setIsPinSetupOpen(true);
      } else {
        throw new Error('Invalid response from user initialization');
      }
    } catch (error) {
      clientLogger.error('Error initializing user', {
        error: error instanceof Error ? error.message : error,
      });
      setErrorMessage(
        error instanceof Error
          ? error.message
          : 'Failed to initialize user. Please try again later.'
      );
      toast({
        title: 'Initialization Failed',
        description: error instanceof Error ? error.message : 'Failed to initialize user',
        variant: 'destructive',
      });
    } finally {
      setIsInitializing(false);
    }
  };

  // Handle PIN setup completion
  const handlePinSetupComplete = () => {
    setIsPinSetupOpen(false);
    toast({
      title: 'Success',
      description: 'Your Circle wallet has been initialized successfully!',
    });
    setIsInitialized(true);

    // Report status update after completion
    onStatusUpdate?.({ isInitialized: true, isPinEnabled: true });
  };

  if (isChecking) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Checking Circle Wallet Status</CardTitle>
          <CardDescription>
            Please wait while we check your initialization status...
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col items-center py-6 gap-4">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
          <Skeleton className="h-4 w-3/4 rounded-lg" />
          <Skeleton className="h-4 w-1/2 rounded-lg" />
        </CardContent>
      </Card>
    );
  }

  if (errorMessage) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-amber-600">
            <AlertTriangle className="h-5 w-5" />
            Error Checking Wallet Status
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive" className="mb-4">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
          <p className="text-sm text-muted-foreground mb-4">
            We encountered a problem checking your Circle wallet status. This might be due to:
          </p>
          <ul className="list-disc pl-5 text-sm space-y-1 text-muted-foreground mb-4">
            <li>A temporary network issue</li>
            <li>The Circle API being temporarily unavailable</li>
            <li>Your authentication token may have expired</li>
          </ul>
        </CardContent>
        <CardFooter>
          <Button onClick={handleRetryCheck} className="w-full gap-2">
            <RefreshCw className="h-4 w-4" />
            Retry
          </Button>
        </CardFooter>
      </Card>
    );
  }

  if (isInitialized) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-green-500" />
            Circle Wallet Initialized
          </CardTitle>
          <CardDescription>
            Your Circle wallet is ready to use. You can now create wallets and perform transactions.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Alert className="bg-green-50 border-green-200 text-green-800 dark:bg-green-900/20 dark:border-green-900 dark:text-green-300">
            <ShieldCheck className="h-4 w-4" />
            <AlertTitle>Wallet Security</AlertTitle>
            <AlertDescription>
              Your Circle wallet is protected by a PIN. Always keep your PIN secure and never share
              it with anyone.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Initialize Circle Wallet</CardTitle>
        <CardDescription>
          You need to initialize your Circle wallet to create wallets and perform transactions.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground mb-4">This is a one-time process that will:</p>
        <ul className="list-disc pl-5 text-sm space-y-1 text-muted-foreground mb-4">
          <li>Create a secure Circle user profile</li>
          <li>Set up a PIN for transaction security</li>
          <li>Allow you to create programmable wallets</li>
        </ul>

        <p className="text-sm text-muted-foreground mb-2">Supported blockchain networks:</p>
        {/* Display filtered networks dynamically */}
        <div className="flex flex-wrap gap-2 mb-4">
          {circleTestnetNetworks.length > 0 ? (
            circleTestnetNetworks.map((network) => (
              <Badge key={network.network.id} variant="secondary">
                {network.network.name}
              </Badge>
            ))
          ) : (
            <Badge variant="outline">No compatible testnets configured</Badge>
          )}
        </div>
      </CardContent>
      <CardFooter>
        <Button
          onClick={handleInitializeUser}
          disabled={isInitializing || circleTestnetNetworks.length === 0}
          className="w-full"
        >
          {isInitializing ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Initializing...
            </>
          ) : (
            'Initialize Circle Wallet'
          )}
        </Button>
      </CardFooter>

      {/* PIN Setup Dialog */}
      <SetupPinDialog
        open={isPinSetupOpen}
        onOpenChange={setIsPinSetupOpen}
        onComplete={handlePinSetupComplete}
        challengeId={challengeId || undefined}
        userData={userData!}
      />
    </Card>
  );
}
