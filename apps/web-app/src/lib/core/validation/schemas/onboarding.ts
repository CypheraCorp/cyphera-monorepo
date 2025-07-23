import * as z from 'zod';
import { isAddress } from 'viem';

export const onboardingFormSchema = z.object({
  first_name: z.string().optional(),
  last_name: z.string().optional(),
  address_line1: z.string().optional(),
  address_line2: z.string().optional(),
  city: z.string().optional(),
  state: z.string().optional(),
  country: z.string().optional(),
  postal_code: z.string().optional(),
  wallet_address: z
    .string()
    .optional()
    .refine(
      (val) => {
        if (!val) return true; // Allow empty value since it's optional
        // First check basic format
        if (!/^0x[a-fA-F0-9]{40}$/.test(val)) return false;
        // Then use viem's isAddress for additional validation
        return isAddress(val);
      },
      {
        message: 'Please enter a valid Ethereum address (0x followed by 40 hexadecimal characters)',
      }
    )
    .transform((val) => (val ? val.toLowerCase() : val)),
});

export type OnboardingFormValues = z.infer<typeof onboardingFormSchema>;
