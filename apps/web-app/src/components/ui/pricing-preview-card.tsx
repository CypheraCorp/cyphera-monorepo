'use client';

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { CURRENCY_SYMBOLS } from '@/lib/constants/currency';
import { PRICE_TYPES, INTERVAL_TYPES } from '@/lib/constants/products';
import { Loader2, Eye } from 'lucide-react';

interface PricingPreviewCardProps {
  productName: string;
  productDescription?: string;
  priceInPennies: number;
  currency: string;
  productType: string;
  intervalType?: string;
  termLength?: number;
  selectedNetworkName?: string;
  selectedTokenSymbol?: string;
  totalPaymentOptions?: number;
  className?: string;
}

export function PricingPreviewCard({
  productName,
  productDescription,
  priceInPennies,
  currency,
  productType,
  intervalType,
  termLength,
  selectedNetworkName,
  selectedTokenSymbol,
  totalPaymentOptions = 0,
  className,
}: PricingPreviewCardProps) {
  const currencySymbol = CURRENCY_SYMBOLS[currency] || currency;
  const price = priceInPennies / 100;

  // Format price display based on currency
  const formatPrice = (amount: number) => {
    if (currency === 'EUR') {
      return `${amount.toFixed(2)}${currencySymbol}`;
    }
    return `${currencySymbol}${amount.toFixed(2)}`;
  };

  // Get interval display text
  const getIntervalText = () => {
    if (productType !== PRICE_TYPES.RECURRING) return null;

    switch (intervalType) {
      case INTERVAL_TYPES.MONTHLY:
        return 'month';
      case INTERVAL_TYPES.YEARLY:
        return 'year';
      case INTERVAL_TYPES.WEEKLY:
        return 'week';
      case INTERVAL_TYPES.DAILY:
        return 'day';
      case INTERVAL_TYPES.FIVE_MINUTES:
        return '5 minutes';
      case INTERVAL_TYPES.ONE_MINUTE:
        return 'minute';
      default:
        return 'interval';
    }
  };

  const intervalText = getIntervalText();

  // Show empty state if no data
  if (priceInPennies === 0 && !productName) {
    return (
      <Card className={`opacity-50 shadow-lg ${className}`}>
        <CardContent className="pt-6">
          <div className="text-center text-muted-foreground">
            <Eye className="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm">Customer preview will appear here</p>
            <p className="text-xs mt-1">Fill in product details to see what customers will see</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card
      className={`shadow-lg border-2 border-blue-200 bg-gradient-to-br from-blue-50 to-purple-50 dark:from-blue-950/50 dark:to-purple-950/50 dark:border-blue-800 ${className}`}
    >
      {/* Header - exactly like real product payment card */}
      <CardHeader className="text-center">
        <div className="flex items-center justify-center gap-2 mb-2">
          <Eye className="w-4 h-4 text-blue-600" />
          <span className="text-xs font-medium text-blue-600 uppercase tracking-wider">
            Customer Preview
          </span>
        </div>
        <CardTitle className="text-xl font-bold">
          {productType === PRICE_TYPES.RECURRING ? 'Subscribe' : 'Purchase'}
        </CardTitle>
        <CardDescription>
          {productDescription || productName || 'Get access to this product'}
        </CardDescription>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Pricing Section - matching real product card */}
        <div className="bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-900/20 dark:to-blue-900/20 rounded-lg p-4 border">
          <div className="text-center space-y-3">
            <div className="text-3xl font-bold text-purple-600 mb-1">{formatPrice(price)}</div>
            {intervalText && (
              <div className="text-sm text-muted-foreground">per {intervalText}</div>
            )}

            {/* Total Cost Calculation - exactly like real card */}
            {productType === PRICE_TYPES.RECURRING && termLength && (
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground bg-white/50 dark:bg-gray-900/50 rounded-full px-3 py-1 inline-block">
                  {termLength} payment term
                </div>
                <div className="border-t border-white/30 dark:border-gray-700/30 pt-2">
                  <div className="text-sm text-muted-foreground">Total Cost</div>
                  <div className="text-xl font-bold text-gray-900 dark:text-white">
                    {formatPrice(price * termLength)}
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Payment Method Section */}
        {selectedNetworkName && selectedTokenSymbol ? (
          <div className="text-center space-y-2">
            <h4 className="font-medium text-sm text-muted-foreground">
              Payment Method{totalPaymentOptions > 1 ? 's' : ''}
            </h4>
            <div className="flex items-center justify-center gap-2">
              <Badge variant="outline" className="flex items-center gap-1 font-medium">
                {selectedTokenSymbol}
              </Badge>
              <span className="text-sm text-muted-foreground">on {selectedNetworkName}</span>
              {totalPaymentOptions > 1 && (
                <Badge variant="secondary" className="text-xs">
                  +{totalPaymentOptions - 1} more
                </Badge>
              )}
            </div>
          </div>
        ) : (
          <div className="text-center space-y-2">
            <h4 className="font-medium text-sm text-muted-foreground">Payment Methods</h4>
            <div className="text-sm text-muted-foreground">
              Select cryptocurrencies to see payment options
            </div>
          </div>
        )}

        {/* Token Amount Display - simulated */}
        <div className="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border-2 border-dashed border-gray-200 dark:border-gray-700 min-h-[80px] flex items-center justify-center">
          {selectedNetworkName && selectedTokenSymbol ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span className="text-sm">Calculating crypto amount...</span>
            </div>
          ) : (
            <span className="text-sm text-muted-foreground">Crypto amount will appear here</span>
          )}
        </div>
      </CardContent>

      <CardFooter className="flex justify-center">
        <Button disabled className="w-full py-6 text-base">
          {productType === PRICE_TYPES.RECURRING ? 'Subscribe Now' : 'Buy Now'}
        </Button>
      </CardFooter>
    </Card>
  );
}
