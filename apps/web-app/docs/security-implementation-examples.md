# Security Implementation Examples

## Complete Examples of Secured API Routes

### 1. Product Creation Route with Full Security

```typescript
// app/api/products/route.ts
import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import { logger } from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { createProductSchema } from '@/lib/validation/schemas/product';

/**
 * POST /api/products
 * Creates a new product with CSRF protection and input validation
 */
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
        if (error instanceof Error && error.message === 'Unauthorized') {
          return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
        }
        logger.error('Error creating product', { error });
        const message = error instanceof Error ? error.message : 'Failed to create product';
        return NextResponse.json({ error: message }, { status: 500 });
      }
    }
  )
);
```

### 2. Customer Update Route with Validation

```typescript
// app/api/customers/[customerId]/route.ts
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { updateCustomerSchema, customerIdParamSchema } from '@/lib/validation/schemas/customer';

export const PUT = withCSRFProtection(
  withValidation(
    {
      bodySchema: updateCustomerSchema,
      paramsSchema: customerIdParamSchema,
    },
    async (request, { body, params }) => {
      try {
        await requireAuth();
        
        const { api, userContext } = await getAPIContextFromSession(request);
        const customer = await api.customers.updateCustomer(
          userContext, 
          params.customerId, 
          body
        );
        
        return NextResponse.json(customer);
      } catch (error) {
        // Error handling...
      }
    }
  )
);
```

### 3. Subscription Creation with Complex Validation

```typescript
// app/api/subscriptions/route.ts
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { subscribeRequestSchema } from '@/lib/validation/schemas/subscription';

export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: subscribeRequestSchema },
    async (request, { body }) => {
      try {
        await requireAuth();
        
        // Validated body includes:
        // - subscriber_address (validated Ethereum address)
        // - price_id (validated UUID)
        // - product_token_id (validated UUID)
        // - token_amount (validated numeric string)
        // - delegation (validated MetaMask delegation object)
        
        const { api, userContext } = await getAPIContextFromSession(request);
        const subscription = await api.subscriptions.createSubscription(userContext, body);
        
        return NextResponse.json(subscription);
      } catch (error) {
        // Error handling...
      }
    }
  )
);
```

## Client-Side Usage Examples

### 1. Using CSRF in React Components

```typescript
// components/product-form.tsx
import { useCSRF } from '@/hooks/security/use-csrf';
import { createProductSchema } from '@/lib/validation/schemas/product';
import type { CreateProductInput } from '@/lib/validation/schemas/product';

export function ProductForm() {
  const { csrfToken, getHeaders } = useCSRF();
  
  const handleSubmit = async (data: CreateProductInput) => {
    // Client-side validation (optional, as server will validate)
    const validation = createProductSchema.safeParse(data);
    if (!validation.success) {
      // Handle validation errors
      return;
    }
    
    const response = await fetch('/api/products', {
      method: 'POST',
      headers: getHeaders({
        'Content-Type': 'application/json',
      }),
      body: JSON.stringify(validation.data),
    });
    
    if (!response.ok) {
      const error = await response.json();
      // Handle error with detailed validation messages
      console.error('Validation failed:', error.details);
    }
  };
}
```

### 2. Using the API Hook with CSRF

```typescript
// components/product-manager.tsx
import { useAPIWithCSRF } from '@/hooks/api/use-api-with-csrf';
import { useAuthStore } from '@/store/auth';

export function ProductManager() {
  const { products } = useAPIWithCSRF();
  const { session } = useAuthStore();
  
  const createProduct = async (productData: CreateProductInput) => {
    if (!session) return;
    
    const userContext = {
      access_token: session.access_token,
      workspace_id: session.workspace_id,
      // CSRF token is automatically included by the hook
    };
    
    try {
      const product = await products.createProduct(userContext, productData);
      // Handle success
    } catch (error) {
      // Handle error
    }
  };
}
```

## Security Checklist

- [ ] All POST/PUT/DELETE routes wrapped with `withCSRFProtection`
- [ ] All routes accepting data have Zod validation schemas
- [ ] CSRFProvider added to root layout
- [ ] CSRF_SECRET environment variable configured
- [ ] Client-side forms use CSRF tokens
- [ ] Error responses don't leak sensitive information
- [ ] Validation errors provide clear user feedback

## Common Patterns

### Combining Multiple Validations

```typescript
export const POST = withCSRFProtection(
  withValidation(
    {
      bodySchema: createOrderSchema,
      querySchema: orderOptionsSchema,
    },
    async (request, { body, query }) => {
      // Both body and query are validated and typed
    }
  )
);
```

### Custom Validation Logic

```typescript
const customSchema = z.object({
  amount: z.number().positive(),
  recipient: z.string(),
}).refine((data) => {
  // Custom validation logic
  return data.amount <= 1000000; // Max amount
}, {
  message: "Amount exceeds maximum allowed",
  path: ["amount"],
});
```

### Handling Validation Errors

```typescript
try {
  const response = await fetch('/api/endpoint', {
    method: 'POST',
    headers: getHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify(data),
  });
  
  if (!response.ok) {
    const error = await response.json();
    if (error.details) {
      // Show field-specific errors
      error.details.forEach(({ field, message }) => {
        console.error(`${field}: ${message}`);
      });
    }
  }
} catch (error) {
  // Network or other errors
}
```