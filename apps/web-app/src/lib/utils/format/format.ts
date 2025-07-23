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
