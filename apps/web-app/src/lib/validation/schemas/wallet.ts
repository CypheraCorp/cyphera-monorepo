import { z } from 'zod';

/**
 * Wallet type enum
 */
export const walletTypeSchema = z.enum(['wallet', 'circle', 'web3auth']);

/**
 * Network type enum
 */
export const networkTypeSchema = z.enum(['evm', 'solana', 'cosmos', 'bitcoin', 'polkadot']);

/**
 * Circle wallet state enum
 */
export const circleWalletStateSchema = z.enum(['LIVE', 'FROZEN', 'PENDING', 'FAILED']);

/**
 * Schema for creating a wallet
 */
export const createWalletSchema = z.object({
  wallet_type: walletTypeSchema,
  wallet_address: z.string()
    .regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address format'),
  network_type: networkTypeSchema,
  nickname: z.string()
    .min(1)
    .max(50, 'Nickname must be less than 50 characters')
    .optional(),
  ens: z.string()
    .regex(/^[a-z0-9-]+\.eth$/, 'Invalid ENS name format')
    .optional(),
  is_primary: z.boolean().optional(),
  verified: z.boolean().optional(),
  metadata: z.record(z.unknown()).optional(),
  // Circle wallet specific fields
  circle_user_id: z.string().uuid().optional(),
  circle_wallet_id: z.string().optional(),
  chain_id: z.number().int().positive().optional(),
  state: circleWalletStateSchema.optional(),
}).refine((data) => {
  // If wallet_type is circle, circle fields are required
  if (data.wallet_type === 'circle') {
    return data.circle_user_id !== undefined && 
           data.circle_wallet_id !== undefined &&
           data.chain_id !== undefined;
  }
  return true;
}, {
  message: 'Circle wallet fields are required for circle_wallet type',
  path: ['circle_user_id'],
});

/**
 * Schema for updating a wallet
 */
export const updateWalletSchema = z.object({
  nickname: z.string()
    .min(1)
    .max(50, 'Nickname must be less than 50 characters')
    .optional(),
  ens: z.string()
    .regex(/^[a-z0-9-]+\.eth$/, 'Invalid ENS name format')
    .optional(),
  is_primary: z.boolean().optional(),
  verified: z.boolean().optional(),
  metadata: z.record(z.unknown()).optional(),
  // Circle wallet specific fields
  state: circleWalletStateSchema.optional(),
});

/**
 * Schema for wallet ID parameter
 */
export const walletIdParamSchema = z.object({
  walletId: z.string().uuid('Invalid wallet ID format'),
});

/**
 * Schema for wallet query parameters
 */
export const walletQuerySchema = z.object({
  page: z.coerce.number().int().positive().optional(),
  limit: z.coerce.number().int().positive().max(100).optional(),
  wallet_type: walletTypeSchema.optional(),
  network_type: networkTypeSchema.optional(),
  is_primary: z.coerce.boolean().optional(),
  verified: z.coerce.boolean().optional(),
  include_circle_data: z.coerce.boolean().optional(),
});

// Type exports
export type CreateWalletInput = z.infer<typeof createWalletSchema>;
export type UpdateWalletInput = z.infer<typeof updateWalletSchema>;
export type WalletQuery = z.infer<typeof walletQuerySchema>;
export type WalletType = z.infer<typeof walletTypeSchema>;
export type NetworkType = z.infer<typeof networkTypeSchema>;