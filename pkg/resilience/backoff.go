package resilience

import (
	"context"
	"math/bits"
	"math/rand/v2"
	"time"
)

// Backoff implements full-jitter exponential backoff.
// Sleep duration is uniformly distributed in [0, min(Cap, Base * 2^attempt)).
type Backoff struct {
	Base time.Duration
	Cap  time.Duration
	rand func(n int64) int64 // injected for tests; defaults to rand.Int64N
}

// NewBackoff creates a new full-jitter Backoff with the given base and cap durations.
func NewBackoff(base, cap time.Duration) Backoff {
	if cap < base {
		cap = base
	}
	return Backoff{
		Base: base,
		Cap:  cap,
		rand: rand.Int64N,
	}
}

// Duration computes the jittered sleep duration for a given attempt number (0-indexed).
func (b Backoff) Duration(attempt int) time.Duration {
	if b.Base <= 0 {
		return 0
	}

	capDur := b.Cap
	if capDur < b.Base {
		capDur = b.Base
	}

	n := attempt
	if n < 0 {
		n = 0
	}

	maxShift := 62
	if capDur > 0 && b.Base > 0 {
		ratio := uint64(capDur / b.Base)
		if l := bits.Len64(ratio); l < maxShift {
			maxShift = l
		}
	}

	shift := min(n, maxShift)
	ceil := b.Base << shift
	if ceil > capDur || ceil <= 0 {
		ceil = capDur
	}

	if ceil <= 0 {
		return 0
	}

	r := b.rand
	if r == nil {
		r = rand.Int64N
	}

	return time.Duration(r(int64(ceil)))
}

// Sleep pauses the current goroutine for Duration(attempt), or returns ctx.Err() if cancelled.
func (b Backoff) Sleep(ctx context.Context, attempt int) error {
	d := b.Duration(attempt)
	if d <= 0 {
		return ctx.Err()
	}

	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
