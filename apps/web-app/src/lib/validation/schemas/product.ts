import { z } from 'zod';

/**
 * Schema for creating a product token without product_id
 * Used when creating tokens as part of a new product
 */
export const createProductTokenWithoutIdSchema = z.object({
  network_id: z.string().uuid('Invalid network ID format'),
  token_id: z.string().uuid('Invalid token ID format'),
  active: z.boolean().optional().default(true),
});

/**
 * Schema for creating a product token with product_id
 * Used when adding tokens to an existing product
 */
export const createProductTokenSchema = z.object({
  product_id: z.string().uuid('Invalid product ID format'),
  network_id: z.string().uuid('Invalid network ID format'),
  token_id: z.string().uuid('Invalid token ID format'),
  active: z.boolean().optional().default(true),
});

/**
 * Schema for updating a product token
 */
export const updateProductTokenSchema = z.object({
  active: z.boolean(),
});

/**
 * Schema for product tokens in update product request
 * product_id is required when updating a product's tokens
 */
export const updateProductTokenRequestSchema = z.object({
  product_id: z.string().uuid('Invalid product ID format'),
  network_id: z.string().uuid('Invalid network ID format'),
  token_id: z.string().uuid('Invalid token ID format'),
  active: z.boolean().optional().default(true),
});

/**
 * Price type enum
 */
export const priceTypeSchema = z.enum(['one_off', 'recurring']);

/**
 * Interval type enum
 */
export const intervalTypeSchema = z.enum(['1min', '5mins', 'daily', 'week', 'month', 'year']);

/**
 * Schema for creating a price
 */
export const createPriceSchema = z.object({
  active: z.boolean(),
  type: priceTypeSchema,
  nickname: z.string().optional(),
  currency: z.string().length(3, 'Currency must be 3 characters (e.g., USD)'),
  unit_amount_in_pennies: z.number().int().positive('Amount must be positive'),
  interval_type: intervalTypeSchema.optional(),
  interval_count: z.number().int().positive().optional(),
  term_length: z.number().int().positive().optional(),
  metadata: z.record(z.unknown()).nullable().optional(),
}).refine((data) => {
  // If type is recurring, interval_type is required
  if (data.type === 'recurring') {
    return data.interval_type !== undefined;
  }
  return true;
}, {
  message: 'Interval type is required for recurring prices',
  path: ['interval_type'],
});

/**
 * Schema for creating a product
 */
export const createProductSchema = z.object({
  name: z.string()
    .min(1, 'Product name is required')
    .max(100, 'Product name must be less than 100 characters'),
  wallet_id: z.string().uuid('Invalid wallet ID format'),
  description: z.string()
    .max(500, 'Description must be less than 500 characters')
    .optional(),
  image_url: z.string().url('Invalid image URL').optional().or(z.literal('')),
  url: z.string().url('Invalid URL').optional().or(z.literal('')),
  active: z.boolean(),
  metadata: z.record(z.unknown()).nullable().optional(),
  prices: z.array(createPriceSchema)
    .min(1, 'At least one price is required'),
  product_tokens: z.array(createProductTokenWithoutIdSchema).optional(),
});

/**
 * Schema for updating a product
 */
export const updateProductSchema = z.object({
  name: z.string()
    .min(1, 'Product name cannot be empty')
    .max(100, 'Product name must be less than 100 characters')
    .optional(),
  wallet_id: z.string().uuid('Invalid wallet ID format').optional(),
  description: z.string()
    .max(500, 'Description must be less than 500 characters')
    .optional(),
  image_url: z.string().url('Invalid image URL').optional().or(z.literal('')),
  url: z.string().url('Invalid URL').optional().or(z.literal('')),
  active: z.boolean().optional(),
  metadata: z.record(z.unknown()).nullable().optional(),
  product_tokens: z.array(updateProductTokenRequestSchema).optional(),
});

/**
 * Schema for product ID parameter
 */
export const productIdParamSchema = z.object({
  productId: z.string().uuid('Invalid product ID format'),
});

/**
 * Schema for price ID parameter
 */
export const priceIdParamSchema = z.object({
  priceId: z.string().uuid('Invalid price ID format'),
});

// Type exports
export type CreateProductTokenWithoutIdInput = z.infer<typeof createProductTokenWithoutIdSchema>;
export type CreateProductTokenInput = z.infer<typeof createProductTokenSchema>;
export type UpdateProductTokenInput = z.infer<typeof updateProductTokenSchema>;
export type CreatePriceInput = z.infer<typeof createPriceSchema>;
export type CreateProductInput = z.infer<typeof createProductSchema>;
export type UpdateProductInput = z.infer<typeof updateProductSchema>;