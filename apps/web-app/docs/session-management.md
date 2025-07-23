# Unified Session Management

## Overview

The Cyphera Web application uses a unified session management system that supports both merchant and customer user types. This system provides a consistent API for creating, reading, updating, and deleting sessions while maintaining the necessary separation between merchant and customer contexts.

## Architecture

### Session Types

```typescript
type UserType = 'merchant' | 'customer';

interface MerchantSession {
  user_type: 'merchant';
  access_token: string;
  user_id?: string;
  account_id?: string;
  workspace_id?: string;
  email?: string;
  expires_at: number;
  created_at: number;
}

interface CustomerSession {
  user_type: 'customer';
  access_token: string;
  customer_id: string;
  customer_email: string;
  customer_name?: string;
  wallet_address?: string;
  wallet_id?: string;
  finished_onboarding?: boolean;
  expires_at: number;
  created_at: number;
}
```

### Storage

Sessions are stored in HTTP-only cookies with the following configuration:

- **Merchant sessions**: `cyphera-session` cookie
- **Customer sessions**: `cyphera-customer-session` cookie
- **Duration**: 7 days
- **Security**: HTTP-only, Secure (production), SameSite=lax

## Server-Side Usage

### UnifiedSessionService

The main service for server-side session management:

```typescript
import { UnifiedSessionService } from '@/lib/auth/session';

// Create a new session
const session = await UnifiedSessionService.create({
  user_type: 'merchant',
  access_token: token,
  user_id: userId,
  email: userEmail,
  // ... other fields
});

// Get current session (checks both types)
const session = await UnifiedSessionService.get();

// Get session by type
const merchantSession = await UnifiedSessionService.getByType('merchant');
const customerSession = await UnifiedSessionService.getByType('customer');

// Update session
const updated = await UnifiedSessionService.update({
  workspace_id: newWorkspaceId,
});

// Clear session
await UnifiedSessionService.clearByType('merchant');
await UnifiedSessionService.clearAll();

// Check for both sessions
const hasBoth = await UnifiedSessionService.hasBothSessions();

// Switch between sessions
const session = await UnifiedSessionService.switchTo('customer');
```

### Type Guards

```typescript
import { isMerchantSession, isCustomerSession } from '@/lib/auth/session';

const session = await UnifiedSessionService.get();
if (session && isMerchantSession(session)) {
  // TypeScript knows this is a MerchantSession
  console.log(session.user_id);
}
```

### Helper Functions

```typescript
import { requireMerchantSession, requireCustomerSession } from '@/lib/auth/session';

// These throw if no session exists
const merchantSession = await requireMerchantSession();
const customerSession = await requireCustomerSession();
```

## Client-Side Usage

### UnifiedSessionClient

For browser-side operations:

```typescript
import { UnifiedSessionClient } from '@/lib/auth/session';

// Get current session
const session = await UnifiedSessionClient.get();

// Get specific session type
const merchant = await UnifiedSessionClient.getByType('merchant');

// Clear sessions
await UnifiedSessionClient.clearByType('merchant');
await UnifiedSessionClient.clearAll();

// Check for both sessions
const hasBoth = await UnifiedSessionClient.hasBothSessions();

// Check if needs onboarding
const needsOnboarding = await UnifiedSessionClient.needsOnboarding('customer');
```

### Convenience Functions

```typescript
import {
  getSession,
  getMerchantSession,
  getCustomerSession,
  clearMerchantSession,
  clearCustomerSession,
  clearAllSessions,
  hasBothSessions,
} from '@/lib/auth/session';

// Direct function calls
const session = await getSession();
const merchant = await getMerchantSession();
await clearMerchantSession();
```

## API Routes

### Sign In

```typescript
// Merchant sign in
// POST /api/auth/signin
const session = await UnifiedSessionService.create({
  user_type: 'merchant',
  access_token: accessToken,
  user_id: response.user?.id,
  account_id: response.account?.id,
  workspace_id: workspaceId,
  email: userEmail,
});

// Customer sign in
// POST /api/auth/customer/signin
const session = await UnifiedSessionService.create({
  user_type: 'customer',
  access_token: accessToken,
  customer_id: customer.id,
  customer_email: customer.email,
  customer_name: customer.name,
  wallet_address: wallet?.wallet_address,
  wallet_id: wallet?.id,
  finished_onboarding: customer.finished_onboarding,
});
```

### Session Validation

```typescript
// GET /api/auth/me (merchant)
const session = await UnifiedSessionService.getByType('merchant');
if (!session) {
  return NextResponse.json({ error: 'No session found' }, { status: 401 });
}

// GET /api/auth/customer/me
const session = await UnifiedSessionService.getByType('customer');
if (!session) {
  return NextResponse.json({ error: 'No customer session found' }, { status: 401 });
}
```

### Logout

```typescript
// POST /api/auth/logout (merchant)
await UnifiedSessionService.clearByType('merchant');

// POST /api/auth/customer/logout
await UnifiedSessionService.clearByType('customer');
```

## Middleware Integration

The middleware automatically validates sessions and injects headers for API routes:

```typescript
// Automatic session validation
const session = await UnifiedSessionService.getFromRequest(request);

// Headers injected for API routes:
// - Authorization: Bearer <access_token>
// - x-customer-id (for customer sessions)
// - x-account-id, x-workspace-id, x-user-id (for merchant sessions)
```

## Migration Guide

### From Legacy Session Functions

```typescript
// Old way
import { getSession, clearSession } from '@/lib/auth/session';
const session = await getSession();
clearSession(response);

// New way
import { UnifiedSessionService } from '@/lib/auth/session';
const session = await UnifiedSessionService.get();
await UnifiedSessionService.clearAll();
```

### From Direct Cookie Access

```typescript
// Old way
const sessionCookie = request.cookies.get('cyphera-session');
const decoded = Buffer.from(sessionCookie.value, 'base64').toString('utf-8');
const sessionData = JSON.parse(decoded);

// New way
const session = await UnifiedSessionService.getFromRequest(request);
```

## Benefits

1. **Unified API**: Single interface for both merchant and customer sessions
2. **Type Safety**: Full TypeScript support with type guards
3. **Automatic Validation**: Built-in expiry checking
4. **Better Security**: Centralized cookie management
5. **Easier Testing**: Mock a single service instead of multiple functions
6. **Session Switching**: Support for users with both account types
7. **Future-Proof**: Easy to add new features like session rotation, Redis storage, etc.

## Best Practices

1. Always use the unified service instead of direct cookie manipulation
2. Use type guards when you need to access type-specific fields
3. Handle session expiry gracefully in your UI
4. Clear sessions on logout to ensure security
5. Use the convenience functions for common operations
6. Check for onboarding status before allowing access to protected features
