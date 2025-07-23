'use client';

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useState, useEffect } from 'react';
import { Badge } from '@/components/ui/badge';
import { PublicProductResponse } from '@/types/product';
import { Web3AuthDelegationButton } from '@/components/public/web3auth-delegation-button';
import { parseUnits, formatUnits } from 'viem';
import { Loader2, AlertTriangle } from 'lucide-react';
import type { TokenQuoteResponse, TokenQuotePayload } from '@/types/token';
import { logger } from '@/lib/core/logger/logger-utils';
interface ProductPaymentCardProps {
  product: PublicProductResponse;
  isAuthenticated?: boolean;
}

export function ProductPaymentCard({ product, isAuthenticated = false }: ProductPaymentCardProps) {
  const [isMounted, setIsMounted] = useState(false);
  const [amountInSelectedToken, setAmountInSelectedToken] = useState<string | null>(null);
  const [amountBigInt, setAmountBigInt] = useState<bigint | null>(null);
  const [isFetchingPrice, setIsFetchingPrice] = useState(false);
  const [priceError, setPriceError] = useState<string | null>(null);

  // Client-side mounting check to prevent hydration mismatches
  useEffect(() => {
    setIsMounted(true);
  }, []);

  // Get primary payment option for display
  const primaryOption = product.product_tokens?.[0];

  // Debug logging
  logger.log('🔍 [ProductPaymentCard] Component state:', {
    productId: product.id,
    productName: product.name,
    productTokens: product.product_tokens,
    primaryOption,
    networkName: primaryOption?.network_name,
    tokenSymbol: primaryOption?.token_symbol,
    isAuthenticated,
  });

  // Add useEffect for fetching token price and calculating amount
  useEffect(() => {
    if (!primaryOption || !primaryOption.token_decimals) {
      setAmountInSelectedToken(null);
      setAmountBigInt(null);
      if (primaryOption && !primaryOption.token_decimals) {
        setPriceError('Token configuration error (missing decimals).');
      }
      return;
    }

    const tokenDecimals = primaryOption.token_decimals;

    const fetchPrice = async () => {
      setIsFetchingPrice(true);
      setPriceError(null);
      setAmountInSelectedToken(null);
      setAmountBigInt(null);

      try {
        const payload: TokenQuotePayload = {
          fiat_symbol: product.price.currency,
          token_symbol: primaryOption.token_symbol,
        };

        const response = await fetch('/api/tokens/quote', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(payload),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to fetch token price (Status: ${response.status})`
          );
        }

        const priceData: TokenQuoteResponse = await response.json();
        const tokenAmountInFiat = priceData.token_amount_in_fiat;

        if (typeof tokenAmountInFiat !== 'number' || tokenAmountInFiat <= 0) {
          throw new Error('Invalid or missing price data received from API.');
        }

        try {
          // Convert product price from pennies to dollars
          const productPriceInFiat = product.price.unit_amount_in_pennies / 100;
          const productPriceStr = productPriceInFiat.toFixed(tokenDecimals);
          const scaledProductPrice = parseUnits(productPriceStr, tokenDecimals);

          const tokenAmountStr = tokenAmountInFiat.toFixed(tokenDecimals);
          const scaledTokenAmount = parseUnits(tokenAmountStr, tokenDecimals);

          if (scaledTokenAmount === BigInt(0)) {
            throw new Error('Token price is zero, cannot calculate amount.');
          }

          // Calculate how many tokens needed: productPrice / tokenPrice
          const multiplier = BigInt(10) ** BigInt(tokenDecimals);
          const numerator = scaledProductPrice * multiplier;
          const halfDivisor = scaledTokenAmount / BigInt(2);
          const adjustedNumerator = numerator + halfDivisor;
          const calculatedBigInt = adjustedNumerator / scaledTokenAmount;

          setAmountBigInt(calculatedBigInt);

          const displayDecimals = Math.min(tokenDecimals, 6);
          const fullFormattedAmount = formatUnits(calculatedBigInt, tokenDecimals);
          const roundedDisplayAmount = parseFloat(fullFormattedAmount).toFixed(displayDecimals);

          setAmountInSelectedToken(roundedDisplayAmount);
        } catch (e) {
          logger.error('Error during BigInt amount calculation:', e);
          throw new Error('Failed to calculate token amount accurately.');
        }
      } catch (error) {
        logger.error('Price fetch error:', error);
        setPriceError(error instanceof Error ? error.message : 'Could not load token price');
        setAmountInSelectedToken(null);
        setAmountBigInt(null);
      } finally {
        setIsFetchingPrice(false);
      }
    };

    fetchPrice();
  }, [primaryOption, product.price.unit_amount_in_pennies, product.price.currency]);

  // Get currency code from product price
  const getCurrency = () => {
    return product.price?.currency?.toUpperCase() || 'USD';
  };

  // Format pricing information with proper currency
  const formatPrice = () => {
    if (
      !product.price ||
      product.price.unit_amount_in_pennies === null ||
      product.price.unit_amount_in_pennies === undefined
    ) {
      return 'Price not available';
    }

    const amount = Number(product.price.unit_amount_in_pennies) / 100; // Convert cents to dollars
    const currency = getCurrency();

    // Use appropriate currency symbol or currency code
    if (currency === 'USD') {
      return `$${amount.toFixed(2)}`;
    } else {
      return `${amount.toFixed(2)} ${currency}`;
    }
  };

  // Don't render until component is mounted (prevents hydration issues)
  if (!isMounted) {
    return (
      <Card className="w-full max-w-md">
        <CardContent className="p-6">
          <div className="animate-pulse">
            <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
            <div className="h-4 bg-gray-200 rounded w-1/2"></div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="w-full max-w-md shadow-lg">
      <CardHeader className="text-center">
        <CardTitle className="text-xl font-bold">Subscribe</CardTitle>
        <CardDescription>{product.description || 'Get access to this product'}</CardDescription>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Subscription Plan */}
        <div className="bg-gradient-to-r from-purple-50 to-blue-50 dark:from-purple-900/20 dark:to-blue-900/20 rounded-lg p-4 border">
          <div className="text-center space-y-3">
            <div className="text-3xl font-bold text-purple-600 mb-1">{formatPrice()}</div>
            <div className="text-sm text-muted-foreground">
              per {product.price?.interval_type || 'subscription'}
            </div>

            {/* Total Cost Calculation */}
            {product.price?.term_length && (
              <div className="space-y-2">
                <div className="text-xs text-muted-foreground bg-white/50 dark:bg-gray-900/50 rounded-full px-3 py-1 inline-block">
                  {product.price.term_length} payment term
                </div>
                <div className="border-t border-white/30 dark:border-gray-700/30 pt-2">
                  <div className="text-sm text-muted-foreground">Total Cost</div>
                  <div className="text-xl font-bold text-gray-900 dark:text-white">
                    {(() => {
                      const unitPrice = Number(product.price.unit_amount_in_pennies) / 100;
                      const termLength = Number(product.price.term_length);
                      const total = unitPrice * termLength;
                      const currency = getCurrency();
                      return currency === 'USD'
                        ? `$${total.toFixed(2)}`
                        : `${total.toFixed(2)} ${currency}`;
                    })()}
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Payment Method */}
        {primaryOption && (
          <div className="text-center space-y-2">
            <h4 className="font-medium text-sm text-muted-foreground">Payment Method</h4>
            <div className="flex items-center justify-center gap-2">
              <Badge variant="outline" className="flex items-center gap-1 font-medium">
                {primaryOption.token_symbol}
              </Badge>
              <span className="text-sm text-muted-foreground">on {primaryOption.network_name}</span>
            </div>
          </div>
        )}

        {/* Token Amount Display */}
        <div className="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border-2 border-dashed border-gray-200 dark:border-gray-700 min-h-[80px] flex items-center justify-center">
          {isFetchingPrice ? (
            <div className="flex items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-purple-600" />
            </div>
          ) : priceError ? (
            <div className="flex items-center gap-2 text-red-600">
              <AlertTriangle className="h-4 w-4" />
              <span className="text-sm">Error: {priceError}</span>
            </div>
          ) : amountInSelectedToken && primaryOption ? (
            <div className="text-center space-y-1">
              <div className="text-xl font-bold text-gray-900 dark:text-white">
                {amountInSelectedToken} {primaryOption.token_symbol}
              </div>
              <div className="text-xs text-muted-foreground">≈ {formatPrice()}</div>
            </div>
          ) : (
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-purple-600 mx-auto"></div>
          )}
        </div>

        {/* Subscription Action */}
        <div className="space-y-4">
          {isAuthenticated ? (
            // Authenticated user - show subscription button
            <div className="space-y-3">
              {primaryOption && product.price && amountBigInt && (
                <>
                  {(() => {
                    logger.log('🔍 [ProductPaymentCard] Web3AuthDelegationButton props:', {
                      priceId: product.price.id,
                      productTokenId: primaryOption.product_token_id,
                      tokenAmount: amountBigInt,
                      productName: product.name,
                      networkName: primaryOption.network_name,
                      primaryOption,
                      hasPrice: !!product.price,
                      hasAmountBigInt: !!amountBigInt,
                    });
                    return null;
                  })()}
                  <Web3AuthDelegationButton
                    key={`delegation-button-${isAuthenticated}`}
                    priceId={product.price.id}
                    productTokenId={primaryOption.product_token_id}
                    tokenAmount={amountBigInt}
                    disabled={isFetchingPrice || !!priceError || !amountBigInt}
                    productName={product.name}
                    productDescription={product.description}
                    networkName={primaryOption.network_name}
                    priceDisplay={formatPrice()}
                    intervalType={product.price.interval_type}
                    termLength={product.price.term_length}
                    tokenDecimals={primaryOption.token_decimals}
                  />
                </>
              )}
            </div>
          ) : (
            // Not authenticated - show sign in message
            <div className="text-center p-4 border rounded-md bg-orange-50 dark:bg-orange-900/20">
              <p className="text-sm text-orange-700 dark:text-orange-300 font-medium">
                🔐 Sign In Required
              </p>
              <p className="text-xs text-orange-600 dark:text-orange-400 mt-2">
                Please sign in using the Login or Sign Up buttons in the header to subscribe to this
                product.
              </p>
            </div>
          )}
        </div>
      </CardContent>

      <CardFooter className="flex justify-center">
        {!isAuthenticated && (
          <Button disabled className="w-full py-6 text-base">
            Sign In to Subscribe
          </Button>
        )}
      </CardFooter>
    </Card>
  );
}
