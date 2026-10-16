package safemap_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/dangnmh/core-kit/pkg/safemap"
)

func TestSafeMap_BasicOperations(t *testing.T) {
	t.Parallel()

	sm := safemap.New[string, int]()

	if sm.Len() != 0 {
		t.Fatalf("expected length 0, got %d", sm.Len())
	}

	sm.Set("apple", 10)
	sm.Set("banana", 20)

	if sm.Len() != 2 {
		t.Fatalf("expected length 2, got %d", sm.Len())
	}

	if val, ok := sm.Get("apple"); !ok || val != 10 {
		t.Fatalf("expected apple=10, got %v, ok=%v", val, ok)
	}

	if !sm.Has("banana") {
		t.Fatal("expected banana to exist")
	}

	if sm.Has("cherry") {
		t.Fatal("expected cherry to not exist")
	}

	sm.Delete("apple")
	if sm.Has("apple") {
		t.Fatal("expected apple to be deleted")
	}

	if sm.Len() != 1 {
		t.Fatalf("expected length 1, got %d", sm.Len())
	}
}

func TestSafeMap_GetOrSet(t *testing.T) {
	t.Parallel()

	sm := safemap.New[string, string]()

	val, loaded := sm.GetOrSet("key", "initial")
	if loaded || val != "initial" {
		t.Fatalf("expected loaded=false, val=initial; got loaded=%v, val=%v", loaded, val)
	}

	val, loaded = sm.GetOrSet("key", "new")
	if !loaded || val != "initial" {
		t.Fatalf("expected loaded=true, val=initial; got loaded=%v, val=%v", loaded, val)
	}
}

func TestSafeMap_KeysValuesRangeClear(t *testing.T) {
	t.Parallel()

	sm := safemap.New[int, string]()
	sm.Set(1, "one")
	sm.Set(2, "two")
	sm.Set(3, "three")

	keys := sm.Keys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}

	vals := sm.Values()
	if len(vals) != 3 {
		t.Fatalf("expected 3 values, got %d", len(vals))
	}

	count := 0
	sm.Range(func(k int, v string) bool {
		count++
		return true
	})
	if count != 3 {
		t.Fatalf("expected Range to iterate 3 times, got %d", count)
	}

	sm.Clear()
	if sm.Len() != 0 {
		t.Fatalf("expected empty map after Clear, got len=%d", sm.Len())
	}
}

func TestSafeMap_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	sm := safemap.New[string, int]()
	var wg sync.WaitGroup

	numGoroutines := 50
	opsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				sm.Set(key, j)
				sm.Get(key)
				if j%2 == 0 {
					sm.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestSafeMap_Compute(t *testing.T) {
	t.Parallel()

	sm := safemap.New[string, int]()

	// 1. Insert when not present
	val := sm.Compute("apple", func(old int, exists bool) (int, bool) {
		if exists {
			t.Errorf("expected apple not to exist initially")
		}
		return 10, true
	})
	if val != 10 {
		t.Fatalf("expected 10, got %d", val)
	}
	if got, ok := sm.Get("apple"); !ok || got != 10 {
		t.Fatalf("expected map to contain apple=10, got %d, ok=%v", got, ok)
	}

	// 2. Update when present
	val = sm.Compute("apple", func(old int, exists bool) (int, bool) {
		if !exists || old != 10 {
			t.Errorf("expected apple to exist with value 10, got %d, ok=%v", old, exists)
		}
		return old + 5, true
	})
	if val != 15 {
		t.Fatalf("expected 15, got %d", val)
	}
	if got, ok := sm.Get("apple"); !ok || got != 15 {
		t.Fatalf("expected map to contain apple=15, got %d, ok=%v", got, ok)
	}

	// 3. Delete when present
	val = sm.Compute("apple", func(old int, exists bool) (int, bool) {
		return 0, false
	})
	if val != 0 {
		t.Fatalf("expected zero on delete, got %d", val)
	}
	if sm.Has("apple") {
		t.Fatalf("expected apple to be deleted")
	}

	// 4. No-op delete when not present
	val = sm.Compute("banana", func(old int, exists bool) (int, bool) {
		return 0, false
	})
	if val != 0 || sm.Has("banana") {
		t.Fatalf("expected banana not present after delete-noop")
	}

	// 5. Concurrent increments on same key
	const numRacers = 100
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	for i := 0; i < numRacers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startBarrier
			sm.Compute("counter", func(old int, exists bool) (int, bool) {
				return old + 1, true
			})
		}()
	}

	close(startBarrier)
	wg.Wait()

	finalVal, ok := sm.Get("counter")
	if !ok || finalVal != numRacers {
		t.Fatalf("expected counter to be %d, got %d (ok=%v)", numRacers, finalVal, ok)
	}
}
