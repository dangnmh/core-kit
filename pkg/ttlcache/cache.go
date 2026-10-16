package ttlcache

import (
	"sync"
	"time"
)

type cacheEntry[V any] struct {
	value      V
	expiration time.Time
}

func (e cacheEntry[V]) isExpired(now time.Time) bool {
	if e.expiration.IsZero() {
		return false
	}
	return now.After(e.expiration)
}

// TTLCache is a thread-safe generic in-memory cache with expiration.
type TTLCache[K comparable, V any] struct {
	mu         sync.RWMutex
	items      map[K]cacheEntry[V]
	defaultTTL time.Duration
	stopChan   chan struct{}
	closeOnce  sync.Once
	wg         sync.WaitGroup
}

// New creates a new initialized TTLCache.
// cleanupInterval controls how frequently expired entries are purged in the background.
func New[K comparable, V any](defaultTTL, cleanupInterval time.Duration) *TTLCache[K, V] {
	c := &TTLCache[K, V]{
		items:      make(map[K]cacheEntry[V]),
		defaultTTL: defaultTTL,
		stopChan:   make(chan struct{}),
	}

	if cleanupInterval > 0 {
		c.wg.Add(1)
		go c.cleanupLoop(cleanupInterval)
	}

	return c
}

func (c *TTLCache[K, V]) cleanupLoop(interval time.Duration) {
	defer c.wg.Done()
	ticker := time.NewTicker(interval)
	defer func() { ticker.Stop() }()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.PurgeExpired()
		}
	}
}

// Set stores a key-value pair with a specific TTL. If ttl <= 0, defaultTTL is used.
func (c *TTLCache[K, V]) Set(key K, val V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp time.Time
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}

	c.items[key] = cacheEntry[V]{
		value:      val,
		expiration: exp,
	}
}

// Get retrieves a key's value if it exists and has not expired.
func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	entry, found := c.items[key]
	if !found {
		c.mu.RUnlock()
		var zero V
		return zero, false
	}

	now := time.Now()
	if !entry.isExpired(now) {
		val := entry.value
		c.mu.RUnlock()
		return val, true
	}
	c.mu.RUnlock()

	// Lazily delete expired entry only if it is still expired under Lock (prevent deleting fresh Set)
	c.mu.Lock()
	if cur, ok := c.items[key]; ok && cur.isExpired(now) {
		delete(c.items, key)
	}
	c.mu.Unlock()

	var zero V
	return zero, false
}

// Delete removes an item by key.
func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Values returns a slice of all non-expired values in the cache.
func (c *TTLCache[K, V]) Values() []V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	res := make([]V, 0, len(c.items))
	for _, entry := range c.items {
		if !entry.isExpired(now) {
			res = append(res, entry.value)
		}
	}
	return res
}

// PurgeExpired actively cleans up expired entries from the cache.
func (c *TTLCache[K, V]) PurgeExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	purged := 0
	for k, entry := range c.items {
		if entry.isExpired(now) {
			delete(c.items, k)
			purged++
		}
	}
	return purged
}

// Len returns the current total number of entries in the cache (including unpurged expired entries).
func (c *TTLCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Close stops the background cleanup loop gracefully.
func (c *TTLCache[K, V]) Close() {
	c.closeOnce.Do(func() {
		close(c.stopChan)
		c.wg.Wait()
	})
}
