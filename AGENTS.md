# AGENTS.md — Core-Kit Project Constitution & Global AI Guardrails
**Repository**: `github.com/dangnmh/core-kit` · **Runtimes**: Go 1.27+, Python 3.14+, TypeScript/Bun 1.4+ · **HADS 1.0.0**

## 1. PROJECT CONSTITUTION (NON-NEGOTIABLE INVARIANTS)
Every agent, human developer, and CI workflow MUST adhere to these binding rules:
- **`CONST-01` (Zero Storage)**: Stateless in-memory platform. Zero database tables or persistent volumes; all state in transient memory.
- **`CONST-02` (Strict Reuse)**: Reuse `pkg/*`, stdlib, and verified libs before writing custom code; no reinvention.
- **`CONST-03` (Unified Telemetry)**: RFC3339 single-line JSON log schema across Go, Python, and TypeScript with `"context"` nesting & hoisted W3C `trace_id`/`span_id`.
- **`CONST-04` (Zero Data Races)**: All concurrent code MUST pass `go test -race ./...` with 0 data races detected under stress.
- **`CONST-05` (Financial Math Precision)**: Never use raw `float64` for financial values without `pkg/decmath.Decimal` or tick-snapping (`SnapToTickFloor`/`SnapToTickCeil`).
- **`CONST-06` (Fail-Fast Config)**: Configurations MUST load via `pkg/bootstrap.MustLoad[T]`. Abort (`exit(1)`) on invalid/missing config before opening ports.
- **`CONST-07` (5-Phase Graceful Shutdown)**: Orderly termination via `pkg/lifecycle.Manager` (`PreStop` -> `StopIngress` -> `Drain` -> `StopEgress` -> `PostStop`).
- **`CONST-08` (4-Layer HADS Documentation)**: Root `AGENTS.md` and component guides `docs/<comp>/README.md` MUST remain `<= 120` lines with 0 broken links.

## 2. SECURITY CONSTITUTION
- **`CONST-SEC-01` (Fail-Closed Auth)**: Bearer auth (`pkg/httpserver.BearerTokenAuth`) panics immediately on startup if token is empty or whitespace.
- **`CONST-SEC-02` (Constant-Time Compare)**: All secret / token comparisons MUST use `crypto/subtle.ConstantTimeCompare`.
- **`CONST-SEC-03` (Zero-Leak Telemetry)**: PII and credentials masked/redacted before writing logs or traces (`pkg/httplog`).
- **`CONST-SEC-04` (Bounded Allocation)**: Inbound HTTP bodies capped at 10MB via `httpserver.MaxBytesMiddleware`.
- **`CONST-SEC-05` (CSPRNG Fast-Panic)**: `pkg/idgen` MUST panic immediately on `crypto/rand.Reader` error; never return zeroed bytes.
- **`CONST-SEC-06` (Log Injection Immunity)**: All log formatters serialize strict JSON, escaping newlines and control characters.

## 3. UNIFIED COMMANDS & VERIFICATION GATES
Run these commands from the repository root:
- `make test` — Runs Go tests (`go test -race ./...`), TS tests (`bun test`), and Py tests (`uv run pytest tests -v`).
- `make lint` — Runs `golangci-lint run ./...`, `bun run lint`, `uv run ruff check .`, and `make doc-lint`.
- `make format` (`make fmt`) — Formats Go (`gofmt -s -w .`), TS (`bun run format`), and Py (`uv run ruff format .`).
- `make doc-lint` — Validates line caps (`<= 120` lines) on Layer 0/1 READMEs and verifies 0 broken markdown links.
- `make ci` — Full automated CI gate (`make doc-lint lint test`).

## 4. TOOLCHAIN TRAPS & PITFALL MEMORY
- **Go errcheck**: Never ignore error returns on `resp.Body.Close()`, `conn.Close()`, or `ws.WriteMessage()`.
- **TypeScript Streams**: In `ts/logger-ts`, never monkey-patch global `process.stdout.write` in tests; inject memory streams via `createLogger({ stream })`.
- **Python Unicode**: `py/logger-py` strips emojis via regex to prevent log aggregator parsing corruptions.
- **Doc Links**: All cross-references must use valid relative links or `file:///` URLs verified by `./scripts/doc-lint.sh`.

## 5. REPOSITORY MAP & NEAREST-WINS CONTEXT
- **Layer 0**: [`AGENTS.md`](file:///home/dangnmh/work/core-kit/AGENTS.md), [`llms.txt`](file:///home/dangnmh/work/core-kit/llms.txt), [`README.md`](file:///home/dangnmh/work/core-kit/README.md), [`CHANGELOG.md`](file:///home/dangnmh/work/core-kit/CHANGELOG.md)
- **Layer 1**: [`pkg/README.md`](file:///home/dangnmh/work/core-kit/pkg/README.md), [`py/logger-py/README.md`](file:///home/dangnmh/work/core-kit/py/logger-py/README.md), [`ts/logger-ts/README.md`](file:///home/dangnmh/work/core-kit/ts/logger-ts/README.md)
- **Layer 2**: [`docs/README.md`](file:///home/dangnmh/work/core-kit/docs/README.md), [`docs/06-feature-matrix-and-roadmap.md`](file:///home/dangnmh/work/core-kit/docs/06-feature-matrix-and-roadmap.md), [`docs/adr/`](file:///home/dangnmh/work/core-kit/docs/adr), [`docs/runbooks/`](file:///home/dangnmh/work/core-kit/docs/runbooks), `docs/<comp>/{README.md,architecture.md,contracts/}`
- **Layer 3**: `docs/<comp>/{tasks,releases,backlog}/`, [`docs/bmad/`](file:///home/dangnmh/work/core-kit/docs/bmad)
