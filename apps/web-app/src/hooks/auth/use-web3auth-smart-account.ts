import { useState, useEffect, useCallback } from 'react';
import { useWeb3Auth } from '@web3auth/modal/react';
import { type Address, createPublicClient, http } from 'viem';
import { isSmartAccountDeployed } from '@/lib/web3/utils/smart-account-deployment';
import { isPimlicoSupportedForChain } from '@/lib/web3/config/networks';
import { getNetworkConfig } from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';
// Types for Web3Auth Smart Account
interface Web3AuthSmartAccountState {
  smartAccountAddress: Address | null;
  isSmartAccountReady: boolean;
  isLoading: boolean;
  error: Error | null;

  // Deployment state
  isDeployed: boolean | null; // null = unknown, true = deployed, false = not deployed
  isDeploying: boolean;
  deploymentError: Error | null;
  deploymentSupported: boolean; // Whether current network supports Pimlico deployment
}

interface UseWeb3AuthSmartAccountReturn extends Web3AuthSmartAccountState {
  refreshSmartAccount: () => Promise<void>;
  checkDeploymentStatus: () => Promise<boolean>;
  deploySmartAccount: () => Promise<void>;
  refreshNetworkState: () => Promise<void>;
  verifyProviderState: () => Promise<boolean>;
  forceProviderRefresh: () => Promise<void>;
}

/**
 * Hook to access Web3Auth embedded smart account instances with deployment functionality
 * Uses AccountAbstractionProvider for seamless smart account management
 */
