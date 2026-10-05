// Command demo is a minimal demonstration of the priorityqueue package.
package main

import (
	"fmt"

	"priorityqueue"
)

func main() {
	q := priorityqueue.New(4)

	for _, e := range []priorityqueue.Element{
		{ID: "low", Priority: 1, Value: "low priority task"},
		{ID: "high", Priority: 9, Value: "urgent task"},
		{ID: "mid-a", Priority: 5, Value: "first medium task"},
		{ID: "mid-b", Priority: 5, Value: "second medium task"},
	} {
		if err := q.Push(e); err != nil {
			fmt.Println("push failed:", err)
		}
	}
	fmt.Printf("len=%d capacity=%d\n", q.Len(), q.Capacity())

	if top, err := q.Peek(); err == nil {
		fmt.Printf("peek: %s (priority %d)\n", top.ID, top.Priority)
	}

	if e, err := q.Pop(); err == nil {
		fmt.Printf("pop: %s (priority %d)\n", e.ID, e.Priority)
	}

	if err := q.Remove("mid-a"); err == nil {
		fmt.Println("removed: mid-a")
	}

	fmt.Println("snapshot:")
	for _, e := range q.Snapshot() {
		fmt.Printf("  %s (priority %d): %s\n", e.ID, e.Priority, e.Value)
	}
}
