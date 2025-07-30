import { useAccount } from 'wagmi';
import { useSmartAccount } from '@/hooks/store/use-wallet-sync';
import type { MetaMaskSmartAccount } from '@metamask/delegation-toolkit';
import type { SmartAccountProvider } from '../types';

/**
 * Wagmi-based smart account provider for delegation
 * Wraps the existing Wagmi/MetaMask integration
 */
export function useWagmiSmartAccountProvider(): SmartAccountProvider {
  const { isConnected } = useAccount();
  const { 
    smartAccount, 
    smartAccountAddress, 
    isWalletCompatible, 
    isMetaMask, 
    createSmartAccount 
  } = useSmartAccount();

  return {
    type: 'wagmi',
    
    // State
    isConnected,
    isAuthenticated: isConnected,
    smartAccountAddress,
    smartAccount: smartAccount as MetaMaskSmartAccount | null,
    isSmartAccountReady: !!smartAccount && !!smartAccountAddress,
    isDeployed: smartAccountAddress ? true : null, // Wagmi doesn't track deployment state explicitly
    deploymentSupported: true,
    isWalletCompatible,
    provider: undefined, // Wagmi handles provider internally
    
    // Actions
    connect: async () => {
      // Wagmi connection is handled by wagmi hooks/UI
      throw new Error('Please connect your wallet using the wallet connection UI');
    },
    
    createSmartAccount: async () => {
      await createSmartAccount();
    },
    
    checkDeploymentStatus: async () => {
      // For Wagmi, we assume if we have an address, it's deployed
      return !!smartAccountAddress;
    },
    
    deploySmartAccount: async () => {
      // Smart account creation handles deployment in Wagmi flow
      await createSmartAccount();
    },
    
    // Network switching not implemented for Wagmi (user handles manually)
    switchNetwork: undefined,
    
    // Display helpers
    getDisplayName: () => 'MetaMask',
    
    getButtonText: () => {
      if (!isConnected) return 'Connect Wallet';
      if (!isMetaMask) return 'MetaMask Required';
      if (!isWalletCompatible) return 'Wallet Not Compatible';
      return 'Create Delegation';
    },
    
    isButtonDisabled: () => {
      return !isConnected || !isMetaMask || !isWalletCompatible;
    }
  };
}