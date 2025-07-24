'use client';

import { useState, useCallback } from 'react';
import { getRateLimitInfo } from '@/lib/api/rate-limit-handler';

/**
 * React hook for displaying rate limit information
 */
export function useRateLimitInfo() {
  const [rateLimitInfo, setRateLimitInfo] = useState<{
    limit: number;
    remaining: number;
    reset: Date;
  } | null>(null);

  const updateRateLimitInfo = useCallback((response: Response) => {
    const info = getRateLimitInfo(response);
    if (info && info.reset) {
      setRateLimitInfo({
        limit: info.limit,
        remaining: info.remaining,
        reset: info.reset,
      });
    }
  }, []);

  return { rateLimitInfo, updateRateLimitInfo };
}