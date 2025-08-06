import { useWeb3AuthSmartAccount } from '@/hooks/auth';
import { useWeb3AuthInitialization } from '@/hooks/auth';
import { useWeb3Auth } from '@web3auth/modal/react';
import { ensureCorrectNetwork } from '@/lib/web3/utils/delegation';
import type { Web3Provider } from '@/lib/web3/utils/delegation';
import type { MetaMaskSmartAccount } from '@metamask/delegation-toolkit';
import type { SmartAccountProvider } from '../types';



/**
 * Web3Auth-based smart account provider for delegation  
 * Wraps the existing Web3Auth integration with enhanced network switching
 */
export function useWeb3AuthSmartAccountProvider(): SmartAccountProvider {
  const { isAuthenticated } = useWeb3AuthInitialization();
  const {
    smartAccountAddress,
    isSmartAccountReady,
    isDeployed,
    deploymentSupported,
    checkDeploymentStatus,
    deploySmartAccount,
  } = useWeb3AuthSmartAccount();
  
  const { web3Auth } = useWeb3Auth();

  // Get the smart account from Web3Auth's AccountAbstractionProvider
  const customSmartAccount = web3Auth?.accountAbstractionProvider?.smartAccount as MetaMaskSmartAccount | null;

  return {
    type: 'web3auth',
    
    // State
    isConnected: isAuthenticated,
    isAuthenticated,
    smartAccountAddress,
    smartAccount: customSmartAccount,
    isSmartAccountReady,
    isDeployed,
    deploymentSupported,
    isWalletCompatible: true, // Web3Auth is always compatible
    provider: web3Auth?.provider as Web3Provider | undefined,
    
    // Actions
    connect: async () => {
      if (!web3Auth) {
        throw new Error('Web3Auth not initialized');
      }
      await web3Auth.connect();
    },
    
    createSmartAccount: async () => {
      // Web3Auth smart accounts are created automatically upon connection
      if (!isAuthenticated) {
        throw new Error('Please authenticate with Web3Auth first');
      }
    },
    
    checkDeploymentStatus,
    deploySmartAccount,
    
    switchNetwork: async (networkName: string) => {
      if (!web3Auth?.provider) {
        throw new Error('Web3Auth provider not available for network switching');
      }
      await ensureCorrectNetwork(web3Auth.provider as Web3Provider, networkName);
    },
    
    // Display helpers
    getDisplayName: () => 'Web3Auth',
    
    getButtonText: () => {
      if (!isAuthenticated) return 'Sign In to Subscribe';
      if (!isSmartAccountReady) return 'Subscribe';
      // For now, don't check deploymentSupported as it may give false negatives
      if (isDeployed === false && deploymentSupported) return 'Subscribe';
      return 'Subscribe';
    },
    
    isButtonDisabled: () => {
      return !isAuthenticated || !isSmartAccountReady;
    }
  };
}