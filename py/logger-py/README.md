# Python Structured Telemetry Library (`py/logger-py`)
**Version 1.0.0** · Telemetry & Observability Team · 2026-10-16 · Standard: HADS 1.0.0

---

## AI READING INSTRUCTION

Read `**[SPEC]**` blocks for schema invariants, entrypoint signatures, and linter commands.
Read `**[BUG]**` blocks before editing formatters, adapters, or exception serialization.
Read `**[NOTE]**` blocks for cross-runtime correlation context.

---

## 1. COMPONENT SPECIFICATION & ENTRYPOINTS

**[SPEC]**
- **Root Path**: `py/logger-py/` (Package import: `logger_py`)
- **Runtime**: Python 3.10 through Python 3.14+ (managed via `uv`)
- **Verification Commands**:
  - `uv run --extra dev pytest tests -v` (10/10 tests green in <50ms)
  - `uv run --extra dev ruff check .` (0 warnings with rules E, W, F, I, B, C4, UP, RUF)
  - `uv run --extra dev ruff format --check .`
- **Canonical Schema**: Single-line RFC3339 JSON object emitted to `sys.stdout`:
  - Root keys: `timestamp` (UTC ISO-8601 with trailing Z), `severity`, `service`, `logger`, `message`, `caller` (`file:line`).
  - Distributed tracing: `trace_id` and `span_id` automatically hoisted to root if present (`CONST-03`).
  - Context isolation: Non-trace extra attributes nested cleanly under `"context"` dict (`CONST-SEC-03`).
- **Core Entrypoints**:
  - `setup_logging(service_name: str, level: str = "INFO", stream: Any = None) -> logging.Logger`: Configures root logger.
  - `get_logger(name: str, service: Optional[str] = None) -> Union[Logger, ServiceLoggerAdapter]`: Instantiates structured logger.
  - `JsonFormatter(service_name: str = "app", datefmt: Optional[str] = None)`: Serializes log records.

---

## 2. KNOWN PITFALLS & BUG MEMORY

**[BUG] Decorative Emojis Corrupting Log Ingestion**
- **Symptom**: LogQL / Elasticsearch pipelines fail regex tokenization or render unreadable unicode glyphs.
- **Cause**: Application handlers log strings containing decorative Unicode or emoji characters.
- **Fix**: `JsonFormatter` applies `_EMOJI_PATTERN.sub("", msg)` to strip non-ASCII decorative symbols before emission (`CONST-07`).

**[BUG] Trace IDs Nested in Context Dictionary**
- **Symptom**: Distributed trace visualization tools fail to link Python spans with upstream Go/TS services.
- **Cause**: Logging extra kwargs indiscriminately bundled `trace_id` and `span_id` into `context`.
- **Fix**: `JsonFormatter` detects and extracts `trace_id`/`span_id` to root JSON fields and purges them from `context` (`AC-PY-2.1`).

---

## 3. PUBLIC CONTRACTS & ARCHITECTURE MAP

**[SPEC]**
- Feature Registry & Matrix: [`docs/py/README.md`](file:///home/dangnmh/work/core-kit/docs/py/README.md)
- Internal Architecture & Formatting Pipeline: [`docs/py/architecture.md`](file:///home/dangnmh/work/core-kit/docs/py/architecture.md)
- Public API Specification: [`docs/py/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/api.md)
- Schema Events & Environment Contract: [`docs/py/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/events-and-data.md)
- Sprint Tasks & Deltas: [`docs/py/tasks/sprint-1-logger-and-ruff.md`](file:///home/dangnmh/work/core-kit/docs/py/tasks/sprint-1-logger-and-ruff.md)
- Deferred Feature Backlog: [`docs/py/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/py/backlog/backlog-v1.md)
