import { describe, it, expect, beforeEach, spyOn } from "bun:test";
import { createLogger, LogLevel, parseLogLevel } from "../src/index.js";

describe("packages/logger-ts unit tests", () => {
  let stdoutData: string[] = [];
  let stderrData: string[] = [];
  let customStream: { write: (chunk: string) => void };
  let errorStream: { write: (chunk: string) => void };

  beforeEach(() => {
    stdoutData = [];
    stderrData = [];

    customStream = {
      write: (chunk: string) => {
        stdoutData.push(chunk);
      }
    };

    errorStream = {
      write: (chunk: string) => {
        stderrData.push(chunk);
      }
    };
  });

  it("should output single line JSON with expected schema for info log", () => {
    const logger = createLogger({
      service: "minecraft-bot",
      module: "main",
      minLevel: LogLevel.INFO,
      stream: customStream,
      errorStream: errorStream
    });

    logger.info("Bot connected successfully", { server: "localhost", port: 25565 });

    expect(stdoutData.length).toBe(1);
    expect(stderrData.length).toBe(0);

    const logLine = stdoutData[0];
    expect(logLine.endsWith("\n")).toBe(true);

    const parsed = JSON.parse(logLine.trim());
    expect(parsed.service).toBe("minecraft-bot");
    expect(parsed.logger).toBe("main");
    expect(parsed.severity).toBe("INFO");
    expect(parsed.message).toBe("Bot connected successfully");
    expect(parsed.context).toEqual({ server: "localhost", port: 25565 });
    expect(new Date(parsed.timestamp).toString()).not.toBe("Invalid Date");
  });

  it("should filter out logs below minLevel", () => {
    const logger = createLogger({
      service: "test-service",
      module: "filter-test",
      minLevel: LogLevel.WARN,
      stream: customStream,
      errorStream: errorStream
    });

    logger.debug("Debug message that should be skipped");
    logger.info("Info message that should be skipped");

    expect(stdoutData.length).toBe(0);
    expect(stderrData.length).toBe(0);

    logger.warn("Warning message that should appear");
    expect(stdoutData.length).toBe(1);

    const parsed = JSON.parse(stdoutData[0].trim());
    expect(parsed.severity).toBe("WARN");
    expect(parsed.message).toBe("Warning message that should appear");
  });

  it("should write ERROR and FATAL to stderr", () => {
    const logger = createLogger({
      service: "test-service",
      module: "error-test",
      minLevel: LogLevel.DEBUG,
      stream: customStream,
      errorStream: errorStream
    });

    logger.error("Something went wrong");
    logger.fatal("System crash");

    expect(stdoutData.length).toBe(0);
    expect(stderrData.length).toBe(2);

    const errorEntry = JSON.parse(stderrData[0].trim());
    expect(errorEntry.severity).toBe("ERROR");
    expect(errorEntry.message).toBe("Something went wrong");

    const fatalEntry = JSON.parse(stderrData[1].trim());
    expect(fatalEntry.severity).toBe("FATAL");
    expect(fatalEntry.message).toBe("System crash");
  });

  it("should format Error instances into context.error object", () => {
    const logger = createLogger({
      service: "test-service",
      module: "exception-test",
      minLevel: LogLevel.DEBUG,
      stream: customStream,
      errorStream: errorStream
    });

    const errorObj = new Error("Connection timed out");
    logger.error("Network failure", errorObj, { attempt: 3 });

    expect(stderrData.length).toBe(1);
    const parsed = JSON.parse(stderrData[0].trim());
    expect(parsed.severity).toBe("ERROR");
    expect(parsed.message).toBe("Network failure");
    expect(parsed.context.attempt).toBe(3);
    expect(parsed.context.error).toBeDefined();
    expect(parsed.context.error.name).toBe("Error");
    expect(parsed.context.error.message).toBe("Connection timed out");
    expect(typeof parsed.context.error.stack).toBe("string");
  });

  it("should support child loggers with module override and merged context", () => {
    const parent = createLogger({
      service: "minecraft-bot",
      module: "main",
      minLevel: LogLevel.DEBUG,
      stream: customStream,
      errorStream: errorStream
    });

    const child = parent.child({
      module: "watchdog",
      context: { botId: "bot-1" }
    });

    child.info("Action started", { action: "mine_block" });

    expect(stdoutData.length).toBe(1);
    const parsed = JSON.parse(stdoutData[0].trim());
    expect(parsed.service).toBe("minecraft-bot");
    expect(parsed.logger).toBe("watchdog");
    expect(parsed.severity).toBe("INFO");
    expect(parsed.message).toBe("Action started");
    expect(parsed.context).toEqual({
      botId: "bot-1",
      action: "mine_block"
    });
  });

  it("should parse log levels correctly from numbers and strings", () => {
    expect(parseLogLevel("debug")).toBe(LogLevel.DEBUG);
    expect(parseLogLevel("INFO")).toBe(LogLevel.INFO);
    expect(parseLogLevel("")).toBe(LogLevel.INFO);
    expect(parseLogLevel("   ")).toBe(LogLevel.INFO);
    expect(parseLogLevel("warn")).toBe(LogLevel.WARN);
    expect(parseLogLevel("warning")).toBe(LogLevel.WARN);
    expect(parseLogLevel("ERROR")).toBe(LogLevel.ERROR);
    expect(parseLogLevel("critical")).toBe(LogLevel.FATAL);
    expect(parseLogLevel("fatal")).toBe(LogLevel.FATAL);
    expect(parseLogLevel("30")).toBe(30);
    expect(parseLogLevel(40)).toBe(40);
  });

  it("should dynamically update log level with setLevel", () => {
    const logger = createLogger({
      service: "test-service",
      module: "dyn-test",
      minLevel: LogLevel.INFO,
      stream: customStream,
      errorStream: errorStream
    });

    logger.debug("hidden debug");
    expect(stdoutData.length).toBe(0);

    logger.setLevel?.("debug");
    logger.debug("visible debug");
    expect(stdoutData.length).toBe(1);

    logger.setLevel?.("");
    logger.debug("hidden debug again");
    expect(stdoutData.length).toBe(1);
  });

  it("should fallback to console.log and console.error when process.stdout is not available", () => {
    const originalStdout = process.stdout;
    const originalStderr = process.stderr;

    const consoleLogs: string[] = [];
    const consoleErrors: string[] = [];
    const consoleLogSpy = spyOn(console, "log").mockImplementation((msg: any) => {
      consoleLogs.push(String(msg));
    });
    const consoleErrorSpy = spyOn(console, "error").mockImplementation((msg: any) => {
      consoleErrors.push(String(msg));
    });

    try {
      // Temporarily simulate browser-like environment
      (process as any).stdout = undefined;
      (process as any).stderr = undefined;

      const logger = createLogger({
        service: "stream-overlay",
        module: "ui",
        minLevel: LogLevel.DEBUG
      });

      logger.info("Overlay loaded");
      logger.error("Socket error");

      expect(consoleLogs.length).toBe(1);
      const parsedLog = JSON.parse(consoleLogs[0]);
      expect(parsedLog.service).toBe("stream-overlay");
      expect(parsedLog.message).toBe("Overlay loaded");
      expect(parsedLog.severity).toBe("INFO");

      expect(consoleErrors.length).toBe(1);
      const parsedErr = JSON.parse(consoleErrors[0]);
      expect(parsedErr.service).toBe("stream-overlay");
      expect(parsedErr.message).toBe("Socket error");
      expect(parsedErr.severity).toBe("ERROR");
    } finally {
      (process as any).stdout = originalStdout;
      (process as any).stderr = originalStderr;
      consoleLogSpy.mockRestore();
      consoleErrorSpy.mockRestore();
    }
  });

  it("should promote trace_id and span_id to root LogEntry fields", () => {
    const logger = createLogger({
      service: "minecraft-bot",
      module: "sensors",
      stream: customStream,
      errorStream: errorStream
    });

    logger.info("Threat detected", {
      trace_id: "4bf92f3577b34da6a3ce929d0e0e4736",
      span_id: "00f067aa0ba902b7",
      mob: "creeper",
      distance: 3.5
    });

    expect(stdoutData.length).toBe(1);
    const parsed = JSON.parse(stdoutData[0]);
    expect(parsed.trace_id).toBe("4bf92f3577b34da6a3ce929d0e0e4736");
    expect(parsed.span_id).toBe("00f067aa0ba902b7");
    expect(parsed.context.mob).toBe("creeper");
  });

  it("should direct all output to single stream when errorStream is not provided", () => {
    const lines: string[] = [];
    const logger = createLogger({
      service: "stream-test",
      stream: { write: (chunk) => lines.push(chunk) }
    });

    logger.info("Normal message");
    logger.error("Error message");

    expect(lines.length).toBe(2);
    expect(JSON.parse(lines[0].trim()).severity).toBe("INFO");
    expect(JSON.parse(lines[1].trim()).severity).toBe("ERROR");
  });

  it("should inherit streams across multiple child logger generations", () => {
    const parent = createLogger({
      service: "parent-service",
      module: "root",
      stream: customStream,
      errorStream: errorStream
    });

    const child = parent.child({ module: "child" });
    const grandChild = child.child({ module: "grandchild", context: { generation: 3 } });

    child.info("Child log");
    grandChild.error("Grandchild error");

    expect(stdoutData.length).toBe(1);
    expect(stderrData.length).toBe(1);
    expect(JSON.parse(stdoutData[0].trim()).logger).toBe("child");
    const grandParsed = JSON.parse(stderrData[0].trim());
    expect(grandParsed.logger).toBe("grandchild");
    expect(grandParsed.context.generation).toBe(3);
  });
});
