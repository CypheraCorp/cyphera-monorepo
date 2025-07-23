'use client';

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { useState } from 'react';
import { NetworkWithTokensResponse } from '@/types/network';
import { TokenResponse } from '@/types/token';

interface NetworkTokenSelectorProps {
  networks: NetworkWithTokensResponse[];
  value: string[];
  onChange: (value: string[]) => void;
}

export function NetworkTokenSelector({ networks, value, onChange }: NetworkTokenSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);

  // Create a flat list of all network/token combinations
  const allOptions = networks.flatMap((network) =>
    network.tokens.map((token: TokenResponse) => ({
      value: `${network.network.id}:${token.id}`,
      networkName: network.network.name,
      tokenName: token.name,
      symbol: token.symbol,
      gasToken: token.gas_token,
      label: `${network.network.name} - ${token.symbol}`,
    }))
  );

  const handleValueChange = (selectedValue: string) => {
    if (value.includes(selectedValue)) {
      onChange(value.filter((v) => v !== selectedValue));
    } else {
      onChange([...value, selectedValue]);
    }
    setIsOpen(false); // Close the select after selection
  };

  return (
    <div className="flex flex-col gap-2">
      <Select
        open={isOpen}
        onOpenChange={setIsOpen}
        value={value[value.length - 1] || ''} // Use the last selected value for display
        onValueChange={handleValueChange}
      >
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Select payment methods...">
            {value.length === 0
              ? 'Select payment methods...'
              : `${value.length} payment method${value.length === 1 ? '' : 's'} selected`}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <ScrollArea className="h-[200px]">
            {allOptions.map((option) => (
              <SelectItem
                key={option.value}
                value={option.value}
                className={cn(
                  'flex items-center justify-between py-2',
                  value.includes(option.value) && 'bg-accent'
                )}
              >
                <div className="flex items-center justify-between w-full">
                  <span>{option.label}</span>
                  {option.gasToken && (
                    <Badge variant="secondary" className="ml-2">
                      Gas Token
                    </Badge>
                  )}
                </div>
              </SelectItem>
            ))}
          </ScrollArea>
        </SelectContent>
      </Select>

      {value.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {value.map((v) => {
            const option = allOptions.find((opt) => opt.value === v);
            if (!option) return null;

            return (
              <Badge key={v} variant="secondary" className="flex items-center gap-1">
                <span>{option.networkName}</span>
                <span className="opacity-50">•</span>
                <span>{option.symbol}</span>
                <button
                  type="button"
                  className="ml-1 ring-offset-background rounded-full outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                  onClick={(e) => {
                    e.stopPropagation();
                    onChange(value.filter((val) => val !== v));
                  }}
                >
                  ×
                </button>
              </Badge>
            );
          })}
        </div>
      )}
    </div>
  );
}
