'use client';

import { useState, useEffect, useMemo, Suspense } from 'react';

import { DeleteWalletButton } from '@/components/wallets/delete-wallet-button';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import type { WalletResponse } from '@/types/wallet';
import { generateExplorerLink } from '@/lib/utils/explorers';
import { NetworkWithTokensResponse } from '@/types/network';
import dynamic from 'next/dynamic';

// Dynamically import lucide icon
const ExternalLink = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.ExternalLink })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded" />,
    ssr: false,
  }
);

interface WalletListProps {
  initialWallets: WalletResponse[];
  networks: NetworkWithTokensResponse[];
}

/**
 * WalletList component
 * Displays a list of added wallets with their details and actions using dynamic network data.
 */
export function WalletList({ initialWallets, networks: initialNetworks }: WalletListProps) {
  const [wallets, setWallets] = useState<WalletResponse[]>(initialWallets);

  useEffect(() => {
    setWallets(initialWallets);
  }, [initialWallets]);

  // Memoize the networks map for performance
  const networksMap = useMemo(() => {
    return initialNetworks.reduce<Record<string, NetworkWithTokensResponse>>((acc, network) => {
      acc[network.network.id] = network;
      return acc;
    }, {});
  }, [initialNetworks]);

  // Group wallets by address
  const groupedWallets = useMemo(() => {
    return wallets.reduce<Record<string, WalletResponse[]>>((acc, wallet) => {
      const address = wallet.wallet_address;
      if (!acc[address]) {
        acc[address] = [];
      }
      acc[address].push(wallet);
      return acc;
    }, {});
  }, [wallets]);

  const handleWalletDeleted = (deletedWalletId: string) => {
    setWallets((currentWallets) =>
      currentWallets.filter((wallet) => wallet.id !== deletedWalletId)
    );
  };

  // Check if a wallet is a Circle wallet
  const isCircleWallet = (wallet: WalletResponse): boolean => {
    return wallet.wallet_type === 'circle_wallet' || !!wallet.circle_data;
  };

  return (
    <div className="space-y-4">
      {Object.keys(groupedWallets).length === 0 ? (
        <div className="p-4 text-center text-muted-foreground">
          No wallets added yet. Click the + button above to add a wallet
        </div>
      ) : (
        Object.entries(groupedWallets).map(([address, walletGroup]) => {
          // Use the first wallet for display information
          const primaryWallet = walletGroup[0];

          // Get supported networks for this wallet group
          const supportedNetworks = walletGroup.map((wallet) => {
            const network = wallet.network_id ? networksMap[wallet.network_id] : null;
            return {
              wallet,
              network,
              networkName: network?.network.name || 'Unknown Network',
            };
          });

          // Generate explorer link using the first wallet's network
          const firstNetwork = supportedNetworks[0]?.network;
          const explorerLink = generateExplorerLink(
            initialNetworks,
            firstNetwork?.network.chain_id,
            'address',
            address
          );

          return (
            <div key={address} className="flex items-center justify-between p-4 border rounded-lg">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <div className="font-medium">
                    {primaryWallet.nickname || 'Unnamed Wallet'}
                    {isCircleWallet(primaryWallet) && (
                      <Badge
                        variant="outline"
                        className="ml-2 bg-[#B4EFE8] text-[#00645A] border-[#B4EFE8]"
                      >
                        Circle Wallet
                      </Badge>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <code className="text-sm bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded">
                    {address}
                  </code>
                  {/* Use the generated explorerLink */}
                  {explorerLink && (
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <a
                            href={explorerLink}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-500 hover:text-blue-700"
                          >
                            <Suspense
                              fallback={<div className="h-4 w-4 bg-muted animate-pulse rounded" />}
                            >
                              <ExternalLink className="h-4 w-4" />
                            </Suspense>
                          </a>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>View on block explorer</p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  )}
                </div>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  {primaryWallet.ens && `ENS: ${primaryWallet.ens}`}
                  {primaryWallet.last_used_at &&
                    `${primaryWallet.ens ? ' • ' : ''}Last used: ${new Date(primaryWallet.last_used_at * 1000).toLocaleDateString()}`}
                </div>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <span>Blockchain:</span>
                  <div className="flex flex-wrap gap-1">
                    {supportedNetworks.map(({ networkName }, index) => (
                      <Badge key={index} variant="secondary" className="text-xs">
                        {networkName}
                      </Badge>
                    ))}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {!isCircleWallet(primaryWallet) && (
                  <DeleteWalletButton
                    walletId={primaryWallet.id}
                    onDeleted={() => handleWalletDeleted(primaryWallet.id)}
                  />
                )}
              </div>
            </div>
          );
        })
      )}
    </div>
  );
}
