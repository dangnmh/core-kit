# Security Architecture, Threat Modeling & DevSecOps Specification (`core-kit`)

**Document ID**: `BMAD-SEC-06-CORE-KIT`  
**Version**: `1.0.0`  
**Status**: `APPROVED / BASELINE SECURITY SPECIFICATION`  
**Author**: Role 06: Principal Security Engineer (`06-security-engineer`)  
**Predecessor Documents**: [`docs/bmad/01-project-brief.md`](file:///home/dangnmh/work/core-kit/docs/bmad/01-project-brief.md), [`docs/bmad/02-prd.md`](file:///home/dangnmh/work/core-kit/docs/bmad/02-prd.md), [`docs/bmad/04-architecture.md`](file:///home/dangnmh/work/core-kit/docs/bmad/04-architecture.md), [`docs/bmad/05-database-spec.md`](file:///home/dangnmh/work/core-kit/docs/bmad/05-database-spec.md), [`docs/pkg/contracts/api.md`](file:///home/dangnmh/work/core-kit/docs/pkg/contracts/api.md)  
**Target Next Agents**: Role 06b: Ecosystem Scout (`06b-ecosystem-scout`), Role 07: Tech Lead (`07-tech-lead`), Role 08: Senior Go Dev (`08-senior-go-dev`), Role 09: Senior Python Dev (`09-senior-py-dev`), Role 10: Senior Node/TS Dev (`10-senior-node-dev`), Role 12: Code Reviewer (`12-code-reviewer`)  
**Repository**: [github.com/dangnmh/core-kit](file:///home/dangnmh/work/core-kit)  
**Date**: 2026-10-16  

---

## 1. Executive Security Summary & Security Constitution (`CONST-SEC-01..06`)

### 1.1 Executive Summary
[`core-kit`](file:///home/dangnmh/work/core-kit) provides foundational engineering primitives, distributed telemetry, and networking infrastructure across Go 1.27+, Python 3.14+, and TypeScript/Bun. As a shared foundation consumed by mission-critical algorithmic trading engines (`trading-engine`) and real-time streaming services (`realtime-gateway`), any vulnerability, footgun, or data leak in `core-kit` propagates directly into downstream microservices.

This specification establishes an exhaustive, defense-in-depth security architecture grounded in:
1. **STRIDE Threat Modeling** across all 34 Go packages, Python, and TypeScript modules.
2. **Sharp Edges & Footgun Remediation** to eliminate dangerous defaults, silent failures, and insecure configurations.
3. **Supply-Chain Dependency Risk Audit** identifying zero direct CVEs and resolving 3 flagged transitive vulnerabilities.
4. **Log Injection & Secret Redaction Standards** enforcing uniform PII sanitization across polyglot runtimes.
5. **AWS Kiro EARS Security Acceptance Criteria (`SEC-*` & `SG-*`)** enforced by automated CI quality gates.

---

### 1.2 Security Constitution (`CONST-SEC-01..CONST-SEC-06`)

Every downstream story, sprint task, and code change **MUST** strictly adhere to these 6 non-negotiable security invariants:

- **`CONST-SEC-01` (Fail-Closed Authentication Invariant)**:
  Under NO circumstances may an empty, null, or whitespace-only credential disable authentication. If a required secret or token is missing, the component MUST fail closed (abort startup or reject the request with HTTP `401 Unauthorized`).
- **`CONST-SEC-02` (Constant-Time Verification Invariant)**:
  All cryptographic identifiers, bearer tokens, HMAC signatures, and sensitive comparisons MUST execute in constant time using [`crypto/subtle.ConstantTimeCompare`](file:///home/dangnmh/work/core-kit/pkg/httpserver/auth.go) or language-native equivalents. Direct variable-length string comparisons (`==`) on secrets are strictly forbidden.
- **`CONST-SEC-03` (Zero-Leak Telemetry Invariant)**:
  No plain-text credential, API key, password, session token, or payment card number may EVER be written to log files, traces, or standard output. All logging middlewares ([`pkg/httplog`](file:///home/dangnmh/work/core-kit/pkg/httplog), [`py/logger-py`](file:///home/dangnmh/work/core-kit/py/logger-py), [`ts/logger-ts`](file:///home/dangnmh/work/core-kit/ts/logger-ts)) MUST sanitize payloads before disk write or stream transport.
- **`CONST-SEC-04` (Bounded Resource Allocation Invariant)**:
  All network inputs—including HTTP request bodies, WebSocket incoming frames, and JSON decode streams—MUST be strictly bounded by size limits (`http.MaxBytesReader` / `maxMsgSize`). Unbounded reads (`io.ReadAll` without length guards) are strictly prohibited to prevent Memory Exhaustion Denial of Service (OOM).
- **`CONST-SEC-05` (Entropy Exhaustion Fast-Panic Invariant)**:
  Cryptographic random number generators ([`pkg/idgen`](file:///home/dangnmh/work/core-kit/pkg/idgen)) MUST fail-fast (`panic`) if the underlying CSPRNG (`crypto/rand.Reader`) returns an error. Silently returning static, zero-filled, or predictable pseudorandom values is classified as a Critical vulnerability.
- **`CONST-SEC-06` (Polyglot Log Injection Immunity Invariant)**:
  All log formatters across Go, Python, and TypeScript MUST serialize structured records into single-line JSON with full escaping of newline characters (`\r`, `\n`) and ASCII control codes. Raw string concatenation into log sinks is strictly prohibited.

---

## 2. Threat Modeling & STRIDE Analysis Across Trust Boundaries

### 2.1 Trust Boundaries & Data Flow Diagram (DFD Level 1)

```mermaid
flowchart TB
    subgraph UntrustedZone["External Untrusted Zone (Internet / Public Clients)"]
        Client["External HTTP / WebSocket Client"]
        Attacker["Malicious Adversary / Botnet"]
    end

    subgraph PerimeterBoundary["Perimeter Boundary (Ingress Security Layer)"]
        CORS["CORS Policy & Origin Filter\n(pkg/httpserver, pkg/wsserver)"]
        RateLimiter["Multi-Tier Rate Limiter\n(pkg/ratelimit)"]
        SizeCap["MaxBytesReader (10 MiB Cap)\n(pkg/httplog/gin_middleware)"]
        AuthMiddleware["BearerTokenAuth (Constant-Time)\n(pkg/httpserver/auth.go)"]
    end

    subgraph ServiceCoreBoundary["Internal Service Boundary (In-Memory Core Execution)"]
        GinRouter["Gin Engine Router\n(pkg/httpserver)"]
        WSHub["WebSocket Broadcast Hub\n(pkg/wsserver)"]
        Lifecycle["5-Phase Lifecycle Manager\n(pkg/lifecycle)"]
        SafeMap["Concurrent SafeMap / RingBuffer\n(pkg/safemap, ringbuffer)"]
        EventBus["Watermill EventBus\n(pkg/eventbus)"]
    end

    subgraph OutboundBoundary["Outbound Egress Boundary (External Services & Storage)"]
        HTTPClient["Resilient HTTP Client & Pool\n(pkg/httpclient)"]
        LogPipeline["Telemetry Hub / Log Collector\n(pkg/httplog, logger-py, logger-ts)"]
        Exchanges["Upstream Crypto Exchanges / LLM APIs"]
    end

    Client -->|HTTP Request / WS Upgrade| CORS
    Attacker -->|Credential Stuffing / Slowloris| RateLimiter
    RateLimiter --> SizeCap
    SizeCap --> AuthMiddleware
    AuthMiddleware -->|Authenticated Request| GinRouter
    AuthMiddleware -->|WebSocket Conn| WSHub

    GinRouter --> SafeMap
    GinRouter --> EventBus
    WSHub --> EventBus

    EventBus --> HTTPClient
    HTTPClient -->|Outbound TLS with Retry/CircuitBreaker| Exchanges
    GinRouter -->|Sanitized JSON Logs| LogPipeline
    WSHub -->|Connection Lifecycle Logs| LogPipeline

    style UntrustedZone fill:#ffebee,stroke:#c62828,stroke-width:2px
    style PerimeterBoundary fill:#fff3e0,stroke:#ef6c00,stroke-width:2px
    style ServiceCoreBoundary fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px
    style OutboundBoundary fill:#e1f5fe,stroke:#0277bd,stroke-width:2px
```

---

### 2.2 Comprehensive STRIDE Threat Matrix

| Threat ID | STRIDE Category | Affected Component | Attack Scenario / Vulnerability Vector | Severity | DREAD Score | Required Mitigation & Control ID |
| :--- | :--- | :--- | :--- | :---: | :---: | :--- |
| **THREAT-01** | **Spoofing** | [`pkg/httpserver/auth.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/auth.go) | Attacker accesses protected admin endpoints when `token` parameter is empty string, which currently logs a warning and allows requests through (`c.Next()`). | **CRITICAL** | **9.2** (High) | **`SEC-CTRL-01`**: Refactor `BearerTokenAuth` to fail closed; reject startup if token empty, or return 401 on every request. |
| **THREAT-02** | **Spoofing** | [`pkg/wsserver/server.go`](file:///home/dangnmh/work/core-kit/pkg/wsserver/server.go) | Attacker establishes WebSocket connection from unauthorized web origin (CSWSH) or without bearer credential. | **HIGH** | **7.8** (High) | **`SEC-CTRL-02`**: Require mandatory origin matching and pre-upgrade authentication hooks in `wsserver`. |
| **THREAT-03** | **Tampering** | [`pkg/idgen/idgen.go`](file:///home/dangnmh/work/core-kit/pkg/idgen/idgen.go) | Entropy exhaustion causes `rand.Read` error, leading `NewPrefixedID` to return identical zero-filled IDs (`"evt_00000000"`), enabling ID collisions and state hijacking. | **HIGH** | **8.0** (High) | **`SEC-CTRL-03`**: Panic on `rand.Read` error (`CONST-SEC-05`), eliminating predictable zero-byte ID generation. |
| **THREAT-04** | **Tampering** | [`pkg/ws/dispatcher.go`](file:///home/dangnmh/work/core-kit/pkg/ws/dispatcher.go) | Malicious WebSocket peer injects forged request IDs to hijack pending RPC promises or prematurely close internal channels. | **MEDIUM** | **6.4** (Med) | **`SEC-CTRL-04`**: Enforce 21-character NanoIDs for all RPC requests and validate response ID ownership. |
| **THREAT-05** | **Repudiation** | [`pkg/eventbus`](file:///home/dangnmh/work/core-kit/pkg/eventbus) | Downstream actor executes trading order or state mutation, but timeline audit log is incomplete or dropped without correlation ID. | **MEDIUM** | **5.8** (Med) | **`SEC-CTRL-05`**: Mandate `msg_id`, `timestamp`, and W3C `trace_id` injection in all `Timeline` audit log entries. |
| **THREAT-06** | **Information Disclosure** | [`pkg/httplog/sanitizer.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/sanitizer.go) | Malformed or partial JSON request body containing passwords causes `json.Unmarshal` to fail, resulting in raw unsanitized body being emitted to disk logs. | **HIGH** | **8.4** (High) | **`SEC-CTRL-06`**: Fallback to regex scrubbing and length truncation when JSON unmarshal fails; never log raw unparsed payloads. |
| **THREAT-07** | **Information Disclosure** | [`py/logger-py`](file:///home/dangnmh/work/core-kit/py/logger-py), [`ts/logger-ts`](file:///home/dangnmh/work/core-kit/ts/logger-ts) | Python and TypeScript loggers output sensitive dictionary fields (`password`, `api_key`, `secret`) in `context` or `extra` without redaction. | **HIGH** | **8.2** (High) | **`SEC-CTRL-07`**: Implement recursive sensitive key redaction in Python `JsonFormatter` and TypeScript `Logger`. |
| **THREAT-08** | **Information Disclosure** | [`pkg/httpserver/server.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/server.go) | Permissive default CORS (`AllowAllOrigins: true`) when `AllowedOrigins` is empty exposes internal APIs to browser-based cross-origin leakage. | **MEDIUM** | **6.2** (Med) | **`SEC-CTRL-08`**: Default CORS to strict same-origin unless explicit origins are configured; explicitly disable `AllowCredentials`. |
| **THREAT-09** | **Denial of Service** | [`pkg/httplog/gin_middleware.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/gin_middleware.go) | Attacker streams 2 GB request body to HTTP server; middleware calls unbounded `io.ReadAll(c.Request.Body)`, triggering OOM crash. | **HIGH** | **8.6** (High) | **`SEC-CTRL-09`**: Wrap request body with `http.MaxBytesReader(c.Writer, c.Request.Body, maxBodySize)` (10 MiB limit). |
| **THREAT-10** | **Denial of Service** | [`pkg/wsserver/server.go`](file:///home/dangnmh/work/core-kit/pkg/wsserver/server.go) | Slow consumer attacks: malicious client connects and refuses to read TCP packets, accumulating unbounded memory in server send queues. | **MEDIUM** | **6.6** (Med) | **`SEC-CTRL-10`**: Non-blocking channel send (`select default`) with immediate eviction of slow consumers (`dropSlowClient`). |
| **THREAT-11** | **Denial of Service** | [`pkg/httpclient/client.go`](file:///home/dangnmh/work/core-kit/pkg/httpclient/client.go) | Upstream exchange returns infinite chunked response stream; client consumes memory until exhaustion. | **MEDIUM** | **6.8** (Med) | **`SEC-CTRL-11`**: Enforce `io.LimitReader(resp.Body, maxResponseBytes)` with default 8 MiB limit (`ErrBodyTooLarge`). |
| **THREAT-12** | **Elevation of Privilege** | [`pkg/config/env.go`](file:///home/dangnmh/work/core-kit/pkg/config/env.go) | Attacker injects environment variables into unvalidated struct fields, overriding security policies or enabling debug modes in production. | **HIGH** | **7.6** (High) | **`SEC-CTRL-12`**: Run `validator.Validate(target)` via `go-playground/validator/v10` immediately following `applyEnv`. |

---

## 3. Authentication, Authorization & Session Security Controls

### 3.1 Bearer Token Authentication Hardening (`pkg/httpserver/auth.go`)

The current implementation contains a critical sharp edge:
```go
// CURRENT DEFECTIVE PATTERN (pkg/httpserver/auth.go:16)
trimmedToken := strings.TrimSpace(token)
if trimmedToken == "" {
    logger.Warn("BearerTokenAuth initialized with empty token; authentication is disabled")
    return func(c *gin.Context) { c.Next() } // DANGEROUS BYPASS!
}
```

#### Remediation Architecture (`SEC-CTRL-01`)
1. **Startup Fail-Closed Validation**: If a server is instantiated with an empty auth token, initialization MUST panic or terminate immediately.
2. **Runtime Constant-Time Rejection**: If configured dynamically, an empty token must reject 100% of incoming requests with HTTP `401 Unauthorized`.
3. **Constant-Time Comparison**: Comparison must always execute in $O(1)$ time regardless of match failure position using `subtle.ConstantTimeCompare`:

```go
// REMEDIATED PRODUCTION CONTRACT
func BearerTokenAuth(expectedToken string) gin.HandlerFunc {
    trimmed := strings.TrimSpace(expectedToken)
    if trimmed == "" {
        // Enforce CONST-SEC-01: Never allow unauthenticated execution
        panic("httpserver.BearerTokenAuth: cannot initialize with empty token")
    }
    expectedBytes := []byte(trimmed)

    return func(c *gin.Context) {
        authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
        const prefix = "bearer "
        if len(authHeader) < len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "message": "missing or malformed authorization header",
            })
            return
        }
        providedBytes := []byte(strings.TrimSpace(authHeader[len(prefix):]))
        if subtle.ConstantTimeCompare(providedBytes, expectedBytes) != 1 {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "message": "invalid credentials",
            })
            return
        }
        c.Next()
    }
}
```

---

### 3.2 Dual-Role RBAC Reference Architecture for Downstream Services

For downstream services implementing Role-Based Access Control (`admin` vs `user` / `system` vs `public`), `core-kit` provides standard context key and claim injection helpers:

```mermaid
flowchart LR
    Token["Bearer JWT / API Key"] --> AuthN["Authentication Filter\n(Identity & Scope Validation)"]
    AuthN --> Claims["Context Claims Injection\n(user_id, role, scopes)"]
    Claims --> AuthZ["Role Guard Middleware\nRequireRole('admin')"]
    AuthZ -->|Authorized| Handler["Protected Endpoint Handler"]
    AuthZ -->|Forbidden| Err403["HTTP 403 Forbidden\n{error: 'insufficient_permissions'}"]
```

#### RBAC Claims Contract
```go
type SecurityContext struct {
    Subject   string   `json:"sub"`
    Role      string   `json:"role"` // "admin", "service", "read_only", "user"
    Scopes    []string `json:"scopes"`
    ExpiresAt int64    `json:"exp"`
}
```

---

### 3.3 WebSocket Handshake Security & CSWSH Defense (`pkg/wsserver`)

WebSockets are not protected by the Same-Origin Policy in browsers. Without explicit validation, an attacker's website can initiate a WebSocket connection to the server on behalf of a victim user (Cross-Site WebSocket Hijacking - CSWSH).

#### Controls Enforced:
1. **Origin Verification (`WithAllowedOrigins`)**:
   - In production, `origins` MUST NOT contain `*`.
   - Origin header is validated against an exact allowlist: `https://app.domain.com`.
   - Non-browser clients (empty `Origin` header) are permitted only if presenting a valid authentication ticket or token.
2. **Ticket-Based Pre-Upgrade Authentication**:
   - WebSockets cannot send custom `Authorization: Bearer` headers during standard browser `new WebSocket(url)` creation.
   - Downstream services MUST generate a short-lived (60-second), single-use cryptographically random ticket (via `pkg/idgen.NanoID(32)`) exchanged over authenticated REST, then passed as a query parameter: `/ws?ticket=<ticket>`.
   - The ticket is verified and invalidated atomically before calling `s.Upgrade(w, r)`.

---

## 4. Anti-Bot, Multi-Tier Rate Limiting & Resource Exhaustion (DoS) Defenses

### 4.1 Multi-Tier Rate Limiting Architecture (`pkg/ratelimit`)

To defend downstream algorithmic orderbook gateways and public APIs from automated scrapers, botnets, and resource starvation, `pkg/ratelimit` implements a 3-tier sliding-window / token-bucket rate limiter:

```mermaid
flowchart TD
    Req["Incoming HTTP / WS Request"] --> T1["Tier 1: Global IP Burst Limiter\n(100 req/sec per IP -> HTTP 429)"]
    T1 -->|Pass| T2["Tier 2: Authenticated Client Limiter\n(Token Bucket: 1,000 req/min)"]
    T2 -->|Pass| T3["Tier 3: Endpoint Weight Limiter\n(Cost: Standard=1, Heavy Math/LLM=10)"]
    T3 -->|Pass| Exec["Service Execution Core"]

    T1 -->|Exceeded| Block1["HTTP 429 Too Many Requests\nRetry-After: <seconds>"]
    T2 -->|Exceeded| Block2["HTTP 429 Too Many Requests\nX-RateLimit-Reset: <timestamp>"]
    T3 -->|Exceeded| Block3["HTTP 429 Resource Exhausted\nBackoff Escalation"]
```

#### Rate Limit Response Headers Contract
Every rate-limited endpoint MUST emit standard IETF rate-limiting headers:
- `X-RateLimit-Limit`: Maximum requests permitted within window.
- `X-RateLimit-Remaining`: Remaining budget in current window.
- `X-RateLimit-Reset`: UTC epoch timestamp when the window resets.
- `Retry-After`: Number of seconds to wait before retrying (on 429).

---

### 4.2 Inbound Payload Size Protection (`http.MaxBytesReader`)

#### Vulnerability Analysis (`THREAT-09`)
In [`pkg/httplog/gin_middleware.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/gin_middleware.go), request bodies are read into memory:
```go
// VULNERABLE: Unbounded ReadAll
reqBodyBytes, err = io.ReadAll(c.Request.Body)
```
An attacker streaming an unbounded multi-gigabyte payload will trigger process memory exhaustion and crash the container via the Linux OOM-killer.

#### Required Remediation (`SEC-CTRL-09`)
The middleware must wrap `c.Request.Body` with `http.MaxBytesReader`:
```go
maxBytes := int64(10 * 1024 * 1024) // 10 MiB default cap
if engine != nil && engine.MaxBodySize() > 0 {
    maxBytes = int64(engine.MaxBodySize())
}
c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
```
If the payload exceeds `maxBytes`, `MaxBytesReader` returns an error, Gin returns `413 StatusRequestEntityTooLarge`, and server memory is preserved.

---

### 4.3 WebSocket Slow-Consumer & Message Buffer Protection

In [`pkg/wsserver`](file:///home/dangnmh/work/core-kit/pkg/wsserver), slow consumers are mitigated via:
1. **Per-Client Channel Buffer Cap**: `defaultSendBuf = 64`.
2. **Non-Blocking Eviction**: If a client fails to consume broadcast data and its buffer fills, `BroadcastDetailed` drops the client immediately (`result.Dropped++`), closes its connection, and frees buffer memory.
3. **Inbound Frame Size Cap**: `defaultMaxMsgSize = 512 * 1024` (512 KiB). Any incoming frame exceeding 512 KiB triggers immediate connection closure with WebSocket close code `1009 (Message Too Big)`.

---

## 5. Secret Redaction, Log Sanitization & Injection Prevention

### 5.1 Tri-Language Canonical Secret Redaction Specification

All logging engines across Go, Python, and TypeScript MUST enforce uniform redaction policies.

```mermaid
flowchart LR
    RawRecord["Raw Log Record / HTTP Payload"] --> KeyCheck{"Matches Sensitive Key\nor Header?"}
    KeyCheck -->|Yes| Strategy{"Configured Strategy"}
    Strategy -->|Redact| RedactVal["[REDACTED]"]
    Strategy -->|Partial| PartialMask["abc****xyz"]
    Strategy -->|Email| EmailMask["u***r@domain.com"]
    Strategy -->|SHA256| HashVal["sha256:<hex>"]
    Strategy -->|Remove| DropKey["(Key Deleted)"]
    KeyCheck -->|No| PlainVal["Original Value"]
```

#### Canonical List of Sensitive Keys & Headers

| Category | Targets (Case-Insensitive Match) | Default Strategy | Replacement |
| :--- | :--- | :---: | :--- |
| **HTTP Headers** | `Authorization`, `Proxy-Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`, `X-Auth-Token` | `StrategyRedact` | `[REDACTED]` |
| **Authentication**| `password`, `passwd`, `pass`, `token`, `secret`, `api_key`, `apikey`, `access_token`, `refresh_token`, `private_key` | `StrategyRedact` | `[REDACTED]` |
| **PII: Identity** | `ssn`, `tax_id`, `national_id`, `id_card`, `passport` | `StrategyRedact` | `[REDACTED]` |
| **PII: Contact**  | `email` | `StrategyEmailMask` | `u***r@example.com` |
| **PII: Financial**| `credit_card`, `card_number`, `pan`, `cvv`, `cvc`, `account_number`, `iban` | `StrategyPartialMask`| `1234********5678` |

---

### 5.2 Malformed JSON Secret Leak Remediation (`pkg/httplog`)

#### Vulnerability Analysis (`THREAT-06`)
In `pkg/httplog/sanitizer.go`:
```go
// CURRENT DEFECTIVE PATTERN (sanitizer.go:98)
var root any
if err := json.Unmarshal(body, &root); err != nil {
    return string(body) // CRITICAL: Emits unparsed payload with plaintext passwords!
}
```

#### Required Remediation (`SEC-CTRL-06`)
When `json.Unmarshal` fails on request payloads with JSON Content-Type:
1. Apply regex fallback scrubbing against sensitive key-value patterns:
   `(?i)("(?:password|secret|token|api_key|authorization)"\s*:\s*")([^"]+)(")` $\rightarrow$ `$1[REDACTED]$3`.
2. If regex fails or payload is binary corrupted, replace entirely with:
   `"[MALFORMED_JSON_PAYLOAD_REDACTED]"`.

---

### 5.3 Polyglot Log Injection (CRLF / Control-Character) Prevention

Log injection occurs when untrusted input contains newline characters (`\r`, `\n`) allowing an attacker to forge log entries, spoof severity levels, or corrupt SIEM analytics:
```text
USER INPUT: "order_created\n{\"timestamp\":\"2026-10-16T12:00:00Z\",\"severity\":\"CRITICAL\",\"service\":\"auth\",\"message\":\"Admin privileges granted to attacker\"}"
```

#### Tri-Language Mitigation Contract:
- **Go (`pkg/logger`)**: Native `log/slog.NewJSONHandler` automatically escapes `\r` to `\r` and `\n` to `\n` inside JSON string attributes.
- **Python (`py/logger-py`)**: `JsonFormatter` MUST use `json.dumps(payload, ensure_ascii=False)`, which strictly encodes newlines as `\n` within string literals. Emojis and decorative symbols are stripped by `_strip_emojis`.
- **TypeScript (`ts/logger-ts`)**: `Logger.write` uses native `JSON.stringify(entry)`, guaranteeing strict RFC8259 escaping of all control characters (`U+0000` through `U+001F`).

---

## 6. Sharp Edges, Footgun Analysis & Misuse-Resistance Audit

Following the **Sharp Edges Analysis** framework, we evaluated all public APIs and configurations across `core-kit` to eliminate footguns where the "path of least resistance" leads to insecurity:

| Footgun ID | Component & File | Dangerous Pattern / Sharp Edge | Risk & Misuse Scenario | Required Misuse-Resistant Refactoring |
| :--- | :--- | :--- | :--- | :--- |
| **`EDGE-01`** | [`pkg/httpserver/auth.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/auth.go) | `BearerTokenAuth("")` disables authentication silently with a warning log. | Developer forgets to wire `AUTH_TOKEN` in Docker Compose; API boots in production completely unauthenticated. | **Fail-Closed**: Constructor panics on empty token; rejects requests with 401 if token is unconfigured. |
| **`EDGE-02`** | [`pkg/idgen/idgen.go`](file:///home/dangnmh/work/core-kit/pkg/idgen/idgen.go) | `rand.Read(b)` error silently falls back to formatting all-zero byte slice (`evt_0000000000000000`). | Entropy pool depletion causes thousands of transactions to receive identical ID `evt_00000000`, causing DB primary key collisions. | **Fast-Panic**: If `rand.Read` returns error, invoke `panic(fmt.Errorf("idgen: crypto/rand failure: %w", err))` (`CONST-SEC-05`). |
| **`EDGE-03`** | [`pkg/httplog/sanitizer.go`](file:///home/dangnmh/work/core-kit/pkg/httplog/sanitizer.go) | `SanitizeJSON` returns raw string unmodified if `json.Unmarshal` returns error. | Client submits invalid trailing comma in JSON login body `{password: "secret",}`; log file records plaintext password. | **Safe Fallback**: Run regex scrubber over raw text or replace with `[MALFORMED_JSON_REDACTED]`. |
| **`EDGE-04`** | [`ts/logger-ts/src/index.ts`](file:///home/dangnmh/work/core-kit/ts/logger-ts/src/index.ts) | `createLogger` interface declares `stream` option in `LoggerOptions`, but constructor completely ignores it and writes directly to `process.stdout`. | Unit test runners cannot capture logger output cleanly; developers mock `process.stdout.write` globally, silencing terminal test output. | **Respect Stream**: Store `stream` in `Logger` and write to `this.stream.write(line + "\n")` defaulting to `process.stdout`. |
| **`EDGE-05`** | [`pkg/httpserver/server.go`](file:///home/dangnmh/work/core-kit/pkg/httpserver/server.go) | When `Config.AllowedOrigins` is empty, `corsCfg.AllowAllOrigins` defaults to `true` (`*`). | Developer provisions internal service without configuring origins; any public website can make cross-origin requests. | **Secure Default**: Empty `AllowedOrigins` defaults to same-origin only (`AllowAllOrigins = false`). |
| **`EDGE-06`** | [`pkg/safemap/safemap.go`](file:///home/dangnmh/work/core-kit/pkg/safemap) | SafeMap `Compute(key, fn)` executes `fn` while holding internal `sync.RWMutex.Lock()`. | Developer calls `s.Get()` or `s.Set()` inside `fn`, triggering an unrecoverable recursive deadlock. | **Documentation & Assertions**: Explicitly document deadlock prohibition in contract and add lock reentrancy detection in debug builds. |

---

## 7. Dependency Supply-Chain Risk Audit & Vulnerability Remediation

### 7.1 Deterministic Tooling Audit Execution
Using the official `supply-chain-risk-auditor` engine (`scripts/collect.py` and `scripts/render.py`), a comprehensive supply-chain audit was performed across all repository package manifests:
- **Manifests Evaluated**: Go (`go.mod`), Python (`py/logger-py/pyproject.toml`), TypeScript (`ts/logger-ts/package.json`).
- **Direct Dependencies Evaluated**: 18 direct Go packages.
- **Transitive Lockfile Packages Evaluated**: 53 registry-verified packages.

### 7.2 Summary of Findings
- **Direct Dependencies**: **0 known advisories** among all 18 direct dependencies.
- **Python Component (`logger-py`)**: **0 external dependencies** (Pure stdlib). Zero supply-chain exposure.
- **TypeScript Component (`logger-ts`)**: **0 external runtime dependencies**. Zero supply-chain exposure.
- **Transitive Dependencies**: **3 packages carry known advisories** at the currently resolved versions in `go.mod`.

---

### 7.3 Detailed Transitive Advisories & Remediation Plan

| Package | Resolved Version | Ecosystem | Advisory IDs | Vulnerability Nature | Upstream Fix & Required Action |
| :--- | :---: | :---: | :--- | :--- | :--- |
| **`golang.org/x/net`** | `0.58.0` | Go (Transitive) | `GO-2026-6603`<br>`GO-2026-6610`<br>`GO-2026-6611`<br>`GO-2026-6612`<br>`GO-2026-6617` | HTTP/2 stream cancellation flood, rapid reset frame amplification, and header parsing memory exhaustion (DoS). | **Upgrade**: Run `go get golang.org/x/net@v0.59.0` (or latest stable) in `go.mod` to clear all 5 advisories. |
| **`golang.org/x/crypto`** | `0.57.0` | Go (Transitive) | `GO-2026-5932` | Cryptographic edge-case panic or timing vulnerability in curve arithmetic. | **Upgrade**: Run `go get golang.org/x/crypto@v0.58.0` (or latest stable) to resolve. |
| **`github.com/redis/go-redis/v9`** | `9.7.0` | Go (Transitive) | `GHSA-92cp-5422-2mw7`<br>`GO-2025-3540` | Redis client protocol connection leak or buffer reuse flaw under concurrent pipelining. | **Upgrade**: Run `go get github.com/redis/go-redis/v9@v9.7.1` (or latest release) to clear advisory. |

#### Remediation Target in `Makefile`:
The automated dependency upgrade target `make upgrade-go` (Story 3.4) MUST execute:
```sh
go get -u golang.org/x/net@latest golang.org/x/crypto@latest github.com/redis/go-redis/v9@latest
go mod tidy
```
This will eliminate 100% of transitive CVEs prior to production release.

---

### 7.4 OpenSSF Scorecard Hygiene Matrix

| Dependency Repository | Scorecard Score | Dangerous Workflow | Binary Artifacts | Code Review | Branch Protection |
| :--- | :---: | :---: | :---: | :---: | :---: |
| `github.com/open-telemetry/opentelemetry-go` | **9.4 / 10** | Clean (10/10) | Clean (10/10) | Clean (10/10) | Enforced |
| `github.com/gin-gonic/gin` | **7.8 / 10** | Clean (10/10) | Clean (10/10) | Clean (8/10) | Enforced |
| `github.com/stretchr/testify` | **8.2 / 10** | Clean (10/10) | Clean (10/10) | Clean (9/10) | Enforced |
| `github.com/shopspring/decimal` | **7.0 / 10** | Clean (10/10) | Clean (10/10) | Clean (6/10) | Active |
| `github.com/gorilla/websocket` | **6.5 / 10** | Unscored | Clean (10/10) | Clean (7/10) | Maintained |
| `github.com/ThreeDotsLabs/watermill` | **7.2 / 10** | Clean (10/10) | Clean (10/10) | Clean (7/10) | Active |

---

## 8. Security Acceptance Criteria (`SEC-*` & `SG-*` Guardrails)

Following **AWS Kiro EARS syntax** (`WHEN/IF ... SHALL ...`) and **Given/When/Then**, the following mandatory acceptance criteria are injected into downstream implementation stories:

### 8.1 Core Security Acceptance Criteria (`SEC-*`)

#### `SEC-01`: Fail-Closed Bearer Token Authentication
- **EARS**: `IF httpserver.BearerTokenAuth is initialized with an empty or whitespace-only token THEN THE component SHALL panic immediately during service startup.`
- **Given**: An HTTP service configured with `AUTH_TOKEN=""` or unset in environment.
- **When**: The service invokes `httpserver.BearerTokenAuth("")`.
- **Then**: Process aborts immediately with a descriptive panic message; no unauthenticated HTTP listener is bound.

#### `SEC-02`: Bounded Request Body Memory Protection
- **EARS**: `WHEN any inbound HTTP request is processed by httplog.GinMiddleware THE middleware SHALL wrap the request body with http.MaxBytesReader constrained to at most 10 MiB.`
- **Given**: An inbound HTTP request with a body size exceeding 10 MiB.
- **When**: The client streams the oversized payload to the server.
- **Then**: The read operation aborts with `http.ErrBodyTooLarge`, the server responds with HTTP `413 Payload Too Large`, and process memory consumption does not exceed the limit.

#### `SEC-03`: Malformed Payload Secret Masking Invariant
- **EARS**: `IF an inbound HTTP request contains malformed or unparseable JSON THEN httplog.SanitizeJSON SHALL scrub sensitive keys using regex fallback or replace the body with [MALFORMED_JSON_REDACTED] without leaking plaintext secrets.`
- **Given**: A raw request body `{"password":"supersecret",invalid_json`.
- **When**: `httplog.SanitizeJSON` processes the body.
- **Then**: The resulting sanitized log string DOES NOT contain `"supersecret"`.

#### `SEC-04`: Cryptographic Random Failure Fast-Panic Invariant
- **EARS**: `IF crypto/rand.Reader returns an error during ID generation THEN idgen.NewPrefixedID and idgen.NanoID SHALL panic immediately rather than returning zero-filled or predictable bytes.`
- **Given**: A simulated CSPRNG entropy failure in `rand.Read`.
- **When**: `idgen.NewPrefixedID("evt")` is invoked.
- **Then**: The runtime raises a panic; it NEVER returns `"evt_0000000000000000"`.

#### `SEC-05`: Polyglot Log Injection Prevention & Control Character Escaping
- **EARS**: `WHEN a log record contains newline characters or ASCII control codes THE loggers in pkg/logger, py/logger-py, and ts/logger-ts SHALL escape newlines as \n within JSON strings to guarantee single-line emission.`
- **Given**: A log message containing `"user login\nCRITICAL admin created"`.
- **When**: Logged at level INFO across Go, Python, or TypeScript.
- **Then**: The output to stdout/stderr is strictly ONE physical line of valid JSON with `"message":"user login\\nCRITICAL admin created"`.

#### `SEC-06`: Transitive Dependency Vulnerability Remediation
- **EARS**: `WHEN make upgrade or make ci is executed THE repository SHALL resolve all direct and transitive Go dependencies such that 0 known CVEs affect locked versions.`
- **Given**: `go.mod` after `make upgrade-go`.
- **When**: Running the vulnerability scanner.
- **Then**: 0 advisories are flagged for `golang.org/x/net`, `golang.org/x/crypto`, and `go-redis/v9`.

---

### 8.2 Security Quality Gate Checks (`SG-*`)

| Gate ID | Automated Verification Command | Gate Assertion / Passing Condition | Failure Action |
| :--- | :--- | :--- | :--- |
| **`SG-01`** | `go test -race ./...` | **0 data races detected** across 10,000 concurrent ops | Blocks CI / Reject PR |
| **`SG-02`** | `golangci-lint run ./...` | **0 warnings** (enforcing `errcheck`, `gosec`, `govet`) | Blocks CI / Reject PR |
| **`SG-03`** | `python3 -m pytest py/logger-py/tests/` | **100% pass rate** including log injection & emoji tests | Blocks CI / Reject PR |
| **`SG-04`** | `cd ts/logger-ts && bun test` | **100% pass rate** with native terminal output visibility | Blocks CI / Reject PR |
| **`SG-05`** | `make doc-lint` | **0 broken links**, `<= 120` lines on Layer 0/1 | Blocks CI / Reject PR |

---

## 🔗 Downstream Agent Handoff

- **Target Next Agents**:
  1. `06b-ecosystem-scout` (Role 06b: Ecosystem & Library Scout): Audit and verify replacement/upgrade candidates for transitive Go packages (`golang.org/x/net`, `golang.org/x/crypto`, `github.com/redis/go-redis/v9`).
  2. `07-tech-lead` (Role 07: Tech Lead & Implementation Planner): Ingest `SEC-01` through `SEC-06` and sharp-edge remediations (`EDGE-01`..`EDGE-06`) directly into sprint implementation stories (`docs/bmad/stories/story-*.md`).
  3. `08-senior-go-dev` (Role 08: Senior Go Developer): Implement `BearerTokenAuth` fail-closed refactoring, `idgen` CSPRNG panic guard, and `httplog` `MaxBytesReader` size bounding.
  4. `09-senior-py-dev` (Role 09: Senior Python Developer): Implement sensitive context key redaction in `py/logger-py`.
  5. `10-senior-node-dev` (Role 10: Senior Node/TS Developer): Wire custom `stream` option in `ts/logger-ts/src/index.ts` to restore test runner isolation without stdout mocking.
  6. `12-code-reviewer` (Role 12: Code Reviewer): Enforce `CONST-SEC-01..06` and Security Quality Gates `SG-01..05` during differential PR review.

- **Mandatory Sections / Contracts to Consume**:
  - Section 1.2: Security Constitution (`CONST-SEC-01` through `CONST-SEC-06`).
  - Section 2.2: STRIDE Threat Matrix (`THREAT-01` through `THREAT-12`).
  - Section 3.1: Bearer Token Fail-Closed Contract (`SEC-CTRL-01`).
  - Section 4.2: Payload Size Protection with `http.MaxBytesReader` (`SEC-CTRL-09`).
  - Section 5.1 & 5.2: Tri-Language Secret Redaction & Malformed JSON Scrubbing.
  - Section 6: Sharp Edges & Footgun Audit (`EDGE-01` through `EDGE-06`).
  - Section 7.3: Transitive Dependency Upgrade Plan.
  - Section 8: Security Acceptance Criteria (`SEC-01` through `SEC-06`).

- **Expected Downstream Output Files**:
  - `docs/bmad/06b-library-manifest.md` (Updated with verified secure package versions).
  - `docs/bmad/stories/story-3.1-idgen-consolidation.md` (Injecting `SEC-04` panic guard).
  - `docs/bmad/stories/story-3.3-linter-remediation.md` (Injecting `SEC-01` and `SEC-02` fixes).
  - `docs/bmad/stories/story-2.4-ts-logger-isolation.md` (Injecting `EDGE-04` stream fix).
