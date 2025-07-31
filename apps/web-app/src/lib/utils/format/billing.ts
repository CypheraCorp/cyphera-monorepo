/**
 * Format billing interval for display
 * @param intervalType - The interval type (day, week, month, year, etc.)
 * @param intervalCount - The interval count (1, 2, 3, etc.)
 * @returns Formatted billing interval string
 */
export function formatBillingInterval(intervalType?: string, intervalCount?: number): string {
  if (!intervalType) {
    return 'One-time';
  }

  const count = intervalCount || 1;
  
  // Handle special interval types that may come from the backend
  const normalizedType = normalizeIntervalType(intervalType);
  
  if (count === 1) {
    switch (normalizedType) {
      case 'day':
        return 'Daily';
      case 'week':
        return 'Weekly';
      case 'month':
        return 'Monthly';
      case 'year':
        return 'Yearly';
      default:
        return `Every ${normalizedType}`;
    }
  }

  // Handle plural cases
  switch (normalizedType) {
    case 'day':
      return `Every ${count} days`;
    case 'week':
      return `Every ${count} weeks`;
    case 'month':
      return `Every ${count} months`;
    case 'year':
      return `Every ${count} years`;
    default:
      return `Every ${count} ${normalizedType}s`;
  }
}

/**
 * Normalize interval type from various backend formats
 * @param intervalType - Raw interval type from backend
 * @returns Normalized interval type
 */
function normalizeIntervalType(intervalType: string): string {
  const normalized = intervalType.toLowerCase();
  
  // Handle specific backend formats
  switch (normalized) {
    case '1min':
      return 'minute';
    case '5mins':
      return '5 minutes';
    case 'daily':
      return 'day';
    case 'week':
    case 'weekly':
      return 'week';
    case 'month':
    case 'monthly':
      return 'month';
    case 'year':
    case 'yearly':
    case 'annual':
      return 'year';
    default:
      return normalized;
  }
}

/**
 * Get short format for billing interval (for compact displays)
 * @param intervalType - The interval type
 * @param intervalCount - The interval count
 * @returns Short formatted billing interval
 */
export function formatBillingIntervalShort(intervalType?: string, intervalCount?: number): string {
  if (!intervalType) {
    return 'One-time';
  }

  const count = intervalCount || 1;
  const normalizedType = normalizeIntervalType(intervalType);
  
  if (count === 1) {
    switch (normalizedType) {
      case 'day':
        return '/day';
      case 'week':
        return '/week';
      case 'month':
        return '/mo';
      case 'year':
        return '/year';
      default:
        return `/${normalizedType}`;
    }
  }

  switch (normalizedType) {
    case 'day':
      return `/${count}d`;
    case 'week':
      return `/${count}w`;
    case 'month':
      return `/${count}mo`;
    case 'year':
      return `/${count}y`;
    default:
      return `/${count}${normalizedType.charAt(0)}`;
  }
}