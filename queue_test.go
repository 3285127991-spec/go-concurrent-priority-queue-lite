package priorityqueue

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func mustNew(t *testing.T, capacity int) *Queue {
	t.Helper()
	q, err := New(capacity)
	if err != nil {
		t.Fatalf("New(%d): %v", capacity, err)
	}
	return q
}

func mustPush(t *testing.T, q *Queue, item Item) {
	t.Helper()
	if err := q.Push(item); err != nil {
		t.Fatalf("Push(%+v): %v", item, err)
	}
}

func popIDs(q *Queue) []string {
	var ids []string
	for {
		item, ok := q.Pop()
		if !ok {
			return ids
		}
		ids = append(ids, item.ID)
	}
}

func TestNewRejectsNonPositiveCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		if _, err := New(capacity); !errors.Is(err, ErrInvalidCapacity) {
			t.Errorf("New(%d) err = %v, want ErrInvalidCapacity", capacity, err)
		}
	}
}

func TestPushPopBasic(t *testing.T) {
	q := mustNew(t, 3)
	mustPush(t, q, Item{ID: "a", Priority: 1, Value: "hello"})
	if q.Len() != 1 || q.Capacity() != 3 {
		t.Fatalf("Len=%d Capacity=%d, want 1/3", q.Len(), q.Capacity())
	}
	item, ok := q.Pop()
	if !ok || item.ID != "a" || item.Value != "hello" {
		t.Fatalf("Pop = %+v, %v", item, ok)
	}
	if _, ok := q.Pop(); ok {
		t.Fatal("Pop on empty queue should report ok=false")
	}
}

func TestPopOrderByPriority(t *testing.T) {
	q := mustNew(t, 5)
	mustPush(t, q, Item{ID: "low", Priority: 1})
	mustPush(t, q, Item{ID: "high", Priority: 9})
	mustPush(t, q, Item{ID: "mid", Priority: 5})
	want := []string{"high", "mid", "low"}
	if got := popIDs(q); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestSamePriorityFIFO(t *testing.T) {
	q := mustNew(t, 4)
	mustPush(t, q, Item{ID: "first", Priority: 7})
	mustPush(t, q, Item{ID: "second", Priority: 7})
	mustPush(t, q, Item{ID: "third", Priority: 7})
	want := []string{"first", "second", "third"}
	if got := popIDs(q); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestPushRejectsEmptyAndDuplicateID(t *testing.T) {
	q := mustNew(t, 2)
	if err := q.Push(Item{Priority: 1}); !errors.Is(err, ErrEmptyID) {
		t.Fatalf("empty ID err = %v, want ErrEmptyID", err)
	}
	mustPush(t, q, Item{ID: "a", Priority: 1})
	if err := q.Push(Item{ID: "a", Priority: 2}); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("duplicate ID err = %v, want ErrDuplicateID", err)
	}
	if q.Len() != 1 {
		t.Fatalf("Len = %d, want 1", q.Len())
	}
	// After removing the ID, pushing it again must succeed.
	if !q.Remove("a") {
		t.Fatal("Remove(a) = false, want true")
	}
	mustPush(t, q, Item{ID: "a", Priority: 2})
}

func TestPushFullQueue(t *testing.T) {
	q := mustNew(t, 2)
	mustPush(t, q, Item{ID: "a", Priority: 1})
	mustPush(t, q, Item{ID: "b", Priority: 2})
	if err := q.Push(Item{ID: "c", Priority: 3}); !errors.Is(err, ErrFull) {
		t.Fatalf("push on full queue err = %v, want ErrFull", err)
	}
	if q.Len() != 2 {
		t.Fatalf("Len = %d, want 2", q.Len())
	}
	// Capacity is enforced, not grown.
	if _, ok := q.Pop(); !ok {
		t.Fatal("Pop failed on non-empty queue")
	}
	mustPush(t, q, Item{ID: "c", Priority: 3})
}

func TestPeek(t *testing.T) {
	q := mustNew(t, 2)
	if _, ok := q.Peek(); ok {
		t.Fatal("Peek on empty queue should report ok=false")
	}
	mustPush(t, q, Item{ID: "low", Priority: 1})
	mustPush(t, q, Item{ID: "high", Priority: 5})
	item, ok := q.Peek()
	if !ok || item.ID != "high" {
		t.Fatalf("Peek = %+v, %v", item, ok)
	}
	if q.Len() != 2 {
		t.Fatalf("Peek must not remove; Len = %d, want 2", q.Len())
	}
}

func TestRemove(t *testing.T) {
	q := mustNew(t, 3)
	mustPush(t, q, Item{ID: "a", Priority: 1})
	mustPush(t, q, Item{ID: "b", Priority: 2})
	mustPush(t, q, Item{ID: "c", Priority: 3})
	if q.Remove("missing") {
		t.Fatal("Remove(missing) = true, want false")
	}
	if !q.Remove("b") {
		t.Fatal("Remove(b) = false, want true")
	}
	if q.Len() != 2 {
		t.Fatalf("Len = %d, want 2", q.Len())
	}
	want := []string{"c", "a"}
	if got := popIDs(q); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestSnapshotOrderAndIsolation(t *testing.T) {
	q := mustNew(t, 4)
	mustPush(t, q, Item{ID: "a", Priority: 1})
	mustPush(t, q, Item{ID: "b", Priority: 9})
	mustPush(t, q, Item{ID: "c", Priority: 9})
	snap := q.Snapshot()
	want := []string{"b", "c", "a"}
	got := make([]string, len(snap))
	for i, item := range snap {
		got[i] = item.ID
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("Snapshot order = %v, want %v", got, want)
	}
	// Mutating the snapshot must not affect the queue.
	snap[0].ID = "corrupted"
	snap[1] = Item{}
	if item, _ := q.Peek(); item.ID != "b" {
		t.Fatalf("queue corrupted via snapshot: Peek = %q", item.ID)
	}
	if q.Len() != 3 {
		t.Fatalf("Len = %d, want 3", q.Len())
	}
}

func TestConcurrentPushPop(t *testing.T) {
	const (
		capacity = 32
		workers  = 8
		perWork  = 16
	)
	q := mustNew(t, capacity)

	// Concurrent pushes: exactly capacity items succeed, the rest get
	// ErrFull; no duplicate IDs are ever generated.
	var wg sync.WaitGroup
	errs := make(chan error, workers*perWork)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWork; i++ {
				id := fmt.Sprintf("w%d-i%d", w, i)
				if err := q.Push(Item{ID: id, Priority: i}); err != nil {
					errs <- err
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if !errors.Is(err, ErrFull) {
			t.Fatalf("unexpected push error: %v", err)
		}
	}
	if q.Len() != capacity {
		t.Fatalf("Len = %d, want %d (capacity)", q.Len(), capacity)
	}

	// Concurrent pops: every item is handed out exactly once.
	seen := make(chan string, capacity)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				item, ok := q.Pop()
				if !ok {
					return
				}
				seen <- item.ID
			}
		}()
	}
	wg.Wait()
	close(seen)
	unique := make(map[string]bool)
	for id := range seen {
		if unique[id] {
			t.Fatalf("ID %q popped twice", id)
		}
		unique[id] = true
	}
	if len(unique) != capacity {
		t.Fatalf("popped %d unique items, want %d", len(unique), capacity)
	}
	if q.Len() != 0 {
		t.Fatalf("Len = %d, want 0", q.Len())
	}
}
