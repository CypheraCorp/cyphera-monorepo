import { useQuery } from '@tanstack/react-query';
import { useAccount } from 'wagmi';
import { useNetworkUIStore } from '@/store/network-ui';
import type { NetworkWithTokensResponse } from '@/types/network';
import type { Address } from 'viem';

/**
 * Hook for fetching available networks - always returns fresh data
 */
export function useNetworks() {
  return useQuery({
    queryKey: ['networks'],
    queryFn: async (): Promise<NetworkWithTokensResponse[]> => {
      const response = await fetch('/api/networks', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch networks');
      }
      
      return response.json();
    },
    staleTime: 5 * 60 * 1000, // Networks don't change often, cache for 5 minutes
    gcTime: 60 * 60 * 1000, // Keep in cache for 1 hour
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    retry: 3,
  });
}

/**
 * Hook for getting the current network based on wallet connection
 */
export function useCurrentNetwork() {
  const { chainId } = useAccount();
  const { data: networks } = useNetworks();
  
  const currentNetwork = networks?.find(
    (network) => network.network.chain_id === chainId
  ) || null;
  
  return {
    currentNetwork,
    currentChainId: chainId || null,
    isConnected: !!chainId,
  };
}

/**
 * Hook for getting USDC contract address on current network
 */
export function useUsdcAddress(): Address | undefined {
  const { currentNetwork } = useCurrentNetwork();
  
  if (!currentNetwork?.tokens) {
    return undefined;
  }
  
  const usdcToken = currentNetwork.tokens.find(
    (token) => token.symbol?.toUpperCase() === 'USDC'
  );
  
  return usdcToken?.contract_address as Address | undefined;
}

/**
 * Hook for getting a specific network by chain ID
 */
export function useNetwork(chainId: number | null) {
  const { data: networks } = useNetworks();
  
  return networks?.find(
    (network) => network.network.chain_id === chainId
  ) || null;
}

/**
 * Combined hook that provides network data and UI state
 * This is the main hook components should use
 */
export function useNetworksWithUI() {
  const networksQuery = useNetworks();
  const { currentNetwork, currentChainId, isConnected } = useCurrentNetwork();
  const uiStore = useNetworkUIStore();
  
  // Filter networks based on UI preferences
  const filteredNetworks = networksQuery.data?.filter((network) => {
    // Filter testnets
    if (!uiStore.showTestnets && (network.network as any).testnet) {
      return false;
    }
    
    // Filter deprecated networks
    if (!uiStore.showDeprecatedNetworks && (network.network as any).deprecated) {
      return false;
    }
    
    return true;
  }) || [];
  
  // Get preferred network
  const preferredNetwork = uiStore.preferredChainId
    ? networksQuery.data?.find(n => n.network.chain_id === uiStore.preferredChainId)
    : null;
  
  return {
    // Data
    networks: filteredNetworks,
    allNetworks: networksQuery.data || [],
    currentNetwork,
    currentChainId,
    preferredNetwork,
    
    // Loading states
    isLoading: networksQuery.isLoading,
    isError: networksQuery.isError,
    error: networksQuery.error,
    
    // Connection state
    isConnected,
    
    // UI State
    ...uiStore,
    
    // Helper functions
    getNetworkByChainId: (chainId: number) => 
      networksQuery.data?.find(n => n.network.chain_id === chainId) || null,
    
    getUsdcAddress: (chainId?: number) => {
      const network = chainId 
        ? networksQuery.data?.find(n => n.network.chain_id === chainId)
        : currentNetwork;
      
      const usdcToken = network?.tokens?.find(
        (token) => token.symbol?.toUpperCase() === 'USDC'
      );
      
      return usdcToken?.contract_address as Address | undefined;
    },
    
    // Actions
    refetch: networksQuery.refetch,
  };
}

/**
 * Hook for supported networks for a specific product
 */
export function useProductNetworks(productId: string | null) {
  return useQuery({
    queryKey: ['products', productId, 'networks'],
    queryFn: async () => {
      if (!productId) throw new Error('No product ID provided');
      
      const response = await fetch(`/api/products/${productId}/networks`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch product networks');
      }
      
      return response.json() as Promise<NetworkWithTokensResponse[]>;
    },
    enabled: !!productId,
    staleTime: 5 * 60 * 1000,
    gcTime: 60 * 60 * 1000,
  });
}