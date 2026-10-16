# Core-Kit Master Documentation Hub (`docs/README.md`)
**Version 1.0.0** · Core Infrastructure & PMO Team · 2026-10-16 · Standard: BMAD 4-Layer HADS 1.0.0

---

## 1. Documentation Architecture Overview

[`core-kit`](file:///home/dangnmh/work/core-kit) enforces the **4-Layer Monorepo HADS Documentation Pyramid**:
- **Layer 0 (Root Constitution & AI Context Router)**: Global rules, non-negotiable invariants, and sitemaps.
- **Layer 1 (Component Nearest-Wins AI Context)**: Local component READMEs with `## AI READING INSTRUCTION`, `**[SPEC]**` invariants, and `**[BUG]**` pitfall memory.
- **Layer 2 (Living System & Component Truth)**: Master architecture, contracts, ADRs, SRE runbooks, and capability matrices.
- **Layer 3 (Sprint Deltas & PMO SDLC Governance)**: OpenSpec sprint tasks, archived release notes, and BMAD team reports.

---

## 2. Master Navigation Index

### 2.1 System Architecture & Technical Governance
| Document | Description | Path |
| :--- | :--- | :--- |
| **System Architecture** | Polyglot platform design, C4 diagrams & lifecycle state machine | [`docs/bmad/04-architecture.md`](file:///home/dangnmh/work/core-kit/docs/bmad/04-architecture.md) |
| **Feature Matrix & Roadmap** | End-to-end traceability from Feature IDs to source and test files | [`docs/06-feature-matrix-and-roadmap.md`](file:///home/dangnmh/work/core-kit/docs/06-feature-matrix-and-roadmap.md) |
| **Security Architecture** | STRIDE threat models, trust boundaries & security invariants | [`docs/bmad/06-security-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/06-security-spec.md) |
| **Database & Memory Spec** | In-memory zero-storage architecture & thread-safe data structures | [`docs/bmad/05-database-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/05-database-spec.md) |
| **Ecosystem Manifest** | Curated external dependencies, package health & verified usage | [`docs/bmad/06b-library-manifest.md`](file:///home/dangnmh/work/core-kit/docs/bmad/06b-library-manifest.md) |
| **Librarian & PMO Hub** | Master drift audit, delta-convergence ledger & governance sign-off | [`docs/bmad/11-project-librarian-hub.md`](file:///home/dangnmh/work/core-kit/docs/bmad/11-project-librarian-hub.md) |

### 2.2 Architecture Decision Records (`docs/adr/`)
| ADR ID | Title | Status | Link |
| :--- | :--- | :---: | :--- |
| **ADR-001** | Unified Polyglot Structured Logging Schema | Accepted | [`ADR-001`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md) |
| **ADR-002** | Sharded Multiplexed WebSocket Pool & Dispatcher | Accepted | [`ADR-002`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-002-websocket-architecture.md) |
| **ADR-003** | Polyglot Linter, Formatter, and CI Toolchain | Accepted | [`ADR-003`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-003-polyglot-linter-and-ci-toolchain.md) |

### 2.3 SRE Operational Runbooks (`docs/runbooks/`)
| Runbook ID | Failure Mode & Scenario | Severity | Link |
| :--- | :--- | :---: | :--- |
| **RB-01** | WebSocket Connection Exhaustion & Reconnect Storm | SEV-2 | [`rb-01-websocket-reconnect-storm.md`](file:///home/dangnmh/work/core-kit/docs/runbooks/rb-01-websocket-connection-exhaustion-and-reconnect-storm.md) |
| **RB-02** | Bearer Auth Startup Panic & Container CrashLoopBackOff | SEV-1 | [`rb-02-bearer-auth-startup-panic.md`](file:///home/dangnmh/work/core-kit/docs/runbooks/rb-02-bearer-auth-startup-panic-and-crashloop.md) |
| **RB-03** | Circuit Breaker Open Cascade & Rate Limit Rejection | SEV-2 | [`rb-03-circuit-breaker-cascade.md`](file:///home/dangnmh/work/core-kit/docs/runbooks/rb-03-circuit-breaker-open-cascade-and-rate-limiting.md) |

---

## 3. Canonical Component Hubs (6 Items Each)

| Canonical Item | Go Foundation (`pkg/*`) | Python Telemetry (`py/logger-py`) | TypeScript Telemetry (`ts/logger-ts`) |
| :--- | :--- | :--- | :--- |
| **1. Nearest-Wins Guide** | [`pkg/README.md`](file:///home/dangnmh/work/core-kit/pkg/README.md) | [`py/logger-py/README.md`](file:///home/dangnmh/work/core-kit/py/logger-py/README.md) | [`ts/logger-ts/README.md`](file:///home/dangnmh/work/core-kit/ts/logger-ts/README.md) |
| **2. Feature Hub** | [`docs/pkg/README.md`](file:///home/dangnmh/work/core-kit/docs/pkg/README.md) | [`docs/py/README.md`](file:///home/dangnmh/work/core-kit/docs/py/README.md) | [`docs/ts/README.md`](file:///home/dangnmh/work/core-kit/docs/ts/README.md) |
| **3. Architecture & Diagrams** | [`docs/pkg/architecture.md`](file:///home/dangnmh/work/core-kit/docs/pkg/architecture.md) | [`docs/py/architecture.md`](file:///home/dangnmh/work/core-kit/docs/py/architecture.md) | [`docs/ts/architecture.md`](file:///home/dangnmh/work/core-kit/docs/ts/architecture.md) |
| **4. API Contract** | [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md) | [`docs/py/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/api.md) | [`docs/ts/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/api.md) |
| **5. Events & Data Contract** | [`docs/pkg/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/events-and-data.md) | [`docs/py/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/py/contracts/events-and-data.md) | [`docs/ts/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/events-and-data.md) |
| **6. Active Sprint Tasks** | [`docs/pkg/tasks/sprint-1-core-standardization.md`](file:///home/dangnmh/work/core-kit/docs/pkg/tasks/sprint-1-core-standardization.md) | [`docs/py/tasks/sprint-1-logger-and-ruff.md`](file:///home/dangnmh/work/core-kit/docs/py/tasks/sprint-1-logger-and-ruff.md) | [`docs/ts/tasks/sprint-1-logger-and-eslint.md`](file:///home/dangnmh/work/core-kit/docs/ts/tasks/sprint-1-logger-and-eslint.md) |
| **7. Release Archive** | [`docs/pkg/releases/v1.0.0-core-standardization.md`](file:///home/dangnmh/work/core-kit/docs/pkg/releases/v1.0.0-core-standardization.md) | [`docs/py/releases/v1.0.0-core-standardization.md`](file:///home/dangnmh/work/core-kit/docs/py/releases/v1.0.0-core-standardization.md) | [`docs/ts/releases/v1.0.0-core-standardization.md`](file:///home/dangnmh/work/core-kit/docs/ts/releases/v1.0.0-core-standardization.md) |
| **8. Component Backlog** | [`docs/pkg/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/pkg/backlog/backlog-v1.md) | [`docs/py/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/py/backlog/backlog-v1.md) | [`docs/ts/backlog/backlog-v1.md`](file:///home/dangnmh/work/core-kit/docs/ts/backlog/backlog-v1.md) |

---

## 4. Multi-Agent SDLC & Governance Reports (`docs/bmad/`)

| Report ID | Title & Primary Scope | Author / Role | Link |
| :--- | :--- | :--- | :--- |
| **BMAD-01** | Project Brief & Strategic Value Pillars | `01-ba-analyst` | [`01-project-brief.md`](file:///home/dangnmh/work/core-kit/docs/bmad/01-project-brief.md) |
| **BMAD-02** | Product Requirements Document (PRD) | `02-po-manager` | [`02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md) |
| **BMAD-03** | UI/UX & CLI Developer Experience Spec | `03-ui-ux-designer` | [`03-ui-ux-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/03-ui-ux-spec.md) |
| **BMAD-04** | System Architecture & Technical Specification | `04-system-architect` | [`04-architecture.md`](file:///home/dangnmh/work/core-kit/docs/bmad/04-architecture.md) |
| **BMAD-05** | Database & Memory State Specification | `05-senior-db-engineer` | [`05-database-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/05-database-spec.md) |
| **BMAD-06** | Security Architecture & Threat Modeling | `06-security-engineer` | [`06-security-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/06-security-spec.md) |
| **BMAD-06b** | Ecosystem Library Discovery & Verified APIs | `06b-ecosystem-scout` | [`06b-library-manifest.md`](file:///home/dangnmh/work/core-kit/docs/bmad/06b-library-manifest.md) |
| **BMAD-07** | Detailed Implementation & Wave Execution Plan | `07-tech-lead` | [`07-execution-plan.md`](file:///home/dangnmh/work/core-kit/docs/bmad/07-execution-plan.md) |
| **BMAD-08** | Code Review & Multi-Perspective Triage | `12-code-reviewer` | [`08-review-report.md`](file:///home/dangnmh/work/core-kit/docs/bmad/08-review-report.md) |
| **BMAD-09** | QA Release Readiness & Test Evidence Report | `13-qa-tester` | [`09-qa-release-report.md`](file:///home/dangnmh/work/core-kit/docs/bmad/09-qa-release-report.md) |
| **BMAD-10** | Technical Documentation & Runbooks Report | `14-tech-writer` | [`10-technical-docs-and-runbooks.md`](file:///home/dangnmh/work/core-kit/docs/bmad/10-technical-docs-and-runbooks.md) |
| **BMAD-11** | PMO Hub, Drift Audit & Release Sign-Off | `15-project-librarian` | [`11-project-librarian-hub.md`](file:///home/dangnmh/work/core-kit/docs/bmad/11-project-librarian-hub.md) |
