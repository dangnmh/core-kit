# Python Telemetry Component Hub (`py/logger-py`)
**Version 1.0.0** · Telemetry & Observability Team · 2026-10-16 · Standard: HADS 1.0.0

---

## AI READING INSTRUCTION

Read `**[SPEC]**` blocks for authoritative feature matrices, schema parity rules, and contract paths.
Read `**[BUG]**` blocks before modifying log formatting, exception serialization, or adapters.
Read `**[NOTE]**` blocks for cross-runtime correlation context.

---

## 1. COMPONENT OVERVIEW & FEATURE MATRIX

**[SPEC]**
The Python telemetry component (`py/logger-py`) provides zero-dependency structured JSON logging:

| Feature / Capability | Description | Status |
| :--- | :--- | :---: |
| **RFC3339 JSON Formatter** | Single-line JSON with ISO-8601 UTC timestamp and trailing Z | `ACTIVE` |
| **Context Nesting** | Custom extra attributes isolated under `"context"` container object | `ACTIVE` |
| **W3C Trace Hoisting** | `trace_id` and `span_id` detected and promoted to root JSON keys | `ACTIVE` |
| **Anti-Slop Emoji Stripping** | Regex `_EMOJI_PATTERN` cleans decorative Unicode and emojis | `ACTIVE` |
| **Service Logger Adapter** | Dynamic service name binding and extra kwargs merging | `ACTIVE` |
| **Exception Serialization** | Full traceback formatting via `formatException` under `"exception"` | `ACTIVE` |
| **Modern Toolchain** | Ruff linter & formatter (E, W, F, I, B, C4, UP, RUF) with 0 warnings | `ACTIVE` |

---

## 2. NAVIGATION HUB & LIVING CONTRACTS

**[SPEC]**
All authoritative technical contracts and specifications are organized in this canonical hub:

| Document | Description | Path |
| :--- | :--- | :--- |
| **Architecture** | Internal design, record lifecycle & flowchart | [`architecture.md`](file:///home/dangnmh/work/core-kit/docs/py/architecture.md) |
| **API Contract** | Module entrypoints, function signatures & types | [`contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/api.md) |
| **Data & Events** | JSON schema specification & environment variables | [`contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/events-and-data.md) |
| **Sprint Tasks** | Sprint 1 OpenSpec delta and EARS acceptance criteria | [`tasks/sprint-1-logger-and-ruff.md`](file:///home/dangnmh/work/core-kit/docs/py/tasks/sprint-1-logger-and-ruff.md) |
| **Backlog** | Deferred non-P0 feature proposals and ideas | [`backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/py/backlog/backlog-v1.md) |

---

## 3. KNOWN PITFALLS & BUG MEMORY

**[BUG] Decorative Emojis Corrupting Log Ingestion**
- **Symptom**: Centralized log ingestors fail parsing or query regex tokenization.
- **Cause**: Application handlers log strings containing decorative Unicode or emoji characters.
- **Fix**: `JsonFormatter` applies `_EMOJI_PATTERN.sub("", msg)` before output (`CONST-07`).

**[BUG] Trace IDs Nested in Context Dictionary**
- **Symptom**: Distributed tracing tools fail to correlate Python spans with Go/TS services.
- **Cause**: Logging extra kwargs indiscriminately bundled `trace_id` and `span_id` into `context`.
- **Fix**: `JsonFormatter` hoists `trace_id`/`span_id` to root JSON keys and purges them from `context` (`AC-PY-2.1`).
