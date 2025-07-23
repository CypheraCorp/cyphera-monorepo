# Security Implementation - Next Steps

## ✅ Completed Security Implementations

### 1. HttpOnly Cookies (Already Implemented)
- Session management using secure httpOnly cookies
- Separate cookies for merchant and customer sessions
- Proper security flags (httpOnly, secure, sameSite)

### 2. CSRF Protection (Implemented)
- Created CSRF token generation and validation system
- Added CSRFProvider to root layout
- Created middleware wrapper for API routes
- Added client-side hooks for CSRF token management
- Updated API base class to support CSRF tokens

### 3. Input Validation with Zod (Implemented)
- Created comprehensive validation schemas for:
  - Products (create, update, pricing)
  - Subscriptions (create, query, cancel)
  - Customers (create, update, sign-in)
- Created validation utilities (validateBody, validateQuery, validateParams)
- Created withValidation HOF for easy integration

## 🚀 Immediate Next Steps

### 1. Apply Security to Remaining Routes

You need to apply the security wrappers to all remaining API routes:

```typescript
// Example for each route type:

// POST routes - need CSRF + validation
export const POST = withCSRFProtection(
  withValidation({ bodySchema }, handler)
);

// PUT routes - need CSRF + validation
export const PUT = withCSRFProtection(
  withValidation({ bodySchema, paramsSchema }, handler)
);

// DELETE routes - need CSRF only
export const DELETE = withCSRFProtection(handler);

// GET routes - may need validation for query params
export const GET = withValidation({ querySchema }, handler);
```

### 2. Routes That Need Security Applied

**High Priority (State-changing operations):**
- [ ] `/api/wallets/route.ts` - POST (create wallet)
- [ ] `/api/wallets/[walletId]/route.ts` - PUT, DELETE
- [ ] `/api/transactions/route.ts` - POST (create transaction)
- [ ] `/api/subscriptions/route.ts` - POST (create subscription)
- [ ] `/api/accounts/onboard/route.ts` - POST (onboard account)
- [ ] `/api/circle/users/route.ts` - POST (create Circle user)
- [ ] `/api/circle/users/pin/create/route.ts` - POST (create PIN)
- [ ] `/api/circle/wallets/route.ts` - POST (create wallet)

**Medium Priority (Query validation):**
- [ ] `/api/transactions/route.ts` - GET (add query validation)
- [ ] `/api/subscriptions/route.ts` - GET (add query validation)
- [ ] `/api/wallets/route.ts` - GET (add query validation)

### 3. Create Missing Validation Schemas

Create validation schemas for:
- [ ] Wallet operations (`/lib/validation/schemas/wallet.ts`)
- [ ] Transaction operations (`/lib/validation/schemas/transaction.ts`)
- [ ] Account operations (`/lib/validation/schemas/account.ts`)
- [ ] Circle API operations (`/lib/validation/schemas/circle.ts`)

### 4. Client-Side Integration

Update your client components to use CSRF tokens:

```typescript
// In your API calls:
const { getHeaders } = useCSRF();

await fetch('/api/endpoint', {
  method: 'POST',
  headers: getHeaders({ 'Content-Type': 'application/json' }),
  body: JSON.stringify(data),
});
```

### 5. Environment Configuration

Add to your `.env` file:
```
CSRF_SECRET=<generate-a-strong-random-string>
```

Generate a strong secret:
```bash
openssl rand -base64 32
```

## 📋 Security Checklist

Before deploying to production:

- [ ] All POST/PUT/DELETE routes have CSRF protection
- [ ] All routes accepting data have input validation
- [ ] CSRF_SECRET is set in production environment
- [ ] Client-side forms include CSRF tokens
- [ ] Error messages don't leak sensitive information
- [ ] Rate limiting is implemented (future task)
- [ ] Security headers are configured (future task)

## 🔍 Testing Your Security

### 1. Test CSRF Protection
```bash
# Should fail without CSRF token
curl -X POST http://localhost:3000/api/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Product"}'

# Should return: {"error": "CSRF token required"}
```

### 2. Test Input Validation
```bash
# Should fail with invalid data
curl -X POST http://localhost:3000/api/products \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: your-token" \
  -d '{"name": "", "wallet_id": "invalid-uuid"}'

# Should return validation errors
```

### 3. Test Session Security
- Check cookies in browser DevTools
- Verify httpOnly flag is set
- Verify secure flag in production

## 📚 Documentation References

- [CSRF Implementation Guide](./security-csrf-guide.md)
- [Validation Guide](./security-validation-guide.md)
- [Security Examples](./security-implementation-examples.md)
- [Implementation Summary](./security-implementation-summary.md)

## 🎯 Future Security Enhancements

1. **Rate Limiting** - Prevent brute force attacks
2. **API Versioning** - Maintain backward compatibility
3. **Request Signing** - Additional security for critical operations
4. **Security Headers** - CSP, HSTS, X-Frame-Options
5. **Audit Logging** - Track all security-relevant events
6. **Penetration Testing** - Regular security assessments