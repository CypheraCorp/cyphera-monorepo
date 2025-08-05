'use client';

import { useState, useEffect, useMemo, Suspense } from 'react';
import { useRouter } from 'next/navigation';
import { SendTransactionDialog } from '@/components/wallets/send-transaction-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { toast } from 'sonner';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import type { WalletResponse } from '@/types/wallet';
import { generateExplorerLink } from '@/lib/utils/explorers';
import { NetworkWithTokensResponse } from '@/types/network';
import dynamic from 'next/dynamic';

// Dynamically import lucide icons
const ExternalLink = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.ExternalLink })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded" />,
    ssr: false,
  }
);

const Send = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.Send })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded" />,
    ssr: false,
  }
);

const MoreVertical = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.MoreVertical })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded" />,
    ssr: false,
  }
);

const Eye = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.Eye })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded" />,
    ssr: false,
  }
);

const Trash2 = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.Trash2 })),
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
  const router = useRouter();
  const [wallets, setWallets] = useState<WalletResponse[]>(initialWallets);
  const [sendDialogOpen, setSendDialogOpen] = useState(false);
  const [selectedWallet, setSelectedWallet] = useState<WalletResponse | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [walletToDelete, setWalletToDelete] = useState<WalletResponse | null>(null);

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

  const handleDeleteWallet = async () => {
    if (!walletToDelete) return;
    
    try {
      // First, fetch CSRF token
      const csrfResponse = await fetch('/api/auth/csrf');
      if (!csrfResponse.ok) {
        throw new Error('Failed to fetch CSRF token');
      }
      const { csrfToken } = await csrfResponse.json();

      const response = await fetch(`/api/wallets/${walletToDelete.id}`, {
        method: 'DELETE',
        headers: {
          'X-CSRF-Token': csrfToken,
        },
      });

      if (!response.ok) {
        toast.error("There was an issue deleting the wallet. Please ensure it's not being used by any products.");
        return;
      }

      handleWalletDeleted(walletToDelete.id);
      setDeleteDialogOpen(false);
      setWalletToDelete(null);
      toast.success('Wallet deleted successfully');
    } catch (error) {
      console.error('Error deleting wallet:', error);
      toast.error('There was an issue deleting the wallet. Please try again.');
    }
  };

  // Check if a wallet is a Circle wallet
  const isCircleWallet = (wallet: WalletResponse): boolean => {
    return wallet.wallet_type === 'circle_wallet' || !!wallet.circle_data;
  };

  // Check if a wallet can be deleted (not Circle or web3auth)
  const canDeleteWallet = (wallet: WalletResponse): boolean => {
    return wallet.wallet_type !== 'circle_wallet' && 
           wallet.wallet_type !== 'circle' && 
           wallet.wallet_type !== 'web3auth' && 
           !wallet.circle_data;
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
            <div 
              key={address} 
              className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors cursor-pointer"
              onClick={(e) => {
                // Don't navigate if clicking on buttons or links
                if ((e.target as HTMLElement).closest('button, a')) return;
                router.push(`/merchants/wallets/address/${encodeURIComponent(address)}`);
              }}
            >
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
                    {primaryWallet.wallet_type === 'web3auth' && (
                      <Badge
                        variant="outline"
                        className="ml-2 bg-purple-100 text-purple-700 border-purple-200"
                      >
                        Web3Auth Wallet
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
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="icon" className="h-8 w-8">
                      <Suspense
                        fallback={<div className="h-4 w-4 bg-muted animate-pulse rounded" />}
                      >
                        <MoreVertical className="h-4 w-4" />
                      </Suspense>
                      <span className="sr-only">Open menu</span>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem
                      onClick={() => router.push(`/merchants/wallets/address/${encodeURIComponent(address)}`)}
                    >
                      <Eye className="mr-2 h-4 w-4" />
                      View Details
                    </DropdownMenuItem>
                    
                    {isCircleWallet(primaryWallet) && (
                      <>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => {
                            setSelectedWallet(primaryWallet);
                            setSendDialogOpen(true);
                          }}
                        >
                          <Send className="mr-2 h-4 w-4" />
                          Send Transaction
                        </DropdownMenuItem>
                      </>
                    )}
                    
                    {canDeleteWallet(primaryWallet) && (
                      <>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => {
                            setWalletToDelete(primaryWallet);
                            setDeleteDialogOpen(true);
                          }}
                          className="text-red-600"
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          Delete Wallet
                        </DropdownMenuItem>
                      </>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          );
        })
      )}
      
      {selectedWallet && sendDialogOpen && (
        <SendTransactionDialog
          open={sendDialogOpen}
          onOpenChange={setSendDialogOpen}
          wallet={selectedWallet}
          workspaceId={selectedWallet.workspace_id}
          networks={initialNetworks}
        />
      )}
      
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete this wallet.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteWallet}
              className="bg-red-600 hover:bg-red-700"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
