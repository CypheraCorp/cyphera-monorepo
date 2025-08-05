'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
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
import { Button } from '@/components/ui/button';
import { Loader2, Wallet } from 'lucide-react';
import { toast } from 'sonner';
// Form schema for adding a wallet
const addWalletFormSchema = z.object({
  wallet_address: z.string()
    .min(1, 'Wallet address is required')
    .regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address format. Must start with 0x followed by 40 hexadecimal characters'),
  nickname: z.string()
    .max(50, 'Nickname must be less than 50 characters')
    .optional()
    .or(z.literal('')),
});

type AddWalletFormValues = z.infer<typeof addWalletFormSchema>;

interface AddWalletDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onWalletAdded?: () => void;
}

export function AddWalletDialog({
  isOpen,
  onOpenChange,
  onWalletAdded,
}: AddWalletDialogProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<AddWalletFormValues>({
    resolver: zodResolver(addWalletFormSchema),
    defaultValues: {
      wallet_address: '',
      nickname: '',
    },
  });

  const onSubmit = async (values: AddWalletFormValues) => {
    try {
      setIsSubmitting(true);

      // First, fetch CSRF token
      const csrfResponse = await fetch('/api/auth/csrf');
      if (!csrfResponse.ok) {
        throw new Error('Failed to fetch CSRF token');
      }
      const { csrfToken } = await csrfResponse.json();

      const response = await fetch('/api/wallets', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': csrfToken,
        },
        body: JSON.stringify({
          wallet_type: 'wallet', // Backend expects: wallet, circle, web3auth
          wallet_address: values.wallet_address,
          network_type: 'evm', // Default to EVM for all wallets
          nickname: values.nickname || undefined,
        }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to add wallet');
      }

      toast.success('Wallet added successfully');
      form.reset();
      onOpenChange(false);
      onWalletAdded?.();
    } catch (error) {
      console.error('Error adding wallet:', error);
      toast.error(error instanceof Error ? error.message : 'Failed to add wallet');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Wallet className="h-5 w-5" />
            Add Existing Wallet
          </DialogTitle>
          <DialogDescription>
            Add an existing wallet address to your account. You can optionally provide a nickname for easy identification.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="wallet_address"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Wallet Address</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="0x..."
                      {...field}
                      disabled={isSubmitting}
                    />
                  </FormControl>
                  <FormDescription>
                    Enter your Ethereum wallet address
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="nickname"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Nickname (Optional)</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="e.g., Main Wallet, Trading Wallet"
                      {...field}
                      disabled={isSubmitting}
                    />
                  </FormControl>
                  <FormDescription>
                    Give your wallet a friendly name for easy identification
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={isSubmitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Adding...
                  </>
                ) : (
                  'Add Wallet'
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}