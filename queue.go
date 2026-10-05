// Package priorityqueue provides a concurrency-safe bounded priority queue.
//
// Elements are dequeued by priority (higher first); elements with equal
// priority are dequeued in the order they were successfully pushed (FIFO).
// The queue is implemented with container/heap plus an ID index, so writes
// are O(log n) and never re-sort the whole queue.
package priorityqueue

import (
	"container/heap"
	"errors"
	"sort"
	"sync"
)

var (
	// ErrEmptyID is returned by Push when the element ID is empty.
	ErrEmptyID = errors.New("priorityqueue: element ID must not be empty")
	// ErrDuplicateID is returned by Push when the ID already exists.
	ErrDuplicateID = errors.New("priorityqueue: element ID already exists")
	// ErrQueueFull is returned by Push when the queue is at capacity.
	ErrQueueFull = errors.New("priorityqueue: queue is full")
	// ErrQueueEmpty is returned by Peek and Pop on an empty queue.
	ErrQueueEmpty = errors.New("priorityqueue: queue is empty")
	// ErrNotFound is returned by Remove when the ID does not exist.
	ErrNotFound = errors.New("priorityqueue: element not found")
)

// Element is a single item stored in the queue.
type Element struct {
	ID       string
	Priority int
	Value    string
}

// item is the internal heap node. seq preserves push order for FIFO
// tie-breaking, and index tracks the node's position in the heap array.
type item struct {
	elem  Element
	seq   uint64
	index int
}

// itemHeap implements heap.Interface: highest priority first, then
// lowest seq (earliest push) first.
type itemHeap []*item

func (h itemHeap) Len() int { return len(h) }

func (h itemHeap) Less(i, j int) bool {
	if h[i].elem.Priority != h[j].elem.Priority {
		return h[i].elem.Priority > h[j].elem.Priority
	}
	return h[i].seq < h[j].seq
}

func (h itemHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *itemHeap) Push(x any) {
	it := x.(*item)
	it.index = len(*h)
	*h = append(*h, it)
}

func (h *itemHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	it.index = -1
	*h = old[:n-1]
	return it
}

// Queue is a concurrency-safe bounded priority queue. The zero value is
// not usable; create one with New.
type Queue struct {
	mu    sync.Mutex
	cap   int
	seq   uint64
	h     itemHeap
	index map[string]*item
}

// New returns a Queue that holds at most capacity elements.
// It panics if capacity is less than 1.
func New(capacity int) *Queue {
	if capacity < 1 {
		panic("priorityqueue: capacity must be at least 1")
	}
	return &Queue{
		cap:   capacity,
		index: make(map[string]*item),
	}
}

// Push adds e to the queue. It returns ErrEmptyID for an empty ID,
// ErrDuplicateID if the ID already exists, or ErrQueueFull if the queue
// is at capacity.
func (q *Queue) Push(e Element) error {
	if e.ID == "" {
		return ErrEmptyID
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.index[e.ID]; ok {
		return ErrDuplicateID
	}
	if len(q.h) >= q.cap {
		return ErrQueueFull
	}
	it := &item{elem: e, seq: q.seq}
	q.seq++
	heap.Push(&q.h, it)
	q.index[e.ID] = it
	return nil
}

// Remove deletes the element with the given ID, or returns ErrNotFound.
func (q *Queue) Remove(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	it, ok := q.index[id]
	if !ok {
		return ErrNotFound
	}
	heap.Remove(&q.h, it.index)
	delete(q.index, id)
	return nil
}

// Peek returns the next element to be dequeued without removing it,
// or ErrQueueEmpty if the queue is empty.
func (q *Queue) Peek() (Element, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.h) == 0 {
		return Element{}, ErrQueueEmpty
	}
	return q.h[0].elem, nil
}

// Pop removes and returns the next element to be dequeued,
// or ErrQueueEmpty if the queue is empty.
func (q *Queue) Pop() (Element, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.h) == 0 {
		return Element{}, ErrQueueEmpty
	}
	it := heap.Pop(&q.h).(*item)
	delete(q.index, it.elem.ID)
	return it.elem, nil
}

// Len returns the number of elements currently in the queue.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.h)
}

// Capacity returns the maximum number of elements the queue can hold.
func (q *Queue) Capacity() int {
	return q.cap // immutable after New, no lock needed
}

// Snapshot returns the current elements in dequeue order. The result is
// a copy; mutating it does not affect the queue.
func (q *Queue) Snapshot() []Element {
	q.mu.Lock()
	defer q.mu.Unlock()
	items := make([]*item, len(q.h))
	copy(items, q.h)
	sort.Slice(items, func(i, j int) bool {
		if items[i].elem.Priority != items[j].elem.Priority {
			return items[i].elem.Priority > items[j].elem.Priority
		}
		return items[i].seq < items[j].seq
	})
	out := make([]Element, len(items))
	for i, it := range items {
		out[i] = it.elem
	}
	return out
}
