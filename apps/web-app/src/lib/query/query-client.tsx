'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode, useState } from 'react';

// Cache durations based on data freshness requirements
const CACHE_DURATIONS = {
  // Static or rarely changing data
  networks: 60 * 60 * 1000, // 1 hour (very stable)
  tokens: 30 * 60 * 1000, // 30 minutes (stable)

  // Semi-static data
  products: 15 * 60 * 1000, // 15 minutes (occasionally updated)
  user: 10 * 60 * 1000, // 10 minutes (profile data)

  // Dynamic but cacheable data
  wallets: 5 * 60 * 1000, // 5 minutes
  customers: 5 * 60 * 1000, // 5 minutes
  subscriptions: 3 * 60 * 1000, // 3 minutes

  // Frequently changing data
  transactions: 1 * 60 * 1000, // 1 minute
  balances: 30 * 1000, // 30 seconds
};

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 2 * 60 * 1000, // 2 minutes default
        gcTime: 10 * 60 * 1000, // 10 minutes garbage collection
        retry: 3,
        refetchOnWindowFocus: false,
        refetchOnReconnect: true,
      },
      mutations: {
        retry: 1,
      },
    },
  });
}

export function QueryProvider({ children }: { children: ReactNode }) {
  // Create client in state to ensure it's stable across renders
  const [queryClient] = useState(() => createQueryClient());

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

export { CACHE_DURATIONS };
