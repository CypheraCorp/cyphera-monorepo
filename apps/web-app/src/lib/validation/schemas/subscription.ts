import { z } from 'zod';

/**
 * Subscription status enum
 */
export const subscriptionStatusSchema = z.enum(['active', 'canceled', 'past_due', 'expired']);

/**
 * Schema for caveat object in delegation
 */
export const caveatSchema = z.object({
  enforcer: z.string(),
  terms: z.string(),
});

/**
 * Schema for delegation object (MetaMask Delegation Toolkit)
 * Matches the standard MetaMask delegation format
 * Reference: https://docs.metamask.io/delegation-toolkit/how-to/create-delegation/
 */
export const delegationSchema = z.object({
  delegate: z.string().regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address'),
  delegator: z.string().regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address'),
  authority: z.string().regex(/^0x[a-fA-F0-9]*$/, 'Invalid hex string'), // Hex-encoded authority
  caveats: z.array(caveatSchema),
  salt: z.string().regex(/^0x[a-fA-F0-9]*$/, 'Invalid hex string'),
  signature: z.string().regex(/^0x[a-fA-F0-9]*$/, 'Invalid hex string'),
}).passthrough(); // Allow additional properties

/**
 * Schema for creating a subscription
 */
export const subscribeRequestSchema = z.object({
  subscriber_address: z.string()
    .regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address format'),
  product_token_id: z.string().uuid('Invalid product token ID format'),
  token_amount: z.string()
    .regex(/^\d+$/, 'Invalid token amount format'), // Only integers, no decimals
  delegation: delegationSchema,
});

/**
 * Schema for subscription query parameters
 */
export const subscriptionQuerySchema = z.object({
  page: z.coerce.number().int().positive().optional(),
  limit: z.coerce.number().int().positive().max(100).optional(),
  status: subscriptionStatusSchema.optional(),
  customer_id: z.string().uuid().optional(),
  product_id: z.string().uuid().optional(),
});

/**
 * Schema for subscription ID parameter
 */
export const subscriptionIdParamSchema = z.object({
  subscriptionId: z.string().uuid('Invalid subscription ID format'),
});

/**
 * Schema for canceling a subscription
 */
export const cancelSubscriptionSchema = z.object({
  reason: z.string().max(500, 'Reason must be less than 500 characters').optional(),
  feedback: z.string().max(1000, 'Feedback must be less than 1000 characters').optional(),
});

// Type exports
export type SubscribeInput = z.infer<typeof subscribeRequestSchema>;
export type SubscriptionQuery = z.infer<typeof subscriptionQuerySchema>;
export type CancelSubscriptionInput = z.infer<typeof cancelSubscriptionSchema>;
export type SubscriptionStatus = z.infer<typeof subscriptionStatusSchema>;