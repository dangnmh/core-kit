# Go Foundation API Contract (`pkg/*`)

**Document ID**: `CONTRACT-GO-API-01`  
**Package Base**: `github.com/dangnmh/core-kit/pkg`  
**Runtime**: Go 1.27.1+ (Backwards-compatible Go 1.22+)  
**Status**: `FROZEN CONTRACT`  
**Related ADRs**: [`ADR-001`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md), [`ADR-002`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-002-websocket-architecture.md), [`ADR-003`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-003-polyglot-linter-and-ci-toolchain.md)

---

## 1. Domain 1: Core & Lifecycle

### 1.1 `pkg/bootstrap`
- **Purpose**: Fail-fast configuration loading and environment validation before listening sockets bind.
- **Key Functions**:
  ```go
  type Exiter func(code int)
  func MustLoad[T any](service string, load func() (*T, error), exit Exiter) *T
  ```
- **Invariants**: If `load()` returns an error, logs critical error with service context and immediately invokes `exit(1)`.

### 1.2 `pkg/config`
- **Purpose**: Generic YAML + JSON/JSONC (`hujson`) loader with environment variable overrides and struct validation (`go-playground/validator/v10`).
- **Key Functions & Types**:
  ```go
  type Option func(*Options)
  func WithPath(p string) Option
  func WithEnvPrefix(prefix string) Option
  func Load[T any](defaultVal *T, opts ...Option) (*T, string, error)
  func LoadInto(target any, opts ...Option) (string, error)
  func LoadJSON[T any](path string) (*T, error)
  func Validate(target any) error
  ```

### 1.3 `pkg/lifecycle`
- **Purpose**: Coordinated 5-phase graceful shutdown manager for HTTP servers, background workers, and WebSocket pools.
- **Phases**:
  1. `PhasePreStop`: Signal received; drain initial notifications.
  2. `PhaseStopIngress`: Stop receiving new external requests (`/readyz` returns 503).
  3. `PhaseDrain`: Allow in-flight requests and queue batches to finish.
  4. `PhaseStopEgress`: Close outgoing database pools, message queues, and upstream connections.
  5. `PhasePostStop`: Flush logs and cleanup local resources.
- **Key Types & Functions**:
  ```go
  type Phase int
  type HookFunc func(ctx context.Context) error
  type Server interface {
      Start(ctx context.Context) error
      Stop(ctx context.Context) error
  }
  type Manager struct { ... }
  func NewManager(opts ...Option) *Manager
  func (m *Manager) AppendHook(phase Phase, name string, fn HookFunc)
  func (m *Manager) Run(ctx context.Context) error
  func Register[T Server](m *Manager, name string, s T)
  ```

### 1.4 `pkg/version`
- **Purpose**: Read-only build-time metadata populated via linker flags (`-ldflags`).
- **Exported Variables**:
  ```go
  var (
      Version   = "dev"
      Commit    = "none"
      BuildTime = "unknown"
  )
  ```

---

## 2. Domain 2: Telemetry & Observability

### 2.1 `pkg/logger`
- **Purpose**: Unified `log/slog` structured JSON logger with W3C trace context injection, dynamic level filtering, deduplication, and dual-mode attribute mapping.
- **Key Functions & Types**:
  ```go
  func ParseLevel(lvl string) slog.Level
  func SetLevel(lvl string)
  func GetLevel() slog.Level
  func NewLogger(service string, level slog.Leveler, w io.Writer) *slog.Logger
  func SetupLogging(service string) *slog.Logger
  func SetupLoggingWithLevel(service, levelStr string) *slog.Logger
  func Get() *slog.Logger
  func SetDefault(l *slog.Logger)
  func Debug(msg string, args ...any)
  func Info(msg string, args ...any)
  func Warn(msg string, args ...any)
  func Error(msg string, args ...any)
  func DebugContext(ctx context.Context, msg string, args ...any)
  func InfoContext(ctx context.Context, msg string, args ...any)
  func WarnContext(ctx context.Context, msg string, args ...any)
  func ErrorContext(ctx context.Context, msg string, args ...any)
  type TraceHandler struct { ... }
  type DedupHandler struct { ... }
  ```
