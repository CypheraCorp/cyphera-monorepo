import { useEffect, useCallback } from 'react';
import { useWalletStore } from '@/store/wallet';
import { useAccount, useWalletClient, usePublicClient, useChainId } from 'wagmi';
import { type SignableMessage } from 'viem';
import { toMetaMaskSmartAccount, Implementation } from '@metamask/delegation-toolkit';
import { useToast } from '@/components/ui/use-toast';
import {
  isSmartAccountDeployed as checkIsSmartAccountDeployed,
  deploySmartAccountWithSponsoredGas,
} from '@/lib/web3/utils/smart-account-deployment';
import { logger } from '@/lib/core/logger/logger-utils';

/**
 * Hook that syncs wallet state with Zustand store and provides smart account functionality
 * Drop-in replacement for useSmartAccount
 */
export function useSmartAccount() {
  const { address, isConnected } = useAccount();
  const { data: walletClient } = useWalletClient();
  const chainId = useChainId();
  const publicClient = usePublicClient();
  const { toast } = useToast();

  // Get state from store
  const smartAccountAddress = useWalletStore((state) => state.smartAccountAddress);
  const smartAccount = useWalletStore((state) => state.smartAccount);
  const isCreatingSmartAccount = useWalletStore((state) => state.isCreatingSmartAccount);
  const isSmartAccountDeployed = useWalletStore((state) => state.isSmartAccountDeployed);
  const isCheckingDeployment = useWalletStore((state) => state.isCheckingDeployment);
  const isDeployingSmartAccount = useWalletStore((state) => state.isDeployingSmartAccount);
  const error = useWalletStore((state) => state.error);
  const isWalletCompatible = useWalletStore((state) => state.isWalletCompatible);
  const isMetaMask = useWalletStore((state) => state.isMetaMask);

  // Actions from store
  const setConnectionState = useWalletStore((state) => state.setConnectionState);
  const setSmartAccount = useWalletStore((state) => state.setSmartAccount);
  const setSmartAccountAddress = useWalletStore((state) => state.setSmartAccountAddress);
  const setSmartAccountDeployed = useWalletStore((state) => state.setSmartAccountDeployed);
  const setCreatingSmartAccount = useWalletStore((state) => state.setCreatingSmartAccount);
  const setCheckingDeployment = useWalletStore((state) => state.setCheckingDeployment);
  const setDeployingSmartAccount = useWalletStore((state) => state.setDeployingSmartAccount);
  const setWalletCompatibility = useWalletStore((state) => state.setWalletCompatibility);
  const setError = useWalletStore((state) => state.setError);
  const resetSmartAccountState = useWalletStore((state) => state.resetSmartAccountState);

  // Sync connection state
  useEffect(() => {
    setConnectionState({
      isConnected,
      address: address || null,
      chainId: chainId || null,
    });
  }, [isConnected, address, chainId, setConnectionState]);

  // Check wallet compatibility
  useEffect(() => {
    if (!isConnected || !walletClient) {
      setWalletCompatibility(true, false);
      return;
    }

    const isMetaMask =
      typeof window !== 'undefined' &&
      typeof window.ethereum !== 'undefined' &&
      window.ethereum.isMetaMask;

    const hasSignMessage = typeof walletClient.signMessage === 'function';
    const hasSignTypedData = typeof walletClient.signTypedData === 'function';
    const isCompatible = hasSignMessage && hasSignTypedData && !!isMetaMask;

    setWalletCompatibility(isCompatible, !!isMetaMask);

    if (!isMetaMask) {
      toast({
        title: 'MetaMask Required',
        description: 'Please connect with MetaMask.',
        variant: 'destructive',
      });
    } else if (!isCompatible) {
      toast({
        title: 'Wallet Not Compatible',
        description: 'Your wallet does not support required signing methods for Smart Accounts.',
        variant: 'destructive',
      });
    }
  }, [isConnected, walletClient, setWalletCompatibility, toast]);

  // Reset state on disconnect
  useEffect(() => {
    if (!isConnected) {
      resetSmartAccountState();
    }
  }, [isConnected, resetSmartAccountState]);

  // Handle network changes
  useEffect(() => {
    // Reset smart account on network change if already created
    if (smartAccount && chainId) {
      logger.log('Network changed, resetting smart account');
      resetSmartAccountState();
      toast({
        title: 'Network Switched',
        description: 'Reinitializing smart account for the new network...',
      });
    }
  }, [chainId, smartAccount, resetSmartAccountState, toast]);

  // Create smart account function
  const createSmartAccount = useCallback(async () => {
    if (
      !isConnected ||
      !address ||
      !isWalletCompatible ||
      !isMetaMask ||
      isCreatingSmartAccount ||
      !walletClient ||
      !publicClient
    )
      return;

    try {
      setCreatingSmartAccount(true);
      setError(null);

      const account = {
        address,
        signMessage: async (params: { message: SignableMessage }) => {
          return walletClient.signMessage(params);
        },
        signTypedData: async (params: Parameters<typeof walletClient.signTypedData>[0]) => {
          return walletClient.signTypedData(params);
        },
      };

      const newSmartAccount = await toMetaMaskSmartAccount({
        client: publicClient,
        implementation: Implementation.Hybrid,
        deployParams: [address, [], [], []],
        deploySalt: '0x',
        signatory: { account } as any,
      });

      // Attach the wallet client to the smart account
      if (newSmartAccount) {
        const smartAccountWithClient = newSmartAccount as any;
        smartAccountWithClient.client = walletClient;
        smartAccountWithClient.walletClient = walletClient;

        setSmartAccount(smartAccountWithClient);
        setSmartAccountAddress(newSmartAccount.address);
      }
    } catch (err) {
      logger.error_sync('Error creating smart account:', err);
      const error = err instanceof Error ? err : new Error('Failed to create smart account');
      setError(error);

      const errorMessage = error.message.toLowerCase();
      const isUserRejection =
        errorMessage.includes('reject') ||
        errorMessage.includes('denied') ||
        errorMessage.includes('cancelled');

      toast({
        title: isUserRejection ? 'Request Rejected' : 'Smart Account Creation Failed',
        description: isUserRejection ? 'You rejected the signature request.' : error.message,
        variant: 'destructive',
      });
    } finally {
      setCreatingSmartAccount(false);
    }
  }, [
    isConnected,
    address,
    isWalletCompatible,
    isMetaMask,
    isCreatingSmartAccount,
    walletClient,
    publicClient,
    setCreatingSmartAccount,
    setError,
    setSmartAccount,
    setSmartAccountAddress,
    toast,
  ]);

  // Deploy smart account function
  const deploySmartAccount = useCallback(async () => {
    if (
      !isConnected ||
      !smartAccount ||
      !isWalletCompatible ||
      !isMetaMask ||
      isDeployingSmartAccount ||
      !chainId
    )
      return;

    try {
      setDeployingSmartAccount(true);
      setError(null);

      await deploySmartAccountWithSponsoredGas(smartAccount, chainId);
      setSmartAccountDeployed(true);

      toast({
        title: 'Smart Account Deployed',
        description: 'Your smart account has been successfully deployed!',
      });
    } catch (err) {
      logger.error_sync('Error deploying smart account:', err);
      const error = err instanceof Error ? err : new Error('Failed to deploy smart account');
      setError(error);

      toast({
        title: 'Smart Account Deployment Failed',
        description: error.message,
        variant: 'destructive',
      });
    } finally {
      setDeployingSmartAccount(false);
    }
  }, [
    isConnected,
    smartAccount,
    isWalletCompatible,
    isMetaMask,
    isDeployingSmartAccount,
    chainId,
    setDeployingSmartAccount,
    setError,
    setSmartAccountDeployed,
    toast,
  ]);

  // Check deployment status function
  const checkDeploymentStatus = useCallback(async () => {
    if (
      !isConnected ||
      !smartAccountAddress ||
      !isWalletCompatible ||
      !isMetaMask ||
      isCheckingDeployment ||
      !publicClient
    )
      return;

    try {
      setCheckingDeployment(true);
      const deployed = await checkIsSmartAccountDeployed(smartAccountAddress, publicClient);
      setSmartAccountDeployed(deployed);
    } catch (err) {
      logger.error_sync('Error checking deployment status:', err);
      const error = err instanceof Error ? err : new Error('Failed to check deployment status');
      setError(error);

      toast({
        title: 'Smart Account Deployment Check Failed',
        description: error.message,
        variant: 'destructive',
      });
    } finally {
      setCheckingDeployment(false);
    }
  }, [
    isConnected,
    smartAccountAddress,
    isWalletCompatible,
    isMetaMask,
    isCheckingDeployment,
    publicClient,
    setCheckingDeployment,
    setSmartAccountDeployed,
    setError,
    toast,
  ]);

  // Auto-create smart account when wallet connects
  useEffect(() => {
    if (
      isConnected &&
      isWalletCompatible &&
      isMetaMask &&
      !smartAccount &&
      !isCreatingSmartAccount &&
      !error &&
      publicClient
    ) {
      createSmartAccount();
    }
  }, [
    isConnected,
    isWalletCompatible,
    isMetaMask,
    publicClient,
    smartAccount,
    isCreatingSmartAccount,
    error,
    createSmartAccount,
  ]);

  // Auto-check deployment status when smart account is created
  useEffect(() => {
    if (
      smartAccount &&
      smartAccountAddress &&
      isSmartAccountDeployed === null &&
      !isCheckingDeployment &&
      publicClient
    ) {
      checkDeploymentStatus();
    }
  }, [
    smartAccount,
    smartAccountAddress,
    isSmartAccountDeployed,
    isCheckingDeployment,
    publicClient,
    checkDeploymentStatus,
  ]);

  return {
    smartAccountAddress,
    smartAccount,
    isCreatingSmartAccount,
    isSmartAccountDeployed,
    isCheckingDeployment,
    isDeployingSmartAccount,
    error,
    isWalletCompatible,
    isMetaMask,
    createSmartAccount,
    deploySmartAccount,
    checkDeploymentStatus,
    publicClient,
  };
}
