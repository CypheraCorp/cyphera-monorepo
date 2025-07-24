# Request Correlation IDs

## Overview

Request correlation IDs are unique identifiers assigned to each API request, enabling distributed tracing and debugging across the Cyphera system. This feature helps track requests as they flow through different services and makes debugging production issues significantly easier.

## How It Works

### API Side

1. **Automatic Generation**: If a request doesn't include a correlation ID, the API automatically generates one using UUID v4
2. **Header Propagation**: The correlation ID is included in the `X-Correlation-ID` response header
3. **Error Responses**: All error responses include the correlation ID in the JSON body:
   ```json
   {
     "error": "Error message",
     "correlation_id": "550e8400-e29b-41d4-a716-446655440000"
   }
   ```
4. **Logging**: All logs include the correlation ID for easy filtering in log aggregation systems

### Frontend Side

1. **Client-Generated IDs**: The frontend can generate its own correlation IDs for client-initiated requests
   - Format: `cyk_client_<timestamp>_<random>`
   - This helps trace requests from their origin

2. **Automatic Header Addition**: The API client automatically adds correlation IDs to all requests

3. **Error Logging**: Errors are logged with their correlation IDs for debugging:
   ```javascript
   console.error('Request failed', {
     correlationId: 'cyk_client_lq3n8k2_x7y9z',
     error: 'Network timeout'
   });
   ```

## Usage Examples

### Frontend - Using the Hook

```typescript
import { useCorrelationId } from '@/hooks/utils/use-correlation-id';

function MyComponent() {
  const { createCorrelationId, logError } = useCorrelationId();
  
  const handleSubmit = async () => {
    try {
      const response = await api.createProduct(data);
    } catch (error) {
      // Automatically extracts and logs correlation ID
      const correlationId = logError('Failed to create product', error);
      
      // Show user-friendly error with correlation ID for support
      toast.error(`Failed to create product. Reference ID: ${correlationId}`);
    }
  };
}
```

### Frontend - Manual Usage

```typescript
import { generateCorrelationId, logErrorWithCorrelation } from '@/lib/utils/correlation';

// Generate correlation ID for a request chain
const correlationId = generateCorrelationId();

// Add to headers
const headers = {
  'X-Correlation-ID': correlationId,
  // other headers...
};

// Log errors with correlation
try {
  const response = await fetch(url, { headers });
} catch (error) {
  logErrorWithCorrelation('API request failed', error, { 
    url, 
    correlationId 
  });
}
```

### Backend - Accessing Correlation ID

```go
// In Gin handlers
correlationID := c.GetHeader("X-Correlation-ID")

// From context
correlationID := middleware.GetCorrelationID(c)

// Logging with correlation ID
logger.Info("Processing request",
    zap.String("correlation_id", correlationID),
    zap.String("action", "create_product"),
)
```

## Benefits

1. **End-to-End Tracing**: Track requests from frontend through all backend services
2. **Faster Debugging**: Quickly find all logs related to a specific request
3. **Better Support**: Users can provide correlation IDs when reporting issues
4. **Performance Analysis**: Measure request duration across services
5. **Error Correlation**: Link errors across distributed systems

## Best Practices

1. **Include in Error Messages**: Always include correlation IDs in user-facing error messages for support reference
2. **Log at Boundaries**: Log correlation IDs when requests enter and exit services
3. **Preserve Existing IDs**: If a request already has a correlation ID, preserve it rather than generating a new one
4. **Use in Async Operations**: Pass correlation IDs to background jobs and async operations

## Logging

### Development Mode
In development, the API logs detailed request/response information including:
- Request headers (sensitive headers are redacted)
- Request body
- Response body
- Response headers
- Duration

### Production Mode
In production, only basic request information is logged:
- Method and path
- Status code
- Duration
- Correlation ID

## Security Considerations

- Correlation IDs should not contain sensitive information
- They are included in response headers and error messages, so they're visible to clients
- Use UUIDs or similar random identifiers to prevent information leakage