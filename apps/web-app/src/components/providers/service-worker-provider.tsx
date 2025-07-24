'use client';

import { useEffect } from 'react';

export function ServiceWorkerProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    // Only register service worker in production
    if (process.env.NODE_ENV === 'production') {
      // Dynamically import to avoid webpack issues in development
      import('@/lib/service-worker').then(({ registerServiceWorker }) => {
        registerServiceWorker();
      }).catch(error => {
        console.error('Failed to load service worker module:', error);
      });
    }
  }, []);

  return <>{children}</>;
}
