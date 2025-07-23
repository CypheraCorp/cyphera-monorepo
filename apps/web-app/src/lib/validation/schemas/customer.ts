import { z } from 'zod';

/**
 * Network type enum
 */
export const networkTypeSchema = z.enum(['evm', 'solana', 'cosmos', 'bitcoin', 'polkadot']);

/**
 * Schema for customer wallet data
 */
export const customerWalletDataSchema = z.object({
  wallet_address: z.string()
    .regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address format'),
  network_type: networkTypeSchema,
  nickname: z.string().max(50).optional(),
  ens: z.string().optional(),
  is_primary: z.boolean().optional(),
  verified: z.boolean().optional(),
  metadata: z.record(z.unknown()).optional(),
});

/**
 * Schema for creating a customer
 */
export const createCustomerSchema = z.object({
  external_id: z.string().optional(),
  email: z.string().email('Invalid email format'),
  name: z.string().min(1).max(100).optional(),
  phone: z.string()
    .regex(/^\+?[1-9]\d{1,14}$/, 'Invalid phone number format')
    .optional(),
  description: z.string().max(500).optional(),
  balance_in_pennies: z.number().int().min(0).optional(),
  currency: z.string().length(3).optional(),
  default_source_id: z.string().uuid().optional(),
  invoice_prefix: z.string().max(20).optional(),
  next_invoice_sequence: z.number().int().positive().optional(),
  tax_exempt: z.boolean().optional(),
  tax_ids: z.record(z.unknown()).optional(),
  metadata: z.record(z.unknown()).optional(),
  livemode: z.boolean().optional(),
  finished_onboarding: z.boolean().optional(),
}).refine((data) => {
  // If balance_in_pennies is set, currency is required
  if (data.balance_in_pennies !== undefined && !data.currency) {
    return false;
  }
  return true;
}, {
  message: 'Currency is required when balance_in_pennies is set',
  path: ['currency'],
});

/**
 * Schema for customer sign-in request
 */
export const customerSignInSchema = z.object({
  email: z.string().email('Invalid email format'),
  name: z.string().min(1).max(100).optional(),
  phone: z.string()
    .regex(/^\+?[1-9]\d{1,14}$/, 'Invalid phone number format')
    .optional(),
  finished_onboarding: z.boolean().optional(),
  metadata: z.object({
    web3auth_id: z.string().min(1, 'Web3Auth ID is required'),
    verifier: z.string().optional(),
    verifier_id: z.string().optional(),
  }).passthrough(),
  wallet_data: customerWalletDataSchema.optional(),
});

/**
 * Schema for updating a customer
 */
export const updateCustomerSchema = z.object({
  external_id: z.string().optional(),
  email: z.string().email('Invalid email format').optional(),
  name: z.string().min(1).max(100).optional(),
  phone: z.string()
    .regex(/^\+?[1-9]\d{1,14}$/, 'Invalid phone number format')
    .optional(),
  description: z.string().max(500).optional(),
  balance_in_pennies: z.number().int().min(0).optional(),
  currency: z.string().length(3).optional(),
  default_source_id: z.string().uuid().optional(),
  invoice_prefix: z.string().max(20).optional(),
  next_invoice_sequence: z.number().int().positive().optional(),
  tax_exempt: z.boolean().optional(),
  tax_ids: z.record(z.unknown()).optional(),
  metadata: z.record(z.unknown()).optional(),
  livemode: z.boolean().optional(),
  finished_onboarding: z.boolean().optional(),
});

/**
 * Schema for updating customer onboarding status
 */
export const updateCustomerOnboardingSchema = z.object({
  finished_onboarding: z.boolean(),
});

/**
 * Schema for customer ID parameter
 */
export const customerIdParamSchema = z.object({
  customerId: z.string().uuid('Invalid customer ID format'),
});

/**
 * Schema for customer query parameters
 */
export const customerQuerySchema = z.object({
  page: z.coerce.number().int().positive().optional(),
  limit: z.coerce.number().int().positive().max(100).optional(),
  email: z.string().email().optional(),
  name: z.string().optional(),
  finished_onboarding: z.boolean().optional(),
});

// Type exports
export type CreateCustomerInput = z.infer<typeof createCustomerSchema>;
export type CustomerSignInInput = z.infer<typeof customerSignInSchema>;
export type UpdateCustomerInput = z.infer<typeof updateCustomerSchema>;
export type UpdateCustomerOnboardingInput = z.infer<typeof updateCustomerOnboardingSchema>;
export type CustomerQuery = z.infer<typeof customerQuerySchema>;