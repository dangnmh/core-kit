---
title: "Resilience Shared Package (Backoff & Circuit Breaker)"
description: "Full-jitter exponential backoff and generic 3-state circuit breaker for Go microservices"
status: "RELEASED"
last_updated: "2026-10-12"
tier: "package"
service: "core-kit"
related_code:
  - "pkg/resilience/backoff.go"
  - "pkg/resilience/breaker.go"
---

# Package resilience

`package resilience` provides shared fault-tolerance and retry primitives across the `core-kit` ecosystem.

## Components

- **`Backoff`**: Full-jitter exponential backoff algorithm (`sleep = U(0, min(Cap, Base * 2^attempt))`). Prevents thundering herd problems with guaranteed integer overflow protection.
- **`Breaker`**: 3-state circuit breaker (`Closed`, `Open`, `HalfOpen`) with thread-safe single-probe semantics during recovery.
- **`Execute[T]`**: Generic function executing an arbitrary operation wrapped by a `Breaker`. Ignores `context.Canceled` to prevent false positive circuit trips from caller aborts.

## Usage

```go
// Backoff example
b := resilience.NewBackoff(100*time.Millisecond, 5*time.Second)
err := b.Sleep(ctx, attempt)

// Circuit breaker example
cb := resilience.NewBreaker(5, 10*time.Second)
result, err := resilience.Execute(ctx, cb, func(ctx context.Context) (MyResponse, error) {
    return client.Do(ctx)
})
```
