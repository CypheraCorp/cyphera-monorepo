'use client';

import { useRouter } from 'next/navigation';
import { PaginationControls } from '@/components/pagination-controls';
import { PaginatedResponse } from '@/types/common';
import { CustomerResponse } from '@/types/customer';

export function CustomersPagination({
  pageData,
}: {
  pageData: PaginatedResponse<CustomerResponse>;
}) {
  const router = useRouter();

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
    // Adjust startItem: must be at least 1 and not exceed totalItems.
    startItem = Math.max(1, Math.min(startItem, totalItems));

    endItem = Math.min(currentPage * perPage, totalItems);
    // Adjust endItem: must be at least startItem (especially if startItem was capped).
    endItem = Math.max(startItem, endItem);
  }

  return (
    <div className="mt-auto pt-6 border-t">
      <PaginationControls
        currentPage={currentPage}
        hasMore={pageData.has_more}
        startItem={startItem}
        endItem={endItem}
        total={totalItems}
        onPageChange={(page) => {
          router.push(`/customers?page=${page}`);
        }}
      />
    </div>
  );
}
