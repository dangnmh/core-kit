package safemap

import "sync"

// SafeMap provides a thread-safe generic map for concurrent access.
// K must be comparable, and V can be any type.
type SafeMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// New creates a new initialized SafeMap.
func New[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		m: make(map[K]V),
	}
}

// Get retrieves the value associated with key, returning (value, true) if found,
// or the zero value of V and false if not found.
func (s *SafeMap[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.m[key]
	return val, ok
}

// Set stores the key-value pair into the map.
func (s *SafeMap[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

// Delete removes the key and its value from the map.
func (s *SafeMap[K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

// Has returns true if the key exists in the map.
func (s *SafeMap[K, V]) Has(key K) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.m[key]
	return ok
}

// Len returns the current number of elements in the map.
func (s *SafeMap[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// Keys returns a slice of all keys currently in the map.
func (s *SafeMap[K, V]) Keys() []K {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]K, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a slice of all values currently in the map.
func (s *SafeMap[K, V]) Values() []V {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vals := make([]V, 0, len(s.m))
	for _, v := range s.m {
		vals = append(vals, v)
	}
	return vals
}

// Clear removes all entries from the map.
func (s *SafeMap[K, V]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m = make(map[K]V)
}

// Range calls fn sequentially for each key and value present in the map.
// If fn returns false, range stops the iteration.
func (s *SafeMap[K, V]) Range(fn func(key K, value V) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.m {
		if !fn(k, v) {
			break
		}
	}
}

// GetOrSet returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (s *SafeMap[K, V]) GetOrSet(key K, value V) (actual V, loaded bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if val, ok := s.m[key]; ok {
		return val, true
	}
	s.m[key] = value
	return value, false
}

// Compute executes fn under a write lock, allowing atomic inspection and mutation of an entry.
// fn receives the existing value (or zero value of V) and whether the key was present in the map.
// fn returns the new value to store and a boolean 'keep'.
// If keep is true, the entry is set to newValue. If keep is false, the entry is deleted.
// Compute returns the resulting value (newValue if keep is true, or the zero value of V if keep is false).
//
// WARNING: fn is executed while holding the SafeMap's write lock. fn must NEVER call methods on
// this SafeMap instance, otherwise a deadlock will occur.
func (s *SafeMap[K, V]) Compute(key K, fn func(old V, exists bool) (V, bool)) V {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, exists := s.m[key]
	newVal, keep := fn(old, exists)
	if !keep {
		delete(s.m, key)
		var zero V
		return zero
	}
	s.m[key] = newVal
	return newVal
}
