# Feature Capability & Traceability Matrix (`docs/06-feature-matrix-and-roadmap.md`)
**Version 1.0.0** · Principal Project Librarian & PMO Architect · 2026-10-16 · Standard: BMAD HADS 1.0.0

---

## 1. Executive Summary & Traceability Model

This document establishes the authoritative, bidirectional traceability matrix linking:
**Business Value & Requirements** $\longleftrightarrow$ **Component & Source Implementation** $\longleftrightarrow$ **Public Contracts** $\longleftrightarrow$ **Automated Test Suites** $\longleftrightarrow$ **Verified Release Status**.

### Status Legend
- `✅ Verified`: 100% implemented, passing tests with race detection, conforming to contracts, and verified in CI.
- `🔄 In-Progress`: Active sprint scope under implementation.
- `⏸️ Backlog`: Formally scoped RFC / proposal deferred to component-specific backlog (`docs/<comp>/backlog/`).

---

## 2. Core Feature & Capability Matrix

| Feature ID | Business Goal & Capability | Component & Source File | Authoritative Contract | Automated Test File | Verified Status |
| :--- | :--- | :--- | :--- | :--- | :---: |
| **DOC-01** | Layer 0 Root Constitution & AI Sitemap | [`AGENTS.md`](file:///home/dangnmh/work/core-kit/AGENTS.md), [`llms.txt`](file:///home/dangnmh/work/core-kit/llms.txt) | [`AGENTS.md`](file:///home/dangnmh/work/core-kit/AGENTS.md) | `scripts/doc-lint.sh` | `✅ Verified` |
| **DOC-02** | Layer 1 Component Nearest-Wins Guardrails | [`pkg/README.md`](file:///home/dangnmh/work/core-kit/pkg/README.md), [`py/logger-py/README.md`](file:///home/dangnmh/work/core-kit/py/logger-py/README.md), [`ts/logger-ts/README.md`](file:///home/dangnmh/work/core-kit/ts/logger-ts/README.md) | [`docs/README.md`](file:///home/dangnmh/work/core-kit/docs/README.md) | `scripts/doc-lint.sh` | `✅ Verified` |
| **DOC-03** | Layer 2 Living System Truth & Contracts | [`docs/bmad/04-architecture.md`](file:///home/dangnmh/work/core-kit/docs/bmad/04-architecture.md), [`docs/adr/`](file:///home/dangnmh/work/core-kit/docs/adr), [`docs/runbooks/`](file:///home/dangnmh/work/core-kit/docs/runbooks) | `docs/{pkg,py,ts}/contracts/*` | `scripts/doc-lint.sh` | `✅ Verified` |
| **DOC-04** | Automated CI Doc-Lint Gate | [`scripts/doc-lint.sh`](file:///home/dangnmh/work/core-kit/scripts/doc-lint.sh), [`Makefile`](file:///home/dangnmh/work/core-kit/Makefile) | [`AGENTS.md`](file:///home/dangnmh/work/core-kit/AGENTS.md) | `make doc-lint` | `✅ Verified` |
| **TEL-01** | Slog dual-mode JSON logging with context nesting & W3C trace correlation | [`pkg/logger/logger.go`](file:///home/dangnmh/work/core-kit/pkg/logger/logger.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/logger/logger_test.go`](file:///home/dangnmh/work/core-kit/pkg/logger/logger_test.go) | `✅ Verified` |
| **TEL-02** | Python structured JSON logging with emoji stripping & trace hoisting | [`py/logger-py/logger_py/formatter.py`](file:///home/dangnmh/work/core-kit/py/logger-py/logger_py/formatter.py) | [`docs/py/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/api.md) | [`py/logger-py/tests/test_logger.py`](file:///home/dangnmh/work/core-kit/py/logger-py/tests/test_logger.py) | `✅ Verified` |
| **TEL-03** | TypeScript structured JSON logging with stream routing & trace hoisting | [`ts/logger-ts/src/index.ts`](file:///home/dangnmh/work/core-kit/ts/logger-ts/src/index.ts) | [`docs/ts/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/api.md) | [`ts/logger-ts/tests/logger.test.ts`](file:///home/dangnmh/work/core-kit/ts/logger-ts/tests/logger.test.ts) | `✅ Verified` |
| **TEL-04** | OpenTelemetry W3C traceparent carrier injection & trace context | [`pkg/tracer/tracer.go`](file:///home/dangnmh/work/core-kit/pkg/tracer/tracer.go), [`pkg/tracectx/tracectx.go`](file:///home/dangnmh/work/core-kit/pkg/tracectx/tracectx.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/tracer/tracer_test.go`](file:///home/dangnmh/work/core-kit/pkg/tracer/tracer_test.go) | `✅ Verified` |
| **TEL-05** | Rule-based HTTP inbound/outbound logger with PII sanitization | [`pkg/httplog/engine.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/engine.go), [`pkg/httplog/sanitizer.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/sanitizer.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/httplog/engine_test.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/engine_test.go) | `✅ Verified` |
| **CR-01** | Cryptographic NanoIDs & prefixed IDs with duplicate alias deprecation | [`pkg/idgen/idgen.go`](file:///home/dangnmh/work/core-kit/pkg/idgen/idgen.go), [`pkg/idutil/nanoid.go`](file:///home/dangnmh/work/core-kit/pkg/idutil/nanoid.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/idgen/idgen_test.go`](file:///home/dangnmh/work/core-kit/pkg/idgen/idgen_test.go) | `✅ Verified` |
| **CR-02** | OpenAI-compatible LLM client & structured JSON extractor with thinking scrubber | [`pkg/llm/client.go`](file:///home/dangnmh/work/core-kit/pkg/llm/client.go), [`pkg/llm/extractor.go`](file:///home/dangnmh/work/core-kit/pkg/llm/extractor.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/llm/extractor_test.go`](file:///home/dangnmh/work/core-kit/pkg/llm/extractor_test.go) | `✅ Verified` |
| **CR-03** | Generic HTTP client with connection pooling, proxy & 429 retry backoff | [`pkg/httpclient/client.go`](file:///home/dangnmh/work/core-kit/pkg/httpclient/client.go), [`pkg/httpclient/pool.go`](file:///home/dangnmh/work/core-kit/pkg/httpclient/pool.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/httpclient/client_test.go`](file:///home/dangnmh/work/core-kit/pkg/httpclient/client_test.go) | `✅ Verified` |
| **CR-04** | Sharded WebSocket pool, auto-reconnecting client & RPC dispatcher | [`pkg/ws/pool.go`](file:///home/dangnmh/work/core-kit/pkg/ws/pool.go), [`pkg/ws/dispatcher.go`](file:///home/dangnmh/work/core-kit/pkg/ws/dispatcher.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/ws/dispatcher_test.go`](file:///home/dangnmh/work/core-kit/pkg/ws/dispatcher_test.go) | `✅ Verified` |
| **CR-05** | High-precision financial decimal arithmetic & orderbook slippage math | [`pkg/decmath/decmath.go`](file:///home/dangnmh/work/core-kit/pkg/decmath/decmath.go), [`pkg/tradecalc/calc.go`](file:///home/dangnmh/work/core-kit/pkg/tradecalc/calc.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/decmath/decmath_test.go`](file:///home/dangnmh/work/core-kit/pkg/decmath/decmath_test.go) | `✅ Verified` |
| **CR-06** | Zero-alloc thread-safe data structures (SafeMap, RingBuffer, TTLCache) | [`pkg/safemap/safemap.go`](file:///home/dangnmh/work/core-kit/pkg/safemap/safemap.go), [`pkg/ringbuffer/ringbuffer.go`](file:///home/dangnmh/work/core-kit/pkg/ringbuffer/ringbuffer.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`tests/property_invariants_test.go`](file:///home/dangnmh/work/core-kit/tests/property_invariants_test.go) | `✅ Verified` |
| **CR-07** | Thread-safe generic PriorityQueue & 2D spatial Quadtree | [`pkg/priorityqueue/priority_queue.go`](file:///home/dangnmh/work/core-kit/pkg/priorityqueue/priority_queue.go), [`pkg/quadtree/quadtree.go`](file:///home/dangnmh/work/core-kit/pkg/quadtree/quadtree.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/quadtree/quadtree_test.go`](file:///home/dangnmh/work/core-kit/pkg/quadtree/quadtree_test.go) | `✅ Verified` |
| **CR-08** | 5-phase Uber Fx graceful shutdown & fail-fast config bootstrapper | [`pkg/lifecycle/lifecycle.go`](file:///home/dangnmh/work/core-kit/pkg/lifecycle/lifecycle.go), [`pkg/bootstrap/bootstrap.go`](file:///home/dangnmh/work/core-kit/pkg/bootstrap/bootstrap.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/lifecycle/lifecycle_test.go`](file:///home/dangnmh/work/core-kit/pkg/lifecycle/lifecycle_test.go) | `✅ Verified` |
| **CR-09** | Circuit breaker, full-jitter backoff & weighted multi-tier rate limiter | [`pkg/resilience/breaker.go`](file:///home/dangnmh/work/core-kit/pkg/resilience/breaker.go), [`pkg/ratelimit/limiter.go`](file:///home/dangnmh/work/core-kit/pkg/ratelimit/limiter.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/resilience/breaker_test.go`](file:///home/dangnmh/work/core-kit/pkg/resilience/breaker_test.go) | `✅ Verified` |
| **CR-10** | In-memory Watermill pub/sub bus with deduplication & audit trail | [`pkg/eventbus/bus.go`](file:///home/dangnmh/work/core-kit/pkg/eventbus/bus.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/eventbus/bus_test.go`](file:///home/dangnmh/work/core-kit/pkg/eventbus/bus_test.go) | `✅ Verified` |
| **CR-11** | Core serialization & pointer helpers (jsonutil, xjson, types, ptr, strutil) | [`pkg/jsonutil/jsonutil.go`](file:///home/dangnmh/work/core-kit/pkg/jsonutil/jsonutil.go), [`pkg/xjson/codec.go`](file:///home/dangnmh/work/core-kit/pkg/xjson/codec.go) | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`pkg/jsonutil/jsonutil_test.go`](file:///home/dangnmh/work/core-kit/pkg/jsonutil/jsonutil_test.go) | `✅ Verified` |
| **TOOL-01** | Unified root Makefile, GolangCI-Lint v2, Ruff & ESLint Flat Config | [`Makefile`](file:///home/dangnmh/work/core-kit/Makefile), [`.golangci.yml`](file:///home/dangnmh/work/core-kit/.golangci.yml) | [`ADR-003`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-003-polyglot-linter-and-ci-toolchain.md) | `make ci` | `✅ Verified` |

---

## 3. Security Controls & Invariants Matrix

| Invariant ID | Security Control & Rule | Implementation File | Verified Test Case | Status |
| :--- | :--- | :--- | :--- | :---: |
| **CONST-SEC-01** | Fail-Closed Bearer Auth: Startup panic on empty or whitespace token | [`pkg/httpserver/auth.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/auth.go) | [`pkg/httpserver/auth_test.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/auth_test.go) | `✅ Verified` |
| **CONST-SEC-02** | Constant-Time Compare: Timing attack immunity for credentials | [`pkg/httpserver/auth.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/auth.go) | [`pkg/httpserver/auth_test.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/auth_test.go) | `✅ Verified` |
| **CONST-SEC-03** | Zero-Leak Telemetry: Automated PII & credential redaction | [`pkg/httplog/sanitizer.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/sanitizer.go) | [`pkg/httplog/module_test.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/module_test.go) | `✅ Verified` |
| **CONST-SEC-04** | Bounded Resource Allocation: Inbound HTTP body capped at 10MB | [`pkg/httpserver/server.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/server.go) | [`pkg/httpserver/server_test.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/server_test.go) | `✅ Verified` |
| **CONST-SEC-05** | Entropy Exhaustion Fast-Panic: Panic immediately on CSPRNG read failure | [`pkg/idgen/idgen.go`](file:///home/dangnmh/work/core-kit/pkg/idgen/idgen.go) | [`pkg/idgen/idgen_test.go`](file:///home/dangnmh/work/core-kit/pkg/idgen/idgen_test.go) | `✅ Verified` |
| **CONST-SEC-06** | Polyglot Log Injection Immunity: Escape newlines and control characters | [`pkg/logger/logger.go`](file:///home/dangnmh/work/core-kit/pkg/logger/logger.go), [`py/logger-py/logger_py/formatter.py`](file:///home/dangnmh/work/core-kit/py/logger-py/logger_py/formatter.py) | [`py/logger-py/tests/test_logger.py`](file:///home/dangnmh/work/core-kit/py/logger-py/tests/test_logger.py) | `✅ Verified` |

---

## 4. Product Roadmap & Deferred Backlog

| Backlog ID | Proposed Feature / Extension | Target Component | Backlog Specification | Planned Release |
| :--- | :--- | :--- | :--- | :---: |
| **BK-01** | `pkg/grpcx`: gRPC client/server interceptors, pooling, health probes | Go Foundation | [`docs/pkg/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/pkg/backlog/backlog-v1.md) | `v1.1.0` |
| **BK-02** | `pkg/eventbus/kafka`: Kafka and Pulsar drivers for Watermill pub/sub | Go Foundation | [`docs/pkg/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/pkg/backlog/backlog-v1.md) | `v1.2.0` |
| **BK-03** | `logger_py/asgi.py`: FastAPI / Litestar ASGI middleware with W3C tracing | Python Telemetry | [`docs/py/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/py/backlog/backlog-v1.md) | `v1.1.0` |
| **BK-04** | `logger-ts/hono.ts`: Hono and Elysia HTTP middleware with Bun streaming | TypeScript Telemetry | [`docs/ts/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/ts/backlog/backlog-v1.md) | `v1.1.0` |
