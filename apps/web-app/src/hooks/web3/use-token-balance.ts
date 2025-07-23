'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import { Address, formatUnits, erc20Abi } from 'viem';
import { useReadContract, useAccount } from 'wagmi';
import { logger } from '@/lib/core/logger/logger-utils';
// Removed hardcoded TOKEN_CONTRACT_ADDRESS

interface UseTokenBalanceArgs {
  userAddress?: Address; // Address of the wallet owner
  tokenAddress?: Address; // Address of the ERC20 token contract
  enabled?: boolean; // Optional flag to enable/disable the hook
}

interface UseTokenBalanceReturn {
  balance: bigint | undefined;
  formattedBalance: string;
  decimals: number | undefined;
  tokenSymbol: string | undefined;
  tokenName: string | undefined;
  isLoading: boolean;
  refetch: () => void; // Combined refetch for all reads
  error: Error | null;
}

/**
 * Hook to get the balance and metadata of a specific ERC20 token for a given address
 * on the currently connected network.
 * @param userAddress The address to check the balance for
 * @param tokenAddress The ERC20 contract address
 * @returns The token balance and metadata
 */
export function useTokenBalance({
  userAddress,
  tokenAddress,
  enabled: hookEnabled = true, // Default to true if not provided
}: UseTokenBalanceArgs): UseTokenBalanceReturn {
  const { isConnected } = useAccount();
  const [error, setError] = useState<Error | null>(null);

  // Determine if hook should be enabled based on inputs AND the prop
  const balanceReadEnabled = hookEnabled && !!userAddress && !!tokenAddress && isConnected;
  const metadataReadEnabled = hookEnabled && !!tokenAddress && isConnected;

  // Read token balance
  const {
    data: balanceData,
    isLoading: isLoadingBalance,
    refetch: refetchBalance,
    error: balanceError,
  } = useReadContract({
    address: tokenAddress, // Use dynamic address
    abi: erc20Abi,
    functionName: 'balanceOf',
    args: userAddress ? [userAddress] : undefined,
    query: {
      enabled: balanceReadEnabled,
      refetchInterval: 30000,
    },
  });

  // Read token decimals
  const {
    data: decimalsData,
    isLoading: isLoadingDecimals,
    refetch: refetchDecimals,
    error: decimalsError,
  } = useReadContract({
    address: tokenAddress, // Use dynamic address
    abi: erc20Abi,
    functionName: 'decimals',
    query: {
      enabled: metadataReadEnabled,
    },
  });

  // Read token symbol
  const {
    data: symbolData,
    isLoading: isLoadingSymbol,
    refetch: refetchSymbol,
    error: symbolError,
  } = useReadContract({
    address: tokenAddress, // Use dynamic address
    abi: erc20Abi,
    functionName: 'symbol',
    query: {
      enabled: metadataReadEnabled,
    },
  });

  // Read token name
  const {
    data: nameData,
    isLoading: isLoadingName,
    refetch: refetchName,
    error: nameError,
  } = useReadContract({
    address: tokenAddress, // Use dynamic address
    abi: erc20Abi,
    functionName: 'name',
    query: {
      enabled: metadataReadEnabled,
    },
  });

  // Combine loading states
  const isLoading = isLoadingBalance || isLoadingDecimals || isLoadingSymbol || isLoadingName;

  // Combine errors
  useEffect(() => {
    const firstError = balanceError || decimalsError || symbolError || nameError;
    setError(firstError || null);
    if (firstError) {
      logger.error('Token Balance Hook Error:', firstError);
    }
  }, [balanceError, decimalsError, symbolError, nameError]);

  // Memoize derived values
  const decimals = useMemo(
    () => (decimalsData !== undefined ? Number(decimalsData) : undefined),
    [decimalsData]
  );
  const balance = useMemo(() => balanceData as bigint | undefined, [balanceData]);
  const formattedBalance = useMemo(() => {
    if (balance === undefined || decimals === undefined) return '0.00';
    return formatUnits(balance, decimals);
  }, [balance, decimals]);
  const tokenSymbol = useMemo(() => symbolData as string | undefined, [symbolData]);
  const tokenName = useMemo(() => nameData as string | undefined, [nameData]);

  // Combined refetch function
  const refetch = useCallback(() => {
    refetchBalance();
    refetchDecimals();
    refetchSymbol();
    refetchName();
  }, [refetchBalance, refetchDecimals, refetchSymbol, refetchName]);

  return {
    balance,
    formattedBalance,
    decimals,
    tokenSymbol,
    tokenName,
    isLoading,
    refetch,
    error,
  };
}
