import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

/**
 * Network UI Store - Only UI state and preferences
 * Actual network data comes from React Query
 */
interface NetworkUIState {
  // User's preferred network (persisted)
  preferredChainId: number | null;
  
  // UI State
  isNetworkSelectorOpen: boolean;
  
  // Network switching state
  isSwitchingNetwork: boolean;
  switchingToChainId: number | null;
  
  // View preferences
  showTestnets: boolean;
  showDeprecatedNetworks: boolean;
}

interface NetworkUIActions {
  // Preference actions
  setPreferredChainId: (chainId: number | null) => void;
  
  // UI actions
  setNetworkSelectorOpen: (open: boolean) => void;
  
  // Network switching actions
  startNetworkSwitch: (chainId: number) => void;
  endNetworkSwitch: () => void;
  
  // View actions
  setShowTestnets: (show: boolean) => void;
  setShowDeprecatedNetworks: (show: boolean) => void;
  
  // Reset
  reset: () => void;
}

const initialState: NetworkUIState = {
  preferredChainId: null,
  isNetworkSelectorOpen: false,
  isSwitchingNetwork: false,
  switchingToChainId: null,
  showTestnets: true,
  showDeprecatedNetworks: false,
};

export const useNetworkUIStore = create<NetworkUIState & NetworkUIActions>()(
  devtools(
    persist(
      (set) => ({
        ...initialState,

        // Preference actions
        setPreferredChainId: (chainId) => set({ preferredChainId: chainId }),
        
        // UI actions
        setNetworkSelectorOpen: (open) => set({ isNetworkSelectorOpen: open }),
        
        // Network switching actions
        startNetworkSwitch: (chainId) => set({ 
          isSwitchingNetwork: true, 
          switchingToChainId: chainId 
        }),
        endNetworkSwitch: () => set({ 
          isSwitchingNetwork: false, 
          switchingToChainId: null 
        }),
        
        // View actions
        setShowTestnets: (show) => set({ showTestnets: show }),
        setShowDeprecatedNetworks: (show) => set({ showDeprecatedNetworks: show }),
        
        // Reset
        reset: () => set(initialState),
      }),
      {
        name: 'network-ui-storage',
        // Only persist user preferences
        partialize: (state) => ({
          preferredChainId: state.preferredChainId,
          showTestnets: state.showTestnets,
          showDeprecatedNetworks: state.showDeprecatedNetworks,
        }),
      }
    ),
    {
      name: 'network-ui-store',
    }
  )
);