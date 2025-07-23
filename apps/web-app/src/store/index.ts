/**
 * Centralized exports for all Zustand stores
 * 
 * Usage:
 * import { useAuthStore, useUIStore } from '@/store';
 */

// Core stores
export { useAuthStore } from './auth';
export { useUIStore } from './ui';

// Domain-specific UI stores
export { useWalletUIStore, useWalletFilters, useSelectedWallet } from './wallet-ui';
export { useNetworkUIStore } from './network-ui';
export { useProductUIStore, useProductFilters, useSelectedProducts, useBulkActionMode } from './product-ui';
export { useSubscriptionUIStore, useSelectedSubscription, useSubscriptionFilters, useSubscriptionViewMode } from './subscription-ui';
export { useTransactionUIStore, useTransactionFilters, useTransactionSort } from './transaction-ui';
export { useCustomerUIStore, useMarketplaceFilters, useCustomerPreferences } from './customer-ui';

// Feature-specific stores
export { useCreateProductStore, FORM_STEPS as CREATE_PRODUCT_STEPS } from './create-product';

// Legacy stores (to be migrated)
export { useWalletStore } from './wallet';
export { useNetworkStore } from './network';

// Store types
export type { AuthState, AuthActions } from './auth';
export type { UIState, UIActions } from './ui';

// Development utilities
if (process.env.NODE_ENV === 'development') {
  // Import stores for dev tools
  const { useAuthStore: authStore } = require('./auth');
  const { useUIStore: uiStore } = require('./ui');
  const { useWalletUIStore: walletUIStore } = require('./wallet-ui');
  const { useNetworkUIStore: networkUIStore } = require('./network-ui');
  const { useProductUIStore: productUIStore } = require('./product-ui');
  const { useSubscriptionUIStore: subscriptionUIStore } = require('./subscription-ui');
  const { useTransactionUIStore: transactionUIStore } = require('./transaction-ui');
  const { useCustomerUIStore: customerUIStore } = require('./customer-ui');
  const { useCreateProductStore: createProductStore } = require('./create-product');
  const { useWalletStore: walletStore } = require('./wallet');
  const { useNetworkStore: networkStore } = require('./network');

  // Make stores available on window for debugging
  if (typeof window !== 'undefined') {
    (window as any).__ZUSTAND_STORES__ = {
      auth: authStore,
      ui: uiStore,
      walletUI: walletUIStore,
      networkUI: networkUIStore,
      productUI: productUIStore,
      subscriptionUI: subscriptionUIStore,
      transactionUI: transactionUIStore,
      customerUI: customerUIStore,
      createProduct: createProductStore,
      // Legacy
      wallet: walletStore,
      network: networkStore,
    };
    
    // Helper to reset all stores
    (window as any).__resetAllStores = () => {
      authStore.getState().logout();
      uiStore.getState().resetUIPreferences();
      walletUIStore.getState().reset();
      networkUIStore.getState().reset();
      productUIStore.getState().reset();
      subscriptionUIStore.getState().reset();
      transactionUIStore.getState().reset();
      customerUIStore.getState().reset();
      createProductStore.getState().reset();
      // Legacy
      walletStore.getState().resetWalletState();
      networkStore.getState().resetNetworkState();
      
      console.log('All stores have been reset');
    };
    
    console.log('Zustand stores available on window.__ZUSTAND_STORES__');
    console.log('Reset all stores with window.__resetAllStores()');
  }
}