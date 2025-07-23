'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Loader2, RefreshCw } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { useWeb3Auth, useWeb3AuthUser } from '@web3auth/modal/react';
import { useState, useEffect, useCallback } from 'react';
import { type Address } from 'viem';
import { PublicProductTokenResponse } from '@/types/product';
import { useToast } from '@/components/ui/use-toast';
import { getUSDCAddress } from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';
// Hook for Web3Auth wallet and USDC balance - following existing patterns
function useWeb3AuthUSDCBalance(productNetwork?: PublicProductTokenResponse) {
  const [walletAddress, setWalletAddress] = useState<Address | null>(null);
  const [isLoadingWallet, setIsLoadingWallet] = useState(false);
  const [formattedBalance, setFormattedBalance] = useState('0.00');
  const [isLoading, setIsLoading] = useState(false);
  const [isRequestingFaucet, setIsRequestingFaucet] = useState(false);
  const [lastFaucetRequest, setLastFaucetRequest] = useState<Date | null>(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem('lastFaucetRequest');
      return stored ? new Date(stored) : null;
    }
    return null;
  });

  const { web3Auth, isConnected } = useWeb3Auth();
  const { userInfo } = useWeb3AuthUser();
  const { toast } = useToast();

  // Rate limiting: 5 minutes between requests (for testing)
  const FAUCET_RATE_LIMIT_MS = 5 * 60 * 1000;
  const canRequestFaucet =
    !lastFaucetRequest || Date.now() - lastFaucetRequest.getTime() >= FAUCET_RATE_LIMIT_MS;
  const timeUntilNextRequest = lastFaucetRequest
    ? Math.max(0, FAUCET_RATE_LIMIT_MS - (Date.now() - lastFaucetRequest.getTime()))
    : null;

  // Get USDC contract address dynamically from network configuration
  const getUsdcContractAddress = async (chainId: number): Promise<Address | null> => {
    return await getUSDCAddress(chainId);
  };

  // Get wallet address effect
  useEffect(() => {
    async function getWalletAddress() {
      if (!isConnected || !web3Auth?.provider || !userInfo) {
        setWalletAddress(null);
        return;
      }

      try {
        setIsLoadingWallet(true);
        const accounts = (await web3Auth.provider.request({
          method: 'eth_accounts',
        })) as string[];

        if (accounts && Array.isArray(accounts) && accounts.length > 0) {
          setWalletAddress(accounts[0] as Address);
        }
      } catch (error) {
        logger.error_sync('❌ Failed to get Web3Auth wallet address:', { error });
        setWalletAddress(null);
      } finally {
        setIsLoadingWallet(false);
      }
    }

    getWalletAddress();
  }, [isConnected, web3Auth?.provider, userInfo]);

  // Fetch USDC balance function
  const refetch = useCallback(async () => {
    if (!walletAddress || !productNetwork || !web3Auth?.provider) {
      logger.log('❌ Missing requirements for balance fetch:', {
        walletAddress: !!walletAddress,
        productNetwork: !!productNetwork,
        provider: !!web3Auth?.provider,
      });
      return;
    }

    const chainIdString = productNetwork.network_chain_id;
    if (!chainIdString) {
      logger.log('❌ Chain ID not found for network:', productNetwork.network_name);
      return;
    }

    const chainId = parseInt(chainIdString, 10);
    if (isNaN(chainId)) {
      logger.log('❌ Invalid chain ID:', chainIdString);
      return;
    }

    const usdcAddress = await getUsdcContractAddress(chainId);
    if (!usdcAddress) {
      logger.log('❌ USDC contract address not found for chain ID:', chainId);
      return;
    }

    try {
      setIsLoading(true);

      // First, check what network we're currently connected to
      const currentChainId = (await web3Auth.provider.request({
        method: 'eth_chainId',
      })) as string;

      logger.log('🔍 Current network details:', {
        productNetworkName: productNetwork.network_name,
        productChainId: productNetwork.network_chain_id,
        currentChainId: currentChainId,
        currentChainIdDecimal: parseInt(currentChainId, 16),
        usdcContractAddress: usdcAddress,
        walletAddress: walletAddress,
      });

      // Get the expected chain ID for this network
      const currentChainIdDecimal = parseInt(currentChainId, 16);

      // Switch network if needed
      if (currentChainIdDecimal !== chainId) {
        logger.log(`🔄 Switching from chain ${currentChainIdDecimal} to ${chainId}`);

        try {
          await web3Auth.provider.request({
            method: 'wallet_switchEthereumChain',
            params: [{ chainId: `0x${chainId.toString(16)}` }],
          });
          logger.log('✅ Network switched successfully');
        } catch (switchError) {
          logger.error_sync('❌ Failed to switch network:', { error: switchError });
          // Continue with current network for now
        }
      }

      // Create contract call data for balanceOf function
      const data = `0x70a08231000000000000000000000000${walletAddress.slice(2)}`;

      logger.log('📞 Making balance call:', {
        to: usdcAddress,
        data: data,
        network: productNetwork.network_name,
      });

      const result = (await web3Auth.provider.request({
        method: 'eth_call',
        params: [
          {
            to: usdcAddress,
            data: data,
          },
          'latest',
        ],
      })) as string;

      logger.log('📞 Raw balance result:', result);

      // Convert hex result to decimal and format (USDC has 6 decimals)
      const hexResult = result === '0x' || !result ? '0x0' : result;

      try {
        const balanceWei = BigInt(hexResult);
        const balanceFormatted = (Number(balanceWei) / 1000000).toFixed(2);
        setFormattedBalance(balanceFormatted);
        logger.log(
          '✅ USDC balance fetched successfully:',
          balanceFormatted,
          'USDC',
          'on',
          productNetwork.network_name
        );
      } catch (_conversionError) {
        logger.error_sync('❌ Failed to convert balance result:', { hexResult });
        setFormattedBalance('0.00');
      }
    } catch (error) {
      logger.error_sync('❌ Failed to fetch USDC balance:', { error });
      setFormattedBalance('0.00');
    } finally {
      setIsLoading(false);
    }
  }, [walletAddress, productNetwork, web3Auth?.provider]);

  // Fetch balance when wallet address or network changes
  useEffect(() => {
    if (walletAddress && productNetwork) {
      refetch();
    }
  }, [walletAddress, productNetwork, refetch]);

  // Request faucet function (following your pattern)
  const requestFaucet = async (address: Address) => {
    if (!productNetwork) {
      toast({
        title: 'Error',
        description: 'Network information not available',
        variant: 'destructive',
      });
      return;
    }

    // Map network name to Circle blockchain identifier
    const getBlockchainIdentifier = (networkName: string) => {
      const networkMap: Record<string, string> = {
        'Polygon Amoy': 'MATIC-AMOY',
        'Ethereum Sepolia': 'ETH-SEPOLIA',
        'Avalanche Fuji': 'AVAX-FUJI',
        'Arbitrum Sepolia': 'ARB-SEPOLIA',
        'Base Sepolia': 'BASE-SEPOLIA',
        'Unichain Sepolia': 'UNI-SEPOLIA',
        'Optimism Sepolia': 'OP-SEPOLIA',
      };
      return networkMap[networkName] || null;
    };

    const blockchainIdentifier = getBlockchainIdentifier(productNetwork.network_name);
    if (!blockchainIdentifier) {
      toast({
        title: 'Error',
        description: `Faucet not available for ${productNetwork.network_name}`,
        variant: 'destructive',
      });
      return;
    }

    if (!canRequestFaucet) {
      const timeLeft = lastFaucetRequest
        ? Math.ceil(
            (FAUCET_RATE_LIMIT_MS - (Date.now() - lastFaucetRequest.getTime())) / (60 * 1000)
          )
        : 0;
      toast({
        title: 'Rate Limited',
        description: `You can request USDC again in ${timeLeft} minute${timeLeft !== 1 ? 's' : ''}`,
        variant: 'destructive',
      });
      return;
    }

    setIsRequestingFaucet(true);

    try {
      const response = await fetch('/api/faucet', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          address,
          blockchain: blockchainIdentifier,
          usdc: true,
        }),
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.message || 'Failed to request USDC from faucet');
      }

      const now = new Date();
      setLastFaucetRequest(now);
      localStorage.setItem('lastFaucetRequest', now.toISOString());

      toast({
        title: 'Success',
        description: 'USDC tokens requested successfully. They should arrive shortly.',
      });

      // Refresh balance after 5 seconds
      setTimeout(() => {
        refetch();
      }, 5000);
    } catch (error) {
      toast({
        title: 'Error',
        description: error instanceof Error ? error.message : 'Failed to request faucet',
        variant: 'destructive',
      });
    } finally {
      setIsRequestingFaucet(false);
    }
  };

  return {
    walletAddress,
    isLoadingWallet,
    isConnected: isConnected && !!userInfo,
    formattedBalance,
    isLoading,
    isRequestingFaucet,
    requestFaucet,
    refetch,
    canRequestFaucet,
    lastFaucetRequest,
    timeUntilNextRequest,
  };
}

