export const SUPPORTED_CURRENCIES = ['USD', 'EUR'] as const;
export const DEFAULT_CURRENCY = 'USD';

export const CURRENCY_SYMBOLS: { [key: string]: string } = {
  USD: '$',
  EUR: '€',
} as const;
