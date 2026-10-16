# Story 04: Polyglot Linter Toolchain, Unified Makefile, Doc-Lint Gate & HADS Governance

**Document ID**: `STORY-TOOL-04`  
**Parent Epic**: Epic 1 & Epic 3 ([`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md))  
**Assigned Specialist Agents**: `11-devops-engineer` (DevOps & CI/CD Lead, `flash`), `14-tech-writer` (Technical Writer, `flash`), `15-project-librarian` (PMO & Governance Architect, `pro`)  
**Recommended Model Tier**: Mixed (`flash` for DevOps/Writer tasks, `pro` for final consolidation)  
**Component Scope**: Monorepo Root, `scripts/`, `docs/`, `Makefile`, `.golangci.yml`  
**Related Contracts & ADRs**: [`ADR-003`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-003-polyglot-linter-and-ci-toolchain.md), [`docs/bmad/04-architecture.md`](file:///home/dangnmh/work/core-kit/docs/bmad/04-architecture.md)  
**Status**: `READY FOR IMPLEMENTATION`  

---

## 1. Constitution Gate Check (`CONST-01..08` & `CONST-SEC-01..06`)

Before writing any code or documentation, the team MUST verify compliance with the following binding invariants:
- **`CONST-08` (BMAD 4-Layer HADS Documentation Standard)**:
  - Root [`AGENTS.md`](file:///home/dangnmh/work/core-kit/AGENTS.md) MUST be `<= 120 lines`.
  - Component Nearest-Wins READMEs ([`docs/pkg/README.md`](file:///home/dangnmh/work/core-kit/docs/pkg), [`docs/py/README.md`](file:///home/dangnmh/work/core-kit/docs/py), [`docs/ts/README.md`](file:///home/dangnmh/work/core-kit/docs/ts)) MUST be `<= 120 lines` and contain `## AI READING INSTRUCTION`, `**[SPEC]**` local invariants, and `**[BUG]**` pitfalls.
  - All markdown links must resolve to valid files (0 broken links).
- **`ADR-003` (Polyglot Linter & CI Architecture)**:
  - Unified GNU `Makefile` orchestrates native toolchains without hardcoded machine paths (`/home/...`).
  - `.golangci.yml` enforces version 2 specification with `errcheck`, `govet`, `staticcheck`.
  - `scripts/doc-lint.sh` automatically enforces line caps and relative link integrity.

---

## 2. OpenSpec Capability Delta

```diff
ADDED:
+ .golangci.yml: GolangCI-Lint v2 configuration enabling errcheck, govet, ineffassign, staticcheck, unused, gosec, revive.
+ scripts/doc-lint.sh: Automated shell gate checking line counts (<= 120 lines on Layer 0/1) and verifying 0 broken markdown links.
+ AGENTS.md: Root Layer 0 Constitution and Tool Traps (<= 120 lines).
+ llms.txt: Root Layer 0 dense AI context sitemap.
+ CHANGELOG.md: Standard Keep a Changelog documentation.
+ docs/pkg/README.md: Layer 1 Go Foundation Nearest-Wins Guardrails (<= 120 lines).
+ docs/py/README.md: Layer 1 Python Telemetry Nearest-Wins Guardrails (<= 120 lines).
+ docs/ts/README.md: Layer 1 TypeScript Telemetry Nearest-Wins Guardrails (<= 120 lines).
+ Makefile targets: lint, lint-go, lint-ts, lint-py, format / fmt, format-go, format-ts, format-py, format-check, upgrade, doc-lint, ci.

MODIFIED:
! Makefile: Modernized to use portable `uv run` and `bun` commands; removed hardcoded paths.

REMOVED:
- Hardcoded developer workstation path `local-venv/bin/pytest`.
```

---

## 3. EARS Acceptance Criteria

- **`AC-TOOL-4.1` (Zero Hardcoded Workstation Paths)**:
  `WHEN` `make test`, `make lint`, or `make ci` is executed on any Linux/macOS environment
  `THE` `Makefile` `SHALL` execute using portable `go`, `bun`, and `uv run` commands without referencing `/home/dangnmh`.
- **`AC-TOOL-4.2` (Unified Multi-Runtime Lint Gate)**:
  `WHEN` `make lint` is executed
  `THE` root `Makefile` `SHALL` sequentially run `golangci-lint run ./...`, `cd ts/logger-ts && bun lint`, `cd py/logger-py && uv run ruff check .`, and `make doc-lint`.
- **`AC-TOOL-4.3` (Automated Doc-Lint Gate)**:
  `WHEN` `make doc-lint` or `./scripts/doc-lint.sh` is executed
  `THE` validator `SHALL` verify that `AGENTS.md` and all `docs/*/README.md` are `<= 120 lines` and that all internal markdown links exist, exiting with code 0 on success or code 1 on failure.
- **`AC-TOOL-4.4` (HADS Documentation Structure Compliance)**:
  `WHEN` an autonomous agent loads `docs/pkg/README.md`, `docs/py/README.md`, or `docs/ts/README.md`
  `THE` file `SHALL` contain sections `## AI READING INSTRUCTION`, `**[SPEC]**` Local Invariants, Key Entrypoints & Module Map, and `**[BUG]**` Known Traps.
- **`AC-TOOL-4.5` (Unified CI Target)**:
  `WHEN` `make ci` is run
  `THE` target `SHALL` execute `make doc-lint`, `make lint`, and `make test`, exiting with code 0.

---

## 4. Exact File Paths & Ownership

- **Assigned Agent**: `11-devops-engineer`
  - Create: [`.golangci.yml`](file:///home/dangnmh/work/core-kit/.golangci.yml)
  - Create: [`scripts/doc-lint.sh`](file:///home/dangnmh/work/core-kit/scripts/doc-lint.sh)
  - Modify: [`Makefile`](file:///home/dangnmh/work/core-kit/Makefile)
- **Assigned Agent**: `14-tech-writer`
  - Create: [`AGENTS.md`](file:///home/dangnmh/work/core-kit/AGENTS.md)
  - Create: [`llms.txt`](file:///home/dangnmh/work/core-kit/llms.txt)
  - Create: [`CHANGELOG.md`](file:///home/dangnmh/work/core-kit/CHANGELOG.md)
  - Create: [`docs/pkg/README.md`](file:///home/dangnmh/work/core-kit/docs/pkg/README.md)
  - Create: [`docs/py/README.md`](file:///home/dangnmh/work/core-kit/docs/py/README.md)
  - Create: [`docs/ts/README.md`](file:///home/dangnmh/work/core-kit/docs/ts/README.md)
- **Assigned Agent**: `15-project-librarian`
  - Consolidate: Monorepo index, release notes, and documentation catalog.

---

## 5. Mandatory Libraries & Configuration Code Cheat-Sheet (`from 06b-library-manifest.md`)

### 5.1 GolangCI-Lint v2 Specification (`.golangci.yml`)
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

### 5.2 Unified Makefile Specification (`Makefile`)
```makefile
.PHONY: all test test-go test-ts test-py lint lint-go lint-ts lint-py format fmt format-go format-ts format-py format-check upgrade upgrade-go upgrade-ts upgrade-py doc-lint ci clean

all: test lint

# ---------------------------------------------------------
# Testing
# ---------------------------------------------------------
test: test-go test-ts test-py

test-go:
	go test -race ./...

test-ts:
	cd ts/logger-ts && bun test

test-py:
	cd py/logger-py && uv run pytest tests -v

# ---------------------------------------------------------
# Linting & Static Analysis
# ---------------------------------------------------------
lint: lint-go lint-ts lint-py doc-lint

lint-go:
	golangci-lint run ./...

lint-ts:
	cd ts/logger-ts && bun run lint

lint-py:
	cd py/logger-py && uv run ruff check .

# ---------------------------------------------------------
# Formatting
# ---------------------------------------------------------
format: fmt
fmt: format-go format-ts format-py

format-go:
	gofmt -s -w .

format-ts:
	cd ts/logger-ts && bun run format

format-py:
	cd py/logger-py && uv run ruff format .

format-check:
	@echo "Checking Go formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Unformatted Go files detected:" && gofmt -l . && exit 1)
	@echo "Checking Python formatting..."
	cd py/logger-py && uv run ruff format --check .
	@echo "Checking TypeScript formatting..."
	cd ts/logger-ts && bun run lint

# ---------------------------------------------------------
# Dependency Management
# ---------------------------------------------------------
upgrade: upgrade-go upgrade-ts upgrade-py

upgrade-go:
	go get -u ./...
	go mod tidy

upgrade-ts:
	cd ts/logger-ts && bun update

upgrade-py:
	cd py/logger-py && uv lock --upgrade

# ---------------------------------------------------------
# Documentation Governance & CI
# ---------------------------------------------------------
doc-lint:
	@chmod +x scripts/doc-lint.sh
	./scripts/doc-lint.sh

ci: doc-lint lint test

clean:
	rm -rf py/logger-py/.pytest_cache py/logger-py/.ruff_cache ts/logger-ts/node_modules
```

### 5.3 Automated HADS Documentation Validator (`scripts/doc-lint.sh`)
```bash
#!/usr/bin/env bash
set -euo pipefail

echo "=== Running BMAD HADS Documentation Lint Gate ==="

ERRORS=0

# 1. Enforce line count caps (<= 120 lines) on Layer 0 and Layer 1 READMEs
FILES_TO_CHECK=(
  "AGENTS.md"
  "docs/pkg/README.md"
  "docs/py/README.md"
  "docs/ts/README.md"
)

for file in "${FILES_TO_CHECK[@]}"; do
  if [[ -f "$file" ]]; then
    LINES=$(wc -l < "$file")
    if [[ "$LINES" -gt 120 ]]; then
      echo "❌ ERROR: $file exceeds 120 lines (currently $LINES lines). Violation of CONST-08."
      ERRORS=$((ERRORS + 1))
    else
      echo "✅ PASS: $file ($LINES lines <= 120)"
    fi
  else
    echo "⚠️ WARNING: $file does not exist yet."
    ERRORS=$((ERRORS + 1))
  fi
done

# 2. Check for broken relative markdown links in all docs
echo "Checking relative markdown links..."
while IFS= read -r md_file; do
  DIR=$(dirname "$md_file")
  # Extract links like [text](link.md)
  LINKS=$(grep -o -E '\[[^]]+\]\(([^)]+\.md)\)' "$md_file" | sed -E 's/\[[^]]+\]\(([^)]+)\)/\1/' || true)
  for link in $LINKS; do
    # Remove anchors
    CLEAN_LINK="${link%%#*}"
    if [[ -n "$CLEAN_LINK" ]]; then
      TARGET="$DIR/$CLEAN_LINK"
      if [[ ! -f "$TARGET" ]]; then
        echo "❌ BROKEN LINK in $md_file: target '$TARGET' does not exist."
        ERRORS=$((ERRORS + 1))
      fi
    fi
  done
done < <(find docs -type f -name "*.md")

if [[ "$ERRORS" -gt 0 ]]; then
  echo "=== Doc-Lint Failed with $ERRORS error(s) ==="
  exit 1
fi

echo "=== All Documentation Checks Passed Cleanly ==="
exit 0
```

---

## 6. Step-by-Step TDD Implementation Plan

- [x] **Step 1: Create `.golangci.yml`**:
  Write configuration file at repository root with valid v2 specification enabling core linters (`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`).
- [x] **Step 2: Create `scripts/doc-lint.sh`**:
  Write shell script checking Layer 0/1 line caps and non-codeblock relative/`file://` markdown links; mark executable (`chmod +x scripts/doc-lint.sh`).
- [x] **Step 3: Update root `Makefile` & Transitive Dependencies**:
  Replace existing `Makefile` with complete multi-runtime specification (`test*`, `lint*`, `format*`, `format-check`, `upgrade*`, `doc-lint`, `ci`), and upgrade transitive CVEs in `go.mod`/`go.sum`.
- [ ] **Step 4: Author Layer 0 documentation (`14-tech-writer`)**:
  - Write root `AGENTS.md` (`<= 120 lines`) establishing project constitution and tool traps.
  - Write root `llms.txt` with dense AI context index.
  - Write `CHANGELOG.md`.
- [ ] **Step 5: Author Layer 1 component documentation (`14-tech-writer`)**:
  - Write `docs/pkg/README.md` (`<= 120 lines`) with HADS 1.0 format.
  - Write `docs/py/README.md` (`<= 120 lines`) with HADS 1.0 format.
  - Write `docs/ts/README.md` (`<= 120 lines`) with HADS 1.0 format.
- [ ] **Step 6: Run `make doc-lint`**:
  Verify 0 line limit violations and 0 broken links.
- [ ] **Step 7: Run `make ci`**:
  Verify full orchestration passing: `doc-lint` -> `lint` -> `test`.

---

## 🔗 Downstream Agent Handoff

- **Target Agent**: `14-tech-writer` (Layer 0 & Layer 1 HADS Documentation), followed by `15-project-librarian` (PMO & Governance Consolidation).
- **Mandatory Sections / Contracts to Consume**:
  - Section 3 (`AC-TOOL-4.3`, `AC-TOOL-4.4` HADS line limits `<= 120` lines).
  - Section 4 (File Ownership: `AGENTS.md`, `llms.txt`, `CHANGELOG.md`, `docs/pkg/README.md`, `docs/py/README.md`, `docs/ts/README.md`).
  - Section 6 (Steps 4 through 7).
- **Expected Deliverable**: Green `make doc-lint` and `make ci` execution with 0 broken links and `<= 120` lines on all Layer 0/1 files.
