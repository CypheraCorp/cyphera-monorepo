'use client';

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { cn } from '@/lib/utils/index';
import { useToast } from '@/components/ui/use-toast';
import { RefreshCw, AlertCircle } from 'lucide-react';
import { WalletResponse } from '@/types/wallet';
import { NetworkWithTokensResponse } from '@/types/network';
import { CircleAPI } from '@/services/cyphera-api/circle';
import { useCircleSDK } from '@/hooks/web3';
import { TokenResponse } from '@/types/token';
import type { UserRequestContext } from '@/services/cyphera-api/api';
import { formatAddress, getBlockchainFromCircleWalletId } from '@/lib/utils/circle';
import { generateExplorerLink } from '@/lib/utils/explorers';
import { logger } from '@/lib/core/logger/logger-utils';

// Simple Progress component since we don't have the shadcn/ui one
const Progress = ({ value = 0, className }: { value?: number; className?: string }) => (
  <div className={cn('w-full bg-secondary overflow-hidden', className)}>
    <div
      className="h-full bg-primary transition-all duration-500 ease-in-out"
      style={{ width: `${Math.max(0, Math.min(100, value))}%` }}
    />
  </div>
);

interface CircleWalletBalancesProps {
  wallet: WalletResponse;
  workspaceId: string;
  networks: NetworkWithTokensResponse[];
}

interface WalletBalance {
  amount: string;
  token: {
    id: string;
    symbol: string;
    decimals: number;
    isNative: boolean;
    blockchain: string;
    address?: string;
    standard?: string;
  } | null;
}

interface BalanceData {
  tokens: WalletBalance[];
  lastUpdated: Date;
}

/**
 * CircleWalletBalances
 *
 * Displays token balances for a Circle wallet.
 * TODO: Refactor fully for multi-network support (pass networks prop, use generateExplorerLink).
 */
