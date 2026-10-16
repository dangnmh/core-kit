# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-10-16

### Added
- **Unified Telemetry Parity (`ADR-001`)**:
  - `pkg/logger`: Introduced `DualModeHandler` supporting nested `"context"` attribute mapping with backward-compatible flat mode (`LOG_LEGACY_FLAT=true`).
  - `py/logger-py`: Added top-level extraction and hoisting for W3C `trace_id` and `span_id` from extra dictionaries or context.
  - `ts/logger-ts`: Added top-level extraction and hoisting for W3C `trace_id` and `span_id` from logger context.
  - Standardized RFC3339 single-line JSON log schema across Go, Python, and TypeScript runtimes.
- **Security Hardening (`CONST-SEC-01..06`)**:
  - `pkg/httpserver`: Added `BearerTokenAuth` fail-closed validation that panics immediately on startup if token is empty or whitespace.
  - `pkg/httpserver`: Added `MaxBytesMiddleware` enforcing strict 10MB inbound HTTP body limits to prevent memory exhaustion DoS.
  - `pkg/idgen`: Enforced fast-panic on cryptographic CSPRNG (`crypto/rand.Reader`) read errors to prevent zero-value ID collisions.
  - `pkg/httpserver`: Validated constant-time comparison via `crypto/subtle.ConstantTimeCompare` across authentication endpoints.
- **Developer Toolchain & CI Automation (`ADR-003`)**:
  - Root `Makefile`: Unified cross-language workflows for `test`, `lint`, `format`, `upgrade`, `doc-lint`, and `ci`.
  - `.golangci.yml`: Added GolangCI-Lint v2 configuration enabling `errcheck`, `govet`, `staticcheck`, `ineffassign`, and `revive`.
  - `py/logger-py`: Configured Ruff linter and code formatter with rules `E`, `W`, `F`, `I`, `B`, `C4`, `UP`, and `RUF`.
  - `ts/logger-ts`: Configured ESLint 9 Flat Config (`eslint.config.mjs`) and dependency-injected test streams.
  - `scripts/doc-lint.sh`: Added automated documentation governance script validating line caps (`<= 120` lines) and checking 0 broken links.
- **BMAD 4-Layer Monorepo HADS Documentation Standard (`CONST-08`)**:
  - Layer 0: Root [`AGENTS.md`](file:///home/dangnmh/work/core-kit/AGENTS.md), [`llms.txt`](file:///home/dangnmh/work/core-kit/llms.txt), [`README.md`](file:///home/dangnmh/work/core-kit/README.md), and [`CHANGELOG.md`](file:///home/dangnmh/work/core-kit/CHANGELOG.md).
  - Layer 1: Component nearest-wins guardrails ([`pkg/README.md`](file:///home/dangnmh/work/core-kit/pkg/README.md), [`py/logger-py/README.md`](file:///home/dangnmh/work/core-kit/py/logger-py/README.md), [`ts/logger-ts/README.md`](file:///home/dangnmh/work/core-kit/ts/logger-ts/README.md)).
  - Layer 2: Living system truth ([`docs/README.md`](file:///home/dangnmh/work/core-kit/docs/README.md), [`docs/06-feature-matrix-and-roadmap.md`](file:///home/dangnmh/work/core-kit/docs/06-feature-matrix-and-roadmap.md)), ADRs (`ADR-001`, `ADR-002`, `ADR-003`), SRE runbooks (`rb-01`, `rb-02`, `rb-03`), and Component Truth hubs (`docs/pkg/`, `docs/py/`, `docs/ts/`).
  - Layer 3: Cross-cutting SDLC deliverables ([`docs/bmad/01..11-*.md`](file:///home/dangnmh/work/core-kit/docs/bmad)) and sprint implementation stories.

### Changed
- `pkg/idutil`: Consolidated duplicate NanoID implementation by refactoring `pkg/idutil` into a deprecated alias delegating directly to `pkg/idgen` (`CONST-02`).
- `pkg/llm`: Refactored `ExtractJSON` to consume `pkg/strutil.TruncateRunes` instead of a private slice allocation routine.
- `pkg/httpclient`: Resolved 5 unhandled `resp.Body.Close()` returns in `round_tripper_test.go` to satisfy `errcheck`.
- `pkg/ws`: Resolved unhandled `c.conn.Close()` and `conn.Close()` in client and test code to satisfy `errcheck`.
- `ts/logger-ts`: Refactored unit test suite to use isolated memory streams via `createLogger({ stream })`, eliminating global monkey-patching of `process.stdout.write`.
- `Makefile`: Replaced fragile workstation path `/home/dangnmh/...` with portable `uv run` and `bun` commands.

### Removed
- Removed duplicate cryptographic random generation code in `pkg/idutil/nanoid.go`.
- Removed private `truncateRunes` function and redundant byte-slice allocations in `pkg/llm/extractor.go`.
- Removed global process stream spying in `ts/logger-ts/tests/logger.test.ts`.
- Removed 14 static lint warnings in `py/logger-py`.
