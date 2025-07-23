'use client';

import { ReactNode, useState, useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Web3Auth imports
import { Web3AuthProvider, type Web3AuthContextConfig } from '@web3auth/modal/react';
import { WagmiProvider } from '@web3auth/modal/react/wagmi'; // Use Web3Auth's Wagmi provider
import { WEB3AUTH_NETWORK } from '@web3auth/modal';

import type { NetworkWithTokensResponse } from '@/types/network';
import { NetworkProvider } from '@/contexts/network-context';
import { NetworkSyncProvider } from '@/components/providers/network-sync-provider';
import { fetchNetworks, getAllNetworkConfigs, getPimlicoUrls } from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';

// Create a TanStack Query client
const queryClient = new QueryClient();

// Web3Auth configuration
const clientId = process.env.NEXT_PUBLIC_WEB3AUTH_CLIENT_ID || '';
const pimlicoApiKey = process.env.NEXT_PUBLIC_PIMLICO_API_KEY || '';

if (!pimlicoApiKey) {
  logger.warn('⚠️ NEXT_PUBLIC_PIMLICO_API_KEY is not set. Account Abstraction features will be disabled.');
}

interface Web3ProviderProps {
  children: ReactNode;
}

export function Web3Provider({ children }: Web3ProviderProps) {
  const [networks, setNetworks] = useState<NetworkWithTokensResponse[]>([]);
  const [web3AuthConfig, setWeb3AuthConfig] = useState<Web3AuthContextConfig | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    async function initializeNetworks() {
      try {
        logger.log('🔄 Fetching dynamic network configuration...');

        // Fetch networks from backend
        const networkData = await fetchNetworks();
        setNetworks(networkData);

        // Get all network configs
        const networkConfigs = await getAllNetworkConfigs();

        // Build Web3Auth account abstraction configuration
        const accountAbstractionChains = pimlicoApiKey
          ? Array.from(networkConfigs.values())
              .filter((config) => {
                const pimlicoUrls = getPimlicoUrls(config.chain.id, pimlicoApiKey);
                return config.isPimlicoSupported && pimlicoUrls !== null;
              })
              .map((config) => {
                const pimlicoUrls = getPimlicoUrls(config.chain.id, pimlicoApiKey)!;

                return {
                  chainId: `0x${config.chain.id.toString(16)}`,
                  bundlerConfig: {
                    url: pimlicoUrls.bundlerUrl,
                  },
                  paymasterConfig: {
                    url: pimlicoUrls.paymasterUrl,
                  },
                };
              })
          : [];

        logger.log('🔍 Account Abstraction configuration:', {
          pimlicoApiKeySet: !!pimlicoApiKey,
          supportedChains: accountAbstractionChains.map(c => parseInt(c.chainId, 16)),
          totalNetworks: networkConfigs.size,
        });

        // Create Web3Auth configuration
        const web3AuthContextConfig: Web3AuthContextConfig = {
          web3AuthOptions: {
            clientId,
            web3AuthNetwork: WEB3AUTH_NETWORK.SAPPHIRE_DEVNET,
            ssr: false,
            // Add account abstraction configuration if we have chains
            ...(accountAbstractionChains.length > 0 && {
              accountAbstractionConfig: {
                chains: accountAbstractionChains,
              },
            }),
            // Set default chain to first available network
            defaultChainId: `0x${Array.from(networkConfigs.keys())[0]?.toString(16) || '1'}`,
          },
        };

        setWeb3AuthConfig(web3AuthContextConfig);
        logger.log('✅ Dynamic network configuration loaded successfully');
      } catch (err) {
        logger.error('❌ Failed to initialize networks:', err);
        setError(err as Error);

        // Fallback to minimal configuration
        const fallbackConfig: Web3AuthContextConfig = {
          web3AuthOptions: {
            clientId,
            web3AuthNetwork: WEB3AUTH_NETWORK.SAPPHIRE_DEVNET,
            ssr: false,
            defaultChainId: '0x1', // Ethereum mainnet as fallback
          },
        };

        setWeb3AuthConfig(fallbackConfig);
      } finally {
        setIsLoading(false);
      }
    }

    initializeNetworks();
  }, []);

  if (isLoading || !web3AuthConfig) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600 mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Loading network configuration...</p>
        </div>
      </div>
    );
  }

  if (error) {
    logger.warn('⚠️ Network initialization error:', { error: error.message });
  }

  return (
    <QueryClientProvider client={queryClient}>
      <Web3AuthProvider config={web3AuthConfig}>
        <WagmiProvider>
          <NetworkProvider networks={networks}>
            <NetworkSyncProvider networks={networks}>{children}</NetworkSyncProvider>
          </NetworkProvider>
        </WagmiProvider>
      </Web3AuthProvider>
    </QueryClientProvider>
  );
}
