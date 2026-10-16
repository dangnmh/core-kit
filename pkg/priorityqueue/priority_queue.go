package priorityqueue

import (
	"container/heap"
	"sync"
)

// item wraps a generic value with its heap index and monotonic sequence ID.
type item[T any] struct {
	value T
	index int
	seq   uint64
}

// internalHeap implements heap.Interface for items of type T.
type internalHeap[T any] struct {
	items []*item[T]
	less  func(a, b T) bool
}

func (h *internalHeap[T]) Len() int { return len(h.items) }
func (h *internalHeap[T]) Less(i, j int) bool {
	a, b := h.items[i], h.items[j]
	if h.less(a.value, b.value) {
		return true
	}
	if h.less(b.value, a.value) {
		return false
	}
	return a.seq < b.seq
}
func (h *internalHeap[T]) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index = i
	h.items[j].index = j
}
func (h *internalHeap[T]) Push(x any) {
	it := x.(*item[T])
	it.index = len(h.items)
	h.items = append(h.items, it)
}
func (h *internalHeap[T]) Pop() any {
	old := h.items
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	it.index = -1
	h.items = old[:n-1]
	return it
}

// PriorityQueue is a thread-safe, generic priority queue backed by a heap with stable FIFO ordering for ties.
type PriorityQueue[T any] struct {
	mu   sync.RWMutex
	h    *internalHeap[T]
	next uint64
}

// New creates a new thread-safe PriorityQueue.
// The less function determines ordering: less(a, b) == true means 'a' has higher priority and dequeues first.
func New[T any](less func(a, b T) bool) *PriorityQueue[T] {
	if less == nil {
		panic("priorityqueue: less function cannot be nil")
	}
	h := &internalHeap[T]{
		items: make([]*item[T], 0),
		less:  less,
	}
	heap.Init(h)
	return &PriorityQueue[T]{
		h: h,
	}
}

// Enqueue adds an item to the priority queue in O(log N) time with stable FIFO sequence.
func (pq *PriorityQueue[T]) Enqueue(val T) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	heap.Push(pq.h, &item[T]{value: val, seq: pq.next})
	pq.next++
}

// Dequeue removes and returns the highest-priority item in O(log N) time.
// Returns (zero value, false) if the queue is empty.
func (pq *PriorityQueue[T]) Dequeue() (T, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.h.Len() == 0 {
		var zero T
		return zero, false
	}
	it := heap.Pop(pq.h).(*item[T])
	return it.value, true
}

// Peek returns the highest-priority item without removing it.
// Returns (zero value, false) if the queue is empty.
func (pq *PriorityQueue[T]) Peek() (T, bool) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	if pq.h.Len() == 0 {
		var zero T
		return zero, false
	}
	return pq.h.items[0].value, true
}

// Len returns the current number of items in the queue.
func (pq *PriorityQueue[T]) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.h.Len()
}

// Clear removes all elements from the queue.
func (pq *PriorityQueue[T]) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.h.items = make([]*item[T], 0)
	pq.next = 0
}

// Drain removes and returns all items currently in priority order.
func (pq *PriorityQueue[T]) Drain() []T {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	result := make([]T, 0, pq.h.Len())
	for pq.h.Len() > 0 {
		it := heap.Pop(pq.h).(*item[T])
		result = append(result, it.value)
	}
	return result
}
