# priorityqueue

A concurrency-safe bounded priority queue for Go, using only the standard
library. Backend library plus a minimal CLI demo; no third-party
dependencies.

## Features

- Bounded: capacity is fixed at creation; Push fails when the queue is full.
- Ordered: elements dequeue by priority (higher first); equal priorities
  dequeue FIFO by original push order.
- Safe for concurrent use by multiple goroutines.
- Implemented with `container/heap` plus an ID index — writes are
  O(log n), never a full re-sort.

## API

```go
q := priorityqueue.New(100) // panics if capacity < 1

err := q.Push(priorityqueue.Element{ID: "a", Priority: 5, Value: "task"})
top, err := q.Peek()   // highest-priority element, not removed
e, err := q.Pop()      // highest-priority element, removed
err = q.Remove("a")    // delete by ID
n := q.Len()
cap := q.Capacity()
items := q.Snapshot()  // copy of current elements in dequeue order
```

Errors: `ErrEmptyID`, `ErrDuplicateID`, `ErrQueueFull`,
`ErrQueueEmpty`, `ErrNotFound` — check with `errors.Is`.

## Run the demo

```sh
go run ./cmd/demo
```

## Run the tests

```sh
go test ./...
go test -race ./...
```
