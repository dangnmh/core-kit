# Sprint 1 Task: TypeScript Telemetry Parity & ESLint Flat Config (`ts/logger-ts`)

**Task ID**: `TASK-TS-SPRINT-1`  
**Assigned Specialist Agent**: `08c-senior-node-dev` (Model: `flash`)  
**Parent Story**: [`docs/bmad/stories/story-03-typescript-logger-parity-and-eslint.md`](file:///home/dangnmh/work/core-kit/docs/bmad/stories/story-03-typescript-logger-parity-and-eslint.md)  
**Component Root**: [`ts/logger-ts/`](file:///home/dangnmh/work/core-kit/ts/logger-ts)  
**Component Manifest**: [`ts/logger-ts/package.json`](file:///home/dangnmh/work/core-kit/ts/logger-ts/package.json) (TypeScript 5+ / Bun 1.4+)  
**Contracts**: [`docs/ts/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/api.md), [`docs/ts/contracts/events-and-data.md`](file:///home/dangnmh/work/core-kit/docs/ts/contracts/events-and-data.md)  
**Status**: `IMPLEMENTED & VERIFIED`  

---

## 1. Constitution Gate Check (`CONST-01..08` & `CONST-SEC-01..06`)

- [x] **`CONST-01` (Zero Storage)**: Pure in-memory streaming logger; zero disk state.
- [x] **`CONST-03` (Unified Telemetry Parity)**: Single-line RFC3339 JSON matching Go and Python schemas.
- [x] **`CONST-SEC-03` (Zero-Leak Telemetry)**: No plain text credentials or secrets in logs.
- [x] **`CONST-SEC-06` (Log Injection Immunity)**: Serialized via `JSON.stringify()`.

---

## 2. OpenSpec Capability Delta

```diff
ADDED:
+ ts/logger-ts/eslint.config.mjs: ESLint 9 Flat Config.
+ ts/logger-ts/src/types.ts: stream property in LoggerOptions.
+ ts/logger-ts/src/index.ts: stream destination support in Logger.write().

MODIFIED:
! ts/logger-ts/tests/logger.test.ts: Use stream injection in tests; eliminate global stdout/stderr spying.
! ts/logger-ts/package.json: Add "lint" and "format" scripts.

REMOVED:
- Global spyOn(process.stdout, "write") and spyOn(process.stderr, "write").
```

---

## 3. EARS Acceptance Criteria

- **`AC-TS-1.1`**: `WHEN` `stream` option is passed to `createLogger`, `THE` logger `SHALL` write lines to that stream.
- **`AC-TS-1.2`**: `WHEN` child logger is spawned via `logger.child()`, `THE` child `SHALL` inherit the parent's custom stream.
- **`AC-TS-1.3`**: `WHEN` `bun test` runs, `THE` test output `SHALL` display full test names and execution times.
- **`AC-TS-1.4`**: `WHEN` context includes `trace_id` and `span_id`, `THE` logger `SHALL` hoist them to top-level fields.

---

## 4. Exact File Paths & Lines to Modify

1. **`ts/logger-ts/src/types.ts:38-44`**: Add `stream?: { write: (chunk: string) => void };` to `LoggerOptions`.
2. **`ts/logger-ts/src/index.ts:52-62`**: Store `this.stream = options.stream;` in constructor.
3. **`ts/logger-ts/src/index.ts:80-97`**: In `write(entry, isErrorOrFatal)`:
   ```typescript
   if (this.stream && typeof this.stream.write === "function") {
     this.stream.write(line + "\n");
     return;
   }
   ```
4. **`ts/logger-ts/src/index.ts:204-209`**: In `child(options)`: pass `stream: this.stream`.
5. **`ts/logger-ts/tests/logger.test.ts:1-25`**: Replace `spyOn` setup with memory stream object.
6. **`ts/logger-ts/eslint.config.mjs`**: Create new file with Flat Config.

---

## 5. Mandatory Libraries & Verified API Cheat-Sheet

### 5.1 Logger Stream Support (`ts/logger-ts/src/index.ts`)
```typescript
export class Logger implements ILogger {
  private service: string;
  private module: string;
  private minLevel: number;
  private baseContext: Record<string, any>;
  private stream?: { write: (chunk: string) => void };

  constructor(options: {
    service: string;
    module?: string;
    minLevel?: string | number;
    baseContext?: Record<string, any>;
    stream?: { write: (chunk: string) => void };
  }) {
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
    // Fallback to process.stdout / process.stderr
  }
}
```

---

## 6. Step-by-Step TDD Checkboxes

- [x] **Step 1**: Add `stream` and `errorStream` to `LoggerOptions` in `src/types.ts`.
- [x] **Step 2**: Update `Logger` class in `src/index.ts` to respect `this.stream` and `this.errorStream`.
- [x] **Step 3**: Refactor `tests/logger.test.ts` to pass `stream: customStream` and `errorStream: errorStream`.
- [x] **Step 4**: Run `bun test` in `ts/logger-ts` and verify 11 tests pass with transparent test reporter.
- [x] **Step 5**: Create `eslint.config.mjs` and `tsconfig.json` in `ts/logger-ts/`.
- [x] **Step 6**: Add `"lint"`, `"format"`, and `"typecheck"` scripts in `package.json`.
- [x] **Step 7**: Run `bun test`, `bun run typecheck`, and `bun run lint` ensuring 100% pass with 0 errors.

---

## 🔗 Downstream Agent Handoff

- **Target Agents**: `12-code-reviewer`, `13-qa-tester`
- **Component**: `ts/logger-ts`
- **Deliverables**:
  - `ts/logger-ts/src/types.ts`: Stream and errorStream options added to `LoggerOptions`.
  - `ts/logger-ts/src/index.ts`: Custom stream routing in `Logger.write()` and inheritance in `child()`.
  - `ts/logger-ts/tests/logger.test.ts`: 11 unit tests using isolated stream callbacks without global process spy.
  - `ts/logger-ts/eslint.config.mjs`: ESLint Flat Config with `typescript-eslint`.
  - `ts/logger-ts/tsconfig.json`: TypeScript configuration for strict typechecking.
  - `ts/logger-ts/package.json`: Scripts for `test`, `typecheck`, `lint`, and `format`.
- **Verification Summary**:
  - `bun test`: 11/11 tests passing (0 fails, 68 expect calls).
  - `bun run typecheck`: 0 errors.
  - `bun run lint`: 0 errors, 0 warnings.
  - `make test-ts`: 100% green.
