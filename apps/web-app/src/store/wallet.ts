import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { type Address } from 'viem';

interface SmartAccount {
  address: Address;
  isDeployed?: () => Promise<boolean>;
  client?: {
    account?: unknown;
    sendTransaction?: (args: unknown) => Promise<string>;
  };
  walletClient?: unknown;
}

interface WalletState {
  // Connection state
  isConnected: boolean;
  address: Address | null;
  chainId: number | null;

  // Smart account state
  smartAccountAddress: Address | null;
  smartAccount: SmartAccount | null;
  isSmartAccountDeployed: boolean | null;

  // Loading states
  isCreatingSmartAccount: boolean;
  isCheckingDeployment: boolean;
  isDeployingSmartAccount: boolean;

  // Compatibility
  isWalletCompatible: boolean;
  isMetaMask: boolean;

  // Error handling
  error: Error | null;
}

interface WalletActions {
  // Connection actions
  setConnectionState: (state: {
    isConnected: boolean;
    address: Address | null;
    chainId: number | null;
  }) => void;

  // Smart account actions
  setSmartAccount: (account: SmartAccount | null) => void;
  setSmartAccountAddress: (address: Address | null) => void;
  setSmartAccountDeployed: (deployed: boolean | null) => void;

  // Loading state actions
  setCreatingSmartAccount: (creating: boolean) => void;
  setCheckingDeployment: (checking: boolean) => void;
  setDeployingSmartAccount: (deploying: boolean) => void;

  // Compatibility actions
  setWalletCompatibility: (compatible: boolean, isMetaMask: boolean) => void;

  // Error actions
  setError: (error: Error | null) => void;
  clearError: () => void;

  // Reset actions
  resetWalletState: () => void;
  resetSmartAccountState: () => void;
}

const initialState: WalletState = {
  isConnected: false,
  address: null,
  chainId: null,
  smartAccountAddress: null,
  smartAccount: null,
  isSmartAccountDeployed: null,
  isCreatingSmartAccount: false,
  isCheckingDeployment: false,
  isDeployingSmartAccount: false,
  isWalletCompatible: true,
  isMetaMask: false,
  error: null,
};

export const useWalletStore = create<WalletState & WalletActions>()(
  devtools(
    persist(
      (set) => ({
        ...initialState,

        // Connection actions
        setConnectionState: ({ isConnected, address, chainId }) =>
          set({ isConnected, address, chainId }),

        // Smart account actions
        setSmartAccount: (account) => set({ smartAccount: account }),
        setSmartAccountAddress: (address) => set({ smartAccountAddress: address }),
        setSmartAccountDeployed: (deployed) => set({ isSmartAccountDeployed: deployed }),

        // Loading state actions
        setCreatingSmartAccount: (creating) => set({ isCreatingSmartAccount: creating }),
        setCheckingDeployment: (checking) => set({ isCheckingDeployment: checking }),
        setDeployingSmartAccount: (deploying) => set({ isDeployingSmartAccount: deploying }),

        // Compatibility actions
        setWalletCompatibility: (compatible, isMetaMask) =>
          set({ isWalletCompatible: compatible, isMetaMask }),

        // Error actions
        setError: (error) => set({ error }),
        clearError: () => set({ error: null }),

        // Reset actions
        resetWalletState: () => set(initialState),
        resetSmartAccountState: () =>
          set({
            smartAccountAddress: null,
            smartAccount: null,
            isSmartAccountDeployed: null,
            isCreatingSmartAccount: false,
            isCheckingDeployment: false,
            isDeployingSmartAccount: false,
            error: null,
          }),
      }),
      {
        name: 'wallet-storage',
        partialize: (state) => ({
          // Only persist connection state
          isConnected: state.isConnected,
          address: state.address,
          chainId: state.chainId,
          isWalletCompatible: state.isWalletCompatible,
          isMetaMask: state.isMetaMask,
        }),
      }
    )
  )
);
