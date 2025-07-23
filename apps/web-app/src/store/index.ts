export * from './auth';
export * from './wallet';
export * from './network';
export * from './ui';

export type StoreState = {
  authState: ReturnType<typeof import('./auth').useAuthStore.getState>;
  walletState: ReturnType<typeof import('./wallet').useWalletStore.getState>;
  networkState: ReturnType<typeof import('./network').useNetworkStore.getState>;
  uiState: ReturnType<typeof import('./ui').useUIStore.getState>;
};
