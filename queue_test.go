package priorityqueue_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"priorityqueue"
)

func mustPush(t *testing.T, q *priorityqueue.Queue, id string, priority int) {
	t.Helper()
	if err := q.Push(priorityqueue.Element{ID: id, Priority: priority, Value: "v-" + id}); err != nil {
		t.Fatalf("Push(%q): unexpected error: %v", id, err)
	}
}

func mustPopID(t *testing.T, q *priorityqueue.Queue) string {
	t.Helper()
	e, err := q.Pop()
	if err != nil {
		t.Fatalf("Pop: unexpected error: %v", err)
	}
	return e.ID
}

func TestPushPopOrdersByPriorityDesc(t *testing.T) {
	q := priorityqueue.New(10)
	mustPush(t, q, "low", 1)
	mustPush(t, q, "high", 9)
	mustPush(t, q, "mid", 5)

	for _, want := range []string{"high", "mid", "low"} {
		if got := mustPopID(t, q); got != want {
			t.Fatalf("Pop = %q, want %q", got, want)
		}
	}
	if q.Len() != 0 {
		t.Fatalf("Len = %d, want 0", q.Len())
	}
}

func TestSamePriorityFIFO(t *testing.T) {
	q := priorityqueue.New(10)
	mustPush(t, q, "first", 5)
	mustPush(t, q, "second", 5)
	mustPush(t, q, "third", 5)

	for _, want := range []string{"first", "second", "third"} {
		if got := mustPopID(t, q); got != want {
			t.Fatalf("Pop = %q, want %q", got, want)
		}
	}
}

func TestPushRejectsEmptyID(t *testing.T) {
	q := priorityqueue.New(1)
	if err := q.Push(priorityqueue.Element{ID: "", Priority: 1}); !errors.Is(err, priorityqueue.ErrEmptyID) {
		t.Fatalf("Push with empty ID = %v, want ErrEmptyID", err)
	}
}

func TestPushRejectsDuplicateID(t *testing.T) {
	q := priorityqueue.New(2)
	mustPush(t, q, "a", 1)
	if err := q.Push(priorityqueue.Element{ID: "a", Priority: 9}); !errors.Is(err, priorityqueue.ErrDuplicateID) {
		t.Fatalf("Push duplicate = %v, want ErrDuplicateID", err)
	}
	if q.Len() != 1 {
		t.Fatalf("Len = %d, want 1", q.Len())
	}
}

func TestPushFailsWhenFull(t *testing.T) {
	q := priorityqueue.New(2)
	mustPush(t, q, "a", 1)
	mustPush(t, q, "b", 2)
	if err := q.Push(priorityqueue.Element{ID: "c", Priority: 3}); !errors.Is(err, priorityqueue.ErrQueueFull) {
		t.Fatalf("Push into full queue = %v, want ErrQueueFull", err)
	}
	if q.Len() != q.Capacity() {
		t.Fatalf("Len = %d, want %d", q.Len(), q.Capacity())
	}
	// After a Pop there is room again.
	mustPopID(t, q)
	mustPush(t, q, "c", 3)
}

func TestRemove(t *testing.T) {
	q := priorityqueue.New(3)
	mustPush(t, q, "a", 1)
	mustPush(t, q, "b", 2)
	mustPush(t, q, "c", 3)

	if err := q.Remove("b"); err != nil {
		t.Fatalf("Remove: unexpected error: %v", err)
	}
	if q.Len() != 2 {
		t.Fatalf("Len = %d, want 2", q.Len())
	}
	// The removed ID can be pushed again.
	mustPush(t, q, "b", 9)
	if got := mustPopID(t, q); got != "b" {
		t.Fatalf("Pop = %q, want %q", got, "b")
	}
}

func TestRemoveMissing(t *testing.T) {
	q := priorityqueue.New(1)
	if err := q.Remove("nope"); !errors.Is(err, priorityqueue.ErrNotFound) {
		t.Fatalf("Remove missing = %v, want ErrNotFound", err)
	}
}

