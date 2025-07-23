'use client';

import { useEffect, ReactNode } from 'react';
import { useNetworkStore } from '@/store/network';
import { useChainId, useSwitchChain } from 'wagmi';
import type { NetworkWithTokensResponse } from '@/types/network';

interface NetworkSyncProviderProps {
  children: ReactNode;
  networks: NetworkWithTokensResponse[];
}

/**
 * Provider that syncs network data with Zustand store
 * This handles network data from API and chain ID from wagmi
 */
export function NetworkSyncProvider({ children, networks }: NetworkSyncProviderProps) {
  const chainId = useChainId();
  const { status: switchChainStatus } = useSwitchChain();

  const setNetworks = useNetworkStore((state) => state.setNetworks);
  const setCurrentChainId = useNetworkStore((state) => state.setCurrentChainId);
  const setSwitchingNetwork = useNetworkStore((state) => state.setSwitchingNetwork);

  // Sync networks from props
  useEffect(() => {
    if (networks && networks.length > 0) {
      setNetworks(networks);
    }
  }, [networks, setNetworks]);

  // Sync chain ID from wagmi
  useEffect(() => {
    setCurrentChainId(chainId || null);
  }, [chainId, setCurrentChainId]);

  // Sync switching status
  useEffect(() => {
    setSwitchingNetwork(switchChainStatus === 'pending');
  }, [switchChainStatus, setSwitchingNetwork]);

  return <>{children}</>;
}
