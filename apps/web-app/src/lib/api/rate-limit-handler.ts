import { clientLogger } from '@/lib/core/logger/logger-client';

/**
 * Rate limit error with retry information
 */
export class RateLimitError extends Error {
  public readonly retryAfter: number;
  public readonly limit: number;
  public readonly remaining: number;
  public readonly reset: number;

  constructor(response: Response) {
    const retryAfter = parseInt(response.headers.get('Retry-After') || '1', 10);
    super(`Rate limit exceeded. Please try again in ${retryAfter} seconds.`);
    
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
    this.limit = parseInt(response.headers.get('X-RateLimit-Limit') || '0', 10);
    this.remaining = parseInt(response.headers.get('X-RateLimit-Remaining') || '0', 10);
    this.reset = parseInt(response.headers.get('X-RateLimit-Reset') || '0', 10);
  }
}

/**
 * Configuration for rate limit retry behavior
 */
export interface RateLimitRetryConfig {
  maxRetries?: number;
  initialDelay?: number;
  maxDelay?: number;
  backoffMultiplier?: number;
}

/**
 * Default retry configuration
 */
const DEFAULT_RETRY_CONFIG: Required<RateLimitRetryConfig> = {
  maxRetries: 3,
  initialDelay: 1000, // 1 second
  maxDelay: 30000, // 30 seconds
  backoffMultiplier: 2,
};

/**
 * Sleep for a specified number of milliseconds
 */
function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Handle rate limited requests with automatic retry
 */
export async function handleRateLimitedRequest<T>(
  request: () => Promise<Response>,
  handleResponse: (response: Response) => Promise<T>,
  config?: RateLimitRetryConfig
): Promise<T> {
  const retryConfig = { ...DEFAULT_RETRY_CONFIG, ...config };
  let lastError: Error | null = null;
  let delay = retryConfig.initialDelay;

  for (let attempt = 0; attempt <= retryConfig.maxRetries; attempt++) {
    try {
      const response = await request();
      
      // Check if rate limited
      if (response.status === 429) {
        const rateLimitError = new RateLimitError(response);
        
        // If this is the last attempt, throw the error
        if (attempt === retryConfig.maxRetries) {
          throw rateLimitError;
        }
        
        // Calculate delay based on Retry-After header or exponential backoff
        const retryAfterMs = rateLimitError.retryAfter * 1000;
        const backoffDelay = Math.min(delay, retryConfig.maxDelay);
        const actualDelay = Math.max(retryAfterMs, backoffDelay);
        
        clientLogger.warn('Rate limit hit, retrying', {
          attempt: attempt + 1,
          maxRetries: retryConfig.maxRetries,
          delayMs: actualDelay,
          limit: rateLimitError.limit,
          remaining: rateLimitError.remaining,
          reset: new Date(rateLimitError.reset * 1000).toISOString(),
        });
        
        // Wait before retrying
        await sleep(actualDelay);
        
        // Increase delay for next attempt
        delay = Math.min(delay * retryConfig.backoffMultiplier, retryConfig.maxDelay);
        
        continue;
      }
      
      // Not rate limited, process response normally
      return await handleResponse(response);
    } catch (error) {
      // If it's not a rate limit error, just throw it
      if (!(error instanceof RateLimitError)) {
        throw error;
      }
      lastError = error;
    }
  }
  
  // If we get here, all retries failed
  throw lastError || new Error('Rate limit retry failed');
}

/**
 * Extract rate limit information from response headers
 */
export function getRateLimitInfo(response: Response): {
  limit: number;
  remaining: number;
  reset: Date | null;
} | null {
  const limit = response.headers.get('X-RateLimit-Limit');
  const remaining = response.headers.get('X-RateLimit-Remaining');
  const reset = response.headers.get('X-RateLimit-Reset');
  
  if (!limit || !remaining || !reset) {
    return null;
  }
  
  return {
    limit: parseInt(limit, 10),
    remaining: parseInt(remaining, 10),
    reset: new Date(parseInt(reset, 10) * 1000),
  };
}

