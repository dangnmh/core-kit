# TypeScript / Bun Telemetry Events, Data & Configuration Contract (`ts/logger-ts`)

**Document ID**: `CONTRACT-TS-DATA-01`  
**Package**: `@dangnmh/logger-ts`  
**Runtime**: Bun 1.1+ / Node.js 20+ (Zero external dependencies)  
**Status**: `FROZEN CONTRACT`  
**Related ADR**: [`docs/adr/ADR-001-unified-polyglot-logging.md`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md)  
**Parent Specification**: [`docs/bmad/05-database-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/05-database-spec.md)  

---

## 1. Zero-Storage Invariant & Stream Architecture

1. **Zero-Storage Invariant (`CONST-01`)**: `@dangnmh/logger-ts` maintains zero persistent disk storage, zero local SQLite files, and zero disk-buffered state.
2. **Stream Routing Invariant**:
   - `DEBUG`, `INFO`, and `WARN` levels write single-line JSON to `process.stdout` (or custom injected `stream`).
   - `ERROR` and `FATAL` levels write single-line JSON to `process.stderr` (or custom injected `stream`).
3. **Test Isolation Invariant (`AC-2.4.1`)**: Unit tests inject a memory stream `{ write: (chunk) => logs.push(chunk) }` into constructor options, completely avoiding global `process.stdout.write` spying so Bun test reporter output is never silenced.

---

## 2. Canonical Telemetry JSON Storage Schema (RFC3339)

Every log entry formatted by `Logger.log()` MUST strictly adhere to the unified polyglot telemetry schema (`ADR-001`):

### 2.1 Emitted Record Specification

```json
{
  "timestamp": "2026-10-16T15:04:05.123Z",
  "severity": "INFO",
  "service": "api-gateway",
  "logger": "auth",
  "message": "Bearer JWT token validated successfully",
  "caller": "auth.ts:65",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "context": {
    "user_id": "usr-88192",
    "roles": ["admin", "trader"]
  }
}
```

### 2.2 Field Invariants & Data Types

| JSON Attribute | Data Type | Nullability | Description & Formatting Invariants |
| :--- | :--- | :---: | :--- |
| `timestamp` | `string` | **NOT NULL** | UTC timestamp formatted via `new Date().toISOString()`. |
| `severity` | `string` | **NOT NULL** | Upper-case string: `"DEBUG"`, `"INFO"`, `"WARN"`, `"ERROR"`, `"FATAL"`. |
| `service` | `string` | **NOT NULL** | Top-level service identifier passed in `LoggerOptions.service`. |
| `logger` | `string` | **NOT NULL** | Module/subsystem name passed in `LoggerOptions.module`. |
| `message` | `string` | **NOT NULL** | Primary log message text. Escaped for valid JSON single-line representation. |
| `caller` | `string` | *Nullable* | Optional source code location. |
| `trace_id` | `string` | *Nullable* | 32-character hexadecimal W3C trace ID extracted from `context.trace_id`. |
| `span_id` | `string` | *Nullable* | 16-character hexadecimal W3C span ID extracted from `context.span_id`. |
| `context` | `object` | *Nullable* | Merged context object combining base context, child context, and call context. |

---

## 3. Child Logger & Context Hierarchy

Child loggers inherit and extend the parent context immutably:
1. `logger.child({ module: "ws", context: { conn_id: "c-101" } })` creates a new `Logger` instance.
2. Context merge precedence: `{ ...parentBaseContext, ...childBaseContext, ...callContext }`.
3. If `childOptions.module` is omitted, the child inherits the parent's `module` string.

---

## 4. Error Object Serialization Invariant

When an `Error` instance is passed as the first parameter or inside context:
```typescript
logger.error("Database connection dropped", err);
```
The error is automatically serialized into the `"context"` dictionary under the `"error"` key with schema:
```json
{
  "context": {
    "error": {
      "name": "ConnectionTimeoutError",
      "message": "Timed out after 5000ms",
      "stack": "ConnectionTimeoutError: Timed out...\n    at Socket.connect..."
    }
  }
}
```

---

## 5. Environment Variables Contract

| Environment Variable | Target Module | Default Value | Description / Accepted Values |
| :--- | :--- | :--- | :--- |
| `LOG_LEVEL` | `parseLogLevel` | `"INFO"` | Minimum logging threshold. Case-insensitive strings (`DEBUG`, `INFO`, `WARN`/`WARNING`, `ERROR`, `FATAL`/`CRITICAL`) or numeric levels (10, 20, 30, 40, 50). |
