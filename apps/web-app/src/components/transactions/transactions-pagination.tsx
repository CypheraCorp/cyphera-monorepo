'use client';

import { useRouter } from 'next/navigation';
import { useTransition } from 'react';
import { PaginationControls } from '@/components/pagination-controls';
import { PaginatedResponse } from '@/types/common';
import { SubscriptionEventFullResponse } from '@/types/subscription-event';

export function TransactionsPagination({
  pageData,
}: {
  pageData: PaginatedResponse<SubscriptionEventFullResponse>;
}) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const currentPage = pageData.pagination?.current_page || 1;
  const perPage = pageData.pagination?.per_page || 10;
  const totalItems = pageData.pagination?.total_items || 0;

  let startItem: number;
  let endItem: number;

  if (totalItems === 0) {
    startItem = 0;
    endItem = 0;
  } else {
    startItem = (currentPage - 1) * perPage + 1;
    startItem = Math.max(1, Math.min(startItem, totalItems)); // Ensure startItem is at least 1 and not > total
    endItem = Math.min(currentPage * perPage, totalItems);
    endItem = Math.max(startItem, endItem); // Ensure endItem is at least startItem
  }

  const handlePageChange = (page: number) => {
    startTransition(() => {
      router.push(`/transactions?page=${page}`);
    });
  };

  return (
    <div className={`mt-auto pt-6 border-t ${isPending ? 'opacity-70' : ''}`}>
      <PaginationControls
        currentPage={currentPage}
        hasMore={pageData.has_more}
        startItem={startItem}
        endItem={endItem}
        total={totalItems}
        onPageChange={handlePageChange}
      />
    </div>
  );
}
