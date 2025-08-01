'use client';

import { useState, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { X, Filter } from 'lucide-react';
import type { InvoiceStatus } from '@/types/invoice';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';

interface InvoiceFiltersProps {
  statusFilter?: InvoiceStatus;
  customerIdFilter?: string;
  onStatusChange: (status: InvoiceStatus | undefined) => void;
  onCustomerIdChange: (customerId: string | undefined) => void;
  onClearFilters: () => void;
  hasActiveFilters: boolean;
}

const statusOptions: { value: InvoiceStatus; label: string }[] = [
  { value: 'draft', label: 'Draft' },
  { value: 'open', label: 'Open' },
  { value: 'paid', label: 'Paid' },
  { value: 'void', label: 'Void' },
  { value: 'uncollectible', label: 'Uncollectible' },
];

export function InvoiceFilters({
  statusFilter,
  customerIdFilter,
  onStatusChange,
  onCustomerIdChange,
  onClearFilters,
  hasActiveFilters,
}: InvoiceFiltersProps) {
  const [localCustomerId, setLocalCustomerId] = useState(customerIdFilter || '');
  const [isOpen, setIsOpen] = useState(false);

  const handleCustomerIdSubmit = useCallback(() => {
    const trimmedId = localCustomerId.trim();
    onCustomerIdChange(trimmedId || undefined);
  }, [localCustomerId, onCustomerIdChange]);

  const handleClearCustomerId = useCallback(() => {
    setLocalCustomerId('');
    onCustomerIdChange(undefined);
  }, [onCustomerIdChange]);

  const activeFilterCount = [statusFilter, customerIdFilter].filter(Boolean).length;

  return (
    <div className="flex items-center gap-2">
      <Popover open={isOpen} onOpenChange={setIsOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" size="sm" className="gap-2">
            <Filter className="h-4 w-4" />
            Filters
            {activeFilterCount > 0 && (
              <span className="ml-1 rounded-full bg-primary px-2 py-0.5 text-xs text-primary-foreground">
                {activeFilterCount}
              </span>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-80" align="end">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="status-filter">Status</Label>
              <Select
                value={statusFilter || 'all'}
                onValueChange={(value) => onStatusChange(value === 'all' ? undefined : value as InvoiceStatus)}
              >
                <SelectTrigger id="status-filter">
                  <SelectValue placeholder="All statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  {statusOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="customer-filter">Customer ID</Label>
              <div className="flex gap-2">
                <Input
                  id="customer-filter"
                  placeholder="Enter customer ID"
                  value={localCustomerId}
                  onChange={(e) => setLocalCustomerId(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleCustomerIdSubmit();
                    }
                  }}
                />
                {localCustomerId && (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleClearCustomerId}
                    className="h-9 w-9"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                )}
              </div>
              {localCustomerId !== customerIdFilter && (
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={handleCustomerIdSubmit}
                  className="w-full"
                >
                  Apply Customer Filter
                </Button>
              )}
            </div>

            {hasActiveFilters && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  onClearFilters();
                  setLocalCustomerId('');
                  setIsOpen(false);
                }}
                className="w-full"
              >
                Clear All Filters
              </Button>
            )}
          </div>
        </PopoverContent>
      </Popover>

      {/* Quick filter chips */}
      {statusFilter && (
        <div className="flex items-center gap-1 rounded-md bg-secondary px-2 py-1">
          <span className="text-sm">Status: {statusFilter}</span>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onStatusChange(undefined)}
            className="h-4 w-4 p-0 hover:bg-transparent"
          >
            <X className="h-3 w-3" />
          </Button>
        </div>
      )}

      {customerIdFilter && (
        <div className="flex items-center gap-1 rounded-md bg-secondary px-2 py-1">
          <span className="text-sm">Customer: {customerIdFilter.slice(0, 8)}...</span>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => {
              onCustomerIdChange(undefined);
              setLocalCustomerId('');
            }}
            className="h-4 w-4 p-0 hover:bg-transparent"
          >
            <X className="h-3 w-3" />
          </Button>
        </div>
      )}
    </div>
  );
}