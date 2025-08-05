'use client';

import React, { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2, AlertCircle, Info } from 'lucide-react';
import { toast } from 'sonner';
import { WalletResponse } from '@/types/wallet';
import { NetworkWithTokensResponse } from '@/types/network';
import { CircleAPI } from '@/services/cyphera-api/circle';
import { useCircleSDK } from '@/hooks/web3';
import { CircleTransactionFeeLevel } from '@/types/circle';
import { formatUnits } from 'viem';
import { getBlockchainFromCircleWalletId } from '@/lib/utils/circle';
import { logger } from '@/lib/core/logger/logger-utils';

// Validation schema for send transaction form
const sendTransactionSchema = z.object({
  destination_address: z.string()
    .min(1, 'Destination address is required')
    .regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address format'),
  amount: z.string()
    .min(1, 'Amount is required')
    .refine((val) => {
      const num = parseFloat(val);
      return !isNaN(num) && num > 0;
    }, 'Amount must be a positive number'),
  token_id: z.string().optional(),
  fee_level: z.enum(['LOW', 'MEDIUM', 'HIGH'] as const).default('MEDIUM'),
});

type SendTransactionFormData = z.infer<typeof sendTransactionSchema>;

interface SendTransactionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  wallet: WalletResponse;
  workspaceId: string;
  networks: NetworkWithTokensResponse[];
}

interface FeeEstimate {
  low: { amount: string; amountInUSD: string };
  medium: { amount: string; amountInUSD: string };
  high: { amount: string; amountInUSD: string };
}

