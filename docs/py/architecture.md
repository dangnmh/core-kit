# Python Telemetry Architecture & Formatting Pipeline (`docs/py/architecture.md`)
**Version 1.0.0** · Telemetry & Observability Team · 2026-10-16 · Standard: HADS 1.0.0

---

## AI READING INSTRUCTION

Read `**[SPEC]**` blocks for internal subsystem boundaries, record formatting flows, and trace hoisting.
Read `**[BUG]**` blocks before modifying regex filters, record dictionary extraction, or stream handlers.
Read `**[NOTE]**` blocks for cross-runtime correlation context.

---

## 1. COMPONENT INTERNAL DESIGN & PACKAGE TREE

**[SPEC]**
The Python telemetry package (`py/logger-py`) provides structured JSON logging with zero external runtime dependencies:

```text
py/logger-py/
├── logger_py/
│   ├── __init__.py          # Public exports: setup_logging, get_logger, JsonFormatter, ServiceLoggerAdapter
│   └── formatter.py         # JsonFormatter, emoji stripping regex, trace hoister, exception serializer
├── tests/
│   ├── __init__.py
│   └── test_logger.py       # 10 comprehensive unit tests covering formatting, hoisting, and adapters
└── pyproject.toml           # Project manifest, Ruff linter rules, Pytest configuration
```

---

## 2. RECORD PROCESSING PIPELINE & WORKFLOW

**[SPEC]**
The `JsonFormatter` executes a deterministic transformation pipeline on each Python `logging.LogRecord`:

```mermaid
flowchart TD
    App["Application Code\nlogger.info('User action', extra={'trace_id': 'abc', 'user_id': 123})"] --> Adapter["ServiceLoggerAdapter\n(Injects service='ai-worker')"]
    Adapter --> Record["logging.LogRecord Emitted"]
    Record --> Formatter["JsonFormatter.format(record)"]
    
    subgraph Pipeline["Formatting Pipeline"]
        Emoji["Step 1: Emoji Stripping\n_EMOJI_PATTERN.sub('', msg)"]
        Caller["Step 2: Caller Tracking\nfilename:lineno extraction"]
        Trace["Step 3: W3C Trace Hoisting\nExtract trace_id & span_id to root"]
        Context["Step 4: Context Container\nBundle remaining extra kwargs into 'context'"]
        Exc["Step 5: Traceback Serialization\nformatException(record.exc_info)"]
    end

    Formatter --> Pipeline
    Pipeline --> JSON["json.dumps(payload, default=str)"]
    JSON --> Stream["sys.stdout\n(Single-line RFC3339 JSON)"]
```

---

## 3. SEQUENCE DIAGRAM: CONTEXT & TRACE PROPAGATION

**[SPEC]**
When downstream workers process distributed requests, trace IDs propagate to root level:

```mermaid
sequenceDiagram
    autonumber
    participant Worker as AI Worker / Service
    participant Logger as ServiceLoggerAdapter
    participant Fmt as JsonFormatter
    participant Output as stdout Stream

    Worker->>Logger: info("Synthesis started", extra={"trace_id": "c7ac12f0", "job_id": 99})
    Logger->>Fmt: format(LogRecord)
    Fmt->>Fmt: Hoist trace_id="c7ac12f0" to root payload
    Fmt->>Fmt: Nest {"job_id": 99} into payload["context"]
    Fmt->>Fmt: Strip emojis from message
    Fmt->>Output: Emit single-line JSON string
```

---

## 4. INVARIANTS & SECURITY BOUNDARIES

**[SPEC]**
1. **Zero Emojis (`CONST-07`)**: Unfiltered emojis or unicode symbols are stripped via regex before serialization.
2. **Schema Parity (`CONST-03`)**: Root fields strictly match Go (`pkg/logger`) and TypeScript (`ts/logger-ts`) schemas.
3. **Log Injection Immunity (`CONST-SEC-06`)**: Messages and contexts are safely serialized via `json.dumps`, preventing CRLF log injection.
