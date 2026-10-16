# Sprint 1 Task: Python Telemetry Parity & Ruff Modernization (`py/logger-py`)

**Task ID**: `TASK-PY-SPRINT-1`  
**Assigned Specialist Agent**: `08b-senior-python-dev` (Model: `flash`)  
**Parent Story**: [`docs/bmad/stories/story-02-python-logger-parity-and-ruff.md`](file:///home/dangnmh/work/core-kit/docs/bmad/stories/story-02-python-logger-parity-and-ruff.md)  
**Component Root**: [`py/logger-py/`](file:///home/dangnmh/work/core-kit/py/logger-py)  
**Component Manifest**: [`py/logger-py/pyproject.toml`](file:///home/dangnmh/work/core-kit/py/logger-py/pyproject.toml) (Python 3.10–3.14+)  
**Contracts**: [`docs/py/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/api.md), [`docs/py/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/events-and-data.md)  
**Status**: `COMPLETED`  

---

## 1. Constitution Gate Check (`CONST-01..08` & `CONST-SEC-01..06`)

- [x] **`CONST-01` (Zero Storage)**: Pure in-memory streaming; zero disk database.
- [x] **`CONST-03` (Unified Telemetry Parity)**: Single-line RFC3339 JSON with root keys `timestamp`, `severity`, `service`, `logger`, `message`, `caller`, and hoisted `trace_id`/`span_id`.
- [x] **`CONST-SEC-03` (Zero-Leak Telemetry)**: No raw credentials in context or messages.
- [x] **`CONST-SEC-06` (Log Injection Immunity)**: Strict JSON serialization escaping newlines and control codes.

---

## 2. OpenSpec Capability Delta

```diff
ADDED:
+ pyproject.toml: [tool.ruff] linting & format section
+ pyproject.toml: [tool.pytest.ini_options] section
+ pyproject.toml: optional-dependencies dev = ["pytest>=8.0.0", "ruff>=0.8.0"]
+ logger_py/formatter.py: Root extraction of trace_id and span_id

MODIFIED:
! logger_py/formatter.py: Ruff code style compliance & formatException handling
! tests/test_logger.py: Added test_json_formatter_hoists_trace_ids

REMOVED:
- 14 static lint warnings reported by Ruff
```

---

## 3. EARS Acceptance Criteria

- **`AC-PY-1.1`**: `WHEN` a record contains `trace_id` or `span_id` (in extra or context), `THE` `JsonFormatter` `SHALL` emit them as top-level JSON fields and remove them from `context`.
- **`AC-PY-1.2`**: `WHEN` extra fields are present, `THE` `JsonFormatter` `SHALL` place them under `"context"`.
- **`AC-PY-1.3`**: `WHEN` `uv run ruff check .` is executed in `py/logger-py`, `THE` command `SHALL` exit with 0 warnings.
- **`AC-PY-1.4`**: `WHEN` `uv run pytest tests -v` is executed in `py/logger-py`, `THE` command `SHALL` exit with code 0 (100% tests passing).

---

## 4. Exact File Paths & Lines to Modify

1. **`py/logger-py/pyproject.toml:1-17`**: Replace with full configuration containing `[tool.ruff]`, `[tool.pytest.ini_options]`, and `dev` dependencies.
2. **`py/logger-py/logger_py/formatter.py:101-133`**: Implement root `trace_id` and `span_id` hoisting before serializing `payload`.
3. **`py/logger-py/tests/test_logger.py:10-40`**: Add test case `test_json_formatter_hoists_trace_ids`.

---

## 5. Mandatory Libraries & Verified API Cheat-Sheet

### 5.1 Trace Hoisting Snippet (`logger_py/formatter.py`)
```python
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

## 6. Step-by-Step TDD Checkboxes

- [x] **Step 1**: Write test in `py/logger-py/tests/test_logger.py` for `trace_id` and `span_id` hoisting.
- [x] **Step 2**: Run `uv run pytest tests` to verify test fails.
- [x] **Step 3**: Implement hoisting in `logger_py/formatter.py`.
- [x] **Step 4**: Verify test passes: `uv run pytest tests -v`.
- [x] **Step 5**: Update `pyproject.toml` with `[tool.ruff]` and `[tool.pytest.ini_options]`.
- [x] **Step 6**: Run `uv run ruff check --fix .` and `uv run ruff format .`.
- [x] **Step 7**: Verify full suite: `uv run ruff check .` and `uv run pytest tests -v`.

---

## 🔗 Downstream Agent Handoff

- **Target Agent**: `12-code-reviewer` & `13-qa-tester`
- **Completed Deliverables**:
  - `py/logger-py/pyproject.toml`: Modernized with `[tool.ruff]`, `[tool.pytest.ini_options]`, and `dev` dependencies.
  - `py/logger-py/logger_py/formatter.py`: Implemented top-level `trace_id` and `span_id` hoisting, clean context preservation, and PEP 585/604 type hints.
  - `py/logger-py/logger_py/__init__.py`: Cleaned and sorted exports (`__all__`).
  - `py/logger-py/tests/test_logger.py`: Added trace hoisting test cases; 10/10 tests green.
- **Verification Commands & Evidence**:
  - `uvx ruff check py/logger-py` -> 0 errors / 0 warnings.
  - `uvx ruff format --check py/logger-py` -> 0 format discrepancies.
  - `uv run --with pytest pytest py/logger-py/tests -v` -> 10 passed in 0.04s.

