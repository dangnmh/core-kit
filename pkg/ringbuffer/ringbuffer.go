package ringbuffer

import "sync"

// RingBuffer implements a thread-safe, generic circular fixed-size buffer.
// It achieves O(1) push operations and zero dynamic re-allocations after initialization.
type RingBuffer[T any] struct {
	mu         sync.RWMutex
	buffer     []T
	head       int   // Index where the next element will be written
	size       int   // Current number of elements stored (0 <= size <= capacity)
	capacity   int   // Maximum capacity of the ring buffer
	seqCounter int64 // Monotonically increasing sequence counter
}

// New creates a new RingBuffer with the specified capacity.
func New[T any](capacity int) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = 100
	}
	return &RingBuffer[T]{
		buffer:     make([]T, capacity),
		capacity:   capacity,
		head:       0,
		size:       0,
		seqCounter: 0,
	}
}

// Push appends an item to the buffer. If the buffer is full, it overwrites
// the oldest item in O(1) time. Automatically increments and returns the monotonic sequence ID.
func (r *RingBuffer[T]) Push(item T) int64 {
	return r.PushWithAssign(item, nil)
}

// PushWithAssign appends an item to the buffer, invoking assignSeq with the newly assigned sequence counter
// while holding the write lock, ensuring thread-safe assignment before the item becomes visible to readers.
// Note: assignSeq runs under the write lock; it MUST NOT call any RingBuffer method.
func (r *RingBuffer[T]) PushWithAssign(item T, assignSeq func(item T, seq int64)) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seqCounter++
	if assignSeq != nil {
		assignSeq(item, r.seqCounter)
	}
	r.buffer[r.head] = item
	r.head = (r.head + 1) % r.capacity
	if r.size < r.capacity {
		r.size++
	}

	return r.seqCounter
}

// LatestSequenceID returns the highest monotonic sequence ID assigned so far.
func (r *RingBuffer[T]) LatestSequenceID() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.seqCounter
}

// Len returns the current number of items stored in the buffer.
func (r *RingBuffer[T]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.size
}

// Capacity returns the maximum capacity of the ring buffer.
func (r *RingBuffer[T]) Capacity() int {
	return r.capacity
}

// Cap returns the maximum capacity of the ring buffer (CONTRACT-GO-API-01).
func (r *RingBuffer[T]) Cap() int {
	return r.Capacity()
}

// Latest returns the most recently pushed item and true, or the zero value and false if empty.
func (r *RingBuffer[T]) Latest() (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 {
		var zero T
		return zero, false
	}
	idx := (r.head - 1 + r.capacity) % r.capacity
	return r.buffer[idx], true
}

// Since returns all items pushed strictly after sequence number seq in chronological order.
func (r *RingBuffer[T]) Since(seq int64) []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 || seq >= r.seqCounter {
		return []T{}
	}

	diff := r.seqCounter - seq
	if diff > int64(r.size) {
		diff = int64(r.size)
	}

	n := int(diff)
	res := make([]T, n)
	startLogical := (r.head - n + r.capacity) % r.capacity
	for i := 0; i < n; i++ {
		idx := (startLogical + i) % r.capacity
		res[i] = r.buffer[idx]
	}
	return res
}

// GetAll returns a snapshot of all items currently stored in chronological
// order (from oldest to newest).
func (r *RingBuffer[T]) GetAll() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 {
		return []T{}
	}

	out := make([]T, r.size)
	if r.size < r.capacity {
		copy(out, r.buffer[:r.size])
		return out
	}

	// Buffer is full (size == capacity):
	// Oldest element is at r.head, newest element is at (r.head - 1 + capacity) % capacity
	n1 := r.capacity - r.head
	copy(out, r.buffer[r.head:r.capacity])
	copy(out[n1:], r.buffer[:r.head])
	return out
}

// GetRecent returns up to n most recent items in chronological order.
func (r *RingBuffer[T]) GetRecent(n int) []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n <= 0 || r.size == 0 {
		return []T{}
	}

	if n > r.size {
		n = r.size
	}

	res := make([]T, n)
	// Calculate the starting logical index in the buffer
	// Start offset from the oldest element is (r.size - n)
	startLogical := (r.head - n + r.capacity) % r.capacity

	for i := 0; i < n; i++ {
		idx := (startLogical + i) % r.capacity
		res[i] = r.buffer[idx]
	}

	return res
}

// Filter returns a slice of all items currently in the buffer that satisfy predicate fn.
func (r *RingBuffer[T]) Filter(fn func(T) bool) []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []T
	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + r.capacity) % r.capacity
		item := r.buffer[idx]
		if fn(item) {
			matched = append(matched, item)
		}
	}
	return matched
}

// ContainsFunc returns true if any item in the buffer satisfies predicate fn.
func (r *RingBuffer[T]) ContainsFunc(fn func(T) bool) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + r.capacity) % r.capacity
		if fn(r.buffer[idx]) {
			return true
		}
	}
	return false
}

// ForEach executes fn for each item currently stored in chronological order.
// To prevent deadlocks on re-entrant RingBuffer calls, it takes a snapshot under read lock
// and iterates over the snapshot outside the lock.
// If fn returns false, iteration stops.
func (r *RingBuffer[T]) ForEach(fn func(T) bool) {
	if fn == nil {
		return
	}
	items := r.GetAll()
	for _, item := range items {
		if !fn(item) {
			break
		}
	}
}

// Clear resets the ring buffer to empty state.
func (r *RingBuffer[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	var zero T
	for i := range r.buffer {
		r.buffer[i] = zero
	}
	r.head = 0
	r.size = 0
}

// GetRecentWindow retrieves items from the buffer going backwards in time
// until an item falls outside the specified duration window.
// It avoids allocating the entire array by only collecting the matching items.
func (r *RingBuffer[T]) GetRecentWindow(durationSeconds float64, getTimestamp func(T) float64) []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 || durationSeconds <= 0 {
		return []T{}
	}

	// The newest element is at (r.head - 1 + r.capacity) % r.capacity
	newestIdx := (r.head - 1 + r.capacity) % r.capacity
	newestItem := r.buffer[newestIdx]
	cutoff := getTimestamp(newestItem) - durationSeconds

	// Count how many items fall within the window (going backwards)
	count := 0
	for i := 0; i < r.size; i++ {
		idx := (r.head - 1 - i + r.capacity) % r.capacity
		item := r.buffer[idx]
		if getTimestamp(item) < cutoff {
			break
		}
		count++
	}

	if count == 0 {
		return []T{}
	}

	// Allocate exact size and fill chronologically (oldest first)
	res := make([]T, count)
	for i := 0; i < count; i++ {
		// Start from oldest in the window, which is count-1 steps back from newest
		idx := (r.head - count + i + r.capacity) % r.capacity
		res[i] = r.buffer[idx]
	}

	return res
}
