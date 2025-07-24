'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Plus, Loader2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { FormInput } from '@/components/ui/form-input';
import { Label } from '@/components/ui/label';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { logger } from '@/lib/core/logger/logger-utils';
import { useValidationErrors } from '@/hooks/use-validation-errors';
import { ValidationErrorDisplay, InlineValidationError } from '@/components/ui/validation-error';
import { isValidationError } from '@/types/validation';
const createCustomerSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  email: z.string().email('Invalid email address'),
  phone: z.string().optional(),
  description: z.string().optional(),
});

type CreateCustomerForm = z.infer<typeof createCustomerSchema>;

export function CreateCustomerButton() {
  const [isOpen, setIsOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const router = useRouter();
  
  const {
    validationErrors,
    clearValidationErrors,
    handleValidationError,
    getFieldError,
  } = useValidationErrors();

  const form = useForm<CreateCustomerForm>({
    resolver: zodResolver(createCustomerSchema),
    defaultValues: {
      name: '',
      email: '',
      phone: '',
      description: '',
    },
  });

  const onSubmit = async (data: CreateCustomerForm) => {
    try {
      setIsSubmitting(true);
      clearValidationErrors(); // Clear any previous validation errors
      
      const response = await fetch('/api/customers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        const errorData = await response.json();
        
        // Check if it's a validation error
        if (response.status === 400 && isValidationError(errorData)) {
          handleValidationError(errorData);
          return; // Don't close dialog, let user fix errors
        }
        
        // Handle other errors
        throw new Error(errorData.error || 'Failed to create customer');
      }

      toast.success('Customer created successfully');
      setIsOpen(false);
      form.reset();
      clearValidationErrors();
      router.refresh(); // Refresh the page to show the new customer
    } catch (error) {
      logger.error('Error creating customer:', error);
      toast.error(error instanceof Error ? error.message : 'Failed to create customer');
    } finally {
      setIsSubmitting(false);
    }
  };

  // Clear errors when dialog closes
  const handleOpenChange = (open: boolean) => {
    setIsOpen(open);
    if (!open) {
      clearValidationErrors();
      form.reset();
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button className="flex items-center gap-2">
          <Plus className="h-4 w-4" />
          Add Customer
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Create Customer</DialogTitle>
        </DialogHeader>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          {/* Show general validation errors at the top */}
          <ValidationErrorDisplay errors={validationErrors} />
          
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <FormInput
              id="name"
              {...form.register('name')}
              error={form.formState.errors.name?.message || getFieldError('name')}
            />
            <InlineValidationError errors={validationErrors} field="name" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <FormInput
              id="email"
              type="email"
              {...form.register('email')}
              error={form.formState.errors.email?.message || getFieldError('email')}
            />
            <InlineValidationError errors={validationErrors} field="email" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="phone">Phone</Label>
            <FormInput
              id="phone"
              {...form.register('phone')}
              error={form.formState.errors.phone?.message || getFieldError('phone')}
            />
            <InlineValidationError errors={validationErrors} field="phone" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <FormInput
              id="description"
              {...form.register('description')}
              error={form.formState.errors.description?.message || getFieldError('description')}
            />
            <InlineValidationError errors={validationErrors} field="description" />
          </div>
          <div className="flex justify-end gap-3">
            <Button type="button" variant="outline" onClick={() => setIsOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                'Create Customer'
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
