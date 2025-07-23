// Service Worker for Cyphera Web
const CACHE_NAME = 'cyphera-v1';
const STATIC_CACHE = 'cyphera-static-v1';
const DYNAMIC_CACHE = 'cyphera-dynamic-v1';

// Assets to cache on install - empty for now to prevent errors
const STATIC_ASSETS = [];

// Cache strategies
const CACHE_STRATEGIES = {
  // Network first, fallback to cache
  NETWORK_FIRST: async (request) => {
    try {
      const response = await fetch(request);
      if (response.ok) {
        const cache = await caches.open(DYNAMIC_CACHE);
        cache.put(request, response.clone());
      }
      return response;
    } catch (error) {
      const cached = await caches.match(request);
      return cached || new Response('Offline', { status: 503 });
    }
  },

  // Cache first, fallback to network
  CACHE_FIRST: async (request) => {
    const cached = await caches.match(request);
    if (cached) return cached;

    try {
      const response = await fetch(request);
      if (response.ok) {
        const cache = await caches.open(STATIC_CACHE);
        cache.put(request, response.clone());
      }
      return response;
    } catch (error) {
      return new Response('Offline', { status: 503 });
    }
  },

  // Stale while revalidate
  STALE_WHILE_REVALIDATE: async (request) => {
    const cached = await caches.match(request);

    const fetchPromise = fetch(request).then((response) => {
      if (response.ok) {
        const cache = caches.open(DYNAMIC_CACHE);
        cache.then((c) => c.put(request, response.clone()));
      }
      return response;
    });

    return cached || fetchPromise;
  },
};

// Install event
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE).then((cache) => {
      return cache.addAll(STATIC_ASSETS);
    })
  );
  self.skipWaiting();
});

// Activate event
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((name) => name !== CACHE_NAME && name !== STATIC_CACHE && name !== DYNAMIC_CACHE)
          .map((name) => caches.delete(name))
      );
    })
  );
  self.clients.claim();
});

// Fetch event
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Skip non-GET requests
  if (request.method !== 'GET') return;

  // Skip Chrome extensions and external requests
  if (!url.origin.includes(self.location.origin)) return;

  // API routes - network first
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(CACHE_STRATEGIES.NETWORK_FIRST(request));
    return;
  }

  // Static assets - cache first
  if (url.pathname.match(/\.(js|css|png|jpg|jpeg|svg|ico|woff2?)$/)) {
    event.respondWith(CACHE_STRATEGIES.CACHE_FIRST(request));
    return;
  }

  // HTML pages - stale while revalidate
  if (request.headers.get('accept') && request.headers.get('accept').includes('text/html')) {
    event.respondWith(CACHE_STRATEGIES.STALE_WHILE_REVALIDATE(request));
    return;
  }

  // Default - network first
  event.respondWith(CACHE_STRATEGIES.NETWORK_FIRST(request));
});

// Background sync for failed requests
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-failed-requests') {
    event.waitUntil(syncFailedRequests());
  }
});

async function syncFailedRequests() {
  // Implement retry logic for failed API requests
  // This would require storing failed requests in IndexedDB
}

// Push notifications (if needed in future)
self.addEventListener('push', (event) => {
  if (event.data) {
    const data = event.data.json();
    event.waitUntil(
      self.registration.showNotification(data.title, {
        body: data.body,
        icon: '/icon-192x192.png',
        badge: '/icon-72x72.png',
      })
    );
  }
});
