'use client';

import { useRouter } from 'next/navigation';
import { PaginationControls } from '@/components/pagination-controls';
import { PaginatedResponse } from '@/types/common';
import { ProductResponse } from '@/types/product';

export function ProductsPagination({ pageData }: { pageData: PaginatedResponse<ProductResponse> }) {
  const router = useRouter();

  const currentPage = pageData.pagination?.current_page || 1;
  const perPage = pageData.pagination?.per_page || 10;
  const totalItems = pageData.pagination?.total_items || 0;

  // Fallback: If pagination data is missing or incorrect, use the actual data length
  const actualDataLength = pageData.data?.length || 0;
  const effectiveTotalItems = totalItems > 0 ? totalItems : actualDataLength;

  let startItem: number;
  let endItem: number;

  if (effectiveTotalItems === 0) {
    startItem = 0;
    endItem = 0;
  } else {
    startItem = (currentPage - 1) * perPage + 1;
    // Adjust startItem: must be at least 1 and not exceed effectiveTotalItems.
    startItem = Math.max(1, Math.min(startItem, effectiveTotalItems));

    endItem = Math.min(currentPage * perPage, effectiveTotalItems);
    // Adjust endItem: must be at least startItem (especially if startItem was capped).
    endItem = Math.max(startItem, endItem);

    // If we have actual data but no pagination, show the actual count
    if (totalItems === 0 && actualDataLength > 0) {
      startItem = 1;
      endItem = actualDataLength;
    }
  }

  return (
    <div className="mt-auto pt-6 border-t">
      <PaginationControls
        currentPage={currentPage}
        hasMore={pageData.has_more}
        startItem={startItem}
        endItem={endItem}
        total={effectiveTotalItems}
        onPageChange={(page) => {
          router.push(`/merchants/products?page=${page}`);
        }}
      />
    </div>
  );
}
