'use client';

import { Button } from '@/components/ui/button';
import { format } from 'date-fns';
import { useSearchParams } from 'next/navigation';
import { useSubscriptions } from '@/hooks/data';
import { Suspense } from 'react';
import { TableSkeleton } from '@/components/ui/loading-states';
import { formatBillingInterval } from '@/lib/utils/format/billing';
import dynamic from 'next/dynamic';

// Dynamically import lucide-react icons

const MoreHorizontal = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.MoreHorizontal })),
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

const SubscriptionsPagination = dynamic(
  () =>
    import('@/components/subscriptions/subscriptions-pagination').then((mod) => ({
      default: mod.SubscriptionsPagination,
    })),
  {
    loading: () => <div className="h-10 w-full bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const ITEMS_PER_PAGE = 10;

export default function SubscriptionsPage() {
  const searchParams = useSearchParams();
  const currentPage = Number(searchParams.get('page')) || 1;

  // Use React Query hook instead of manual state management
  const {
    data: subscriptionsData,
    isLoading: loading,
    error,
  } = useSubscriptions(currentPage, ITEMS_PER_PAGE);

  function getStatusColor(status: string) {
    switch (status) {
      case 'active':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100';
      case 'canceled':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-100';
      case 'past_due':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-100';
      case 'expired':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100';
      case 'completed':
        return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-100';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100';
    }
  }

  function formatSubscriptionStatus(status: string) {
    switch (status) {
      case 'active':
        return 'Active';
      case 'canceled':
        return 'Canceled';
      case 'past_due':
        return 'Past Due';
      case 'expired':
        return 'Expired';
      case 'completed':
        return 'Completed';
      default:
        return status;
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

  const subscriptions = subscriptionsData?.data || [];
  const total = subscriptionsData?.pagination?.total_items || 0;
  const hasMore = subscriptionsData?.has_more || false;
  const startItem = total === 0 ? 0 : (currentPage - 1) * ITEMS_PER_PAGE + 1;
  const endItem = Math.min(currentPage * ITEMS_PER_PAGE, total);

  return (
    <div className="flex flex-col min-h-[calc(100vh-200px)]">
      <div className="flex-1 space-y-6">
        <div className="rounded-md border">
          <Suspense fallback={<div className="h-64 bg-muted animate-pulse rounded-md" />}>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Customer</TableHead>
                  <TableHead>Product</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Amount</TableHead>
                  <TableHead>Billing</TableHead>
                  <TableHead>Current Period</TableHead>
                  <TableHead className="w-[50px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {subscriptions.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center py-8">
                      <div className="text-muted-foreground">No subscriptions found</div>
                    </TableCell>
                  </TableRow>
                ) : (
                  subscriptions.map((subscription) => (
                    <TableRow key={subscription.id}>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-medium">
                            {subscription.customer_name || 'Unknown'}
                          </span>
                          <span className="text-sm text-muted-foreground">
                            {subscription.customer_email || 'No email'}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-medium">{subscription.product.name}</span>
                          <span className="text-sm text-muted-foreground">
                            {subscription.product_token.token_symbol} on{' '}
                            {subscription.product_token.network_name}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Suspense
                          fallback={<div className="h-5 w-12 bg-muted animate-pulse rounded-md" />}
                        >
                          <Badge className={getStatusColor(subscription.status)}>
                            {formatSubscriptionStatus(subscription.status)}
                          </Badge>
                        </Suspense>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-medium">
                            ${(subscription.price.unit_amount_in_pennies / 100).toFixed(2)}
                          </span>
                          <span className="text-sm text-muted-foreground">
                            {subscription.price.currency?.toUpperCase() || 'USD'}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-medium">
                            {formatBillingInterval(
                              subscription.price.interval_type,
                              subscription.price.interval_count
                            )}
                          </span>
                          {subscription.price.term_length && (
                            <span className="text-sm text-muted-foreground">
                              {subscription.price.term_length} term{subscription.price.term_length > 1 ? 's' : ''}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col text-sm">
                          <span>
                            {subscription.current_period_start
                              ? format(new Date(subscription.current_period_start), 'MMM dd, yyyy')
                              : 'N/A'}
                          </span>
                          <span className="text-muted-foreground">
                            to{' '}
                            {subscription.current_period_end
                              ? format(new Date(subscription.current_period_end), 'MMM dd, yyyy')
                              : 'N/A'}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Suspense
                          fallback={<div className="h-8 w-8 bg-muted animate-pulse rounded-md" />}
                        >
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem>View Details</DropdownMenuItem>
                              <DropdownMenuItem>Cancel Subscription</DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </Suspense>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </Suspense>
        </div>
      </div>

      {subscriptionsData && (
        <div className="mt-8 pt-4">
          <Suspense fallback={<div className="h-10 w-full bg-muted animate-pulse rounded-md" />}>
            <SubscriptionsPagination
              currentPage={currentPage}
              hasMore={hasMore}
              startItem={startItem}
              endItem={endItem}
              total={total}
            />
          </Suspense>
        </div>
      )}
    </div>
  );
}
