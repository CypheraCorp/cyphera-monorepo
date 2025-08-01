'use client';

import { Button } from '@/components/ui/button';
import { useSearchParams } from 'next/navigation';
import { useCustomers } from '@/hooks/data';
import { TableSkeleton } from '@/components/ui/loading-states';
import dynamic from 'next/dynamic';
import { Suspense } from 'react';

// Dynamically import lucide-react icons to reduce bundle size
const Users2 = dynamic(() => import('lucide-react').then((mod) => ({ default: mod.Users2 })), {
  loading: () => <div className="h-12 w-12 bg-muted animate-pulse rounded-full" />,
  ssr: false,
});

const Mail = dynamic(() => import('lucide-react').then((mod) => ({ default: mod.Mail })), {
  loading: () => <div className="h-3 w-3 bg-muted animate-pulse rounded-sm" />,
  ssr: false,
});

const MoreHorizontal = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.MoreHorizontal })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />,
    ssr: false,
  }
);

// Dynamically import heavy components to reduce initial bundle size
const Table = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.Table })),
  {
    loading: () => <div className="h-64 bg-muted animate-pulse rounded-md" />,
  }
);

const TableBody = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableBody })),
  {
    loading: () => <tbody className="h-8 bg-muted animate-pulse rounded-md" />,
  }
);

const TableCell = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableCell })),
  {
    loading: () => <td className="h-8 bg-muted animate-pulse rounded-md" />,
  }
);

const TableHead = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableHead })),
  {
    loading: () => <th className="h-8 bg-muted animate-pulse rounded-md" />,
  }
);

const TableHeader = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableHeader })),
  {
    loading: () => <thead className="h-8 bg-muted animate-pulse rounded-md" />,
  }
);

const TableRow = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableRow })),
  {
    loading: () => <tr className="h-8 bg-muted animate-pulse rounded-md" />,
  }
);

const DropdownMenu = dynamic(
  () => import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenu })),
  {
    loading: () => <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />,
  }
);

const DropdownMenuContent = dynamic(
  () =>
    import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenuContent })),
  {
    loading: () => <div className="h-20 w-32 bg-muted animate-pulse rounded-md" />,
  }
);

const DropdownMenuItem = dynamic(
  () => import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenuItem })),
  {
    loading: () => <div className="h-8 bg-muted animate-pulse rounded-md" />,
  }
);

const DropdownMenuTrigger = dynamic(
  () =>
    import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenuTrigger })),
  {
    loading: () => <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />,
  }
);

// Lazy load customer action components (heavy form components)
const CreateCustomerButton = dynamic(
  () =>
    import('@/components/customers/create-customer-button').then((mod) => ({
      default: mod.CreateCustomerButton,
    })),
  {
    loading: () => <div className="h-10 w-32 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const EditCustomerButton = dynamic(
  () =>
    import('@/components/customers/edit-customer-button').then((mod) => ({
      default: mod.EditCustomerButton,
    })),
  {
    loading: () => <div className="h-8 w-16 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const DeleteCustomerButton = dynamic(
  () =>
    import('@/components/customers/delete-customer-button').then((mod) => ({
      default: mod.DeleteCustomerButton,
    })),
  {
    loading: () => <div className="h-8 w-16 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

// Lazy load pagination component (only shown when there are customers)
const CustomersPagination = dynamic(
  () =>
    import('@/components/customers/customers-pagination').then((mod) => ({
      default: mod.CustomersPagination,
    })),
  {
    loading: () => <div className="h-16 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const ITEMS_PER_PAGE = 10;

export default function MerchantCustomersPage() {
  const searchParams = useSearchParams();
  const currentPage = Number(searchParams.get('page')) || 1;

  const {
    data: customersData,
    isLoading: loading,
    error,
  } = useCustomers(currentPage, ITEMS_PER_PAGE);

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div className="h-10 w-32 bg-muted animate-pulse rounded-md" />
        </div>
        <div className="flex items-center gap-4">
          <div className="h-10 w-64 bg-muted animate-pulse rounded-md" />
          <div className="h-10 w-20 bg-muted animate-pulse rounded-md" />
        </div>
        <div className="rounded-md border">
          <TableSkeleton rows={5} columns={5} />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="text-red-500">Error: {error.message}</div>
      </div>
    );
  }

  const customers = customersData?.data || [];

  return (
    <div className="flex flex-col min-h-[calc(100vh-200px)]">
      <div className="flex-1 space-y-6">
        <div className="flex justify-end">
          <Suspense fallback={<div className="h-10 w-32 bg-muted animate-pulse rounded-md" />}>
            <CreateCustomerButton />
          </Suspense>
        </div>

        <div className="rounded-md border">
          <Suspense fallback={<div className="h-64 bg-muted animate-pulse rounded-md" />}>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[100px]">ID</TableHead>
                  <TableHead>Customer</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead className="text-right">Revenue</TableHead>
                  <TableHead className="w-[50px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {customers.map((customer) => (
                  <TableRow key={customer.id}>
                    <TableCell className="font-mono text-sm">
                      #{customer.num_id}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <div className="flex flex-col">
                          <span className="font-medium">{customer.name}</span>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col text-sm">
                        <span className="flex items-center gap-1">
                          <Suspense
                            fallback={<div className="h-3 w-3 bg-muted animate-pulse rounded-sm" />}
                          >
                            <Mail className="h-3 w-3" />
                          </Suspense>
                          {customer.email}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <span className="text-sm text-muted-foreground">
                        ${((customer.total_revenue || 0) / 100).toFixed(2)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Suspense
                        fallback={<div className="h-8 w-8 bg-muted animate-pulse rounded-md" />}
                      >
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-8 w-8">
                              <Suspense
                                fallback={
                                  <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />
                                }
                              >
                                <MoreHorizontal className="h-4 w-4" />
                              </Suspense>
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem>View Details</DropdownMenuItem>
                            <EditCustomerButton customer={customer} />
                            <DeleteCustomerButton customerId={customer.id} />
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </Suspense>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Suspense>
        </div>

        {customers.length === 0 && (
          <div className="flex flex-col items-center justify-center py-12 border rounded-lg bg-white dark:bg-neutral-900">
            <Suspense fallback={<div className="h-12 w-12 bg-muted animate-pulse rounded-full" />}>
              <Users2 className="h-12 w-12 text-muted-foreground mb-4" />
            </Suspense>
            <p className="text-muted-foreground text-center mb-4">
              No customers found. Add your first customer to get started.
            </p>
            <Suspense fallback={<div className="h-10 w-32 bg-muted animate-pulse rounded-md" />}>
              <CreateCustomerButton />
            </Suspense>
          </div>
        )}
      </div>

      {customersData && (
        <div className="mt-8 pt-4">
          <Suspense fallback={<div className="h-16 bg-muted animate-pulse rounded-md" />}>
            <CustomersPagination pageData={customersData} />
          </Suspense>
        </div>
      )}
    </div>
  );
}
