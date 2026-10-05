// Command demo is a minimal demonstration of the priorityqueue package.
package main

import (
	"fmt"
	"log"

	"priorityqueue"
)

func main() {
	q, err := priorityqueue.New(4)
	if err != nil {
		log.Fatal(err)
	}

	items := []priorityqueue.Item{
		{ID: "task-low", Priority: 1, Value: "clean up logs"},
		{ID: "task-high", Priority: 9, Value: "fix production outage"},
		{ID: "task-mid", Priority: 5, Value: "review pull request"},
		{ID: "task-mid2", Priority: 5, Value: "update dependencies"},
	}
	for _, item := range items {
		if err := q.Push(item); err != nil {
			log.Fatalf("push %q: %v", item.ID, err)
		}
	}
	fmt.Printf("pushed %d items (capacity %d)\n", q.Len(), q.Capacity())

	if item, ok := q.Peek(); ok {
		fmt.Printf("peek:      %s (priority %d) -> %q\n", item.ID, item.Priority, item.Value)
	}

	if item, ok := q.Pop(); ok {
		fmt.Printf("pop:       %s (priority %d) -> %q\n", item.ID, item.Priority, item.Value)
	}

	if q.Remove("task-low") {
		fmt.Println("removed:   task-low")
	}

	fmt.Println("snapshot (dequeue order):")
	for _, item := range q.Snapshot() {
		fmt.Printf("  %s (priority %d) -> %q\n", item.ID, item.Priority, item.Value)
	}
}
