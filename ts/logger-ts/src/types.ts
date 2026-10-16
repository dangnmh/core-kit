export enum LogLevel {
  DEBUG = 10,
  INFO = 20,
  WARN = 30,
  ERROR = 40,
  FATAL = 50
}

export type LogLevelName = "DEBUG" | "INFO" | "WARN" | "ERROR" | "FATAL";

export interface LogEntry {
  timestamp: string;
  severity: LogLevelName;
  service: string;
  logger: string;
  message: string;
  caller?: string;
  trace_id?: string;
  span_id?: string;
  context?: Record<string, any>;
}

export interface LoggerOptions {
  service: string;
  module?: string;
  minLevel?: string | number;
  baseContext?: Record<string, any>;
  /** Custom destination stream; defaults to process.stdout */
  stream?: { write: (chunk: string) => void };
  /** Custom destination stream for ERROR and FATAL logs; defaults to process.stderr */
  errorStream?: { write: (chunk: string) => void };
}

export interface ChildLoggerOptions {
  module?: string;
  context?: Record<string, any>;
}

export interface ILogger {
  debug(message: string, context?: Record<string, any>): void;
  info(message: string, context?: Record<string, any>): void;
  warn(message: string, context?: Record<string, any>): void;
  error(message: string, errorOrContext?: Error | Record<string, any>, context?: Record<string, any>): void;
  fatal(message: string, errorOrContext?: Error | Record<string, any>, context?: Record<string, any>): void;
  child(options: ChildLoggerOptions): ILogger;
  setLevel?(level: string | number): void;
  getMinLevel?(): number;
  getService?(): string;
  getModule?(): string;
}

