import { logger } from '@/lib/core/logger/logger-utils';

export async function registerServiceWorker() {
  if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
    return;
  }

  // Only register in production
  if (process.env.NODE_ENV !== 'production') {
    logger.debug('Service worker registration skipped in development');
    return;
  }

  try {
    const registration = await navigator.serviceWorker.register('/sw.js', {
      scope: '/',
    });

    logger.info('Service worker registered', {
      scope: registration.scope,
    });

    // Check for updates
    registration.addEventListener('updatefound', () => {
      const newWorker = registration.installing;
      if (!newWorker) return;

      newWorker.addEventListener('statechange', () => {
        if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
          // New service worker available
          logger.info('New service worker available, reload to update');

          // You could show a toast here to prompt user to reload
          if (window.confirm('A new version is available. Reload to update?')) {
            window.location.reload();
          }
        }
      });
    });
  } catch (error) {
    logger.error('Service worker registration failed', error);
  }
}

export async function unregisterServiceWorker() {
  if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
    return;
  }

  try {
    const registrations = await navigator.serviceWorker.getRegistrations();

    for (const registration of registrations) {
      await registration.unregister();
    }

    logger.info('Service workers unregistered');
  } catch (error) {
    logger.error('Service worker unregistration failed', error);
  }
}
