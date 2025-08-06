import { useState, useEffect, useCallback, useRef } from 'react';
import { useAccount, useWalletClient, usePublicClient, useChainId } from 'wagmi';
import { type Address, type PublicClient, type SignableMessage } from 'viem';
import { toMetaMaskSmartAccount, Implementation } from '@metamask/delegation-toolkit';
import { useToast } from '@/components/ui/use-toast';
import {
  isSmartAccountDeployed,
  deploySmartAccountWithSponsoredGas,
} from '@/lib/web3/utils/smart-account-deployment';
import { logger } from '@/lib/core/logger/logger-utils';

// Types
interface SmartAccount {
  address: Address;
  isDeployed?: () => Promise<boolean>;
  client?: {
    account?: unknown;
    sendTransaction?: (args: unknown) => Promise<string>;
  };
  walletClient?: unknown;
}

interface SmartAccountState {
  address: Address | null;
  instance: SmartAccount | null;
  isCreating: boolean;
  isDeployed: boolean | null; // null = unknown, true = deployed, false = not deployed
  isCheckingDeployment: boolean;
  isDeploying: boolean;
  error: Error | null;
  isWalletCompatible: boolean;
  isMetaMask: boolean;
}

interface UseSmartAccountReturn {
  smartAccountAddress: Address | null;
  smartAccount: SmartAccount | null;
  isCreatingSmartAccount: boolean;
  isSmartAccountDeployed: boolean | null;
  isCheckingDeployment: boolean;
  isDeployingSmartAccount: boolean;
  error: Error | null;
  isWalletCompatible: boolean;
  isMetaMask: boolean;
  createSmartAccount: () => Promise<void>;
  deploySmartAccount: () => Promise<void>;
  checkDeploymentStatus: () => Promise<void>;
  publicClient: PublicClient | undefined;
}

