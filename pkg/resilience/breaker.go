package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned when the circuit breaker is open or a probe is already running in half-open state.
var ErrOpen = errors.New("resilience: circuit open")

// State represents the current lifecycle state of a circuit breaker.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// Breaker implements a 3-state circuit breaker.
type Breaker struct {
	mu        sync.Mutex
	state     State
	failures  int
	threshold int
	cooldown  time.Duration
	openUntil time.Time
	probing   bool
	now       func() time.Time
}

// NewBreaker creates a new Breaker with failure threshold and open cooldown duration.
func NewBreaker(threshold int, cooldown time.Duration) *Breaker {
	if threshold <= 0 {
		threshold = 1
	}
	if cooldown <= 0 {
		cooldown = time.Second
	}
	return &Breaker{
		threshold: threshold,
		cooldown:  cooldown,
		now:       time.Now,
	}
}

// State returns the current State of the circuit breaker.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()

	nowFn := b.now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()

	if b.state == StateOpen && !now.Before(b.openUntil) {
		b.state = StateHalfOpen
	}
	return b.state
}

// Execute runs fn protected by the circuit breaker b.
// It is thread-safe and executes fn outside the internal mutex lock.
func Execute[T any](ctx context.Context, b *Breaker, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	if b == nil {
		if fn != nil {
			return fn(ctx)
		}
		return zero, errors.New("resilience: nil breaker and nil function")
	}
	if fn == nil {
		return zero, errors.New("resilience: nil function")
	}

	b.mu.Lock()
	nowFn := b.now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()

	if b.state == StateOpen {
		if now.Before(b.openUntil) {
			b.mu.Unlock()
			return zero, ErrOpen
		}
		b.state = StateHalfOpen
	}

	if b.state == StateHalfOpen {
		if b.probing {
			b.mu.Unlock()
			return zero, ErrOpen
		}
		b.probing = true
	}
	b.mu.Unlock()

	val, err := fn(ctx)

	b.mu.Lock()
	defer b.mu.Unlock()

	if errors.Is(err, context.Canceled) {
		b.probing = false
		return val, err
	}

	if err != nil {
		b.failures++
		if b.state == StateHalfOpen || b.failures >= b.threshold {
			b.state = StateOpen
			nowFn = b.now
			if nowFn == nil {
				nowFn = time.Now
			}
			b.openUntil = nowFn().Add(b.cooldown)
		}
		b.probing = false
		return val, err
	}

	// Success
	b.failures = 0
	b.state = StateClosed
	b.probing = false
	return val, nil
}
