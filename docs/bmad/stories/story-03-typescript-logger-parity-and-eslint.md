# Story 03: TypeScript Telemetry Parity, Test Stream Isolation & ESLint Flat Config

**Document ID**: `STORY-TS-03`  
**Parent Epic**: Epic 2 & Epic 3 ([`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md))  
**Assigned Specialist Agent**: `08c-senior-node-dev` (Senior TypeScript / Node.js Engineer)  
**Recommended Model Tier**: `flash`  
**Component Manifest**: [`ts/logger-ts/package.json`](file:///home/dangnmh/work/core-kit/ts/logger-ts/package.json) (TypeScript 5+ / Bun 1.4+)  
**Related Contracts**: [`CONTRACT-TS-API-01`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/api.md), [`CONTRACT-TS-DATA-01`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/events-and-data.md), [`ADR-001`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-001-unified-polyglot-logging.md), [`ADR-003`](file:///home/dangnmh/work/core-kit/docs/adr/ADR-003-polyglot-linter-and-ci-toolchain.md)  
**Status**: `READY FOR IMPLEMENTATION`  

---

## 1. Constitution Gate Check (`CONST-01..08` & `CONST-SEC-01..06`)

Before writing any code, the engineer MUST verify compliance with the following binding invariants:
- **`CONST-01` (Zero Storage)**: Stateless in-memory logger; zero disk storage or disk buffering.
- **`CONST-03` (Unified Telemetry Parity)**: Emitted JSON records MUST contain root keys: `timestamp`, `severity`, `service`, `logger`, `message`, and optionally `caller`, `trace_id`, `span_id`, and `context`.
- **`CONST-SEC-03` (Zero-Leak Telemetry)**: Sensitive tokens, passwords, and private keys must not be logged.
- **`CONST-SEC-06` (Log Injection Immunity)**: All output lines serialized via `JSON.stringify()` ending with `\n`.

---

## 2. OpenSpec Capability Delta

```diff
ADDED:
+ ts/logger-ts/eslint.config.mjs: ESLint 9 Flat Config with typescript-eslint.
+ ts/logger-ts/src/types.ts: Added stream?: { write: (chunk: string) => void } to LoggerOptions.
+ ts/logger-ts/src/index.ts: Support stream injection in Logger constructor to direct output to custom streams.
+ ts/logger-ts/package.json: Added scripts: "lint": "eslint .", "format": "eslint . --fix".

MODIFIED:
! ts/logger-ts/src/index.ts: Logger.write uses injected stream when provided before falling back to process.stdout / process.stderr.
! ts/logger-ts/tests/logger.test.ts: Refactored unit tests to inject custom memory stream instead of global spyOn(process.stdout, "write") and spyOn(process.stderr, "write").

REMOVED:
- Global standard output mocking in unit tests, restoring Bun test runner reporter visibility.
```

---

## 3. EARS Acceptance Criteria

- **`AC-TS-3.1` (Stream Injection & Test Isolation)**:
  `WHEN` a `Logger` instance is constructed with `stream: { write: (chunk) => logs.push(chunk) }`
  `THE` `Logger` `SHALL` write formatted JSON records to the injected stream without mutating or spying on `process.stdout.write`.
- **`AC-TS-3.2` (Bun Reporter Transparency)**:
  `WHEN` `bun test` is executed inside `ts/logger-ts`
  `THE` Bun test runner `SHALL` display full test suite progress, test names, and pass summary without suppression.
- **`AC-TS-3.3` (Trace Correlation Hoisting)**:
  `WHEN` a context object containing `trace_id` or `span_id` is supplied
  `THE` `Logger` `SHALL` hoist `trace_id` and `span_id` to root fields in the emitted JSON record.
- **`AC-TS-3.4` (Error Object Serialization)**:
  `WHEN` an `Error` instance is passed to `logger.error("Failed", err)`
  `THE` `Logger` `SHALL` serialize it into `context.error` with `name`, `message`, and `stack`.
- **`AC-TS-3.5` (ESLint Cleanliness)**:
  `WHEN` `bun lint` or `bun run lint` is executed in `ts/logger-ts`
  `THE` command `SHALL` exit with 0 errors.

---

## 4. Exact File Paths & Ownership

- **Assigned Agent**: `08c-senior-node-dev`
- **Create**: [`ts/logger-ts/eslint.config.mjs`](file:///home/dangnmh/work/core-kit/ts/logger-ts/eslint.config.mjs)
- **Modify**: [`ts/logger-ts/package.json`](file:///home/dangnmh/work/core-kit/ts/logger-ts/package.json)
- **Modify**: [`ts/logger-ts/src/types.ts`](file:///home/dangnmh/work/core-kit/ts/logger-ts/src/types.ts)
- **Modify**: [`ts/logger-ts/src/index.ts`](file:///home/dangnmh/work/core-kit/ts/logger-ts/src/index.ts)
- **Modify**: [`ts/logger-ts/tests/logger.test.ts`](file:///home/dangnmh/work/core-kit/ts/logger-ts/tests/logger.test.ts)

---

## 5. Mandatory Libraries & Verified API Cheat-Sheet (`from 06b-library-manifest.md`)

> **FORBIDDEN: Do NOT use legacy `.eslintrc.json` or mutate `process.stdout.write` globally in unit tests.**

### 5.1 ESLint Flat Config (`ts/logger-ts/eslint.config.mjs`)
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

### 5.2 Stream Support in `Logger` (`ts/logger-ts/src/index.ts`)
```typescript
export interface LoggerOptions {
  service: string;
  module?: string;
  minLevel?: string | number;
  baseContext?: Record<string, any>;
  /** Custom destination stream; defaults to process.stdout */
  stream?: { write: (chunk: string) => void };
}

export class Logger implements ILogger {
  private service: string;
  private module: string;
  private minLevel: number;
  private baseContext: Record<string, any>;
  private stream?: { write: (chunk: string) => void };

  constructor(options: LoggerOptions) {
    this.service = options.service;
    this.module = options.module || "";
    this.minLevel = parseLogLevel(options.minLevel);
    this.baseContext = options.baseContext ? { ...options.baseContext } : {};
    this.stream = options.stream;
  }

  private write(entry: LogEntry, isErrorOrFatal: boolean): void {
    const line = JSON.stringify(entry);
    if (this.stream && typeof this.stream.write === "function") {
      this.stream.write(line + "\n");
      return;
    }

    if (typeof process !== "undefined" && process.stdout && typeof process.stdout.write === "function") {
      if (isErrorOrFatal && process.stderr && typeof process.stderr.write === "function") {
        process.stderr.write(line + "\n");
      } else {
        process.stdout.write(line + "\n");
      }
    } else {
      if (isErrorOrFatal && typeof console !== "undefined" && typeof console.error === "function") {
        console.error(line);
      } else if (typeof console !== "undefined" && typeof console.log === "function") {
        console.log(line);
      }
    }
  }
}
```

### 5.3 Test Suite Memory Stream Pattern (`tests/logger.test.ts`)
```typescript
describe("packages/logger-ts unit tests", () => {
  let stdoutData: string[] = [];
  let customStream: { write: (chunk: string) => void };

  beforeEach(() => {
    stdoutData = [];
    customStream = {
      write: (chunk: string) => {
        stdoutData.push(chunk);
      }
    };
  });

  it("should output single line JSON with expected schema for info log", () => {
    const logger = createLogger({
      service: "test-service",
      module: "main",
      minLevel: LogLevel.INFO,
      stream: customStream,
    });

    logger.info("Service initialized", { version: "1.0.0" });
    expect(stdoutData.length).toBe(1);
    const parsed = JSON.parse(stdoutData[0].trim());
    expect(parsed.service).toBe("test-service");
    expect(parsed.logger).toBe("main");
    expect(parsed.severity).toBe("INFO");
    expect(parsed.context).toEqual({ version: "1.0.0" });
  });
});
```

---

## 6. Step-by-Step TDD Implementation Plan

- [ ] **Step 1: Update `ts/logger-ts/src/types.ts`**:
  Add `stream?: { write: (chunk: string) => void };` to `LoggerOptions`.
- [ ] **Step 2: Update `ts/logger-ts/src/index.ts`**:
  Store `this.stream` and check `if (this.stream) { this.stream.write(line + "\n"); return; }` in `write()`.
  Ensure `child()` forwards `stream: this.stream` to child instances.
- [ ] **Step 3: Refactor `ts/logger-ts/tests/logger.test.ts`**:
  Remove `spyOn(process.stdout, "write")` and `spyOn(process.stderr, "write")`.
  Use `stream: customStream` for stdout tests and `errorStream` for error tests.
- [ ] **Step 4: Run `bun test`**:
  `cd ts/logger-ts && bun test` (verify all 9 tests pass with visible reporter output).
- [ ] **Step 5: Create `ts/logger-ts/eslint.config.mjs`**:
  Write ESLint Flat Config matching Section 5.1.
- [ ] **Step 6: Update `ts/logger-ts/package.json`**:
  Add `"lint": "eslint ."` and `"format": "eslint . --fix"`.
  Add devDependencies `@eslint/js` and `typescript-eslint` if needed, or run linter via bun.
- [ ] **Step 7: Verify clean lint run**:
  `cd ts/logger-ts && bun test`.

---

## 🔗 Downstream Agent Handoff

- **Target Agent**: `08c-senior-node-dev`
- **Mandatory Sections**: Section 4 (Files), Section 5 (Snippets), Section 6 (TDD Checkboxes)
- **Expected Deliverable**: `ts/logger-ts` passing `bun test` with clean stream isolation and flat ESLint config.
