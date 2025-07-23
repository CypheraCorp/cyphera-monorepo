'use client';

import { useState } from 'react';
import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { DataTable } from '@/components/ui/data-table';
import {
  TextField,
  TextAreaField,
  SelectField,
  CheckboxField,
  RadioGroupField,
} from '@/components/ui/form-field';
import {
  LoadingSpinner,
  TableSkeleton,
  CardSkeleton,
  ProductCardSkeleton,
  ListSkeleton,
  FormSkeleton,
} from '@/components/ui/loading-states';
import { PageErrorBoundary } from '@/components/ui/error-boundary';
import { useLoadingState } from '@/lib/utils/loading-manager';
import { useOptimisticUpdate, optimisticAddToList } from '@/lib/utils/optimistic-updates';
import { ColumnDef } from '@tanstack/react-table';
import { toast } from '@/components/ui/use-toast';

// Example form schema
const exampleFormSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters'),
  email: z.string().email('Invalid email address'),
  bio: z.string().optional(),
  role: z.enum(['admin', 'user', 'guest']),
  newsletter: z.boolean().default(false),
  plan: z.enum(['free', 'pro', 'enterprise']),
});

type ExampleFormData = z.infer<typeof exampleFormSchema>;

// Example data type
interface ExampleData {
  id: string;
  name: string;
  email: string;
  role: string;
  status: 'active' | 'inactive';
}

// Example table columns
const columns: ColumnDef<ExampleData>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
  },
  {
    accessorKey: 'email',
    header: 'Email',
  },
  {
    accessorKey: 'role',
    header: 'Role',
    cell: ({ row }) => <span className="capitalize">{row.getValue('role')}</span>,
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => (
      <span className={row.getValue('status') === 'active' ? 'text-green-600' : 'text-gray-500'}>
        {row.getValue('status')}
      </span>
    ),
  },
];

// Mock data
const mockData: ExampleData[] = [
  { id: '1', name: 'John Doe', email: 'john@example.com', role: 'admin', status: 'active' },
  { id: '2', name: 'Jane Smith', email: 'jane@example.com', role: 'user', status: 'active' },
  { id: '3', name: 'Bob Johnson', email: 'bob@example.com', role: 'guest', status: 'inactive' },
];

export function ComponentShowcase() {
  const [showError, setShowError] = useState(false);
  const [data, setData] = useState(mockData);
  const { isLoading, execute } = useLoadingState();
  const { update } = useOptimisticUpdate<ExampleData[]>();

  // Form setup
  const methods = useForm<ExampleFormData>({
    resolver: zodResolver(exampleFormSchema),
    defaultValues: {
      newsletter: false,
      role: 'user',
      plan: 'free',
    },
  });

  const onSubmit = async (formData: ExampleFormData) => {
    await execute(
      new Promise((resolve) => {
        setTimeout(() => {
          toast({
            title: 'Form submitted!',
            description: 'Check the console for form data.',
          });
          console.log('Form data:', formData);
          resolve(true);
        }, 1500);
      })
    );
  };

  // Simulate adding data with optimistic update
  const handleAddData = async () => {
    const newItem: ExampleData = {
      id: Date.now().toString(),
      name: 'New User',
      email: 'new@example.com',
      role: 'user',
      status: 'active',
    };

    await update(
      {
        queryKey: ['example-data'],
        updateFn: (oldData) => optimisticAddToList(oldData || data, newItem),
      },
      newItem,
      async () => {
        // Simulate API call
        await new Promise((resolve) => setTimeout(resolve, 1000));
        setData((prev) => [newItem, ...prev]);
        return [...data, newItem];
      }
    );
  };

  if (showError) {
    throw new Error('This is a test error!');
  }

  return (
    <div className="space-y-8 p-8">
      <h1 className="text-3xl font-bold">Component Library Showcase</h1>

      {/* Form Components */}
      <Card>
        <CardHeader>
          <CardTitle>Form Components</CardTitle>
          <CardDescription>Standardized form components with validation</CardDescription>
        </CardHeader>
        <CardContent>
          <FormProvider {...methods}>
            <form onSubmit={methods.handleSubmit(onSubmit)} className="space-y-4">
              <TextField name="name" label="Name" placeholder="Enter your name" required />

              <TextField
                name="email"
                label="Email"
                type="email"
                placeholder="email@example.com"
                required
              />

              <TextAreaField name="bio" label="Bio" placeholder="Tell us about yourself" rows={3} />

              <SelectField
                name="role"
                label="Role"
                required
                options={[
                  { label: 'Admin', value: 'admin' },
                  { label: 'User', value: 'user' },
                  { label: 'Guest', value: 'guest' },
                ]}
              />

              <RadioGroupField
                name="plan"
                label="Subscription Plan"
                required
                options={[
                  { label: 'Free', value: 'free', description: 'Basic features' },
                  { label: 'Pro', value: 'pro', description: '$10/month' },
                  { label: 'Enterprise', value: 'enterprise', description: 'Custom pricing' },
                ]}
              />

              <CheckboxField
                name="newsletter"
                label="Subscribe to newsletter"
                description="Get updates about new features"
              />

              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Submitting...' : 'Submit Form'}
              </Button>
            </form>
          </FormProvider>
        </CardContent>
      </Card>

      {/* Data Table */}
      <Card>
        <CardHeader>
          <CardTitle>Data Table Component</CardTitle>
          <CardDescription>Reusable table with search, sorting, and pagination</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <Button onClick={handleAddData}>Add New Row (Optimistic)</Button>
            <DataTable
              columns={columns}
              data={data}
              searchPlaceholder="Search users..."
              searchKey="name"
            />
          </div>
        </CardContent>
      </Card>

      {/* Loading States */}
      <Card>
        <CardHeader>
          <CardTitle>Loading States</CardTitle>
          <CardDescription>Various loading component states</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <h3 className="font-medium mb-2">Loading Spinner</h3>
              <LoadingSpinner message="Loading data..." />
            </div>

            <div>
              <h3 className="font-medium mb-2">Card Skeleton</h3>
              <CardSkeleton />
            </div>

            <div>
              <h3 className="font-medium mb-2">Product Card Skeleton</h3>
              <ProductCardSkeleton />
            </div>

            <div>
              <h3 className="font-medium mb-2">List Skeleton</h3>
              <ListSkeleton items={3} />
            </div>

            <div className="md:col-span-2">
              <h3 className="font-medium mb-2">Table Skeleton</h3>
              <TableSkeleton rows={3} columns={4} />
            </div>

            <div>
              <h3 className="font-medium mb-2">Form Skeleton</h3>
              <FormSkeleton fields={3} />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Error Boundary */}
      <Card>
        <CardHeader>
          <CardTitle>Error Boundary</CardTitle>
          <CardDescription>Error handling with fallback UI</CardDescription>
        </CardHeader>
        <CardContent>
          <Button onClick={() => setShowError(true)} variant="destructive">
            Trigger Error
          </Button>
          <p className="text-sm text-muted-foreground mt-2">
            Click to see the error boundary in action
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

// Wrap the showcase in error boundary
export default function ComponentShowcasePage() {
  return (
    <PageErrorBoundary>
      <ComponentShowcase />
    </PageErrorBoundary>
  );
}
