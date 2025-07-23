# Input Validation with Zod - Implementation Guide

## Overview

Input validation using Zod schemas has been implemented to ensure data integrity and prevent invalid data from entering the system.

## Implementation Status

### ✅ Completed
1. **Validation Schemas Created**:
   - Product schemas (`/lib/validation/schemas/product.ts`)
   - Subscription schemas (`/lib/validation/schemas/subscription.ts`)
   - Customer schemas (`/lib/validation/schemas/customer.ts`)

2. **Validation Utilities**:
   - `validateBody` - Validates request body
   - `validateQuery` - Validates query parameters
   - `validateParams` - Validates route parameters
   - `withValidation` - HOF for wrapping route handlers

## How to Use Input Validation

### 1. Basic Validation in API Routes

```typescript
// app/api/products/route.ts
import { NextRequest, NextResponse } from 'next/server';
import { createProductSchema } from '@/lib/validation/schemas/product';
import { validateBody } from '@/lib/validation/validate';

export async function POST(request: NextRequest) {
  // Validate request body
  const { data, error } = await validateBody(request, createProductSchema);
  if (error) return error;

  // Use validated data
  const product = await createProduct(data);
  return NextResponse.json(product);
}
```

### 2. Using the withValidation HOF

```typescript
// app/api/products/[productId]/route.ts
import { withValidation } from '@/lib/validation/validate';
import { updateProductSchema, productIdParamSchema } from '@/lib/validation/schemas/product';

export const PUT = withValidation(
  {
    bodySchema: updateProductSchema,
    paramsSchema: productIdParamSchema,
  },
  async (request, { body, params }) => {
    // body and params are already validated and typed
    const product = await updateProduct(params.productId, body);
    return NextResponse.json(product);
  }
);
```

### 3. Validating Query Parameters

```typescript
// app/api/customers/route.ts
import { customerQuerySchema } from '@/lib/validation/schemas/customer';
import { validateQuery } from '@/lib/validation/validate';

export async function GET(request: NextRequest) {
  const { data: query, error } = validateQuery(request, customerQuerySchema);
  if (error) return error;

  // Use validated query params
  const customers = await getCustomers({
    page: query.page || 1,
    limit: query.limit || 20,
    email: query.email,
  });
  
  return NextResponse.json(customers);
}
```

## Available Validation Schemas

### Product Schemas
- `createProductSchema` - For creating products
- `updateProductSchema` - For updating products
- `createPriceSchema` - For creating prices
- `productIdParamSchema` - For product ID route params

### Subscription Schemas
- `subscribeRequestSchema` - For creating subscriptions
- `subscriptionQuerySchema` - For subscription queries
- `cancelSubscriptionSchema` - For canceling subscriptions
- `subscriptionIdParamSchema` - For subscription ID route params

### Customer Schemas
- `createCustomerSchema` - For creating customers
- `customerSignInSchema` - For customer sign-in
- `updateCustomerSchema` - For updating customers
- `customerQuerySchema` - For customer queries
- `customerIdParamSchema` - For customer ID route params

## Error Response Format

When validation fails, the API returns a structured error response:

```json
{
  "error": "Validation failed",
  "details": [
    {
      "field": "email",
      "message": "Invalid email format"
    },
    {
      "field": "unit_amount_in_pennies",
      "message": "Amount must be positive"
    }
  ]
}
```

## Best Practices

1. **Use Specific Schemas**: Create specific schemas for each operation rather than reusing generic ones
2. **Custom Error Messages**: Provide clear, user-friendly error messages in schemas
3. **Type Exports**: Export inferred types from schemas for TypeScript usage
4. **Validation at the Edge**: Validate data as early as possible in the request lifecycle
5. **Consistent Patterns**: Use the same validation patterns across all API routes

## Example: Complete API Route with Validation

```typescript
// app/api/products/route.ts
import { NextRequest, NextResponse } from 'next/server';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { createProductSchema } from '@/lib/validation/schemas/product';
import { requireAuth } from '@/lib/auth/guards/require-auth';

export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: createProductSchema },
    async (request, { body }) => {
      try {
        await requireAuth();
        
        const { api, userContext } = await getAPIContextFromSession(request);
        const product = await api.products.createProduct(userContext, body);
        
        return NextResponse.json(product);
      } catch (error) {
        // Error handling
      }
    }
  )
);
```

## Next Steps

1. Apply validation to all existing API routes
2. Create validation schemas for remaining entities (wallets, transactions, etc.)
3. Add client-side validation using the same schemas
4. Set up integration tests for validation