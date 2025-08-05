'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Trash2 } from 'lucide-react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { useToast } from '@/components/ui/use-toast';
import { logger } from '@/lib/core/logger/logger-utils';

interface DeleteWalletButtonProps {
  walletId: string;
  onDeleted?: () => void;
}

export function DeleteWalletButton({ walletId, onDeleted }: DeleteWalletButtonProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const { toast } = useToast();

  const handleDelete = async () => {
    try {
      setIsDeleting(true);
      
      // First, fetch CSRF token
      const csrfResponse = await fetch('/api/auth/csrf');
      if (!csrfResponse.ok) {
        throw new Error('Failed to fetch CSRF token');
      }
      const { csrfToken } = await csrfResponse.json();

      const response = await fetch(`/api/wallets/${walletId}`, {
        method: 'DELETE',
        headers: {
          'X-CSRF-Token': csrfToken,
        },
      });

      if (!response.ok) {
        toast({
          title: 'Error',
          description:
            "There was an issue deleting the wallet. Please ensure it's not being used by any products.",
          variant: 'destructive',
        });
        return;
      }

      setIsOpen(false);
      onDeleted?.();
    } catch (error) {
      logger.error('Error deleting wallet:', error);
      toast({
        title: 'Error',
        description: 'There was an issue deleting the wallet. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => setIsOpen(true)}
        className="h-8 w-8 text-red-600 dark:text-red-400"
      >
        <Trash2 className="h-4 w-4" />
      </Button>

      <AlertDialog open={isOpen} onOpenChange={setIsOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete this wallet.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={isDeleting}
              className="bg-red-600 hover:bg-red-700"
            >
              {isDeleting ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