- **Canonical Schema Parity (ADR-001)**:
  - Produces single-line JSON with keys: `timestamp`, `severity`, `service`, `logger`, `message`, `caller`, `trace_id`, `span_id`, and `context`.

### 2.2 `pkg/httplog`
- **Purpose**: High-performance HTTP request/response logger with rule-based PII sanitization (redact, mask, sha256).
- **Key Functions & Types**:
  ```go
  type Engine interface { ... }
  type Strategy string // "redact", "mask", "sha256"
  func GinMiddleware(engine Engine) gin.HandlerFunc
  func NewRoundTripper(base http.RoundTripper, engine Engine) http.RoundTripper
  func SanitizeJSON(body []byte, rules []CompiledFieldRule, maskVal string) string
  func SanitizeForm(body []byte, rules []CompiledFieldRule, maskVal string) string
  ```

### 2.3 `pkg/tracer`
- **Purpose**: OpenTelemetry W3C distributed traceparent propagation, OTLP HTTP exporter, and tracer provider initialization.
- **Key Functions & Types**:
  ```go
  type OTelConfig struct {
      ServiceName string
      Endpoint    string
      Insecure    bool
  }
  func InitOTel(ctx context.Context, cfg OTelConfig) (*sdktrace.TracerProvider, func(context.Context) error, error)
  func ExtractHTTP(ctx context.Context, header http.Header) context.Context
  func InjectHTTP(ctx context.Context, header http.Header)
  func ExtractTraceparent(ctx context.Context, traceparent string) context.Context
  func InjectTraceparent(ctx context.Context) string
  func ExtractTraceAndSpanID(ctx context.Context) (traceID string, spanID string)
  func StartSpan(ctx context.Context, tracerName, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
  ```

### 2.4 `pkg/tracectx`
- **Purpose**: Context correlation utilities for local request IDs (`x-request-id`).
- **Key Functions**:
  ```go
  func NewID() string
  func RequestID(ctx context.Context) string
  func WithRequestID(ctx context.Context) context.Context
  func WithRequestIDValue(ctx context.Context, id string) context.Context
  ```

---

## 3. Domain 3: Networking & Resilience

### 3.1 `pkg/httpclient`
- **Purpose**: Generic type-safe HTTP client with connection pooling, proxy routing, rate-limit retry, and OTel tracing.
- **Key Functions & Types**:
  ```go
  type Client struct { ... }
  type PoolConfig struct {
      MaxIdleConns        int
      MaxIdleConnsPerHost int
      IdleConnTimeout     time.Duration
      LocalAddr           string
      ProxyURL            string
  }
  func NewClient(opts ...Option) *Client
  func NewPool(cfg PoolConfig) *http.Client
  func Get[T any](ctx context.Context, c *Client, path string, headers ...map[string]string) (*T, error)
  func Post[T any](ctx context.Context, c *Client, path string, reqBody any, headers ...map[string]string) (*T, error)
  ```

### 3.2 `pkg/httpserver`
- **Purpose**: Production Gin engine with Uber Fx lifecycle hooks, CORS, health probes (`/healthz`, `/readyz`), and OTel middleware.
- **Key Functions & Types**:
  ```go
  type Config struct {
      Port         int
      ReadTimeout  time.Duration
      WriteTimeout time.Duration
      AllowOrigins []string
  }
  func NewEngine(p EngineParams) *gin.Engine
  func NewHTTPServer(p Params) *http.Server
  func BearerTokenAuth(token string) gin.HandlerFunc
  func TraceMiddleware(serviceName string) gin.HandlerFunc
  ```

