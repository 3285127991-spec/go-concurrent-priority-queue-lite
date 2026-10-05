// Package priorityqueue provides a concurrent, bounded priority queue.
//
// Elements are dequeued by priority (higher first); elements with equal
// priority are dequeued in FIFO order of their original successful Push.
// All methods are safe for concurrent use by multiple goroutines.
package priorityqueue

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrInvalidCapacity is returned by New when capacity is not positive.
	ErrInvalidCapacity = errors.New("priorityqueue: capacity must be positive")
	// ErrEmptyID is returned by Push when the item ID is empty.
	ErrEmptyID = errors.New("priorityqueue: item ID must not be empty")
	// ErrDuplicateID is returned by Push when the ID already exists in the queue.
	ErrDuplicateID = errors.New("priorityqueue: duplicate item ID")
	// ErrFull is returned by Push when the queue has reached its capacity.
	ErrFull = errors.New("priorityqueue: queue is full")
)

// Item is an element stored in the queue.
type Item struct {
	ID       string
	Priority int
	Value    string
}

// node is the internal heap entry. seq orders equal-priority items by
// insertion (FIFO); index tracks the position for O(log n) removal.
type node struct {
	item  Item
	seq   uint64
	index int
}

// nodeHeap implements heap.Interface as a max-heap on priority, with
// insertion sequence as the FIFO tie-breaker.
type nodeHeap []*node

func (h nodeHeap) Len() int { return len(h) }

func (h nodeHeap) Less(i, j int) bool {
	if h[i].item.Priority != h[j].item.Priority {
		return h[i].item.Priority > h[j].item.Priority
	}
	return h[i].seq < h[j].seq
}

func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *nodeHeap) Push(x any) {
	n := x.(*node)
	n.index = len(*h)
	*h = append(*h, n)
}

func (h *nodeHeap) Pop() any {
	old := *h
	n := old[len(old)-1]
	old[len(old)-1] = nil
	n.index = -1
	*h = old[:len(old)-1]
	return n
}

// Queue is a concurrent, bounded priority queue. The zero value is not
// usable; create one with New.
type Queue struct {
	mu   sync.Mutex
	h    nodeHeap
	byID map[string]*node
	cap  int
	seq  uint64
}

// New creates a Queue that holds at most capacity items.
func New(capacity int) (*Queue, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &Queue{byID: make(map[string]*node), cap: capacity}, nil
}

// Push adds item to the queue. It fails with ErrEmptyID, ErrDuplicateID,
// or ErrFull, leaving the queue unchanged.
func (q *Queue) Push(item Item) error {
	if item.ID == "" {
		return ErrEmptyID
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.byID[item.ID]; ok {
		return fmt.Errorf("%w: %q", ErrDuplicateID, item.ID)
	}
	if len(q.h) >= q.cap {
		return ErrFull
	}
	n := &node{item: item, seq: q.seq}
	q.seq++
	q.byID[item.ID] = n
	heap.Push(&q.h, n)
	return nil
}

// Pop removes and returns the highest-priority item. The second return
// value is false when the queue is empty.
func (q *Queue) Pop() (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.h) == 0 {
		return Item{}, false
	}
	n := heap.Pop(&q.h).(*node)
	delete(q.byID, n.item.ID)
	return n.item, true
}

// Peek returns the highest-priority item without removing it. The second
// return value is false when the queue is empty.
func (q *Queue) Peek() (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.h) == 0 {
		return Item{}, false
	}
	return q.h[0].item, true
}

// Remove deletes the item with the given ID. It reports whether such an
// item was present.
func (q *Queue) Remove(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	n, ok := q.byID[id]
	if !ok {
		return false
	}
	heap.Remove(&q.h, n.index)
	delete(q.byID, id)
	return true
}

// Len returns the number of items currently in the queue.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.h)
}

// Capacity returns the maximum number of items the queue can hold.
// It is fixed at construction and never changes.
func (q *Queue) Capacity() int {
	return q.cap
}

// Snapshot returns the current items in dequeue order (priority
// descending, ties FIFO). The result is a copy; mutating it does not
// affect the queue.
func (q *Queue) Snapshot() []Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	sorted := make(nodeHeap, len(q.h))
	copy(sorted, q.h)
	sortNodes(sorted)
	items := make([]Item, len(sorted))
	for i, n := range sorted {
		items[i] = n.item
	}
	return items
}

func sortNodes(h nodeHeap) {
	// Insertion sort on the heap comparator; queue sizes are bounded and
	// this runs on a private copy, keeping Snapshot simple and exact.
	// Swaps must not touch node.index, which tracks live heap positions.
	for i := 1; i < len(h); i++ {
		for j := i; j > 0 && h.Less(j, j-1); j-- {
			h[j], h[j-1] = h[j-1], h[j]
		}
	}
}
