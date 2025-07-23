import { useEffect } from 'react';
import { useNetworkStore } from '@/store/network';
import { useChainId, useSwitchChain } from 'wagmi';
import type { NetworkWithTokensResponse } from '@/types/network';

/**
 * Hook that provides network functionality using Zustand store
 * Drop-in replacement for useNetworkContext
 */
export function useNetwork() {
  const chainId = useChainId();
  const { switchChain, switchChainAsync, status: switchChainStatus } = useSwitchChain();

  // Get state from store
  const networks = useNetworkStore((state) => state.networks);
  const currentNetwork = useNetworkStore((state) => state.currentNetwork);
  const isSwitchingNetwork = useNetworkStore((state) => state.isSwitchingNetwork);
  const getUsdcContractAddress = useNetworkStore((state) => state.getUsdcContractAddress);

  // Actions
  const setCurrentChainId = useNetworkStore((state) => state.setCurrentChainId);
  const setSwitchingNetwork = useNetworkStore((state) => state.setSwitchingNetwork);

  // Sync chain ID with store
  useEffect(() => {
    setCurrentChainId(chainId || null);
  }, [chainId, setCurrentChainId]);

  // Sync switching status
  useEffect(() => {
    setSwitchingNetwork(switchChainStatus === 'pending');
  }, [switchChainStatus, setSwitchingNetwork]);

  return {
    networks,
    currentNetwork,
    getUsdcContractAddress,
    isWalletConnected: !!chainId,
    isSwitchingNetwork,
    switchChain,
    switchChainAsync,
  };
}

/**
 * Hook to sync network data from API to store
 */
export function useNetworkSync(networksData?: NetworkWithTokensResponse[]) {
  const setNetworks = useNetworkStore((state) => state.setNetworks);

  useEffect(() => {
    if (networksData && networksData.length > 0) {
      setNetworks(networksData);
    }
  }, [networksData, setNetworks]);
}