### 3.3 `pkg/ws`
- **Purpose**: Sharded multiplexed WebSocket connection pool with topic routing and type-safe asynchronous RPC request dispatcher (`ADR-002`).
- **Key Functions & Types**:
  ```go
  type ClientOption func(*Client)
  type Handler func(data []byte)
  type Pool struct { ... }
  func NewPool(wsURL string, maxPairs int, logger *slog.Logger, opts ...ClientOption) *Pool
  func (p *Pool) Subscribe(topic string, subMsg, unsubMsg any, handler Handler) error
  func (p *Pool) Unsubscribe(topic string) error
  func (p *Pool) Start(ctx context.Context) error
  func (p *Pool) Stop()

  type RequestDispatcher[T any] struct { ... }
  func NewRequestDispatcher[T any](timeout time.Duration) *RequestDispatcher[T]
  func (d *RequestDispatcher[T]) Dispatch(ctx context.Context, reqID string, sendFn func() error) (*T, error)
  func (d *RequestDispatcher[T]) HandleResponse(reqID string, resp *T) bool
  ```

### 3.4 `pkg/wsclient` & `pkg/wsserver`
- **`pkg/wsclient`**: Callback-oriented auto-reconnecting WebSocket client.
  ```go
  type Client struct { ... }
  func New(opts ...Option) *Client
  func (c *Client) SendPayload(payload any) error
  ```
- **`pkg/wsserver`**: High-concurrency broadcast WebSocket server.
  ```go
  type Server struct { ... }
  func New(opts ...Option) *Server
  func (s *Server) BroadcastJSON(v any) BroadcastResult
  ```

### 3.5 `pkg/ratelimit`
- **Purpose**: Weighted rate limiter with global capacity and per-path token cost tiers.
- **Key Types & Functions**:
  ```go
  type ExchangeRateLimiter struct { ... }
  type WeightedRateLimiter = ExchangeRateLimiter
  func NewExchangeRateLimiter(rate int, interval time.Duration) *ExchangeRateLimiter
  func (l *ExchangeRateLimiter) Wait(ctx context.Context, endpoint string) error
  func WithSkipRateLimit(ctx context.Context) context.Context
  ```

### 3.6 `pkg/resilience`
- **Purpose**: Generic Circuit Breaker and Full-Jitter Exponential Backoff.
- **Key Types & Functions**:
  ```go
  type State int // StateClosed, StateHalfOpen, StateOpen
  type Breaker struct { ... }
  func NewBreaker(name string, opts ...BreakerOption) *Breaker
  func Execute[T any](ctx context.Context, b *Breaker, fn func(context.Context) (T, error)) (T, error)
  type Backoff struct { ... }
  func (b *Backoff) Duration(attempt int) time.Duration
  ```

### 3.7 `pkg/ticker`
- **Purpose**: Context-aware periodic background loop with uniform random jitter.
- **Key Functions**:
  ```go
  type Task func() bool
  func Run(ctx context.Context, interval time.Duration, task Task)
  func RunImmediate(ctx context.Context, interval time.Duration, task Task)
  func RunWithJitter(ctx context.Context, baseInterval, jitter time.Duration, task Task)
  ```

### 3.8 `pkg/eventbus`
- **Purpose**: In-memory Watermill pub/sub bus with message deduplication and audit trail.
- **Key Types & Functions**:
  ```go
  type Bus struct { ... }
  func NewBus(logger *slog.Logger) *Bus
  func (b *Bus) Publish(topic string, msg any) error
  func (b *Bus) Subscribe(topic string, handler func(msg any) error) error
  ```

---

## 4. Domain 4: Data Structures & Algorithms

### 4.1 `pkg/ringbuffer`
- **Purpose**: Thread-safe generic circular ring buffer with sequence numbers and window queries.
- **Allocations**: 0 allocs/op in steady-state overwrite.
- **Interface**:
  ```go
  type RingBuffer[T any] struct { ... }
  func New[T any](capacity int) *RingBuffer[T]
  func (r *RingBuffer[T]) Push(item T) int64
  func (r *RingBuffer[T]) PushWithAssign(item T, assignSeq func(item T, seq int64)) int64
  func (r *RingBuffer[T]) Latest() (T, bool)
  func (r *RingBuffer[T]) Since(seq int64) []T
  func (r *RingBuffer[T]) LatestSequenceID() int64
  func (r *RingBuffer[T]) Len() int
  func (r *RingBuffer[T]) Cap() int
  ```

