# Frontend Rate Limiting Guide

## Overview

The Cyphera web application now includes automatic rate limiting handling for all API requests. When the API returns a 429 (Too Many Requests) status, the frontend will automatically retry the request with exponential backoff.

## Key Components

### 1. Rate Limit Handler (`/lib/api/rate-limit-handler.ts`)

The core functionality that:
- Detects 429 status codes
- Extracts rate limit information from headers
- Implements automatic retry with exponential backoff
- Provides `RateLimitError` class with retry information

### 2. API Service Integration

All API service classes now use `fetchWithRateLimit` method which automatically:
- Handles rate limiting
- Retries failed requests (up to 3 times by default)
- Uses exponential backoff with jitter

### 3. UI Components

#### Rate Limit Notification Component
```tsx
import { useRateLimitNotification } from '@/components/ui/rate-limit-notification';

function MyComponent() {
  const { RateLimitNotification, handleError } = useRateLimitNotification();
  
  const fetchData = async () => {
    try {
      const data = await api.getData();
      // Handle success
    } catch (error) {
      if (!handleError(error)) {
        // Handle other errors
      }
    }
  };

  return (
    <>
      <RateLimitNotification onRetry={fetchData} />
      {/* Your component content */}
    </>
  );
}
```

#### Using the Rate Limit Hook
```tsx
import { useRateLimitHandler } from '@/hooks/api/use-rate-limit-handler';

function MyComponent() {
  const { handleApiCall, isRetrying } = useRateLimitHandler();
  const [data, setData] = useState(null);
  
  const fetchData = async () => {
    const result = await handleApiCall(
      () => api.getData(),
      {
        onSuccess: (data) => setData(data),
        onError: (error) => console.error('Failed:', error),
        showToast: true, // Show toast notification on rate limit
      }
    );
  };
  
  return (
    <Button onClick={fetchData} disabled={isRetrying}>
      {isRetrying ? 'Retrying...' : 'Fetch Data'}
    </Button>
  );
}
```

## Rate Limit Information

The frontend has access to the following rate limit information from response headers:
- `X-RateLimit-Limit` - Total requests allowed per window
- `X-RateLimit-Remaining` - Requests remaining in current window
- `X-RateLimit-Reset` - Unix timestamp when the window resets
- `Retry-After` - Seconds to wait before retrying (on 429 responses)

## Automatic Retry Configuration

Default configuration:
- **Max Retries**: 3
- **Initial Delay**: 1 second
- **Max Delay**: 30 seconds
- **Backoff Multiplier**: 2x

You can customize this per request:
```typescript
const data = await handleRateLimitedRequest(
  () => fetch(url, options),
  (response) => response.json(),
  {
    maxRetries: 5,
    initialDelay: 2000, // 2 seconds
    maxDelay: 60000, // 60 seconds
    backoffMultiplier: 1.5,
  }
);
```

## Best Practices

1. **Use the built-in handlers** - All API services already include rate limit handling
2. **Show user feedback** - Use the notification component for user-facing operations
3. **Disable UI during retry** - Prevent duplicate requests while retrying
4. **Monitor rate limit headers** - Track usage to avoid hitting limits

## Example: Product List with Rate Limiting

```tsx
'use client';

import { useState, useEffect } from 'react';
import { useRateLimitNotification } from '@/components/ui/rate-limit-notification';
import { ProductsAPI } from '@/services/cyphera-api/products';
import { useAPIContext } from '@/hooks/use-api-context';

export function ProductList() {
  const [products, setProducts] = useState([]);
  const [loading, setLoading] = useState(false);
  const { context } = useAPIContext();
  const { RateLimitNotification, handleError, clearError } = useRateLimitNotification();
  
  const loadProducts = async () => {
    setLoading(true);
    clearError();
    
    try {
      const api = new ProductsAPI();
      const response = await api.getProducts(context);
      setProducts(response.data);
    } catch (error) {
      if (!handleError(error)) {
        // Handle non-rate-limit errors
        console.error('Failed to load products:', error);
      }
    } finally {
      setLoading(false);
    }
  };
  
  useEffect(() => {
    loadProducts();
  }, []);
  
  return (
    <div>
      <RateLimitNotification onRetry={loadProducts} />
      
      {loading ? (
        <div>Loading products...</div>
      ) : (
        <div>
          {products.map(product => (
            <div key={product.id}>{product.name}</div>
          ))}
        </div>
      )}
    </div>
  );
}
```

## Migration Notes

No changes are required for existing code. The rate limiting is handled automatically at the API service layer. However, for better user experience, consider:

1. Adding rate limit notifications to user-facing operations
2. Disabling UI elements during automatic retries
3. Monitoring rate limit usage in production

## Troubleshooting

1. **Requests still failing after retries**
   - Check if the rate limit window has reset
   - Verify the API key or user has appropriate limits
   - Consider implementing request batching

2. **UI not updating during retry**
   - Use the `isRetrying` state from hooks
   - Implement proper loading states

3. **Too many retry attempts**
   - Adjust the retry configuration
   - Implement request queuing for bulk operations