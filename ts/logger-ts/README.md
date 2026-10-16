# TypeScript & Bun Telemetry Library (`ts/logger-ts`)
**Version 1.0.0** · Telemetry & Observability Team · 2026-10-16 · Standard: HADS 1.0.0

---

## AI READING INSTRUCTION

Read `**[SPEC]**` blocks for exported interfaces, stream redirection, and Bun verification commands.
Read `**[BUG]**` blocks before modifying stream routing, log level weights, or child logger factories.
Read `**[NOTE]**` blocks for cross-runtime correlation context.

---

## 1. COMPONENT SPECIFICATION & ENTRYPOINTS

**[SPEC]**
- **Root Path**: `ts/logger-ts/` (Package: `@dangnmh/logger-ts`)
- **Runtime**: Bun 1.1+ / Node.js 20+ (Zero runtime dependencies, strict TypeScript)
- **Verification Commands**:
  - `bun test` (11/11 tests pass with full native reporter output)
  - `bun run lint` (ESLint 9 Flat Config via `eslint.config.mjs`, 0 errors)
  - `bun run format:check`
- **Canonical Schema**: Emits single-line RFC3339 JSON:
  - Root fields: `timestamp` (UTC ISO-8601 with ms), `severity`, `service`, `logger`, `message`, `caller` (optional).
  - Trace hoisting: `trace_id` and `span_id` hoisted to root when present in context (`CONST-03`).
  - Severity routing: `ERROR` and `FATAL` route to `errorStream` (default `stderr`); `DEBUG`, `INFO`, `WARN` route to `stream` (default `stdout`).
- **Core Entrypoints**:
  - `createLogger(options: LoggerOptions): ILogger`: Root logger factory.
  - `logger.child(options: ChildLoggerOptions): ILogger`: Spawns child logger inheriting parent stream and base context.
  - `LogLevel`: Enum (`DEBUG=10`, `INFO=20`, `WARN=30`, `ERROR=40`, `FATAL=50`).

---

## 2. KNOWN PITFALLS & BUG MEMORY

**[BUG] Global stdout Monkey-Patching Suppressing Test Runner**
- **Symptom**: `bun test` output suppresses suite reporting, or concurrent test output is dropped.
- **Cause**: Tests used `spyOn(process.stdout, "write")` which globally intercepted the runtime stdout.
- **Fix**: Replaced monkey-patching with dependency-injected memory streams via `createLogger({ stream })` (`AC-TS-3.1`).

**[BUG] Distributed Trace Disconnection from Nested Context**
- **Symptom**: Trace correlation fails across distributed Go-TS pipelines in Jaeger / Grafana Tempo.
- **Cause**: Trace IDs were trapped within the nested `context` object rather than root level.
- **Fix**: `Logger.write` detects `trace_id` and `span_id`, promoting them to root JSON keys (`AC-TS-3.3`).

---

## 3. PUBLIC CONTRACTS & ARCHITECTURE MAP

**[SPEC]**
- Feature Registry & Matrix: [`docs/ts/README.md`](file:///home/dangnmh/work/core-kit/docs/ts/README.md)
- Internal Architecture & Stream Dispatcher: [`docs/ts/architecture.md`](file:///home/dangnmh/work/core-kit/docs/ts/architecture.md)
- Public API Specification: [`docs/ts/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/api.md)
- Schema Events & Environment Contract: [`docs/ts/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/events-and-data.md)
- Sprint Tasks & Deltas: [`docs/ts/tasks/sprint-1-logger-and-eslint.md`](file:///home/dangnmh/work/core-kit/docs/ts/tasks/sprint-1-logger-and-eslint.md)
- Deferred Feature Backlog: [`docs/ts/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/ts/backlog/backlog-v1.md)