### 4.2 `pkg/safemap`
- **Purpose**: Thread-safe generic RWMutex-backed map with atomic compute operations.
- **Interface**:
  ```go
  type SafeMap[K comparable, V any] struct { ... }
  func New[K comparable, V any]() *SafeMap[K, V]
  func (m *SafeMap[K, V]) Set(key K, val V)
  func (m *SafeMap[K, V]) Get(key K) (V, bool)
  func (m *SafeMap[K, V]) Delete(key K)
  func (m *SafeMap[K, V]) Compute(key K, fn func(val V, exists bool) V) V
  func (m *SafeMap[K, V]) GetOrSet(key K, val V) (V, bool)
  func (m *SafeMap[K, V]) Keys() []K
  func (m *SafeMap[K, V]) Len() int
  ```

### 4.3 `pkg/ttlcache`
- **Purpose**: In-memory key-value cache with per-item TTL and background janitor.
- **Interface**:
  ```go
  type TTLCache[K comparable, V any] struct { ... }
  func New[K comparable, V any](cleanupInterval time.Duration) *TTLCache[K, V]
  func (c *TTLCache[K, V]) Set(key K, value V, ttl time.Duration)
  func (c *TTLCache[K, V]) Get(key K) (V, bool)
  func (c *TTLCache[K, V]) Close()
  ```

### 4.4 `pkg/priorityqueue`
- **Purpose**: Thread-safe generic Min/Max heap.
- **Interface**:
  ```go
  type PriorityQueue[T any] struct { ... }
  func New[T any](less func(a, b T) bool) *PriorityQueue[T]
  func (pq *PriorityQueue[T]) Push(item T)
  func (pq *PriorityQueue[T]) Pop() (T, bool)
  func (pq *PriorityQueue[T]) Peek() (T, bool)
  func (pq *PriorityQueue[T]) Len() int
  ```

### 4.5 `pkg/quadtree`
- **Purpose**: 2D spatial indexing quadtree for proximity and boundary searches.
- **Interface**:
  ```go
  type Point struct { X, Y float64 }
  type Rect struct { MinX, MinY, MaxX, MaxY float64 }
  type Quadtree[T any] struct { ... }
  func New[T any](boundary Rect, capacity int) *Quadtree[T]
  func (q *Quadtree[T]) Insert(pt Point, data T) bool
  func (q *Quadtree[T]) Query(rangeRect Rect) []T
  ```

---

## 5. Domain 5: Financial Precision & Math

### 5.1 `pkg/decmath`
- **Purpose**: Precision-safe math wrapping `shopspring/decimal` for exchange price and volume calculations.
- **Key Functions & Types**:
  ```go
  type Decimal = decimal.Decimal
  func FromFloat(v float64) Decimal
  func ToFloat(v Decimal) float64
  func SnapToTickFloor(price, tick float64) float64
  func SnapToTickCeil(price, tick float64) float64
  func SnapDecimalToTickFloor(price, tick Decimal) Decimal
  func SnapDecimalToTickCeil(price, tick Decimal) Decimal
  func RoundToScale(v float64, scale int) float64
  func FloorToScale(v float64, scale int) float64
  func CeilToScale(v float64, scale int) float64
  func Add(a, b float64) float64
  func Sub(a, b float64) float64
  func Mul(a, b float64) float64
  func Div(a, b float64) float64
  func Equal(a, b float64) bool
  ```
- **Precision Invariant**: Never use raw `float64` arithmetic for exchange order execution.

