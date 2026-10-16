# Go Foundation Engineering Platform (`pkg/*`)
**Version 1.0.0** · Core Infrastructure Team · 2026-10-16 · Standard: HADS 1.0.0

---

## AI READING INSTRUCTION

Read `**[SPEC]**` blocks for authoritative feature matrices, architecture links, and frozen contracts.
Read `**[BUG]**` blocks to avoid repeating verified historical failure modes.
Read `**[NOTE]**` blocks for design rationale and consumer service context.

---

## 1. COMPONENT OVERVIEW & FEATURE MATRIX

**[SPEC]**
The Go foundation component (`pkg/*`) provides 34 zero-alloc, high-performance packages for downstream services:

| Domain | Subsystems / Packages | Supported Capabilities | Status |
| :--- | :--- | :--- | :---: |
| **D1: Telemetry** | `logger`, `tracer`, `httplog`, `tracectx` | Dual-mode slog JSON, OTel W3C traceparent, PII redaction | `ACTIVE` |
| **D2: Networking** | `httpserver`, `httpclient`, `ws`, `wsclient`, `wsserver` | Gin 10MB body cap, Bearer auth panic, sharded WS pool | `ACTIVE` |
| **D3: Resilience** | `resilience`, `ratelimit`, `ticker`, `eventbus`, `lifecycle` | Circuit breaker, jittered backoff, rate limit, 5-phase shutdown | `ACTIVE` |
| **D4: Data Structures** | `ringbuffer`, `safemap`, `ttlcache`, `priorityqueue`, `quadtree` | O(1) circular buffer, RWMutex compute, spatial 2D index | `ACTIVE` |
| **D5: Financial Math** | `decmath`, `tradecalc` | Decimal tick floor/ceil, zero float64 precision drift | `ACTIVE` |
| **D6: Config** | `bootstrap`, `config`, `version` | Fail-fast MustLoad validation, JSONC env expansion | `ACTIVE` |
| **D7: Utilities** | `idgen`, `idutil`, `jsonutil`, `xjson`, `maputil`, `strutil`, `ptr`, `types` | Cryptographic nanoids, panic-free map accessors | `ACTIVE` |

---

## 2. NAVIGATION HUB & LIVING CONTRACTS

**[SPEC]**
All authoritative technical contracts and specifications are organized in this canonical hub:

| Document | Description | Path |
| :--- | :--- | :--- |
| **Architecture** | Internal design, package dependency tree & state diagrams | [`architecture.md`](file:///home/dangnmh/work/core-kit/docs/pkg/architecture.md) |
| **API Contract** | Synchronous interfaces, types, guards, and consumers | [`contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) |
| **Data & Events** | Memory structures, emitted events & environment variables | [`contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/events-and-data.md) |
| **Sprint Tasks** | Sprint 1 OpenSpec delta and EARS acceptance criteria | [`tasks/sprint-1-core-standardization.md`](file:///home/dangnmh/work/core-kit/docs/pkg/tasks/sprint-1-core-standardization.md) |
| **Backlog** | Deferred non-P0 feature proposals and ideas | [`backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/pkg/backlog/backlog-v1.md) |

---

## 3. KNOWN PITFALLS & BUG MEMORY

**[BUG] Fail-Open Bearer Auth Under Empty Configuration**
- **Symptom**: Unauthenticated endpoints accept requests when `BEARER_TOKEN` env var is empty.
- **Cause**: Middleware historically logged warning and called `c.Next()`.
- **Fix**: `httpserver.BearerTokenAuth(token)` panics immediately at startup if token is empty or whitespace (`CONST-SEC-01`).

**[BUG] Inbound HTTP Memory Exhaustion via Unbounded Body**
- **Symptom**: Pod OOM killed during large file upload or denial-of-service attack.
- **Cause**: Gin router unbuffered body consumption.
- **Fix**: Inbound HTTP request body bounded to 10MB via `httpserver.MaxBytesMiddleware` (`CONST-SEC-04`).

**[BUG] Cryptographic Entropy Depletion Emits Colliding IDs**
- **Symptom**: High-concurrency workers emit duplicate `"evt_0000000000000000"` transaction IDs.
- **Cause**: Fallback string formatter formatted zero-byte slice on `rand.Read` error.
- **Fix**: `idgen.NewPrefixedID` panics immediately on `rand.Read` failure (`CONST-SEC-05`).
