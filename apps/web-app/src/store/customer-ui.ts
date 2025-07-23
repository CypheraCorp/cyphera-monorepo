import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

interface CustomerUIState {
  // Dashboard preferences (persisted)
  dashboardLayout: 'default' | 'compact' | 'detailed';
  showWelcomeBanner: boolean;
  preferredPaymentMethod: string | null;
  
  // Marketplace state
  marketplace: {
    searchQuery: string;
    selectedCategory: string | null;
    priceRange: { min?: number; max?: number };
    sortBy: 'popular' | 'newest' | 'price_low' | 'price_high';
    viewMode: 'grid' | 'list';
  };
  
  // Subscription management
  subscriptionFilters: {
    status?: 'active' | 'canceled' | 'past_due';
    sortBy: 'next_billing' | 'created_at' | 'amount';
  };
  
  // Wallet preferences
  preferredWalletId: string | null;
  showBalanceInUSD: boolean;
  
  // Modals
  activeSubscriptionModal: string | null;
  showPaymentMethodModal: boolean;
  showCancelSubscriptionModal: boolean;
  cancellingSubscriptionId: string | null;
}

interface CustomerUIActions {
  // Dashboard actions
  setDashboardLayout: (layout: CustomerUIState['dashboardLayout']) => void;
  setShowWelcomeBanner: (show: boolean) => void;
  setPreferredPaymentMethod: (method: string | null) => void;
  
  // Marketplace actions
  setMarketplaceSearch: (query: string) => void;
  setMarketplaceCategory: (category: string | null) => void;
  setMarketplacePriceRange: (range: CustomerUIState['marketplace']['priceRange']) => void;
  setMarketplaceSortBy: (sortBy: CustomerUIState['marketplace']['sortBy']) => void;
  setMarketplaceViewMode: (mode: CustomerUIState['marketplace']['viewMode']) => void;
  clearMarketplaceFilters: () => void;
  
  // Subscription actions
  setSubscriptionFilters: (filters: CustomerUIState['subscriptionFilters']) => void;
  
  // Wallet actions
  setPreferredWallet: (walletId: string | null) => void;
  toggleShowBalanceInUSD: () => void;
  
  // Modal actions
  openSubscriptionModal: (subscriptionId: string) => void;
  closeSubscriptionModal: () => void;
  openPaymentMethodModal: () => void;
  closePaymentMethodModal: () => void;
  openCancelSubscriptionModal: (subscriptionId: string) => void;
  closeCancelSubscriptionModal: () => void;
  
  // Reset
  resetMarketplace: () => void;
  reset: () => void;
}

const initialMarketplaceState = {
  searchQuery: '',
  selectedCategory: null,
  priceRange: {},
  sortBy: 'popular' as const,
  viewMode: 'grid' as const,
};

const initialState: CustomerUIState = {
  dashboardLayout: 'default',
  showWelcomeBanner: true,
  preferredPaymentMethod: null,
  marketplace: initialMarketplaceState,
  subscriptionFilters: {
    sortBy: 'next_billing',
  },
  preferredWalletId: null,
  showBalanceInUSD: true,
  activeSubscriptionModal: null,
  showPaymentMethodModal: false,
  showCancelSubscriptionModal: false,
  cancellingSubscriptionId: null,
};

export const useCustomerUIStore = create<CustomerUIState & CustomerUIActions>()(
  devtools(
    persist(
      (set) => ({
        ...initialState,

        // Dashboard actions
        setDashboardLayout: (layout) => set({ dashboardLayout: layout }),
        setShowWelcomeBanner: (show) => set({ showWelcomeBanner: show }),
        setPreferredPaymentMethod: (method) => set({ preferredPaymentMethod: method }),
        
        // Marketplace actions
        setMarketplaceSearch: (query) => set((state) => ({
          marketplace: { ...state.marketplace, searchQuery: query }
        })),
        setMarketplaceCategory: (category) => set((state) => ({
          marketplace: { ...state.marketplace, selectedCategory: category }
        })),
        setMarketplacePriceRange: (range) => set((state) => ({
          marketplace: { ...state.marketplace, priceRange: range }
        })),
        setMarketplaceSortBy: (sortBy) => set((state) => ({
          marketplace: { ...state.marketplace, sortBy }
        })),
        setMarketplaceViewMode: (mode) => set((state) => ({
          marketplace: { ...state.marketplace, viewMode: mode }
        })),
        clearMarketplaceFilters: () => set((state) => ({
          marketplace: {
            ...initialMarketplaceState,
            viewMode: state.marketplace.viewMode, // Preserve view preference
          }
        })),
        
        // Subscription actions
        setSubscriptionFilters: (filters) => set({ subscriptionFilters: filters }),
        
        // Wallet actions
        setPreferredWallet: (walletId) => set({ preferredWalletId: walletId }),
        toggleShowBalanceInUSD: () => set((state) => ({ 
          showBalanceInUSD: !state.showBalanceInUSD 
        })),
        
        // Modal actions
        openSubscriptionModal: (subscriptionId) => set({ 
          activeSubscriptionModal: subscriptionId 
        }),
        closeSubscriptionModal: () => set({ 
          activeSubscriptionModal: null 
        }),
        openPaymentMethodModal: () => set({ 
          showPaymentMethodModal: true 
        }),
        closePaymentMethodModal: () => set({ 
          showPaymentMethodModal: false 
        }),
        openCancelSubscriptionModal: (subscriptionId) => set({
          showCancelSubscriptionModal: true,
          cancellingSubscriptionId: subscriptionId,
        }),
        closeCancelSubscriptionModal: () => set({
          showCancelSubscriptionModal: false,
          cancellingSubscriptionId: null,
        }),
        
        // Reset actions
        resetMarketplace: () => set({ marketplace: initialMarketplaceState }),
        reset: () => set(initialState),
      }),
      {
        name: 'customer-ui-storage',
        // Persist user preferences
        partialize: (state) => ({
          dashboardLayout: state.dashboardLayout,
          showWelcomeBanner: state.showWelcomeBanner,
          preferredPaymentMethod: state.preferredPaymentMethod,
          preferredWalletId: state.preferredWalletId,
          showBalanceInUSD: state.showBalanceInUSD,
          marketplace: {
            viewMode: state.marketplace.viewMode,
            sortBy: state.marketplace.sortBy,
          },
        }),
      }
    ),
    {
      name: 'customer-ui-store',
    }
  )
);

// Selectors
export const useMarketplaceFilters = () => 
  useCustomerUIStore((state) => state.marketplace);

export const useCustomerPreferences = () => 
  useCustomerUIStore((state) => ({
    dashboardLayout: state.dashboardLayout,
    showBalanceInUSD: state.showBalanceInUSD,
    preferredPaymentMethod: state.preferredPaymentMethod,
  }));