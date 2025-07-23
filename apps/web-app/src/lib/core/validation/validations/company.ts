import { z } from 'zod';

const BUSINESS_TYPES = [
  'llc',
  'corporation',
  'sole_proprietorship',
  'partnership',
  'non_profit',
] as const;

export const companyFormSchema = z.object({
  business_name: z.string().min(2, 'Business name must be at least 2 characters').optional(),
  business_type: z
    .enum(BUSINESS_TYPES, {
      invalid_type_error: 'Please select a valid business type',
    })
    .optional(),
});

export type CompanyFormData = z.infer<typeof companyFormSchema>;
export type BusinessType = z.infer<typeof companyFormSchema>['business_type'];
