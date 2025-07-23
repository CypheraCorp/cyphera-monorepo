import winston from 'winston';
import DailyRotateFile from 'winston-daily-rotate-file';

// Define log levels
const levels = {
  error: 0,
  warn: 1,
  info: 2,
  http: 3,
  debug: 4,
};

// Define colors for each level
const colors = {
  error: 'red',
  warn: 'yellow',
  info: 'green',
  http: 'magenta',
  debug: 'blue',
};

// Tell winston about our colors
winston.addColors(colors);

// Define log format
const format = winston.format.combine(
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss:ms' }),
  winston.format.errors({ stack: true }),
  winston.format.json()
);

// Define format for console (development)
const consoleFormat = winston.format.combine(
  winston.format.colorize({ all: true }),
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss:ms' }),
  winston.format.printf(
    (info) =>
      `${info.timestamp} [${info.level}]: ${info.message}${info.stack ? '\n' + info.stack : ''}`
  )
);

// Define which transports to use based on environment
const transports: winston.transport[] = [];

// Always use console transport
transports.push(
  new winston.transports.Console({
    format: process.env.NODE_ENV === 'development' ? consoleFormat : format,
  })
);

// In production, also write to files
if (process.env.NODE_ENV === 'production') {
  // Daily rotate file for all logs
  transports.push(
    new DailyRotateFile({
      filename: 'logs/application-%DATE%.log',
      datePattern: 'YYYY-MM-DD',
      zippedArchive: true,
      maxSize: '20m',
      maxFiles: '14d',
      format,
    })
  );

  // Daily rotate file for errors only
  transports.push(
    new DailyRotateFile({
      filename: 'logs/error-%DATE%.log',
      datePattern: 'YYYY-MM-DD',
      zippedArchive: true,
      maxSize: '20m',
      maxFiles: '14d',
      level: 'error',
      format,
    })
  );
}

// Create the logger instance
const logger = winston.createLogger({
  level: process.env.LOG_LEVEL || (process.env.NODE_ENV === 'development' ? 'debug' : 'info'),
  levels,
  format,
  transports,
  exceptionHandlers: [new winston.transports.File({ filename: 'logs/exceptions.log' })],
  rejectionHandlers: [new winston.transports.File({ filename: 'logs/rejections.log' })],
});

// Export logger instance
export default logger;
export { logger };

// Export convenience methods
export const logError = (error: Error | unknown, context?: Record<string, unknown>) => {
  if (error instanceof Error) {
    logger.error(error.message, { stack: error.stack, ...context });
  } else {
    logger.error('Unknown error', { error, ...context });
  }
};

export const logInfo = (message: string, context?: Record<string, unknown>) => {
  logger.info(message, context);
};

export const logWarn = (message: string, context?: Record<string, unknown>) => {
  logger.warn(message, context);
};

export const logDebug = (message: string, context?: Record<string, unknown>) => {
  logger.debug(message, context);
};

export const logHttp = (message: string, context?: Record<string, unknown>) => {
  logger.http(message, context);
};
