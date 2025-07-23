'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Loader2 } from 'lucide-react';
import { FormInput } from '@/components/ui/form-input';
import { Label } from '@/components/ui/label';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import type { CustomerResponse } from '@/types/customer';
import { logger } from '@/lib/core/logger/logger-utils';
const updateCustomerSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  email: z.string().email('Invalid email address'),
  phone: z.string().optional(),
  description: z.string().optional(),
});

type UpdateCustomerForm = z.infer<typeof updateCustomerSchema>;

interface EditCustomerButtonProps {
  customer: CustomerResponse;
}

export function EditCustomerButton({ customer }: EditCustomerButtonProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const router = useRouter();

  const form = useForm<UpdateCustomerForm>({
    resolver: zodResolver(updateCustomerSchema),
    defaultValues: {
      name: customer.name,
      email: customer.email,
      phone: customer.phone || '',
      description: customer.description || '',
    },
  });

  const onSubmit = async (data: UpdateCustomerForm) => {
    try {
      setIsSubmitting(true);
      const response = await fetch(`/api/customers/${customer.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        throw new Error('Failed to update customer');
      }

      setIsOpen(false);
      router.refresh(); // Refresh the page to show updated data
    } catch (error) {
      logger.error('Error updating customer:', error);
      // You would typically show an error message here
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <DropdownMenuItem onSelect={(e) => e.preventDefault()}>Edit Customer</DropdownMenuItem>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Edit Customer</DialogTitle>
        </DialogHeader>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <FormInput
              id="name"
              {...form.register('name')}
              error={form.formState.errors.name?.message}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <FormInput
              id="email"
              type="email"
              {...form.register('email')}
              error={form.formState.errors.email?.message}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="phone">Phone</Label>
            <FormInput
              id="phone"
              {...form.register('phone')}
              error={form.formState.errors.phone?.message}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <FormInput
              id="description"
              {...form.register('description')}
              error={form.formState.errors.description?.message}
            />
          </div>
          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => setIsOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Saving...
                </>
              ) : (
                'Save Changes'
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
