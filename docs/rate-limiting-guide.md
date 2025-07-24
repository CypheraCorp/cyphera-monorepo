# Rate Limiting Guide

## Overview

The Cyphera API implements rate limiting to protect against abuse and ensure fair usage across all clients. Rate limiting is applied globally with different limits for different types of endpoints.

## Implementation Details

### Rate Limit Configurations

1. **Default Rate Limit** (100 req/sec, burst: 200)
   - Applied to all standard API endpoints
   - Suitable for normal API operations

2. **Strict Rate Limit** (10 req/sec, burst: 20)
   - Applied to sensitive endpoints like authentication
   - Currently applied to:
     - `/api/v1/admin/accounts/signin`
     - `/api/v1/admin/customers/signin`

3. **Relaxed Rate Limit** (500 req/sec, burst: 1000)
   - Available for read-heavy endpoints
   - Not currently in use but available for future optimization

### Client Identification

Rate limits are applied per client, identified by (in order of priority):
1. **API Key** - First 8 characters used as identifier
2. **User ID** - For authenticated users (JWT auth)
3. **IP Address** - Fallback for unauthenticated requests

### Rate Limit Headers

All responses include rate limit information headers:
- `X-RateLimit-Limit` - The rate limit ceiling for that request
- `X-RateLimit-Remaining` - Number of requests left for the time window
- `X-RateLimit-Reset` - Time when the rate limit window resets (Unix timestamp)
- `Retry-After` - Seconds to wait before retrying (only on 429 responses)

### Exemptions

The following endpoints are exempt from rate limiting:
- `/health`
- `/healthz`

## Usage Examples

### Successful Request
```bash
curl -i https://api.cyphera.com/api/v1/products

HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1699564800
```

### Rate Limited Request
```bash
curl -i https://api.cyphera.com/api/v1/products

HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1699564800
Retry-After: 1
Content-Type: application/json

{
  "error": "Too many requests. Please try again later.",
  "retry_after": 1
}
```

## Best Practices for Clients

1. **Monitor Rate Limit Headers** - Check remaining requests before making additional calls
2. **Implement Exponential Backoff** - When receiving 429 errors, wait and retry with increasing delays
3. **Use API Keys** - API keys have separate rate limits from IP-based limiting
4. **Batch Operations** - Combine multiple operations where possible to reduce request count

## Configuration

Rate limits can be adjusted by modifying the configurations in `/libs/go/middleware/ratelimit.go`:

```go
var (
    // DefaultRateLimiter for general API endpoints
    DefaultRateLimiter = NewRateLimiter(100, 200)
    
    // StrictRateLimiter for sensitive endpoints
    StrictRateLimiter = NewRateLimiter(10, 20)
    
    // RelaxedRateLimiter for read-heavy endpoints
    RelaxedRateLimiter = NewRateLimiter(500, 1000)
)
```

## Monitoring

Rate limit violations are logged with the following information:
- Client identifier (API key prefix, user ID, or IP)
- Request path and method
- Timestamp

Monitor logs for patterns of rate limit violations to identify potential abuse or legitimate clients needing higher limits.

## Future Enhancements

1. **Redis-based Rate Limiting** - For distributed deployments
2. **Dynamic Rate Limits** - Different limits based on subscription tiers
3. **Endpoint-specific Limits** - Custom limits for specific endpoints
4. **Rate Limit Bypass** - For internal services or premium clients