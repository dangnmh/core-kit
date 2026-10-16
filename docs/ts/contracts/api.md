# TypeScript / Bun Telemetry API Contract (`ts/logger-ts`)

**Document ID**: `CONTRACT-TS-API-01`  
**Package**: `@dangnmh/logger-ts`  
**Runtime**: Bun 1.1+ / Node.js 20+ (Zero external dependencies)  
**Status**: `FROZEN CONTRACT`  
**Related ADR**: [`docs/adr/ADR-001-unified-polyglot-logging.md`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md)

---

## 1. Exported Module Interface

### 1.1 Types and Enums

```typescript
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
  /** Custom destination stream; defaults to process.stdout */
  stream?: { write: (chunk: string) => void };
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
```

### 1.2 Factory & Helper Functions

```typescript
export function createLogger(options: LoggerOptions): ILogger;
export function parseLogLevel(level?: string | number): LogLevel;
export class Logger implements ILogger { ... }
```

---

## 2. JSON Serialization Contract (Single-Line RFC3339)

Logs are formatted as single-line JSON strings followed by `\n`. Output is directed to `process.stdout` (for `DEBUG`, `INFO`, `WARN`) and `process.stderr` (for `ERROR`, `FATAL`), or the injected `stream`.

### 2.1 Schema Definition

| JSON Key | Type | Mandatory? | Value Description / Invariants |
| :--- | :--- | :---: | :--- |
| `timestamp` | `string` | **YES** | ISO-8601 UTC timestamp (`new Date().toISOString()`). |
| `severity` | `string` | **YES** | Upper-case string: `"DEBUG"`, `"INFO"`, `"WARN"`, `"ERROR"`, `"FATAL"`. |
| `service` | `string` | **YES** | Top-level service identifier passed in `LoggerOptions.service`. |
| `logger` | `string` | **YES** | Subsystem/module name (`LoggerOptions.module`). Defaults to empty string or component name. |
| `message` | `string` | **YES** | Primary log message text. Escaped for valid JSON single-line representation. |
| `caller` | `string` | *Optional* | Optional caller location. |
| `trace_id` | `string` | *Optional* | Extracted from `context.trace_id` if present and hoisted to root. |
| `span_id` | `string` | *Optional* | Extracted from `context.span_id` if present and hoisted to root. |
| `context` | `object` | *Optional* | Merged context object combining parent base context and call-time context. |

### 2.2 Canonical JSON Example

```json
{"timestamp":"2026-10-16T15:04:05.123Z","severity":"INFO","service":"gateway","logger":"http","message":"Inbound HTTP request received","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","context":{"path":"/api/v1/orders","method":"POST","ip":"10.0.4.12"}}
```

---

## 3. Child Logger & Context Inheritance Invariants

1. **Context Merge Precedence**:
   - Calling `logger.child({ module: "ws", context: { connection_id: "c-123" } })` creates a new `ILogger`.
   - Any log call on the child merges `{ ...parentBaseContext, ...childBaseContext, ...callContext }`.
2. **Module Name Inheritance**:
   - If `childOptions.module` is omitted, the child inherits the parent's `module`.
   - If provided, the child's `module` replaces the parent's `module` in emitted `"logger"` fields.

---

## 4. Test Runner & Stream Isolation Invariant

- **Audit Resolution (AUDIT-06 / AC-2.4.1)**:
  - Unit tests MUST NOT spy on and silence global `process.stdout.write` during test suite runs.
  - Test suites inject an isolated writable stream (`stream: { write: (chunk) => lines.push(chunk) }`) into the constructor.
  - This guarantees that Bun's native test reporter (`bun test`) outputs real-time test progress directly to the terminal with 100% transparency.
