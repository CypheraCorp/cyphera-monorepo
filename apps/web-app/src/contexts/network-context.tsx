'use client';

import React, { createContext, useContext, useMemo, ReactNode, useCallback } from 'react';
import { useAccount, useChainId, useSwitchChain } from 'wagmi';
import type { NetworkWithTokensResponse } from '@/types/network';
import type { Address } from 'viem';
import { logger } from '@/lib/core/logger/logger-utils';
// Define the type for the switchChain function more explicitly
type SwitchChainFunction = ReturnType<typeof useSwitchChain>['switchChain'];
type SwitchChainAsyncFunction = ReturnType<typeof useSwitchChain>['switchChainAsync'];

interface INetworkContext {
  networks: NetworkWithTokensResponse[]; // All configured networks
  currentNetwork: NetworkWithTokensResponse | null; // Network matching current chainId
  getUsdcContractAddress: () => Address | undefined; // Helper to find USDC on current network
  isWalletConnected: boolean; // Is a wallet connected?
  isSwitchingNetwork: boolean; // Is a network switch in progress?
  switchChain: SwitchChainFunction | undefined; // Function to switch network
  switchChainAsync: SwitchChainAsyncFunction | undefined; // Function to switch network async
}

const NetworkContext = createContext<INetworkContext | undefined>(undefined);

interface NetworkProviderProps {
  children: ReactNode;
  networks: NetworkWithTokensResponse[]; // Pass fetched networks here
}

export function NetworkProvider({ children, networks = [] }: NetworkProviderProps) {
  const { status: accountStatus } = useAccount(); // Get account status
  const { switchChain, switchChainAsync, status: switchChainStatus } = useSwitchChain(); // Get switchChain function and status
  const chainId = useChainId(); // Get the currently connected chain ID from Wagmi

  // Derive status flags
  const isWalletConnected = useMemo(() => accountStatus === 'connected', [accountStatus]);
  const isSwitchingNetwork = useMemo(() => switchChainStatus === 'pending', [switchChainStatus]);

  // Find the network config that matches the current chainId
  const currentNetwork = useMemo(() => {
    logger.log('🔍 [NetworkProvider] Current network lookup:', {
      chainId,
      networksCount: networks.length,
      availableChainIds: networks.map((n) => n.network.chain_id),
      accountStatus,
      isWalletConnected,
    });

    if (!chainId || networks.length === 0) {
      logger.log('🔍 [NetworkProvider] No chainId or networks available');
      return null;
    }

    const found = networks.find((network) => network.network.chain_id === chainId);
    logger.log('🔍 [NetworkProvider] Network found:', {
      chainId,
      found: !!found,
      foundNetwork: found?.network.name,
    });

    return found || null;
  }, [chainId, networks, accountStatus, isWalletConnected]);

  // Wrap the helper in useCallback
  const getUsdcContractAddress = useCallback((): Address | undefined => {
    if (!currentNetwork || !currentNetwork.tokens) {
      return undefined;
    }
    // Find token with symbol 'USDC' (case-insensitive)
    const usdcToken = currentNetwork.tokens.find((token) => token.symbol?.toUpperCase() === 'USDC');
    return usdcToken?.contract_address as Address | undefined;
  }, [currentNetwork]); // Depend on currentNetwork

  const value = useMemo(
    () => ({
      networks,
      currentNetwork,
      getUsdcContractAddress,
      isWalletConnected, // Add status flag
      isSwitchingNetwork, // Add status flag
      switchChain, // Add switch function
      switchChainAsync, // Add switch async function
    }),
    // Include the memoized function and new values in the dependencies
    [
      networks,
      currentNetwork,
      getUsdcContractAddress,
      isWalletConnected,
      isSwitchingNetwork,
      switchChain,
      switchChainAsync,
    ]
  );

  return <NetworkContext.Provider value={value}>{children}</NetworkContext.Provider>;
}

export function useNetworkContext(): INetworkContext {
  const context = useContext(NetworkContext);
  if (context === undefined) {
    throw new Error('useNetworkContext must be used within a NetworkProvider');
  }
  return context;
}
