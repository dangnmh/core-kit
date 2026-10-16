# ADR-002: Sharded Multiplexed WebSocket Pool and Dispatcher Architecture

## Status

**Accepted**

## Context

Real-time streaming services (`realtime-gateway`) and algorithmic fintech engines (`trading-engine`) require sustained bidirectional WebSocket communication with external cryptocurrency exchanges (Binance, Bybit, OKX) and internal frontend gateways.

### Challenges & Constraints
1. **Exchange Subscription Limits**: Major exchanges cap subscriptions per connection (e.g., 30 symbols/topics per WebSocket connection). Exceeding this limit causes immediate connection termination.
2. **Blast Radius of Disconnects**: If a single monolithic connection drops, all market data feeds and active order streams are severed simultaneously, halting algorithmic trading.
3. **RPC over WebSocket (Request-Response Correlation)**: Several exchange protocols require sending an asynchronous command over WebSocket and awaiting a correlated response matching a specific request ID (`id` or `req_id`).
4. **Network Jitter & Reconnection Storms**: Temporary network drops must trigger full-jitter exponential backoff reconnection rather than hammering upstream exchanges with synchronized retries.

## Decision Drivers

- **Connection Sharding**: Dynamically partition hundreds of topic subscriptions across multiple managed WebSocket clients without exceeding per-connection limits (`maxPairs`).
- **Fault Isolation**: A failure or reconnection on Shard #2 must not interrupt traffic on Shard #1 or Shard #3.
- **RPC Request Matching**: Generic, type-safe correlation of requests to asynchronous responses with timeout handling and round-trip time (RTT) tracking.
- **Zero Allocations & Thread Safety**: Concurrency state must be protected by fine-grained RWMutex locks and zero data races.

## Considered Options

### Option 1: Single Monolithic WebSocket Connection
- **Pros**: Simplest to implement.
- **Cons**: Hard subscription limits cannot be bypassed; single point of failure; disconnects drop all market feeds.

### Option 2: Dedicated WebSocket Connection Per Topic
- **Pros**: Maximum isolation.
- **Cons**: Severe socket descriptor exhaustion; violates exchange IP connection rate limits; high memory overhead.

### Option 3: Sharded Multiplexed Connection Pool with Topic Routing & RPC Dispatcher (Selected)
- Implement a sharded pool (`pkg/ws.Pool`) that manages an array of underlying clients:
  - Tracks subscription density (`clientSubCount`) and allocates new topics to the least-loaded shard.
  - Maintains `topicRouting map[string]int` for deterministic message routing and unsubscription.
  - Isolates private authenticated channels (order fills, balance updates) on a dedicated client from public market streams.
  - Integrates `RequestDispatcher[T]` for type-safe asynchronous RPC matching.
  - Provides callback-oriented client (`pkg/wsclient`) and broadcast server (`pkg/wsserver`) for server-side topologies.

## Decision

We adopt **Option 3**: The Sharded Multiplexed WebSocket Pool architecture (`pkg/ws`, `pkg/wsclient`, `pkg/wsserver`).

### Architectural Topology

```mermaid
flowchart TD
    subgraph App["Application Layer (trading-engine / realtime-gateway)"]
        Consumer["Market Data / Order Consumer"]
    end

    subgraph WSPool["pkg/ws.Pool (Sharded Manager)"]
        Router["Topic Router & Shard Allocator\n(maxPairs: 30)"]
        Dispatcher["RequestDispatcher[T]\n(RPC Correlation & RTT)"]
    end

    subgraph Shards["Multiplexed WebSocket Shards"]
        S1["Shard 1 Client\n(Pairs 1..30)"]
        S2["Shard 2 Client\n(Pairs 31..60)"]
        S3["Shard 3 Client\n(Pairs 61..90)"]
        SPriv["Private Dedicated Client\n(Orders / Positions)"]
    end

    subgraph Upstream["External Exchanges / Gateways"]
        Ex1["Exchange Public Stream"]
        Ex2["Exchange Private Stream"]
    end

    Consumer -->|Subscribe("BTC/USDT")| Router
    Consumer -->|SendRPC(req)| Dispatcher
    Router -->|Allocates| S1
    Router -->|Allocates| S2
    Router -->|Allocates| S3
    Router -->|Private Auth| SPriv

    S1 <-->|WS / Ping-Pong| Ex1
    S2 <-->|WS / Ping-Pong| Ex1
    S3 <-->|WS / Ping-Pong| Ex1
    SPriv <-->|WS / HMAC Auth| Ex2
```

### Key Components & Responsibilities

1. **`pkg/ws.Pool`**:
   - Manages an array of public clients (`publicClients []*Client`) and a dedicated private client (`privateClient *Client`).
   - Balances topic subscriptions across shards up to `maxPairs` (default: 30).
   - Re-subscribes active topics automatically upon shard reconnection.
2. **`pkg/ws.RequestDispatcher[T]`**:
   - Thread-safe generic request-response correlator (`SafeMap[string, chan *T]`).
   - Enforces execution timeouts via `context.Context`.
   - Tracks Round-Trip Time (RTT) per request for latency observability.
3. **`pkg/wsclient.Client`**:
   - Standalone auto-reconnecting client for 1-to-1 WebSocket connections.
   - Background read/write pumps with Gorilla WebSocket ping/pong heartbeat enforcement.
4. **`pkg/wsserver.Server`**:
   - High-throughput WebSocket server handling client connection pools, heartbeats, and broadcast message distribution (`BroadcastJSON[T]`).

## Consequences

### Positive
- **Limit Bypass**: Seamlessly subscribes to hundreds of trading pairs across multiple exchange shards.
- **Resilience**: Independent reconnection loops; if Shard #1 disconnects, Shards #2 and #3 continue streaming without interruption.
- **Deterministic RPC**: Asynchronous exchange orders receive synchronous-like Go channel ergonomics via `RequestDispatcher.Dispatch(ctx, id, sendFn)`.

### Negative
- Higher memory footprint than a single raw connection.
- Shard rebalancing logic adds synchronization complexity (mitigated by fine-grained `sync.RWMutex`).

### Risks & Mitigations
- **Risk**: Unhandled `conn.Close()` error in background reconnection loop triggers linter warning.
- **Mitigation**: Explicitly ignore or log `_ = conn.Close()` in `pkg/ws/client.go` to satisfy `errcheck` static analysis.

## Related Documents
- [`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md) (Epic 3, Story 3.3; Epic 4, Story 4.2)
- [`docs/bmad/04-architecture.md`](file:///home/dangnmh/work/core-kit/docs/bmad/04-architecture.md)
- [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md)
