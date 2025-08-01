'use client';

import { useState, useMemo } from 'react';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { InvoiceStatusBadge } from './invoice-status-badge';
import { formatCurrency } from '@/lib/utils';
import { format } from 'date-fns';
import { ChevronRight, FileText, Download, Copy, MoreHorizontal } from 'lucide-react';
import type { Invoice } from '@/types/invoice';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { toast } from '@/components/ui/use-toast';

interface InvoiceListProps {
  invoices: Invoice[];
  isLoading?: boolean;
  onVoidInvoice?: (invoiceId: string) => Promise<void>;
  onMarkPaid?: (invoiceId: string) => Promise<void>;
  onDuplicate?: (invoiceId: string) => Promise<void>;
}

export function InvoiceList({ 
  invoices, 
  isLoading = false,
  onVoidInvoice,
  onMarkPaid,
  onDuplicate,
}: InvoiceListProps) {
  const router = useRouter();
  const [processingId, setProcessingId] = useState<string | null>(null);

  const handleCopyInvoiceNumber = (invoiceNumber: string) => {
    navigator.clipboard.writeText(invoiceNumber);
    toast({
      title: 'Copied',
      description: 'Invoice number copied to clipboard',
    });
  };

  const handleAction = async (
    action: ((id: string) => Promise<void>) | undefined,
    invoiceId: string,
    actionName: string
  ) => {
    if (!action) return;

    setProcessingId(invoiceId);
    try {
      await action(invoiceId);
      toast({
        title: 'Success',
        description: `Invoice ${actionName} successfully`,
      });
    } catch (error) {
      toast({
        title: 'Error',
        description: `Failed to ${actionName} invoice`,
        variant: 'destructive',
      });
    } finally {
      setProcessingId(null);
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="flex items-center justify-center py-8">
            <div className="animate-pulse text-muted-foreground">Loading invoices...</div>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (invoices.length === 0) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <FileText className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No invoices found</h3>
            <p className="text-sm text-muted-foreground mt-2">
              Invoices will appear here when created
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Invoices</CardTitle>
        <CardDescription>
          Manage your invoices and track payment status
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Invoice Number</TableHead>
                <TableHead>Customer</TableHead>
                <TableHead>Date</TableHead>
                <TableHead>Due Date</TableHead>
                <TableHead>Amount</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {invoices.map((invoice) => (
                <TableRow key={invoice.id}>
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                      <Link
                        href={`/merchants/invoices/${invoice.id}`}
                        className="hover:underline"
                      >
                        {invoice.invoice_number}
                      </Link>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={() => handleCopyInvoiceNumber(invoice.invoice_number)}
                      >
                        <Copy className="h-3 w-3" />
                      </Button>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Link
                      href={`/merchants/customers/${invoice.customer_id}`}
                      className="hover:underline text-sm"
                    >
                      View Customer
                    </Link>
                  </TableCell>
                  <TableCell>
                    {format(new Date(invoice.created_at), 'MMM d, yyyy')}
                  </TableCell>
                  <TableCell>
                    {invoice.due_date 
                      ? format(new Date(invoice.due_date), 'MMM d, yyyy')
                      : '-'
                    }
                  </TableCell>
                  <TableCell>
                    {formatCurrency(invoice.total_amount, invoice.currency)}
                  </TableCell>
                  <TableCell>
                    <InvoiceStatusBadge status={invoice.status} />
                  </TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          disabled={processingId === invoice.id}
                        >
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuLabel>Actions</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => router.push(`/merchants/invoices/${invoice.id}`)}
                        >
                          <FileText className="mr-2 h-4 w-4" />
                          View Details
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => window.open(`/api/invoices/${invoice.id}/pdf`, '_blank')}
                        >
                          <Download className="mr-2 h-4 w-4" />
                          Download PDF
                        </DropdownMenuItem>
                        {invoice.status === 'open' && onMarkPaid && (
                          <DropdownMenuItem
                            onClick={() => handleAction(onMarkPaid, invoice.id, 'marked as paid')}
                          >
                            Mark as Paid
                          </DropdownMenuItem>
                        )}
                        {['draft', 'open'].includes(invoice.status) && onVoidInvoice && (
                          <DropdownMenuItem
                            onClick={() => handleAction(onVoidInvoice, invoice.id, 'voided')}
                            className="text-destructive"
                          >
                            Void Invoice
                          </DropdownMenuItem>
                        )}
                        {onDuplicate && (
                          <DropdownMenuItem
                            onClick={() => handleAction(onDuplicate, invoice.id, 'duplicated')}
                          >
                            Duplicate
                          </DropdownMenuItem>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}