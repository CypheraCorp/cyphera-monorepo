export const PRICE_TYPES = {
  ONE_TIME: 'one_time',
  RECURRING: 'recurring',
} as const;

export const INTERVAL_TYPES = {
  ONE_MINUTE: '1min',
  FIVE_MINUTES: '5mins',
  DAILY: 'daily',
  WEEKLY: 'week',
  MONTHLY: 'month',
  YEARLY: 'year',
} as const;

export type PriceType = (typeof PRICE_TYPES)[keyof typeof PRICE_TYPES];
export type IntervalType = (typeof INTERVAL_TYPES)[keyof typeof INTERVAL_TYPES];

// Add a helper function to validate interval types
export function isValidIntervalType(interval: string): interval is IntervalType {
  return Object.values(INTERVAL_TYPES).includes(interval as IntervalType);
}