export function CircleWalletBalances({ wallet, workspaceId, networks }: CircleWalletBalancesProps) {
  const [balanceData, setBalanceData] = useState<BalanceData | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { toast } = useToast();
  const { userToken } = useCircleSDK();

  const currentNetwork = useMemo(() => {
    if (!wallet.circle_data?.circle_wallet_id || !networks || networks.length === 0) {
      return null;
    }
    const blockchainId = getBlockchainFromCircleWalletId(wallet.circle_data.circle_wallet_id);
    return (
      networks.find(
        (n) => n.network.circle_network_type === blockchainId || n.network.id === blockchainId
      ) || null
    );
  }, [wallet, networks]);

  // Find native token info from the current network's tokens array
  const nativeTokenInfo = useMemo(() => {
    return currentNetwork?.tokens?.find((token) => token.gas_token);
  }, [currentNetwork]);

  const fetchBalances = useCallback(async () => {
    if (!wallet.circle_data || !wallet.circle_data.circle_wallet_id) {
      setError('This is not a Circle wallet or wallet data is missing');
      setIsLoading(false);
      return;
    }

    if (!userToken || !workspaceId) {
      setError('Missing user token or workspace ID for balance check.');
      setIsLoading(false);
      return;
    }

    try {
      setIsLoading(true);
      setError(null);

      const context: UserRequestContext = {
        access_token: userToken,
        workspace_id: workspaceId,
      };

      const circleApi = new CircleAPI();
      const balancesResponse = await circleApi.getWalletBalance(
        context,
        wallet.circle_data.circle_wallet_id,
        { include_all: true }
      );

      if (balancesResponse?.data?.balances) {
        const sortedBalances = balancesResponse.data.balances.sort(
          (a: WalletBalance, b: WalletBalance) => {
            const aIsUsdc = a.token?.symbol?.toLowerCase() === 'usdc';
            const bIsUsdc = b.token?.symbol?.toLowerCase() === 'usdc';
            if (aIsUsdc && !bIsUsdc) return -1;
            if (!aIsUsdc && bIsUsdc) return 1;
            // TODO: Add secondary sort (e.g., native first, then value)
            return 0;
          }
        );

        setBalanceData({ tokens: sortedBalances, lastUpdated: new Date() });
      } else {
        setBalanceData({ tokens: [], lastUpdated: new Date() });
        setError('No balance information available');
      }
    } catch (err) {
      logger.error('Failed to fetch balances:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch balances');
      toast({
        title: 'Error',
        description: 'Failed to load wallet balances',
        variant: 'destructive',
      });
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, [wallet.circle_data, userToken, workspaceId, toast]);

  const getDisplayDecimals = (symbol: string | undefined, amount: number): number => {
    if (!symbol) return 2;
    const isStablecoin =
      symbol.toLowerCase() === 'usdc' ||
      symbol.toLowerCase() === 'usdt' ||
      symbol.toLowerCase() === 'dai';

    if (isStablecoin) return 2;

    if (amount < 0.001 && amount !== 0) return 6;
    if (amount < 0.1 && amount !== 0) return 4;
    if (amount < 1 && amount !== 0) return 3;
    return 2;
  };

  const handleRefresh = () => {
    setIsRefreshing(true);
    fetchBalances();
  };

  const getLastUpdatedText = () => {
    if (!balanceData?.lastUpdated) return 'Never updated';
    const now = new Date();
    const seconds = Math.floor((now.getTime() - balanceData.lastUpdated.getTime()) / 1000);

    if (seconds < 5) return 'Updated just now';
    if (seconds < 60) return `Updated ${seconds} seconds ago`;
    if (seconds < 3600) return `Updated ${Math.floor(seconds / 60)} minutes ago`;
    return `Updated ${Math.floor(seconds / 3600)} hours ago`;
  };

  useEffect(() => {
    fetchBalances();

    const intervalId = setInterval(() => {
      fetchBalances();
    }, 30000);

    return () => clearInterval(intervalId);
  }, [wallet.circle_data?.circle_wallet_id, fetchBalances]);

  // Use chain_id for explorer link generation
  const explorerLink = useMemo(() => {
    return generateExplorerLink(
      networks,
      currentNetwork?.network.chain_id,
      'address',
      wallet.wallet_address
    );
  }, [networks, currentNetwork, wallet.wallet_address]);

  const networkName = currentNetwork?.network.name || 'Unknown Network';
  // Get native currency details from the found native token
  const nativeSymbol = nativeTokenInfo?.symbol || 'NATIVE';
  // Use default 18 if decimals field is missing on native token info
  // TODO: Ensure API response includes decimals for gas_token
  const nativeDecimals = (nativeTokenInfo as TokenResponse & { decimals?: number })?.decimals || 18;

  return (
    <Card className="overflow-hidden">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <div>
          <CardTitle className="text-lg">Wallet Balances ({networkName})</CardTitle>
          {!isLoading && !error && balanceData?.lastUpdated && (
            <CardDescription>{getLastUpdatedText()}</CardDescription>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            disabled={isLoading || isRefreshing}
            className="h-8 w-8 p-0"
          >
            <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
            <span className="sr-only">Refresh</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-8 p-0"
            disabled={!explorerLink}
            asChild={!!explorerLink}
          >
            {explorerLink ? (
              <a href={explorerLink} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="h-4 w-4" />
                <span className="sr-only">View on Explorer</span>
              </a>
            ) : (
              <>
                <ExternalLink className="h-4 w-4" />
                <span className="sr-only">View on Explorer (Unavailable)</span>
              </>
            )}
          </Button>
        </div>
      </CardHeader>

      <CardContent>
        {isLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            {isRefreshing && (
              <div className="pt-2">
                <Progress value={45} className="h-1" />
              </div>
            )}
          </div>
        ) : error ? (
          <div className="space-y-4">
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
            <Button variant="outline" size="sm" onClick={handleRefresh} className="w-full">
              Try Again
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            {!balanceData?.tokens || balanceData.tokens.length === 0 ? (
              <div className="text-center py-6">
                <p className="text-muted-foreground mb-2">No tokens found in this wallet</p>
                <Button variant="outline" size="sm" onClick={handleRefresh}>
                  Refresh
                </Button>
              </div>
            ) : (
              <div className="divide-y">
                {balanceData.tokens.map((balance: WalletBalance) => {
                  const isNative = !balance.token;
                  const symbol = isNative ? nativeSymbol : balance.token?.symbol;
                  // Use dynamic native decimals, falling back to 18
                  const decimals = isNative
                    ? nativeDecimals
                    : balance.token?.decimals || (symbol?.toLowerCase() === 'usdc' ? 6 : 18);
                  const amountInDecimal = balance.amount
                    ? parseFloat(balance.amount) / Math.pow(10, decimals)
                    : 0;
                  const displayDecimals = getDisplayDecimals(symbol, amountInDecimal);
                  const tokenAddress = isNative ? undefined : balance.token?.address;
                  const tokenId = isNative
                    ? `native-${currentNetwork?.network.id}`
                    : balance.token?.id;

                  return (
                    <div
                      key={tokenId || `balance-${symbol}-${balance.amount}`}
                      className="py-3 first:pt-0 last:pb-0"
                    >
                      <div className="flex justify-between items-center">
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{symbol || 'Unknown Token'}</span>
                            {isNative && (
                              <span className="text-xs bg-secondary text-secondary-foreground px-1.5 py-0.5 rounded">
                                Native
                              </span>
                            )}
                          </div>
                          {tokenAddress && (
                            <div className="text-xs text-muted-foreground mt-0.5">
                              {formatAddress(tokenAddress, 4, 4)}
                            </div>
                          )}
                        </div>
                        <div className="text-right">
                          <div className="font-semibold">
                            {amountInDecimal.toFixed(displayDecimals)} {symbol || ''}
                          </div>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
