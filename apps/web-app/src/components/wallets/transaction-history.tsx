'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { format, parseISO, subDays, startOfDay, endOfDay } from 'date-fns';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  ArrowUpRight,
  ArrowDownLeft,
  ExternalLink,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  AlertCircle,
  Loader2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { WalletResponse } from '@/types/wallet';
import { NetworkWithTokensResponse } from '@/types/network';
import { CircleTransaction } from '@/types/circle';
import { CircleAPI } from '@/services/cyphera-api/circle';
import { useCircleSDK } from '@/hooks/web3';
import { formatUnits } from 'viem';
import { generateExplorerLink } from '@/lib/utils/explorers';
import { formatAddress } from '@/lib/utils/circle';
import { logger } from '@/lib/core/logger/logger-utils';

interface TransactionHistoryProps {
  wallet: WalletResponse;
  workspaceId: string;
  networks: NetworkWithTokensResponse[];
}

interface PaginationInfo {
  hasBefore: boolean;
  hasAfter: boolean;
  before?: string;
  after?: string;
}

const TRANSACTION_STATES = [
  { value: 'all', label: 'All States' },
  { value: 'INITIATED', label: 'Initiated' },
  { value: 'PENDING', label: 'Pending' },
  { value: 'CONFIRMED', label: 'Confirmed' },
  { value: 'COMPLETE', label: 'Complete' },
  { value: 'FAILED', label: 'Failed' },
  { value: 'CANCELLED', label: 'Cancelled' },
];

const PAGE_SIZE = 10;