func TestPeekDoesNotRemove(t *testing.T) {
	q := priorityqueue.New(2)
	if _, err := q.Peek(); !errors.Is(err, priorityqueue.ErrQueueEmpty) {
		t.Fatalf("Peek on empty = %v, want ErrQueueEmpty", err)
	}
	mustPush(t, q, "a", 1)
	mustPush(t, q, "b", 7)
	e, err := q.Peek()
	if err != nil || e.ID != "b" {
		t.Fatalf("Peek = %v, %v; want b, nil", e, err)
	}
	if q.Len() != 2 {
		t.Fatalf("Len after Peek = %d, want 2", q.Len())
	}
}

func TestPopEmpty(t *testing.T) {
	q := priorityqueue.New(1)
	if _, err := q.Pop(); !errors.Is(err, priorityqueue.ErrQueueEmpty) {
		t.Fatalf("Pop on empty = %v, want ErrQueueEmpty", err)
	}
}

func TestSnapshotOrderAndIsolation(t *testing.T) {
	q := priorityqueue.New(5)
	mustPush(t, q, "low", 1)
	mustPush(t, q, "mid-a", 5)
	mustPush(t, q, "high", 9)
	mustPush(t, q, "mid-b", 5)

	snap := q.Snapshot()
	want := []string{"high", "mid-a", "mid-b", "low"}
	if len(snap) != len(want) {
		t.Fatalf("Snapshot len = %d, want %d", len(snap), len(want))
	}
	for i, id := range want {
		if snap[i].ID != id {
			t.Fatalf("Snapshot[%d] = %q, want %q", i, snap[i].ID, id)
		}
	}

	// Mutating the returned slice must not affect the queue.
	snap[0].ID = "corrupted"
	snap[0].Priority = -100
	again := q.Snapshot()
	if again[0].ID != "high" || again[0].Priority != 9 {
		t.Fatalf("Snapshot leaked internal state: got %+v", again[0])
	}
	if q.Len() != 4 {
		t.Fatalf("Len = %d, want 4", q.Len())
	}
}

func TestConcurrentPushPop(t *testing.T) {
	const capacity = 32
	q := priorityqueue.New(capacity)
	var wg sync.WaitGroup

	// Producers: push unique IDs; a full queue is an acceptable outcome.
	for p := 0; p < 4; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				err := q.Push(priorityqueue.Element{
					ID:       fmt.Sprintf("p%d-%d", p, i),
					Priority: i,
				})
				if err != nil && !errors.Is(err, priorityqueue.ErrQueueFull) {
					t.Errorf("Push: unexpected error: %v", err)
				}
			}
		}(p)
	}

	// Consumers: pop whatever is available; an empty queue is acceptable.
	var mu sync.Mutex
	popped := make(map[string]struct{})
	for c := 0; c < 2; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				e, err := q.Pop()
				if errors.Is(err, priorityqueue.ErrQueueEmpty) {
					continue
				}
				if err != nil {
					t.Errorf("Pop: unexpected error: %v", err)
					continue
				}
				mu.Lock()
				if _, dup := popped[e.ID]; dup {
					t.Errorf("ID %q popped twice", e.ID)
				}
				popped[e.ID] = struct{}{}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if q.Len() > capacity {
		t.Fatalf("Len = %d exceeds capacity %d", q.Len(), capacity)
	}
	// No ID may be both popped and still queued, and the queue must not
	// contain duplicate IDs.
	seen := make(map[string]struct{}, len(popped))
	for id := range popped {
		seen[id] = struct{}{}
	}
	for _, e := range q.Snapshot() {
		if _, ok := seen[e.ID]; ok {
			t.Errorf("ID %q duplicated or both popped and queued", e.ID)
		}
		seen[e.ID] = struct{}{}
	}
}
