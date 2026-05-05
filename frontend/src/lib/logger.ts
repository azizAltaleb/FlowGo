/**
 * Frontend Logger with structured logging and correlation ID support
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

interface LogEntry {
  timestamp: string;
  level: LogLevel;
  component: string;
  correlationId?: string;
  message: string;
  fields?: Record<string, unknown>;
  duration_ms?: number;
}

// Get log level from environment or default to 'info'
const getLogLevel = (): LogLevel => {
  const level = (import.meta.env.VITE_LOG_LEVEL || 'info').toLowerCase();
  if (['debug', 'info', 'warn', 'error'].includes(level)) {
    return level as LogLevel;
  }
  return 'info';
};

const LOG_LEVEL_PRIORITY: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

const currentLogLevel = getLogLevel();

const shouldLog = (level: LogLevel): boolean => {
  return LOG_LEVEL_PRIORITY[level] >= LOG_LEVEL_PRIORITY[currentLogLevel];
};

// Generate a unique correlation ID
export const generateCorrelationId = (): string => {
  const timestamp = Date.now().toString(36);
  const random = Math.random().toString(36).substring(2, 8);
  return `${timestamp}-${random}`;
};

// Store correlation ID for the current session/request
let currentCorrelationId: string | null = null;

export const setCorrelationId = (id: string): void => {
  currentCorrelationId = id;
};

export const getCorrelationId = (): string => {
  if (!currentCorrelationId) {
    currentCorrelationId = generateCorrelationId();
  }
  return currentCorrelationId;
};

class Logger {
  private component: string;

  constructor(component: string) {
    this.component = component;
  }

  private log(level: LogLevel, message: string, fields?: Record<string, unknown>): void {
    if (!shouldLog(level)) return;

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      component: this.component,
      correlationId: currentCorrelationId || undefined,
      message,
      fields: fields && Object.keys(fields).length > 0 ? fields : undefined,
    };

    // Format for console
    const prefix = `[${entry.timestamp}] [${level.toUpperCase()}] [${this.component}]`;
    const correlationPart = entry.correlationId ? ` [${entry.correlationId}]` : '';
    const fieldsPart = entry.fields ? ` ${JSON.stringify(entry.fields)}` : '';

    switch (level) {
      case 'debug':
        console.debug(`${prefix}${correlationPart} ${message}${fieldsPart}`);
        break;
      case 'info':
        console.info(`${prefix}${correlationPart} ${message}${fieldsPart}`);
        break;
      case 'warn':
        console.warn(`${prefix}${correlationPart} ${message}${fieldsPart}`);
        break;
      case 'error':
        console.error(`${prefix}${correlationPart} ${message}${fieldsPart}`);
        break;
    }

    // In production, you could send logs to a backend service here
    // sendToLogService(entry);
  }

  debug(message: string, fields?: Record<string, unknown>): void {
    this.log('debug', message, fields);
  }

  info(message: string, fields?: Record<string, unknown>): void {
    this.log('info', message, fields);
  }

  warn(message: string, fields?: Record<string, unknown>): void {
    this.log('warn', message, fields);
  }

  error(message: string, fields?: Record<string, unknown>): void {
    this.log('error', message, fields);
  }

  // Helper for timing operations
  time(operation: string): () => void {
    const start = performance.now();
    this.debug(`${operation} started`);
    return () => {
      const duration = Math.round(performance.now() - start);
      this.debug(`${operation} completed`, { duration_ms: duration });
    };
  }
}

// Factory function to create loggers
export const createLogger = (component: string): Logger => {
  return new Logger(component);
};

// Default logger for general use
export const logger = createLogger('app');