export function TransactionHistory({
  wallet,
  workspaceId,
  networks,
}: TransactionHistoryProps) {
  const [transactions, setTransactions] = useState<CircleTransaction[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [stateFilter, setStateFilter] = useState('all');
  const [dateFrom, setDateFrom] = useState(
    format(subDays(new Date(), 30), 'yyyy-MM-dd')
  );
  const [dateTo, setDateTo] = useState(
    format(new Date(), 'yyyy-MM-dd')
  );
  const [pagination, setPagination] = useState<PaginationInfo>({
    hasBefore: false,
    hasAfter: false,
  });
  const [currentPage, setCurrentPage] = useState<'first' | string>('first');
  const { userToken } = useCircleSDK();

  // Get current network
  const currentNetwork = useMemo(() => {
    if (!wallet.network_id) return null;
    return networks.find((n) => n.network.id === wallet.network_id);
  }, [wallet.network_id, networks]);

  // Get native token for fee display
  const nativeToken = useMemo(() => {
    return currentNetwork?.tokens?.find((t) => t.gas_token);
  }, [currentNetwork]);

  const fetchTransactions = useCallback(
    async (pageToken?: string) => {
      if (!userToken || !workspaceId) {
        setError('Missing authentication');
        setIsLoading(false);
        return;
      }

      try {
        setError(null);
        const circleApi = new CircleAPI();

        const params: any = {
          wallet_ids: wallet.circle_data?.circle_wallet_id || wallet.id,
          page_size: PAGE_SIZE,
        };

        // Add state filter
        if (stateFilter !== 'all') {
          params.state = stateFilter;
        }

        // Add date range filter
        if (dateFrom) {
          params.from = startOfDay(new Date(dateFrom)).toISOString();
        }
        if (dateTo) {
          params.to = endOfDay(new Date(dateTo)).toISOString();
        }

        // Add pagination
        if (pageToken && pageToken !== 'first') {
          params.page_after = pageToken;
        }

        const response = await circleApi.listTransactions(
          { access_token: userToken, workspace_id: workspaceId },
          params
        );

        if (response?.data?.transactions) {
          setTransactions(response.data.transactions);
          setPagination({
            hasBefore: response.pagination?.hasBefore || false,
            hasAfter: response.pagination?.hasAfter || false,
            before: response.pagination?.before,
            after: response.pagination?.after,
          });
        } else {
          setTransactions([]);
        }
      } catch (err) {
        logger.error('Failed to fetch transactions:', err);
        setError('Failed to load transaction history');
      } finally {
        setIsLoading(false);
        setIsRefreshing(false);
      }
    },
    [userToken, workspaceId, wallet, stateFilter, dateFrom, dateTo]
  );

  useEffect(() => {
    setIsLoading(true);
    setCurrentPage('first');
    fetchTransactions('first');
  }, [stateFilter, dateFrom, dateTo]);

  const handleRefresh = () => {
    setIsRefreshing(true);
    fetchTransactions(currentPage);
  };

  const handlePageChange = (direction: 'next' | 'prev') => {
    if (direction === 'next' && pagination.hasAfter && pagination.after) {
      setCurrentPage(pagination.after);
      fetchTransactions(pagination.after);
    } else if (direction === 'prev' && pagination.hasBefore && pagination.before) {
      setCurrentPage(pagination.before);
      fetchTransactions(pagination.before);
    }
  };

  const getTransactionIcon = (tx: CircleTransaction) => {
    const isIncoming = tx.destinationAddress?.toLowerCase() === wallet.wallet_address.toLowerCase();
    return isIncoming ? (
      <ArrowDownLeft className="h-4 w-4 text-green-600" />
    ) : (
      <ArrowUpRight className="h-4 w-4 text-red-600" />
    );
  };

  const getTransactionAmount = (tx: CircleTransaction) => {
    if (!tx.amounts || tx.amounts.length === 0) return '0';
    
    // Find token info by checking if it's native or matching token ID
    // For now, assume native token if no tokenId
    const tokenInfo = !tx.tokenId ? nativeToken : 
      currentNetwork?.tokens?.find((t) => t.id === tx.tokenId) || nativeToken;

    const decimals = tokenInfo?.decimals || 18;
    const amount = formatUnits(BigInt(tx.amounts[0]), decimals);
    const symbol = tokenInfo?.symbol || 'UNKNOWN';

    return `${parseFloat(amount).toFixed(6)} ${symbol}`;
  };

  const getTransactionFee = (tx: CircleTransaction) => {
    if (!tx.networkFee || !nativeToken) return '--';
    
    const feeAmount = formatUnits(BigInt(tx.networkFee), nativeToken.decimals || 18);
    return `${parseFloat(feeAmount).toFixed(6)} ${nativeToken.symbol}`;
  };

  const getStatusBadge = (state: string) => {
    const variants: Record<string, { variant: any; className: string }> = {
      INITIATED: { variant: 'secondary', className: 'bg-gray-100 text-gray-700' },
      PENDING: { variant: 'secondary', className: 'bg-yellow-100 text-yellow-700' },
      CONFIRMED: { variant: 'secondary', className: 'bg-blue-100 text-blue-700' },
      COMPLETE: { variant: 'default', className: 'bg-green-100 text-green-700' },
      FAILED: { variant: 'destructive', className: '' },
      CANCELLED: { variant: 'secondary', className: 'bg-gray-100 text-gray-700' },
    };

    const config = variants[state] || variants.PENDING;
    return (
      <Badge variant={config.variant} className={config.className}>
        {state}
      </Badge>
    );
  };

  const getExplorerLink = (txHash?: string) => {
    if (!txHash || !currentNetwork) return null;
    return generateExplorerLink(
      networks,
      currentNetwork.network.chain_id,
      'tx',
      txHash
    );
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Transaction History</CardTitle>
            <CardDescription>
              View all transactions for this wallet
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={isLoading || isRefreshing}
          >
            <RefreshCw className={cn("h-4 w-4 mr-2", isRefreshing && "animate-spin")} />
            Refresh
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {/* Filters */}
        <div className="flex flex-col sm:flex-row gap-4 mb-6">
          <div className="flex-1">
            <Label htmlFor="state-filter">Status</Label>
            <Select value={stateFilter} onValueChange={setStateFilter}>
              <SelectTrigger id="state-filter">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {TRANSACTION_STATES.map((state) => (
                  <SelectItem key={state.value} value={state.value}>
                    {state.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex-1">
            <Label htmlFor="date-from">From Date</Label>
            <Input
              id="date-from"
              type="date"
              value={dateFrom}
              onChange={(e) => setDateFrom(e.target.value)}
              max={dateTo}
            />
          </div>

          <div className="flex-1">
            <Label htmlFor="date-to">To Date</Label>
            <Input
              id="date-to"
              type="date"
              value={dateTo}
              onChange={(e) => setDateTo(e.target.value)}
              min={dateFrom}
              max={format(new Date(), 'yyyy-MM-dd')}
            />
          </div>
        </div>

        {/* Transaction List */}
        {isLoading ? (
          <div className="space-y-3">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        ) : error ? (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : transactions.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No transactions found for the selected filters
          </div>
        ) : (
          <>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[50px]">Type</TableHead>
                    <TableHead>Date</TableHead>
                    <TableHead>To/From</TableHead>
                    <TableHead>Amount</TableHead>
                    <TableHead>Fee</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactions.map((tx) => {
                    const isIncoming = tx.destinationAddress?.toLowerCase() === 
                      wallet.wallet_address.toLowerCase();
                    const otherAddress = isIncoming 
                      ? tx.sourceAddress 
                      : tx.destinationAddress;
                    const explorerLink = getExplorerLink(tx.txHash);

                    return (
                      <TableRow key={tx.id}>
                        <TableCell>{getTransactionIcon(tx)}</TableCell>
                        <TableCell>
                          {format(parseISO(tx.createDate), 'MMM d, yyyy HH:mm')}
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-col">
                            <span className="text-sm">
                              {isIncoming ? 'From' : 'To'}
                            </span>
                            <code className="text-xs">
                              {formatAddress(otherAddress || 'Unknown')}
                            </code>
                          </div>
                        </TableCell>
                        <TableCell>{getTransactionAmount(tx)}</TableCell>
                        <TableCell>{getTransactionFee(tx)}</TableCell>
                        <TableCell>{getStatusBadge(tx.state)}</TableCell>
                        <TableCell className="text-right">
                          {explorerLink && (
                            <Button
                              variant="ghost"
                              size="icon"
                              asChild
                            >
                              <a
                                href={explorerLink}
                                target="_blank"
                                rel="noopener noreferrer"
                              >
                                <ExternalLink className="h-4 w-4" />
                              </a>
                            </Button>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>

            {/* Pagination */}
            {(pagination.hasBefore || pagination.hasAfter) && (
              <div className="flex items-center justify-between mt-4">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handlePageChange('prev')}
                  disabled={!pagination.hasBefore}
                >
                  <ChevronLeft className="h-4 w-4 mr-2" />
                  Previous
                </Button>
                <div className="text-sm text-muted-foreground">
                  Showing up to {PAGE_SIZE} transactions
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handlePageChange('next')}
                  disabled={!pagination.hasAfter}
                >
                  Next
                  <ChevronRight className="h-4 w-4 ml-2" />
                </Button>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}