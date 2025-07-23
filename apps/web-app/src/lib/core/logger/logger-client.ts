// Client-side logger that works in the browser
// In production, this could send logs to a service like Sentry or LogRocket

type LogLevel = 'error' | 'warn' | 'info' | 'debug';

interface LogContext {
  [key: string]: unknown;
}

class ClientLogger {
  private isDevelopment = process.env.NODE_ENV === 'development';
  private logLevel: LogLevel = this.isDevelopment ? 'debug' : 'info';

  private shouldLog(level: LogLevel): boolean {
    const levels: Record<LogLevel, number> = {
      error: 0,
      warn: 1,
      info: 2,
      debug: 3,
    };

    return levels[level] <= levels[this.logLevel];
  }

  private formatMessage(level: LogLevel, message: string, context?: LogContext): string {
    const timestamp = new Date().toISOString();
    const contextStr = context ? ` ${JSON.stringify(context)}` : '';
    return `[${timestamp}] [${level.toUpperCase()}] ${message}${contextStr}`;
  }

  error(message: string, error?: Error | unknown, context?: LogContext) {
    if (!this.shouldLog('error')) return;

    const errorContext = {
      ...context,
      ...(error instanceof Error
        ? {
            errorMessage: error.message,
            errorStack: error.stack,
          }
        : { error }),
    };

    if (this.isDevelopment) {
      console.error(this.formatMessage('error', message, errorContext));
      if (error instanceof Error) {
        console.error(error);
      }
    } else {
      // In production, send to error tracking service
      // Example: Sentry.captureException(error, { extra: errorContext });
    }
  }

  warn(message: string, context?: LogContext) {
    if (!this.shouldLog('warn')) return;

    if (this.isDevelopment) {
      console.warn(this.formatMessage('warn', message, context));
    } else {
      // In production, could send to logging service
    }
  }

  info(message: string, context?: LogContext) {
    if (!this.shouldLog('info')) return;

    if (this.isDevelopment) {
      console.info(this.formatMessage('info', message, context));
    } else {
      // In production, could send to logging service
    }
  }

  debug(message: string, context?: LogContext) {
    if (!this.shouldLog('debug')) return;

    if (this.isDevelopment) {
      console.log(this.formatMessage('debug', message, context));
    }
    // Debug logs are typically not sent in production
  }
}

// Create singleton instance
const clientLogger = new ClientLogger();

export default clientLogger;
export { clientLogger };

// Export convenience methods
export const logError = (message: string, error?: Error | unknown, context?: LogContext) => {
  clientLogger.error(message, error, context);
};

export const logWarn = (message: string, context?: LogContext) => {
  clientLogger.warn(message, context);
};

export const logInfo = (message: string, context?: LogContext) => {
  clientLogger.info(message, context);
};

export const logDebug = (message: string, context?: LogContext) => {
  clientLogger.debug(message, context);
};
