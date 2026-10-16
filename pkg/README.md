# Go Foundation Engineering Platform (`pkg/*`)
**Version 1.0.0** · Core Infrastructure Team · 2026-10-16 · Standard: HADS 1.0.0

---

## AI READING INSTRUCTION

Read `**[SPEC]**` blocks for non-negotiable invariants, domain entrypoints, and verification commands.
Read `**[BUG]**` blocks before modifying networking, authentication, concurrency, or telemetry code.
Read `**[NOTE]**` blocks for design rationale and consumer service context.

---

## 1. COMPONENT SPECIFICATION & ENTRYPOINTS

**[SPEC]**
- **Root Path**: `pkg/` (Go module: `github.com/dangnmh/core-kit`)
- **Go Toolchain**: Go 1.22+ (tested on Go 1.27.1), zero persistent DB dependencies (`CONST-01`).
- **Verification Commands**:
  - `go test -race ./pkg/...` (Enforces 0 data races across all 34 packages under `-race`, `CONST-04`).
  - `golangci-lint run ./...` (Configured via `.golangci.yml` v2, 0 issues tolerated).
  - `go vet ./pkg/...`
- **7 Functional Domains & Key Packages**:
  1. *Telemetry & Tracing*: `logger` (slog dual-mode handler), `tracer` (OTel W3C traceparent), `httplog` (PII sanitizer), `tracectx`.
  2. *Server & Networking*: `httpserver` (Gin engine, MaxBytes 10MB, Bearer auth), `httpclient` (connection pool, retry), `ws` (sharded pool), `wsclient`, `wsserver`.
  3. *Resilience & Concurrency*: `resilience` (gobreaker circuit breaker, cenkalti backoff), `ratelimit` (weighted token bucket), `ticker` (jittered loop), `eventbus` (Watermill pub/sub), `lifecycle` (5-phase graceful shutdown).
  4. *Data Structures*: `ringbuffer` (O(1) circular buffer with sequence index), `safemap` (RWMutex compute), `ttlcache` (expiring cache), `priorityqueue` (min/max heap), `quadtree` (2D spatial index).
  5. *Financial Math*: `decmath` (decimal tick floor/ceil, zero float64 precision drift, `CONST-05`), `tradecalc` (slippage, fee math).
  6. *Configuration & Bootstrapping*: `bootstrap` (MustLoad fail-fast validator, `CONST-06`), `config` (JSONC env expansion), `version`.
  7. *Utilities*: `idgen` (cryptographic nanoid/prefixed ID), `idutil` (alias to idgen), `jsonutil`, `xjson`, `maputil`, `strutil`, `ptr`, `types`.

---

## 2. KNOWN PITFALLS & BUG MEMORY

**[BUG] Inbound HTTP Memory Exhaustion via Unbounded Body**
- **Symptom**: Out-of-memory container crashes under large malicious payload ingestion.
- **Cause**: Gin router defaults to unbuffered and unbounded stream consumption.
- **Fix**: Mount `httpserver.MaxBytesMiddleware(10 << 20)` (10MB body cap, `CONST-SEC-04`).

**[BUG] Fail-Open Bearer Auth Under Empty Configuration**
- **Symptom**: Unauthenticated endpoints accept requests when `BEARER_TOKEN` env var is unset or whitespace.
- **Cause**: Historical middleware logged warning and called `c.Next()` on empty tokens.
- **Fix**: `httpserver.BearerTokenAuth(token)` panics immediately at startup if `strings.TrimSpace(token) == ""` (`CONST-SEC-01`).

**[BUG] Zero-Byte ID Collision on Cryptographic Entropy Exhaustion**
- **Symptom**: Colliding `"evt_0000000000000000"` transaction IDs across high-throughput instances.
- **Cause**: Fallback string formatting on `rand.Read` error emitted zero-value byte slices.
- **Fix**: `idgen.NewPrefixedID` panics immediately on `rand.Read` failure (`CONST-SEC-05`).

---

## 3. PUBLIC CONTRACTS & ARCHITECTURE MAP

**[SPEC]**
- Feature Registry & Matrix: [`docs/pkg/README.md`](file:///home/dangnmh/work/core-kit/docs/pkg/README.md)
- Internal Architecture & State Diagrams: [`docs/pkg/architecture.md`](file:///home/dangnmh/work/core-kit/docs/pkg/architecture.md)
- Synchronous Public API Contract: [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md)
- Events, Memory Tables & Env Vars Contract: [`docs/pkg/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/events-and-data.md)
- Active Sprint Tasks & Deltas: [`docs/pkg/tasks/sprint-1-core-standardization.md`](file:///home/dangnmh/work/core-kit/docs/pkg/tasks/sprint-1-core-standardization.md)
- Deferred Feature Backlog: [`docs/pkg/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/pkg/backlog/backlog-v1.md)
