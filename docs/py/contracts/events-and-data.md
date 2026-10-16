# Python Telemetry Events, Data & Configuration Contract (`py/logger-py`)

**Document ID**: `CONTRACT-PY-DATA-01`  
**Package**: `logger-py` (import `logger_py`)  
**Runtime**: Python 3.10+ / Python 3.14+  
**Status**: `FROZEN CONTRACT`  
**Related ADR**: [`docs/adr/ADR-001-unified-polyglot-logging.md`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md)  
**Parent Specification**: [`docs/bmad/05-database-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/05-database-spec.md)  

---

## 1. Zero-Storage Invariant & Stream Architecture

1. **Zero-Storage Invariant (`CONST-01`)**: `py/logger-py` is a stateless, pure-logging telemetry library. It creates zero persistent SQLite files, zero state databases, and manages zero disk caching layers.
2. **Stream Routing**: By default, logs format into single-line UTF-8 JSON records and stream directly to `sys.stdout`. Custom destinations (e.g. `sys.stderr`, `io.StringIO`, or rotating log sinks) can be injected via the `stream` argument of `setup_logging()`.

---

## 2. Canonical Telemetry JSON Storage Schema (RFC3339)

Every formatted log line emitted by `JsonFormatter` MUST strictly conform to the unified polyglot telemetry schema (`ADR-001`):

### 2.1 Emitted Record Specification

```json
{
  "timestamp": "2026-10-16T15:04:05.145Z",
  "severity": "INFO",
  "service": "ai-worker",
  "logger": "inference",
  "message": "Model generated response cleanly",
  "caller": "worker.py:42",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "context": {
    "model": "gpt-4o",
    "prompt_tokens": 128,
    "completion_tokens": 42
  }
}
```

### 2.2 Field Invariants & Data Types

| JSON Attribute | Data Type | Nullability | Description & Formatting Invariants |
| :--- | :--- | :---: | :--- |
| `timestamp` | `string` | **NOT NULL** | UTC timestamp with millisecond precision and trailing `Z` (`YYYY-MM-DDTHH:MM:SS.mmmZ`). |
| `severity` | `string` | **NOT NULL** | Standardized level: `DEBUG`, `INFO`, `WARNING` (or `WARN`), `ERROR`, `CRITICAL`. |
| `service` | `string` | **NOT NULL** | Logical service identifier provided via `setup_logging` or overridden per logger. |
| `logger` | `string` | **NOT NULL** | Subsystem/logger hierarchy name (`record.name`). Matches Go `"logger"` attribute. |
| `message` | `string` | **NOT NULL** | Main log message. Formatted without Unicode emojis or decorative icons. |
| `caller` | `string` | **NOT NULL** | Calling code location formatted as `filename:lineno`. |
| `trace_id` | `string` | *Nullable* | 32-character hexadecimal W3C distributed trace ID hoisted from `extra`. |
| `span_id` | `string` | *Nullable* | 16-character hexadecimal W3C distributed span ID hoisted from `extra`. |
| `context` | `object` | *Nullable* | Nested key-value dictionary containing contextual parameters from `extra`. |
| `exception` | `string` | *Nullable* | Multi-line formatted traceback string when `exc_info` is provided. |

---

## 3. Context & Attribute Sanitization Invariants

### 3.1 Reserved Record Attribute Filtering
To prevent polluting the `"context"` dictionary with Python internal `logging.LogRecord` plumbing, the following keys are excluded from `context`:
```python
_RESERVED_RECORD_ATTRS = {
    "args", "asctime", "created", "exc_info", "exc_text", "filename", "funcName",
    "levelname", "levelno", "lineno", "module", "msecs", "message", "msg",
    "name", "pathname", "process", "processName", "relativeCreated", "stack_info",
    "thread", "threadName", "taskName", "service", "extra_fields", "context"
}
```

### 3.2 Anti-Slop Emoji Sanitization Invariant
The formatter applies regex `_strip_emojis` against `record.getMessage()`:
- Strips ranges `\U0001F000-\U0001FAFF`, `\u2600-\u27BF`, `\u2300-\u23FF`, `\u2B50`, `\uFE0F`, `\u200D`, and symbols `\u203C\u2049\u2139`.
- Preserves intentional leading and trailing whitespace to maintain alignment in terminal and log aggregator displays.

---

## 4. Environment Variables Contract

| Environment Variable | Target Module | Default Value | Description / Accepted Values |
| :--- | :--- | :--- | :--- |
| `LOG_LEVEL` | `logger_py.setup_logging` | `"INFO"` | Minimum logging threshold. Case-insensitive: `DEBUG`, `INFO`, `WARN` / `WARNING`, `ERROR`, `CRITICAL`. |
| `SERVICE_NAME` | `logger_py.setup_logging` | `"app"` | Default logical service identifier for emitted records. |
