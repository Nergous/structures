package pqueue_test

import (
	"cmp"
	"fmt"
	"time"

	"github.com/Nergous/structures/pqueue"
)

// A queue hands out the key with the smallest priority first, regardless of
// the order in which keys were pushed.
func Example() {
	q := pqueue.New[string, int]()
	q.Push("write docs", 3)
	q.Push("fix outage", 1)
	q.Push("review PR", 2)

	for !q.IsEmpty() {
		task, priority, _ := q.Pop()
		fmt.Println(priority, task)
	}
	// Output:
	// 1 fix outage
	// 2 review PR
	// 3 write docs
}

// Pushing a key that is already queued changes its priority in place instead
// of adding a duplicate. Here a shortest-path search lowers a vertex's tentative
// distance whenever it finds a shorter route: classic Dijkstra.
func Example_dijkstra() {
	type edge struct {
		to     string
		weight int
	}
	graph := map[string][]edge{
		"A": {{"B", 4}, {"C", 1}},
		"C": {{"B", 2}, {"D", 5}},
		"B": {{"D", 1}},
	}

	dist := map[string]int{"A": 0}
	q := pqueue.New[string, int]()
	q.Push("A", 0)

	for !q.IsEmpty() {
		v, d, _ := q.Pop()
		for _, e := range graph[v] {
			nd := d + e.weight
			if old, seen := dist[e.to]; !seen || nd < old {
				dist[e.to] = nd
				q.Push(e.to, nd) // inserts, or lowers the queued distance
			}
		}
	}

	for _, v := range []string{"A", "B", "C", "D"} {
		fmt.Println(v, dist[v])
	}
	// Output:
	// A 0
	// B 3
	// C 1
	// D 4
}

// NewMax puts the largest priority on top. Keys with equal priorities come out
// in the order they were first pushed.
func ExampleNewMax() {
	q := pqueue.NewMax[string, int]()
	q.Push("bronze", 1)
	q.Push("gold", 3)
	q.Push("silver", 2)
	q.Push("gold, tied", 3)

	for _, it := range q.PopN(q.Len()) {
		fmt.Println(it.Priority, it.Key)
	}
	// Output:
	// 3 gold
	// 3 gold, tied
	// 2 silver
	// 1 bronze
}

// NewFunc orders any priority type. Here jobs are scheduled by deadline.
func ExampleNewFunc() {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	q := pqueue.NewFunc[string](func(a, b time.Time) int { return a.Compare(b) })

	q.Push("send report", start.Add(3*time.Hour))
	q.Push("backup", start.Add(1*time.Hour))
	q.Push("rotate logs", start.Add(2*time.Hour))

	job, at, _ := q.Peek()
	fmt.Println(job, at.Format("15:04"))
	// Output: backup 10:00
}

// Push reports whether the key was new, and changing an existing key's
// priority keeps the queue ordered.
func ExamplePriorityQueue_Push() {
	q := pqueue.New[string, int]()
	fmt.Println(q.Push("task", 5))
	fmt.Println(q.Push("other", 3))
	fmt.Println(q.Push("task", 1)) // already queued: priority changes

	top, p, _ := q.Peek()
	fmt.Println(top, p, q.Len())
	// Output:
	// true
	// true
	// false
	// task 1 2
}

// Remove cancels a queued key by its identity, without scanning the queue.
func ExamplePriorityQueue_Remove() {
	q := pqueue.New[int, string]()
	q.Push(101, "b")
	q.Push(102, "a")
	q.Push(103, "c")

	p, ok := q.Remove(102)
	fmt.Println(p, ok)

	_, ok = q.Remove(102)
	fmt.Println(ok)

	fmt.Println(q.Sorted())
	// Output:
	// a true
	// false
	// [{101 b} {103 c}]
}

// Sorted returns a priority-ordered snapshot without consuming the queue.
func ExamplePriorityQueue_Sorted() {
	q := pqueue.NewFunc[string](func(a, b float64) int { return cmp.Compare(a, b) })
	q.Push("x", 2.5)
	q.Push("y", 0.5)
	q.Push("z", 1.5)

	fmt.Println(q.Sorted())
	fmt.Println(q.Len())
	// Output:
	// [{y 0.5} {z 1.5} {x 2.5}]
	// 3
}