export function useWeb3AuthSmartAccount(): UseWeb3AuthSmartAccountReturn {
  // Safe Web3Auth hooks - call directly to comply with React Hook rules
  let web3Auth: {
    provider: {
      request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
    } | null;
    accountAbstractionProvider?: unknown;
  } | null = null;
  let isWeb3AuthConnected = false;

  try {
    const web3AuthResult = useWeb3Auth();
    web3Auth = web3AuthResult.web3Auth;
    isWeb3AuthConnected = web3AuthResult.isConnected;
  } catch {
    logger.warn('⚠️ Web3Auth context not available');
  }

  // Remove Wagmi dependencies - we'll get everything from Web3Auth
  const [currentChainId, setCurrentChainId] = useState<number | null>(null);
  const [publicClient, setPublicClient] = useState<ReturnType<typeof createPublicClient> | null>(
    null
  );
  const [providerInitialized, setProviderInitialized] = useState(false);

  const [state, setState] = useState<Web3AuthSmartAccountState>({
    smartAccountAddress: null,
    isSmartAccountReady: false,
    isLoading: false,
    error: null,
    isDeployed: null,
    isDeploying: false,
    deploymentError: null,
    deploymentSupported: false,
  });

  // Check if deployment is supported on current network
  // Requires both network support AND AccountAbstractionProvider availability
  const [deploymentSupported, setDeploymentSupported] = useState(false);

  // Add setupNetworkInfo as a callable function (moved before the effect)
  const setupNetworkInfo = useCallback(async () => {
    if (!web3Auth?.provider || !isWeb3AuthConnected) {
      setCurrentChainId(null);
      setPublicClient(null);
      setDeploymentSupported(false);
      return;
    }

    try {
      // Get current chain ID from Web3Auth provider
      const chainIdHex = (await web3Auth.provider.request({
        method: 'eth_chainId',
      })) as string;
      const chainIdDecimal = parseInt(chainIdHex, 16);

      logger.log('🔍 [setupNetworkInfo] Got chain ID from Web3Auth:', {
        chainIdHex,
        chainIdDecimal,
      });

      setCurrentChainId(chainIdDecimal);

      // Create public client for this network
      const networkConfig = await getNetworkConfig(chainIdDecimal);
      if (networkConfig) {
        const client = createPublicClient({
          chain: networkConfig.chain,
          transport: http(networkConfig.rpcUrl),
        });
        setPublicClient(client);

        logger.log('🔍 [setupNetworkInfo] Created public client for network:', {
          chainId: chainIdDecimal,
          networkName: networkConfig.chain.name,
        });
      }

      // Check deployment support
      const networkSupported = isPimlicoSupportedForChain(chainIdDecimal);
      const hasAAProvider = !!web3Auth?.accountAbstractionProvider;

      logger.log('🔍 [setupNetworkInfo] Deployment support check:', {
        chainId: chainIdDecimal,
        networkSupported,
        hasAAProvider,
        deploymentSupported: networkSupported && hasAAProvider,
      });

      if (networkSupported && !hasAAProvider) {
        logger.log(
          `ℹ️ Network ${chainIdDecimal} supports AA but AccountAbstractionProvider not available`
        );
      }

      setDeploymentSupported(networkSupported && hasAAProvider);
    } catch (error) {
      logger.error(
        '🔍 [setupNetworkInfo] Failed to get network info from Web3Auth:',
        error
      );
      setCurrentChainId(null);
      setPublicClient(null);
      setDeploymentSupported(false);
    }
  }, [web3Auth?.provider, web3Auth?.accountAbstractionProvider, isWeb3AuthConnected]);

  // Effect to get current chain ID from Web3Auth and setup public client
  useEffect(() => {
    setupNetworkInfo();
  }, [setupNetworkInfo]);

  // Check deployment status with direct bytecode checking
  const checkDeploymentStatus = useCallback(async (): Promise<boolean> => {
    if (!state.smartAccountAddress || !publicClient) {
      return false;
    }

    try {
      logger.log('🔍 Checking smart account deployment status via bytecode...');
      const isDeployed = await isSmartAccountDeployed(
        state.smartAccountAddress,
        publicClient as Parameters<typeof isSmartAccountDeployed>[1]
      );

      setState((prev) => ({ ...prev, isDeployed }));

      logger.log(`✅ Smart account deployment status: ${isDeployed ? 'DEPLOYED' : 'NOT DEPLOYED'}`);
      return isDeployed;
    } catch (error) {
      logger.error('❌ Failed to check deployment status:', error);
      setState((prev) => ({
        ...prev,
        isDeployed: false,
        deploymentError: error as Error,
      }));
      return false;
    }
  }, [state.smartAccountAddress, publicClient]);

  // Declare these functions before deploySmartAccount to avoid dependency issues
  const verifyProviderStateInternal = useCallback(async (): Promise<boolean> => {
    if (!web3Auth?.provider || !currentChainId) {
      return false;
    }

    try {
      logger.log('🔍 Verifying provider state consistency...');
      
      // Check surface level chain ID
      const surfaceChainId = (await web3Auth.provider.request({
        method: 'eth_chainId',
      })) as string;
      const surfaceChainIdDecimal = parseInt(surfaceChainId, 16);
      
      // Check if AccountAbstractionProvider is properly configured
      const hasAAProvider = !!web3Auth?.accountAbstractionProvider;
      
      // Verify network support
      const networkSupported = isPimlicoSupportedForChain(currentChainId);
      
      const isConsistent = surfaceChainIdDecimal === currentChainId && 
                          (!networkSupported || hasAAProvider);
      
      logger.log('🔍 Provider state verification:', {
        currentChainId,
        surfaceChainId: surfaceChainIdDecimal,
        hasAAProvider,
        networkSupported,
        isConsistent,
      });
      
      return isConsistent;
    } catch (error) {
      logger.error('❌ Provider state verification failed:', error);
      return false;
    }
  }, [web3Auth?.provider, web3Auth?.accountAbstractionProvider, currentChainId]);

  const forceProviderRefreshInternal = useCallback(async (): Promise<void> => {
    if (!web3Auth?.provider) {
      return;
    }

    try {
      logger.log('🔄 Forcing Web3Auth provider refresh...');
      setProviderInitialized(false);
      
      // Add a delay to allow Web3Auth to stabilize
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // Re-setup network info which will trigger all necessary updates
      await setupNetworkInfo();
      
      // Verify the provider state after refresh
      const isConsistent = await verifyProviderStateInternal();
      if (!isConsistent) {
        logger.warn('⚠️ Provider state still inconsistent after refresh');
      } else {
        logger.log('✅ Provider state refreshed successfully');
        setProviderInitialized(true);
      }
    } catch (error) {
      logger.error('❌ Failed to force provider refresh:', error);
    }
  }, [web3Auth?.provider, verifyProviderStateInternal, setupNetworkInfo]);

  // Deploy smart account using AccountAbstractionProvider
  const deploySmartAccount = useCallback(async (): Promise<void> => {
    if (!state.smartAccountAddress || !deploymentSupported) {
      throw new Error('Smart account deployment not available');
    }

    if (state.isDeploying) {
      logger.log('⏳ Deployment already in progress...');
      return;
    }

    if (!web3Auth?.accountAbstractionProvider) {
      logger.warn('⚠️ AccountAbstractionProvider not available - this usually means:');
      logger.warn('  1. NEXT_PUBLIC_PIMLICO_API_KEY is missing');
      logger.warn('  2. Account Abstraction configuration failed');
      logger.warn('  3. Current network is not supported for AA');
      throw new Error('AccountAbstractionProvider not available. Check console for details.');
    }

    setState((prev) => ({
      ...prev,
      isDeploying: true,
      deploymentError: null,
    }));

    try {
      logger.log('🚀 Starting smart account deployment via AccountAbstractionProvider...');
      
      // Verify provider state before deployment
      const providerConsistent = await verifyProviderStateInternal();
      if (!providerConsistent) {
        logger.warn('⚠️ Provider state inconsistent, attempting refresh...');
        await forceProviderRefreshInternal();
        
        // Re-verify after refresh
        const stillInconsistent = !(await verifyProviderStateInternal());
        if (stillInconsistent) {
          throw new Error('Provider state inconsistent. Please refresh the page and try again.');
        }
      }

      // Check if already deployed first (safety check with bytecode)
      const alreadyDeployed = await checkDeploymentStatus();
      if (alreadyDeployed) {
        logger.log('✅ Smart account is already deployed (verified via bytecode)');
        setState((prev) => ({ ...prev, isDeploying: false }));
        return;
      }

      // Use AccountAbstractionProvider to deploy smart account
      interface BundlerClient {
        sendUserOperation: (args: {
          account: unknown;
          calls: Array<{ to: string; value: bigint; data: string }>;
        }) => Promise<string>;
        waitForUserOperationReceipt: (args: {
          hash: string;
          timeout?: number;
        }) => Promise<{ receipt: { transactionHash: string } }>;
      }
      
      interface AccountAbstractionProvider {
        bundlerClient?: BundlerClient;
        smartAccount?: unknown;
      }
      const aaProvider = web3Auth.accountAbstractionProvider as AccountAbstractionProvider;
      const bundlerClient = aaProvider.bundlerClient;
      const smartAccount = aaProvider.smartAccount;

      if (!bundlerClient || !smartAccount) {
        throw new Error(
          'BundlerClient or SmartAccount not available from AccountAbstractionProvider'
        );
      }

      logger.log('🔄 Sending deployment transaction via bundler...');

      // Send a minimal self-transfer to trigger deployment
      // The AccountAbstractionProvider will automatically include deployment
      const userOpHash = await (bundlerClient as BundlerClient).sendUserOperation({
        account: smartAccount,
        calls: [
          {
            to: state.smartAccountAddress,
            value: BigInt(0), // 0 ETH self-transfer
            data: '0x', // No data needed
          },
        ],
      });

      logger.log('✅ Deployment UserOperation sent:', userOpHash);

      // Wait for UserOperation receipt
      logger.log('⏳ Waiting for UserOperation receipt...');
      const receipt = await (bundlerClient as BundlerClient).waitForUserOperationReceipt({
        hash: userOpHash,
        timeout: 120000, // 2 minutes timeout
      });

      logger.log('📥 UserOperation confirmed:', receipt.receipt.transactionHash);

      // Verify deployment with bytecode check
      const verifyDeployed = await checkDeploymentStatus();
      if (!verifyDeployed) {
        logger.warn('⚠️ UserOperation completed but bytecode check shows not deployed');
      }

      setState((prev) => ({
        ...prev,
        isDeploying: false,
        isDeployed: verifyDeployed,
        deploymentError: null,
      }));

      logger.log('🎉 Smart account deployed successfully via AccountAbstractionProvider!');
    } catch (error) {
      logger.error('❌ Smart account deployment failed:', error);

      let errorMessage = 'Smart account deployment failed';

      if (error instanceof Error) {
        if (error.message?.includes('User rejected')) {
          errorMessage = 'Deployment cancelled by user';
        } else if (error.message?.includes('insufficient funds')) {
          errorMessage = 'Insufficient funds for deployment. Please contact support.';
        } else if (error.message?.includes('timeout')) {
          errorMessage = 'Deployment transaction timed out';
        } else if (error.message?.includes('AccountAbstractionProvider not available')) {
          errorMessage = 'Smart account deployment not configured. Please check API keys.';
        } else {
          errorMessage = `Deployment failed: ${error.message}`;
        }
      }

      setState((prev) => ({
        ...prev,
        isDeploying: false,
        deploymentError: new Error(errorMessage),
      }));
      throw new Error(errorMessage);
    }
  }, [
    state.smartAccountAddress,
    state.isDeploying,
    deploymentSupported,
    web3Auth?.accountAbstractionProvider,
    checkDeploymentStatus,
    verifyProviderStateInternal,
    forceProviderRefreshInternal,
  ]);

  const refreshSmartAccount = useCallback(async () => {
    if (state.isLoading) return;

    try {
      setState((prev) => ({ ...prev, isLoading: true, error: null }));

      if (!web3Auth || !isWeb3AuthConnected) {
        logger.log('🔍 Web3Auth smart account requirements not met:', {
          hasWeb3Auth: !!web3Auth,
          isWeb3AuthConnected,
        });

        setState({
          smartAccountAddress: null,
          isSmartAccountReady: false,
          isLoading: false,
          error: null,
          isDeployed: null,
          isDeploying: false,
          deploymentError: null,
          deploymentSupported: false,
        });
        return;
      }

      logger.log(
        '[DEBUG useWeb3AuthSmartAccount] Web3Auth connected, getting smart account address...'
      );

      // Get smart account address directly from Web3Auth provider
      try {
        const accounts = (await web3Auth.provider!.request({
          method: 'eth_accounts',
        })) as string[];

        if (accounts && accounts.length > 0) {
          const smartAccountAddress = accounts[0] as Address;

          logger.log(
            '[DEBUG useWeb3AuthSmartAccount] ✅ Smart account address found:',
            smartAccountAddress
          );

          setState((prev) => ({
            ...prev,
            smartAccountAddress,
            isSmartAccountReady: true,
            isLoading: false,
            error: null,
            deploymentSupported,
          }));

          logger.log('[DEBUG useWeb3AuthSmartAccount] Smart account ready:', {
            smartAccountAddress,
            deploymentSupported,
            hasAccountAbstractionProvider: !!web3Auth?.accountAbstractionProvider,
          });
        } else {
          logger.log('[DEBUG useWeb3AuthSmartAccount] ❌ No accounts found from Web3Auth provider');
          setState({
            smartAccountAddress: null,
            isSmartAccountReady: false,
            isLoading: false,
            error: new Error('Smart account address not available'),
            isDeployed: null,
            isDeploying: false,
            deploymentError: null,
            deploymentSupported,
          });
        }
      } catch (providerError) {
        logger.error(
          '[DEBUG useWeb3AuthSmartAccount] Error getting accounts from Web3Auth provider:',
          providerError
        );
        setState({
          smartAccountAddress: null,
          isSmartAccountReady: false,
          isLoading: false,
          error: providerError as Error,
          isDeployed: null,
          isDeploying: false,
          deploymentError: null,
          deploymentSupported,
        });
      }
    } catch (error) {
      logger.error('[DEBUG useWeb3AuthSmartAccount] Error accessing smart account:', error);
      setState({
        smartAccountAddress: null,
        isSmartAccountReady: false,
        isLoading: false,
        error: error as Error,
        isDeployed: null,
        isDeploying: false,
        deploymentError: null,
        deploymentSupported,
      });
    }
  }, [web3Auth, isWeb3AuthConnected, state.isLoading, deploymentSupported]);

  // Auto-refresh smart account when Web3Auth connects
  useEffect(() => {
    if (isWeb3AuthConnected && !state.isSmartAccountReady && !state.isLoading) {
      logger.log('🔄 Auto-refreshing Web3Auth smart account...');
      refreshSmartAccount();
    }
  }, [isWeb3AuthConnected, state.isSmartAccountReady, state.isLoading, refreshSmartAccount]);

  // Auto-check deployment status when smart account becomes ready
  useEffect(() => {
    if (
      state.isSmartAccountReady &&
      state.smartAccountAddress &&
      state.isDeployed === null &&
      !state.isDeploying
    ) {
      logger.log('🔄 Auto-checking deployment status via bytecode...');
      checkDeploymentStatus();
    }
  }, [
    state.isSmartAccountReady,
    state.smartAccountAddress,
    state.isDeployed,
    state.isDeploying,
    checkDeploymentStatus,
  ]);

  // Reset deployment state when disconnected
  useEffect(() => {
    if (!isWeb3AuthConnected) {
      setState((prev) => ({
        ...prev,
        smartAccountAddress: null,
        isSmartAccountReady: false,
        isLoading: false,
        error: null,
        isDeployed: null,
        isDeploying: false,
        deploymentError: null,
        deploymentSupported: false,
      }));
    }
  }, [isWeb3AuthConnected]);

  // Update deployment support when network changes
  useEffect(() => {
    setState((prev) => ({ ...prev, deploymentSupported }));
  }, [deploymentSupported]);

  // Refresh network state after a network switch
  const refreshNetworkState = useCallback(async () => {
    if (!web3Auth?.provider || !isWeb3AuthConnected) {
      return;
    }

    try {
      logger.log('🔄 Refreshing network state after switch...');
      
      // Get current chain ID from Web3Auth provider
      const chainIdHex = (await web3Auth.provider.request({
        method: 'eth_chainId',
      })) as string;
      const chainIdDecimal = parseInt(chainIdHex, 16);

      logger.log('🔍 [refreshNetworkState] New chain ID:', {
        chainIdHex,
        chainIdDecimal,
      });

      setCurrentChainId(chainIdDecimal);

      // Create public client for this network
      const networkConfig = await getNetworkConfig(chainIdDecimal);
      if (networkConfig) {
        const client = createPublicClient({
          chain: networkConfig.chain,
          transport: http(networkConfig.rpcUrl),
        });
        setPublicClient(client);

        logger.log('🔍 [refreshNetworkState] Created new public client for network:', {
          chainId: chainIdDecimal,
          networkName: networkConfig.chain.name,
        });
      }

      // Check deployment support
      const networkSupported = isPimlicoSupportedForChain(chainIdDecimal);
      const hasAAProvider = !!web3Auth?.accountAbstractionProvider;

      logger.log('🔍 [refreshNetworkState] Updated deployment support:', {
        chainId: chainIdDecimal,
        networkSupported,
        hasAAProvider,
        deploymentSupported: networkSupported && hasAAProvider,
      });

      setDeploymentSupported(networkSupported && hasAAProvider);
      
      // Reset deployment status since we're on a new network
      setState((prev) => ({
        ...prev,
        isDeployed: null,
        deploymentSupported: networkSupported && hasAAProvider,
      }));

      // Refresh smart account info
      await refreshSmartAccount();
    } catch (error) {
      logger.error('❌ Failed to refresh network state:', error);
    }
  }, [web3Auth?.provider, web3Auth?.accountAbstractionProvider, isWeb3AuthConnected, refreshSmartAccount]);


  return {
    ...state,
    refreshSmartAccount,
    checkDeploymentStatus,
    deploySmartAccount,
    refreshNetworkState,
    verifyProviderState: verifyProviderStateInternal,
    forceProviderRefresh: forceProviderRefreshInternal,
  };
}
