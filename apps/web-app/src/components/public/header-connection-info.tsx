'use client';

import { useAccount, useSwitchChain } from 'wagmi';
import { useNetworkStore } from '@/store/network';
import dynamic from 'next/dynamic';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button'; // Use Button for the trigger if Badge is not suitable
import { Loader2, Check } from 'lucide-react';
import { logger } from '@/lib/core/logger/logger-utils';
// Dynamically import the ConnectWalletButton with SSR disabled
const ConnectWalletButton = dynamic(
  () => import('@/components/public/connect-wallet-button').then((mod) => mod.ConnectWalletButton),
  { ssr: false }
);

export function HeaderConnectionInfo() {
  const { isConnected } = useAccount();
  const currentNetwork = useNetworkStore((state) => state.currentNetwork);
  const networks = useNetworkStore((state) => state.networks);
  const isSwitchingNetwork = useNetworkStore((state) => state.isSwitchingNetwork);
  const { switchChainAsync } = useSwitchChain();

  const handleNetworkSwitch = async (chainId: number) => {
    if (!switchChainAsync) return;
    try {
      await switchChainAsync({ chainId });
    } catch (error) {
      logger.error('Network switch failed:', error);
      // Optionally add a toast notification here for the user
    }
  };

  return (
    <div className="flex items-center gap-2">
      {isConnected && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" className="flex items-center gap-1.5">
              {isSwitchingNetwork && <Loader2 className="h-4 w-4 animate-spin" />}
              <span>{currentNetwork?.network.name || 'Select Network'}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>Available Networks</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {networks && networks.length > 0 ? (
              networks.map((network) => (
                <DropdownMenuItem
                  key={network.network.id}
                  disabled={
                    isSwitchingNetwork ||
                    currentNetwork?.network.chain_id === network.network.chain_id
                  }
                  onSelect={() => handleNetworkSwitch(Number(network.network.chain_id))}
                  className="flex items-center justify-between"
                >
                  <span>{network.network.name}</span>
                  {currentNetwork?.network.chain_id === network.network.chain_id && (
                    <Check className="h-4 w-4 text-green-500" />
                  )}
                </DropdownMenuItem>
              ))
            ) : (
              <DropdownMenuItem disabled>No networks available</DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
      <div className="inline-block w-48">
        {' '}
        {/* Maintain width for button */}
        <ConnectWalletButton />
      </div>
    </div>
  );
}
