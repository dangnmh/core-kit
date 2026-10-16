# Component Backlog: Go Core Foundation (`pkg/*`)

**Document ID**: `BACKLOG-PKG-V1`  
**Status**: `⏸️ Backlog / Deferred (HADS **[?]**)`  
**Component**: Go Core Infrastructure (`pkg/*`)  
**Parent PRD**: [`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md)  
**Last Updated**: 2026-10-16  

---

## 1. Overview & HADS Isolation Rule

In accordance with the **BMAD 4-Layer Monorepo Documentation Standard (HADS 1.0)**, this document isolates all deferred, future, and unbuilt proposals (`**[?]**`) for the Go Core Foundation (`pkg/*`). These items are explicitly **OUT OF SCOPE** for the current zero-to-prod cycle to prevent diluting the Living Truth component documentation in [`docs/pkg/README.md`](file:///home/dangnmh/work/core-kit/docs/pkg).

---

## 2. Deferred Proposals & Architectural Items

### 2.1 **[?]** `pkg/grpcx`: Enterprise gRPC Client & Server Interceptor Suite
- **Feature ID**: `BK-01`
- **Priority**: P2 / Phase 3
- **Motivation**: While `core-kit` provides production-grade HTTP ([`pkg/httpclient`](file:///home/dangnmh/work/core-kit/pkg/httpclient), [`pkg/httpserver`](file:///home/dangnmh/work/core-kit/pkg/httpserver)) and WebSocket ([`pkg/ws`](file:///home/dangnmh/work/core-kit/pkg/ws)) tooling, high-throughput inter-service RPC in microservice topologies often requires binary Protobuf/gRPC transport.
- **Proposed Scope**:
  - Unary & Stream client interceptors with automatic W3C traceparent injection.
  - Uber Fx integration for gRPC server registration and health check (`grpc.health.v1`).
  - Tuned HTTP/2 connection pooling with client-side keepalive pings and exponential backoff retry.
- **Revisit Trigger**: When a downstream microservice initiates gRPC service mesh migration or requires sub-millisecond Protobuf RPC.

---

### 2.2 **[?]** `pkg/eventbus/kafka`: Distributed EventBus Drivers for Watermill
- **Feature ID**: `BK-02`
- **Priority**: P2 / Phase 3
- **Motivation**: Current [`pkg/eventbus`](file:///home/dangnmh/work/core-kit/pkg/eventbus) is built on Watermill's in-memory GoChannel pub/sub. While ideal for single-binary orchestration, distributed microservices require persistent brokers.
- **Proposed Scope**:
  - Add pluggable Kafka (via `confluent-kafka-go` or `segmentio/kafka-go`) and Apache Pulsar publisher/subscriber adapters.
  - At-least-once message delivery guarantees with dead-letter queue (DLQ) routing.
  - Automatic JSON schema header propagation and correlation ID preservation.
- **Revisit Trigger**: When multi-node message passing across distinct cluster nodes is scheduled for `realtime-gateway` or `trading-engine`.

---

### 2.3 **[?]** `pkg/ringbuffer/shm`: Shared-Memory IPC RingBuffer
- **Feature ID**: `BK-05`
- **Priority**: P2 / Phase 3
- **Motivation**: Ultra-low latency algorithmic trading execution between independent processes on the same host (e.g. market data receiver daemon and execution engine) cannot afford Unix domain socket or TCP overhead.
- **Proposed Scope**:
  - POSIX shared-memory (`shm_open`, `mmap`) ringbuffer backing store.
  - Lock-free atomic sequence counter synchronization between independent OS processes.
  - Sub-microsecond (< 500ns) message delivery latency.
- **Revisit Trigger**: When colocation trading requirements demand sub-microsecond cross-process IPC.

---

### 2.4 **[?]** `pkg/metrics`: OpenTelemetry Metrics Meter & OTLP Exporter
- **Feature ID**: `BK-06`
- **Priority**: P2 / Phase 3
- **Motivation**: `core-kit` currently implements distributed tracing ([`pkg/tracer`](file:///home/dangnmh/work/core-kit/pkg/tracer)) and structured logging ([`pkg/logger`](file:///home/dangnmh/work/core-kit/pkg/logger)), but lacks a standardized wrapper for OpenTelemetry metrics (counters, histograms, gauges).
- **Proposed Scope**:
  - Generic metric instrument helper with automatic service and environment tagging.
  - Prometheus pull exporter endpoint (`/metrics`) and OTLP gRPC push exporter.
  - Standard runtime metric collectors (Go runtime memory, GC pauses, goroutine count).
- **Revisit Trigger**: When Prometheus/Grafana metric standardization is mandated across all platform clusters.

---

### 2.5 **[?]** Non-Gorilla WebSocket Engine Evaluation (`nhooyr/websocket` or `coder/websocket`)
- **Feature ID**: `BK-07`
- **Priority**: P2 / Evaluation
- **Motivation**: [`pkg/ws`](file:///home/dangnmh/work/core-kit/pkg/ws) currently uses [`github.com/gorilla/websocket`](file:///home/dangnmh/work/core-kit/go.mod). While Gorilla is battle-tested, newer alternatives (`coder/websocket`) provide native context cancellation and lower memory allocations per connection.
- **Proposed Scope**:
  - Benchmark Gorilla vs Coder WebSocket under 10,000 concurrent streaming connections.
  - Provide zero-allocation read/write loops if performance gains exceed 25%.
- **Revisit Trigger**: Post-v1.0 release during scheduled performance tuning sprints.
