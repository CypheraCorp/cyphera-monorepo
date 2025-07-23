import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

/**
 * Wallet UI Store - Only UI state, no wallet data
 * Actual wallet data comes from React Query hooks
 */
interface WalletUIState {
  // UI State
  selectedWalletId: string | null;
  isCreateModalOpen: boolean;
  isImportModalOpen: boolean;
  viewMode: 'grid' | 'list';
  
  // Filter state
  filters: {
    network?: string;
    search?: string;
  };
  
  // Temporary operation states
  operationInProgress: {
    walletId?: string;
    operation?: 'transfer' | 'receive' | 'swap';
  } | null;
}

interface WalletUIActions {
  // UI Actions
  setSelectedWallet: (walletId: string | null) => void;
  setCreateModalOpen: (open: boolean) => void;
  setImportModalOpen: (open: boolean) => void;
  setViewMode: (mode: 'grid' | 'list') => void;
  
  // Filter actions
  setFilters: (filters: WalletUIState['filters']) => void;
  clearFilters: () => void;
  
  // Operation actions
  startOperation: (walletId: string, operation: 'transfer' | 'receive' | 'swap') => void;
  clearOperation: () => void;
  
  // Reset
  reset: () => void;
}

const initialState: WalletUIState = {
  selectedWalletId: null,
  isCreateModalOpen: false,
  isImportModalOpen: false,
  viewMode: 'grid',
  filters: {},
  operationInProgress: null,
};

export const useWalletUIStore = create<WalletUIState & WalletUIActions>()(
  devtools(
    (set) => ({
      ...initialState,

      // UI Actions
      setSelectedWallet: (walletId) => set({ selectedWalletId: walletId }),
      setCreateModalOpen: (open) => set({ isCreateModalOpen: open }),
      setImportModalOpen: (open) => set({ isImportModalOpen: open }),
      setViewMode: (mode) => set({ viewMode: mode }),
      
      // Filter actions
      setFilters: (filters) => set((state) => ({ 
        filters: { ...state.filters, ...filters } 
      })),
      clearFilters: () => set({ filters: {} }),
      
      // Operation actions
      startOperation: (walletId, operation) => set({ 
        operationInProgress: { walletId, operation } 
      }),
      clearOperation: () => set({ operationInProgress: null }),
      
      // Reset
      reset: () => set(initialState),
    }),
    {
      name: 'wallet-ui-store',
    }
  )
);

// Selectors
export const useSelectedWallet = () => 
  useWalletUIStore((state) => state.selectedWalletId);

export const useWalletFilters = () => 
  useWalletUIStore((state) => state.filters);

export const useWalletViewMode = () => 
  useWalletUIStore((state) => state.viewMode);