# Story 02: Python Telemetry Parity, W3C Trace Extraction & Ruff Modernization

**Document ID**: `STORY-PY-02`  
**Parent Epic**: Epic 2 & Epic 3 ([`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md))  
**Assigned Specialist Agent**: `08b-senior-python-dev` (Senior Python Engineer)  
**Recommended Model Tier**: `flash`  
**Component Manifest**: [`py/logger-py/pyproject.toml`](file:///home/dangnmh/work/core-kit/py/logger-py/pyproject.toml) (Python 3.10–3.14+)  
**Related Contracts**: [`CONTRACT-PY-API-01`](file:///home/dangnmh/work/core-kit/docs/py/contracts/api.md), [`CONTRACT-PY-DATA-01`](file:///home/dangnmh/work/core-kit/docs/py/contracts/events-and-data.md), [`ADR-001`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md), [`ADR-003`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-003-polyglot-linter-and-ci-toolchain.md)  
**Status**: `READY FOR IMPLEMENTATION`  

---

## 1. Constitution Gate Check (`CONST-01..08` & `CONST-SEC-01..06`)

Before writing any code, the engineer MUST verify compliance with the following binding invariants:
- **`CONST-01` (Zero Storage)**: Pure in-memory streaming library; zero disk persistence, zero SQLite caching.
- **`CONST-03` (Unified Telemetry Parity)**: Emitted JSON records MUST contain root keys: `timestamp`, `severity`, `service`, `logger`, `message`, `caller`, and optionally `trace_id`, `span_id`, `context`, `exception`.
- **`CONST-SEC-03` (Zero-Leak Telemetry)**: Sensitive fields in `context` or `extra` must not expose raw auth tokens or secrets.
- **`CONST-SEC-06` (Log Injection Immunity)**: All string payloads must be serialized via `json.dumps()` ensuring clean escaping of newlines (`\r`, `\n`) and ASCII control characters.

---

## 2. OpenSpec Capability Delta

```diff
ADDED:
+ pyproject.toml [tool.ruff]: Comprehensive Ruff 0.8+ linting and formatting configuration (E, W, F, I, B, C4, UP, RUF).
+ pyproject.toml [tool.pytest.ini_options]: Configured testpaths = ["tests"] and pythonpath = ["."].
+ pyproject.toml [project.optional-dependencies]: dev = ["pytest>=8.0.0", "ruff>=0.8.0"].
+ logger_py/formatter.py: Root trace extraction: hoist trace_id and span_id from record extras or context to root JSON payload.

MODIFIED:
! logger_py/formatter.py: Clean up typing and imports to satisfy Ruff rules; ensure formatException newline handling.
! tests/test_logger.py: Add test cases verifying root trace_id and span_id extraction and emoji stripping.

REMOVED:
- 14 static lint warnings reported by Ruff in formatter.py and test_logger.py.
```

---

## 3. EARS Acceptance Criteria

- **`AC-PY-2.1` (Trace ID Hoisting)**:
  `WHEN` a log call includes `extra={"trace_id": "4bf92f3577b34da6a3ce929d0e0e4736", "span_id": "00f067aa0ba902b7"}`
  `THE` `JsonFormatter` `SHALL` place `"trace_id"` and `"span_id"` at the top level of the emitted JSON object, matching Go and TypeScript.
- **`AC-PY-2.2` (Context Dictionary Isolation)**:
  `WHEN` extra fields other than `trace_id` and `span_id` are provided
  `THE` `JsonFormatter` `SHALL` encapsulate them inside the `"context"` dictionary without polluting the root namespace.
- **`AC-PY-2.3` (Anti-Slop Emoji Stripping)**:
  `WHEN` the log message contains Unicode emojis (e.g. `"Starting worker 🚀"`)
  `THE` `JsonFormatter` `SHALL` strip all emojis and emit clean text (`"Starting worker"`).
- **`AC-PY-2.4` (Ruff Zero-Warning Gate)**:
  `WHEN` `uv run ruff check py/logger-py` is executed
  `THE` linter `SHALL` exit with 0 errors and 0 warnings.
- **`AC-PY-2.5` (Pytest Suite 100% Pass)**:
  `WHEN` `uv run pytest py/logger-py/tests -v` is executed
  `THE` test runner `SHALL` pass 100% with all tests green.

---

## 4. Exact File Paths & Ownership

- **Assigned Agent**: `08b-senior-python-dev`
- **Modify**: [`py/logger-py/pyproject.toml`](file:///home/dangnmh/work/core-kit/py/logger-py/pyproject.toml)
- **Modify**: [`py/logger-py/logger_py/formatter.py`](file:///home/dangnmh/work/core-kit/py/logger-py/logger_py/formatter.py)
- **Modify**: [`py/logger-py/tests/test_logger.py`](file:///home/dangnmh/work/core-kit/py/logger-py/tests/test_logger.py)

---

## 5. Mandatory Libraries & Verified API Cheat-Sheet (`from 06b-library-manifest.md`)

> **FORBIDDEN: Do NOT hand-roll custom formatting or regex parsers beyond the verified pattern below.**

### 5.1 Verified `pyproject.toml` Configuration
```toml
[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.build_meta"

[project]
name = "logger-py"
version = "0.1.0"
description = "Structured JSON Logger for Core-Kit"
readme = "README.md"
authors = [{ name = "Core-Kit Team" }]
requires-python = ">=3.10"
dependencies = []

[project.optional-dependencies]
dev = [
    "pytest>=8.0.0",
    "ruff>=0.8.0",
]

[tool.setuptools.packages.find]
where = ["."]
include = ["logger_py*"]

[tool.ruff]
line-length = 100
target-version = "py310"

[tool.ruff.lint]
select = [
    "E",    # pycodestyle errors
    "W",    # pycodestyle warnings
    "F",    # pyflakes
    "I",    # isort
    "B",    # flake8-bugbear
    "C4",   # flake8-comprehensions
    "UP",   # pyupgrade
    "RUF",  # Ruff-specific rules
]
ignore = [
    "UP006", # allow typing.Dict in legacy modules
]

[tool.ruff.format]
quote-style = "double"
indent-style = "space"

[tool.pytest.ini_options]
testpaths = ["tests"]
pythonpath = ["."]
```

### 5.2 W3C Trace Extraction Logic (`logger_py/formatter.py`)
```python
# In JsonFormatter.format(self, record: logging.LogRecord) -> str:
payload: Dict[str, Any] = {
    "timestamp": timestamp,
    "severity": record.levelname,
    "service": service,
    "logger": record.name,
    "message": message,
    "caller": caller,
}

# Hoist trace_id and span_id if present in record
trace_id = getattr(record, "trace_id", None)
span_id = getattr(record, "span_id", None)

context = None
if hasattr(record, "extra_fields") and record.extra_fields is not None:
    context = dict(record.extra_fields)
elif hasattr(record, "context") and record.context is not None:
    context = dict(record.context)
else:
    extras = {
        k: v
        for k, v in record.__dict__.items()
        if k not in _RESERVED_RECORD_ATTRS and not k.startswith("_")
    }
    if extras:
        context = extras

if context:
    if not trace_id and "trace_id" in context:
        trace_id = context.pop("trace_id")
    if not span_id and "span_id" in context:
        span_id = context.pop("span_id")

if trace_id:
    payload["trace_id"] = str(trace_id)
if span_id:
    payload["span_id"] = str(span_id)

if context:
    payload["context"] = context
```

---

## 6. Step-by-Step TDD Implementation Plan

- [ ] **Step 1: Write failing test in `py/logger-py/tests/test_logger.py`**:
  ```python
  def test_json_formatter_hoists_trace_ids():
      formatter = JsonFormatter(service_name="test-worker")
      record = logging.LogRecord(
          name="inference",
          level=logging.INFO,
          pathname="worker.py",
          lineno=42,
          msg="Execution completed",
          args=(),
          exc_info=None,
      )
      record.trace_id = "4bf92f3577b34da6a3ce929d0e0e4736"
      record.span_id = "00f067aa0ba902b7"
      record.extra_fields = {"tokens": 120}

      line = formatter.format(record)
      data = json.loads(line)
      assert data["trace_id"] == "4bf92f3577b34da6a3ce929d0e0e4736"
      assert data["span_id"] == "00f067aa0ba902b7"
      assert data["context"] == {"tokens": 120}
      assert "trace_id" not in data.get("context", {})
  ```
- [ ] **Step 2: Run pytest to verify failure**:
  `uv run pytest py/logger-py/tests -k test_json_formatter_hoists_trace_ids`
- [ ] **Step 3: Update `logger_py/formatter.py`**:
  Implement trace ID hoisting and context extraction from Section 5.2.
- [ ] **Step 4: Update `py/logger-py/pyproject.toml`**:
  Add `[tool.ruff]`, `[tool.pytest.ini_options]`, and dev dependencies.
- [ ] **Step 5: Run Ruff static analysis**:
  `cd py/logger-py && uv run ruff check --fix . && uv run ruff format .`
- [ ] **Step 6: Run full pytest suite**:
  `cd py/logger-py && uv run pytest tests -v` (100% passing).

---

## 🔗 Downstream Agent Handoff

- **Target Agent**: `08b-senior-python-dev`
- **Mandatory Sections**: Section 4 (Files), Section 5 (API Snippets), Section 6 (TDD Checkboxes)
- **Expected Deliverable**: `py/logger-py` passing `uv run ruff check` and `uv run pytest` with 100% green tests.
