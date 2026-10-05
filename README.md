# priorityqueue

A concurrent, bounded priority queue for Go. Standard library only
(`container/heap` + ID index); no third-party dependencies.

## Features

- Bounded capacity set at construction; `Push` fails with `ErrFull` when full.
- Unique, non-empty string IDs; duplicates fail with `ErrDuplicateID`.
- Dequeue order: higher `Priority` first; equal priorities are FIFO by
  original successful `Push`.
- `Push`, `Pop`, `Remove` are O(log n); `Peek`, `Len`, `Capacity` are O(1).
  No full re-sort on writes.
- Safe for concurrent use from multiple goroutines.
- `Snapshot` returns a copy of the current items in dequeue order; callers
  cannot corrupt internal state.

## API

```go
q, err := priorityqueue.New(100)          // capacity must be > 0
err = q.Push(priorityqueue.Item{ID: "a", Priority: 5, Value: "job"})
item, ok := q.Peek()                      // highest priority, not removed
item, ok = q.Pop()                        // highest priority, removed
removed := q.Remove("a")                  // delete by ID
n := q.Len()                              // current size
cap := q.Capacity()                       // max size
items := q.Snapshot()                     // []Item in dequeue order (copy)
```

Errors: `ErrInvalidCapacity`, `ErrEmptyID`, `ErrDuplicateID`, `ErrFull`
(use `errors.Is` to check).

## Run the demo

```sh
go run ./cmd/demo
```

## Run the tests

```sh
go test ./...
go test -race ./...
```
