'use client';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { Check, Wallet, Plus, Copy } from 'lucide-react';
import type { WalletResponse } from '@/types/wallet';
import { logger } from '@/lib/core/logger/logger-utils';
interface WalletCardSelectorProps {
  wallets: WalletResponse[];
  selectedWalletId?: string;
  onWalletSelect: (walletId: string) => void;
  onCreateWallet?: () => void;
  disabled?: boolean;
  className?: string;
}

export function WalletCardSelector({
  wallets,
  selectedWalletId,
  onWalletSelect,
  onCreateWallet,
  disabled = false,
  className,
}: WalletCardSelectorProps) {
  const copyToClipboard = async (address: string, e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent wallet selection when copying
    try {
      await navigator.clipboard.writeText(address);
      // You could add a toast notification here
    } catch (err) {
      logger.error('Failed to copy address:', err);
    }
  };

  const truncateAddress = (address: string) => {
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  return (
    <div className={cn('space-y-4', className)}>
      <div className="flex items-center justify-between">
        <h4 className="font-medium text-sm">Choose Receiving Wallet</h4>
        {onCreateWallet && (
          <Button
            variant="outline"
            size="sm"
            onClick={onCreateWallet}
            disabled={disabled}
            className="flex items-center gap-2 text-xs"
          >
            <Plus className="h-3 w-3" />
            Create Wallet
          </Button>
        )}
      </div>

      <div className="space-y-3">
        {wallets.length === 0 ? (
          <Card className="border-dashed border-2">
            <CardContent className="p-6 text-center">
              <Wallet className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
              <p className="text-sm text-muted-foreground mb-3">
                No wallets found. Create your first wallet to receive payments.
              </p>
              {onCreateWallet && (
                <Button
                  variant="outline"
                  onClick={onCreateWallet}
                  disabled={disabled}
                  className="flex items-center gap-2"
                >
                  <Plus className="h-4 w-4" />
                  Create Your First Wallet
                </Button>
              )}
            </CardContent>
          </Card>
        ) : (
          wallets.map((wallet) => (
            <Card
              key={wallet.id}
              className={cn(
                'cursor-pointer transition-all duration-200 hover:shadow-md',
                selectedWalletId === wallet.id
                  ? 'border-purple-500 bg-purple-50 dark:bg-purple-950'
                  : 'hover:border-purple-200'
              )}
              onClick={() => !disabled && onWalletSelect(wallet.id)}
            >
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-purple-400 to-purple-600 flex items-center justify-center">
                      <Wallet className="h-5 w-5 text-white" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <h5 className="font-medium truncate">
                          {wallet.nickname || `Wallet ${wallet.id}`}
                        </h5>
                        {wallet.wallet_type && (
                          <Badge variant="secondary" className="text-xs">
                            {wallet.wallet_type}
                          </Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        <p className="text-sm text-muted-foreground font-mono">
                          {truncateAddress(wallet.wallet_address)}
                        </p>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-5 w-5 p-0 opacity-60 hover:opacity-100"
                          onClick={(e) => copyToClipboard(wallet.wallet_address, e)}
                        >
                          <Copy className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {selectedWalletId === wallet.id && (
                      <div className="w-6 h-6 rounded-full bg-purple-600 flex items-center justify-center">
                        <Check className="h-4 w-4 text-white" />
                      </div>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}
