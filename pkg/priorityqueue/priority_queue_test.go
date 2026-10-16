package priorityqueue_test

import (
	"sync"
	"testing"

	"github.com/dangnmh/core-kit/pkg/priorityqueue"
)

type TestTask struct {
	Name     string
	Priority int // Lower number = higher priority
}

func TestPriorityQueue_Order(t *testing.T) {
	t.Parallel()

	pq := priorityqueue.New(func(a, b TestTask) bool {
		return a.Priority < b.Priority
	})

	pq.Enqueue(TestTask{Name: "medium", Priority: 2})
	pq.Enqueue(TestTask{Name: "lowest", Priority: 3})
	pq.Enqueue(TestTask{Name: "urgent", Priority: 0})
	pq.Enqueue(TestTask{Name: "high", Priority: 1})

	if pq.Len() != 4 {
		t.Fatalf("expected len 4, got %d", pq.Len())
	}

	peek, ok := pq.Peek()
	if !ok || peek.Name != "urgent" {
		t.Fatalf("expected peek urgent, got %+v", peek)
	}

	item, ok := pq.Dequeue()
	if !ok || item.Name != "urgent" {
		t.Fatalf("expected urgent, got %+v", item)
	}

	item, ok = pq.Dequeue()
	if !ok || item.Name != "high" {
		t.Fatalf("expected high, got %+v", item)
	}

	item, ok = pq.Dequeue()
	if !ok || item.Name != "medium" {
		t.Fatalf("expected medium, got %+v", item)
	}

	item, ok = pq.Dequeue()
	if !ok || item.Name != "lowest" {
		t.Fatalf("expected lowest, got %+v", item)
	}

	_, ok = pq.Dequeue()
	if ok {
		t.Fatal("expected queue to be empty")
	}
}

func TestPriorityQueue_Drain(t *testing.T) {
	t.Parallel()

	pq := priorityqueue.New(func(a, b int) bool {
		return a < b // min-heap
	})

	pq.Enqueue(50)
	pq.Enqueue(10)
	pq.Enqueue(30)

	items := pq.Drain()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0] != 10 || items[1] != 30 || items[2] != 50 {
		t.Fatalf("expected [10, 30, 50], got %v", items)
	}
	if pq.Len() != 0 {
		t.Fatalf("expected empty queue after drain, got len %d", pq.Len())
	}
}

func TestPriorityQueue_Concurrent(t *testing.T) {
	t.Parallel()

	pq := priorityqueue.New(func(a, b int) bool {
		return a < b
	})

	var wg sync.WaitGroup
	workers := 20
	itemsPerWorker := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < itemsPerWorker; j++ {
				pq.Enqueue(workerID*1000 + j)
				pq.Peek()
			}
		}(i)
	}

	wg.Wait()
	if pq.Len() != workers*itemsPerWorker {
		t.Fatalf("expected len %d, got %d", workers*itemsPerWorker, pq.Len())
	}
}

func TestPriorityQueue_StableFIFO(t *testing.T) {
	t.Parallel()

	type queueItem struct {
		id       int
		priority int
	}

	tests := []struct {
		name string
		give []int // priority of items inserted in order 0, 1, 2...
		want []int // original insertion index in dequeue order
	}{
		{
			name: "all equal priority",
			give: []int{1, 1, 1, 1},
			want: []int{0, 1, 2, 3},
		},
		{
			name: "mixed priority with ties",
			give: []int{2, 1, 2, 1},
			want: []int{1, 3, 0, 2},
		},
		{
			name: "empty",
			give: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pq := priorityqueue.New(func(a, b queueItem) bool {
				return a.priority < b.priority // lower priority value dequeues first
			})

			for idx, p := range tt.give {
				pq.Enqueue(queueItem{id: idx, priority: p})
			}

			var got []int
			for {
				item, ok := pq.Dequeue()
				if !ok {
					break
				}
				got = append(got, item.id)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("index %d: expected %d, got %d (full: %v)", i, tt.want[i], got[i], got)
				}
			}
		})
	}
}

func TestPriorityQueue_NilLessPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic when less is nil, got none")
		}
	}()

	_ = priorityqueue.New[int](nil)
}
