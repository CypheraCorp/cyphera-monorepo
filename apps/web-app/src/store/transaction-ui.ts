import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

interface TransactionUIState {
  // Selection state
  selectedTransactionId: string | null;
  
  // View preferences (persisted)
  viewMode: 'table' | 'cards';
  itemsPerPage: 10 | 25 | 50 | 100;
  
  // Filter state
  filters: {
    status?: 'pending' | 'completed' | 'failed' | 'processing';
    type?: 'payment' | 'refund' | 'payout' | 'fee';
    walletId?: string;
    customerId?: string;
    minAmount?: number;
    maxAmount?: number;
    dateRange?: {
      from: Date;
      to: Date;
    };
  };
  
  // Modal state
  detailsModalOpen: boolean;
  refundModalOpen: boolean;
  refundingTransactionId: string | null;
  
  // Sort state
  sortBy: 'created_at' | 'amount' | 'status' | 'type';
  sortOrder: 'asc' | 'desc';
  
  // Export state
  isExporting: boolean;
  exportFormat: 'csv' | 'json' | 'pdf';
}

interface TransactionUIActions {
  // Selection actions
  setSelectedTransaction: (id: string | null) => void;
  
  // View actions
  setViewMode: (mode: TransactionUIState['viewMode']) => void;
  setItemsPerPage: (count: TransactionUIState['itemsPerPage']) => void;
  
  // Filter actions
  setFilters: (filters: Partial<TransactionUIState['filters']>) => void;
  updateFilter: <K extends keyof TransactionUIState['filters']>(
    key: K,
    value: TransactionUIState['filters'][K]
  ) => void;
  clearFilters: () => void;
  setDateRange: (from: Date, to: Date) => void;
  
  // Modal actions
  openDetailsModal: (transactionId: string) => void;
  closeDetailsModal: () => void;
  openRefundModal: (transactionId: string) => void;
  closeRefundModal: () => void;
  
  // Sort actions
  setSortBy: (sortBy: TransactionUIState['sortBy']) => void;
  toggleSortOrder: () => void;
  
  // Export actions
  setExporting: (exporting: boolean) => void;
  setExportFormat: (format: TransactionUIState['exportFormat']) => void;
  
  // Reset
  reset: () => void;
}

const initialState: TransactionUIState = {
  selectedTransactionId: null,
  viewMode: 'table',
  itemsPerPage: 25,
  filters: {},
  detailsModalOpen: false,
  refundModalOpen: false,
  refundingTransactionId: null,
  sortBy: 'created_at',
  sortOrder: 'desc',
  isExporting: false,
  exportFormat: 'csv',
};

export const useTransactionUIStore = create<TransactionUIState & TransactionUIActions>()(
  devtools(
    persist(
      (set) => ({
        ...initialState,

        // Selection actions
        setSelectedTransaction: (id) => set({ 
          selectedTransactionId: id,
          detailsModalOpen: !!id 
        }),
        
        // View actions
        setViewMode: (mode) => set({ viewMode: mode }),
        setItemsPerPage: (count) => set({ itemsPerPage: count }),
        
        // Filter actions
        setFilters: (filters) => set((state) => ({
          filters: { ...state.filters, ...filters }
        })),
        updateFilter: (key, value) => set((state) => ({
          filters: { ...state.filters, [key]: value }
        })),
        clearFilters: () => set({ filters: {} }),
        setDateRange: (from, to) => set((state) => ({
          filters: { ...state.filters, dateRange: { from, to } }
        })),
        
        // Modal actions
        openDetailsModal: (transactionId) => set({
          selectedTransactionId: transactionId,
          detailsModalOpen: true,
        }),
        closeDetailsModal: () => set({
          detailsModalOpen: false,
        }),
        openRefundModal: (transactionId) => set({
          refundModalOpen: true,
          refundingTransactionId: transactionId,
        }),
        closeRefundModal: () => set({
          refundModalOpen: false,
          refundingTransactionId: null,
        }),
        
        // Sort actions
        setSortBy: (sortBy) => set({ sortBy }),
        toggleSortOrder: () => set((state) => ({
          sortOrder: state.sortOrder === 'asc' ? 'desc' : 'asc'
        })),
        
        // Export actions
        setExporting: (exporting) => set({ isExporting: exporting }),
        setExportFormat: (format) => set({ exportFormat: format }),
        
        // Reset
        reset: () => set(initialState),
      }),
      {
        name: 'transaction-ui-storage',
        // Only persist view preferences
        partialize: (state) => ({
          viewMode: state.viewMode,
          itemsPerPage: state.itemsPerPage,
          sortBy: state.sortBy,
          sortOrder: state.sortOrder,
          exportFormat: state.exportFormat,
        }),
      }
    ),
    {
      name: 'transaction-ui-store',
    }
  )
);

// Selectors
export const useTransactionFilters = () => 
  useTransactionUIStore((state) => state.filters);

export const useTransactionSort = () => 
  useTransactionUIStore((state) => ({
    sortBy: state.sortBy,
    sortOrder: state.sortOrder,
  }));