interface USDCBalanceCardProps {
  productNetwork?: PublicProductTokenResponse;
}

export function USDCBalanceCard({ productNetwork }: USDCBalanceCardProps) {
  const {
    walletAddress,
    isLoadingWallet,
    isConnected,
    formattedBalance,
    isLoading,
    isRequestingFaucet,
    requestFaucet,
    refetch,
    canRequestFaucet,
    lastFaucetRequest,
    timeUntilNextRequest,
  } = useWeb3AuthUSDCBalance(productNetwork);
  // Format time until next request (5 minute cooldown)
  const formatTimeRemaining = () => {
    if (!timeUntilNextRequest) return null;

    const minutes = Math.floor(timeUntilNextRequest / (60 * 1000));
    const seconds = Math.floor((timeUntilNextRequest % (60 * 1000)) / 1000);

    return `${minutes}m ${seconds}s`;
  };

  const handleRequestFaucet = async () => {
    if (!walletAddress) return;
    await requestFaucet(walletAddress);
  };

  // Hide component when not connected (following your requirement)
  if (!isConnected) {
    return null;
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">USDC Balance</CardTitle>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => refetch()}
            disabled={isLoading}
            title="Refresh balance"
          >
            <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
            <span className="sr-only">Refresh balance</span>
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {isLoadingWallet || isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-8 w-32" />
              {productNetwork && <Skeleton className="h-3 w-40" />}
            </div>
          ) : !walletAddress ? (
            <div className="text-center py-4 text-muted-foreground">
              Unable to get wallet address
            </div>
          ) : (
            <>
              <div className="flex flex-col space-y-1">
                <div className="text-2xl font-bold">{formattedBalance} USDC</div>
                {productNetwork && (
                  <div className="text-xs text-muted-foreground">
                    Network: {productNetwork.network_name}
                  </div>
                )}
              </div>

              {/* Only show faucet for USDC tokens and in development mode */}
              {productNetwork?.token_symbol === 'USDC' &&
                process.env.NODE_ENV === 'development' && (
                  <Button
                    className="w-full"
                    onClick={handleRequestFaucet}
                    disabled={isRequestingFaucet || !canRequestFaucet || !walletAddress}
                  >
                    {isRequestingFaucet ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Requesting USDC...
                      </>
                    ) : !canRequestFaucet ? (
                      `Request again in ${formatTimeRemaining()}`
                    ) : (
                      'Request USDC from Faucet'
                    )}
                  </Button>
                )}

              {lastFaucetRequest && process.env.NODE_ENV === 'development' && (
                <div className="text-xs text-muted-foreground text-center">
                  Last request: {formatDistanceToNow(lastFaucetRequest, { addSuffix: true })}
                </div>
              )}
            </>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
