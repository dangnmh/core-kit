# Go Foundation Events, In-Memory State & Data Contract (`pkg/*`)

**Document ID**: `CONTRACT-GO-DATA-01`  
**Package Base**: `github.com/dangnmh/core-kit/pkg`  
**Runtime**: Go 1.27.1+ (Backwards-compatible Go 1.22+)  
**Status**: `FROZEN CONTRACT`  
**Related ADRs**: [`ADR-001`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md), [`ADR-002`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-002-websocket-architecture.md), [`ADR-003`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-003-polyglot-linter-and-ci-toolchain.md)  
**Parent Specification**: [`docs/bmad/05-database-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/05-database-spec.md)  

---

## 1. Zero-Storage Invariant & In-Memory Data Models

Per `CONST-01`, `pkg/*` owns **zero persistent databases** and **zero disk tables**. All state structures reside exclusively in memory. Downstream services persisting data models follow the specifications below.

### 1.1 In-Memory Concurrency Primitives & Memory Layouts

#### 1.1.1 `pkg/safemap.SafeMap[K, V]`
- **Underlying Type**: `map[K]V` protected by `sync.RWMutex`.
- **Invariants**:
  - `Get(key)` acquires `RLock()`, returning value and existence boolean with 0 allocations for scalar types.
  - `Compute(key, fn)` acquires exclusive `Lock()`. The callback `fn(old V, exists bool) (V, bool)` executes atomically. Returning `keep=false` deletes the entry; returning `keep=true` stores `newVal`.
  - **Deadlock Prohibition**: `fn` MUST NOT call any methods on the same `SafeMap` instance.
  - `Range(fn)` acquires `RLock()`. Iteration aborts immediately if `fn` returns `false`.

#### 1.1.2 `pkg/ringbuffer.RingBuffer[T]`
- **Underlying Type**: Pre-allocated contiguous slice `buffer []T` of fixed length `capacity`.
- **Invariants**:
  - Zero dynamic heap re-allocation during steady-state `Push(item)` (`NFR-PERF-02`).
  - Monotonic 64-bit sequence counter (`seqCounter int64`) incremented on every push.
  - Overwrites oldest entry in $O(1)$ time when full: `head = (head + 1) % capacity`.
  - `GetRecentWindow(durationSeconds, getTimestamp)` walks backward from `head-1` to count matching items, allocating an exact-sized slice once.
  - Thread safety enforced via internal `sync.RWMutex`.

#### 1.1.3 `pkg/ttlcache.TTLCache[K, V]`
- **Underlying Type**: `map[K]cacheEntry[V]` where `cacheEntry[V] = { value V, expiration time.Time }`.
- **Invariants**:
  - `Set(key, val, ttl)` acquires `Lock()`. If `ttl <= 0`, defaults to `defaultTTL`.
  - `Get(key)` uses double-checked lazy eviction: checks expiration under `RLock()`, promotes to `Lock()`, and re-verifies `isExpired()` before deleting to prevent dropping concurrent fresh writes.
  - Background janitor ticker runs at `cleanupInterval` in a dedicated goroutine. `Close()` terminates the janitor cleanly via `sync.WaitGroup`.

#### 1.1.4 `pkg/priorityqueue.PriorityQueue[T]`
- **Underlying Type**: Min/Max binary heap (`container/heap.Interface`) over `[]*item[T]` where `item[T] = { value T, index int, seq uint64 }`.
- **Invariants**:
  - Stable FIFO tie-breaking: when user `less(a, b)` evaluates equal, the entry with smaller `seq` counter dequeues first.
  - `Enqueue` runs in $O(\log N)$ time under write lock.
  - `Peek` runs in $O(1)$ time under read lock.

#### 1.1.5 `pkg/quadtree.Quadtree[T]`
- **Underlying Type**: 4-ary spatial tree dividing `Rect{MinX, MinZ, MaxX, MaxZ}` into quadrants (`NW`, `NE`, `SW`, `SE`).
- **Invariants**:
  - Nodes split upon exceeding `capacity` entries unless `depth >= maxDepth`.
  - Bounding box intersections prune empty branches during `QueryRange` and `UpdateRange`.
  - `QueryNearest` searches children ordered by distance to target `(x, z)`, pruning branches whose bounding box minimum distance exceeds the current nearest candidate.

---

## 2. In-Memory EventBus Topics & Event Schemas (`pkg/eventbus`)

`pkg/eventbus.Bus` provides intra-process, cycle-scoped event fan-out backed by Watermill `GoChannel`.

### 2.1 Bus Configuration Invariants
- **Buffer**: `OutputChannelBuffer: 64`.
- **Persistence**: `Persistent: false` (Zero disk persistence; discarded on cycle termination).
- **Delivery Model**: At-most-once within cycle; non-blocking publish (`BlockPublishUntilSubscriberAck: false`).

### 2.2 Event Envelope & Deduplication Contract
- **Message Serialization**: JSON payload stored in `watermill.Message.Payload`.
- **Deduplication Interface**:
  ```go
  type Deduplicatable interface {
      DeduplicateKey() string
  }
  ```
- **Deduplication Rule**: If payload implements `Deduplicatable`, its returned string key is registered in `middleware.MapExpiringKeyRepository` with a 24-hour expiration and 5-second check timeout. Duplicate publications within the window are dropped with a warning log.

### 2.3 Event Timeline Audit Contract (`Timeline`)
Every published event is captured sequentially in an in-memory chronological slice:

```go
type LogEntry struct {
    Time    time.Time       `json:"time"`
    Topic   string          `json:"topic"`
    MsgID   string          `json:"msg_id"`
    Payload json.RawMessage `json:"payload"`
}
```
- `Timeline()` returns a shallow copy of all recorded events in chronological order for audit, debugging, and post-cycle inspection.

### 2.4 Standard Event Topics Naming Matrix

| Topic Pattern | Emitting Subsystem | Payload Struct / Type | Description |
| :--- | :--- | :--- | :--- |
| `cycle.order.placed` | Trading Engine | `OrderPlacedEvent` | Emitted when an IOC limit order is sent to exchange. |
| `cycle.order.filled` | WebSocket Consumer | `OrderFilledEvent` | Emitted when an execution fill report is received. |
| `cycle.position.updated`| Risk Manager | `PositionUpdateEvent` | Emitted when inventory or margin balances update. |
| `cycle.state.transition`| Lifecycle Manager | `StateTransitionEvent` | Emitted on 5-phase shutdown transitions. |
| `system.alert.circuit` | Resilience Engine | `CircuitBreakerEvent` | Emitted when circuit transitions to `StateOpen`. |

---

## 3. Canonical Telemetry & Log Schema Contract (`pkg/logger`)

Per [`ADR-001`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md), Go logs emitted via `pkg/logger` MUST conform to the cross-language RFC3339 JSON logging standard.

### 3.1 Emitted JSON Schema Specification

```json
{
  "timestamp": "2026-10-16T15:04:05.123456789Z",
  "severity": "INFO",
  "service": "trading-engine",
  "logger": "engine",
  "message": "Market tick snapped to price unit",
  "caller": "engine.go:88",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "context": {
    "symbol": "BTC/USDT",
    "snapped_price": "65120.50",
    "tick_size": "0.01"
  }
}
```

### 3.2 Dual-Mode Attribute Mapping Contract
1. **Canonical Mode (Default)**: Any custom key-value attributes passed to `logger.Info(msg, args...)` are placed inside the `"context"` sub-object.
2. **Legacy Flattening Mode (`LOG_LEGACY_FLAT=true`)**: If set, contextual key-values are mirrored at the root level and under `"context"` to prevent breakage during downstream pipeline upgrades.
3. **Trace Context Propagation**: If the incoming `context.Context` contains an active OpenTelemetry W3C span (`trace.SpanContextFromContext`), `trace_id` and `span_id` are hoisted to top-level keys automatically.

---

## 4. Environment Variables Contract

The following environment variables govern configuration loading ([`pkg/config`](file:///home/dangnmh/work/core-kit/pkg/config)), bootstrapping ([`pkg/bootstrap`](file:///home/dangnmh/work/core-kit/pkg/bootstrap)), logging ([`pkg/logger`](file:///home/dangnmh/work/core-kit/pkg/logger)), and distributed tracing ([`pkg/tracer`](file:///home/dangnmh/work/core-kit/pkg/tracer)):

| Environment Variable | Target Package | Value Type | Default Value | Description / Validation Rules |
| :--- | :--- | :--- | :--- | :--- |
| `CONFIG_PATH` | `pkg/config` | `string` | *(Empty / Candidate list)* | Explicit path to YAML, JSON, or JSONC configuration file. |
| `LOG_LEVEL` | `pkg/logger` | `string` | `"info"` | Log level filter: `DEBUG`, `INFO`, `WARN`, `ERROR`. Case-insensitive. |
| `LOG_LEGACY_FLAT` | `pkg/logger` | `bool` | `false` | When `true`, mirrors context fields to root JSON for legacy pipelines. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `pkg/tracer` | `string` | `""` | OpenTelemetry OTLP HTTP/gRPC collector endpoint (e.g. `http://otel-collector:4318`). |
| `OTEL_SERVICE_NAME` | `pkg/tracer` | `string` | `"core-kit-service"` | Service name identifier exported in OTel resource attributes. |
| `OTEL_INSECURE` | `pkg/tracer` | `bool` | `true` | When `true`, allows plaintext HTTP transport to the OTel collector. |
| `HTTP_PORT` | `pkg/httpserver` | `int` | `8080` | Port for the Gin HTTP engine (`/healthz`, `/readyz`). |

---

## 5. Downstream Mathematical Precision Contract (`pkg/decmath`)

Downstream persistence layers mapping `pkg/decmath.Decimal` MUST observe the following numeric storage standards:

- **PostgreSQL Column Type**: `NUMERIC(28, 10)` (Exact fixed-point decimal; zero floating point precision drift).
- **ClickHouse Column Type**: `Decimal128(10)` or `String`.
- **JSON Serialization**: Emitted as exact string representation (`price.StringFixed(scale)`) across API boundaries to prevent IEEE-754 precision loss in downstream JavaScript/Python number parsers.
