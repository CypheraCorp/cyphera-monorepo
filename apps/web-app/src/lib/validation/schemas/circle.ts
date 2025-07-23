import { z } from 'zod';

/**
 * Circle transaction fee level enum
 */
export const circleFeeLevelSchema = z.enum(['LOW', 'MEDIUM', 'HIGH']);

/**
 * Circle blockchain enum
 */
export const circleBlockchainSchema = z.enum([
  'ETH',
  'MATIC',
  'ARB',
  'BASE',
  'AVAX',
  'SOL',
]);

/**
 * Circle account type enum
 */
export const circleAccountTypeSchema = z.enum(['EOA', 'SCA']);

/**
 * Circle wallet state enum
 */
export const circleWalletStateSchema = z.enum(['LIVE', 'FROZEN']);

/**
 * Schema for creating a Circle user
 */
export const createCircleUserSchema = z.object({
  external_user_id: z.string()
    .min(1, 'External user ID is required')
    .max(100, 'External user ID must be less than 100 characters'),
});

/**
 * Schema for initializing a Circle user
 */
export const initializeCircleUserSchema = z.object({
  idempotency_key: z.string()
    .uuid('Idempotency key must be a valid UUID'),
  account_type: circleAccountTypeSchema.optional(),
  blockchains: z.array(circleBlockchainSchema)
    .min(1, 'At least one blockchain is required'),
  metadata: z.array(z.object({
    name: z.string().max(50),
    ref_id: z.string().max(100),
  })).optional(),
});

/**
 * Schema for creating Circle wallets
 */
export const createCircleWalletsSchema = z.object({
  idempotency_key: z.string()
    .uuid('Idempotency key must be a valid UUID'),
  blockchains: z.array(circleBlockchainSchema)
    .min(1, 'At least one blockchain is required'),
  account_type: circleAccountTypeSchema,
  user_token: z.string().min(1, 'User token is required'),
  metadata: z.array(z.object({
    name: z.string().max(50),
    ref_id: z.string().max(100),
  })).optional(),
});

/**
 * Schema for creating a Circle transaction
 */
export const createCircleTransactionSchema = z.object({
  idempotency_key: z.string()
    .uuid('Idempotency key must be a valid UUID'),
  amounts: z.array(z.string()
    .regex(/^\d+(\.\d+)?$/, 'Invalid amount format'))
    .min(1, 'At least one amount is required'),
  destination_address: z.string()
    .min(1, 'Destination address is required'),
  token_id: z.string().optional(),
  wallet_id: z.string()
    .min(1, 'Wallet ID is required'),
  fee_level: circleFeeLevelSchema,
  ref_id: z.string().max(100).optional(),
});

/**
 * Schema for Circle PIN creation
 */
export const createCirclePinSchema = z.object({
  idempotency_key: z.string()
    .uuid('Idempotency key must be a valid UUID'),
  user_token: z.string().min(1, 'User token is required'),
});

/**
 * Schema for Circle user ID parameter
 */
export const circleUserIdParamSchema = z.object({
  circleUserId: z.string().min(1, 'Circle user ID is required'),
});

/**
 * Schema for Circle wallet query parameters
 */
export const circleWalletQuerySchema = z.object({
  blockchain: circleBlockchainSchema.optional(),
  state: circleWalletStateSchema.optional(),
  pageAfter: z.string().optional(),
  pageBefore: z.string().optional(),
  pageSize: z.coerce.number().int().positive().max(50).optional(),
});

/**
 * Schema for Circle transaction query parameters
 */
export const circleTransactionQuerySchema = z.object({
  blockchain: circleBlockchainSchema.optional(),
  walletId: z.string().optional(),
  sourceAddress: z.string().optional(),
  destinationAddress: z.string().optional(),
  state: z.string().optional(),
  pageAfter: z.string().optional(),
  pageBefore: z.string().optional(),
  pageSize: z.coerce.number().int().positive().max(50).optional(),
});

// Type exports
export type CreateCircleUserInput = z.infer<typeof createCircleUserSchema>;
export type InitializeCircleUserInput = z.infer<typeof initializeCircleUserSchema>;
export type CreateCircleWalletsInput = z.infer<typeof createCircleWalletsSchema>;
export type CreateCircleTransactionInput = z.infer<typeof createCircleTransactionSchema>;
export type CreateCirclePinInput = z.infer<typeof createCirclePinSchema>;
export type CircleWalletQuery = z.infer<typeof circleWalletQuerySchema>;
export type CircleTransactionQuery = z.infer<typeof circleTransactionQuerySchema>;
export type CircleFeeLevel = z.infer<typeof circleFeeLevelSchema>;
export type CircleBlockchain = z.infer<typeof circleBlockchainSchema>;