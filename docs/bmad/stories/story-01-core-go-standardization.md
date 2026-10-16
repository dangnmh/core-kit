# Story 01: Go Core Foundation Standardization, Dual-Mode Telemetry & Linter Hardening

**Document ID**: `STORY-CORE-01`  
**Parent Epic**: Epic 2 & Epic 3 ([`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md))  
**Assigned Specialist Agents**: `08a-senior-go-dev` (Lead Go Engineer) & `10-ai-engineer` (for `pkg/llm`)  
**Recommended Model Tier**: `flash`  
**Component Manifest**: [`go.mod`](file:///home/dangnmh/work/core-kit/go.mod) (Go 1.27.1+)  
**Related Contracts**: [`CONTRACT-GO-API-01`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md), [`CONTRACT-GO-DATA-01`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/events-and-data.md), [`ADR-001`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md), [`ADR-002`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-002-websocket-architecture.md)  
**Status**: `READY FOR IMPLEMENTATION`  

---

## 1. Constitution Gate Check (`CONST-01..08` & `CONST-SEC-01..06`)

Before writing any code, the engineer MUST verify compliance with the following binding invariants:
- **`CONST-01` (Zero Storage)**: No persistent files, databases, or SQLite connections may be created in `pkg/*`.
- **`CONST-02` (Strict Reuse)**: Replace duplicate `pkg/idutil` with an alias delegating directly to `pkg/idgen`. Refactor `pkg/llm` to reuse `pkg/strutil.TruncateRunes`.
- **`CONST-03` (Unified Telemetry Parity)**: `pkg/logger` MUST emit single-line JSON with `"logger"` and serialize user contextual key-values under the `"context": { ... }` sub-object by default, with dual-mode root attribute mirroring activated when `LOG_LEGACY_FLAT=true` or `LOG_LEGACY_FLAT=1`.
- **`CONST-04` (Zero Data Races)**: All changes in `pkg/logger`, `pkg/idgen`, `pkg/ws`, and `pkg/httpclient` must pass `go test -race ./...` with 0 races detected.
- **`CONST-SEC-01` (Fail-Closed Auth)**: `pkg/httpserver.BearerTokenAuth` MUST panic on startup if passed an empty or whitespace-only token string.
- **`CONST-SEC-04` (Bounded Resource Allocation)**: Enforce `http.MaxBytesReader` via `pkg/httpserver.MaxBytesMiddleware`.

---

## 2. OpenSpec Capability Delta

```diff
ADDED:
+ pkg/logger.DualModeHandler: Slog Handler wrapping inner handler to nest user attributes under "context" and optionally mirror to root when LOG_LEGACY_FLAT is enabled.
+ pkg/httpserver.MaxBytesMiddleware: Gin middleware bounding incoming HTTP request bodies via http.MaxBytesReader (default 10MB).
+ pkg/httpserver.BearerTokenAuth: Startup panic validation on empty/whitespace token string (SEC-01 / CONST-SEC-01).

MODIFIED:
! pkg/logger/logger.go: Default logger initializes with DualModeHandler; replaceAttr standardizes "logger" field key.
! pkg/idutil/nanoid.go: Entire implementation replaced with type and function aliases pointing to pkg/idgen, with formal @deprecated notice.
! pkg/llm/extractor.go: Deleted private truncateRunes; imported and invoked pkg/strutil.TruncateRunes.
! pkg/httpclient/round_tripper_test.go: Enforce explicit error check or `_ = resp.Body.Close()` on 5 defer statements flagged by errcheck.
! pkg/ws/client.go & pkg/ws/ws_test.go: Handle or explicitly ignore unhandled conn.Close() and resp.Body.Close() statements.

REMOVED:
- Duplicate NanoID generation logic in pkg/idutil/nanoid.go.
- Private slice-allocating truncateRunes in pkg/llm/extractor.go.
```

---

## 3. EARS Acceptance Criteria

- **`AC-1.1` (Canonical Context Nesting)**:
  `WHEN` a developer logs custom key-value pairs via `logger.Info("Order placed", "symbol", "BTC/USDT", "amount", 1.5)`
  `THE` `pkg/logger` package `SHALL` serialize `"symbol"` and `"amount"` inside a nested `"context": {"symbol": "BTC/USDT", "amount": 1.5}` JSON object.
- **`AC-1.2` (Dual-Mode Legacy Mirroring)**:
  `IF` environment variable `LOG_LEGACY_FLAT` is set to `"true"` or `"1"`
  `THE` `pkg/logger` package `SHALL` serialize user attributes both at the root level (`"symbol": "BTC/USDT"`) and inside `"context"`.
- **`AC-1.3` (Package Redundancy Delegation)**:
  `WHEN` external or internal code calls `idutil.NanoID(21)` or `idutil.NanoIDWithAlphabet(10, alphabet)`
  `THE` `pkg/idutil` package `SHALL` delegate execution to `idgen.NanoID` and `idgen.NanoIDWithAlphabet` without duplicating RNG logic.
- **`AC-1.4` (String Truncation Memory Efficiency)**:
  `WHEN` `llm.ExtractJSON[T](content)` encounters an error
  `THE` `pkg/llm` package `SHALL` format the error string using `pkg/strutil.TruncateRunes(content, 256)` without allocating a rune slice.
- **`AC-1.5` (Fail-Closed Token Validation)**:
  `WHEN` `httpserver.BearerTokenAuth("")` or `httpserver.BearerTokenAuth("   ")` is invoked during server startup
  `THE` `pkg/httpserver` package `SHALL` panic immediately with message: `"FATAL: httpserver.BearerTokenAuth initialized with empty or whitespace-only token; aborting startup"`.
- **`AC-1.6` (Errcheck Compliance)**:
  `WHEN` `golangci-lint run ./pkg/httpclient ./pkg/ws` is executed
  `THE` linter `SHALL` exit with 0 errors, resolving all 7 previously unhandled `Close()` calls.
- **`AC-1.7` (Zero Concurrency Data Races)**:
  `WHEN` `go test -race ./pkg/...` is executed across all modified packages
  `THE` test runner `SHALL` pass 100% with 0 data races.

---

## 4. Exact File Paths & Ownership

- **Assigned Agent**: `08a-senior-go-dev`
  - Modify: [`pkg/logger/logger.go`](file:///home/dangnmh/work/core-kit/pkg/logger/logger.go)
  - Modify: [`pkg/logger/logger_test.go`](file:///home/dangnmh/work/core-kit/pkg/logger/logger_test.go)
  - Modify: [`pkg/idutil/nanoid.go`](file:///home/dangnmh/work/core-kit/pkg/idutil/nanoid.go)
  - Modify: [`pkg/idutil/nanoid_test.go`](file:///home/dangnmh/work/core-kit/pkg/idutil/nanoid_test.go)
  - Modify: [`pkg/httpserver/auth.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/auth.go)
  - Modify: [`pkg/httpserver/server.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/server.go)
  - Modify: [`pkg/httpclient/round_tripper_test.go`](file:///home/dangnmh/work/core-kit/pkg/httpclient/round_tripper_test.go)
  - Modify: [`pkg/ws/client.go`](file:///home/dangnmh/work/core-kit/pkg/ws/client.go)
  - Modify: [`pkg/ws/ws_test.go`](file:///home/dangnmh/work/core-kit/pkg/ws/ws_test.go)
- **Assigned Agent**: `10-ai-engineer`
  - Modify: [`pkg/llm/extractor.go`](file:///home/dangnmh/work/core-kit/pkg/llm/extractor.go)
  - Modify: [`pkg/llm/extractor_test.go`](file:///home/dangnmh/work/core-kit/pkg/llm/extractor_test.go)

---

## 5. Mandatory Libraries & Verified API Cheat-Sheet (`from 06b-library-manifest.md`)

> **FORBIDDEN: Do NOT hand-roll custom string slicing, retry loops, or ID generators. You MUST compose the exact functions below.**

### 5.1 Go DualMode Slog Handler (`pkg/logger`)
```go
package logger

import (
	"context"
	"log/slog"
	"os"
)

// DualModeHandler wraps an slog.Handler to group caller attributes under "context".
type DualModeHandler struct {
	inner      slog.Handler
	legacyFlat bool
}

func NewDualModeHandler(inner slog.Handler) *DualModeHandler {
	legacy := os.Getenv("LOG_LEGACY_FLAT") == "true" || os.Getenv("LOG_LEGACY_FLAT") == "1"
	return &DualModeHandler{inner: inner, legacyFlat: legacy}
}

func (h *DualModeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
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
			newRecord.AddAttrs(userAttrs...)
		}
		newRecord.AddAttrs(contextGroup)
		return h.inner.Handle(ctx, newRecord)
	}
	return h.inner.Handle(ctx, r)
}

func (h *DualModeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &DualModeHandler{inner: h.inner.WithAttrs(attrs), legacyFlat: h.legacyFlat}
}

func (h *DualModeHandler) WithGroup(name string) slog.Handler {
	return &DualModeHandler{inner: h.inner.WithGroup(name), legacyFlat: h.legacyFlat}
}

func attrsToAny(attrs []slog.Attr) []any {
	res := make([]any, len(attrs))
	for i, a := range attrs {
		res[i] = a
	}
	return res
}
```

### 5.2 `pkg/idutil` Delegation to `pkg/idgen`
```go
// Package idutil provides cryptographic identifier generation utilities.
// Deprecated: idutil is preserved for backward compatibility; use pkg/idgen directly.
package idutil

import "github.com/dangnmh/core-kit/pkg/idgen"

const Alphabet = idgen.Alphabet

func NanoID(size int) string {
	return idgen.NanoID(size)
}

func NanoIDWithAlphabet(size int, alphabet string) string {
	return idgen.NanoIDWithAlphabet(size, alphabet)
}
```

### 5.3 `pkg/llm` String Truncation Reuse
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

### 5.4 Fail-Closed Bearer Authentication (`pkg/httpserver/auth.go`)
```go
package httpserver

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"github.com/gin-gonic/gin"
)

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

## 6. Step-by-Step TDD Implementation Plan

### Phase A: `pkg/logger` Dual-Mode Context & Parity
- [ ] **Step 1: Write failing test in `pkg/logger/logger_test.go`**:
  ```go
  func TestDualModeHandler_ContextNesting(t *testing.T) {
      var buf bytes.Buffer
      lvl := new(slog.LevelVar)
      lvl.Set(slog.LevelInfo)
      l := NewLogger("test-svc", lvl, &buf)
      l.Info("user logged in", "user_id", "u-123", "ip", "127.0.0.1")

      var parsed map[string]any
      require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
      assert.Equal(t, "test-svc", parsed["service"])
      assert.Equal(t, "user logged in", parsed["message"])

      // Invariant: context object exists and holds user_id and ip
      ctxObj, ok := parsed["context"].(map[string]any)
      require.True(t, ok, "context key must be a JSON object")
      assert.Equal(t, "u-123", ctxObj["user_id"])
      assert.Equal(t, "127.0.0.1", ctxObj["ip"])
  }
  ```
- [ ] **Step 2: Run test to confirm failure**:
  `go test -v ./pkg/logger -run TestDualModeHandler_ContextNesting` (fails because attributes are at root).
- [ ] **Step 3: Implement `DualModeHandler` in `pkg/logger/logger.go`**:
  Add `DualModeHandler` and wrap `traceHandler` with `DualModeHandler` in `NewLogger`.
- [ ] **Step 4: Verify test passes**:
  `go test -v -race ./pkg/logger`

### Phase B: `pkg/idutil` Redundancy Consolidation
- [ ] **Step 5: Write unit test in `pkg/idutil/nanoid_test.go`**:
  Assert `idutil.NanoID(21)` produces 21-character string using `Alphabet` without panicking.
- [ ] **Step 6: Replace `pkg/idutil/nanoid.go` with alias wrapper**:
  Point directly to `github.com/dangnmh/core-kit/pkg/idgen`.
- [ ] **Step 7: Run verification**:
  `go test -v -race ./pkg/idutil ./pkg/idgen`

### Phase C: `pkg/llm` String Truncation Algorithm Reuse
- [ ] **Step 8: Write unit test in `pkg/llm/extractor_test.go`**:
  Verify multi-byte Vietnamese Unicode strings (e.g., `"Xin chào thế giới 🚀"`) truncate without splitting runes or panicking.
- [ ] **Step 9: Refactor `pkg/llm/extractor.go`**:
  Delete `truncateRunes` and call `strutil.TruncateRunes`.
- [ ] **Step 10: Run verification**:
  `go test -v -race ./pkg/llm`

### Phase D: `pkg/httpserver` Fail-Closed Security & Body Protection
- [ ] **Step 11: Write panic test in `pkg/httpserver/auth_test.go`**:
  Verify `BearerTokenAuth("")` and `BearerTokenAuth("   ")` panic with the exact expected error string.
- [ ] **Step 12: Implement validation in `pkg/httpserver/auth.go`**:
  Check `strings.TrimSpace(token) == ""`.
- [ ] **Step 13: Add `MaxBytesMiddleware(10 << 20)` in `pkg/httpserver/server.go`**:
  Add `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)`.
- [ ] **Step 14: Run verification**:
  `go test -v -race ./pkg/httpserver`

### Phase E: Static Linter Error Remediation (`errcheck`)
- [ ] **Step 15: Fix `pkg/httpclient/round_tripper_test.go`**:
  Wrap all `defer resp.Body.Close()` in `defer func() { _ = resp.Body.Close() }()`.
- [ ] **Step 16: Fix `pkg/ws/client.go` & `pkg/ws/ws_test.go`**:
  Explicitly handle or ignore connection closure returns (`_ = c.conn.Close()`, `defer func() { _ = conn.Close() }()`).
- [ ] **Step 17: Full suite verification**:
  `go test -race ./...`

---

## 🔗 Downstream Agent Handoff

- **Target Agent**: `08a-senior-go-dev` & `10-ai-engineer`
- **Mandatory Sections to Consume**: Section 4 (Exact Files), Section 5 (API Cheat-Sheet), Section 6 (TDD Plan).
- **Expected Downstream Output**: Modified Go packages with 100% test pass rate (`go test -race ./...`).
