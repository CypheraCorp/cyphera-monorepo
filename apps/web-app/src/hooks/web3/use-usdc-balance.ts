'use client';

import { useState, useEffect, useMemo } from 'react';
import { Address, formatUnits } from 'viem';
import { useReadContract, useAccount } from 'wagmi';
import { useToast } from '@/components/ui/use-toast';
import { useNetworkStore } from '@/store/network';
import { logger } from '@/lib/core/logger/logger-utils';
// Rate limiting: one request per hour
const FAUCET_RATE_LIMIT_MS = 60 * 60 * 1000; // 1 hour

interface UseUSDCBalanceReturn {
  balance: bigint | undefined;
  formattedBalance: string;
  isLoading: boolean;
  isRequestingFaucet: boolean;
  requestFaucet: (address: Address) => Promise<void>;
  refetch: () => void;
  lastFaucetRequest: Date | null;
  canRequestFaucet: boolean;
  timeUntilNextRequest: number | null;
  error: Error | null;
  usdcAddress: Address | undefined;
  isFaucetAvailableOnCurrentNetwork: boolean;
}

// USDC contract ABI for balance checking
const USDC_ABI = [
  {
    constant: true,
    inputs: [{ name: 'owner', type: 'address' }],
    name: 'balanceOf',
    outputs: [{ name: '', type: 'uint256' }],
    type: 'function',
  },
] as const;

export function useUSDCBalance(address?: Address): UseUSDCBalanceReturn {
  const { toast } = useToast();
  const { isConnected } = useAccount();
  const getUsdcContractAddress = useNetworkStore((state) => state.getUsdcContractAddress);
  const currentNetwork = useNetworkStore((state) => state.currentNetwork);

  const [isRequestingFaucet, setIsRequestingFaucet] = useState(false);
  const [lastFaucetRequest, setLastFaucetRequest] = useState<Date | null>(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem('lastFaucetRequest');
      return stored ? new Date(stored) : null;
    }
    return null;
  });
  const [timeUntilNextRequest, setTimeUntilNextRequest] = useState<number | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const usdcAddress = getUsdcContractAddress();

  const isFaucetAvailableOnCurrentNetwork = useMemo(() => {
    return !!(
      currentNetwork &&
      currentNetwork.network.is_testnet &&
      usdcAddress &&
      currentNetwork.network.circle_network_type
    );
  }, [currentNetwork, usdcAddress]);

  const {
    data: balanceData,
    isLoading,
    refetch,
    error: readError,
  } = useReadContract({
    address: usdcAddress,
    abi: USDC_ABI,
    functionName: 'balanceOf',
    args: address ? [address] : undefined,
    query: {
      enabled: !!address && isConnected && !!usdcAddress,
      refetchInterval: 30000,
    },
  });

  useEffect(() => {
    if (readError) {
      setError(readError);
      logger.error('USDC Balance Read Error:', readError);
    }
  }, [readError]);

  const balance = balanceData as bigint | undefined;
  const formattedBalance = balance ? formatUnits(balance, 6) : '0.00';

  const canRequestFaucet =
    !lastFaucetRequest || Date.now() - lastFaucetRequest.getTime() >= FAUCET_RATE_LIMIT_MS;

  useEffect(() => {
    if (!lastFaucetRequest || canRequestFaucet) {
      setTimeUntilNextRequest(null);
      return;
    }
    const interval = setInterval(() => {
      const timeLeft = FAUCET_RATE_LIMIT_MS - (Date.now() - lastFaucetRequest.getTime());
      setTimeUntilNextRequest(timeLeft > 0 ? timeLeft : null);
      if (timeLeft <= 0) {
        clearInterval(interval);
      }
    }, 1000);
    return () => clearInterval(interval);
  }, [lastFaucetRequest, canRequestFaucet]);

  const requestFaucet = async (address: Address) => {
    if (!isFaucetAvailableOnCurrentNetwork) {
      toast({
        title: 'Faucet Unavailable',
        description: 'The faucet is not available for the currently connected network.',
        variant: 'destructive',
      });
      return;
    }

    if (!address) {
      toast({ title: 'Error', description: 'No address provided', variant: 'destructive' });
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
        description: `You can request USDC again in ${timeLeft} minutes`,
        variant: 'destructive',
      });
      return;
    }

    try {
      setIsRequestingFaucet(true);

      const blockchainIdentifier = currentNetwork!.network.circle_network_type;

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
    balance,
    formattedBalance,
    isLoading,
    isRequestingFaucet,
    requestFaucet,
    refetch,
    lastFaucetRequest,
    canRequestFaucet,
    timeUntilNextRequest,
    error,
    usdcAddress,
    isFaucetAvailableOnCurrentNetwork,
  };
}
