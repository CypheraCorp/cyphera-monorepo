'use client';

import { useState, useMemo, Suspense } from 'react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button';
import { useRouter } from 'next/navigation';
import type { NetworkWithTokensResponse } from '@/types/network';
import dynamic from 'next/dynamic';

// AddWalletDialog was removed - functionality moved elsewhere

const CreateCircleWalletDialog = dynamic(
  () =>
    import('./create-circle-wallet-dialog').then((mod) => ({
      default: mod.CreateCircleWalletDialog,
    })),
  {
    loading: () => null,
    ssr: false,
  }
);

// Dynamically import lucide icons
const MoreVertical = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.MoreVertical })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded" />,
    ssr: false,
  }
);

const CircleIcon = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.CircleIcon })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded" />,
    ssr: false,
  }
);

// WalletIcon removed as Add Existing Wallet functionality is disabled

// Define props interface
interface WalletActionsProps {
  networks: NetworkWithTokensResponse[];
}

/**
 * WalletActions component
 *
 * Provides a dropdown menu for wallet management actions including:
 * - Connecting an existing wallet
 * - Creating a Circle wallet
 */
export function WalletActions({ networks }: WalletActionsProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [showCreateCircle, setShowCreateCircle] = useState(false);
  const router = useRouter();

  // Handle wallet creation callback
  const handleWalletCreated = async () => {
    router.refresh();
    setTimeout(() => {
      window.location.reload();
    }, 200);
  };

  // Filter networks for Circle compatibility
  const circleCompatibleNetworks = useMemo(() => {
    return networks.filter((network) => !!network.network.circle_network_type);
  }, [networks]);

  return (
    <div className="flex justify-end items-center">
      <DropdownMenu open={isOpen} onOpenChange={setIsOpen}>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="icon">
            <Suspense fallback={<div className="h-4 w-4 bg-muted animate-pulse rounded" />}>
              <MoreVertical className="h-4 w-4" />
            </Suspense>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>Add Wallet</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {/* Add Existing Wallet functionality temporarily disabled */}
          <DropdownMenuItem
            onClick={() => {
              setShowCreateCircle(true);
              setIsOpen(false);
            }}
            className="flex items-center cursor-pointer"
          >
            <Suspense fallback={<div className="mr-2 h-4 w-4 bg-muted animate-pulse rounded" />}>
              <CircleIcon className="mr-2 h-4 w-4" />
            </Suspense>
            Create Circle Wallet
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* AddWalletDialog removed - connect wallet functionality moved elsewhere */}

      {showCreateCircle && (
        <Suspense fallback={null}>
          <CreateCircleWalletDialog
            isOpen={showCreateCircle}
            onOpenChange={setShowCreateCircle}
            onWalletCreated={handleWalletCreated}
            networks={circleCompatibleNetworks}
          />
        </Suspense>
      )}
    </div>
  );
}
