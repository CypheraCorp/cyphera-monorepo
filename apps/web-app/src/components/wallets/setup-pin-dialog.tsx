'use client';

import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Loader2, AlertCircle } from 'lucide-react';
import { useCircleSDK } from '@/hooks/web3';
import { useToast } from '@/components/ui/use-toast';
import { v4 as uuidv4 } from 'uuid';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CircleUserData } from '@/types/circle';
import { logger } from '@/lib/core/logger/logger-utils';

interface SetupPinDialogProps {
  /**
   * Whether the dialog is open
   */
  open: boolean;

  /**
   * Callback when the open state changes
   */
  onOpenChange: (open: boolean) => void;

  /**
   * Callback when PIN setup is complete
   */
  onComplete?: () => void;

  /**
   * Optional challenge ID to use (if already created)
   */
  challengeId?: string;

  /**
   * Validated Circle user data
   */
  userData: CircleUserData;
}

export function SetupPinDialog({
  open,
  onOpenChange,
  onComplete,
  challengeId: initialChallengeId,
  userData,
}: SetupPinDialogProps) {
  const { client, userToken } = useCircleSDK();
  const { toast } = useToast();
  const [isProcessing, setIsProcessing] = useState(false);
  const [challengeId, setChallengeId] = useState<string | undefined>(initialChallengeId);
  const [error, setError] = useState<string | null>(null);

  // Create a PIN challenge when the dialog opens
  useEffect(() => {
    if (open && !challengeId && !isProcessing) {
      createPinChallenge();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, challengeId, isProcessing]);

  // When the dialog opens or closes
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      // Reset state when dialog closes without completing
      setChallengeId(undefined);
      setError(null);
    }

    onOpenChange(newOpen);
  };

  // Create a PIN challenge using the backend API
  const createPinChallenge = async () => {
    if (!client) {
      toast({
        title: 'SDK Not Ready',
        description: 'Circle SDK is not initialized',
        variant: 'destructive',
      });
      return;
    }

    if (!userToken) {
      toast({
        title: 'User Token Missing',
        description: 'Circle user token is required for PIN setup',
        variant: 'destructive',
      });
      setError('User token is missing. Please try again or recreate your wallet.');
      return;
    }

    try {
      setIsProcessing(true);
      setError(null);

      // Check PIN status from the provided userData
      if (userData.pinStatus === 'LOCKED') {
        toast({
          title: 'PIN Locked',
          description: 'Your PIN is currently locked. Please try again later.',
          variant: 'destructive',
        });
        onOpenChange(false);
        return;
      }

      if (userData.pinStatus === 'ENABLED' || userData.isPinSetUp) {
        toast({
          title: 'PIN Already Set',
          description: 'Your PIN is already set up. You can proceed with wallet creation.',
        });
        onComplete?.();
        onOpenChange(false);
        return;
      }

      // Only create a PIN challenge if the status is UNSET
      if (userData.pinStatus !== 'UNSET') {
        throw new Error(`Unexpected PIN status: ${userData.pinStatus}`);
      }

      // Create a unique idempotency key
      const idempotencyKey = uuidv4();

      // Create the PIN challenge via the updated API route
      const challengeResponse = await fetch(`/api/circle/users/pin/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          idempotency_key: idempotencyKey,
          user_token: userToken,
        }),
      });

      if (!challengeResponse.ok) {
        const errorData = await challengeResponse.json();
        throw new Error(errorData.error || 'Failed to create PIN challenge');
      }

      const challengeData = await challengeResponse.json();

      if (!challengeData.data) {
        throw new Error('Invalid response: missing data object');
      }

      // Get the challenge ID from the response
      const newChallengeId = challengeData.data.challenge?.id || challengeData.data.challengeId;

      if (!newChallengeId) {
        throw new Error('Invalid response: missing challenge ID');
      }

      setChallengeId(newChallengeId);
    } catch (error) {
      logger.error('Error creating PIN challenge:', error);
      const errorMsg =
        error instanceof Error ? error.message : 'There was an error initiating PIN setup';
      setError(errorMsg);
      toast({
        title: 'PIN Setup Failed',
        description: errorMsg,
        variant: 'destructive',
      });
    } finally {
      setIsProcessing(false);
    }
  };

  // Execute a challenge using the Circle SDK
  const executeChallenge = async (challengeId: string) => {
    if (!client) {
      toast({
        title: 'SDK Not Ready',
        description: 'Circle SDK is not initialized',
        variant: 'destructive',
      });
      return;
    }

    try {
      setIsProcessing(true);
      setError(null);

      // Execute the challenge - this will show the Circle UI for PIN setup
      await new Promise<void>((resolve, reject) => {
        client.execute(challengeId, (error) => {
          if (error) {
            logger.error('PIN setup failed:', error);
            // Handle specific error codes
            if (error.code === 155703) {
              reject(new Error('PINs do not match. Please try again with matching PINs.'));
            } else {
              reject(error);
            }
          } else {
            // Challenge completed successfully
            resolve();
          }
        });
      });

      // PIN setup was successful
      toast({
        title: 'PIN Setup Complete',
        description: "You'll now be able to create Circle wallets",
      });

      // Call the complete callback
      onComplete?.();

      // Close the dialog
      onOpenChange(false);
    } catch (error) {
      logger.error('Error executing challenge:', error);
      let errorMsg: string;

      // Handle specific error cases
      if (error instanceof Error) {
        errorMsg = error.message;
      } else if (typeof error === 'object' && error !== null) {
        const err = error as { code?: number; message?: string };
        if (err.code === 155703) {
          errorMsg = 'PINs do not match. Please try again with matching PINs.';
        } else {
          errorMsg = err.message || 'There was an error during PIN setup';
        }
      } else {
        errorMsg = 'There was an error during PIN setup';
      }

      setError(errorMsg);
      toast({
        title: 'PIN Setup Failed',
        description: errorMsg,
        variant: 'destructive',
      });

      // Create a new challenge for retry
      if (error instanceof Error && error.message.includes('PINs do not match')) {
        handleRetry();
      }
    } finally {
      setIsProcessing(false);
    }
  };

  // Retry creating the challenge
  const handleRetry = () => {
    setError(null);
    createPinChallenge();
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Set Up Circle Wallet PIN</DialogTitle>
          <DialogDescription>
            You need to set up a secure PIN to protect your Circle wallet and transactions.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <Alert variant="destructive" className="mt-2">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="py-4">
          <p className="text-sm text-muted-foreground mb-4">Your PIN should be:</p>
          <ul className="list-disc pl-5 text-sm space-y-1 text-muted-foreground">
            <li>At least 6 digits long</li>
            <li>Not easily guessable (avoid sequential numbers or repeating digits)</li>
            <li>Kept private and not shared with others</li>
          </ul>
        </div>

        <DialogFooter className="flex flex-col sm:flex-row gap-2 sm:gap-0">
          {error && (
            <Button variant="outline" onClick={handleRetry} disabled={isProcessing}>
              Retry
            </Button>
          )}
          <Button
            onClick={() => challengeId && executeChallenge(challengeId)}
            disabled={isProcessing || !challengeId}
          >
            {isProcessing ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Processing...
              </>
            ) : (
              'Continue'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
