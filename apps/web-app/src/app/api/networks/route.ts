import { NextRequest, NextResponse } from 'next/server';
import { NetworksAPI } from '@/services/cyphera-api/networks';
import { logger } from '@/lib/core/logger/logger';

// Type for cached network data
interface CachedNetworkData {
  data: unknown; // Will be the NetworkWithTokensResponse[] from the API
  timestamp: number;
}

// Simple in-memory cache for networks
const networksCache = new Map<string, CachedNetworkData>();
const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes

/**
 * GET handler for public networks endpoint
 * Fetches networks with filtering options (active, testnet)
 */
export async function GET(request: NextRequest) {
  try {
    // Get query parameters
    const { searchParams } = new URL(request.url);
    const testnetParam = searchParams.get('testnet');

    // Create cache key
    const cacheKey = `networks_active_true_testnet_${testnetParam || 'undefined'}`;

    // Check cache first
    const cached = networksCache.get(cacheKey);
    if (cached && Date.now() - cached.timestamp < CACHE_DURATION) {
      logger.debug('Networks cache hit', { cacheKey });
      const response = NextResponse.json(cached.data);
      // Add cache headers for client-side caching
      response.headers.set('Cache-Control', 'public, s-maxage=300, stale-while-revalidate=600');
      return response;
    }

    logger.debug('Networks cache miss, fetching from API', { cacheKey });

    // Instantiate the NetworksAPI service
    const networksApi = new NetworksAPI();

    // Start with mandatory active param
    const apiParams: { active: boolean; testnet?: boolean } = {
      active: true, // Assuming active=true is always desired for this route
    };

    // Only add testnet to params if the query parameter exists
    if (testnetParam !== null) {
      // Convert string 'true' or 'false' to boolean
      apiParams.testnet = testnetParam === 'true';
    }

    // Call the public API method with potentially optional testnet
    const networks = await networksApi.getNetworksWithTokens(apiParams);

    // Cache the result
    networksCache.set(cacheKey, { data: networks, timestamp: Date.now() });
    logger.debug('Networks cached result', { cacheKey });

    // Return networks as JSON with cache headers
    const response = NextResponse.json(networks);
    response.headers.set('Cache-Control', 'public, s-maxage=300, stale-while-revalidate=600');
    return response;
  } catch (error: unknown) {
    logger.error('Public Networks API route error', {
      error: error instanceof Error ? error.message : error,
    });
    const errorMessage = error instanceof Error ? error.message : 'Failed to fetch networks';
    return NextResponse.json({ error: errorMessage }, { status: 500 });
  }
}
