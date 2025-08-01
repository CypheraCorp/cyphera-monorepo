'use client';

import { useInvoicesData, useInvoiceActions } from '@/hooks/invoices/use-invoices-data';
import { 
  InvoiceList, 
  InvoicePagination, 
  InvoiceFilters 
} from '@/components/invoices';
import { PageHeader } from '@/components/ui/page-header';
import { Button } from '@/components/ui/button';
import { Plus, Download } from 'lucide-react';
import { useRouter } from 'next/navigation';

export default function InvoicesPage() {
  const router = useRouter();
  const {
    invoices,
    isLoading,
    error,
    currentPage,
    totalPages,
    hasNextPage,
    hasPreviousPage,
    pageInfo,
    statusFilter,
    customerIdFilter,
    updateStatusFilter,
    updateCustomerIdFilter,
    clearFilters,
    hasActiveFilters,
    goToPage,
    goToNextPage,
    goToPreviousPage,
    invalidateInvoices,
  } = useInvoicesData({ limit: 20 });

  const { voidInvoice, markInvoicePaid, duplicateInvoice } = useInvoiceActions();

  const handleVoidInvoice = async (invoiceId: string) => {
    await voidInvoice(invoiceId);
    invalidateInvoices();
  };

  const handleMarkPaid = async (invoiceId: string) => {
    await markInvoicePaid(invoiceId);
    invalidateInvoices();
  };

  const handleDuplicate = async (invoiceId: string) => {
    await duplicateInvoice(invoiceId);
    invalidateInvoices();
  };

  return (
    <div className="container mx-auto py-6 space-y-6">
      <PageHeader
        title="Invoices"
        description="Manage your invoices and track payment status"
      >
        <div className="flex items-center gap-2">
          <InvoiceFilters
            statusFilter={statusFilter}
            customerIdFilter={customerIdFilter}
            onStatusChange={updateStatusFilter}
            onCustomerIdChange={updateCustomerIdFilter}
            onClearFilters={clearFilters}
            hasActiveFilters={hasActiveFilters}
          />
          <Button
            variant="outline"
            size="sm"
            onClick={() => router.push('/merchants/invoices/export')}
          >
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>
          <Button
            size="sm"
            onClick={() => router.push('/merchants/invoices/new')}
          >
            <Plus className="h-4 w-4 mr-2" />
            Create Invoice
          </Button>
        </div>
      </PageHeader>

      {error ? (
        <div className="rounded-lg border border-destructive/50 p-4">
          <p className="text-sm text-destructive">
            Failed to load invoices. Please try again later.
          </p>
        </div>
      ) : (
        <>
          <InvoiceList
            invoices={invoices}
            isLoading={isLoading}
            onVoidInvoice={handleVoidInvoice}
            onMarkPaid={handleMarkPaid}
            onDuplicate={handleDuplicate}
          />

          <InvoicePagination
            currentPage={currentPage}
            totalPages={totalPages}
            hasNextPage={hasNextPage}
            hasPreviousPage={hasPreviousPage}
            pageInfo={pageInfo}
            onPageChange={goToPage}
            onNextPage={goToNextPage}
            onPreviousPage={goToPreviousPage}
          />
        </>
      )}
    </div>
  );
}