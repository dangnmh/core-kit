# Sprint 1 Task: Go Core Foundation Standardization & Telemetry Parity (`pkg/*`)

**Task ID**: `TASK-PKG-SPRINT-1`  
**Assigned Specialist Agent**: `08a-senior-go-dev` (Model: `flash`) & `10-ai-engineer` (Model: `flash` for `pkg/llm`)  
**Parent Story**: [`docs/bmad/stories/story-01-core-go-standardization.md`](file:///home/dangnmh/work/core-kit/docs/bmad/stories/story-01-core-go-standardization.md)  
**Component Root**: [`pkg/`](file:///home/dangnmh/work/core-kit/pkg)  
**Component Manifest**: [`go.mod`](file:///home/dangnmh/work/core-kit/go.mod) (Go 1.27.1+)  
**Contracts**: [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md), [`docs/pkg/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/events-and-data.md)  
**Status**: `READY FOR IMPLEMENTATION`  

---

## 1. Constitution Gate Check (`CONST-01..08` & `CONST-SEC-01..06`)

- [x] **`CONST-01` (Zero Storage)**: No database tables or disk persistence.
- [x] **`CONST-02` (Strict Reuse)**: `idutil` aliases `idgen`; `llm` reuses `strutil.TruncateRunes`.
- [x] **`CONST-03` (Telemetry Parity)**: Single-line RFC3339 JSON with `"context"` nesting and optional legacy flat mode.
- [x] **`CONST-04` (Zero Data Races)**: All packages pass `go test -race ./pkg/...`.
- [x] **`CONST-SEC-01` (Fail-Closed Auth)**: Bearer auth panics if token is empty or whitespace.
- [x] **`CONST-SEC-04` (Bounded Allocation)**: Inbound HTTP body bounded to 10MB via `http.MaxBytesReader`.

---

## 2. OpenSpec Capability Delta

```diff
ADDED:
+ pkg/logger.DualModeHandler
+ pkg/httpserver.MaxBytesMiddleware(maxBytes int64) gin.HandlerFunc
+ pkg/httpserver.BearerTokenAuth fail-closed panic validation

MODIFIED:
! pkg/logger/logger.go: Wire DualModeHandler into NewLogger; ensure field name parity
! pkg/idutil/nanoid.go: Full alias to pkg/idgen with @deprecated notice
! pkg/llm/extractor.go: Replace truncateRunes with strutil.TruncateRunes
! pkg/httpclient/round_tripper_test.go: Handle 5 unhandled resp.Body.Close()
! pkg/ws/client.go: Handle unhandled c.conn.Close()
! pkg/ws/ws_test.go: Handle unhandled conn.Close()

REMOVED:
- Duplicate nanoid implementation in pkg/idutil/nanoid.go
- Private truncateRunes in pkg/llm/extractor.go
```

---

## 3. EARS Acceptance Criteria

- **`AC-PKG-1.1`**: `WHEN` user attributes are passed to `logger.Info` / `logger.Error`, `THE` `pkg/logger` package `SHALL` nest them inside `"context"` key in emitted JSON.
- **`AC-PKG-1.2`**: `IF` `LOG_LEGACY_FLAT=true` is set, `THE` `pkg/logger` package `SHALL` emit attributes both in root and in `"context"`.
- **`AC-PKG-1.3`**: `WHEN` `idutil.NanoID(size)` is called, `THE` package `SHALL` execute `idgen.NanoID(size)`.
- **`AC-PKG-1.4`**: `WHEN` `llm.ExtractJSON` formats errors, `THE` package `SHALL` invoke `strutil.TruncateRunes(content, 256)`.
- **`AC-PKG-1.5`**: `IF` token passed to `httpserver.BearerTokenAuth(token)` is empty or whitespace, `THE` function `SHALL` panic immediately.
- **`AC-PKG-1.6`**: `WHEN` `golangci-lint run ./pkg/httpclient ./pkg/ws` runs, `THE` exit code `SHALL` be 0 with 0 warnings.

---

## 4. Exact File Paths & Lines to Modify

1. **`pkg/logger/logger.go:29-58`**: Update `replaceAttr` to ensure `"logger"` key mapping.
2. **`pkg/logger/logger.go:60-75`**: Add `DualModeHandler` struct and methods.
3. **`pkg/logger/logger.go:115-121`**: Wrap `traceHandler` with `NewDualModeHandler`.
4. **`pkg/idutil/nanoid.go:1-35`**: Replace with alias importing `github.com/dangnmh/core-kit/pkg/idgen`.
5. **`pkg/llm/extractor.go:21-31`**: Delete `truncateRunes`; import `pkg/strutil` and update line 82 to call `strutil.TruncateRunes`.
6. **`pkg/httpserver/auth.go:10-30`**: Add `strings.TrimSpace(token) == ""` panic check.
7. **`pkg/httpserver/server.go:60-80`**: Attach `MaxBytesMiddleware(10 << 20)`.
8. **`pkg/httpclient/round_tripper_test.go:34, 55, 78, 102, 128`**: Update `defer resp.Body.Close()` to `defer func() { _ = resp.Body.Close() }()`.
9. **`pkg/ws/client.go:197`**: Update `c.conn.Close()` to `_ = c.conn.Close()`.
10. **`pkg/ws/ws_test.go:44, 349, 382, 400, 418`**: Handle unhandled `conn.Close()`.

---

## 5. Mandatory Libraries & Verified API Cheat-Sheet

> **FORBIDDEN**: Do NOT hand-roll custom string slicing, retry loops, or ID generators. You MUST compose the exact functions below.

### 5.1 DualModeHandler (`pkg/logger/logger.go`)
```go
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
```

### 5.2 NanoID Alias (`pkg/idutil/nanoid.go`)
```go
package idutil

import "github.com/dangnmh/core-kit/pkg/idgen"

// Alphabet defines English letters and digits.
const Alphabet = idgen.Alphabet

// NanoID generates a secure random NanoID.
// Deprecated: Use pkg/idgen.NanoID directly.
func NanoID(size int) string {
	return idgen.NanoID(size)
}

// NanoIDWithAlphabet generates a secure random NanoID with a custom alphabet.
// Deprecated: Use pkg/idgen.NanoIDWithAlphabet directly.
func NanoIDWithAlphabet(size int, alphabet string) string {
	return idgen.NanoIDWithAlphabet(size, alphabet)
}
```

---

## 6. Step-by-Step TDD Checkboxes

- [x] **Step 1**: Write test in `pkg/logger/logger_test.go` asserting `"context"` sub-object nesting.
- [x] **Step 2**: Run `go test -v ./pkg/logger` to confirm failure.
- [x] **Step 3**: Implement `DualModeHandler` and update `NewLogger`.
- [x] **Step 4**: Verify `go test -v -race ./pkg/logger` passes.
- [x] **Step 5**: Replace `pkg/idutil/nanoid.go` with alias to `pkg/idgen`.
- [x] **Step 6**: Run `go test -v -race ./pkg/idutil ./pkg/idgen`.
- [ ] **Step 7**: Update `pkg/llm/extractor.go` to import `pkg/strutil.TruncateRunes`. (Assigned to `10-ai-engineer`)
- [ ] **Step 8**: Run `go test -v -race ./pkg/llm`. (Assigned to `10-ai-engineer`)
- [x] **Step 9**: Add panic check to `pkg/httpserver/auth.go` and test in `pkg/httpserver/auth_test.go`.
- [x] **Step 10**: Remediate 7 `errcheck` occurrences in `pkg/httpclient` and `pkg/ws`.
- [x] **Step 11**: Run full race-free test suite: `go test -race ./pkg/...`.

---

## 🔗 Downstream Agent Handoff

- **Target Agent**: `08a-senior-go-dev` & `10-ai-engineer`
- **Mandatory Sections**: Section 4 (Files), Section 5 (API Snippets), Section 6 (TDD Checkboxes)
- **Expected Deliverable**: All Go packages passing `go test -race ./...` with zero data races.
