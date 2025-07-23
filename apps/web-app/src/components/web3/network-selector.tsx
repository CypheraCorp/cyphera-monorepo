'use client';

import * as React from 'react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useAccount, useSwitchChain } from 'wagmi';
import { useNetworkStore } from '@/store/network';
import { Check, ChevronsUpDown, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

export function NetworkSelector() {
  const { isConnected } = useAccount();
  const currentNetwork = useNetworkStore((state) => state.currentNetwork); // Get current network from store
  const { chains, switchChain, isPending, error } = useSwitchChain(); // Wagmi hook for switching

  const handleSwitchNetwork = (chainId: number) => {
    if (switchChain) {
      switchChain({ chainId });
    }
  };

  // Determine button text
  const buttonText = currentNetwork
    ? currentNetwork.network.name
    : isConnected
      ? 'Unsupported Network'
      : 'Connect Wallet';

  // Disable if not connected or switching
  const isDisabled = !isConnected || isPending;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          disabled={isDisabled}
          className="w-[200px] justify-between"
        >
          {isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <span className="truncate">{buttonText}</span>
          )}
          {!isPending && <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-[200px] p-0">
        <DropdownMenuLabel>Select Network</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {chains.map((chain) => (
          <DropdownMenuItem
            key={chain.id}
            disabled={isPending} // Disable item while pending
            onClick={() => handleSwitchNetwork(chain.id)}
          >
            <Check
              className={cn(
                'mr-2 h-4 w-4',
                currentNetwork?.network.chain_id === chain.id ? 'opacity-100' : 'opacity-0'
              )}
            />
            {chain.name}
            {/* Optionally add (Testnet) indicator */}
            {/* {chain.testnet && <span className="ml-auto text-xs opacity-60">Testnet</span>} */}
          </DropdownMenuItem>
        ))}
        {error && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuLabel className="text-destructive text-xs px-2 py-1.5">
              {error.message}
            </DropdownMenuLabel>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
