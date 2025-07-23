import { z } from 'zod';
import { createWalletSchema } from './wallet';

/**
 * Account type enum
 */
export const accountTypeSchema = z.enum(['admin', 'merchant']);

/**
 * Schema for creating an account
 */
export const createAccountSchema = z.object({
  name: z.string()
    .min(1, 'Name is required')
    .max(100, 'Name must be less than 100 characters'),
  account_type: accountTypeSchema,
  description: z.string()
    .max(500, 'Description must be less than 500 characters')
    .optional(),
  business_name: z.string()
    .min(1)
    .max(100, 'Business name must be less than 100 characters')
    .optional(),
  business_type: z.string()
    .max(50, 'Business type must be less than 50 characters')
    .optional(),
  website_url: z.string()
    .url('Invalid website URL')
    .optional()
    .or(z.literal('')),
  support_email: z.string()
    .email('Invalid email format')
    .optional(),
  support_phone: z.string()
    .regex(/^\+?[1-9]\d{1,14}$/, 'Invalid phone number format')
    .optional(),
  finished_onboarding: z.boolean().optional(),
  metadata: z.record(z.unknown()).optional(),
  // Web3Auth embedded wallet data
  wallet_data: createWalletSchema.optional(),
});

/**
 * Schema for account onboarding request
 */
export const accountOnboardingSchema = z.object({
  address_line1: z.string()
    .min(1)
    .max(100, 'Address line 1 must be less than 100 characters')
    .optional(),
  address_line2: z.string()
    .max(100, 'Address line 2 must be less than 100 characters')
    .optional(),
  city: z.string()
    .min(1)
    .max(50, 'City must be less than 50 characters')
    .optional(),
  state: z.string()
    .length(2, 'State must be 2 characters')
    .optional(),
  postal_code: z.string()
    .regex(/^\d{5}(-\d{4})?$/, 'Invalid postal code format')
    .optional(),
  country: z.string()
    .length(2, 'Country must be 2-letter ISO code')
    .optional(),
  first_name: z.string()
    .min(1)
    .max(50, 'First name must be less than 50 characters')
    .optional(),
  last_name: z.string()
    .min(1)
    .max(50, 'Last name must be less than 50 characters')
    .optional(),
  wallet_address: z.string()
    .regex(/^0x[a-fA-F0-9]{40}$/, 'Invalid Ethereum address format')
    .optional(),
  finished_onboarding: z.boolean().optional(),
});

/**
 * Schema for updating an account
 */
export const updateAccountSchema = z.object({
  name: z.string()
    .min(1)
    .max(100, 'Name must be less than 100 characters')
    .optional(),
  description: z.string()
    .max(500, 'Description must be less than 500 characters')
    .optional(),
  business_name: z.string()
    .min(1)
    .max(100, 'Business name must be less than 100 characters')
    .optional(),
  business_type: z.string()
    .max(50, 'Business type must be less than 50 characters')
    .optional(),
  website_url: z.string()
    .url('Invalid website URL')
    .optional()
    .or(z.literal('')),
  support_email: z.string()
    .email('Invalid email format')
    .optional(),
  support_phone: z.string()
    .regex(/^\+?[1-9]\d{1,14}$/, 'Invalid phone number format')
    .optional(),
  account_type: accountTypeSchema.optional(),
  finished_onboarding: z.boolean().optional(),
  metadata: z.record(z.unknown()).optional(),
});

/**
 * Schema for account ID parameter
 */
export const accountIdParamSchema = z.object({
  accountId: z.string().uuid('Invalid account ID format'),
});

// Type exports
export type CreateAccountInput = z.infer<typeof createAccountSchema>;
export type AccountOnboardingInput = z.infer<typeof accountOnboardingSchema>;
export type UpdateAccountInput = z.infer<typeof updateAccountSchema>;
export type AccountType = z.infer<typeof accountTypeSchema>;