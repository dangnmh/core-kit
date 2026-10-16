# TypeScript & Bun Telemetry Component Hub (`ts/logger-ts`)
**Version 1.0.0** · Telemetry & Observability Team · 2026-10-16 · Standard: HADS 1.0.0

---

## AI READING INSTRUCTION

Read `**[SPEC]**` blocks for authoritative feature matrices, schema parity rules, and contract paths.
Read `**[BUG]**` blocks before modifying stream routing, log level weights, or child logger factories.
Read `**[NOTE]**` blocks for cross-runtime correlation context.

---

## 1. COMPONENT OVERVIEW & FEATURE MATRIX

**[SPEC]**
The TypeScript telemetry component (`ts/logger-ts`) provides zero-dependency structured JSON logging:

| Feature / Capability | Description | Status |
| :--- | :--- | :---: |
| **RFC3339 JSON Serialization** | Single-line JSON with ISO-8601 UTC timestamp and millisecond precision | `ACTIVE` |
| **Numeric Level Filtering** | Threshold filtering (`DEBUG=10`, `INFO=20`, `WARN=30`, `ERROR=40`, `FATAL=50`) | `ACTIVE` |
| **Stream Injection** | Destination routing via `LoggerOptions.stream` and `errorStream` | `ACTIVE` |
| **W3C Trace Hoisting** | `trace_id` and `span_id` detected and promoted to root JSON keys | `ACTIVE` |
| **Error Instance Formatting** | Automatic formatting of `Error` instances into `context.error` object | `ACTIVE` |
| **Child Logger Decoupling** | Hierarchical context inheritance via `logger.child({ module, context })` | `ACTIVE` |
| **Modern Toolchain** | ESLint 9 Flat Config (`eslint.config.mjs`) passing with 0 warnings | `ACTIVE` |

---

## 2. NAVIGATION HUB & LIVING CONTRACTS

**[SPEC]**
All authoritative technical contracts and specifications are organized in this canonical hub:

| Document | Description | Path |
| :--- | :--- | :--- |
| **Architecture** | Internal design, stream routing & sequence diagram | [`architecture.md`](file:///home/dangnmh/work/core-kit/docs/ts/architecture.md) |
| **API Contract** | Module entrypoints, function signatures & interfaces | [`contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/api.md) |
| **Data & Events** | JSON schema specification & environment variables | [`contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/events-and-data.md) |
| **Sprint Tasks** | Sprint 1 OpenSpec delta and EARS acceptance criteria | [`tasks/sprint-1-logger-and-eslint.md`](file:///home/dangnmh/work/core-kit/docs/ts/tasks/sprint-1-logger-and-eslint.md) |
| **Backlog** | Deferred non-P0 feature proposals and ideas | [`backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/ts/backlog/backlog-v1.md) |

---

## 3. KNOWN PITFALLS & BUG MEMORY

**[BUG] Global stdout Monkey-Patching Suppressing Test Runner**
- **Symptom**: `bun test` output suppresses suite reporting, or concurrent test output is dropped.
- **Cause**: Tests used `spyOn(process.stdout, "write")` which globally intercepted the runtime stdout.
- **Fix**: Replaced monkey-patching with dependency-injected memory streams via `createLogger({ stream })` (`AC-TS-3.1`).

**[BUG] Distributed Trace Disconnection from Nested Context**
- **Symptom**: Trace correlation fails across distributed Go-TS pipelines in Jaeger / Grafana Tempo.
- **Cause**: Trace IDs were trapped within the nested `context` object rather than root level.
- **Fix**: `Logger.write` detects `trace_id` and `span_id`, promoting them to root JSON keys (`AC-TS-3.3`).
