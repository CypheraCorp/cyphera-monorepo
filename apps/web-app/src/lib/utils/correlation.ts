/**
 * Correlation ID utilities for request tracing
 */

/**
 * Generate a correlation ID for client-initiated requests
 * Format: cyk_client_<timestamp>_<random>
 */
export function generateCorrelationId(): string {
  const timestamp = Date.now().toString(36);
  const random = Math.random().toString(36).substring(2, 9);
  return `cyk_client_${timestamp}_${random}`;
}

/**
 * Extract correlation ID from error response
 */
export function getCorrelationIdFromError(error: any): string | undefined {
  if (error?.correlation_id) {
    return error.correlation_id;
  }
  return undefined;
}

/**
 * Log error with correlation ID for debugging
 */
export function logErrorWithCorrelation(
  message: string,
  error: any,
  additionalData?: Record<string, any>
): void {
  const correlationId = getCorrelationIdFromError(error);
  
  console.error(message, {
    error: error?.error || error?.message || error,
    correlationId,
    timestamp: new Date().toISOString(),
    ...additionalData,
  });
}

/**
 * Create headers with correlation ID
 */
export function createHeadersWithCorrelationId(
  existingHeaders?: Record<string, string>,
  correlationId?: string
): Record<string, string> {
  const headers = { ...existingHeaders };
  
  if (correlationId) {
    headers['X-Correlation-ID'] = correlationId;
  } else if (!headers['X-Correlation-ID']) {
    // Generate a new correlation ID if none exists
    headers['X-Correlation-ID'] = generateCorrelationId();
  }
  
  return headers;
}