'use client';

import { Button } from '@/components/ui/button';
import { useAccount, useConnect, useDisconnect } from 'wagmi';
import { Loader2, AlertTriangle } from 'lucide-react';
import { useState, useEffect } from 'react';
import { useToast } from '@/components/ui/use-toast';
import { LucideIcon } from 'lucide-react';
import { formatAddress } from '@/lib/utils/circle';
import { logger } from '@/lib/core/logger/logger-utils';

interface ClientOnlyIconProps {
  icon: LucideIcon;
  className?: string;
  [key: string]: unknown;
}

// Client-only icon component to prevent hydration mismatches
function ClientOnlyIcon({ icon: Icon, ...props }: ClientOnlyIconProps) {
  const [isMounted, setIsMounted] = useState(false);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  if (!isMounted) {
    return <span className={props.className || ''} />;
  }

  return <Icon {...props} />;
}

export function ConnectWalletButton() {
  const [mounted, setMounted] = useState(false);
  const { address, isConnecting, isConnected } = useAccount();
  const { connect, connectors } = useConnect();
  const { disconnect } = useDisconnect();
  const [isMetaMaskInstalled, setIsMetaMaskInstalled] = useState(false);
  const [hasPendingRequest, setHasPendingRequest] = useState(false);
  const { toast } = useToast();
  const [isInNavbar, setIsInNavbar] = useState(false);

  // Check if MetaMask is installed on mount and if button is in navbar
  useEffect(() => {
    setMounted(true);

    const checkMetaMask =
      typeof window !== 'undefined' &&
      typeof window.ethereum !== 'undefined' &&
      window.ethereum.isMetaMask;

    setIsMetaMaskInstalled(checkMetaMask);

    // Check if parent has inline-block class (navbar)
    if (typeof document !== 'undefined') {
      const buttonElement = document.querySelector('[data-connect-wallet-button]');
      if (buttonElement) {
        const parent = buttonElement.parentElement;
        if (parent && parent.classList.contains('inline-block')) {
          setIsInNavbar(true);
        }
      }
    }
  }, []);

  // Get button class based on location
  const getButtonClass = () => {
    return isInNavbar ? 'w-full text-center' : 'w-full';
  };

  // Handle connection
  const handleConnect = async () => {
    if (isConnecting) {
      // If already connecting, show a message
      toast({
        title: 'Connection in progress',
        description: 'Please check your MetaMask extension for pending requests.',
      });
      return;
    }

    if (hasPendingRequest) {
      toast({
        title: 'Request Already Pending',
        description: 'Please open MetaMask and respond to the existing connection request.',
        variant: 'destructive',
      });
      return;
    }

    try {
      // Find the MetaMask connector
      const metaMaskConnector = connectors.find(
        (c) => c.name === 'MetaMask' || c.id === 'metaMask'
      );

      if (!metaMaskConnector) {
        toast({
          title: 'Connection Error',
          description: 'MetaMask connector not found. Please refresh the page and try again.',
          variant: 'destructive',
        });
        return;
      }

      // Try to detect if there's a pending request first
      if (window.ethereum && window.ethereum.isMetaMask) {
        try {
          // This is a quick way to check if there's a pending request
          // It will fail with code -32002 if there's already a pending request
          await window.ethereum.request({ method: 'eth_accounts' });
        } catch (error: unknown) {
          const ethError = error as { code?: number };
          if (ethError.code === -32002) {
            setHasPendingRequest(true);
            toast({
              title: 'MetaMask Request Pending',
              description:
                "There's already a connection request pending. Please open MetaMask and respond to it.",
              variant: 'destructive',
            });
            return;
          }
        }
      }

      // Connect using the MetaMask connector
      connect({ connector: metaMaskConnector });
    } catch (error: unknown) {
      logger.error('Connection error:', error);

      // Check for the specific "already pending" error
      const ethError = error as { code?: number; message?: string };
      if (
        ethError.code === -32002 ||
        (ethError.message && ethError.message.includes('already pending'))
      ) {
        setHasPendingRequest(true);
        toast({
          title: 'Request Already Pending',
          description: 'Please open MetaMask and respond to the existing connection request.',
          variant: 'destructive',
        });
      } else {
        toast({
          title: 'Connection Failed',
          description: 'Failed to connect to MetaMask. Please try again.',
          variant: 'destructive',
        });
      }
    }
  };

  // Handle disconnection
  const handleDisconnect = () => {
    disconnect();
    setHasPendingRequest(false);
  };

  // During SSR and initial client-side render, return a loading state
  if (!mounted) {
    return (
      <Button
        variant="outline"
        className={getButtonClass()}
        suppressHydrationWarning
        data-connect-wallet-button
      >
        Loading...
      </Button>
    );
  }

  if (!isMetaMaskInstalled) {
    return (
      <Button
        variant="outline"
        className={getButtonClass()}
        onClick={() => window.open('https://metamask.io/download/', '_blank')}
        suppressHydrationWarning
        data-connect-wallet-button
      >
        Install MetaMask
      </Button>
    );
  }

  if (hasPendingRequest) {
    return (
      <div className="space-y-2">
        <Button
          variant="outline"
          disabled
          className={getButtonClass()}
          suppressHydrationWarning
          data-connect-wallet-button
        >
          {mounted && (
            <ClientOnlyIcon icon={AlertTriangle} className="mr-2 h-4 w-4 text-yellow-500" />
          )}
          Request Pending
        </Button>
        <p className="text-xs text-muted-foreground text-center" suppressHydrationWarning>
          Open MetaMask to approve the connection request
        </p>
      </div>
    );
  }

  if (isConnecting) {
    return (
      <Button
        variant="outline"
        disabled
        className={getButtonClass()}
        suppressHydrationWarning
        data-connect-wallet-button
      >
        {mounted && <ClientOnlyIcon icon={Loader2} className="mr-2 h-4 w-4 animate-spin" />}
        Connecting...
      </Button>
    );
  }

  if (isConnected && address) {
    return (
      <Button
        variant="outline"
        onClick={handleDisconnect}
        className={getButtonClass()}
        suppressHydrationWarning
        data-connect-wallet-button
      >
        <span className="truncate" suppressHydrationWarning>
          {formatAddress(address)}
        </span>
      </Button>
    );
  }

  return (
    <Button
      variant="default"
      onClick={handleConnect}
      disabled={isConnecting}
      className={getButtonClass()}
      suppressHydrationWarning
      data-connect-wallet-button
    >
      {isConnecting ? (
        <>
          {mounted && <ClientOnlyIcon icon={Loader2} className="mr-2 h-4 w-4 animate-spin" />}
          Connecting...
        </>
      ) : (
        'Connect Wallet'
      )}
    </Button>
  );
}
