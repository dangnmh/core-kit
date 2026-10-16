# Python Telemetry API Contract (`py/logger-py`)

**Document ID**: `CONTRACT-PY-API-01`  
**Package**: `logger-py` (import `logger_py`)  
**Runtime**: Python 3.10+ / Python 3.14+  
**Status**: `FROZEN CONTRACT`  
**Related ADR**: [`docs/adr/ADR-001-unified-polyglot-logging.md`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md)

---

## 1. Exported Module Interface

### 1.1 Core Functions

```python
def setup_logging(
    service_name: str,
    level: Union[str, int] = "INFO",
    stream: Optional[Any] = None,
) -> logging.Logger:
    """
    Configures the root logger with a StreamHandler using JsonFormatter.
    Clears preexisting handlers to avoid duplicate output.
    
    :param service_name: Logical service identifier (e.g., 'ai-worker').
    :param level: Minimum logging level ('DEBUG', 'INFO', 'WARN', 'ERROR', or logging.* constants).
    :param stream: Target writable stream (defaults to sys.stdout).
    :return: The configured root logger.
    """

def get_logger(
    name: str,
    service: Optional[str] = None,
) -> Union[logging.Logger, ServiceLoggerAdapter]:
    """
    Returns a standard logging.Logger instance or wraps it in ServiceLoggerAdapter
    if an explicit service name is specified.
    
    :param name: Subsystem or module name (mapped to 'logger' in output JSON).
    :param service: Optional service name override.
    :return: Logger instance ready for structured emission.
    """
```

### 1.2 Formatter & Adapter Classes

```python
class JsonFormatter(logging.Formatter):
    """
    Serializes standard logging.LogRecord objects into single-line RFC3339 JSON.
    """
    def __init__(
        self,
        service_name: str = "app",
        datefmt: Optional[str] = None,
    ) -> None: ...

    def format(self, record: logging.LogRecord) -> str: ...

class ServiceLoggerAdapter(logging.LoggerAdapter):
    """
    LoggerAdapter that binds service name and automatically merges extra dicts.
    """
    def __init__(self, logger: logging.Logger, service: str) -> None: ...
    def process(self, msg: Any, kwargs: Dict[str, Any]) -> tuple[Any, Dict[str, Any]]: ...
```

---

## 2. JSON Serialization Contract (Single-Line RFC3339)

Every formatted record MUST be emitted as a single-line, UTF-8 JSON object ending with `\n`.

### 2.1 Schema Definition

| JSON Key | Type | Mandatory? | Value Description / Invariants |
| :--- | :--- | :---: | :--- |
| `timestamp` | `string` | **YES** | ISO-8601 UTC with millisecond precision and trailing `Z` (`YYYY-MM-DDTHH:MM:SS.mmmZ`). |
| `severity` | `string` | **YES** | Upper-case level: `DEBUG`, `INFO`, `WARNING` (or `WARN`), `ERROR`, `CRITICAL`. |
| `service` | `string` | **YES** | Service name configured during initialization (e.g. `core-worker`). |
| `logger` | `string` | **YES** | Logger hierarchy name (`record.name`). Corresponds to Go `logger` attribute. |
| `message` | `string` | **YES** | Formatted message string with all unicode emojis and decorative symbols stripped. |
| `caller` | `string` | **YES** | File and line location formatted as `filename:lineno` (e.g. `worker.py:45`). |
| `trace_id` | `string` | *Optional* | 32-hex W3C trace ID extracted from record extra or traceparent context. |
| `span_id` | `string` | *Optional* | 16-hex W3C span ID extracted from record extra or traceparent context. |
| `context` | `object` | *Optional* | Key-value dictionary containing contextual business metadata. |
| `exception` | `string` | *Optional* | Formatted traceback string if `exc_info` is present. |

### 2.2 Canonical JSON Example

```json
{"timestamp":"2026-10-16T15:04:05.145Z","severity":"INFO","service":"ai-worker","logger":"inference","message":"Parsed structured response cleanly","caller":"worker.py:45","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","context":{"model":"gpt-4o","tokens":142,"temperature":0.2}}
```

---

## 3. Emoji Stripping & Text Sanitization Invariant

The formatter enforces a strict anti-slop invariant via `_strip_emojis`:
- Strips ranges `\U0001F000-\U0001FAFF`, `\u2600-\u27BF`, `\u2300-\u23FF`, `\u2B50`, `\uFE0F`, `\u200D`, and symbols `\u203C\u2049\u2139`.
- Preserves intentional surrounding whitespace (e.g. `"hello 🚀 world"` becomes `"hello  world"`, `"🚀start"` becomes `"start"`).
- Guarantees terminal column alignment and regex parsing safety in downstream log pipelines.

---

## 4. W3C Trace Context Extraction Contract

When log calls include trace context:
```python
logger.info(
    "Processing batch",
    extra={
        "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
        "span_id": "00f067aa0ba902b7",
        "batch_size": 50,
    }
)
```
1. `trace_id` and `span_id` are hoisted to the root JSON object for index correlation.
2. The remaining extra keys (`batch_size`) are encapsulated within `"context"`.