### 5.2 `pkg/tradecalc`
- **Purpose**: Limit IOC execution price, slippage, orderbook wall detection, and SL/TP calculators.
- **Key Functions**:
  ```go
  type Side string // "BUY", "SELL"
  func CalculateIOCPrice(side Side, bestBid, bestAsk, maxPriceDiffPercent, priceUnit float64, priceScale int) (float64, error)
  func CalculateSlippage(refPrice, maxPriceDiffPercent, priceUnit float64) float64
  func CalculateStopLossPrice(side Side, entryPrice, slPct, priceUnit float64, priceScale int) float64
  func CalculateStaticTakeProfitPrice(side Side, entryPrice, tpPct, priceUnit float64, priceScale int) float64
  func FindWallPrice(entryPrice float64, levels []OrderBookEntry, side Side, minWallLevels int, wallThreshold float64) float64
  ```

---

## 6. Domain 6: Types & Serialization

### 6.1 `pkg/idgen` & `pkg/idutil` (Consolidation Contract)
- **Authoritative Implementation**: `pkg/idgen`
  ```go
  const Alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
  func NanoID(size int) string
  func NanoIDWithAlphabet(size int, alphabet string) string
  func NewPrefixedID(prefix string, byteLen ...int) string
  ```
- **Deprecated Alias (`pkg/idutil`)**:
  ```go
  // Deprecated: idutil is deprecated. Use github.com/dangnmh/core-kit/pkg/idgen instead.
  const Alphabet = idgen.Alphabet
  var NanoID = idgen.NanoID
  var NanoIDWithAlphabet = idgen.NanoIDWithAlphabet
  ```

### 6.2 `pkg/jsonutil` & `pkg/xjson`
- **`pkg/jsonutil`**: Generic Unmarshal helper:
  ```go
  func Unmarshal[T any](data []byte) (T, error)
  ```
- **`pkg/xjson`**: Resilient numbers, flex-time unmarshaling, and sonic/json codec:
  ```go
  type Number string
  type FlexTime int64
  func ToFloat64(n Number) float64
  func ToInt64(n Number) int64
  func Marshal(v any) ([]byte, error)
  func Unmarshal(data []byte, v any) error
  ```

### 6.3 `pkg/ptr`, `pkg/strutil`, `pkg/maputil`, `pkg/formatutil`, `pkg/types`
- **`pkg/ptr`**: `To[T](v T) *T`, `Value[T](p *T, def T) T`, `ValueOrZero[T](p *T) T`, `Equal[T](a, b *T) bool`.
- **`pkg/strutil`**: `TruncateRunes(s string, maxRunes int) string` (Rune-boundary safe).
- **`pkg/maputil`**: `GetString`, `GetInt`, `GetFloat`, `GetBool` (Panic-free map accessors).
- **`pkg/formatutil`**: `FormatCompactUSD`, `FormatUSDWithCommas`, `FormatPriceWithCommas`.
- **`pkg/types`**: `type Duration time.Duration` (JSON unmarshaling supporting `"30s"` and `30000`).

---

## 7. Domain 7: AI & LLM Integration

### 7.1 `pkg/llm`
- **Purpose**: OpenAI-compatible client, structured JSON extractor (`ExtractJSON[T]`), thinking tag scrubber (`<think>...`), and JSONL audit logging.
- **Key Functions & Types**:
  ```go
  type Client struct { ... }
  type Config struct {
      APIKey  string
      BaseURL string
      Model   string
  }
  func NewClient(cfg Config, opts ...ClientOption) *Client
  func CleanText(content string) string
  func ExtractJSON[T any](content string) (*T, error)
  func GenerateJSON[T any](ctx context.Context, c *Client, messages []openai.ChatCompletionMessage) (*T, error)
  func SafeLog(logger DebugLogger, record DebugRecord) error
  ```
- **Reuse-Over-Reinvent Invariant**: Uses `pkg/strutil.TruncateRunes` for error string truncation (AUDIT-04 / Story 3.2).
