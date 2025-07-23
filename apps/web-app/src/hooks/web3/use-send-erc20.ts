'use client';

import { useState, useEffect, useCallback } from 'react';
import { useAccount, useWriteContract, useWaitForTransactionReceipt } from 'wagmi';
import {
  type Address,
  type Hash,
  type WriteContractErrorType,
  type TransactionReceipt,
  erc20Abi,
} from 'viem';
import { useNetworkStore } from '@/store/network';
import { useToast } from '@/components/ui/use-toast';
import { logger } from '@/lib/core/logger/logger-utils';
interface UseSendErc20Args {
  onSuccess?: (data: { hash: Hash; receipt: TransactionReceipt }) => void;
  onError?: (error: WriteContractErrorType | Error) => void;
}

interface SendErc20FunctionArgs {
  tokenAddress: Address;
  to: Address;
  amount: bigint;
}

interface UseSendErc20Return {
  sendErc20: (args: SendErc20FunctionArgs) => Promise<void>;
  isPendingSignature: boolean;
  isConfirming: boolean;
  isSuccess: boolean;
  isError: boolean;
  error: WriteContractErrorType | Error | null;
  hash: Hash | undefined;
  receipt: TransactionReceipt | null;
}

/**
 * Hook to send any ERC20 token.
 * Handles contract interaction, transaction state, and basic error feedback.
 */
export function useSendErc20({ onSuccess, onError }: UseSendErc20Args = {}): UseSendErc20Return {
  const { isConnected } = useAccount();
  const currentNetwork = useNetworkStore((state) => state.currentNetwork);
  const { toast } = useToast();

  const {
    data: hash,
    writeContract,
    isPending: isPendingSignature,
    error: writeError,
    reset: resetWriteContract,
  } = useWriteContract();

  const {
    data: receipt,
    isLoading: isConfirming,
    isSuccess,
    error: confirmError,
  } = useWaitForTransactionReceipt({ hash });

  const [internalError, setInternalError] = useState<Error | null>(null);

  const error = writeError || confirmError || internalError;
  const isError = !!error;

  useEffect(() => {
    if (!isPendingSignature) {
      setInternalError(null);
    }
  }, [isPendingSignature]);

  const sendErc20 = useCallback(
    async ({ tokenAddress, to, amount }: SendErc20FunctionArgs) => {
      resetWriteContract();
      setInternalError(null);

      if (!isConnected) {
        const err = new Error('Wallet not connected.');
        setInternalError(err);
        onError?.(err);
        toast({ title: 'Error', description: err.message, variant: 'destructive' });
        return;
      }
      if (!currentNetwork) {
        const err = new Error('Not connected to a supported network.');
        setInternalError(err);
        onError?.(err);
        toast({ title: 'Error', description: err.message, variant: 'destructive' });
        return;
      }
      if (!tokenAddress) {
        const err = new Error('Token address not provided.');
        setInternalError(err);
        onError?.(err);
        toast({ title: 'Error', description: err.message, variant: 'destructive' });
        return;
      }
      if (!to || amount <= BigInt(0)) {
        const err = new Error('Invalid recipient address or amount.');
        setInternalError(err);
        onError?.(err);
        toast({ title: 'Error', description: err.message, variant: 'destructive' });
        return;
      }

      logger.log(
        `Attempting to send ${amount} tokens [${tokenAddress}] to ${to} on network ${currentNetwork.network.name}`
      );

      writeContract({
        address: tokenAddress,
        abi: erc20Abi,
        functionName: 'transfer',
        args: [to, amount],
        chainId: currentNetwork.network.chain_id,
      });
    },
    [isConnected, currentNetwork, writeContract, resetWriteContract, onError, toast]
  );

  useEffect(() => {
    if (isSuccess && receipt && hash) {
      logger.log('ERC20 Transfer successful:', { hash, receipt });
      toast({ title: 'Success', description: 'Token transfer confirmed.' });
      onSuccess?.({ hash, receipt });
      resetWriteContract();
    }
  }, [isSuccess, receipt, hash, onSuccess, toast, resetWriteContract]);

  useEffect(() => {
    if (error) {
      logger.error('ERC20 Transfer Error:', error);
      if (error !== internalError) {
        toast({
          title: 'Transaction Failed',
          description: error.message,
          variant: 'destructive',
        });
      }
      onError?.(error);
    }
  }, [error, onError, toast, internalError]);

  return {
    sendErc20,
    isPendingSignature,
    isConfirming,
    isSuccess,
    isError,
    error,
    hash,
    receipt: receipt || null,
  };
}
