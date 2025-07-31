/**
 * Formats a price value to display with 2 decimal places (cents)
 * @param price - The price value to format
 * @param currency - Optional currency symbol/code to append
 * @returns Formatted price string
 */
export function formatPrice(price: number | undefined, currency?: string): string {
  if (typeof price !== 'number') return '';
  return `${price.toFixed(2)}${currency ? ` ${currency}` : ''}`.trim();
}

/**
 * Formats a monetary value from cents to display format
 * @param cents - The amount in cents
 * @param currencyCode - The currency code (e.g., 'USD')
 * @returns Formatted money string
 */
export function formatMoney(cents: number, currencyCode: string = 'USD'): string {
  const amount = cents / 100;
  
  // Use Intl.NumberFormat for proper currency formatting
  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currencyCode,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(amount);
  } catch {
    // Fallback if currency code is not recognized
    return `${amount.toFixed(2)} ${currencyCode}`;
  }
}

/**
 * Formats a percentage value
 * @param value - The percentage value (0-100)
 * @param decimals - Number of decimal places (default: 1)
 * @returns Formatted percentage string
 */
export function formatPercentage(value: number, decimals: number = 1): string {
  return `${value.toFixed(decimals)}%`;
}

/**
 * Formats a large number with K/M/B suffixes
 * @param value - The number to format
 * @returns Formatted number string
 */
export function formatCompactNumber(value: number): string {
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(1)}B`;
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(1)}M`;
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(1)}K`;
  }
  return value.toString();
}
