import { NextResponse } from 'next/server';
import crypto from 'crypto';

export type CacheStrategy = 'public' | 'private' | 'no-cache';

interface CacheOptions {
  strategy?: CacheStrategy;
  maxAge?: number; // seconds
  staleWhileRevalidate?: number; // seconds
  mustRevalidate?: boolean;
  immutable?: boolean;
}

// Predefined cache durations (in seconds)
export const CACHE_DURATIONS = {
  NONE: 0,
  SHORT: 60, // 1 minute
  MEDIUM: 300, // 5 minutes
  LONG: 600, // 10 minutes
  VERY_LONG: 1800, // 30 minutes
  HOUR: 3600, // 1 hour
  DAY: 86400, // 24 hours
} as const;

// Default cache configurations for different data types
export const CACHE_CONFIGS = {
  // Frequently changing data
  DYNAMIC: {
    strategy: 'private' as CacheStrategy,
    maxAge: CACHE_DURATIONS.SHORT,
    staleWhileRevalidate: CACHE_DURATIONS.MEDIUM,
  },

  // User-specific data
  USER_DATA: {
    strategy: 'private' as CacheStrategy,
    maxAge: CACHE_DURATIONS.MEDIUM,
    staleWhileRevalidate: CACHE_DURATIONS.LONG,
  },

  // Shared data that changes occasionally
  SHARED_DATA: {
    strategy: 'public' as CacheStrategy,
    maxAge: CACHE_DURATIONS.LONG,
    staleWhileRevalidate: CACHE_DURATIONS.VERY_LONG,
  },

  // Static configuration data
  CONFIG_DATA: {
    strategy: 'public' as CacheStrategy,
    maxAge: CACHE_DURATIONS.VERY_LONG,
    staleWhileRevalidate: CACHE_DURATIONS.HOUR,
  },

  // No caching for sensitive data
  SENSITIVE: {
    strategy: 'no-cache' as CacheStrategy,
    maxAge: CACHE_DURATIONS.NONE,
  },
} as const;

/**
 * Generate ETag from data
 */
export function generateETag(data: unknown): string {
  const hash = crypto.createHash('md5');
  hash.update(JSON.stringify(data));
  return `"${hash.digest('hex')}"`;
}

/**
 * Set cache headers on a NextResponse
 */
export function setCacheHeaders(
  response: NextResponse,
  options: CacheOptions = CACHE_CONFIGS.DYNAMIC
): NextResponse {
  const {
    strategy = 'private',
    maxAge = CACHE_DURATIONS.MEDIUM,
    staleWhileRevalidate,
    mustRevalidate = false,
    immutable = false,
  } = options;

  // Build Cache-Control header
  const directives: string[] = [strategy];

  if (strategy !== 'no-cache') {
    directives.push(`max-age=${maxAge}`);

    if (staleWhileRevalidate) {
      directives.push(`stale-while-revalidate=${staleWhileRevalidate}`);
    }

    if (mustRevalidate) {
      directives.push('must-revalidate');
    }

    if (immutable) {
      directives.push('immutable');
    }
  }

  response.headers.set('Cache-Control', directives.join(', '));

  return response;
}

/**
 * Create a cached response with proper headers
 */
export function createCachedResponse<T>(
  data: T,
  options: CacheOptions = CACHE_CONFIGS.DYNAMIC
): NextResponse<T> {
  const response = NextResponse.json(data);

  // Set cache headers
  setCacheHeaders(response, options);

  // Set ETag
  const etag = generateETag(data);
  response.headers.set('ETag', etag);

  return response;
}

/**
 * Check if request has valid ETag
 */
export function checkETag(request: Request, data: unknown): boolean {
  const clientETag = request.headers.get('If-None-Match');
  if (!clientETag) return false;

  const currentETag = generateETag(data);
  return clientETag === currentETag;
}

/**
 * Return 304 Not Modified response
 */
export function notModifiedResponse(): NextResponse {
  return new NextResponse(null, { status: 304 });
}
