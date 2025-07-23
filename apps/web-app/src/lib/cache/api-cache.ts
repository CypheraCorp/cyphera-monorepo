// Simple in-memory cache for API responses
// In production, this should be replaced with Redis or similar

interface CacheEntry<T = unknown> {
  data: T;
  timestamp: number;
  ttl: number; // Time to live in milliseconds
}

class APICache {
  private cache = new Map<string, CacheEntry<unknown>>();
  private maxSize = 1000; // Maximum cache entries

  set<T = unknown>(key: string, data: T, ttlMs: number = 5 * 60 * 1000): void {
    // Clean up if cache is getting too large
    if (this.cache.size >= this.maxSize) {
      this.cleanup();
    }

    this.cache.set(key, {
      data: data as unknown,
      timestamp: Date.now(),
      ttl: ttlMs,
    });
  }

  get<T = unknown>(key: string): T | null {
    const entry = this.cache.get(key);

    if (!entry) {
      return null;
    }

    // Check if entry has expired
    if (Date.now() - entry.timestamp > entry.ttl) {
      this.cache.delete(key);
      return null;
    }

    return entry.data as T;
  }

  delete(key: string): void {
    this.cache.delete(key);
  }

  clear(): void {
    this.cache.clear();
  }

  // Clean up expired entries
  private cleanup(): void {
    const now = Date.now();
    for (const [key, entry] of this.cache.entries()) {
      if (now - entry.timestamp > entry.ttl) {
        this.cache.delete(key);
      }
    }
  }

  // Get cache stats for monitoring
  getStats() {
    return {
      size: this.cache.size,
      maxSize: this.maxSize,
    };
  }
}

// Export singleton instance
export const apiCache = new APICache();

// Cache duration constants (in milliseconds)
export const CACHE_DURATIONS = {
  USER: 5 * 60 * 1000, // 5 minutes
  PRODUCTS: 10 * 60 * 1000, // 10 minutes
  NETWORKS: 30 * 60 * 1000, // 30 minutes (rarely change)
  WALLETS: 5 * 60 * 1000, // 5 minutes
  CUSTOMERS: 2 * 60 * 1000, // 2 minutes
  SUBSCRIPTIONS: 2 * 60 * 1000, // 2 minutes
  TRANSACTIONS: 1 * 60 * 1000, // 1 minute
};

// Helper function to generate cache keys
export function generateCacheKey(endpoint: string, params?: Record<string, unknown>): string {
  const baseKey = endpoint.replace(/^\/api\//, '').replace(/\//g, ':');

  if (!params || Object.keys(params).length === 0) {
    return baseKey;
  }

  const sortedParams = Object.keys(params)
    .sort()
    .map((key) => `${key}=${params[key]}`)
    .join('&');

  return `${baseKey}?${sortedParams}`;
}

// Wrapper function for caching API responses
export async function withCache<T>(
  cacheKey: string,
  fetchFn: () => Promise<T>,
  ttlMs: number = 5 * 60 * 1000
): Promise<T> {
  // Try to get from cache first
  const cached = apiCache.get(cacheKey);
  if (cached !== null) {
    return cached as T;
  }

  // Fetch fresh data
  const data = await fetchFn();

  // Cache the result
  apiCache.set(cacheKey, data, ttlMs);

  return data;
}
