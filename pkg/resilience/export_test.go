package resilience

import "time"

// SetNow overrides the clock function for testing.
func (b *Breaker) SetNow(fn func() time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.now = fn
}

// SetRand overrides the random generator function for testing.
func (b *Backoff) SetRand(fn func(int64) int64) {
	b.rand = fn
}