export function useSmartAccount(): UseSmartAccountReturn {
  const { address, isConnected } = useAccount();
  const { data: walletClient } = useWalletClient();
  const chainId = useChainId();
  const { toast } = useToast();

  const publicClient = usePublicClient();

  const [state, setState] = useState<SmartAccountState>({
    address: null,
    instance: null,
    isCreating: false,
    isDeployed: null,
    isCheckingDeployment: false,
    isDeploying: false,
    error: null,
    isWalletCompatible: true,
    isMetaMask: false,
  });

  // Effect: Check wallet compatibility
  useEffect(() => {
    if (!isConnected || !walletClient) {
      setState((prev) => ({
        ...prev,
        isMetaMask: false,
        isWalletCompatible: true,
      }));
      return;
    }

    const checkAndSetupWallet = () => {
      const isMetaMask =
        typeof window !== 'undefined' &&
        typeof window.ethereum !== 'undefined' &&
        window.ethereum.isMetaMask;

      const hasSignMessage = typeof walletClient.signMessage === 'function';
      const hasSignTypedData = typeof walletClient.signTypedData === 'function';
      const isCompatible = hasSignMessage && hasSignTypedData && !!isMetaMask;

      setState((prev) => ({
        ...prev,
        isMetaMask: !!isMetaMask,
        isWalletCompatible: isCompatible,
      }));

      if (!isMetaMask) {
        toast({
          title: 'MetaMask Required',
          description: 'Please connect with MetaMask.',
          variant: 'destructive',
        });
        return;
      }

      if (!isCompatible) {
        toast({
          title: 'Wallet Not Compatible',
          description: 'Your wallet does not support required signing methods for Smart Accounts.',
          variant: 'destructive',
        });
      }
    };

    checkAndSetupWallet();
  }, [isConnected, walletClient, toast]);

  // Effect: Reset state on disconnect
  useEffect(() => {
    if (!isConnected) {
      setState({
        address: null,
        instance: null,
        isCreating: false,
        isDeployed: null,
        isCheckingDeployment: false,
        isDeploying: false,
        error: null,
        isWalletCompatible: true,
        isMetaMask: false,
      });
    }
  }, [isConnected]);

  // Main function: Create smart account
  const createSmartAccount = useCallback(async () => {
    if (
      !isConnected ||
      !address ||
      !state.isWalletCompatible ||
      !state.isMetaMask ||
      state.isCreating ||
      !walletClient ||
      !publicClient
    )
      return;

    try {
      setState((prev) => ({ ...prev, isCreating: true, error: null }));

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

      // Attach the wallet client to the smart account so it can send transactions
      if (newSmartAccount) {
        const smartAccount = newSmartAccount as SmartAccount;
        smartAccount.client = walletClient as typeof smartAccount.client;
        smartAccount.walletClient = walletClient;
      }

      setState((prev) => ({
        ...prev,
        address: newSmartAccount.address,
        instance: newSmartAccount,
      }));
    } catch (err) {
      logger.error_sync('Error creating smart account:', err);
      const error = err instanceof Error ? err : new Error('Failed to create smart account');

      setState((prev) => ({ ...prev, error }));

      const errorMessage = error.message.toLowerCase();
      const isUserRejection =
        errorMessage.includes('reject') ||
        errorMessage.includes('denied') ||
        errorMessage.includes('cancelled');
      const isPendingRequest = errorMessage.includes('already pending');
      const isSigningError =
        errorMessage.includes('does not support signmessage') ||
        errorMessage.includes('does not support signtypeddata');

      if (isPendingRequest) {
        toast({
          title: 'Request Already Pending',
          description: 'Please check MetaMask and respond to the existing request.',
          variant: 'destructive',
        });
      } else if (isSigningError) {
        toast({
          title: 'Wallet Not Compatible',
          description: 'Your wallet does not support required signing methods.',
          variant: 'destructive',
        });
        setState((prev) => ({ ...prev, isWalletCompatible: false }));
      } else {
        toast({
          title: isUserRejection ? 'Request Rejected' : 'Smart Account Creation Failed',
          description: isUserRejection ? 'You rejected the signature request.' : error.message,
          variant: 'destructive',
        });
      }
    } finally {
      setState((prev) => ({ ...prev, isCreating: false }));
    }
  }, [
    isConnected,
    address,
    state.isWalletCompatible,
    state.isMetaMask,
    state.isCreating,
    walletClient,
    publicClient,
    toast,
  ]);

  // Effect: Automatically create smart account when wallet connects
  useEffect(() => {
    if (
      isConnected &&
      state.isWalletCompatible &&
      state.isMetaMask &&
      !state.instance &&
      !state.isCreating &&
      !state.error &&
      publicClient
    ) {
      createSmartAccount();
    }
  }, [
    isConnected,
    state.isWalletCompatible,
    state.isMetaMask,
    publicClient,
    state.instance,
    state.isCreating,
    state.error,
    createSmartAccount,
  ]);

  // Additional functions
  const deploySmartAccount = useCallback(async () => {
    if (
      !isConnected ||
      !state.instance ||
      !state.isWalletCompatible ||
      !state.isMetaMask ||
      state.isDeploying ||
      !chainId
    )
      return;

    try {
      setState((prev) => ({ ...prev, isDeploying: true, error: null }));

      await deploySmartAccountWithSponsoredGas(state.instance, chainId);

      setState((prev) => ({ ...prev, isDeployed: true }));

      toast({
        title: 'Smart Account Deployed',
        description: 'Your smart account has been successfully deployed!',
      });
    } catch (err) {
      logger.error_sync('Error deploying smart account:', err);
      const error = err instanceof Error ? err : new Error('Failed to deploy smart account');

      setState((prev) => ({ ...prev, error }));

      toast({
        title: 'Smart Account Deployment Failed',
        description: error.message,
        variant: 'destructive',
      });
    } finally {
      setState((prev) => ({ ...prev, isDeploying: false }));
    }
  }, [
    isConnected,
    state.instance,
    state.isWalletCompatible,
    state.isMetaMask,
    state.isDeploying,
    chainId,
    toast,
  ]);

  const checkDeploymentStatus = useCallback(async () => {
    if (
      !isConnected ||
      !state.address ||
      !state.isWalletCompatible ||
      !state.isMetaMask ||
      state.isCheckingDeployment ||
      !publicClient
    )
      return;

    try {
      setState((prev) => ({ ...prev, isCheckingDeployment: true }));

      const deployed = await isSmartAccountDeployed(state.address, publicClient);

      setState((prev) => ({ ...prev, isDeployed: deployed }));
    } catch (err) {
      logger.error_sync('Error checking deployment status:', err);
      const error = err instanceof Error ? err : new Error('Failed to check deployment status');

      setState((prev) => ({ ...prev, error }));

      toast({
        title: 'Smart Account Deployment Check Failed',
        description: error.message,
        variant: 'destructive',
      });
    } finally {
      setState((prev) => ({ ...prev, isCheckingDeployment: false }));
    }
  }, [
    isConnected,
    state.address,
    state.isWalletCompatible,
    state.isMetaMask,
    state.isCheckingDeployment,
    publicClient,
    toast,
  ]);

  // Effect: Automatically check deployment status when smart account is created
  useEffect(() => {
    if (
      state.instance &&
      state.address &&
      state.isDeployed === null &&
      !state.isCheckingDeployment &&
      publicClient
    ) {
      checkDeploymentStatus();
    }
  }, [
    state.instance,
    state.address,
    state.isDeployed,
    state.isCheckingDeployment,
    publicClient,
    checkDeploymentStatus,
  ]);

  // Track the previous chainId to detect network changes
  const prevChainIdRef = useRef<number | undefined>(undefined);
  const networkSwitchRef = useRef<boolean>(false);

  // Effect: Handle network switching - reinitialize smart account when chain changes
  useEffect(() => {
    if (!isConnected || !chainId) return;

    // Check if this is actually a network change (not initial load)
    const isNetworkChange =
      prevChainIdRef.current !== undefined && prevChainIdRef.current !== chainId;

    if (isNetworkChange) {
      logger.log('Network switched from', prevChainIdRef.current, 'to', chainId);
      networkSwitchRef.current = true;

      // Use setState with a function to access current state without dependencies
      setState((currentState) => {
        // Only reset if we actually have a smart account to reset
        if (currentState.instance && currentState.address) {
          logger.log('Resetting smart account for new network');

          return {
            ...currentState,
            address: null,
            instance: null,
            isDeployed: null,
            error: null,
          };
        }

        // No smart account to reset, return current state unchanged
        return currentState;
      });
    }

    // Update the previous chainId reference
    prevChainIdRef.current = chainId;
  }, [chainId, isConnected]);

  // Separate effect to show toast after network switch (prevents toast during render)
  useEffect(() => {
    if (networkSwitchRef.current) {
      toast({
        title: 'Network Switched',
        description: 'Reinitializing smart account for the new network...',
      });
      networkSwitchRef.current = false;
    }
  }, [state.address, toast]);

  return {
    smartAccountAddress: state.address,
    smartAccount: state.instance,
    isCreatingSmartAccount: state.isCreating,
    isSmartAccountDeployed: state.isDeployed,
    isCheckingDeployment: state.isCheckingDeployment,
    isDeployingSmartAccount: state.isDeploying,
    error: state.error,
    isWalletCompatible: state.isWalletCompatible,
    isMetaMask: state.isMetaMask,
    createSmartAccount,
    deploySmartAccount,
    checkDeploymentStatus,
    publicClient,
  };
}
