'use client';

import { Button } from '@/components/ui/button';
import { format } from 'date-fns';
import { generateExplorerLink } from '@/lib/utils/explorers';
import { useSearchParams } from 'next/navigation';
import { usePayments, useNetworks } from '@/hooks/data';
import { Suspense } from 'react';
import { TableSkeleton } from '@/components/ui/loading-states';
import dynamic from 'next/dynamic';

// Dynamically import lucide-react icons
const Calendar = dynamic(() => import('lucide-react').then((mod) => ({ default: mod.Calendar })), {
  loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />,
  ssr: false,
});

const MoreHorizontal = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.MoreHorizontal })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />,
    ssr: false,
  }
);

const ExternalLink = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.ExternalLink })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />,
    ssr: false,
  }
);

// Dynamically import heavy table components to reduce initial bundle size
const Table = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.Table })),
  {
    loading: () => <div className="h-64 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const TableBody = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableBody })),
  {
    loading: () => <tbody className="h-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const TableCell = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableCell })),
  {
    loading: () => <td className="h-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const TableHead = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableHead })),
  {
    loading: () => <th className="h-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const TableHeader = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableHeader })),
  {
    loading: () => <thead className="h-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const TableRow = dynamic(
  () => import('@/components/ui/table').then((mod) => ({ default: mod.TableRow })),
  {
    loading: () => <tr className="h-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const DropdownMenu = dynamic(
  () => import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenu })),
  {
    loading: () => <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const DropdownMenuContent = dynamic(
  () =>
    import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenuContent })),
  {
    loading: () => <div className="h-20 w-32 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const DropdownMenuItem = dynamic(
  () => import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenuItem })),
  {
    loading: () => <div className="h-8 w-full bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const DropdownMenuTrigger = dynamic(
  () =>
    import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenuTrigger })),
  {
    loading: () => <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const Badge = dynamic(
  () => import('@/components/ui/badge').then((mod) => ({ default: mod.Badge })),
  {
    loading: () => <div className="h-5 w-12 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const PaymentsPagination = dynamic(
  () =>
    import('@/components/pagination/generic-pagination').then((mod) => ({
      default: mod.GenericPagination,
    })),
  {
    loading: () => <div className="h-10 w-full bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const ITEMS_PER_PAGE = 10;

export default function PaymentsPage() {
  const searchParams = useSearchParams();
  const currentPage = Number(searchParams.get('page')) || 1;

  // Use React Query hooks instead of manual state management
  const {
    data: paymentsData,
    isLoading: paymentsLoading,
    error: paymentsError,
  } = usePayments(currentPage, ITEMS_PER_PAGE);
  const { data: networks, isLoading: networksLoading, error: networksError } = useNetworks(true);

  const loading = paymentsLoading || networksLoading;
  const error = paymentsError || networksError;

  function getPaymentStatusLabel(status: string): string {
    switch (status) {
      case 'completed':
        return 'Completed';
      case 'pending':
        return 'Pending';
      case 'failed':
        return 'Failed';
      case 'processing':
        return 'Processing';
      default:
        return status;
    }
  }

  function getPaymentStatusColor(status: string): string {
    switch (status) {
      case 'completed':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-100';
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-100';
      case 'processing':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-100';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100';
    }
  }

  function getPaymentMethodLabel(method: string): string {
    switch (method) {
      case 'crypto':
        return 'Crypto';
      case 'card':
        return 'Card';
      case 'bank':
        return 'Bank';
      default:
        return method;
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-2">
            <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />
            <div className="h-8 w-32 bg-muted animate-pulse rounded-md" />
          </div>
          <div className="flex items-center gap-4">
            <div className="h-10 w-64 bg-muted animate-pulse rounded-md" />
            <div className="h-10 w-16 bg-muted animate-pulse rounded-md" />
          </div>
        </div>
        <div className="rounded-md border">
          <TableSkeleton rows={5} columns={7} />
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

  if (!paymentsData || !networks) {
    return (
      <div className="flex h-[200px] items-center justify-center">
        <div className="text-center">
          <p className="text-muted-foreground">Failed to load data</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col min-h-[calc(100vh-200px)]">
      <div className="flex-1 space-y-6">
        <div className="rounded-md border">
          <Suspense fallback={<div className="h-64 bg-muted animate-pulse rounded-md" />}>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Transaction</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Date</TableHead>
                  <TableHead>Customer</TableHead>
                  <TableHead>Product</TableHead>
                  <TableHead>Method</TableHead>
                  <TableHead className="text-right">Amount</TableHead>
                  <TableHead className="w-[50px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paymentsData?.data?.map((payment) => {
                  const explorerLink = payment.network && payment.transaction_hash
                    ? generateExplorerLink(
                        networks,
                        payment.network.chain_id,
                        'tx',
                        payment.transaction_hash
                      )
                    : null;

                  return (
                    <TableRow key={payment.id}>
                      <TableCell>
                        {payment.transaction_hash ? (
                          <a
                            href={explorerLink || '#'}
                            target="_blank"
                            rel="noopener noreferrer"
                            className={`flex items-center ${explorerLink ? 'text-blue-600 hover:underline' : 'text-muted-foreground cursor-not-allowed'}`}
                            title={explorerLink ? 'View on explorer' : 'Explorer link unavailable'}
                          >
                            <span className="text-sm truncate max-w-[120px]">
                              {payment.transaction_hash.substring(0, 8)}...
                              {payment.transaction_hash.substring(
                                payment.transaction_hash.length - 6
                              )}
                            </span>
                            {explorerLink && (
                              <Suspense
                                fallback={
                                  <div className="h-3 w-3 bg-muted animate-pulse rounded-sm ml-1" />
                                }
                              >
                                <ExternalLink className="h-3 w-3 ml-1" />
                              </Suspense>
                            )}
                          </a>
                        ) : (
                          <span className="text-sm text-muted-foreground">No transaction</span>
                        )}
                        {payment.error_message && (
                          <div className="text-xs text-red-500 mt-1">
                            Error: {payment.error_message}
                          </div>
                        )}
                      </TableCell>
                      <TableCell>
                        <Suspense
                          fallback={<div className="h-5 w-12 bg-muted animate-pulse rounded-md" />}
                        >
                          <Badge className={getPaymentStatusColor(payment.status)}>
                            {getPaymentStatusLabel(payment.status)}
                          </Badge>
                        </Suspense>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Suspense
                            fallback={<div className="h-3 w-3 bg-muted animate-pulse rounded-sm" />}
                          >
                            <Calendar className="h-3 w-3" />
                          </Suspense>
                          <span className="text-sm">
                            {format(new Date(payment.created_at), 'MMM d, yyyy')}
                          </span>
                        </div>
                        <div className="text-sm text-muted-foreground">
                          {format(new Date(payment.created_at), 'h:mm a')}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-medium">
                            {payment.customer?.name || 'Unknown'}
                          </span>
                          <span className="text-sm text-muted-foreground">
                            {payment.customer?.email || 'No email'}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col">
                          {payment.product_name ? (
                            <>
                              <span className="font-medium">{payment.product_name}</span>
                              {payment.network && payment.token && (
                                <span className="text-sm text-muted-foreground">
                                  {payment.token.symbol} on {payment.network.name}
                                </span>
                              )}
                            </>
                          ) : (
                            <span className="text-sm text-muted-foreground">No product</span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">
                          {getPaymentMethodLabel(payment.payment_method)}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {payment.formatted_amount || `$${(payment.amount_in_cents / 100).toFixed(2)}`}
                      </TableCell>
                      <TableCell>
                        <Suspense
                          fallback={<div className="h-8 w-8 bg-muted animate-pulse rounded-md" />}
                        >
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon" className="h-8 w-8">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem>View Details</DropdownMenuItem>
                              {explorerLink && (
                                <DropdownMenuItem>
                                  <a
                                    href={explorerLink}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="flex items-center"
                                  >
                                    View on Explorer
                                    <Suspense
                                      fallback={
                                        <div className="h-3 w-3 bg-muted animate-pulse rounded-sm ml-1" />
                                      }
                                    >
                                      <ExternalLink className="h-3 w-3 ml-1" />
                                    </Suspense>
                                  </a>
                                </DropdownMenuItem>
                              )}
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </Suspense>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </Suspense>
        </div>

        <div className="text-sm text-muted-foreground">
          {paymentsData?.data?.length || 0} payments found
        </div>
      </div>

      {paymentsData && (
        <div className="mt-8 pt-4">
          <Suspense fallback={<div className="h-10 w-32 bg-muted animate-pulse rounded-md" />}>
            <PaymentsPagination pageData={paymentsData} basePath="/merchants/payments" />
          </Suspense>
        </div>
      )}
    </div>
  );
}