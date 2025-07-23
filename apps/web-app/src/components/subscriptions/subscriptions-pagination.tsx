'use client';

import { useRouter } from 'next/navigation';
import { useTransition } from 'react';
import { PaginationControls } from '@/components/pagination-controls';

interface SubscriptionsPaginationProps {
  currentPage: number;
  hasMore: boolean;
  startItem: number;
  endItem: number;
  total: number;
}

export function SubscriptionsPagination({
  currentPage,
  hasMore,
  startItem,
  endItem,
  total,
}: SubscriptionsPaginationProps) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const handlePageChange = (page: number) => {
    startTransition(() => {
      router.push(`/subscriptions?page=${page}`);
    });
  };

  return (
    <div className={`mt-auto pt-6 border-t ${isPending ? 'opacity-70' : ''}`}>
      <PaginationControls
        currentPage={currentPage}
        hasMore={hasMore}
        startItem={startItem}
        endItem={endItem}
        total={total}
        onPageChange={handlePageChange}
      />
    </div>
  );
}
