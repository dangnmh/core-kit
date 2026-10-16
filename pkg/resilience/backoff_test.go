package resilience_test

import (
	"context"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/resilience"
)

func TestBackoff_Duration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give int
		want time.Duration
	}{
		{name: "attempt 0", give: 0, want: 100*time.Millisecond - 1},
		{name: "attempt 3", give: 3, want: 800*time.Millisecond - 1},
		{name: "capped", give: 20, want: 5*time.Second - 1},
		{name: "overflow guard", give: 1000, want: 5*time.Second - 1},
		{name: "negative", give: -1, want: 100*time.Millisecond - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := resilience.NewBackoff(100*time.Millisecond, 5*time.Second)
			b.SetRand(func(n int64) int64 {
				return n - 1
			})

			got := b.Duration(tt.give)
			if got != tt.want {
				t.Fatalf("Duration(%d) = %v, want %v", tt.give, got, tt.want)
			}
		})
	}
}

func TestBackoff_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero base", func(t *testing.T) {
		t.Parallel()
		b := resilience.NewBackoff(0, 5*time.Second)
		if got := b.Duration(1); got != 0 {
			t.Fatalf("want 0, got %v", got)
		}
	})

	t.Run("cap less than base", func(t *testing.T) {
		t.Parallel()
		b := resilience.NewBackoff(500*time.Millisecond, 100*time.Millisecond)
		b.SetRand(func(n int64) int64 { return n - 1 })
		if got := b.Duration(0); got != 500*time.Millisecond-1 {
			t.Fatalf("want 499ms, got %v", got)
		}
	})

	t.Run("default random is bounded", func(t *testing.T) {
		t.Parallel()
		b := resilience.NewBackoff(50*time.Millisecond, 200*time.Millisecond)
		for i := 0; i < 20; i++ {
			d := b.Duration(i)
			if d < 0 || d > 200*time.Millisecond {
				t.Fatalf("duration out of bounds: %v", d)
			}
		}
	})
}

func TestBackoff_Sleep(t *testing.T) {
	t.Parallel()

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()
		b := resilience.NewBackoff(1*time.Second, 2*time.Second)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := b.Sleep(ctx, 0)
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("zero duration returns ctx err or nil", func(t *testing.T) {
		t.Parallel()
		b := resilience.NewBackoff(0, 0)
		ctx := context.Background()
		if err := b.Sleep(ctx, 0); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("successful short sleep", func(t *testing.T) {
		t.Parallel()
		b := resilience.NewBackoff(1*time.Millisecond, 2*time.Millisecond)
		ctx := context.Background()
		if err := b.Sleep(ctx, 0); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}
