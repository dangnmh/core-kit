import {
  LogLevel,
  LogLevelName,
  LogEntry,
  LoggerOptions,
  ChildLoggerOptions,
  ILogger
} from "./types.js";

export * from "./types.js";

export function parseLogLevel(level?: string | number): LogLevel {
  if (typeof level === "number") {
    return level;
  }
  if (typeof level === "string") {
    const normalized = level.trim().toUpperCase();
    switch (normalized) {
      case "DEBUG":
        return LogLevel.DEBUG;
      case "INFO":
      case "":
        return LogLevel.INFO;
      case "WARN":
      case "WARNING":
        return LogLevel.WARN;
      case "ERROR":
        return LogLevel.ERROR;
      case "FATAL":
      case "CRITICAL":
        return LogLevel.FATAL;
      default: {
        const parsed = parseInt(normalized, 10);
        if (!isNaN(parsed)) {
          return parsed;
        }
      }
    }
  }
  if (typeof process !== "undefined" && process.env && process.env.LOG_LEVEL) {
    return parseLogLevel(process.env.LOG_LEVEL);
  }
  return LogLevel.INFO;
}

export class Logger implements ILogger {
  private service: string;
  private module: string;
  private minLevel: number;
  private baseContext: Record<string, any>;
  private stream?: { write: (chunk: string) => void };
  private errorStream?: { write: (chunk: string) => void };

  constructor(options: LoggerOptions) {
    this.service = options.service;
    this.module = options.module || "";
    this.minLevel = parseLogLevel(options.minLevel);
    this.baseContext = options.baseContext ? { ...options.baseContext } : {};
    this.stream = options.stream;
    this.errorStream = options.errorStream;
  }

  public getService(): string {
    return this.service;
  }

  public getModule(): string {
    return this.module;
  }

  public getMinLevel(): number {
    return this.minLevel;
  }

  public setLevel(level: string | number): void {
    this.minLevel = parseLogLevel(level);
  }

  private write(entry: LogEntry, isErrorOrFatal: boolean): void {
    const line = JSON.stringify(entry);

    if (isErrorOrFatal && this.errorStream && typeof this.errorStream.write === "function") {
      this.errorStream.write(line + "\n");
      return;
    }

    if (!isErrorOrFatal && this.stream && typeof this.stream.write === "function") {
      this.stream.write(line + "\n");
      return;
    }

    if (this.stream && !this.errorStream && typeof this.stream.write === "function") {
      this.stream.write(line + "\n");
      return;
    }

    if (typeof process !== "undefined" && process.stdout && typeof process.stdout.write === "function") {
      if (isErrorOrFatal && process.stderr && typeof process.stderr.write === "function") {
        process.stderr.write(line + "\n");
      } else {
        process.stdout.write(line + "\n");
      }
    } else {
      /* eslint-disable no-console */
      if (isErrorOrFatal && typeof console !== "undefined" && typeof console.error === "function") {
        console.error(line);
      } else if (typeof console !== "undefined" && typeof console.log === "function") {
        console.log(line);
      }
      /* eslint-enable no-console */
    }
  }

  private resolveContext(callContext?: Record<string, any>): Record<string, any> | undefined {
    const hasBase = Object.keys(this.baseContext).length > 0;
    const hasCall = callContext !== undefined && Object.keys(callContext).length > 0;

    if (hasBase || hasCall) {
      return {
        ...this.baseContext,
        ...(callContext || {})
      };
    }
    return callContext;
  }

  private resolveErrorContext(
    errorOrContext?: Error | Record<string, any>,
    context?: Record<string, any>
  ): Record<string, any> | undefined {
    let errContext: Record<string, any> | undefined;

    if (errorOrContext instanceof Error) {
      errContext = {
        error: {
          name: errorOrContext.name,
          message: errorOrContext.message,
          stack: errorOrContext.stack
        }
      };
    } else if (typeof errorOrContext === "string") {
      errContext = { error: errorOrContext };
    } else if (errorOrContext && typeof errorOrContext === "object") {
      errContext = errorOrContext;
    }

    const mergedCall = errContext || context ? { ...(errContext || {}), ...(context || {}) } : undefined;
    return this.resolveContext(mergedCall);
  }

  private log(
    level: LogLevel,
    levelName: LogLevelName,
    message: string,
    resolvedContext?: Record<string, any>
  ): void {
    if (level < this.minLevel) {
      return;
    }

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      severity: levelName,
      service: this.service,
      logger: this.module,
      message: message
    };

    if (resolvedContext !== undefined) {
      if (resolvedContext.trace_id) {
        entry.trace_id = resolvedContext.trace_id;
      }
      if (resolvedContext.span_id) {
        entry.span_id = resolvedContext.span_id;
      }
      if (resolvedContext.caller) {
        entry.caller = resolvedContext.caller;
      }
      entry.context = resolvedContext;
    }

    const isErrorOrFatal = level >= LogLevel.ERROR;
    this.write(entry, isErrorOrFatal);
  }

  public debug(message: string, context?: Record<string, any>): void {
    this.log(LogLevel.DEBUG, "DEBUG", message, this.resolveContext(context));
  }

  public info(message: string, context?: Record<string, any>): void {
    this.log(LogLevel.INFO, "INFO", message, this.resolveContext(context));
  }

  public warn(message: string, context?: Record<string, any>): void {
    this.log(LogLevel.WARN, "WARN", message, this.resolveContext(context));
  }

  public error(
    message: string,
    errorOrContext?: Error | Record<string, any>,
    context?: Record<string, any>
  ): void {
    const merged = this.resolveErrorContext(errorOrContext, context);
    this.log(LogLevel.ERROR, "ERROR", message, merged);
  }

  public fatal(
    message: string,
    errorOrContext?: Error | Record<string, any>,
    context?: Record<string, any>
  ): void {
    const merged = this.resolveErrorContext(errorOrContext, context);
    this.log(LogLevel.FATAL, "FATAL", message, merged);
  }

  public child(options: ChildLoggerOptions): ILogger {
    const childModule = options.module !== undefined ? options.module : this.module;
    const childContext = {
      ...this.baseContext,
      ...(options.context || {})
    };
    return new Logger({
      service: this.service,
      module: childModule,
      minLevel: this.minLevel,
      baseContext: childContext,
      stream: this.stream,
      errorStream: this.errorStream
    });
  }
}

export function createLogger(options: LoggerOptions): ILogger {
  return new Logger(options);
}
