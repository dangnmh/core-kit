# Core-Kit (`github.com/dangnmh/core-kit`)

Enterprise polyglot foundation platform uniting **Go 1.27+**, **Python 3.14+**, and **TypeScript/Bun 1.4+**.  
Engineered for zero-alloc concurrency, financial-grade precision math, resilient networking, and unified distributed observability.

---

## 1. Documentation & Governance Links

`core-kit` strictly implements the **BMAD 4-Layer Monorepo HADS Documentation Standard (HADS 1.0)**:

- 🏛️ **Constitution & AI Guardrails**: [`AGENTS.md`](file:///home/dangnmh/work/core-kit/AGENTS.md) (`CONST-01..08` non-negotiable invariants, `<= 120` lines)
- 🗺️ **Dense AI Context Sitemap**: [`llms.txt`](file:///home/dangnmh/work/core-kit/llms.txt) (Machine-readable AI router)
- 📚 **Master Documentation Hub**: [`docs/README.md`](file:///home/dangnmh/work/core-kit/docs/README.md) (Single navigation index for all architecture, ADRs & runbooks)
- 🧭 **Living Feature & Traceability Matrix**: [`docs/06-feature-matrix-and-roadmap.md`](file:///home/dangnmh/work/core-kit/docs/06-feature-matrix-and-roadmap.md)
- 📜 **Release Ledger**: [`CHANGELOG.md`](file:///home/dangnmh/work/core-kit/CHANGELOG.md) (Semantic Versioning & Keep a Changelog)

### Nearest-Wins Component Guides
- **Go Foundation (`pkg/*`)**: [`pkg/README.md`](file:///home/dangnmh/work/core-kit/pkg/README.md) · Hub: [`docs/pkg/README.md`](file:///home/dangnmh/work/core-kit/docs/pkg/README.md)
- **Python Telemetry (`py/logger-py`)**: [`py/logger-py/README.md`](file:///home/dangnmh/work/core-kit/py/logger-py/README.md) · Hub: [`docs/py/README.md`](file:///home/dangnmh/work/core-kit/docs/py/README.md)
- **TypeScript Telemetry (`ts/logger-ts`)**: [`ts/logger-ts/README.md`](file:///home/dangnmh/work/core-kit/ts/logger-ts/README.md) · Hub: [`docs/ts/README.md`](file:///home/dangnmh/work/core-kit/docs/ts/README.md)

---

## 2. Repository Layout

```text
core-kit/
├── AGENTS.md               # [Layer 0] Project Constitution & Global AI Guardrails (<= 120 lines)
├── llms.txt                # [Layer 0] Dense AI Context Sitemap & Router (<= 120 lines)
├── README.md               # [Layer 0] Developer onboarding & component overview
├── CHANGELOG.md            # [Layer 0] Master release ledger & semantic version history
├── Makefile                # Unified multi-runtime orchestration (make test, lint, ci)
├── go.mod                  # module github.com/dangnmh/core-kit
├── pkg/                    # 34 Go generic & infrastructure packages
│   ├── bootstrap/          # Fail-fast configuration bootstrapper (MustLoad[T], CONST-06)
│   ├── config/             # Generic YAML + JSON/JSONC (hujson) loader with env expansion
│   ├── decmath/            # Financial decimal math & tick-snapping (CONST-05)
│   ├── eventbus/           # In-memory Watermill pub/sub bus with deduplication
│   ├── formatutil/         # Currency, compact USD, commas & duration formatters
│   ├── httpclient/         # Generic HTTP client (Get[T]/Post[T]), pooling, proxy & retry
│   ├── httplog/            # Rule-based HTTP inbound/outbound logger & PII sanitizer
│   ├── httpserver/         # Gin engine with Uber Fx, 10MB body cap & fail-closed Bearer auth
│   ├── idgen/              # Cryptographic Prefixed IDs & NanoID generator (CONST-SEC-05)
│   ├── idutil/             # Deprecated backward-compatible alias delegating to idgen
│   ├── jsonutil/           # Generic JSON unmarshal helper (Unmarshal[T])
│   ├── lifecycle/          # 5-phase graceful shutdown manager (CONST-07)
│   ├── llm/                # OpenAI-compatible client, structured JSON extractor & scrubber
│   ├── logger/             # Unified slog dual-mode JSON logger with W3C trace correlation (ADR-001)
│   ├── maputil/            # Panic-free typed extractors for map[string]any
│   ├── priorityqueue/      # Thread-safe generic Min/Max-Heap (PriorityQueue[T])
│   ├── ptr/                # Generic pointer utilities (To[T], Value[T], Equal[T])
│   ├── quadtree/           # Thread-safe generic 2D spatial index (Quadtree[T])
│   ├── ratelimit/          # Multi-tier weighted rate limiter (WeightedRateLimiter)
│   ├── resilience/         # Full-jitter exponential Backoff & Circuit Breaker (Execute[T])
│   ├── ringbuffer/         # Thread-safe circular buffer with sequence indexing (RingBuffer[T])
│   ├── safemap/            # Thread-safe generic RWMutex map with atomic Compute (SafeMap[K, V])
│   ├── strutil/            # UTF-8 rune-safe string truncation (TruncateRunes)
│   ├── ticker/             # Context-aware periodic runner with uniform random jitter
│   ├── tracectx/           # Request ID context correlation
│   ├── tracer/             # OpenTelemetry W3C traceparent propagation & OTLP HTTP exporter
│   ├── tradecalc/          # Limit price, slippage, and orderbook wall calculators
│   ├── ttlcache/           # Thread-safe generic in-memory TTL cache with background janitor
│   ├── types/              # JSON duration unmarshaler ("30s" and numeric milliseconds)
│   ├── version/            # Build-time ldflags metadata (Version, Commit, BuildTime)
│   ├── ws/                 # Multiplexed WebSocket Client, Sharded Connection Pool & RPC Dispatcher (ADR-002)
│   ├── wsclient/           # Auto-reconnecting WebSocket client with typed payload sending
│   ├── wsserver/           # High-throughput WebSocket broadcast server
│   └── xjson/              # Flexible JSON types (Number, FlexTime) & codec wrappers
├── ts/
│   └── logger-ts/          # @dangnmh/logger-ts — Bun/Node structured JSON logger
├── py/
│   └── logger-py/          # logger-py — Python 3.14+ structured JSON logger with emoji stripping
└── docs/                   # BMAD 4-Layer Documentation Platform, ADRs & SRE Runbooks
```

---

## 3. Usage Across Microservices

### 3.1 Go Services (`go.mod`)
```go
require github.com/dangnmh/core-kit v1.0.0

// Local monorepo development before tagging:
// replace github.com/dangnmh/core-kit => ../core-kit
```

### 3.2 TypeScript & Bun Services (`package.json`)
```json
{
  "dependencies": {
    "@dangnmh/logger-ts": "file:../core-kit/ts/logger-ts"
  }
}
```

### 3.3 Python Services (`pyproject.toml` / `uv`)
```bash
uv add --editable ../core-kit/py/logger-py
```

---

## 4. Multi-Runtime Developer Commands

All workflows are orchestrated via the root `Makefile` without machine-specific paths:

| Workflow | Command | Target Actions |
| :--- | :--- | :--- |
| **Test** | `make test` | Runs Go (`go test -race ./...`), TS (`bun test`), and Py (`pytest`). |
| **Lint** | `make lint` | Runs `golangci-lint run ./...`, `bun lint`, `uv ruff check`, and `make doc-lint`. |
| **Format** | `make format` | Formats Go (`gofmt`), TS (`bun run format`), and Py (`ruff format`). |
| **Doc Gate** | `make doc-lint` | Validates line count caps (`<= 120` lines) and checks 0 broken links. |
| **Full CI** | `make ci` | Runs `doc-lint`, `lint`, and `test` across all ecosystems. |
