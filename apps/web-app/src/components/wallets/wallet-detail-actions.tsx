'use client';

import { RefreshCw } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { useRouter } from 'next/navigation';
import { useToast } from '@/components/ui/use-toast';
import type { WalletResponse } from '@/types/wallet';

interface WalletDetailActionsProps {
  wallet: WalletResponse;
}

/**
 * Client component for wallet detail actions like refresh and copy
 */
export function WalletDetailActions({ wallet }: WalletDetailActionsProps) {
  const [isRefreshing, setIsRefreshing] = useState(false);
  const router = useRouter();
  const { toast } = useToast();

  const handleRefresh = () => {
    setIsRefreshing(true);
    // Refresh the current page data
    router.refresh();

    // Reset the refreshing state after a delay
    setTimeout(() => {
      setIsRefreshing(false);
    }, 1000);
  };

  const handleCopyAddress = () => {
    navigator.clipboard.writeText(wallet.wallet_address);
    toast({
      title: 'Address Copied',
      description: 'Wallet address copied to clipboard',
    });
  };

  return (
    <div className="flex gap-2">
      <Button
        variant="outline"
        size="sm"
        onClick={handleRefresh}
        disabled={isRefreshing}
        className="gap-2"
      >
        <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
        Refresh
      </Button>

      <Button variant="outline" size="sm" onClick={handleCopyAddress}>
        Copy Address
      </Button>
    </div>
  );
}
