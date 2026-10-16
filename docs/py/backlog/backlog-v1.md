# Component Backlog: Python Foundation (`py/logger-py`)

**Document ID**: `BACKLOG-PY-V1`  
**Status**: `⏸️ Backlog / Deferred (HADS **[?]**)`  
**Component**: Python Telemetry & Logging (`py/logger-py`)  
**Parent PRD**: [`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md)  
**Last Updated**: 2026-10-16  

---

## 1. Overview & HADS Isolation Rule

In accordance with the **BMAD 4-Layer Monorepo Documentation Standard (HADS 1.0)**, this document isolates all deferred, future, and unbuilt proposals (`**[?]**`) for the Python logging library (`py/logger-py`). These items are explicitly **OUT OF SCOPE** for the current zero-to-prod cycle to protect the Living Truth component documentation in [`docs/py/README.md`](file:///home/dangnmh/work/core-kit/docs/py).

---

## 2. Deferred Proposals & Architectural Items

### 2.1 **[?]** `logger_py.asgi`: FastAPI & Litestar ASGI Middleware
- **Feature ID**: `BK-03`
- **Priority**: P2 / Phase 3
- **Motivation**: While `py/logger-py` formats logs emitted by standard Python loggers, Web framework requests in FastAPI/Litestar require inbound/outbound HTTP request logging with automatic W3C traceparent extraction, request duration tracking, and client IP extraction.
- **Proposed Scope**:
  - Pure ASGI middleware (`ASGIMiddleware`) compatible with Starlette, FastAPI, and Litestar.
  - Automatically read `traceparent` and `x-request-id` headers and inject them into Python `contextvars`.
  - Log inbound HTTP requests and outbound responses (status code, duration_ms) conforming to the canonical JSON log schema.
  - Configurable path exclusion (e.g. skip logging `/healthz`, `/metrics`).
- **Revisit Trigger**: When Python backend microservices expose public REST endpoints requiring automated request access logs.

---

### 2.2 **[?]** Non-Blocking Async QueueHandler / QueueListener Architecture
- **Feature ID**: `BK-08`
- **Priority**: P2 / Phase 3
- **Motivation**: High-frequency logging in compute-heavy AI inference pipelines (e.g. streaming LLM token generation or TTS audio synthesis) can cause GIL contention or I/O latency spikes if writing synchronously to `sys.stdout`.
- **Proposed Scope**:
  - Provide an optional `setup_async_logging` helper utilizing Python standard library `logging.handlers.QueueHandler` and `QueueListener`.
  - Decouple worker thread logging from disk/stdout I/O by pushing records to an in-memory queue drained by a dedicated background listener thread.
  - Implement graceful shutdown hook to flush remaining queue entries before process termination.
- **Revisit Trigger**: When Python audio synthesis or inference throughput benchmarks indicate stdout I/O bottlenecking (> 5% CPU wait).

---

### 2.3 **[?]** Automated PyPI Publishing & Wheel Packaging Pipeline
- **Feature ID**: `BK-09`
- **Priority**: P2 / Phase 3
- **Motivation**: `logger-py` is currently consumed via local workspace paths or editable git references (`uv add --editable ../core-kit/py/logger-py`). External microservices or standalone repositories will eventually need standard versioned wheels.
- **Proposed Scope**:
  - Configure `pyproject.toml` with `hatchling` or `flit-core` build backend.
  - Automated GitHub Actions workflow to build source distributions (`sdist`) and pure-Python wheels (`.whl`).
  - Automated publishing to private internal Artifactory / GCP Artifact Registry upon semantic release tagging (`vX.Y.Z`).
- **Revisit Trigger**: When an external repository without monorepo filesystem access requires `logger-py`.
