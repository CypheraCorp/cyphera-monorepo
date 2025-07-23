'use client';

import { useWalletsPageData } from '@/hooks/data';
import { Suspense } from 'react';
import { CardSkeleton } from '@/components/ui/loading-states';
import dynamic from 'next/dynamic';

// Dynamically import components to reduce initial bundle size

const WalletList = dynamic(
  () => import('@/components/wallets/wallet-list').then((mod) => ({ default: mod.WalletList })),
  {
    loading: () => <div className="h-64 w-full bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const WalletActions = dynamic(
  () =>
    import('@/components/wallets/wallet-actions').then((mod) => ({ default: mod.WalletActions })),
  {
    loading: () => <div className="h-10 w-full bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

/**
 * Wallets page component
 * Lists all wallets for the current account
 */
export default function WalletsPage() {
  const { wallets, networks, isLoading: loading, error } = useWalletsPageData();

  if (loading) {
    return (
      <Suspense fallback={<div className="h-screen w-full bg-muted animate-pulse rounded-md" />}>
        <div className="container mx-auto py-6 px-4">
          <div className="mb-6">
            <div className="h-8 w-48 bg-muted animate-pulse rounded-md mb-2" />
            <div className="h-4 w-96 bg-muted animate-pulse rounded-md" />
          </div>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <CardSkeleton key={i} />
            ))}
          </div>
        </div>
      </Suspense>
    );
  }

  if (error) {
    return (
      <Suspense fallback={<div className="h-screen w-full bg-muted animate-pulse rounded-md" />}>
        <div className="container mx-auto py-6 px-4">
          <div className="h-12 w-full bg-muted animate-pulse rounded-md" />
          <div className="flex items-center justify-center py-8">
            <div className="text-red-500">Error: {error.message}</div>
          </div>
        </div>
      </Suspense>
    );
  }

  const safeWallets = wallets || [];
  const safeNetworks = networks || [];

  return (
    <Suspense fallback={<div className="h-screen w-full bg-muted animate-pulse rounded-md" />}>
      <div className="container mx-auto py-6 px-4">
        <div className="flex flex-col space-y-6">
          <div className="flex justify-end">
            <Suspense fallback={<div className="h-10 w-full bg-muted animate-pulse rounded-md" />}>
              <WalletActions networks={safeNetworks} />
            </Suspense>
          </div>
          <Suspense fallback={<div className="h-64 w-full bg-muted animate-pulse rounded-md" />}>
            <WalletList initialWallets={safeWallets} networks={safeNetworks} />
          </Suspense>
        </div>
      </div>
    </Suspense>
  );
}