export function SendTransactionDialog({
  open,
  onOpenChange,
  wallet,
  workspaceId,
  networks,
}: SendTransactionDialogProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [isValidatingAddress, setIsValidatingAddress] = useState(false);
  const [isEstimatingFee, setIsEstimatingFee] = useState(false);
  const [addressValid, setAddressValid] = useState<boolean | null>(null);
  const [feeEstimate, setFeeEstimate] = useState<FeeEstimate | null>(null);
  const [selectedToken, setSelectedToken] = useState<string>('native');
  const { userToken, executeChallenge } = useCircleSDK();

  const form = useForm<SendTransactionFormData>({
    resolver: zodResolver(sendTransactionSchema),
    defaultValues: {
      destination_address: '',
      amount: '',
      fee_level: 'MEDIUM',
    },
  });

  // Get current network and available tokens
  const currentNetwork = networks.find((n) => {
    if (!wallet.circle_data?.circle_wallet_id) return false;
    const blockchainId = getBlockchainFromCircleWalletId(wallet.circle_data.circle_wallet_id);
    return n.network.circle_network_type === blockchainId;
  });

  const availableTokens = currentNetwork?.tokens || [];
  const nativeToken = availableTokens.find((t) => t.gas_token);

  // Validate address when it changes
  useEffect(() => {
    const subscription = form.watch(async (value, { name }) => {
      if (name === 'destination_address' && value.destination_address) {
        const address = value.destination_address;
        if (/^0x[a-fA-F0-9]{40}$/.test(address)) {
          setIsValidatingAddress(true);
          try {
            const circleApi = new CircleAPI();
            const result = await circleApi.validateAddress(
              address,
              currentNetwork?.network.circle_network_type || ''
            );
            setAddressValid(result.isValid);
          } catch (error) {
            logger.error('Failed to validate address:', error);
            setAddressValid(null);
          } finally {
            setIsValidatingAddress(false);
          }
        } else {
          setAddressValid(null);
        }
      }
    });
    return () => subscription.unsubscribe();
  }, [form, currentNetwork]);

  // Estimate fees when amount or token changes
  useEffect(() => {
    const subscription = form.watch(async (value, { name }) => {
      if ((name === 'amount' || name === 'token_id') && 
          value.amount && 
          parseFloat(value.amount) > 0 &&
          value.destination_address &&
          addressValid) {
        setIsEstimatingFee(true);
        try {
          const circleApi = new CircleAPI();
          const tokenInfo = selectedToken === 'native' 
            ? nativeToken 
            : availableTokens.find(t => t.id === selectedToken);

          if (!tokenInfo || !wallet.circle_data?.circle_wallet_id) return;

          const decimals = tokenInfo.decimals || 18;
          const amountInWei = (parseFloat(value.amount) * Math.pow(10, decimals)).toString();

          const estimate = await circleApi.estimateFee({
            wallet_id: wallet.circle_data.circle_wallet_id,
            destination_address: value.destination_address,
            amount: amountInWei,
            token_id: selectedToken === 'native' ? undefined : selectedToken,
            blockchain: currentNetwork?.network.circle_network_type || '',
          });

          setFeeEstimate(estimate);
        } catch (error) {
          logger.error('Failed to estimate fee:', error);
          toast.error('Failed to estimate transaction fee');
        } finally {
          setIsEstimatingFee(false);
        }
      }
    });
    return () => subscription.unsubscribe();
  }, [form, selectedToken, addressValid, availableTokens, nativeToken, wallet, currentNetwork]);

  const onSubmit = async (data: SendTransactionFormData) => {
    if (!userToken || !wallet.circle_data?.circle_wallet_id) {
      toast.error('Missing wallet data');
      return;
    }

    setIsLoading(true);
    try {
      const circleApi = new CircleAPI();
      const tokenInfo = selectedToken === 'native' 
        ? nativeToken 
        : availableTokens.find(t => t.id === selectedToken);

      if (!tokenInfo) {
        throw new Error('Invalid token selected');
      }

      const decimals = tokenInfo.decimals || 18;
      const amountInWei = (parseFloat(data.amount) * Math.pow(10, decimals)).toString();

      // Create transfer
      const transferResponse = await circleApi.createTransfer({
        idempotency_key: crypto.randomUUID(),
        amounts: [amountInWei],
        destination_address: data.destination_address,
        token_id: selectedToken === 'native' ? undefined : selectedToken,
        wallet_id: wallet.circle_data.circle_wallet_id,
        fee_level: data.fee_level as CircleTransactionFeeLevel,
      });

      // Execute the challenge
      if (transferResponse.challenge_id) {
        await executeChallenge(transferResponse.challenge_id);
        toast.success('Transaction sent successfully!');
        onOpenChange(false);
        form.reset();
      }
    } catch (error) {
      logger.error('Failed to send transaction:', error);
      toast.error('Failed to send transaction');
    } finally {
      setIsLoading(false);
    }
  };

  const getFeeDisplay = (feeLevel: 'low' | 'medium' | 'high') => {
    if (!feeEstimate || !nativeToken) return '--';
    const fee = feeEstimate[feeLevel];
    const feeInEth = formatUnits(BigInt(fee.amount), nativeToken.decimals || 18);
    return `${parseFloat(feeInEth).toFixed(6)} ${nativeToken.symbol} ($${fee.amountInUSD})`;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Send Transaction</DialogTitle>
          <DialogDescription>
            Send tokens from your Circle wallet to another address.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="destination_address"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Recipient Address</FormLabel>
                  <FormControl>
                    <div className="relative">
                      <Input
                        {...field}
                        placeholder="0x..."
                        disabled={isLoading}
                      />
                      {isValidatingAddress && (
                        <Loader2 className="absolute right-3 top-3 h-4 w-4 animate-spin" />
                      )}
                      {!isValidatingAddress && addressValid === true && (
                        <div className="absolute right-3 top-3 h-4 w-4 rounded-full bg-green-500" />
                      )}
                      {!isValidatingAddress && addressValid === false && (
                        <div className="absolute right-3 top-3 h-4 w-4 rounded-full bg-red-500" />
                      )}
                    </div>
                  </FormControl>
                  <FormDescription>
                    The Ethereum address to send tokens to
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormItem>
              <FormLabel>Token</FormLabel>
              <Select
                value={selectedToken}
                onValueChange={setSelectedToken}
                disabled={isLoading}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select token" />
                </SelectTrigger>
                <SelectContent>
                  {nativeToken && (
                    <SelectItem value="native">
                      {nativeToken.symbol} (Native)
                    </SelectItem>
                  )}
                  {availableTokens
                    .filter(t => !t.gas_token)
                    .map((token) => (
                      <SelectItem key={token.id} value={token.id}>
                        {token.symbol}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              <FormDescription>
                Select the token to send
              </FormDescription>
            </FormItem>

            <FormField
              control={form.control}
              name="amount"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Amount</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      type="number"
                      step="any"
                      placeholder="0.0"
                      disabled={isLoading}
                    />
                  </FormControl>
                  <FormDescription>
                    Amount of tokens to send
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="fee_level"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Fee Level</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    disabled={isLoading}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="LOW">Low - {getFeeDisplay('low')}</SelectItem>
                      <SelectItem value="MEDIUM">Medium - {getFeeDisplay('medium')}</SelectItem>
                      <SelectItem value="HIGH">High - {getFeeDisplay('high')}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    Transaction speed preference
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {isEstimatingFee && (
              <Alert>
                <Loader2 className="h-4 w-4 animate-spin" />
                <AlertDescription>Estimating transaction fees...</AlertDescription>
              </Alert>
            )}

            {feeEstimate && !isEstimatingFee && (
              <Alert>
                <Info className="h-4 w-4" />
                <AlertDescription>
                  Estimated network fee based on current gas prices
                </AlertDescription>
              </Alert>
            )}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={isLoading}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={isLoading || !addressValid || isEstimatingFee}
              >
                {isLoading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Sending...
                  </>
                ) : (
                  'Send Transaction'
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}