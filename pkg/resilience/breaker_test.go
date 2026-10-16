package resilience_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/resilience"
)

func TestBreaker_Transitions(t *testing.T) {
	t.Parallel()

	now := time.Now()
	nowMu := sync.Mutex{}
	clock := func() time.Time {
		nowMu.Lock()
		defer nowMu.Unlock()
		return now
	}
	advance := func(d time.Duration) {
		nowMu.Lock()
		defer nowMu.Unlock()
		now = now.Add(d)
	}

	b := resilience.NewBreaker(2, 5*time.Second)
	b.SetNow(clock)

	ctx := context.Background()
	dummyErr := errors.New("transient error")

	// 1. Initially closed
	if b.State() != resilience.StateClosed {
		t.Fatalf("expected StateClosed, got %v", b.State())
	}

	// 2. First failure: remains closed (threshold = 2)
	_, err := resilience.Execute(ctx, b, func(ctx context.Context) (string, error) {
		return "", dummyErr
	})
	if !errors.Is(err, dummyErr) {
		t.Fatalf("expected dummyErr, got %v", err)
	}
	if b.State() != resilience.StateClosed {
		t.Fatalf("expected StateClosed after 1 failure, got %v", b.State())
	}

	// 3. Second failure: transitions to StateOpen
	_, err = resilience.Execute(ctx, b, func(ctx context.Context) (string, error) {
		return "", dummyErr
	})
	if !errors.Is(err, dummyErr) {
		t.Fatalf("expected dummyErr, got %v", err)
	}
	if b.State() != resilience.StateOpen {
		t.Fatalf("expected StateOpen after 2 failures, got %v", b.State())
	}

	// 4. Open rejects immediately with ErrOpen without invoking fn
	called := false
	_, err = resilience.Execute(ctx, b, func(ctx context.Context) (string, error) {
		called = true
		return "ok", nil
	})
	if !errors.Is(err, resilience.ErrOpen) {
		t.Fatalf("expected ErrOpen, got %v", err)
	}
	if called {
		t.Fatalf("expected fn not to be called when Open")
	}

	// 5. Advance clock past cooldown -> HalfOpen
	advance(6 * time.Second)
	if b.State() != resilience.StateHalfOpen {
		t.Fatalf("expected StateHalfOpen after cooldown, got %v", b.State())
	}

	// 6. Failed probe in HalfOpen -> transitions back to Open
	_, err = resilience.Execute(ctx, b, func(ctx context.Context) (string, error) {
		return "", dummyErr
	})
	if !errors.Is(err, dummyErr) {
		t.Fatalf("expected dummyErr, got %v", err)
	}
	if b.State() != resilience.StateOpen {
		t.Fatalf("expected StateOpen after failed probe, got %v", b.State())
	}

	// 7. Advance clock past cooldown again -> HalfOpen
	advance(6 * time.Second)
	if b.State() != resilience.StateHalfOpen {
		t.Fatalf("expected StateHalfOpen after cooldown, got %v", b.State())
	}

	// 8. Successful probe in HalfOpen -> transitions to Closed
	val, err := resilience.Execute(ctx, b, func(ctx context.Context) (string, error) {
		return "recovered", nil
	})
	if err != nil || val != "recovered" {
		t.Fatalf("expected success, got val=%q, err=%v", val, err)
	}
	if b.State() != resilience.StateClosed {
		t.Fatalf("expected StateClosed after successful probe, got %v", b.State())
	}
}

func TestBreaker_ConcurrentHalfOpenSingleProbe(t *testing.T) {
	t.Parallel()

	now := time.Now()
	b := resilience.NewBreaker(1, 100*time.Millisecond)
	b.SetNow(func() time.Time {
		return now
	})

	ctx := context.Background()

	// Trip breaker to Open
	_, _ = resilience.Execute(ctx, b, func(ctx context.Context) (int, error) {
		return 0, errors.New("trip")
	})

	// Advance time past cooldown
	now = now.Add(200 * time.Millisecond)

	// In HalfOpen state, run 50 goroutines concurrently. Exactly 1 probe should run.
	const goroutines = 50
	var probeInvocations atomic.Int64
	var openRejections atomic.Int64
	var successResults atomic.Int64

	probeStartCh := make(chan struct{})
	probeReleaseCh := make(chan struct{})

	// Barrier to release all goroutines simultaneously
	startBarrier := make(chan struct{})

	var settledWg sync.WaitGroup
	settledWg.Add(goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-startBarrier
			enteredProbe := false
			_, err := resilience.Execute(ctx, b, func(ctx context.Context) (int, error) {
				enteredProbe = true
				count := probeInvocations.Add(1)
				if count == 1 {
					close(probeStartCh)
				}
				settledWg.Done()
				<-probeReleaseCh
				return 42, nil
			})
			if !enteredProbe {
				settledWg.Done()
			}
			if errors.Is(err, resilience.ErrOpen) {
				openRejections.Add(1)
			} else if err == nil {
				successResults.Add(1)
			}
		}()
	}

	// Release all goroutines simultaneously
	close(startBarrier)

	// Wait for the single probe to begin and all other goroutines to settle while probe is held
	<-probeStartCh
	settledWg.Wait()

	// Release probe
	close(probeReleaseCh)
	wg.Wait()

	if probeInvocations.Load() != 1 {
		t.Fatalf("expected exactly 1 probe execution, got %d", probeInvocations.Load())
	}
	if successResults.Load() != 1 {
		t.Fatalf("expected exactly 1 success result from the probe, got %d", successResults.Load())
	}
	if openRejections.Load() != goroutines-1 {
		t.Fatalf("expected %d ErrOpen rejections, got %d", goroutines-1, openRejections.Load())
	}
	if b.State() != resilience.StateClosed {
		t.Fatalf("expected StateClosed after probe success, got %v", b.State())
	}
}

func TestBreaker_ContextCanceledIgnored(t *testing.T) {
	t.Parallel()

	b := resilience.NewBreaker(1, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resilience.Execute(ctx, b, func(ctx context.Context) (string, error) {
		return "", ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// Should still be closed because context.Canceled does not increment failure count
	if b.State() != resilience.StateClosed {
		t.Fatalf("expected StateClosed after context.Canceled, got %v", b.State())
	}
}

func TestBreaker_NilGuards(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	val, err := resilience.Execute(ctx, nil, func(ctx context.Context) (string, error) {
		return "passthrough", nil
	})
	if err != nil || val != "passthrough" {
		t.Fatalf("expected passthrough on nil breaker, got val=%q err=%v", val, err)
	}

	b := resilience.NewBreaker(0, 0)
	_, err = resilience.Execute[string](ctx, b, nil)
	if err == nil {
		t.Fatalf("expected error for nil fn")
	}
}
