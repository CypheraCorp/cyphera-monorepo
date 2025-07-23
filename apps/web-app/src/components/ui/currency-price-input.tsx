'use client';

import { useState, useEffect, useMemo } from 'react';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import { CURRENCY_SYMBOLS } from '@/lib/constants/currency';

interface CurrencyPriceInputProps {
  currency: string;
  onCurrencyChange: (currency: string) => void;
  priceInPennies: number;
  onPriceChange: (priceInPennies: number) => void;
  supportedCurrencies: readonly string[];
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

export function CurrencyPriceInput({
  currency,
  onCurrencyChange,
  priceInPennies,
  onPriceChange,
  supportedCurrencies,
  placeholder,
  disabled = false,
  className,
}: CurrencyPriceInputProps) {
  // Convert pennies to natural display value (no forced formatting)
  const displayValue = useMemo(() => {
    if (priceInPennies === 0) return '';
    const value = priceInPennies / 100;
    // Return natural decimal representation without forced zeros
    return value % 1 === 0 ? value.toString() : value.toString();
  }, [priceInPennies]);

  const [inputValue, setInputValue] = useState(displayValue);

  // Update input value when priceInPennies changes externally
  useEffect(() => {
    setInputValue(displayValue);
  }, [displayValue]);

  const currencySymbol = CURRENCY_SYMBOLS[currency] || currency;

  // Handle natural character-by-character input
  const handleInputChange = (value: string) => {
    // Remove non-numeric characters except decimal point
    const cleanValue = value.replace(/[^0-9.]/g, '');

    // Ensure only one decimal point and max 2 decimal places
    const parts = cleanValue.split('.');
    let finalValue = parts[0]; // Integer part

    if (parts.length > 1) {
      // Only allow up to 2 decimal places
      const decimalPart = parts[1].substring(0, 2);
      finalValue += '.' + decimalPart;
    }

    // Update input value as user types
    setInputValue(finalValue);

    // Convert to pennies for parent component
    const numericValue = parseFloat(finalValue) || 0;
    const pennies = Math.round(numericValue * 100);
    onPriceChange(pennies);
  };

  // Natural preview formatting - only format what user has typed
  const getPreviewDisplay = (value: string) => {
    if (!value) return '';

    const numValue = parseFloat(value);
    if (isNaN(numValue)) return '';

    // Different currencies have different symbol positions
    switch (currency) {
      case 'EUR':
        return `${value}${currencySymbol}`;
      case 'USD':
      default:
        return `${currencySymbol}${value}`;
    }
  };

  const previewText = inputValue ? getPreviewDisplay(inputValue) : '';

  return (
    <div className={cn('space-y-2', className)}>
      {/* Currency and Price Input Row */}
      <div className="flex gap-2">
        {/* Currency Selector */}
        <div className="w-24">
          <Select value={currency} onValueChange={onCurrencyChange} disabled={disabled}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {supportedCurrencies.map((curr) => (
                <SelectItem key={curr} value={curr}>
                  {curr}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Price Input */}
        <div className="flex-1 relative">
          <Input
            type="text"
            inputMode="decimal"
            value={inputValue}
            onChange={(e) => handleInputChange(e.target.value)}
            placeholder={placeholder || `19.99`}
            disabled={disabled}
            className="pl-8"
          />
          {/* Currency Symbol Overlay */}
          <div className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none text-sm">
            {currencySymbol}
          </div>
        </div>
      </div>

      {/* Price Preview */}
      {previewText && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Customer will pay:</span>
          <span className="font-medium text-green-600 dark:text-green-400">{previewText}</span>
        </div>
      )}
    </div>
  );
}
