import { ProductResponse } from '@/types/product';
import { PRICE_TYPES } from '@/lib/constants/products';

export function formatProductInterval(product: ProductResponse): string {
  if (product.prices?.[0]?.type !== PRICE_TYPES.RECURRING) {
    return 'One-time payment';
  }

  const interval = product.prices?.[0]?.interval_type;
  const terms = product.prices?.[0]?.term_length || 0;

  return `${interval} • ${terms} ${terms === 1 ? 'term' : 'terms'}`;
}

export function formatProductType(type: string): string {
  return type === PRICE_TYPES.RECURRING ? 'Recurring' : 'One-time';
}
