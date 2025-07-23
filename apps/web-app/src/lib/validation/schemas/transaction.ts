import { z } from 'zod';

/**
 * Transaction type enum
 */
export const transactionTypeSchema = z.enum(['charge', 'refund', 'transfer', 'deposit']);

/**
 * Transaction status enum
 */
export const transactionStatusSchema = z.enum(['succeeded', 'failed', 'pending', 'processing']);

/**
 * Schema for creating a transaction
 */
export const createTransactionSchema = z.object({
  type: transactionTypeSchema,
  amount: z.number()
    .positive('Amount must be positive')
    .max(1000000, 'Amount exceeds maximum limit'),
  customer: z.string().min(1, 'Customer is required'),
  description: z.string()
    .min(1, 'Description is required')
    .max(500, 'Description must be less than 500 characters'),
  paymentMethod: z.string().min(1, 'Payment method is required'),
  // Circle wallet specific fields
  walletId: z.string().uuid().optional(),
  blockchain: z.string().optional(),
  sourceAddress: z.string()
    .regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address format')
    .optional(),
  destinationAddress: z.string()
    .regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address format')
    .optional(),
  tokenSymbol: z.string().optional(),
}).refine((data) => {
  // If type is transfer, source and destination addresses are required
  if (data.type === 'transfer') {
    return data.sourceAddress !== undefined && data.destinationAddress !== undefined;
  }
  return true;
}, {
  message: 'Source and destination addresses are required for transfers',
  path: ['sourceAddress'],
});

/**
 * Schema for transaction query parameters
 */
export const transactionQuerySchema = z.object({
  page: z.coerce.number().int().positive().optional(),
  limit: z.coerce.number().int().positive().max(100).optional(),
  type: transactionTypeSchema.optional(),
  status: transactionStatusSchema.optional(),
  customer: z.string().optional(),
  walletId: z.string().uuid().optional(),
  startDate: z.string().datetime().optional(),
  endDate: z.string().datetime().optional(),
  minAmount: z.coerce.number().positive().optional(),
  maxAmount: z.coerce.number().positive().optional(),
}).refine((data) => {
  // If minAmount and maxAmount are both provided, min should be less than max
  if (data.minAmount !== undefined && data.maxAmount !== undefined) {
    return data.minAmount <= data.maxAmount;
  }
  return true;
}, {
  message: 'Minimum amount must be less than or equal to maximum amount',
  path: ['minAmount'],
});

/**
 * Schema for transaction ID parameter
 */
export const transactionIdParamSchema = z.object({
  transactionId: z.string().uuid('Invalid transaction ID format'),
});

/**
 * Schema for refund request
 */
export const refundTransactionSchema = z.object({
  amount: z.number()
    .positive('Refund amount must be positive')
    .optional(), // If not provided, full refund
  reason: z.string()
    .max(500, 'Reason must be less than 500 characters')
    .optional(),
});

// Type exports
export type CreateTransactionInput = z.infer<typeof createTransactionSchema>;
export type TransactionQuery = z.infer<typeof transactionQuerySchema>;
export type RefundTransactionInput = z.infer<typeof refundTransactionSchema>;
export type TransactionType = z.infer<typeof transactionTypeSchema>;
export type TransactionStatus = z.infer<typeof transactionStatusSchema>;