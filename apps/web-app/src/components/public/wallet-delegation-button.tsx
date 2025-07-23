'use client';

import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { useAccount } from 'wagmi';
import { Loader2, Shield } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useToast } from '@/components/ui/use-toast';
import { useSmartAccount } from '@/hooks/store/use-wallet-sync';
import { createAndSignDelegation, formatDelegation } from '@/lib/web3/utils/delegation';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { useEnvConfig } from '@/components/env/client';
import { logger } from '@/lib/core/logger/logger-utils';

// Client-only icon component to prevent hydration mismatches
interface ClientOnlyIconProps {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  className?: string;
  [key: string]: unknown;
}

function ClientOnlyIcon({ icon: Icon, className, ...props }: ClientOnlyIconProps) {
  const [isMounted, setIsMounted] = useState(false);
  useEffect(() => {
    setIsMounted(true);
  }, []);

  if (!isMounted) {
    return <span className={className || ''} />;
  }

  return <Icon className={className} {...(props as React.SVGProps<SVGSVGElement>)} />;
}

interface WalletDelegationButtonProps {
  disabled?: boolean;
}

// Get delegate address from env or API
async function getCypheraDelegateAddress(envDelegateAddress?: string): Promise<`0x${string}`> {
  try {
    // First check if we have the address from our environment config
    if (envDelegateAddress?.startsWith('0x')) return envDelegateAddress as `0x${string}`;

    // Fallback to API request if not available in config
    const response = await fetch('/api/config/delegate-address');
    const data = await response.json();
    if (!data.success || !data.address) throw new Error('Failed to get delegate address');
    return data.address as `0x${string}`;
  } catch (error) {
    logger.error('Error getting delegate address:', { error });
    throw new Error('Cyphera delegate address is not configured');
  }
}

export function WalletDelegationButton({ disabled = false }: WalletDelegationButtonProps) {
  const { isConnected } = useAccount();
  const { smartAccount, smartAccountAddress, isWalletCompatible, isMetaMask, createSmartAccount } =
    useSmartAccount();
  const envConfig = useEnvConfig();

  const [showDelegationDialog, setShowDelegationDialog] = useState(false);
  const [signedDelegation, setSignedDelegation] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);
  const [status, setStatus] = useState<'idle' | 'creating' | 'signing'>('idle');
  const { toast } = useToast();

  useEffect(() => setMounted(true), []);

  async function handleCreateDelegation() {
    if (status !== 'idle') {
      toast({
        title: 'Please wait',
        description: 'A request is already being processed.',
        variant: 'destructive',
      });
      return;
    }

    if (!isConnected) {
      toast({
        title: 'Wallet not connected',
        description: 'Please connect your wallet to create delegation.',
        variant: 'destructive',
      });
      return;
    }

    try {
      // Create smart account if needed
      if (!smartAccountAddress || !smartAccount) {
        setStatus('creating');
        await createSmartAccount();

        if (!smartAccountAddress || !smartAccount) {
          throw new Error('Failed to create smart account');
        }
      }

      // Create delegation
      setStatus('signing');
      const delegateAddress = await getCypheraDelegateAddress(envConfig.delegateAddress);
      const delegation = await createAndSignDelegation(smartAccount as Parameters<typeof createAndSignDelegation>[0], delegateAddress);
      const formattedDelegation = formatDelegation(delegation);
      setSignedDelegation(formattedDelegation);

      setShowDelegationDialog(true);
      toast({
        title: 'Delegation created!',
        description: 'You have successfully created a delegation for your smart account.',
      });
    } catch (error) {
      logger.error('Delegation error:', { error });
      toast({
        title: 'Delegation failed',
        description: error instanceof Error ? error.message : 'An unexpected error occurred',
        variant: 'destructive',
      });
    } finally {
      setStatus('idle');
    }
  }

  const buttonText = !isConnected
    ? 'Connect Wallet'
    : !isMetaMask
      ? 'MetaMask Required'
      : !isWalletCompatible
        ? 'Wallet Not Compatible'
        : 'Create Delegation';

  const tooltipText = !isConnected
    ? 'Connect your wallet to create delegation'
    : !isMetaMask
      ? 'MetaMask is required for delegation'
      : !isWalletCompatible
        ? 'Your wallet does not support message signing'
        : !smartAccountAddress
          ? 'Create a smart account first'
          : !smartAccount
            ? 'Smart account object not available'
            : status !== 'idle'
              ? 'Creating delegation...'
              : 'Create delegation for your smart account';

  return (
    <>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex w-full">
              <Button
                onClick={handleCreateDelegation}
                disabled={disabled || status !== 'idle'}
                variant="default"
                className="w-full"
                suppressHydrationWarning
              >
                {status !== 'idle' ? (
                  <>
                    {mounted && (
                      <ClientOnlyIcon icon={Loader2} className="mr-2 h-4 w-4 animate-spin" />
                    )}
                    <span suppressHydrationWarning>
                      {status === 'creating'
                        ? 'Creating Smart Account...'
                        : 'Waiting for MetaMask...'}
                    </span>
                  </>
                ) : (
                  <>
                    <Shield className="mr-2 h-4 w-4" />
                    <span suppressHydrationWarning>{buttonText}</span>
                  </>
                )}
              </Button>
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <p>{tooltipText}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      {signedDelegation && (
        <Dialog open={showDelegationDialog} onOpenChange={setShowDelegationDialog}>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Delegation Created</DialogTitle>
              <DialogDescription>
                Your delegation has been created successfully. You can share this delegation to
                allow others to perform actions on behalf of your smart account.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="p-4 bg-muted rounded-lg">
                <pre className="text-xs whitespace-pre-wrap break-all">{signedDelegation}</pre>
              </div>
              <Button
                onClick={() => {
                  navigator.clipboard.writeText(signedDelegation);
                  toast({
                    title: 'Copied!',
                    description: 'Delegation copied to clipboard',
                  });
                }}
                className="w-full"
              >
                Copy Delegation
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}
