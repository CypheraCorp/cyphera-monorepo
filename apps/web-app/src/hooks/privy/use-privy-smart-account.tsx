'use client';

import React, { useState, useEffect, useContext, useCallback } from 'react';
import { usePrivy, useWallets, useSignTypedData } from '@privy-io/react-auth';
import { createPublicClient, createWalletClient, custom, http } from 'viem';
import { baseSepolia } from 'viem/chains';
import type { Address } from 'viem';
import { createPimlicoClient } from 'permissionless/clients/pimlico';
import { createBundlerClient, createPaymasterClient } from 'viem/account-abstraction';
import { logger } from '@/lib/core/logger/logger-utils';
import { getNetworkConfig } from '@/lib/web3/dynamic-networks';
import { getDelegationToolkit } from '@/lib/web3/delegation-toolkit-wrapper';

/** Interface returned by custom `usePrivySmartAccount` hook */
interface PrivySmartAccountInterface {
  /** Privy embedded wallet, used as a signer for the smart account */
  eoa: any | undefined;
  /** Bundler client for sending user operations */
  bundlerClient: any | null;
  /** Smart account instance for delegation signing */
  smartAccount: any | null;
  /** Smart account address */
  smartAccountAddress: Address | undefined;
  /** Boolean to indicate whether the smart account state has initialized */
  smartAccountReady: boolean;
  /** Is the smart account deployed on chain */
  isDeployed: boolean | null;
  /** Check deployment status */
  checkDeploymentStatus: () => Promise<boolean>;
  /** Deploy the smart account */
  deploySmartAccount: () => Promise<void>;
  /** Switch to a different network */
  switchNetwork: (chainId: number) => Promise<void>;
  /** Get display name for the provider */
  getDisplayName: () => string;
  /** Get button text */
  getButtonText: () => string;
  /** Check if button should be disabled */
  isButtonDisabled: () => boolean;
  /** Check if authenticated */
  isAuthenticated: boolean;
}

const PrivySmartAccountContext = React.createContext<PrivySmartAccountInterface>({
  eoa: undefined,
  bundlerClient: null,
  smartAccount: null,
  smartAccountAddress: undefined,
  smartAccountReady: false,
  isDeployed: null,
  checkDeploymentStatus: async () => false,
  deploySmartAccount: async () => {},
  switchNetwork: async () => {},
  getDisplayName: () => 'Privy',
  getButtonText: () => 'Subscribe with Privy',
  isButtonDisabled: () => true,
  isAuthenticated: false,
});

export const usePrivySmartAccount = () => {
  return useContext(PrivySmartAccountContext);
};

