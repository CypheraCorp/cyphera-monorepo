'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Loader2, AlertTriangle, KeyRound } from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import { useCircleSDK } from '@/hooks/web3';
import { v4 as uuidv4 } from 'uuid';
import { CircleAPI } from '@/services/cyphera-api/circle';
import { WalletResponse } from '@/types/wallet';
import type { UserRequestContext } from '@/services/cyphera-api/api';
import { logger } from '@/lib/core/logger/logger-utils';

interface CircleWalletRecoveryProps {
  wallet: WalletResponse;
  workspaceId: string;
  onRecoveryComplete?: () => void;
}

/**
 * CircleWalletRecovery component
 *
 * Allows users to recover access to their Circle wallet by resetting their PIN
 */
export function CircleWalletRecovery({
  wallet,
  workspaceId,
  onRecoveryComplete,
}: CircleWalletRecoveryProps) {
  const [open, setOpen] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const { client, userToken } = useCircleSDK();
  const { toast } = useToast();

  // Initiate PIN recovery process
  const handleRecoverPin = async () => {
    if (!client) {
      toast({
        title: 'SDK Not Ready',
        description: 'Circle SDK is not initialized',
        variant: 'destructive',
      });
      return;
    }

    if (!wallet.circle_data?.circle_user_id) {
      toast({
        title: 'Invalid Wallet',
        description: 'This wallet does not have Circle user data',
        variant: 'destructive',
      });
      return;
    }

    // Create context
    if (!userToken || !workspaceId) {
      toast({
        title: 'Error',
        description: 'Missing user token or workspace ID for recovery.',
        variant: 'destructive',
      });
      return;
    }
    const context: UserRequestContext = {
      access_token: userToken,
      workspace_id: workspaceId,
      // Add account_id/user_id if needed and available
    };

    try {
      setIsProcessing(true);
      const circleApi = new CircleAPI();

      // Create a unique idempotency key
      const idempotencyKey = uuidv4();

      // Call service method with context
      const challengeResponse = await circleApi.createPinRestoreChallenge(context, idempotencyKey);

      if (!challengeResponse || !challengeResponse.data || !challengeResponse.data.challenge) {
        throw new Error('Failed to create PIN restore challenge');
      }

      const challengeId = challengeResponse.data.challenge.id;

      // Execute the challenge with the Circle SDK
      await new Promise<void>((resolve, reject) => {
        client.execute(challengeId, (error) => {
          if (error) {
            logger.error('PIN recovery failed:', error);
            reject(error);
          } else {
            // Challenge completed successfully
            resolve();
          }
        });
      });

      // Recovery was successful
      toast({
        title: 'PIN Recovery Complete',
        description: 'Your PIN has been successfully reset',
      });

      setOpen(false);
      onRecoveryComplete?.();
    } catch (error) {
      logger.error('PIN recovery failed:', error);
      toast({
        title: 'Recovery Failed',
        description:
          error instanceof Error ? error.message : 'There was an error recovering your PIN',
        variant: 'destructive',
      });
    } finally {
      setIsProcessing(false);
    }
  };

  // Check if this is a Circle wallet
  const isCircleWallet = wallet.wallet_type === 'circle_wallet' || !!wallet.circle_data;

  if (!isCircleWallet) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2">
          <KeyRound className="h-4 w-4" />
          Recover PIN
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Recover Wallet PIN</DialogTitle>
          <DialogDescription>
            Reset your wallet PIN using your security questions.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <div className="rounded-md bg-amber-50 p-4 text-amber-800 dark:bg-amber-900/20 dark:text-amber-200">
            <div className="flex items-start gap-2">
              <AlertTriangle className="h-5 w-5 mt-0.5" />
              <div>
                <p className="font-medium">Important</p>
                <p className="text-sm mt-1">
                  If you&apos;ve forgotten your PIN, you can recover access by answering your
                  security questions. You&apos;ll be asked to create a new PIN after successfully
                  completing this process.
                </p>
              </div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button onClick={handleRecoverPin} disabled={isProcessing}>
            {isProcessing ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Processing...
              </>
            ) : (
              'Begin Recovery'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
