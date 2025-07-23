'use client';

import { useEffect, useState } from 'react';
import { Receipt, MoreHorizontal, ArrowUpRight, ArrowDownRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import type { TransactionResponse } from '@/types/transaction';
import { TableSkeleton } from '@/components/ui/loading-states';
import { ErrorState } from '@/components/ui/error-state';
import { EmptyState } from '@/components/ui/empty-state';

/**
 * TransactionList component
 * Displays a list of transactions with their details and actions
 */
export function TransactionList() {
  const [transactions, setTransactions] = useState<TransactionResponse[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchTransactions() {
      try {
        const response = await fetch('/api/transactions');
        if (!response.ok) {
          throw new Error('Failed to fetch transactions');
        }
        const data = await response.json();
        setTransactions(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch transactions');
      } finally {
        setIsLoading(false);
      }
    }

    fetchTransactions();
  }, []);

  const getStatusColor = (status: TransactionResponse['status']) => {
    switch (status) {
      case 'succeeded':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100';
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-100';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-100';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100';
    }
  };

  const getTypeIcon = (type: TransactionResponse['type']) => {
    switch (type) {
      case 'charge':
        return <ArrowUpRight className="h-4 w-4 text-green-600" />;
      case 'refund':
        return <ArrowDownRight className="h-4 w-4 text-red-600" />;
      default:
        return null;
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <div className="flex-1">
            <Input placeholder="Search transactions..." className="max-w-sm" disabled />
          </div>
          <Button variant="outline" disabled>
            Filter
          </Button>
        </div>
        <TableSkeleton rows={5} columns={8} />
      </div>
    );
  }

  if (error) {
    return (
      <ErrorState
        title="Failed to load transactions"
        message={error}
        onRetry={() => {
          setIsLoading(true);
          setError(null);
          window.location.reload();
        }}
      />
    );
  }

  if (transactions.length === 0) {
    return (
      <EmptyState
        icon={Receipt}
        title="No transactions found"
        description="Your transactions will appear here once customers start making payments."
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <div className="flex-1">
          <Input placeholder="Search transactions..." className="max-w-sm" />
        </div>
        <Button variant="outline">Filter</Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Date</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Amount</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Customer</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>Payment Method</TableHead>
              <TableHead className="w-[50px]"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {transactions.map((transaction) => (
              <TableRow key={transaction.id}>
                <TableCell className="font-medium">{transaction.date}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-1">
                    {getTypeIcon(transaction.type)}
                    <span className="capitalize">{transaction.type}</span>
                  </div>
                </TableCell>
                <TableCell
                  className={transaction.type === 'refund' ? 'text-red-600' : 'text-green-600'}
                >
                  ${transaction.amount.toLocaleString()}
                </TableCell>
                <TableCell>
                  <Badge className={getStatusColor(transaction.status)}>{transaction.status}</Badge>
                </TableCell>
                <TableCell>{transaction.customer}</TableCell>
                <TableCell className="max-w-[200px] truncate">{transaction.description}</TableCell>
                <TableCell>{transaction.paymentMethod}</TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-8 w-8">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem>View Details</DropdownMenuItem>
                      <DropdownMenuItem>Download Receipt</DropdownMenuItem>
                      {transaction.type === 'charge' && (
                        <DropdownMenuItem className="text-red-600">Issue Refund</DropdownMenuItem>
                      )}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
