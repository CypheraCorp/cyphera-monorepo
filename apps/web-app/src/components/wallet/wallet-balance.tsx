'use client';

import React, { useState, useEffect } from 'react';
import { usePrivySmartAccount } from '@/hooks/privy/use-privy-smart-account';
import { createPublicClient, http, formatEther, type Address } from 'viem';
import { getNetworkConfig, getUSDCAddress } from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';

interface TokenBalance {
  symbol: string;
  name: string;
  balance: string;
  formattedBalance: string;
  decimals: number;
  address: Address;
  isNative: boolean;
}

interface WalletBalanceProps {
  className?: string;
}

export const WalletBalance: React.FC<WalletBalanceProps> = ({ className = '' }) => {
  const { smartAccountAddress, smartAccountReady, currentChainId } = usePrivySmartAccount();
  
  const [balances, setBalances] = useState<TokenBalance[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchBalances = async () => {
    if (!smartAccountAddress || !smartAccountReady) return;

    try {
      setIsLoading(true);
      setError(null);

      let networkConfig = await getNetworkConfig(currentChainId);
      
      // Fallback to hardcoded configurations if dynamic fetch fails (same as smart account hook)
      if (!networkConfig && currentChainId === 84532) {
        logger.log('⚠️ Using fallback Base Sepolia configuration');
        const infuraApiKey = process.env.NEXT_PUBLIC_INFURA_API_KEY;
        networkConfig = {
          chain: {
            id: 84532,
            name: 'Base Sepolia',
            nativeCurrency: { name: 'Ethereum', symbol: 'ETH', decimals: 18 },
            rpcUrls: {
              default: {
                http: [infuraApiKey 
                  ? `https://base-sepolia.infura.io/v3/${infuraApiKey}`
                  : 'https://sepolia.base.org'
                ]
              }
            }
          } as any,
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
      } else if (!networkConfig && currentChainId === 11155111) {
        logger.log('⚠️ Using fallback Ethereum Sepolia configuration');
        const infuraApiKey = process.env.NEXT_PUBLIC_INFURA_API_KEY;
        networkConfig = {
          chain: {
            id: 11155111,
            name: 'Ethereum Sepolia',
            nativeCurrency: { name: 'Ethereum', symbol: 'ETH', decimals: 18 },
            rpcUrls: {
              default: {
                http: [infuraApiKey 
                  ? `https://sepolia.infura.io/v3/${infuraApiKey}`
                  : 'https://rpc.sepolia.org'
                ]
              }
            }
          } as any,
          rpcUrl: infuraApiKey 
            ? `https://sepolia.infura.io/v3/${infuraApiKey}`
            : 'https://rpc.sepolia.org',
          circleNetworkType: 'ETH-SEPOLIA',
          isPimlicoSupported: true,
          isCircleSupported: true,
          tokens: [{
            address: '0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238' as Address,
            symbol: 'USDC',
            name: 'USD Coin',
            decimals: 6,
            isGasToken: false,
          }],
        };
      }
      
      if (!networkConfig) {
        throw new Error(`Network configuration not found for chain ${currentChainId}`);
      }

      const publicClient = createPublicClient({
        chain: networkConfig.chain,
        transport: http(networkConfig.rpcUrl),
      });

      const tokenBalances: TokenBalance[] = [];

      // Get native token (ETH) balance
      logger.log('🔍 Fetching native token balance...');
      const nativeBalance = await publicClient.getBalance({
        address: smartAccountAddress,
      });

      tokenBalances.push({
        symbol: networkConfig.chain.nativeCurrency.symbol,
        name: networkConfig.chain.nativeCurrency.name,
        balance: nativeBalance.toString(),
        formattedBalance: formatEther(nativeBalance),
        decimals: networkConfig.chain.nativeCurrency.decimals,
        address: '0x0000000000000000000000000000000000000000' as Address,
        isNative: true,
      });

      // Get USDC balance if available
      const usdcAddress = await getUSDCAddress(currentChainId);
      if (usdcAddress) {
        logger.log('🔍 Fetching USDC balance...');
        
        // USDC contract ABI for balanceOf
        const usdcAbi = [
          {
            name: 'balanceOf',
            type: 'function',
            stateMutability: 'view',
            inputs: [{ name: 'account', type: 'address' }],
            outputs: [{ name: 'balance', type: 'uint256' }],
          },
          {
            name: 'decimals',
            type: 'function',
            stateMutability: 'view',
            inputs: [],
            outputs: [{ name: 'decimals', type: 'uint8' }],
          },
        ] as const;

        try {
          const [usdcBalance, decimals] = await Promise.all([
            publicClient.readContract({
              address: usdcAddress,
              abi: usdcAbi,
              functionName: 'balanceOf',
              args: [smartAccountAddress],
            }),
            publicClient.readContract({
              address: usdcAddress,
              abi: usdcAbi,
              functionName: 'decimals',
            }),
          ]);

          const divisor = BigInt(10 ** decimals);
          const formattedBalance = (Number(usdcBalance) / Number(divisor)).toFixed(2);

          tokenBalances.push({
            symbol: 'USDC',
            name: 'USD Coin',
            balance: usdcBalance.toString(),
            formattedBalance,
            decimals: Number(decimals),
            address: usdcAddress,
            isNative: false,
          });
        } catch (usdcError) {
          logger.error('❌ Failed to fetch USDC balance:', usdcError);
        }
      }

      setBalances(tokenBalances);
      logger.log('✅ Balances fetched successfully:', tokenBalances);
    } catch (error) {
      logger.error('❌ Failed to fetch balances:', error);
      setError(error instanceof Error ? error.message : 'Failed to fetch balances');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (smartAccountReady && smartAccountAddress) {
      fetchBalances();
    }
  }, [smartAccountReady, smartAccountAddress, currentChainId]);

  const getNetworkName = () => {
    switch (currentChainId) {
      case 84532:
        return 'Base Sepolia';
      case 11155111:
        return 'Ethereum Sepolia';
      default:
        return `Chain ${currentChainId}`;
    }
  };

  if (!smartAccountReady) {
    return (
      <div className={`bg-gray-50 border border-gray-200 rounded-lg p-4 ${className}`}>
        <div className="text-center text-gray-500">
          <div className="animate-pulse">Initializing wallet...</div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white border border-gray-200 rounded-lg shadow-sm ${className}`}>
      <div className="p-4 border-b border-gray-100">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">Wallet Balance</h3>
          <button
            onClick={fetchBalances}
            disabled={isLoading}
            className="text-sm text-blue-600 hover:text-blue-800 disabled:opacity-50"
          >
            {isLoading ? 'Refreshing...' : '↻ Refresh'}
          </button>
        </div>
        <div className="flex items-center mt-1">
          <div className={`w-2 h-2 rounded-full ${
            currentChainId === 84532 ? 'bg-blue-500' : 'bg-gray-500'
          }`} />
          <span className="ml-2 text-sm text-gray-600">{getNetworkName()}</span>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border-b border-red-100">
          <p className="text-red-800 text-sm">⚠️ {error}</p>
        </div>
      )}

      <div className="p-4">
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2].map((i) => (
              <div key={i} className="animate-pulse">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <div className="w-8 h-8 bg-gray-200 rounded-full"></div>
                    <div>
                      <div className="w-16 h-4 bg-gray-200 rounded"></div>
                      <div className="w-24 h-3 bg-gray-200 rounded mt-1"></div>
                    </div>
                  </div>
                  <div className="w-20 h-4 bg-gray-200 rounded"></div>
                </div>
              </div>
            ))}
          </div>
        ) : balances.length === 0 ? (
          <div className="text-center text-gray-500 py-8">
            <p>No balance information available</p>
            <button
              onClick={fetchBalances}
              className="mt-2 text-blue-600 hover:text-blue-800 text-sm"
            >
              Try again
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            {balances.map((token) => (
              <div key={token.symbol} className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-bold ${
                    token.isNative ? 'bg-blue-500' : 'bg-green-500'
                  }`}>
                    {token.symbol.charAt(0)}
                  </div>
                  <div>
                    <p className="font-medium text-gray-900">{token.symbol}</p>
                    <p className="text-xs text-gray-500">{token.name}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-mono text-gray-900">
                    {token.isNative 
                      ? parseFloat(token.formattedBalance).toFixed(4)
                      : token.formattedBalance
                    }
                  </p>
                  <p className="text-xs text-gray-500 font-mono">
                    {token.symbol}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}

        {smartAccountAddress && (
          <div className="mt-6 pt-4 border-t border-gray-100">
            <p className="text-xs text-gray-500 mb-2">Smart Account Address:</p>
            <div className="flex items-center space-x-2">
              <code className="text-xs bg-gray-100 px-2 py-1 rounded font-mono break-all">
                {smartAccountAddress}
              </code>
              <button
                onClick={() => navigator.clipboard.writeText(smartAccountAddress)}
                className="text-xs text-blue-600 hover:text-blue-800 whitespace-nowrap"
              >
                Copy
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};