export const PrivySmartAccountProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  // Get Privy hooks
  const { ready, authenticated, signMessage } = usePrivy();
  const { wallets } = useWallets();
  const { signTypedData } = useSignTypedData();
  
  // Find the embedded wallet by finding the entry with walletClientType === 'privy'
  const embeddedWallet = wallets.find(
    (wallet) => wallet.walletClientType === 'privy'
  );

  // States to store the smart account and its status
  const [eoa, setEoa] = useState<any | undefined>();
  const [bundlerClient, setBundlerClient] = useState<any | null>(null);
  const [pimlicoClient, setPimlicoClient] = useState<any | null>(null);
  const [smartAccount, setSmartAccount] = useState<any | null>(null);
  const [smartAccountAddress, setSmartAccountAddress] = useState<Address | undefined>();
  const [smartAccountReady, setSmartAccountReady] = useState(false);
  const [isDeployed, setIsDeployed] = useState<boolean | null>(null);
  const [currentChainId, setCurrentChainId] = useState<number>(baseSepolia.id);

  // Check deployment status
  const checkDeploymentStatus = useCallback(async (): Promise<boolean> => {
    if (!smartAccountAddress) return false;

    try {
      const networkConfig = await getNetworkConfig(currentChainId);
      if (!networkConfig) return false;

      const publicClient = createPublicClient({
        chain: networkConfig.chain,
        transport: http(networkConfig.rpcUrl),
      });

      const bytecode = await publicClient.getCode({ address: smartAccountAddress });
      const deployed = !!bytecode && bytecode !== '0x';
      
      setIsDeployed(deployed);
      logger.log(`✅ Smart account deployment status: ${deployed ? 'DEPLOYED' : 'NOT DEPLOYED'}`);
      
      return deployed;
    } catch (error) {
      logger.error('❌ Failed to check deployment status:', error);
      return false;
    }
  }, [smartAccountAddress, currentChainId]);

  // Deploy smart account with server-style retry logic
  const deploySmartAccount = useCallback(async (): Promise<void> => {
    if (!bundlerClient || !smartAccount || !pimlicoClient) {
      throw new Error('Smart account not initialized');
    }

    const MAX_RETRIES = 3;
    const RETRY_DELAY_MS = 2000;
    let retries = 0;
    let userOpHash: string | undefined;

    const onStatusUpdate = (status: string) => logger.log(`📊 ${status}`);

    try {
      logger.log('🚀 Deploying MetaMask smart account...');
      logger.log('📊 Deployment context:', {
        smartAccountAddress,
        bundlerClient: !!bundlerClient,
        pimlicoClient: !!pimlicoClient,
        chainId: currentChainId,
      });
      
      // Check if already deployed
      const alreadyDeployed = await smartAccount.isDeployed();
      if (alreadyDeployed) {
        logger.log('✅ Smart account is already deployed');
        setIsDeployed(true);
        return;
      }

      // Retry loop following server pattern
      while (retries <= MAX_RETRIES) {
        try {
          // Fetch gas prices using server pattern
          logger.log('⛽ Fetching gas prices from Pimlico...');
          const gasInfo = await pimlicoClient.getUserOperationGasPrice();
          
          // Ensure gas prices are BigInt for consistency with Viem
          const gasPrices = {
            maxFeePerGas: BigInt(gasInfo.fast.maxFeePerGas),
            maxPriorityFeePerGas: BigInt(gasInfo.fast.maxPriorityFeePerGas),
          };
          
          onStatusUpdate(`Using gas prices: maxFeePerGas: ${gasPrices.maxFeePerGas.toString()} wei, maxPriorityFeePerGas: ${gasPrices.maxPriorityFeePerGas.toString()} wei`);

          // Bundler client with integrated paymaster support
          logger.log('💰 Using integrated paymaster for gas-free deployment');
          
          // Send UserOperation following exact server pattern - no custom calls for deployment
          onStatusUpdate('Sending UserOperation...');
          const startTime = Date.now();
          
          // Create a minimal valid call to satisfy MetaMask delegation requirements
          // Use a 0-value ETH transfer to the smart account itself as a deployment trigger
          userOpHash = await bundlerClient.sendUserOperation({
            account: smartAccount,
            calls: [
              {
                to: smartAccountAddress!,
                value: BigInt(0), // 0-value transfer to trigger deployment
                data: '0x', // Empty calldata
              }
            ],
            // Always include gas prices like the server
            maxFeePerGas: gasPrices.maxFeePerGas,
            maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas,
          });

          onStatusUpdate(`UserOperation sent: ${userOpHash}`);
          
          // Wait for confirmation
          onStatusUpdate('Waiting for UserOperation confirmation...');
          const receipt = await bundlerClient.waitForUserOperationReceipt({
            hash: userOpHash,
            timeout: 60000,
          });

          if (!receipt.success) {
            throw new Error('UserOperation did not succeed');
          }

          // Verify deployment following server pattern
          const isNowDeployed = await smartAccount.isDeployed();
          if (isNowDeployed) {
            onStatusUpdate(`Smart Account ${smartAccountAddress} is confirmed deployed.`);
          } else {
            // Double-check with bytecode like server implementation
            const networkConfig = await getNetworkConfig(currentChainId);
            if (networkConfig) {
              const publicClient = createPublicClient({
                chain: networkConfig.chain,
                transport: http(networkConfig.rpcUrl),
              });
              const code = await publicClient.getCode({ address: smartAccountAddress! });
              if (!code || code === '0x') {
                onStatusUpdate(`Warning: SA ${smartAccountAddress} bytecode not found after UserOp, but UserOp reported success.`);
              } else {
                onStatusUpdate(`Bytecode found at ${smartAccountAddress}. Assuming deployed despite isDeployed() returning false.`);
              }
            }
          }

          const transactionHash = receipt.receipt.transactionHash;
          const totalTime = (Date.now() - startTime) / 1000;
          onStatusUpdate(`Transaction confirmed in ${totalTime}s: ${transactionHash}`);

          logger.log('✅ Smart account deployed successfully! Tx:', transactionHash);
          setIsDeployed(true);
          return;

        } catch (error: any) {
          const errorMessage = `Error during UserOperation (hash: ${userOpHash || 'N/A'}): ${error.message || 'Unknown error'}`;
          
          if (retries < MAX_RETRIES) {
            retries++;
            onStatusUpdate(`Retry ${retries}/${MAX_RETRIES} after error: ${errorMessage}`);
            await new Promise(resolve => setTimeout(resolve, RETRY_DELAY_MS));
            continue;
          }

          // Check deployment status on error
          try {
            const isDeployed = await smartAccount.isDeployed();
            onStatusUpdate(`Smart account deployment status on error: ${isDeployed}`);
          } catch {
            // Ignore deployment check errors
          }

          logger.error('❌ UserOperation failed after all retries:', error);
          throw new Error(`Failed to deploy smart account after ${MAX_RETRIES} retries: ${errorMessage}`);
        }
      }

    } catch (error) {
      logger.error('❌ Failed to deploy smart account:', error);
      throw error;
    }
  }, [bundlerClient, smartAccount, smartAccountAddress, pimlicoClient, currentChainId]);

  // Switch network
  const switchNetwork = useCallback(async (chainId: number): Promise<void> => {
    if (!embeddedWallet) {
      throw new Error('No embedded wallet available');
    }

    try {
      await embeddedWallet.switchChain(chainId);
      setCurrentChainId(chainId);
      logger.log(`✅ Switched to chain ${chainId}`);
    } catch (error) {
      logger.error('❌ Failed to switch network:', error);
      throw error;
    }
  }, [embeddedWallet]);

  // Provider interface methods
  const getDisplayName = useCallback(() => 'Privy', []);
  
  const getButtonText = useCallback(() => {
    if (!authenticated) return 'Sign In with Privy';
    if (!smartAccountReady) return 'Initializing...';
    return 'Subscribe with Privy';
  }, [authenticated, smartAccountReady]);

  const isButtonDisabled = useCallback(() => {
    return !authenticated || !smartAccountReady;
  }, [authenticated, smartAccountReady]);

  // Create smart account when embedded wallet is available
  useEffect(() => {
    const createSmartWallet = async (eoa: any) => {
      logger.log('🚀 Starting smart account creation process...', {
        eoa: eoa?.address,
        ready,
        authenticated,
      });
      
      setEoa(eoa);
      
      try {
        // Get an EIP1193 provider for the EOA
        logger.log('📱 Getting EIP1193 provider from EOA...');
        const eip1193provider = await eoa.getEthereumProvider();
        
        // Get current chain from wallet
        const chainId = await eoa.chainId;
        let chainIdDecimal: number;
        
        // Parse chain ID - handle CAIP-2 format (eip155:84532) or hex format
        if (typeof chainId === 'string') {
          if (chainId.includes(':')) {
            // CAIP-2 format: eip155:84532
            const parts = chainId.split(':');
            chainIdDecimal = parseInt(parts[1], 10);
          } else if (chainId.startsWith('0x')) {
            // Hex format: 0x14a34
            chainIdDecimal = parseInt(chainId, 16);
          } else {
            // Decimal string: "84532"
            chainIdDecimal = parseInt(chainId, 10);
          }
        } else {
          chainIdDecimal = chainId;
        }
        
        logger.log('🔗 Chain ID:', { 
          raw: chainId, 
          parsed: chainIdDecimal,
          expectedBaseSepolia: baseSepolia.id 
        });
        
        // Default to Base Sepolia if chain ID is not available
        const targetChainId = chainIdDecimal || baseSepolia.id;
        logger.log('🎯 Target chain ID:', targetChainId);
        
        let networkConfig = await getNetworkConfig(targetChainId);
        
        // Fallback to hardcoded Base Sepolia config if dynamic fetch fails
        if (!networkConfig && targetChainId === baseSepolia.id) {
          logger.log('⚠️ Using fallback Base Sepolia configuration');
          const infuraApiKey = process.env.NEXT_PUBLIC_INFURA_API_KEY;
          networkConfig = {
            chain: baseSepolia,
            rpcUrl: infuraApiKey 
              ? `https://base-sepolia.infura.io/v3/${infuraApiKey}`
              : 'https://sepolia.base.org',
            circleNetworkType: 'BASE-SEPOLIA',
            isPimlicoSupported: true,
            isCircleSupported: true,
            tokens: [{
              address: '0x036CbD53842c5426634e7929541eC2318f3dCF7e' as Address,
              symbol: 'USDC',
              name: 'USD Coin',
              decimals: 6,
              isGasToken: false,
            }],
          };
        }
        
        if (!networkConfig) {
          logger.error('❌ Network not supported:', targetChainId);
          setSmartAccountReady(false);
          return;
        }

        logger.log('🌐 Network config:', {
          name: networkConfig.chain.name,
          id: networkConfig.chain.id,
          rpcUrl: networkConfig.rpcUrl,
        });

        // Create a public client for the blockchain interactions
        const publicClient = createPublicClient({
          chain: networkConfig.chain,
          transport: http(networkConfig.rpcUrl),
        });

        // Load the delegation toolkit
        let delegationToolkit;
        try {
          logger.log('📦 Loading MetaMask delegation toolkit...');
          delegationToolkit = await getDelegationToolkit();
          logger.log('✅ Delegation toolkit loaded successfully');
        } catch (error) {
          logger.error('❌ Failed to load delegation toolkit:', error);
          setSmartAccountReady(false);
          return;
        }

        // Verify factory contract exists before creating smart account
        const FACTORY_ADDRESS = '0x69Aa2f9fe1572F1B640E1bbc512f5c3a734fc77c' as Address;
        try {
          logger.log('🏭 Verifying factory contract exists...');
          const factoryCode = await publicClient.getCode({ address: FACTORY_ADDRESS });
          if (!factoryCode || factoryCode === '0x') {
            throw new Error(`Factory contract not deployed at ${FACTORY_ADDRESS}`);
          }
          logger.log('✅ Factory contract verified at:', FACTORY_ADDRESS);
        } catch (factoryError) {
          logger.error('❌ Factory contract verification failed:', factoryError);
          throw new Error(`Factory contract verification failed: ${factoryError}`);
        }

        // Create MetaMask Hybrid Smart Account
        logger.log('🔧 Creating MetaMask smart account...', {
          implementation: 'Hybrid',
          signatoryAddress: eoa.address,
          factoryAddress: FACTORY_ADDRESS,
        });
        
        const { toMetaMaskSmartAccount, Implementation } = delegationToolkit;
        
        logger.log('📊 Delegation toolkit exports:', {
          hasToMetaMaskSmartAccount: !!toMetaMaskSmartAccount,
          hasImplementation: !!Implementation,
          implementationKeys: Implementation ? Object.keys(Implementation) : [],
        });
        
        // Create smart account with proper error handling
        let metaMaskSmartAccount;
        try {
          logger.log('🔨 Calling toMetaMaskSmartAccount with params:', {
            hasClient: !!publicClient,
            signatoryAddress: eoa.address,
            implementation: 'Hybrid',
          });
          
          // OFFICIAL PRIVY-VIEM INTEGRATION: Use the documented pattern from Privy's Viem integration guide
          // This creates a proper Viem WalletClient that should be fully compatible with MetaMask Delegation Toolkit
          logger.log('🔧 Creating Viem WalletClient using official Privy integration pattern...');
          
          // Get the EIP1193 provider from the Privy wallet
          logger.log('📱 Getting EIP1193 provider from Privy wallet...');
          const provider = await eoa.getEthereumProvider();
          
          // Create Viem WalletClient following official Privy documentation
          const walletClient = createWalletClient({
            account: eoa.address as Address,
            chain: networkConfig.chain,
            transport: custom(provider),
          });
          
          logger.log('💼 Created Viem WalletClient:', {
            hasWalletClient: !!walletClient,
            chain: walletClient.chain?.name,
            account: walletClient.account,
            transport: !!walletClient.transport,
          });
          
          metaMaskSmartAccount = await toMetaMaskSmartAccount({
            client: publicClient,
            implementation: Implementation.Hybrid,
            signatory: { account: walletClient },  // Use the official Viem WalletClient
            deploySalt: '0x' as `0x${string}`,
            deployParams: [
              eoa.address as Address,  // Owner address
              [],  // Empty initial permissions  
              [],  // Empty initial permissions
              []   // Empty initial permissions
            ],
          });
          
          // Validate that the smart account was created properly
          logger.log('🔍 Smart account creation validation:', {
            address: metaMaskSmartAccount.address,
            signatoryAddress: eoa.address,
            hasWalletClient: !!walletClient,
            expectedOwner: eoa.address,
            walletClientAccount: walletClient.account,
          });
        } catch (smartAccountError) {
          logger.error('❌ Failed to create MetaMask smart account:', smartAccountError);
          logger.error('Smart account creation error details:', {
            message: smartAccountError instanceof Error ? smartAccountError.message : 'Unknown error',
            stack: smartAccountError instanceof Error ? smartAccountError.stack : undefined,
          });
          throw smartAccountError;
        }

        logger.log('📍 MetaMask smart account created:', {
          address: metaMaskSmartAccount.address,
          type: metaMaskSmartAccount.constructor?.name || 'Unknown',
          isDeployedMethod: typeof metaMaskSmartAccount.isDeployed === 'function',
          availableMethods: Object.getOwnPropertyNames(Object.getPrototypeOf(metaMaskSmartAccount || {})).filter(m => typeof (metaMaskSmartAccount as any)[m] === 'function').slice(0, 10),
        });

        // Get Pimlico configuration
        const pimlicoApiKey = process.env.NEXT_PUBLIC_PIMLICO_API_KEY;
        
        if (!pimlicoApiKey) {
          logger.warn('⚠️ Pimlico API key not configured');
          setSmartAccount(metaMaskSmartAccount);
          setSmartAccountAddress(metaMaskSmartAccount.address);
          setSmartAccountReady(true);
          // Check deployment status
          const deployed = await metaMaskSmartAccount.isDeployed();
          setIsDeployed(deployed);
          return;
        }
        
        // Use Pimlico v2 API format for all chains
        const bundlerUrl = `https://api.pimlico.io/v2/${networkConfig.chain.id}/rpc?apikey=${pimlicoApiKey}`;
        logger.log('🌐 Using Pimlico v2 API for chain', networkConfig.chain.id);

        logger.log('🔑 Pimlico configured:', {
          bundlerUrl,
          chain: networkConfig.chain.name,
          chainId: networkConfig.chain.id,
        });

        // Test Pimlico connectivity first
        try {
          const testResponse = await fetch(bundlerUrl, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              jsonrpc: '2.0',
              method: 'eth_chainId',
              params: [],
              id: 1,
            }),
          });
          const testResult = await testResponse.json();
          logger.log('🤝 Pimlico bundler connectivity test:', {
            status: testResponse.status,
            chainId: testResult.result,
            hasError: !!testResult.error,
          });
        } catch (connectError) {
          logger.error('❌ Pimlico bundler connectivity test failed:', connectError);
        }

        // Create paymaster client for sponsored transactions
        const paymasterClient = createPaymasterClient({
          transport: http(bundlerUrl),
        });

        // Create bundler client with paymaster integration
        logger.log('🔧 Creating bundler client with paymaster...');
        
        const bundlerClient = createBundlerClient({
          chain: networkConfig.chain,
          transport: http(bundlerUrl),
          paymaster: paymasterClient,
        });

        // Create Pimlico client for gas price operations
        const pimlicoClient = createPimlicoClient({
          chain: networkConfig.chain,
          transport: http(bundlerUrl),
        });
        
        logger.log('✅ Bundler and Pimlico clients created:', {
          hasSendUserOperation: typeof bundlerClient.sendUserOperation === 'function',
          hasWaitForUserOperationReceipt: typeof bundlerClient.waitForUserOperationReceipt === 'function',
          hasPimlicoGasPrice: typeof pimlicoClient.getUserOperationGasPrice === 'function',
          hasPimlicoSponsor: typeof pimlicoClient.sponsorUserOperation === 'function',
        });

        setPimlicoClient(pimlicoClient);
        setBundlerClient(bundlerClient);
        setSmartAccount(metaMaskSmartAccount);
        setSmartAccountAddress(metaMaskSmartAccount.address);
        setSmartAccountReady(true);
        setCurrentChainId(networkConfig.chain.id);

        // Check if already deployed
        const deployed = await metaMaskSmartAccount.isDeployed();
        setIsDeployed(deployed);

        logger.log('✅ Privy MetaMask smart account initialized:', {
          address: metaMaskSmartAccount.address,
          deployed,
          network: networkConfig.chain.name,
          hasBundlerClient: !!bundlerClient,
          hasPimlicoClient: !!pimlicoClient,
        });
      } catch (error) {
        logger.error('❌ Failed to create smart account:', error);
        logger.error('Error details:', {
          message: error instanceof Error ? error.message : 'Unknown error',
          stack: error instanceof Error ? error.stack : undefined,
        });
        setSmartAccountReady(false);
      }
    };

    if (ready && authenticated && embeddedWallet) {
      logger.log('🎯 Conditions met for smart account creation:', {
        ready,
        authenticated,
        hasEmbeddedWallet: !!embeddedWallet,
        embeddedWalletAddress: embeddedWallet?.address,
      });
      createSmartWallet(embeddedWallet);
    } else {
      logger.log('⏳ Waiting for conditions to create smart account:', {
        ready,
        authenticated,
        hasEmbeddedWallet: !!embeddedWallet,
      });
      // Reset state when not authenticated
      setEoa(undefined);
      setSmartAccount(null);
      setBundlerClient(null);
      setPimlicoClient(null);
      setSmartAccountAddress(undefined);
      setSmartAccountReady(false);
      setIsDeployed(null);
    }
  }, [ready, authenticated, embeddedWallet, signMessage, signTypedData]);

  // Auto-check deployment status
  useEffect(() => {
    if (smartAccountReady && smartAccountAddress && isDeployed === null) {
      checkDeploymentStatus();
    }
  }, [smartAccountReady, smartAccountAddress, isDeployed, checkDeploymentStatus]);

  return (
    <PrivySmartAccountContext.Provider
      value={{
        eoa,
        bundlerClient,
        smartAccount,
        smartAccountAddress,
        smartAccountReady,
        isDeployed,
        checkDeploymentStatus,
        deploySmartAccount,
        switchNetwork,
        getDisplayName,
        getButtonText,
        isButtonDisabled,
        isAuthenticated: authenticated,
      }}
    >
      {children}
    </PrivySmartAccountContext.Provider>
  );
};