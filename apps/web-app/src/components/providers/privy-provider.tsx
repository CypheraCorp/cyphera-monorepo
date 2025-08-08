'use client';

import { ReactNode, useEffect, useState } from 'react';
import { PrivyProvider as BasePrivyProvider } from '@privy-io/react-auth';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { getAllNetworkConfigs } from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';
import { baseSepolia, base, polygon, arbitrum, optimism, sepolia } from 'viem/chains';
import type { Chain } from 'viem/chains';

// Create a TanStack Query client
const queryClient = new QueryClient();

// Privy configuration
const privyAppId = process.env.NEXT_PUBLIC_PRIVY_APP_ID || '';

if (!privyAppId) {
  logger.error('❌ NEXT_PUBLIC_PRIVY_APP_ID is not set');
}

interface PrivyProviderProps {
  children: ReactNode;
}

// Map chain IDs to viem chains for Privy
const chainIdToViemChain: Record<number, Chain> = {
  84532: baseSepolia,
  8453: base,
  137: polygon,
  42161: arbitrum,
  10: optimism,
  11155111: sepolia, // Ethereum Sepolia
};

export function PrivyProvider({ children }: PrivyProviderProps) {
  const [supportedChains, setSupportedChains] = useState<Chain[]>([baseSepolia, sepolia]);
  const [defaultChain, setDefaultChain] = useState<Chain>(baseSepolia);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function initializeNetworks() {
      try {
        logger.log('🔄 Loading dynamic network configuration for Privy...');

        // Get all network configs
        const networkConfigs = await getAllNetworkConfigs();

        // Convert to viem chains for Privy
        const chains: Chain[] = [];
        let firstChain: Chain | null = null;

        for (const [chainId, config] of networkConfigs.entries()) {
          const viemChain = chainIdToViemChain[chainId] || config.chain;
          if (!firstChain) firstChain = viemChain;
          chains.push(viemChain);
        }

        // Always ensure essential chains are included (Base Sepolia and Ethereum Sepolia)
        const essentialChains = [baseSepolia, sepolia];
        const finalChains = [...essentialChains];
        
        // Add any additional chains from dynamic config that aren't already included
        for (const chain of chains) {
          if (!finalChains.some(c => c.id === chain.id)) {
            finalChains.push(chain);
          }
        }

        if (finalChains.length > 0) {
          setSupportedChains(finalChains);
          setDefaultChain(firstChain || baseSepolia);
          logger.log('✅ Privy network configuration loaded:', {
            supportedChains: finalChains.map(c => ({ id: c.id, name: c.name })),
            defaultChain: (firstChain || baseSepolia).name,
            essentialChainsIncluded: essentialChains.map(c => ({ id: c.id, name: c.name })),
          });
        }
      } catch (err) {
        logger.error('❌ Failed to initialize networks for Privy:', err);
        // Keep default fallback
      } finally {
        setIsLoading(false);
      }
    }

    initializeNetworks();
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600 mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Loading Privy configuration...</p>
        </div>
      </div>
    );
  }

  return (
    <QueryClientProvider client={queryClient}>
      <BasePrivyProvider
        appId={privyAppId}
        config={{
          // Appearance
          appearance: {
            theme: 'light' as const,  // Use 'light' or 'dark' instead of 'auto'
            accentColor: '#4F46E5',
            // logo: '/logos/privy-logo.png', // Remove logo to fix 404 error
          },
          
          // Login methods - prioritize embedded wallet creation
          loginMethods: [
            'email',
            'google',
            'twitter',
            'discord',
            'github',
            'apple',
            'wallet', // External wallets as fallback
          ],
          
          // Embedded wallets configuration
          embeddedWallets: {
            createOnLogin: 'users-without-wallets',
            requireUserPasswordOnCreate: false,
            // noPromptOnSignature removed as it's not a valid property
          },
          
          // Network configuration
          defaultChain,
          supportedChains,
          
          // Wallet configuration
          walletConnectCloudProjectId: process.env.NEXT_PUBLIC_WALLET_CONNECT_PROJECT_ID,
        }}
      >
        {children}
      </BasePrivyProvider>
    </QueryClientProvider>
  );
}