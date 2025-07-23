// Unified logger that works on both client and server

export const logger = {
  error: async (message: string, error?: Error | unknown, context?: Record<string, unknown>) => {
    if (typeof window === 'undefined') {
      // Server-side
      const { logError } = await import('./logger');
      logError(error || new Error(message), { message, ...context });
    } else {
      // Client-side
      const { logError } = await import('./logger-client');
      logError(message, error, context);
    }
  },

  warn: async (message: string, context?: Record<string, unknown>) => {
    if (typeof window === 'undefined') {
      const { logWarn } = await import('./logger');
      logWarn(message, context);
    } else {
      const { logWarn } = await import('./logger-client');
      logWarn(message, context);
    }
  },

  info: async (message: string, context?: Record<string, unknown>) => {
    if (typeof window === 'undefined') {
      const { logInfo } = await import('./logger');
      logInfo(message, context);
    } else {
      const { logInfo } = await import('./logger-client');
      logInfo(message, context);
    }
  },

  debug: async (message: string, context?: Record<string, unknown>) => {
    if (typeof window === 'undefined') {
      const { logDebug } = await import('./logger');
      logDebug(message, context);
    } else {
      const { logDebug } = await import('./logger-client');
      logDebug(message, context);
    }
  },

  http: async (message: string, context?: Record<string, unknown>) => {
    if (typeof window === 'undefined') {
      const { logHttp } = await import('./logger');
      logHttp(message, context);
    }
    // No HTTP logging on client side
  },

  // Synchronous versions for cases where async isn't possible
  log: (...args: unknown[]) => {
    console.log(...args);
  },

  error_sync: (message: string, error?: Error | unknown) => {
    console.error(message, error);
  },

  warn_sync: (message: string) => {
    console.warn(message);
  },

  info_sync: (message: string) => {
    console.info(message);
  },

  debug_sync: (message: string) => {
    console.debug(message);
  },
};
