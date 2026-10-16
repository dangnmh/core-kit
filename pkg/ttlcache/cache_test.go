package ttlcache_test

import (
	"sync"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/ttlcache"
)

func TestTTLCache_BasicAndExpiration(t *testing.T) {
	t.Parallel()

	cache := ttlcache.New[string, int](50*time.Millisecond, 0)
	defer func() { cache.Close() }()

	cache.Set("short", 100, 20*time.Millisecond)
	cache.Set("long", 200, 200*time.Millisecond)

	val, ok := cache.Get("short")
	if !ok || val != 100 {
		t.Fatalf("expected short=100, got val=%d, ok=%v", val, ok)
	}

	val, ok = cache.Get("long")
	if !ok || val != 200 {
		t.Fatalf("expected long=200, got val=%d, ok=%v", val, ok)
	}

	time.Sleep(30 * time.Millisecond)

	_, ok = cache.Get("short")
	if ok {
		t.Fatal("expected short to have expired")
	}

	val, ok = cache.Get("long")
	if !ok || val != 200 {
		t.Fatalf("expected long to still be valid, got val=%d, ok=%v", val, ok)
	}
}

func TestTTLCache_ValuesAndPurge(t *testing.T) {
	t.Parallel()

	cache := ttlcache.New[string, string](100*time.Millisecond, 0)
	defer func() { cache.Close() }()

	cache.Set("a", "alpha", 10*time.Millisecond)
	cache.Set("b", "beta", 500*time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	values := cache.Values()
	if len(values) != 1 || values[0] != "beta" {
		t.Fatalf("expected [beta], got %v", values)
	}

	purged := cache.PurgeExpired()
	if purged != 1 {
		t.Fatalf("expected 1 purged, got %d", purged)
	}
}

func TestTTLCache_Concurrent(t *testing.T) {
	t.Parallel()

	cache := ttlcache.New[int, int](50*time.Millisecond, 10*time.Millisecond)
	defer func() { cache.Close() }()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				cache.Set(id*100+j, j, 30*time.Millisecond)
				cache.Get(id*100 + j)
			}
		}(i)
	}
	wg.Wait()
}

func TestTTLCache_GetDoesNotDeleteFreshSet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		give int
	}{
		{name: "interleaved set/get 10k", give: 10000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := ttlcache.New[string, int](time.Hour, 0)
			defer c.Close()

			stop := make(chan struct{})
			var readerWg sync.WaitGroup
			readerWg.Add(1)
			go func() {
				defer readerWg.Done()
				for {
					select {
					case <-stop:
						return
					default:
						c.Get("k")
					}
				}
			}()

			misses := 0
			for i := 0; i < tt.give; i++ {
				c.Set("k", i, time.Nanosecond)
				c.Set("k", i, time.Hour)
				if _, ok := c.Get("k"); !ok {
					misses++
				}
			}
			close(stop)
			readerWg.Wait()

			if misses > 0 {
				t.Fatalf("expected 0 misses after Set with 1h TTL, got %d misses", misses)
			}
		})
	}
}

func TestTTLCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	c := ttlcache.New[int, int](50*time.Millisecond, 10*time.Millisecond)
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				switch j % 3 {
				case 0:
					c.Set(j, id, 10*time.Millisecond)
				case 1:
					c.Get(j)
				case 2:
					c.Delete(j)
				}
			}
		}(i)
	}
	wg.Wait()
}
