'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { toast } from 'sonner';
import { logger } from '@/lib/core/logger/logger-utils';
interface DeleteCustomerButtonProps {
  customerId: string;
}

export function DeleteCustomerButton({ customerId }: DeleteCustomerButtonProps) {
  const [isDeleting, setIsDeleting] = useState(false);
  const router = useRouter();

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this customer?')) {
      return;
    }

    try {
      setIsDeleting(true);
      const response = await fetch(`/api/customers/${customerId}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error('Failed to delete customer');
      }

      toast.success('Customer deleted successfully');
      router.refresh(); // This will trigger a re-fetch of the customers data
    } catch (error) {
      toast.error('Failed to delete customer');
      logger.error('Failed to delete customer:', error);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <DropdownMenuItem
      className="text-red-600"
      disabled={isDeleting}
      onClick={handleDelete}
      onSelect={(e) => e.preventDefault()}
    >
      {isDeleting ? 'Deleting...' : 'Delete Customer'}
    </DropdownMenuItem>
  );
}
