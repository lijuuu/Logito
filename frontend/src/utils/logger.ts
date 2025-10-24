// Simple logging utility for the frontend
export enum LogLevel {
  DEBUG = 'debug',
  INFO = 'info',
  WARN = 'warn',
  ERROR = 'error',
}

class Logger {
  private isDevelopment = process.env.NODE_ENV === 'development';

  private formatMessage(level: LogLevel, message: string, ...args: any[]): string {
    const timestamp = new Date().toISOString();
    const prefix = `[${timestamp}] [${level.toUpperCase()}]`;
    return `${prefix} ${message}`;
  }

  debug(message: string, ...args: any[]): void {
    if (this.isDevelopment) {
      console.debug(this.formatMessage(LogLevel.DEBUG, message), ...args);
    }
  }

  info(message: string, ...args: any[]): void {
    console.info(this.formatMessage(LogLevel.INFO, message), ...args);
  }

  warn(message: string, ...args: any[]): void {
    console.warn(this.formatMessage(LogLevel.WARN, message), ...args);
  }

  error(message: string, ...args: any[]): void {
    console.error(this.formatMessage(LogLevel.ERROR, message), ...args);
  }

  // API request logging
  logApiRequest(method: string, url: string, params?: any): void {
    this.debug(`API Request: ${method} ${url}`, params);
  }

  logApiResponse(method: string, url: string, status: number, duration: number): void {
    this.info(`API Response: ${method} ${url} - ${status} (${duration}ms)`);
  }

  logApiError(method: string, url: string, error: Error): void {
    this.error(`API Error: ${method} ${url}`, error);
  }

  // User action logging
  logUserAction(action: string, details?: any): void {
    this.info(`User Action: ${action}`, details);
  }

  // Performance logging
  logPerformance(operation: string, duration: number): void {
    this.info(`Performance: ${operation} took ${duration}ms`);
  }
}

export const logger = new Logger();
export default logger;
