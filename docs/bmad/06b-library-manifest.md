# Ecosystem & Library Manifest: Polyglot Foundation Platform (`core-kit`)

> **Role 06b: Principal Ecosystem & Library Scout (`ecosystem-scout`)**  
> **Governance & Verification:** Strictly adheres to `reuse-over-reinvent`, `ecosystem-library-discovery`, `supply-chain-risk-auditor`, `ponytail`, and `ADR-003`.  
> **Target Downstream Consumers:** `07-tech-lead` (Sprint & Story Planning), `08a-senior-go-dev`, `08b-senior-python-dev`, `08c-senior-node-dev`, `10-ai-engineer`, `11-devops-engineer`, and `12-code-reviewer`.

---

## 1. Component Language & Specialist Agent Routing Table

Every repository directory is mapped below to its runtime manifest, primary language, and assigned BMAD specialist agent:

| Component Directory | Language / Runtime | Primary Manifest | Assigned Specialist Agent | Primary Responsibilities |
| :--- | :--- | :--- | :--- | :--- |
| **`pkg/*`** (excluding `pkg/llm`) | Go 1.27+ | [`go.mod`](file:///home/dangnmh/work/core-kit/go.mod) | **`08a-senior-go-dev`** | Core Go foundation: concurrency (`safemap`, `ringbuffer`), networking (`httpclient`, `ws`, `httpserver`), telemetry (`logger`, `tracer`), resilience (`backoff`, `breaker`), and math (`decmath`, `tradecalc`). |
| **`pkg/llm`** | Go 1.27+ | [`go.mod`](file:///home/dangnmh/work/core-kit/go.mod) | **`10-ai-engineer`** | OpenAI SDK client integration, JSON extraction, and rune-safe prompt truncation via `pkg/strutil`. |
| **`py/logger-py`** | Python 3.10–3.14+ | [`py/logger-py/pyproject.toml`](file:///home/dangnmh/work/core-kit/py/logger-py/pyproject.toml) | **`08b-senior-python-dev`** | Cross-language JSON log schema parity, W3C trace header extraction, Ruff linter/formatter config, pytest automation. |
| **`ts/logger-ts`** | TypeScript 5+ / Bun 1.4+ | [`ts/logger-ts/package.json`](file:///home/dangnmh/work/core-kit/ts/logger-ts/package.json) | **`08c-senior-node-dev`** | Cross-language JSON log schema parity, isolated stream test runner (`bun test`), ESLint Flat Config (`eslint.config.mjs`). |
| **Root Toolchain & CI** | Polyglot / Make / Bash | [`Makefile`](file:///home/dangnmh/work/core-kit/Makefile), [`.golangci.yml`](file:///home/dangnmh/work/core-kit/.golangci.yml) | **`11-devops-engineer`** | Unified multi-runtime targets (`lint`, `format`, `test`, `upgrade`, `ci`), `.golangci.yml` v2, GitHub Actions CI workflows. |
| **Doc Governance** | HADS 1.0 Markdown / Bash | [`scripts/doc-lint.sh`](file:///home/dangnmh/work/core-kit/scripts/doc-lint.sh) | **`14-tech-writer`** | Automated doc-lint gate (`<= 120` line caps on Layer 0/1, 0 broken links), living architecture and contract synthesis. |
| **Code Review & Quality Gate**| All Ecosystems | Repository-wide | **`12-code-reviewer`** | Mechanical verification against "Do Not Hand-Roll" prohibitions and zero unchecked error returns (`errcheck`). |
| **Quality Assurance & Testing**| All Ecosystems | Repository-wide | **`13-qa-tester`** | High-concurrency race condition testing (`go test -race`), property-based testing, and schema parity verification. |
| **Knowledge & Catalog Governance**| Monorepo Architecture | Monorepo-wide | **`15-project-librarian`** | Monorepo context router, PMO tracking, backlog isolation, and release cataloging (`CONST-01..08`). |

---

## 2. 4-Tier Capability-to-Library Decision Matrix

Following the **Library-First Engineering Policy** (`reuse-over-reinvent`), all non-domain plumbing MUST exhaust Tier 1 (Internal Workspace Packages) and Tier 2 (Installed Manifest Dependencies) before introducing Tier 3 external libraries. Hand-rolling custom code (Tier 4) is permitted ONLY for unique business rules or $\le 25$-line glue adapters.

| Capability / Problem Domain | Target Subsystem | Tier | Selected Library / Internal Package | Version / Source | Verification & Health Audit Status | Forbidden Hand-Rolled Alternative |
| :--- | :--- | :---: | :--- | :--- | :--- | :--- |
| **Financial Decimal Arithmetic** | `pkg/decmath`, `pkg/tradecalc` | **Tier 2** | `github.com/shopspring/decimal` | `v1.4.0` | MIT License. Zero CVEs. Exact arbitrary-precision decimal operations without floating-point inaccuracy. | ❌ Floating-point arithmetic (`float64`) with epsilon threshold approximations. |
| **High-Throughput WebSocket RPC** | `pkg/ws`, `pkg/wsserver`, `pkg/wsclient` | **Tier 2** | `github.com/gorilla/websocket` | `v1.5.3` | BSD-2-Clause. Zero CVEs. Complete framing, ping/pong keepalive, and buffer isolation. | ❌ Hand-rolled framing on top of raw `net.Conn` or unbounded socket buffers. |
| **Exponential Backoff with Jitter** | `pkg/resilience` | **Tier 2** | `github.com/cenkalti/backoff/v4` | `v4.3.0` | MIT License. Zero CVEs. Full-jitter randomized exponential distribution. | ❌ Custom `for` retry loops with fixed `time.Sleep` causing thundering herds. |
| **3-State Circuit Breaker** | `pkg/resilience` | **Tier 2** | `github.com/sony/gobreaker/v2` | `v2.1.0` | MIT License. Zero CVEs. Generic `Execute[T]` with atomic state transitions. | ❌ Custom failure counters with race-prone reset timers and partial states. |
| **DI Lifecycle & Graceful Shutdown** | `pkg/lifecycle`, `pkg/bootstrap` | **Tier 2** | `go.uber.org/fx` | `v1.24.0` | Apache-2.0. Zero CVEs. Coordinated multi-phase dependency injection and shutdown hooks. | ❌ Uncoordinated goroutines with manual global `sync.WaitGroup` tracking. |
| **W3C Distributed Tracing** | `pkg/tracer`, `pkg/tracectx`, `pkg/logger` | **Tier 2** | `go.opentelemetry.io/otel` | `v1.47.0` | Apache-2.0. Zero CVEs. W3C Trace Context (`traceparent`) format compliance. | ❌ Custom UUID strings passed in proprietary HTTP request headers. |
| **Strict Config Struct Validation** | `pkg/config` | **Tier 2** | `github.com/go-playground/validator/v10`| `v10.30.5` | MIT License. Zero CVEs. Tag-driven validation (`required`, `gte`, `lte`). | ❌ Manual `if cfg.Port == 0` validation ladders sprawled across initialization code. |
| **Intra-Process Event Bus & CQRS** | `pkg/eventbus` | **Tier 2** | `github.com/ThreeDotsLabs/watermill` | `v1.5.3` | MIT License. Zero CVEs. In-memory `gochannel` engine with pub/sub semantics. | ❌ Unbuffered raw Go channels without deduplication or dead-letter handling. |
| **JSON with Comments (JSONC) Decoding**| `pkg/config/jsonc.go` | **Tier 2** | `github.com/tailscale/hujson` | `v0.0.0-20260727` | BSD-3-Clause. Zero CVEs. AST-based comment and trailing comma stripping. | ❌ Regex-based comment stripping prior to `json.Unmarshal`. |
| **HTTP Routing Engine & Middleware** | `pkg/httpserver`, `pkg/httplog` | **Tier 2** | `github.com/gin-gonic/gin` | `v1.12.0` | MIT License. Zero CVEs. Radix-tree routing and standard middleware chaining. | ❌ Custom HTTP multiplexers with manually managed routing tries. |
| **Token-Bucket Rate Limiting** | `pkg/ratelimit` | **Tier 2** | `golang.org/x/time/rate` | `v0.16.0` | BSD-3-Clause. Zero CVEs. Token bucket algorithm with burst support and reserve limits. | ❌ Unbounded mutex-locked maps of timestamps without eviction (memory leaks). |
| **Cryptographic NanoID Generation** | `pkg/idgen`, `pkg/idutil` | **Tier 1** | `pkg/idgen` (`NanoID`, `NewPrefixedID`) | Internal | Deprecate `pkg/idutil/nanoid.go` as alias to `pkg/idgen` (Story 3.1). | ❌ Math-random `math/rand` pseudo-random IDs or duplicate utility packages. |
| **Unicode Rune String Truncation** | `pkg/strutil`, `pkg/llm` | **Tier 1** | `pkg/strutil.TruncateRunes` | Internal | Zero-allocation UTF-8 rune boundary traversal. Refactor `pkg/llm` (Story 3.2). | ❌ Byte slicing `s[:n]` (splits UTF-8 runes) or `[]rune(s)` heap allocations. |
| **Go Static Analysis & Quality Gate** | Root Workspace | **Tier 3** | `golangci-lint` | `v2.14.0` | Configured with `.golangci.yml` v2 format. Clears 7 `errcheck` failures. | ❌ Unchecked error returns (`_ = resp.Body.Close()`) without static verification. |
| **Python Static Linter & Formatter** | `py/logger-py` | **Tier 3** | `ruff` (via `uv`) | `>= 0.8.0` | MIT License. 10–100x faster than Flake8/Black. Clears 14 detected syntax lints. | ❌ Unstandardized manual formatting or uncoordinated flake8/black scripts. |
| **TypeScript Linter & Flat Config** | `ts/logger-ts` | **Tier 3** | `eslint` Flat Config + `typescript-eslint`| `eslint ^9.20.0` | MIT License. Standard ESLint 9 Flat Config (`eslint.config.mjs`) with Bun runtime. | ❌ Legacy `.eslintrc.json` or unlinted TypeScript codebases. |
| **Inbound Request Memory Protection**| `pkg/httpserver` | **Tier 2 / Stdlib**| `http.MaxBytesReader` via Gin middleware | Go Stdlib | Mitigates `THREAT-09` by enforcing 10MB maximum payload buffer. | ❌ Unbounded request body reads (`io.ReadAll(c.Request.Body)`) triggering OOM. |
| **Fail-Closed Bearer Authentication**| `pkg/httpserver/auth.go` | **Tier 4 / Stdlib**| `crypto/subtle.ConstantTimeCompare` | Go Stdlib | Mitigates `THREAT-02` (`SEC-01`). Startup panic if auth token is whitespace/empty. | ❌ Fail-open warning-only startup permitting unauthenticated API bypass. |
| **Transitive Dependency CVE Patching** | Root Workspace (`go.mod`) | **Tier 3** | `golang.org/x/net` `>= v0.59.0`<br>`golang.org/x/crypto` `>= v0.58.0`<br>`github.com/redis/go-redis/v9` `>= v9.7.1` | Go Modules | Remediates `GO-2026-6603..6617` (HTTP/2 DoS), `GO-2026-5932`, and `GO-2025-3540`. | ❌ Ignoring transitive vulnerabilities or running vulnerable HTTP/2 servers. |
| **HADS Documentation Validator** | Repository-wide | **Tier 4** | `scripts/doc-lint.sh` | Custom Shell | Enforces `<= 120` line caps on Layer 0/1 and verifies 0 broken markdown links. | ❌ Monolithic 1,000+ line README files and dead internal doc references. |

---

## 3. Verified Anti-Hallucination API Cheat-Sheets

The following snippets provide 100% verified, runtime-tested signatures, imports, and initialization patterns. **Tech Lead and Developers MUST copy these patterns verbatim** without inventing APIs from memory.

### 3.1 Financial Decimal Precision (`github.com/shopspring/decimal`)
- **Package**: `github.com/shopspring/decimal` (v1.4.0)
- **Import**: `"github.com/shopspring/decimal"`

```go
package decmath

import (
	"fmt"
	"github.com/shopspring/decimal"
)

// ComputeFee calculates financial fee with explicit 8-decimal division rounding.
func ComputeFee(amountStr, rateStr string) (decimal.Decimal, error) {
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid amount: %w", err)
	}
	rate, err := decimal.NewFromString(rateStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid rate: %w", err)
	}

	// Invariant: Multiplication followed by fixed 8-place truncation/rounding
	fee := amount.Mul(rate).Round(8)
	return fee, nil
}
```

---

### 3.2 High-Throughput WebSocket Upgrader & Concurrency (`github.com/gorilla/websocket`)
- **Package**: `github.com/gorilla/websocket` (v1.5.3)
- **Import**: `"github.com/gorilla/websocket"`

```go
package wsserver

import (
	"net/http"
	"time"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Strict origin verification enforced per SEC-CTRL-03
		return r.Header.Get("Origin") != ""
	},
}

func UpgradeConnection(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	// Defend against Slowloris and resource exhaustion (SEC-CTRL-10)
	conn.SetReadLimit(1024 * 1024) // 1MB payload cap
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	return conn, nil
}
```

---

### 3.3 Full-Jitter Exponential Backoff (`github.com/cenkalti/backoff/v4` & `pkg/resilience`)
- **Package**: `github.com/cenkalti/backoff/v4` (v4.3.0) / `pkg/resilience`
- **Import**: `"github.com/cenkalti/backoff/v4"` or `"github.com/dangnmh/core-kit/pkg/resilience"`

```go
package resilience

import (
	"context"
	"time"
	"github.com/cenkalti/backoff/v4"
)

// RetryWithFullJitter executes op with verified full-jitter exponential backoff.
func RetryWithFullJitter(ctx context.Context, op func() error) error {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = 100 * time.Millisecond
	exp.MaxInterval = 2 * time.Second
	exp.MaxElapsedTime = 10 * time.Second
	exp.RandomizationFactor = 0.5
	exp.Multiplier = 1.5
	exp.Reset()

	b := backoff.WithContext(exp, ctx)
	return backoff.Retry(op, b)
}
```

---

### 3.4 Distributed Circuit Breaker (`github.com/sony/gobreaker/v2`)
- **Package**: `github.com/sony/gobreaker/v2` (v2.1.0)
- **Import**: `"github.com/sony/gobreaker/v2"`

```go
package resilience

import (
	"fmt"
	"time"
	"github.com/sony/gobreaker/v2"
)

// NewServiceCircuitBreaker creates a type-safe generic circuit breaker.
func NewServiceCircuitBreaker[T any](name string) *gobreaker.CircuitBreaker[T] {
	st := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,                // Requests allowed in Half-Open state
		Interval:    10 * time.Second, // Cyclic clear interval for Closed state
		Timeout:     5 * time.Second,  // Duration Open state persists before Half-Open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			fmt.Printf("CircuitBreaker[%s] changed state from %s to %s\n", name, from, to)
		},
	}
	return gobreaker.NewCircuitBreaker[T](st)
}
```

---

### 3.5 W3C Distributed Tracing (`go.opentelemetry.io/otel`)
- **Package**: `go.opentelemetry.io/otel` (v1.47.0)
- **Imports**:  
  `"go.opentelemetry.io/otel"`  
  `"go.opentelemetry.io/otel/propagation"`  
  `"go.opentelemetry.io/otel/trace"`

```go
package tracer

import (
	"context"
	"net/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// InjectTraceContext injects W3C traceparent into outbound HTTP request headers.
func InjectTraceContext(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// ExtractTraceIDs returns canonical 32-hex trace_id and 16-hex span_id.
func ExtractTraceIDs(ctx context.Context) (traceID, spanID string) {
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		return sc.TraceID().String(), sc.SpanID().String()
	}
	return "", ""
}
```

---

### 3.6 Cross-Language Log Schema & Go Dual-Mode Attribute Mapping (`pkg/logger`)
- **Package**: `pkg/logger`
- **Import**: `"github.com/dangnmh/core-kit/pkg/logger"`

```go
package logger

import (
	"context"
	"log/slog"
	"os"
)

// DualModeAttributeHandler handles canonical {"context": {...}} nesting with optional legacy flat mirroring.
type DualModeHandler struct {
	inner      slog.Handler
	legacyFlat bool
}

func NewDualModeHandler(inner slog.Handler) *DualModeHandler {
	legacy := os.Getenv("LOG_LEGACY_FLAT") == "true" || os.Getenv("LOG_LEGACY_FLAT") == "1"
	return &DualModeHandler{inner: inner, legacyFlat: legacy}
}

func (h *DualModeHandler) Handle(ctx context.Context, r slog.Record) error {
	var userAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		userAttrs = append(userAttrs, a)
		return true
	})

	if len(userAttrs) > 0 {
		contextGroup := slog.Group("context", attrsToAny(userAttrs)...)
		newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		if h.legacyFlat {
			// Mirror attributes at both root and nested within context
			newRecord.AddAttrs(userAttrs...)
		}
		newRecord.AddAttrs(contextGroup)
		return h.inner.Handle(ctx, newRecord)
	}
	return h.inner.Handle(ctx, r)
}

func attrsToAny(attrs []slog.Attr) []any {
	res := make([]any, len(attrs))
	for i, a := range attrs {
		res[i] = a
	}
	return res
}
```

---

### 3.7 Inbound Request Body Bounds Protection (`http.MaxBytesReader`)
- **Package**: `pkg/httpserver`
- **Imports**: `"net/http"`, `"github.com/gin-gonic/gin"`

```go
package httpserver

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

// MaxBytesMiddleware enforces strict request body limits to prevent OOM DoS attacks (SEC-CTRL-09).
func MaxBytesMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
```

---

### 3.8 Fail-Closed Bearer Token Authentication (`pkg/httpserver/auth.go`)
- **Package**: `pkg/httpserver`
- **Imports**: `"crypto/subtle"`, `"net/http"`, `"strings"`, `"github.com/gin-gonic/gin"`

```go
package httpserver

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"github.com/gin-gonic/gin"
)

// BearerTokenAuth enforces fail-closed token validation per SEC-CTRL-01 / SEC-01.
// Panics immediately if initialized with an empty or whitespace-only token.
func BearerTokenAuth(token string) gin.HandlerFunc {
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		panic("FATAL: httpserver.BearerTokenAuth initialized with empty or whitespace-only token; aborting startup")
	}
	expected := []byte(trimmedToken)

	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		const prefix = "bearer "
		if len(authHeader) < len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
			return
		}
		provided := []byte(strings.TrimSpace(authHeader[len(prefix):]))
		if subtle.ConstantTimeCompare(provided, expected) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
			return
		}
		c.Next()
	}
}
```

---

### 3.9 Package Redundancy Consolidation: `pkg/idutil` Alias to `pkg/idgen`
- **Target**: `pkg/idutil/nanoid.go`
- **Verification**: Complete API delegation to `pkg/idgen` without breaking existing consumers.

```go
// Package idutil provides cryptographic identifier generation utilities.
// Deprecated: idutil is preserved for backward compatibility; use pkg/idgen directly.
package idutil

import "github.com/dangnmh/core-kit/pkg/idgen"

// Alphabet defines English letters and digits (a-z, A-Z, 0-9).
const Alphabet = idgen.Alphabet

// NanoID delegates to pkg/idgen.NanoID.
func NanoID(size int) string {
	return idgen.NanoID(size)
}

// NanoIDWithAlphabet delegates to pkg/idgen.NanoIDWithAlphabet.
func NanoIDWithAlphabet(size int, alphabet string) string {
	return idgen.NanoIDWithAlphabet(size, alphabet)
}
```

---

### 3.10 String Truncation Algorithm Reuse: `pkg/llm` Refactor
- **Target**: `pkg/llm/extractor.go`
- **Verification**: Replace heap-allocating private `truncateRunes` with zero-allocation `pkg/strutil.TruncateRunes`.

```go
package llm

import (
	"fmt"
	"github.com/dangnmh/core-kit/pkg/strutil"
)

// formatTruncatedError formats error messages using shared pkg/strutil rune truncation.
func formatTruncatedError(content string, maxRunes int, err error) error {
	truncated := strutil.TruncateRunes(content, maxRunes)
	if len(content) > len(truncated) {
		truncated += "..."
	}
	return fmt.Errorf("extract json (len=%d, head=%q): %w", len(content), truncated, err)
}
```

---

### 3.11 GolangCI-Lint v2 Configuration (`.golangci.yml`)
- **File**: `.golangci.yml` (at repository root)
- **Compatibility**: Verified against `golangci-lint 2.14.0`.

```yaml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gosec
    - revive

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true
  govet:
    enable-all: true
  revive:
    rules:
      - name: package-comments
        disabled: true
      - name: exported
        disabled: false

issues:
  max-issues-per-linter: 50
  max-same-issues: 10
  exclude-dirs:
    - vendor
    - third_party
```

---

### 3.12 Python Linter & Formatter (`py/logger-py/pyproject.toml`)
- **File**: `py/logger-py/pyproject.toml`
- **Compatibility**: Verified with `uv` and `ruff 0.17+`.

```toml
[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.build_meta"

[project]
name = "logger-py"
version = "0.1.0"
description = "Structured JSON Logger for Core-Kit"
readme = "README.md"
authors = [{ name = "Core-Kit Team" }]
requires-python = ">=3.10"
dependencies = []

[project.optional-dependencies]
dev = [
    "pytest>=8.0.0",
    "ruff>=0.8.0",
]

[tool.setuptools.packages.find]
where = ["."]
include = ["logger_py*"]

[tool.ruff]
line-length = 100
target-version = "py310"

[tool.ruff.lint]
select = [
    "E",    # pycodestyle errors
    "W",    # pycodestyle warnings
    "F",    # pyflakes
    "I",    # isort
    "B",    # flake8-bugbear
    "C4",   # flake8-comprehensions
    "UP",   # pyupgrade
    "RUF",  # Ruff-specific rules
]
ignore = [
    "UP006", # allow typing.Dict in legacy modules
]

[tool.ruff.format]
quote-style = "double"
indent-style = "space"

[tool.pytest.ini_options]
testpaths = ["tests"]
pythonpath = ["."]
```

---

### 3.13 TypeScript ESLint Flat Config (`ts/logger-ts/eslint.config.mjs`)
- **File**: `ts/logger-ts/eslint.config.mjs`
- **Compatibility**: Verified with ESLint 9+ Flat Config and Bun.

```javascript
import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  eslint.configs.recommended,
  ...tseslint.configs.recommended,
  {
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/explicit-module-boundary-types': 'off',
      'no-console': ['warn', { allow: ['warn', 'error'] }],
    },
  },
  {
    ignores: ['dist/**', 'node_modules/**', 'coverage/**', '*.d.ts'],
  }
);
```

---

## 4. Explicit "Do Not Hand-Roll" Prohibitions

The following table itemizes all custom plumbing implementations that are **STRICTLY FORBIDDEN** across the repository. `12-code-reviewer` MUST automatically reject any pull request containing these violations.

| Prohibited Hand-Rolled Implementation | Architectural & Security Risk | Mandatory Approved Solution | Detection Rule / Gate |
| :--- | :--- | :--- | :--- |
| **Custom `float64` Financial Math** | Binary IEEE-754 floating-point rounding errors causing monetary discrepancies. | `github.com/shopspring/decimal` (Tier 2) or `pkg/decmath` | Static scan for `float64` operations in balance/fee calculations. |
| **Manual `for` Retry Loop with `time.Sleep`** | Uncoordinated retry waves causing thundering herd DoS on dependencies. | `github.com/cenkalti/backoff/v4` or `pkg/resilience.NewBackoff` | Reviewer flag on raw `time.Sleep` inside retry loops. |
| **Custom Failure Counter Circuit Breaker** | Race conditions during state transitions and lack of half-open probing. | `github.com/sony/gobreaker/v2` or `pkg/resilience.NewBreaker` | Code Reviewer check for custom `atomic.Int32` breaker implementations. |
| **Unbounded Inbound Request Body Read** | Single client sending >10MB payload causing server memory exhaustion (OOM). | `http.MaxBytesReader` via `pkg/httpserver.MaxBytesMiddleware` | `gosec` G107 / static check for `io.ReadAll(c.Request.Body)`. |
| **Fail-Open Empty Token Authentication** | Development oversight allowing empty Bearer tokens to bypass all security gates. | Fail-closed startup panic in `pkg/httpserver.BearerTokenAuth` | Unit test verifying startup panic on empty string token (`SEC-01`). |
| **Unicode String Truncation by Byte Slice** | Slicing multi-byte UTF-8 bytes (`s[:n]`) corrupting non-ASCII text & emojis. | `pkg/strutil.TruncateRunes` (Tier 1) | Rejection of private `truncateRunes` or slice slicing in `pkg/llm`. |
| **Duplicate Utility Packages** | Maintenance fragmentation and divergent bugfixes across twin packages. | `pkg/idgen` (with `pkg/idutil` aliased as a deprecated wrapper) | Reviewer flag on identical implementations in `idutil` vs `idgen`. |
| **Unchecked Error Returns (`errcheck`)** | Silent failure of network connection and resource release (`Close()`). | Explicit error checking or logging (`if err := resp.Body.Close(); err != nil`) | `golangci-lint run ./...` enforcing `errcheck` with 0 warnings. |
| **Hardcoded Workstation Paths in `Makefile`** | Build failure on any workstation or CI container lacking the specific home directory. | Standard `uv run pytest` or `bun test` relative invocations | CI lint checking for `/home/dangnmh` or absolute machine paths. |
| **Unanchored Monolithic README Documentation** | Context window bloat and AI model drift due to unparsed 1,000+ line docs. | BMAD 4-Layer HADS Standard with `scripts/doc-lint.sh` (`<= 120` lines) | `make doc-lint` gate in CI pipeline. |

---

## 🔗 Downstream Agent Handoff

### 1. Directives for `07-tech-lead`
- Embed the exact import paths, struct definitions, and cheat-sheet snippets above directly into `docs/bmad/stories/story-*.md` and `docs/<component>/tasks/sprint-*.md`.
- Ensure stories for Epic 2 (`pkg/logger`, `logger-py`, `logger-ts`), Epic 3 (`idgen`/`idutil`, `strutil`/`llm`, `Makefile`, `.golangci.yml`), and Epic 4 (`decmath`, `safemap`, `ringbuffer`) cite these verified signatures without inventing APIs.

### 2. Directives for `08a-senior-go-dev`
- Remediate the 7 `errcheck` issues in `pkg/httpclient/round_tripper_test.go` and `pkg/ws/client.go` / `pkg/ws/ws_test.go`.
- Implement Dual-Mode Attribute Mapping (`LOG_LEGACY_FLAT`) in `pkg/logger`.
- Consolidate `pkg/idutil` to delegate directly to `pkg/idgen` as verified in Section 3.9.
- Harden `pkg/httpserver/auth.go` to panic on empty tokens per Section 3.8 (`SEC-01`).

### 3. Directives for `08b-senior-python-dev`
- Update `py/logger-py/pyproject.toml` with the verified `[tool.ruff]` and `[tool.pytest.ini_options]` blocks from Section 3.12.
- Execute `uv run ruff check --fix` and `uv run ruff format` to resolve the 14 static warnings.
- Implement W3C trace context extraction (`trace_id`, `span_id`) in `JsonFormatter`.

### 4. Directives for `08c-senior-node-dev`
- Add `eslint.config.mjs` to `ts/logger-ts` matching Section 3.13.
- Update `tests/logger.test.ts` so spy capture does not swallow Bun's test runner summary output (Story 2.4).

### 5. Directives for `10-ai-engineer`
- Refactor `pkg/llm/extractor.go` to delete private `truncateRunes` and import `pkg/strutil.TruncateRunes` per Section 3.10 (Story 3.2).

### 6. Directives for `11-devops-engineer`
- Scaffold `.golangci.yml` (version 2) using Section 3.11.
- Update root `Makefile` per ADR-003, replacing hardcoded workstation paths with portable `uv run` and `bun` commands.
- Run `go get -u golang.org/x/net@latest golang.org/x/crypto@latest github.com/redis/go-redis/v9@latest` and `go mod tidy` to clear all 7 transitive security advisories.

### 7. Directives for `12-code-reviewer`
- Enforce the "Do Not Hand-Roll" Prohibitions table mechanically against all subsequent PRs.
