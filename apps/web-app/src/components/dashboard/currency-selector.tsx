'use client';

import { useCurrency } from '@/hooks/use-currency';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';

export function CurrencySelector() {
  const {
    currencies,
    defaultCurrency,
    isLoading,
    setDefaultCurrency,
    isSettingDefault,
  } = useCurrency();

  if (isLoading) {
    return <Skeleton className="h-10 w-32" />;
  }

  if (!currencies.length) {
    return null;
  }

  return (
    <Select
      value={defaultCurrency?.code}
      onValueChange={setDefaultCurrency}
      disabled={isSettingDefault}
    >
      <SelectTrigger className="w-32">
        <SelectValue placeholder="Select currency" />
      </SelectTrigger>
      <SelectContent>
        {currencies.map((currency) => (
          <SelectItem key={currency.code} value={currency.code}>
            {currency.symbol} {currency.code}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}