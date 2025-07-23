# CSRF Protection Implementation Guide

## Overview

CSRF (Cross-Site Request Forgery) protection has been implemented to prevent unauthorized actions on behalf of authenticated users.

## Implementation Status

### ✅ Completed
1. **CSRF Token Generation** - API endpoint at `/api/auth/csrf`
2. **CSRF Middleware** - Protection wrapper for API routes
3. **Client-side Hook** - `useCSRF` hook for React components
4. **Provider Component** - `CSRFProvider` for automatic token management

### ⚠️ To Be Implemented
1. **Apply to all POST/PUT/DELETE routes** - Wrap existing API routes with CSRF protection
2. **Integrate with API client** - Update service classes to include CSRF tokens
3. **Add to root layout** - Include CSRFProvider in the app layout

## How to Use CSRF Protection

### 1. Protecting API Routes

Wrap your API route handlers with the `withCSRFProtection` middleware:

```typescript
// app/api/products/route.ts
import { withCSRFProtection } from '@/lib/security/csrf-middleware';

export const POST = withCSRFProtection(async (request: NextRequest) => {
  // Your existing POST handler code
});

export const PUT = withCSRFProtection(async (request: NextRequest) => {
  // Your existing PUT handler code
});

export const DELETE = withCSRFProtection(async (request: NextRequest) => {
  // Your existing DELETE handler code
});
```

### 2. Client-side Usage

Use the `useCSRF` hook in your components:

```typescript
import { useCSRF } from '@/hooks/security/use-csrf';

function MyComponent() {
  const { csrfToken, getHeaders } = useCSRF();

  const handleSubmit = async () => {
    const response = await fetch('/api/products', {
      method: 'POST',
      headers: getHeaders({
        'Content-Type': 'application/json',
      }),
      body: JSON.stringify(data),
    });
  };
}
```

### 3. Adding CSRFProvider

Add the provider to your root layout:

```typescript
// app/layout.tsx
import { CSRFProvider } from '@/components/providers/csrf-provider';

export default function RootLayout({ children }) {
  return (
    <html>
      <body>
        <CSRFProvider>
          {/* Other providers */}
          {children}
        </CSRFProvider>
      </body>
    </html>
  );
}
```

### 4. Updating API Services

Update your API service classes to include CSRF tokens:

```typescript
// In your API service class
class ProductService extends CypheraAPI {
  async createProduct(context: UserRequestContext, data: CreateProductRequest, csrfToken?: string) {
    const response = await fetch(`${this.baseUrl}/products`, {
      method: 'POST',
      headers: this.getHeaders(context, csrfToken),
      body: JSON.stringify(data),
    });
    return this.handleResponse(response);
  }
}
```

## Security Notes

1. **Token Storage**: CSRF tokens are stored in httpOnly cookies
2. **Token Validation**: Tokens are validated on every state-changing request
3. **Safe Methods**: GET, HEAD, and OPTIONS requests don't require CSRF tokens
4. **Excluded Routes**: Some routes (webhooks, external callbacks) are excluded from CSRF protection

## Environment Configuration

Add to your `.env` file:

```
CSRF_SECRET=your-secret-key-here
```

## Next Steps

1. Apply `withCSRFProtection` to all state-changing API routes
2. Update all service classes to accept and pass CSRF tokens
3. Add CSRFProvider to the root layout
4. Test CSRF protection with tools like Postman or curl