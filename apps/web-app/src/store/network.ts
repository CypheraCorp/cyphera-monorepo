import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import type { NetworkWithTokensResponse } from '@/types/network';
import type { Address } from 'viem';

interface NetworkState {
  // Available networks
  networks: NetworkWithTokensResponse[];

  // Current network
  currentNetwork: NetworkWithTokensResponse | null;
  currentChainId: number | null;

  // Network switching state
  isSwitchingNetwork: boolean;
  switchError: Error | null;

  // Loading state
  isLoadingNetworks: boolean;
  networksError: Error | null;
}

interface NetworkActions {
  // Network management
  setNetworks: (networks: NetworkWithTokensResponse[]) => void;
  setCurrentChainId: (chainId: number | null) => void;

  // Network switching
  setSwitchingNetwork: (switching: boolean) => void;
  setSwitchError: (error: Error | null) => void;

  // Loading state
  setLoadingNetworks: (loading: boolean) => void;
  setNetworksError: (error: Error | null) => void;

  // Helper methods
  getUsdcContractAddress: () => Address | undefined;
  getNetworkByChainId: (chainId: number) => NetworkWithTokensResponse | null;

  // Reset
  resetNetworkState: () => void;
}

const initialState: NetworkState = {
  networks: [],
  currentNetwork: null,
  currentChainId: null,
  isSwitchingNetwork: false,
  switchError: null,
  isLoadingNetworks: false,
  networksError: null,
};

export const useNetworkStore = create<NetworkState & NetworkActions>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialState,

        // Network management
        setNetworks: (networks) => {
          const { currentChainId } = get();
          const currentNetwork = currentChainId
            ? networks.find((n) => n.network.chain_id === currentChainId) || null
            : null;
          set({ networks, currentNetwork });
        },

        setCurrentChainId: (chainId) => {
          const { networks } = get();
          const currentNetwork = chainId
            ? networks.find((n) => n.network.chain_id === chainId) || null
            : null;
          set({ currentChainId: chainId, currentNetwork });
        },

        // Network switching
        setSwitchingNetwork: (switching) => set({ isSwitchingNetwork: switching }),
        setSwitchError: (error) => set({ switchError: error }),

        // Loading state
        setLoadingNetworks: (loading) => set({ isLoadingNetworks: loading }),
        setNetworksError: (error) => set({ networksError: error }),

        // Helper methods
        getUsdcContractAddress: () => {
          const { currentNetwork } = get();
          if (!currentNetwork || !currentNetwork.tokens) {
            return undefined;
          }
          const usdcToken = currentNetwork.tokens.find(
            (token) => token.symbol?.toUpperCase() === 'USDC'
          );
          return usdcToken?.contract_address as Address | undefined;
        },

        getNetworkByChainId: (chainId) => {
          const { networks } = get();
          return networks.find((n) => n.network.chain_id === chainId) || null;
        },

        // Reset
        resetNetworkState: () => set(initialState),
      }),
      {
        name: 'network-storage',
        partialize: (state) => ({
          // Only persist networks and current chain ID
          networks: state.networks,
          currentChainId: state.currentChainId,
        }),
      }
    )
  )
);
