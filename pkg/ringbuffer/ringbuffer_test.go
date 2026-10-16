package ringbuffer_test

import (
	"sync"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/ringbuffer"
)

func TestRingBuffer_BasicPushAndGet(t *testing.T) {
	rb := ringbuffer.New[string](3)
	if rb.Capacity() != 3 {
		t.Fatalf("expected capacity 3, got %d", rb.Capacity())
	}
	if rb.Len() != 0 {
		t.Fatalf("expected len 0, got %d", rb.Len())
	}

	seq1 := rb.Push("a")
	seq2 := rb.Push("b")

	if seq1 != 1 || seq2 != 2 {
		t.Fatalf("unexpected sequence IDs: %d, %d", seq1, seq2)
	}
	if rb.Len() != 2 {
		t.Fatalf("expected len 2, got %d", rb.Len())
	}

	all := rb.GetAll()
	if len(all) != 2 || all[0] != "a" || all[1] != "b" {
		t.Fatalf("unexpected all: %v", all)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := ringbuffer.New[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4) // overwrites 1
	rb.Push(5) // overwrites 2

	if rb.Len() != 3 {
		t.Fatalf("expected len 3, got %d", rb.Len())
	}

	all := rb.GetAll()
	expected := []int{3, 4, 5}
	for i, v := range expected {
		if all[i] != v {
			t.Fatalf("at index %d: expected %d, got %d", i, v, all[i])
		}
	}

	recent2 := rb.GetRecent(2)
	if len(recent2) != 2 || recent2[0] != 4 || recent2[1] != 5 {
		t.Fatalf("unexpected recent 2: %v", recent2)
	}

	recent10 := rb.GetRecent(10)
	if len(recent10) != 3 {
		t.Fatalf("expected recent 10 to clamp to 3, got %d", len(recent10))
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := ringbuffer.New[string](5)
	rb.Push("x")
	rb.Push("y")
	rb.Clear()

	if rb.Len() != 0 {
		t.Fatalf("expected len 0 after clear, got %d", rb.Len())
	}
	if len(rb.GetAll()) != 0 {
		t.Fatalf("expected empty GetAll after clear")
	}
}

func TestRingBuffer_Concurrency(t *testing.T) {
	rb := ringbuffer.New[int](50)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				rb.Push(workerID*1000 + j)
				_ = rb.GetAll()
				_ = rb.GetRecent(5)
			}
		}(i)
	}

	wg.Wait()
	if rb.Len() != 50 {
		t.Fatalf("expected len 50, got %d", rb.Len())
	}
}

func TestRingBuffer_FilterAndContains(t *testing.T) {
	t.Parallel()

	rb := ringbuffer.New[int](4)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4)
	rb.Push(5) // Overwrites 1 -> contents: [2, 3, 4, 5]

	evens := rb.Filter(func(n int) bool {
		return n%2 == 0
	})
	if len(evens) != 2 || evens[0] != 2 || evens[1] != 4 {
		t.Fatalf("expected evens [2, 4], got %v", evens)
	}

	if !rb.ContainsFunc(func(n int) bool { return n == 3 }) {
		t.Fatal("expected ContainsFunc(3) to be true")
	}

	if rb.ContainsFunc(func(n int) bool { return n == 1 }) {
		t.Fatal("expected ContainsFunc(1) to be false after being overwritten")
	}

	var collected []int
	rb.ForEach(func(n int) bool {
		collected = append(collected, n)
		return n < 4 // stop when reaching 4
	})
	if len(collected) != 3 { // 2, 3, 4
		t.Fatalf("expected 3 elements before stopping, got %v", collected)
	}
}

func TestRingBuffer_ForEachReentrant(t *testing.T) {
	t.Parallel()

	rb := ringbuffer.New[int](5)
	rb.Push(10)
	rb.Push(20)
	rb.Push(30)

	done := make(chan struct{})

	go func() {
		defer close(done)
		count := 0
		rb.ForEach(func(item int) bool {
			count++
			// Re-entrant calls to rb methods that require locks
			_ = rb.Len()
			_ = rb.Capacity()
			_ = rb.GetAll()
			return true
		})
		if count != 3 {
			t.Errorf("expected 3 iterations, got %d", count)
		}
	}()

	select {
	case <-done:
		// success: completed without deadlock
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock in re-entrant ForEach call")
	}

	// Empty buffer should not call callback
	emptyRb := ringbuffer.New[int](5)
	emptyInvoked := false
	emptyRb.ForEach(func(n int) bool {
		emptyInvoked = true
		return true
	})
	if emptyInvoked {
		t.Fatal("ForEach invoked callback on empty buffer")
	}
}

func TestRingBuffer_Cap_Latest_Since(t *testing.T) {
	rb := ringbuffer.New[string](3)
	if rb.Cap() != 3 {
		t.Fatalf("expected Cap() == 3, got %d", rb.Cap())
	}

	// Empty Latest
	_, ok := rb.Latest()
	if ok {
		t.Fatal("expected Latest() on empty buffer to return false")
	}

	// Push 1
	rb.Push("first")
	item, ok := rb.Latest()
	if !ok || item != "first" {
		t.Fatalf("expected Latest() to be 'first', got %v (ok=%v)", item, ok)
	}

	rb.Push("second")
	rb.Push("third")
	// Buffer is now [first, second, third], seqCounter = 3

	// Since(0) -> all 3
	s0 := rb.Since(0)
	if len(s0) != 3 || s0[0] != "first" || s0[2] != "third" {
		t.Fatalf("unexpected Since(0): %v", s0)
	}

	// Since(1) -> ["second", "third"]
	s1 := rb.Since(1)
	if len(s1) != 2 || s1[0] != "second" || s1[1] != "third" {
		t.Fatalf("unexpected Since(1): %v", s1)
	}

	// Since(2) -> ["third"]
	s2 := rb.Since(2)
	if len(s2) != 1 || s2[0] != "third" {
		t.Fatalf("unexpected Since(2): %v", s2)
	}

	// Since(3) -> empty
	s3 := rb.Since(3)
	if len(s3) != 0 {
		t.Fatalf("unexpected Since(3): %v", s3)
	}

	// Push fourth (overwrites "first", buffer: ["second", "third", "fourth"], seqCounter = 4)
	rb.Push("fourth")
	item, ok = rb.Latest()
	if !ok || item != "fourth" {
		t.Fatalf("expected Latest() to be 'fourth', got %v", item)
	}

	// Since(0) -> max capacity items ["second", "third", "fourth"]
	s0 = rb.Since(0)
	if len(s0) != 3 || s0[0] != "second" || s0[2] != "fourth" {
		t.Fatalf("unexpected Since(0) after overwrite: %v", s0)
	}
}
