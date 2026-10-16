# Component Backlog: TypeScript Foundation (`ts/logger-ts`)

**Document ID**: `BACKLOG-TS-V1`  
**Status**: `⏸️ Backlog / Deferred (HADS **[?]**)`  
**Component**: TypeScript / Bun Telemetry & Logging (`ts/logger-ts`)  
**Parent PRD**: [`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md)  
**Last Updated**: 2026-10-16  

---

## 1. Overview & HADS Isolation Rule

In accordance with the **BMAD 4-Layer Monorepo Documentation Standard (HADS 1.0)**, this document isolates all deferred, future, and unbuilt proposals (`**[?]**`) for the TypeScript logging library (`ts/logger-ts`). These items are explicitly **OUT OF SCOPE** for the current zero-to-prod cycle to maintain the Living Truth integrity in [`docs/ts/README.md`](file:///home/dangnmh/work/core-kit/docs/ts).

---

## 2. Deferred Proposals & Architectural Items

### 2.1 **[?]** `logger-ts/hono`: Native Hono & Elysia HTTP Middleware
- **Feature ID**: `BK-04`
- **Priority**: P2 / Phase 3
- **Motivation**: Edge services and API gateways built on Bun using Hono or Elysia require standardized HTTP request logging with zero overhead, automatic W3C traceparent extraction, and downstream header propagation.
- **Proposed Scope**:
  - Hono middleware package export: `import { honoLogger } from "@dangnmh/logger-ts/hono"`.
  - Automatic extraction of `traceparent`, `tracestate`, and `x-request-id` into Hono context variables (`c.get("traceId")`).
  - Single-line JSON access logs emitted on response completion including HTTP status, method, path, and duration in milliseconds.
  - Zero-allocation header parsing utilizing Bun native APIs.
- **Revisit Trigger**: When an ingress API gateway using Hono is deployed in the ecosystem.

---

### 2.2 **[?]** Rotating JSONL Disk File Transport
- **Feature ID**: `BK-10`
- **Priority**: P2 / Phase 3
- **Motivation**: While cloud-native deployments stream logs directly to `stdout`/`stderr` for container runtime ingestion, certain edge nodes, local CLI agents, or high-throughput bots benefit from persistent local audit files with automatic size/time rotation.
- **Proposed Scope**:
  - Optional file transport destination: `createFileTransport({ path: "./logs/app.log", maxSizeMb: 50, maxFiles: 5 })`.
  - High-performance non-blocking asynchronous disk writes using Bun's native `Bun.file` and file sinks.
  - Automatic rotation and gzip compression of archived log segments.
- **Revisit Trigger**: When autonomous desktop agents or standalone bots require local disk-backed log retention without external log collectors.

---

### 2.3 **[?]** Dual ESM / CommonJS Build & NPM Publishing Workflow
- **Feature ID**: `BK-11`
- **Priority**: P2 / Phase 3
- **Motivation**: `@dangnmh/logger-ts` is currently configured for direct Bun / TypeScript workspace consumption (`ts/logger-ts/src/index.ts`). Supporting legacy Node.js CommonJS consumers outside the monorepo requires compiled distributions.
- **Proposed Scope**:
  - Add build step utilizing `tsup` or Bun build generating `dist/index.mjs` (ESM) and `dist/index.cjs` (CommonJS), along with TypeScript declaration maps (`dist/index.d.ts`).
  - Automated package publishing to private Verdaccio or NPM registry via GitHub Actions on git tag triggers.
- **Revisit Trigger**: When an external Node.js CJS project requests integration with the shared logger.
