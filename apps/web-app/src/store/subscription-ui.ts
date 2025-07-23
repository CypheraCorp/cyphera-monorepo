import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

interface SubscriptionUIState {
  // Selection state
  selectedSubscriptionId: string | null;
  
  // View state
  viewMode: 'grid' | 'list' | 'table';
  
  // Filter state
  filters: {
    status?: 'active' | 'canceled' | 'past_due' | 'trialing';
    productId?: string;
    customerId?: string;
    dateRange?: {
      from: Date;
      to: Date;
    };
  };
  
  // Modal state
  cancelModalOpen: boolean;
  cancellingSubscriptionId: string | null;
  detailsModalOpen: boolean;
  
  // Sort state
  sortBy: 'created_at' | 'status' | 'amount' | 'next_billing_date';
  sortOrder: 'asc' | 'desc';
}

interface SubscriptionUIActions {
  // Selection actions
  setSelectedSubscription: (id: string | null) => void;
  
  // View actions
  setViewMode: (mode: SubscriptionUIState['viewMode']) => void;
  
  // Filter actions
  setFilters: (filters: Partial<SubscriptionUIState['filters']>) => void;
  updateFilter: <K extends keyof SubscriptionUIState['filters']>(
    key: K,
    value: SubscriptionUIState['filters'][K]
  ) => void;
  clearFilters: () => void;
  
  // Modal actions
  openCancelModal: (subscriptionId: string) => void;
  closeCancelModal: () => void;
  setDetailsModalOpen: (open: boolean) => void;
  
  // Sort actions
  setSortBy: (sortBy: SubscriptionUIState['sortBy']) => void;
  toggleSortOrder: () => void;
  
  // Reset
  reset: () => void;
}

const initialState: SubscriptionUIState = {
  selectedSubscriptionId: null,
  viewMode: 'table',
  filters: {},
  cancelModalOpen: false,
  cancellingSubscriptionId: null,
  detailsModalOpen: false,
  sortBy: 'created_at',
  sortOrder: 'desc',
};

export const useSubscriptionUIStore = create<SubscriptionUIState & SubscriptionUIActions>()(
  devtools(
    (set) => ({
      ...initialState,

      // Selection actions
      setSelectedSubscription: (id) => set({ selectedSubscriptionId: id }),
      
      // View actions
      setViewMode: (mode) => set({ viewMode: mode }),
      
      // Filter actions
      setFilters: (filters) => set((state) => ({
        filters: { ...state.filters, ...filters }
      })),
      updateFilter: (key, value) => set((state) => ({
        filters: { ...state.filters, [key]: value }
      })),
      clearFilters: () => set({ filters: {} }),
      
      // Modal actions
      openCancelModal: (subscriptionId) => set({
        cancelModalOpen: true,
        cancellingSubscriptionId: subscriptionId,
      }),
      closeCancelModal: () => set({
        cancelModalOpen: false,
        cancellingSubscriptionId: null,
      }),
      setDetailsModalOpen: (open) => set({ detailsModalOpen: open }),
      
      // Sort actions
      setSortBy: (sortBy) => set({ sortBy }),
      toggleSortOrder: () => set((state) => ({
        sortOrder: state.sortOrder === 'asc' ? 'desc' : 'asc'
      })),
      
      // Reset
      reset: () => set(initialState),
    }),
    {
      name: 'subscription-ui-store',
    }
  )
);

// Selectors for common use cases
export const useSelectedSubscription = () => 
  useSubscriptionUIStore((state) => state.selectedSubscriptionId);

export const useSubscriptionFilters = () => 
  useSubscriptionUIStore((state) => state.filters);

export const useSubscriptionViewMode = () => 
  useSubscriptionUIStore((state) => state.viewMode);