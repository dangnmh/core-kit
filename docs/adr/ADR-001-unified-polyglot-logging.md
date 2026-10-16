# ADR-001: Unified Polyglot Structured Logging Schema (Go, Python, TypeScript)

## Status

**Accepted**

## Context

`core-kit` provides foundation logging libraries for microservices running across three language ecosystems:
- Go (`pkg/logger`) — built on standard library `log/slog`
- Python (`py/logger-py`) — built on standard library `logging` with `JsonFormatter`
- TypeScript / Bun (`ts/logger-ts`) — zero-dependency structured logger emitting single-line JSON

In distributed production environments (such as `realtime-gateway` and `trading-engine`), a single user transaction or algorithmic execution order traverses all three runtimes:
1. Ingress and WebSocket routing through Bun / TypeScript (`ts/logger-ts`)
2. Order matching and core business execution in Go (`pkg/logger`)
3. Autonomous AI model inference and text summarization in Python (`py/logger-py`)

### Problem

A forensic audit identified schema drift among the three implementations:
1. **Field Naming Inconsistency**: Go emitted `"module"`, whereas Python and TypeScript emitted `"logger"` to identify the emitting subsystem or package.
2. **Context Attribute Placement**: Python and TypeScript encapsulated contextual key-value pairs inside a nested `"context": { ... }` object, whereas Go (`slog.NewJSONHandler`) flattened custom attributes directly into the root JSON object (`{"service": "...", "symbol": "BTC/USDT", "slippage": "0.0001"}`).
3. **Log Aggregator Friction**: In Grafana Loki, ClickHouse, and Datadog, engineers had to write divergent queries depending on the originating service language (e.g. `context.symbol` for Python vs `symbol` for Go), resulting in failed cross-service joins and alert discrepancies.

## Decision Drivers

- **Single Query Syntax**: SRE and on-call engineers must be able to filter cross-service traces using identical queries in Grafana Loki / ClickHouse (e.g., `| json | context.request_id = "..."`).
- **W3C Distributed Trace Correlation**: All three runtimes must seamlessly correlate logs via W3C `traceparent` headers (`trace_id` and `span_id`).
- **Backward Compatibility**: Existing Go services actively parsing top-level JSON fields must not experience breaking ingestion failures.
- **Minimal Allocation & High Throughput**: Telemetry serialization must maintain sub-microsecond latency and near-zero allocations in the hot path.

## Considered Options

### Option 1: Flatten All Context Attributes at the Root (Go Style)
- **Pros**: Matches default Go `slog` behavior without custom Handler wrapping.
- **Cons**: Root JSON namespace becomes polluted with arbitrary business attributes; collisions occur if a business key matches a core metadata key (e.g. `"message"` or `"service"`); breaks Python and TypeScript standards.

### Option 2: Strictly Force Nested `"context"` Everywhere Immediately
- **Pros**: Cleanest conceptual architecture; guarantees identical JSON structure across all three languages.
- **Cons**: High breaking-change risk for existing downstream Go services (`realtime-gateway`) and LogQL pipelines expecting top-level fields.

### Option 3: Canonical Schema with Nested `"context"` and Go Dual-Mode Support (Selected)
- Standardize the canonical contract on:
  - `timestamp` (RFC3339 UTC string with nanosecond or millisecond precision and trailing `Z`)
  - `severity` (`DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`)
  - `service` (string service identifier)
  - `logger` (string logger/module identifier; `"module"` supported as legacy alias)
  - `message` (clean string, emojis stripped)
  - `caller` (file:line, optional)
  - `trace_id` (32-character hex string, optional)
  - `span_id` (16-character hex string, optional)
  - `context` (nested key-value JSON object for custom fields)
- Go `pkg/logger` introduces a dual-mode slog Handler:
  - By default, contextual attributes are nested under `"context"`.
  - Configurable legacy mode or root attribute mirroring ensures existing LogQL queries do not break during migration.
- Python and TypeScript loggers standardize on the same exact keys and extract `trace_id`/`span_id` into the root.

## Decision

We adopt **Option 3**: Standardize on a canonical JSON schema with a nested `"context"` object and `"logger"` identifier across Go, Python, and TypeScript, backed by Go dual-mode attribute mapping.

### Canonical JSON Log Record Format

```json
{
  "timestamp": "2026-10-16T15:04:05.123456Z",
  "severity": "INFO",
  "service": "order-svc",
  "logger": "engine",
  "message": "Order limit snapshot computed",
  "caller": "order.go:142",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "context": {
    "symbol": "BTC/USDT",
    "slippage": "0.0001",
    "user_id": 48921
  }
}
```

## Consequences

### Positive
- **100% Cross-Language LogQL Correlation**: Queries such as `{service=~".+"} | json | context.user_id = 48921` return matching logs from Go, Python, and TypeScript in chronological order.
- **Clean Root Namespace**: Protects core metadata keys from collision with user-defined context keys.
- **Native W3C Traceparent Integration**: Out-of-the-box trace visualization in Grafana Tempo and OpenTelemetry collectors.

### Negative
- Go `pkg/logger` requires a dedicated `ContextHandler` or `slog.Group("context")` wrapping logic.
- Minor overhead (one additional JSON object nesting) during serialization.

### Risks & Mitigations
- **Risk**: Downstream log ingestor parses legacy root fields in Go logs.
- **Mitigation**: Go's `pkg/logger` provides backwards-compatible attribute aliasing or dual-emission via `pkg/logger.SetLegacyFlattening(true)` if required during blue-green migration.

## Related Documents
- [`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md) (Epic 2, Stories 2.1-2.4)
- [`docs/bmad/04-architecture.md`](file:///home/dangnmh/work/core-kit/docs/bmad/04-architecture.md)
- [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md)
- [`docs/py/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/api.md)
- [`docs/ts/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/api.md)
