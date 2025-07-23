'use client';

import { useTransition } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';

import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { useToast } from '@/components/ui/use-toast';
import type { CypheraUser } from '@/lib/auth/session/session';

const formSchema = z.object({
  business_name: z.string().min(2, 'Business name must be at least 2 characters'),
  business_type: z.string().optional(),
  website_url: z.string().url('Invalid URL').optional(),
  support_email: z.string().email('Invalid email address'),
  support_phone: z.string().optional(),
});

type FormData = z.infer<typeof formSchema>;

// Create an interface that ensures user has required properties
interface CompanyRegistrationFormProps {
  user: CypheraUser & {
    user_id: string;
    account_id: string;
    workspace_id: string;
  };
}

/**
 * CompanyRegistrationForm component
 * Form for updating company registration information
 */
export function CompanyRegistrationForm({ user }: CompanyRegistrationFormProps) {
  const { toast } = useToast();
  const [isPending, startTransition] = useTransition();

  const form = useForm<FormData>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      business_name: '',
      business_type: '',
      website_url: '',
      support_email: '',
      support_phone: '',
    },
  });

  // Show not authenticated state
  if (!user) {
    return (
      <Alert>
        <AlertDescription>Please log in to continue.</AlertDescription>
      </Alert>
    );
  }

  const onSubmit = async (data: FormData) => {
    startTransition(async () => {
      try {
        const response = await fetch('/api/accounts', {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            ...data,
            account_id: user.account_id, // Add account_id to the request
          }),
        });

        if (!response.ok) {
          throw new Error('Failed to update account');
        }

        toast({
          title: 'Success',
          description: 'Your company information has been updated.',
        });
      } catch (error) {
        toast({
          variant: 'destructive',
          title: 'Error',
          description: error instanceof Error ? error.message : 'Failed to update account',
        });
      }
    });
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <FormField
          control={form.control}
          name="business_name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Business Name</FormLabel>
              <FormControl>
                <Input placeholder="Enter your business name" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="business_type"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Business Type</FormLabel>
              <FormControl>
                <Input placeholder="Enter your business type" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="website_url"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Website URL</FormLabel>
              <FormControl>
                <Input placeholder="https://example.com" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="support_email"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Support Email</FormLabel>
              <FormControl>
                <Input placeholder="support@example.com" type="email" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="support_phone"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Support Phone</FormLabel>
              <FormControl>
                <Input placeholder="+1 (555) 123-4567" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <Button type="submit" disabled={isPending}>
          {isPending ? 'Saving...' : 'Save Changes'}
        </Button>
      </form>
    </Form>
  );
}
